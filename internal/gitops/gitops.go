package gitops

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Result struct {
	Path   string
	Status string
	Detail string
}

func runGit(args ...string) (stdout, stderr string, code int) {
	cmd := exec.Command("git", args...)
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

func firstNonEmpty(a, b string) string {
	if s := strings.TrimSpace(a); s != "" {
		return s
	}
	return strings.TrimSpace(b)
}

func GitExtraHeaderArgs(token string) []string {
	if token == "" {
		return nil
	}
	return []string{"-c", "http.extraHeader=PRIVATE-TOKEN: " + token}
}

func IsDirty(path string) bool {
	out, _, rc := runGit("-C", path, "status", "--porcelain")
	return rc == 0 && strings.TrimSpace(out) != ""
}

func LocalInfo(path string) (branch string, dirty, hasLocal bool) {
	branch = CurrentBranch(path)
	out, _, rc := runGit("-C", path, "status", "--porcelain")
	if rc != 0 {
		return branch, false, false
	}
	var lines []string
	for _, l := range strings.Split(out, "\n") {
		if strings.TrimSpace(l) != "" {
			lines = append(lines, l)
		}
	}
	dirty = len(lines) > 0
	for _, l := range lines {
		if !strings.HasPrefix(l, "??") {
			hasLocal = true
			break
		}
	}
	return branch, dirty, hasLocal
}

func CurrentBranch(path string) string {
	out, _, rc := runGit("-C", path, "symbolic-ref", "--short", "-q", "HEAD")
	if rc != 0 {
		return ""
	}
	return strings.TrimSpace(out)
}

func RebaseInProgress(path string) bool {
	out, _, rc := runGit("-C", path, "rev-parse", "--git-dir")
	if rc != 0 {
		return false
	}
	gitDir := strings.TrimSpace(out)
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(path, gitDir)
	}
	for _, d := range []string{"rebase-merge", "rebase-apply"} {
		if fi, err := os.Stat(filepath.Join(gitDir, d)); err == nil && fi.IsDir() {
			return true
		}
	}
	return false
}

func HasBranch(path, branch string) bool {
	for _, ref := range []string{"refs/heads/" + branch, "refs/remotes/origin/" + branch} {
		if _, _, rc := runGit("-C", path, "rev-parse", "--verify", "--quiet", ref); rc == 0 {
			return true
		}
	}
	return false
}

func LocalBranchSHA(path, branch string) string {
	out, _, rc := runGit("-C", path, "rev-parse", "--verify", "refs/heads/"+branch)
	if rc != 0 {
		return ""
	}
	return strings.TrimSpace(out)
}

func StashCount(path string) int {
	out, _, rc := runGit("-C", path, "stash", "list")
	if rc != 0 {
		return 0
	}
	n := 0
	for _, l := range strings.Split(out, "\n") {
		if strings.TrimSpace(l) != "" {
			n++
		}
	}
	return n
}

func CloneOne(url, localTarget, token string, force bool) (string, bool, string) {
	if parent := filepath.Dir(localTarget); parent != "" {
		_ = os.MkdirAll(parent, 0o755)
	}
	if force {
		if _, err := os.Stat(localTarget); err == nil {
			if err := os.RemoveAll(localTarget); err != nil {
				return localTarget, false, fmt.Sprintf("failed to remove existing path: %v", err)
			}
		}
	}
	args := append([]string{}, GitExtraHeaderArgs(token)...)
	args = append(args, "clone", "--quiet", url, localTarget)
	out, errOut, rc := runGit(args...)
	if rc == 0 {
		return localTarget, true, ""
	}
	return localTarget, false, firstNonEmpty(errOut, out)
}

func PullOne(path, token string) Result {
	args := []string{"-C", path}
	args = append(args, GitExtraHeaderArgs(token)...)
	args = append(args, "pull", "--ff-only", "--quiet")
	out, errOut, rc := runGit(args...)
	if rc != 0 {
		return Result{path, "failed", firstNonEmpty(errOut, out)}
	}

	combined := strings.TrimSpace(out + errOut)
	if combined == "" {
		return Result{path, "up-to-date", ""}
	}
	return Result{path, "updated", combined}
}

func isAncestor(path, a, b string) bool {
	_, _, rc := runGit("-C", path, "merge-base", "--is-ancestor", a, b)
	return rc == 0
}

