package commands

import (
	"fmt"
	"sort"

	"github.com/dimkarp93/mono-vcs/internal/aliases"
	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/colors"
	"github.com/dimkarp93/mono-vcs/internal/output"
	"github.com/dimkarp93/mono-vcs/internal/remotes"
	"github.com/dimkarp93/mono-vcs/internal/repos"
	"github.com/dimkarp93/mono-vcs/internal/selector"
)

type aliasKind struct {
	name     string
	resolve  func(ctx *app.Context, target string) (string, error)
	system   func(ctx *app.Context) []output.AliasRow
	takenBy  func(ctx *app.Context, name string) bool
	describe string
}

func aliasKinds() map[string]aliasKind {
	return map[string]aliasKind{
		aliases.KindRemote: {
			name:     aliases.KindRemote,
			resolve:  resolveRemoteTarget,
			system:   systemRemoteRows,
			takenBy:  remoteAliasTaken,
			describe: "git remotes, matched by host or full URL",
		},
	}
}

func Alias(ctx *app.Context) int {
	args := ctx.Args.Action
	kinds := aliasKinds()

	if len(args) == 0 || args[0] == "ls" {
		var only string
		if len(args) > 1 {
			only = args[1]
		}
		if only != "" && !aliases.KnownKind(only) {
			output.Die(ctx.Stderr, unknownKind(only))
			return 2
		}
		return aliasList(ctx, kinds, only)
	}

	kind, ok := kinds[args[0]]
	if !ok {
		output.Die(ctx.Stderr, unknownKind(args[0]))
		return 2
	}
	rest := args[1:]

	if ctx.Args.Delete {
		if len(rest) != 1 {
			output.Die(ctx.Stderr, fmt.Sprintf("usage: mono-vcs alias %s -d <name>", kind.name))
			return 2
		}
		return aliasDelete(ctx, kind, rest[0])
	}
	switch len(rest) {
	case 0:
		return aliasList(ctx, kinds, kind.name)
	case 2:
		return aliasSet(ctx, kind, rest[0], rest[1])
	default:
		output.Die(ctx.Stderr, fmt.Sprintf("usage: mono-vcs alias %s <name> <target>", kind.name))
		return 2
	}
}

func unknownKind(kind string) string {
	return fmt.Sprintf("unknown alias kind: %s (supported: %v)", kind, aliases.Kinds)
}

func aliasSet(ctx *app.Context, kind aliasKind, name, target string) int {
	if err := aliases.ValidName(name); err != nil {
		output.Die(ctx.Stderr, err.Error())
		return 2
	}
	if kind.takenBy(ctx, name) {
		output.Die(ctx.Stderr, fmt.Sprintf("%q is already a system %s alias; pick another name", name, kind.name))
		return 2
	}
	value, err := kind.resolve(ctx, target)
	if err != nil {
		output.Die(ctx.Stderr, err.Error())
		return 1
	}

	st, oerr := aliases.Open(ctx.Args.GetAliasesPath())
	if oerr != nil {
		fmt.Fprintf(ctx.Stderr, "warning: %s\n", oerr.Error())
	}
	previous, existed := st.Get(kind.name, name)
	st.Set(kind.name, name, value)
	if err := st.Flush(); err != nil {
		output.Die(ctx.Stderr, err.Error())
		return 1
	}
	if existed && previous != value {
		fmt.Fprintf(ctx.Stdout, "%s alias %s: %s -> %s\n", kind.name, name, previous, value)
		return 0
	}
	fmt.Fprintf(ctx.Stdout, "%s alias %s -> %s\n", kind.name, name, value)
	return 0
}

func aliasDelete(ctx *app.Context, kind aliasKind, name string) int {
	st, oerr := aliases.Open(ctx.Args.GetAliasesPath())
	if oerr != nil {
		fmt.Fprintf(ctx.Stderr, "warning: %s\n", oerr.Error())
	}
	if !st.Delete(kind.name, name) {
		output.Die(ctx.Stderr, fmt.Sprintf("no custom %s alias named %q", kind.name, name))
		return 1
	}
	if err := st.Flush(); err != nil {
		output.Die(ctx.Stderr, err.Error())
		return 1
	}
	fmt.Fprintf(ctx.Stdout, "removed %s alias %s\n", kind.name, name)
	return 0
}

func aliasList(ctx *app.Context, kinds map[string]aliasKind, only string) int {
	st, oerr := aliases.Open(ctx.Args.GetAliasesPath())
	if oerr != nil {
		fmt.Fprintf(ctx.Stderr, "warning: %s\n", oerr.Error())
	}
	useColor := colors.Enabled()

	names := make([]string, 0, len(kinds))
	for n := range kinds {
		if only == "" || only == n {
			names = append(names, n)
		}
	}
	sort.Strings(names)

	for i, n := range names {
		if i > 0 {
			fmt.Fprintln(ctx.Stdout)
		}
		kind := kinds[n]
		rows := kind.system(ctx)
		for _, custom := range st.Names(n) {
			v, _ := st.Get(n, custom)
			rows = append(rows, output.AliasRow{Alias: custom, Type: "custom", Value: v})
		}
		output.PrintAliasTable(ctx.Stdout, kind.name, kind.describe, rows, useColor)
	}
	fmt.Fprintf(ctx.Stdout, "\ncustom aliases are stored in %s\n", ctx.Args.GetAliasesPath())
	return 0
}

func remoteSet(ctx *app.Context) *remotes.Set {
	local := repos.SortedKeys(repos.ScanLocalRepos("."))
	return remotes.New(local, ctx.Args.GetJobs(), ctx.Stderr)
}

func systemRemoteRows(ctx *app.Context) []output.AliasRow {
	set := remoteSet(ctx)
	var rows []output.AliasRow
	for _, h := range set.Hosts() {
		rows = append(rows, output.AliasRow{Alias: set.Alias(h), Type: "system", Value: h})
	}
	return rows
}

func remoteAliasTaken(ctx *app.Context, name string) bool {
	_, ok := remoteSet(ctx).HostByAlias(name)
	return ok
}

func resolveRemoteTarget(ctx *app.Context, target string) (string, error) {
	set := remoteSet(ctx)
	value := set.Resolve(target, selector.CustomAliases(ctx))
	if value == "" {
		return "", fmt.Errorf("cannot resolve %q to a remote host or URL", target)
	}
	return value, nil
}
