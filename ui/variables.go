package ui

import (
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/go-delve/delve/service/api"
)

type variable struct {
	Name     string
	Value    string
	Type     string
	Kind     reflect.Kind
	Path     []string
	Indent   int
	Children []variable
}

func variablesRender(vars []variable, width, height, lineStart, lineCursor int, active bool) ([]string, []int) {
	if len(vars) == 0 {
		return []string{}, []int{}
	}

	lineCursor = lineCursor - lineStart
	vars = vars[lineStart:]

	lines := make([]string, 0, len(vars))
	lens := make([]int, 0, len(vars))
	padding := strings.Repeat(" ", width)

	var maxNameLen int
	var maxTypeLen int
	for i := range vars {
		maxNameLen = max(maxNameLen, len(vars[i].Name)+vars[i].Indent*2)
		maxTypeLen = max(maxTypeLen, len(vars[i].Type))
	}

	for i := range vars {
		nameLen := len(vars[i].Name) + vars[i].Indent*2
		typeLen := len(vars[i].Type)

		var buf strings.Builder
		if i == lineCursor {
			if active {
				buf.WriteString("\033[32m=> ")
			} else {
				buf.WriteString("\033[90m=> ")
			}
		} else {
			buf.WriteString("\033[37m   ")
		}
		buf.WriteString(padding[:vars[i].Indent*2])

		if i == lineCursor && active {
			buf.WriteString("\033[97m")
		} else {
			buf.WriteString("\033[37m")
		}

		buf.WriteString(vars[i].Name)
		buf.WriteString(padding[:maxNameLen-nameLen+1])
		buf.WriteString("\033[34m")
		buf.WriteString(vars[i].Type)

		if i == lineCursor && active {
			buf.WriteString("\033[97m")
		} else {
			buf.WriteString("\033[37m")
		}

		buf.WriteString(padding[:maxTypeLen-typeLen+1])
		buf.WriteString("= ")
		buf.WriteString(vars[i].Value)

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
	case reflect.String:
		return "\"" + v.Value + "\""
	default:
		return v.Value
	}
}

func flattenVariables(vars []variable, expandedVars [][]string) []variable {
	var out []variable

	expanded := func(path []string) bool {
		for i := range expandedVars {
			if slices.Equal(expandedVars[i], path) {
				return true
			}
		}
		return false
	}

	var flatten func([]variable, int)
	flatten = func(vars []variable, indent int) {
		for i := range vars {
			vars[i].Indent = indent
			out = append(out, vars[i])

			if expanded(vars[i].Path) {
				flatten(vars[i].Children, indent+1)
			}
		}
	}

	flatten(vars, 0)
	return out
}

func transformVariables(vars []api.Variable) []variable {
	out := make([]variable, len(vars))
	for i := range vars {
		out[i] = transformVariable(vars[i], nil)
	}
	return out
}

func transformVariable(v api.Variable, path []string) variable {
	var out variable
	out.Name = v.Name
	out.Value = variableValue(v)
	out.Type = simpleType(v.Type)
	out.Kind = v.Kind
	out.Path = append(path, v.Name)

	switch v.Kind {
	case reflect.Slice, reflect.Array:
		for i := range v.Children {
			v.Children[i].Name = strconv.Itoa(i)
			nv := transformVariable(v.Children[i], out.Path)
			out.Children = append(out.Children, nv)
		}
	case reflect.Struct:
		for i := range v.Children {
			nv := transformVariable(v.Children[i], out.Path)
			out.Children = append(out.Children, nv)
		}
	case reflect.Map:
		for i := 0; i < len(v.Children); i += 2 {
			if v.Children[i+1].Kind == reflect.Interface {
				v.Children[i+1] = v.Children[i+1].Children[0]
			}
			v.Children[i+1].Name = v.Children[i].Value
			nv := transformVariable(v.Children[i+1], out.Path)
			out.Children = append(out.Children, nv)
		}
	case reflect.Interface:
		if len(v.Children) > 0 {
			out.Type = v.Children[0].Type
		}
	case reflect.Chan:
		elems := v.Children[2].Children[0].Children  // Index 2 holds the channel values
		recv, _ := strconv.Atoi(v.Children[7].Value) // Index 7 holds the receive index
		for i := recv; i < len(elems)+recv; i++ {
			elems[i%len(elems)].Name = strconv.Itoa(i - recv)
			out.Children = append(out.Children, transformVariable(elems[i%len(elems)], out.Path))
		}
	}
	return out
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
