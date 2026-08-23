package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

const (
	DefaultSessionTimeout = 60 * time.Minute
	DefaultMaxUploadBytes = int64(1_000_000_000_000)
	DefaultSearchResults  = 1000
)

type Config struct {
	Server    Server    `toml:"server"`
	Share     Share     `toml:"share"`
	Shares    []Share   `toml:"shares"`
	Anonymous Principal `toml:"anonymous"`
	Users     []User    `toml:"users"`
	Access    Access    `toml:"access"`
	WebDAV    WebDAV    `toml:"webdav"`
	UI        UI        `toml:"ui"`
	Features  Features  `toml:"features"`
	Logging   Logging   `toml:"logging"`
}

type Server struct {
	Listen         string `toml:"listen"`
	PublicURL      string `toml:"public_url"`
	SessionTimeout string `toml:"session_timeout"`
	TLS            TLS    `toml:"tls"`
}

type TLS struct {
	CertFile string `toml:"cert_file"`
	KeyFile  string `toml:"key_file"`
}

type Share struct {
	Name string `toml:"name"`
	Path string `toml:"path"`
}

type Principal struct {
	Permissions string `toml:"permissions"`
	Rules       []Rule `toml:"rules"`
}

type User struct {
	Name        string `toml:"name"`
	Password    string `toml:"password"`
	Permissions string `toml:"permissions"`
	Rules       []Rule `toml:"rules"`
}

type Rule struct {
	Pattern     string `toml:"pattern"`
	Permissions string `toml:"permissions"`
}

type Access struct {
	Allow          []string `toml:"allow"`
	Deny           []string `toml:"deny"`
	TrustedProxies []string `toml:"trusted_proxies"`
}

type WebDAV struct {
	Enabled *bool  `toml:"enabled"`
	Prefix  string `toml:"prefix"`
}

type UI struct {
	Title           string `toml:"title"`
	Notice          string `toml:"notice"`
	DefaultLanguage string `toml:"default_language"`
}

type Features struct {
	MaxUploadBytes            int64 `toml:"max_upload_bytes"`
	MaxSearchResults          int   `toml:"max_search_results"`
	LeafDirectoryDownloadOnly bool  `toml:"leaf_directory_download_only"`
	ImagePreview              *bool `toml:"image_preview"`
}

type Logging struct {
	Directory string `toml:"directory"`
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
	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c *Config) applyDefaults() {
	if len(c.Shares) == 0 && c.Share.Path != "" {
		c.Shares = []Share{c.Share}
	}
	if c.Server.SessionTimeout == "" {
		c.Server.SessionTimeout = DefaultSessionTimeout.String()
	}
	if c.WebDAV.Enabled == nil {
		enabled := true
		c.WebDAV.Enabled = &enabled
	}
	if c.WebDAV.Prefix == "" {
		c.WebDAV.Prefix = "/webdav"
	}
	if c.UI.Title == "" {
		c.UI.Title = "MDFS"
	}
	if c.UI.DefaultLanguage == "" {
		c.UI.DefaultLanguage = "zh-CN"
	}
	if c.Features.MaxUploadBytes == 0 {
		c.Features.MaxUploadBytes = DefaultMaxUploadBytes
	}
	if c.Features.MaxSearchResults == 0 {
		c.Features.MaxSearchResults = DefaultSearchResults
	}
	if c.Features.ImagePreview == nil {
		enabled := true
		c.Features.ImagePreview = &enabled
	}
	for i := range c.Shares {
		if c.Shares[i].Name == "" {
			c.Shares[i].Name = filepath.Base(filepath.Clean(c.Shares[i].Path))
		}
	}
}

func (c Config) SessionDuration() time.Duration {
	duration, _ := time.ParseDuration(c.Server.SessionTimeout)
	return duration
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
	if _, err := time.ParseDuration(c.Server.SessionTimeout); err != nil {
		return fmt.Errorf("server.session_timeout: %w", err)
	}
	if (c.Server.TLS.CertFile == "") != (c.Server.TLS.KeyFile == "") {
		return fmt.Errorf("server.tls cert_file and key_file must be configured together")
	}
	if len(c.Shares) == 0 {
		return fmt.Errorf("at least one share is required")
	}
	names := make(map[string]struct{}, len(c.Shares))
	for i, share := range c.Shares {
		if share.Name == "" || strings.ContainsAny(share.Name, `/\\`) || share.Name == "." || share.Name == ".." {
			return fmt.Errorf("shares[%d].name is invalid", i)
		}
		if _, exists := names[strings.ToLower(share.Name)]; exists {
			return fmt.Errorf("duplicate share name %q", share.Name)
		}
		names[strings.ToLower(share.Name)] = struct{}{}
		info, err := os.Stat(share.Path)
		if err != nil {
			return fmt.Errorf("shares[%d].path: %w", i, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("shares[%d].path is not a directory", i)
		}
	}
	if c.Features.MaxUploadBytes < 1 {
		return fmt.Errorf("features.max_upload_bytes must be positive")
	}
	if c.Features.MaxSearchResults < 1 {
		return fmt.Errorf("features.max_search_results must be positive")
	}
	if !strings.HasPrefix(c.WebDAV.Prefix, "/") || c.WebDAV.Prefix == "/" {
		return fmt.Errorf("webdav.prefix must be an absolute non-root path")
	}
	if err := validatePermissions("anonymous.permissions", c.Anonymous.Permissions); err != nil {
		return err
	}
	users := map[string]struct{}{}
	for i, user := range c.Users {
		if user.Name == "" || user.Password == "" {
			return fmt.Errorf("users[%d] name and password are required", i)
		}
		key := strings.ToLower(user.Name)
		if _, exists := users[key]; exists {
			return fmt.Errorf("duplicate user %q", user.Name)
		}
		users[key] = struct{}{}
		if err := validatePermissions(fmt.Sprintf("users[%d].permissions", i), user.Permissions); err != nil {
			return err
		}
		for j, rule := range user.Rules {
			if rule.Pattern == "" {
				return fmt.Errorf("users[%d].rules[%d].pattern is required", i, j)
			}
			if err := validatePermissions(fmt.Sprintf("users[%d].rules[%d].permissions", i, j), rule.Permissions); err != nil {
				return err
			}
		}
	}
	return nil
}

func validatePermissions(field, value string) error {
	for _, char := range strings.ToUpper(value) {
		if !strings.ContainsRune("RWD", char) {
			return fmt.Errorf("%s contains invalid permission %q", field, char)
		}
	}
	return nil
}
