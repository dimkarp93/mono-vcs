package gitops_test

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/gitops"
	"github.com/dimkarp93/mono-vcs/internal/testutil"
)

func write(t *testing.T, repo, file, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repo, file), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestIsDirtyCleanRepo(t *testing.T) {
	if gitops.IsDirty(testutil.MakeRepo(t, t.TempDir(), "r", "main")) {
		t.Fatal("clean repo reported dirty")
	}
}

func TestIsDirtyUnstaged(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "main")
	write(t, r, "README", "changed")
	if !gitops.IsDirty(r) {
		t.Fatal("expected dirty")
	}
}

func TestIsDirtyStaged(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "main")
	write(t, r, "new", "x")
	testutil.Run(t, r, "git", "add", "new")
	if !gitops.IsDirty(r) {
		t.Fatal("expected dirty")
	}
}

func TestIsDirtyUntracked(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "main")
	write(t, r, "untracked", "x")
	if !gitops.IsDirty(r) {
		t.Fatal("expected dirty")
	}
}

func TestLocalInfoClean(t *testing.T) {
	b, d, h := gitops.LocalInfo(testutil.MakeRepo(t, t.TempDir(), "r", "main"))
	if b != "main" || d || h {
		t.Fatalf("got %q %v %v", b, d, h)
	}
}

func TestLocalInfoUntrackedIsDirtyNotLocal(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "main")
	write(t, r, "untracked", "x")
	_, d, h := gitops.LocalInfo(r)
	if !d || h {
		t.Fatalf("dirty=%v hasLocal=%v", d, h)
	}
}

func TestLocalInfoUnstagedIsLocal(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "main")
	write(t, r, "README", "changed")
	_, d, h := gitops.LocalInfo(r)
	if !d || !h {
		t.Fatalf("dirty=%v hasLocal=%v", d, h)
	}
}

func TestLocalInfoStagedIsLocal(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "main")
	write(t, r, "new", "x")
	testutil.Run(t, r, "git", "add", "new")
	_, d, h := gitops.LocalInfo(r)
	if !d || !h {
		t.Fatalf("dirty=%v hasLocal=%v", d, h)
	}
}

func TestCurrentBranchReturnsName(t *testing.T) {
	if b := gitops.CurrentBranch(testutil.MakeRepo(t, t.TempDir(), "r", "main")); b != "main" {
		t.Fatalf("got %q", b)
	}
}

func TestCurrentBranchDetachedReturnsEmpty(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "main")
	sha := testutil.Run(t, r, "git", "rev-parse", "HEAD")
	testutil.Run(t, r, "git", "checkout", "--detach", sha[:len(sha)-1])
	if b := gitops.CurrentBranch(r); b != "" {
		t.Fatalf("got %q", b)
	}
}

func TestHasBranchLocal(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "main")
	testutil.Run(t, r, "git", "branch", "feature")
	if !gitops.HasBranch(r, "feature") {
		t.Fatal("expected feature")
	}
}

func TestHasBranchAbsent(t *testing.T) {
	if gitops.HasBranch(testutil.MakeRepo(t, t.TempDir(), "r", "main"), "nope") {
		t.Fatal("did not expect branch")
	}
}

func TestHasBranchViaOrigin(t *testing.T) {
	dir := t.TempDir()
	local, _ := testutil.LinkedRepo(t, dir)
	testutil.Run(t, local, "git", "checkout", "-b", "feature")
	write(t, local, "f.txt", "x")
	testutil.Run(t, local, "git", "add", "f.txt")
	testutil.Run(t, local, "git", "commit", "-m", "f")
	testutil.Run(t, local, "git", "push", "-u", "origin", "feature")
	testutil.Run(t, local, "git", "checkout", "main")
	testutil.Run(t, local, "git", "branch", "-D", "feature")
	if !gitops.HasBranch(local, "feature") {
		t.Fatal("expected origin/feature")
	}
}

func TestRebaseInProgressFalse(t *testing.T) {
	if gitops.RebaseInProgress(testutil.MakeRepo(t, t.TempDir(), "r", "main")) {
		t.Fatal("expected false")
	}
}

