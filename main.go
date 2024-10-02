package main

import (
	"os"

	"github.com/philippta/godbg/debug"
	"github.com/philippta/godbg/dlv"
	"github.com/philippta/godbg/ui"
)

func main() {
	debug.Truncate()

	program := "/Users/philipp/code/hellworld/helloworld"
	if len(os.Args) > 1 {
		program = os.Args[1]
	}

	dbg, err := dlv.Open(program)
	if err != nil {
		panic(err)
	}
	defer dbg.Close()

	ui.Run(dbg)
}
