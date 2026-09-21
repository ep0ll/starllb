# starllb

**Production-ready Starlark bindings for BuildKit LLB.**

Write BuildKit LLB graphs using the Starlark configuration language.  
Under the hood everything is pure BuildKit `client/llb` — you get the full power of concurrent, cache-efficient builds with a clean, chainable Starlark API.

## Why starllb?

- **Starlark** is the language of Bazel, deterministic, hermetic and safe for configuration.
- **BuildKit LLB** is the intermediate representation that powers Docker Buildx, Earthly, envd, HLB, etc.
- starllb gives you a **first-class, idiomatic Starlark port** of the LLB API so you can define complex multi-stage builds without writing Go.

## Features

- Full method chaining (`state.run(...).add_mount(...).root()`)
- Core constructors: `llb.image`, `llb.scratch`, `llb.git`, `llb.local`, `llb.http`
- File operations: `copy`, `mkdir`, `mkfile`, `rm` (both standalone and chained)
- ExecState with `add_mount` / `get_mount` / `root`
- Production-ready Go library + CLI (`starllb`)
- Follows Go best practices (small packages, clear interfaces, no global state)

## Quick start

```bash
go install github.com/ep0ll/starllb/cmd/starllb@latest
```

Create `hello.star`:

```python
def build():
    alpine = llb.image("docker.io/library/alpine:3.20")
    return alpine.run("echo 'Hello from starllb!'").root()
```

Run it:

```bash
starllb run hello.star -o definition.pb
# then feed definition.pb to buildctl / buildkitd
```

## Library usage (Go)

```go
package main

import (
    "github.com/ep0ll/starllb/starlark"
    "go.starlark.net/starlark"
)

func main() {
    rt := starlark.New()
    globals, err := rt.ExecFile("build.star", nil)
    // ...
    result, _ := starlark.Call(rt.thread, globals["build"], nil, nil)
    // convert result to *llb.State and Marshal
}
```

## API surface (Starlark)

```python
# Constructors
st = llb.image("alpine:latest")
st = llb.scratch()
st = llb.git("https://github.com/moby/buildkit", "master")
st = llb.local("context")
st = llb.http("https://example.com/file")

# State methods (chainable)
st = st.add_env("KEY", "value")
st = st.dir("/work")
st = st.user("nobody")
es = st.run("sh -c 'echo hi'")          # returns ExecState
st = es.root()
st = es.add_mount("/src", other_state, readonly=True)

# File ops
st = st.mkdir("/tmp/foo", mode=0o755)
st = st.mkfile("/tmp/bar", "content", mode=0o644)
st = st.copy(src="/from", dest="/to", input=other)
st = st.rm("/path")
```

## Project layout

```
starllb/
├── llb/          # Starlark value types wrapping BuildKit LLB
├── starlark/     # Runtime / interpreter helpers
├── cmd/starllb/  # CLI
├── examples/     # Sample .star scripts
└── README.md
```

## Status

v0.1.0 – core API complete, ready for production use in frontends and tooling.

Contributions welcome.

## License

Apache-2.0
