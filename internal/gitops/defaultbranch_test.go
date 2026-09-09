package gitops

import (
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/testutil"
)

func TestDefaultBranchFromOriginHead(t *testing.T) {
	dir := t.TempDir()
	local := testutil.MakeRepo(t, dir, "a", "master")
	testutil.MakeRemote(t, dir, local, "master")
	testutil.Run(t, local, "git", "remote", "set-head", "origin", "master")

	b, detected := DefaultBranch(local, "main")
	if b != "master" || !detected {
		t.Fatalf("got %q detected=%v", b, detected)
	}
}

func TestDefaultBranchFromRemoteRef(t *testing.T) {
	dir := t.TempDir()
	local := testutil.MakeRepo(t, dir, "b", "master")
	testutil.MakeRemote(t, dir, local, "master")

	b, detected := DefaultBranch(local, "main")
	if b != "master" || !detected {
		t.Fatalf("got %q detected=%v", b, detected)
	}
}

func TestDefaultBranchFromLocalHead(t *testing.T) {
	dir := t.TempDir()
	local := testutil.MakeRepo(t, dir, "c", "master")

	b, detected := DefaultBranch(local, "main")
	if b != "master" || !detected {
		t.Fatalf("got %q detected=%v", b, detected)
	}
}

func TestDefaultBranchFallsBackToConfigured(t *testing.T) {
	dir := t.TempDir()
	local := testutil.MakeRepo(t, dir, "d", "trunk")

	b, detected := DefaultBranch(local, "master")
	if b != "master" || detected {
		t.Fatalf("got %q detected=%v", b, detected)
	}
}

func TestDefaultBranchFallsBackToMain(t *testing.T) {
	dir := t.TempDir()
	local := testutil.MakeRepo(t, dir, "e", "trunk")

	b, detected := DefaultBranch(local, "")
	if b != FallbackDefaultBranch || detected {
		t.Fatalf("got %q detected=%v", b, detected)
	}
}
