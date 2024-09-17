package debug

import (
	"fmt"
	"os"
)

func Truncate() {
	os.Truncate("debug.log", 0)
}

func Logf(format string, args ...any) {
	f, err := os.OpenFile("debug.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}
	f.WriteString(fmt.Sprintf(format, args...))
	f.Write([]byte("\n"))
	f.Close()
}
