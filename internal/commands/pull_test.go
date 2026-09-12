package commands_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/commands"
	"github.com/dimkarp93/mono-vcs/internal/testutil"
)

func pullArgs(fake *testutil.FakeGitLab) *app.Args {
	return &app.Args{GLURL: testutil.S(fake.URL), GLToken: "T", Jobs: testutil.I(1)}
}

func makeBehindRepo(t *testing.T, ws, name, branch string) string {
	t.Helper()
	remotes := t.TempDir()
	seed := filepath.Join(remotes, "seed.git")
	testutil.InitRepo(t, seed, true, branch)

	local := filepath.Join(ws, name)
	testutil.Run(t, ws, "git", "clone", seed, local)
	testutil.Run(t, local, "git", "config", "user.email", "a@b")
	testutil.Run(t, local, "git", "config", "user.name", "a")
	testutil.Commit(t, local, "init", "f", "x")
	testutil.Run(t, local, "git", "push", "-u", "origin", branch)
	testutil.Run(t, local, "git", "remote", "set-head", "origin", branch)

	side := filepath.Join(remotes, "side")
	testutil.Run(t, remotes, "git", "clone", seed, side)
	testutil.Run(t, side, "git", "config", "user.email", "a@b")
	testutil.Run(t, side, "git", "config", "user.name", "a")
	testutil.Commit(t, side, "advance", "g", "y")
	testutil.Run(t, side, "git", "push", "origin", branch)
	return local
}

func localSHA(t *testing.T, repo, ref string) string {
	t.Helper()
	return strings.TrimSpace(testutil.Run(t, repo, "git", "rev-parse", ref))
}

func TestPullUpdatesRepoSittingOnFeatureBranch(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	fake := testutil.NewFakeGitLab(t)
	local := makeBehindRepo(t, ws, "alpha/p", "main")
	testutil.Run(t, local, "git", "checkout", "-b", "feature")
	writeFile(local, "f", "work in progress")
	fake.AddProject("alpha/p", "")

	ctx, out, _ := newCtx(t, pullArgs(fake), "")
	if rc := commands.Pull(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "updated: 1")
	if b := strings.TrimSpace(testutil.Run(t, local, "git", "rev-parse", "--abbrev-ref", "HEAD")); b != "feature" {
		t.Fatalf("branch=%q", b)
	}
	if localSHA(t, local, "refs/heads/main") != localSHA(t, local, "refs/remotes/origin/main") {
		t.Fatal("main must have caught up with origin/main")
	}
	if exists(filepath.Join(local, "g")) {
		t.Fatal("the worktree must not be touched")
	}
}

func TestPullUpdatesRepoOnDefaultBranch(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	fake := testutil.NewFakeGitLab(t)
	local := makeBehindRepo(t, ws, "alpha/p", "main")
	fake.AddProject("alpha/p", "")

	ctx, out, _ := newCtx(t, pullArgs(fake), "")
	if rc := commands.Pull(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "updated: 1")
	if !exists(filepath.Join(local, "g")) {
		t.Fatal("expected g pulled into the worktree")
	}
}

func TestPullReportsUpToDate(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	fake := testutil.NewFakeGitLab(t)
	testutil.MakeClonedRepo(t, ws, "alpha/p", "main")
	fake.AddProject("alpha/p", "")

	ctx, out, _ := newCtx(t, pullArgs(fake), "")
	if rc := commands.Pull(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "up-to-date: 1")
	contains(t, out.String(), "updated: 0")
}

func TestPullUpdatesRepoMissingFromGitLab(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	fake := testutil.NewFakeGitLab(t)
	local := makeBehindRepo(t, ws, "alpha/p", "main")
	testutil.Run(t, local, "git", "checkout", "-b", "feature")

	ctx, out, _ := newCtx(t, pullArgs(fake), "")
	if rc := commands.Pull(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "updated: 1")
	if localSHA(t, local, "refs/heads/main") != localSHA(t, local, "refs/remotes/origin/main") {
		t.Fatal("main must have caught up with origin/main")
	}
}

