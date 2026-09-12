package commands

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/output"
)

func DefaultBranch(ctx *app.Context) int {
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

	defs := resolveDefaults(ctx, local, nil)
	branches := make(map[string]string, len(local))
	detected := make(map[string]bool, len(local))
	for _, p := range local {
		b, ok := defs.get(p)
		branches[p] = b
		detected[p] = ok
	}

	width := 0
	for _, p := range local {
		if w := utf8.RuneCountInString(p); w > width {
			width = w
		}
	}
	counts := map[string]int{}
	var unresolved []string
	for _, p := range local {
		if !detected[p] {
			unresolved = append(unresolved, p)
			fmt.Fprintf(out, "%-*s  (unresolved)\n", width, p)
			continue
		}
		fmt.Fprintf(out, "%-*s  %s\n", width, p, branches[p])
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
	parts := make([]string, 0, len(names)+1)
	for _, b := range names {
		parts = append(parts, fmt.Sprintf("%s=%d", b, counts[b]))
	}
	if len(unresolved) > 0 {
		parts = append(parts, fmt.Sprintf("unresolved=%d", len(unresolved)))
	}
	fmt.Fprintf(out, "\n%d repo(s): %s\n", len(local), strings.Join(parts, ", "))
	if len(unresolved) > 0 {
		reportUnresolved(ctx.Stderr, unresolved)
		return 1
	}
	return 0
}
