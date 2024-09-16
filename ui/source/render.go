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
	bp     []int
	lines  [][]byte
	path   string
}

func (v *View) LoadFile(path string, pc int) {
	if path == v.path {
		v.pc = pc
		v.ScrollTo(pc)
		return
	}

	src, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	v.lines = bytes.Split(src, []byte{'\n'})
	v.pc = pc
	v.ScrollTo(pc)
}

func (v *View) Resize(width, height int) {
	v.width = width
	v.height = height
}

func (v *View) ScrollBy(n int) {
	v.start = max(0, min(v.start+n, len(v.lines)-v.height-1))
}

func (v *View) ScrollTo(n int) {
	v.start = max(0, min(n-v.height/2, len(v.lines)-v.height-1))
}

func (v *View) Render() string {
	return string(RenderLines(v.lines, v.start, v.width, v.height, v.pc, v.bp))
}

func RenderLines(lines [][]byte, start, width, height, pc int, bp []int) []byte {
	if len(lines) == 0 {
		return []byte{}
	}

	const iotaBufCap = 10

	var (
		iotaBuf      = [iotaBufCap]byte{}
		padBuf       = [iotaBufCap]byte{}
		lineNumWidth = numDigits(len(lines))
		srcWidth     = width - lineNumWidth
		linecount    = 1
	)

	for i := 0; i < iotaBufCap; i++ {
		padBuf[i] = ' '
	}

	buf := make([]byte, 0, width*height)

	for i := start; i < min(start+height, len(lines)); i++ {
		if i == pc-1 {
			buf = append(buf, '=', '>', ' ')
		} else if slices.Contains(bp, i) {
			buf = append(buf, '*', ' ', ' ')
		} else {
			buf = append(buf, ' ', ' ', ' ')
		}

		paddedItoa(iotaBuf[:], i+1)

		buf = append(buf, iotaBuf[iotaBufCap-lineNumWidth:]...)
		buf = append(buf, ' ')

		ll := len(lines[i])

		endc := min(srcWidth, ll)
		buf = append(buf, lines[i][:endc]...)
		buf = append(buf, '\n')

		if endc < ll {
			buf = append(buf, padBuf[iotaBufCap-lineNumWidth-4:]...)
			buf = append(buf, lines[i][endc:min(endc+srcWidth, ll)]...)
			buf = append(buf, '\n')
			endc = min(endc+srcWidth, ll)
			linecount++

			if linecount > height {
				break
			}
		}

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
