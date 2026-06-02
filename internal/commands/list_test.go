package commands_test

import (
	"testing"

	"mono-vcs/internal/app"
	"mono-vcs/internal/commands"
	"mono-vcs/internal/testutil"
)

func listArgs(fake *testutil.FakeGitLab) *app.Args {
	return &app.Args{
		GLURL: testutil.S(fake.URL), GLToken: "T", Jobs: testutil.I(1),
		NoColor: testutil.B(true), MainBranch: testutil.S("main"),
	}
}

func TestListGreenSyncedPairHiddenByDefault(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	fake := testutil.NewFakeGitLab(t)
	makeSyncedPair(t, ws, fake, "grp/proj")

	ctx, out, _ := newCtx(listArgs(fake), "")
	commands.List(ctx)
	notContains(t, out.String(), "grp/proj")

	a := listArgs(fake)
	a.All = true
	ctx2, out2, _ := newCtx(a, "")
	commands.List(ctx2)
	contains(t, out2.String(), "grp/proj")
}

func TestListYellowWhenSHADiverges(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	fake := testutil.NewFakeGitLab(t)
	_, id, _ := makeSyncedPair(t, ws, fake, "alpha/p")
	makeSyncedPair(t, ws, fake, "beta/q")
	fake.SetBranchSHA(id, "main", "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef")

	ctx, out, _ := newCtx(listArgs(fake), "")
	commands.List(ctx)
	contains(t, out.String(), "alpha/p")
}

func TestListRedLocalOnly(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	fake := testutil.NewFakeGitLab(t)
	testutil.MakeRepo(t, ws, "lonely", "main")

	a := listArgs(fake)
	a.NonOrigin = true
	ctx, out, _ := newCtx(a, "")
	commands.List(ctx)
	contains(t, out.String(), "lonely")
}

func TestListGrayRemoteOnly(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	fake := testutil.NewFakeGitLab(t)
	fake.AddProject("alpha/p", "")
	fake.AddProject("beta/q", "")

	ctx, out, _ := newCtx(listArgs(fake), "")
	commands.List(ctx)
	contains(t, out.String(), "alpha/p")
	contains(t, out.String(), "beta/q")
}

func TestListBlueFeatureBranch(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	fake := testutil.NewFakeGitLab(t)
	repo, _, _ := makeSyncedPair(t, ws, fake, "alpha/p")
	makeSyncedPair(t, ws, fake, "beta/q")
	testutil.Run(t, repo, "git", "checkout", "-b", "feature")

	ctx, out, _ := newCtx(listArgs(fake), "")
	commands.List(ctx)
	contains(t, out.String(), "alpha/p")
	contains(t, out.String(), "[feature]")
}

func TestListDirtyMarker(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	fake := testutil.NewFakeGitLab(t)
	repo, _, _ := makeSyncedPair(t, ws, fake, "alpha/p")
	makeSyncedPair(t, ws, fake, "beta/q")
	if err := writeFile(repo, "README", "dirty"); err != nil {
		t.Fatal(err)
	}

	a := listArgs(fake)
	a.Dirty = true
	ctx, out, _ := newCtx(a, "")
	commands.List(ctx)
	contains(t, out.String(), "alpha/p")
	contains(t, out.String(), "✗")
}

func TestListCommonTopGroupStripped(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	fake := testutil.NewFakeGitLab(t)
	fake.AddProject("mono/a", "")
	fake.AddProject("mono/b", "")

	a := listArgs(fake)
	a.All = true
	ctx, out, _ := newCtx(a, "")
	commands.List(ctx)
	contains(t, out.String(), "stripped common top-level group: mono/")
}

func TestListRepoFilter(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	fake := testutil.NewFakeGitLab(t)
	makeSyncedPair(t, ws, fake, "libs/a")
	makeSyncedPair(t, ws, fake, "tools/b")

	a := listArgs(fake)
	a.All = true
	a.Repo = []string{"libs/"}
	ctx, out, _ := newCtx(a, "")
	commands.List(ctx)
	contains(t, out.String(), "libs/a")
	notContains(t, out.String(), "tools/b")
}
