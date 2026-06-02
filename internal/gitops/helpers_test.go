package gitops_test

import (
	"os/exec"
	"strings"
)

func osRun(dir string, args ...string) error {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	return cmd.Run()
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }
