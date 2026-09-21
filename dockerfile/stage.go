// Package dockerfile provides a high-level, Dockerfile-as-code API on top of
// BuildKit LLB. Method names and options deliberately mirror the official
// Dockerfile reference (https://docs.docker.com/reference/dockerfile/) so that
// existing Dockerfiles can be translated almost mechanically.
package dockerfile

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ep0ll/starllb/llb"
	bkllb "github.com/moby/buildkit/client/llb"
	"go.starlark.net/starlark"
)

// Stage is a build stage (one FROM … until the next FROM).
// It holds both the LLB graph and the image configuration metadata that would
// normally be written by Dockerfile instructions such as ENV, LABEL, ENTRYPOINT, etc.
type Stage struct {
	state   *llb.State // underlying LLB state
	name    string     // optional stage name (AS name)
	workdir string
	user    string
	env     map[string]string
	labels  map[string]string
	expose  []string
	volumes []string
	// entrypoint / cmd are stored as JSON-style argv
	entrypoint []string
	cmd        []string
	// shell form vs exec form
	shell []string // default ["/bin/sh", "-c"]
}

// Ensure Stage implements starlark.Value / HasAttrs.
var (
	_ starlark.Value    = (*Stage)(nil)
	_ starlark.HasAttrs = (*Stage)(nil)
)

// NewStage creates a Stage from a base image reference.
func NewStage(ref string, name string) *Stage {
	st := llb.NewState(bkllb.Image(ref))
	return &Stage{
		state:   st,
		name:    name,
		env:     make(map[string]string),
		labels:  make(map[string]string),
		shell:   []string{"/bin/sh", "-c"},
		workdir: "/",
	}
}

// NewScratchStage creates a stage from scratch.
func NewScratchStage(name string) *Stage {
	st := llb.NewState(bkllb.Scratch())
	return &Stage{
		state:   st,
		name:    name,
		env:     make(map[string]string),
		labels:  make(map[string]string),
		shell:   []string{"/bin/sh", "-c"},
		workdir: "/",
	}
}

// Underlying returns the low-level LLB State for advanced use.
func (s *Stage) Underlying() *llb.State { return s.state }

// Name returns the stage name (AS name) if set.
func (s *Stage) Name() string { return s.name }

// ---------- starlark.Value ----------

func (s *Stage) String() string {
	if s.name != "" {
		return fmt.Sprintf("<dockerfile.Stage %q>", s.name)
	}
	return fmt.Sprintf("<dockerfile.Stage %p>", s)
}
func (s *Stage) Type() string         { return "dockerfile.Stage" }
func (s *Stage) Freeze()              {}
func (s *Stage) Truth() starlark.Bool { return starlark.True }
func (s *Stage) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable type: dockerfile.Stage")
}

func (s *Stage) AttrNames() []string {
	return []string{
		"run", "copy", "add", "env", "workdir", "user",
		"label", "expose", "volume", "entrypoint", "cmd",
		"arg", "healthcheck", "shell",
		"state", // escape hatch to low-level llb.State
	}
}

func (s *Stage) Attr(name string) (starlark.Value, error) {
	switch name {
	case "run":
		return starlark.NewBuiltin("run", s.run), nil
	case "copy":
		return starlark.NewBuiltin("copy", s.copy), nil
	case "add":
		return starlark.NewBuiltin("add", s.add), nil
	case "env":
		return starlark.NewBuiltin("env", s.envMethod), nil
	case "workdir":
		return starlark.NewBuiltin("workdir", s.workdirMethod), nil
	case "user":
		return starlark.NewBuiltin("user", s.userMethod), nil
	case "label":
		return starlark.NewBuiltin("label", s.labelMethod), nil
	case "expose":
		return starlark.NewBuiltin("expose", s.exposeMethod), nil
	case "volume":
		return starlark.NewBuiltin("volume", s.volumeMethod), nil
	case "entrypoint":
		return starlark.NewBuiltin("entrypoint", s.entrypointMethod), nil
	case "cmd":
		return starlark.NewBuiltin("cmd", s.cmdMethod), nil
	case "shell":
		return starlark.NewBuiltin("shell", s.shellMethod), nil
	case "state":
		return s.state, nil
	default:
		return nil, nil
	}
}

