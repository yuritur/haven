package main

import (
	"fmt"
	"os"

	"github.com/havenapp/haven/internal/cli"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	return cli.NewRootCmd().Execute()
}
