package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ringclaw/ringclaw/ringcentral"
)

func newTestServer() *Server {
	auth := ringcentral.NewAuth("id", "secret", "jwt", "https://example.com")
	creds := &ringcentral.Credentials{
		ClientID:     "id",
		ClientSecret: "secret",
		JWTToken:     "jwt",
		ChatID:       "default-chat",
		ServerURL:    "https://example.com",
	}
	client := ringcentral.NewClient(creds)
	_ = auth
	return NewServer(client, "127.0.0.1:0")
}

func TestHealthEndpoint(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok\n"))
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandleSend_InvalidMethod(t *testing.T) {
	s := newTestServer()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/send", s.handleSend)

	req := httptest.NewRequest(http.MethodGet, "/api/send", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleSend_InvalidJSON(t *testing.T) {
	s := newTestServer()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/send", s.handleSend)

	req := httptest.NewRequest(http.MethodPost, "/api/send", bytes.NewBufferString("not json"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleSend_MissingFields(t *testing.T) {
	s := newTestServer()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/send", s.handleSend)

	body, _ := json.Marshal(SendRequest{To: "chat1"})
	req := httptest.NewRequest(http.MethodPost, "/api/send", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
