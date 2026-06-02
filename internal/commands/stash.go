package commands

import (
	"fmt"

	"mono-vcs/internal/app"
	"mono-vcs/internal/colors"
	"mono-vcs/internal/dryrun"
	"mono-vcs/internal/gitops"
	"mono-vcs/internal/jobs"
	"mono-vcs/internal/output"
)

func Stash(ctx *app.Context) int {
	if ctx.Args.DryRun {
		dryrun.Stash(ctx.Stdout, ctx.Args, dryRepos(ctx))
		return 0
	}
	return jobs.RunPerRepo(ctx, "stashing", gitops.StashOne, map[string]bool{"stashed": true, "nothing": true})
}

func Unstash(ctx *app.Context) int {
	if ctx.Args.DryRun {
		dryrun.Unstash(ctx.Stdout, ctx.Args, dryRepos(ctx))
		return 0
	}
	return jobs.RunPerRepo(ctx, "popping stash", gitops.UnstashOne, map[string]bool{"popped": true, "nothing": true})
}

func ClearStash(ctx *app.Context) int {
	if ctx.Args.DryRun {
		dryrun.ClearStash(ctx.Stdout, ctx.Args, dryRepos(ctx))
		return 0
	}
	return jobs.RunPerRepo(ctx, "clearing stash", gitops.ClearStashOne, map[string]bool{"cleared": true, "nothing": true})
}

func HistoryStash(ctx *app.Context) int {
	a := ctx.Args
	out := ctx.Stdout
	if !hasGit() {
		output.Die(ctx.Stderr, "git not found in PATH")
		return 1
	}
	local := selectLocal(ctx)
	if len(local) == 0 {
		fmt.Fprintln(out, "no local git repositories found under current directory")
		return 0
	}
	useColor := colors.Enabled(a.GetNoColor())
	total, empty := 0, 0
	for _, p := range local {
		entries, err := gitops.StashList(p)
		if err != nil {
			fmt.Fprintf(ctx.Stderr, "%s: failed — %s\n", colors.Colorize(p, colors.Red, useColor), err.Error())
			continue
		}
		if len(entries) == 0 {
			empty++
			continue
		}
		total += len(entries)
		fmt.Fprintln(out, colors.Colorize(p, colors.Green, useColor))
		for _, line := range entries {
			fmt.Fprintf(out, "  %s\n", line)
		}
	}
	fmt.Fprintln(out)
	noun := "entries"
	if total == 1 {
		noun = "entry"
	}
	fmt.Fprintf(out, "total: %d stash %s across %d repo(s); %d repo(s) had no stash\n",
		total, noun, len(local)-empty, empty)
	return 0
}

func dryRepos(ctx *app.Context) []string {
	return selectLocal(ctx)
}
