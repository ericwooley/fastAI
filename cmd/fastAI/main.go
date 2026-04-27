package main

import (
	"context"
	"os"

	"github.com/ericwooley/fastAI/internal/cli"
)

func main() {
	os.Exit(cli.Execute(context.Background(), os.Args[1:], cli.DefaultDependencies(os.Stdout, os.Stderr)))
}
