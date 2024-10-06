package ui

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/go-delve/delve/service/api"
)

//go:embed testdata/vars.json
var variablesJSON []byte

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