// ---------- instruction implementations ----------

// run implements RUN [OPTIONS] <command>
func (s *Stage) run(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var cmd string
	var shellForm = true
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "cmd", &cmd, "shell?", &shellForm); err != nil {
		return nil, err
	}

	var runOpts []bkllb.RunOption
	if shellForm {
		full := strings.Join(append(s.shell, cmd), " ")
		runOpts = append(runOpts, bkllb.Shlex(full))
	} else {
		runOpts = append(runOpts, bkllb.Args(strings.Fields(cmd)))
	}

	st := s.state.Underlying()
	if s.workdir != "" && s.workdir != "/" {
		st = st.Dir(s.workdir)
	}
	if s.user != "" {
		st = st.User(s.user)
	}
	for k, v := range s.env {
		st = st.AddEnv(k, v)
	}

	es := st.Run(runOpts...)
	newState := llb.NewState(es.Root())
	return s.cloneWithState(newState), nil
}

// copy implements COPY [OPTIONS] <src>… <dest>
// Options: from_ (Stage), chown, chmod
func (s *Stage) copy(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var src, dest string
	var fromStage starlark.Value
	var chown string
	var chmod int = -1
	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"src", &src, "dest", &dest,
		"from_?", &fromStage, "chown?", &chown, "chmod?", &chmod); err != nil {
		return nil, err
	}

	var input bkllb.CopyInput = s.state.Underlying()
	if fromStage != nil {
		switch v := fromStage.(type) {
		case *Stage:
			input = v.state.Underlying()
		case starlark.String:
			return nil, fmt.Errorf("copy from_ currently requires a Stage object, got string %q", string(v))
		default:
			return nil, fmt.Errorf("copy from_ must be a Stage, got %s", fromStage.Type())
		}
	}

	fa := bkllb.Copy(input, src, dest)
	_ = chown
	_ = chmod

	st := s.state.Underlying().File(fa)
	return s.cloneWithState(llb.NewState(st)), nil
}

// add implements ADD [OPTIONS] <src>… <dest>
func (s *Stage) add(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var src, dest string
	var chown string
	var chmod int = -1
	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"src", &src, "dest", &dest, "chown?", &chown, "chmod?", &chmod); err != nil {
		return nil, err
	}

	var st bkllb.State
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		http := bkllb.HTTP(src)
		fa := bkllb.Copy(http, "/", dest)
		st = s.state.Underlying().File(fa)
	} else {
		fa := bkllb.Copy(s.state.Underlying(), src, dest)
		st = s.state.Underlying().File(fa)
	}
	_ = chown
	_ = chmod
	return s.cloneWithState(llb.NewState(st)), nil
}

// env implements ENV <key>=<value> …
func (s *Stage) envMethod(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) == 1 {
		dict, ok := args[0].(*starlark.Dict)
		if !ok {
			return nil, fmt.Errorf("env: expected dict or (key, value)")
		}
		ns := s.clone()
		for _, item := range dict.Items() {
			k, ok1 := starlark.AsString(item[0])
			v, ok2 := starlark.AsString(item[1])
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("env: keys and values must be strings")
			}
			ns.env[k] = v
			ns.state = llb.NewState(ns.state.Underlying().AddEnv(k, v))
		}
		return ns, nil
	}
	var key, value string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "key", &key, "value", &value); err != nil {
		return nil, err
	}
	ns := s.clone()
	ns.env[key] = value
	ns.state = llb.NewState(ns.state.Underlying().AddEnv(key, value))
	return ns, nil
}

