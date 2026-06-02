package commands_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mono-vcs/internal/app"
	"mono-vcs/internal/testutil"
)

func writeFile(repo, name, content string) error {
	return os.WriteFile(filepath.Join(repo, name), []byte(content), 0o644)
}

func placeLinked(t *testing.T, ws, name string) string {
	t.Helper()
	local := testutil.MakeRepo(t, ws, name, "main")
	testutil.MakeRemote(t, t.TempDir(), local, "main")
	return local
}

func newCtx(args *app.Args, stdin string) (*app.Context, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	return &app.Context{
		Args:   args,
		Stdin:  strings.NewReader(stdin),
		Stdout: &out,
		Stderr: &errb,
	}, &out, &errb
}

func headSHA(t *testing.T, repo string) string {
	t.Helper()
	return strings.TrimSpace(testutil.Run(t, repo, "git", "rev-parse", "HEAD"))
}

func makeSyncedPair(t *testing.T, ws string, fake *testutil.FakeGitLab, name string) (repo string, id int, sha string) {
	t.Helper()
	repo = testutil.MakeRepo(t, ws, name, "main")
	sha = headSHA(t, repo)
	id = fake.AddProject(name, "")
	fake.SetBranchSHA(id, "main", sha)
	return repo, id, sha
}

func contains(t *testing.T, out, sub string) {
	t.Helper()
	if !strings.Contains(out, sub) {
		t.Fatalf("expected %q in:\n%s", sub, out)
	}
}

func notContains(t *testing.T, out, sub string) {
	t.Helper()
	if strings.Contains(out, sub) {
		t.Fatalf("did not expect %q in:\n%s", sub, out)
	}
}
