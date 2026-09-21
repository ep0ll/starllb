// Package starlark provides the Starlark execution environment for starllb.
package starlark

import (
	"fmt"

	"github.com/ep0ll/starllb/llb"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// Runtime holds the Starlark thread and predeclared globals for LLB scripts.
type Runtime struct {
	thread *starlark.Thread
	globals starlark.StringDict
}

// New creates a new Runtime with the llb module predeclared.
func New() *Runtime {
	predeclared := starlark.StringDict{
		"llb": llb.Module(),
	}
	// Also expose a few convenience aliases at top level if desired.
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

// GetState extracts an llb.State from a Starlark value if possible.
func GetState(v starlark.Value) (*llb.State, error) {
	if s, ok := v.(*llb.State); ok {
		return s, nil
	}
	return nil, fmt.Errorf("expected llb.State, got %s", v.Type())
}

// Module is re-exported for convenience.
func Module() *starlarkstruct.Module {
	return llb.Module()
}
