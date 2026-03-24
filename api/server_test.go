package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ringclaw/ringclaw/ringcentral"
)

func newTestServer() *Server {
	creds := &ringcentral.Credentials{
		ClientID:     "id",
		ClientSecret: "secret",
		JWTToken:     "jwt",
		ChatID:       "default-chat",
		ServerURL:    "https://example.com",
	}
	client := ringcentral.NewClient(creds)
	return NewServer(client, "127.0.0.1:0")
}

func newTestServerWithBackend(backend *httptest.Server) *Server {
	creds := &ringcentral.Credentials{
		ClientID:     "id",
		ClientSecret: "secret",
		JWTToken:     "jwt",
		ChatID:       "default-chat",
		ServerURL:    backend.URL,
	}
	client := ringcentral.NewClient(creds)
	// Pre-set a valid token so auth doesn't need to call the real endpoint
	client.Auth().SetTokenForTest("test-token", time.Now().Add(1*time.Hour))
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

func TestHandleSend_Success(t *testing.T) {
	var receivedText string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock the RingCentral SendPost endpoint
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		receivedText = body["text"]
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "post-1", "text": receivedText})
	}))
	defer backend.Close()

	s := newTestServerWithBackend(backend)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/send", s.handleSend)

	body, _ := json.Marshal(SendRequest{Text: "hello from test"})
	req := httptest.NewRequest(http.MethodPost, "/api/send", bytes.NewBuffer(body))
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if receivedText != "hello from test" {
		t.Errorf("backend received %q, want %q", receivedText, "hello from test")
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
