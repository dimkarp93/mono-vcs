package output

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/dimkarp93/mono-vcs/internal/colors"
)

func Die(stderr io.Writer, msg string) {
	fmt.Fprintf(stderr, "error: %s\n", msg)
}

func termWidth() int {
	if c := os.Getenv("COLUMNS"); c != "" {
		if n, err := strconv.Atoi(c); err == nil && n > 0 {
			return n
		}
	}
	return 100
}

func WrapCSV(items []string, width int) []string {
	if len(items) == 0 {
		return []string{""}
	}
	var lines []string
	cur := ""
	for i, it := range items {
		sep := ", "
		if i == len(items)-1 {
			sep = ""
		}
		token := it + sep
		if cur != "" && len(cur)+len(token) > width {
			lines = append(lines, trimTrailingComma(cur))
			cur = token
		} else {
			cur += token
		}
	}
	if cur != "" {
		lines = append(lines, trimTrailingComma(cur))
	}
	return lines
}

func trimTrailingComma(s string) string {
	s = strings.TrimRight(s, " ")
	return strings.TrimRight(s, ",")
}

type FeatureRow struct {
	Branch string
	Repos  []string
	Active []string
}

func ljust(s string, n int) string {
	w := utf8.RuneCountInString(s)
	if w >= n {
		return s
	}
	return s + strings.Repeat(" ", n-w)
}

func PrintFeaturesTable(w io.Writer, rows []FeatureRow, useColor bool) {
	col1Label, col2Label, col3Label := "branch", "repos", "active"
	col1W := len(col1Label)
	for _, r := range rows {
		if l := utf8.RuneCountInString(r.Branch); l > col1W {
			col1W = l
		}
	}
	remaining := termWidth() - col1W - 10
	if remaining < 40 {
		remaining = 40
	}
	col2W := remaining / 2
	if col2W < len(col2Label) {
		col2W = len(col2Label)
	}
	col3W := remaining - col2W
	if col3W < len(col3Label) {
		col3W = len(col3Label)
	}

	bar1 := strings.Repeat("─", col1W+2)
	bar2 := strings.Repeat("─", col2W+2)
	bar3 := strings.Repeat("─", col3W+2)
	top := "┌" + bar1 + "┬" + bar2 + "┬" + bar3 + "┐"
	sep := "├" + bar1 + "┼" + bar2 + "┼" + bar3 + "┤"
	bot := "└" + bar1 + "┴" + bar2 + "┴" + bar3 + "┘"

	emit := func(left string, midLines, rightLines []string, leftColor string) {
		n := len(midLines)
		if len(rightLines) > n {
			n = len(rightLines)
		}
		if n < 1 {
			n = 1
		}
		for len(midLines) < n {
			midLines = append(midLines, "")
		}
		for len(rightLines) < n {
			rightLines = append(rightLines, "")
		}
		for i := 0; i < n; i++ {
			lRaw := ""
			if i == 0 {
				lRaw = left
			}
			lPadded := ljust(lRaw, col1W)
			if leftColor != "" && lRaw != "" {
				lPadded = colors.Colorize(lRaw, leftColor, useColor) +
					strings.Repeat(" ", col1W-utf8.RuneCountInString(lRaw))
			}
			fmt.Fprintf(w, "│ %s │ %s │ %s │\n", lPadded, ljust(midLines[i], col2W), ljust(rightLines[i], col3W))
		}
	}

	fmt.Fprintln(w, top)
	emit(col1Label, []string{col2Label}, []string{col3Label}, "")
	fmt.Fprintln(w, sep)
	for i, r := range rows {
		var color string
		if len(r.Active) == 0 {
			color = colors.Blue
		} else if len(r.Active) == len(r.Repos) {
			color = colors.Green
		} else {
			color = colors.Yellow
		}
		right := WrapCSV(r.Active, col3W)
		if len(r.Active) == 0 {
			right = []string{"—"}
		}
		emit(r.Branch, WrapCSV(r.Repos, col2W), right, color)
		if i < len(rows)-1 {
			fmt.Fprintln(w, sep)
		}
	}
	fmt.Fprintln(w, bot)
}
