package main

import (
	"embed"
	"flag"
	"fmt"
	"log"

	"github.com/kuzmin/local-start-page/config"
	"github.com/kuzmin/local-start-page/server"
)

//go:embed web
var webFS embed.FS

func main() {
	configPath := flag.String("config", "./config.toml", "path to config file")
	port := flag.Int("port", 1221, "port to listen on")
	flag.Parse()

	if err := config.EnsureExists(*configPath); err != nil {
		log.Fatalf("failed to create default config: %v", err)
	}

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("starting local-start-page on http://localhost%s", addr)
	if err := server.Start(addr, *configPath, webFS); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