func makeConflict(t *testing.T) string {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "main")
	write(t, r, "f", "a\n")
	testutil.Run(t, r, "git", "add", "f")
	testutil.Run(t, r, "git", "commit", "-m", "base")
	testutil.Run(t, r, "git", "checkout", "-b", "feature")
	write(t, r, "f", "feature\n")
	testutil.Run(t, r, "git", "commit", "-am", "feature")
	testutil.Run(t, r, "git", "checkout", "main")
	write(t, r, "f", "main\n")
	testutil.Run(t, r, "git", "commit", "-am", "main-change")
	testutil.Run(t, r, "git", "checkout", "feature")
	return r
}

func TestRebaseInProgressTrueOnConflict(t *testing.T) {
	r := makeConflict(t)
	exec := func(args ...string) { _ = osRun(r, args...) }
	exec("git", "rebase", "main")
	if !gitops.RebaseInProgress(r) {
		t.Fatal("expected rebase in progress")
	}
	testutil.Run(t, r, "git", "rebase", "--abort")
	if gitops.RebaseInProgress(r) {
		t.Fatal("expected no rebase after abort")
	}
}

func TestLocalBranchSHA(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "main")
	sha := testutil.Run(t, r, "git", "rev-parse", "HEAD")
	if got := gitops.LocalBranchSHA(r, "main"); got != sha[:len(sha)-1] {
		t.Fatalf("got %q", got)
	}
}

func TestLocalBranchSHAMissing(t *testing.T) {
	if got := gitops.LocalBranchSHA(testutil.MakeRepo(t, t.TempDir(), "r", "main"), "ghost"); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestStashCountZero(t *testing.T) {
	if n := gitops.StashCount(testutil.MakeRepo(t, t.TempDir(), "r", "main")); n != 0 {
		t.Fatalf("got %d", n)
	}
}

func TestStashCountAfterPushes(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "main")
	write(t, r, "README", "a")
	testutil.Run(t, r, "git", "stash", "push", "-m", "one")
	if n := gitops.StashCount(r); n != 1 {
		t.Fatalf("got %d", n)
	}
	write(t, r, "README", "b")
	testutil.Run(t, r, "git", "stash", "push", "-m", "two")
	if n := gitops.StashCount(r); n != 2 {
		t.Fatalf("got %d", n)
	}
}

func TestExtraHeaderArgsEmpty(t *testing.T) {
	if len(gitops.GitExtraHeaderArgs("")) != 0 {
		t.Fatal("expected empty for no token")
	}
}

func TestExtraHeaderArgsWithToken(t *testing.T) {
	out := gitops.GitExtraHeaderArgs("T")
	if len(out) != 2 || out[0] != "-c" || out[1] != "http.extraHeader=PRIVATE-TOKEN: T" {
		t.Fatalf("got %v", out)
	}
}

func TestCloneOneClonesFresh(t *testing.T) {
	dir := t.TempDir()
	_, remote := testutil.LinkedRepo(t, dir)
	target := filepath.Join(dir, "dest")
	_, ok, errMsg := gitops.CloneOne(remote, target, "", false)
	if !ok || errMsg != "" {
		t.Fatalf("ok=%v err=%q", ok, errMsg)
	}
	if _, err := os.Stat(filepath.Join(target, ".git")); err != nil {
		t.Fatal(".git missing")
	}
}

func TestCloneOneForceRemovesNonRepo(t *testing.T) {
	dir := t.TempDir()
	_, remote := testutil.LinkedRepo(t, dir)
	target := filepath.Join(dir, "dest")
	os.MkdirAll(target, 0o755)
	write(t, target, "junk", "x")
	_, ok, errMsg := gitops.CloneOne(remote, target, "", true)
	if !ok {
		t.Fatalf("err=%q", errMsg)
	}
	if _, err := os.Stat(filepath.Join(target, ".git")); err != nil {
		t.Fatal(".git missing")
	}
	if _, err := os.Stat(filepath.Join(target, "junk")); err == nil {
		t.Fatal("junk should be gone")
	}
}

func advancedRemote(t *testing.T) (local, dir string) {
	t.Helper()
	dir = t.TempDir()
	local, remote := testutil.LinkedRepo(t, dir)
	side := filepath.Join(dir, "side")
	testutil.Run(t, dir, "git", "clone", remote, side)
	testutil.Commit(t, side, "advance", "extra", "y")
	testutil.Run(t, side, "git", "push", "origin", "main")
	return local, dir
}

