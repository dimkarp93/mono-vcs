package commands

import (
	"fmt"
	"os"
	"sync"

	"mono-vcs/internal/app"
	"mono-vcs/internal/gitlab"
	"mono-vcs/internal/gitops"
	"mono-vcs/internal/output"
	"mono-vcs/internal/prompts"
	"mono-vcs/internal/repos"
)

type cloneCandidate struct {
	url        string
	target     string
	needsForce bool
}

func Clone(ctx *app.Context) int {
	a := ctx.Args
	out := ctx.Stdout
	if !hasGit() {
		output.Die(ctx.Stderr, "git not found in PATH")
		return 1
	}
	projects, err := gitlab.FetchProjects(a.GetGLURL(), a.GLToken)
	if err != nil {
		output.Die(ctx.Stderr, err.Error())
		return 1
	}
	local := repos.ScanLocalRepos(".")

	var candidates []cloneCandidate
	skipped := 0
	for _, p := range projects {
		target := p.PathWithNamespace
		if local[target] {
			skipped++
			continue
		}
		_, statErr := os.Stat(target)
		candidates = append(candidates, cloneCandidate{p.HTTPURLToRepo, target, statErr == nil})
	}
	if skipped > 0 {
		fmt.Fprintf(out, "skipping %d already-cloned repo(s)\n", skipped)
	}

	type todoItem struct {
		url, target string
		force       bool
	}
	var todo []todoItem
	forced, forceSkipped := 0, 0
	bulk := ""
	pr := prompts.New(ctx.Stdin, ctx.Stdout, ctx.Stderr)
	for _, c := range candidates {
		if !c.needsForce {
			todo = append(todo, todoItem{c.url, c.target, false})
			continue
		}
		choice := bulk
		if choice == "" {
			ch, err := pr.ForceDelete(c.target)
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
			forced++
			todo = append(todo, todoItem{c.url, c.target, true})
		} else {
			forceSkipped++
		}
	}

	if forced > 0 {
		fmt.Fprintf(out, "force-replacing %d non-repo director(y/ies) before cloning\n", forced)
	}
	if forceSkipped > 0 {
		fmt.Fprintf(out, "skipping %d non-repo director(y/ies) (user declined)\n", forceSkipped)
	}
	if len(todo) == 0 {
		fmt.Fprintln(out, "nothing to clone — all remote projects already present locally")
		return 0
	}

	fmt.Fprintf(out, "cloning %d repos (jobs=%d)\n", len(todo), a.GetJobs())
	type cloneResult struct {
		path string
		ok   bool
		err  string
	}
	results := make(chan cloneResult, len(todo))
	sem := make(chan struct{}, a.GetJobs())
	var wg sync.WaitGroup
	for _, t := range todo {
		wg.Add(1)
		sem <- struct{}{}
		go func(t todoItem) {
			defer wg.Done()
			defer func() { <-sem }()
			path, ok, errMsg := gitops.CloneOne(t.url, t.target, a.GLToken, t.force)
			results <- cloneResult{path, ok, errMsg}
		}(t)
	}
	go func() { wg.Wait(); close(results) }()

	failures, done, total := 0, 0, len(todo)
	for r := range results {
		done++
		if r.ok {
			fmt.Fprintf(out, "[%d/%d] cloned %s\n", done, total, r.path)
		} else {
			failures++
			fmt.Fprintf(ctx.Stderr, "[%d/%d] FAILED %s: %s\n", done, total, r.path, r.err)
		}
	}
	if failures > 0 {
		fmt.Fprintf(ctx.Stderr, "%d clone(s) failed\n", failures)
		return 1
	}
	return 0
}
