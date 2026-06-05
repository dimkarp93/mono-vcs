package cli

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"mono-vcs/internal/app"
	"mono-vcs/internal/config"
)

func newRunner(t *testing.T) (*Runner, *capture) {
	t.Helper()
	config.SetPath(filepath.Join(t.TempDir(), "no-config"))
	t.Cleanup(func() { config.SetPath("") })
	r := New(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	return r, &capture{}
}

func newRunnerWithGLURL(t *testing.T) (*Runner, *capture) {
	t.Helper()
	config.SetPath(filepath.Join(t.TempDir(), "config"))
	t.Cleanup(func() { config.SetPath("") })
	if err := config.Save(config.Config{GLURL: "https://x"}); err != nil {
		t.Fatal(err)
	}
	r := New(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	return r, &capture{}
}

type capture struct {
	args   *app.Args
	called bool
}

func (c *capture) handler(ctx *app.Context) int {
	c.called = true
	c.args = ctx.Args
	return 0
}

func TestSubcommandsRegistered(t *testing.T) {
	got := map[string]bool{}
	for k := range DefaultCommands() {
		got[k] = true
	}
	want := map[string]bool{
		"list": true, "clone": true, "pull": true, "update": true,
		"stash": true, "unstash": true, "clear-stash": true, "history-stash": true,
		"prune": true, "features": true, "do": true, "switch": true,
		"cancel": true, "new": true, "init": true,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
}

func TestEveryCommandIsGroupedAndDescribed(t *testing.T) {
	grouped := map[string]bool{}
	for _, g := range commandGroups {
		for _, n := range g.commands {
			if grouped[n] {
				t.Fatalf("%q listed in more than one group", n)
			}
			grouped[n] = true
			if commandSummaries[n] == "" {
				t.Fatalf("%q has no summary", n)
			}
		}
	}
	for n := range DefaultCommands() {
		if !grouped[n] {
			t.Fatalf("registered command %q is not in any help group", n)
		}
	}
}

func TestHelpDoesNotCrash(t *testing.T) {
	for _, arg := range []string{"--help", "-help", "-h", "help"} {
		r, _ := newRunner(t)
		if rc := r.Run([]string{arg}); rc != 0 {
			t.Fatalf("%s: rc=%d", arg, rc)
		}
	}
}

func TestListRequiresGLURL(t *testing.T) {
	r, _ := newRunner(t)
	if rc := r.Run([]string{"list"}); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
}

func TestPullDryRunSkipsToken(t *testing.T) {
	r, c := newRunnerWithGLURL(t)
	r.Commands["pull"] = c.handler
	r.TokenFunc = func(optional bool) (string, error) {
		t.Fatal("prompt_token must not be called in pull --dry-run")
		return "", nil
	}
	if rc := r.Run([]string{"pull", "--dry-run"}); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	if c.args.GLToken != "" {
		t.Fatalf("token=%q", c.args.GLToken)
	}
}

func TestListRequestsToken(t *testing.T) {
	r, c := newRunnerWithGLURL(t)
	r.Commands["list"] = c.handler
	r.TokenFunc = func(optional bool) (string, error) {
		if optional {
			return "", nil
		}
		return "TOK", nil
	}
	r.Run([]string{"list"})
	if c.args.GLToken != "TOK" {
		t.Fatalf("token=%q", c.args.GLToken)
	}
}

func TestUpdateTokenOptional(t *testing.T) {
	r, c := newRunner(t)
	r.Commands["update"] = c.handler
	gotOptional := false
	r.TokenFunc = func(optional bool) (string, error) {
		gotOptional = optional
		return "", nil
	}
	r.Run([]string{"update"})
	if !gotOptional || c.args.GLToken != "" {
		t.Fatalf("optional=%v token=%q", gotOptional, c.args.GLToken)
	}
}

func TestStashDoesNotRequestToken(t *testing.T) {
	r, c := newRunner(t)
	r.Commands["stash"] = c.handler
	r.TokenFunc = func(bool) (string, error) { t.Fatal("token should not be asked"); return "", nil }
	if rc := r.Run([]string{"stash"}); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
}

func TestPruneDoesNotRequestToken(t *testing.T) {
	r, c := newRunner(t)
	r.Commands["prune"] = c.handler
	r.TokenFunc = func(bool) (string, error) { t.Fatal("token should not be asked"); return "", nil }
	if rc := r.Run([]string{"prune"}); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
}

func TestInitSkipsConfigAndToken(t *testing.T) {
	r, c := newRunner(t)
	r.Commands["init"] = c.handler
	r.TokenFunc = func(bool) (string, error) { t.Fatal("token should not be asked"); return "", nil }
	if rc := r.Run([]string{"init"}); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
}

func TestNewDoesNotRequestToken(t *testing.T) {
	r, c := newRunner(t)
	r.Commands["new"] = c.handler
	r.TokenFunc = func(bool) (string, error) { t.Fatal("token should not be asked"); return "", nil }
	if rc := r.Run([]string{"new", "MVPAY-290"}); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	if c.args.Branch != "MVPAY-290" {
		t.Fatalf("branch=%q", c.args.Branch)
	}
}

func TestFeatAndRepoMutuallyExclusive(t *testing.T) {
	r, c := newRunner(t)
	r.Commands["do"] = c.handler
	if rc := r.Run([]string{"do", "-repo", "a", "-feat", "x", "git", "status"}); rc != 2 {
		t.Fatalf("rc=%d", rc)
	}
	if c.called {
		t.Fatal("handler should not run when -repo and -feat conflict")
	}
}

func TestFeatFlagParses(t *testing.T) {
	r, c := newRunner(t)
	r.Commands["do"] = c.handler
	if rc := r.Run([]string{"do", "-feat", "MVPAY-290", "git", "status"}); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	if c.args.Feature != "MVPAY-290" {
		t.Fatalf("feature=%q", c.args.Feature)
	}
}

func TestFeatShorthandRejected(t *testing.T) {
	r, c := newRunner(t)
	r.Commands["do"] = c.handler
	if rc := r.Run([]string{"do", "-f", "MVPAY-290", "git", "status"}); rc != 2 {
		t.Fatalf("rc=%d", rc)
	}
	if c.called {
		t.Fatal("handler should not run for removed -f shorthand")
	}
}

func TestSwitchDryRunSkipsToken(t *testing.T) {
	r, c := newRunner(t)
	r.Commands["switch"] = c.handler
	r.TokenFunc = func(bool) (string, error) { t.Fatal("token should not be asked"); return "", nil }
	if rc := r.Run([]string{"switch", "feature", "--dry-run"}); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
}
