package commands

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/dryrun"
	"github.com/dimkarp93/mono-vcs/internal/output"
)

func Do(ctx *app.Context) int {
	a := ctx.Args
	out := ctx.Stdout
	if len(a.Action) == 0 {
		output.Die(ctx.Stderr, "action is required, e.g.: mono-vcs do git status -s")
		return 1
	}
	action := strings.Join(a.Action, " ")
	local := selectLocal(ctx)
	if len(local) == 0 {
		fmt.Fprintln(out, "no local git repositories found under current directory")
		return 0
	}
	if a.DryRun {
		dryrun.Do(out, a, local)
		return 0
	}
	fmt.Fprintf(out, "running `%s` in %d repos (sequential, stdin/stdout passthrough)\n", action, len(local))
	failures, total := 0, len(local)
	for i, path := range local {
		fmt.Fprintf(out, "\n[%d/%d] :: %s\n", i+1, total, path)
		cmd := exec.Command("sh", "-c", action)
		cmd.Dir = path
		cmd.Stdin = ctx.Stdin
		cmd.Stdout = out
		cmd.Stderr = ctx.Stderr
		if err := cmd.Run(); err != nil {
			code := 1
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				code = ee.ExitCode()
			}
			failures++
			fmt.Fprintf(ctx.Stderr, "  exit %d\n", code)
		}
	}
	if failures > 0 {
		fmt.Fprintf(ctx.Stderr, "\n%d command(s) failed\n", failures)
		return 1
	}
	return 0
}
