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
		{"gitlab.company.link:2222", "gitlab.company.link:2222", "gitlab.company.link:2222"},
		{"gitlab.company.link:22", "gitlab.company.link", "gitlab.company.link"},
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

func TestNormalizeDomainLevels(t *testing.T) {
	cases := []struct {
		in   string
		host string
		key  string
	}{
		{"gitlab", LocalHost, "local/gitlab"},
		{"gitlab/grp/api", LocalHost, "local/gitlab/grp/api"},
		{"gitlab:2222", "gitlab:2222", "gitlab:2222"},
		{"https://gitlab/grp/api.git", "gitlab", "gitlab/grp/api"},
		{"git@gitlab:grp/api.git", "gitlab", "gitlab/grp/api"},
		{"ssh://git@gitlab:2222/grp/api.git", "gitlab:2222", "gitlab:2222/grp/api"},

		{"gitlab.com", "gitlab.com", "gitlab.com"},
		{"gitlab.com/grp/api", "gitlab.com", "gitlab.com/grp/api"},
		{"gitlab.com:2222", "gitlab.com:2222", "gitlab.com:2222"},
		{"https://gitlab.com/grp/api.git", "gitlab.com", "gitlab.com/grp/api"},
		{"git@gitlab.com:grp/api.git", "gitlab.com", "gitlab.com/grp/api"},
		{"ssh://git@gitlab.com:2222/grp/api.git", "gitlab.com:2222", "gitlab.com:2222/grp/api"},

		{"gitlab.corp.com", "gitlab.corp.com", "gitlab.corp.com"},
		{"gitlab.corp.com/grp/api", "gitlab.corp.com", "gitlab.corp.com/grp/api"},
		{"gitlab.corp.com:2222", "gitlab.corp.com:2222", "gitlab.corp.com:2222"},
		{"https://gitlab.corp.com/grp/api.git", "gitlab.corp.com", "gitlab.corp.com/grp/api"},
		{"git@gitlab.corp.com:grp/api.git", "gitlab.corp.com", "gitlab.corp.com/grp/api"},
		{"ssh://git@gitlab.corp.com:2222/grp/api.git", "gitlab.corp.com:2222", "gitlab.corp.com:2222/grp/api"},

		{"gitlab.dev.corp.com", "gitlab.dev.corp.com", "gitlab.dev.corp.com"},
		{"gitlab.dev.corp.com/grp/api", "gitlab.dev.corp.com", "gitlab.dev.corp.com/grp/api"},
		{"gitlab.dev.corp.com:2222", "gitlab.dev.corp.com:2222", "gitlab.dev.corp.com:2222"},
		{"https://gitlab.dev.corp.com/grp/api.git", "gitlab.dev.corp.com", "gitlab.dev.corp.com/grp/api"},
		{"git@gitlab.dev.corp.com:grp/api.git", "gitlab.dev.corp.com", "gitlab.dev.corp.com/grp/api"},
		{"ssh://git@gitlab.dev.corp.com:2222/grp/api.git", "gitlab.dev.corp.com:2222", "gitlab.dev.corp.com:2222/grp/api"},

		{"gitlab.eu.dev.corp.com", "gitlab.eu.dev.corp.com", "gitlab.eu.dev.corp.com"},
		{"gitlab.eu.dev.corp.com/grp/api", "gitlab.eu.dev.corp.com", "gitlab.eu.dev.corp.com/grp/api"},
		{"gitlab.eu.dev.corp.com:2222", "gitlab.eu.dev.corp.com:2222", "gitlab.eu.dev.corp.com:2222"},
		{"https://gitlab.eu.dev.corp.com/grp/api.git", "gitlab.eu.dev.corp.com", "gitlab.eu.dev.corp.com/grp/api"},
		{"git@gitlab.eu.dev.corp.com:grp/api.git", "gitlab.eu.dev.corp.com", "gitlab.eu.dev.corp.com/grp/api"},
		{"ssh://git@gitlab.eu.dev.corp.com:2222/grp/api.git", "gitlab.eu.dev.corp.com:2222", "gitlab.eu.dev.corp.com:2222/grp/api"},
	}
	for _, c := range cases {
		host, key := Normalize(c.in)
		if host != c.host || key != c.key {
			t.Errorf("Normalize(%q) = %q, %q; want %q, %q", c.in, host, key, c.host, c.key)
		}
	}
}

