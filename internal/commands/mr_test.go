package commands_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/commands"
	"github.com/dimkarp93/mono-vcs/internal/testutil"
)

func mrArgs(branch string) *app.Args {
	return &app.Args{Command: "mr", Branch: branch, Jobs: testutil.I(1)}
}

func remoteOf(t *testing.T, repo string) string {
	t.Helper()
	return strings.TrimSpace(testutil.Run(t, repo, "git", "remote", "get-url", "origin"))
}

func allowPushOptions(t *testing.T, repo string) {
	t.Helper()
	testutil.Run(t, remoteOf(t, repo), "git", "config", "receive.advertisePushOptions", "true")
}

func remoteHasBranch(t *testing.T, repo, branch string) bool {
	t.Helper()
	cmd := exec.Command("git", "-C", remoteOf(t, repo), "rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
	return cmd.Run() == nil
}

func onBranch(t *testing.T, repo, branch string) {
	t.Helper()
	testutil.Run(t, repo, "git", "checkout", "-b", branch)
	testutil.Commit(t, repo, "work on "+branch, "work.txt", branch)
}

func TestMRExplicitBranchMissing(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	placeLinked(t, ws, "a")
	ctx, _, errb := newCtx(t, mrArgs("absent"), "")
	if rc := commands.MR(ctx); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, errb.String(), "no local repo has a branch named `absent`")
}

func TestMRExplicitBranchNotCheckedOutEverywhere(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := placeLinked(t, ws, "a")
	b := placeLinked(t, ws, "b")
	onBranch(t, a, "PROJ-1-feat")
	onBranch(t, b, "PROJ-1-feat")
	testutil.Run(t, b, "git", "checkout", "main")

	ctx, out, errb := newCtx(t, mrArgs("PROJ-1-feat"), "")
	if rc := commands.MR(ctx); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, errb.String(), "is not checked out in every repo")
	contains(t, errb.String(), "b")
	notContains(t, out.String(), "pushing")
}

func TestMRInferredFromContext(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := placeLinked(t, ws, "a")
	b := placeLinked(t, ws, "b")
	placeLinked(t, ws, "c")
	onBranch(t, a, "PROJ-1-feat")
	onBranch(t, b, "PROJ-1-feat")
	allowPushOptions(t, a)
	allowPushOptions(t, b)

	ctx, out, errb := newCtx(t, mrArgs(""), "")
	if rc := commands.MR(ctx); rc != 0 {
		t.Fatalf("rc=%d err=%s out=%s", rc, errb.String(), out.String())
	}
	o := out.String()
	contains(t, o, "pushing `PROJ-1-feat`")
	contains(t, o, "pushed: 2")
	notContains(t, o, "c")
	for _, p := range []string{a, b} {
		sha := strings.TrimSpace(testutil.Run(t, remoteOf(t, p), "git", "rev-parse", "refs/heads/PROJ-1-feat"))
		if sha != headSHA(t, p) {
			t.Fatalf("%s: remote sha=%s", p, sha)
		}
	}
}

func TestMRSecondRunIsUpToDate(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := placeLinked(t, ws, "a")
	onBranch(t, a, "PROJ-1-feat")
	allowPushOptions(t, a)

	ctx, _, _ := newCtx(t, mrArgs("PROJ-1-feat"), "")
	if rc := commands.MR(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	ctx2, out, _ := newCtx(t, mrArgs("PROJ-1-feat"), "")
	if rc := commands.MR(ctx2); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "up-to-date: 1")
}

func TestMRAmbiguousContext(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := placeLinked(t, ws, "a")
	b := placeLinked(t, ws, "b")
	onBranch(t, a, "PROJ-1-feat")
	onBranch(t, b, "feat-y")

	ctx, _, errb := newCtx(t, mrArgs(""), "")
	if rc := commands.MR(ctx); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, errb.String(), "several feature branches are checked out")
	contains(t, errb.String(), "PROJ-1-feat, feat-y")
}

func TestMRNoActiveFeature(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	placeLinked(t, ws, "a")
	ctx, _, errb := newCtx(t, mrArgs(""), "")
	if rc := commands.MR(ctx); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, errb.String(), "every repo is on its default branch")
}

func TestMRInferredButRepoNotOnFeature(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := placeLinked(t, ws, "a")
	b := placeLinked(t, ws, "b")
	onBranch(t, a, "PROJ-1-feat")
	onBranch(t, b, "PROJ-1-feat")
	testutil.Run(t, b, "git", "checkout", "main")

	ctx, _, errb := newCtx(t, mrArgs(""), "")
	if rc := commands.MR(ctx); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, errb.String(), "is not checked out in every repo")
}

func TestMRDirtyRefusesEverything(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := placeLinked(t, ws, "a")
	b := placeLinked(t, ws, "b")
	onBranch(t, a, "PROJ-1-feat")
	onBranch(t, b, "PROJ-1-feat")
	allowPushOptions(t, a)
	allowPushOptions(t, b)
	writeFile(b, "README", "dirty")

	ctx, out, errb := newCtx(t, mrArgs("PROJ-1-feat"), "")
	if rc := commands.MR(ctx); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, errb.String(), "uncommitted changes")
	notContains(t, out.String(), "pushing")
	if remoteHasBranch(t, a, "PROJ-1-feat") {
		t.Fatal("push happened despite dirty repo")
	}
}

func TestMRDryRunTouchesNothing(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := placeLinked(t, ws, "a")
	onBranch(t, a, "PROJ-1-feat")
	args := mrArgs("")
	args.DryRun = true
	args.Title = "my mr"

	ctx, out, _ := newCtx(t, args, "")
	if rc := commands.MR(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	o := out.String()
	contains(t, o, "DRY-RUN: mr PROJ-1-feat")
	contains(t, o, "merge_request.create")
	contains(t, o, "merge_request.title=[PROJ-1] my mr")
	if remoteHasBranch(t, a, "PROJ-1-feat") {
		t.Fatal("dry-run pushed")
	}
}

func TestMRRefusesBranchWithoutTicket(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := placeLinked(t, ws, "a")
	onBranch(t, a, "practice-improves")
	allowPushOptions(t, a)

	ctx, out, errb := newCtx(t, mrArgs("practice-improves"), "")
	if rc := commands.MR(ctx); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, errb.String(), "does not look like a ticket branch")
	notContains(t, out.String(), "pushing")
	if remoteHasBranch(t, a, "practice-improves") {
		t.Fatal("push happened for a branch without a ticket")
	}
}
