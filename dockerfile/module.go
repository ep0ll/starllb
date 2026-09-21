package dockerfile

import (
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// Module returns the top-level "dockerfile" Starlark module.
// Users write:
//
//	load("dockerfile", "from_", "scratch")
//	def build():
//	    s = from_("alpine:3.20")
//	    s = s.run("apk add --no-cache curl")
//	    s = s.copy(src=".", dest="/app")
//	    return s
func Module() *starlarkstruct.Module {
	return &starlarkstruct.Module{
		Name: "dockerfile",
		Members: starlark.StringDict{
			"from_":   starlark.NewBuiltin("from_", from_),
			"scratch": starlark.NewBuiltin("scratch", scratch),
			// aliases that feel more natural
			"FROM":    starlark.NewBuiltin("FROM", from_),
			"SCRATCH": starlark.NewBuiltin("SCRATCH", scratch),
		},
	}
}

// from_ implements FROM <image> [AS name]
func from_(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var ref, name string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "ref", &ref, "as_?", &name); err != nil {
		return nil, err
	}
	return NewStage(ref, name), nil
}

// scratch implements FROM scratch [AS name]
func scratch(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var name string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "as_?", &name); err != nil {
		return nil, err
	}
	return NewScratchStage(name), nil
}
