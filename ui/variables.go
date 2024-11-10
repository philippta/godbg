package ui

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/go-delve/delve/service/api"
	"github.com/philippta/godbg/frame"
)

type Variables struct {
	Focused    bool
	Size       Size
	Variables  []Variable
	Expanded   map[string]bool
	NumVisible int
	LineCursor int
	LineStart  int
}

func (v *Variables) Resize(w, h int) {
	v.Size.Width, v.Size.Height = w, h
}

func (v *Variables) Load(vars []api.Variable) {
	v.Variables = flattenVariables(fillValues(vars))
	v.NumVisible = visibleVariables(v.Variables, v.Expanded)
	v.AlignCursor()
}

func (v *Variables) MoveUp() {
	v.LineCursor = max(0, v.LineCursor-1)
	v.AlignCursor()
}

func (v *Variables) MoveDown() {
	v.LineCursor = min(v.LineCursor+1, v.NumVisible-1)
	v.AlignCursor()
}

func (v *Variables) AlignCursor() {
	if v.LineCursor < v.LineStart+2 {
		v.LineStart = max(0, v.LineCursor-2)
	}
	if v.LineCursor > v.LineStart+v.Size.Height-3 {
		v.LineStart = max(0, min(v.LineCursor-v.Size.Height+3, v.NumVisible-v.Size.Height))
	}
	if v.Size.Height > v.NumVisible-v.LineStart {
		v.LineStart = max(0, v.NumVisible-v.Size.Height)
	}
	if v.LineCursor > v.NumVisible-1 {
		v.LineCursor = 0
	}
}

func (v *Variables) ResetCursor(viewHeight int) {
	v.LineCursor = 0
	v.AlignCursor()
}

func (v *Variables) Expand() {
	if v.Expanded == nil {
		v.Expanded = map[string]bool{}
	}
	expandVariable(v.Variables, v.LineCursor, v.Expanded)
	v.NumVisible = visibleVariables(v.Variables, v.Expanded)
}

func (v *Variables) Collapse() {
	collapseVariable(v.Variables, &v.LineCursor, v.Expanded)
	v.NumVisible = visibleVariables(v.Variables, v.Expanded)
	v.AlignCursor()
}

func (v *Variables) RenderFrame() (*frame.Frame, *frame.Frame) {
	text := frame.New(v.Size.Height, v.Size.Width)
	text.FillSpace()

	colors := frame.New(v.Size.Height, v.Size.Width)

	var linenum int
	for _, va := range v.Variables {
		if !isVariableVisible(va, v.Expanded) {
			continue
		}
		if linenum < v.LineStart {
			linenum++
			continue
		}
		if linenum-v.LineStart == v.Size.Height {
			break
		}

		y := linenum - v.LineStart
		x := 0

		if v.Focused {
			colors.SetColor(y, x, 3, frame.ColorFGGreen)
		} else {
			colors.SetColor(y, x, 3, frame.ColorFGBlack)
		}
		if linenum == v.LineCursor {
			x = text.WriteString(y, x, "=> ")
		} else {
			x = text.WriteString(y, x, "   ")
		}

		x = x + va.Depth*2

		colors.SetColor(y, x, len(va.Name), frame.ColorFGBlue)
		x = text.WriteString(y, x, va.Name)

		if linenum == v.LineCursor && v.Focused {
			colors.SetColor(y, x, v.Size.Width-x, frame.ColorFGWhite)
		}
		x = text.WriteString(y, x, " = ")
		x = text.WriteString(y, x, va.Value)

		typePadSize := v.Size.Width - x - len(va.Type)
		if typePadSize > 0 {
			text.WriteString(y, v.Size.Width-len(va.Type), va.Type)
		}

		linenum++
	}

	return text, colors
}

type Variable struct {
	Name     string
	Value    string
	Type     string
	Kind     reflect.Kind
	Path     []string
	Depth    int
	HasChild bool
}

var globsb strings.Builder

