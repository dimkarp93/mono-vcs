package commands_test

import (
	"strings"
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/commands"
	"github.com/dimkarp93/mono-vcs/internal/output"
	"github.com/dimkarp93/mono-vcs/internal/testutil"
)

func listArgs(fake *testutil.FakeGitLab) *app.Args {
	return &app.Args{
		GLURL: testutil.S(fake.URL), GLToken: "T", Jobs: testutil.I(1),
		MainBranch: testutil.S("main"),
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

func TestListUsesFullNamespacePath(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	fake := testutil.NewFakeGitLab(t)
	fake.AddProject("mono/a", "")
	fake.AddProject("mono/b", "")

	a := listArgs(fake)
	a.All = true
	ctx, out, _ := newCtx(a, "")
	commands.List(ctx)
	contains(t, out.String(), "mono/a")
	contains(t, out.String(), "mono/b")
	notContains(t, out.String(), "stripped common top-level group")
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

func TestListAlignsBranchColumn(t *testing.T) {
	t.Setenv("COLUMNS", "100")
	ws := t.TempDir()
	t.Chdir(ws)
	fake := testutil.NewFakeGitLab(t)
	short, _, _ := makeSyncedPair(t, ws, fake, "a/s")
	long, _, _ := makeSyncedPair(t, ws, fake, "group/subgroup/much-longer-name")
	testutil.Run(t, short, "git", "checkout", "-b", "feat")
	testutil.Run(t, long, "git", "checkout", "-b", "feat")

	a := listArgs(fake)
	a.All = true
	ctx, out, _ := newCtx(a, "")
	commands.List(ctx)

	var cols []int
	for _, line := range strings.Split(out.String(), "\n") {
		i := strings.Index(line, "[feat]")
		if i < 0 {
			continue
		}
		cols = append(cols, output.DisplayWidth(line[:i]))
	}
	if len(cols) != 2 {
		t.Fatalf("expected 2 branch cells, got %d:\n%s", len(cols), out.String())
	}
	if cols[0] != cols[1] {
		t.Fatalf("branch column not aligned: %d vs %d\n%s", cols[0], cols[1], out.String())
	}
}

func TestListTruncatesOverlongPaths(t *testing.T) {
	t.Setenv("COLUMNS", "60")
	ws := t.TempDir()
	t.Chdir(ws)
	fake := testutil.NewFakeGitLab(t)
	name := "группа/подгруппа/очень-длинное-имя-репозитория-не-влезающее-в-строку"
	makeSyncedPair(t, ws, fake, name)

	a := listArgs(fake)
	a.All = true
	ctx, out, _ := newCtx(a, "")
	commands.List(ctx)

	for _, line := range strings.Split(out.String(), "\n") {
		if !strings.HasPrefix(line, "группа/") {
			continue
		}
		if w := output.DisplayWidth(line); w > 60 {
			t.Fatalf("line too wide (%d): %q", w, line)
		}
		if !strings.Contains(line, "…") {
			t.Fatalf("expected an ellipsis in %q", line)
		}
		return
	}
	t.Fatalf("repo line not found:\n%s", out.String())
}
