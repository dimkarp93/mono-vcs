package commands_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/commands"
	"github.com/dimkarp93/mono-vcs/internal/testutil"
)

func pruneArgs(yes bool) *app.Args {
	return &app.Args{Jobs: testutil.I(1), Yes: yes}
}

func TestPruneRemovesUntrackedFile(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := testutil.MakeRepo(t, ws, "p", "main")
	junk := filepath.Join(p, "junk.txt")
	os.WriteFile(junk, []byte("garbage"), 0o644)
	ctx, out, _ := newCtx(pruneArgs(true), "")
	if rc := commands.Prune(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "pruned")
	if exists(junk) {
		t.Fatal("junk should be removed")
	}
}

func TestPruneRemovesUntrackedDirectory(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := testutil.MakeRepo(t, ws, "p", "main")
	nested := filepath.Join(p, "build", "out")
	os.MkdirAll(nested, 0o755)
	os.WriteFile(filepath.Join(nested, "artifact.bin"), []byte("x"), 0o644)
	ctx, out, _ := newCtx(pruneArgs(true), "")
	if rc := commands.Prune(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "pruned")
	if exists(filepath.Join(p, "build")) {
		t.Fatal("build should be removed")
	}
}

func TestPruneReportsItemCount(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := testutil.MakeRepo(t, ws, "p", "main")
	os.WriteFile(filepath.Join(p, "a.txt"), []byte("1"), 0o644)
	os.WriteFile(filepath.Join(p, "b.txt"), []byte("2"), 0o644)
	ctx, out, _ := newCtx(pruneArgs(true), "")
	if rc := commands.Prune(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "removed 2 items")
}

func TestPruneCleanRepoReportsNothing(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	testutil.MakeRepo(t, ws, "p", "main")
	ctx, out, _ := newCtx(pruneArgs(true), "")
	if rc := commands.Prune(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "nothing to prune")
	notContains(t, out.String(), "pruned")
}

func TestPruneRevertsTrackedChanges(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := testutil.MakeRepo(t, ws, "p", "main")
	writeFile(p, "README", "modified, but tracked")
	ctx, out, _ := newCtx(pruneArgs(true), "")
	if rc := commands.Prune(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "pruned")
	data, _ := os.ReadFile(filepath.Join(p, "README"))
	if string(data) != "x" {
		t.Fatalf("tracked file should be reverted to HEAD, got %q", string(data))
	}
}

func TestPruneRevertsStagedChanges(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := testutil.MakeRepo(t, ws, "p", "main")
	writeFile(p, "README", "staged change")
	testutil.Run(t, p, "git", "add", "README")
	ctx, out, _ := newCtx(pruneArgs(true), "")
	if rc := commands.Prune(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "pruned")
	data, _ := os.ReadFile(filepath.Join(p, "README"))
	if string(data) != "x" {
		t.Fatalf("staged file should be reverted to HEAD, got %q", string(data))
	}
}

func TestPruneRespectsGitignore(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := testutil.MakeRepo(t, ws, "p", "main")
	writeFile(p, ".gitignore", "ignored.log\n")
	testutil.Run(t, p, "git", "add", ".gitignore")
	testutil.Run(t, p, "git", "commit", "-m", "ignore")
	os.WriteFile(filepath.Join(p, "ignored.log"), []byte("keep me"), 0o644)
	os.WriteFile(filepath.Join(p, "kill.txt"), []byte("remove me"), 0o644)
	ctx, out, _ := newCtx(pruneArgs(true), "")
	if rc := commands.Prune(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "pruned")
	if !exists(filepath.Join(p, "ignored.log")) {
		t.Fatal("gitignored file should remain")
	}
	if exists(filepath.Join(p, "kill.txt")) {
		t.Fatal("untracked file should be removed")
	}
}

func TestPruneRepoFilter(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := testutil.MakeRepo(t, ws, "a", "main")
	b := testutil.MakeRepo(t, ws, "b", "main")
	os.WriteFile(filepath.Join(a, "junk"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(b, "junk"), []byte("x"), 0o644)
	args := pruneArgs(true)
	args.Repo = []string{"a"}
	ctx, _, _ := newCtx(args, "")
	if rc := commands.Prune(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	if exists(filepath.Join(a, "junk")) {
		t.Fatal("filtered-in repo should be cleaned")
	}
	if !exists(filepath.Join(b, "junk")) {
		t.Fatal("filtered-out repo should be untouched")
	}
}

func TestPruneNoRepos(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	ctx, out, _ := newCtx(pruneArgs(true), "")
	if rc := commands.Prune(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "no local git repositories")
}

func TestPruneListsFilesBeforeDeleting(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := testutil.MakeRepo(t, ws, "p", "main")
	os.WriteFile(filepath.Join(p, "junk.txt"), []byte("garbage"), 0o644)
	ctx, out, _ := newCtx(pruneArgs(true), "")
	commands.Prune(ctx)
	contains(t, out.String(), "junk.txt")
	notContains(t, out.String(), "Would remove")
}

func TestPrunePromptYesDeletes(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := testutil.MakeRepo(t, ws, "p", "main")
	junk := filepath.Join(p, "junk.txt")
	os.WriteFile(junk, []byte("garbage"), 0o644)
	ctx, out, _ := newCtx(pruneArgs(false), "y\n")
	if rc := commands.Prune(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "pruned")
	if exists(junk) {
		t.Fatal("junk should be removed")
	}
}

func TestPrunePromptSkipKeeps(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := testutil.MakeRepo(t, ws, "p", "main")
	junk := filepath.Join(p, "junk.txt")
	os.WriteFile(junk, []byte("garbage"), 0o644)
	ctx, out, _ := newCtx(pruneArgs(false), "s\n")
	commands.Prune(ctx)
	contains(t, out.String(), "user declined")
	contains(t, out.String(), "nothing pruned")
	if !exists(junk) {
		t.Fatal("junk should remain")
	}
}

func TestPrunePromptYesToAll(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := testutil.MakeRepo(t, ws, "a", "main")
	b := testutil.MakeRepo(t, ws, "b", "main")
	os.WriteFile(filepath.Join(a, "junk"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(b, "junk"), []byte("x"), 0o644)
	ctx, _, _ := newCtx(pruneArgs(false), "a\n")
	if rc := commands.Prune(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	if exists(filepath.Join(a, "junk")) || exists(filepath.Join(b, "junk")) {
		t.Fatal("both should be cleaned via yes-all")
	}
}

func TestPrunePromptSkipToAll(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := testutil.MakeRepo(t, ws, "a", "main")
	b := testutil.MakeRepo(t, ws, "b", "main")
	os.WriteFile(filepath.Join(a, "junk"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(b, "junk"), []byte("x"), 0o644)
	ctx, out, _ := newCtx(pruneArgs(false), "sa\n")
	commands.Prune(ctx)
	contains(t, out.String(), "nothing pruned")
	if !exists(filepath.Join(a, "junk")) || !exists(filepath.Join(b, "junk")) {
		t.Fatal("both should remain via skip-all")
	}
}

func TestPruneDryRun(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := testutil.MakeRepo(t, ws, "p", "main")
	junk := filepath.Join(p, "junk.txt")
	os.WriteFile(junk, []byte("garbage"), 0o644)
	a := pruneArgs(false)
	a.DryRun = true
	ctx, out, _ := newCtx(a, "")
	if rc := commands.Prune(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "DRY-RUN: prune")
	if !exists(junk) {
		t.Fatal("nothing should be removed in dry-run")
	}
}
