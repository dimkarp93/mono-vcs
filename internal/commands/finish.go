package commands

import (
	"fmt"
	"sort"
	"sync"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/dryrun"
	"github.com/dimkarp93/mono-vcs/internal/gitops"
	"github.com/dimkarp93/mono-vcs/internal/output"
)

func Finish(ctx *app.Context) int {
	a := ctx.Args
	out := ctx.Stdout
	if !hasGit() {
		output.Die(ctx.Stderr, "git not found in PATH")
		return 1
	}
	branch := a.Branch

	local := selectLocal(ctx)
	if len(local) == 0 {
		fmt.Fprintln(out, "no local git repositories found under current directory")
		return 0
	}
	if a.DryRun {
		dryrun.Finish(out, a, local)
		return 0
	}

	defs := resolveDefaults(ctx, local, nil)

	jobs := a.GetJobs()
	fmt.Fprintf(out, "finishing `%s` (discard local changes, switch to the default branch + pull, delete) in %d repo(s) (jobs=%d)\n",
		branch, len(local), jobs)

	results := make(chan gitops.Result, len(local))
	sem := make(chan struct{}, jobs)
	var wg sync.WaitGroup
	for _, p := range local {
		wg.Add(1)
		sem <- struct{}{}
		go func(p string) {
			defer wg.Done()
			defer func() { <-sem }()
			main, ok := defs.get(p)
			if !ok {
				results <- gitops.Result{Path: p, Status: "unresolved", Detail: unresolvedDefaultBranch}
				return
			}
			if branch == main {
				results <- gitops.Result{Path: p, Status: "default"}
				return
			}
			results <- gitops.FinishOne(p, branch, main, a.GLToken)
		}(p)
	}
	go func() { wg.Wait(); close(results) }()

	deleted, absent, isDefault := 0, 0, 0
	var unresolved []string
	var failed [][2]string
	done, total := 0, len(local)
	for res := range results {
		done++
		switch res.Status {
		case "deleted":
			deleted++
			line := fmt.Sprintf("[%d/%d] deleted    %s", done, total, res.Path)
			if res.Detail != "" {
				line += " — " + res.Detail
			}
			fmt.Fprintln(out, line)
		case "absent":
			absent++
		case "default":
			isDefault++
			fmt.Fprintf(out, "[%d/%d] default    %s — `%s` is the default branch here\n", done, total, res.Path, branch)
		case "unresolved":
			unresolved = append(unresolved, res.Path)
		default:
			failed = append(failed, [2]string{res.Path, res.Detail})
		}
	}

	fmt.Fprintln(out)
	fmt.Fprintf(out, "deleted: %d, branch absent: %d, is-default: %d, unresolved: %d, failed: %d\n",
		deleted, absent, isDefault, len(unresolved), len(failed))
	reportUnresolved(ctx.Stderr, unresolved)
	if len(failed) > 0 {
		sort.Slice(failed, func(i, j int) bool { return failed[i][0] < failed[j][0] })
		fmt.Fprintf(ctx.Stderr, "\nbranch delete failed (%d):\n", len(failed))
		for _, f := range failed {
			fmt.Fprintf(ctx.Stderr, "  %s: %s\n", f[0], f[1])
		}
	}
	if len(unresolved) > 0 || len(failed) > 0 {
		return 1
	}
	return 0
}
