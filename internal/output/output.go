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
	Branch     string
	Repos      []string
	Active     []string
	Remotes    []string}
type featureTag struct {
	text  string
	color string
}

func (r FeatureRow) tags() []featureTag {
	panic("not implemented")
}
func (r FeatureRow) label() string {
	panic("not implemented")
}
func (r FeatureRow) colorLabel(shown, branchColor string, useColor bool) string {
	panic("not implemented")
}
func featureColumns(total int, rows []FeatureRow) (int, int, int, int) {
	const minCell = 8
	avail := total - 13
	branchW := DisplayWidth("branch")
	for _, r := range rows {
		if l := DisplayWidth(r.Branch); l > branchW {
			branchW = l
		}
	}
	remoteW := DisplayWidth("remotes")
	for _, r := range rows {
		for _, a := range r.Remotes {
			if l := DisplayWidth(a); l > remoteW {
				remoteW = l
			}
		}
	}
	branchCap := avail * 3 / 10
	if branchCap < DisplayWidth("branch") {
		branchCap = DisplayWidth("branch")
	}
	if branchW > branchCap {
		branchW = branchCap
	}
	remoteCap := avail / 5
	if remoteCap < DisplayWidth("remotes") {
		remoteCap = DisplayWidth("remotes")
	}
	if remoteW > remoteCap {
		remoteW = remoteCap
	}
	if rest := avail - branchW - remoteW; rest < 2*minCell {
		branchW = avail - remoteW - 2*minCell
		if branchW < DisplayWidth("branch") {
			branchW = DisplayWidth("branch")
		}
	}
	rest := avail - branchW - remoteW
	if rest < 2 {
		rest = 2
	}
	col2W := (rest + 1) / 2
	return branchW, col2W, rest - col2W, remoteW
}

type featuresTable struct {
	w        io.Writer
	widths   [4]int
	useColor bool
}

func PrintFeaturesTable(w io.Writer, rows []FeatureRow, useColor bool) {
	t := featuresTable{w: w, useColor: useColor}
	t.widths[0], t.widths[1], t.widths[2], t.widths[3] = featureColumns(Width(w), rows)
	top, sep, bot := t.rules()

	fmt.Fprintln(w, top)
	t.emit("branch", [3][]string{{"repos"}, {"active"}, {"remotes"}}, nil)
	fmt.Fprintln(w, sep)
	for i, r := range rows {
		t.emitRow(r)
		if i < len(rows)-1 {
			fmt.Fprintln(w, sep)
		}
	}
	fmt.Fprintln(w, bot)
}

func (t featuresTable) rules() (string, string, string) {
	var bars [4]string
	for i, width := range t.widths {
		bars[i] = strings.Repeat("─", width+2)
	}

	top := "┌" + strings.Join(bars[:], "┬") + "┐"
	sep := "├" + strings.Join(bars[:], "┼") + "┤"
	bot := "└" + strings.Join(bars[:], "┴") + "┘"
	return top, sep, bot
}

func (t featuresTable) emitRow(r FeatureRow) {
	color := rowColor(r)
	cols := [3][]string{
		Items(r.Repos, t.widths[1]),
		orDash(Items(r.Active, t.widths[2]), len(r.Active)),
		orDash(Items(r.Remotes, t.widths[3]), len(r.Remotes)),
	}
	t.emit(r.Branch, cols, func(shown string) string { return colors.Colorize(shown, color, t.useColor) })
}

func rowColor(r FeatureRow) string {
	if len(r.Active) == 0 {
		return colors.Gray
	}

	return colors.Blue
}

func orDash(lines []string, n int) []string {
	if n == 0 {
		return []string{"—"}
	}

	return lines
}

func (t featuresTable) emit(left string, cols [3][]string, colorLeft func(string) string) {
	n := max(len(cols[0]), len(cols[1]), len(cols[2]), 1)
	for i := range n {
		fmt.Fprintf(t.w, "│ %s │ %s │ %s │ %s │\n", t.leftCell(i, left, colorLeft),
			Cell(lineAt(cols[0], i), t.widths[1]), Cell(lineAt(cols[1], i), t.widths[2]), Cell(lineAt(cols[2], i), t.widths[3]))
	}
}

func (t featuresTable) leftCell(i int, left string, colorLeft func(string) string) string {
	if i != 0 {
		return Pad("", t.widths[0])
	}

	shown := Truncate(left, t.widths[0])
	if colorLeft != nil && shown != "" {
		return PadColored(shown, colorLeft(shown), t.widths[0])
	}
	return Pad(shown, t.widths[0])
}

func lineAt(lines []string, i int) string {
	if i < len(lines) {
		return lines[i]
	}

	return ""
}
