package aliases_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/aliases"
)

func TestRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "aliases.json")
	st, err := aliases.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Set(aliases.KindRemote, "work", "gitlab.company.com") {
		t.Fatal("expected a change")
	}
	if st.Set(aliases.KindRemote, "work", "gitlab.company.com") {
		t.Fatal("expected no change on the same value")
	}
	if err := st.Flush(); err != nil {
		t.Fatal(err)
	}

	again, err := aliases.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	v, ok := again.Get(aliases.KindRemote, "work")
	if !ok || v != "gitlab.company.com" {
		t.Fatalf("value=%q ok=%v", v, ok)
	}
	raw, _ := os.ReadFile(path)
	if !strings.Contains(string(raw), `"remotes"`) {
		t.Fatalf("expected a remotes section:\n%s", raw)
	}
}

func TestDeleteRemovesEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aliases.json")
	st, _ := aliases.Open(path)
	st.Set(aliases.KindRemote, "work", "gitlab.company.com")
	if err := st.Flush(); err != nil {
		t.Fatal(err)
	}
	if !st.Delete(aliases.KindRemote, "work") {
		t.Fatal("expected a delete")
	}
	if st.Delete(aliases.KindRemote, "work") {
		t.Fatal("expected no second delete")
	}
	if err := st.Flush(); err != nil {
		t.Fatal(err)
	}
	again, _ := aliases.Open(path)
	if _, ok := again.Get(aliases.KindRemote, "work"); ok {
		t.Fatal("expected the alias to be gone")
	}
}

func TestOpenMissingFileIsEmpty(t *testing.T) {
	st, err := aliases.Open(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Names(aliases.KindRemote)) != 0 {
		t.Fatal("expected an empty store")
	}
}

func TestOpenBrokenFileReportsErrorAndStaysUsable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aliases.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := aliases.Open(path)
	if err == nil {
		t.Fatal("expected a parse error")
	}
	if len(st.Names(aliases.KindRemote)) != 0 {
		t.Fatal("expected an empty store")
	}
}

func TestUnknownKindIsIgnoredOnRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aliases.json")
	if err := os.WriteFile(path, []byte(`{"remotes":{"a":"h"},"branches":{"b":"c"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := aliases.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := st.Get(aliases.KindRemote, "a"); !ok || v != "h" {
		t.Fatalf("value=%q ok=%v", v, ok)
	}
}

func TestFlushWithoutChangesWritesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aliases.json")
	st, _ := aliases.Open(path)
	if err := st.Flush(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("err=%v", err)
	}
}

func TestValidName(t *testing.T) {
	for _, bad := range []string{"", "a/b", "a:b", "a b", "a,b"} {
		if err := aliases.ValidName(bad); err == nil {
			t.Fatalf("expected %q to be rejected", bad)
		}
	}
	if err := aliases.ValidName("work-gl"); err != nil {
		t.Fatal(err)
	}
}

func TestKnownKind(t *testing.T) {
	if !aliases.KnownKind(aliases.KindRemote) {
		t.Fatal("remote must be known")
	}
	if aliases.KnownKind("branch") {
		t.Fatal("branch must not be known yet")
	}
}
