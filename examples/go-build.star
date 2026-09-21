# go-build.star - build a Go binary using LLB via Starlark
# Demonstrates chaining, env, dir, mounts and file ops.

def build():
    # Base image with Go toolchain
    go = llb.image("docker.io/library/golang:1.22-alpine")
    go = go.add_env("CGO_ENABLED", "0")
    go = go.add_env("GOOS", "linux")
    go = go.add_env("GOARCH", "amd64")

    # Create a simple main.go via file ops
    main_go = '''
package main

import "fmt"

func main() {
    fmt.Println("Built with starllb + BuildKit LLB")
}
'''
    work = llb.scratch().mkfile("/src/main.go", main_go, mode=0644)

    # Run the build
    built = go.dir("/src").run(
        "go build -o /out/hello /src/main.go"
    ).add_mount("/src", work).add_mount("/out", llb.scratch()).root()

    return built
