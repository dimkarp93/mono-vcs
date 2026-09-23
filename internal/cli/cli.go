package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/commands"
	"github.com/dimkarp93/mono-vcs/internal/config"
	"github.com/dimkarp93/mono-vcs/internal/output"
	"github.com/dimkarp93/mono-vcs/internal/prompts"
)

type Handler func(*app.Context) int

type Runner struct {
	Stdin     io.Reader
	Stdout    io.Writer
	Stderr    io.Writer
	TokenFunc func(optional bool) (string, error)
	Commands  map[string]Handler
}

func New(stdin io.Reader, stdout, stderr io.Writer) *Runner {
	r := &Runner{Stdin: stdin, Stdout: stdout, Stderr: stderr}
	pr := prompts.New(stdin, stdout, stderr)
	r.TokenFunc = pr.Token
	r.Commands = DefaultCommands()
	return r
}

func DefaultCommands() map[string]Handler {
	return map[string]Handler{
		"list":           commands.List,
		"default-branch": commands.DefaultBranch,
		"clone":          commands.Clone,
		"pull":           commands.Pull,
		"update":         commands.Update,
		"stash":          commands.Stash,
		"unstash":        commands.Unstash,
		"clear-stash":    commands.ClearStash,
		"history-stash":  commands.HistoryStash,
		"prune":          commands.Prune,
		"features":       commands.Features,
		"do":             commands.Do,
		"switch":         commands.Switch,
		"finish":         commands.Finish,
		"done":           commands.Done,
		"new":            commands.New,
		"mr":             commands.MR,
		"init":           commands.Init,
		"alias":          commands.Alias,
	}
}

var errHelp = errors.New("help requested")

func has(set []string, s string) bool {
	for _, x := range set {
		if x == s {
			return true
		}
	}
	return false
}

var (
	jobsCommands = []string{"list", "clone", "pull", "update", "stash",
		"unstash", "switch", "clear-stash", "finish", "done", "new", "prune", "default-branch", "mr"}
	glURLCommands = []string{"list", "clone", "pull", "update", "switch",
		"finish", "done", "new", "features", "default-branch", "mr"}
)

var positionalArgs = map[string]string{
	"new":    "<feat-name>",
	"switch": "[branch]",
	"finish": "<branch>",
	"mr":     "[branch]",
	"do":     "<command>...",
	"alias":  "<kind> [name] [target] | ls [kind]",
}

func (r *Runner) Run(argv []string) int {
	a, err := r.parse(argv)
	if err != nil {
		if errors.Is(err, errHelp) || errors.Is(err, flag.ErrHelp) {
			return 0
		}
		output.Die(r.Stderr, err.Error())
		return 2
	}

	if a.Command != "init" {
		if err := config.ApplyDefaults(a); err != nil {
			output.Die(r.Stderr, err.Error())
			return 1
		}
	}
	if has(glURLCommands, a.Command) && a.GetGLURL() == "" {
		output.Die(r.Stderr, fmt.Sprintf("gl-url is not configured; set it via `mono-vcs init` (file: %s)", config.Path()))
		return 1
	}
	if has(jobsCommands, a.Command) && a.Jobs != nil && *a.Jobs < 1 {
		output.Die(r.Stderr, "--jobs must be >= 1")
		return 1
	}

	switch {
	case a.Command == "pull" && a.DryRun:
		a.GLToken = ""
	case a.Command == "list" || a.Command == "clone" || a.Command == "pull":
		tok, err := r.TokenFunc(false)
		if err != nil {
			output.Die(r.Stderr, err.Error())
			return 1
		}
		a.GLToken = tok
	case a.Command == "update" || a.Command == "switch" || a.Command == "finish" ||
		a.Command == "mr" || a.Command == "done":
		if a.DryRun {
			a.GLToken = ""
		} else {
			tok, err := r.TokenFunc(true)
			if err != nil {
				output.Die(r.Stderr, err.Error())
				return 1
			}
			a.GLToken = tok
		}
	}

	h := r.Commands[a.Command]
	return h(&app.Context{Args: a, Stdin: r.Stdin, Stdout: r.Stdout, Stderr: r.Stderr, TokenFunc: r.TokenFunc})
}

