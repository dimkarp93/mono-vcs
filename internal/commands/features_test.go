package commands_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"testing"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/commands"
	"github.com/dimkarp93/mono-vcs/internal/testutil"
)

func featuresArgs() *app.Args {
	return &app.Args{}
}

func TestFeaturesEmpty(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	testutil.MakeClonedRepo(t, ws, "p", "main")
	ctx, out, _ := newCtx(t, featuresArgs(), "")
	if rc := commands.Features(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "no feature branches found")
}

func TestFeaturesListsBranchAndActives(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := testutil.MakeClonedRepo(t, ws, "a", "main")
	b := testutil.MakeClonedRepo(t, ws, "b", "main")
	testutil.Run(t, a, "git", "branch", "feat-x")
	testutil.Run(t, b, "git", "checkout", "-b", "feat-x")
	ctx, out, _ := newCtx(t, featuresArgs(), "")
	if rc := commands.Features(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	o := out.String()
	contains(t, o, "feat-x")
	contains(t, o, "a")
	contains(t, o, "b")
	contains(t, o, "1 feature branch(es) across 2 repo(s)")
}

func TestFeaturesShowsRemoteAliasColumnAndLegend(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := testutil.MakeClonedRepo(t, ws, "a", "main")
	b := testutil.MakeClonedRepo(t, ws, "b", "main")
	testutil.Run(t, a, "git", "remote", "set-url", "origin", "https://gitlab.company.com/grp/a.git")
	testutil.Run(t, b, "git", "remote", "set-url", "origin", "git@github.com:grp/b.git")
	testutil.Run(t, a, "git", "branch", "feat-x")
	testutil.Run(t, b, "git", "branch", "feat-x")

	ctx, out, _ := newCtx(t, featuresArgs(), "")
	if rc := commands.Features(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	o := out.String()
	for _, want := range []string{"remotes", "gitlab.company", "github", "gitlab.company.com", "github.com"} {
		contains(t, o, want)
	}
}

func TestFeaturesRemoteFilter(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := testutil.MakeClonedRepo(t, ws, "a", "main")
	b := testutil.MakeClonedRepo(t, ws, "b", "main")
	testutil.Run(t, a, "git", "remote", "set-url", "origin", "https://gitlab.company.com/grp/a.git")
	testutil.Run(t, b, "git", "remote", "set-url", "origin", "git@github.com:grp/b.git")
	testutil.Run(t, a, "git", "branch", "feat-a")
	testutil.Run(t, b, "git", "branch", "feat-b")

	args := featuresArgs()
	args.Remote = []string{"github"}
	ctx, out, _ := newCtx(t, args, "")
	if rc := commands.Features(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	o := out.String()
	contains(t, o, "feat-b")
	notContains(t, o, "feat-a")
}

func taggedFeatures(t *testing.T) string {
	t.Helper()

	ws := t.TempDir()
	t.Chdir(ws)
	t.Setenv("COLUMNS", "200")
	a := testutil.MakeClonedRepo(t, ws, "a", "main")
	b := testutil.MakeClonedRepo(t, ws, "b", "main")
	testutil.Run(t, a, "git", "branch", "wt-only")
	testutil.Run(t, a, "git", "worktree", "add", "-q", t.TempDir()+"/wt-only", "wt-only")
	testutil.Run(t, a, "git", "checkout", "-q", "-b", "br-only")
	testutil.Run(t, a, "git", "branch", "mixed")
	testutil.Run(t, a, "git", "worktree", "add", "-q", t.TempDir()+"/mixed", "mixed")
	testutil.Run(t, b, "git", "branch", "mixed")

	ctx, out, _ := newMultiCtx(t, featuresArgs())
	require.Equal(t, 0, commands.Features(ctx))
	return out.String()
}

func TestFeaturesTagsWorktreeAndBranchWithGitMulti(t *testing.T) {
	out := taggedFeatures(t)

	for _, want := range []string{"wt-only [use worktree] ", "br-only [use branch] ", "mixed [use worktree] [use branch]"} {
		t.Run(want, func(t *testing.T) {
			assert.Contains(t, out, want)
		})
	}
}

func TestFeaturesHasNoTagsWithoutGitMulti(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := testutil.MakeClonedRepo(t, ws, "a", "main")
	testutil.Run(t, a, "git", "branch", "wt-only")
	testutil.Run(t, a, "git", "worktree", "add", "-q", t.TempDir()+"/wt-only", "wt-only")
	ctx, out, _ := newCtx(t, featuresArgs(), "")

	rc := commands.Features(ctx)

	require.Equal(t, 0, rc)
	assert.NotContains(t, out.String(), "[use")
}
