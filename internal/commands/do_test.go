package commands_test

import (
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/commands"
	"github.com/dimkarp93/mono-vcs/internal/testutil"
)

func doArgs(action ...string) *app.Args {
	return &app.Args{Action: action, Jobs: testutil.I(1), MainBranch: testutil.S("main")}
}

func TestDoRequiresAction(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	testutil.MakeRepo(t, ws, "p", "main")
	ctx, _, _ := newCtx(doArgs(), "")
	if rc := commands.Do(ctx); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
}

func TestDoRunsCommandInEachRepo(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	testutil.MakeRepo(t, ws, "a", "main")
	testutil.MakeRepo(t, ws, "b", "main")
	ctx, out, _ := newCtx(doArgs("pwd"), "")
	if rc := commands.Do(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), ":: a")
	contains(t, out.String(), ":: b")
}

func TestDoFailurePropagates(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	testutil.MakeRepo(t, ws, "a", "main")
	ctx, _, errb := newCtx(doArgs("false"), "")
	if rc := commands.Do(ctx); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, errb.String(), "1 command")
}

func TestDoDryRun(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	testutil.MakeRepo(t, ws, "a", "main")
	a := doArgs("echo", "hi")
	a.DryRun = true
	ctx, out, _ := newCtx(a, "")
	if rc := commands.Do(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "DRY-RUN: do echo hi")
}
