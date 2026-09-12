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
	db := filepath.Join(t.TempDir(), "state.json")
	ctx, _, _ := newCtx(t, &app.Args{}, "https://gl.example\n4\n"+db+"\n")
	if rc := commands.Init(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	data, _ := os.ReadFile(cfg)
	text := string(data)
	for _, want := range []string{`"gl-url": "https://gl.example"`, `"jobs": 4`, `"db-path": "` + db + `"`} {
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
	ctx, _, _ := newCtx(t, &app.Args{}, "\n\n\n")
	if rc := commands.Init(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	data, _ := os.ReadFile(cfg)
	text := string(data)
	for _, want := range []string{`"gl-url": "https://old"`, `"jobs": 1`} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in:\n%s", want, text)
		}
	}
	if strings.Contains(text, "main-branch") {
		t.Fatalf("retired key must not be written back:\n%s", text)
	}
}

func TestInitInvalidJobsDies(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "vcs")
	config.SetPath(cfg)
	t.Cleanup(func() { config.SetPath("") })
	ctx, _, _ := newCtx(t, &app.Args{}, "https://x\nabc\n\n")
	if rc := commands.Init(ctx); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
}
