package dlv

import (
	"fmt"
	"path/filepath"

	"github.com/go-delve/delve/pkg/proc"
	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/debugger"
	"github.com/philippta/godbg/build"
)

type Debugger struct {
	dbg   *debugger.Debugger
	state *api.DebuggerState
}

func Test(path string, funcExpr string) (*Debugger, error) {
	binpath, err := build.Test(path)
	if err != nil {
		return nil, fmt.Errorf("build test executable: %w", err)
	}

	pkg, err := build.PackageInfo(path)
	if err != nil {
		return nil, fmt.Errorf("package info: %w", err)
	}

	funcs, err := build.TestFunctions(binpath, funcExpr)
	if err != nil {
		return nil, fmt.Errorf("package info: %w", err)
	}

	cfg := &debugger.Config{
		WorkingDir:     filepath.Dir(path),
		Backend:        "default",
		ExecuteKind:    debugger.ExecutingGeneratedTest,
		CheckGoVersion: true,
		Stdout: proc.OutputRedirect{
			Path: "/dev/null",
		},
		Stderr: proc.OutputRedirect{
			Path: "/dev/null",
		},
	}

	processArgs := []string{binpath}
	if funcExpr != "" {
		processArgs = append(processArgs, "-test.run", funcExpr)
	}
	dbg, err := debugger.New(cfg, processArgs)
	if err != nil {
		return nil, fmt.Errorf("start debugger :%w", err)
	}

	d := &Debugger{dbg: dbg, state: &api.DebuggerState{}}
	for _, f := range funcs {
		if err := d.CreateFunctionBreakpoint(pkg.ImportPath + "." + f); err != nil {
			if err := d.CreateFunctionBreakpoint(pkg.ImportPath + "_test." + f); err != nil {
				panic(err)
			}
		}
	}
	d.Continue()

	return d, nil
}

func Build(path string, args []string) (*Debugger, error) {
	pkg, err := build.PackageInfo(path)
	if err != nil {
		return nil, fmt.Errorf("package info: %w", err)
	}
	if pkg.Name != "main" {
		return nil, fmt.Errorf("package is not main")
	}

	binpath, err := build.Build(path)
	if err != nil {
		return nil, fmt.Errorf("build executable: %w", err)
	}

	cfg := &debugger.Config{
		WorkingDir:     filepath.Dir(path),
		Backend:        "default",
		ExecuteKind:    debugger.ExecutingGeneratedFile,
		CheckGoVersion: true,
		Stdout: proc.OutputRedirect{
			Path: "/dev/null",
		},
		Stderr: proc.OutputRedirect{
			Path: "/dev/null",
		},
	}

	processArgs := []string{binpath}
	processArgs = append(processArgs, args...)
	dbg, err := debugger.New(cfg, processArgs)
	if err != nil {
		return nil, fmt.Errorf("start debugger :%w", err)
	}

	d := &Debugger{dbg: dbg, state: &api.DebuggerState{}}
	if err := d.CreateFunctionBreakpoint("main.main"); err != nil {
		panic(err)
	}
	d.Continue()

	return d, nil
}

func Exec(program string) (*Debugger, error) {
	cfg := &debugger.Config{
		WorkingDir:     filepath.Dir(program),
		Backend:        "default",
		ExecuteKind:    debugger.ExecutingExistingFile,
		CheckGoVersion: true,
		Stdout: proc.OutputRedirect{
			Path: "/dev/null",
		},
		Stderr: proc.OutputRedirect{
			Path: "/dev/null",
		},
	}
	dbg, err := debugger.New(cfg, []string{program})
	if err != nil {
		return nil, fmt.Errorf("start debugger :%w", err)
	}

	d := &Debugger{dbg: dbg, state: &api.DebuggerState{}}
	d.CreateFunctionBreakpoint("main.main")
	d.Continue()

	return d, nil
}

func (d *Debugger) Step() error {
	state, err := d.dbg.Command(&api.DebuggerCommand{Name: api.Next}, nil, nil)
	if err != nil {
		return err
	}
	d.state = state
	return nil
}

func (d *Debugger) StepIn() error {
	state, err := d.dbg.Command(&api.DebuggerCommand{Name: api.Step}, nil, nil)
	if err != nil {
		return err
	}
	d.state = state
	return nil
}

func (d *Debugger) StepOut() error {
	state, err := d.dbg.Command(&api.DebuggerCommand{Name: api.StepOut}, nil, nil)
	if err != nil {
		return err
	}
	d.state = state
	return nil
}

func (d *Debugger) Continue() error {
	state, err := d.dbg.Command(&api.DebuggerCommand{Name: api.Continue}, nil, nil)
	if err != nil {
		return err
	}
	d.state = state
	return nil
}

func (d *Debugger) Variables() ([]api.Variable, error) {
	cfg := proc.LoadConfig{
		FollowPointers:     true,
		MaxVariableRecurse: 1,
		MaxStringLen:       100,
		MaxArrayValues:     64,
		MaxStructFields:    -1,
	}

	args, err := d.dbg.FunctionArguments(d.state.CurrentThread.GoroutineID, 0, 0, cfg)
	if err != nil {
		return nil, err
	}
	locals, err := d.dbg.LocalVariables(d.state.CurrentThread.GoroutineID, 0, 0, cfg)
	if err != nil {
		return nil, err
	}

	return api.ConvertVars(append(args, locals...)), nil
}

func (d *Debugger) CreateFileBreakpoint(file string, line int) error {
	_, err := d.dbg.CreateBreakpoint(&api.Breakpoint{File: file, Line: line}, "", nil, false)
	return err
}

func (d *Debugger) CreateFunctionBreakpoint(name string) error {
	_, err := d.dbg.CreateBreakpoint(&api.Breakpoint{FunctionName: name}, "", nil, false)
	return err
}

func (d *Debugger) ClearBreakpoint(id int) error {
	_, err := d.dbg.ClearBreakpoint(&api.Breakpoint{ID: id})
	return err
}

func (d *Debugger) Breakpoints() []*api.Breakpoint {
	return d.dbg.Breakpoints(true)
}

func (d *Debugger) Exited() bool {
	return d.state.Exited
}

func (d *Debugger) Location() (string, int) {
	if d.state.CurrentThread == nil || d.state.CurrentThread.File == "(autogenerated)" {
		return "", 0
	}
	return d.state.CurrentThread.File, d.state.CurrentThread.Line
}

func (d *Debugger) Close() error {
	return d.dbg.Detach(true)
}
