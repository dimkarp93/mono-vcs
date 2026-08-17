package commands_test

import (
	"path/filepath"
	"testing"

	"mono-vcs/internal/app"
	"mono-vcs/internal/commands"
	"mono-vcs/internal/testutil"
)

func pullArgs(fake *testutil.FakeGitLab) *app.Args {
	return &app.Args{GLURL: testutil.S(fake.URL), GLToken: "T", Jobs: testutil.I(1), MainBranch: testutil.S("main")}
}

func TestPullNothingWhenAllGreen(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	fake := testutil.NewFakeGitLab(t)
	makeSyncedPair(t, ws, fake, "alpha/p")
	testutil.MakeRepo(t, ws, "beta/q", "main")

	ctx, out, _ := newCtx(pullArgs(fake), "")
	if rc := commands.Pull(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "nothing to pull")
}

func TestPullMatchesFullNamespacePathForSingleGroup(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	fake := testutil.NewFakeGitLab(t)
	makeSyncedPair(t, ws, fake, "mono/a")
	makeSyncedPair(t, ws, fake, "mono/b")

	ctx, out, _ := newCtx(pullArgs(fake), "")
	if rc := commands.Pull(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "skipping 2 green repo(s)")
	notContains(t, out.String(), "red repo(s)")
}

func TestPullRunsOnlyOnYellow(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	fake := testutil.NewFakeGitLab(t)
	remotes := t.TempDir()
	seed := filepath.Join(remotes, "y.git")
	testutil.InitRepo(t, seed, true, "main")

	yellowLocal := filepath.Join(ws, "alpha", "p")
	testutil.Run(t, ws, "git", "clone", seed, yellowLocal)
	testutil.Run(t, yellowLocal, "git", "config", "user.email", "a@b")
	testutil.Run(t, yellowLocal, "git", "config", "user.name", "a")
	testutil.Commit(t, yellowLocal, "init", "f", "x")
	testutil.Run(t, yellowLocal, "git", "push", "-u", "origin", "main")

	side := filepath.Join(remotes, "side")
	testutil.Run(t, remotes, "git", "clone", seed, side)
	testutil.Run(t, side, "git", "config", "user.email", "a@b")
	testutil.Run(t, side, "git", "config", "user.name", "a")
	testutil.Commit(t, side, "advance", "g", "y")
	testutil.Run(t, side, "git", "push", "origin", "main")
	remoteSHA := headSHA(t, side)

	id := fake.AddProject("alpha/p", "")
	fake.SetBranchSHA(id, "main", remoteSHA)
	fake.AddProject("beta/other", "")

	ctx, out, _ := newCtx(pullArgs(fake), "")
	if rc := commands.Pull(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "pulling 1 yellow")
	if !exists(filepath.Join(yellowLocal, "g")) {
		t.Fatal("expected g pulled into local")
	}
}

func TestPullDryRunSkipsNetwork(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	testutil.MakeRepo(t, ws, "alpha/p", "main")
	a := &app.Args{GLURL: testutil.S("https://gl.example"), GLToken: "", Jobs: testutil.I(1), MainBranch: testutil.S("main"), DryRun: true}
	ctx, out, _ := newCtx(a, "")
	if rc := commands.Pull(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "DRY-RUN: pull")
}

func TestPullNoLocalReturnsZero(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	fake := testutil.NewFakeGitLab(t)
	ctx, out, _ := newCtx(pullArgs(fake), "")
	if rc := commands.Pull(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "no local git repositories")
}
