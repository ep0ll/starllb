package llb

import (
	"fmt"
	"os"

	"github.com/moby/buildkit/client/llb"
	"go.starlark.net/starlark"
)

// FileAction wraps llb.FileAction for chaining file operations in Starlark.
type FileAction struct {
	action *llb.FileAction
}

var (
	_ starlark.Value    = (*FileAction)(nil)
	_ starlark.HasAttrs = (*FileAction)(nil)
)

// NewFileAction creates a Starlark FileAction.
func NewFileAction(a *llb.FileAction) *FileAction {
	return &FileAction{action: a}
}

func (f *FileAction) String() string       { return fmt.Sprintf("<llb.FileAction %p>", f) }
func (f *FileAction) Type() string         { return "llb.FileAction" }
func (f *FileAction) Freeze()              {}
func (f *FileAction) Truth() starlark.Bool { return starlark.True }
func (f *FileAction) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable type: llb.FileAction")
}

func (f *FileAction) AttrNames() []string {
	return []string{"copy", "mkdir", "mkfile", "rm"}
}

func (f *FileAction) Attr(name string) (starlark.Value, error) {
	switch name {
	case "copy":
		return starlark.NewBuiltin("copy", f.copy), nil
	case "mkdir":
		return starlark.NewBuiltin("mkdir", f.mkdir), nil
	case "mkfile":
		return starlark.NewBuiltin("mkfile", f.mkfile), nil
	case "rm":
		return starlark.NewBuiltin("rm", f.rm), nil
	default:
		return nil, nil
	}
}

func (f *FileAction) copy(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var src, dest string
	var input starlark.Value
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "src", &src, "dest", &dest, "input?", &input); err != nil {
		return nil, err
	}
	var ci llb.CopyInput
	if input != nil {
		if st, ok := input.(*State); ok {
			ci = st.state
		} else {
			return nil, fmt.Errorf("input must be llb.State")
		}
	}
	newFA := f.action.Copy(ci, src, dest)
	return NewFileAction(newFA), nil
}

func (f *FileAction) mkdir(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path string
	var mode int = 0755
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "path", &path, "mode?", &mode); err != nil {
		return nil, err
	}
	newFA := f.action.Mkdir(path, os.FileMode(mode))
	return NewFileAction(newFA), nil
}

func (f *FileAction) mkfile(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path, data string
	var mode int = 0644
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "path", &path, "data", &data, "mode?", &mode); err != nil {
		return nil, err
	}
	newFA := f.action.Mkfile(path, os.FileMode(mode), []byte(data))
	return NewFileAction(newFA), nil
}

func (f *FileAction) rm(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "path", &path); err != nil {
		return nil, err
	}
	newFA := f.action.Rm(path)
	return NewFileAction(newFA), nil
}