func AheadBehind(path, token, branch string) string {
	args := []string{"-C", path}
	args = append(args, GitExtraHeaderArgs(token)...)
	args = append(args, "fetch", "--quiet", "origin", branch)
	if _, _, rc := runGit(args...); rc != 0 {
		return "unknown"
	}
	local := LocalBranchSHA(path, branch)
	out, _, rc := runGit("-C", path, "rev-parse", "--verify", "--quiet", "FETCH_HEAD")
	if rc != 0 {
		return "unknown"
	}
	remote := strings.TrimSpace(out)
	if local == "" || remote == "" {
		return "unknown"
	}
	if local == remote {
		return "synced"
	}
	if isAncestor(path, remote, local) {
		return "ahead"
	}
	if isAncestor(path, local, remote) {
		return "behind"
	}
	return "diverged"
}

func UpdateMainOne(path, token, branch string) (Result, string) {
	if IsDirty(path) {
		return Result{path, "dirty", fmt.Sprintf("uncommitted changes — refusing to touch %s", branch)}, ""
	}
	current := CurrentBranch(path)
	orig := ""
	if current != "" && current != branch {
		orig = current
	}
	if current != branch {
		out, errOut, rc := runGit("-C", path, "checkout", branch)
		if rc != 0 {
			return Result{path, "failed", firstNonEmpty(errOut, out)}, ""
		}
	}
	args := []string{"-C", path}
	args = append(args, GitExtraHeaderArgs(token)...)
	args = append(args, "pull", "--ff-only", "--quiet")
	out, errOut, rc := runGit(args...)
	if rc != 0 {
		if orig != "" {
			runGit("-C", path, "checkout", orig)
		}
		return Result{path, "failed", firstNonEmpty(errOut, out)}, ""
	}
	combined := strings.TrimSpace(out + errOut)
	status := "up-to-date"
	if combined != "" {
		status = "updated"
	}
	return Result{path, status, combined}, orig
}

func RebaseOne(path, origBranch, branch string, stdin io.Reader, stdout, stderr io.Writer) (string, string) {
	out, errOut, rc := runGit("-C", path, "checkout", origBranch)
	if rc != 0 {
		return "failed", firstNonEmpty(errOut, out)
	}

	cmd := exec.Command("git", "-C", path, "rebase", branch)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if cmd.Run() == nil {
		return "rebased", fmt.Sprintf("%s rebased onto %s", origBranch, branch)
	}
	if RebaseInProgress(path) {
		return "conflict", fmt.Sprintf("%s: rebase onto %s hit conflicts", origBranch, branch)
	}
	return "failed", fmt.Sprintf("rebase of %s onto %s failed", origBranch, branch)
}

func StashOne(path string) Result {
	if _, _, rc := runGit("-C", path, "diff", "--quiet"); rc == 0 {
		return Result{path, "nothing", ""}
	}
	out, errOut, rc := runGit("-C", path, "stash", "push", "--keep-index",
		"--include-untracked", "-m", "mono-vcs stash")
	if rc != 0 {
		return Result{path, "failed", firstNonEmpty(errOut, out)}
	}
	return Result{path, "stashed", ""}
}

func UnstashOne(path string) Result {
	if StashCount(path) == 0 {
		return Result{path, "nothing", ""}
	}
	out, errOut, rc := runGit("-C", path, "stash", "pop")
	if rc != 0 {
		return Result{path, "failed", firstNonEmpty(errOut, out)}
	}
	return Result{path, "popped", ""}
}

func ClearStashOne(path string) Result {
	n := StashCount(path)
	if n == 0 {
		return Result{path, "nothing", ""}
	}
	out, errOut, rc := runGit("-C", path, "stash", "clear")
	if rc != 0 {
		return Result{path, "failed", firstNonEmpty(errOut, out)}
	}
	ent := "entries"
	if n == 1 {
		ent = "entry"
	}
	return Result{path, "cleared", fmt.Sprintf("dropped %d %s", n, ent)}
}

func PruneTargets(path string) ([]string, error) {
	out, errOut, rc := runGit("-C", path, "status", "--porcelain")
	if rc != 0 {
		return nil, errors.New(firstNonEmpty(errOut, out))
	}
	var targets []string
	for _, line := range strings.Split(out, "\n") {
		if len(line) < 4 {
			continue
		}

		code := strings.TrimSpace(line[:2])
		targets = append(targets, code+" "+line[3:])
	}
	return targets, nil
}

