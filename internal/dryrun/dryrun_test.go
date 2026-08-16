package dryrun

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"mono-vcs/internal/app"
	"mono-vcs/internal/testutil"
)

func ns() *app.Args {
	return &app.Args{MainBranch: testutil.S("main")}
}

func expectHeader(t *testing.T, out, label string, n int) {
	t.Helper()
	if !strings.Contains(out, fmt.Sprintf("DRY-RUN: %s — будет применено к %d репозитори(й/ев):", label, n)) {
		t.Fatalf("missing header for %q n=%d:\n%s", label, n, out)
	}
	if !strings.Contains(out, "План действий") {
		t.Fatalf("missing 'План действий':\n%s", out)
	}
}

func render(f func(*bytes.Buffer)) string {
	var b bytes.Buffer
	f(&b)
	return b.String()
}

func TestDryPull(t *testing.T) {
	out := render(func(b *bytes.Buffer) { Pull(b, ns(), []string{"a", "b"}) })
	expectHeader(t, out, "pull", 2)
	mustContain(t, out, "pull --ff-only --quiet", "<default-branch>")
}

func TestDryUpdate(t *testing.T) {
	out := render(func(b *bytes.Buffer) { Update(b, ns(), []string{"a"}) })
	expectHeader(t, out, "update", 1)
	mustContain(t, out, "checkout <default-branch>", "rebase <default-branch>", "незакоммиченные")
}

func TestDryStash(t *testing.T) {
	out := render(func(b *bytes.Buffer) { Stash(b, ns(), []string{"a"}) })
	expectHeader(t, out, "stash", 1)
	mustContain(t, out, "stash push --keep-index --include-untracked")
}

func TestDryUnstash(t *testing.T) {
	out := render(func(b *bytes.Buffer) { Unstash(b, ns(), []string{"a"}) })
	expectHeader(t, out, "unstash", 1)
	mustContain(t, out, "stash pop")
}

func TestDryClearStash(t *testing.T) {
	out := render(func(b *bytes.Buffer) { ClearStash(b, ns(), []string{"a"}) })
	expectHeader(t, out, "clear-stash", 1)
	mustContain(t, out, "stash clear")
}

func TestDrySwitch(t *testing.T) {
	a := ns()
	a.Branch = "feature"
	out := render(func(b *bytes.Buffer) { Switch(b, a, []string{"a"}) })
	expectHeader(t, out, "switch feature", 1)
	mustContain(t, out, "checkout feature", "checkout <default-branch>")
}

func TestDrySwitchWithoutBranch(t *testing.T) {
	out := render(func(b *bytes.Buffer) { Switch(b, ns(), []string{"a"}) })
	expectHeader(t, out, "switch", 1)
	mustContain(t, out, "checkout <default-branch>")
}

func TestDryCancel(t *testing.T) {
	a := ns()
	a.Branch = "feature"
	out := render(func(b *bytes.Buffer) { Cancel(b, a, []string{"a"}) })
	expectHeader(t, out, "cancel feature", 1)
	mustContain(t, out, "branch -D feature", "checkout <default-branch>")
}

func TestDryDo(t *testing.T) {
	a := ns()
	a.Action = []string{"echo", "hi"}
	out := render(func(b *bytes.Buffer) { Do(b, a, []string{"a"}) })
	expectHeader(t, out, "do echo hi", 1)
	mustContain(t, out, "cd <repo-name> && echo hi")
}

func TestDryTableRendersRepos(t *testing.T) {
	out := render(func(b *bytes.Buffer) { PrintTable(b, []string{"a", "b", "c", "d", "e", "f", "g"}, 3) })
	mustContain(t, out, "┌", "┐", "└", "┘")
	for _, n := range []string{"a", "b", "c", "d", "e", "f", "g"} {
		mustContain(t, out, n)
	}
}

func TestDryTableEmptyIsSilent(t *testing.T) {
	out := render(func(b *bytes.Buffer) { PrintTable(b, nil, 3) })
	if out != "" {
		t.Fatalf("expected empty, got %q", out)
	}
}

func mustContain(t *testing.T, out string, subs ...string) {
	t.Helper()
	for _, s := range subs {
		if !strings.Contains(out, s) {
			t.Fatalf("missing %q in:\n%s", s, out)
		}
	}
}
