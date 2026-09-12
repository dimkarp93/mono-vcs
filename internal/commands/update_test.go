package commands_test

import (
	"strings"
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/commands"
	"github.com/dimkarp93/mono-vcs/internal/gitops"
	"github.com/dimkarp93/mono-vcs/internal/testutil"
)

func updateArgs() *app.Args {
	return &app.Args{Jobs: testutil.I(1), GLToken: ""}
}

func TestUpdateDirtyIsBlocked(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := placeLinked(t, ws, "p")
	writeFile(p, "README", "dirty")
	ctx, _, errb := newCtx(t, updateArgs(), "")
	if rc := commands.Update(ctx); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, errb.String(), "ERROR")
	contains(t, errb.String(), "uncommitted")
}

func TestUpdateAlreadyOnMainReturnsZero(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	placeLinked(t, ws, "p")
	ctx, out, _ := newCtx(t, updateArgs(), "")
	if rc := commands.Update(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	o := out.String()
	if !strings.Contains(o, "up-to-date") && !strings.Contains(o, "updated") {
		t.Fatalf("out=%s", o)
	}
}

func TestUpdateFeatureTriggersRebase(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := placeLinked(t, ws, "p")
	testutil.Run(t, p, "git", "checkout", "-b", "feature")
	ctx, out, _ := newCtx(t, updateArgs(), "")
	if rc := commands.Update(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "rebasing 1 feature branch")
}

func TestUpdateDryRunDoesNotTouch(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	placeLinked(t, ws, "p")
	a := updateArgs()
	a.DryRun = true
	ctx, out, _ := newCtx(t, a, "")
	if rc := commands.Update(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "DRY-RUN: update")
}

func TestUpdateUsesPerRepoDefaultBranch(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := testutil.MakeClonedRepo(t, ws, "p", "master")
	testutil.Run(t, p, "git", "checkout", "-b", "feature")

	ctx, out, _ := newCtx(t, updateArgs(), "")
	if rc := commands.Update(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	o := out.String()
	contains(t, o, "[master]")
	contains(t, o, "rebase onto master")
	if got := gitops.CurrentBranch(p); got != "feature" {
		t.Fatalf("branch=%q", got)
	}
}

func TestUpdateSkipsRepoWithoutResolvableDefault(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	testutil.MakeClonedRepo(t, ws, "good", "main")
	testutil.MakeRepo(t, ws, "orphan", "trunk")

	ctx, out, errb := newCtx(t, updateArgs(), "")
	if rc := commands.Update(ctx); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, errb.String(), "the default branch is unknown")
	contains(t, errb.String(), "1 repo(s) skipped")
	contains(t, out.String(), "good")
}
