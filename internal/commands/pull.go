package commands

import (
	"fmt"
	"sort"
	"sync"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/dryrun"
	"github.com/dimkarp93/mono-vcs/internal/gitlab"
	"github.com/dimkarp93/mono-vcs/internal/gitops"
	"github.com/dimkarp93/mono-vcs/internal/output"
)

func Pull(ctx *app.Context) int {
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
		dryrun.Pull(out, a, local)
		return 0
	}

	byPath := map[string]gitlab.Project{}
	if projects, err := gitlab.FetchProjects(a.GetGLURL(), a.GLToken); err != nil {
		fmt.Fprintf(ctx.Stderr, "warning: %s — falling back to the state db\n", err.Error())
	} else {
		byPath = projectsByPath(projects)
	}

	defs := resolveDefaults(ctx, local, byPath)
	if defs.refreshed > 0 {
		fmt.Fprintf(out, "refreshed the default branch of %d repo(s) from GitLab\n", defs.refreshed)
	}

	var targets []string
	for _, p := range local {
		if _, ok := defs.get(p); ok {
			targets = append(targets, p)
		}
	}
	unresolved := defs.missing
	if len(targets) == 0 {
		fmt.Fprintln(out, "nothing to pull")
		if len(unresolved) > 0 {
			reportUnresolved(ctx.Stderr, unresolved)
			return 1
		}
		return 0
	}

	jobs := a.GetJobs()
	fmt.Fprintf(out, "fetching the default branch of %d repo(s) (jobs=%d)\n", len(targets), jobs)

	results := make(chan gitops.Result, len(targets))
	sem := make(chan struct{}, jobs)
	var wg sync.WaitGroup
	for _, p := range targets {
		wg.Add(1)
		sem <- struct{}{}
		go func(p string) {
			defer wg.Done()
			defer func() { <-sem }()
			branch, _ := defs.get(p)
			results <- gitops.PullDefaultOne(p, a.GLToken, branch)
		}(p)
	}
	go func() { wg.Wait(); close(results) }()

	updated, upToDate := 0, 0
	var diverged, dirty []string
	var failed [][2]string
	done, total := 0, len(targets)
	for res := range results {
		done++
		switch res.Status {
		case "updated":
			updated++
			fmt.Fprintf(out, "[%d/%d] updated    %s — %s\n", done, total, res.Path, res.Detail)
		case "up-to-date":
			upToDate++
			fmt.Fprintf(out, "[%d/%d] up-to-date %s\n", done, total, res.Path)
		case "diverged":
			diverged = append(diverged, res.Path)
		case "dirty":
			dirty = append(dirty, res.Path)
		default:
			failed = append(failed, [2]string{res.Path, res.Detail})
		}
	}

	fmt.Fprintln(out)
	fmt.Fprintf(out, "updated: %d, up-to-date: %d, diverged: %d, dirty: %d, failed: %d, unresolved: %d\n",
		updated, upToDate, len(diverged), len(dirty), len(failed), len(unresolved))

	if len(diverged) > 0 {
		sort.Strings(diverged)
		fmt.Fprintf(ctx.Stderr, "\nskipped — the local default branch has diverged from origin (%d):\n", len(diverged))
		for _, p := range diverged {
			fmt.Fprintf(ctx.Stderr, "  %s\n", p)
		}
	}
	if len(dirty) > 0 {
		sort.Strings(dirty)
		fmt.Fprintf(ctx.Stderr, "\nskipped — uncommitted changes block the fast-forward (%d):\n", len(dirty))
		for _, p := range dirty {
			fmt.Fprintf(ctx.Stderr, "  %s\n", p)
		}
	}
	if len(failed) > 0 {
		sort.Slice(failed, func(i, j int) bool { return failed[i][0] < failed[j][0] })
		fmt.Fprintf(ctx.Stderr, "\nfetch failed (%d):\n", len(failed))
		for _, f := range failed {
			fmt.Fprintf(ctx.Stderr, "  %s: %s\n", f[0], f[1])
		}
	}
	reportUnresolved(ctx.Stderr, unresolved)

	if len(failed) > 0 || len(diverged) > 0 || len(dirty) > 0 || len(unresolved) > 0 {
		return 1
	}
	return 0
}
