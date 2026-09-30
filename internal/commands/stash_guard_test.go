package commands_test

import (
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/commands"
	"github.com/dimkarp93/mono-vcs/internal/testutil"
)

func leaveUpdateStash(t *testing.T, repo, branch string) {
	t.Helper()
	writeFile(repo, "pending.txt", "mine")
	testutil.Run(t, repo, "git", "stash", "push", "--include-untracked", "-m", "mono-vcs update "+branch)
}

func TestMRBlockedByPendingUpdateStash(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := placeLinked(t, ws, "a")
	b := placeLinked(t, ws, "b")
	onBranch(t, a, "PROJ-1-feat")
	onBranch(t, b, "PROJ-1-feat")
	leaveUpdateStash(t, b, "PROJ-1-feat")
	fakeGlab(t)

	ctx, out, errb := newCtx(t, mrArgs(""), "")
	if rc := commands.MR(ctx); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, errb.String(), "git stash")
	contains(t, errb.String(), "b")
	notContains(t, out.String(), "pushing")
}

func TestMRIgnoresForeignStash(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := placeLinked(t, ws, "a")
	onBranch(t, a, "PROJ-1-feat")
	writeFile(a, "other.txt", "x")
	testutil.Run(t, a, "git", "stash", "push", "--include-untracked", "-m", "my own stash")
	fakeGlab(t)

	ctx, _, errb := newCtx(t, mrArgs(""), "")
	if rc := commands.MR(ctx); rc != 0 {
		t.Fatalf("rc=%d err=%s", rc, errb.String())
	}
}

func TestFeaturesShowsPendingUpdateStash(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := testutil.MakeClonedRepo(t, ws, "a", "main")
	testutil.MakeClonedRepo(t, ws, "b", "main")
	testutil.Run(t, a, "git", "checkout", "-b", "feat-x")
	leaveUpdateStash(t, a, "feat-x")

	ctx, out, _ := newCtx(t, featuresArgs(), "")
	if rc := commands.Features(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "hold local changes in `git stash`")
}

func TestListMarksPendingUpdateStash(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	fake := testutil.NewFakeGitLab(t)
	repo, _, _ := makeSyncedPair(t, ws, fake, "grp/proj")
	leaveUpdateStash(t, repo, "feat-x")

	ctx, out, _ := newCtx(t, listArgs(fake), "")
	commands.List(ctx)
	contains(t, out.String(), "grp/proj")
	contains(t, out.String(), "⚑")
}
