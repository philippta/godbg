package ui

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/go-delve/delve/service/api"
)

type variable struct {
	Name     string
	Value    string
	Type     string
	Kind     reflect.Kind
	Children []variable
	Expanded bool
}

func variablesRender(vars []variable, width, height int, lineCursor int) ([]string, []int) {
	if len(vars) == 0 {
		return []string{}, []int{}
	}

	var lines []string
	var lens []int
	expandVariables(&lines, &lens, vars, 0)

	for i := range lines {
		if i == lineCursor {
			lines[i] = "\033[32m=> " + lines[i]
		} else {
			lines[i] = "\033[37m   " + lines[i]
		}
		lens[i] += 8
	}

	return lines, lens

	lines = make([]string, 0, len(vars))
	lens = make([]int, 0, len(vars))
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
		if i == lineCursor {
			buf.WriteString("\033[32m=> ")
		} else {
			buf.WriteString("\033[37m   ")
		}
		buf.WriteString("\033[37m")
		buf.WriteString(vars[i].Name)
		buf.WriteString(padding[:maxNameLen-nameLen+1])
		buf.WriteString("\033[34m")
		buf.WriteString(vars[i].Type)
		buf.WriteString("\033[37m")
		buf.WriteString(padding[:maxTypeLen-typeLen+1])
		buf.WriteString("= ")
		buf.WriteString(vars[i].Value)

		lines = append(lines, buf.String())
		lens = append(lens, buf.Len()-15 /* ansi seq */)
	}

	return lines, lens
}

func expandVariables(lines *[]string, lens *[]int, vars []variable, indent int) {
	var padding = strings.Repeat(" ", 500)

	var maxNameLen int
	for i := range vars {
		maxNameLen = max(maxNameLen, len(vars[i].Name))
	}

	findMaxTypeLen := func(vars []variable) int {
		var l int
		for i := range vars {
			l = max(l, len(vars[i].Type))
			if vars[i].Expanded && vars[i].Children != nil {
				break
			}
		}
		return l
	}

	var maxTypeLen int
	for i := range vars {
		if maxTypeLen == 0 {
			maxTypeLen = findMaxTypeLen(vars[i:])
		}

		nameLen := len(vars[i].Name)
		typeLen := len(vars[i].Type)

		var buf strings.Builder
		buf.WriteString("\033[37m")
		buf.WriteString(padding[:indent*4])
		buf.WriteString(vars[i].Name)
		buf.WriteString(padding[:maxNameLen-nameLen+1])
		buf.WriteString("\033[34m")
		buf.WriteString(vars[i].Type)
		buf.WriteString(padding[:maxTypeLen-typeLen+1])
		buf.WriteString("\033[37m")
		buf.WriteString("= ")
		buf.WriteString(vars[i].Value)
		*lines = append(*lines, buf.String())
		*lens = append(*lens, buf.Len()-15)

		if vars[i].Expanded && vars[i].Children != nil {
			expandVariables(lines, lens, vars[i].Children, indent+1)
			maxTypeLen = 0
		}
	}
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

func transformVariables(vars []api.Variable) []variable {
	out := make([]variable, len(vars))
	for i := range vars {
		out[i] = transformVariable(vars[i])
	}
	return out
}

func transformVariable(v api.Variable) variable {
	var out variable
	out.Name = v.Name
	out.Value = variableValue(v)
	out.Type = simpleType(v.Type)
	out.Kind = v.Kind

	switch v.Kind {
	case reflect.Slice, reflect.Array:
		for i := range v.Children {
			nv := transformVariable(v.Children[i])
			nv.Name = strconv.Itoa(i)
			out.Children = append(out.Children, nv)
		}
	case reflect.Struct:
		for i := range v.Children {
			nv := transformVariable(v.Children[i])
			out.Children = append(out.Children, nv)
		}
	case reflect.Map:
		for i := 0; i < len(v.Children); i += 2 {
			v.Children[i+1].Name = v.Children[i].Value
			nv := transformVariable(v.Children[i+1])
			out.Children = append(out.Children, nv)
		}
	case reflect.Interface:
		// nv := transformVariable(v.Children[0])
		// nv.Name = v.Name
		out.Type = v.Children[0].Type
		// out.Children = append(out.Children, nv)
	case reflect.Chan:
		elems := v.Children[2].Children[0]           // Index 2 holds the channel values
		recv, _ := strconv.Atoi(v.Children[7].Value) // Index 7 holds the receive index
		nc := transformVariable(elems).Children
		for i := recv; i < len(nc)+recv; i++ {
			out.Children = append(out.Children, nc[i%len(nc)])
		}
		for i := range out.Children {
			out.Children[i].Name = strconv.Itoa(i)
		}
	}
	return out
}

func variableLines(vars []variable) int {
	if vars == nil {
		return 0
	}
	lines := len(vars)
	for i := range vars {
		if vars[i].Expanded {
			lines += variableLines(vars[i].Children)
		}
	}

	return lines
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
