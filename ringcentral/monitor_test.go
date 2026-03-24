package ringcentral

import (
	"context"
	"encoding/json"
	"sync"
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

func newTestMonitor(chatID string, handler MessageHandler) *Monitor {
	creds := &Credentials{
		ClientID:     "id",
		ClientSecret: "secret",
		JWTToken:     "jwt",
		ChatID:       chatID,
	}
	client := NewClient(creds)
	return NewMonitor(client, handler)
}

func makeWSMessage(post Post) []byte {
	header := map[string]string{"type": "ServerNotification"}
	event := WSEvent{
		UUID:  "test-uuid",
		Event: "/team-messaging/v1/posts",
		Body:  post,
	}
	arr := []interface{}{header, event}
	data, _ := json.Marshal(arr)
	return data
}

func TestMonitor_HandleWSMessage_PostAdded(t *testing.T) {
	var mu sync.Mutex
	var received []Post

	m := newTestMonitor("chat-1", func(ctx context.Context, client *Client, post Post) {
		mu.Lock()
		received = append(received, post)
		mu.Unlock()
	})

	msg := makeWSMessage(Post{
		ID:        "p1",
		GroupID:   "chat-1",
		Type:      "TextMessage",
		Text:      "hello from user",
		CreatorID: "user-1",
		EventType: "PostAdded",
	})

	m.handleWSMessage(context.Background(), msg)

	// handler is called in a goroutine, wait briefly
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("expected 1 post dispatched, got %d", len(received))
	}
	if received[0].ID != "p1" {
		t.Errorf("expected post ID p1, got %s", received[0].ID)
	}
}

func TestMonitor_HandleWSMessage_IgnoreBotMessage(t *testing.T) {
	var called bool
	m := newTestMonitor("chat-1", func(ctx context.Context, client *Client, post Post) {
		called = true
	})

	// "Thinking..." is a bot marker
	msg := makeWSMessage(Post{
		ID:        "p2",
		GroupID:   "chat-1",
		Type:      "TextMessage",
		Text:      "Thinking...",
		CreatorID: "bot-1",
		EventType: "PostAdded",
	})

	m.handleWSMessage(context.Background(), msg)
	time.Sleep(50 * time.Millisecond)

	if called {
		t.Error("handler should not be called for bot messages")
	}
}

func TestMonitor_HandleWSMessage_FilterByChatID(t *testing.T) {
	var called bool
	m := newTestMonitor("chat-1", func(ctx context.Context, client *Client, post Post) {
		called = true
	})

	// Message from a different chat
	msg := makeWSMessage(Post{
		ID:        "p3",
		GroupID:   "chat-OTHER",
		Type:      "TextMessage",
		Text:      "hello",
		CreatorID: "user-1",
		EventType: "PostAdded",
	})

	m.handleWSMessage(context.Background(), msg)
	time.Sleep(50 * time.Millisecond)

	if called {
		t.Error("handler should not be called for messages from other chats")
	}
}

func TestMonitor_HandleWSMessage_IgnoreNonText(t *testing.T) {
	var called bool
	m := newTestMonitor("chat-1", func(ctx context.Context, client *Client, post Post) {
		called = true
	})

	msg := makeWSMessage(Post{
		ID:        "p4",
		GroupID:   "chat-1",
		Type:      "PersonJoined",
		Text:      "",
		CreatorID: "user-1",
		EventType: "PostAdded",
	})

	m.handleWSMessage(context.Background(), msg)
	time.Sleep(50 * time.Millisecond)

	if called {
		t.Error("handler should not be called for non-text messages")
	}
}

func TestMonitor_HandleWSMessage_IgnoreSentPost(t *testing.T) {
	var called bool
	m := newTestMonitor("chat-1", func(ctx context.Context, client *Client, post Post) {
		called = true
	})

	// Mark post as sent by bot
	m.MarkSentPost("p5")

	msg := makeWSMessage(Post{
		ID:        "p5",
		GroupID:   "chat-1",
		Type:      "TextMessage",
		Text:      "bot reply",
		CreatorID: "bot-1",
		EventType: "PostAdded",
	})

	m.handleWSMessage(context.Background(), msg)
	time.Sleep(50 * time.Millisecond)

	if called {
		t.Error("handler should not be called for bot's own sent posts")
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
