package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"

	"mono-vcs/internal/app"
	"mono-vcs/internal/commands"
	"mono-vcs/internal/config"
	"mono-vcs/internal/output"
	"mono-vcs/internal/prompts"
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
		"list":          commands.List,
		"clone":         commands.Clone,
		"pull":          commands.Pull,
		"update-main":   commands.UpdateMain,
		"stash":         commands.Stash,
		"unstash":       commands.Unstash,
		"clear-stash":   commands.ClearStash,
		"history-stash": commands.HistoryStash,
		"prune":         commands.Prune,
		"features":      commands.Features,
		"do":            commands.Do,
		"switch":        commands.Switch,
		"cancel":        commands.Cancel,
		"new":           commands.New,
		"init":          commands.Init,
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
	jobsCommands = []string{"list", "clone", "pull", "update-main", "stash",
		"unstash", "switch", "clear-stash", "cancel", "new", "prune"}
	glURLCommands = []string{"list", "clone", "pull"}
)

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
		output.Die(r.Stderr, fmt.Sprintf("--gl-url is required (pass as flag or set it via `mono-vcs init`, file: %s)", config.Path()))
		return 1
	}
	if has(jobsCommands, a.Command) && a.Jobs != nil && *a.Jobs < 1 {
		output.Die(r.Stderr, "--jobs must be >= 1")
		return 1
	}

	// Token policy (mirrors cli.py): required for list/clone/pull (but not
	// pull --dry-run), optional for update-main/switch/cancel unless --dry-run,
	// not requested otherwise.
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
	case a.Command == "update-main" || a.Command == "switch" || a.Command == "cancel":
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
	return h(&app.Context{Args: a, Stdin: r.Stdin, Stdout: r.Stdout, Stderr: r.Stderr})
}

func (r *Runner) parse(argv []string) (*app.Args, error) {
	if len(argv) == 0 {
		return nil, errors.New("a subcommand is required")
	}
	cmd := argv[0]
	if cmd == "-h" || cmd == "--help" || cmd == "help" {
		r.printUsage()
		return nil, errHelp
	}
	if _, ok := r.Commands[cmd]; !ok {
		return nil, fmt.Errorf("unknown command: %s", cmd)
	}

	a := &app.Args{Command: cmd}
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(r.Stderr)

	glURL := func() { fs.Var(&strPtr{&a.GLURL}, "gl-url", "GitLab base URL (default: from ~/.config/mono-vcs)") }
	jobs := func() { fs.Var(&intPtr{&a.Jobs}, "jobs", "parallelism (default: from config, else 1)") }
	mainBranch := func() { fs.Var(&strPtr{&a.MainBranch}, "main-branch", "main branch name") }
	noColor := func() { fs.Var(&boolPtr{&a.NoColor}, "no-color", "disable ANSI colors") }
	repo := func() {
		fs.Var(&repoFlag{&a.Repo}, "repo", "restrict to these repos (repeatable; comma-separated; bare name matches by repo name, trailing / matches a folder, a path like group/repo matches that exact path)")
		fs.StringVar(&a.Feature, "feat", "", "restrict to repos that have a local branch with this name (mutually exclusive with -repo)")
		fs.StringVar(&a.Feature, "f", "", "shorthand for -feat")
	}
	dryRun := func() { fs.BoolVar(&a.DryRun, "dry-run", false, "print the git commands that would run") }

	switch cmd {
	case "list":
		glURL()
		jobs()
		mainBranch()
		noColor()
		repo()
		fs.BoolVar(&a.All, "all", false, "show every repo")
		fs.BoolVar(&a.Changed, "changed", false, "changed repos (default filter)")
		fs.BoolVar(&a.Dirty, "dirty", false, "repos with any uncommitted change")
		fs.BoolVar(&a.Local, "local", false, "repos with tracked changes only")
		fs.BoolVar(&a.Features, "features", false, "repos on a feature branch")
		fs.BoolVar(&a.NonOrigin, "non-origin", false, "repos that exist only locally")
		fs.BoolVar(&a.OnlyOrigin, "only-origin", false, "repos that exist only on GitLab")
	case "clone":
		glURL()
		jobs()
	case "pull":
		glURL()
		jobs()
		mainBranch()
		repo()
		dryRun()
	case "update-main":
		jobs()
		mainBranch()
		repo()
		dryRun()
	case "stash", "unstash", "clear-stash":
		jobs()
		repo()
		dryRun()
	case "history-stash":
		noColor()
		repo()
	case "prune":
		jobs()
		fs.BoolVar(&a.Yes, "yes", false, "auto-confirm")
		fs.BoolVar(&a.Yes, "y", false, "auto-confirm")
		noColor()
		repo()
		dryRun()
	case "features":
		mainBranch()
		noColor()
		repo()
	case "do":
		repo()
		dryRun()
	case "switch":
		jobs()
		mainBranch()
		repo()
		dryRun()
	case "cancel":
		jobs()
		mainBranch()
		repo()
		dryRun()
	case "new":
		jobs()
		mainBranch()
		repo()
		dryRun()
	case "init":
	}

	switch cmd {
	case "switch", "cancel", "new":
		// We allow flags after the positional branch; stdlib flag stops at the
		// first non-flag, so parse them intermixed.
		pos, err := parseIntermixed(fs, argv[1:])
		if err != nil {
			return nil, err
		}
		if len(pos) < 1 {
			return nil, errors.New("branch is required")
		}
		a.Branch = pos[0]
	case "do":
		// REMAINDER: flags must precede the action; everything after is the
		// action verbatim, flag-like tokens included.
		if err := fs.Parse(argv[1:]); err != nil {
			return nil, err
		}
		a.Action = fs.Args()
	default:
		if err := fs.Parse(argv[1:]); err != nil {
			return nil, err
		}
	}

	if a.MainBranch != nil && !has(config.MainBranchChoices, *a.MainBranch) {
		return nil, fmt.Errorf("--main-branch must be one of: %s", strings.Join(config.MainBranchChoices, ", "))
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

func (r *Runner) printUsage() {
	names := make([]string, 0, len(r.Commands))
	for n := range r.Commands {
		names = append(names, n)
	}
	sort.Strings(names)
	fmt.Fprintln(r.Stdout, "mono-vcs — bulk-sync local clones with a GitLab instance.")
	fmt.Fprintln(r.Stdout, "\nusage: mono-vcs <command> [options]")
	fmt.Fprintln(r.Stdout, "\ncommands:")
	for _, n := range names {
		fmt.Fprintf(r.Stdout, "  %s\n", n)
	}
	fmt.Fprintln(r.Stdout, "\nrun `mono-vcs <command> -h` for command options.")
}
