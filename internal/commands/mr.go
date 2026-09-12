package commands

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/dryrun"
	"github.com/dimkarp93/mono-vcs/internal/gitlab"
	"github.com/dimkarp93/mono-vcs/internal/gitops"
	"github.com/dimkarp93/mono-vcs/internal/output"
	"github.com/dimkarp93/mono-vcs/internal/repos"
)

type featureScan struct {
	reposByBranch map[string][]string
	current       map[string]string
	mainByPath    map[string]string
	unresolved    []string
}

func scanFeatures(ctx *app.Context, local []string, defs defaultBranches) featureScan {
	s := featureScan{
		reposByBranch: map[string][]string{},
		current:       map[string]string{},
		mainByPath:    map[string]string{},
	}
	for _, p := range local {
		main, ok := defs.get(p)
		if !ok {
			s.unresolved = append(s.unresolved, p)
			continue
		}
		branches, err := gitops.LocalBranches(p)
		if err != nil {
			fmt.Fprintf(ctx.Stderr, "warning: %s: failed to list branches — %s\n", p, err.Error())
			continue
		}
		s.mainByPath[p] = main
		s.current[p] = gitops.CurrentBranch(p)
		for _, b := range branches {
			if b == "" || b == main {
				continue
			}
			s.reposByBranch[b] = append(s.reposByBranch[b], p)
		}
	}
	return s
}

func (s featureScan) reposOn(branch string) ([]string, error) {
	rs := append([]string{}, s.reposByBranch[branch]...)
	if len(rs) == 0 {
		return nil, fmt.Errorf("no local repo has a branch named `%s`", branch)
	}
	sort.Strings(rs)
	var notOn []string
	for _, p := range rs {
		if s.current[p] != branch {
			notOn = append(notOn, p)
		}
	}
	if len(notOn) > 0 {
		return nil, fmt.Errorf("`%s` is not checked out in every repo that has it — run `mono-vcs switch %s` first:\n  %s",
			branch, branch, strings.Join(notOn, "\n  "))
	}
	return rs, nil
}

func (s featureScan) inferFeature() (string, []string, error) {
	active := map[string]bool{}
	for p, cur := range s.current {
		if cur == "" || cur == s.mainByPath[p] {
			continue
		}
		active[cur] = true
	}
	names := sortedSet(active)
	switch len(names) {
	case 0:
		return "", nil, fmt.Errorf("every repo is on its default branch — pass the feature explicitly: `mono-vcs mr <branch>`")
	case 1:
		rs, err := s.reposOn(names[0])
		if err != nil {
			return "", nil, err
		}
		return names[0], rs, nil
	default:
		return "", nil, fmt.Errorf("several feature branches are checked out (%s) — pass the one to publish: `mono-vcs mr <branch>`",
			strings.Join(names, ", "))
	}
}

func MR(ctx *app.Context) int {
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

	var fetched map[string]gitlab.Project
	if a.DryRun {
		fetched = map[string]gitlab.Project{}
	}
	defs := resolveDefaults(ctx, local, fetched)
	scan := scanFeatures(ctx, local, defs)

	var feature string
	var featureRepos []string
	var err error
	if a.Branch != "" {
		feature = a.Branch
		featureRepos, err = scan.reposOn(feature)
	} else {
		feature, featureRepos, err = scan.inferFeature()
	}
	if err != nil {
		output.Die(ctx.Stderr, err.Error())
		reportUnresolved(ctx.Stderr, scan.unresolved)
		return 1
	}

	if a.DryRun {
		dryrun.MR(out, a, feature, featureRepos)
		return 0
	}

	var dirty []string
	for _, p := range featureRepos {
		if gitops.IsDirty(p) {
			dirty = append(dirty, p)
		}
	}
	if len(dirty) > 0 {
		output.Die(ctx.Stderr, fmt.Sprintf("uncommitted changes — commit or stash them so `%s` is published consistently (%d):\n  %s",
			feature, len(dirty), strings.Join(dirty, "\n  ")))
		return 1
	}

	jobs := a.GetJobs()
	fmt.Fprintf(out, "pushing `%s` and opening merge requests in %d repo(s) (jobs=%d)\n", feature, len(featureRepos), jobs)

	results := make(chan gitops.Result, len(featureRepos))
	sem := make(chan struct{}, jobs)
	var wg sync.WaitGroup
	for _, p := range featureRepos {
		wg.Add(1)
		sem <- struct{}{}
		go func(p string) {
			defer wg.Done()
			defer func() { <-sem }()
			results <- gitops.PushMROne(p, feature, a.GLToken, a.Title)
		}(p)
	}
	go func() { wg.Wait(); close(results) }()

	pushed, upToDate := 0, 0
	var failed [][2]string
	done, total := 0, len(featureRepos)
	for res := range results {
		done++
		switch res.Status {
		case "pushed":
			pushed++
			line := fmt.Sprintf("[%d/%d] pushed     %s", done, total, res.Path)
			if res.Detail != "" {
				line += " — " + res.Detail
			}
			fmt.Fprintln(out, line)
		case "up-to-date":
			upToDate++
			fmt.Fprintf(out, "[%d/%d] up-to-date %s — nothing to push, no merge request created\n", done, total, res.Path)
		default:
			failed = append(failed, [2]string{res.Path, res.Detail})
		}
	}

	fmt.Fprintln(out)
	fmt.Fprintf(out, "pushed: %d, up-to-date: %d, unresolved: %d, failed: %d\n",
		pushed, upToDate, len(scan.unresolved), len(failed))
	reportUnresolved(ctx.Stderr, scan.unresolved)
	if len(failed) > 0 {
		sort.Slice(failed, func(i, j int) bool { return failed[i][0] < failed[j][0] })
		fmt.Fprintf(ctx.Stderr, "\npush failed (%d):\n", len(failed))
		for _, f := range failed {
			fmt.Fprintf(ctx.Stderr, "  %s: %s\n", f[0], f[1])
		}
	}
	if len(scan.unresolved) > 0 || len(failed) > 0 {
		return 1
	}
	return 0
}
