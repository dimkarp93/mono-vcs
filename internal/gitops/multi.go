package gitops

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
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
	out, errOut, rc := runGit("-C", path, "worktree", "list", "--porcelain")
	if rc != 0 {
		return nil, errors.New(firstNonEmpty(errOut, out))
	}
	return linkedBranches(out), nil
}
func linkedBranches(porcelain string) map[string]string {
	branches := map[string]string{}
	for i, block := range strings.Split(strings.TrimSpace(porcelain), "\n\n") {
		if i == 0 {
			continue
		}
		if dir, branch, ok := worktreeBranch(block); ok {
			branches[branch] = dir
		}
	}
	return branches
}
func worktreeBranch(block string) (string, string, bool) {
	var dir, branch string
	for _, line := range strings.Split(block, "\n") {
		if v, ok := strings.CutPrefix(line, "worktree "); ok {
			dir = v
		}
		if v, ok := strings.CutPrefix(line, "branch refs/heads/"); ok {
			branch = v
		}
	}
	return dir, branch, branch != ""
}
func MultiNewOne(path, branch string) Result {
	panic("not implemented")
}
func MultiSwitchIDE(path, branch string) Result {
	panic("not implemented")
}
