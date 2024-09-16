package dap_test

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/google/go-dap"
)

var seq = 1

func TestDap(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:54444")
	if err != nil {
		panic(err)
	}

	go func() {
		exec.Command("dlv", "dap", "--client-addr", "127.0.0.1:54444").Run()
	}()

	conn, err := ln.Accept()

	reader := bufio.NewReader(conn)
	go func() {
		for {
			msg, err := dap.ReadProtocolMessage(reader)
			if err != nil {
				os.Exit(0)
			}
			fmt.Printf("%#v\n\n", msg)
		}
	}()

	init := &dap.InitializeRequest{Request: newRequest("initialize")}
	init.Arguments = dap.InitializeRequestArguments{
		AdapterID:                    "go",
		PathFormat:                   "path",
		LinesStartAt1:                true,
		ColumnsStartAt1:              true,
		SupportsVariableType:         true,
		SupportsVariablePaging:       true,
		SupportsRunInTerminalRequest: true,
		Locale:                       "en-us",
	}
	send(conn, init)
	time.Sleep(500 * time.Millisecond)

	launch := &dap.LaunchRequest{Request: newRequest("launch")}
	launch.Arguments = []byte(`{
            "request": "launch",
            "mode": "exec",
            "program": "/Users/philipp/code/hellworld/helloworld",
            "stopOnEntry": true
}`)
	send(conn, launch)
	time.Sleep(500 * time.Millisecond)

	threads := &dap.ThreadsRequest{Request: newRequest("threads")}
	send(conn, threads)
	time.Sleep(500 * time.Millisecond)

	breakpoint := &dap.SetFunctionBreakpointsRequest{Request: newRequest("setFunctionBreakpoints")}
	breakpoint.Arguments = dap.SetFunctionBreakpointsArguments{
		Breakpoints: []dap.FunctionBreakpoint{{
			Name: "main.main",
		}},
	}
	send(conn, breakpoint)
	time.Sleep(500 * time.Millisecond)

	// breakpoint := &dap.SetBreakpointsRequest{Request: newRequest("setBreakpoints")}
	// breakpoint.Arguments = dap.SetBreakpointsArguments{
	// 	Source: dap.Source{
	// 		Name: "main.go",
	// 		Path: "/Users/philipp/code/hellworld/main.go",
	// 	},
	// 	Breakpoints: []dap.SourceBreakpoint{{
	// 		Line: 23,
	// 	}},
	// }
	// send(conn, breakpoint)
	// time.Sleep(500 * time.Millisecond)

	cont := &dap.ContinueRequest{Request: newRequest("continue")}
	cont.Arguments = dap.ContinueArguments{
		SingleThread: false,
	}
	send(conn, cont)
	time.Sleep(500 * time.Millisecond)

	stack := &dap.StackTraceRequest{Request: newRequest("stackTrace")}
	stack.Arguments = dap.StackTraceArguments{
		ThreadId: 1,
	}
	send(conn, stack)
	time.Sleep(500 * time.Millisecond)

	time.Sleep(2 * time.Second)
}

func send(conn io.Writer, request dap.Message) {
	dap.WriteProtocolMessage(conn, request)
}

func newRequest(command string) dap.Request {
	req := dap.Request{}
	req.Type = "request"
	req.Command = command
	req.Seq = seq
	seq++
	return req
}
