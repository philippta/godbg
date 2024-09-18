package dap

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os/exec"

	"github.com/google/go-dap"
)

var seq = 1

func Launch(path string) (net.Conn, <-chan dap.Message) {
	conn, err := Open()
	if err != nil {
		panic(err)
	}

	reader := bufio.NewReader(conn)
	msgs := make(chan dap.Message)
	go func() {
		for {
			msg, err := dap.ReadProtocolMessage(reader)
			if err != nil {
				break
			}
			// b, _ := json.MarshalIndent(msg, "", "  ")
			// debug.Logf("%s", string(b))
			msgs <- msg
		}
	}()

	Init(conn)
	Exec(conn, path)

	return conn, msgs
}

func Open() (net.Conn, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:54444")
	if err != nil {
		return nil, fmt.Errorf("listen tcp: %w", err)
	}

	go func() {
		if err := exec.Command("dlv", "dap", "--client-addr", "127.0.0.1:54444").Run(); err != nil {
			panic(err)
		}
	}()

	conn, err := ln.Accept()
	if err != nil {
		return nil, fmt.Errorf("accept: %w", err)
	}

	return conn, nil
}

func Init(w io.Writer) {
	req := &dap.InitializeRequest{Request: newRequest("initialize")}
	req.Arguments = dap.InitializeRequestArguments{
		AdapterID:                    "go",
		PathFormat:                   "path",
		LinesStartAt1:                true,
		ColumnsStartAt1:              true,
		SupportsVariableType:         true,
		SupportsVariablePaging:       true,
		SupportsRunInTerminalRequest: true,
		Locale:                       "en-us",
	}
	dap.WriteProtocolMessage(w, req)
}

func Exec(w io.Writer, path string) {
	req := &dap.LaunchRequest{Request: newRequest("launch")}
	req.Arguments = []byte(fmt.Sprintf(`{
            "request": "launch",
            "mode": "exec",
            "program": "%s",
            "stopOnEntry": true
	}`, path))
	dap.WriteProtocolMessage(w, req)
}

func Stack(w io.Writer, thread int) {
	req := &dap.StackTraceRequest{Request: newRequest("stackTrace")}
	req.Arguments = dap.StackTraceArguments{
		ThreadId: thread,
		Levels:   1,
	}
	dap.WriteProtocolMessage(w, req)
}

func Scopes(w io.Writer, frameID int) {
	req := &dap.ScopesRequest{Request: newRequest("scopes")}
	req.Arguments = dap.ScopesArguments{
		FrameId: frameID,
	}
	dap.WriteProtocolMessage(w, req)
}

func Threads(w io.Writer) {
	req := &dap.ThreadsRequest{Request: newRequest("threads")}
	dap.WriteProtocolMessage(w, req)
}

func Continue(w io.Writer) {
	req := &dap.ContinueRequest{Request: newRequest("continue")}
	req.Arguments = dap.ContinueArguments{
		SingleThread: false,
	}
	dap.WriteProtocolMessage(w, req)
}

func BreakpointFunc(w io.Writer, name string) {
	req := &dap.SetFunctionBreakpointsRequest{Request: newRequest("setFunctionBreakpoints")}
	req.Arguments = dap.SetFunctionBreakpointsArguments{
		Breakpoints: []dap.FunctionBreakpoint{{
			Name: name,
		}},
	}
	dap.WriteProtocolMessage(w, req)
}

type Breakpoint struct {
	Path string
	Line int
}

func BreakpointsFile(w io.Writer, path string, lines []int) {
	bps := make([]dap.SourceBreakpoint, len(lines))
	for i, line := range lines {
		bps[i] = dap.SourceBreakpoint{Line: line + 1}
	}

	req := &dap.SetBreakpointsRequest{Request: newRequest("setBreakpoints")}
	req.Arguments = dap.SetBreakpointsArguments{
		Source: dap.Source{
			Name: path,
			Path: path,
		},
		Breakpoints: bps,
	}
	dap.WriteProtocolMessage(w, req)
}

func Next(w io.Writer, thread int) {
	req := &dap.NextRequest{Request: newRequest("next")}
	req.Arguments = dap.NextArguments{
		ThreadId:     thread,
		SingleThread: true,
	}
	dap.WriteProtocolMessage(w, req)
}

func StepIn(w io.Writer, thread int) {
	req := &dap.StepInRequest{Request: newRequest("stepIn")}
	req.Arguments = dap.StepInArguments{
		ThreadId:     thread,
		SingleThread: true,
	}
	dap.WriteProtocolMessage(w, req)
}

func StepOut(w io.Writer, thread int) {
	req := &dap.StepOutRequest{Request: newRequest("stepOut")}
	req.Arguments = dap.StepOutArguments{
		ThreadId:     thread,
		SingleThread: true,
	}
	dap.WriteProtocolMessage(w, req)
}

func Variables(w io.Writer, ref int) {
	req := &dap.VariablesRequest{Request: newRequest("variables")}
	req.Arguments = dap.VariablesArguments{
		VariablesReference: ref,
	}
	dap.WriteProtocolMessage(w, req)
}

func newRequest(command string) dap.Request {
	req := dap.Request{}
	req.Type = "request"
	req.Command = command
	req.Seq = seq
	seq++
	return req
}
