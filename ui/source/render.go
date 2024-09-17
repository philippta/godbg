package source

import (
	"bytes"
	"os"
	"slices"
)

type View struct {
	width  int
	height int
	start  int
	pc     int
	cursor int
	bp     []int
	lines  [][]byte
	path   string
}

func (v *View) LoadFile(path string, pc int, bp []int) {
	if path == v.path {
		v.pc = pc
		v.cursor = pc
		v.ScrollTo(pc)
		return
	}

	src, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	src = bytes.ReplaceAll(src, []byte{'\t'}, []byte("    "))
	v.lines = bytes.Split(src, []byte{'\n'})
	v.pc = pc
	v.cursor = pc
	v.ScrollTo(pc)
	v.path = path
	v.bp = bp
}

func (v *View) SetBreakpoints(bp []int) {
	v.bp = bp
}

func (v *View) Location() (string, int) {
	return v.path, v.cursor
}

func (v *View) Resize(width, height int) {
	v.width = width
	v.height = height
}

func (v *View) ScrollBy(n int) {
	v.cursor = max(1, min(v.cursor+n, len(v.lines)-1))

	if n > 0 && v.cursor > v.start+v.height-3 {
		v.start = min(v.start+n, len(v.lines)-1-v.height)
	}
	if n < 0 && v.cursor < v.start+3 {
		v.start = max(0, v.start+n)
	}
}

func (v *View) ScrollTo(n int) {
	v.cursor = max(1, min(n, len(v.lines)-1))
	v.start = max(0, min(v.cursor-v.height/2, len(v.lines)-1-v.height))
}

func (v *View) Render() string {
	return string(RenderLines(v.lines, v.start, v.width, v.height, v.pc, v.cursor, v.bp))
}

func RenderLines(lines [][]byte, start, width, height, pc, cursor int, bp []int) []byte {
	if len(lines) == 0 {
		return []byte{}
	}

	const iotaBufCap = 10

	var (
		iotaBuf      = [iotaBufCap]byte{}
		padBuf       = [iotaBufCap]byte{}
		lineNumWidth = numDigits(len(lines))
		srcWidth     = width - lineNumWidth - 4 // pc marker + ":"
		linecount    = 1
	)

	for i := 0; i < iotaBufCap; i++ {
		padBuf[i] = ' '
	}

	buf := make([]byte, 0, width*height)

	for i := start; i < min(start+height, len(lines)-1); i++ {
		buf = append(buf, "\033[0m"...)

		if i == pc-1 {
			buf = append(buf, "\033[93m"...)
			buf = append(buf, "=> "...)
		} else if i == cursor-1 {
			buf = append(buf, "\033[32m"...)
			buf = append(buf, "=> "...)
		} else {
			buf = append(buf, "   "...)
		}

		if slices.Contains(bp, i+1) {
			buf = append(buf, "\033[91m"...)
			buf = append(buf, "* "...)
		} else {
			buf = append(buf, "  "...)
		}

		paddedItoa(iotaBuf[:], i+1)

		buf = append(buf, "\033[34m"...)
		buf = append(buf, iotaBuf[iotaBufCap-lineNumWidth:]...)
		buf = append(buf, ':')
		buf = append(buf, ' ')

		ll := len(lines[i])
		endc := min(srcWidth, ll)

		if i == pc-1 {
			buf = append(buf, "\033[97m"...)
		} else {
			buf = append(buf, "\033[37m"...)
		}
		buf = append(buf, lines[i][:endc]...)
		buf = append(buf, '\n')

		linecount++
		if linecount > height {
			break
		}
	}
	return buf[:len(buf)-1]
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
