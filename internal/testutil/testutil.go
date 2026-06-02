package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func S(s string) *string { return &s }
func I(i int) *int       { return &i }
func B(b bool) *bool     { return &b }

func gitEnv() []string {
	return append(os.Environ(),
		"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com",
	)
}

func Run(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Env = gitEnv()
	var o, e bytes.Buffer
	cmd.Stdout = &o
	cmd.Stderr = &e
	if err := cmd.Run(); err != nil {
		t.Fatalf("%v in %s: %v\n%s", args, dir, err, e.String())
	}
	return o.String()
}

func InitRepo(t *testing.T, path string, bare bool, branch string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if bare {
		Run(t, path, "git", "init", "--bare", "-b", branch)
		return
	}
	Run(t, path, "git", "init", "-b", branch)
	Run(t, path, "git", "config", "user.email", "test@example.com")
	Run(t, path, "git", "config", "user.name", "Test")
}

func Commit(t *testing.T, path, message, file, content string) string {
	t.Helper()
	if file == "" {
		file = "README"
	}
	if content == "" {
		content = "x"
	}
	if err := os.WriteFile(filepath.Join(path, file), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	Run(t, path, "git", "add", file)
	Run(t, path, "git", "commit", "-m", message)
	return strings.TrimSpace(Run(t, path, "git", "rev-parse", "HEAD"))
}

func MakeRepo(t *testing.T, dir, name, branch string) string {
	t.Helper()
	if branch == "" {
		branch = "main"
	}
	repo := filepath.Join(dir, name)
	InitRepo(t, repo, false, branch)
	Commit(t, repo, "init", "", "")
	return repo
}

func MakeRemote(t *testing.T, dir, source, branch string) string {
	t.Helper()
	if branch == "" {
		branch = "main"
	}
	remote := filepath.Join(dir, "remote-"+filepath.Base(source)+".git")
	InitRepo(t, remote, true, branch)
	Run(t, source, "git", "remote", "add", "origin", remote)
	Run(t, source, "git", "push", "-u", "origin", branch)
	return remote
}

func LinkedRepo(t *testing.T, dir string) (local, remote string) {
	local = MakeRepo(t, dir, "linked", "main")
	remote = MakeRemote(t, dir, local, "main")
	return local, remote
}

type fakeProject struct {
	ID                int    `json:"id"`
	PathWithNamespace string `json:"path_with_namespace"`
	HTTPURLToRepo     string `json:"http_url_to_repo"`
}

type FakeGitLab struct {
	mu       sync.Mutex
	projects []fakeProject
	branches map[string]string
	server   *httptest.Server
	URL      string
}

func NewFakeGitLab(t *testing.T) *FakeGitLab {
	f := &FakeGitLab{branches: map[string]string{}}
	f.server = httptest.NewServer(http.HandlerFunc(f.handle))
	f.URL = f.server.URL
	t.Cleanup(f.server.Close)
	return f
}

func (f *FakeGitLab) AddProject(pathWithNamespace, httpURL string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := len(f.projects) + 1
	if httpURL == "" {
		httpURL = "https://gitlab.example/" + pathWithNamespace + ".git"
	}
	f.projects = append(f.projects, fakeProject{id, pathWithNamespace, httpURL})
	return id
}

func (f *FakeGitLab) SetBranchSHA(projectID int, branch, sha string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.branches[fmt.Sprintf("%d|%s", projectID, branch)] = sha
}

func (f *FakeGitLab) handle(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	path := r.URL.EscapedPath()
	if strings.HasSuffix(path, "/api/v4/projects") {
		q := r.URL.Query()
		page := atoiDefault(q.Get("page"), 1)
		perPage := atoiDefault(q.Get("per_page"), 100)
		start := (page - 1) * perPage
		end := start + perPage
		if start > len(f.projects) {
			start = len(f.projects)
		}
		if end > len(f.projects) {
			end = len(f.projects)
		}
		writeJSON(w, f.projects[start:end])
		return
	}
	const marker = "/api/v4/projects/"
	if i := strings.LastIndex(path, marker); i >= 0 {
		rest := path[i+len(marker):]
		if pidStr, branchEnc, ok := strings.Cut(rest, "/repository/branches/"); ok {
			pid, _ := strconv.Atoi(pidStr)
			branch, _ := url.PathUnescape(branchEnc)
			sha, found := f.branches[fmt.Sprintf("%d|%s", pid, branch)]
			if !found {
				http.Error(w, "Not Found", http.StatusNotFound)
				return
			}
			writeJSON(w, map[string]any{"commit": map[string]string{"id": sha}})
			return
		}
	}
	http.Error(w, "unhandled fake URL", http.StatusNotFound)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func atoiDefault(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}
