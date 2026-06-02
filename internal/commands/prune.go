package commands

import (
	"fmt"
	"sync"

	"mono-vcs/internal/app"
	"mono-vcs/internal/colors"
	"mono-vcs/internal/dryrun"
	"mono-vcs/internal/gitops"
	"mono-vcs/internal/output"
	"mono-vcs/internal/prompts"
)

func Prune(ctx *app.Context) int {
	a := ctx.Args
	out := ctx.Stdout
	if a.DryRun {
		local := selectLocal(ctx)
		dryrun.Prune(out, a, local)
		return 0
	}
	if !hasGit() {
		output.Die(ctx.Stderr, "git not found in PATH")
		return 1
	}
	local := selectLocal(ctx)
	if len(local) == 0 {
		fmt.Fprintln(out, "no local git repositories found under current directory")
		return 0
	}
	useColor := colors.Enabled()

	type plan struct {
		path    string
		targets []string
	}
	var plans []plan
	failures := 0
	for _, p := range local {
		targets, err := gitops.PruneTargets(p)
		if err != nil {
			failures++
			fmt.Fprintf(ctx.Stderr, "%s: failed — %s\n", colors.Colorize(p, colors.Red, useColor), err.Error())
			continue
		}
		if len(targets) > 0 {
			plans = append(plans, plan{p, targets})
		}
	}
	if len(plans) == 0 {
		if failures > 0 {
			fmt.Fprintf(ctx.Stderr, "%d repo(s) failed to inspect\n", failures)
			return 1
		}
		fmt.Fprintln(out, "nothing to prune — no uncommitted changes under current directory")
		return 0
	}

	pr := prompts.New(ctx.Stdin, ctx.Stdout, ctx.Stderr)
	var todo []string
	declined := 0
	bulk := ""
	for _, pl := range plans {
		fmt.Fprintln(out, colors.Colorize(pl.path, colors.Green, useColor))
		for _, t := range pl.targets {
			fmt.Fprintf(out, "  %s\n", t)
		}
		if a.Yes {
			todo = append(todo, pl.path)
			continue
		}
		choice := bulk
		if choice == "" {
			ch, err := pr.Prune(pl.path, len(pl.targets))
			if err != nil {
				output.Die(ctx.Stderr, err.Error())
				return 1
			}
			choice = ch
		}
		switch choice {
		case "yes-all":
			bulk = "yes-all"
			choice = "yes"
		case "skip-all":
			bulk = "skip-all"
			choice = "skip"
		}
		if choice == "yes" {
			todo = append(todo, pl.path)
		} else {
			declined++
		}
	}

	if declined > 0 {
		fmt.Fprintf(out, "skipping %d repo(s) (user declined)\n", declined)
	}
	if len(todo) == 0 {
		fmt.Fprintln(out, "nothing pruned")
		if failures > 0 {
			return 1
		}
		return 0
	}

	fmt.Fprintf(out, "pruning %d repos (jobs=%d)\n", len(todo), a.GetJobs())
	results := make(chan gitops.Result, len(todo))
	sem := make(chan struct{}, a.GetJobs())
	var wg sync.WaitGroup
	for _, p := range todo {
		wg.Add(1)
		sem <- struct{}{}
		go func(p string) {
			defer wg.Done()
			defer func() { <-sem }()
			results <- gitops.PruneOne(p)
		}(p)
	}
	go func() { wg.Wait(); close(results) }()

	done, total := 0, len(todo)
	for res := range results {
		done++
		if res.Status == "failed" {
			failures++
			fmt.Fprintf(ctx.Stderr, "[%d/%d] FAILED     %s: %s\n", done, total, res.Path, res.Detail)
		} else {
			line := fmt.Sprintf("[%d/%d] %-10s %s", done, total, res.Status, res.Path)
			if res.Detail != "" {
				line += " — " + res.Detail
			}
			fmt.Fprintln(out, line)
		}
	}
	if failures > 0 {
		fmt.Fprintf(ctx.Stderr, "%d prune(s) failed\n", failures)
		return 1
	}
	return 0
}
