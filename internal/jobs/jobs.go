package jobs

import (
	"fmt"
	"os/exec"
	"sync"

	"mono-vcs/internal/app"
	"mono-vcs/internal/gitops"
	"mono-vcs/internal/output"
	"mono-vcs/internal/repos"
)

type Worker func(path string) gitops.Result

func RunPerRepo(ctx *app.Context, label string, worker Worker, successStates map[string]bool) int {
	// successStates is currently unused; only "failed" counts as failure here.
	_ = successStates
	if _, err := exec.LookPath("git"); err != nil {
		output.Die(ctx.Stderr, "git not found in PATH")
		return 1
	}
	local := repos.SortedKeys(repos.ScanLocalRepos("."))
	local = repos.FilterRepos(local, ctx.Args.Repo, ctx.Stderr)
	if len(local) == 0 {
		fmt.Fprintln(ctx.Stdout, "no local git repositories found under current directory")
		return 0
	}
	fmt.Fprintf(ctx.Stdout, "%s in %d repos (jobs=%d)\n", label, len(local), ctx.Args.GetJobs())

	results := make(chan gitops.Result, len(local))
	sem := make(chan struct{}, ctx.Args.GetJobs())
	var wg sync.WaitGroup
	for _, p := range local {
		wg.Add(1)
		sem <- struct{}{}
		go func(p string) {
			defer wg.Done()
			defer func() { <-sem }()
			results <- worker(p)
		}(p)
	}
	go func() { wg.Wait(); close(results) }()

	failures, done, total := 0, 0, len(local)
	for res := range results {
		done++
		if res.Status == "failed" {
			failures++
			fmt.Fprintf(ctx.Stderr, "[%d/%d] FAILED     %s: %s\n", done, total, res.Path, res.Detail)
			continue
		}
		line := fmt.Sprintf("[%d/%d] %-10s %s", done, total, res.Status, res.Path)
		if res.Detail != "" {
			line += " — " + res.Detail
		}
		fmt.Fprintln(ctx.Stdout, line)
	}
	if failures > 0 {
		fmt.Fprintf(ctx.Stderr, "%d %s(s) failed\n", failures, label)
		return 1
	}
	return 0
}
