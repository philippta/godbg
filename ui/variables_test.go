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
