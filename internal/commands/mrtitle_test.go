package commands

import "testing"

func TestTicketOf(t *testing.T) {
	cases := []struct {
		branch string
		ticket string
		ok     bool
	}{
		{"PROJ-123-add-login", "PROJ-123", true},
		{"PROJ-123-x", "PROJ-123", true},
		{"PROJ-123", "PROJ-123", true},
		{"PROJ-123-", "PROJ-123", true},
		{"practice-improves", "", false},
		{"hotfix", "", false},
		{"-123-add", "", false},
		{"PROJ--add", "", false},
		{"PROJ-12a-add", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		ticket, ok := ticketOf(c.branch)
		if ticket != c.ticket || ok != c.ok {
			t.Fatalf("ticketOf(%q) = %q, %v; want %q, %v", c.branch, ticket, ok, c.ticket, c.ok)
		}
	}
}

func TestMRTitle(t *testing.T) {
	cases := []struct {
		ticket string
		title  string
		want   string
	}{
		{"PROJ-123", "", "[PROJ-123]"},
		{"PROJ-123", "fix login", "[PROJ-123] fix login"},
		{"PROJ-123", "[PROJ-123] fix login", "[PROJ-123] fix login"},
		{"PROJ-123", "  [OTHER-1] fix", "  [OTHER-1] fix"},
	}
	for _, c := range cases {
		if got := mrTitle(c.ticket, c.title); got != c.want {
			t.Fatalf("mrTitle(%q, %q) = %q; want %q", c.ticket, c.title, got, c.want)
		}
	}
}

func TestStripTicket(t *testing.T) {
	cases := []struct {
		ticket  string
		subject string
		want    string
	}{
		{"PROJ-123", "fix problem", "fix problem"},
		{"PROJ-123", "[PROJ-123] fix problem", "fix problem"},
		{"PROJ-123", "  [PROJ-123]   fix problem  ", "fix problem"},
		{"PROJ-123", "[PROJ-123]", ""},
		{"PROJ-123", "[OTHER-1] fix problem", "[OTHER-1] fix problem"},
		{"PROJ-123", "", ""},
	}
	for _, c := range cases {
		if got := stripTicket(c.ticket, c.subject); got != c.want {
			t.Fatalf("stripTicket(%q, %q) = %q; want %q", c.ticket, c.subject, got, c.want)
		}
	}
}

func TestMRTitleFor(t *testing.T) {
	cases := []struct {
		explicit string
		subject  string
		want     string
	}{
		{"", "fix problem", "[PROJ-123] fix problem"},
		{"", "[PROJ-123] fix problem", "[PROJ-123] fix problem"},
		{"", "", "[PROJ-123]"},
		{"manual", "fix problem", "[PROJ-123] manual"},
	}
	for _, c := range cases {
		if got := mrTitleFor("PROJ-123", c.explicit)(c.subject); got != c.want {
			t.Fatalf("mrTitleFor(%q)(%q) = %q; want %q", c.explicit, c.subject, got, c.want)
		}
	}
}
