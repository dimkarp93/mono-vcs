package commands_test

import (
	"path/filepath"
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/commands"
	"github.com/dimkarp93/mono-vcs/internal/testutil"
)

func emptyStatePath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "state.json")
}

func TestDefaultsFetchedFromGitLabWhenStateIsEmpty(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	fake := testutil.NewFakeGitLab(t)
	repo := testutil.MakeClonedRepo(t, ws, "alpha/p", "main")
	id := fake.AddProject("alpha/p", "")
	fake.SetDefaultBranch(id, "main")

	db := emptyStatePath(t)
	a := &app.Args{Branch: "MVPAY-1", Jobs: testutil.I(1), GLURL: testutil.S(fake.URL), DBPath: testutil.S(db), GLToken: "t"}
	ctx, out, _ := newCtx(t, a, "")
	if rc := commands.New(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "created")
	if b := testutil.StoredDefaultBranch(t, db, repo); b != "main" {
		t.Fatalf("stored=%q", b)
	}
}

func TestDefaultsPromptForTokenOnlyWhenStateMisses(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	fake := testutil.NewFakeGitLab(t)
	testutil.MakeClonedRepo(t, ws, "alpha/p", "main")
	id := fake.AddProject("alpha/p", "")
	fake.SetDefaultBranch(id, "main")

	db := emptyStatePath(t)
	a := &app.Args{Branch: "MVPAY-1", Jobs: testutil.I(1), GLURL: testutil.S(fake.URL), DBPath: testutil.S(db)}
	ctx, _, _ := newCtx(t, a, "")
	asked := 0
	ctx.TokenFunc = func(bool) (string, error) {
		asked++
		return "t", nil
	}
	if rc := commands.New(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	if asked != 1 {
		t.Fatalf("asked=%d", asked)
	}
}

func TestDefaultsServedFromStateWithoutGitLab(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	repo := testutil.MakeClonedRepo(t, ws, "alpha/p", "main")

	db := testutil.StatePath(t, map[string]string{repo: "main"})
	a := &app.Args{Branch: "MVPAY-1", Jobs: testutil.I(1), GLURL: testutil.S("http://127.0.0.1:1"), DBPath: testutil.S(db)}
	ctx, out, _ := newCtx(t, a, "")
	ctx.TokenFunc = func(bool) (string, error) {
		t.Fatal("GitLab must not be contacted when the state db knows the branch")
		return "", nil
	}
	if rc := commands.New(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "created")
}

func TestDefaultsStateWinsOverStaleOriginHead(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	repo := testutil.MakeClonedRepo(t, ws, "alpha/p", "master")
	testutil.Run(t, repo, "git", "branch", "main")

	db := testutil.StatePath(t, map[string]string{repo: "main"})
	a := &app.Args{Jobs: testutil.I(1), GLURL: testutil.S("http://127.0.0.1:1"), DBPath: testutil.S(db)}
	ctx, out, _ := newCtx(t, a, "")
	if rc := commands.DefaultBranch(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "alpha/p  main")
}

func TestDefaultsUnresolvedWhenGitLabHasNoProject(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	fake := testutil.NewFakeGitLab(t)
	testutil.MakeClonedRepo(t, ws, "alpha/p", "main")

	db := emptyStatePath(t)
	a := &app.Args{Jobs: testutil.I(1), GLURL: testutil.S(fake.URL), DBPath: testutil.S(db), GLToken: "t"}
	ctx, out, errb := newCtx(t, a, "")
	if rc := commands.DefaultBranch(ctx); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "alpha/p  (unresolved)")
	contains(t, errb.String(), "the default branch is unknown")
}

func TestDefaultsUnreachableGitLabIsReported(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	testutil.MakeClonedRepo(t, ws, "alpha/p", "main")

	db := emptyStatePath(t)
	a := &app.Args{Jobs: testutil.I(1), GLURL: testutil.S("http://127.0.0.1:1"), DBPath: testutil.S(db), GLToken: "t"}
	ctx, out, errb := newCtx(t, a, "")
	if rc := commands.DefaultBranch(ctx); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, errb.String(), "Cannot reach GitLab")
	contains(t, out.String(), "alpha/p  (unresolved)")
}
