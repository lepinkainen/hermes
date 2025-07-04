package datastore

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDatasetteClient_BatchInsert_Success(t *testing.T) {
	// Mock server that returns 200 OK
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := NewDatasetteClient(ts.URL, "testtoken")
	if err := client.Connect(); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	records := []map[string]any{{"foo": "bar"}}
	if err := client.BatchInsert("hermes", "test_table", records); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestDatasetteClient_BatchInsert_APIError(t *testing.T) {
	// Mock server that returns 403 Forbidden with JSON error
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		if err := json.NewEncoder(w).Encode(map[string]any{"error": "forbidden"}); err != nil {
			t.Errorf("Failed to encode error response: %v", err)
		}
	}))
	defer ts.Close()

	client := NewDatasetteClient(ts.URL, "testtoken")
	if err := client.Connect(); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	records := []map[string]any{{"foo": "bar"}}
	if err := client.BatchInsert("hermes", "test_table", records); err == nil {
		t.Errorf("expected error, got nil")
	}
}
