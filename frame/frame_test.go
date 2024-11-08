package frame_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/philippta/godbg/frame"
)

func TestWriteString(t *testing.T) {
	var n int
	src := frame.New(3, 20)
	src.Fill('.')
	n = src.WriteString(1, n, "hello ")
	n = src.WriteString(1, n, "world ")
	n = src.WriteString(1, n, "whats")
	n = src.WriteString(1, n, "up")
	src.Print(os.Stdout)
	fmt.Println(n)
}

func TestColoredFrame(t *testing.T) {
	colors := frame.New(43, 112)
	for i := 0; i < len(colors.Buf); i++ {
		colors.Buf[i] = byte(i) / 3 % frame.ColorCount
	}

	text := frame.New(43, 112)
	text.Fill('@')

	lastColor := byte(frame.ColorCount)
	for i := range text.Buf {
		if lastColor != colors.Buf[i] {
			os.Stdout.Write(frame.Colors[colors.Buf[i]])
			lastColor = colors.Buf[i]
		}
		os.Stdout.Write([]byte{text.Buf[i]})
	}
}

func BenchmarkColoredFrame(b *testing.B) {
	colors := frame.New(100, 100)
	for i := 0; i < len(colors.Buf); i++ {
		colors.Buf[i] = byte(i) % frame.ColorCount
	}

	text := frame.New(100, 100)
	text.Fill('@')

	var buf bytes.Buffer
	buf.Grow(100 * 100 * 10)

	b.SetBytes(100 * 100)
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		lastColor := byte(frame.ColorCount)
		for i := range text.Buf {
			if lastColor != colors.Buf[i] {
				buf.Write(frame.Colors[colors.Buf[i]])
				lastColor = colors.Buf[i]
			}
			buf.WriteByte(text.Buf[i])
		}
	}
}

func TestFillSpace(t *testing.T) {
	src := frame.New(9, 54)
	src.FillSpace()
	src.PrintDebug(os.Stdout)
}

func BenchmarkFillSpace(b *testing.B) {
	src := frame.New(512, 512)

	b.SetBytes(int64(len(src.Buf)))
	for n := 0; n < b.N; n++ {
		src.FillSpace()
	}
}

func TestWriteSlice(t *testing.T) {
	src := frame.New(3, 3)
	src.WriteLine(0, []byte("AAA"))
	src.WriteLine(1, []byte("BBB"))
	src.WriteLine(2, []byte("CCC"))
	src.Print(os.Stdout)
}

func TestCopyTo(t *testing.T) {
	src := frame.New(17, 17)
	src.Fill('A')

	dst := frame.New(33, 33)
	dst.Fill('-')
	dst.CopyFrom(7, 9, src)
	dst.Print(os.Stdout)
	println()
}

func BenchmarkCopyTo(b *testing.B) {
	src := frame.New(512, 512)
	src.Fill('A')
	dst := frame.New(1024, 1024)

	b.SetBytes(512 * 512)
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		dst.CopyFrom(256, 256, src)
	}
}
