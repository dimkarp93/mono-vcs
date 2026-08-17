package repos

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func mark(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(path, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestScanReturnsRelativeForwardSlashPaths(t *testing.T) {
	root := t.TempDir()
	mark(t, filepath.Join(root, "group", "repo"))
	mark(t, filepath.Join(root, "other", "sub", "repo"))
	got := ScanLocalRepos(root)
	want := map[string]bool{"group/repo": true, "other/sub/repo": true}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestScanSkipsHiddenDirs(t *testing.T) {
	root := t.TempDir()
	mark(t, filepath.Join(root, ".cache", "repo"))
	mark(t, filepath.Join(root, "real", "repo"))
	got := ScanLocalRepos(root)
	if !reflect.DeepEqual(got, map[string]bool{"real/repo": true}) {
		t.Fatalf("got %v", got)
	}
}

func TestScanDoesNotDescendIntoRepo(t *testing.T) {
	root := t.TempDir()
	mark(t, filepath.Join(root, "outer"))
	mark(t, filepath.Join(root, "outer", "inner"))
	got := ScanLocalRepos(root)
	if !reflect.DeepEqual(got, map[string]bool{"outer": true}) {
		t.Fatalf("got %v", got)
	}
}

func TestScanExcludesRootItself(t *testing.T) {
	root := t.TempDir()
	mark(t, root)
	if got := ScanLocalRepos(root); len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}

func TestScanEmptyDir(t *testing.T) {
	if got := ScanLocalRepos(t.TempDir()); len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}

func TestFilterNoneIsPassthrough(t *testing.T) {
	got := FilterRepos([]string{"a/b", "c"}, nil, nw())
	if !reflect.DeepEqual(got, []string{"a/b", "c"}) {
		t.Fatalf("got %v", got)
	}
}

func TestFilterBareNameMatchesBasename(t *testing.T) {
	got := FilterRepos([]string{"a/b/repo", "x/other"}, []string{"repo"}, nw())
	if !reflect.DeepEqual(got, []string{"a/b/repo"}) {
		t.Fatalf("got %v", got)
	}
}

func TestFilterTrailingSlashMatchesFolderRecursively(t *testing.T) {
	got := FilterRepos([]string{"libs/a", "libs/sub/b", "tools/c"}, []string{"libs/"}, nw())
	if !reflect.DeepEqual(got, []string{"libs/a", "libs/sub/b"}) {
		t.Fatalf("got %v", got)
	}
}

func TestFilterCommaSeparatedInOneEntry(t *testing.T) {
	got := FilterRepos([]string{"libs/a", "libs/b", "tools/sync", "x/y"}, []string{"libs/,sync"}, nw())
	if !reflect.DeepEqual(got, []string{"libs/a", "libs/b", "tools/sync"}) {
		t.Fatalf("got %v", got)
	}
}

func TestFilterSpaceSeparatedArgs(t *testing.T) {
	got := FilterRepos([]string{"libs/a", "tools/sync"}, []string{"libs/", "sync"}, nw())
	if !reflect.DeepEqual(got, []string{"libs/a", "tools/sync"}) {
		t.Fatalf("got %v", got)
	}
}

func TestFilterUnmatchedWarns(t *testing.T) {
	var buf bytes.Buffer
	got := FilterRepos([]string{"libs/a"}, []string{"nope/", "ghost"}, &buf)
	if len(got) != 0 {
		t.Fatalf("got %v", got)
	}
	err := buf.String()
	if !strings.Contains(err, "ghost") || !strings.Contains(err, "nope") || !strings.Contains(err, "warning") {
		t.Fatalf("warning text: %q", err)
	}
}

func TestFilterEmptyEntriesCollapseToPassthrough(t *testing.T) {
	got := FilterRepos([]string{"a/b"}, []string{", ,"}, nw())
	if !reflect.DeepEqual(got, []string{"a/b"}) {
		t.Fatalf("got %v", got)
	}
}

func TestFilterDedupsAndSorts(t *testing.T) {
	got := FilterRepos([]string{"a/x", "b/x", "c/y"}, []string{"x"}, nw())
	if !reflect.DeepEqual(got, []string{"a/x", "b/x"}) {
		t.Fatalf("got %v", got)
	}
}

func TestFilterFullPathMatchesExactly(t *testing.T) {
	got := FilterRepos([]string{"a/b/repo", "x/repo", "repo"}, []string{"a/b/repo"}, nw())
	if !reflect.DeepEqual(got, []string{"a/b/repo"}) {
		t.Fatalf("got %v", got)
	}
}

func TestFilterFullPathMatchesBySuffix(t *testing.T) {
	got := FilterRepos([]string{"mono/backend/api", "mono/libs/api"}, []string{"backend/api"}, nw())
	if !reflect.DeepEqual(got, []string{"mono/backend/api"}) {
		t.Fatalf("got %v", got)
	}
}

func TestFilterSuffixRespectsSegmentBoundary(t *testing.T) {
	got := FilterRepos([]string{"mono/xbackend/api"}, []string{"backend/api"}, nw())
	if len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}

func TestFilterFullPathVsBareName(t *testing.T) {
	got := FilterRepos([]string{"a/b/repo", "x/repo", "repo"}, []string{"repo"}, nw())
	if !reflect.DeepEqual(got, []string{"a/b/repo", "repo", "x/repo"}) {
		t.Fatalf("got %v", got)
	}
}

func TestFilterFullPathUnmatchedWarns(t *testing.T) {
	var buf bytes.Buffer
	got := FilterRepos([]string{"a/repo"}, []string{"a/missing"}, &buf)
	if len(got) != 0 {
		t.Fatalf("got %v", got)
	}
	if w := buf.String(); !strings.Contains(w, "a/missing") || !strings.Contains(w, "warning") {
		t.Fatalf("warning text: %q", w)
	}
}

func TestFilterMixedKinds(t *testing.T) {
	got := FilterRepos(
		[]string{"libs/a", "libs/b", "tools/sync", "a/b/repo"},
		[]string{"libs/,a/b/repo,sync"},
		nw(),
	)
	if !reflect.DeepEqual(got, []string{"a/b/repo", "libs/a", "libs/b", "tools/sync"}) {
		t.Fatalf("got %v", got)
	}
}

func nw() *bytes.Buffer { return &bytes.Buffer{} }
