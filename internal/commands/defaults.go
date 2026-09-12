package commands

import (
	"errors"
	"fmt"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/config"
	"github.com/dimkarp93/mono-vcs/internal/gitlab"
	"github.com/dimkarp93/mono-vcs/internal/state"
)

const unresolvedDefaultBranch = "the default branch is unknown — no entry in the state db and GitLab reports no project — run `mono-vcs list` to refresh"

type defaultBranches struct {
	byPath    map[string]string
	missing   []string
	refreshed int
}

func (d defaultBranches) get(path string) (string, bool) {
	b, ok := d.byPath[path]
	return b, ok && b != ""
}

func gitlabToken(ctx *app.Context) (string, error) {
	if ctx.Args.GLToken != "" {
		return ctx.Args.GLToken, nil
	}
	if ctx.TokenFunc == nil {
		return "", errors.New("a GitLab token is required to resolve the default branch")
	}
	tok, err := ctx.TokenFunc(false)
	if err != nil {
		return "", err
	}
	ctx.Args.GLToken = tok
	return tok, nil
}

func fetchProjectsByPath(ctx *app.Context) (map[string]gitlab.Project, error) {
	if ctx.Args.GetGLURL() == "" {
		return nil, fmt.Errorf("gl-url is not configured; set it via `mono-vcs init` (file: %s)", config.Path())
	}
	token, err := gitlabToken(ctx)
	if err != nil {
		return nil, err
	}
	projects, err := gitlab.FetchProjects(ctx.Args.GetGLURL(), token)
	if err != nil {
		return nil, err
	}
	return projectsByPath(projects), nil
}

func projectsByPath(projects []gitlab.Project) map[string]gitlab.Project {
	byPath := make(map[string]gitlab.Project, len(projects))
	for _, p := range projects {
		byPath[p.PathWithNamespace] = p
	}
	return byPath
}

func resolveDefaults(ctx *app.Context, local []string, fetched map[string]gitlab.Project) defaultBranches {
	res := defaultBranches{byPath: make(map[string]string, len(local))}
	st, err := state.Open(ctx.Args.GetDBPath())
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "warning: %s\n", err.Error())
	}

	var unknown []string
	for _, p := range local {
		if pr, ok := fetched[p]; ok && pr.DefaultBranch != "" {
			if st.SetDefaultBranch(p, pr.DefaultBranch) {
				res.refreshed++
			}
			res.byPath[p] = pr.DefaultBranch
			continue
		}
		if b, ok := st.DefaultBranch(p); ok {
			res.byPath[p] = b
			continue
		}
		if fetched != nil {
			res.missing = append(res.missing, p)
			continue
		}
		unknown = append(unknown, p)
	}

	if len(unknown) > 0 {
		byPath, ferr := fetchProjectsByPath(ctx)
		if ferr != nil {
			fmt.Fprintf(ctx.Stderr, "warning: %s\n", ferr.Error())
		}
		for _, p := range unknown {
			pr, ok := byPath[p]
			if !ok || pr.DefaultBranch == "" {
				res.missing = append(res.missing, p)
				continue
			}
			if st.SetDefaultBranch(p, pr.DefaultBranch) {
				res.refreshed++
			}
			res.byPath[p] = pr.DefaultBranch
		}
	}

	if err := st.Flush(); err != nil {
		fmt.Fprintf(ctx.Stderr, "warning: %s\n", err.Error())
	}
	return res
}
