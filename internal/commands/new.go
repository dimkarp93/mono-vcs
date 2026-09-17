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

func New(ctx *app.Context) int {
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
		dryrun.New(out, a, local)
		return 0
	}

	defs := resolveDefaults(ctx, local, nil)

	jobs := a.GetJobs()
	fmt.Fprintf(out, "switching to `%s` (created off the default branch where missing, local changes carried over) in %d repo(s) (jobs=%d)\n",
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
			results <- gitops.NewBranchOne(p, branch, main)
		}(p)
	}
	go func() { wg.Wait(); close(results) }()

	created, switched, already, absent, isDefault := 0, 0, 0, 0, 0
	var unresolved []string
	var conflicts [][2]string
	var failed [][2]string
	done, total := 0, len(local)
	for res := range results {
		done++
		switch res.Status {
		case "created", "switched":
			if res.Status == "created" {
				created++
			} else {
				switched++
			}
			line := fmt.Sprintf("[%d/%d] %-10s %s", done, total, res.Status, res.Path)
			if res.Detail != "" {
				line += " — " + res.Detail
			}
			fmt.Fprintln(out, line)
		case "already":
			already++
			fmt.Fprintf(out, "[%d/%d] already    %s — already on `%s`\n", done, total, res.Path, branch)
		case "absent":
			absent++
			fmt.Fprintf(out, "[%d/%d] absent     %s — no local default branch to base on\n", done, total, res.Path)
		case "default":
			isDefault++
			fmt.Fprintf(out, "[%d/%d] default    %s — `%s` is the default branch here\n", done, total, res.Path, branch)
		case "unresolved":
			unresolved = append(unresolved, res.Path)
		case "conflict":
			conflicts = append(conflicts, [2]string{res.Path, res.Detail})
			fmt.Fprintf(out, "[%d/%d] conflict   %s — %s\n", done, total, res.Path, res.Detail)
		default:
			failed = append(failed, [2]string{res.Path, res.Detail})
		}
	}

	fmt.Fprintln(out)
	fmt.Fprintf(out, "created: %d, switched: %d, already there: %d, absent: %d, is-default: %d, unresolved: %d, conflicts: %d, failed: %d\n",
		created, switched, already, absent, isDefault, len(unresolved), len(conflicts), len(failed))
	reportUnresolved(ctx.Stderr, unresolved)
	if len(conflicts) > 0 {
		sort.Slice(conflicts, func(i, j int) bool { return conflicts[i][0] < conflicts[j][0] })
		fmt.Fprintf(ctx.Stderr, "\nlocal changes left stashed — resolve by hand (`git stash list`) (%d):\n", len(conflicts))
		for _, c := range conflicts {
			fmt.Fprintf(ctx.Stderr, "  %s: %s\n", c[0], c[1])
		}
	}
	if len(failed) > 0 {
		sort.Slice(failed, func(i, j int) bool { return failed[i][0] < failed[j][0] })
		fmt.Fprintf(ctx.Stderr, "\nbranch create failed (%d):\n", len(failed))
		for _, f := range failed {
			fmt.Fprintf(ctx.Stderr, "  %s: %s\n", f[0], f[1])
		}
	}
	if len(unresolved) > 0 || len(conflicts) > 0 || len(failed) > 0 {
		return 1
	}
	return 0
}