func branchSHA(t *testing.T, repo, ref string) string {
	t.Helper()
	return strings.TrimSpace(testutil.Run(t, repo, "git", "rev-parse", ref))
}

func TestPullDefaultOneUpToDate(t *testing.T) {
	local, _ := testutil.LinkedRepo(t, t.TempDir())
	res := gitops.PullDefaultOne(local, "", "main")
	if res.Path != local || res.Status != "up-to-date" {
		t.Fatalf("got %+v", res)
	}
}

func TestPullDefaultOneOnDefaultBranchUpdatesWorktree(t *testing.T) {
	local, _ := advancedRemote(t)
	res := gitops.PullDefaultOne(local, "", "main")
	if res.Status != "updated" {
		t.Fatalf("got %+v", res)
	}
	if _, err := os.Stat(filepath.Join(local, "extra")); err != nil {
		t.Fatal("expected extra file pulled")
	}
}

func TestPullDefaultOneFromFeatureBranchLeavesWorktreeAlone(t *testing.T) {
	local, _ := advancedRemote(t)
	testutil.Run(t, local, "git", "checkout", "-b", "feature")
	write(t, local, "README", "dirty")

	res := gitops.PullDefaultOne(local, "", "main")
	if res.Status != "updated" {
		t.Fatalf("got %+v", res)
	}
	if b := gitops.CurrentBranch(local); b != "feature" {
		t.Fatalf("branch=%q", b)
	}
	if !gitops.IsDirty(local) {
		t.Fatal("uncommitted changes must survive")
	}
	if _, err := os.Stat(filepath.Join(local, "extra")); err == nil {
		t.Fatal("the worktree must not be touched")
	}
	if branchSHA(t, local, "refs/heads/main") != branchSHA(t, local, "refs/remotes/origin/main") {
		t.Fatal("main and origin/main must match after the fetch")
	}
}

func TestPullDefaultOneDivergedIsReported(t *testing.T) {
	dir := t.TempDir()
	local, remote := testutil.LinkedRepo(t, dir)
	side := filepath.Join(dir, "side")
	testutil.Run(t, dir, "git", "clone", remote, side)
	testutil.Commit(t, side, "side", "x", "side")
	testutil.Run(t, side, "git", "push", "origin", "main")
	testutil.Commit(t, local, "local diverge", "y", "local")
	testutil.Run(t, local, "git", "checkout", "-b", "feature")
	before := branchSHA(t, local, "refs/heads/main")

	res := gitops.PullDefaultOne(local, "", "main")
	if res.Status != "diverged" {
		t.Fatalf("got %+v", res)
	}
	if branchSHA(t, local, "refs/heads/main") != before {
		t.Fatal("main must not move when it has diverged")
	}
}

func TestPullDefaultOneDirtyOnDefaultBranch(t *testing.T) {
	local, _ := advancedRemote(t)
	write(t, local, "extra", "mine")

	res := gitops.PullDefaultOne(local, "", "main")
	if res.Status != "dirty" {
		t.Fatalf("got %+v", res)
	}
}

func TestUpdateMainDirtySkipped(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "main")
	write(t, r, "README", "dirty")
	res, orig := gitops.UpdateMainOne(r, "", "main")
	if res.Status != "dirty" || orig != "" {
		t.Fatalf("got %+v orig=%q", res, orig)
	}
}

func TestUpdateMainAlreadyOnMain(t *testing.T) {
	local, _ := testutil.LinkedRepo(t, t.TempDir())
	res, orig := gitops.UpdateMainOne(local, "", "main")
	if (res.Status != "up-to-date" && res.Status != "updated") || orig != "" {
		t.Fatalf("got %+v orig=%q", res, orig)
	}
}

func TestUpdateMainFromFeatureReturnsOrig(t *testing.T) {
	local, _ := testutil.LinkedRepo(t, t.TempDir())
	testutil.Run(t, local, "git", "checkout", "-b", "feature")
	res, orig := gitops.UpdateMainOne(local, "", "main")
	if (res.Status != "up-to-date" && res.Status != "updated") || orig != "feature" {
		t.Fatalf("got %+v orig=%q", res, orig)
	}
}

