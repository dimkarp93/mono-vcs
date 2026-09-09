package commands_test

import (
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/commands"
	"github.com/dimkarp93/mono-vcs/internal/gitops"
	"github.com/dimkarp93/mono-vcs/internal/testutil"
)

func cancelArgs(branch string) *app.Args {
	return &app.Args{Branch: branch, Jobs: testutil.I(1), MainBranch: testutil.S("main"), GLToken: ""}
}

func TestCancelSkipsDefaultBranch(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	placeLinked(t, ws, "p")
	ctx, out, _ := newCtx(cancelArgs("main"), "")
	if rc := commands.Cancel(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "is-default: 1")
}

func TestCancelAbsentSilent(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	placeLinked(t, ws, "p")
	ctx, out, _ := newCtx(cancelArgs("ghost"), "")
	if rc := commands.Cancel(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "branch absent: 1")
}

func TestCancelDeletesNonCurrent(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := placeLinked(t, ws, "p")
	testutil.Run(t, p, "git", "branch", "feature")
	ctx, out, _ := newCtx(cancelArgs("feature"), "")
	if rc := commands.Cancel(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "deleted    p")
	if gitops.HasBranch(p, "feature") {
		t.Fatal("feature should be gone")
	}
}

func TestCancelCurrentCleanSwitchesThenDeletes(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := placeLinked(t, ws, "p")
	testutil.Run(t, p, "git", "checkout", "-b", "feature")
	ctx, out, _ := newCtx(cancelArgs("feature"), "")
	if rc := commands.Cancel(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "deleted")
	if gitops.CurrentBranch(p) != "main" {
		t.Fatalf("branch=%q", gitops.CurrentBranch(p))
	}
}

func TestCancelCurrentDirtyListed(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := placeLinked(t, ws, "p")
	testutil.Run(t, p, "git", "checkout", "-b", "feature")
	writeFile(p, "README", "dirty")
	ctx, _, errb := newCtx(cancelArgs("feature"), "")
	if rc := commands.Cancel(ctx); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, errb.String(), "uncommitted")
}
