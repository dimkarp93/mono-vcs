package gitops

import (
	"bytes"
	"errors"
	"os/exec"
)

func HasMulti() bool {
	_, err := exec.LookPath("git-multi")
	return err == nil
}
func runMulti(args ...string) (stdout, stderr string, code int) {
	cmd := exec.Command("git-multi", args...)
	var o, e bytes.Buffer
	cmd.Stdout = &o
	cmd.Stderr = &e
	err := cmd.Run()
	if err == nil {
		return o.String(), e.String(), 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return o.String(), e.String(), ee.ExitCode()
	}
	return o.String(), e.String() + err.Error(), -1
}
func WorktreeBranches(path string) (map[string]string, error) {
	panic("not implemented")
}
func linkedBranches(porcelain string) map[string]string {
	panic("not implemented")
}
func worktreeBranch(block string) (string, string, bool) {
	panic("not implemented")
}
func MultiNewOne(path, branch string) Result {
	panic("not implemented")
}
func MultiSwitchIDE(path, branch string) Result {
	panic("not implemented")
}