func PruneOne(path string) Result {
	targets, err := PruneTargets(path)
	if err != nil {
		return Result{path, "failed", err.Error()}
	}
	if len(targets) == 0 {
		return Result{path, "nothing", ""}
	}

	if out, errOut, rc := runGit("-C", path, "reset", "--hard", "HEAD"); rc != 0 {
		return Result{path, "failed", firstNonEmpty(errOut, out)}
	}
	if out, errOut, rc := runGit("-C", path, "clean", "-fd"); rc != 0 {
		return Result{path, "failed", firstNonEmpty(errOut, out)}
	}
	n := len(targets)
	item := "items"
	if n == 1 {
		item = "item"
	}
	return Result{path, "pruned", fmt.Sprintf("removed %d %s", n, item)}
}

func SwitchOne(path, branch, mainBranch, token string) Result {
	if IsDirty(path) {
		return Result{path, "dirty", "uncommitted changes"}
	}
	if HasBranch(path, branch) {
		if CurrentBranch(path) == branch {
			return Result{path, "already", ""}
		}
		out, errOut, rc := runGit("-C", path, "checkout", branch)
		if rc != 0 {
			return Result{path, "failed", firstNonEmpty(errOut, out)}
		}
		return Result{path, "switched", ""}
	}
	if branch == mainBranch || !HasBranch(path, mainBranch) {
		return Result{path, "absent", ""}
	}
	if CurrentBranch(path) != mainBranch {
		out, errOut, rc := runGit("-C", path, "checkout", mainBranch)
		if rc != 0 {
			return Result{path, "failed", fmt.Sprintf("checkout %s: %s", mainBranch, firstNonEmpty(errOut, out))}
		}
	}
	args := []string{"-C", path}
	args = append(args, GitExtraHeaderArgs(token)...)
	args = append(args, "pull", "--ff-only", "--quiet")
	out, errOut, rc := runGit(args...)
	if rc != 0 {
		return Result{path, "failed", fmt.Sprintf("pull %s: %s", mainBranch, firstNonEmpty(errOut, out))}
	}
	state := "up-to-date"
	if strings.TrimSpace(out+errOut) != "" {
		state = "updated"
	}
	return Result{path, "synced-main", fmt.Sprintf("on %s (%s)", mainBranch, state)}
}

func CancelOne(path, branch, mainBranch, token string) Result {
	if _, _, rc := runGit("-C", path, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch); rc != 0 {
		return Result{path, "absent", ""}
	}
	detail := ""
	if CurrentBranch(path) == branch {
		if IsDirty(path) {
			return Result{path, "dirty", fmt.Sprintf("uncommitted changes — cannot leave `%s` to delete it", branch)}
		}
		if !HasBranch(path, mainBranch) {
			return Result{path, "failed", fmt.Sprintf("no `%s` to fall back to", mainBranch)}
		}
		out, errOut, rc := runGit("-C", path, "checkout", mainBranch)
		if rc != 0 {
			return Result{path, "failed", fmt.Sprintf("checkout %s: %s", mainBranch, firstNonEmpty(errOut, out))}
		}
		args := []string{"-C", path}
		args = append(args, GitExtraHeaderArgs(token)...)
		args = append(args, "pull", "--ff-only", "--quiet")
		out2, errOut2, rc2 := runGit(args...)
		if rc2 != 0 {
			return Result{path, "failed", fmt.Sprintf("pull %s: %s", mainBranch, firstNonEmpty(errOut2, out2))}
		}
		state := "up-to-date"
		if strings.TrimSpace(out2+errOut2) != "" {
			state = "updated"
		}
		detail = fmt.Sprintf("switched to `%s` (%s), ", mainBranch, state)
	}
	out, errOut, rc := runGit("-C", path, "branch", "-D", branch)
	if rc != 0 {
		return Result{path, "failed", firstNonEmpty(errOut, out)}
	}
	return Result{path, "deleted", detail + fmt.Sprintf("removed `%s`", branch)}
}

func NewBranchOne(path, branch, mainBranch string) Result {
	if IsDirty(path) {
		return Result{path, "dirty", "uncommitted changes"}
	}
	if _, _, rc := runGit("-C", path, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch); rc == 0 {
		return Result{path, "exists", ""}
	}
	if _, _, rc := runGit("-C", path, "rev-parse", "--verify", "--quiet", "refs/heads/"+mainBranch); rc != 0 {
		return Result{path, "absent", ""}
	}
	out, errOut, rc := runGit("-C", path, "checkout", "-b", branch, mainBranch)
	if rc != 0 {
		return Result{path, "failed", firstNonEmpty(errOut, out)}
	}
	return Result{path, "created", ""}
}
