package commands_test

import (
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/commands"
	"github.com/dimkarp93/mono-vcs/internal/gitops"
	"github.com/dimkarp93/mono-vcs/internal/testutil"
)

func doneArgs() *app.Args {
	return &app.Args{Jobs: testutil.I(1)}
}

func mergeIntoDefault(t *testing.T, repo, branch string) {
	t.Helper()
	testutil.Run(t, repo, "git", "checkout", "main")
	testutil.Run(t, repo, "git", "merge", "--no-ff", "-m", "merge "+branch, branch)
	testutil.Run(t, repo, "git", "push", "origin", "main")
	testutil.Run(t, repo, "git", "fetch", "origin")
}

func TestDoneDeletesMergedBranch(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := placeLinked(t, ws, "p")
	testutil.Run(t, p, "git", "checkout", "-b", "feature")
	testutil.Commit(t, p, "work", "f", "f")
	mergeIntoDefault(t, p, "feature")

	ctx, out, _ := newCtx(t, doneArgs(), "")
	if rc := commands.Done(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "deleted: 1")
	if gitops.HasBranch(p, "feature") {
		t.Fatal("feature should be gone")
	}
}

func TestDoneKeepsUnmergedBranch(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := placeLinked(t, ws, "p")
	testutil.Run(t, p, "git", "checkout", "-b", "feature")
	testutil.Commit(t, p, "work", "f", "f")
	testutil.Run(t, p, "git", "checkout", "main")

	ctx, out, _ := newCtx(t, doneArgs(), "")
	if rc := commands.Done(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "still open: 1")
	if !gitops.HasBranch(p, "feature") {
		t.Fatal("unmerged feature must survive")
	}
}

func TestDoneKeepsDirtyCurrentBranch(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := placeLinked(t, ws, "p")
	testutil.Run(t, p, "git", "checkout", "-b", "feature")
	testutil.Commit(t, p, "work", "f", "f")
	mergeIntoDefault(t, p, "feature")
	testutil.Run(t, p, "git", "checkout", "feature")
	writeFile(p, "untracked.txt", "junk")

	ctx, out, _ := newCtx(t, doneArgs(), "")
	if rc := commands.Done(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "dirty: 1")
	if !gitops.HasBranch(p, "feature") {
		t.Fatal("a dirty checked-out branch must survive")
	}
}

func TestDoneSwitchesOffDeletedCurrentBranch(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := placeLinked(t, ws, "p")
	testutil.Run(t, p, "git", "checkout", "-b", "feature")
	testutil.Commit(t, p, "work", "f", "f")
	mergeIntoDefault(t, p, "feature")
	testutil.Run(t, p, "git", "checkout", "feature")

	ctx, _, _ := newCtx(t, doneArgs(), "")
	if rc := commands.Done(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	if gitops.CurrentBranch(p) != "main" || gitops.HasBranch(p, "feature") {
		t.Fatalf("branch=%q", gitops.CurrentBranch(p))
	}
}

func TestDoneIgnoresRepoAndFeatureFilters(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := placeLinked(t, ws, "a")
	b := placeLinked(t, ws, "b")
	for _, p := range []string{a, b} {
		testutil.Run(t, p, "git", "checkout", "-b", "feature")
		testutil.Commit(t, p, "work", "f", "f")
		mergeIntoDefault(t, p, "feature")
	}

	args := doneArgs()
	args.Repo = []string{"a"}
	args.Feature = "feature"
	ctx, out, _ := newCtx(t, args, "")
	if rc := commands.Done(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "deleted: 2")
	if gitops.HasBranch(a, "feature") || gitops.HasBranch(b, "feature") {
		t.Fatal("done must clean every repo regardless of filters")
	}
}

func TestDoneNothingToDo(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	placeLinked(t, ws, "p")
	ctx, out, _ := newCtx(t, doneArgs(), "")
	if rc := commands.Done(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "nothing to clean up")
}

func TestDoneReportsUnresolvedDefault(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	testutil.MakeRepo(t, ws, "orphan", "trunk")
	ctx, _, errb := newCtx(t, doneArgs(), "")
	if rc := commands.Done(ctx); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, errb.String(), "the default branch is unknown")
}

func TestDoneDryRun(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := placeLinked(t, ws, "p")
	testutil.Run(t, p, "git", "checkout", "-b", "feature")
	testutil.Commit(t, p, "work", "f", "f")
	mergeIntoDefault(t, p, "feature")

	a := doneArgs()
	a.DryRun = true
	ctx, out, _ := newCtx(t, a, "")
	if rc := commands.Done(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "DRY-RUN: done")
	if !gitops.HasBranch(p, "feature") {
		t.Fatal("dry-run must not delete anything")
	}
}
