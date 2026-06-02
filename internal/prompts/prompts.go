package prompts

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

var ForcePromptChoices = map[string]string{
	"y": "yes", "yes": "yes",
	"a": "yes-all", "yes-all": "yes-all", "yestoall": "yes-all",
	"s": "skip", "skip": "skip", "n": "skip", "no": "skip",
	"sa": "skip-all", "skip-all": "skip-all", "skiptoall": "skip-all",
}

type Prompter struct {
	raw io.Reader
	r   *bufio.Reader
	out io.Writer
	err io.Writer
}

func New(in io.Reader, out, err io.Writer) *Prompter {
	return &Prompter{raw: in, r: bufio.NewReader(in), out: out, err: err}
}

func (p *Prompter) readLine() (string, error) {
	line, err := p.r.ReadString('\n')
	line = strings.TrimRight(line, "\r\n")
	if err != nil && line == "" {
		return "", err
	}
	return line, nil
}

// Token prefers the GITLAB_TOKEN env var (no prompt when set); otherwise it
// reads a line with terminal echo suppressed on a best-effort basis.
func (p *Prompter) Token(optional bool) (string, error) {
	if v := strings.TrimSpace(os.Getenv("GITLAB_TOKEN")); v != "" {
		return v, nil
	}
	label := "GitLab token: "
	if optional {
		label = "GitLab token (Enter to skip): "
	}
	fmt.Fprint(p.err, label)
	restore := p.disableEcho()
	val, rerr := p.readLine()
	restore()
	if isTTY(p.raw) {
		fmt.Fprintln(p.err)
	}
	if rerr != nil && val == "" {
		return "", errors.New("token input aborted")
	}
	val = strings.TrimSpace(val)
	if val == "" {
		if optional {
			return "", nil
		}
		return "", errors.New("token is required")
	}
	return val, nil
}

func (p *Prompter) Choice(label, current string, choices []string, def string) (string, error) {
	base := def
	if contains(choices, current) {
		base = current
	}
	suffix := fmt.Sprintf(" (%s) [%s]", strings.Join(choices, "/"), base)
	for {
		fmt.Fprintf(p.out, "%s%s: ", label, suffix)
		val, err := p.readLine()
		if err != nil && val == "" {
			fmt.Fprintln(p.out)
			return "", errors.New("init aborted")
		}
		val = strings.ToLower(strings.TrimSpace(val))
		if val == "" {
			return base, nil
		}
		if contains(choices, val) {
			return val, nil
		}
		fmt.Fprintf(p.out, "  must be one of: %s\n", strings.Join(choices, ", "))
	}
}

func (p *Prompter) Free(label, current string, required bool) (string, error) {
	suffix := ""
	if current != "" {
		suffix = fmt.Sprintf(" [%s]", current)
	}
	for {
		fmt.Fprintf(p.out, "%s%s: ", label, suffix)
		val, err := p.readLine()
		if err != nil && val == "" {
			fmt.Fprintln(p.out)
			return "", errors.New("init aborted")
		}
		val = strings.TrimSpace(val)
		if val != "" {
			return val, nil
		}
		if current != "" {
			return current, nil
		}
		if !required {
			return "", nil
		}
		fmt.Fprintln(p.out, "  value is required")
	}
}

func (p *Prompter) forceLoop(prompt string) (string, error) {
	for {
		fmt.Fprint(p.out, prompt)
		val, err := p.readLine()
		if err != nil && val == "" {
			fmt.Fprintln(p.out)
			return "", errors.New("aborted")
		}
		v := strings.ToLower(strings.TrimSpace(val))
		v = strings.ReplaceAll(v, " ", "")
		v = strings.ReplaceAll(v, "_", "")
		if r, ok := ForcePromptChoices[v]; ok {
			return r, nil
		}
		fmt.Fprintln(p.out, "  please answer: y / a / s / sa")
	}
}

func (p *Prompter) Prune(path string, count int) (string, error) {
	noun := "items"
	if count == 1 {
		noun = "item"
	}
	prompt := fmt.Sprintf("discard %d uncommitted %s in %q?\n"+
		"  [y]es  [a] yes to all  [s]kip  [sa] skip to all: ", count, noun, path)
	return p.forceLoop(prompt)
}

func (p *Prompter) ForceDelete(path string) (string, error) {
	prompt := fmt.Sprintf("%q exists but is not a git repo — delete it and clone?\n"+
		"  [y]es  [a] yes to all  [s]kip  [sa] skip to all: ", path)
	return p.forceLoop(prompt)
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

func isTTY(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func (p *Prompter) disableEcho() func() {
	if !isTTY(p.raw) {
		return func() {}
	}
	f := p.raw.(*os.File)
	c := exec.Command("stty", "-echo")
	c.Stdin = f
	if c.Run() != nil {
		return func() {}
	}
	return func() {
		r := exec.Command("stty", "echo")
		r.Stdin = f
		_ = r.Run()
	}
}
