package ui

import (
	"bytes"
	"os"
	"strings"

	"github.com/go-delve/delve/service/api"
	"github.com/philippta/godbg/dlv"
	"github.com/philippta/godbg/frame"
)

type Size struct {
	Width  int
	Height int
}

type Cursors struct {
	PC   int
	Line int
}

type File struct {
	Name       string
	Lines      [][]byte
	LineOffset int
}

type Source struct {
	Focused     bool
	Size        Size
	File        File
	Cursors     Cursors
	Breakpoints []*api.Breakpoint
}

func (s *Source) Resize(w, h int) {
	s.Size.Width, s.Size.Height = w, h
}

func (s *Source) MoveUp() {
	s.Cursors.Line = max(0, s.Cursors.Line-1)
	s.AlignCursor()
}

func (s *Source) MoveDown() {
	s.Cursors.Line = min(s.Cursors.Line+1, len(s.File.Lines)-1)
	s.AlignCursor()
}

func (s *Source) AlignCursor() {
	if s.Cursors.Line < s.File.LineOffset+2 {
		s.File.LineOffset = max(0, s.Cursors.Line-2)
	}
	if s.Cursors.Line > s.File.LineOffset+s.Size.Height-3 {
		s.File.LineOffset = max(0, min(s.Cursors.Line-s.Size.Height+3, len(s.File.Lines)-s.Size.Height))
	}
}

func (s *Source) CenterCursor() {
	s.File.LineOffset = max(0, min(s.Cursors.Line-s.Size.Height/2, len(s.File.Lines)-s.Size.Height))
}

func (s *Source) InitBreakpoints(dbg *dlv.Debugger) {
	s.Breakpoints = dbg.Breakpoints()
}

func (s *Source) ToggleBreakpoint(dbg *dlv.Debugger) {
	var activeBP *api.Breakpoint
	for _, bp := range s.Breakpoints {
		if bp.File == s.File.Name && bp.Line == s.Cursors.Line+1 {
			activeBP = bp
			break
		}
	}

	if activeBP == nil {
		dbg.CreateFileBreakpoint(s.File.Name, s.Cursors.Line+1)
	} else {
		dbg.ClearBreakpoint(activeBP.ID)
	}

	s.Breakpoints = dbg.Breakpoints()
}

func (s *Source) LoadLocation(file string, line int) {
	if file == "" {
		return
	}

	s.Cursors.PC = line - 1
	s.Cursors.Line = line - 1

	if s.File.Name != file {
		src, err := os.ReadFile(file)
		must(err)

		src = bytes.ReplaceAll(src, []byte{'\t'}, []byte("    "))
		if src[len(src)-1] == '\n' {
			src = src[:len(src)-1]
		}

		s.File.Lines = bytes.Split(src, []byte{'\n'})
		s.File.Name = file
		s.CenterCursor()
	} else {
		s.AlignCursor()
	}
}

func (s *Source) Render() ([]string, []int) {
	if len(s.File.Lines) == 0 {
		return []string{}, []int{}
	}

	const iotaBufCap = 5

	var (
		breakpoints  = fileBreakpoints(s.Breakpoints, s.File.Name)
		iotaBuf      = [iotaBufCap]byte{}
		padBuf       = [iotaBufCap]byte{}
		lineNumWidth = numDigits(len(s.File.Lines))
		lineEnd      = min(s.File.LineOffset+s.Size.Height, len(s.File.Lines))
		lens         = make([]int, 0, s.Size.Height)
	)

	for i := 0; i < iotaBufCap; i++ {
		padBuf[i] = ' '
	}

	var buf strings.Builder
	buf.Grow(s.Size.Width*s.Size.Height + 1000)

	for i := s.File.LineOffset; i < lineEnd; i++ {
		// line len: 5
		if i == s.Cursors.Line {
			if s.Focused {
				buf.WriteString("\033[38;92m=> ")
			} else {
				buf.WriteString("\033[38;90m=> ")
			}
		} else if i == s.Cursors.PC {
			if s.Focused {
				buf.WriteString("\033[38;93m=> ")
			} else {
				buf.WriteString("\033[38;90m=> ")
			}
		} else {
			buf.WriteString("\033[0m   ")
		}

		// line len: 2
		if contains(breakpoints, i) {
			buf.WriteString("\033[38;91m* ")
		} else {
			buf.WriteString("  ")
		}

		paddedItoa(iotaBuf[:], i+1)

		// line len: lineNumWidth
		buf.WriteString("\033[38;94m")
		buf.Write(iotaBuf[iotaBufCap-lineNumWidth:])
		buf.WriteString(": ")

		ll := len(s.File.Lines[i])

		// line len: endc
		if i == s.Cursors.Line && s.Focused {
			buf.WriteString("\033[97m")
		} else {
			buf.WriteString("\033[37m")
		}
		buf.Write(s.File.Lines[i])

		// line len: 1 (ignored)
		if i < lineEnd-1 {
			buf.WriteByte('\n')
		}

		lens = append(lens, 5+2+ll+lineNumWidth)
	}

	return strings.Split(buf.String(), "\n"), lens
}

