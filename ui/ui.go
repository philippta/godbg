package ui

import (
	"encoding/json"
	"net"

	tea "github.com/charmbracelet/bubbletea"
	godap "github.com/google/go-dap"
	"github.com/philippta/godbg/dap"
	"github.com/philippta/godbg/debug"
	"github.com/philippta/godbg/ui/source"
)

func Run(program string) {
	conn, msgs := dap.Launch(program)
	dap.BreakpointFunc(conn, "main.main")
	dap.Continue(conn)

	v := view{
		conn:   conn,
		msgs:   msgs,
		source: source.View{},
	}

	p := tea.NewProgram(v, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		panic(err)
	}
}

type view struct {
	source source.View
	thread int
	conn   net.Conn
	msgs   <-chan godap.Message
	width  int
	height int
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
		}
	case tea.WindowSizeMsg:
		v.source.Resize(msg.Width, msg.Height)
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
			if len(dmsg.Body.StackFrames) > 0 && dmsg.Body.StackFrames[0].Source != nil {
				v.source.LoadFile(dmsg.Body.StackFrames[0].Source.Path, dmsg.Body.StackFrames[0].Line)
			}
		case *godap.TerminatedEvent:
			return v, tea.Quit
		}
	}

	return v, v.waitForDapMsg()
}

func (v view) View() string {
	return v.source.Render()
}

func (v view) waitForDapMsg() tea.Cmd {
	return func() tea.Msg {
		return <-v.msgs
	}
}
