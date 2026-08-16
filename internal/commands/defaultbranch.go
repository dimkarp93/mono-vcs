package commands

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"mono-vcs/internal/app"
	"mono-vcs/internal/gitops"
	"mono-vcs/internal/output"
)

func DefaultBranch(ctx *app.Context) int {
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

	fallback := a.GetMainBranch()
	branches := make(map[string]string, len(local))
	detected := make(map[string]bool, len(local))
	var mu sync.Mutex
	sem := make(chan struct{}, a.GetJobs())
	var wg sync.WaitGroup
	for _, p := range local {
		wg.Add(1)
		sem <- struct{}{}
		go func(p string) {
			defer wg.Done()
			defer func() { <-sem }()
			b, ok := gitops.DefaultBranch(p, fallback)
			mu.Lock()
			branches[p] = b
			detected[p] = ok
			mu.Unlock()
		}(p)
	}
	wg.Wait()

	width := 0
	for _, p := range local {
		if w := utf8.RuneCountInString(p); w > width {
			width = w
		}
	}
	counts := map[string]int{}
	for _, p := range local {
		line := fmt.Sprintf("%-*s  %s", width, p, branches[p])
		if !detected[p] {
			line += "  (fallback)"
		}
		fmt.Fprintln(out, line)
		counts[branches[p]]++
	}

	names := make([]string, 0, len(counts))
	for b := range counts {
		names = append(names, b)
	}
	sort.Slice(names, func(i, j int) bool {
		if counts[names[i]] != counts[names[j]] {
			return counts[names[i]] > counts[names[j]]
		}
		return names[i] < names[j]
	})
	parts := make([]string, 0, len(names))
	for _, b := range names {
		parts = append(parts, fmt.Sprintf("%s=%d", b, counts[b]))
	}
	fmt.Fprintf(out, "\n%d repo(s): %s\n", len(local), strings.Join(parts, ", "))
	return 0
}
