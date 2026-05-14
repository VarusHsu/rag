package vector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type QdrantStore struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	collection string
}

type VectorRecord struct {
	ID       string
	Vector   []float32
	Metadata map[string]string
}

func NewQdrantStore(host string, port int, apiKey string, collection string) (*QdrantStore, error) {
	return &QdrantStore{
		baseURL:    fmt.Sprintf("http://%s:%d", host, port),
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		collection: collection,
	}, nil
}

func (s *QdrantStore) CreateCollection(ctx context.Context, vectorSize int) error {
	payload := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     vectorSize,
			"distance": "Cosine",
		},
	}

	data, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "PUT",
		s.baseURL+"/collections/"+s.collection, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("create collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create collection failed: %d %s", resp.StatusCode, body)
	}

	return nil
}

func (s *QdrantStore) UpsertVectors(ctx context.Context, records []VectorRecord) error {
	if len(records) == 0 {
		return nil
	}

	var points []map[string]interface{}
	for _, r := range records {
		payload := make(map[string]interface{})
		for k, v := range r.Metadata {
			payload[k] = v
		}

		points = append(points, map[string]interface{}{
			"id":      r.ID,
			"vector":  r.Vector,
			"payload": payload,
		})
	}

	reqBody := map[string]interface{}{
		"points": points,
	}
	data, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(ctx, "PUT",
		s.baseURL+"/collections/"+s.collection+"/points", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("upsert vectors: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upsert failed: %d %s", resp.StatusCode, body)
	}

	return nil
}

func (s *QdrantStore) SearchSimilar(ctx context.Context, vector []float32, limit uint64) ([]VectorRecord, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *QdrantStore) Close() error {
	return nil
}
