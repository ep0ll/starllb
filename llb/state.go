// Package llb provides Starlark-friendly wrappers around BuildKit's LLB State.
// It enables writing production-ready BuildKit LLB graphs using the Starlark language.
package llb

import (
	"context"
	"fmt"
	"os"

	"github.com/moby/buildkit/client/llb"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// State is a Starlark value wrapping llb.State. It supports method chaining
// for a fluent API that mirrors the Go LLB API while remaining idiomatic in Starlark.
type State struct {
	state llb.State
}

// Ensure State implements starlark.Value and starlark.HasAttrs.
var (
	_ starlark.Value    = (*State)(nil)
	_ starlark.HasAttrs = (*State)(nil)
)

// NewState wraps an underlying llb.State.
func NewState(s llb.State) *State {
	return &State{state: s}
}

// Underlying returns the raw llb.State for advanced use or marshaling.
func (s *State) Underlying() llb.State {
	return s.state
}

// String implements starlark.Value.
func (s *State) String() string {
	return fmt.Sprintf("<llb.State %p>", s)
}

// Type implements starlark.Value.
func (s *State) Type() string { return "llb.State" }

// Freeze implements starlark.Value. States are immutable from Starlark's perspective.
func (s *State) Freeze() {}

// Truth implements starlark.Value.
func (s *State) Truth() starlark.Bool { return starlark.True }

// Hash implements starlark.Value. States are not hashable.
func (s *State) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable type: llb.State")
}

// AttrNames implements starlark.HasAttrs.
func (s *State) AttrNames() []string {
	return []string{
		"run", "add_env", "dir", "user", "add_mount", "file",
		"copy", "mkdir", "mkfile", "rm",
		"with", "root", "marshal",
	}
}

// Attr implements starlark.HasAttrs. Dispatches to methods.
func (s *State) Attr(name string) (starlark.Value, error) {
	switch name {
	case "run":
		return starlark.NewBuiltin("run", s.run), nil
	case "add_env":
		return starlark.NewBuiltin("add_env", s.addEnv), nil
	case "dir":
		return starlark.NewBuiltin("dir", s.dir), nil
	case "user":
		return starlark.NewBuiltin("user", s.user), nil
	case "add_mount":
		return starlark.NewBuiltin("add_mount", s.addMount), nil
	case "file":
		return starlark.NewBuiltin("file", s.file), nil
	case "copy":
		return starlark.NewBuiltin("copy", s.copy), nil
	case "mkdir":
		return starlark.NewBuiltin("mkdir", s.mkdir), nil
	case "mkfile":
		return starlark.NewBuiltin("mkfile", s.mkfile), nil
	case "rm":
		return starlark.NewBuiltin("rm", s.rm), nil
	case "with":
		return starlark.NewBuiltin("with", s.with), nil
	case "root":
		return starlark.NewBuiltin("root", s.root), nil
	case "marshal":
		return starlark.NewBuiltin("marshal", s.marshal), nil
	default:
		return nil, nil
	}
}

// run implements state.run(cmd, **kwargs) -> ExecState
func (s *State) run(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var cmd string
	var readonly bool
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "cmd", &cmd, "readonly?", &readonly); err != nil {
		return nil, err
	}
	opts := []llb.RunOption{llb.Shlex(cmd)}
	if readonly {
		opts = append(opts, llb.ReadonlyRootFS())
	}
	es := s.state.Run(opts...)
	return NewExecState(es), nil
}

// addEnv implements state.add_env(key, value) -> State
func (s *State) addEnv(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var key, value string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "key", &key, "value", &value); err != nil {
		return nil, err
	}
	return NewState(s.state.AddEnv(key, value)), nil
}

// dir implements state.dir(path) -> State
func (s *State) dir(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "path", &path); err != nil {
		return nil, err
	}
	return NewState(s.state.Dir(path)), nil
}

// user implements state.user(name) -> State
func (s *State) user(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var name string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "name", &name); err != nil {
		return nil, err
	}
	return NewState(s.state.User(name)), nil
}

// addMount is limited on State; prefer run(...).add_mount(...)
func (s *State) addMount(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return nil, fmt.Errorf("add_mount on State is limited; prefer using run(...).add_mount(...)")
}

// file implements state.file(action) -> State
func (s *State) file(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var action starlark.Value
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "action", &action); err != nil {
		return nil, err
	}
	fa, ok := action.(*FileAction)
	if !ok {
		return nil, fmt.Errorf("action must be llb.FileAction")
	}
	return NewState(s.state.File(fa.action)), nil
}

func (s *State) copy(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var src, dest string
	var input starlark.Value
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "src", &src, "dest", &dest, "input?", &input); err != nil {
		return nil, err
	}
	var ci llb.CopyInput = s.state
	if input != nil {
		if st, ok := input.(*State); ok {
			ci = st.state
		}
	}
	fa := llb.Copy(ci, src, dest)
	return NewState(s.state.File(fa)), nil
}

func (s *State) mkdir(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path string
	var mode int = 0755
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "path", &path, "mode?", &mode); err != nil {
		return nil, err
	}
	fa := llb.Mkdir(path, osFileMode(mode))
	return NewState(s.state.File(fa)), nil
}

func (s *State) mkfile(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path, data string
	var mode int = 0644
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "path", &path, "data", &data, "mode?", &mode); err != nil {
		return nil, err
	}
	fa := llb.Mkfile(path, osFileMode(mode), []byte(data))
	return NewState(s.state.File(fa)), nil
}

func (s *State) rm(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "path", &path); err != nil {
		return nil, err
	}
	fa := llb.Rm(path)
	return NewState(s.state.File(fa)), nil
}

func (s *State) with(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return s, nil
}

func (s *State) root(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return s, nil
}

func (s *State) marshal(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	ctx := context.Background()
	def, err := s.state.Marshal(ctx)
	if err != nil {
		return nil, err
	}
	return starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
		"definition": starlark.String(fmt.Sprintf("%v", def)),
	}), nil
}

func osFileMode(mode int) os.FileMode {
	return os.FileMode(mode)
}
