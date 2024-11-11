package ui

import (
	"strings"

	"github.com/junegunn/fzf/src/util"
	"github.com/philippta/godbg/debug"
	"github.com/philippta/godbg/frame"
	"github.com/philippta/godbg/fuzzy"
)

type Files struct {
	Size          Size
	Dir           string
	Search        string
	SearchCursor  int
	FileCursor    int
	FileNames     []util.Chars
	FilteredFiles []string
}

func (f *Files) LoadFiles() {
	f.FileNames = fuzzy.FindFiles(f.Dir)
	f.FilterFiles()
}

func (f *Files) Resize(w, h int) {
	f.Size.Width, f.Size.Height = w, h
}

func (f *Files) CursorPosition() (y, x int) {
	y = f.Size.Height - 1
	x = f.SearchCursor + 5
	return
}

func (f *Files) Reset() {
	f.FileCursor = 0
	f.SearchCursor = 0
	f.Search = ""
	f.FilterFiles()
}

func (f *Files) HandleInput(key rune, more []rune) {
	switch key {
	case '\t':
	case 127: // DEL
		f.Search = f.Search[:max(0, len(f.Search)-1)]
		f.SearchCursor = max(0, f.SearchCursor-1)
	case 27: // ESC
		debug.Logf("%v", more)
		if len(more) != 2 {
			break
		}
		if more[0] == 91 { // Arrow
			switch more[1] {
			case 65: // Up
				f.FileCursor = min(f.FileCursor+1, len(f.FilteredFiles)-1)
			case 66: // Down
				f.FileCursor = max(0, f.FileCursor-1)
			case 67: // Right
				f.SearchCursor = min(f.SearchCursor+1, len(f.Search))
			case 68: // Left
				f.SearchCursor = max(0, f.SearchCursor-1)
			}
		}
	default:
		f.Search += string(key)
		f.SearchCursor++
	}

	f.FilterFiles()
}

func (f *Files) FilterFiles() {
	f.FilteredFiles = fuzzy.Match(f.FileNames, f.Search)
	if len(f.FilteredFiles) > f.Size.Height-4 {
		f.FilteredFiles = f.FilteredFiles[:f.Size.Height-4]
	}
}

func (f *Files) RenderFrame(text, colors *frame.Frame, offsetY, offsetX int) {
	y := offsetY
	x := offsetX

	text.FillSpaceRegion(offsetY, offsetX, f.Size.Width, f.Size.Height)
	colors.FillZeroesRegion(offsetY, offsetX, f.Size.Width, f.Size.Height)

	// Horizontal Lines
	for i := 1; i < f.Size.Width-1; i++ {
		text.WriteAt(y+0, x+i, '─')
		text.WriteAt(y+f.Size.Height-1, x+i, '─')
		text.WriteAt(y+f.Size.Height-3, x+i, '─')
	}
	colors.SetColor(y+0, x, f.Size.Width, frame.ColorFGBlue)
	colors.SetColor(y+f.Size.Height-1, x, f.Size.Width, frame.ColorFGBlue)
	colors.SetColor(y+f.Size.Height-3, x, f.Size.Width, frame.ColorFGBlue)

	// Veritcal Lines
	for i := 1; i < f.Size.Height-1; i++ {
		text.WriteAt(y+i, x+0, '│')
		text.WriteAt(y+i, x+f.Size.Width-1, '│')
		colors.SetColor(y+i, x, 1, frame.ColorFGBlue)
		colors.SetColor(y+i, x+f.Size.Width-1, 1, frame.ColorFGBlue)
	}

	// Corners
	text.WriteAt(y+0, x+0, '┌')
	text.WriteAt(y+0, x+f.Size.Width-1, '┐')
	text.WriteAt(y+f.Size.Height-1, x+0, '└')
	text.WriteAt(y+f.Size.Height-1, x+f.Size.Width-1, '┘')

	// Search Box
	text.WriteAt(y+f.Size.Height-3, x+0, '├')
	text.WriteAt(y+f.Size.Height-3, x+f.Size.Width-1, '┤')

	// Write search term
	text.WriteString(y+f.Size.Height-2, x+2, "> "+f.Search)
	colors.SetColor(y+f.Size.Height-2, x+2, 1, frame.ColorFGGreen)

	debug.Logf("DIR:%v", f.Dir)

	listheight := f.Size.Height - 4
	for i, name := range f.FilteredFiles {
		if f.FileCursor == i {
			text.WriteString(y+listheight-i, x+2, "> "+strings.TrimPrefix(name, f.Dir+"/"))
			colors.SetColor(y+listheight-i, x+2, 1, frame.ColorFGGreen)
		} else {
			text.WriteString(y+listheight-i, x+4, strings.TrimPrefix(name, f.Dir+"/"))
		}
	}

}
