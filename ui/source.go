package ui

import (
	"bytes"
	"os"

	"github.com/go-delve/delve/service/api"
	"github.com/philippta/godbg/debug"
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
		debug.Logf("%v", dbg.CreateFileBreakpoint(s.File.Name, s.Cursors.Line+1))
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

func (s *Source) RenderFrame(text, colors *frame.Frame, offsetY, offsetX int) {
	if len(s.File.Lines) == 0 {
		return
	}

	const iotaBufCap = 5
	var (
		breakpoints  = fileBreakpoints(s.Breakpoints, s.File.Name)
		iotaBuf      = [iotaBufCap]byte{' ', ' ', ' ', ' ', ' '}
		lineNumWidth = numDigits(len(s.File.Lines))
		lineEnd      = min(s.File.LineOffset+s.Size.Height, len(s.File.Lines))
	)

	for i := s.File.LineOffset; i < lineEnd; i++ {
		y := i - s.File.LineOffset + offsetY
		x := offsetX

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
		x = text.WriteString(y, x, string(iotaBuf[iotaBufCap-lineNumWidth:]))
		x = text.WriteString(y, x, "  ")

		line := s.File.Lines[i]
		x = text.WriteString(y, x, string(line[:min(len(line), s.Size.Width-x)]))
	}

	for i := s.File.LineOffset; i < lineEnd; i++ {
		y := i - s.File.LineOffset + offsetY
		x := offsetX

		if !s.Focused {
			colors.SetColor(y, x, 3, frame.ColorFGBlack)
		} else if i == s.Cursors.Line {
			colors.SetColor(y, x, 3, frame.ColorFGGreen)
		} else if i == s.Cursors.PC {
			colors.SetColor(y, x, 3, frame.ColorFGYellow)
		} else {
			colors.SetColor(y, x, 3, frame.ColorReset)
		}

		if contains(breakpoints, i) {
			colors.SetColor(y, x+3, 1, frame.ColorFGRed)
		}

		colors.SetColor(y, x+5, lineNumWidth+1, frame.ColorFGBlue)

		if i == s.Cursors.Line {
			offset := x + lineNumWidth + 6
			colors.SetColor(y, offset, s.Size.Width-offset, frame.ColorFGWhite)
		}
	}
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
