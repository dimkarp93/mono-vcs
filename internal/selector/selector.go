package selector

import (
	"fmt"
	"io"

	"github.com/dimkarp93/mono-vcs/internal/aliases"
	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/gitops"
	"github.com/dimkarp93/mono-vcs/internal/remotes"
	"github.com/dimkarp93/mono-vcs/internal/repos"
)

func SelectLocal(ctx *app.Context) []string {
	all := repos.SortedKeys(repos.ScanLocalRepos("."))
	var picked []string
	if ctx.Args.Feature != "" {
		picked = ReposWithFeature(all, ctx.Args.Feature, ctx.Stderr)
	} else {
		picked = repos.FilterRepos(all, ctx.Args.Repo, ctx.Stderr)
	}
	return FilterByRemote(ctx, picked)
}

func FilterByRemote(ctx *app.Context, paths []string) []string {
	if len(ctx.Args.Remote) == 0 {
		return paths
	}
	set, custom := RemoteContext(ctx, paths)
	return set.Match(paths, ctx.Args.Remote, custom, ctx.Stderr)
}

func RemoteContext(ctx *app.Context, paths []string) (*remotes.Set, map[string]string) {
	return remotes.New(paths, ctx.Args.GetJobs(), ctx.Stderr), CustomAliases(ctx)
}

func CustomAliases(ctx *app.Context) map[string]string {
	st, err := aliases.Open(ctx.Args.GetAliasesPath())
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "warning: %s\n", err.Error())
	}
	return st.All(aliases.KindRemote)
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
