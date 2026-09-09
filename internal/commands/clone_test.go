package commands_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/commands"
	"github.com/dimkarp93/mono-vcs/internal/testutil"
)

func cloneArgs(fake *testutil.FakeGitLab, jobs int) *app.Args {
	return &app.Args{GLURL: testutil.S(fake.URL), GLToken: "T", Jobs: testutil.I(jobs)}
}

func seedRemote(t *testing.T, dir, name string) string {
	t.Helper()
	src := testutil.MakeRepo(t, dir, "seed-"+name, "main")
	return testutil.MakeRemote(t, dir, src, "main")
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

func TestCloneFresh(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	remotes := t.TempDir()
	fake := testutil.NewFakeGitLab(t)
	fake.AddProject("alpha/p", seedRemote(t, remotes, "a"))
	fake.AddProject("beta/q", seedRemote(t, remotes, "b"))

	ctx, _, _ := newCtx(cloneArgs(fake, 2), "")
	if rc := commands.Clone(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	if !exists(filepath.Join(ws, "alpha", "p", ".git")) || !exists(filepath.Join(ws, "beta", "q", ".git")) {
		t.Fatal("expected both clones")
	}
}

func TestCloneKeepsFullNamespacePathForSingleGroup(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	remotes := t.TempDir()
	fake := testutil.NewFakeGitLab(t)
	fake.AddProject("mono/a", seedRemote(t, remotes, "a"))
	fake.AddProject("mono/b", seedRemote(t, remotes, "b"))

	ctx, _, _ := newCtx(cloneArgs(fake, 2), "")
	if rc := commands.Clone(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	if !exists(filepath.Join(ws, "mono", "a", ".git")) || !exists(filepath.Join(ws, "mono", "b", ".git")) {
		t.Fatal("expected clones under mono/")
	}
	if exists(filepath.Join(ws, "a")) || exists(filepath.Join(ws, "b")) {
		t.Fatal("expected no stripped top-level dirs")
	}
}

func TestCloneSkipsExisting(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	remotes := t.TempDir()
	fake := testutil.NewFakeGitLab(t)
	r := seedRemote(t, remotes, "a")
	fake.AddProject("alpha/p", r)
	fake.AddProject("beta/q", r)
	testutil.MakeRepo(t, ws, "alpha/p", "main")

	ctx, out, _ := newCtx(cloneArgs(fake, 1), "")
	if rc := commands.Clone(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "skipping 1 already-cloned")
	if !exists(filepath.Join(ws, "beta", "q", ".git")) {
		t.Fatal("expected beta cloned")
	}
}

func TestClonePromptsForNonRepoDirSkip(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	remotes := t.TempDir()
	fake := testutil.NewFakeGitLab(t)
	fake.AddProject("alpha/p", seedRemote(t, remotes, "a"))
	junk := filepath.Join(ws, "alpha", "p")
	os.MkdirAll(junk, 0o755)
	os.WriteFile(filepath.Join(junk, "stuff"), []byte("data"), 0o644)

	ctx, _, _ := newCtx(cloneArgs(fake, 1), "s\n")
	if rc := commands.Clone(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	if !exists(filepath.Join(junk, "stuff")) || exists(filepath.Join(junk, ".git")) {
		t.Fatal("expected junk kept, no clone")
	}
}

func TestCloneForceYesAllReplaces(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	remotes := t.TempDir()
	fake := testutil.NewFakeGitLab(t)
	fake.AddProject("alpha/p", seedRemote(t, remotes, "a"))
	fake.AddProject("beta/q", seedRemote(t, remotes, "b"))
	for _, p := range []string{filepath.Join(ws, "alpha", "p"), filepath.Join(ws, "beta", "q")} {
		os.MkdirAll(p, 0o755)
		os.WriteFile(filepath.Join(p, "junk"), []byte("x"), 0o644)
	}

	ctx, _, _ := newCtx(cloneArgs(fake, 1), "a\n")
	if rc := commands.Clone(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	if !exists(filepath.Join(ws, "alpha", "p", ".git")) || !exists(filepath.Join(ws, "beta", "q", ".git")) {
		t.Fatal("expected both replaced & cloned")
	}
}

func TestCloneNothingToDo(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	fake := testutil.NewFakeGitLab(t)
	ctx, out, _ := newCtx(cloneArgs(fake, 1), "")
	if rc := commands.Clone(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "nothing to clone")
}
