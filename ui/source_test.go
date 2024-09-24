package ui

import (
	"bytes"
	_ "embed"
	"fmt"
	"testing"
	"unicode/utf8"
)

//go:embed testdata/format.go
var testfile []byte

func TestSourceRender(t *testing.T) {
	src := bytes.ReplaceAll(testfile, []byte{'\t'}, []byte("    "))
	lines := bytes.Split(src, []byte{'\n'})

	source, lens := sourceRender(
		lines,
		50,
		50,
		20,
		40,
		40,
		nil,
	)

	for i := range source {
		fmt.Println(lens[i], source[i])
	}
}

func TestRuneCount(t *testing.T) {
	fmt.Println(len([]byte("\033[93m=> ")))
	fmt.Println(utf8.RuneCount([]byte("\033[93m=> ")))
}

func BenchmarkSourceRenderV1(b *testing.B) {
	src := bytes.ReplaceAll(testfile, []byte{'\t'}, []byte("    "))
	srcLines := bytes.Split(src, []byte{'\n'})

	b.ResetTimer()
	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		sourceRender(srcLines, 90, 49, 90, 100, 100, []int{})
	}
}
