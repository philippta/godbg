package main

import (
	"os"
	"time"

	"github.com/philippta/godbg/debug"
	"github.com/philippta/godbg/ui"
)

func main() {
	debug.Truncate()

	program := "/Users/philipp/code/hellworld/helloworld"
	if len(os.Args) > 1 {
		program = os.Args[1]
	}

	ui.Run(program)
}

func minisleep() {
	time.Sleep(100 * time.Millisecond)
}
