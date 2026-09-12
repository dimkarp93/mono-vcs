package app

import "io"

type Args struct {
	Command string

	GLURL  *string
	Jobs   *int
	DBPath *string

	GLToken string

	Repo    []string
	Feature string
	DryRun  bool
	Yes     bool
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
