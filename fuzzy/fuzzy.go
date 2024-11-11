package fuzzy

import (
	"io/fs"
	"path/filepath"
	"sort"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

func FindFiles(dir string) []util.Chars {
	dir, _ = filepath.Abs(dir)

	var files []util.Chars
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		isdir := d.IsDir()
		if isdir && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if !isdir {
			files = append(files, util.ToChars([]byte(path)))
		}
		return nil
	})
	return files
}

func Match(files []util.Chars, pattern string) []string {
	type scored struct {
		score int
		index int
	}

	if pattern == "" {
		out := make([]string, len(files))
		for i, f := range files {
			out[i] = f.ToString()
		}
		sort.Slice(out, func(i, j int) bool {
			return len(out[i]) < len(out[j])
		})
		return out
	}

	var found []scored
	for i, file := range files {
		res, _ := algo.FuzzyMatchV2(false, false, false, &file, []rune(pattern), true, nil)
		if res.Score > 0 {
			found = append(found, scored{res.Score, i})
		}
	}

	sort.Slice(found, func(i, j int) bool {
		if found[i].score == found[j].score {
			return files[found[i].index].Length() < files[found[i].index].Length()
		}
		return found[i].score > found[j].score
	})

	out := make([]string, len(found))
	for i, f := range found {
		out[i] = files[f.index].ToString()
	}

	return out

}
