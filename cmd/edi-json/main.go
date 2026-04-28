package main

import (
	"context"
	"fmt"
	"os"

	"github.com/openedi/ediforge/internal/cli"
)

func main() {
	if err := cli.Execute(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(cli.ExitCode(err))
	}
}
