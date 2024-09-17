package main

import (
	"os"
	"time"

	"github.com/philippta/godbg/ui"
)

func main() {
	program := "/Users/philipp/code/hellworld/helloworld"
	if len(os.Args) > 1 {
		program = os.Args[1]
	}

	ui.Run(program)
}

func minisleep() {
	time.Sleep(100 * time.Millisecond)
}
