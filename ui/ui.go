package ui

import (
	"encoding/json"
	"net"
	"slices"

	tea "github.com/charmbracelet/bubbletea"
	godap "github.com/google/go-dap"
	"github.com/philippta/godbg/dap"
	"github.com/philippta/godbg/debug"
	"github.com/philippta/godbg/ui/source"
	"github.com/philippta/godbg/ui/variables"
)

func Run(program string) {
	conn, msgs := dap.Launch(program)
	dap.BreakpointFunc(conn, "main.main")
	dap.Continue(conn)

	v := view{
		conn:        conn,
		msgs:        msgs,
		source:      source.View{},
		breakpoints: map[string][]int{},
	}

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
	currentFile string
	currentLine int

	source source.View
	thread int
	conn   net.Conn
	msgs   <-chan godap.Message
	width  int
	height int

	breakpoints map[string][]int
	variables   []godap.Variable
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
		case "k":
			v.source.ScrollBy(-1)
		case "j":
			v.source.ScrollBy(1)
		case "n", "s":
			dap.Next(v.conn, v.thread)
		case "i":
			dap.StepIn(v.conn, v.thread)
		case "o":
			dap.StepOut(v.conn, v.thread)
		case "c":
			dap.Continue(v.conn)
		case "b":
			path, line := v.source.Location()
			bps := v.breakpoints[path]
			toggleSliceInt(&bps, line)
			v.breakpoints[path] = bps
			dap.BreakpointsFile(v.conn, path, bps)
			v.source.SetBreakpoints(bps)
		}
	case tea.WindowSizeMsg:
		v.source.Resize(msg.Width, msg.Height-5)
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
			if len(dmsg.Body.StackFrames) > 0 {
				if dmsg.Body.StackFrames[0].Source != nil {
					path := dmsg.Body.StackFrames[0].Source.Path
					line := dmsg.Body.StackFrames[0].Line
					v.source.LoadFile(path, line, v.breakpoints[path])
					v.currentFile = path
					v.currentLine = line
				}
				dap.Scopes(v.conn, dmsg.Body.StackFrames[0].Id)
			}
		case *godap.ScopesResponse:
			if len(dmsg.Body.Scopes) > 0 {
				dap.Variables(v.conn, dmsg.Body.Scopes[0].VariablesReference)
			}
		case *godap.VariablesResponse:
			v.variables = dmsg.Body.Variables
		case *godap.TerminatedEvent:
			return v, tea.Quit
		}
	}

	return v, v.waitForDapMsg()
}

func (v view) View() string {
	return v.source.Render() + "\n" + variables.Print(v.variables)
}

func (v view) waitForDapMsg() tea.Cmd {
	return func() tea.Msg {
		return <-v.msgs
	}
}

func toggleSliceInt(ii *[]int, i int) {
	if slices.Contains(*ii, i) {
		*ii = slices.DeleteFunc(*ii, func(x int) bool { return x == i })
	} else {
		*ii = append(*ii, i)
	}
}
