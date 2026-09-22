package output

import (
	"bytes"
	"strings"
	"testing"
)

func rowsFixture() []FeatureRow {
	longBranch := "feature/" + strings.Repeat("very-long-branch-segment-", 3)
	longRepo := "группа/подгруппа/очень-длинное-имя-репозитория-которое-не-влезает"
	return []FeatureRow{
		{Branch: "feat-x", Repos: []string{"a", "b"}, Active: []string{"a"}},
		{Branch: longBranch, Repos: []string{longRepo, longRepo + "-2"}, Active: nil},
		{Branch: "日本語のブランチ", Repos: []string{"group/日本語リポジトリ"}, Active: []string{"group/日本語リポジトリ"}},
	}
}

func assertRectangular(t *testing.T, out string, limit int) {
	t.Helper()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) < 3 {
		t.Fatalf("too few lines:\n%s", out)
	}
	want := DisplayWidth(lines[0])
	if want > limit {
		t.Fatalf("table width %d exceeds terminal %d:\n%s", want, limit, out)
	}
	for i, l := range lines {
		if w := DisplayWidth(l); w != want {
			t.Fatalf("line %d width %d, want %d: %q\n%s", i, w, want, l, out)
		}
	}
}

func TestFeaturesTableKeepsBordersAligned(t *testing.T) {
	for _, width := range []string{"60", "80", "100", "140", "240"} {
		t.Setenv("COLUMNS", width)
		var b bytes.Buffer
		PrintFeaturesTable(&b, rowsFixture(), false)
		assertRectangular(t, b.String(), atoi(t, width))
	}
}

func TestFeaturesTableWithColorKeepsBordersAligned(t *testing.T) {
	t.Setenv("COLUMNS", "100")
	var b bytes.Buffer
	PrintFeaturesTable(&b, rowsFixture(), true)
	out := b.String()
	if !strings.Contains(out, "\x1b[") {
		t.Fatalf("expected ansi colors in output")
	}
	for _, l := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if w := DisplayWidth(stripANSI(l)); w != 100 {
			t.Fatalf("line width %d: %q", w, l)
		}
	}
}

func TestFeatureColumnsGrowWithTerminal(t *testing.T) {
	rows := []FeatureRow{{Branch: "feat", Repos: []string{"a"}}}
	_, narrow, _, _ := featureColumns(100, rows)
	_, wide, _, _ := featureColumns(240, rows)
	if wide <= narrow {
		t.Fatalf("path column did not grow: %d -> %d", narrow, wide)
	}
	branchW, col2W, col3W, col4W := featureColumns(100, rowsFixture())
	if branchW+col2W+col3W+col4W+13 != 100 {
		t.Fatalf("columns %d/%d/%d/%d do not fill 100", branchW, col2W, col3W, col4W)
	}
	if branchW > 100 {
		t.Fatalf("branch column unbounded: %d", branchW)
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '\x1b' {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			i++
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func atoi(t *testing.T, s string) int {
	t.Helper()
	n := 0
	for _, r := range s {
		n = n*10 + int(r-'0')
	}
	return n
}

func TestItemsPutsEachRepoOnOwnLine(t *testing.T) {
	lines := Items([]string{"группа/репо-один", "группа/репо-два"}, 20)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), lines)
	}
	for _, l := range lines {
		if w := DisplayWidth(l); w > 20 {
			t.Fatalf("line too wide (%d): %q", w, l)
		}
		if strings.Contains(l, ",") {
			t.Fatalf("repos still joined by comma: %q", l)
		}
	}
	long := Items([]string{strings.Repeat("x", 50)}, 10)
	if len(long) != 1 || DisplayWidth(long[0]) > 10 {
		t.Fatalf("overlong item not truncated: %q", long)
	}
	if got := Items(nil, 10); len(got) != 1 || got[0] != "" {
		t.Fatalf("empty items: %q", got)
	}
}

func TestFeaturesTableRowPerRepo(t *testing.T) {
	t.Setenv("COLUMNS", "100")
	var b bytes.Buffer
	PrintFeaturesTable(&b, []FeatureRow{
		{Branch: "feat-x", Repos: []string{"grp/a", "grp/b", "grp/c"}, Active: []string{"grp/a"}},
	}, false)
	out := b.String()
	if strings.Contains(out, ",") {
		t.Fatalf("repos still joined by comma:\n%s", out)
	}
	cells := map[string]int{}
	for _, l := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		parts := strings.Split(l, "│")
		if len(parts) != 6 {
			continue
		}
		if repos := strings.TrimSpace(parts[2]); repos != "" && repos != "repos" {
			cells[repos]++
		}
	}
	for _, repo := range []string{"grp/a", "grp/b", "grp/c"} {
		if cells[repo] != 1 {
			t.Fatalf("%s is not alone on its line:\n%s", repo, out)
		}
	}
	if len(cells) != 3 {
		t.Fatalf("expected 3 repo lines, got %d:\n%s", len(cells), out)
	}
}
