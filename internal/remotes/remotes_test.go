package remotes

import (
	"reflect"
	"testing"
)

func TestNormalize(t *testing.T) {
	cases := []struct {
		in   string
		host string
		key  string
	}{
		{"https://gitlab.company.com/mono/backend/api.git", "gitlab.company.com", "gitlab.company.com/mono/backend/api"},
		{"https://user@gitlab.company.com/mono/api/", "gitlab.company.com", "gitlab.company.com/mono/api"},
		{"git@github.com:owner/repo.git", "github.com", "github.com/owner/repo"},
		{"ssh://git@gitea.example.org:2222/owner/repo.git", "gitea.example.org:2222", "gitea.example.org:2222/owner/repo"},
		{"ssh://git@gitea.example.org:22/owner/repo", "gitea.example.org", "gitea.example.org/owner/repo"},
		{"HTTPS://GitLab.Company.COM/A/B", "gitlab.company.com", "gitlab.company.com/A/B"},
		{"github.com", "github.com", "github.com"},
		{"gitlab.company.com/mono/api", "gitlab.company.com", "gitlab.company.com/mono/api"},
		{"/srv/git/repo.git", LocalHost, "local/srv/git/repo"},
		{"file:///srv/git/repo.git", LocalHost, "local/srv/git/repo"},
		{"../sibling", LocalHost, "local/../sibling"},
		{"", "", ""},
	}
	for _, c := range cases {
		host, key := Normalize(c.in)
		if host != c.host || key != c.key {
			t.Fatalf("Normalize(%q) = %q, %q; want %q, %q", c.in, host, key, c.host, c.key)
		}
	}
}

func TestSystemAliasDropsLastLabel(t *testing.T) {
	cases := map[string]string{
		"github.com":            "github",
		"gitlab.com":            "gitlab",
		"gitlab.mycompany.com":  "gitlab.mycompany",
		"localhost":             "localhost",
		"10.0.0.7":              "10.0.0.7",
		"gitea.example.org:222": "gitea.example",
	}
	for in, want := range cases {
		if got := SystemAlias(in); got != want {
			t.Fatalf("SystemAlias(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAssignAliasesFallsBackToFullHostOnCollision(t *testing.T) {
	got := AssignAliases([]string{"gitlab.com", "gitlab.org", "github.com"})
	want := map[string]string{
		"gitlab.com": "gitlab.com",
		"gitlab.org": "gitlab.org",
		"github.com": "github",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func newSet(byRepo map[string][]Remote) *Set {
	s := &Set{byRepo: byRepo}
	s.refresh()
	return s
}

func fixture() *Set {
	return newSet(map[string][]Remote{
		"grp/api": {Make("origin", "https://gitlab.company.com/grp/api.git")},
		"grp/web": {
			Make("origin", "git@github.com:grp/web.git"),
			Make("upstream", "https://gitlab.company.com/grp/web.git"),
		},
	})
}

func TestOriginPrefersOrigin(t *testing.T) {
	r, ok := Origin(fixture().Of("grp/web"))
	if !ok || r.Host != "github.com" {
		t.Fatalf("got %+v, ok=%v", r, ok)
	}
}

func TestOriginAliasUsesSystemAlias(t *testing.T) {
	if got := fixture().OriginAlias("grp/api"); got != "gitlab.company" {
		t.Fatalf("got %q", got)
	}
}

func TestMatchBySystemAlias(t *testing.T) {
	got := fixture().Match([]string{"grp/api", "grp/web"}, []string{"github"}, nil, nil)
	if !reflect.DeepEqual(got, []string{"grp/web"}) {
		t.Fatalf("got %v", got)
	}
}

func TestMatchHitsSecondaryRemote(t *testing.T) {
	got := fixture().Match([]string{"grp/api", "grp/web"}, []string{"gitlab.company"}, nil, nil)
	if !reflect.DeepEqual(got, []string{"grp/api", "grp/web"}) {
		t.Fatalf("got %v", got)
	}
}

func TestMatchByFullURLAndCustomAlias(t *testing.T) {
	set := fixture()
	byURL := set.Match([]string{"grp/api", "grp/web"}, []string{"https://gitlab.company.com/grp/api.git"}, nil, nil)
	if !reflect.DeepEqual(byURL, []string{"grp/api"}) {
		t.Fatalf("by url: %v", byURL)
	}
	custom := map[string]string{"work": "gitlab.company.com/grp/api"}
	byAlias := set.Match([]string{"grp/api", "grp/web"}, []string{"work"}, custom, nil)
	if !reflect.DeepEqual(byAlias, []string{"grp/api"}) {
		t.Fatalf("by custom alias: %v", byAlias)
	}
}

func TestMatchSplitsCommasAndWarnsOnMiss(t *testing.T) {
	var errb testWriter
	got := fixture().Match([]string{"grp/api", "grp/web"}, []string{"github, nowhere"}, nil, &errb)
	if !reflect.DeepEqual(got, []string{"grp/web"}) {
		t.Fatalf("got %v", got)
	}
	if !errb.contains("nowhere") || errb.contains("github,") {
		t.Fatalf("warning was %q", errb.text)
	}
}

func TestLegendMarksCustom(t *testing.T) {
	rows := fixture().Legend(map[string]string{"work": "gitlab.company.com"})
	if len(rows) != 3 {
		t.Fatalf("got %d rows: %+v", len(rows), rows)
	}
	last := rows[len(rows)-1]
	if !last.Custom || last.Alias != "work" || last.Host != "gitlab.company.com" {
		t.Fatalf("got %+v", last)
	}
}

type testWriter struct{ text string }

func (w *testWriter) Write(p []byte) (int, error) {
	w.text += string(p)
	return len(p), nil
}

func (w *testWriter) contains(s string) bool {
	for i := 0; i+len(s) <= len(w.text); i++ {
		if w.text[i:i+len(s)] == s {
			return true
		}
	}
	return false
}
