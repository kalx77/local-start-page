package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/kalx77/local-start-page/config"
)

// testFS is a minimal in-memory web filesystem.
var testFS = fstest.MapFS{
	"index.html": {Data: []byte(`<!DOCTYPE html><html><body></body></html>`)},
	"style.css":  {Data: []byte(`body{}`)},
	"app.js":     {Data: []byte(`'use strict';`)},
}

func newTestMux(t *testing.T, cfgPath string) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	registerHandlers(mux, cfgPath, testFS)
	return mux
}

func saveTestConfig(t *testing.T, cfg *config.Config) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := config.Save(path, cfg); err != nil {
		t.Fatalf("saveTestConfig: %v", err)
	}
	return path
}

// ── GET /api/config ──────────────────────────────────────────────────────────

func TestGetConfig_OK(t *testing.T) {
	cfgPath := saveTestConfig(t, &config.Config{
		Background: "#aabbcc",
		Port:       1221,
		Groups:     []config.Group{{Name: "G1", X: 0, Y: 0, W: 2, H: 1}},
	})
	mux := newTestMux(t, cfgPath)

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/config", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", w.Code)
	}
	var got config.Config
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Background != "#aabbcc" {
		t.Errorf("background: got %q, want #aabbcc", got.Background)
	}
	if len(got.Groups) != 1 || got.Groups[0].Name != "G1" {
		t.Errorf("groups: %+v", got.Groups)
	}
}

func TestGetConfig_ContentType(t *testing.T) {
	cfgPath := saveTestConfig(t, &config.Config{})
	mux := newTestMux(t, cfgPath)

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/config", nil))

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type: got %q, want application/json", ct)
	}
}

// ── POST /api/config ─────────────────────────────────────────────────────────

func TestPostConfig_Saves(t *testing.T) {
	cfgPath := saveTestConfig(t, &config.Config{Background: "#000000"})
	mux := newTestMux(t, cfgPath)

	updated := config.Config{
		Background: "#ffffff",
		Port:       9090,
		Groups: []config.Group{
			{Name: "New", Color: "#112233", X: 1, Y: 2, W: 3, H: 2},
		},
	}
	body, _ := json.Marshal(updated)
	req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want 204", w.Code)
	}

	saved, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Background != "#ffffff" {
		t.Errorf("saved background: got %q, want #ffffff", saved.Background)
	}
	if len(saved.Groups) != 1 {
		t.Fatalf("saved groups len: got %d, want 1", len(saved.Groups))
	}
	if saved.Groups[0].H != 2 {
		t.Errorf("saved group H: got %d, want 2", saved.Groups[0].H)
	}
	if saved.Groups[0].Collapsed {
		t.Error("saved group Collapsed: got true, want false")
	}
}

func TestPostConfig_InvalidJSON(t *testing.T) {
	cfgPath := saveTestConfig(t, &config.Config{})
	mux := newTestMux(t, cfgPath)

	req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader([]byte("not json {")))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

// ── Method not allowed ────────────────────────────────────────────────────────

func TestConfigMethodNotAllowed(t *testing.T) {
	cfgPath := saveTestConfig(t, &config.Config{})
	mux := newTestMux(t, cfgPath)

	for _, method := range []string{http.MethodPut, http.MethodDelete, http.MethodPatch} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(method, "/api/config", nil))
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s /api/config: got %d, want 405", method, w.Code)
		}
	}
}

// ── GET /bg ──────────────────────────────────────────────────────────────────

func TestGetBg_NotFound(t *testing.T) {
	cfgPath := saveTestConfig(t, &config.Config{})
	mux := newTestMux(t, cfgPath)

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/bg", nil))

	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", w.Code)
	}
}

// ── GET / (index) ────────────────────────────────────────────────────────────

func TestServeIndex(t *testing.T) {
	cfgPath := saveTestConfig(t, &config.Config{})
	mux := newTestMux(t, cfgPath)

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", w.Code)
	}
}

func TestPostConfig_LinkOrderPreserved(t *testing.T) {
	cfgPath := saveTestConfig(t, &config.Config{})
	mux := newTestMux(t, cfgPath)

	// POST with links in specific order
	ordered := config.Config{
		Groups: []config.Group{{
			Name: "G", X: 0, Y: 0, W: 1, H: 1,
			Links: []config.Link{
				{Name: "B", URL: "https://b.example.com"},
				{Name: "A", URL: "https://a.example.com"},
				{Name: "C", URL: "https://c.example.com"},
			},
		}},
	}
	body, _ := json.Marshal(ordered)
	req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("POST status: got %d, want 204", w.Code)
	}

	// GET and verify same order
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/api/config", nil))
	var got config.Config
	if err := json.NewDecoder(w2.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	want := []string{"B", "A", "C"}
	if len(got.Groups[0].Links) != 3 {
		t.Fatalf("links len: got %d, want 3", len(got.Groups[0].Links))
	}
	for i, name := range want {
		if got.Groups[0].Links[i].Name != name {
			t.Errorf("link[%d]: got %q, want %q", i, got.Groups[0].Links[i].Name, name)
		}
	}
}

func TestServeStaticFile(t *testing.T) {
	cfgPath := saveTestConfig(t, &config.Config{})
	mux := newTestMux(t, cfgPath)

	for _, path := range []string{"/style.css", "/app.js"} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusOK {
			t.Errorf("GET %s: got %d, want 200", path, w.Code)
		}
	}
}
