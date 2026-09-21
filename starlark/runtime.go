// Package starlark provides the Starlark execution environment for starllb.
package starlark

import (
	"fmt"

	"github.com/ep0ll/starllb/dockerfile"
	"github.com/ep0ll/starllb/llb"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// Runtime holds the Starlark thread and predeclared globals for LLB / Dockerfile scripts.
type Runtime struct {
	thread  *starlark.Thread
	globals starlark.StringDict
}

// New creates a new Runtime with both the low-level llb module and the
// high-level dockerfile module predeclared.
func New() *Runtime {
	predeclared := starlark.StringDict{
		"llb":        llb.Module(),
		"dockerfile": dockerfile.Module(),
		// convenience top-level aliases so users can write from_(…) without load()
		"from_":   dockerfile.Module().Members["from_"],
		"scratch": dockerfile.Module().Members["scratch"],
		"FROM":    dockerfile.Module().Members["FROM"],
		"SCRATCH": dockerfile.Module().Members["SCRATCH"],
	}
	thread := &starlark.Thread{
		Name: "starllb",
		Print: func(_ *starlark.Thread, msg string) {
			fmt.Println(msg)
		},
	}
	return &Runtime{
		thread:  thread,
		globals: predeclared,
	}
}

// ExecFile executes a Starlark file and returns the globals after execution.
func (r *Runtime) ExecFile(filename string, src interface{}) (starlark.StringDict, error) {
	globals, err := starlark.ExecFile(r.thread, filename, src, r.globals)
	if err != nil {
		return nil, err
	}
	return globals, nil
}

// Eval evaluates a Starlark expression in the current environment.
func (r *Runtime) Eval(expr string) (starlark.Value, error) {
	return starlark.Eval(r.thread, "<expr>", expr, r.globals)
}

// Call calls a named function from the globals with the given arguments.
func (r *Runtime) Call(name string, args ...starlark.Value) (starlark.Value, error) {
	v, ok := r.globals[name]
	if !ok {
		return nil, fmt.Errorf("undefined: %s", name)
	}
	fn, ok := v.(starlark.Callable)
	if !ok {
		return nil, fmt.Errorf("%s is not callable", name)
	}
	return starlark.Call(r.thread, fn, starlark.Tuple(args), nil)
}

// GetState extracts an llb.State from a Starlark value (supports both low-level
// State and high-level Stage).
func GetState(v starlark.Value) (*llb.State, error) {
	switch s := v.(type) {
	case *llb.State:
		return s, nil
	case *dockerfile.Stage:
		return s.Underlying(), nil
	default:
		return nil, fmt.Errorf("expected llb.State or dockerfile.Stage, got %s", v.Type())
	}
}

// GetStage extracts a dockerfile.Stage if possible.
func GetStage(v starlark.Value) (*dockerfile.Stage, error) {
	if s, ok := v.(*dockerfile.Stage); ok {
		return s, nil
	}
	return nil, fmt.Errorf("expected dockerfile.Stage, got %s", v.Type())
}

// Module is re-exported for convenience (low-level).
func Module() *starlarkstruct.Module {
	return llb.Module()
}
