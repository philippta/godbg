package ui

import (
	_ "embed"
	"encoding/json"
	"os"
	"testing"

	"github.com/go-delve/delve/service/api"
)

//go:embed testdata/vars.json
var variablesJSON []byte

func TestFlattenVariables(t *testing.T) {
	var vv []api.Variable
	if err := json.Unmarshal(variablesJSON, &vv); err != nil {
		t.Fatal(err)
	}

	flat := flattenVariables(fillValues(vv))

	exp := map[string]bool{}
	linenum := 0

	for _, f := range flat {
		exp[pathKey(f.Path)] = true
	}

	v := Variables{
		Focused:    true,
		Variables:  flat,
		Expanded:   exp,
		Size:       Size{Width: 90, Height: 30},
		LineStart:  0,
		LineCursor: linenum,
	}

	text, colors := v.RenderFrame()
	text.PrintLinesColored(os.Stdout, colors)
}
