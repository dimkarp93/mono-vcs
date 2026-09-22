package commands

import (
	"io"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/selector"
)

func selectLocal(ctx *app.Context) []string {
	return selector.SelectLocal(ctx)
}

func reposWithFeature(local []string, feat string, stderr io.Writer) []string {
	return selector.ReposWithFeature(local, feat, stderr)
}
