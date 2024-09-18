package ui

import (
	"encoding/json"
	"net"
	"slices"

	tea "github.com/charmbracelet/bubbletea"
	godap "github.com/google/go-dap"
	"github.com/philippta/godbg/dap"
	"github.com/philippta/godbg/debug"
)

func Run(program string) {
	conn, msgs := dap.Launch(program)
	dap.BreakpointFunc(conn, "main.main")
	dap.Continue(conn)

	v := view{
		conn: conn,
		msgs: msgs,
	}
	v.sourceView.breakpoints = map[string][]int{}

	p := tea.NewProgram(v, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		panic(err)
	}
}

type breakpoint struct {
	file string
	line int
}

type view struct {
	sourceView struct {
		width       int
		height      int
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

func (v view) Init() tea.Cmd {
	return v.waitForDapMsg()
}

func (v view) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return v, tea.Quit
		case "k": // Move up
			v.sourceMoveUp()
		case "j": // Move down
			v.sourceMoveDown()
		case "s": // Step
			dap.Next(v.conn, v.thread)
		case "i": // Step in
			dap.StepIn(v.conn, v.thread)
		case "o": // Step out
			dap.StepOut(v.conn, v.thread)
		case "c": // Continue
			dap.Continue(v.conn)
		case "b": // Breakpoint
			v.sourceToggleBreakpoint(v.sourceView.lineCursor)
		}
	case tea.WindowSizeMsg:
		v.sourceView.height = msg.Height
		v.sourceView.width = msg.Width
	case godap.Message:
		debug.Logf("%T", msg)
		b, _ := json.MarshalIndent(msg, "", "  ")
		debug.Logf("%s", b)
		debug.Logf("")

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
		case *godap.TerminatedEvent:
			return v, tea.Quit
		}
	}

	return v, v.waitForDapMsg()
}

func (v view) View() string {
	source := sourceRender(
		v.sourceView.lines,
		v.sourceView.width,
		v.sourceView.height,
		v.sourceView.lineStart,
		v.sourceView.pcCursor,
		v.sourceView.lineCursor,
		v.sourceView.breakpoints[v.sourceView.file],
	)

	return source
}

func (v view) waitForDapMsg() tea.Cmd {
	return func() tea.Msg {
		return <-v.msgs
	}
}

func (v *view) sourceLoadFile(path string, line int) {
	if v.sourceView.file != path {
		v.sourceView.file = path
		v.sourceView.lines = sourceLoadFile(path)
	}
	v.sourceView.pcCursor = line - 1
	v.sourceView.lineCursor = line - 1
}

func (v *view) sourceMoveUp() {
	v.sourceView.lineCursor = max(0, v.sourceView.lineCursor-1)
	if v.sourceView.lineCursor < v.sourceView.lineStart+2 {
		v.sourceView.lineStart = max(0, v.sourceView.lineStart-1)
	}
}

func (v *view) sourceMoveDown() {
	v.sourceView.lineCursor = min(v.sourceView.lineCursor+1, len(v.sourceView.lines)-2)
	if v.sourceView.lineCursor > v.sourceView.lineStart+v.sourceView.height-3 {
		v.sourceView.lineStart = min(v.sourceView.lineStart+1, len(v.sourceView.lines)-1-v.sourceView.height)
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
