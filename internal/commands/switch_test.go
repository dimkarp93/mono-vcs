package commands_test

import (
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/commands"
	"github.com/dimkarp93/mono-vcs/internal/gitops"
	"github.com/dimkarp93/mono-vcs/internal/testutil"
)

func switchArgs(branch string) *app.Args {
	return &app.Args{Branch: branch, Jobs: testutil.I(1), GLToken: ""}
}

func TestSwitchExistingLocalBranch(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := placeLinked(t, ws, "p")
	testutil.Run(t, p, "git", "branch", "feature")
	ctx, out, _ := newCtx(t, switchArgs("feature"), "")
	if rc := commands.Switch(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "switched")
	if gitops.CurrentBranch(p) != "feature" {
		t.Fatalf("branch=%q", gitops.CurrentBranch(p))
	}
}

func TestSwitchAlreadyOnBranch(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	placeLinked(t, ws, "p")
	ctx, out, _ := newCtx(t, switchArgs("main"), "")
	if rc := commands.Switch(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "already")
}

func TestSwitchDirtyListed(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := placeLinked(t, ws, "p")
	writeFile(p, "README", "dirty")
	ctx, _, errb := newCtx(t, switchArgs("feature"), "")
	if rc := commands.Switch(ctx); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
	e := errb.String()
	if !indexed(e, "uncommitted") && !indexed(e, "could not switch") {
		t.Fatalf("err=%s", e)
	}
}

func TestSwitchFallbackToMain(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	placeLinked(t, ws, "p")
	ctx, out, _ := newCtx(t, switchArgs("no-such"), "")
	if rc := commands.Switch(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "main-sync")
}

func indexed(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestSwitchWithoutBranchGoesToDefaultPerRepo(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := testutil.MakeClonedRepo(t, ws, "a", "main")
	b := testutil.MakeClonedRepo(t, ws, "b", "master")
	testutil.Run(t, a, "git", "checkout", "-b", "feature")
	testutil.Run(t, b, "git", "checkout", "-b", "feature")

	ctx, out, _ := newCtx(t, switchArgs(""), "")
	if rc := commands.Switch(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "their default branch")
	if got := gitops.CurrentBranch(a); got != "main" {
		t.Fatalf("a branch=%q", got)
	}
	if got := gitops.CurrentBranch(b); got != "master" {
		t.Fatalf("b branch=%q", got)
	}
}

func TestSwitchToDefaultSkipsUnresolvedRepo(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	testutil.MakeRepo(t, ws, "orphan", "trunk")

	ctx, out, errb := newCtx(t, switchArgs(""), "")
	if rc := commands.Switch(ctx); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "unresolved: 1")
	contains(t, errb.String(), "the default branch is unknown")
}

func TestSwitchToNamedBranchWorksWithoutOriginHead(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	repo := testutil.MakeRepo(t, ws, "orphan", "trunk")
	testutil.Run(t, repo, "git", "branch", "feat")

	ctx, out, _ := newCtx(t, switchArgs("feat"), "")
	if rc := commands.Switch(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "switched")
}