func TestPullFallsBackToStateDbWhenGitLabIsUnreachable(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	local := makeBehindRepo(t, ws, "alpha/p", "main")
	testutil.Run(t, local, "git", "checkout", "-b", "feature")

	a := &app.Args{GLURL: testutil.S("http://127.0.0.1:1"), GLToken: "T", Jobs: testutil.I(1)}
	ctx, out, errb := newCtx(t, a, "")
	if rc := commands.Pull(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, errb.String(), "falling back to the state db")
	contains(t, out.String(), "updated: 1")
}

func TestPullReportsDivergedDefaultBranch(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	fake := testutil.NewFakeGitLab(t)
	local := makeBehindRepo(t, ws, "alpha/p", "main")
	testutil.Commit(t, local, "local only", "own", "mine")
	before := localSHA(t, local, "refs/heads/main")
	testutil.Run(t, local, "git", "checkout", "-b", "feature")
	fake.AddProject("alpha/p", "")

	ctx, out, errb := newCtx(t, pullArgs(fake), "")
	if rc := commands.Pull(ctx); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "diverged: 1")
	contains(t, errb.String(), "has diverged from origin")
	if localSHA(t, local, "refs/heads/main") != before {
		t.Fatal("a diverged main must stay untouched")
	}
}

func TestPullDryRunSkipsNetwork(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	testutil.MakeRepo(t, ws, "alpha/p", "main")
	a := &app.Args{GLURL: testutil.S("https://gl.example"), GLToken: "", Jobs: testutil.I(1), DryRun: true}
	ctx, out, _ := newCtx(t, a, "")
	if rc := commands.Pull(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "DRY-RUN: pull")
}

func TestPullNoLocalReturnsZero(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	fake := testutil.NewFakeGitLab(t)
	ctx, out, _ := newCtx(t, pullArgs(fake), "")
	if rc := commands.Pull(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "no local git repositories")
}

func makeStaleHeadRepo(t *testing.T, ws string, fake *testutil.FakeGitLab, name string) int {
	t.Helper()
	local := testutil.MakeRepo(t, ws, name, "master")
	testutil.MakeRemote(t, t.TempDir(), local, "master")
	testutil.Run(t, local, "git", "branch", "main")
	testutil.Run(t, local, "git", "push", "origin", "main")
	testutil.Run(t, local, "git", "remote", "set-head", "origin", "master")

	id := fake.AddProject(name, "")
	fake.SetDefaultBranch(id, "main")
	fake.SetBranchSHA(id, "main", headSHA(t, local))
	return id
}

func TestPullAdoptsDefaultBranchFromGitLab(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	fake := testutil.NewFakeGitLab(t)
	makeStaleHeadRepo(t, ws, fake, "alpha/p")

	ctx, out, _ := newCtx(t, pullArgs(fake), "")
	if rc := commands.Pull(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "refreshed the default branch of 1 repo(s) from GitLab")
	contains(t, out.String(), "up-to-date: 1")

	if b := testutil.StoredDefaultBranch(t, ctx.Args.GetDBPath(), filepath.Join(ws, "alpha", "p")); b != "main" {
		t.Fatalf("stored=%q", b)
	}
}

func TestPullKeepsStateDbBranchWhenGitLabOmitsDefaultBranch(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	fake := testutil.NewFakeGitLab(t)
	local := testutil.MakeClonedRepo(t, ws, "alpha/p", "master")
	id := fake.AddProject("alpha/p", "")
	fake.SetBranchSHA(id, "master", headSHA(t, local))

	ctx, out, _ := newCtx(t, pullArgs(fake), "")
	if rc := commands.Pull(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	notContains(t, out.String(), "refreshed the default branch")
	contains(t, out.String(), "up-to-date: 1")
}
