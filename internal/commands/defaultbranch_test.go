package commands_test

import (
	"testing"

	"mono-vcs/internal/app"
	"mono-vcs/internal/commands"
	"mono-vcs/internal/testutil"
)

func defaultBranchArgs() *app.Args {
	return &app.Args{Jobs: testutil.I(1), MainBranch: testutil.S("main")}
}

func makeMixedWorkspace(t *testing.T, ws string) {
	t.Helper()
	a := testutil.MakeRepo(t, ws, "a", "main")
	testutil.MakeRemote(t, t.TempDir(), a, "main")
	b := testutil.MakeRepo(t, ws, "b", "master")
	testutil.MakeRemote(t, t.TempDir(), b, "master")
}

func TestDefaultBranchListsPerRepo(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	makeMixedWorkspace(t, ws)

	ctx, out, _ := newCtx(defaultBranchArgs(), "")
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
	ctx, out, _ := newCtx(a, "")
	if rc := commands.DefaultBranch(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	o := out.String()
	contains(t, o, "b  master")
	notContains(t, o, "main")
}

func TestDefaultBranchMarksFallback(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	testutil.MakeRepo(t, ws, "odd", "trunk")

	ctx, out, _ := newCtx(defaultBranchArgs(), "")
	if rc := commands.DefaultBranch(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "odd  main  (fallback)")
}