func TestRebaseOneSuccess(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "main")
	testutil.Commit(t, r, "base", "base", "base")
	testutil.Run(t, r, "git", "checkout", "-b", "feature")
	testutil.Commit(t, r, "feat", "feat", "feat")
	testutil.Run(t, r, "git", "checkout", "main")
	testutil.Commit(t, r, "main2", "main2", "main")
	status, _ := gitops.RebaseOne(r, "feature", "main", nil, io.Discard, io.Discard)
	if status != "rebased" {
		t.Fatalf("status=%q", status)
	}
}

func TestRebaseOneConflictLeftInProgress(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "main")
	write(t, r, "f", "a\n")
	testutil.Run(t, r, "git", "add", "f")
	testutil.Run(t, r, "git", "commit", "-m", "base")
	testutil.Run(t, r, "git", "checkout", "-b", "feature")
	write(t, r, "f", "feature\n")
	testutil.Run(t, r, "git", "commit", "-am", "feature")
	testutil.Run(t, r, "git", "checkout", "main")
	write(t, r, "f", "main\n")
	testutil.Run(t, r, "git", "commit", "-am", "main-change")
	status, _ := gitops.RebaseOne(r, "feature", "main", nil, io.Discard, io.Discard)
	if status != "conflict" {
		t.Fatalf("status=%q", status)
	}
	if !gitops.RebaseInProgress(r) {
		t.Fatal("expected rebase in progress")
	}
	testutil.Run(t, r, "git", "rebase", "--abort")
}

func TestSwitchDirty(t *testing.T) {
	local, _ := testutil.LinkedRepo(t, t.TempDir())
	write(t, local, "README", "dirty")
	if res := gitops.SwitchOne(local, "anything", "main", ""); res.Status != "dirty" {
		t.Fatalf("status=%q", res.Status)
	}
}

func TestSwitchAlready(t *testing.T) {
	local, _ := testutil.LinkedRepo(t, t.TempDir())
	if res := gitops.SwitchOne(local, "main", "main", ""); res.Status != "already" {
		t.Fatalf("status=%q", res.Status)
	}
}

func TestSwitchLocalBranch(t *testing.T) {
	local, _ := testutil.LinkedRepo(t, t.TempDir())
	testutil.Run(t, local, "git", "branch", "feature")
	if res := gitops.SwitchOne(local, "feature", "main", ""); res.Status != "switched" {
		t.Fatalf("status=%q", res.Status)
	}
}

func TestSwitchRemoteOnlyBranch(t *testing.T) {
	dir := t.TempDir()
	local, remote := testutil.LinkedRepo(t, dir)
	side := filepath.Join(dir, "side")
	testutil.Run(t, dir, "git", "clone", remote, side)
	testutil.Run(t, side, "git", "checkout", "-b", "feature")
	testutil.Commit(t, side, "f", "f", "f")
	testutil.Run(t, side, "git", "push", "-u", "origin", "feature")
	testutil.Run(t, local, "git", "fetch")
	if res := gitops.SwitchOne(local, "feature", "main", ""); res.Status != "switched" {
		t.Fatalf("status=%q", res.Status)
	}
}

func TestSwitchFallbackToMain(t *testing.T) {
	local, _ := testutil.LinkedRepo(t, t.TempDir())
	res := gitops.SwitchOne(local, "no-such", "main", "")
	if res.Status != "synced-main" || !contains(res.Detail, "main") {
		t.Fatalf("got %+v", res)
	}
}

func TestSwitchAbsentWhenNoBranchNoMain(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "trunk")
	if res := gitops.SwitchOne(r, "no-such", "main", ""); res.Status != "absent" {
		t.Fatalf("status=%q", res.Status)
	}
}

func TestFinishAbsent(t *testing.T) {
	local, _ := testutil.LinkedRepo(t, t.TempDir())
	if res := gitops.FinishOne(local, "ghost", "main", ""); res.Status != "absent" {
		t.Fatalf("status=%q", res.Status)
	}
}

