package ui

import (
	"strings"
)

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
