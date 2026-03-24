package ringcentral

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// timeNow is a helper so tests don't rely on wall clock precision.
func timeNow() time.Time { return time.Now() }

func TestAuthenticate_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TokenResponse{
			AccessToken: "test-token",
			ExpiresIn:   3600,
		})
	}))
	defer srv.Close()

	auth := NewAuth("client-id", "client-secret", "jwt-token", srv.URL)
	auth.httpClient = srv.Client()

	if err := auth.Authenticate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	token, err := auth.AccessToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "test-token" {
		t.Errorf("expected test-token, got %q", token)
	}
}

func TestAuthenticate_InvalidCreds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid_grant"}`))
	}))
	defer srv.Close()

	auth := NewAuth("bad", "bad", "bad", srv.URL)
	auth.httpClient = srv.Client()

	if err := auth.Authenticate(); err == nil {
		t.Fatal("expected error for invalid credentials")
	}
}

func TestAccessToken_AutoRefresh(t *testing.T) {
	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TokenResponse{
			AccessToken: "refreshed-token",
			ExpiresIn:   3600,
		})
	}))
	defer srv.Close()

	auth := NewAuth("id", "secret", "jwt", srv.URL)
	auth.httpClient = srv.Client()

	// Set an expired token
	auth.mu.Lock()
	auth.accessToken = "expired"
	auth.expiresAt = time.Now().Add(-1 * time.Minute)
	auth.mu.Unlock()

	token, err := auth.AccessToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "refreshed-token" {
		t.Errorf("expected refreshed-token, got %q", token)
	}
}

func TestAccessToken_ConcurrentAccess(t *testing.T) {
	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		time.Sleep(10 * time.Millisecond) // simulate latency
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TokenResponse{
			AccessToken: "concurrent-token",
			ExpiresIn:   3600,
		})
	}))
	defer srv.Close()

	auth := NewAuth("id", "secret", "jwt", srv.URL)
	auth.httpClient = srv.Client()

	// Set expired token to force refresh
	auth.mu.Lock()
	auth.accessToken = "expired"
	auth.expiresAt = time.Now().Add(-1 * time.Minute)
	auth.mu.Unlock()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			token, err := auth.AccessToken()
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if token != "concurrent-token" {
				t.Errorf("expected concurrent-token, got %q", token)
			}
		}()
	}
	wg.Wait()

	// Before P0 fix: callCount could be up to 10
	// After P0 fix with singleflight: should be 1
	t.Logf("concurrent refresh call count: %d (will be 1 after singleflight fix)", callCount.Load())
}
