package main

import (
	"fmt"
	"os"

	"mono-vcs/internal/cli"
)

// version is the bare semver, injected at build time via
// -ldflags "-X main.version=<ver>" (see Makefile / release workflow).
var version = "dev"

func main() {
	args := os.Args[1:]
	if len(args) > 0 && (args[0] == "--version" || args[0] == "-v" || args[0] == "version") {
		fmt.Println(version)
		return
	}
	os.Exit(cli.New(os.Stdin, os.Stdout, os.Stderr).Run(args))
}
