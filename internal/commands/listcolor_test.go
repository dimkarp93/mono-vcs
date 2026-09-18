package commands

import (
	"testing"

	"github.com/dimkarp93/mono-vcs/internal/colors"
)

func TestListColor(t *testing.T) {
	cases := []struct {
		name      string
		inRemote  bool
		inLocal   bool
		onFeature bool
		sync      int
		want      string
	}{
		{"local only on feature", false, true, true, -1, colors.Red},
		{"local only on default", false, true, false, -1, colors.Red},
		{"both on feature", true, true, true, 1, colors.Blue},
		{"both out of sync", true, true, false, 0, colors.Yellow},
		{"both synced", true, true, false, 1, colors.Green},
		{"remote only", true, false, false, -1, colors.Gray},
	}
	for _, c := range cases {
		if got := listColor(c.inRemote, c.inLocal, c.onFeature, c.sync); got != c.want {
			t.Fatalf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}
