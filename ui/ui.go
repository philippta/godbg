package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"slices"
	"syscall"

	"github.com/go-delve/delve/service/api"
	"github.com/mattn/go-tty"
	"github.com/philippta/godbg/dlv"
	"github.com/philippta/godbg/term"
)

func Run(dbg *dlv.Debugger) {
	v := &view{}
	v.dbg = dbg
	v.paneNum = 2

	tty, err := tty.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer tty.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	out := tty.Output()
	out.Write(term.AltScreen)
	out.Write(term.HideCursor)
	defer out.Write(term.ShowCursor)
	defer out.Write(term.ExitAltScreen)
	resize(v, tty)

	repaintCh := make(chan struct{})
	go listenresize(v, tty, repaintCh)
	go paintloop(v, tty, repaintCh)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				out.Write(term.ShowCursor)
				out.Write(term.ExitAltScreen)
				tty.Close()
				panic(err)
			}
		}()

		inputloop(v, tty, repaintCh)
		cancel()
	}()

	v.sourceView.breakpoints = v.dbg.Breakpoints()
	v.sourceLoadFile()
	v.variablesLoad()

	repaintCh <- struct{}{}

	<-ctx.Done()
	close(repaintCh)
}

func resize(v *view, tty *tty.TTY) {
	width, height, err := tty.Size()
	if err != nil {
		panic(err)
	}
	v.width = width
	v.height = height
}

func paintloop(v *view, tty *tty.TTY, repaint <-chan struct{}) {
	out := tty.Output()
	for range repaint {
		out.Write(term.ResetCursor)
		out.WriteString(render(v))
	}
}

func inputloop(v *view, tty *tty.TTY, repaint chan<- struct{}) {
	for {
		var err error
		key, err := tty.ReadRune()
		if err != nil {
			panic(err)
		}

		switch key {
		case '\t':
			v.paneActive = (v.paneActive + 1) % v.paneNum
		case 'q':
			return
		case 'k': // Move up
			switch v.paneActive {
			case 0:
				v.sourceMoveUp()
			case 1:
				v.variablesMoveUp()
			}
		case 'j': // Move down
			switch v.paneActive {
			case 0:
				v.sourceMoveDown()
			case 1:
				v.variablesMoveDown()
			}
		case 'l': // Expand
			v.variablesExpand()
		case 'h': // Collapse
			v.variablesCollapse()
		case 's': // Step
			must(v.dbg.Step())
			v.sourceLoadFile()
			v.variablesLoad()
		case 'i': // Step in
			must(v.dbg.StepIn())
			v.sourceLoadFile()
			v.variablesLoad()
		case 'o': // Step out
			must(v.dbg.StepOut())
			v.sourceLoadFile()
			v.variablesLoad()
		case 'c': // Continue
			must(v.dbg.Continue())
			v.sourceLoadFile()
			v.variablesLoad()
		case 'b': // Breakpoint
			v.sourceToggleBreakpoint()
			// default:
			// debug.Logf("%s", key)
		case 'v':
			vars, err := v.dbg.Variables()
			must(err)
			b, _ := json.MarshalIndent(vars, "", "  ")
			os.WriteFile("ui/testdata/vars.json", b, 0o644)
		}

		if v.dbg.Exited() {
			return
		}

		repaint <- struct{}{}
	}
}

func render(v *view) string {
	// start := time.Now()
	// defer func() {
	// 	debug.Logf("Render time: %v", time.Since(start))
	// }()

	source, sourceLens := sourceRender(
		v.sourceView.lines,
		v.width/2,
		v.height,
		v.sourceView.lineStart,
		v.sourceView.pcCursor,
		v.sourceView.lineCursor,
		fileBreakpoints(v.sourceView.breakpoints, v.sourceView.file),
		v.paneActive == 0,
	)

	if len(source) == 0 {
		return ""
	}

	variables, variablesLens := variablesRender(
		v.variablesView.flattened,
		v.width/2,
		v.height,
		v.variablesView.lineStart,
		v.variablesView.lineCursor,
		v.paneActive == 1,
	)

	return verticalSplit(
		v.width, v.height,
		block{source, sourceLens},
		block{variables, variablesLens},
	)
}

func listenresize(v *view, tty *tty.TTY, repaint chan<- struct{}) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	for range ch {
		resize(v, tty)
		repaint <- struct{}{}
	}
}

type view struct {
	width      int
	height     int
	paneActive int
	paneNum    int

	sourceView struct {
		file        string
		lines       [][]byte
		lineStart   int
		lineCursor  int
		pcCursor    int
		breakpoints []*api.Breakpoint
	}
	variablesView struct {
		rawvars []api.Variable

		variables  []variable
		flattened  []variable
		expanded   [][]string
		lineCursor int
		lineStart  int
	}

	dbg *dlv.Debugger
}

