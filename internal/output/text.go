package output

import "strings"

const ellipsis = "…"

func runeWidth(r rune) int {
	switch {
	case r == 0x200d, r == 0xfe0e, r == 0xfe0f:
		return 0
	case r >= 0x0300 && r <= 0x036f,
		r >= 0x1ab0 && r <= 0x1aff,
		r >= 0x20d0 && r <= 0x20ff:
		return 0
	case r >= 0x1100 && r <= 0x115f,
		r >= 0x2e80 && r <= 0x303e,
		r >= 0x3041 && r <= 0xa4cf,
		r >= 0xac00 && r <= 0xd7a3,
		r >= 0xf900 && r <= 0xfaff,
		r >= 0xfe30 && r <= 0xfe6f,
		r >= 0xff00 && r <= 0xff60,
		r >= 0xffe0 && r <= 0xffe6,
		r >= 0x1f300 && r <= 0x1f64f,
		r >= 0x1f900 && r <= 0x1f9ff,
		r >= 0x20000 && r <= 0x3fffd:
		return 2
	}
	return 1
}

func DisplayWidth(s string) int {
	w := 0
	for _, r := range s {
		w += runeWidth(r)
	}
	return w
}

func takePrefix(s string, n int) string {
	w := 0
	for i, r := range s {
		rw := runeWidth(r)
		if w+rw > n {
			return s[:i]
		}
		w += rw
	}
	return s
}

func takeSuffix(s string, n int) string {
	runes := []rune(s)
	w := 0
	for i := len(runes) - 1; i >= 0; i-- {
		rw := runeWidth(runes[i])
		if w+rw > n {
			return string(runes[i+1:])
		}
		w += rw
	}
	return s
}

func Truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if DisplayWidth(s) <= n {
		return s
	}
	if n == 1 {
		return ellipsis
	}
	keep := n - 1
	head := (keep + 1) / 2
	return takePrefix(s, head) + ellipsis + takeSuffix(s, keep-head)
}

func Pad(s string, n int) string {
	w := DisplayWidth(s)
	if w >= n {
		return s
	}
	return s + strings.Repeat(" ", n-w)
}

func PadColored(plain, colored string, n int) string {
	w := DisplayWidth(plain)
	if w >= n {
		return colored
	}
	return colored + strings.Repeat(" ", n-w)
}

func Cell(s string, n int) string {
	return Pad(Truncate(s, n), n)
}
