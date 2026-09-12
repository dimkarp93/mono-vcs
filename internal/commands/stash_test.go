package commands_test

import (
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/commands"
	"github.com/dimkarp93/mono-vcs/internal/gitops"
	"github.com/dimkarp93/mono-vcs/internal/testutil"
)

func stashArgs() *app.Args {
	return &app.Args{Jobs: testutil.I(1)}
}

func TestStashSkipsClean(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	testutil.MakeRepo(t, ws, "p", "main")
	ctx, out, _ := newCtx(t, stashArgs(), "")
	if rc := commands.Stash(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "nothing")
}

func TestStashStashesUnstaged(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := testutil.MakeRepo(t, ws, "p", "main")
	writeFile(p, "README", "dirty")
	ctx, out, _ := newCtx(t, stashArgs(), "")
	if rc := commands.Stash(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "stashed")
	if gitops.StashCount(p) != 1 {
		t.Fatal("expected 1 stash")
	}
}

func TestUnstashPops(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := testutil.MakeRepo(t, ws, "p", "main")
	writeFile(p, "README", "dirty")
	c1, _, _ := newCtx(t, stashArgs(), "")
	commands.Stash(c1)
	ctx, out, _ := newCtx(t, stashArgs(), "")
	if rc := commands.Unstash(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "popped")
	if gitops.StashCount(p) != 0 {
		t.Fatal("expected 0 stash")
	}
}

func TestClearStashDropsAll(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := testutil.MakeRepo(t, ws, "p", "main")
	writeFile(p, "README", "a")
	commands.Stash(mustCtx(t, stashArgs()))
	writeFile(p, "README", "b")
	commands.Stash(mustCtx(t, stashArgs()))
	if gitops.StashCount(p) != 2 {
		t.Fatal("expected 2")
	}
	ctx, out, _ := newCtx(t, stashArgs(), "")
	if rc := commands.ClearStash(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "cleared")
	if gitops.StashCount(p) != 0 {
		t.Fatal("expected 0")
	}
}

func TestHistoryStashListsEntries(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := testutil.MakeRepo(t, ws, "p", "main")
	writeFile(p, "README", "a")
	commands.Stash(mustCtx(t, stashArgs()))
	ctx, out, _ := newCtx(t, stashArgs(), "")
	if rc := commands.HistoryStash(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "mono-vcs stash")
	contains(t, out.String(), "total: 1")
}

func TestHistoryStashEmpty(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	testutil.MakeRepo(t, ws, "p", "main")
	ctx, out, _ := newCtx(t, stashArgs(), "")
	if rc := commands.HistoryStash(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "total: 0")
}

func TestStashDryRun(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	testutil.MakeRepo(t, ws, "p", "main")
	a := stashArgs()
	a.DryRun = true
	ctx, out, _ := newCtx(t, a, "")
	if rc := commands.Stash(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "DRY-RUN: stash")
}

func mustCtx(t *testing.T, a *app.Args) *app.Context {
	t.Helper()
	ctx, _, _ := newCtx(t, a, "")
	return ctx
}
