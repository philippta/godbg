package variables

import (
	"bytes"
	"fmt"

	"github.com/google/go-dap"
)

func Print(vars []dap.Variable) string {
	if len(vars) == 0 {
		return ""
	}
	var buf bytes.Buffer
	for i := 0; i < len(vars); i++ {
		buf.WriteString(fmt.Sprintf("%10s %10s\n", vars[i].Name+" "+vars[i].Type, vars[i].Value))
	}

	return string(buf.Bytes()[:buf.Len()-1])
}