func TestFinishDeletesNonCurrent(t *testing.T) {
	local, _ := testutil.LinkedRepo(t, t.TempDir())
	testutil.Run(t, local, "git", "branch", "feature")
	res := gitops.FinishOne(local, "feature", "main", "")
	if res.Status != "deleted" || !contains(res.Detail, "feature") {
		t.Fatalf("got %+v", res)
	}
	if gitops.HasBranch(local, "feature") {
		t.Fatal("feature should be gone")
	}
}

func TestFinishCurrentClean(t *testing.T) {
	local, _ := testutil.LinkedRepo(t, t.TempDir())
	testutil.Run(t, local, "git", "checkout", "-b", "feature")
	res := gitops.FinishOne(local, "feature", "main", "")
	if res.Status != "deleted" || gitops.CurrentBranch(local) != "main" || !contains(res.Detail, "feature") {
		t.Fatalf("got %+v branch=%q", res, gitops.CurrentBranch(local))
	}
}

func TestFinishCurrentDirtyDiscardsAndDeletes(t *testing.T) {
	local, _ := testutil.LinkedRepo(t, t.TempDir())
	testutil.Run(t, local, "git", "checkout", "-b", "feature")
	write(t, local, "README", "dirty")
	write(t, local, "untracked.txt", "junk")
	res := gitops.FinishOne(local, "feature", "main", "")
	if res.Status != "deleted" || !contains(res.Detail, "discarded 2 items") {
		t.Fatalf("got %+v", res)
	}
	if gitops.CurrentBranch(local) != "main" || gitops.IsDirty(local) {
		t.Fatalf("branch=%q dirty=%v", gitops.CurrentBranch(local), gitops.IsDirty(local))
	}
	if gitops.HasBranch(local, "feature") {
		t.Fatal("feature should be gone")
	}
}

func TestFinishDirtyOnOtherBranchStillCleans(t *testing.T) {
	local, _ := testutil.LinkedRepo(t, t.TempDir())
	testutil.Run(t, local, "git", "branch", "feature")
	write(t, local, "README", "dirty")
	if res := gitops.FinishOne(local, "feature", "main", ""); res.Status != "deleted" {
		t.Fatalf("status=%q", res.Status)
	}
	if gitops.IsDirty(local) {
		t.Fatal("expected a clean tree")
	}
}

func TestFinishFailsWithoutDefaultBranch(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "trunk")
	testutil.Run(t, r, "git", "branch", "feature")
	if res := gitops.FinishOne(r, "feature", "main", ""); res.Status != "failed" {
		t.Fatalf("status=%q", res.Status)
	}
}

func TestMergedIntoAncestor(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "main")
	testutil.Run(t, r, "git", "branch", "feature")
	if !gitops.MergedInto(r, "feature", "refs/heads/main") {
		t.Fatal("a branch at the tip of main must count as merged")
	}
}

func TestMergedIntoRejectsUniqueCommits(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "main")
	testutil.Run(t, r, "git", "checkout", "-b", "feature")
	testutil.Commit(t, r, "work", "f", "f")
	if gitops.MergedInto(r, "feature", "refs/heads/main") {
		t.Fatal("a branch with its own commits must not count as merged")
	}
}

func TestMergedIntoAfterFastForward(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "main")
	testutil.Run(t, r, "git", "checkout", "-b", "feature")
	testutil.Commit(t, r, "work", "f", "f")
	testutil.Run(t, r, "git", "checkout", "main")
	testutil.Run(t, r, "git", "merge", "--ff-only", "feature")
	if !gitops.MergedInto(r, "feature", "refs/heads/main") {
		t.Fatal("expected merged after fast-forward")
	}
}

