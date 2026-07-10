package tools

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsHTTPImageURL(t *testing.T) {
	tests := map[string]bool{
		"https://example.com/a.png": true,
		"http://example.com/a.png":  true,
		"httpx://example.com/a.png": false,
		"ftp://example.com/a.png":   false,
		"not a url":                 false,
		"":                          false,
	}

	for rawURL, want := range tests {
		if got := IsHTTPImageURL(rawURL); got != want {
			t.Fatalf("IsHTTPImageURL(%q) = %v, want %v", rawURL, got, want)
		}
	}
}

func TestGetCDNImageBytesRejectsOversizedContentLength(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "15728641")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	if _, err := GetCDNImageBytes(context.Background(), server.URL); err == nil {
		t.Fatal("expected oversized image error")
	}
}

func TestBatchGetCDNImageBytesReportsFailures(t *testing.T) {
	pngBytes := testPNG(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(pngBytes)
		case "/bad-status.png":
			http.Error(w, "boom", http.StatusInternalServerError)
		default:
			_, _ = w.Write([]byte("not an image"))
		}
	}))
	defer server.Close()

	result, err := BatchGetCDNImageBytes(context.Background(), []string{
		server.URL + "/ok.png",
		server.URL + "/ok.png",
		server.URL + "/bad-status.png",
		server.URL + "/text.txt",
		"httpx://bad",
	}, 2)
	if err != nil {
		t.Fatalf("BatchGetCDNImageBytes returned error: %v", err)
	}
	if len(result.Images) != 1 {
		t.Fatalf("images = %d, want 1", len(result.Images))
	}
	if len(result.Failures) != 3 {
		t.Fatalf("failures = %d, want 3: %#v", len(result.Failures), result.Failures)
	}
	if !IsPNGMagicNumber(result.Images[server.URL+"/ok.png"]) {
		t.Fatal("downloaded image should be normalized to png")
	}
}

func testPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}
