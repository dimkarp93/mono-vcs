package commands_test

import (
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/commands"
	"github.com/dimkarp93/mono-vcs/internal/testutil"
)

func TestDoFeatSelectsReposWithBranch(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := testutil.MakeRepo(t, ws, "a", "main")
	b := testutil.MakeRepo(t, ws, "b", "main")
	testutil.MakeRepo(t, ws, "c", "main")
	testutil.Run(t, a, "git", "branch", "feat-x")
	testutil.Run(t, b, "git", "checkout", "-b", "feat-x")

	args := doArgs("pwd")
	args.Feature = "feat-x"
	ctx, out, _ := newCtx(args, "")
	if rc := commands.Do(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	o := out.String()
	contains(t, o, ":: a")
	contains(t, o, ":: b")
	notContains(t, o, ":: c")
}

func TestDoFeatNoMatchRunsNowhere(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	testutil.MakeRepo(t, ws, "a", "main")
	args := doArgs("pwd")
	args.Feature = "absent-branch"
	ctx, out, errb := newCtx(args, "")
	if rc := commands.Do(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	notContains(t, out.String(), ":: a")
	contains(t, errb.String(), "no local repo has a branch named")
}
