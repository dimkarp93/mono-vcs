package commands_test

import (
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/commands"
	"github.com/dimkarp93/mono-vcs/internal/gitops"
	"github.com/dimkarp93/mono-vcs/internal/testutil"
)

func newArgs(branch string) *app.Args {
	return &app.Args{Branch: branch, Jobs: testutil.I(1), MainBranch: testutil.S("main")}
}

func TestNewCreatesLocalBranch(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := testutil.MakeRepo(t, ws, "p", "main")
	ctx, out, _ := newCtx(newArgs("MVPAY-290"), "")
	if rc := commands.New(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "created")
	if gitops.CurrentBranch(p) != "MVPAY-290" {
		t.Fatalf("branch=%q", gitops.CurrentBranch(p))
	}
}

func TestNewSkipsDefaultBranch(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	testutil.MakeRepo(t, ws, "p", "main")
	ctx, out, _ := newCtx(newArgs("main"), "")
	if rc := commands.New(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "is-default: 1")
}

func TestNewExistingIsNotError(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := testutil.MakeRepo(t, ws, "p", "main")
	testutil.Run(t, p, "git", "branch", "feature")
	ctx, out, _ := newCtx(newArgs("feature"), "")
	if rc := commands.New(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "exists")
}

func TestNewDirtyIsSkipped(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := testutil.MakeRepo(t, ws, "p", "main")
	writeFile(p, "README", "dirty")
	ctx, _, errb := newCtx(newArgs("feature"), "")
	if rc := commands.New(ctx); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, errb.String(), "uncommitted")
}

func TestNewRepoFilter(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := testutil.MakeRepo(t, ws, "a", "main")
	b := testutil.MakeRepo(t, ws, "b", "main")
	args := newArgs("feature")
	args.Repo = []string{"a"}
	ctx, _, _ := newCtx(args, "")
	if rc := commands.New(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	if !gitops.HasBranch(a, "feature") {
		t.Fatal("a should have feature")
	}
	if gitops.HasBranch(b, "feature") {
		t.Fatal("b should be untouched")
	}
}

func TestNewDryRun(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := testutil.MakeRepo(t, ws, "p", "main")
	a := newArgs("feature")
	a.DryRun = true
	ctx, out, _ := newCtx(a, "")
	if rc := commands.New(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "DRY-RUN: new feature")
	if gitops.HasBranch(p, "feature") {
		t.Fatal("dry-run must not create the branch")
	}
}
