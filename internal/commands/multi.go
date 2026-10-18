package commands

import (
	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/gitops"
	"io"
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
	panic("not implemented")
}
func release(sem chan struct{}) {
	panic("not implemented")
}
func moveActive(p string, defs defaultBranches) gitops.Result {
	panic("not implemented")
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
	panic("not implemented")
}
func worktreeFeatures(ctx *app.Context, local []string) map[string][]string {
	panic("not implemented")
}
