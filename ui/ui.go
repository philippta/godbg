package ui

import (
	"bytes"
	"log"
	"net"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	godap "github.com/google/go-dap"
	"github.com/mattn/go-tty"
	"github.com/philippta/godbg/dap"
	"github.com/philippta/godbg/debug"
	"github.com/philippta/godbg/term"
)

func Run(program string) {
	conn, msgs := dap.Launch(program)
	dap.BreakpointFunc(conn, "main.main")
	dap.Continue(conn)

	v := &view{}
	v.conn = conn
	v.msgs = msgs
	v.sourceView.breakpoints = map[string][]int{}

	tty, err := tty.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer tty.Close()

	out := tty.Output()
	out.Write(term.AltScreen)
	out.Write(term.HideCursor)
	defer out.Write(term.ShowCursor)
	defer out.Write(term.ExitAltScreen)
	resize(v, tty)

	repaintCh := make(chan struct{})
	go listenresize(v, tty, repaintCh)
	go paintloop(v, tty, repaintCh)
	go daploop(v, msgs, repaintCh)
	inputloop(v, tty, repaintCh)
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

func daploop(v *view, msgs <-chan godap.Message, repaint chan<- struct{}) {
	for msg := range msgs {
		switch dmsg := msg.(type) {
		case *godap.StoppedEvent:
			v.thread = dmsg.Body.ThreadId
			dap.Stack(v.conn, dmsg.Body.ThreadId)
		case *godap.StackTraceResponse:
			if len(dmsg.Body.StackFrames) == 0 {
				break
			}
			frame := dmsg.Body.StackFrames[0]
			if frame.Source != nil {
				v.sourceLoadFile(frame.Source.Path, frame.Line)
			}
			dap.Scopes(v.conn, frame.Id)
		case *godap.ScopesResponse:
			if len(dmsg.Body.Scopes) > 0 {
				dap.Variables(v.conn, dmsg.Body.Scopes[0].VariablesReference)
			}
		case *godap.VariablesResponse:
			v.variablesView.variables = dmsg.Body.Variables
			repaint <- struct{}{}
		case *godap.SetBreakpointsResponse:
			repaint <- struct{}{}
		case *godap.TerminatedEvent:
			return
		}

	}
}

func inputloop(v *view, tty *tty.TTY, repaint chan<- struct{}) {
	for {
		key, err := tty.ReadRune()
		if err != nil {
			panic(err)
		}

		switch key {
		case 'q':
			return
		case 'k': // Move up
			v.sourceMoveUp()
			repaint <- struct{}{}
		case 'j': // Move down
			v.sourceMoveDown()
			repaint <- struct{}{}
		case 's': // Step
			dap.Next(v.conn, v.thread)
		case 'i': // Step in
			dap.StepIn(v.conn, v.thread)
		case 'o': // Step out
			dap.StepOut(v.conn, v.thread)
		case 'c': // Continue
			dap.Continue(v.conn)
		case 'b': // Breakpoint
			v.sourceToggleBreakpoint(v.sourceView.lineCursor)
		}
	}
}

func render(v *view) string {
	start := time.Now()
	defer func() {
		debug.Logf("Render time: %v", time.Since(start))
	}()

	source, sourceLens := sourceRender(
		v.sourceView.lines,
		v.width/2,
		v.height,
		v.sourceView.lineStart,
		v.sourceView.pcCursor,
		v.sourceView.lineCursor,
		v.sourceView.breakpoints[v.sourceView.file],
	)

	if len(source) == 0 {
		return ""
	}

	variables := []string{"Line1", "Line2 HELLO WORLD", "Line3"}
	variablesLens := []int{5, 17, 5}

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

type breakpoint struct {
	file string
	line int
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
		breakpoints map[string][]int
	}
	variablesView struct {
		variables []godap.Variable
	}

	thread int
	conn   net.Conn
	msgs   <-chan godap.Message
}

func (v *view) sourceLoadFile(path string, line int) {
	if v.sourceView.file != path {
		src, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}

		src = bytes.ReplaceAll(src, []byte{'\t'}, []byte("    "))

		v.sourceView.lines = bytes.Split(src, []byte{'\n'})
		v.sourceView.file = path
	}
	v.sourceView.pcCursor = line - 1
	v.sourceView.lineCursor = line - 1
	v.sourceView.lineStart = max(0, min(line-1-v.height/2, len(v.sourceView.lines)-1-v.height))
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
func (v *view) sourceToggleBreakpoint(i int) {
	bp := v.sourceView.breakpoints[v.sourceView.file]
	if slices.Contains(bp, i) {
		bp = slices.DeleteFunc(bp, func(x int) bool { return x == i })
	} else {
		bp = append(bp, i)
	}
	v.sourceView.breakpoints[v.sourceView.file] = bp

	dap.BreakpointsFile(v.conn, v.sourceView.file, bp)
}
