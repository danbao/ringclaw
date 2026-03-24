package ringcentral

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestClientWithServer(handler http.HandlerFunc) (*Client, *httptest.Server) {
	srv := httptest.NewServer(handler)
	auth := &Auth{
		accessToken: "test-token",
		expiresAt:   time.Now().Add(1 * time.Hour),
		httpClient:  &http.Client{},
		serverURL:   srv.URL,
	}
	client := &Client{
		serverURL:  srv.URL,
		chatID:     "test-chat",
		auth:       auth,
		httpClient: &http.Client{},
	}
	return client, srv
}

func TestSendPost_Success(t *testing.T) {
	client, srv := newTestClientWithServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Post{ID: "p1", Text: "hello"})
	})
	defer srv.Close()

	post, err := client.SendPost(context.Background(), "chat1", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if post.ID != "p1" {
		t.Errorf("expected post ID p1, got %s", post.ID)
	}
}

func TestSendPost_HTTPError(t *testing.T) {
	client, srv := newTestClientWithServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	})
	defer srv.Close()

	_, err := client.SendPost(context.Background(), "chat1", "hello")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestUpdatePost_Success(t *testing.T) {
	client, srv := newTestClientWithServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Post{ID: "p1", Text: "updated"})
	})
	defer srv.Close()

	post, err := client.UpdatePost(context.Background(), "chat1", "p1", "updated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if post.Text != "updated" {
		t.Errorf("expected text 'updated', got %q", post.Text)
	}
}

func TestUploadFile_Success(t *testing.T) {
	client, srv := newTestClientWithServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]FileUploadResponse{{ID: "f1", Name: "test.png"}})
	})
	defer srv.Close()

	resp, err := client.UploadFile(context.Background(), "chat1", "test.png", []byte("data"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "f1" {
		t.Errorf("expected file ID f1, got %s", resp.ID)
	}
}

func TestListPosts_Pagination(t *testing.T) {
	client, srv := newTestClientWithServer(func(w http.ResponseWriter, r *http.Request) {
		rc := r.URL.Query().Get("recordCount")
		if rc != "50" {
			t.Errorf("expected recordCount=50, got %s", rc)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PostList{Records: []Post{{ID: "p1"}}})
	})
	defer srv.Close()

	list, err := client.ListPosts(context.Background(), "chat1", ListPostsOpts{RecordCount: 50})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list.Records) != 1 {
		t.Errorf("expected 1 record, got %d", len(list.Records))
	}
}

func TestInferContentType(t *testing.T) {
	tests := []struct {
		fileName string
		want     string
	}{
		{"photo.png", "image/png"},
		{"photo.jpg", "image/jpeg"},
		{"photo.gif", "image/gif"},
		{"video.mp4", "video/mp4"},
		{"doc.pdf", "application/pdf"},
	}
	for _, tt := range tests {
		got := inferContentType(tt.fileName)
		if got != tt.want {
			t.Errorf("inferContentType(%q) = %q, want %q", tt.fileName, got, tt.want)
		}
	}
}
