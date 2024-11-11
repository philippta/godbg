package frame

import (
	"bytes"
	"fmt"
	"os"
	"strings"
)

const (
	ColorReset rune = iota
	ColorFGBlack
	ColorFGRed
	ColorFGGreen
	ColorFGYellow
	ColorFGBlue
	ColorFGWhite

	ColorCount
)

var Colors = [][]byte{
	[]byte("\033[m"),
	[]byte("\033[38;90m"), // FG Black (Grey)
	[]byte("\033[38;91m"), // FG Red
	[]byte("\033[38;92m"), // FG Green
	[]byte("\033[38;93m"), // FG Yellow
	[]byte("\033[38;94m"), // FG Blue
	[]byte("\033[38;97m"), // FG White
}

const numSpaces = 1024

var spaces = strings.Repeat(" ", numSpaces)
var zeroes = make([]rune, numSpaces)

func New(rows, cols int) *Frame {
	return &Frame{
		Rows: rows,
		Cols: cols,
		Buf:  make([]rune, rows*cols),
	}
}

type Frame struct {
	Rows int
	Cols int
	Buf  []rune
}

func (f *Frame) FillSpace() {
	for cursor := 0; cursor < len(f.Buf); cursor += len(spaces) {
		copy(f.Buf[cursor:], []rune(spaces))
	}
}

func (f *Frame) FillSpaceRegion(y, x, w, h int) {
	for row := y; row < y+h; row++ {
		startIdx := row*f.Cols + x
		copy(f.Buf[startIdx:startIdx+w], []rune(spaces[:w]))
	}
}

func (f *Frame) FillZeroesRegion(y, x, w, h int) {
	for row := y; row < y+h; row++ {
		startIdx := row*f.Cols + x
		copy(f.Buf[startIdx:startIdx+w], zeroes[:w])
	}
}

func (f *Frame) Fill(b rune) {
	if b == ' ' {
		f.FillSpace()
		return
	}
	for i := 0; i < len(f.Buf); i++ {
		f.Buf[i] = b
	}
}

func (f *Frame) CopyFrom(y, x int, src *Frame) {
	for i := 0; i < src.Rows; i++ {
		srcRowStart := i * src.Cols
		dstRowStart := (i+y)*f.Cols + x
		copy(f.Buf[dstRowStart:dstRowStart+src.Cols], src.Buf[srcRowStart:srcRowStart+src.Cols])
	}
}

func (f *Frame) WriteAt(y, x int, b rune) {
	f.Buf[f.Cols*y+x] = b
}

func (f *Frame) WriteString(y, x int, s string) int {
	if x > f.Cols-1 {
		return x
	}
	offset := f.Cols*y + x
	length := min(f.Cols-x, len(s))
	end := offset + length
	copy(f.Buf[offset:end], []rune(s))
	return length + x
}

func (f *Frame) SetColor(y, x, width int, color rune) {
	for w := range width {
		f.Buf[f.Cols*y+x+w] = color
	}
}

func (f *Frame) Print(out *os.File) {
	for i := 0; i < f.Rows; i++ {
		out.WriteString(string(f.Buf[i*f.Cols : i*f.Cols+f.Cols]))
		out.Write([]byte{'\n'})
	}
}

func (f *Frame) PrintColored(out *os.File, colors *Frame) {
	var buf bytes.Buffer
	buf.Grow(f.Rows * f.Cols * 2)

	lastColor := rune(ColorCount)
	for i := range f.Buf {
		if lastColor != colors.Buf[i] {
			color := colors.Buf[i]
			buf.Write(Colors[color])
			lastColor = color
		}
		buf.WriteRune(f.Buf[i])
	}

	os.Stdout.Write(buf.Bytes())
}

func (f *Frame) PrintDebug(out *os.File) {
	for i := 0; i < f.Rows; i++ {
		out.WriteString(fmt.Sprintf("%x\n", f.Buf[i*f.Cols:i*f.Cols+f.Cols]))
	}
}

func (f *Frame) PrintLinesColored(out *os.File, colors *Frame) {
	lastColor := rune(ColorCount)
	for y := range f.Rows {
		for x := range f.Cols {
			if lastColor != colors.Buf[f.Cols*y+x] {
				color := colors.Buf[f.Cols*y+x]
				os.Stdout.Write(Colors[color])
				lastColor = color
			}
			os.Stdout.WriteString(string(f.Buf[f.Cols*y+x]))
		}
		os.Stdout.Write([]byte{'\n'})
	}
}
