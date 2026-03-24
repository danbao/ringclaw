package ringcentral

import (
	"testing"
	"time"
)

func TestMonitor_MarkAndCheckSentPost(t *testing.T) {
	m := &Monitor{sentPosts: make(map[string]time.Time)}
	m.MarkSentPost("post-1")

	if !m.IsSentPost("post-1") {
		t.Error("expected post-1 to be marked as sent")
	}
	if m.IsSentPost("post-2") {
		t.Error("expected post-2 to NOT be marked as sent")
	}
}

func TestMonitor_SentPostExpiry(t *testing.T) {
	m := &Monitor{sentPosts: make(map[string]time.Time)}

	// Manually insert an expired entry
	m.mu.Lock()
	m.sentPosts["old-post"] = time.Now().Add(-10 * time.Minute)
	m.mu.Unlock()

	if m.IsSentPost("old-post") {
		t.Error("expected expired post to return false")
	}

	// Verify it was cleaned up
	m.mu.Lock()
	_, exists := m.sentPosts["old-post"]
	m.mu.Unlock()
	if exists {
		t.Error("expected expired post to be deleted from map")
	}
}

func TestMonitor_CalcBackoff(t *testing.T) {
	m := &Monitor{sentPosts: make(map[string]time.Time)}

	m.failures = 1
	d := m.calcBackoff()
	if d != initialBackoff {
		t.Errorf("failures=1: got %v, want %v", d, initialBackoff)
	}

	m.failures = 2
	d = m.calcBackoff()
	if d != initialBackoff*2 {
		t.Errorf("failures=2: got %v, want %v", d, initialBackoff*2)
	}

	m.failures = 100
	d = m.calcBackoff()
	if d != maxBackoff {
		t.Errorf("failures=100: got %v, want %v (maxBackoff)", d, maxBackoff)
	}
}

func TestIsBotMessage(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{"--------answer--------\nhello\n---------end----------", true},
		{"Thinking...", true},
		{"hello world", false},
		{"", false},
	}
	for _, tt := range tests {
		got := isBotMessage(tt.text)
		if got != tt.want {
			t.Errorf("isBotMessage(%q) = %v, want %v", tt.text, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		s    string
		n    int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello..."},
		{"", 5, ""},
	}
	for _, tt := range tests {
		got := truncate(tt.s, tt.n)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.n, got, tt.want)
		}
	}
}
