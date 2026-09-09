package jobs

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/gitops"
)

func mark(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, name, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func ctxIn(t *testing.T, repoArg []string) (*app.Context, *bytes.Buffer, *bytes.Buffer) {
	ws := t.TempDir()
	t.Chdir(ws)
	var out, errb bytes.Buffer
	return &app.Context{
		Args:   &app.Args{Repo: repoArg, Jobs: ptrI(1)},
		Stdin:  strings.NewReader(""),
		Stdout: &out,
		Stderr: &errb,
	}, &out, &errb
}

func ptrI(i int) *int { return &i }

func TestRunPerRepoDispatchesToEach(t *testing.T) {
	ctx, out, _ := ctxIn(t, nil)
	mark(t, ".", "a")
	mark(t, ".", "b")
	var seen []string
	worker := func(p string) gitops.Result { seen = append(seen, p); return gitops.Result{Path: p, Status: "ok"} }
	if rc := RunPerRepo(ctx, "test", worker, map[string]bool{"ok": true}); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	sort.Strings(seen)
	if strings.Join(seen, ",") != "a,b" {
		t.Fatalf("seen=%v", seen)
	}
	if !strings.Contains(out.String(), "[1/2] ok") || !strings.Contains(out.String(), "[2/2] ok") {
		t.Fatalf("out=%s", out.String())
	}
}

func TestRunPerRepoFailureReturnsNonzero(t *testing.T) {
	ctx, _, errb := ctxIn(t, nil)
	mark(t, ".", "a")
	mark(t, ".", "b")
	worker := func(p string) gitops.Result {
		if p == "b" {
			return gitops.Result{Path: p, Status: "failed", Detail: "boom"}
		}
		return gitops.Result{Path: p, Status: "ok"}
	}
	if rc := RunPerRepo(ctx, "test", worker, map[string]bool{"ok": true}); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
	if !strings.Contains(errb.String(), "FAILED") || !strings.Contains(errb.String(), "b") {
		t.Fatalf("err=%s", errb.String())
	}
}

func TestRunPerRepoNoReposReturnsZero(t *testing.T) {
	ctx, out, _ := ctxIn(t, nil)
	rc := RunPerRepo(ctx, "test", func(p string) gitops.Result { return gitops.Result{Path: p, Status: "ok"} }, map[string]bool{"ok": true})
	if rc != 0 || !strings.Contains(out.String(), "no local git") {
		t.Fatalf("rc=%d out=%s", rc, out.String())
	}
}

func TestRunPerRepoHonorsRepoFilter(t *testing.T) {
	ctx, _, _ := ctxIn(t, []string{"a"})
	mark(t, ".", "a")
	mark(t, ".", "b")
	var seen []string
	worker := func(p string) gitops.Result { seen = append(seen, p); return gitops.Result{Path: p, Status: "ok"} }
	if rc := RunPerRepo(ctx, "test", worker, map[string]bool{"ok": true}); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	if len(seen) != 1 || seen[0] != "a" {
		t.Fatalf("seen=%v", seen)
	}
}

func TestRunPerRepoNonzeroWithoutGit(t *testing.T) {
	ctx, _, _ := ctxIn(t, nil)
	mark(t, ".", "a")
	t.Setenv("PATH", "")
	if rc := RunPerRepo(ctx, "test", func(p string) gitops.Result { return gitops.Result{Path: p, Status: "ok"} }, map[string]bool{"ok": true}); rc == 0 {
		t.Fatal("expected nonzero when git missing")
	}
}

func TestRunPerRepoDetailAppended(t *testing.T) {
	ctx, out, _ := ctxIn(t, nil)
	mark(t, ".", "a")
	rc := RunPerRepo(ctx, "test", func(p string) gitops.Result { return gitops.Result{Path: p, Status: "ok", Detail: "more info"} }, map[string]bool{"ok": true})
	if rc != 0 || !strings.Contains(out.String(), "more info") {
		t.Fatalf("rc=%d out=%s", rc, out.String())
	}
}