func (r *Runner) parse(argv []string) (*app.Args, error) {
	if len(argv) == 0 {
		return nil, errors.New("a subcommand is required")
	}
	cmd := argv[0]
	if cmd == "-h" || cmd == "-help" || cmd == "--help" || cmd == "help" {
		r.printUsage()
		return nil, errHelp
	}
	if _, ok := r.Commands[cmd]; !ok {
		return nil, fmt.Errorf("unknown command: %s", cmd)
	}

	a := &app.Args{Command: cmd}
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(r.Stderr)
	fs.Usage = func() {
		if pos := positionalArgs[cmd]; pos != "" {
			fmt.Fprintf(r.Stderr, "usage: mono-vcs %s [options] %s\n\noptions:\n", cmd, pos)
		} else {
			fmt.Fprintf(r.Stderr, "usage: mono-vcs %s [options]\n\noptions:\n", cmd)
		}
		fs.PrintDefaults()
	}

	jobs := func() { fs.Var(&intPtr{&a.Jobs}, "jobs", "parallelism (default: from config, else 1)") }
	repo := func() {
		fs.Var(&repoFlag{&a.Repo}, "repo", "restrict to these repos (repeatable; comma-separated; bare name matches by repo name, trailing / matches a folder, a path like group/repo matches that exact path)")
		fs.StringVar(&a.Feature, "feat", "", "restrict to repos that have a local branch with this name (mutually exclusive with -repo)")
		fs.StringVar(&a.Feature, "f", "", "shorthand for -feat")
		fs.Var(&repoFlag{&a.Remote}, "remote", "restrict to repos with a matching git remote (repeatable; comma-separated; full remote URL, host, system alias or custom alias)")
	}
	dryRun := func() { fs.BoolVar(&a.DryRun, "dry-run", false, "print the git commands that would run") }

	switch cmd {
	case "list":
		jobs()
		repo()
		fs.BoolVar(&a.All, "all", false, "show every repo")
		fs.BoolVar(&a.Changed, "changed", false, "changed repos (default filter)")
		fs.BoolVar(&a.Dirty, "dirty", false, "repos with any uncommitted change")
		fs.BoolVar(&a.Local, "local", false, "repos with tracked changes only")
		fs.BoolVar(&a.Features, "features", false, "repos on a feature branch")
		fs.BoolVar(&a.NonOrigin, "non-origin", false, "repos that exist only locally")
		fs.BoolVar(&a.OnlyOrigin, "only-origin", false, "repos that exist only on GitLab")
	case "clone":
		jobs()
	case "pull":
		jobs()
		repo()
		dryRun()
	case "update":
		jobs()
		repo()
		dryRun()
	case "stash", "unstash", "clear-stash":
		jobs()
		repo()
		dryRun()
	case "history-stash":
		repo()
	case "default-branch":
		jobs()
		repo()
	case "prune":
		jobs()
		repo()
		dryRun()
	case "features":
		repo()
	case "do":
		repo()
		dryRun()
	case "switch":
		jobs()
		repo()
		dryRun()
	case "finish":
		jobs()
		repo()
		dryRun()
	case "done":
		jobs()
		dryRun()
	case "new":
		jobs()
		repo()
		dryRun()
	case "mr":
		jobs()
		dryRun()
		fs.StringVar(&a.Title, "title", "", "merge request title (passed as merge_request.title push option)")
	case "alias":
		fs.BoolVar(&a.Delete, "delete", false, "delete the named alias")
		fs.BoolVar(&a.Delete, "d", false, "shorthand for -delete")
	case "init":
	}

	switch cmd {
	case "alias":

		pos, err := parseIntermixed(fs, argv[1:])
		if err != nil {
			return nil, err
		}
		a.Action = pos
	case "switch", "finish", "new", "mr":

		pos, err := parseIntermixed(fs, argv[1:])
		if err != nil {
			return nil, err
		}
		if len(pos) < 1 {
			if cmd != "switch" && cmd != "mr" {
				return nil, errors.New("branch is required")
			}
		} else {
			a.Branch = pos[0]
		}
	case "do":

		if err := fs.Parse(argv[1:]); err != nil {
			return nil, err
		}
		a.Action = fs.Args()
	default:
		if err := fs.Parse(argv[1:]); err != nil {
			return nil, err
		}
	}

	if a.Feature != "" && len(a.Repo) > 0 {
		return nil, errors.New("-repo and -feat are mutually exclusive; pass only one")
	}
	return a, nil
}

