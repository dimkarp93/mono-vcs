package commands

import (
	"fmt"
	"sort"

	"mono-vcs/internal/app"
	"mono-vcs/internal/colors"
	"mono-vcs/internal/gitops"
	"mono-vcs/internal/output"
)

func Features(ctx *app.Context) int {
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

	main := a.GetMainBranch()
	branchesByName := map[string][]string{}
	activeByName := map[string]map[string]bool{}
	for _, p := range local {
		branches, err := gitops.LocalBranches(p)
		if err != nil {
			fmt.Fprintf(ctx.Stderr, "warning: %s: failed to list branches — %s\n", p, err.Error())
			continue
		}
		cur := gitops.CurrentBranch(p)
		for _, b := range branches {
			if b == "" || b == main {
				continue
			}
			branchesByName[b] = append(branchesByName[b], p)
			if cur == b {
				if activeByName[b] == nil {
					activeByName[b] = map[string]bool{}
				}
				activeByName[b][p] = true
			}
		}
	}

	if len(branchesByName) == 0 {
		fmt.Fprintf(out, "no feature branches found (every local repo only has `%s`)\n", main)
		return 0
	}

	names := make([]string, 0, len(branchesByName))
	for n := range branchesByName {
		names = append(names, n)
	}
	sort.Strings(names)

	var rows []output.FeatureRow
	for _, n := range names {
		all := append([]string{}, branchesByName[n]...)
		sort.Strings(all)
		var active []string
		for p := range activeByName[n] {
			active = append(active, p)
		}
		sort.Strings(active)
		rows = append(rows, output.FeatureRow{Branch: n, Repos: all, Active: active})
	}
	output.PrintFeaturesTable(out, rows, colors.Enabled())
	fmt.Fprintf(out, "\n%d feature branch(es) across %d repo(s)\n", len(rows), len(local))
	return 0
}
