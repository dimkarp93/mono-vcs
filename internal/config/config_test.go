package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mono-vcs/internal/app"
	"mono-vcs/internal/testutil"
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

func TestLoadReturnsValues(t *testing.T) {
	p := filepath.Join(t.TempDir(), "vcs")
	os.WriteFile(p, []byte(`{"gl-url":"https://example","jobs":4,"no-color":true,"main-branch":"master"}`), 0o644)
	pointTo(t, p)
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	want := Config{GLURL: "https://example", Jobs: 4, NoColor: true, MainBranch: "master"}
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
	if err := Save(Config{GLURL: "https://example", Jobs: 4, MainBranch: "main"}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(p)
	text := string(data)
	for _, want := range []string{`"gl-url": "https://example"`, `"jobs": 4`, `"main-branch": "main"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in:\n%s", want, text)
		}
	}
	if strings.Contains(text, "no-color") {
		t.Fatalf("expected no-color (false) to be omitted:\n%s", text)
	}
	if strings.Contains(strings.ToLower(text), "token") {
		t.Fatalf("token must never be written:\n%s", text)
	}
}

func TestSaveThenLoadRoundtrip(t *testing.T) {
	pointTo(t, filepath.Join(t.TempDir(), "vcs"))
	in := Config{GLURL: "https://x", Jobs: 7, NoColor: true, MainBranch: "master"}
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
	os.WriteFile(p, []byte(`{"gl-url":"https://from-config","jobs":9,"no-color":true,"main-branch":"master"}`), 0o644)
	pointTo(t, p)
	a := &app.Args{GLURL: testutil.S("https://from-cli"), Jobs: testutil.I(2), NoColor: testutil.B(false), MainBranch: testutil.S("main")}
	if err := ApplyDefaults(a); err != nil {
		t.Fatal(err)
	}
	if a.GetGLURL() != "https://from-cli" || a.GetJobs() != 2 || a.GetNoColor() || a.GetMainBranch() != "main" {
		t.Fatalf("CLI values should win: %+v", a)
	}
}

func TestApplyDefaultsFillsNils(t *testing.T) {
	p := filepath.Join(t.TempDir(), "vcs")
	os.WriteFile(p, []byte(`{"gl-url":"https://from-config","jobs":9,"no-color":true,"main-branch":"master"}`), 0o644)
	pointTo(t, p)
	a := &app.Args{}
	if err := ApplyDefaults(a); err != nil {
		t.Fatal(err)
	}
	if a.GetGLURL() != "https://from-config" || a.GetJobs() != 9 || !a.GetNoColor() || a.GetMainBranch() != "master" {
		t.Fatalf("got %+v", a)
	}
}

func TestApplyDefaultsMainBranchFallback(t *testing.T) {
	p := filepath.Join(t.TempDir(), "vcs")
	os.WriteFile(p, []byte(`{"gl-url":"https://x"}`), 0o644)
	pointTo(t, p)
	a := &app.Args{}
	ApplyDefaults(a)
	if a.GetMainBranch() != DefaultMainBranch || DefaultMainBranch != "main" {
		t.Fatalf("got %q", a.GetMainBranch())
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

func TestApplyDefaultsNoColorFromConfig(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want bool
	}{{"true", true}, {"false", false}} {
		p := filepath.Join(t.TempDir(), "vcs")
		os.WriteFile(p, []byte(`{"gl-url":"https://x","no-color":`+tc.raw+`}`), 0o644)
		pointTo(t, p)
		a := &app.Args{}
		ApplyDefaults(a)
		if a.GetNoColor() != tc.want {
			t.Fatalf("raw=%s got %v", tc.raw, a.GetNoColor())
		}
	}
}