func (s *Source) RenderFrame() (*frame.Frame, *frame.Frame) {
	text := frame.New(s.Size.Height, s.Size.Width)
	text.FillSpace()

	colors := frame.New(s.Size.Height, s.Size.Width)

	if len(s.File.Lines) == 0 {
		return text, colors
	}

	const iotaBufCap = 5
	var (
		breakpoints  = fileBreakpoints(s.Breakpoints, s.File.Name)
		iotaBuf      = [iotaBufCap]byte{' ', ' ', ' ', ' ', ' '}
		lineNumWidth = numDigits(len(s.File.Lines))
		lineEnd      = min(s.File.LineOffset+s.Size.Height, len(s.File.Lines))
	)

	for i := s.File.LineOffset; i < lineEnd; i++ {
		y := i - s.File.LineOffset
		x := 0

		if i == s.Cursors.Line || i == s.Cursors.PC {
			x = text.WriteString(y, x, "=> ")
		} else {
			x = text.WriteString(y, x, "   ")
		}

		if contains(breakpoints, i) {
			x = text.WriteString(y, x, "* ")
		} else {
			x = text.WriteString(y, x, "  ")
		}

		paddedItoa(iotaBuf[:], i+1)
		x = text.WriteBytes(y, x, iotaBuf[iotaBufCap-lineNumWidth:])
		x = text.WriteString(y, x, ": ")
		x = text.WriteBytes(y, x, s.File.Lines[i])
	}

	for i := s.File.LineOffset; i < lineEnd; i++ {
		y := i - s.File.LineOffset

		if i == s.Cursors.Line {
			colors.SetColor(y, 0, 3, frame.ColorFGGreen)
		} else if i == s.Cursors.PC {
			colors.SetColor(y, 0, 3, frame.ColorFGYellow)
		} else {
			colors.SetColor(y, 0, 3, frame.ColorReset)
		}

		if contains(breakpoints, i) {
			colors.SetColor(y, 3, 1, frame.ColorFGRed)
		}

		colors.SetColor(y, 5, lineNumWidth+1, frame.ColorFGBlue)
	}

	return text, colors
}

func numDigits(i int) int {
	if i == 0 {
		return 1
	}
	count := 0
	for i != 0 {
		i /= 10
		count++
	}
	return count
}

func paddedItoa(buf []byte, n int) {
	for i := len(buf) - 1; i >= 0; i-- {
		if n > 0 {
			buf[i] = byte(n%10) + '0'
			n = n / 10
		} else {
			buf[i] = ' '
		}
	}

}

func contains(nn []int, n int) bool {
	for _, x := range nn {
		if x == n {
			return true
		}
	}
	return false
}

func fileBreakpoints(bps []*api.Breakpoint, file string) []int {
	var lineNums []int
	for _, bp := range bps {
		if bp.File == file {
			lineNums = append(lineNums, bp.Line-1)
		}
	}
	return lineNums
}
