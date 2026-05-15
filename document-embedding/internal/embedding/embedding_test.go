package embedding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEmbedSupportsOpenAIResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want Bearer test-key", got)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}

		var req EmbeddingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Input != "hello" || req.Model != "test-model" {
			t.Fatalf("request = %#v, want input/model populated", req)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1,0.2,0.3]}]}`))
	}))
	defer server.Close()

	client := NewEmbeddingClient(server.URL, "test-key", "test-model")
	got, err := client.Embed(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if len(got) != 3 || got[0] != float32(0.1) || got[2] != float32(0.3) {
		t.Fatalf("Embed() = %#v, want [0.1 0.2 0.3]", got)
	}
}

func TestEmbedSupportsOllamaResponseWithoutAuthHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("Authorization = %q, want empty", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"embeddings":[[1.5,2.5]]}`))
	}))
	defer server.Close()

	client := NewEmbeddingClient(server.URL, "", "bge-m3")
	got, err := client.Embed(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if len(got) != 2 || got[0] != float32(1.5) || got[1] != float32(2.5) {
		t.Fatalf("Embed() = %#v, want [1.5 2.5]", got)
	}
}

func TestEmbedReturnsAPIErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer server.Close()

	client := NewEmbeddingClient(server.URL, "", "bge-m3")
	_, err := client.Embed(context.Background(), "hello")
	if err == nil {
		t.Fatal("Embed() error = nil, want error")
	}
	if got := err.Error(); got == "" || !strings.Contains(got, "status=502") || !strings.Contains(got, "boom") {
		t.Fatalf("Embed() error = %q, want status and body", got)
	}
}

func TestEmbedReturnsErrorForEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	client := NewEmbeddingClient(server.URL, "", "bge-m3")
	_, err := client.Embed(context.Background(), "hello")
	if err == nil {
		t.Fatal("Embed() error = nil, want error")
	}
	if err.Error() != "no embeddings in response" {
		t.Fatalf("Embed() error = %q, want no embeddings in response", err.Error())
	}
}
