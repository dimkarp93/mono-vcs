package dryrun

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/output"
)

func ns() *app.Args {
	return &app.Args{}
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

func TestDryFinish(t *testing.T) {
	a := ns()
	a.Branch = "feature"
	out := render(func(b *bytes.Buffer) { Finish(b, a, []string{"a"}) })
	expectHeader(t, out, "finish feature", 1)
	mustContain(t, out, "branch -D feature", "checkout <default-branch>",
		"reset --hard HEAD", "clean -fd")
}

func TestDryDone(t *testing.T) {
	out := render(func(b *bytes.Buffer) { Done(b, ns(), []string{"a"}) })
	expectHeader(t, out, "done", 1)
	mustContain(t, out, "fetch --prune --quiet origin",
		"merge-base --is-ancestor <B> <target>", "branch -D <B>",
		"спросить подтверждение")
}

func TestDryDoneWithYes(t *testing.T) {
	a := ns()
	a.Yes = true
	out := render(func(b *bytes.Buffer) { Done(b, a, []string{"a"}) })
	mustContain(t, out, "-y передан")
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

func tableLines(out string) []string {
	var lines []string
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, "┌") || strings.HasPrefix(l, "│") ||
			strings.HasPrefix(l, "├") || strings.HasPrefix(l, "└") {
			lines = append(lines, l)
		}
	}
	return lines
}

func TestPrintTableFitsTerminal(t *testing.T) {
	long := []string{}
	for i := 0; i < 7; i++ {
		long = append(long, fmt.Sprintf("группа/подгруппа/очень-длинный-репозиторий-%d", i))
	}
	for _, width := range []int{60, 100, 200} {
		t.Setenv("COLUMNS", strconv.Itoa(width))
		out := render(func(b *bytes.Buffer) { PrintTable(b, long, 6) })
		lines := tableLines(out)
		if len(lines) == 0 {
			t.Fatalf("no table:\n%s", out)
		}
		want := output.DisplayWidth(lines[0])
		if want > width {
			t.Fatalf("width %d exceeds terminal %d:\n%s", want, width, out)
		}
		for _, l := range lines {
			if w := output.DisplayWidth(l); w != want {
				t.Fatalf("ragged line (%d != %d): %q\n%s", w, want, l, out)
			}
		}
	}
}

func TestPrintTableColumnsAdaptToWidth(t *testing.T) {
	repos := []string{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "cccccccccccccccccccccccccccccc"}
	t.Setenv("COLUMNS", "60")
	narrow := tableLines(render(func(b *bytes.Buffer) { PrintTable(b, repos, 6) }))
	t.Setenv("COLUMNS", "200")
	wide := tableLines(render(func(b *bytes.Buffer) { PrintTable(b, repos, 6) }))
	if !(len(narrow) > len(wide)) {
		t.Fatalf("narrow terminal must produce more rows: %d vs %d", len(narrow), len(wide))
	}
}

func TestDryDoQuotesMultipleArgs(t *testing.T) {
	a := ns()
	a.Action = []string{"git", "commit", "-m", "two words"}
	out := render(func(b *bytes.Buffer) { Do(b, a, []string{"a"}) })
	mustContain(t, out, "cd <repo-name> && git commit -m 'two words'")
}

func TestDryNewCarriesChangesOver(t *testing.T) {
	a := ns()
	a.Branch = "feature"
	out := render(func(b *bytes.Buffer) { New(b, a, []string{"a"}) })
	expectHeader(t, out, "new feature", 1)
	mustContain(t, out, "stash push --include-untracked", "stash pop",
		"checkout -b feature <default-branch>", "checkout feature")
}
