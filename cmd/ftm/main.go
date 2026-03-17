package main

import (
	"fmt"
	"os"

	"foundry-tunnel/internal/app"
	"foundry-tunnel/internal/version"
)

var Version string

func main() {
	if Version == "" {
		Version = version.Version
	}

	application, err := app.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := application.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
