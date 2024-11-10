package ui

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/mattn/go-tty"
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

func Run(dbg *dlv.Debugger) {
	tty, err := tty.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer tty.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	v := &View{
		dbg:    dbg,
		tty:    tty,
		focus:  PaneSource,
		source: Source{Focused: true},
	}

	out := v.tty.Output()
	out.Write(term.AltScreen)
	out.Write(term.HideCursor)

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
		for v.tty.Buffered() {
			v.tty.ReadRune()
		}

		switch key {
		case '\t':
			v.focus = (v.focus + 1) % PaneCount
			v.source.Focused = v.focus == PaneSource
			v.variables.Focused = v.focus == PaneVariables
		case 'q':
			return
		case 'k': // Move up
			switch v.focus {
			case PaneSource:
				v.source.MoveUp()
			case PaneVariables:
				v.variables.MoveUp()
			}
		case 'j': // Move down
			switch v.focus {
			case PaneSource:
				v.source.MoveDown()
			case PaneVariables:
				v.variables.MoveDown()
			}
		case 'l': // Expand
			v.variables.Expand()
		case 'h': // Collapse
			v.variables.Collapse()
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
			// case 'v':
			// 	vars, err := v.dbg.Variables()
			// 	must(err)
			// 	b, _ := json.MarshalIndent(vars, "", "  ")
			// 	os.WriteFile("ui/testdata/vars.json", b, 0o644)
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
		colors.SetColor(i, v.width/2, 1, frame.ColorReset)
		text.WriteAt(i, v.width/2, '|')
	}

	sourceText, sourceColors := v.source.RenderFrame()
	text.CopyFrom(0, 0, sourceText)
	colors.CopyFrom(0, 0, sourceColors)
	p.Mark("Render Source")

	varsText, varsColors := v.variables.RenderFrame()
	text.CopyFrom(0, v.width/2+1, varsText)
	colors.CopyFrom(0, v.width/2+1, varsColors)
	p.Mark("Render Variables")

	out := v.tty.Output()
	out.Write(term.ResetCursor)
	text.PrintColored(out, colors)
	p.Mark("Print Output")
	p.End()
}

func (v *View) Resize(width, height int) {
	v.width = width
	v.height = height
	v.source.Resize(width/2, height)
	v.variables.Resize(width/2-1, height)
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
