package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/webdav"

	"mdfs/internal/audit"
	"mdfs/internal/auth"
	"mdfs/internal/config"
	"mdfs/internal/filebrowser"
)

type Server struct {
	browser *filebrowser.Browser
	assets  fs.FS
	config  config.Config
	auth    *auth.Manager
	audit   *audit.Logger
	ip      *ipRules
	version string
	dav     http.Handler
}

func New(browser *filebrowser.Browser, assets fs.FS, cfg config.Config, version string) (http.Handler, error) {
	ip, err := newIPRules(cfg.Access)
	if err != nil {
		return nil, err
	}
	logger, err := audit.New(cfg.Logging.Directory)
	if err != nil {
		return nil, fmt.Errorf("open audit log: %w", err)
	}
	if version == "" {
		version = "dev"
	}
	server := &Server{browser: browser, assets: assets, config: cfg, auth: auth.New(cfg), audit: logger, ip: ip, version: version}
	server.dav = &webdav.Handler{Prefix: strings.TrimSuffix(cfg.WebDAV.Prefix, "/"), FileSystem: davFS{browser: browser}, LockSystem: webdav.NewMemLS()}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/meta", server.meta)
	mux.HandleFunc("GET /api/v1/session", server.sessionStatus)
	mux.HandleFunc("POST /api/v1/session", server.login)
	mux.HandleFunc("DELETE /api/v1/session", server.logout)
	mux.HandleFunc("GET /api/v1/files", server.listFiles)
	mux.HandleFunc("GET /api/v1/search", server.search)
	mux.HandleFunc("GET /api/v1/content", server.content)
	mux.HandleFunc("GET /api/v1/archive", server.archive)
	mux.HandleFunc("POST /api/v1/uploads", server.upload)
	mux.HandleFunc("POST /api/v1/directories", server.createDirectory)
	mux.HandleFunc("POST /api/v1/text-files", server.createText)
	mux.HandleFunc("PUT /api/v1/text-files", server.updateText)
	mux.HandleFunc("PATCH /api/v1/files", server.rename)
	mux.HandleFunc("DELETE /api/v1/files", server.deleteFile)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "API endpoint not found")
	})
	if *cfg.WebDAV.Enabled {
		prefix := strings.TrimSuffix(cfg.WebDAV.Prefix, "/")
		mux.Handle(prefix+"/", server.webDAV())
		mux.HandleFunc(prefix, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, prefix+"/", http.StatusPermanentRedirect)
		})
	}
	mux.Handle("/", spaHandler(assets))
	return server.middleware(mux), nil
}

func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := s.ip.clientIP(r)
		if !s.ip.allowed(clientIP) {
			writeError(w, http.StatusForbidden, "ip_denied", "client IP is not allowed")
			return
		}
		started := time.Now()
		recorder := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		principal := s.auth.FromRequest(r)
		username := principal.Name
		if basicUser, _, ok := r.BasicAuth(); ok {
			username = basicUser
		}
		s.audit.Write(audit.Event{Time: started.UTC(), User: username, IP: clientIP.String(), Method: r.Method, Path: r.URL.Path, Status: recorder.status, Bytes: recorder.bytes, Duration: time.Since(started)})
	})
}

func (s *Server) webDAV() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal := s.auth.Anonymous()
		if username, password, ok := r.BasicAuth(); ok {
			var valid bool
			principal, valid = s.auth.Basic(username, password)
			if !valid {
				w.Header().Set("WWW-Authenticate", `Basic realm="MDFS WebDAV"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		virtual := strings.TrimPrefix(r.URL.Path, strings.TrimSuffix(s.config.WebDAV.Prefix, "/"))
		if virtual == "" {
			virtual = "/"
		}
		permissions := principal.Policy.For(virtual)
		allowed := principal.Policy.CanDiscover(virtual)
		switch r.Method {
		case http.MethodDelete:
			allowed = permissions.Delete
		case "COPY":
			allowed = permissions.Read && principal.Policy.For(webDAVDestination(r, s.config.WebDAV.Prefix)).Write
		case "MOVE":
			allowed = permissions.Write && principal.Policy.For(webDAVDestination(r, s.config.WebDAV.Prefix)).Write
		case http.MethodPut, "MKCOL", "PROPPATCH", "LOCK", "UNLOCK", http.MethodPost:
			allowed = permissions.Write
		case "OPTIONS":
			allowed = true
		}
		if !allowed {
			if !principal.Authenticated {
				w.Header().Set("WWW-Authenticate", `Basic realm="MDFS WebDAV"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		s.dav.ServeHTTP(w, r)
	})
}

func webDAVDestination(r *http.Request, prefix string) string {
	destination := r.Header.Get("Destination")
	if parsed, err := url.Parse(destination); err == nil && parsed.Path != "" {
		destination = parsed.Path
	}
	destination = strings.TrimPrefix(destination, strings.TrimSuffix(prefix, "/"))
	if destination == "" {
		return "/"
	}
	return destination
}

func (s *Server) principal(r *http.Request) auth.Principal { return s.auth.FromRequest(r) }

func requireOrigin(w http.ResponseWriter, r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil || !strings.EqualFold(parsed.Host, r.Host) {
		writeError(w, http.StatusForbidden, "invalid_origin", "request origin is not allowed")
		return false
	}
	return true
}

func spaHandler(assets fs.FS) http.Handler {
	files := http.FileServer(http.FS(assets))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if requested == "." || requested == "" {
			requested = "index.html"
		}
		if info, err := fs.Stat(assets, requested); err == nil && !info.IsDir() {
			files.ServeHTTP(w, r)
			return
		}
		index, err := fs.ReadFile(assets, "index.html")
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "ui_unavailable", "web UI has not been built")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Length", strconv.Itoa(len(index)))
		_, _ = w.Write(index)
	})
}

func writeFileError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, filebrowser.ErrInvalidPath), errors.Is(err, filebrowser.ErrVirtualRoot):
		writeError(w, http.StatusBadRequest, "invalid_path", "invalid path")
	case errors.Is(err, filebrowser.ErrConflict), errors.Is(err, os.ErrExist):
		writeError(w, http.StatusConflict, "conflict", "path already exists")
	case errors.Is(err, os.ErrNotExist):
		writeError(w, http.StatusNotFound, "not_found", "path not found")
	case errors.Is(err, os.ErrPermission):
		writeError(w, http.StatusForbidden, "forbidden", "permission denied")
	default:
		writeError(w, http.StatusInternalServerError, "filesystem_error", "unable to access path")
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	written, err := r.ResponseWriter.Write(data)
	r.bytes += int64(written)
	return written, err
}

func (r *responseRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }
