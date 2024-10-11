package ui

import (
	"bytes"
	"os"
	"strings"

	"github.com/go-delve/delve/service/api"
	"github.com/philippta/godbg/dlv"
)

type Source struct {
	Width       int
	Height      int
	File        string
	Lines       [][]byte
	LineStart   int
	LineCursor  int
	PCCursor    int
	Breakpoints []*api.Breakpoint
}

func (s *Source) Resize(w, h int) {
	s.Width, s.Height = w, h
}

func (s *Source) Render(active bool) ([]string, []int) {
	breakpoints := fileBreakpoints(s.Breakpoints, s.File)
	return sourceRender(
		s.Lines,
		s.Width,
		s.Height,
		s.LineStart,
		s.PCCursor,
		s.LineCursor,
		breakpoints,
		active,
	)
}

func (s *Source) MoveUp(viewHeight int) {
	s.LineCursor = max(0, s.LineCursor-1)
	s.AlignCursor(viewHeight)
}

func (s *Source) MoveDown(viewHeight int) {
	s.LineCursor = min(s.LineCursor+1, len(s.Lines)-1)
	s.AlignCursor(viewHeight)
}

func (s *Source) AlignCursor(viewHeight int) {
	if s.LineCursor < s.LineStart+2 {
		s.LineStart = max(0, s.LineCursor-2)
	}
	if s.LineCursor > s.LineStart+viewHeight-3 {
		s.LineStart = max(0, min(s.LineCursor-viewHeight+3, len(s.Lines)-viewHeight))
	}
}

func (s *Source) CenterCursor(viewHeight int) {
	s.LineStart = max(0, min(s.LineCursor-viewHeight/2, len(s.Lines)-viewHeight))
}

func (s *Source) InitBreakpoints(dbg *dlv.Debugger) {
	s.Breakpoints = dbg.Breakpoints()
}

func (s *Source) ToggleBreakpoint(dbg *dlv.Debugger) {
	var activeBP *api.Breakpoint
	for _, bp := range s.Breakpoints {
		if bp.File == s.File && bp.Line == s.LineCursor+1 {
			activeBP = bp
			break
		}
	}

	if activeBP == nil {
		dbg.CreateFileBreakpoint(s.File, s.LineCursor+1)
	} else {
		dbg.ClearBreakpoint(activeBP.ID)
	}

	s.Breakpoints = dbg.Breakpoints()
}

func (s *Source) LoadLocation(dbg *dlv.Debugger, viewHeight int) (changed bool) {
	path, line := dbg.Location()
	if path == "" {
		return
	}

	changed = s.File != path
	s.PCCursor = line - 1
	s.LineCursor = line - 1

	if s.File != path {
		src, err := os.ReadFile(path)
		must(err)

		src = bytes.ReplaceAll(src, []byte{'\t'}, []byte("    "))
		if src[len(src)-1] == '\n' {
			src = src[:len(src)-1]
		}

		s.Lines = bytes.Split(src, []byte{'\n'})
		s.File = path
		s.CenterCursor(viewHeight)
	} else {
		s.AlignCursor(viewHeight)
	}
	return changed
}

func sourceRender(lines [][]byte, width, height, lineStart, pcCursor, lineCursor int, breakpoints []int, active bool) ([]string, []int) {
	if len(lines) == 0 {
		return []string{}, []int{}
	}

	const iotaBufCap = 5

	var (
		iotaBuf      = [iotaBufCap]byte{}
		padBuf       = [iotaBufCap]byte{}
		lineNumWidth = numDigits(len(lines))
		lineEnd      = min(lineStart+height, len(lines))
		lens         = make([]int, 0, height)
	)

	for i := 0; i < iotaBufCap; i++ {
		padBuf[i] = ' '
	}

	var buf strings.Builder
	buf.Grow(width*height + 1000)

	for i := lineStart; i < lineEnd; i++ {
		// line len: 5
		if i == pcCursor {
			if active {
				buf.WriteString("\033[93m=> ")
			} else {
				buf.WriteString("\033[90m=> ")
			}
		} else if i == lineCursor {
			if active {
				buf.WriteString("\033[32m=> ")
			} else {
				buf.WriteString("\033[90m=> ")
			}
		} else {
			buf.WriteString("\033[0m   ")
		}

		// line len: 2
		if contains(breakpoints, i) {
			buf.WriteString("\033[91m* ")
		} else {
			buf.WriteString("  ")
		}

		paddedItoa(iotaBuf[:], i+1)

		// line len: lineNumWidth
		buf.WriteString("\033[34m")
		buf.Write(iotaBuf[iotaBufCap-lineNumWidth:])
		buf.WriteString(": ")

		ll := len(lines[i])

		// line len: endc
		if i == lineCursor && active {
			buf.WriteString("\033[97m")
		} else {
			buf.WriteString("\033[37m")
		}
		buf.Write(lines[i])

		// line len: 1 (ignored)
		if i < lineEnd-1 {
			buf.WriteByte('\n')
		}

		lens = append(lens, 5+2+ll+lineNumWidth)
	}

	return strings.Split(buf.String(), "\n"), lens
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
