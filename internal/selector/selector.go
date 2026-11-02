package selector

import (
	"fmt"
	"io"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/gitops"
	"github.com/dimkarp93/mono-vcs/internal/repos"
)

func SelectLocal(ctx *app.Context) []string {
	all := repos.SortedKeys(repos.ScanLocalRepos("."))
	if ctx.Args.Feature != "" {
		return ReposWithFeature(all, ctx.Args.Feature, ctx.Stderr)
	}
	return repos.FilterRepos(all, ctx.Args.Repo, ctx.Stderr)
}

func ReposWithFeature(local []string, feat string, stderr io.Writer) []string {
	var out []string
	for _, p := range local {
		bs, err := gitops.LocalBranches(p)
		if err != nil {
			fmt.Fprintf(stderr, "warning: %s: failed to list branches — %s\n", p, err.Error())
			continue
		}
		for _, b := range bs {
			if b == feat {
				out = append(out, p)
				break
			}
		}
	}
	if len(out) == 0 {
		fmt.Fprintf(stderr, "warning: no local repo has a branch named %q\n", feat)
	}
	return out
}