func fillValue(v *api.Variable) {
	globsb.Reset()

	v.Type = simpleType(v.Type)

	switch v.Kind {
	case reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr,
		reflect.Float32,
		reflect.Float64:
		if v.Value == "" {
			v.Value = "0"
		}
	case reflect.Complex64,
		reflect.Complex128:
		if v.Value == "" {
			v.Value = "(0+0i)"
		}
	case reflect.Interface:
		if len(v.Children) > 0 {
			v.Value = v.Children[0].Value
		} else {
			v.Value = "<nil>"
		}
	case reflect.String:
		v.Value = "\"" + v.Value + "\""
	case reflect.Slice, reflect.Array:
		globsb.WriteByte('[')
		for i := range v.Children {
			v.Children[i].Name = "[" + strconv.Itoa(i) + "]"
			globsb.WriteString(v.Children[i].Value)
			if i < len(v.Children)-1 {
				globsb.WriteString(",")
			}
		}
		globsb.WriteByte(']')
		v.Value = globsb.String()
	case reflect.Struct:
		globsb.WriteByte('{')
		for i := range v.Children {
			globsb.WriteString(v.Children[i].Name)
			globsb.WriteString(": ")
			globsb.WriteString(v.Children[i].Value)
			if i < len(v.Children)-1 {
				globsb.WriteString(", ")
			}
		}
		globsb.WriteByte('}')
		v.Value = globsb.String()
	case reflect.Map:
		for i := 0; i < len(v.Children); i += 2 {
			v.Children[i+1].Name = v.Children[i].Value
			v.Children[i/2] = v.Children[i+1]
		}
		v.Children = v.Children[:len(v.Children)/2]

		globsb.WriteByte('{')
		for i := range v.Children {
			globsb.WriteString(v.Children[i].Name)
			globsb.WriteString(": ")
			globsb.WriteString(v.Children[i].Value)
			if i < len(v.Children)-1 {
				globsb.WriteString(", ")
			}
		}
		globsb.WriteByte('}')
		v.Value = globsb.String()
	case reflect.Func:
		if v.Value == "" {
			v.Value = "<nil>"
		}
	case reflect.Pointer, reflect.UnsafePointer:
		if v.Value == "0" {
			v.Value = "<nil>"
			v.Children = nil
		} else {
			i, _ := strconv.ParseInt(v.Value, 10, 64)
			v.Value = "0x" + strconv.FormatInt(i, 16)
		}
	case reflect.Chan:
		buf := v.Children[2].Children[0]
		recv, _ := strconv.Atoi(v.Children[7].Value)

		globsb.WriteByte('[')
		for i := recv; i < len(buf.Children)+recv; i++ {
			globsb.WriteString(buf.Children[i%len(buf.Children)].Value)
			if i < len(buf.Children)+recv-1 {
				globsb.WriteString(",")
			}
		}
		globsb.WriteByte(']')
		v.Value = globsb.String()
	}
}

func fillValues(vars []api.Variable) []api.Variable {
	for i := range vars {
		if len(vars[i].Children) > 0 {
			vars[i].Children = fillValues(vars[i].Children)
		}
		fillValue(&vars[i])
	}
	return vars
}

func flattenVariables(vars []api.Variable) []Variable {
	var flat []Variable

	var flatten func(v api.Variable, path []string, depth int)
	flatten = func(v api.Variable, path []string, depth int) {
		flat = append(flat, Variable{
			Name:     v.Name,
			Type:     v.Type,
			Value:    v.Value,
			Kind:     v.Kind,
			Path:     path,
			Depth:    depth,
			HasChild: len(v.Children) > 0,
		})

		for _, child := range v.Children {
			childPath := append(path, child.Name)
			flatten(child, childPath, depth+1)
		}
	}

	for _, v := range vars {
		flatten(v, []string{v.Name}, 0)
	}

	return flat
}

func visibleVariables(vars []Variable, exp map[string]bool) int {
	var sum int
	for _, v := range vars {
		if isVariableVisible(v, exp) {
			sum++
		}
	}
	return sum
}

func pathKey(path []string) string {
	return strings.Join(path, ".")
}

func isVariableVisible(v Variable, exp map[string]bool) bool {
	if v.Depth == 0 {
		return true
	}

	for i := 1; i < len(v.Path); i++ {
		parentPath := v.Path[:i]
		if !exp[pathKey(parentPath)] {
			return false
		}
	}

	return true
}

func expandVariable(vars []Variable, cursor int, exp map[string]bool) {
	count := 0
	for _, v := range vars {
		if !isVariableVisible(v, exp) {
			continue
		}

		if count == cursor {
			if v.HasChild {
				exp[pathKey(v.Path)] = true
			}
			break
		}
		count++
	}
}

func collapseVariable(vars []Variable, cursor *int, exp map[string]bool) {
	count := 0
	for _, v := range vars {
		if !isVariableVisible(v, exp) {
			continue
		}

		if count == *cursor {
			currPathKey := pathKey(v.Path)

			if _, ok := exp[currPathKey]; ok {
				delete(exp, currPathKey)
			} else {
				parentPath := v.Path[:max(len(v.Path)-1, 1)]
				parentPathKey := pathKey(parentPath)

				delete(exp, parentPathKey)

				newcursor := 0
				for _, w := range vars {
					if !isVariableVisible(w, exp) {
						continue
					}
					if len(w.Path) == len(parentPath) && pathKey(w.Path) == parentPathKey {
						break
					}
					newcursor++
				}
				*cursor = newcursor
			}
			break
		}
		count++
	}
}

func simpleType(t string) string {
	if strings.HasSuffix(t, "interface {}") {
		return strings.Replace(t, "interface {}", "any", 1)
	}
	return t
}
