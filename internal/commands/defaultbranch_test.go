package commands_test

import (
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/commands"
	"github.com/dimkarp93/mono-vcs/internal/testutil"
)

func defaultBranchArgs() *app.Args {
	return &app.Args{Jobs: testutil.I(1)}
}

func makeMixedWorkspace(t *testing.T, ws string) {
	t.Helper()
	testutil.MakeClonedRepo(t, ws, "a", "main")
	testutil.MakeClonedRepo(t, ws, "b", "master")
}

func TestDefaultBranchListsPerRepo(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	makeMixedWorkspace(t, ws)

	ctx, out, _ := newCtx(t, defaultBranchArgs(), "")
	if rc := commands.DefaultBranch(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	o := out.String()
	contains(t, o, "a  main")
	contains(t, o, "b  master")
	contains(t, o, "2 repo(s): main=1, master=1")
}

func TestDefaultBranchRespectsRepoFilter(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	makeMixedWorkspace(t, ws)

	a := defaultBranchArgs()
	a.Repo = []string{"b"}
	ctx, out, _ := newCtx(t, a, "")
	if rc := commands.DefaultBranch(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	o := out.String()
	contains(t, o, "b  master")
	notContains(t, o, "main")
}

func TestDefaultBranchReportsUnresolved(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	testutil.MakeRepo(t, ws, "odd", "trunk")

	ctx, out, errb := newCtx(t, defaultBranchArgs(), "")
	if rc := commands.DefaultBranch(ctx); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "odd  (unresolved)")
	contains(t, out.String(), "1 repo(s): unresolved=1")
	contains(t, errb.String(), "the default branch is unknown")
}

func TestDefaultBranchUnresolvedWithoutStateEntry(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	repo := testutil.MakeRepo(t, ws, "named", "main")
	testutil.MakeRemote(t, t.TempDir(), repo, "main")

	ctx, out, _ := newCtx(t, defaultBranchArgs(), "")
	if rc := commands.DefaultBranch(ctx); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "named  (unresolved)")
}
