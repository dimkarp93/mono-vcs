package commands

import (
	"fmt"
	"strconv"
	"strings"

	"mono-vcs/internal/app"
	"mono-vcs/internal/config"
	"mono-vcs/internal/output"
	"mono-vcs/internal/prompts"
)

func Init(ctx *app.Context) int {
	out := ctx.Stdout

	// init's job is to (re)write the config, so an unparseable existing file
	// is non-fatal here.
	existing, err := config.Load()
	if err != nil {
		existing = config.Config{}
	}
	if existing != (config.Config{}) {
		fmt.Fprintf(out, "Updating %s\n", config.Path())
	} else {
		fmt.Fprintf(out, "Creating %s\n", config.Path())
	}
	fmt.Fprint(out, "(press Enter to keep the current value shown in [brackets])\n\n")

	pr := prompts.New(ctx.Stdin, ctx.Stdout, ctx.Stderr)

	glURL, err := pr.Free("GitLab URL", existing.GLURL, true)
	if err != nil {
		output.Die(ctx.Stderr, err.Error())
		return 1
	}

	jobsCurrent := "1"
	if existing.Jobs > 0 {
		jobsCurrent = strconv.Itoa(existing.Jobs)
	}
	jobsRaw, err := pr.Free("Parallel jobs", jobsCurrent, false)
	if err != nil {
		output.Die(ctx.Stderr, err.Error())
		return 1
	}
	jobs, jerr := strconv.Atoi(jobsRaw)
	if jerr != nil || jobs < 1 {
		output.Die(ctx.Stderr, fmt.Sprintf("jobs must be a positive integer, got %q", jobsRaw))
		return 1
	}

	ncCurrent := "false"
	if existing.NoColor {
		ncCurrent = "true"
	}
	ncRaw, err := pr.Free("Disable ANSI colors (true/false)", ncCurrent, false)
	if err != nil {
		output.Die(ctx.Stderr, err.Error())
		return 1
	}
	noColor := config.TrueValues[strings.ToLower(ncRaw)]

	mainBranch, err := pr.Choice("Main branch name", existing.MainBranch, config.MainBranchChoices, config.DefaultMainBranch)
	if err != nil {
		output.Die(ctx.Stderr, err.Error())
		return 1
	}

	if err := config.Save(config.Config{
		GLURL:      glURL,
		Jobs:       jobs,
		NoColor:    noColor,
		MainBranch: mainBranch,
	}); err != nil {
		output.Die(ctx.Stderr, err.Error())
		return 1
	}
	fmt.Fprintf(out, "\nwrote %s\n", config.Path())
	fmt.Fprintln(out, "note: --gl-token is intentionally not stored; pass it per-invocation.")
	return 0
}
