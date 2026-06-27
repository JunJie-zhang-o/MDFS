package httpserver

import (
	"encoding/json"
	"errors"
	"io/fs"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"

	"mdfs/internal/filebrowser"
)

func New(browser *filebrowser.Browser, assets fs.FS) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/files", listFiles(browser))
	mux.HandleFunc("GET /api/download", downloadFile(browser))
	mux.HandleFunc("/api/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "API endpoint not found")
	})
	mux.Handle("/", spaHandler(assets))
	return mux
}

func listFiles(browser *filebrowser.Browser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		listing, err := browser.List(r.URL.Query().Get("path"))
		if err != nil {
			writeFileError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listing)
	}
}

func downloadFile(browser *filebrowser.Browser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file, info, err := browser.Open(r.URL.Query().Get("path"))
		if err != nil {
			writeFileError(w, err)
			return
		}
		defer file.Close()

		w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(info.Name()))
		if contentType := mime.TypeByExtension(path.Ext(info.Name())); contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		http.ServeContent(w, r, info.Name(), info.ModTime(), file)
	}
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
			writeError(w, http.StatusServiceUnavailable, "web UI has not been built")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Length", strconv.Itoa(len(index)))
		w.Write(index)
	})
}

func writeFileError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, filebrowser.ErrInvalidPath):
		writeError(w, http.StatusBadRequest, "invalid path")
	case errors.Is(err, os.ErrNotExist):
		writeError(w, http.StatusNotFound, "path not found")
	case errors.Is(err, os.ErrPermission):
		writeError(w, http.StatusForbidden, "permission denied")
	default:
		writeError(w, http.StatusInternalServerError, "unable to access path")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
