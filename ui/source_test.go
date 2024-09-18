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

func TestRuneCount(t *testing.T) {
	fmt.Println(len([]byte("\033[93m=> ")))
	fmt.Println(utf8.RuneCount([]byte("\033[93m=> ")))
}

func BenchmarkSourceReader(b *testing.B) {
	src := bytes.ReplaceAll(testfile, []byte{'\t'}, []byte("    "))
	srcLines := bytes.Split(src, []byte{'\n'})

	b.ResetTimer()
	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		sourceRender(srcLines, 90, 49, 90, 100, 100, []int{})
	}
}

func BenchmarkSourceReaderV2(b *testing.B) {
	src := bytes.ReplaceAll(testfile, []byte{'\t'}, []byte("    "))
	srcLines := bytes.Split(src, []byte{'\n'})

	b.ResetTimer()
	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		sourceRenderV2(srcLines, 90, 49, 90, 100, 100, []int{})
	}
}

func BenchmarkSourceReaderV3(b *testing.B) {
	src := bytes.ReplaceAll(testfile, []byte{'\t'}, []byte("    "))
	srcLines := bytes.Split(src, []byte{'\n'})

	b.ResetTimer()
	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		sourceRenderV3(srcLines, 90, 49, 90, 100, 100, []int{})
	}
}
