package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/danbao/ringclaw/messaging"
	"github.com/danbao/ringclaw/ringcentral"
)

// Server provides an HTTP API for sending messages.
type Server struct {
	client *ringcentral.Client
	addr   string
}

// NewServer creates an API server.
func NewServer(client *ringcentral.Client, addr string) *Server {
	if addr == "" {
		addr = "127.0.0.1:18011"
	}
	return &Server{client: client, addr: addr}
}

// SendRequest is the JSON body for POST /api/send.
type SendRequest struct {
	To       string `json:"to"`
	Text     string `json:"text,omitempty"`
	MediaURL string `json:"media_url,omitempty"`
}

// Run starts the HTTP server. Blocks until ctx is cancelled.
func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/send", s.handleSend)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})

	srv := &http.Server{Addr: s.addr, Handler: mux}

	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()

	log.Printf("[api] listening on %s", s.addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) handleSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var req SendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Use specified chat ID or fall back to configured default
	chatID := req.To
	if chatID == "" {
		chatID = s.client.ChatID()
	}
	if chatID == "" {
		http.Error(w, `"to" is required (no default chat configured)`, http.StatusBadRequest)
		return
	}
	if req.Text == "" && req.MediaURL == "" {
		http.Error(w, `"text" or "media_url" is required`, http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	if req.Text != "" {
		if err := messaging.SendTextReply(ctx, s.client, chatID, req.Text); err != nil {
			log.Printf("[api] send text failed: %v", err)
			http.Error(w, "send text failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("[api] sent text to %s: %q", chatID, req.Text)
	}

	if req.MediaURL != "" {
		if err := messaging.SendMediaFromURL(ctx, s.client, chatID, req.MediaURL); err != nil {
			log.Printf("[api] send media failed: %v", err)
			http.Error(w, "send media failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("[api] sent media to %s: %s", chatID, req.MediaURL)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
