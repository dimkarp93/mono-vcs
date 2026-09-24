package remotes

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	"github.com/dimkarp93/mono-vcs/internal/gitops"
)

const LocalHost = "local"

type Remote struct {
	Name string
	URL  string
	Host string
	Key  string
}

func Make(name, url string) Remote {
	host, key := Normalize(url)
	return Remote{Name: name, URL: url, Host: host, Key: key}
}

func Normalize(raw string) (string, string) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", ""
	}
	s = strings.TrimSuffix(s, "/")
	s = strings.TrimSuffix(s, ".git")
	s = strings.TrimSuffix(s, "/")
	if s == "" {
		return "", ""
	}

	if scheme, rest, ok := strings.Cut(s, "://"); ok {
		if strings.EqualFold(scheme, "file") {
			return LocalHost, localKey(rest)
		}
		hostPart, path, _ := strings.Cut(rest, "/")
		return hostAndKey(hostPart, path)
	}
	if strings.HasPrefix(s, "/") || strings.HasPrefix(s, ".") || strings.HasPrefix(s, "~") {
		return LocalHost, localKey(s)
	}
	if i := strings.Index(s, ":"); i > 0 && !strings.Contains(s[:i], "/") {
		if _, port := splitPort(s); port != "" {
			return hostAndKey(s, "")
		}
		return hostAndKey(s[:i], s[i+1:])
	}
	if head, path, _ := strings.Cut(s, "/"); looksLikeHost(head) {
		return hostAndKey(head, path)
	}
	return LocalHost, localKey(s)
}

func looksLikeHost(head string) bool {
	if head == "" {
		return false
	}
	if h, _ := splitPort(head); h != head {
		return true
	}
	return strings.Contains(head, ".") || strings.EqualFold(head, "localhost")
}

func hostAndKey(hostPart, path string) (string, string) {
	if i := strings.LastIndex(hostPart, "@"); i >= 0 {
		hostPart = hostPart[i+1:]
	}
	host := strings.ToLower(strings.Trim(hostPart, "/"))
	if h, port := splitPort(host); port == "22" || port == "80" || port == "443" {
		host = h
	}
	if host == "" {
		return LocalHost, localKey(path)
	}
	path = strings.Trim(path, "/")
	if path == "" {
		return host, host
	}
	return host, host + "/" + path
}

func splitPort(host string) (string, string) {
	i := strings.LastIndex(host, ":")
	if i < 0 {
		return host, ""
	}
	port := host[i+1:]
	if port == "" {
		return host, ""
	}
	for _, r := range port {
		if r < '0' || r > '9' {
			return host, ""
		}
	}
	return host[:i], port
}

func localKey(path string) string {
	path = strings.Trim(path, "/")
	if path == "" {
		return LocalHost
	}
	return LocalHost + "/" + path
}

func SystemAlias(host string) string {
	h, _ := splitPort(host)
	if h == "" {
		h = host
	}
	labels := strings.Split(h, ".")
	if len(labels) < 2 {
		return h
	}
	if isIPv4(labels) {
		return h
	}
	return strings.Join(labels[:len(labels)-1], ".")
}

