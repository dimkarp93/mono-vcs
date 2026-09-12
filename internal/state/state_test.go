package state_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/state"
)

func TestRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "state.json")
	st, err := state.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if !st.SetDefaultBranch("repo", "main") {
		t.Fatal("expected a change")
	}
	if st.SetDefaultBranch("repo", "main") {
		t.Fatal("expected no change on the same value")
	}
	if err := st.Flush(); err != nil {
		t.Fatal(err)
	}

	again, err := state.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	b, ok := again.DefaultBranch("repo")
	if !ok || b != "main" {
		t.Fatalf("branch=%q ok=%v", b, ok)
	}
}

func TestKeysAreAbsolute(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	path := filepath.Join(t.TempDir(), "state.json")
	st, _ := state.Open(path)
	st.SetDefaultBranch("alpha/p", "master")
	if err := st.Flush(); err != nil {
		t.Fatal(err)
	}
	again, _ := state.Open(path)
	if b, ok := again.DefaultBranch(filepath.Join(ws, "alpha", "p")); !ok || b != "master" {
		t.Fatalf("branch=%q ok=%v", b, ok)
	}
}

func TestOpenMissingFileIsEmpty(t *testing.T) {
	st, err := state.Open(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := st.DefaultBranch("repo"); ok {
		t.Fatal("expected an empty store")
	}
}

func TestOpenBrokenFileReportsErrorAndStaysUsable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := state.Open(path)
	if err == nil {
		t.Fatal("expected a parse error")
	}
	if _, ok := st.DefaultBranch("repo"); ok {
		t.Fatal("expected an empty store")
	}
}

func TestFlushWithoutChangesWritesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	st, _ := state.Open(path)
	if err := st.Flush(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("err=%v", err)
	}
}

func TestEmptyBranchIsIgnored(t *testing.T) {
	st, _ := state.Open(filepath.Join(t.TempDir(), "state.json"))
	if st.SetDefaultBranch("repo", "") {
		t.Fatal("empty branch must not be stored")
	}
	if _, ok := st.DefaultBranch("repo"); ok {
		t.Fatal("expected an empty store")
	}
}
