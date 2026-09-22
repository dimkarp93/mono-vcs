package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/dimkarp93/mono-vcs/internal/colors"
)

type AliasRow struct {
	Alias string
	Type  string
	Value string
}

func bold(s string, useColor bool) string {
	return colors.Colorize(s, colors.Bold, useColor)
}

func aliasColumns(rows []AliasRow, total int) (int, int, int) {
	aliasW, typeW, valueW := DisplayWidth("alias"), DisplayWidth("type"), DisplayWidth("value")
	for _, r := range rows {
		if l := DisplayWidth(r.Alias); l > aliasW {
			aliasW = l
		}
		if l := DisplayWidth(r.Type); l > typeW {
			typeW = l
		}
		if l := DisplayWidth(r.Value); l > valueW {
			valueW = l
		}
	}
	avail := total - 10
	if rest := avail - aliasW - typeW; valueW > rest {
		valueW = rest
	}
	if valueW < DisplayWidth("value") {
		valueW = DisplayWidth("value")
	}
	return aliasW, typeW, valueW
}

func PrintAliasTable(w io.Writer, kind, describe string, rows []AliasRow, useColor bool) {
	title := kind
	if describe != "" {
		title += " — " + describe
	}
	fmt.Fprintln(w, bold(title, useColor))

	aliasW, typeW, valueW := aliasColumns(rows, Width(w))
	bar1 := strings.Repeat("─", aliasW+2)
	bar2 := strings.Repeat("─", typeW+2)
	bar3 := strings.Repeat("─", valueW+2)
	fmt.Fprintln(w, "┌"+bar1+"┬"+bar2+"┬"+bar3+"┐")
	fmt.Fprintf(w, "│ %s │ %s │ %s │\n",
		PadColored("alias", bold("alias", useColor), aliasW),
		PadColored("type", bold("type", useColor), typeW),
		PadColored("value", bold("value", useColor), valueW))
	fmt.Fprintln(w, "├"+bar1+"┼"+bar2+"┼"+bar3+"┤")
	if len(rows) == 0 {
		fmt.Fprintf(w, "│ %s │ %s │ %s │\n", Cell("—", aliasW), Cell("—", typeW), Cell("no aliases yet", valueW))
	}
	for _, r := range rows {
		color := colors.Green
		if r.Type == "custom" {
			color = colors.Blue
		}
		a := Truncate(r.Alias, aliasW)
		fmt.Fprintf(w, "│ %s │ %s │ %s │\n",
			PadColored(a, colors.Colorize(a, color, useColor), aliasW),
			Cell(r.Type, typeW),
			Cell(r.Value, valueW))
	}
	fmt.Fprintln(w, "└"+bar1+"┴"+bar2+"┴"+bar3+"┘")
}

type RemoteLegendRow struct {
	Alias  string
	Host   string
	Value  string
	Custom bool
}

func PrintRemotesLegend(w io.Writer, rows []RemoteLegendRow, useColor bool) {
	if len(rows) == 0 {
		return
	}
	aliasW, hostW := 0, 0
	for _, r := range rows {
		if l := DisplayWidth(r.Alias); l > aliasW {
			aliasW = l
		}
		if l := DisplayWidth(r.Host); l > hostW {
			hostW = l
		}
	}
	fmt.Fprintln(w, "remotes:")
	for _, r := range rows {
		suffix := ""
		if r.Custom {
			suffix = " (custom)"
		}
		fmt.Fprintf(w, "  %s  %s  %s%s\n",
			PadColored(r.Alias, colors.Colorize(r.Alias, colors.Gray, useColor), aliasW),
			Pad(r.Host, hostW), r.Value, suffix)
	}
}
