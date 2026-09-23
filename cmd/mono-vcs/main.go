package main

import (
	"os"

	"github.com/dimkarp93/install-libs/buildinfo"
	"github.com/dimkarp93/mono-vcs/internal/cli"
)

var (
	version  string
	origin   string
	upstream string
	commit   string
	channel  string
)

func build() buildinfo.Info {
	return buildinfo.Info{
		Version:  version,
		Origin:   origin,
		Upstream: upstream,
		Commit:   commit,
		Channel:  channel,
	}
}

func main() {
	args := os.Args[1:]
	if build().Handle(args) {
		return
	}
	os.Exit(cli.New(os.Stdin, os.Stdout, os.Stderr).Run(args))
}