func (v *view) variablesLoad() {
	vars, err := v.dbg.Variables()
	must(err)
	v.variablesView.rawvars = vars
	v.variablesView.variables = transformVariables(vars)
	v.variablesView.flattened = flattenVariables(v.variablesView.variables, v.variablesView.expanded)
}

func (v *view) variablesMoveUp() {
	v.variablesView.lineCursor = max(0, v.variablesView.lineCursor-1)
	if v.variablesView.lineCursor < v.variablesView.lineStart+2 {
		v.variablesView.lineStart = max(0, v.variablesView.lineStart-1)
	}
}

func (v *view) variablesMoveDown() {
	v.variablesView.lineCursor = min(v.variablesView.lineCursor+1, len(v.variablesView.flattened)-1)
	if v.variablesView.lineCursor > v.variablesView.lineStart+v.height-3 {
		v.variablesView.lineStart = min(v.variablesView.lineStart+1, len(v.variablesView.flattened)-v.height)
	}
}

func (v *view) variablesExpand() {
	hl := v.variablesView.flattened[v.variablesView.lineCursor]
	if hl.Children == nil {
		return
	}
	var found bool
	for i := range v.variablesView.expanded {
		if slices.Equal(v.variablesView.expanded[i], hl.Path) {
			found = true
			break
		}
	}
	if !found {
		v.variablesView.expanded = append(v.variablesView.expanded, hl.Path)
	}
	v.variablesView.flattened = flattenVariables(v.variablesView.variables, v.variablesView.expanded)
}

func (v *view) variablesCollapse() {
	path := v.variablesView.flattened[v.variablesView.lineCursor].Path

	for len(path) > 0 {
		exists := slices.ContainsFunc(v.variablesView.expanded, func(p []string) bool {
			return slices.Equal(p, path)
		})
		if exists {
			break
		} else {
			path = path[:len(path)-1]
		}
	}

	for i := range v.variablesView.flattened {
		if slices.Equal(v.variablesView.flattened[i].Path, path) {
			v.variablesView.lineCursor = i
		}
	}

	v.variablesView.expanded = slices.DeleteFunc(v.variablesView.expanded, func(p []string) bool {
		return slices.Equal(p, path)
	})
	v.variablesView.flattened = flattenVariables(v.variablesView.variables, v.variablesView.expanded)
}

func (v *view) sourceLoadFile() {
	path, line := v.dbg.Location()
	if path == "" {
		return
	}

	if v.sourceView.file != path {
		src, err := os.ReadFile(path)
		must(err)

		src = bytes.ReplaceAll(src, []byte{'\t'}, []byte("    "))

		v.sourceView.lines = bytes.Split(src, []byte{'\n'})
		v.sourceView.file = path
		v.sourceView.pcCursor = line - 1
		v.sourceView.lineCursor = line - 1
		v.sourceView.lineStart = max(0, min(line-1-v.height/2, len(v.sourceView.lines)-1-v.height))
	} else {
		v.sourceView.pcCursor = line - 1
		v.sourceView.lineCursor = line - 1

		if v.sourceView.lineCursor < v.sourceView.lineStart+2 || v.sourceView.lineCursor > v.sourceView.lineStart+v.height-3 {
			v.sourceView.lineStart = max(0, min(line-1-v.height/2, len(v.sourceView.lines)-1-v.height))
		}
	}
}

func (v *view) sourceMoveUp() {
	v.sourceView.lineCursor = max(0, v.sourceView.lineCursor-1)
	if v.sourceView.lineCursor < v.sourceView.lineStart+2 {
		v.sourceView.lineStart = max(0, v.sourceView.lineStart-1)
	}
}

func (v *view) sourceMoveDown() {
	v.sourceView.lineCursor = min(v.sourceView.lineCursor+1, len(v.sourceView.lines)-2)
	if v.sourceView.lineCursor > v.sourceView.lineStart+v.height-3 {
		v.sourceView.lineStart = min(v.sourceView.lineStart+1, len(v.sourceView.lines)-1-v.height)
	}
}

func (v *view) sourceToggleBreakpoint() {
	var activeBP *api.Breakpoint
	for _, bp := range v.sourceView.breakpoints {
		if bp.File == v.sourceView.file && bp.Line == v.sourceView.lineCursor+1 {
			activeBP = bp
			break
		}
	}

	if activeBP == nil {
		v.dbg.CreateFileBreakpoint(v.sourceView.file, v.sourceView.lineCursor+1)
	} else {
		v.dbg.ClearBreakpoint(activeBP.ID)
	}

	v.sourceView.breakpoints = v.dbg.Breakpoints()
}

func fileBreakpoints(bps []*api.Breakpoint, file string) []int {
	var lineNums []int
	for _, bp := range bps {
		if bp.File == file {
			lineNums = append(lineNums, bp.Line-1)
		}
	}
	return lineNums
}

func countlens(ss []string) []int {
	ll := make([]int, len(ss))
	for i := range ss {
		ll[i] = len(ss[i])
	}
	return ll
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
