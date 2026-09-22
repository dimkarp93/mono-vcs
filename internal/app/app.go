package app

import (
	"io"
	"strings"
)

type Args struct {
	Command string

	GLURL       *string
	Jobs        *int
	DBPath      *string
	AliasesPath *string

	GLToken string

	Repo    []string
	Remote  []string
	Feature string
	DryRun  bool
	Yes     bool
	Delete  bool
	Branch  string
	Action  []string
	Title   string

	All        bool
	Changed    bool
	Dirty      bool
	Local      bool
	Features   bool
	NonOrigin  bool
	OnlyOrigin bool
}

func (a *Args) GetGLURL() string {
	if a.GLURL == nil {
		return ""
	}
	return *a.GLURL
}

func (a *Args) GetDBPath() string {
	if a.DBPath == nil {
		return ""
	}
	return *a.DBPath
}

func (a *Args) GetAliasesPath() string {
	if a.AliasesPath == nil {
		return ""
	}
	return *a.AliasesPath
}

func (a *Args) GetJobs() int {
	if a.Jobs == nil || *a.Jobs < 1 {
		return 1
	}
	return *a.Jobs
}

type Context struct {
	Args      *Args
	Stdin     io.Reader
	Stdout    io.Writer
	Stderr    io.Writer
	TokenFunc func(optional bool) (string, error)
}

func (a *Args) ShellAction() string {
	if len(a.Action) == 0 {
		return ""
	}
	if len(a.Action) == 1 {
		return a.Action[0]
	}
	parts := make([]string, len(a.Action))
	for i, arg := range a.Action {
		parts[i] = shellQuote(arg)
	}
	return strings.Join(parts, " ")
}

func shellQuote(s string) string {
	if s != "" && !strings.ContainsAny(s, " \t\n\r\"'\\$`&|;<>()*?[]{}#~!") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
