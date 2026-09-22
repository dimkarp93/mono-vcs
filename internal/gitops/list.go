package gitops

import (
	"errors"
	"strings"
)

func LocalBranches(path string) ([]string, error) {
	out, errOut, rc := runGit("-C", path, "for-each-ref", "--format=%(refname:short)", "refs/heads/")
	if rc != 0 {
		return nil, errors.New(firstNonEmpty(errOut, out))
	}
	var bs []string
	for _, l := range strings.Split(out, "\n") {
		if s := strings.TrimSpace(l); s != "" {
			bs = append(bs, s)
		}
	}
	return bs, nil
}

func StashList(path string) ([]string, error) {
	out, errOut, rc := runGit("-C", path, "stash", "list")
	if rc != 0 {
		return nil, errors.New(firstNonEmpty(errOut, out))
	}
	var es []string
	for _, l := range strings.Split(out, "\n") {
		if strings.TrimSpace(l) != "" {
			es = append(es, l)
		}
	}
	return es, nil
}

type Remote struct {
	Name string
	URL  string
}

func Remotes(path string) ([]Remote, error) {
	out, errOut, rc := runGit("-C", path, "remote", "-v")
	if rc != 0 {
		return nil, errors.New(firstNonEmpty(errOut, out))
	}
	var rs []Remote
	seen := map[string]bool{}
	for _, l := range strings.Split(out, "\n") {
		fields := strings.Fields(l)
		if len(fields) < 2 || seen[fields[0]] {
			continue
		}
		seen[fields[0]] = true
		rs = append(rs, Remote{Name: fields[0], URL: fields[1]})
	}
	return rs, nil
}
