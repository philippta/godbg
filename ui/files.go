package ui

import (
	"bytes"
	"os"
	"strings"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/junegunn/fzf/src/util"
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
	Preview       []string
	PreviewCache  *lru.Cache[string, []string]
}

func (f *Files) LoadFiles() {
	f.FileNames = fuzzy.FindFiles(f.Dir)
	f.FilterFiles()
}

func (f *Files) Resize(w, h int) {
	f.Size.Width, f.Size.Height = w, h
}

func (f *Files) CursorPosition() (y, x int) {
	y = 2
	x = f.SearchCursor + 3
	return
}

func (f *Files) Reset() {
	f.FileCursor = 0
	f.SearchCursor = 0
	f.Search = ""
	f.FilterFiles()
	f.LoadPreview()
}

func (f *Files) HandleInput(key rune, more []rune) {
	switch key {
	case '\t':
	case 127: // DEL
		f.Search = f.Search[:max(0, len(f.Search)-1)]

		searchBoxWidth := f.Size.Width/2 - 4
		f.SearchCursor = min(len(f.Search), searchBoxWidth)
	case 27: // ESC
		if len(more) != 2 {
			break
		}
		if more[0] == 91 { // Arrow
			switch more[1] {
			case 65: // Up
				f.FileCursor = max(0, f.FileCursor-1)
			case 66: // Down
				f.FileCursor = min(f.FileCursor+1, len(f.FilteredFiles)-1)
			case 67: // Right
				f.SearchCursor = min(f.SearchCursor+1, len(f.Search))
			case 68: // Left
				f.SearchCursor = max(0, f.SearchCursor-1)
			}
		}
	default:
		f.Search += string(key)

		searchBoxWidth := f.Size.Width/2 - 4
		f.SearchCursor = min(len(f.Search), searchBoxWidth)
		f.FileCursor = 0
	}

	f.FilterFiles()
	f.LoadPreview()
}

func (f *Files) LoadPreview() {
	if f.FileCursor >= len(f.FilteredFiles) {
		return
	}

	file := f.FilteredFiles[f.FileCursor]
	preview, ok := f.PreviewCache.Get(file)
	if !ok {
		preview = readFileLines(file, f.Size.Width*f.Size.Height, f.Size.Height-2)
		f.PreviewCache.Add(file, preview)
	}
	f.Preview = preview
}

func readFileLines(file string, maxBytes int, maxLines int) []string {
	f, err := os.Open(file)
	if err != nil {
		return nil
	}
	defer f.Close()

	buf := make([]byte, maxBytes)
	n, err := f.Read(buf)
	if err != nil {
		return nil
	}
	if n == 0 {
		return []string{"(empty)"}
	}

	buf = buf[:n]

	for _, b := range buf {
		if b == 0 {
			return []string{"(binary file)"}
		}
	}

	buf = bytes.ReplaceAll(buf, []byte("\t"), []byte("    "))
	return strings.SplitN(string(buf), "\n", maxLines+1)
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
	}
	colors.SetColor(y+0, x, f.Size.Width, frame.ColorFGBlue)
	colors.SetColor(y+f.Size.Height-1, x, f.Size.Width, frame.ColorFGBlue)

	// Veritcal Lines
	for i := 1; i < f.Size.Height-1; i++ {
		text.WriteAt(y+i, x+0, '│')
		text.WriteAt(y+i, x+f.Size.Width-1, '│')
		text.WriteAt(y+i, x+f.Size.Width/2-1, '│')
		text.WriteAt(y+i, x+f.Size.Width/2, '│')
		colors.SetColor(y+i, x, 1, frame.ColorFGBlue)
		colors.SetColor(y+i, x+f.Size.Width-1, 1, frame.ColorFGBlue)
		colors.SetColor(y+i, x+f.Size.Width/2-1, 1, frame.ColorFGBlue)
		colors.SetColor(y+i, x+f.Size.Width/2, 1, frame.ColorFGBlue)
	}

	// Corners
	// Left pane
	text.WriteAt(y+0, x+0, '┌')
	text.WriteAt(y+0, x+f.Size.Width/2-1, '┐')
	text.WriteAt(y+f.Size.Height-1, x+0, '└')
	text.WriteAt(y+f.Size.Height-1, x+f.Size.Width/2-1, '┘')
	// Right pane
	text.WriteAt(y+0, x+f.Size.Width/2, '┌')
	text.WriteAt(y+0, x+f.Size.Width-1, '┐')
	text.WriteAt(y+f.Size.Height-1, x+f.Size.Width/2, '└')
	text.WriteAt(y+f.Size.Height-1, x+f.Size.Width-1, '┘')

	// Search Box
	text.WriteAt(y+2, x+0, '├')
	text.WriteAt(y+2, x+f.Size.Width/2-1, '┤')
	for i := 1; i < f.Size.Width/2-1; i++ {
		text.WriteAt(y+2, x+i, '─')
	}
	colors.SetColor(y+2, x, f.Size.Width, frame.ColorFGBlue)

	// Write search term
	searchBoxWidth := f.Size.Width/2 - 4
	searchTerm := f.Search
	if len(searchTerm) > searchBoxWidth {
		searchTerm = searchTerm[len(searchTerm)-searchBoxWidth:]
	}

	text.WriteString(y+1, x+2, searchTerm)

	// listheight := f.Size.Height - 4
	for i, file := range f.FilteredFiles {
		trimmedFile := strings.TrimPrefix(file, f.Dir+"/")
		if f.FileCursor == i {
			text.WriteString(y+i+3, x+2, "> "+trimmedFile)
			colors.SetColor(y+i+3, x+2, 2, frame.ColorFGGreen)
			colors.SetColor(y+i+3, x+4, len(trimmedFile), frame.ColorFGGreen)
		} else {
			text.WriteString(y+i+3, x+4, trimmedFile)
		}
	}

	for i := 0; i < len(f.Preview) && i < f.Size.Height-2; i++ {
		text.WriteString(y+i+1, x+f.Size.Width/2+2, f.Preview[i])
	}

}
