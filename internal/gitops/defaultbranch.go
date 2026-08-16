package gitops

import (
	"path/filepath"
	"strings"
	"sync"
)

const FallbackDefaultBranch = "main"

var defaultBranchCandidates = []string{"main", "master", "develop"}

var defaultBranchCache sync.Map

type defaultBranchResult struct {
	branch   string
	detected bool
}

func DefaultBranch(path, fallback string) (string, bool) {
	key := path
	if abs, err := filepath.Abs(path); err == nil {
		key = abs
	}
	key += "\x00" + fallback
	if v, ok := defaultBranchCache.Load(key); ok {
		r := v.(defaultBranchResult)
		return r.branch, r.detected
	}
	branch, detected := detectDefaultBranch(path, fallback)
	defaultBranchCache.Store(key, defaultBranchResult{branch, detected})
	return branch, detected
}

func detectDefaultBranch(path, fallback string) (string, bool) {
	if out, _, rc := runGit("-C", path, "symbolic-ref", "--short", "-q", "refs/remotes/origin/HEAD"); rc == 0 {
		if b := strings.TrimPrefix(strings.TrimSpace(out), "origin/"); b != "" {
			return b, true
		}
	}
	for _, prefix := range []string{"refs/remotes/origin/", "refs/heads/"} {
		for _, c := range defaultBranchCandidates {
			if _, _, rc := runGit("-C", path, "rev-parse", "--verify", "--quiet", prefix+c); rc == 0 {
				return c, true
			}
		}
	}
	if fallback != "" {
		return fallback, false
	}
	return FallbackDefaultBranch, false
}
