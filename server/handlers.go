package server

import (
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/kalx77/local-start-page/config"
)

func registerHandlers(mux *http.ServeMux, configPath string, webFS fs.FS) {
	fileServer := http.FileServer(http.FS(webFS))

	// Serve index.html at root, delegate everything else to the file server
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFileFS(w, r, webFS, "index.html")
			return
		}
		fileServer.ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			cfg, err := config.Load(configPath)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(cfg)

		case http.MethodPost:
			var cfg config.Config
			if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := config.Save(configPath, &cfg); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/upload/background", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		file, header, err := r.FormFile("image")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		ext := strings.ToLower(filepath.Ext(header.Filename))
		configDir := filepath.Dir(configPath)

		// Remove any existing background file
		for _, e := range []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".avif"} {
			os.Remove(filepath.Join(configDir, "background"+e))
		}

		dst, err := os.Create(filepath.Join(configDir, "background"+ext))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer dst.Close()
		if _, err := io.Copy(dst, file); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"url": "/bg"})
	})

	mux.HandleFunc("/bg", func(w http.ResponseWriter, r *http.Request) {
		configDir := filepath.Dir(configPath)
		for _, ext := range []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".avif"} {
			p := filepath.Join(configDir, "background"+ext)
			if _, err := os.Stat(p); err == nil {
				http.ServeFile(w, r, p)
				return
			}
		}
		http.NotFound(w, r)
	})
}
