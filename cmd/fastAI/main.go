package main

import (
	"context"
	"os"

	"github.com/ericwooley/fastAI/internal/cli"
)

var version = "dev"

func main() {
	dependencies := cli.DefaultDependencies(os.Stdout, os.Stderr)
	dependencies.Version = version
	os.Exit(cli.Execute(context.Background(), os.Args[1:], dependencies))
}
