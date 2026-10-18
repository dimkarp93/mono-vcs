package gitops_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dimkarp93/mono-vcs/internal/gitops"
	"github.com/dimkarp93/mono-vcs/internal/testutil"
)

func worktreeBranches(t *testing.T, p string) map[string]string {
	t.Helper()

	got, err := gitops.WorktreeBranches(p)
	require.NoError(t, err)

	return got
}

func resolved(t *testing.T, path string) string {
	t.Helper()

	dir, err := filepath.EvalSymlinks(path)
	require.NoError(t, err)

	return dir
}

func TestWorktreeBranchesSkipsMainCheckout(t *testing.T) {
	p := testutil.MakeRepo(t, t.TempDir(), "p", "main")
	testutil.Run(t, p, "git", "branch", "feat")
	testutil.Run(t, p, "git", "branch", "plain")
	wt := filepath.Join(t.TempDir(), "feat")
	testutil.Run(t, p, "git", "worktree", "add", "-q", wt, "feat")
	testutil.Run(t, p, "git", "worktree", "add", "-q", "--detach", filepath.Join(t.TempDir(), "_ide"))

	got := worktreeBranches(t, p)

	require.Len(t, got, 1)
	assert.Equal(t, resolved(t, wt), resolved(t, got["feat"]))
}

func TestWorktreeBranchesEmptyWithoutWorktrees(t *testing.T) {
	p := testutil.MakeRepo(t, t.TempDir(), "p", "main")
	testutil.Run(t, p, "git", "checkout", "-q", "-b", "feat")

	assert.Empty(t, worktreeBranches(t, p))
}

func dirtyFeature(t *testing.T) string {
	t.Helper()

	p := testutil.MakeClonedRepo(t, t.TempDir(), "p", "main")
	testutil.Run(t, p, "git", "checkout", "-q", "-b", "feat")
	require.NoError(t, os.WriteFile(filepath.Join(p, "README"), []byte("dirty"), 0o644))

	return p
}

func TestMultiNewOneMovesBranchIntoWorktree(t *testing.T) {
	log := testutil.FakeMulti(t)
	p := dirtyFeature(t)

	res := gitops.MultiNewOne(p, "feat")

	t.Run("moved", func(t *testing.T) {
		assert.Equal(t, "moved", res.Status)
	})
	t.Run("main checkout back on main", func(t *testing.T) {
		assert.Equal(t, "main", gitops.CurrentBranch(p))
	})
	t.Run("changes moved", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(res.Detail, "README"))
		require.NoError(t, err)
		assert.Equal(t, "dirty", string(data))
	})
	t.Run("git-multi calls", func(t *testing.T) {
		assert.Equal(t, []string{p + " new -y"}, testutil.MultiCalls(t, log))
	})
}

func TestMultiNewOneReportsFailure(t *testing.T) {
	testutil.FakeMulti(t)
	t.Setenv("GIT_MULTI_FAIL", "new")
	p := testutil.MakeClonedRepo(t, t.TempDir(), "p", "main")
	testutil.Run(t, p, "git", "checkout", "-q", "-b", "feat")

	res := gitops.MultiNewOne(p, "feat")

	t.Run("failed", func(t *testing.T) {
		assert.Equal(t, "failed", res.Status)
	})
	t.Run("detail", func(t *testing.T) {
		assert.Contains(t, res.Detail, "fake new failure")
	})
}

func TestMultiSwitchIDE(t *testing.T) {
	log := testutil.FakeMulti(t)
	p := testutil.MakeRepo(t, t.TempDir(), "p", "main")

	res := gitops.MultiSwitchIDE(p, "feat")

	t.Run("status", func(t *testing.T) {
		assert.Equal(t, "ide", res.Status)
	})
	t.Run("git-multi calls", func(t *testing.T) {
		assert.Equal(t, []string{p + " switch _ide feat -no-fetch"}, testutil.MultiCalls(t, log))
	})
}

func TestHasMultiFollowsPath(t *testing.T) {
	testutil.FakeMulti(t)

	assert.True(t, gitops.HasMulti())
}
