package repos

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func ScanLocalRepos(root string) map[string]bool {
	found := map[string]bool{}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return found
	}
	_ = filepath.WalkDir(rootAbs, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}

		if path != rootAbs && strings.HasPrefix(d.Name(), ".") {
			return fs.SkipDir
		}
		if _, e := os.Stat(filepath.Join(path, ".git")); e == nil {
			if rel, e2 := filepath.Rel(rootAbs, path); e2 == nil && rel != "." {
				found[filepath.ToSlash(rel)] = true
			}
			return fs.SkipDir
		}
		return nil
	})
	return found
}

func CommonTopGroup(remotePaths []string) string {
	tops := map[string]bool{}
	for _, p := range remotePaths {
		if i := strings.Index(p, "/"); i >= 0 {
			tops[p[:i]] = true
		}
	}
	if len(tops) != 1 {
		return ""
	}
	for k := range tops {
		return k
	}
	return ""
}

func StripPrefix(path, prefix string) string {
	if prefix == "" {
		return path
	}
	head := prefix + "/"
	if strings.HasPrefix(path, head) {
		return path[len(head):]
	}
	return path
}

func matchesGroup(repoPath, group string) bool {
	parts := strings.Split(repoPath, "/")
	parent := strings.Join(parts[:len(parts)-1], "/")
	if parent == "" {
		return false
	}
	return parent == group ||
		strings.HasPrefix(parent, group+"/") ||
		strings.Contains("/"+parent+"/", "/"+group+"/")
}

func FilterRepos(local []string, names []string, stderr io.Writer) []string {
	if len(names) == 0 {
		return local
	}
	var items []string
	for _, n := range names {
		for _, s := range strings.Split(n, ",") {
			if s = strings.TrimSpace(s); s != "" {
				items = append(items, s)
			}
		}
	}
	if len(items) == 0 {
		return local
	}

	var groups, bareRepos, paths []string
	for _, n := range items {
		switch {
		case strings.HasSuffix(n, "/"):
			groups = append(groups, strings.TrimRight(n, "/"))
		case strings.Contains(n, "/"):
			paths = append(paths, n)
		default:
			bareRepos = append(bareRepos, n)
		}
	}

	byName := map[string][]string{}
	localSet := map[string]bool{}
	for _, p := range local {
		b := filepath.Base(p)
		byName[b] = append(byName[b], p)
		localSet[p] = true
	}

	matched := map[string]bool{}
	matchedGroups := map[string]bool{}
	matchedRepos := map[string]bool{}
	matchedPaths := map[string]bool{}

	for _, r := range bareRepos {
		if hits := byName[r]; len(hits) > 0 {
			for _, h := range hits {
				matched[h] = true
			}
			matchedRepos[r] = true
		}
	}
	for _, fp := range paths {
		if localSet[fp] {
			matched[fp] = true
			matchedPaths[fp] = true
		}
	}
	for _, g := range groups {
		for _, p := range local {
			if matchesGroup(p, g) {
				matched[p] = true
				matchedGroups[g] = true
			}
		}
	}

	missingSet := map[string]bool{}
	for _, g := range groups {
		if !matchedGroups[g] {
			missingSet[g] = true
		}
	}
	for _, r := range bareRepos {
		if !matchedRepos[r] {
			missingSet[r] = true
		}
	}
	for _, fp := range paths {
		if !matchedPaths[fp] {
			missingSet[fp] = true
		}
	}
	if len(missingSet) > 0 {
		missing := keysSorted(missingSet)
		fmt.Fprintf(stderr, "warning: --repo entries matched nothing: %s\n", strings.Join(missing, ", "))
	}

	return keysSorted(matched)
}

func keysSorted(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func SortedKeys(m map[string]bool) []string {
	return keysSorted(m)
}
