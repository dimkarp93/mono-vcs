package commands_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

func writeFakeGlab(t *testing.T, script string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "glab")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return dir
}

func fakeGlab(t *testing.T) {
	t.Helper()
	writeFakeGlab(t, "#!/bin/sh\nexit 0\n")
}

func fakeGlabRecording(t *testing.T) string {
	t.Helper()
	log := filepath.Join(t.TempDir(), "glab.log")
	writeFakeGlab(t, fmt.Sprintf("#!/bin/sh\necho \"$@\" >> %s\n", log))
	return log
}

func pathWithoutGlab(t *testing.T) {
	t.Helper()
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Symlink(gitPath, filepath.Join(dir, "git")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
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
	fakeGlab(t)

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
	fakeGlab(t)

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
	contains(t, o, "glab mr create")
	contains(t, o, "--title [PROJ-1] my mr")
	if remoteHasBranch(t, a, "PROJ-1-feat") {
		t.Fatal("dry-run pushed")
	}
}

func TestMRRefusesBranchWithoutTicket(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := placeLinked(t, ws, "a")
	onBranch(t, a, "practice-improves")

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

func TestMRFailsWithoutGlab(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := placeLinked(t, ws, "a")
	onBranch(t, a, "PROJ-1-feat")
	pathWithoutGlab(t)

	ctx, out, errb := newCtx(t, mrArgs("PROJ-1-feat"), "")
	if rc := commands.MR(ctx); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, errb.String(), "glab not found in PATH")
	notContains(t, out.String(), "pushing")
	if remoteHasBranch(t, a, "PROJ-1-feat") {
		t.Fatal("push happened without glab")
	}
}

func TestMRPassesTitleAndRemoveSourceBranchToGlab(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := placeLinked(t, ws, "a")
	onBranch(t, a, "PROJ-1-feat")
	log := fakeGlabRecording(t)

	args := mrArgs("PROJ-1-feat")
	args.Title = "fix login"
	ctx, _, errb := newCtx(t, args, "")
	if rc := commands.MR(ctx); rc != 0 {
		t.Fatalf("rc=%d err=%s", rc, errb.String())
	}

	data, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	call := string(data)
	contains(t, call, "mr create")
	contains(t, call, "--source-branch PROJ-1-feat")
	contains(t, call, "--remove-source-branch")
	contains(t, call, "--yes")
	contains(t, call, "--title [PROJ-1] fix login")
}

func TestMRTitleFromFirstCommitOfBranch(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := placeLinked(t, ws, "a")
	testutil.Run(t, a, "git", "checkout", "-b", "PROJ-1-feat")
	testutil.Commit(t, a, "[PROJ-1] fix problem", "one.txt", "1")
	testutil.Commit(t, a, "second commit", "two.txt", "2")
	log := fakeGlabRecording(t)

	ctx, _, errb := newCtx(t, mrArgs("PROJ-1-feat"), "")
	if rc := commands.MR(ctx); rc != 0 {
		t.Fatalf("rc=%d err=%s", rc, errb.String())
	}

	data, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	call := string(data)
	contains(t, call, "--title [PROJ-1] fix problem")
	contains(t, call, "--fill")
	notContains(t, call, "second commit")
}

func TestMRTitleFromFirstCommitWithoutTicketPrefix(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := placeLinked(t, ws, "a")
	testutil.Run(t, a, "git", "checkout", "-b", "PROJ-1-feat")
	testutil.Commit(t, a, "fix problem", "one.txt", "1")
	log := fakeGlabRecording(t)

	ctx, _, errb := newCtx(t, mrArgs("PROJ-1-feat"), "")
	if rc := commands.MR(ctx); rc != 0 {
		t.Fatalf("rc=%d err=%s", rc, errb.String())
	}

	data, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	contains(t, string(data), "--title [PROJ-1] fix problem")
}

func TestMRTitleDiffersPerRepo(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	a := placeLinked(t, ws, "a")
	b := placeLinked(t, ws, "b")
	testutil.Run(t, a, "git", "checkout", "-b", "PROJ-1-feat")
	testutil.Commit(t, a, "change in a", "one.txt", "1")
	testutil.Run(t, b, "git", "checkout", "-b", "PROJ-1-feat")
	testutil.Commit(t, b, "[PROJ-1] change in b", "one.txt", "1")
	log := fakeGlabRecording(t)

	ctx, _, errb := newCtx(t, mrArgs("PROJ-1-feat"), "")
	if rc := commands.MR(ctx); rc != 0 {
		t.Fatalf("rc=%d err=%s", rc, errb.String())
	}

	data, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	contains(t, string(data), "--title [PROJ-1] change in a")
	contains(t, string(data), "--title [PROJ-1] change in b")
}
