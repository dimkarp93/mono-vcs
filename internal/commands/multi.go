package commands

import (
	"fmt"
	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/gitops"
	"io"
	"sort"
	"sync"
)

type multiSummary struct {
	moved, skipped, ide int
	unresolved          []string
	failed              [][2]string
}

func Multi(ctx *app.Context) int {
	panic("not implemented")
}
func multiTargets(ctx *app.Context) ([]string, int, bool) {
	panic("not implemented")
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
	go func() {
		wg.Wait()
		close(results)
	}()
	return results
}
func release(sem chan struct{}) {
	<-sem
}
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
	panic("not implemented")
}
func (s *multiSummary) switchIDE(ctx *app.Context, local []string) {
	panic("not implemented")
}
func (s *multiSummary) print(ctx *app.Context) {
	panic("not implemented")
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
