package commands_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/commands"
	"github.com/dimkarp93/mono-vcs/internal/config"
)

func TestInitCreatesConfig(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "vcs")
	config.SetPath(cfg)
	t.Cleanup(func() { config.SetPath("") })
	ctx, _, _ := newCtx(&app.Args{}, "https://gl.example\n4\nmain\n")
	if rc := commands.Init(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	data, _ := os.ReadFile(cfg)
	text := string(data)
	for _, want := range []string{`"gl-url": "https://gl.example"`, `"jobs": 4`, `"main-branch": "main"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in:\n%s", want, text)
		}
	}
	if strings.Contains(strings.ToLower(text), "token") {
		t.Fatalf("token must not be written:\n%s", text)
	}
}

func TestInitUpdatesExisting(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "vcs")
	os.WriteFile(cfg, []byte(`{"gl-url":"https://old","jobs":1,"main-branch":"master"}`), 0o644)
	config.SetPath(cfg)
	t.Cleanup(func() { config.SetPath("") })
	ctx, _, _ := newCtx(&app.Args{}, "\n\n\n")
	if rc := commands.Init(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	data, _ := os.ReadFile(cfg)
	text := string(data)
	for _, want := range []string{`"gl-url": "https://old"`, `"jobs": 1`, `"main-branch": "master"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in:\n%s", want, text)
		}
	}
}

func TestInitInvalidJobsDies(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "vcs")
	config.SetPath(cfg)
	t.Cleanup(func() { config.SetPath("") })
	ctx, _, _ := newCtx(&app.Args{}, "https://x\nabc\n")
	if rc := commands.Init(ctx); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
}
