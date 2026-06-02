package colors

import "testing"

func withTTY(v bool) func() {
	old := IsTTY
	IsTTY = func() bool { return v }
	return func() { IsTTY = old }
}

func TestColorizeEnabled(t *testing.T) {
	if got := Colorize("x", Red, true); got != Red+"x"+Reset {
		t.Fatalf("got %q", got)
	}
}

func TestColorizeDisabledIsPassthrough(t *testing.T) {
	if got := Colorize("x", Red, false); got != "x" {
		t.Fatalf("got %q", got)
	}
}

func TestEnabledRespectsNoColorFlag(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	defer withTTY(true)()
	if Enabled(true) {
		t.Fatal("expected false when --no-color set")
	}
}

func TestEnabledRespectsEnv(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	defer withTTY(true)()
	if Enabled(false) {
		t.Fatal("expected false when NO_COLOR set")
	}
}

func TestEnabledFalseWhenNotTTY(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	defer withTTY(false)()
	if Enabled(false) {
		t.Fatal("expected false when not a TTY")
	}
}

func TestEnabledTrueWhenTTYAndNoFlag(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	defer withTTY(true)()
	if !Enabled(false) {
		t.Fatal("expected true when TTY and no flag")
	}
}
