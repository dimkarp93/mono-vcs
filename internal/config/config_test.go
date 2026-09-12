package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/state"
	"github.com/dimkarp93/mono-vcs/internal/testutil"
)

func pointTo(t *testing.T, path string) {
	SetPath(path)
	t.Cleanup(func() { SetPath("") })
}

func TestLoadMissingFile(t *testing.T) {
	pointTo(t, filepath.Join(t.TempDir(), "vcs"))
	c, err := Load()
	if err != nil || c != (Config{}) {
		t.Fatalf("got %+v, %v", c, err)
	}
}

func TestLoadIgnoresRetiredMainBranchKey(t *testing.T) {
	p := filepath.Join(t.TempDir(), "vcs")
	os.WriteFile(p, []byte(`{"gl-url":"https://example","jobs":4,"main-branch":"master"}`), 0o644)
	pointTo(t, p)
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	want := Config{GLURL: "https://example", Jobs: 4}
	if c != want {
		t.Fatalf("got %+v want %+v", c, want)
	}
}

func TestLoadMalformedErrors(t *testing.T) {
	p := filepath.Join(t.TempDir(), "vcs")
	os.WriteFile(p, []byte("# not json\n[broken"), 0o644)
	pointTo(t, p)
	if _, err := Load(); err == nil {
		t.Fatal("expected error on malformed config")
	}
}

func TestSaveCreatesParentAndSkipsEmpty(t *testing.T) {
	p := filepath.Join(t.TempDir(), "newdir", "vcs")
	pointTo(t, p)
	if err := Save(Config{GLURL: "https://example", Jobs: 4}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(p)
	text := string(data)
	for _, want := range []string{`"gl-url": "https://example"`, `"jobs": 4`} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in:\n%s", want, text)
		}
	}
	if strings.Contains(strings.ToLower(text), "token") {
		t.Fatalf("token must never be written:\n%s", text)
	}
}

func TestSaveThenLoadRoundtrip(t *testing.T) {
	pointTo(t, filepath.Join(t.TempDir(), "vcs"))
	in := Config{GLURL: "https://x", Jobs: 7}
	if err := Save(in); err != nil {
		t.Fatal(err)
	}
	out, err := Load()
	if err != nil || out != in {
		t.Fatalf("got %+v, %v", out, err)
	}
}

func TestApplyDefaultsCLIWins(t *testing.T) {
	p := filepath.Join(t.TempDir(), "vcs")
	os.WriteFile(p, []byte(`{"gl-url":"https://from-config","jobs":9,"main-branch":"master"}`), 0o644)
	pointTo(t, p)
	a := &app.Args{GLURL: testutil.S("https://from-cli"), Jobs: testutil.I(2)}
	if err := ApplyDefaults(a); err != nil {
		t.Fatal(err)
	}
	if a.GetGLURL() != "https://from-cli" || a.GetJobs() != 2 {
		t.Fatalf("CLI values should win: %+v", a)
	}
}

func TestApplyDefaultsFillsNils(t *testing.T) {
	p := filepath.Join(t.TempDir(), "vcs")
	os.WriteFile(p, []byte(`{"gl-url":"https://from-config","jobs":9,"main-branch":"master"}`), 0o644)
	pointTo(t, p)
	a := &app.Args{}
	if err := ApplyDefaults(a); err != nil {
		t.Fatal(err)
	}
	if a.GetGLURL() != "https://from-config" || a.GetJobs() != 9 {
		t.Fatalf("got %+v", a)
	}
}

func TestApplyDefaultsJobsFallback(t *testing.T) {
	p := filepath.Join(t.TempDir(), "vcs")
	os.WriteFile(p, []byte(`{"gl-url":"https://x"}`), 0o644)
	pointTo(t, p)
	a := &app.Args{}
	ApplyDefaults(a)
	if a.GetJobs() != 1 {
		t.Fatalf("got %d", a.GetJobs())
	}
}

func TestDBPathDefaultsUnderLocal(t *testing.T) {
	dir := t.TempDir()
	pointTo(t, filepath.Join(dir, "vcs"))
	a := &app.Args{}
	if err := ApplyDefaults(a); err != nil {
		t.Fatal(err)
	}
	if a.GetDBPath() != state.DefaultPath() {
		t.Fatalf("db-path=%q", a.GetDBPath())
	}
	if !strings.HasSuffix(a.GetDBPath(), filepath.Join(".local", "mono-vcs", "state.json")) {
		t.Fatalf("db-path=%q", a.GetDBPath())
	}
}

func TestDBPathComesFromConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "vcs")
	if err := os.WriteFile(cfg, []byte(`{"gl-url":"https://gl","db-path":"/tmp/custom.json"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	pointTo(t, cfg)
	a := &app.Args{}
	if err := ApplyDefaults(a); err != nil {
		t.Fatal(err)
	}
	if a.GetDBPath() != "/tmp/custom.json" {
		t.Fatalf("db-path=%q", a.GetDBPath())
	}
}
