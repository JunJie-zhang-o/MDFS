package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"mdfs/internal/config"
	"mdfs/internal/filebrowser"
	"mdfs/internal/httpserver"
	"mdfs/internal/webui"
)

var version = "dev"

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

	shares := make([]filebrowser.Share, 0, len(cfg.Shares))
	for _, share := range cfg.Shares {
		shares = append(shares, filebrowser.Share{Name: share.Name, Path: share.Path})
	}
	browser, err := filebrowser.NewShares(shares)
	if err != nil {
		log.Fatalf("open shared directory: %v", err)
	}

	assets, err := webui.Files()
	if err != nil {
		log.Fatalf("open embedded web UI: %v", err)
	}

	handler, err := httpserver.New(browser, assets, cfg, version)
	if err != nil {
		log.Fatalf("configure HTTP server: %v", err)
	}
	log.Printf("serving %s on http://%s", browser.Root(), cfg.Server.Listen)
	server := &http.Server{Addr: cfg.Server.Listen, Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	if cfg.Server.TLS.CertFile != "" {
		log.Printf("TLS enabled")
		err = server.ListenAndServeTLS(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile)
	} else {
		err = server.ListenAndServe()
	}
	if err != nil {
		log.Fatal(err)
	}
}
