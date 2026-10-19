package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/health", nil)

	healthHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "ok" {
		t.Fatalf("expected body %q, got %q", "ok", w.Body.String())
	}
}

func TestVersionHandler(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/version", nil)

	versionHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "1.0.0" {
		t.Fatalf("expected body %q, got %q", "1.0.0", w.Body.String())
	}
}
