package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kuzmin/local-start-page/config"
)

func TestEnsureExists_Creates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := config.EnsureExists(path); err != nil {
		t.Fatalf("EnsureExists: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}
}

func TestEnsureExists_NoOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	original := []byte(`background = "#ffffff"` + "\n")
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}
	if err := config.EnsureExists(path); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Errorf("file was overwritten:\ngot  %q\nwant %q", got, original)
	}
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	want := &config.Config{
		Background: "#112233",
		Port:       8080,
		Groups: []config.Group{
			{
				Name:      "Test",
				Color:     "#aabbcc",
				Collapsed: true,
				X:         0, Y: 0, W: 2, H: 3,
				Links: []config.Link{
					{Name: "Example", URL: "https://example.com", Icon: "🔗"},
				},
			},
		},
	}

	if err := config.Save(path, want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got.Background != want.Background {
		t.Errorf("Background: got %q, want %q", got.Background, want.Background)
	}
	if got.Port != want.Port {
		t.Errorf("Port: got %d, want %d", got.Port, want.Port)
	}
	if len(got.Groups) != 1 {
		t.Fatalf("Groups len: got %d, want 1", len(got.Groups))
	}
	g := got.Groups[0]
	if g.Name != "Test" {
		t.Errorf("Name: got %q, want %q", g.Name, "Test")
	}
	if g.Color != "#aabbcc" {
		t.Errorf("Color: got %q, want %q", g.Color, "#aabbcc")
	}
	if !g.Collapsed {
		t.Error("Collapsed: got false, want true")
	}
	if g.W != 2 {
		t.Errorf("W: got %d, want 2", g.W)
	}
	if g.H != 3 {
		t.Errorf("H: got %d, want 3", g.H)
	}
	if len(g.Links) != 1 {
		t.Fatalf("Links len: got %d, want 1", len(g.Links))
	}
	if g.Links[0].URL != "https://example.com" {
		t.Errorf("Link URL: got %q", g.Links[0].URL)
	}
}

func TestLoad_DefaultFileIsValid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := config.EnsureExists(path); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load default config: %v", err)
	}
	if cfg.Background == "" {
		t.Error("default Background is empty")
	}
	if len(cfg.Groups) == 0 {
		t.Error("default config has no groups")
	}
}

func TestLoad_InvalidTOML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.toml")
	os.WriteFile(path, []byte("not valid toml ][[["), 0644)
	if _, err := config.Load(path); err == nil {
		t.Error("expected error for invalid TOML, got nil")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	if _, err := config.Load("/nonexistent/path/config.toml"); err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestSaveAndLoad_LinkOrderPreserved(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	links := []config.Link{
		{Name: "First", URL: "https://first.example.com", Icon: ""},
		{Name: "Second", URL: "https://second.example.com", Icon: "🔗"},
		{Name: "Third", URL: "https://third.example.com", Icon: ""},
	}
	cfg := &config.Config{
		Groups: []config.Group{{Name: "G", X: 0, Y: 0, W: 1, H: 1, Links: links}},
	}

	if err := config.Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got.Groups[0].Links) != 3 {
		t.Fatalf("links len: got %d, want 3", len(got.Groups[0].Links))
	}
	for i, want := range links {
		if got.Groups[0].Links[i].Name != want.Name {
			t.Errorf("link[%d] name: got %q, want %q", i, got.Groups[0].Links[i].Name, want.Name)
		}
	}
}

func TestSaveAndLoad_ReorderedLinks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")

	// Save original order A, B, C
	cfg := &config.Config{
		Groups: []config.Group{{Name: "G", X: 0, Y: 0, W: 1, H: 1, Links: []config.Link{
			{Name: "A", URL: "https://a.example.com"},
			{Name: "B", URL: "https://b.example.com"},
			{Name: "C", URL: "https://c.example.com"},
		}}},
	}
	if err := config.Save(path, cfg); err != nil {
		t.Fatal(err)
	}

	// Simulate moveLink: swap B and C → order becomes A, C, B
	cfg.Groups[0].Links[1], cfg.Groups[0].Links[2] = cfg.Groups[0].Links[2], cfg.Groups[0].Links[1]
	if err := config.Save(path, cfg); err != nil {
		t.Fatal(err)
	}

	got, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"A", "C", "B"}
	for i, name := range want {
		if got.Groups[0].Links[i].Name != name {
			t.Errorf("link[%d]: got %q, want %q", i, got.Groups[0].Links[i].Name, name)
		}
	}
}
