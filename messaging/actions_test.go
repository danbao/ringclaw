package messaging

import "testing"

func TestIsActionCommand(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{"/task list", true},
		{"/task create test", true},
		{"/note list", true},
		{"/event create meeting 2026-01-01T10:00:00Z 2026-01-01T11:00:00Z", true},
		{"/Task list", true},
		{"/help", false},
		{"/status", false},
		{"hello", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := IsActionCommand(tt.text); got != tt.want {
			t.Errorf("IsActionCommand(%q) = %v, want %v", tt.text, got, tt.want)
		}
	}
}

func TestExtractAfter(t *testing.T) {
	tests := []struct {
		raw, keyword, want string
	}{
		{"/task create buy milk", "create", "buy milk"},
		{"/note create Meeting Notes | body text", "create", "Meeting Notes | body text"},
		{"/task create", "create", ""},
		{"no match", "create", ""},
	}
	for _, tt := range tests {
		if got := extractAfter(tt.raw, tt.keyword); got != tt.want {
			t.Errorf("extractAfter(%q, %q) = %q, want %q", tt.raw, tt.keyword, got, tt.want)
		}
	}
}

func TestSplitNoteTitleBody(t *testing.T) {
	tests := []struct {
		content, wantTitle, wantBody string
	}{
		{"Meeting Notes | discussed API design", "Meeting Notes", "discussed API design"},
		{"Quick Note", "Quick Note", ""},
		{"Title | ", "Title", ""},
	}
	for _, tt := range tests {
		title, body := splitNoteTitleBody(tt.content)
		if title != tt.wantTitle || body != tt.wantBody {
			t.Errorf("splitNoteTitleBody(%q) = (%q, %q), want (%q, %q)", tt.content, title, body, tt.wantTitle, tt.wantBody)
		}
	}
}

func TestParseKeyValues(t *testing.T) {
	tests := []struct {
		input string
		want  []keyValue
	}{
		{"subject=new title", []keyValue{{key: "subject", value: "new title"}}},
		{"title=hello", []keyValue{{key: "title", value: "hello"}}},
		{"", nil},
	}
	for _, tt := range tests {
		got := parseKeyValues(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("parseKeyValues(%q) returned %d items, want %d", tt.input, len(got), len(tt.want))
			continue
		}
		for i := range got {
			if got[i].key != tt.want[i].key || got[i].value != tt.want[i].value {
				t.Errorf("parseKeyValues(%q)[%d] = {%q, %q}, want {%q, %q}", tt.input, i, got[i].key, got[i].value, tt.want[i].key, tt.want[i].value)
			}
		}
	}
}

func TestStatusEmoji(t *testing.T) {
	tests := []struct {
		status, want string
	}{
		{"Completed", "[v]"},
		{"InProgress", "[~]"},
		{"Pending", "[ ]"},
		{"", "[ ]"},
	}
	for _, tt := range tests {
		if got := statusEmoji(tt.status); got != tt.want {
			t.Errorf("statusEmoji(%q) = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestFormatActionHelp(t *testing.T) {
	for _, cmd := range []string{"/task", "/note", "/event"} {
		help := formatActionHelp(cmd)
		if help == "" {
			t.Errorf("formatActionHelp(%q) returned empty string", cmd)
		}
	}
	help := formatActionHelp("/unknown")
	if help == "" {
		t.Error("formatActionHelp(/unknown) returned empty string")
	}
}
