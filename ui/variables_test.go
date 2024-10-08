package ui

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/go-delve/delve/service/api"
)

//go:embed testdata/vars.json
var variablesJSON []byte

func TestFlattenVariables2(t *testing.T) {
	var vv []api.Variable
	if err := json.Unmarshal(variablesJSON, &vv); err != nil {
		t.Fatal(err)
	}

	flat := flattenVariables(fillValues([]api.Variable{vv[17]}))

	exp := map[string]bool{}
	linenum := 0

	for _, f := range flat {
		exp[pathKey(f.Path)] = true
	}

	for i := 0; i < 139; i++ {
		fmt.Print("+")
	}
	fmt.Println()

	lines, _ := renderVariables2(flat, exp, 139, 0, 0, linenum, true)
	fmt.Println(strings.Join(lines, "\n"))
}
