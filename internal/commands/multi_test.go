package commands_test

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/commands"
	"github.com/dimkarp93/mono-vcs/internal/gitops"
	"github.com/dimkarp93/mono-vcs/internal/testutil"
)

func multiArgs() *app.Args {
	return &app.Args{Jobs: testutil.I(1)}
}

func sortedCalls(t *testing.T, log string) []string {
	t.Helper()

	calls := testutil.MultiCalls(t, log)
	sort.Strings(calls)
	return calls
}

func runMulti(t *testing.T, log string) multiRun {
	t.Helper()

	ctx, out, errb := newMultiCtx(t, multiArgs())
	rc := commands.Multi(ctx)
	return multiRun{rc: rc, out: out.String(), err: errb.String(), calls: sortedCalls(t, log)}
}

func TestMultiMovesActiveBranchesAndPointsIDEAtSingleFeature(t *testing.T) {
	ws, log := multiWorkspace(t)
	a := testutil.MakeClonedRepo(t, ws, "a", "main")
	b := testutil.MakeClonedRepo(t, ws, "b", "main")
	testutil.MakeClonedRepo(t, ws, "c", "main")
	testutil.Run(t, a, "git", "checkout", "-q", "-b", "feat")
	testutil.Run(t, b, "git", "checkout", "-q", "-b", "feat")

	res := runMulti(t, log)

	t.Run("exit code", func(t *testing.T) {
		assert.Equal(t, 0, res.rc, res.err)
	})
	t.Run("summary", func(t *testing.T) {
		assert.Contains(t, res.out, "moved: 2, skipped (no active feature): 1, _ide switched: 2")
	})
	t.Run("main checkouts back on main", func(t *testing.T) {
		assert.Equal(t, []string{"main", "main"}, []string{gitops.CurrentBranch(a), gitops.CurrentBranch(b)})
	})
	t.Run("git-multi calls", func(t *testing.T) {
		assert.Equal(t, []string{"a new -y", "a switch _ide feat -no-fetch", "b new -y", "b switch _ide feat -no-fetch"}, res.calls)
	})
}

func TestMultiLeavesIDEWithSeveralFeatures(t *testing.T) {
	ws, log := multiWorkspace(t)
	a := testutil.MakeClonedRepo(t, ws, "a", "main")
	b := testutil.MakeClonedRepo(t, ws, "b", "main")
	testutil.Run(t, a, "git", "checkout", "-q", "-b", "feat-a")
	testutil.Run(t, b, "git", "checkout", "-q", "-b", "feat-b")

	res := runMulti(t, log)

	require.Equal(t, 0, res.rc, res.out)
	assert.Equal(t, []string{"a new -y", "b new -y"}, res.calls)
}

func TestMultiCountsExistingWorktreesForIDE(t *testing.T) {
	ws, log := multiWorkspace(t)
	a := testutil.MakeClonedRepo(t, ws, "a", "main")
	b := testutil.MakeClonedRepo(t, ws, "b", "main")
	testutil.Run(t, a, "git", "checkout", "-q", "-b", "feat")
	testutil.Run(t, b, "git", "branch", "other")
	testutil.Run(t, b, "git", "worktree", "add", "-q", t.TempDir()+"/other", "other")

	res := runMulti(t, log)

	require.Equal(t, 0, res.rc, res.out)
	assert.Equal(t, []string{"a new -y"}, res.calls)
}

func TestMultiReportsGitMultiFailure(t *testing.T) {
	ws, log := multiWorkspace(t)
	t.Setenv("GIT_MULTI_FAIL", "new")
	a := testutil.MakeClonedRepo(t, ws, "a", "main")
	testutil.Run(t, a, "git", "checkout", "-q", "-b", "feat")

	res := runMulti(t, log)

	t.Run("exit code", func(t *testing.T) {
		assert.Equal(t, 1, res.rc)
	})
	t.Run("reports failure", func(t *testing.T) {
		assert.Contains(t, res.err, "fake new failure")
	})
}

func TestMultiDryRunChangesNothing(t *testing.T) {
	ws, log := multiWorkspace(t)
	a := testutil.MakeClonedRepo(t, ws, "a", "main")
	testutil.Run(t, a, "git", "checkout", "-q", "-b", "feat")
	args := multiArgs()
	args.DryRun = true
	ctx, out, _ := newMultiCtx(t, args)

	rc := commands.Multi(ctx)

	t.Run("exit code", func(t *testing.T) {
		assert.Equal(t, 0, rc)
	})
	t.Run("describes the plan", func(t *testing.T) {
		assert.Contains(t, out.String(), "git-multi -C <repo-name> new -y")
	})
	t.Run("no git-multi calls", func(t *testing.T) {
		assert.Empty(t, testutil.MultiCalls(t, log))
	})
}

func TestMultiRequiresGitMulti(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	testutil.MakeClonedRepo(t, ws, "a", "main")
	ctx, _, errb := newCtx(t, multiArgs(), "")

	rc := commands.Multi(ctx)

	t.Run("exit code", func(t *testing.T) {
		assert.Equal(t, 1, rc)
	})
	t.Run("explains", func(t *testing.T) {
		assert.Contains(t, errb.String(), "git-multi not found")
	})
}
