package ui

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/mattn/go-tty"
	"github.com/philippta/godbg/debug"
	"github.com/philippta/godbg/dlv"
	"github.com/philippta/godbg/frame"
	"github.com/philippta/godbg/perf"
	"github.com/philippta/godbg/term"
)

const (
	PaneSource = iota
	PaneVariables
	PaneCount
)

func Run(dbg *dlv.Debugger, dir string) {
	tty, err := tty.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer tty.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	previewCache, err := lru.New[string, []string](100)
	if err != nil {
		log.Fatal(err)
	}

	v := &View{
		dbg:   dbg,
		tty:   tty,
		focus: PaneSource,
		files: Files{
			Dir:          dir,
			PreviewCache: previewCache,
		},
	}

	out := v.tty.Output()
	out.Write(term.AltScreen)
	out.Write(term.HideCursor)
	v.UpdateFocus()

	w, h, _ := v.tty.Size()
	v.Resize(w, h)
	go v.ResizeLoop()

	v.files.LoadFiles()
	v.source.InitBreakpoints(v.dbg)
	v.Update()
	v.Paint()

	defer v.Close()

	go func() {
		v.InputLoop()
		cancel()
	}()

	<-ctx.Done()
}

type View struct {
	tty *tty.TTY

	width    int
	height   int
	focus    int
	prevFile string

	source    Source
	variables Variables
	files     Files
	filesOpen bool

	dbg *dlv.Debugger
}

func (v *View) InputLoop() {
	defer func() {
		if err := recover(); err != nil {
			v.Close()
			panic(err)
		}
	}()

	for {
		var err error
		key, err := v.tty.ReadRune()
		if err != nil {
			panic(err)
		}

		if !v.filesOpen {
			switch v.focus {
			case PaneSource:
				switch key {
				case '\t':
					v.focus = (v.focus + 1) % PaneCount
					v.UpdateFocus()
				case 'k': // Move up
					v.source.MoveUp()
				case 'j': // Move down
					v.source.MoveDown()
				case 's': // Step
					v.dbg.Step()
					v.Update()
				case 'i': // Step in
					v.dbg.StepIn()
					v.Update()
				case 'o': // Step out
					v.dbg.StepOut()
					v.Update()
				case 'c': // Continue
					v.dbg.Continue()
					v.Update()
				case 'b': // Breakpoint
					v.source.ToggleBreakpoint(v.dbg)
				case 'q':
					return
				case 16:
					v.filesOpen = true
					v.files.Reset()
				}
			case PaneVariables:
				switch key {
				case '\t':
					v.focus = (v.focus + 1) % PaneCount
					v.UpdateFocus()
				case 'k': // Move up
					v.variables.MoveUp()
				case 'j': // Move down
					v.variables.MoveDown()
				case 'l': // Expand
					v.variables.Expand()
				case 'h': // Collapse
					v.variables.Collapse()
				case 'q':
					return
				case 16:
					v.filesOpen = true
					v.files.Reset()
				}
			}
		} else {
			switch key {
			case 16: // CTRL+P
				v.filesOpen = false
			default:
				var more []rune
				for v.tty.Buffered() {
					key, _ := v.tty.ReadRune()
					more = append(more, key)
				}
				if key == 27 && len(more) == 0 { // ESC
					v.filesOpen = false
					v.files.Reset()
					break
				}
				if key == 13 { // Enter
					requestedFile := v.files.FilteredFiles[v.files.FileCursor]
					v.filesOpen = false
					v.files.Reset()

					debugFile, debugLine := v.dbg.Location()
					if debugFile == requestedFile {
						v.source.LoadLocation(debugFile, debugLine)
					} else {
						v.source.LoadLocation(requestedFile, 1)
						v.source.Cursors.PC = -1
					}
					break
				}
				v.files.HandleInput(key, more)
			}
		}

		for v.tty.Buffered() {
			key, _ := v.tty.ReadRune()
			debug.Logf("  Buffered: %v", key)
		}

		if v.dbg.Exited() {
			return
		}

		v.Paint()
	}
}

func (v *View) Update() {
	p := perf.Start("Update")

	p.Mark("Location")
	vars, _ := v.dbg.Variables()
	p.Mark("Variables")

	v.source.LoadLocation(v.dbg.Location())
	p.Mark("LoadLoc")

	v.variables.Load(vars)
	p.Mark("LoadVar")
	if v.source.File.Name != v.prevFile {
		v.variables.ResetCursor(v.height)
	}

	v.prevFile = v.source.File.Name
	p.End()
}

func (v *View) Paint() {
	p := perf.Start("Paint")
	text := frame.New(v.height, v.width)
	text.FillSpace()
	p.Mark("Text Frame")

	colors := frame.New(v.height, v.width)
	p.Mark("Color Frame")

	for i := 0; i < v.height; i++ {
		colors.SetColor(i, v.source.Size.Width, 1, frame.ColorFGBlack)
		text.WriteAt(i, v.source.Size.Width, '│')
	}
	p.Mark("Render VBar")

	v.source.RenderFrame(text, colors, 0, 0)
	p.Mark("Render Source")

	v.variables.RenderFrame(text, colors, 0, v.source.Size.Width+1)
	p.Mark("Render Variables")

	filesY, filesX := 3, 16
	if v.filesOpen {
		colors.Fill(frame.ColorFGBlack)
		v.files.RenderFrame(text, colors, filesY, filesX)
		p.Mark("Render Files")
	}

	out := v.tty.Output()
	out.Write(term.HideCursor)
	out.Write(term.ResetCursor)
	text.PrintColored(out, colors)

	if v.filesOpen {
		cy, cx := v.files.CursorPosition()
		out.Write(term.ShowCursor)
		out.Write(term.PositionCursor(cy+filesY, cx+filesX))
	}

	p.Mark("Print Output")
	p.End()
}

func (v *View) UpdateFocus() {
	v.source.Focused = v.focus == PaneSource
	v.variables.Focused = v.focus == PaneVariables
}

func (v *View) Resize(width, height int) {
	v.width = width
	v.height = height

	v.source.Resize(width*5/7, height)
	v.variables.Resize(width-1-v.source.Size.Width, height)
	v.files.Resize(width-32, height-6)
}

func (v *View) ResizeLoop() {
	for size := range v.tty.SIGWINCH() {
		v.Resize(size.W, size.H)
		v.Paint()
	}
}

func (v *View) Close() {
	out := v.tty.Output()
	out.Write(term.ShowCursor)
	out.Write(term.ExitAltScreen)
	v.tty.Close()
}
func must(err error) {
	if err != nil {
		panic(err)
	}
}
