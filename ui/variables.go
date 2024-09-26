package ui

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/go-delve/delve/service/api"
)

func variablesRender(vars []api.Variable, width, height int) ([]string, []int) {
	if len(vars) == 0 {
		return []string{}, []int{}
	}

	// for _, v := range vars {
	// 	debug.LogJSON(v)
	// }

	lines := make([]string, 0, len(vars))
	lens := make([]int, 0, len(vars))
	padding := strings.Repeat(" ", width)

	var maxNameLen int
	var maxTypeLen int
	for i := range vars {
		vars[i].Type = simpleType(vars[i].Type)
		maxNameLen = max(maxNameLen, len(vars[i].Name))
		maxTypeLen = max(maxTypeLen, len(vars[i].Type))
	}

	for i := range vars {
		nameLen := len(vars[i].Name)
		typeLen := len(vars[i].Type)

		var buf strings.Builder
		buf.WriteString(" ")
		buf.WriteString("\033[37m")
		buf.WriteString(vars[i].Name)
		buf.WriteString(padding[:maxNameLen-nameLen+1])
		buf.WriteString("\033[34m")
		buf.WriteString(vars[i].Type)
		buf.WriteString("\033[37m")
		buf.WriteString(padding[:maxTypeLen-typeLen+1])
		buf.WriteString("= ")
		buf.WriteString(variableValue(vars[i]))

		lines = append(lines, buf.String())
		lens = append(lens, buf.Len()-15 /* ansi seq */)
	}

	return lines, lens
}

func variableValue(v api.Variable) string {
	if v.Unreadable != "" {
		return "???"
	}

	switch v.Kind {
	case reflect.Slice:
		if v.Type == "[]any" && len(v.Children) == 1 {
			return variableValue(v.Children[0].Children[0])
		}
		var sb strings.Builder
		sb.WriteString("[")
		for i := range v.Children {
			sb.WriteString(variableValue(v.Children[i]))
			if i < len(v.Children)-1 {
				sb.WriteString(",")
			}
		}
		sb.WriteByte(']')
		return sb.String()
	case reflect.Interface:
		if len(v.Children) == 0 {
			return "??? no val in interface"
		}
		return variableValue(v.Children[0])
	case reflect.Struct:
		var sb strings.Builder
		sb.WriteString("{")
		for i := range v.Children {
			sb.WriteString(v.Children[i].Name)
			sb.WriteString(": ")
			sb.WriteString(variableValue(v.Children[i]))
			if i < len(v.Children)-1 {
				sb.WriteString(", ")
			}
		}
		sb.WriteString("}")
		return sb.String()
	case reflect.Array:
		var sb strings.Builder
		sb.WriteString("[")
		for i := range v.Children {
			sb.WriteString(variableValue(v.Children[i]))
			if i < len(v.Children)-1 {
				sb.WriteString(",")
			}
		}
		sb.WriteByte(']')
		return sb.String()
	case reflect.Pointer:
		return variableValue(v.Children[0])
	case reflect.Map:
		var sb strings.Builder
		sb.WriteString("{")
		for i := 0; i < len(v.Children); i += 2 {
			sb.WriteString(variableValue(v.Children[i]))
			sb.WriteString(": ")
			sb.WriteString(variableValue(v.Children[i+1]))
			if i < len(v.Children)-2 {
				sb.WriteString(", ")
			}
		}
		sb.WriteString("}")
		return sb.String()
	case reflect.Chan:
		var buf api.Variable
		var recvx api.Variable
		for i := range v.Children {
			if v.Children[i].Name == "buf" {
				buf = v.Children[i].Children[0]
			}
			if v.Children[i].Name == "recvx" {
				recvx = v.Children[i]
			}
		}
		start, _ := strconv.Atoi(recvx.Value)

		var sb strings.Builder
		sb.WriteString("[")
		for i := start; i < len(buf.Children)+start; i++ {
			sb.WriteString(variableValue(buf.Children[i%len(buf.Children)]))
			if i < len(buf.Children)+start-1 {
				sb.WriteString(",")
			}
		}
		sb.WriteByte(']')
		return sb.String()
	case reflect.UnsafePointer:
		p, err := strconv.ParseInt(v.Value, 10, 64)
		if err != nil {
			return "(unknown pointer)"
		}
		return "0x" + strconv.FormatInt(p, 16)
	default:
		return v.SinglelineStringWithShortTypes()
	}
}

func simpleType(t string) string {
	if strings.HasSuffix(t, "interface {}") {
		return strings.Replace(t, "interface {}", "any", 1)
	}
	if strings.HasPrefix(t, "struct {") {
		return "struct"
	}
	if strings.HasPrefix(t, "func(") {
		return "func"
	}
	return t
}
