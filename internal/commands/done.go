package commands

import (
	"fmt"
	"sort"
	"sync"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/colors"
	"github.com/dimkarp93/mono-vcs/internal/dryrun"
	"github.com/dimkarp93/mono-vcs/internal/gitops"
	"github.com/dimkarp93/mono-vcs/internal/output"
	"github.com/dimkarp93/mono-vcs/internal/repos"
)

type donePlan struct {
	path     string
	main     string
	branches []string
}

type doneScan struct {
	plans      []donePlan
	open       int
	dirty      int
	unresolved []string
	fetchFail  [][2]string
}

func Done(ctx *app.Context) int {
	a := ctx.Args
	out := ctx.Stdout
	if !hasGit() {
		output.Die(ctx.Stderr, "git not found in PATH")
		return 1
	}

	local := repos.SortedKeys(repos.ScanLocalRepos("."))
	if len(local) == 0 {
		fmt.Fprintln(out, "no local git repositories found under current directory")
		return 0
	}
	if a.DryRun {
		dryrun.Done(out, a, local)
		return 0
	}

	defs := resolveDefaults(ctx, local, nil)
	scan := scanDone(ctx, local, defs)

	reportUnresolved(ctx.Stderr, scan.unresolved)
	if len(scan.fetchFail) > 0 {
		sort.Slice(scan.fetchFail, func(i, j int) bool { return scan.fetchFail[i][0] < scan.fetchFail[j][0] })
		fmt.Fprintf(ctx.Stderr, "\nskipped — fetch failed (%d):\n", len(scan.fetchFail))
		for _, f := range scan.fetchFail {
			fmt.Fprintf(ctx.Stderr, "  %s: %s\n", f[0], f[1])
		}
	}
	badScan := len(scan.unresolved) > 0 || len(scan.fetchFail) > 0

	if len(scan.plans) == 0 {
		fmt.Fprintf(out, "nothing to clean up — still open: %d, dirty: %d\n", scan.open, scan.dirty)
		if badScan {
			return 1
		}
		return 0
	}

	printDonePlans(ctx, scan.plans)
	deleted, failed := deleteDone(ctx, scan.plans)

	fmt.Fprintln(out)
	fmt.Fprintf(out, "deleted: %d, still open: %d, dirty: %d, unresolved: %d, fetch failed: %d, failed: %d\n",
		deleted, scan.open, scan.dirty, len(scan.unresolved), len(scan.fetchFail), len(failed))
	if len(failed) > 0 {
		sort.Slice(failed, func(i, j int) bool { return failed[i][0] < failed[j][0] })
		fmt.Fprintf(ctx.Stderr, "\nbranch delete failed (%d):\n", len(failed))
		for _, f := range failed {
			fmt.Fprintf(ctx.Stderr, "  %s: %s\n", f[0], f[1])
		}
	}
	if badScan || len(failed) > 0 {
		return 1
	}
	return 0
}

func scanDone(ctx *app.Context, local []string, defs defaultBranches) doneScan {
	a := ctx.Args
	results := make(chan doneScan, len(local))
	sem := make(chan struct{}, a.GetJobs())
	var wg sync.WaitGroup
	for _, p := range local {
		wg.Add(1)
		sem <- struct{}{}
		go func(p string) {
			defer wg.Done()
			defer func() { <-sem }()
			results <- scanOne(p, defs, a.GLToken)
		}(p)
	}
	go func() { wg.Wait(); close(results) }()

	var scan doneScan
	for res := range results {
		scan.open += res.open
		scan.dirty += res.dirty
		scan.unresolved = append(scan.unresolved, res.unresolved...)
		scan.fetchFail = append(scan.fetchFail, res.fetchFail...)
		scan.plans = append(scan.plans, res.plans...)
	}
	sort.Slice(scan.plans, func(i, j int) bool { return scan.plans[i].path < scan.plans[j].path })
	return scan
}

func scanOne(path string, defs defaultBranches, token string) (res doneScan) {
	main, ok := defs.get(path)
	if !ok {
		res.unresolved = []string{path}
		return res
	}
	if err := gitops.FetchPrune(path, token); err != nil {
		res.fetchFail = [][2]string{{path, err.Error()}}
		return res
	}
	target := gitops.DefaultCompareRef(path, main)
	if target == "" {
		res.fetchFail = [][2]string{{path, fmt.Sprintf("no `%s` to compare against", main)}}
		return res
	}
	branches, err := gitops.LocalBranches(path)
	if err != nil {
		res.fetchFail = [][2]string{{path, err.Error()}}
		return res
	}

	current := gitops.CurrentBranch(path)
	dirtyTree := gitops.IsDirty(path)
	plan := donePlan{path: path, main: main}
	for _, b := range branches {
		if b == "" || b == main {
			continue
		}
		if b == current && dirtyTree {
			res.dirty++
			continue
		}
		if !gitops.MergedInto(path, b, target) {
			res.open++
			continue
		}
		plan.branches = append(plan.branches, b)
	}
	if len(plan.branches) > 0 {
		sort.Strings(plan.branches)
		res.plans = []donePlan{plan}
	}
	return res
}

func printDonePlans(ctx *app.Context, plans []donePlan) {
	out := ctx.Stdout
	useColor := colors.Enabled()
	for _, pl := range plans {
		fmt.Fprintln(out, colors.Colorize(pl.path, colors.Green, useColor))
		for _, b := range pl.branches {
			fmt.Fprintf(out, "  %s — merged into %s\n", b, pl.main)
		}
	}
}

func deleteDone(ctx *app.Context, todo []donePlan) (int, [][2]string) {
	a := ctx.Args
	out := ctx.Stdout

	type repoOutcome struct {
		path    string
		deleted []string
		failed  [][2]string
	}
	results := make(chan repoOutcome, len(todo))
	sem := make(chan struct{}, a.GetJobs())
	var wg sync.WaitGroup
	for _, pl := range todo {
		wg.Add(1)
		sem <- struct{}{}
		go func(pl donePlan) {
			defer wg.Done()
			defer func() { <-sem }()
			oc := repoOutcome{path: pl.path}
			for _, b := range pl.branches {
				r := gitops.DeleteMergedOne(pl.path, b, pl.main, a.GLToken)
				if r.Status == "deleted" {
					oc.deleted = append(oc.deleted, b)
					continue
				}
				oc.failed = append(oc.failed, [2]string{pl.path + ": " + b, r.Detail})
			}
			results <- oc
		}(pl)
	}
	go func() { wg.Wait(); close(results) }()

	deleted := 0
	var failed [][2]string
	done, total := 0, len(todo)
	for oc := range results {
		done++
		deleted += len(oc.deleted)
		failed = append(failed, oc.failed...)
		if len(oc.deleted) > 0 {
			fmt.Fprintf(out, "[%d/%d] deleted    %s — %v\n", done, total, oc.path, oc.deleted)
		}
	}
	return deleted, failed
}
