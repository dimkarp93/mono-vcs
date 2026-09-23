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