// workdir implements WORKDIR <path>
func (s *Stage) workdirMethod(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "path", &path); err != nil {
		return nil, err
	}
	ns := s.clone()
	if !strings.HasPrefix(path, "/") {
		path = strings.TrimSuffix(ns.workdir, "/") + "/" + path
	}
	ns.workdir = path
	ns.state = llb.NewState(ns.state.Underlying().Dir(path))
	return ns, nil
}

// user implements USER <user>[:<group>]
func (s *Stage) userMethod(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var name string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "name", &name); err != nil {
		return nil, err
	}
	ns := s.clone()
	ns.user = name
	ns.state = llb.NewState(ns.state.Underlying().User(name))
	return ns, nil
}

// label implements LABEL <key>=<value> …
func (s *Stage) labelMethod(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var key, value string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "key", &key, "value", &value); err != nil {
		return nil, err
	}
	ns := s.clone()
	ns.labels[key] = value
	return ns, nil
}

// expose implements EXPOSE <port>[/<proto>] …
func (s *Stage) exposeMethod(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var port string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "port", &port); err != nil {
		return nil, err
	}
	ns := s.clone()
	ns.expose = append(ns.expose, port)
	return ns, nil
}

// volume implements VOLUME <path> …
func (s *Stage) volumeMethod(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "path", &path); err != nil {
		return nil, err
	}
	ns := s.clone()
	ns.volumes = append(ns.volumes, path)
	return ns, nil
}

// entrypoint implements ENTRYPOINT (exec or shell form)
func (s *Stage) entrypointMethod(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) == 1 {
		if list, ok := args[0].(*starlark.List); ok {
			ns := s.clone()
			ns.entrypoint = listToStrings(list)
			return ns, nil
		}
	}
	var cmd string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "cmd", &cmd); err != nil {
		return nil, err
	}
	ns := s.clone()
	ns.entrypoint = append(s.shell, cmd)
	return ns, nil
}

// cmd implements CMD (exec or shell form)
func (s *Stage) cmdMethod(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) == 1 {
		if list, ok := args[0].(*starlark.List); ok {
			ns := s.clone()
			ns.cmd = listToStrings(list)
			return ns, nil
		}
	}
	var cmd string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "cmd", &cmd); err != nil {
		return nil, err
	}
	ns := s.clone()
	ns.cmd = append(s.shell, cmd)
	return ns, nil
}

// shell implements SHELL ["executable", "param1", …]
func (s *Stage) shellMethod(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("shell expects a list")
	}
	list, ok := args[0].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("shell expects a list")
	}
	ns := s.clone()
	ns.shell = listToStrings(list)
	return ns, nil
}

// ---------- helpers ----------

func (s *Stage) clone() *Stage {
	ns := *s
	ns.env = copyMap(s.env)
	ns.labels = copyMap(s.labels)
	ns.expose = append([]string{}, s.expose...)
	ns.volumes = append([]string{}, s.volumes...)
	ns.entrypoint = append([]string{}, s.entrypoint...)
	ns.cmd = append([]string{}, s.cmd...)
	ns.shell = append([]string{}, s.shell...)
	return &ns
}

func (s *Stage) cloneWithState(st *llb.State) *Stage {
	ns := s.clone()
	ns.state = st
	return ns
}

func copyMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func listToStrings(l *starlark.List) []string {
	out := make([]string, 0, l.Len())
	iter := l.Iterate()
	defer iter.Done()
	var v starlark.Value
	for iter.Next(&v) {
		if s, ok := starlark.AsString(v); ok {
			out = append(out, s)
		} else {
			out = append(out, v.String())
		}
	}
	return out
}

func parseChown(s string) (uid, gid int, err error) {
	parts := strings.SplitN(s, ":", 2)
	uid, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	if len(parts) == 2 {
		gid, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, err
		}
	}
	return uid, gid, nil
}
