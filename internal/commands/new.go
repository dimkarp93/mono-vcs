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
	fallback := a.GetMainBranch()

	local := selectLocal(ctx)
	if len(local) == 0 {
		fmt.Fprintln(out, "no local git repositories found under current directory")
		return 0
	}
	if a.DryRun {
		dryrun.New(out, a, local)
		return 0
	}

	jobs := a.GetJobs()
	fmt.Fprintf(out, "creating `%s` from the default branch (local only) in %d repo(s) (jobs=%d)\n", branch, len(local), jobs)

	results := make(chan gitops.Result, len(local))
	sem := make(chan struct{}, jobs)
	var wg sync.WaitGroup
	for _, p := range local {
		wg.Add(1)
		sem <- struct{}{}
		go func(p string) {
			defer wg.Done()
			defer func() { <-sem }()
			main, _ := gitops.DefaultBranch(p, fallback)
			if branch == main {
				results <- gitops.Result{Path: p, Status: "default"}
				return
			}
			results <- gitops.NewBranchOne(p, branch, main)
		}(p)
	}
	go func() { wg.Wait(); close(results) }()

	created, exists, absent, isDefault := 0, 0, 0, 0
	var dirty []string
	var failed [][2]string
	done, total := 0, len(local)
	for res := range results {
		done++
		switch res.Status {
		case "created":
			created++
			fmt.Fprintf(out, "[%d/%d] created    %s\n", done, total, res.Path)
		case "exists":
			exists++
			fmt.Fprintf(out, "[%d/%d] exists     %s — already has `%s`\n", done, total, res.Path, branch)
		case "absent":
			absent++
			fmt.Fprintf(out, "[%d/%d] absent     %s — no local default branch to base on\n", done, total, res.Path)
		case "default":
			isDefault++
			fmt.Fprintf(out, "[%d/%d] default    %s — `%s` is the default branch here\n", done, total, res.Path, branch)
		case "dirty":
			dirty = append(dirty, res.Path)
		default:
			failed = append(failed, [2]string{res.Path, res.Detail})
		}
	}

	fmt.Fprintln(out)
	fmt.Fprintf(out, "created: %d, already exists: %d, absent: %d, is-default: %d, dirty: %d, failed: %d\n",
		created, exists, absent, isDefault, len(dirty), len(failed))
	if len(dirty) > 0 {
		sort.Strings(dirty)
		fmt.Fprintf(ctx.Stderr, "\nskipped due to uncommitted changes (%d):\n", len(dirty))
		for _, p := range dirty {
			fmt.Fprintf(ctx.Stderr, "  %s\n", p)
		}
	}
	if len(failed) > 0 {
		sort.Slice(failed, func(i, j int) bool { return failed[i][0] < failed[j][0] })
		fmt.Fprintf(ctx.Stderr, "\nbranch create failed (%d):\n", len(failed))
		for _, f := range failed {
			fmt.Fprintf(ctx.Stderr, "  %s: %s\n", f[0], f[1])
		}
	}
	if len(dirty) > 0 || len(failed) > 0 {
		return 1
	}
	return 0
}
