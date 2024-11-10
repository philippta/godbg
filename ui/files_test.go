package ui

import (
	"os"
	"testing"
)

func TestFilesRender(t *testing.T) {
	f := &Files{
		Size:   Size{70, 25},
		Search: "main.go",
	}

	text, colors := f.RenderFrame()
	text.PrintLinesColored(os.Stdout, colors)
}
