# starllb

**Production-ready Starlark bindings for BuildKit LLB.**

Write BuildKit LLB graphs using the Starlark configuration language.  
Under the hood everything is pure BuildKit `client/llb`.

## Quick start

```bash
go install github.com/ep0ll/starllb/cmd/starllb@latest
```

```python
def build():
    alpine = llb.image("docker.io/library/alpine:3.20")
    return alpine.run("echo 'Hello from starllb!'").root()
```

## License

Apache-2.0
