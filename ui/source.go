package ui

import (
	"bytes"
	"os"
	"slices"
)

func sourceLoadFile(path string) [][]byte {
	src, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	src = bytes.ReplaceAll(src, []byte{'\t'}, []byte("    "))
	return bytes.Split(src, []byte{'\n'})
}

func sourceRender(lines [][]byte, width, height, lineStart, pcCursor, lineCursor int, breakpoints []int) string {
	if len(lines) == 0 {
		return ""
	}

	const iotaBufCap = 10

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
		buf = append(buf, "\033[0m"...)

		if i == pcCursor {
			buf = append(buf, "\033[93m"...)
			buf = append(buf, "=> "...)
		} else if i == lineCursor {
			buf = append(buf, "\033[32m"...)
			buf = append(buf, "=> "...)
		} else {
			buf = append(buf, "   "...)
		}

		if slices.Contains(breakpoints, i) {
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

		if i == pcCursor {
			buf = append(buf, "\033[97m"...)
		} else {
			buf = append(buf, "\033[37m"...)
		}
		buf = append(buf, lines[i][:endc]...)
		buf = append(buf, '\n')
	}

	return string(buf[:len(buf)-1])
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
