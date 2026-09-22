package selector_test

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/aliases"
	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/selector"
	"github.com/dimkarp93/mono-vcs/internal/testutil"
)

func workspace(t *testing.T) string {
	t.Helper()
	ws := t.TempDir()
	t.Chdir(ws)
	api := testutil.MakeRepo(t, ws, "grp/api", "main")
	testutil.Run(t, api, "git", "remote", "add", "origin", "https://gitlab.company.com/grp/api.git")
	web := testutil.MakeRepo(t, ws, "grp/web", "main")
	testutil.Run(t, web, "git", "remote", "add", "origin", "git@github.com:grp/web.git")
	tools := testutil.MakeRepo(t, ws, "other/tools", "main")
	testutil.Run(t, tools, "git", "remote", "add", "origin", "git@github.com:other/tools.git")
	return ws
}

func ctxFor(t *testing.T, args *app.Args) (*app.Context, *bytes.Buffer) {
	t.Helper()
	var errb bytes.Buffer
	if args.AliasesPath == nil {
		args.AliasesPath = testutil.S(filepath.Join(t.TempDir(), "aliases.json"))
	}
	return &app.Context{Args: args, Stdin: strings.NewReader(""), Stdout: &bytes.Buffer{}, Stderr: &errb}, &errb
}

func TestSelectLocalWithoutRemoteReturnsEverything(t *testing.T) {
	workspace(t)
	ctx, _ := ctxFor(t, &app.Args{})
	got := selector.SelectLocal(ctx)
	if len(got) != 3 {
		t.Fatalf("got %v", got)
	}
}

func TestSelectLocalBySystemAlias(t *testing.T) {
	workspace(t)
	ctx, _ := ctxFor(t, &app.Args{Remote: []string{"github"}})
	got := selector.SelectLocal(ctx)
	want := []string{"grp/web", "other/tools"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestSelectLocalCombinesRepoAndRemote(t *testing.T) {
	workspace(t)
	ctx, _ := ctxFor(t, &app.Args{Repo: []string{"grp/"}, Remote: []string{"github"}})
	got := selector.SelectLocal(ctx)
	if strings.Join(got, ",") != "grp/web" {
		t.Fatalf("got %v", got)
	}
}

func TestSelectLocalByCustomAlias(t *testing.T) {
	workspace(t)
	path := filepath.Join(t.TempDir(), "aliases.json")
	st, err := aliases.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	st.Set(aliases.KindRemote, "work", "gitlab.company.com")
	if err := st.Flush(); err != nil {
		t.Fatal(err)
	}
	ctx, _ := ctxFor(t, &app.Args{AliasesPath: testutil.S(path), Remote: []string{"work"}})
	got := selector.SelectLocal(ctx)
	if strings.Join(got, ",") != "grp/api" {
		t.Fatalf("got %v", got)
	}
}

func TestSelectLocalByFullURL(t *testing.T) {
	workspace(t)
	ctx, _ := ctxFor(t, &app.Args{Remote: []string{"git@github.com:grp/web.git"}})
	if got := selector.SelectLocal(ctx); strings.Join(got, ",") != "grp/web" {
		t.Fatalf("got %v", got)
	}
}

func TestSelectLocalWarnsOnUnmatchedRemote(t *testing.T) {
	workspace(t)
	ctx, errb := ctxFor(t, &app.Args{Remote: []string{"gitea"}})
	if got := selector.SelectLocal(ctx); len(got) != 0 {
		t.Fatalf("got %v", got)
	}
	if !strings.Contains(errb.String(), "-remote entries matched nothing: gitea") {
		t.Fatalf("stderr was %q", errb.String())
	}
}
