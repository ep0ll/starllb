package llb

import (
	"fmt"

	"github.com/moby/buildkit/client/llb"
	"go.starlark.net/starlark"
)

// ExecState wraps llb.ExecState for Starlark method chaining after Run.
type ExecState struct {
	es llb.ExecState
}

var (
	_ starlark.Value    = (*ExecState)(nil)
	_ starlark.HasAttrs = (*ExecState)(nil)
)

// NewExecState creates a Starlark ExecState.
func NewExecState(es llb.ExecState) *ExecState {
	return &ExecState{es: es}
}

// Underlying returns the raw llb.ExecState.
func (e *ExecState) Underlying() llb.ExecState {
	return e.es
}

func (e *ExecState) String() string { return fmt.Sprintf("<llb.ExecState %p>", e) }
func (e *ExecState) Type() string   { return "llb.ExecState" }
func (e *ExecState) Freeze()        {}
func (e *ExecState) Truth() starlark.Bool { return starlark.True }
func (e *ExecState) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable type: llb.ExecState")
}

func (e *ExecState) AttrNames() []string {
	return []string{"root", "add_mount", "get_mount"}
}

func (e *ExecState) Attr(name string) (starlark.Value, error) {
	switch name {
	case "root":
		return starlark.NewBuiltin("root", e.root), nil
	case "add_mount":
		return starlark.NewBuiltin("add_mount", e.addMount), nil
	case "get_mount":
		return starlark.NewBuiltin("get_mount", e.getMount), nil
	default:
		return nil, nil
	}
}

func (e *ExecState) root(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return NewState(e.es.Root()), nil
}

func (e *ExecState) addMount(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var dest string
	var source starlark.Value
	var readonly bool
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "dest", &dest, "source", &source, "readonly?", &readonly); err != nil {
		return nil, err
	}
	src, ok := source.(*State)
	if !ok {
		return nil, fmt.Errorf("source must be llb.State, got %s", source.Type())
	}
	opts := []llb.MountOption{}
	if readonly {
		opts = append(opts, llb.Readonly)
	}
	// AddMount returns State
	st := e.es.AddMount(dest, src.state, opts...)
	return NewState(st), nil
}

func (e *ExecState) getMount(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var dest string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "dest", &dest); err != nil {
		return nil, err
	}
	return NewState(e.es.GetMount(dest)), nil
}
