package app

import "io"

type Args struct {
	Command string

	GLURL      *string
	Jobs       *int
	MainBranch *string

	GLToken string

	Repo    []string
	Feature string
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
