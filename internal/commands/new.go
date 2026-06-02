package commands

import (
	"fmt"
	"sort"
	"sync"

	"mono-vcs/internal/app"
	"mono-vcs/internal/dryrun"
	"mono-vcs/internal/gitops"
	"mono-vcs/internal/output"
)

func New(ctx *app.Context) int {
	a := ctx.Args
	out := ctx.Stdout
	if !hasGit() {
		output.Die(ctx.Stderr, "git not found in PATH")
		return 1
	}
	branch := a.Branch
	main := a.GetMainBranch()
	if branch == main {
		output.Die(ctx.Stderr, fmt.Sprintf("refusing to create `%s` — it is the configured main branch", branch))
		return 1
	}

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
	fmt.Fprintf(out, "creating `%s` from `%s` (local only) in %d repo(s) (jobs=%d)\n", branch, main, len(local), jobs)

	results := make(chan gitops.Result, len(local))
	sem := make(chan struct{}, jobs)
	var wg sync.WaitGroup
	for _, p := range local {
		wg.Add(1)
		sem <- struct{}{}
		go func(p string) {
			defer wg.Done()
			defer func() { <-sem }()
			results <- gitops.NewBranchOne(p, branch, main)
		}(p)
	}
	go func() { wg.Wait(); close(results) }()

	created, exists, absent := 0, 0, 0
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
			fmt.Fprintf(out, "[%d/%d] absent     %s — no local `%s` to base on\n", done, total, res.Path, main)
		case "dirty":
			dirty = append(dirty, res.Path)
		default:
			failed = append(failed, [2]string{res.Path, res.Detail})
		}
	}

	fmt.Fprintln(out)
	fmt.Fprintf(out, "created: %d, already exists: %d, absent: %d, dirty: %d, failed: %d\n",
		created, exists, absent, len(dirty), len(failed))
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
