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

func Switch(ctx *app.Context) int {
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
		dryrun.Switch(out, a, local)
		return 0
	}

	defs := resolveDefaults(ctx, local, nil)

	jobs := a.GetJobs()
	label := "the default branch"
	if branch != "" {
		label = "`" + branch + "`"
	}
	if branch == "" {
		fmt.Fprintf(out, "switching %d repo(s) to their default branch (jobs=%d)\n", len(local), jobs)
	} else {
		fmt.Fprintf(out, "switching to `%s` (fallback: pull the default branch) in %d repo(s) (jobs=%d)\n", branch, len(local), jobs)
	}

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
			target := branch
			if target == "" {
				if !ok {
					results <- gitops.Result{Path: p, Status: "unresolved", Detail: unresolvedDefaultBranch}
					return
				}
				target = main
			}
			results <- gitops.SwitchOne(p, target, main, a.GLToken)
		}(p)
	}
	go func() { wg.Wait(); close(results) }()

	var unresolved []string
	var dirty []string
	var failed [][2]string
	switched, already, syncedMain, absent := 0, 0, 0, 0
	done, total := 0, len(local)
	for res := range results {
		done++
		switch res.Status {
		case "switched":
			switched++
			fmt.Fprintf(out, "[%d/%d] switched   %s\n", done, total, res.Path)
		case "already":
			already++
			fmt.Fprintf(out, "[%d/%d] already    %s\n", done, total, res.Path)
		case "synced-main":
			syncedMain++
			fmt.Fprintf(out, "[%d/%d] main-sync  %s — %s\n", done, total, res.Path, res.Detail)
		case "absent":
			absent++
			fmt.Fprintf(out, "[%d/%d] absent     %s — no %s to check out\n", done, total, res.Path, label)
		case "unresolved":
			unresolved = append(unresolved, res.Path)
		case "dirty":
			dirty = append(dirty, res.Path)
		default:
			failed = append(failed, [2]string{res.Path, res.Detail})
		}
	}

	fmt.Fprintln(out)
	fmt.Fprintf(out, "switched: %d, already on %s: %d, main-synced: %d, absent: %d, unresolved: %d, dirty: %d, failed: %d\n",
		switched, label, already, syncedMain, absent, len(unresolved), len(dirty), len(failed))
	reportUnresolved(ctx.Stderr, unresolved)
	if len(dirty) > 0 {
		sort.Strings(dirty)
		fmt.Fprintf(ctx.Stderr, "\nskipped due to uncommitted/unstaged changes — could not switch to %s (%d):\n", label, len(dirty))
		for _, p := range dirty {
			fmt.Fprintf(ctx.Stderr, "  %s\n", p)
		}
	}
	if len(failed) > 0 {
		sort.Slice(failed, func(i, j int) bool { return failed[i][0] < failed[j][0] })
		fmt.Fprintf(ctx.Stderr, "\ncheckout/pull failed (%d):\n", len(failed))
		for _, f := range failed {
			fmt.Fprintf(ctx.Stderr, "  %s: %s\n", f[0], f[1])
		}
	}
	if len(unresolved) > 0 || len(dirty) > 0 || len(failed) > 0 {
		return 1
	}
	return 0
}
