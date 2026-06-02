package commands

import (
	"fmt"
	"sort"
	"sync"

	"mono-vcs/internal/app"
	"mono-vcs/internal/colors"
	"mono-vcs/internal/dryrun"
	"mono-vcs/internal/gitops"
	"mono-vcs/internal/output"
)

func UpdateMain(ctx *app.Context) int {
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
	if a.DryRun {
		dryrun.UpdateMain(out, a, local)
		return 0
	}

	branch := a.GetMainBranch()
	fmt.Fprintf(out, "updating `%s` in %d repos (jobs=%d)\n", branch, len(local), a.GetJobs())

	type umResult struct {
		res  gitops.Result
		orig string
	}
	results := make(chan umResult, len(local))
	sem := make(chan struct{}, a.GetJobs())
	var wg sync.WaitGroup
	for _, p := range local {
		wg.Add(1)
		sem <- struct{}{}
		go func(p string) {
			defer wg.Done()
			defer func() { <-sem }()
			r, orig := gitops.UpdateMainOne(p, a.GLToken, branch)
			results <- umResult{r, orig}
		}(p)
	}
	go func() { wg.Wait(); close(results) }()

	failures, blocked, done, total := 0, 0, 0, len(local)
	var toRebase [][2]string
	for r := range results {
		done++
		switch r.res.Status {
		case "failed":
			failures++
			fmt.Fprintf(ctx.Stderr, "[%d/%d] FAILED     %s: %s\n", done, total, r.res.Path, r.res.Detail)
		case "dirty":
			blocked++
			fmt.Fprintf(ctx.Stderr, "[%d/%d] ERROR      %s: %s\n", done, total, r.res.Path, r.res.Detail)
		default:
			fmt.Fprintf(out, "[%d/%d] %-10s %s\n", done, total, r.res.Status, r.res.Path)
			if r.orig != "" {
				toRebase = append(toRebase, [2]string{r.res.Path, r.orig})
			}
		}
	}

	var conflicts []string
	if len(toRebase) > 0 {
		sort.Slice(toRebase, func(i, j int) bool { return toRebase[i][0] < toRebase[j][0] })
		fmt.Fprintf(out, "\nrebasing %d feature branch(es) onto `%s`\n", len(toRebase), branch)
		for _, pr := range toRebase {
			path, orig := pr[0], pr[1]
			fmt.Fprintf(out, "  → %s: checkout %s + rebase onto %s\n", path, orig, branch)
			status, detail := gitops.RebaseOne(path, orig, branch, ctx.Stdin, out, ctx.Stderr)
			switch status {
			case "rebased":
				fmt.Fprintf(out, "    %srebased%s    %s\n", colors.Green, colors.Reset, detail)
			case "conflict":
				conflicts = append(conflicts, path)
				fmt.Fprintf(ctx.Stderr, "    %sconflict%s   %s\n", colors.Yellow, colors.Reset, detail)
			default:
				failures++
				fmt.Fprintf(ctx.Stderr, "    %sFAILED%s     %s\n", colors.Red, colors.Reset, detail)
			}
		}
	}

	if len(conflicts) > 0 {
		fmt.Fprintf(ctx.Stderr, "\n%d repo(s) left mid-rebase — resolve, then `git rebase --continue` (or `git rebase --abort`):\n", len(conflicts))
		for _, path := range conflicts {
			fmt.Fprintf(ctx.Stderr, "  %s\n", path)
		}
	}

	if failures > 0 || blocked > 0 {
		if blocked > 0 {
			fmt.Fprintf(ctx.Stderr, "%d repo(s) skipped due to uncommitted changes\n", blocked)
		}
		if failures > 0 {
			fmt.Fprintf(ctx.Stderr, "%d update(s) failed\n", failures)
		}
		return 1
	}
	if len(conflicts) > 0 {
		return 1
	}
	return 0
}
