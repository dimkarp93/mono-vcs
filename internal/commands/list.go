package commands

import (
	"fmt"
	"strings"
	"sync"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/colors"
	"github.com/dimkarp93/mono-vcs/internal/gitlab"
	"github.com/dimkarp93/mono-vcs/internal/gitops"
	"github.com/dimkarp93/mono-vcs/internal/output"
	"github.com/dimkarp93/mono-vcs/internal/repos"
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

func listColumns(paths []string, local map[string]bool, info map[string]localInfo, width int) (int, int) {
	pathW, branchW := 0, 0
	for _, path := range paths {
		if l := output.DisplayWidth(path); l > pathW {
			pathW = l
		}
		if !local[path] {
			continue
		}
		if b := info[path].branch; b != "" {
			if l := output.DisplayWidth(b) + 2; l > branchW {
				branchW = l
			}
		}
	}
	if limit := width / 3; branchW > limit {
		branchW = limit
	}
	if limit := width - branchW - 4; pathW > limit {
		pathW = limit
	}
	if pathW < 1 {
		pathW = 1
	}
	return pathW, branchW
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
	projectsByPath := projectsByPath(projects)
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
	jobs := a.GetJobs()

	inSync := map[string]int{}
	info := map[string]localInfo{}
	defaultOf := resolveDefaults(ctx, repos.SortedKeys(local), projectsByPath).byPath
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
				v, known := mainInSync(projectsByPath[path], path, a.GetGLURL(), a.GLToken, defaultOf[path])
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
		onFeature := ci.branch != "" && ci.branch != defaultOf[path]
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

	pathW, branchW := listColumns(paths, local, info, output.Width(out))

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
		onFeature := cur != "" && cur != defaultOf[path]
		color := colors.Gray
		switch {
		case inRemote && inLocal:
			color = colors.Green
			if onFeature {
				color = colors.Blue
			} else if v, ok := inSync[path]; ok && v == 0 {
				color = colors.Yellow
			}
		case inLocal:
			color = colors.Red
			if onFeature {
				color = colors.Blue
			}
		}
		name := output.Truncate(path, pathW)
		line := output.PadColored(name, colors.Colorize(name, color, useColor), pathW)
		if inLocal && branchW > 0 {
			cell := ""
			if cur != "" {
				b := output.Truncate("["+cur+"]", branchW)
				cell = output.PadColored(b, colors.Colorize(b, colors.Red, useColor), branchW)
			} else {
				cell = strings.Repeat(" ", branchW)
			}
			line += " " + cell
		}
		if inLocal && dirty {
			line += " " + colors.Colorize("✗", colors.Yellow, useColor)
		}
		fmt.Fprintln(out, strings.TrimRight(line, " "))
	}

	fmt.Fprintln(out)
	if len(both) > 0 && !haveGit {
		fmt.Fprintln(out, "(git not found in PATH — skipped the default-branch sync check)")
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
	fmt.Fprintf(out, "  %s  — local & remote, local default branch matches remote\n", colors.Colorize("green", colors.Green, useColor))
	fmt.Fprintf(out, "  %s — local & remote, local default branch differs from remote (likely behind)\n", colors.Colorize("yellow", colors.Yellow, useColor))
	fmt.Fprintf(out, "  %s   — local repo currently on a feature branch (not its default branch)\n", colors.Colorize("blue", colors.Blue, useColor))
	fmt.Fprintf(out, "  %s    — local only\n", colors.Colorize("red", colors.Red, useColor))
	fmt.Fprintf(out, "  %s   — remote only\n", colors.Colorize("gray", colors.Gray, useColor))
	fmt.Fprintf(out, "  %s — current branch of local repo\n", colors.Colorize("[branch]", colors.Red, useColor))
	fmt.Fprintf(out, "  %s        — uncommitted changes in working tree\n", colors.Colorize("✗", colors.Yellow, useColor))
	return 0
}
