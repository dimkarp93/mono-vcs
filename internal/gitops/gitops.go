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

func PullDefaultOne(path, token, branch string) Result {
	before := LocalBranchSHA(path, branch)
	args := []string{"-C", path}
	args = append(args, GitExtraHeaderArgs(token)...)
	if CurrentBranch(path) == branch {
		args = append(args, "pull", "--ff-only", "--quiet")
	} else {
		args = append(args, "fetch", "origin",
			"refs/heads/"+branch+":refs/heads/"+branch,
			"+refs/heads/"+branch+":refs/remotes/origin/"+branch)
	}
	out, errOut, rc := runGit(args...)
	if rc != 0 {
		detail := firstNonEmpty(errOut, out)
		switch {
		case strings.Contains(detail, "non-fast-forward") || strings.Contains(detail, "rejected"):
			return Result{path, "diverged", fmt.Sprintf("local %s has diverged from origin/%s", branch, branch)}
		case IsDirty(path):
			return Result{path, "dirty", detail}
		default:
			return Result{path, "failed", detail}
		}
	}
	if after := LocalBranchSHA(path, branch); after != before {
		return Result{path, "updated", fmt.Sprintf("%s -> %s", shortSHA(before), shortSHA(after))}
	}
	return Result{path, "up-to-date", ""}
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	if sha == "" {
		return "(none)"
	}
	return sha
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
		return Result{path, "failed", firstNonEmpty(errOut, out)}, orig
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

func pullDefault(path, mainBranch, token string) (string, string, bool) {
	args := []string{"-C", path}
	args = append(args, GitExtraHeaderArgs(token)...)
	args = append(args, "pull", "--ff-only", "--quiet")
	out, errOut, rc := runGit(args...)
	if rc != 0 {
		return "", firstNonEmpty(errOut, out), false
	}
	if strings.TrimSpace(out+errOut) != "" {
		return "updated", "", true
	}
	return "up-to-date", "", true
}

func hardClean(path string) (int, string, bool) {
	targets, err := PruneTargets(path)
	if err != nil {
		return 0, err.Error(), false
	}
	if len(targets) == 0 {
		return 0, "", true
	}
	if out, errOut, rc := runGit("-C", path, "reset", "--hard", "HEAD"); rc != 0 {
		return 0, firstNonEmpty(errOut, out), false
	}
	if out, errOut, rc := runGit("-C", path, "clean", "-fd"); rc != 0 {
		return 0, firstNonEmpty(errOut, out), false
	}
	return len(targets), "", true
}

func FinishOne(path, branch, mainBranch, token string) Result {
	if _, _, rc := runGit("-C", path, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch); rc != 0 {
		return Result{path, "absent", ""}
	}
	if !HasBranch(path, mainBranch) {
		return Result{path, "failed", fmt.Sprintf("no `%s` to fall back to", mainBranch)}
	}

	var parts []string
	discarded, detail, ok := hardClean(path)
	if !ok {
		return Result{path, "failed", detail}
	}
	if discarded > 0 {
		parts = append(parts, fmt.Sprintf("discarded %s", items(discarded)))
	}

	if CurrentBranch(path) != mainBranch {
		out, errOut, rc := runGit("-C", path, "checkout", mainBranch)
		if rc != 0 {
			return Result{path, "failed", fmt.Sprintf("checkout %s: %s", mainBranch, firstNonEmpty(errOut, out))}
		}
	}
	state, detail, ok := pullDefault(path, mainBranch, token)
	if !ok {
		return Result{path, "failed", fmt.Sprintf("pull %s: %s", mainBranch, detail)}
	}
	parts = append(parts, fmt.Sprintf("switched to `%s` (%s)", mainBranch, state))

	out, errOut, rc := runGit("-C", path, "branch", "-D", branch)
	if rc != 0 {
		return Result{path, "failed", firstNonEmpty(errOut, out)}
	}
	parts = append(parts, fmt.Sprintf("removed `%s`", branch))
	return Result{path, "deleted", strings.Join(parts, ", ")}
}

func items(n int) string {
	if n == 1 {
		return "1 item"
	}
	return fmt.Sprintf("%d items", n)
}

func NewBranchOne(path, branch, mainBranch string) Result {
	if CurrentBranch(path) == branch {
		return Result{path, "already", ""}
	}
	hasBranch := LocalBranchSHA(path, branch) != ""
	if !hasBranch && LocalBranchSHA(path, mainBranch) == "" {
		return Result{path, "absent", ""}
	}

	stashed := false
	if IsDirty(path) {
		out, errOut, rc := runGit("-C", path, "stash", "push", "--include-untracked",
			"-m", "mono-vcs new "+branch)
		if rc != 0 {
			return Result{path, "failed", firstNonEmpty(errOut, out)}
		}
		stashed = true
	}

	status := "created"
	args := []string{"-C", path, "checkout", "-b", branch, mainBranch}
	if hasBranch {
		status = "switched"
		args = []string{"-C", path, "checkout", branch}
	}
	if out, errOut, rc := runGit(args...); rc != 0 {
		detail := firstNonEmpty(errOut, out)
		if stashed {
			runGit("-C", path, "stash", "pop")
		}
		return Result{path, "failed", detail}
	}

	if stashed {
		out, errOut, rc := runGit("-C", path, "stash", "pop")
		if rc != 0 {
			return Result{path, "conflict", fmt.Sprintf("on `%s`, but restoring local changes failed: %s",
				branch, firstNonEmpty(errOut, out))}
		}
		return Result{path, status, "carried local changes over"}
	}
	return Result{path, status, ""}
}

func FetchPrune(path, token string) error {
	args := []string{"-C", path}
	args = append(args, GitExtraHeaderArgs(token)...)
	args = append(args, "fetch", "--prune", "--quiet", "origin")
	out, errOut, rc := runGit(args...)
	if rc != 0 {
		return errors.New(firstNonEmpty(errOut, out))
	}
	return nil
}

func DefaultCompareRef(path, mainBranch string) string {
	remote := "refs/remotes/origin/" + mainBranch
	if _, _, rc := runGit("-C", path, "rev-parse", "--verify", "--quiet", remote); rc == 0 {
		return remote
	}
	local := "refs/heads/" + mainBranch
	if _, _, rc := runGit("-C", path, "rev-parse", "--verify", "--quiet", local); rc == 0 {
		return local
	}
	return ""
}

func MergedInto(path, branch, target string) bool {
	_, _, rc := runGit("-C", path, "merge-base", "--is-ancestor", "refs/heads/"+branch, target)
	return rc == 0
}

func DeleteMergedOne(path, branch, mainBranch, token string) Result {
	if CurrentBranch(path) == branch {
		out, errOut, rc := runGit("-C", path, "checkout", mainBranch)
		if rc != 0 {
			return Result{path, "failed", fmt.Sprintf("checkout %s: %s", mainBranch, firstNonEmpty(errOut, out))}
		}
		if _, detail, ok := pullDefault(path, mainBranch, token); !ok {
			return Result{path, "failed", fmt.Sprintf("pull %s: %s", mainBranch, detail)}
		}
	}
	out, errOut, rc := runGit("-C", path, "branch", "-D", branch)
	if rc != 0 {
		return Result{path, "failed", firstNonEmpty(errOut, out)}
	}
	return Result{path, "deleted", ""}
}

func RemoteBranchSHA(path, branch string) string {
	out, _, rc := runGit("-C", path, "rev-parse", "--verify", "--quiet", "refs/remotes/origin/"+branch)
	if rc != 0 {
		return ""
	}
	return strings.TrimSpace(out)
}

func mergeRequestURL(output string) string {
	for _, line := range strings.Split(output, "\n") {
		for _, f := range strings.Fields(line) {
			if strings.HasPrefix(f, "http") && strings.Contains(f, "/merge_requests/") {
				return f
			}
		}
	}
	return ""
}

func PushMROne(path, branch, token, title string) Result {
	local := LocalBranchSHA(path, branch)
	if local == "" {
		return Result{path, "failed", fmt.Sprintf("no local branch `%s`", branch)}
	}
	if local == RemoteBranchSHA(path, branch) {
		return Result{path, "up-to-date", fmt.Sprintf("origin/%s already at %s", branch, shortSHA(local))}
	}
	args := []string{"-C", path}
	args = append(args, GitExtraHeaderArgs(token)...)
	args = append(args, "push", "-u", "origin", "refs/heads/"+branch+":refs/heads/"+branch,
		"-o", "merge_request.create",
		"-o", "merge_request.remove_source_branch")
	if title != "" {
		args = append(args, "-o", "merge_request.title="+title)
	}
	out, errOut, rc := runGit(args...)
	if rc != 0 {
		return Result{path, "failed", firstNonEmpty(errOut, out)}
	}
	return Result{path, "pushed", mergeRequestURL(out + "\n" + errOut)}
}
