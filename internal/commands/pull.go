package commands

import (
	"fmt"
	"sync"

	"mono-vcs/internal/app"
	"mono-vcs/internal/dryrun"
	"mono-vcs/internal/gitlab"
	"mono-vcs/internal/gitops"
	"mono-vcs/internal/output"
	"mono-vcs/internal/repos"
)

func classifyForPull(p *gitlab.Project, localPath, glURL, token, branch string) string {
	if cur := gitops.CurrentBranch(localPath); cur != "" && cur != branch {
		return "feature"
	}
	if p == nil {
		return "red"
	}
	inSync, known := mainInSync(*p, localPath, glURL, token, branch)
	if known && !inSync {
		return "yellow"
	}
	return "green"
}

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

	projects, err := gitlab.FetchProjects(a.GetGLURL(), a.GLToken)
	if err != nil {
		output.Die(ctx.Stderr, err.Error())
		return 1
	}
	allPaths := make([]string, 0, len(projects))
	for _, p := range projects {
		allPaths = append(allPaths, p.PathWithNamespace)
	}
	prefix := repos.CommonTopGroup(allPaths)
	byPath := map[string]gitlab.Project{}
	for _, p := range projects {
		byPath[repos.StripPrefix(p.PathWithNamespace, prefix)] = p
	}

	branch := a.GetMainBranch()
	jobs := a.GetJobs()
	fmt.Fprintf(out, "checking `%s` status of %d local repo(s) against GitLab...\n", branch, len(local))

	colorOf := map[string]string{}
	var mu sync.Mutex
	sem := make(chan struct{}, jobs)
	var wg sync.WaitGroup
	for _, p := range local {
		wg.Add(1)
		sem <- struct{}{}
		go func(p string) {
			defer wg.Done()
			defer func() { <-sem }()
			var proj *gitlab.Project
			if pr, ok := byPath[p]; ok {
				proj = &pr
			}
			c := classifyForPull(proj, p, a.GetGLURL(), a.GLToken, branch)
			mu.Lock()
			colorOf[p] = c
			mu.Unlock()
		}(p)
	}
	wg.Wait()

	var yellow, green, red, feature []string
	for _, p := range local {
		switch colorOf[p] {
		case "yellow":
			yellow = append(yellow, p)
		case "green":
			green = append(green, p)
		case "red":
			red = append(red, p)
		case "feature":
			feature = append(feature, p)
		}
	}
	if len(green) > 0 {
		fmt.Fprintf(out, "  skipping %d green repo(s) — local `%s` already matches remote\n", len(green), branch)
	}
	if len(red) > 0 {
		fmt.Fprintf(out, "  skipping %d red repo(s) — not visible in GitLab, nothing to pull from\n", len(red))
	}
	if len(feature) > 0 {
		fmt.Fprintf(out, "  skipping %d repo(s) — not on `%s` branch\n", len(feature), branch)
	}
	if len(yellow) == 0 {
		fmt.Fprintln(out, "nothing to pull")
		return 0
	}

	fmt.Fprintf(out, "pulling %d yellow repo(s) (jobs=%d)\n", len(yellow), jobs)
	results := make(chan gitops.Result, len(yellow))
	sem2 := make(chan struct{}, jobs)
	var wg2 sync.WaitGroup
	for _, p := range yellow {
		wg2.Add(1)
		sem2 <- struct{}{}
		go func(p string) {
			defer wg2.Done()
			defer func() { <-sem2 }()
			results <- gitops.PullOne(p, a.GLToken)
		}(p)
	}
	go func() { wg2.Wait(); close(results) }()

	failures, done, total := 0, 0, len(yellow)
	for res := range results {
		done++
		if res.Status == "failed" {
			failures++
			fmt.Fprintf(ctx.Stderr, "[%d/%d] FAILED %s: %s\n", done, total, res.Path, res.Detail)
		} else {
			fmt.Fprintf(out, "[%d/%d] %-10s %s\n", done, total, res.Status, res.Path)
		}
	}
	if failures > 0 {
		fmt.Fprintf(ctx.Stderr, "%d pull(s) failed\n", failures)
		return 1
	}
	return 0
}
