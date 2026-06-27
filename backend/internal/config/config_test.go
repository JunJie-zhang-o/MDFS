package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	shared := t.TempDir()
	configFile := filepath.Join(t.TempDir(), "mdfs.toml")
	content := "[server]\nlisten = \"127.0.0.1:8080\"\n[share]\npath = \"" + shared + "\"\n"
	if err := os.WriteFile(configFile, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Server.Listen != "127.0.0.1:8080" || cfg.Share.Path != shared {
		t.Fatalf("Load() = %#v", cfg)
	}
}

func TestLoadRejectsInvalidConfigs(t *testing.T) {
	shared := t.TempDir()
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{name: "invalid TOML", content: "[server", want: "parse TOML"},
		{name: "missing listen", content: "[share]\npath = \"" + shared + "\"", want: "server.listen"},
		{name: "invalid listen", content: "[server]\nlisten = \"8080\"\n[share]\npath = \"" + shared + "\"", want: "server.listen"},
		{name: "missing share", content: "[server]\nlisten = \"127.0.0.1:8080\"", want: "share.path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := filepath.Join(t.TempDir(), "mdfs.toml")
			if err := os.WriteFile(filename, []byte(tt.content), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := Load(filename)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Load() error = %v, want containing %q", err, tt.want)
			}
		})
	}
}
