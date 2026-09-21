// Command starllb executes Starlark scripts that produce BuildKit LLB definitions.
//
// Usage:
//
//	starllb run script.star          # execute and print definition summary
//	starllb run script.star -o out.pb # write binary definition
//	starllb version
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ep0ll/starllb/llb"
	starllbruntime "github.com/ep0ll/starllb/starlark"
	bkllb "github.com/moby/buildkit/client/llb"
	"go.starlark.net/starlark"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		runCmd(os.Args[2:])
	case "version":
		fmt.Println("starllb", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `starllb - Starlark bindings for BuildKit LLB

Usage:
  starllb run <script.star> [flags]
  starllb version

Flags for run:
  -o, --output   Write LLB definition (protobuf) to file
  -f, --func     Function name to call (default: "build")

Example script.star:

  def build():
      alpine = llb.image("docker.io/library/alpine:latest")
      return alpine.run("echo hello").root()
`)
}

func runCmd(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	output := fs.String("o", "", "output file for LLB definition")
	funcName := fs.String("f", "build", "function to invoke")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "missing script path")
		os.Exit(1)
	}
	scriptPath := fs.Arg(0)

	data, err := os.ReadFile(scriptPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read script: %v\n", err)
		os.Exit(1)
	}

	rt := starllbruntime.New()
	globals, err := rt.ExecFile(filepath.Base(scriptPath), data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "exec: %v\n", err)
		os.Exit(1)
	}

	fnVal, ok := globals[*funcName]
	if !ok {
		fmt.Fprintf(os.Stderr, "function %q not found in script\n", *funcName)
		os.Exit(1)
	}
	fn, ok := fnVal.(starlark.Callable)
	if !ok {
		fmt.Fprintf(os.Stderr, "%q is not callable\n", *funcName)
		os.Exit(1)
	}

	thread := &starlark.Thread{Name: "starllb-run"}
	result, err := starlark.Call(thread, fn, nil, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "call %s: %v\n", *funcName, err)
		os.Exit(1)
	}

	var state bkllb.State
	switch v := result.(type) {
	case *llb.State:
		state = v.Underlying()
	case *llb.ExecState:
		state = v.Underlying().Root()
	default:
		fmt.Fprintf(os.Stderr, "result must be llb.State or llb.ExecState, got %s\n", result.Type())
		os.Exit(1)
	}

	ctx := context.Background()
	def, err := state.Marshal(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal: %v\n", err)
		os.Exit(1)
	}

	if *output != "" {
		f, err := os.Create(*output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "create output: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		if err := bkllb.WriteTo(def, f); err != nil {
			fmt.Fprintf(os.Stderr, "write: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("wrote LLB definition to %s\n", *output)
	} else {
		fmt.Printf("LLB definition generated successfully\n")
		fmt.Println("Use -o <file> to write the binary definition.")
	}
}
