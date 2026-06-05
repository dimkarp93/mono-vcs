package commands_test

import (
	"strings"
	"testing"

	"mono-vcs/internal/app"
	"mono-vcs/internal/commands"
	"mono-vcs/internal/testutil"
)

func updateArgs() *app.Args {
	return &app.Args{Jobs: testutil.I(1), MainBranch: testutil.S("main"), GLToken: ""}
}

func TestUpdateMainDirtyIsBlocked(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := placeLinked(t, ws, "p")
	writeFile(p, "README", "dirty")
	ctx, _, errb := newCtx(updateArgs(), "")
	if rc := commands.UpdateMain(ctx); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, errb.String(), "ERROR")
	contains(t, errb.String(), "uncommitted")
}

func TestUpdateMainAlreadyOnMainReturnsZero(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	placeLinked(t, ws, "p")
	ctx, out, _ := newCtx(updateArgs(), "")
	if rc := commands.UpdateMain(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	o := out.String()
	if !strings.Contains(o, "up-to-date") && !strings.Contains(o, "updated") {
		t.Fatalf("out=%s", o)
	}
}

func TestUpdateMainFeatureTriggersRebase(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := placeLinked(t, ws, "p")
	testutil.Run(t, p, "git", "checkout", "-b", "feature")
	ctx, out, _ := newCtx(updateArgs(), "")
	if rc := commands.UpdateMain(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "rebasing 1 feature branch")
}

func TestUpdateMainDryRunDoesNotTouch(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	placeLinked(t, ws, "p")
	a := updateArgs()
	a.DryRun = true
	ctx, out, _ := newCtx(a, "")
	if rc := commands.UpdateMain(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "DRY-RUN: update")
}
