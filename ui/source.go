package ui

import (
	"bytes"
	"strings"
)

func sourceRender(lines [][]byte, width, height, lineStart, pcCursor, lineCursor int, breakpoints []int) []byte {
	if len(lines) == 0 {
		return []byte{}
	}

	const iotaBufCap = 5

	var (
		iotaBuf      = [iotaBufCap]byte{}
		padBuf       = [iotaBufCap]byte{}
		lineNumWidth = numDigits(len(lines))
		srcWidth     = width - lineNumWidth - 4 // pc marker + ":"
	)

	for i := 0; i < iotaBufCap; i++ {
		padBuf[i] = ' '
	}

	buf := make([]byte, 0, width*height)

	for i := lineStart; i < min(lineStart+height, len(lines)); i++ {
		if i == pcCursor {
			buf = append(buf, "\033[93m=> "...)
		} else if i == lineCursor {
			buf = append(buf, "\033[32m-> "...)
		} else {
			buf = append(buf, "\033[0m   "...)
		}

		if contains(breakpoints, i) {
			buf = append(buf, "\033[91m* "...)
		} else {
			buf = append(buf, "  "...)
		}

		paddedItoa(iotaBuf[:], i+1)

		buf = append(buf, "\033[34m"...)
		buf = append(buf, iotaBuf[iotaBufCap-lineNumWidth:]...)
		buf = append(buf, ": "...)

		ll := len(lines[i])
		endc := min(srcWidth, ll)

		if i == pcCursor {
			buf = append(buf, "\033[97m"...)
		} else {
			buf = append(buf, "\033[37m"...)
		}
		buf = append(buf, lines[i][:endc]...)
		buf = append(buf, '\n')
	}

	return buf[:len(buf)-1]
}

func sourceRenderV2(lines [][]byte, width, height, lineStart, pcCursor, lineCursor int, breakpoints []int) [][]byte {
	if len(lines) == 0 {
		return [][]byte{}
	}

	const iotaBufCap = 5

	var (
		iotaBuf      = [iotaBufCap]byte{}
		padBuf       = [iotaBufCap]byte{}
		lineNumWidth = numDigits(len(lines))
		srcWidth     = width - lineNumWidth - 4 // pc marker + ":"
	)

	for i := 0; i < iotaBufCap; i++ {
		padBuf[i] = ' '
	}

	buf := make([]byte, 0, width*height)

	for i := lineStart; i < min(lineStart+height, len(lines)); i++ {
		if i == pcCursor {
			buf = append(buf, "\033[93m=> "...)
		} else if i == lineCursor {
			buf = append(buf, "\033[32m=> "...)
		} else {
			buf = append(buf, "\033[0m   "...)
		}

		if contains(breakpoints, i) {
			buf = append(buf, "\033[91m* "...)
		} else {
			buf = append(buf, "  "...)
		}

		paddedItoa(iotaBuf[:], i+1)

		buf = append(buf, "\033[34m"...)
		buf = append(buf, iotaBuf[iotaBufCap-lineNumWidth:]...)
		buf = append(buf, ": "...)

		ll := len(lines[i])
		endc := min(srcWidth, ll)

		if i == pcCursor {
			buf = append(buf, "\033[97m"...)
		} else {
			buf = append(buf, "\033[37m"...)
		}
		buf = append(buf, lines[i][:endc]...)
		buf = append(buf, '\n')
	}

	return bytes.Split(buf[:len(buf)-1], []byte{'\n'})
}

func sourceRenderV3(lines [][]byte, width, height, lineStart, pcCursor, lineCursor int, breakpoints []int) ([]string, []int) {
	if len(lines) == 0 {
		return []string{}, []int{}
	}

	const iotaBufCap = 5

	var (
		iotaBuf      = [iotaBufCap]byte{}
		padBuf       = [iotaBufCap]byte{}
		lineNumWidth = numDigits(len(lines))
		srcWidth     = width - lineNumWidth - 4 // pc marker + ":"
	)

	for i := 0; i < iotaBufCap; i++ {
		padBuf[i] = ' '
	}

	buf := make([]byte, 0, width*height)
	lens := make([]int, 0, height)

	for i := lineStart; i < min(lineStart+height, len(lines)); i++ {
		// line len: 5
		if i == pcCursor {
			buf = append(buf, "\033[93m=> "...)
		} else if i == lineCursor {
			buf = append(buf, "\033[32m=> "...)
		} else {
			buf = append(buf, "\033[0m   "...)
		}

		// line len: 2
		if contains(breakpoints, i) {
			buf = append(buf, "\033[91m* "...)
		} else {
			buf = append(buf, "  "...)
		}

		paddedItoa(iotaBuf[:], i+1)

		// line len: lineNumWidth
		buf = append(buf, "\033[34m"...)
		buf = append(buf, iotaBuf[iotaBufCap-lineNumWidth:]...)
		buf = append(buf, ": "...)

		ll := len(lines[i])
		endc := min(srcWidth, ll)

		// line len: endc
		if i == pcCursor {
			buf = append(buf, "\033[97m"...)
		} else {
			buf = append(buf, "\033[37m"...)
		}
		buf = append(buf, lines[i][:endc]...)

		// line len: 1 (ignored)
		buf = append(buf, '\n')

		lens = append(lens, 5+2+endc+lineNumWidth)
	}

	return strings.Split(string(buf[:len(buf)-1]), "\n"), lens
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
