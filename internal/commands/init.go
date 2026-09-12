package commands

import (
	"fmt"
	"strconv"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/config"
	"github.com/dimkarp93/mono-vcs/internal/output"
	"github.com/dimkarp93/mono-vcs/internal/prompts"
	"github.com/dimkarp93/mono-vcs/internal/state"
)

func Init(ctx *app.Context) int {
	out := ctx.Stdout

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

	dbCurrent := existing.DBPath
	if dbCurrent == "" {
		dbCurrent = state.DefaultPath()
	}
	dbPath, err := pr.Free("State db path", dbCurrent, true)
	if err != nil {
		output.Die(ctx.Stderr, err.Error())
		return 1
	}

	if err := config.Save(config.Config{
		GLURL:  glURL,
		Jobs:   jobs,
		DBPath: dbPath,
	}); err != nil {
		output.Die(ctx.Stderr, err.Error())
		return 1
	}
	fmt.Fprintf(out, "\nwrote %s\n", config.Path())
	fmt.Fprintln(out, "note: --gl-token is intentionally not stored; pass it per-invocation.")
	return 0
}
