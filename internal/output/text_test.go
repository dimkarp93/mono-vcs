package output

import (
	"strings"
	"testing"
)

func TestDisplayWidth(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"abc", 3},
		{"группа/репозиторий", 18},
		{"日本語", 6},
		{"é", 1},
	}
	for _, c := range cases {
		if got := DisplayWidth(c.in); got != c.want {
			t.Errorf("DisplayWidth(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestTruncateKeepsWidth(t *testing.T) {
	inputs := []string{
		"group/subgroup/very-long-repo-name",
		"группа/подгруппа/очень-длинное-имя",
		"日本語日本語日本語日本語",
		"short",
	}
	for _, in := range inputs {
		for n := 0; n <= 40; n++ {
			got := Truncate(in, n)
			if w := DisplayWidth(got); w > n {
				t.Fatalf("Truncate(%q, %d) = %q has width %d", in, n, got, w)
			}
		}
	}
}

func TestTruncateMiddleEllipsis(t *testing.T) {
	got := Truncate("group/subgroup/very-long-repo-name", 24)
	if !strings.Contains(got, ellipsis) {
		t.Fatalf("no ellipsis in %q", got)
	}
	if !strings.HasPrefix(got, "group/") {
		t.Fatalf("head lost in %q", got)
	}
	if !strings.HasSuffix(got, "repo-name") {
		t.Fatalf("tail lost in %q", got)
	}
	if Truncate("abcdef", 1) != ellipsis {
		t.Fatalf("width 1 must be a bare ellipsis")
	}
	if Truncate("abcdef", 0) != "" {
		t.Fatalf("width 0 must be empty")
	}
	if Truncate("abc", 10) != "abc" {
		t.Fatalf("short string must stay intact")
	}
}

func TestCellAlwaysExactWidth(t *testing.T) {
	for _, in := range []string{"", "a", "very-long-repository-name", "имя-репозитория"} {
		for n := 1; n <= 30; n++ {
			if w := DisplayWidth(Cell(in, n)); w != n {
				t.Fatalf("Cell(%q, %d) width = %d", in, n, w)
			}
		}
	}
}

func TestPadColoredUsesPlainWidth(t *testing.T) {
	got := PadColored("abc", "\x1b[32mabc\x1b[0m", 6)
	if !strings.HasSuffix(got, "   ") {
		t.Fatalf("expected 3 trailing spaces, got %q", got)
	}
	if PadColored("abcdef", "colored", 3) != "colored" {
		t.Fatalf("overlong plain must not be padded")
	}
}
