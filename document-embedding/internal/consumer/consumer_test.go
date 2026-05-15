package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"document-embedding/internal/parser"
	"document-embedding/internal/vector"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type mockParser struct {
	doc *parser.ParsedDocument
	err error
}

func (m mockParser) Parse(_ []byte, _ string) (*parser.ParsedDocument, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.doc, nil
}

type mockEmbedding struct {
	vectors map[string][]float32
	err     error
	calls   []string
}

func (m *mockEmbedding) Embed(_ context.Context, text string) ([]float32, error) {
	m.calls = append(m.calls, text)
	if m.err != nil {
		return nil, m.err
	}
	if v, ok := m.vectors[text]; ok {
		return v, nil
	}
	return []float32{0.1}, nil
}

type mockVectorStore struct {
	records []vector.VectorRecord
	err     error
}

func (m *mockVectorStore) UpsertVectors(_ context.Context, records []vector.VectorRecord) error {
	m.records = append([]vector.VectorRecord(nil), records...)
	return m.err
}

func TestBuildPointIDDeterministicAndValidUUID(t *testing.T) {
	id1 := buildPointID("2629730c-1016-4184-bc41-33e7cd673141", 0)
	id2 := buildPointID("2629730c-1016-4184-bc41-33e7cd673141", 0)
	id3 := buildPointID("2629730c-1016-4184-bc41-33e7cd673141", 1)

	if id1 != id2 {
		t.Fatalf("buildPointID() not deterministic: %q != %q", id1, id2)
	}
	if id1 == id3 {
		t.Fatalf("buildPointID() should vary by chunk index: %q == %q", id1, id3)
	}
	if len(id1) != 36 {
		t.Fatalf("buildPointID() = %q, want UUID length 36", id1)
	}
}

func TestDownloadFileSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello"))
	}))
	defer server.Close()

	c := &DocumentConsumer{httpClient: server.Client()}
	data, err := c.downloadFile(server.URL)
	if err != nil {
		t.Fatalf("downloadFile() error = %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("downloadFile() = %q, want hello", string(data))
	}
}

func TestDownloadFileHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	c := &DocumentConsumer{httpClient: server.Client()}
	_, err := c.downloadFile(server.URL)
	if err == nil {
		t.Fatal("downloadFile() error = nil, want error")
	}
}

func TestHandleMessageSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("file body"))
	}))
	defer server.Close()

	emb := &mockEmbedding{vectors: map[string][]float32{
		"first chunk":  {1, 2},
		"second chunk": {3, 4},
	}}
	store := &mockVectorStore{}
	c := &DocumentConsumer{
		parser:      mockParser{doc: &parser.ParsedDocument{FileName: "test.txt", Chunks: []parser.Chunk{{Content: "first chunk", Page: 1}, {Content: "second chunk", Page: 2}}}},
		embedding:   emb,
		vectorStore: store,
		logger:      zap.NewNop(),
		httpClient:  server.Client(),
	}

	msgBody, err := json.Marshal(UploadMessage{
		FileID:    "2629730c-1016-4184-bc41-33e7cd673141",
		FileName:  "test.txt",
		FileURL:   server.URL,
		UserID:    "user-1",
		ObjectKey: "uploads/user-1/test.txt",
	})
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}

	if err := c.handleMessage(context.Background(), amqp.Delivery{Body: msgBody}); err != nil {
		t.Fatalf("handleMessage() error = %v", err)
	}

	if len(emb.calls) != 2 {
		t.Fatalf("Embed() calls = %d, want 2", len(emb.calls))
	}
	if len(store.records) != 2 {
		t.Fatalf("upserted records = %d, want 2", len(store.records))
	}
	if store.records[0].Metadata["file_id"] != "2629730c-1016-4184-bc41-33e7cd673141" {
		t.Fatalf("metadata file_id = %q, want original file id", store.records[0].Metadata["file_id"])
	}
	if store.records[0].Metadata["chunk_index"] != "0" || store.records[1].Metadata["chunk_index"] != "1" {
		t.Fatalf("chunk indexes = %q,%q, want 0,1", store.records[0].Metadata["chunk_index"], store.records[1].Metadata["chunk_index"])
	}
	if store.records[0].ID == "2629730c-1016-4184-bc41-33e7cd673141-0" {
		t.Fatalf("record ID = %q, want UUID not raw fileID-index", store.records[0].ID)
	}
}

func TestHandleMessageReturnsEmbedError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("file body"))
	}))
	defer server.Close()

	c := &DocumentConsumer{
		parser:      mockParser{doc: &parser.ParsedDocument{FileName: "test.txt", Chunks: []parser.Chunk{{Content: "chunk", Page: 1}}}},
		embedding:   &mockEmbedding{err: errors.New("embed failed")},
		vectorStore: &mockVectorStore{},
		logger:      zap.NewNop(),
		httpClient:  &http.Client{Timeout: time.Second},
	}

	msgBody, err := json.Marshal(UploadMessage{FileID: "file-1", FileName: "test.txt", FileURL: server.URL})
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}

	err = c.handleMessage(context.Background(), amqp.Delivery{Body: msgBody})
	if err == nil {
		t.Fatal("handleMessage() error = nil, want error")
	}
	if err.Error() != "embed chunk 0: embed failed" {
		t.Fatalf("handleMessage() error = %q, want wrapped embed error", err.Error())
	}
}

func TestHandleMessageReturnsUpsertError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("file body"))
	}))
	defer server.Close()

	c := &DocumentConsumer{
		parser:      mockParser{doc: &parser.ParsedDocument{FileName: "test.txt", Chunks: []parser.Chunk{{Content: "chunk", Page: 1}}}},
		embedding:   &mockEmbedding{},
		vectorStore: &mockVectorStore{err: errors.New("upsert failed")},
		logger:      zap.NewNop(),
		httpClient:  server.Client(),
	}

	msgBody, err := json.Marshal(UploadMessage{FileID: "file-1", FileName: "test.txt", FileURL: server.URL})
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}

	err = c.handleMessage(context.Background(), amqp.Delivery{Body: msgBody})
	if err == nil {
		t.Fatal("handleMessage() error = nil, want error")
	}
	if err.Error() != "upsert vectors: upsert failed" {
		t.Fatalf("handleMessage() error = %q, want wrapped upsert error", err.Error())
	}
}
