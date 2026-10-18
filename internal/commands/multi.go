package commands

import (
	"fmt"
	"io"
	"sort"
	"sync"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/dryrun"
	"github.com/dimkarp93/mono-vcs/internal/gitops"
	"github.com/dimkarp93/mono-vcs/internal/output"
)

type multiSummary struct {
	moved, skipped, ide int
	unresolved          []string
	failed              [][2]string
}

func Multi(ctx *app.Context) int {
	local, code, ok := multiTargets(ctx)
	if !ok {
		return code
	}

	out := ctx.Stdout
	defs := resolveDefaults(ctx, local, nil)
	jobs := ctx.Args.GetJobs()
	fmt.Fprintf(out, "moving active feature branches into git-multi worktrees in %d repo(s) (jobs=%d)\n", len(local), jobs)

	var sum multiSummary
	sum.collect(out, moveAll(local, defs, jobs), len(local))
	sum.switchIDE(ctx, local)

	sum.print(ctx)
	if len(sum.unresolved) > 0 || len(sum.failed) > 0 {
		return 1
	}
	return 0
}

func multiTargets(ctx *app.Context) ([]string, int, bool) {
	if !hasGit() {
		output.Die(ctx.Stderr, "git not found in PATH")
		return nil, 1, false
	}
	if !multiAvailable(ctx) {
		output.Die(ctx.Stderr, "git-multi not found in PATH")
		return nil, 1, false
	}

	local := selectLocal(ctx)
	if len(local) == 0 {
		fmt.Fprintln(ctx.Stdout, "no local git repositories found under current directory")
		return nil, 0, false
	}
	if ctx.Args.DryRun {
		dryrun.Multi(ctx.Stdout, ctx.Args, local)
		return nil, 0, false
	}
	return local, 0, true
}

func moveAll(local []string, defs defaultBranches, jobs int) <-chan gitops.Result {
	results := make(chan gitops.Result, len(local))
	sem := make(chan struct{}, jobs)
	var wg sync.WaitGroup
	for _, p := range local {
		wg.Add(1)
		sem <- struct{}{}
		go func(p string) {
			defer wg.Done()
			defer release(sem)
			results <- moveActive(p, defs)
		}(p)
	}

	go func() { wg.Wait(); close(results) }()
	return results
}

func release(sem chan struct{}) { <-sem }

func moveActive(p string, defs defaultBranches) gitops.Result {
	main, ok := defs.get(p)
	if !ok {
		return gitops.Result{Path: p, Status: "unresolved", Detail: unresolvedDefaultBranch}
	}

	cur := gitops.CurrentBranch(p)
	if cur == "" || cur == main {
		return gitops.Result{Path: p, Status: "skipped"}
	}

	res := gitops.MultiNewOne(p, cur)
	if res.Status == "moved" {
		res.Detail = "`" + cur + "` -> " + res.Detail
	}
	return res
}

func (s *multiSummary) collect(out io.Writer, results <-chan gitops.Result, total int) {
	done := 0
	for res := range results {
		done++
		switch res.Status {
		case "moved":
			s.moved++
			fmt.Fprintf(out, "[%d/%d] moved      %s — %s\n", done, total, res.Path, res.Detail)
		case "skipped":
			s.skipped++
		case "unresolved":
			s.unresolved = append(s.unresolved, res.Path)
		default:
			s.failed = append(s.failed, [2]string{res.Path, res.Detail})
		}
	}
}

func (s *multiSummary) switchIDE(ctx *app.Context, local []string) {
	feature, repos := singleWorktreeFeature(ctx, local)
	if feature == "" {
		return
	}

	fmt.Fprintf(ctx.Stdout, "\nthe only feature in worktrees is `%s` — pointing _ide at it in %d repo(s)\n", feature, len(repos))
	for _, p := range repos {
		res := gitops.MultiSwitchIDE(p, feature)
		if res.Status != "ide" {
			s.failed = append(s.failed, [2]string{res.Path, res.Detail})
			continue
		}
		s.ide++
		fmt.Fprintf(ctx.Stdout, "  _ide       %s\n", p)
	}
}

func (s *multiSummary) print(ctx *app.Context) {
	fmt.Fprintln(ctx.Stdout)
	fmt.Fprintf(ctx.Stdout, "moved: %d, skipped (no active feature): %d, _ide switched: %d, unresolved: %d, failed: %d\n",
		s.moved, s.skipped, s.ide, len(s.unresolved), len(s.failed))
	reportUnresolved(ctx.Stderr, s.unresolved)

	if len(s.failed) == 0 {
		return
	}
	sort.Slice(s.failed, func(i, j int) bool { return s.failed[i][0] < s.failed[j][0] })
	fmt.Fprintf(ctx.Stderr, "\ngit-multi failed (%d):\n", len(s.failed))
	for _, f := range s.failed {
		fmt.Fprintf(ctx.Stderr, "  %s: %s\n", f[0], f[1])
	}
}

func singleWorktreeFeature(ctx *app.Context, local []string) (string, []string) {
	reposByFeature := worktreeFeatures(ctx, local)
	if len(reposByFeature) != 1 {
		return "", nil
	}

	for b, repos := range reposByFeature {
		sort.Strings(repos)
		return b, repos
	}
	return "", nil
}

func worktreeFeatures(ctx *app.Context, local []string) map[string][]string {
	reposByFeature := map[string][]string{}
	for _, p := range local {
		worktrees, err := gitops.WorktreeBranches(p)
		if err != nil {
			fmt.Fprintf(ctx.Stderr, "warning: %s: failed to list worktrees — %s\n", p, err.Error())
			continue
		}
		for b := range worktrees {
			reposByFeature[b] = append(reposByFeature[b], p)
		}
	}
	return reposByFeature
}
