package ui

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
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

	flat := flattenVariables([]api.Variable{vv[14], vv[15]})
	flat[0].Name = ""
	flat[1].Name = ""

	exp := map[string]bool{}
	linenum := 0

	lines, _ := renderVariables2(flat, exp, 0, 0, 0, linenum, true)
	fmt.Println(strings.Join(lines, "\n"))
	fmt.Println("=========================")

	expandVariable(flat, linenum, exp)
	lines, _ = renderVariables2(flat, exp, 0, 0, 0, linenum, true)
	fmt.Println(strings.Join(lines, "\n"))
	fmt.Println("=========================")

	linenum = 2

	lines, _ = renderVariables2(flat, exp, 0, 0, 0, linenum, true)
	fmt.Println(strings.Join(lines, "\n"))
	fmt.Println("=========================")

	collapseVariable(flat, &linenum, exp)

	lines, _ = renderVariables2(flat, exp, 0, 0, 0, linenum, true)
	fmt.Println(strings.Join(lines, "\n"))
	fmt.Println("=========================")

	collapseVariable(flat, &linenum, exp)

	lines, _ = renderVariables2(flat, exp, 0, 0, 0, linenum, true)
	fmt.Println(strings.Join(lines, "\n"))
	fmt.Println("=========================")
}

func TestTransformVariables(t *testing.T) {
	var vv []api.Variable
	if err := json.Unmarshal(variablesJSON, &vv); err != nil {
		t.Fatal(err)
	}

	out := transformVariables([]api.Variable{vv[2]})
	b, _ := json.MarshalIndent(out, "", "  ")
	os.Stdout.Write(b)

	fmt.Println()
}

func TestExpandVariables(t *testing.T) {
	var vv []api.Variable
	if err := json.Unmarshal(variablesJSON, &vv); err != nil {
		t.Fatal(err)
	}

	out := transformVariables([]api.Variable{vv[16], vv[17]})
	b, _ := json.MarshalIndent(out, "", "  ")
	os.Stdout.Write(b)
	fmt.Println()

	expanded := []expansion{}
	changeVariableExpansion(out, &expanded, 0, true)
	changeVariableExpansion(out, &expanded, 1, true)
	changeVariableExpansion(out, &expanded, 5, true)
	changeVariableExpansion(out, &expanded, 1, false)

	b, _ = json.MarshalIndent(expanded, "", "  ")
	os.Stdout.Write(b)
	fmt.Println()
}

func BenchmarkTransformVariables(b *testing.B) {
	var vv []api.Variable
	if err := json.Unmarshal(variablesJSON, &vv); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		transformVariables(vv)
	}
}
