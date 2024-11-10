package term

import (
	"fmt"
	"os"
	"strings"
)

var (
	AltScreen       = []byte("\033[?1049h")
	ExitAltScreen   = []byte("\033[?1049l")
	ClearScreen     = []byte("\033[2J")
	ClearScreenFull = []byte("\033[H")
	ClearLine       = []byte("\033[2K")
	ResetCursor     = []byte("\033[1;1H")
	ShowCursor      = []byte("\033[?25h")
	HideCursor      = []byte("\033[?25l")
)

func Clear(w *os.File, width, height int) {
	line := strings.Repeat(" ", width)

	w.Write(ResetCursor)
	for i := 0; i < height; i++ {
		w.WriteString(line)
	}
	w.Write(ResetCursor)
}

func PositionCursor(y, x int) []byte {
	return []byte(fmt.Sprintf("\033[%d;%dH", y, x))
}
