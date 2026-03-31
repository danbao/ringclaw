package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNanoClawAgent_Chat_JSONReply(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Fatalf("unexpected auth header: %q", got)
		}
		var req nanoClawRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.ConversationID != "conv-1" {
			t.Fatalf("unexpected conversation ID: %q", req.ConversationID)
		}
		if req.GroupJID != "rc-group" {
			t.Fatalf("unexpected group JID: %q", req.GroupJID)
		}
		json.NewEncoder(w).Encode(nanoClawResponse{Reply: "hello from andy"})
	}))
	defer srv.Close()

	ag := NewNanoClawAgent(NanoClawAgentConfig{
		Name:        "andy",
		Endpoint:    srv.URL,
		APIKey:      "secret",
		GroupJID:    "rc-group",
		Sender:      "Andy",
		ContextMode: "group",
	})

	reply, err := ag.Chat(context.Background(), "conv-1", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply != "hello from andy" {
		t.Fatalf("unexpected reply: %q", reply)
	}
}

func TestNanoClawAgent_Chat_PlainTextReply(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("plain text reply"))
	}))
	defer srv.Close()

	ag := NewNanoClawAgent(NanoClawAgentConfig{Endpoint: srv.URL})
	reply, err := ag.Chat(context.Background(), "conv-1", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply != "plain text reply" {
		t.Fatalf("unexpected reply: %q", reply)
	}
}

func TestNanoClawAgent_SetCwd(t *testing.T) {
	ag := NewNanoClawAgent(NanoClawAgentConfig{})
	ag.SetCwd("/tmp/project")
	if ag.cwd != "/tmp/project" {
		t.Fatalf("unexpected cwd: %q", ag.cwd)
	}
}
