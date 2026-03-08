package server

import (
	"embed"
	"io/fs"
	"net/http"
)

func Start(addr, configPath string, webFS embed.FS) error {
	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	registerHandlers(mux, configPath, sub)
	return http.ListenAndServe(addr, mux)
}
