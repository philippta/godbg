package ui

import (
	"github.com/philippta/godbg/debug"
	"github.com/philippta/godbg/frame"
)

type Files struct {
	Focused bool
	Size    Size
	Search  string
	Cursor  int
}

func (f *Files) Resize(w, h int) {
	f.Size.Width, f.Size.Height = w, h
}

func (f *Files) CursorPosition() (y, x int) {
	y = f.Size.Height - 1
	x = f.Cursor + 3
	return
}

func (f *Files) HandleInput(key rune, more []rune) {
	switch key {
	case 127: // DEL
		f.Search = f.Search[:max(0, len(f.Search)-1)]
		f.Cursor = max(0, f.Cursor-1)
	case 27: // ESC
		debug.Logf("%v", more)
		if len(more) == 2 {
			first, second := more[0], more[1]
			if first == 91 { // Arrow
				if second == 68 { // Left
					f.Cursor = max(0, f.Cursor-1)
				} else if second == 67 { // Right
					f.Cursor = min(f.Cursor+1, len(f.Search))
				}
			}

		}
		// Ignore
	default:
		f.Search += string(key)
		f.Cursor++
	}
}

func (f *Files) RenderFrame() (*frame.Frame, *frame.Frame) {
	text := frame.New(f.Size.Height, f.Size.Width)
	text.FillSpace()

	// Horizontal Lines
	for i := 1; i < f.Size.Width-1; i++ {
		text.WriteAt(0, i, '─')
		text.WriteAt(f.Size.Height-1, i, '─')
		text.WriteAt(f.Size.Height-3, i, '─')
	}

	// Veritcal Lines
	for i := 1; i < f.Size.Height-1; i++ {
		text.WriteAt(i, 0, '│')
		text.WriteAt(i, f.Size.Width-1, '│')
	}

	// Corners
	text.WriteAt(0, 0, '┌')
	text.WriteAt(0, f.Size.Width-1, '┐')
	text.WriteAt(f.Size.Height-1, 0, '└')
	text.WriteAt(f.Size.Height-1, f.Size.Width-1, '┘')

	// Search Box
	text.WriteAt(f.Size.Height-3, 0, '├')
	text.WriteAt(f.Size.Height-3, f.Size.Width-1, '┤')

	// Write search term
	text.WriteString(f.Size.Height-2, 2, f.Search)

	colors := frame.New(f.Size.Height, f.Size.Width)
	return text, colors
}
