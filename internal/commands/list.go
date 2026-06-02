package commands

import (
	"fmt"
	"strings"
	"sync"

	"mono-vcs/internal/app"
	"mono-vcs/internal/colors"
	"mono-vcs/internal/gitlab"
	"mono-vcs/internal/gitops"
	"mono-vcs/internal/output"
	"mono-vcs/internal/repos"
)

type localInfo struct {
	branch   string
	dirty    bool
	hasLocal bool
}

func mainInSync(p gitlab.Project, localPath, glURL, token, branch string) (bool, bool) {
	remoteSHA := gitlab.FetchRemoteBranchSHA(glURL, token, p.ID, branch)
	localSHA := gitops.LocalBranchSHA(localPath, branch)
	if remoteSHA == "" || localSHA == "" {
		return false, false
	}
	return remoteSHA == localSHA, true
}

func List(ctx *app.Context) int {
	a := ctx.Args
	out := ctx.Stdout
	useColor := colors.Enabled(a.GetNoColor())

	projects, err := gitlab.FetchProjects(a.GetGLURL(), a.GLToken)
	if err != nil {
		output.Die(ctx.Stderr, err.Error())
		return 1
	}
	allPaths := make([]string, 0, len(projects))
	for _, p := range projects {
		allPaths = append(allPaths, p.PathWithNamespace)
	}
	prefix := repos.CommonTopGroup(allPaths)
	projectsByPath := map[string]gitlab.Project{}
	for _, p := range projects {
		projectsByPath[repos.StripPrefix(p.PathWithNamespace, prefix)] = p
	}
	remote := map[string]bool{}
	for k := range projectsByPath {
		remote[k] = true
	}
	local := repos.ScanLocalRepos(".")

	if a.Feature != "" {
		allowed := toSet(reposWithFeature(repos.SortedKeys(local), a.Feature, ctx.Stderr))
		remote = intersect(remote, allowed)
		local = intersect(local, allowed)
	} else if len(a.Repo) > 0 {
		allowed := toSet(repos.FilterRepos(sortedSet(unionKeys(remote, local)), a.Repo, ctx.Stderr))
		remote = intersect(remote, allowed)
		local = intersect(local, allowed)
	}

	both := sortedSet(intersect(remote, local))
	haveGit := hasGit()
	branch := a.GetMainBranch()
	jobs := a.GetJobs()

	inSync := map[string]int{}
	info := map[string]localInfo{}
	if haveGit {
		var mu sync.Mutex
		sem := make(chan struct{}, jobs)
		var wg sync.WaitGroup
		for _, path := range both {
			wg.Add(1)
			sem <- struct{}{}
			go func(path string) {
				defer wg.Done()
				defer func() { <-sem }()
				v, known := mainInSync(projectsByPath[path], path, a.GetGLURL(), a.GLToken, branch)
				code := -1
				if known {
					if v {
						code = 1
					} else {
						code = 0
					}
				}
				mu.Lock()
				inSync[path] = code
				mu.Unlock()
			}(path)
		}
		for path := range local {
			wg.Add(1)
			sem <- struct{}{}
			go func(path string) {
				defer wg.Done()
				defer func() { <-sem }()
				b, d, h := gitops.LocalInfo(path)
				mu.Lock()
				info[path] = localInfo{b, d, h}
				mu.Unlock()
			}(path)
		}
		wg.Wait()
	}

	changed := a.Changed
	if !a.All && !(a.Changed || a.Dirty || a.Local || a.Features || a.NonOrigin || a.OnlyOrigin) {
		changed = true
	}

	matches := func(path string) bool {
		inRemote := remote[path]
		inLocal := local[path]
		ci := info[path]
		onFeature := ci.branch != "" && ci.branch != branch
		if a.Dirty && inLocal && ci.dirty {
			return true
		}
		if a.Local && inLocal && ci.hasLocal {
			return true
		}
		if a.Features && inLocal && onFeature {
			return true
		}
		if a.NonOrigin && inLocal && !inRemote {
			return true
		}
		if a.OnlyOrigin && inRemote && !inLocal {
			return true
		}
		if changed {
			if inRemote && !inLocal {
				return true
			}
			if inLocal && ci.dirty {
				return true
			}
			if inLocal && onFeature {
				return true
			}
			if inRemote && inLocal {
				if v, ok := inSync[path]; ok && v == 0 {
					return true
				}
			}
		}
		return false
	}

	paths := sortedSet(unionKeys(remote, local))
	if !a.All {
		var filtered []string
		for _, p := range paths {
			if matches(p) {
				filtered = append(filtered, p)
			}
		}
		paths = filtered
	}

	for _, path := range paths {
		inRemote := remote[path]
		inLocal := local[path]
		var cur string
		var dirty bool
		if inLocal {
			ci := info[path]
			cur = ci.branch
			dirty = ci.dirty
		}
		onFeature := cur != "" && cur != branch
		var line string
		switch {
		case inRemote && inLocal:
			color := colors.Green
			if onFeature {
				color = colors.Blue
			} else if v, ok := inSync[path]; ok && v == 0 {
				color = colors.Yellow
			}
			line = colors.Colorize(path, color, useColor)
		case inLocal:
			color := colors.Red
			if onFeature {
				color = colors.Blue
			}
			line = colors.Colorize(path, color, useColor)
		default:
			line = colors.Colorize(path, colors.Gray, useColor)
		}
		if inLocal {
			if cur != "" {
				line += " " + colors.Colorize("["+cur+"]", colors.Red, useColor)
			}
			if dirty {
				line += " " + colors.Colorize("✗", colors.Yellow, useColor)
			}
		}
		fmt.Fprintln(out, line)
	}

	fmt.Fprintln(out)
	if prefix != "" {
		fmt.Fprintf(out, "(stripped common top-level group: %s/)\n", prefix)
	}
	if len(both) > 0 && !haveGit {
		fmt.Fprintf(out, "(git not found in PATH — skipped `%s` sync check)\n", branch)
	}
	if !a.All {
		flags := []struct {
			name string
			on   bool
		}{
			{"--changed", changed},
			{"--dirty", a.Dirty},
			{"--local", a.Local},
			{"--features", a.Features},
			{"--non-origin", a.NonOrigin},
			{"--only-origin", a.OnlyOrigin},
		}
		var active []string
		for _, f := range flags {
			if f.on {
				active = append(active, f.name)
			}
		}
		fmt.Fprintf(out, "(active filter(s): %s; pass --all to see everything)\n", strings.Join(active, ", "))
	}
	fmt.Fprintln(out, "legend:")
	fmt.Fprintf(out, "  %s  — local & remote, local `%s` matches remote\n", colors.Colorize("green", colors.Green, useColor), branch)
	fmt.Fprintf(out, "  %s — local & remote, local `%s` differs from remote (likely behind)\n", colors.Colorize("yellow", colors.Yellow, useColor), branch)
	fmt.Fprintf(out, "  %s   — local repo currently on a non-`%s` branch (a feature branch)\n", colors.Colorize("blue", colors.Blue, useColor), branch)
	fmt.Fprintf(out, "  %s    — local only\n", colors.Colorize("red", colors.Red, useColor))
	fmt.Fprintf(out, "  %s   — remote only\n", colors.Colorize("gray", colors.Gray, useColor))
	fmt.Fprintf(out, "  %s — current branch of local repo\n", colors.Colorize("[branch]", colors.Red, useColor))
	fmt.Fprintf(out, "  %s        — uncommitted changes in working tree\n", colors.Colorize("✗", colors.Yellow, useColor))
	return 0
}
