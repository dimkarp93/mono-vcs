package app

import "io"

type Args struct {
	Command string

	// Tri-state: nil means "not set on the CLI", so config.ApplyDefaults can
	// fill it from ~/.config/vcs-go (the equivalent of argparse's default=None).
	GLURL      *string
	Jobs       *int
	NoColor    *bool
	MainBranch *string

	GLToken string

	Repo    []string
	Feature string // -feat / -f; empty = not set (branch names are never empty)
	DryRun  bool
	Yes     bool
	Branch  string
	Action  []string

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

func (a *Args) GetJobs() int {
	if a.Jobs == nil || *a.Jobs < 1 {
		return 1
	}
	return *a.Jobs
}

func (a *Args) GetNoColor() bool {
	return a.NoColor != nil && *a.NoColor
}

func (a *Args) GetMainBranch() string {
	if a.MainBranch == nil {
		return ""
	}
	return *a.MainBranch
}

type Context struct {
	Args   *Args
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}