func isIPv4(labels []string) bool {
	if len(labels) != 4 {
		return false
	}
	for _, l := range labels {
		if l == "" {
			return false
		}
		for _, r := range l {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

func AssignAliases(hosts []string) map[string]string {
	byAlias := map[string][]string{}
	for _, h := range hosts {
		a := SystemAlias(h)
		byAlias[a] = append(byAlias[a], h)
	}
	out := make(map[string]string, len(hosts))
	for _, h := range hosts {
		a := SystemAlias(h)
		if len(byAlias[a]) > 1 {
			out[h] = h
			continue
		}
		out[h] = a
	}
	return out
}

func Origin(rs []Remote) (Remote, bool) {
	for _, r := range rs {
		if r.Name == "origin" {
			return r, true
		}
	}
	if len(rs) == 0 {
		return Remote{}, false
	}
	best := rs[0]
	for _, r := range rs[1:] {
		if r.Name < best.Name {
			best = r
		}
	}
	return best, true
}

type Set struct {
	byRepo map[string][]Remote
	sys    map[string]string
}

func New(paths []string, jobs int, stderr io.Writer) *Set {
	s := &Set{byRepo: map[string][]Remote{}}
	if jobs < 1 {
		jobs = 1
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, jobs)
	for _, p := range paths {
		wg.Add(1)
		sem <- struct{}{}
		go func(p string) {
			defer wg.Done()
			defer func() { <-sem }()
			rs, err := gitops.Remotes(p)
			if err != nil {
				if stderr != nil {
					fmt.Fprintf(stderr, "warning: %s: failed to list remotes — %s\n", p, err.Error())
				}
				return
			}
			conv := make([]Remote, 0, len(rs))
			for _, r := range rs {
				conv = append(conv, Make(r.Name, r.URL))
			}
			mu.Lock()
			s.byRepo[p] = conv
			mu.Unlock()
		}(p)
	}
	wg.Wait()
	s.refresh()
	return s
}

func (s *Set) Add(path string, rs ...Remote) {
	if len(rs) == 0 {
		return
	}
	s.byRepo[path] = append(s.byRepo[path], rs...)
	s.refresh()
}

func (s *Set) refresh() {
	s.sys = AssignAliases(s.Hosts())
}

func (s *Set) Of(path string) []Remote { return s.byRepo[path] }

func (s *Set) Subset(paths []string) *Set {
	out := &Set{byRepo: map[string][]Remote{}, sys: s.sys}
	for _, p := range paths {
		if rs, ok := s.byRepo[p]; ok {
			out.byRepo[p] = rs
		}
	}
	return out
}

func (s *Set) Hosts() []string {
	seen := map[string]bool{}
	for _, rs := range s.byRepo {
		for _, r := range rs {
			if r.Host != "" {
				seen[r.Host] = true
			}
		}
	}
	out := make([]string, 0, len(seen))
	for h := range seen {
		out = append(out, h)
	}
	sort.Strings(out)
	return out
}

func (s *Set) Alias(host string) string {
	if a, ok := s.sys[host]; ok {
		return a
	}
	return SystemAlias(host)
}

func (s *Set) HostByAlias(alias string) (string, bool) {
	for h, a := range s.sys {
		if a == alias {
			return h, true
		}
	}
	return "", false
}

func (s *Set) OriginAlias(path string) string {
	r, ok := Origin(s.byRepo[path])
	if !ok || r.Host == "" {
		return ""
	}
	return s.Alias(r.Host)
}

func (s *Set) AliasesOf(paths []string) []string {
	seen := map[string]bool{}
	for _, p := range paths {
		if a := s.OriginAlias(p); a != "" {
			seen[a] = true
		}
	}
	out := make([]string, 0, len(seen))
	for a := range seen {
		out = append(out, a)
	}
	sort.Strings(out)
	return out
}

func Split(tokens []string) []string {
	var out []string
	for _, t := range tokens {
		for s := range strings.SplitSeq(t, ",") {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
	}
	return out
}

func (s *Set) Resolve(token string, custom map[string]string) string {
	seen := map[string]bool{}
	for !seen[token] {
		seen[token] = true
		if v, ok := custom[token]; ok && v != "" {
			token = v
			continue
		}
		if h, ok := s.HostByAlias(token); ok {
			return h
		}
		break
	}
	_, key := Normalize(token)
	return key
}

func (s *Set) Match(paths, tokens []string, custom map[string]string, stderr io.Writer) []string {
	items := Split(tokens)
	if len(items) == 0 {
		return paths
	}
	targets := make([]string, 0, len(items))
	for _, t := range items {
		targets = append(targets, s.Resolve(t, custom))
	}

	hit := make([]bool, len(targets))
	var out []string
	for _, p := range paths {
		matched := false
		for i, target := range targets {
			if target == "" {
				continue
			}
			for _, r := range s.byRepo[p] {
				if r.Host == target || r.Key == target {
					hit[i] = true
					matched = true
					break
				}
			}
		}
		if matched {
			out = append(out, p)
		}
	}

	var missing []string
	for i, ok := range hit {
		if !ok {
			missing = append(missing, items[i])
		}
	}
	if len(missing) > 0 && stderr != nil {
		fmt.Fprintf(stderr, "warning: -remote entries matched nothing: %s\n", strings.Join(missing, ", "))
	}
	return out
}

type LegendRow struct {
	Alias  string
	Host   string
	Value  string
	Custom bool
}

func (s *Set) Legend(custom map[string]string) []LegendRow {
	sample := map[string]string{}
	for _, rs := range s.byRepo {
		for _, r := range rs {
			if r.Host == "" {
				continue
			}
			if cur, ok := sample[r.Host]; !ok || r.URL < cur {
				sample[r.Host] = r.URL
			}
		}
	}
	var rows []LegendRow
	for _, h := range s.Hosts() {
		rows = append(rows, LegendRow{Alias: s.Alias(h), Host: h, Value: sample[h]})
	}
	names := make([]string, 0, len(custom))
	for n := range custom {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		host, _ := Normalize(custom[n])
		rows = append(rows, LegendRow{Alias: n, Host: host, Value: custom[n], Custom: true})
	}
	return rows
}
