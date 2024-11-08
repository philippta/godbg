package ui

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"testing"
	"unicode/utf8"

	"github.com/go-delve/delve/service/api"
	"github.com/mattn/go-tty"
)

//go:embed testdata/format.go
var testfile []byte

func TestSourceRender(t *testing.T) {
	tty, err := tty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer tty.Close()
	w, _, _ := tty.Size()

	src := bytes.ReplaceAll(testfile, []byte{'\t'}, []byte("    "))
	lines := bytes.Split(src, []byte{'\n'})

	source := Source{
		Focused:     true,
		Size:        Size{Width: w, Height: 50},
		File:        File{Lines: lines, LineOffset: 45},
		Cursors:     Cursors{PC: 60, Line: 64},
		Breakpoints: []*api.Breakpoint{{Line: 61}},
	}

	text, colors := source.RenderFrame()
	text.PrintRawColored(os.Stdout, colors)
}

func TestRuneCount(t *testing.T) {
	fmt.Println(len([]byte("\033[93m=> ")))
	fmt.Println(utf8.RuneCount([]byte("\033[93m=> ")))
}

func BenchmarkSourceRender(b *testing.B) {
	src := bytes.ReplaceAll(testfile, []byte{'\t'}, []byte("    "))
	lines := bytes.Split(src, []byte{'\n'})

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(90 * 49)

	source := Source{
		Focused:     true,
		Size:        Size{Width: 90, Height: 50},
		File:        File{Lines: lines, LineOffset: 45},
		Cursors:     Cursors{PC: 60, Line: 64},
		Breakpoints: []*api.Breakpoint{{Line: 61}},
	}
	for n := 0; n < b.N; n++ {
		source.Render()
	}
}

func BenchmarkSourceRenderToFrame(b *testing.B) {
	src := bytes.ReplaceAll(testfile, []byte{'\t'}, []byte("    "))
	lines := bytes.Split(src, []byte{'\n'})

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(90 * 49)

	source := Source{
		Focused:     true,
		Size:        Size{Width: 90, Height: 50},
		File:        File{Lines: lines, LineOffset: 45},
		Cursors:     Cursors{PC: 60, Line: 64},
		Breakpoints: []*api.Breakpoint{{Line: 61}},
	}

	for n := 0; n < b.N; n++ {
		source.RenderFrame()
	}
}
