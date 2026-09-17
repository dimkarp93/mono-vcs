package gitops_test

import (
	"strings"
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/gitops"
	"github.com/dimkarp93/mono-vcs/internal/testutil"
)

func TestNewBranchCreatesFromMain(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "main")
	res := gitops.NewBranchOne(r, "feature", "main")
	if res.Status != "created" {
		t.Fatalf("status=%q", res.Status)
	}
	if gitops.CurrentBranch(r) != "feature" {
		t.Fatalf("branch=%q", gitops.CurrentBranch(r))
	}
}

func TestNewBranchSwitchesToExisting(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "main")
	testutil.Run(t, r, "git", "checkout", "-b", "feature")
	testutil.Commit(t, r, "work", "f", "f")
	featSHA := testutil.Run(t, r, "git", "rev-parse", "HEAD")
	testutil.Run(t, r, "git", "checkout", "main")
	if res := gitops.NewBranchOne(r, "feature", "main"); res.Status != "switched" {
		t.Fatalf("status=%q", res.Status)
	}
	if got := testutil.Run(t, r, "git", "rev-parse", "HEAD"); got != featSHA {
		t.Fatalf("existing history was rewritten: %q != %q", got, featSHA)
	}
}

func TestNewBranchAlreadyOnTarget(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "main")
	testutil.Run(t, r, "git", "checkout", "-b", "feature")
	write(t, r, "README", "dirty")
	if res := gitops.NewBranchOne(r, "feature", "main"); res.Status != "already" {
		t.Fatalf("status=%q", res.Status)
	}
	if !gitops.IsDirty(r) {
		t.Fatal("local changes must be left alone")
	}
}

func TestNewBranchCarriesLocalChangesOver(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "main")
	write(t, r, "README", "dirty")
	write(t, r, "untracked.txt", "junk")
	res := gitops.NewBranchOne(r, "feature", "main")
	if res.Status != "created" {
		t.Fatalf("status=%q", res.Status)
	}
	if gitops.CurrentBranch(r) != "feature" {
		t.Fatalf("branch=%q", gitops.CurrentBranch(r))
	}
	status := testutil.Run(t, r, "git", "status", "--porcelain")
	if !strings.Contains(status, "README") || !strings.Contains(status, "untracked.txt") {
		t.Fatalf("local changes did not follow: %q", status)
	}
	if gitops.StashCount(r) != 0 {
		t.Fatal("stash must be empty after a successful pop")
	}
}

func TestNewBranchCarriesChangesOntoExistingBranch(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "main")
	testutil.Run(t, r, "git", "branch", "feature")
	write(t, r, "untracked.txt", "junk")
	if res := gitops.NewBranchOne(r, "feature", "main"); res.Status != "switched" {
		t.Fatalf("status=%q", res.Status)
	}
	if !strings.Contains(testutil.Run(t, r, "git", "status", "--porcelain"), "untracked.txt") {
		t.Fatal("local changes did not follow")
	}
}

func TestNewBranchAbsentNoMain(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "trunk")
	if res := gitops.NewBranchOne(r, "feature", "main"); res.Status != "absent" {
		t.Fatalf("status=%q", res.Status)
	}
}

func TestNewBranchBasesOnMainNotCurrent(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "main")
	mainSHA := testutil.Run(t, r, "git", "rev-parse", "HEAD")
	testutil.Run(t, r, "git", "checkout", "-b", "other")
	testutil.Commit(t, r, "advance other", "x", "y")
	res := gitops.NewBranchOne(r, "feature", "main")
	if res.Status != "created" {
		t.Fatalf("status=%q", res.Status)
	}
	featSHA := testutil.Run(t, r, "git", "rev-parse", "HEAD")
	if featSHA != mainSHA {
		t.Fatalf("expected feature at main tip %q, got %q", mainSHA, featSHA)
	}
}
