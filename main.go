package main

import (
	"os"
	"time"

	"github.com/philippta/godbg/ui"
)

func main() {
	// w, msg := dap.Launch("/Users/philipp/code/hellworld/helloworld")
	// // dap.Threads(conn)
	// // minisleep()
	// dap.BreakpointFunc(w, "main.main")
	// minisleep()
	// dap.Continue(w)
	// minisleep()
	// dap.Threads(w)
	// minisleep()
	// dap.Stack(w, 1)
	// minisleep()
	// dap.Next(w, 1)
	// minisleep()
	// dap.Stack(w, 1)
	// minisleep()

	program := "/Users/philipp/code/hellworld/helloworld"
	if len(os.Args) > 1 {
		program = os.Args[1]
	}

	ui.Run(program)
}

func minisleep() {
	time.Sleep(100 * time.Millisecond)
}
