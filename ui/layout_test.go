package ui

import (
	"fmt"
	"testing"
)

func TestVerticalSplit(t *testing.T) {
	blockA := block{
		lines: []string{"\033[37mALine 1", "ALine 2aa"},
		lens:  []int{7, 9},
	}
	blockB := block{
		lines: []string{"BLine 1", "BLine 2aa", "BLine 3"},
		lens:  []int{7, 9, 7},
	}
	got := verticalSplit(198, blockB, blockA)

	fmt.Println(got)
}
