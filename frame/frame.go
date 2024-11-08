package frame

import (
	"bytes"
	"fmt"
	"os"
)

const (
	ColorReset byte = iota
	ColorFGBlack
	ColorFGRed
	ColorFGGreen
	ColorFGYellow
	ColorFGBlue

	ColorCount
)

var Colors = [][]byte{
	[]byte("\033[m"),
	[]byte("\033[38;90m"), // FG Black (Grey)
	[]byte("\033[38;91m"), // FG Red
	[]byte("\033[38;92m"), // FG Green
	[]byte("\033[38;93m"), // FG Yellow
	[]byte("\033[38;94m"), // FG Blue
}

const numSpaces = 1024

var spaces = bytes.Repeat([]byte{' '}, numSpaces)

func New(rows, cols int) *Frame {
	return &Frame{
		Rows: rows,
		Cols: cols,
		Buf:  make([]byte, rows*cols),
	}
}

type Frame struct {
	Rows int
	Cols int
	Buf  []byte
}

func (f *Frame) FillSpace() {
	cursor := 0

	for cursor < len(f.Buf) {
		copy(f.Buf[cursor:], spaces)
		cursor += len(spaces)
	}
	//
	// need := len(f.Buf)
	// rows := need / numSpaces
	//
	// for i := 0; i < rows; i++ {
	// 	copy(f.Buf[i*numSpaces:], spaces)
	// }
	//
	// rest := need % numSpaces
	// if rest > 0 {
	// 	copy(f.Buf[rows:], spaces[:rest])
	// }
}

func (f *Frame) Fill(b byte) {
	if b == ' ' {
		f.FillSpace()
		return
	}
	for i := 0; i < len(f.Buf); i++ {
		f.Buf[i] = b
	}
}

func (f *Frame) CopyFrom(x, y int, src *Frame) {
	for i := 0; i < src.Rows; i++ {
		srcRowStart := i * src.Cols
		dstRowStart := (i+y)*f.Cols + x
		copy(f.Buf[dstRowStart:dstRowStart+src.Cols], src.Buf[srcRowStart:srcRowStart+src.Cols])
	}
}

func (f *Frame) WriteAt(x, y int, b byte) {
	f.Buf[f.Cols*y+x] = b
}

func (f *Frame) WriteLine(y int, b []byte) {
	off := f.Cols * y
	end := off + min(f.Cols, len(b))
	copy(f.Buf[off:end], b)
}

func (f *Frame) WriteBytes(y, x int, b []byte) int {
	if x > f.Cols-1 {
		return x
	}
	offset := f.Cols*y + x
	length := min(f.Cols-x, len(b))
	end := offset + length
	copy(f.Buf[offset:end], b)
	return length + x
}

func (f *Frame) WriteString(y, x int, s string) int {
	if x > f.Cols-1 {
		return x
	}
	offset := f.Cols*y + x
	length := min(f.Cols-x, len(s))
	end := offset + length
	copy(f.Buf[offset:end], s)
	return length + x
}

func (f *Frame) SetColor(y, x, width int, color byte) {
	for w := range width {
		f.Buf[f.Cols*y+x+w] = color
	}
}

func (f *Frame) Print(out *os.File) {
	for i := 0; i < f.Rows; i++ {
		out.Write(f.Buf[i*f.Cols : i*f.Cols+f.Cols])
		out.Write([]byte{'\n'})
	}
}

func (f *Frame) PrintColored(out *os.File, colors *Frame) {
	var buf bytes.Buffer
	buf.Grow(f.Rows * f.Cols * 2)

	lastColor := byte(ColorCount)
	for i := range f.Buf {
		if lastColor != colors.Buf[i] {
			color := colors.Buf[i]
			buf.Write(Colors[color])
			lastColor = color
		}
		buf.WriteByte(f.Buf[i])
	}

	os.Stdout.Write(buf.Bytes())
}

func (f *Frame) PrintDebug(out *os.File) {
	for i := 0; i < f.Rows; i++ {
		out.Write([]byte(fmt.Sprintf("%x\n", f.Buf[i*f.Cols:i*f.Cols+f.Cols])))
	}
}

func (f *Frame) PrintLinesColored(out *os.File, colors *Frame) {
	lastColor := byte(ColorCount)
	for y := range f.Rows {
		for x := range f.Cols {
			if lastColor != colors.Buf[f.Cols*y+x] {
				color := colors.Buf[f.Cols*y+x]
				os.Stdout.Write(Colors[color])
				lastColor = color
			}
			os.Stdout.Write([]byte{f.Buf[f.Cols*y+x]})
		}
		os.Stdout.Write([]byte{'\n'})
	}
}
