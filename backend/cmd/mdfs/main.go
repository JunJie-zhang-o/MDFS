package main

import (
	"flag"
	"log"
	"net/http"

	"mdfs/internal/config"
	"mdfs/internal/filebrowser"
	"mdfs/internal/httpserver"
	"mdfs/internal/webui"
)

func main() {
	configPath := flag.String("config", "", "path to the TOML configuration file")
	flag.Parse()

	if *configPath == "" {
		log.Fatal("--config is required")
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	browser, err := filebrowser.New(cfg.Share.Path)
	if err != nil {
		log.Fatalf("open shared directory: %v", err)
	}

	assets, err := webui.Files()
	if err != nil {
		log.Fatalf("open embedded web UI: %v", err)
	}

	handler := httpserver.New(browser, assets)
	log.Printf("serving %s on http://%s", browser.Root(), cfg.Server.Listen)
	if err := http.ListenAndServe(cfg.Server.Listen, handler); err != nil {
		log.Fatal(err)
	}
}
