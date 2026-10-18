package gitops

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLinkedBranches(t *testing.T) {
	porcelain := "worktree /r\nHEAD a\nbranch refs/heads/main\n\nworktree /w/feat\nHEAD b\nbranch refs/heads/feat\n\nworktree /w/_ide\nHEAD c\ndetached\n"

	assert.Equal(t, map[string]string{"feat": "/w/feat"}, linkedBranches(porcelain))
}
