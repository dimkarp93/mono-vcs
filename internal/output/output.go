package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/dimkarp93/mono-vcs/internal/colors"
)

func Die(stderr io.Writer, msg string) {
	fmt.Fprintf(stderr, "error: %s\n", msg)
}

func Items(items []string, width int) []string {
	if len(items) == 0 {
		return []string{""}
	}
	lines := make([]string, 0, len(items))
	for _, it := range items {
		lines = append(lines, Truncate(it, width))
	}
	return lines
}

type FeatureRow struct {
	Branch string
	Repos  []string
	Active []string
}

func featureColumns(total int, rows []FeatureRow) (int, int, int) {
	const minCell = 8
	avail := total - 10
	branchW := DisplayWidth("branch")
	for _, r := range rows {
		if l := DisplayWidth(r.Branch); l > branchW {
			branchW = l
		}
	}
	branchCap := avail * 3 / 10
	if branchCap < DisplayWidth("branch") {
		branchCap = DisplayWidth("branch")
	}
	if branchW > branchCap {
		branchW = branchCap
	}
	if rest := avail - branchW; rest < 2*minCell {
		branchW = avail - 2*minCell
		if branchW < DisplayWidth("branch") {
			branchW = DisplayWidth("branch")
		}
	}
	rest := avail - branchW
	if rest < 2 {
		rest = 2
	}
	col2W := (rest + 1) / 2
	return branchW, col2W, rest - col2W
}

func PrintFeaturesTable(w io.Writer, rows []FeatureRow, useColor bool) {
	col1Label, col2Label, col3Label := "branch", "repos", "active"
	col1W, col2W, col3W := featureColumns(Width(w), rows)

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
				lRaw = Truncate(left, col1W)
			}
			lPadded := Pad(lRaw, col1W)
			if leftColor != "" && lRaw != "" {
				lPadded = PadColored(lRaw, colors.Colorize(lRaw, leftColor, useColor), col1W)
			}
			fmt.Fprintf(w, "│ %s │ %s │ %s │\n", lPadded, Cell(midLines[i], col2W), Cell(rightLines[i], col3W))
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
		right := Items(r.Active, col3W)
		if len(r.Active) == 0 {
			right = []string{"—"}
		}
		emit(r.Branch, Items(r.Repos, col2W), right, color)
		if i < len(rows)-1 {
			fmt.Fprintln(w, sep)
		}
	}
	fmt.Fprintln(w, bot)
}
