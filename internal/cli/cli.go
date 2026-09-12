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
		"cancel":         commands.Cancel,
		"new":            commands.New,
		"init":           commands.Init,
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
		"unstash", "switch", "clear-stash", "cancel", "new", "prune", "default-branch"}
	glURLCommands = []string{"list", "clone", "pull", "update", "switch",
		"cancel", "new", "features", "default-branch"}
)

var positionalArgs = map[string]string{
	"new":    "<feat-name>",
	"switch": "[branch]",
	"cancel": "<branch>",
	"do":     "<command>...",
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
	case a.Command == "update" || a.Command == "switch" || a.Command == "cancel":
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
		fs.BoolVar(&a.Yes, "yes", false, "auto-confirm")
		fs.BoolVar(&a.Yes, "y", false, "auto-confirm")
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
	case "cancel":
		jobs()
		repo()
		dryRun()
	case "new":
		jobs()
		repo()
		dryRun()
	case "init":
	}

	switch cmd {
	case "switch", "cancel", "new":

		pos, err := parseIntermixed(fs, argv[1:])
		if err != nil {
			return nil, err
		}
		if len(pos) < 1 {
			if cmd != "switch" {
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
		[]string{"features", "new", "switch", "cancel"},
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
		[]string{"init", "help"},
	},
}

var commandSummaries = map[string]string{
	"features":       "list feature branches across repos",
	"new":            "create a feature branch off the default branch in every repo (local only)",
	"switch":         "switch every repo to a branch (default branch when omitted)",
	"cancel":         "delete a feature branch and return to the default branch",
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