func parseIntermixed(fs *flag.FlagSet, args []string) ([]string, error) {
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		args = fs.Args()
		if len(args) == 0 {
			return pos, nil
		}
		pos = append(pos, args[0])
		args = args[1:]
	}
}

type commandGroup struct {
	title    string
	desc     string
	commands []string
}

var commandGroups = []commandGroup{
	{
		"Feature branches",
		"create, switch, inspect and drop feature branches across every repo",
		[]string{"features", "new", "switch", "mr", "done", "finish"},
	},
	{
		"Repositories",
		"clone and sync the raw git clones and their working trees",
		[]string{"clone", "pull", "update", "list", "default-branch", "stash", "unstash", "clear-stash", "history-stash", "prune"},
	},
	{
		"Shell delegation",
		"fan an arbitrary shell command out to every repo",
		[]string{"do"},
	},
	{
		"Utility",
		"configure mono-vcs and show help",
		[]string{"init", "alias", "help"},
	},
}

var commandSummaries = map[string]string{
	"features":       "list feature branches across repos",
	"new":            "switch every repo to a feature branch, creating it off the default branch and carrying local changes over",
	"switch":         "switch every repo to a branch (default branch when omitted)",
	"mr":             "push the feature branch everywhere it exists and open merge requests",
	"finish":         "discard local changes, return to the default branch and delete a feature branch",
	"done":           "delete every local feature branch already merged into the default branch",
	"clone":          "clone every GitLab project locally",
	"default-branch": "show the detected default branch of every repo",
	"pull":           "fast-forward the default branch against GitLab",
	"update":         "update the default branch and rebase feature branches onto it",
	"list":           "show the status of every repo",
	"stash":          "stash uncommitted changes in each repo",
	"unstash":        "pop the most recent stash in each repo",
	"clear-stash":    "drop stashed changes in each repo",
	"history-stash":  "show stash entries per repo",
	"prune":          "discard uncommitted working-tree changes",
	"do":             "run a shell command in every repo",
	"init":           "create or update the config file",
	"alias":          "define and list aliases (currently: remote)",
	"help":           "show this help",
}

func commandLabel(n string) string {
	if pos := positionalArgs[n]; pos != "" {
		return n + " " + pos
	}
	return n
}

func (r *Runner) printUsage() {
	fmt.Fprintln(r.Stdout, "mono-vcs — bulk-sync local clones with a GitLab instance.")
	fmt.Fprintln(r.Stdout, "\nusage: mono-vcs <command> [options]")
	width := 0
	for _, g := range commandGroups {
		for _, n := range g.commands {
			if w := len(commandLabel(n)); w > width {
				width = w
			}
		}
	}
	for _, g := range commandGroups {
		fmt.Fprintf(r.Stdout, "\n%s — %s\n", g.title, g.desc)
		for _, n := range g.commands {
			fmt.Fprintf(r.Stdout, "  %-*s  %s\n", width, commandLabel(n), commandSummaries[n])
		}
	}
	fmt.Fprintln(r.Stdout, "\nrun `mono-vcs <command> -h` for command options.")
}
