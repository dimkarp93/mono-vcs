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
	branch    string
	dirty     bool
	hasLocal  bool
	untracked bool
	conflict  bool
	when      string
	hash      string
}

func mainInSync(p gitlab.Project, localPath, glURL, token, branch string) (bool, bool) {
	remoteSHA := gitlab.FetchRemoteBranchSHA(glURL, token, p.ID, branch)
	localSHA := gitops.LocalBranchSHA(localPath, branch)
	if remoteSHA == "" || localSHA == "" {
		return false, false
	}
	return remoteSHA == localSHA, true
}

func mainStatus(p gitlab.Project, localPath, glURL, token, branch string) string {
	inSync, known := mainInSync(p, localPath, glURL, token, branch)
	if !known {
		return "unknown"
	}
	if inSync {
		return "synced"
	}
	if s := gitops.AheadBehind(localPath, token, branch); s != "unknown" {
		return s
	}
	return "behind"
}

func outOfSync(s string) bool {
	return s == "ahead" || s == "behind" || s == "diverged"
}

func List(ctx *app.Context) int {
	a := ctx.Args
	out := ctx.Stdout
	useColor := colors.Enabled()

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

	status := map[string]string{}
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
				s := mainStatus(projectsByPath[path], path, a.GetGLURL(), a.GLToken, branch)
				mu.Lock()
				status[path] = s
				mu.Unlock()
			}(path)
		}
		for path := range local {
			wg.Add(1)
			sem <- struct{}{}
			go func(path string) {
				defer wg.Done()
				defer func() { <-sem }()
				b, d, h, u, c := gitops.LocalInfo(path)
				when, hash := gitops.LastCommit(path)
				mu.Lock()
				info[path] = localInfo{b, d, h, u, c, when, hash}
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
				if outOfSync(status[path]) {
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
		var hasLocal, untracked, conflict bool
		var when, hash string
		if inLocal {
			ci := info[path]
			cur = ci.branch
			hasLocal = ci.hasLocal
			untracked = ci.untracked
			conflict = ci.conflict
			when = ci.when
			hash = ci.hash
		}
		onFeature := cur != "" && cur != branch
		var line string
		switch {
		case inRemote && inLocal:
			color := colors.Green
			if onFeature {
				color = colors.Blue
			} else {
				switch status[path] {
				case "behind":
					color = colors.Yellow
				case "ahead":
					color = colors.Orange
				case "diverged":
					color = colors.Brown
				}
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
			if when != "" {
				line += " " + colors.Colorize("("+when+")", colors.Orange, useColor)
			}
			if hash != "" {
				line += " " + colors.Colorize("<"+hash+">", colors.Blue, useColor)
			}
			if hasLocal {
				line += " " + colors.Colorize("✗", colors.Yellow, useColor)
			}
			if untracked {
				line += " " + colors.Colorize("●", colors.Gray, useColor)
			}
		}
		if conflict {
			line = colors.Colorize("[!]", colors.Red, useColor) + " " + line
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
	fmt.Fprintf(out, "  %s — local & remote, local `%s` is behind remote\n", colors.Colorize("yellow", colors.Yellow, useColor), branch)
	fmt.Fprintf(out, "  %s — local & remote, local `%s` is ahead of remote\n", colors.Colorize("orange", colors.Orange, useColor), branch)
	fmt.Fprintf(out, "  %s  — local & remote, local `%s` diverged — neither side fast-forwards the other\n", colors.Colorize("brown", colors.Brown, useColor), branch)
	fmt.Fprintf(out, "  %s   — local repo currently on a non-`%s` branch (a feature branch)\n", colors.Colorize("blue", colors.Blue, useColor), branch)
	fmt.Fprintf(out, "  %s    — local only\n", colors.Colorize("red", colors.Red, useColor))
	fmt.Fprintf(out, "  %s   — remote only\n", colors.Colorize("gray", colors.Gray, useColor))
	fmt.Fprintf(out, "  %s      — repo in a conflict state (unresolved merge/rebase)\n", colors.Colorize("[!]", colors.Red, useColor))
	fmt.Fprintf(out, "  %s — current branch of local repo\n", colors.Colorize("[branch]", colors.Red, useColor))
	fmt.Fprintf(out, "  %s — last commit time (local)\n", colors.Colorize("(YYYY-MM-DD HH:mm)", colors.Orange, useColor))
	fmt.Fprintf(out, "  %s — last commit hash (local)\n", colors.Colorize("<hash>", colors.Blue, useColor))
	fmt.Fprintf(out, "  %s        — uncommitted changes to tracked files\n", colors.Colorize("✗", colors.Yellow, useColor))
	fmt.Fprintf(out, "  %s        — untracked files in working tree\n", colors.Colorize("●", colors.Gray, useColor))
	return 0
}
