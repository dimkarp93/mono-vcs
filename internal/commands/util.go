package commands

import (
	"os/exec"
	"sort"
)

func hasGit() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

func sortedSet(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func toSet(ss []string) map[string]bool {
	m := make(map[string]bool, len(ss))
	for _, s := range ss {
		m[s] = true
	}
	return m
}

func unionKeys(a, b map[string]bool) map[string]bool {
	out := make(map[string]bool, len(a)+len(b))
	for k := range a {
		out[k] = true
	}
	for k := range b {
		out[k] = true
	}
	return out
}

func intersect(a, allowed map[string]bool) map[string]bool {
	out := map[string]bool{}
	for k := range a {
		if allowed[k] {
			out[k] = true
		}
	}
	return out
}
