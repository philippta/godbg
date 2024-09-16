package source

import (
	"bytes"
	_ "embed"
	"os"
	"testing"
)

//go:embed testdata/source.txt
var testSourceFile []byte

func TestRenderLines(t *testing.T) {
	lines := bytes.Split(testSourceFile, []byte{'\n'})
	os.Stdout.Write(RenderLines(lines, 100, 80, 10, 105, []int{107}))
}

func BenchmarkRenderLines(b *testing.B) {
	lines := bytes.Split(testSourceFile, []byte{'\n'})

	b.ResetTimer()
	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		RenderLines(lines, 100, 80, 30, 105, nil)
	}
}
