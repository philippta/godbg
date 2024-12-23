package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/philippta/godbg/debug"
	"github.com/philippta/godbg/dlv"
	"github.com/philippta/godbg/ui"
)

const usage = "Usage: godbg <debug|test|exec> [path] [func regex]"

func main() {
	debug.Truncate()
	args := os.Args[1:]

	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, usage)
		return
	}

	switch args[0] {
	case "debug":
		var path string
		if len(args) > 1 {
			path = args[1]
		}

		dbg, err := dlv.Build(path, args[2:])
		if err != nil {
			panic(err)
		}
		defer dbg.Close()

		path, _ = filepath.Abs(filepath.Dir(path))
		ui.Run(dbg, path)
	case "test":
		var path string
		if len(args) > 1 {
			path = args[1]
		}
		var funcExpr string
		if len(args) > 2 {
			funcExpr = args[2]
		}

		dbg, err := dlv.Test(path, funcExpr)
		if err != nil {
			panic(err)
		}
		defer dbg.Close()

		path, _ = filepath.Abs(filepath.Dir(path))
		ui.Run(dbg, path)
	case "exec":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, usage)
			return
		}
		path := args[1]
		dbg, err := dlv.Exec(path)
		if err != nil {
			panic(err)
		}
		defer dbg.Close()

		path, _ = filepath.Abs(filepath.Dir(path))
		ui.Run(dbg, path)
	}

	fmt.Fprintln(os.Stderr, "Usage: godbg <debug|test> [path] [func regex]")
}
