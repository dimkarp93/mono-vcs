package commands_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/commands"
	"github.com/dimkarp93/mono-vcs/internal/testutil"
)

func aliasWorkspace(t *testing.T) {
	t.Helper()
	ws := t.TempDir()
	t.Chdir(ws)
	api := testutil.MakeRepo(t, ws, "grp/api", "main")
	testutil.Run(t, api, "git", "remote", "add", "origin", "https://gitlab.company.com/grp/api.git")
	web := testutil.MakeRepo(t, ws, "grp/web", "main")
	testutil.Run(t, web, "git", "remote", "add", "origin", "git@github.com:grp/web.git")
}

func aliasCtx(t *testing.T, action []string) (*app.Context, *bytes.Buffer, *bytes.Buffer, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "aliases.json")
	ctx, out, errb := newCtx(t, &app.Args{Action: action, AliasesPath: testutil.S(path)}, "")
	return ctx, out, errb, path
}

func TestAliasSetStoresFullValueForSystemAlias(t *testing.T) {
	aliasWorkspace(t)
	ctx, _, _, path := aliasCtx(t, []string{"remote", "work", "gitlab.company"})
	if rc := commands.Alias(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"work": "gitlab.company.com"`) {
		t.Fatalf("stored:\n%s", raw)
	}
}

func TestAliasSetNormalizesFullURL(t *testing.T) {
	aliasWorkspace(t)
	ctx, _, _, path := aliasCtx(t, []string{"remote", "api", "https://gitlab.company.com/grp/api.git"})
	if rc := commands.Alias(ctx); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	raw, _ := os.ReadFile(path)
	if !strings.Contains(string(raw), `"api": "gitlab.company.com/grp/api"`) {
		t.Fatalf("stored:\n%s", raw)
	}
}

func TestAliasSetRejectsSystemAliasName(t *testing.T) {
	aliasWorkspace(t)
	ctx, _, _, _ := aliasCtx(t, []string{"remote", "github", "gitlab.company"})
	if rc := commands.Alias(ctx); rc == 0 {
		t.Fatal("expected a failure")
	}
}

func TestAliasSetRejectsBadName(t *testing.T) {
	aliasWorkspace(t)
	ctx, _, _, _ := aliasCtx(t, []string{"remote", "a/b", "github"})
	if rc := commands.Alias(ctx); rc == 0 {
		t.Fatal("expected a failure")
	}
}

func TestAliasUnknownKind(t *testing.T) {
	aliasWorkspace(t)
	ctx, _, errb, _ := aliasCtx(t, []string{"branch", "x", "y"})
	if rc := commands.Alias(ctx); rc == 0 {
		t.Fatal("expected a failure")
	}
	contains(t, errb.String(), "unknown alias kind: branch")
}

func TestAliasDelete(t *testing.T) {
	aliasWorkspace(t)
	path := filepath.Join(t.TempDir(), "aliases.json")
	set, _, _ := newCtx(t, &app.Args{Action: []string{"remote", "work", "github"}, AliasesPath: testutil.S(path)}, "")
	if rc := commands.Alias(set); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	del, out, _ := newCtx(t, &app.Args{Action: []string{"remote", "work"}, Delete: true, AliasesPath: testutil.S(path)}, "")
	if rc := commands.Alias(del); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	contains(t, out.String(), "removed remote alias work")

	again, _, _ := newCtx(t, &app.Args{Action: []string{"remote", "work"}, Delete: true, AliasesPath: testutil.S(path)}, "")
	if rc := commands.Alias(again); rc == 0 {
		t.Fatal("expected a failure on the second delete")
	}
}

func TestAliasLsShowsSystemAndCustom(t *testing.T) {
	aliasWorkspace(t)
	path := filepath.Join(t.TempDir(), "aliases.json")
	set, _, _ := newCtx(t, &app.Args{Action: []string{"remote", "work", "gitlab.company"}, AliasesPath: testutil.S(path)}, "")
	if rc := commands.Alias(set); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	ls, out, _ := newCtx(t, &app.Args{Action: []string{"ls"}, AliasesPath: testutil.S(path)}, "")
	if rc := commands.Alias(ls); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	text := out.String()
	for _, want := range []string{"remote", "alias", "type", "value", "github", "system", "work", "custom", "┌", "│"} {
		contains(t, text, want)
	}
}

func TestAliasLsRejectsUnknownKind(t *testing.T) {
	aliasWorkspace(t)
	ls, _, errb := newCtx(t, &app.Args{Action: []string{"ls", "branch"}}, "")
	if rc := commands.Alias(ls); rc == 0 {
		t.Fatal("expected a failure")
	}
	contains(t, errb.String(), "unknown alias kind: branch")
}
