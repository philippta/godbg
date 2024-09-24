package ui

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

var (
	blockA = createDummyBlock(60, 95)
	blockB = createDummyBlock(60, 95)
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
	got := verticalSplit(190, 60, blockA, blockB)

	fmt.Println(got)
}

func BenchmarkVerticalSplitV1(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		verticalSplit(190, 60, blockA, blockB)
	}
}

func createDummyBlock(l, ll int) block {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	lines := make([]string, l)
	lens := make([]int, l)

	for i := 0; i < l; i++ {
		line := strings.Repeat(string(chars[rand.Intn(len(chars))]), rand.Intn(ll))
		lines[i] = line
		lens[i] = len(line)
	}

	return block{
		lines: lines,
		lens:  lens,
	}
}
