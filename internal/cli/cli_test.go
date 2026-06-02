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
		"list": true, "clone": true, "pull": true, "update-main": true,
		"stash": true, "unstash": true, "clear-stash": true, "history-stash": true,
		"prune": true, "features": true, "do": true, "switch": true,
		"cancel": true, "new": true, "init": true,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
}

func TestHelpDoesNotCrash(t *testing.T) {
	r, _ := newRunner(t)
	if rc := r.Run([]string{"--help"}); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
}

func TestListRequiresGLURL(t *testing.T) {
	r, _ := newRunner(t)
	if rc := r.Run([]string{"list"}); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
}

func TestJobsZeroDies(t *testing.T) {
	r, c := newRunner(t)
	r.Commands["list"] = c.handler
	if rc := r.Run([]string{"list", "--gl-url", "https://x", "--jobs", "0"}); rc != 1 {
		t.Fatalf("rc=%d", rc)
	}
}

func TestPullDryRunSkipsToken(t *testing.T) {
	r, c := newRunner(t)
	r.Commands["pull"] = c.handler
	r.TokenFunc = func(optional bool) (string, error) {
		t.Fatal("prompt_token must not be called in pull --dry-run")
		return "", nil
	}
	if rc := r.Run([]string{"pull", "--gl-url", "https://x", "--dry-run"}); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	if c.args.GLToken != "" {
		t.Fatalf("token=%q", c.args.GLToken)
	}
}

func TestListRequestsToken(t *testing.T) {
	r, c := newRunner(t)
	r.Commands["list"] = c.handler
	r.TokenFunc = func(optional bool) (string, error) {
		if optional {
			return "", nil
		}
		return "TOK", nil
	}
	r.Run([]string{"list", "--gl-url", "https://x"})
	if c.args.GLToken != "TOK" {
		t.Fatalf("token=%q", c.args.GLToken)
	}
}

func TestUpdateMainTokenOptional(t *testing.T) {
	r, c := newRunner(t)
	r.Commands["update-main"] = c.handler
	gotOptional := false
	r.TokenFunc = func(optional bool) (string, error) {
		gotOptional = optional
		return "", nil
	}
	r.Run([]string{"update-main"})
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

func TestPruneJobsZeroDies(t *testing.T) {
	r, c := newRunner(t)
	r.Commands["prune"] = c.handler
	if rc := r.Run([]string{"prune", "--jobs", "0"}); rc != 1 {
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

func TestFeatShorthandParses(t *testing.T) {
	r, c := newRunner(t)
	r.Commands["do"] = c.handler
	if rc := r.Run([]string{"do", "-f", "MVPAY-290", "git", "status"}); rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	if c.args.Feature != "MVPAY-290" {
		t.Fatalf("feature=%q", c.args.Feature)
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
