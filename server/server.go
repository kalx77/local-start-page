package server

import (
	"embed"
	"net/http"
)

func Start(addr, configPath string, webFS embed.FS) error {
	mux := http.NewServeMux()
	registerHandlers(mux, configPath, webFS)
	return http.ListenAndServe(addr, mux)
}
