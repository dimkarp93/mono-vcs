package commands_test

import (
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/commands"
	"github.com/dimkarp93/mono-vcs/internal/gitops"
	"github.com/dimkarp93/mono-vcs/internal/testutil"
)

func finishArgs(branch string) *app.Args {
	return &app.Args{Branch: branch, Jobs: testutil.I(1), GLToken: ""}
}

func TestFinishSkipsDefaultBranch(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	placeLinked(t, ws, "p")
	ctx, out, _ := newCtx(t, finishArgs("main"), "")
	if rc := commands.Finish(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "is-default: 1")
}

func TestFinishAbsentSilent(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	placeLinked(t, ws, "p")
	ctx, out, _ := newCtx(t, finishArgs("ghost"), "")
	if rc := commands.Finish(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "branch absent: 1")
}

func TestFinishDeletesNonCurrent(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := placeLinked(t, ws, "p")
	testutil.Run(t, p, "git", "branch", "feature")
	ctx, out, _ := newCtx(t, finishArgs("feature"), "")
	if rc := commands.Finish(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "deleted    p")
	if gitops.HasBranch(p, "feature") {
		t.Fatal("feature should be gone")
	}
}

func TestFinishCurrentCleanSwitchesThenDeletes(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := placeLinked(t, ws, "p")
	testutil.Run(t, p, "git", "checkout", "-b", "feature")
	ctx, out, _ := newCtx(t, finishArgs("feature"), "")
	if rc := commands.Finish(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "deleted")
	if gitops.CurrentBranch(p) != "main" {
		t.Fatalf("branch=%q", gitops.CurrentBranch(p))
	}
}

func TestFinishCurrentDirtyIsDeletedAnyway(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := placeLinked(t, ws, "p")
	testutil.Run(t, p, "git", "checkout", "-b", "feature")
	writeFile(p, "README", "dirty")
	writeFile(p, "untracked.txt", "junk")
	ctx, out, _ := newCtx(t, finishArgs("feature"), "")
	if rc := commands.Finish(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "discarded 2 items")
	if gitops.CurrentBranch(p) != "main" || gitops.IsDirty(p) {
		t.Fatalf("branch=%q dirty=%v", gitops.CurrentBranch(p), gitops.IsDirty(p))
	}
	if gitops.HasBranch(p, "feature") {
		t.Fatal("feature should be gone")
	}
}

func TestFinishDryRun(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := placeLinked(t, ws, "p")
	testutil.Run(t, p, "git", "branch", "feature")
	a := finishArgs("feature")
	a.DryRun = true
	ctx, out, _ := newCtx(t, a, "")
	if rc := commands.Finish(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "DRY-RUN: finish feature")
	if !gitops.HasBranch(p, "feature") {
		t.Fatal("dry-run must not delete the branch")
	}
}
