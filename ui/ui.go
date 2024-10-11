package ui

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/mattn/go-tty"
	"github.com/philippta/godbg/dlv"
	"github.com/philippta/godbg/term"
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
		dbg:        dbg,
		tty:        tty,
		paneNum:    2,
		paneActive: 0,
		repaint:    make(chan struct{}),
	}
	v.Init()
	defer v.Close()

	go func() {
		v.InputLoop()
		cancel()
	}()

	<-ctx.Done()
}

type View struct {
	tty     *tty.TTY
	repaint chan struct{}

	width      int
	height     int
	paneActive int
	paneNum    int

	sourceView    Source
	variablesView Variables

	dbg *dlv.Debugger
}

func (v *View) Init() {
	out := v.tty.Output()
	out.Write(term.AltScreen)
	out.Write(term.HideCursor)

	w, h, _ := v.tty.Size()
	v.Resize(w, h)

	go v.ResizeLoop()
	go v.RepaintLoop()

	v.sourceView.InitBreakpoints(v.dbg)
	v.sourceLoadFile()
	v.variablesView.Load(v.dbg, v.height)

	v.Repaint()
}

func (v *View) Close() {
	out := v.tty.Output()
	out.Write(term.ShowCursor)
	out.Write(term.ExitAltScreen)
	v.tty.Close()
	close(v.repaint)
}

func (v *View) Resize(width, height int) {
	v.width = width
	v.height = height
	v.sourceView.Resize(width/2, height)
	v.variablesView.Resize(width/2-1, height)
}

func (v *View) ResizeLoop() {
	for size := range v.tty.SIGWINCH() {
		v.Resize(size.W, size.H)
		v.repaint <- struct{}{}
	}
}

func (v *View) Repaint() {
	v.repaint <- struct{}{}
}

func (v *View) RepaintLoop() {
	for range v.repaint {
		out := v.tty.Output()
		out.Write(term.ResetCursor)
		out.WriteString(v.Render())
	}
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

		switch key {
		case '\t':
			v.paneActive = (v.paneActive + 1) % v.paneNum
		case 'q':
			return
		case 'k': // Move up
			switch v.paneActive {
			case 0:
				v.sourceView.MoveUp(v.height)
			case 1:
				v.variablesView.MoveUp()
			}
		case 'j': // Move down
			switch v.paneActive {
			case 0:
				v.sourceView.MoveDown(v.height)
			case 1:
				v.variablesView.MoveDown()
			}
		case 'l': // Expand
			v.variablesView.Expand()
		case 'h': // Collapse
			v.variablesView.Collapse(v.height)
		case 's': // Step
			v.dbg.Step()
			v.sourceLoadFile()
			v.variablesView.Load(v.dbg, v.height)
		case 'i': // Step in
			v.dbg.StepIn()
			v.sourceLoadFile()
			v.variablesView.Load(v.dbg, v.height)
		case 'o': // Step out
			v.dbg.StepOut()
			v.sourceLoadFile()
			v.variablesView.Load(v.dbg, v.height)
		case 'c': // Continue
			v.dbg.Continue()
			v.sourceLoadFile()
			v.variablesView.Load(v.dbg, v.height)
		case 'b': // Breakpoint
			v.sourceView.ToggleBreakpoint(v.dbg)
			// case 'v':
			// 	vars, err := v.dbg.Variables()
			// 	must(err)
			// 	b, _ := json.MarshalIndent(vars, "", "  ")
			// 	os.WriteFile("ui/testdata/vars.json", b, 0o644)
		}

		if v.dbg.Exited() {
			return
		}

		v.Repaint()
	}
}

func (v *View) Render() string {
	// start := time.Now()
	// defer func() {
	// 	debug.Logf("Render time: %v", time.Since(start))
	// }()

	source, sourceLens := v.sourceView.Render(v.paneActive == 0)
	if len(source) == 0 {
		return ""
	}

	variables, variablesLens := v.variablesView.Render(v.paneActive == 1)

	return verticalSplit(
		v.width, v.height,
		block{source, sourceLens},
		block{variables, variablesLens},
	)
}

func listenresize(v *View, tty *tty.TTY, repaint chan<- struct{}) {
	for size := range tty.SIGWINCH() {
		v.width = size.W
		v.height = size.H
		v.variablesView.Resize(size.W, size.H)
		repaint <- struct{}{}
	}
}

func (v *View) sourceLoadFile() {
	changed := v.sourceView.LoadLocation(v.dbg, v.height)
	if changed {
		v.variablesView.ResetCursor(v.height)
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
