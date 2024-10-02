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
		if len(v.Children) == 0 {
			return "???"
		}
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
		out[i].Name = vars[i].Name
		out[i].Value = vars[i].Value
		out[i].Type = simpleType(vars[i].Type)
		out[i].Kind = vars[i].Kind

		switch vars[i].Kind {
		case reflect.Bool:
			if out[i].Value == "" {
				out[i].Value = "false"
			}
		case reflect.Int, reflect.Int8, reflect.Int16,
			reflect.Int32, reflect.Int64, reflect.Uint,
			reflect.Uint8, reflect.Uint16, reflect.Uint32,
			reflect.Uint64, reflect.Uintptr:
			if out[i].Value == "" {
				out[i].Value = "0"
			}
		case reflect.String:
			out[i].Value = "\"" + out[i].Value + "\""
		case reflect.Func:
			if out[i].Value == "" {
				out[i].Value = "<nil>"
			}
		case reflect.Slice, reflect.Array:
			var b strings.Builder
			b.WriteByte('[')
			if len(vars[i].Children) > 0 {
				out[i].Children = transformVariables(vars[i].Children)

				for j := range out[i].Children {
					// set array index while we're at it
					out[i].Children[j].Name = strconv.Itoa(j)

					b.WriteString(out[i].Children[j].Value)
					if j < len(out[i].Children)-1 {
						b.WriteString(",")
					}
				}
			}
			b.WriteByte(']')
			out[i].Value = b.String()
		case reflect.Interface:
			if len(vars[i].Children) == 0 {
				out[i].Value = "<nil>"
			} else {
				if vars[i].Children[0].Type == "void" {
					out[i].Value = "<nil>"
				} else {
					if vars[i].Children[0].Kind == reflect.String {
						out[i].Value = "\"" + vars[i].Children[0].Value + "\""
					} else {
						out[i].Value = vars[i].Children[0].Value
					}

					out[i].Type = simpleType(vars[i].Children[0].Type)
					out[i].Kind = vars[i].Children[0].Kind
					if len(vars[i].Children[0].Children) > 0 {
						out[i].Children = transformVariables(vars[i].Children[0].Children)
					}
				}
			}
		case reflect.Pointer:
			if len(vars[i].Children) == 0 {
				out[i].Value = "<nil>"
			} else {
				out[i].Children = transformVariables(vars[i].Children)
				out[i].Value = out[i].Children[0].Value
			}
		case reflect.Struct:
			var b strings.Builder
			b.WriteByte('{')
			if len(vars[i].Children) > 0 {
				out[i].Children = transformVariables(vars[i].Children)

				for j := range out[i].Children {
					b.WriteString(out[i].Children[j].Name)
					b.WriteString(": ")
					b.WriteString(out[i].Children[j].Value)
					if j < len(out[i].Children)-1 {
						b.WriteString(", ")
					}
				}
			}
			b.WriteByte('}')
			out[i].Value = b.String()

		case reflect.Map:
			var b strings.Builder
			b.WriteByte('{')
			if len(vars[i].Children) > 0 {
				for j := 0; j < len(vars[i].Children); j += 2 {
					if vars[i].Children[j].Kind == reflect.String {
						vars[i].Children[j+1].Name = "\"" + vars[i].Children[j].Value + "\""
					} else {
						vars[i].Children[j+1].Name = vars[i].Children[j].Value
					}
					vars[i].Children[j/2] = vars[i].Children[j+1]
				}
				out[i].Children = transformVariables(vars[i].Children[:len(vars[i].Children)/2])

				for j := range out[i].Children {
					b.WriteString(out[i].Children[j].Name)
					b.WriteString(": ")
					b.WriteString(out[i].Children[j].Value)
					if j < len(out[i].Children)-1 {
						b.WriteString(", ")
					}
				}
			}
			b.WriteByte('}')
			out[i].Value = b.String()
		case reflect.UnsafePointer:
			out[i].Value = prettyPointer(out[i].Value)
			if len(vars[i].Children) > 0 && vars[i].Children[0].Type != "void" {
				out[i].Children = transformVariables(vars[i].Children)
			}
		case reflect.Chan:
			if len(vars[i].Children) == 0 {
				vars[i].Value = "<nil>"
			} else {
				buf := vars[i].Children[2].Children[0]
				recv, _ := strconv.Atoi(vars[i].Children[7].Value)

				newc := make([]api.Variable, len(buf.Children))
				for j := recv; j < len(buf.Children)+recv; j++ {
					newc[j-recv] = buf.Children[j%len(buf.Children)]
				}
				vars[i].Children[2].Children[0].Children = newc

				transv := transformVariables(vars[i].Children[2].Children)[0]
				out[i].Children = transv.Children
				out[i].Value = transv.Value
			}
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

func prettyPointer(p string) string {
	i, _ := strconv.ParseInt(p, 10, 64)
	return "0x" + strconv.FormatInt(i, 16)
}
