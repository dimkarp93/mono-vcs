package commands_test

import (
	"os"
	"os/exec"
	"path/filepath"
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
	orphan := testutil.MakeRepo(t, ws, "orphan", "trunk")
	testutil.MakeRemote(t, t.TempDir(), orphan, "trunk")

	ctx, out, errb := newCtx(t, updateArgs(), "")
	if rc := commands.Update(ctx); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, errb.String(), "the default branch is unknown")
	contains(t, errb.String(), "1 repo(s) skipped")
	contains(t, out.String(), "good")
}

func TestUpdateSkipsRepoWithoutRemote(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	placeLinked(t, ws, "with")
	testutil.MakeRepo(t, ws, "without", "main")

	ctx, out, errb := newCtx(t, updateArgs(), "")
	if rc := commands.Update(ctx); rc != 0 {
		t.Fatalf("rc=%d err=%s", rc, errb.String())
	}
	contains(t, out.String(), "with")
	notContains(t, out.String(), "without")
	notContains(t, errb.String(), "without")
}

func TestUpdateNoReposWithRemote(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	testutil.MakeRepo(t, ws, "without", "main")

	ctx, out, _ := newCtx(t, updateArgs(), "")
	if rc := commands.Update(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "with a remote")
}

func TestUpdateRemoteFlagFilters(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	placeLinked(t, ws, "p")
	a := updateArgs()
	a.Remote = []string{"github.com"}

	ctx, out, _ := newCtx(t, a, "")
	if rc := commands.Update(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	notContains(t, out.String(), "[1/1]")
}

func pushUpstream(t *testing.T, repo, branch, file, content string) {
	t.Helper()
	remote := strings.TrimSpace(testutil.Run(t, repo, "git", "remote", "get-url", "origin"))
	other := filepath.Join(t.TempDir(), "other")
	testutil.Run(t, filepath.Dir(other), "git", "clone", "-b", branch, remote, other)
	testutil.Run(t, other, "git", "config", "user.email", "test@example.com")
	testutil.Run(t, other, "git", "config", "user.name", "Test")
	testutil.Commit(t, other, "upstream "+file, file, content)
	testutil.Run(t, other, "git", "push", "origin", branch)
}

func fileExists(repo, name string) bool {
	_, err := os.Stat(filepath.Join(repo, name))
	return err == nil
}

func TestUpdateDirtyOnDefaultBranchKeepsChanges(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := placeLinked(t, ws, "p")
	pushUpstream(t, p, "main", "upstream.txt", "up")
	writeFile(p, "local.txt", "mine")

	ctx, _, errb := newCtx(t, updateArgs(), "")
	if rc := commands.Update(ctx); rc != 0 {
		t.Fatalf("rc=%d err=%s", rc, errb.String())
	}
	if !fileExists(p, "upstream.txt") || !fileExists(p, "local.txt") {
		t.Fatal("expected both upstream and local files")
	}
	if ref := gitops.UpdateStashRef(p); ref != "" {
		t.Fatalf("stash left behind: %s", ref)
	}
}

func TestUpdateDirtyFeatureIsRebasedAndRestored(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := placeLinked(t, ws, "p")
	testutil.Run(t, p, "git", "checkout", "-b", "feature")
	testutil.Commit(t, p, "feature work", "feature.txt", "f")
	pushUpstream(t, p, "main", "upstream.txt", "up")
	writeFile(p, "local.txt", "mine")

	ctx, _, errb := newCtx(t, updateArgs(), "")
	if rc := commands.Update(ctx); rc != 0 {
		t.Fatalf("rc=%d err=%s", rc, errb.String())
	}
	if got := gitops.CurrentBranch(p); got != "feature" {
		t.Fatalf("branch=%q", got)
	}
	for _, f := range []string{"upstream.txt", "feature.txt", "local.txt"} {
		if !fileExists(p, f) {
			t.Fatalf("missing %s", f)
		}
	}
	if ref := gitops.UpdateStashRef(p); ref != "" {
		t.Fatalf("stash left behind: %s", ref)
	}
}

func TestUpdateRebaseConflictKeepsStash(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := placeLinked(t, ws, "p")
	testutil.Run(t, p, "git", "checkout", "-b", "feature")
	testutil.Commit(t, p, "feature edit", "shared.txt", "feature")
	pushUpstream(t, p, "main", "shared.txt", "upstream")
	writeFile(p, "local.txt", "mine")

	ctx, _, errb := newCtx(t, updateArgs(), "")
	if rc := commands.Update(ctx); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, errb.String(), "git stash")
	if !gitops.RebaseInProgress(p) {
		t.Fatal("expected rebase in progress")
	}
	if gitops.UpdateStashRef(p) == "" {
		t.Fatal("stash was lost")
	}
}

func TestUpdateStashPopConflictKeepsStash(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := placeLinked(t, ws, "p")
	pushUpstream(t, p, "main", "README", "upstream")
	writeFile(p, "README", "mine")

	ctx, _, errb := newCtx(t, updateArgs(), "")
	if rc := commands.Update(ctx); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, errb.String(), "CONFLICT")
	if gitops.UpdateStashRef(p) == "" {
		t.Fatal("stash was lost")
	}
}

func TestUpdateRefusesDuringRebase(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	p := placeLinked(t, ws, "p")
	testutil.Run(t, p, "git", "checkout", "-b", "feature")
	testutil.Commit(t, p, "feature edit", "shared.txt", "feature")
	testutil.Run(t, p, "git", "checkout", "main")
	testutil.Commit(t, p, "main edit", "shared.txt", "main")
	testutil.Run(t, p, "git", "checkout", "feature")
	cmd := exec.Command("git", "-C", p, "rebase", "main")
	cmd.Run()
	if !gitops.RebaseInProgress(p) {
		t.Fatal("setup: expected rebase in progress")
	}

	ctx, _, errb := newCtx(t, updateArgs(), "")
	if rc := commands.Update(ctx); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, errb.String(), "rebase already in progress")
}
