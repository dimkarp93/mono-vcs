package commands_test

import (
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/commands"
	"github.com/dimkarp93/mono-vcs/internal/gitops"
	"github.com/dimkarp93/mono-vcs/internal/testutil"
)

func newArgs(branch string) *app.Args {
	return &app.Args{Branch: branch, Jobs: testutil.I(1)}
}

func TestNewCreatesLocalBranch(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := testutil.MakeClonedRepo(t, ws, "p", "main")
	ctx, out, _ := newCtx(t, newArgs("MVPAY-290"), "")
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
	testutil.MakeClonedRepo(t, ws, "p", "main")
	ctx, out, _ := newCtx(t, newArgs("main"), "")
	if rc := commands.New(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "is-default: 1")
}

func TestNewSwitchesToExistingBranch(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := testutil.MakeClonedRepo(t, ws, "p", "main")
	testutil.Run(t, p, "git", "branch", "feature")
	ctx, out, _ := newCtx(t, newArgs("feature"), "")
	if rc := commands.New(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "switched")
	if gitops.CurrentBranch(p) != "feature" {
		t.Fatalf("branch=%q", gitops.CurrentBranch(p))
	}
}

func TestNewCarriesDirtyWorkTreeOver(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := testutil.MakeClonedRepo(t, ws, "p", "main")
	writeFile(p, "README", "dirty")
	writeFile(p, "untracked.txt", "junk")
	ctx, out, _ := newCtx(t, newArgs("feature"), "")
	if rc := commands.New(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "carried local changes over")
	if gitops.CurrentBranch(p) != "feature" {
		t.Fatalf("branch=%q", gitops.CurrentBranch(p))
	}
	status := testutil.Run(t, p, "git", "status", "--porcelain")
	contains(t, status, "README")
	contains(t, status, "untracked.txt")
}

func TestNewBasesOnDefaultNotCurrentBranch(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := testutil.MakeClonedRepo(t, ws, "p", "main")
	mainSHA := headSHA(t, p)
	testutil.Run(t, p, "git", "checkout", "-b", "other")
	testutil.Commit(t, p, "advance other", "x", "y")
	writeFile(p, "untracked.txt", "junk")
	ctx, _, _ := newCtx(t, newArgs("feature"), "")
	if rc := commands.New(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	if got := headSHA(t, p); got != mainSHA {
		t.Fatalf("feature must start at the default branch tip: %q != %q", got, mainSHA)
	}
	contains(t, testutil.Run(t, p, "git", "status", "--porcelain"), "untracked.txt")
}

func TestNewAlreadyOnBranch(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := testutil.MakeClonedRepo(t, ws, "p", "main")
	testutil.Run(t, p, "git", "checkout", "-b", "feature")
	ctx, out, _ := newCtx(t, newArgs("feature"), "")
	if rc := commands.New(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "already")
}

func TestNewRepoFilter(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := testutil.MakeClonedRepo(t, ws, "a", "main")
	b := testutil.MakeClonedRepo(t, ws, "b", "main")
	args := newArgs("feature")
	args.Repo = []string{"a"}
	ctx, _, _ := newCtx(t, args, "")
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
	p := testutil.MakeClonedRepo(t, ws, "p", "main")
	a := newArgs("feature")
	a.DryRun = true
	ctx, out, _ := newCtx(t, a, "")
	if rc := commands.New(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "DRY-RUN: new feature")
	if gitops.HasBranch(p, "feature") {
		t.Fatal("dry-run must not create the branch")
	}
}

func TestNewSkipsRepoWithoutResolvableDefault(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	testutil.MakeRepo(t, ws, "orphan", "trunk")

	ctx, out, errb := newCtx(t, newArgs("feat"), "")
	if rc := commands.New(ctx); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "unresolved: 1")
	contains(t, errb.String(), "the default branch is unknown")
}
