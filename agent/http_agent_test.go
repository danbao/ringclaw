package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPAgent_Chat_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": "response text"}},
			},
		})
	}))
	defer srv.Close()

	ag := NewHTTPAgent(HTTPAgentConfig{
		Endpoint: srv.URL,
		Model:    "test-model",
	})

	reply, err := ag.Chat(context.Background(), "conv1", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply != "response text" {
		t.Errorf("expected 'response text', got %q", reply)
	}
}

func TestHTTPAgent_Chat_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("rate limited"))
	}))
	defer srv.Close()

	ag := NewHTTPAgent(HTTPAgentConfig{Endpoint: srv.URL})
	_, err := ag.Chat(context.Background(), "conv1", "hello")
	if err == nil {
		t.Fatal("expected error for 429 response")
	}
}

func TestHTTPAgent_Chat_EmptyChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"choices": []interface{}{}})
	}))
	defer srv.Close()

	ag := NewHTTPAgent(HTTPAgentConfig{Endpoint: srv.URL})
	_, err := ag.Chat(context.Background(), "conv1", "hello")
	if err == nil {
		t.Fatal("expected error for empty choices")
	}
}

func TestHTTPAgent_HistoryTrimming(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": "ok"}},
			},
		})
	}))
	defer srv.Close()

	ag := NewHTTPAgent(HTTPAgentConfig{
		Endpoint:   srv.URL,
		MaxHistory: 3,
	})

	// Send enough messages to trigger trimming
	for i := 0; i < 5; i++ {
		_, err := ag.Chat(context.Background(), "conv1", "msg")
		if err != nil {
			t.Fatalf("unexpected error on msg %d: %v", i, err)
		}
	}

	ag.mu.Lock()
	histLen := len(ag.history["conv1"])
	ag.mu.Unlock()

	// maxHistory=3 means 3*2=6 entries max (user+assistant pairs)
	if histLen > 6 {
		t.Errorf("history not trimmed: got %d entries, max should be 6", histLen)
	}
}

func TestHTTPAgent_BuildMessages_WithSystemPrompt(t *testing.T) {
	ag := NewHTTPAgent(HTTPAgentConfig{
		SystemPrompt: "you are a bot",
	})

	ag.mu.Lock()
	msgs := ag.buildMessages("conv1", "hello")
	ag.mu.Unlock()

	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "system" || msgs[0].Content != "you are a bot" {
		t.Errorf("unexpected system message: %+v", msgs[0])
	}
	if msgs[1].Role != "user" || msgs[1].Content != "hello" {
		t.Errorf("unexpected user message: %+v", msgs[1])
	}
}

func TestHTTPAgent_BuildMessages_WithHistory(t *testing.T) {
	ag := NewHTTPAgent(HTTPAgentConfig{})

	ag.mu.Lock()
	ag.history["conv1"] = []ChatMessage{
		{Role: "user", Content: "prev"},
		{Role: "assistant", Content: "prev reply"},
	}
	msgs := ag.buildMessages("conv1", "new msg")
	ag.mu.Unlock()

	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Content != "prev" {
		t.Errorf("unexpected history[0]: %+v", msgs[0])
	}
	if msgs[2].Role != "user" || msgs[2].Content != "new msg" {
		t.Errorf("unexpected last msg: %+v", msgs[2])
	}
}
