# hello.star - minimal Starlark LLB example
# Run with: starllb run examples/hello.star

def build():
    """Return a simple alpine-based state that prints hello."""
    alpine = llb.image("docker.io/library/alpine:3.20")
    return alpine.run("echo 'Hello from starllb!'").root()
