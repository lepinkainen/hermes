package tmdb

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"

	"github.com/lepinkainen/hermes/internal/testutil"
)

func TestGetCoverURLByID(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"poster_path":"/poster.jpg"}`))
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	client := NewClient("key",
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
		WithImageBaseURL("https://images.example"),
	)

	cover, err := client.GetCoverURLByID(context.Background(), 101, "movie")
	if err != nil {
		t.Fatalf("GetCoverURLByID error = %v", err)
	}
	if cover != "https://images.example/poster.jpg" {
		t.Fatalf("GetCoverURLByID cover = %s, want %s", cover, "https://images.example/poster.jpg")
	}

	_, err = client.GetCoverURLByID(context.Background(), 101, "unknown")
	if err == nil {
		t.Fatalf("expected error for invalid media type")
	}
}

func TestGetCoverURLByIDNoPoster(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"poster_path":""}`))
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	client := NewClient("key",
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
	)

	// Use unique movie ID to avoid cache collision with other tests
	_, err := client.GetCoverURLByID(context.Background(), 102, "movie")
	if err == nil || err != ErrNoPoster {
		t.Fatalf("expected ErrNoPoster, got %v", err)
	}
}

func TestGetCoverAndMetadataByIDWithMissingPoster(t *testing.T) {
	var calls int32
	handler := func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"poster_path":"","runtime":123}`))
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	client := NewClient("key",
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
	)

	// Use unique movie ID to avoid cache collision with other tests
	cover, meta, err := client.GetCoverAndMetadataByID(context.Background(), 103, "movie")
	if err != nil {
		t.Fatalf("GetCoverAndMetadataByID error = %v", err)
	}
	if cover != "" {
		t.Fatalf("expected no cover, got %s", cover)
	}
	if meta == nil || meta.Runtime == nil || *meta.Runtime != 123 {
		t.Fatalf("expected runtime metadata, got %+v", meta)
	}
	// With caching enabled, both GetCoverURLByID and GetMetadataByID use CachedGetMovieDetails
	// The first call fetches and caches, the second call uses the cache
	// So we expect 1 HTTP call, not 2
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("expected 1 call due to caching, got %d", calls)
	}
}

func TestDownloadAndResizeImage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png encode: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(buf.Bytes())
	}))
	defer server.Close()

	client := NewClient("key", WithHTTPClient(server.Client()))

	env := testutil.NewTestEnv(t)
	path := env.Path("poster.png")

	if err := client.DownloadAndResizeImage(context.Background(), server.URL, path, 0); err != nil {
		t.Fatalf("DownloadAndResizeImage error = %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to be written: %v", err)
	}
}
