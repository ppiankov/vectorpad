package stash

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEmbedderAvailableWithModel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			resp := map[string]interface{}{
				"models": []map[string]string{
					{"name": "nomic-embed-text:latest"},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		}
	}))
	defer srv.Close()

	e := NewEmbedderWithURL(srv.URL, "nomic-embed-text")
	if !e.Available() {
		t.Error("should be available when model is listed")
	}
}

func TestEmbedderUnavailableNoModel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"models": []map[string]string{
				{"name": "llama3:latest"},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	e := NewEmbedderWithURL(srv.URL, "nomic-embed-text")
	if e.Available() {
		t.Error("should not be available when model is not listed")
	}
}

func TestEmbedderUnavailableNoServer(t *testing.T) {
	e := NewEmbedderWithURL("http://localhost:1", "nomic-embed-text")
	if e.Available() {
		t.Error("should not be available when server is unreachable")
	}
}

func TestEmbedReturnsVector(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"embedding": []float64{0.1, 0.2, 0.3},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	e := NewEmbedderWithURL(srv.URL, "nomic-embed-text")
	vec, err := e.Embed("hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vec) != 3 {
		t.Fatalf("expected 3-dim vector, got %d", len(vec))
	}
	if vec[0] < 0.09 || vec[0] > 0.11 {
		t.Errorf("expected vec[0] ~0.1, got %f", vec[0])
	}
}

func TestEmbedServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	e := NewEmbedderWithURL(srv.URL, "nomic-embed-text")
	_, err := e.Embed("hello")
	if err == nil {
		t.Error("expected error on server 500")
	}
}

func TestEmbedderAvailabilityCached(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		resp := map[string]interface{}{
			"models": []map[string]string{
				{"name": "nomic-embed-text:latest"},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	e := NewEmbedderWithURL(srv.URL, "nomic-embed-text")
	e.Available()
	e.Available()
	e.Available()
	if calls != 1 {
		t.Errorf("expected 1 availability check, got %d", calls)
	}
}
