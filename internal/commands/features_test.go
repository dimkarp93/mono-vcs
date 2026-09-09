package commands_test

import (
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/commands"
	"github.com/dimkarp93/mono-vcs/internal/testutil"
)

func featuresArgs() *app.Args {
	return &app.Args{MainBranch: testutil.S("main")}
}

func TestFeaturesEmpty(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	testutil.MakeRepo(t, ws, "p", "main")
	ctx, out, _ := newCtx(featuresArgs(), "")
	if rc := commands.Features(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "no feature branches found")
}

func TestFeaturesListsBranchAndActives(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := testutil.MakeRepo(t, ws, "a", "main")
	b := testutil.MakeRepo(t, ws, "b", "main")
	testutil.Run(t, a, "git", "branch", "feat-x")
	testutil.Run(t, b, "git", "checkout", "-b", "feat-x")
	ctx, out, _ := newCtx(featuresArgs(), "")
	if rc := commands.Features(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	o := out.String()
	contains(t, o, "feat-x")
	contains(t, o, "a")
	contains(t, o, "b")
	contains(t, o, "1 feature branch(es) across 2 repo(s)")
}
