package cli

import "github.com/dimkarp93/install-libs/shellcomplete"

var jobsFlag = shellcomplete.Flag{Name: "-jobs"}
var dryRunFlag = shellcomplete.Flag{Name: "-dry-run", Bool: true}
var repoFlags = []shellcomplete.Flag{
	{Name: "-repo"},
	{Name: "-feat"},
	{Name: "-f"},
	{Name: "-remote"},
}

func flagsFor(names ...shellcomplete.Flag) []shellcomplete.Flag {
	return names
}

var completionSpec = shellcomplete.Spec{
	Bin: "mono-vcs",
	Flags: []shellcomplete.Flag{
		{Name: "-h", Bool: true},
		{Name: "-help", Bool: true},
		{Name: "--version", Bool: true},
		{Name: "--origin", Bool: true},
		{Name: "--buildinfo", Bool: true},
	},
	Commands: []shellcomplete.Command{
		{Name: "list", Flags: flagsFor(append([]shellcomplete.Flag{jobsFlag}, append(repoFlags,
			shellcomplete.Flag{Name: "-all", Bool: true},
			shellcomplete.Flag{Name: "-changed", Bool: true},
			shellcomplete.Flag{Name: "-dirty", Bool: true},
			shellcomplete.Flag{Name: "-local", Bool: true},
			shellcomplete.Flag{Name: "-features", Bool: true},
			shellcomplete.Flag{Name: "-non-origin", Bool: true},
			shellcomplete.Flag{Name: "-only-origin", Bool: true},
		)...)...)},
		{Name: "default-branch", Flags: flagsFor(append([]shellcomplete.Flag{jobsFlag}, repoFlags...)...)},
		{Name: "clone", Flags: flagsFor(jobsFlag)},
		{Name: "pull", Flags: flagsFor(append([]shellcomplete.Flag{jobsFlag, dryRunFlag}, repoFlags...)...)},
		{Name: "update", Flags: flagsFor(append([]shellcomplete.Flag{jobsFlag, dryRunFlag}, repoFlags...)...)},
		{Name: "stash", Flags: flagsFor(append([]shellcomplete.Flag{jobsFlag, dryRunFlag}, repoFlags...)...)},
		{Name: "unstash", Flags: flagsFor(append([]shellcomplete.Flag{jobsFlag, dryRunFlag}, repoFlags...)...)},
		{Name: "clear-stash", Flags: flagsFor(append([]shellcomplete.Flag{jobsFlag, dryRunFlag}, repoFlags...)...)},
		{Name: "history-stash", Flags: flagsFor(repoFlags...)},
		{Name: "prune", Flags: flagsFor(append([]shellcomplete.Flag{jobsFlag, dryRunFlag}, repoFlags...)...)},
		{Name: "features", Flags: flagsFor(repoFlags...)},
		{Name: "do", Flags: flagsFor(repoFlags...), Args: shellcomplete.Anything},
		{Name: "switch", Flags: flagsFor(append([]shellcomplete.Flag{jobsFlag, dryRunFlag}, repoFlags...)...), Args: shellcomplete.Anything, Intermixed: true},
		{Name: "finish", Flags: flagsFor(append([]shellcomplete.Flag{jobsFlag, dryRunFlag}, repoFlags...)...), Args: shellcomplete.Anything, Intermixed: true},
		{Name: "done", Flags: flagsFor(jobsFlag, dryRunFlag)},
		{Name: "new", Flags: flagsFor(append([]shellcomplete.Flag{jobsFlag, dryRunFlag}, repoFlags...)...), Args: shellcomplete.Anything, Intermixed: true},
		{Name: "mr", Flags: flagsFor(jobsFlag, dryRunFlag, shellcomplete.Flag{Name: "-title"}), Args: shellcomplete.Anything, Intermixed: true},
		{Name: "init"},
		{Name: "alias", Flags: flagsFor(
			shellcomplete.Flag{Name: "-delete", Bool: true},
			shellcomplete.Flag{Name: "-d", Bool: true},
		), Args: shellcomplete.Positional("remote", "ls"), Intermixed: true},
	},
}
