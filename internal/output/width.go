package output

import (
	"io"
	"os"
	"strconv"
)

const (
	defaultWidth = 100
	minWidth     = 60
)

type winsize struct {
	rows   uint16
	cols   uint16
	xpixel uint16
	ypixel uint16
}

func Width(w io.Writer) int {
	n := detectWidth(w)
	if n < minWidth {
		n = minWidth
	}
	return n
}

func detectWidth(w io.Writer) int {
	if c := os.Getenv("COLUMNS"); c != "" {
		if n, err := strconv.Atoi(c); err == nil && n > 0 {
			return n
		}
	}
	f, ok := w.(*os.File)
	if !ok {
		return defaultWidth
	}
	if n := ioctlWidth(f.Fd()); n > 0 {
		return n
	}
	return defaultWidth
}
