package vector

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateCollectionCreatesWhenAbsent(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/collections/documents" {
			t.Fatalf("path = %s, want /collections/documents", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := &QdrantStore{baseURL: server.URL, httpClient: server.Client(), collection: "documents"}
	if err := store.CreateCollection(context.Background(), 1024); err != nil {
		t.Fatalf("CreateCollection() error = %v", err)
	}

	vectors := body["vectors"].(map[string]any)
	if vectors["size"].(float64) != 1024 {
		t.Fatalf("size = %v, want 1024", vectors["size"])
	}
	if vectors["distance"].(string) != "Cosine" {
		t.Fatalf("distance = %v, want Cosine", vectors["distance"])
	}
}

func TestCreateCollectionRecreatesOnDimensionMismatch(t *testing.T) {
	var putCount, getCount, deleteCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/collections/documents":
			putCount++
			if putCount == 1 {
				w.WriteHeader(http.StatusConflict)
				return
			}
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && r.URL.Path == "/collections/documents":
			getCount++
			_, _ = w.Write([]byte(`{"result":{"config":{"params":{"vectors":{"size":1536}}}}}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/collections/documents":
			deleteCount++
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	store := &QdrantStore{baseURL: server.URL, httpClient: server.Client(), collection: "documents"}
	if err := store.CreateCollection(context.Background(), 1024); err != nil {
		t.Fatalf("CreateCollection() error = %v", err)
	}
	if putCount != 2 || getCount != 1 || deleteCount != 1 {
		t.Fatalf("calls = put:%d get:%d delete:%d, want 2/1/1", putCount, getCount, deleteCount)
	}
}

func TestCreateCollectionKeepsExistingWhenDimensionMatches(t *testing.T) {
	var putCount, getCount, deleteCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/collections/documents":
			putCount++
			w.WriteHeader(http.StatusConflict)
		case r.Method == http.MethodGet && r.URL.Path == "/collections/documents":
			getCount++
			_, _ = w.Write([]byte(`{"result":{"config":{"params":{"vectors":{"size":1024}}}}}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/collections/documents":
			deleteCount++
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	store := &QdrantStore{baseURL: server.URL, httpClient: server.Client(), collection: "documents"}
	if err := store.CreateCollection(context.Background(), 1024); err != nil {
		t.Fatalf("CreateCollection() error = %v", err)
	}
	if putCount != 1 || getCount != 1 || deleteCount != 0 {
		t.Fatalf("calls = put:%d get:%d delete:%d, want 1/1/0", putCount, getCount, deleteCount)
	}
}

func TestUpsertVectorsSendsPoints(t *testing.T) {
	var reqBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/collections/documents/points" {
			t.Fatalf("path = %s, want /collections/documents/points", r.URL.Path)
		}
		var err error
		reqBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := &QdrantStore{baseURL: server.URL, httpClient: server.Client(), collection: "documents"}
	err := store.UpsertVectors(context.Background(), []VectorRecord{{
		ID:     "123e4567-e89b-12d3-a456-426614174000",
		Vector: []float32{0.1, 0.2},
		Metadata: map[string]string{
			"file_id": "file-1",
		},
	}})
	if err != nil {
		t.Fatalf("UpsertVectors() error = %v", err)
	}

	var payload struct {
		Points []struct {
			ID      string            `json:"id"`
			Vector  []float64         `json:"vector"`
			Payload map[string]string `json:"payload"`
		} `json:"points"`
	}
	if err := json.Unmarshal(reqBody, &payload); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	if len(payload.Points) != 1 {
		t.Fatalf("points len = %d, want 1", len(payload.Points))
	}
	if payload.Points[0].ID != "123e4567-e89b-12d3-a456-426614174000" {
		t.Fatalf("point ID = %q, want UUID", payload.Points[0].ID)
	}
	if payload.Points[0].Payload["file_id"] != "file-1" {
		t.Fatalf("payload file_id = %q, want file-1", payload.Points[0].Payload["file_id"])
	}
}

func TestCreateCollectionRejectsInvalidVectorSize(t *testing.T) {
	store := &QdrantStore{}
	if err := store.CreateCollection(context.Background(), 0); err == nil {
		t.Fatal("CreateCollection() error = nil, want error")
	}
}
