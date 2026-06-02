package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"mono-vcs/internal/app"
)

const (
	PerPage           = 100
	DefaultMainBranch = "main"
)

var MainBranchChoices = []string{"main", "master"}

type Config struct {
	GLURL      string `json:"gl-url,omitempty"`
	Jobs       int    `json:"jobs,omitempty"`
	MainBranch string `json:"main-branch,omitempty"`
}

var pathOverride string

func SetPath(p string) { pathOverride = p }

func Path() string {
	if pathOverride != "" {
		return pathOverride
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = ""
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "mono-vcs")
}

func Load() (Config, error) {
	var c Config
	p := Path()
	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return c, nil
		}
		return c, fmt.Errorf("failed to read %s: %v", p, err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return c, nil
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return c, fmt.Errorf("failed to parse %s: %v", p, err)
	}
	return c, nil
}

func Save(c Config) error {
	p := Path()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(p, data, 0o644)
}

func ApplyDefaults(a *app.Args) error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	if a.GLURL == nil {
		v := cfg.GLURL
		a.GLURL = &v
	}
	if a.Jobs == nil {
		j := cfg.Jobs
		if j < 1 {
			j = 1
		}
		a.Jobs = &j
	}
	if a.MainBranch == nil {
		v := cfg.MainBranch
		if v == "" {
			v = DefaultMainBranch
		}
		a.MainBranch = &v
	}
	return nil
}
