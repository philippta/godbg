package ui

import (
	"context"
	"log"
	"os/signal"
	"syscall"

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
	PaneFiles
	PaneCount
)

func Run(dbg *dlv.Debugger) {
	tty, err := tty.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer tty.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	v := &View{
		dbg:   dbg,
		tty:   tty,
		focus: PaneFiles,
	}

	out := v.tty.Output()
	out.Write(term.AltScreen)
	out.Write(term.HideCursor)

	v.UpdateFocus()

	w, h, _ := v.tty.Size()
	v.Resize(w, h)
	go v.ResizeLoop()

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

		debug.Logf("Input: %v", key)

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
				v.focus = PaneFiles
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
			case 'q':
				return
			case 16:
				v.focus = PaneFiles
			}
		case PaneFiles:
			switch key {
			case '\t':
				v.focus = (v.focus + 1) % PaneCount
				v.UpdateFocus()
			case 16: // CTRL+P
				v.focus = PaneSource
			default:
				var more []rune
				for v.tty.Buffered() {
					key, _ := v.tty.ReadRune()
					more = append(more, key)
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

	file, line := v.dbg.Location()
	p.Mark("Location")
	vars, _ := v.dbg.Variables()
	p.Mark("Variables")

	v.source.LoadLocation(file, line)
	p.Mark("LoadLoc")

	v.variables.Load(vars)
	p.Mark("LoadVar")
	if file != v.prevFile {
		v.variables.ResetCursor(v.height)
	}

	v.prevFile = file
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
		colors.SetColor(i, v.width/2, 1, frame.ColorFGBlack)
		text.WriteAt(i, v.width/2, '│')
	}

	sourceText, sourceColors := v.source.RenderFrame()
	text.CopyFrom(0, 0, sourceText)
	colors.CopyFrom(0, 0, sourceColors)
	p.Mark("Render Source")

	varsText, varsColors := v.variables.RenderFrame()
	text.CopyFrom(0, v.width/2+1, varsText)
	colors.CopyFrom(0, v.width/2+1, varsColors)
	p.Mark("Render Variables")

	var filesY, filesX int
	if v.focus == PaneFiles {
		filesY, filesX = 1, 10
		filesText, filesColors := v.files.RenderFrame()
		text.CopyFrom(filesY, filesX, filesText)
		colors.CopyFrom(filesY, filesX, filesColors)
		p.Mark("Render Files")
	}

	out := v.tty.Output()
	out.Write(term.HideCursor)
	out.Write(term.ResetCursor)
	text.PrintColored(out, colors)

	if v.focus == PaneFiles {
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
	v.files.Focused = v.focus == PaneFiles
}

func (v *View) Resize(width, height int) {
	v.width = width
	v.height = height
	v.source.Resize(width/2, height)
	v.variables.Resize(width/2-1, height)
	v.files.Resize(width/2, height/2)
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
