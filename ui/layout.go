package ui

import (
	"bytes"
	"strings"
)

type block struct {
	lines []string
	lens  []int
}

func verticalSplit(width int, blocks ...block) string {
	maxLines := 0
	for i := range blocks {
		maxLines = max(maxLines, len(blocks[i].lines))
	}

	splitWidth := width / len(blocks)
	lineFill := strings.Repeat(".", splitWidth)

	var buf bytes.Buffer

	for i := 0; i < maxLines; i++ {
		for j := 0; j < len(blocks); j++ {
			if i >= len(blocks[j].lines) {
				continue
			}

			line := blocks[j].lines[i]
			lineLen := blocks[j].lens[i]

			maxFillLen := min(lineLen, splitWidth)
			buf.WriteString(line)
			buf.WriteString(lineFill[:splitWidth-maxFillLen])
		}
		buf.WriteByte('\n')
	}

	return string(buf.Bytes()[:buf.Len()-1])
}
