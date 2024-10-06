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
	Indent   int
	Children []variable
}

type expansion struct {
	Name     string
	Children []expansion
}

func variablesRender(vars []variable, expanded *[]expansion, width, height, lineStart, lineCursor int, active bool) ([]string, []int) {
	if len(vars) == 0 {
		return []string{}, []int{}
	}

	lines := make([]string, 0, len(vars))
	lens := make([]int, 0, len(vars))
	linecount := 0
	padding := strings.Repeat(" ", width)

	var buf strings.Builder
	var render func(vars []variable, indent int, expanded *[]expansion)
	render = func(vars []variable, indent int, expanded *[]expansion) {
		maxNameLen := 0
		for i := range vars {
			maxNameLen = max(maxNameLen, len(vars[i].Name))
		}

		for i := range vars {
			if linecount >= lineStart {
				buf.Reset()
				if linecount == lineCursor {
					if active {
						buf.WriteString("\033[32m=> ")
					} else {
						buf.WriteString("\033[90m=> ")
					}
				} else {
					buf.WriteString("\033[37m   ")
				}

				for i := 0; i < indent; i++ {
					buf.WriteString("  ")
				}

				buf.WriteString("\033[34m")
				buf.WriteString(vars[i].Name)

				if linecount == lineCursor && active {
					buf.WriteString("\033[97m")
				} else {
					buf.WriteString("\033[37m")
				}
				buf.WriteString(padding[:maxNameLen-len(vars[i].Name)])
				buf.WriteString(" = ")
				buf.WriteString(vars[i].Value)

				lines = append(lines, buf.String())
				lens = append(lens, buf.Len()-15 /* ansi seq */)
			}

			linecount++

			if vars[i].Children != nil {
				if exp, ok := findExpansion(vars[i].Name, expanded); ok {
					render(vars[i].Children, indent+1, exp)
				}
			}

		}
	}

	render(vars, 0, expanded)

	return lines, lens
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
					out[i].Value = vars[i].Children[0].Value
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

func findExpansion(varname string, expanded *[]expansion) (*[]expansion, bool) {
	for i := range *expanded {
		if (*expanded)[i].Name == varname {
			return &((*expanded)[i].Children), true
		}
	}
	return nil, false
}

func changeVariableExpansion(vars []variable, expansions *[]expansion, index int, expand bool) {
	count := 0

	var f func(vars []variable, expansions *[]expansion, index int) bool
	f = func(vars []variable, expansions *[]expansion, index int) bool {
		for _, v := range vars {
			if count == index {
				if expand {
					*expansions = append(*expansions, expansion{Name: v.Name})
				} else {
					for i := range *expansions {
						if (*expansions)[i].Name == v.Name {
							*expansions = append((*expansions)[:i], (*expansions)[i+1:]...)
							break
						}
					}
				}
				return true
			}
			count++

			if v.Children != nil {
				if exp, ok := findExpansion(v.Name, expansions); ok {
					if f(v.Children, exp, index) {
						return true
					}
				}
			}
		}

		return false
	}
	f(vars, expansions, index)
}

func countVisibleVariables(vars []variable, expanded *[]expansion) int {
	sum := len(vars)
	for i := range vars {
		if vars[i].Children != nil {
			if exp, ok := findExpansion(vars[i].Name, expanded); ok {
				sum += countVisibleVariables(vars[i].Children, exp)
			}
		}
	}
	return sum
}
