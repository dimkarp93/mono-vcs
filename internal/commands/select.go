package commands

import (
	"fmt"
	"io"

	"mono-vcs/internal/app"
	"mono-vcs/internal/gitops"
	"mono-vcs/internal/repos"
)

func selectLocal(ctx *app.Context) []string {
	all := repos.SortedKeys(repos.ScanLocalRepos("."))
	if ctx.Args.Feature != "" {
		return reposWithFeature(all, ctx.Args.Feature, ctx.Stderr)
	}
	return repos.FilterRepos(all, ctx.Args.Repo, ctx.Stderr)
}

func reposWithFeature(local []string, feat string, stderr io.Writer) []string {
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
