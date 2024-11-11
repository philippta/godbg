package fuzzy

import (
	"fmt"
	"testing"
)

func TestMatch(t *testing.T) {
	files := FindFiles("..")
	found := Match(files, "fu")
	for _, f := range found {
		fmt.Println(f)
	}
}
