package app_test

import (
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/app"
)

func TestShellAction(t *testing.T) {
	cases := []struct {
		name   string
		action []string
		want   string
	}{
		{"empty", nil, ""},
		{"single passthrough", []string{"git status -s && git branch"}, "git status -s && git branch"},
		{"plain args", []string{"git", "status", "-s"}, "git status -s"},
		{"spaces quoted", []string{"git", "commit", "-m", "two words"}, "git commit -m 'two words'"},
		{"single quote escaped", []string{"echo", "it's"}, `echo 'it'\''s'`},
		{"metachars quoted", []string{"echo", "a&b"}, "echo 'a&b'"},
		{"empty arg quoted", []string{"echo", ""}, "echo ''"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := &app.Args{Action: c.action}
			if got := a.ShellAction(); got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}
