package llb

import (
	"fmt"

	"github.com/moby/buildkit/client/llb"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// Module returns the top-level "llb" Starlark module containing constructors
// for images, scratch, git, local, http, and file actions.
func Module() *starlarkstruct.Module {
	return &starlarkstruct.Module{
		Name: "llb",
		Members: starlark.StringDict{
			"image":   starlark.NewBuiltin("llb.image", image),
			"scratch": starlark.NewBuiltin("llb.scratch", scratch),
			"git":     starlark.NewBuiltin("llb.git", git),
			"local":   starlark.NewBuiltin("llb.local", local),
			"http":    starlark.NewBuiltin("llb.http", http),
			"copy":    starlark.NewBuiltin("llb.copy", copyAction),
			"mkdir":   starlark.NewBuiltin("llb.mkdir", mkdirAction),
			"mkfile":  starlark.NewBuiltin("llb.mkfile", mkfileAction),
			"rm":      starlark.NewBuiltin("llb.rm", rmAction),
		},
	}
}

func image(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var ref string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "ref", &ref); err != nil {
		return nil, err
	}
	return NewState(llb.Image(ref)), nil
}

func scratch(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs(b.Name(), args, kwargs); err != nil {
		return nil, err
	}
	return NewState(llb.Scratch()), nil
}

func git(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var remote, ref string
	var keepGitDir bool
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "remote", &remote, "ref?", &ref, "keep_git_dir?", &keepGitDir); err != nil {
		return nil, err
	}
	if ref == "" {
		ref = "HEAD"
	}
	opts := []llb.GitOption{}
	if keepGitDir {
		opts = append(opts, llb.KeepGitDir())
	}
	return NewState(llb.Git(remote, ref, opts...)), nil
}

func local(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var name string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "name", &name); err != nil {
		return nil, err
	}
	return NewState(llb.Local(name)), nil
}

func http(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var url string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "url", &url); err != nil {
		return nil, err
	}
	return NewState(llb.HTTP(url)), nil
}

func copyAction(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var src, dest string
	var input starlark.Value
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "src", &src, "dest", &dest, "input?", &input); err != nil {
		return nil, err
	}
	var ci llb.CopyInput = llb.Scratch()
	if input != nil {
		st, ok := input.(*State)
		if !ok {
			return nil, fmt.Errorf("input must be llb.State")
		}
		ci = st.state
	}
	return NewFileAction(llb.Copy(ci, src, dest)), nil
}

func mkdirAction(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path string
	var mode int = 0755
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "path", &path, "mode?", &mode); err != nil {
		return nil, err
	}
	return NewFileAction(llb.Mkdir(path, osFileMode(mode))), nil
}

func mkfileAction(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path, data string
	var mode int = 0644
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "path", &path, "data", &data, "mode?", &mode); err != nil {
		return nil, err
	}
	return NewFileAction(llb.Mkfile(path, osFileMode(mode), []byte(data))), nil
}

func rmAction(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "path", &path); err != nil {
		return nil, err
	}
	return NewFileAction(llb.Rm(path)), nil
}