func TestSystemAliasDomainLevels(t *testing.T) {
	cases := []struct {
		host  string
		alias string
	}{
		{"gitlab", "gitlab"},
		{"gitlab:2222", "gitlab"},
		{"gitlab.com", "gitlab"},
		{"gitlab.com:2222", "gitlab"},
		{"gitlab.corp.com", "gitlab.corp"},
		{"gitlab.corp.com:2222", "gitlab.corp"},
		{"gitlab.dev.corp.com", "gitlab.dev.corp"},
		{"gitlab.dev.corp.com:2222", "gitlab.dev.corp"},
		{"gitlab.eu.dev.corp.com", "gitlab.eu.dev.corp"},
		{"gitlab.eu.dev.corp.com:2222", "gitlab.eu.dev.corp"},
	}
	for _, c := range cases {
		if got := SystemAlias(c.host); got != c.alias {
			t.Errorf("SystemAlias(%q) = %q, want %q", c.host, got, c.alias)
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

func TestMatchByCustomAliasToHostWithPort(t *testing.T) {
	set := newSet(map[string][]Remote{
		"grp/api": {Make("origin", "ssh://git@gitlab.companypay.link:2222/grp/api.git")},
		"grp/web": {Make("origin", "git@github.com:grp/web.git")},
	})
	custom := map[string]string{"companypay": "gitlab.companypay.link:2222"}
	got := set.Match([]string{"grp/api", "grp/web"}, []string{"companypay"}, custom, nil)
	if !reflect.DeepEqual(got, []string{"grp/api"}) {
		t.Fatalf("got %v", got)
	}
}

func TestResolveFollowsAliasChain(t *testing.T) {
	set := fixture()
	custom := map[string]string{"a": "b", "b": "gitlab.company"}
	if got := set.Resolve("a", custom); got != "gitlab.company.com" {
		t.Fatalf("got %q", got)
	}
}

func chainFixture() *Set {
	return newSet(map[string][]Remote{
		"grp/api":  {Make("origin", "https://gitlab.company.com/grp/api.git")},
		"grp/web":  {Make("origin", "git@github.com:grp/web.git")},
		"grp/pay":  {Make("origin", "ssh://git@gitlab.companypay.link:2222/grp/pay.git")},
		"grp/deep": {Make("origin", "git@git.eu.dev.corp.com:grp/deep.git")},
	})
}

func TestResolveFollowsCustomAliasChains(t *testing.T) {
	cases := []struct {
		name   string
		custom map[string]string
		token  string
		want   string
	}{
		{"one level to host", map[string]string{"a": "gitlab.company.com"}, "a", "gitlab.company.com"},
		{"two levels to system alias", map[string]string{"a": "b", "b": "gitlab.company"}, "a", "gitlab.company.com"},
		{"three levels to system alias", map[string]string{"a": "b", "b": "c", "c": "github"}, "a", "github.com"},
		{"four levels to host with port", map[string]string{"a": "b", "b": "c", "c": "d", "d": "gitlab.companypay.link:2222"}, "a", "gitlab.companypay.link:2222"},
		{"five levels to system alias with port", map[string]string{"a": "b", "b": "c", "c": "d", "d": "e", "e": "gitlab.companypay"}, "a", "gitlab.companypay.link:2222"},
		{"three levels to full url", map[string]string{"a": "b", "b": "c", "c": "https://gitlab.company.com/grp/api.git"}, "a", "gitlab.company.com/grp/api"},
		{"three levels to scp url", map[string]string{"a": "b", "b": "c", "c": "git@github.com:grp/web.git"}, "a", "github.com/grp/web"},
		{"three levels to deep domain alias", map[string]string{"a": "b", "b": "c", "c": "git.eu.dev.corp"}, "a", "git.eu.dev.corp.com"},
		{"start from middle of chain", map[string]string{"a": "b", "b": "c", "c": "github"}, "b", "github.com"},
		{"custom shadows system alias", map[string]string{"github": "mine", "mine": "gitlab.company"}, "github", "gitlab.company.com"},
		{"chain to unknown host", map[string]string{"a": "b", "b": "unknown.example.org"}, "a", "unknown.example.org"},
		{"empty value stops chain", map[string]string{"a": "b", "b": ""}, "a", "local/b"},
		{"self reference", map[string]string{"a": "a"}, "a", "local/a"},
		{"two element cycle", map[string]string{"a": "b", "b": "a"}, "a", "local/a"},
		{"three element cycle", map[string]string{"a": "b", "b": "c", "c": "a"}, "a", "local/a"},
		{"cycle reached after prefix", map[string]string{"a": "b", "b": "c", "c": "b"}, "a", "local/b"},
	}
	set := chainFixture()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := set.Resolve(c.token, c.custom); got != c.want {
				t.Fatalf("Resolve(%q) = %q, want %q", c.token, got, c.want)
			}
		})
	}
}

func TestMatchByMultiLevelCustomAlias(t *testing.T) {
	custom := map[string]string{
		"pay":        "companypay",
		"companypay": "work",
		"work":       "gitlab.companypay.link:2222",
		"corp":       "eu",
		"eu":         "git.eu.dev.corp",
	}
	paths := []string{"grp/api", "grp/deep", "grp/pay", "grp/web"}
	got := chainFixture().Match(paths, []string{"pay,corp"}, custom, nil)
	if !reflect.DeepEqual(got, []string{"grp/deep", "grp/pay"}) {
		t.Fatalf("got %v", got)
	}
}

func TestResolveStopsOnAliasCycle(t *testing.T) {
	custom := map[string]string{"a": "b", "b": "a"}
	if got := fixture().Resolve("a", custom); got != "local/a" {
		t.Fatalf("got %q", got)
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
