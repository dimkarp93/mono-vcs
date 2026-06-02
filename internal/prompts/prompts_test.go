package prompts

import (
	"bytes"
	"strings"
	"testing"
)

func newP(in string) *Prompter {
	return New(strings.NewReader(in), &bytes.Buffer{}, &bytes.Buffer{})
}

func TestEnvTokenUsedWithoutPrompt(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "envtok")

	if got, _ := newP("WRONG\n").Token(false); got != "envtok" {
		t.Fatalf("got %q", got)
	}
	if got, _ := newP("WRONG\n").Token(true); got != "envtok" {
		t.Fatalf("got %q", got)
	}
}

func TestEnvTokenStripped(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "  envtok  ")
	if got, _ := newP("WRONG\n").Token(false); got != "envtok" {
		t.Fatalf("got %q", got)
	}
}

func TestBlankEnvFallsThroughToPrompt(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "   ")
	if got, _ := newP("typed\n").Token(false); got != "typed" {
		t.Fatalf("got %q", got)
	}
}

func TestNoEnvUsesPrompt(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	if got, _ := newP("typed\n").Token(false); got != "typed" {
		t.Fatalf("got %q", got)
	}
}

func TestRequiredEmptyAborts(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	if _, err := newP("\n").Token(false); err == nil {
		t.Fatal("expected error on empty required token")
	}
}

func TestOptionalEmptyReturnsBlank(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	if got, err := newP("\n").Token(true); got != "" || err != nil {
		t.Fatalf("got %q, %v", got, err)
	}
}
