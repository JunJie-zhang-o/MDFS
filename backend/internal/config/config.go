package config

import (
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Server Server `toml:"server"`
	Share  Share  `toml:"share"`
}

type Server struct {
	Listen string `toml:"listen"`
}

type Share struct {
	Path string `toml:"path"`
}

func Load(filename string) (Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse TOML: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.Server.Listen == "" {
		return fmt.Errorf("server.listen is required")
	}
	_, portText, err := net.SplitHostPort(c.Server.Listen)
	if err != nil {
		return fmt.Errorf("server.listen: %w", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("server.listen has invalid port %q", portText)
	}
	if c.Share.Path == "" {
		return fmt.Errorf("share.path is required")
	}
	info, err := os.Stat(c.Share.Path)
	if err != nil {
		return fmt.Errorf("share.path: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("share.path is not a directory")
	}
	return nil
}
