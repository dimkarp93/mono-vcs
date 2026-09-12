package gitlab

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func projectPages(pages [][]Project, calls *[]int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := 1
		fmt.Sscanf(r.URL.Query().Get("page"), "%d", &page)
		*calls = append(*calls, page)
		if page-1 < len(pages) {
			json.NewEncoder(w).Encode(pages[page-1])
		} else {
			json.NewEncoder(w).Encode([]Project{})
		}
	}))
}

func mkProjects(n, start int) []Project {
	out := make([]Project, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, Project{ID: start + i, PathWithNamespace: fmt.Sprintf("g/r%d", start+i)})
	}
	return out
}

func TestFetchProjectsSingleShortPage(t *testing.T) {
	var urls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		urls = append(urls, r.URL.String())
		json.NewEncoder(w).Encode([]Project{{ID: 1, PathWithNamespace: "g/r"}})
	}))
	defer srv.Close()
	out, err := FetchProjects(srv.URL, "T")
	if err != nil || len(out) != 1 || len(urls) != 1 {
		t.Fatalf("out=%v err=%v urls=%v", out, err, urls)
	}
	if !strings.Contains(urls[0], "per_page=100") || !strings.Contains(urls[0], "page=1") {
		t.Fatalf("url=%s", urls[0])
	}
}

func TestFetchProjectsPaginatesUntilShort(t *testing.T) {
	var calls []int
	srv := projectPages([][]Project{mkProjects(100, 0), mkProjects(50, 100)}, &calls)
	defer srv.Close()
	out, err := FetchProjects(srv.URL, "T")
	if err != nil || len(out) != 150 {
		t.Fatalf("len=%d err=%v", len(out), err)
	}
	if fmt.Sprint(calls) != "[1 2]" {
		t.Fatalf("calls=%v", calls)
	}
}

func TestFetchProjectsPaginatesUntilEmpty(t *testing.T) {
	var calls []int
	srv := projectPages([][]Project{mkProjects(100, 0)}, &calls)
	defer srv.Close()
	out, err := FetchProjects(srv.URL, "T")
	if err != nil || len(out) != 100 {
		t.Fatalf("len=%d err=%v", len(out), err)
	}
	if fmt.Sprint(calls) != "[1 2]" {
		t.Fatalf("calls=%v", calls)
	}
}

func TestFetchProjects401Dies(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Unauthorized", 401)
	}))
	defer srv.Close()
	_, err := FetchProjects(srv.URL, "BAD")
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "401") && !strings.Contains(msg, "Unauthorized") && !strings.Contains(msg, "gl-token") {
		t.Fatalf("msg=%s", msg)
	}
}

func TestFetchProjects500DiesWithCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Server Error", 500)
	}))
	defer srv.Close()
	_, err := FetchProjects(srv.URL, "T")
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("err=%v", err)
	}
}

func TestFetchProjectsTransportErrorDies(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()
	_, err := FetchProjects(url, "T")
	if err == nil || !strings.Contains(err.Error(), url) || !strings.Contains(err.Error(), "Cannot reach GitLab") {
		t.Fatalf("err=%v", err)
	}
}

func TestFetchRemoteBranchSHASuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.EscapedPath(), "/projects/42/repository/branches/main") {
			t.Errorf("path=%s", r.URL.EscapedPath())
		}
		json.NewEncoder(w).Encode(map[string]any{"name": "main", "commit": map[string]string{"id": "deadbeef"}})
	}))
	defer srv.Close()
	if sha := FetchRemoteBranchSHA(srv.URL, "T", 42, "main"); sha != "deadbeef" {
		t.Fatalf("sha=%q", sha)
	}
}

func TestFetchRemoteBranchSHA404IsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not Found", 404)
	}))
	defer srv.Close()
	if sha := FetchRemoteBranchSHA(srv.URL, "T", 1, "main"); sha != "" {
		t.Fatalf("sha=%q", sha)
	}
}

func TestFetchRemoteBranchSHAURLEncodesSlash(t *testing.T) {
	var captured string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.URL.EscapedPath()
		json.NewEncoder(w).Encode(map[string]any{"commit": map[string]string{"id": "x"}})
	}))
	defer srv.Close()
	FetchRemoteBranchSHA(srv.URL, "T", 1, "feature/x")
	if !strings.Contains(captured, "feature%2Fx") {
		t.Fatalf("captured=%s", captured)
	}
}

func TestFetchRemoteBranchSHATransportErrorIsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()
	if sha := FetchRemoteBranchSHA(url, "T", 1, "m"); sha != "" {
		t.Fatalf("sha=%q", sha)
	}
}

func TestFetchProjectsParsesDefaultBranch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"id":7,"path_with_namespace":"g/r","http_url_to_repo":"https://h/g/r.git","default_branch":"trunk"}]`)
	}))
	defer srv.Close()
	out, err := FetchProjects(srv.URL, "T")
	if err != nil || len(out) != 1 {
		t.Fatalf("out=%v err=%v", out, err)
	}
	if out[0].DefaultBranch != "trunk" {
		t.Fatalf("default branch=%q", out[0].DefaultBranch)
	}
}

func TestFetchProjectsToleratesMissingDefaultBranch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"id":7,"path_with_namespace":"g/r"}]`)
	}))
	defer srv.Close()
	out, err := FetchProjects(srv.URL, "T")
	if err != nil || len(out) != 1 || out[0].DefaultBranch != "" {
		t.Fatalf("out=%v err=%v", out, err)
	}
}
