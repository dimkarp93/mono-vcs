package gitops_test

import (
	"testing"

	"mono-vcs/internal/gitops"
	"mono-vcs/internal/testutil"
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

func TestNewBranchExists(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "main")
	testutil.Run(t, r, "git", "branch", "feature")
	if res := gitops.NewBranchOne(r, "feature", "main"); res.Status != "exists" {
		t.Fatalf("status=%q", res.Status)
	}
}

func TestNewBranchDirty(t *testing.T) {
	r := testutil.MakeRepo(t, t.TempDir(), "r", "main")
	write(t, r, "README", "dirty")
	if res := gitops.NewBranchOne(r, "feature", "main"); res.Status != "dirty" {
		t.Fatalf("status=%q", res.Status)
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
