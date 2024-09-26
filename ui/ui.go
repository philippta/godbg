package ui

import (
	"bytes"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"github.com/mattn/go-tty"
	"github.com/philippta/godbg/debug"
	"github.com/philippta/godbg/term"
)

func Run(dlv *rpc2.RPCClient) {
	v := &view{}
	v.dlv = dlv

	tty, err := tty.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer tty.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	out := tty.Output()
	out.Write(term.AltScreen)
	out.Write(term.HideCursor)
	defer out.Write(term.ShowCursor)
	defer out.Write(term.ExitAltScreen)
	resize(v, tty)

	repaintCh := make(chan struct{})
	go listenresize(v, tty, repaintCh)
	go paintloop(v, tty, repaintCh)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				debug.Logf("%v", err)
				cancel()
			}
		}()

		inputloop(v, tty, repaintCh)
		cancel()
	}()

	dlv.CreateBreakpoint(&api.Breakpoint{FunctionName: "main.main"})
	v.state = <-dlv.Continue()
	v.sourceView.breakpoints, _ = dlv.ListBreakpoints(true)
	v.sourceLoadFile()

	repaintCh <- struct{}{}

	<-ctx.Done()
	close(repaintCh)
}

func resize(v *view, tty *tty.TTY) {
	width, height, err := tty.Size()
	if err != nil {
		panic(err)
	}
	v.width = width
	v.height = height
}

func paintloop(v *view, tty *tty.TTY, repaint <-chan struct{}) {
	out := tty.Output()
	for range repaint {
		out.Write(term.ResetCursor)
		out.WriteString(render(v))
	}
}

func inputloop(v *view, tty *tty.TTY, repaint chan<- struct{}) {
	for {
		var err error
		key, err := tty.ReadRune()
		if err != nil {
			panic(err)
		}

		switch key {
		case 'q':
			return
		case 'k': // Move up
			v.sourceMoveUp()
		case 'j': // Move down
			v.sourceMoveDown()
		case 's': // Step
			v.state, err = v.dlv.Next()
			must(err)
			v.sourceLoadFile()
			v.loadVariables()
		case 'i': // Step in
			v.state, err = v.dlv.Step()
			must(err)
			v.sourceLoadFile()
			v.loadVariables()
		case 'o': // Step out
			v.state, err = v.dlv.StepOut()
			must(err)
			v.sourceLoadFile()
			v.loadVariables()
		case 'c': // Continue
			v.state = <-v.dlv.Continue()
			v.sourceLoadFile()
			v.loadVariables()
		case 'b': // Breakpoint
			v.sourceToggleBreakpoint()
		}

		if v.state.Exited {
			return
		}

		repaint <- struct{}{}
	}
}

func render(v *view) string {
	// start := time.Now()
	// defer func() {
	// 	debug.Logf("Render time: %v", time.Since(start))
	// }()

	source, sourceLens := sourceRender(
		v.sourceView.lines,
		v.width/2,
		v.height,
		v.sourceView.lineStart,
		v.sourceView.pcCursor,
		v.sourceView.lineCursor,
		fileBreakpoints(v.sourceView.breakpoints, v.sourceView.file),
	)

	if len(source) == 0 {
		return ""
	}

	variables, variablesLens := variablesRender(v.variablesView.variables, v.width/2, v.height)

	return verticalSplit(
		v.width, v.height,
		block{source, sourceLens},
		block{variables, variablesLens},
	)
}

func listenresize(v *view, tty *tty.TTY, repaint chan<- struct{}) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	for range ch {
		resize(v, tty)
		repaint <- struct{}{}
	}
}

type view struct {
	width  int
	height int

	sourceView struct {
		file        string
		lines       [][]byte
		lineStart   int
		lineCursor  int
		pcCursor    int
		breakpoints []*api.Breakpoint
	}
	variablesView struct {
		variables []api.Variable
	}

	dlv   *rpc2.RPCClient
	state *api.DebuggerState
}

func (v *view) sourceLoadFile() error {
	if v.state.CurrentThread == nil {
		return nil
	}

	path := v.state.CurrentThread.File
	line := v.state.CurrentThread.Line

	if v.sourceView.file != path {
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		src = bytes.ReplaceAll(src, []byte{'\t'}, []byte("    "))

		v.sourceView.lines = bytes.Split(src, []byte{'\n'})
		v.sourceView.file = path
	}
	v.sourceView.pcCursor = line - 1
	v.sourceView.lineCursor = line - 1

	if v.sourceView.lineCursor < v.sourceView.lineStart+2 || v.sourceView.lineCursor > v.sourceView.lineStart+v.height-3 {
		v.sourceView.lineStart = max(0, min(line-1-v.height/2, len(v.sourceView.lines)-1-v.height))
	}

	return nil
}

func (v *view) loadVariables() {
	args, err := v.dlv.ListFunctionArgs(
		api.EvalScope{GoroutineID: v.state.CurrentThread.GoroutineID},
		api.LoadConfig{
			FollowPointers:     true,
			MaxVariableRecurse: 2,
			MaxStringLen:       100,
			MaxArrayValues:     100,
			MaxStructFields:    -1,
		})
	must(err)

	locals, err := v.dlv.ListLocalVariables(
		api.EvalScope{GoroutineID: v.state.CurrentThread.GoroutineID},
		api.LoadConfig{
			FollowPointers:     true,
			MaxVariableRecurse: 2,
			MaxStringLen:       100,
			MaxArrayValues:     100,
			MaxStructFields:    -1,
		})
	must(err)

	v.variablesView.variables = append(args, locals...)
}

func (v *view) sourceMoveUp() {
	v.sourceView.lineCursor = max(0, v.sourceView.lineCursor-1)
	if v.sourceView.lineCursor < v.sourceView.lineStart+2 {
		v.sourceView.lineStart = max(0, v.sourceView.lineStart-1)
	}
}

func (v *view) sourceMoveDown() {
	v.sourceView.lineCursor = min(v.sourceView.lineCursor+1, len(v.sourceView.lines)-2)
	if v.sourceView.lineCursor > v.sourceView.lineStart+v.height-3 {
		v.sourceView.lineStart = min(v.sourceView.lineStart+1, len(v.sourceView.lines)-1-v.height)
	}
}

func (v *view) sourceToggleBreakpoint() {
	var activeBP *api.Breakpoint
	for _, bp := range v.sourceView.breakpoints {
		if bp.File == v.sourceView.file && bp.Line == v.sourceView.lineCursor+1 {
			activeBP = bp
			break
		}
	}

	if activeBP == nil {
		v.dlv.CreateBreakpoint(&api.Breakpoint{
			File: v.sourceView.file,
			Line: v.sourceView.lineCursor + 1,
		})
	} else {
		_, err := v.dlv.ClearBreakpoint(activeBP.ID)
		must(err)
	}

	var err error
	v.sourceView.breakpoints, err = v.dlv.ListBreakpoints(true)
	must(err)
}

func fileBreakpoints(bps []*api.Breakpoint, file string) []int {
	var lineNums []int
	for _, bp := range bps {
		if bp.File == file {
			lineNums = append(lineNums, bp.Line-1)
		}
	}
	return lineNums
}

func countlens(ss []string) []int {
	ll := make([]int, len(ss))
	for i := range ss {
		ll[i] = len(ss[i])
	}
	return ll
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
