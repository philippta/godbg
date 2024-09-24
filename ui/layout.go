package ui

import (
	"strings"
)

type block struct {
	lines []string
	lens  []int
}

var lineFill = strings.Repeat(" ", 1000)

func verticalSplit(width int, height int, blocks ...block) string {
	var buf strings.Builder
	buf.Grow(width*height + 1000)

	splitWidth := width / len(blocks)

	for i := 0; i < height; i++ {
		for j := 0; j < len(blocks); j++ {
			if i >= len(blocks[j].lines) {
				buf.WriteString(lineFill[:splitWidth])
				continue
			}

			line := blocks[j].lines[i]
			lineLen := blocks[j].lens[i]

			numSpecialChars := len(line) - lineLen

			endc := min(lineLen+numSpecialChars, splitWidth+numSpecialChars)

			maxFillLen := min(lineLen, splitWidth)
			buf.WriteString(line[:endc])
			buf.WriteString(lineFill[:splitWidth-maxFillLen])
		}
		if i < height-1 {
			buf.WriteByte('\n')
		}
	}

	return buf.String()
}