func TestDefaultCompareRefPrefersOrigin(t *testing.T) {
	local, _ := testutil.LinkedRepo(t, t.TempDir())
	if got := gitops.DefaultCompareRef(local, "main"); got != "refs/remotes/origin/main" {
		t.Fatalf("got %q", got)
	}
	r := testutil.MakeRepo(t, t.TempDir(), "r", "main")
	if got := gitops.DefaultCompareRef(r, "main"); got != "refs/heads/main" {
		t.Fatalf("got %q", got)
	}
	if got := gitops.DefaultCompareRef(r, "nope"); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestDeleteMergedOneLeavesDefaultCheckedOut(t *testing.T) {
	local, _ := testutil.LinkedRepo(t, t.TempDir())
	testutil.Run(t, local, "git", "checkout", "-b", "feature")
	if res := gitops.DeleteMergedOne(local, "feature", "main", ""); res.Status != "deleted" {
		t.Fatalf("got %+v", res)
	}
	if gitops.CurrentBranch(local) != "main" || gitops.HasBranch(local, "feature") {
		t.Fatalf("branch=%q", gitops.CurrentBranch(local))
	}
}

func TestStashOneNothing(t *testing.T) {
	if res := gitops.StashOne(testutil.MakeRepo(t, t.TempDir(), "r", "main")); res.Status != "nothing" {
		t.Fatalf("status=%q", res.Status)
	}
}

func TestStashOneStashesUnstaged(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "main")
	write(t, r, "README", "changed")
	if res := gitops.StashOne(r); res.Status != "stashed" {
		t.Fatalf("status=%q", res.Status)
	}
	if gitops.StashCount(r) != 1 {
		t.Fatal("expected 1 stash")
	}
}

func TestUnstashOneNothing(t *testing.T) {
	if res := gitops.UnstashOne(testutil.MakeRepo(t, t.TempDir(), "r", "main")); res.Status != "nothing" {
		t.Fatalf("status=%q", res.Status)
	}
}

func TestUnstashOnePops(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "main")
	write(t, r, "README", "changed")
	gitops.StashOne(r)
	if res := gitops.UnstashOne(r); res.Status != "popped" {
		t.Fatalf("status=%q", res.Status)
	}
	if gitops.StashCount(r) != 0 {
		t.Fatal("expected 0 stash")
	}
}

func TestClearStashOneNothing(t *testing.T) {
	if res := gitops.ClearStashOne(testutil.MakeRepo(t, t.TempDir(), "r", "main")); res.Status != "nothing" {
		t.Fatalf("status=%q", res.Status)
	}
}

func TestClearStashOneDropsAll(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "main")
	write(t, r, "README", "a")
	gitops.StashOne(r)
	write(t, r, "README", "b")
	gitops.StashOne(r)
	if gitops.StashCount(r) != 2 {
		t.Fatal("expected 2")
	}
	res := gitops.ClearStashOne(r)
	if res.Status != "cleared" || !contains(res.Detail, "2") {
		t.Fatalf("got %+v", res)
	}
	if gitops.StashCount(r) != 0 {
		t.Fatal("expected 0")
	}
}

func TestPruneTargetsListsUntracked(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "main")
	write(t, r, "a.txt", "1")
	targets, err := gitops.PruneTargets(r)
	if err != nil || len(targets) != 1 || targets[0] != "?? a.txt" {
		t.Fatalf("targets=%v err=%v", targets, err)
	}
}

func TestPruneTargetsListsTrackedChange(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "main")
	write(t, r, "README", "changed")
	targets, err := gitops.PruneTargets(r)
	if err != nil || len(targets) != 1 || targets[0] != "M README" {
		t.Fatalf("targets=%v err=%v", targets, err)
	}
}

func TestRemotesListsEveryRemoteOnce(t *testing.T) {
	ws := t.TempDir()
	local := testutil.MakeClonedRepo(t, ws, "grp/api", "main")
	testutil.Run(t, local, "git", "remote", "add", "upstream", "git@github.com:grp/api.git")

	rs, err := gitops.Remotes(local)
	if err != nil {
		t.Fatal(err)
	}
	if len(rs) != 2 {
		t.Fatalf("got %+v", rs)
	}
	byName := map[string]string{}
	for _, r := range rs {
		byName[r.Name] = r.URL
	}
	if byName["upstream"] != "git@github.com:grp/api.git" {
		t.Fatalf("got %+v", byName)
	}
	if byName["origin"] == "" {
		t.Fatalf("origin missing: %+v", byName)
	}
}

func TestRemotesOnRepoWithoutRemotesIsEmpty(t *testing.T) {
	repo := testutil.MakeRepo(t, t.TempDir(), "solo", "main")
	rs, err := gitops.Remotes(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(rs) != 0 {
		t.Fatalf("got %+v", rs)
	}
}
