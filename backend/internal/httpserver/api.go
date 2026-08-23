package httpserver

import (
	"archive/zip"
	"encoding/json"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"mdfs/internal/access"
	"mdfs/internal/auth"
	"mdfs/internal/filebrowser"
)

type apiEntry struct {
	filebrowser.Entry
	Permissions access.Permissions `json:"permissions"`
}

type apiListing struct {
	Path        string             `json:"path"`
	VirtualRoot bool               `json:"virtualRoot"`
	Permissions access.Permissions `json:"permissions"`
	Entries     []apiEntry         `json:"entries"`
}

func (s *Server) meta(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"title": s.config.UI.Title, "version": s.version, "notice": s.config.UI.Notice,
		"defaultLanguage": s.config.UI.DefaultLanguage, "publicURL": s.config.Server.PublicURL,
		"features": map[string]any{"webdav": *s.config.WebDAV.Enabled, "imagePreview": *s.config.Features.ImagePreview, "directoryUpload": true},
	})
}

func (s *Server) sessionStatus(w http.ResponseWriter, r *http.Request) {
	principal := s.principal(r)
	requestPath := r.URL.Query().Get("path")
	if requestPath == "" {
		requestPath = "/"
	}
	writeJSON(w, http.StatusOK, map[string]any{"authenticated": principal.Authenticated, "user": principal.Name, "permissions": principal.Policy.For(requestPath)})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if !requireOrigin(w, r) {
		return
	}
	var input struct{ Username, Password string }
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid login request")
		return
	}
	token, principal, ok := s.auth.Login(input.Username, input.Password)
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid username or password")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: auth.CookieName, Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: r.TLS != nil || s.config.Server.TLS.CertFile != "", MaxAge: int(s.config.SessionDuration().Seconds())})
	writeJSON(w, http.StatusOK, map[string]any{"authenticated": true, "user": principal.Name, "permissions": principal.Policy.For("/")})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if !requireOrigin(w, r) {
		return
	}
	if cookie, err := r.Cookie(auth.CookieName); err == nil {
		s.auth.Logout(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: auth.CookieName, Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listFiles(w http.ResponseWriter, r *http.Request) {
	requestPath := queryPath(r)
	principal := s.principal(r)
	if !principal.Policy.CanDiscover(requestPath) {
		permissionDenied(w, principal)
		return
	}
	listing, err := s.browser.List(requestPath)
	if err != nil {
		writeFileError(w, err)
		return
	}
	entries := make([]apiEntry, 0, len(listing.Entries))
	for _, entry := range listing.Entries {
		if !principal.Policy.CanDiscover(entry.Path) {
			continue
		}
		if entry.PreviewKind == "image" && !*s.config.Features.ImagePreview {
			entry.PreviewKind = "none"
		}
		entries = append(entries, apiEntry{Entry: entry, Permissions: principal.Policy.For(entry.Path)})
	}
	permissions := principal.Policy.For(listing.Path)
	if listing.VirtualRoot {
		permissions.Write = false
		permissions.Delete = false
	}
	writeJSON(w, http.StatusOK, apiListing{Path: listing.Path, VirtualRoot: listing.VirtualRoot, Permissions: permissions, Entries: entries})
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	requestPath := queryPath(r)
	principal := s.principal(r)
	if !principal.Policy.CanDiscover(requestPath) {
		permissionDenied(w, principal)
		return
	}
	entries, truncated, err := s.browser.Search(requestPath, r.URL.Query().Get("query"), s.config.Features.MaxSearchResults, principal.Policy.CanDiscover)
	if err != nil {
		writeFileError(w, err)
		return
	}
	result := make([]apiEntry, 0, len(entries))
	for _, entry := range entries {
		permissions := principal.Policy.For(entry.Path)
		if permissions.Read {
			result = append(result, apiEntry{Entry: entry, Permissions: permissions})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": result, "truncated": truncated})
}

func (s *Server) content(w http.ResponseWriter, r *http.Request) {
	requestPath := queryPath(r)
	principal := s.principal(r)
	if !principal.Policy.For(requestPath).Read {
		permissionDenied(w, principal)
		return
	}
	file, info, err := s.browser.Open(requestPath)
	if err != nil {
		writeFileError(w, err)
		return
	}
	defer file.Close()
	disposition := r.URL.Query().Get("disposition")
	if disposition != "inline" {
		disposition = "attachment"
	}
	w.Header().Set("Content-Disposition", disposition+"; filename*=UTF-8''"+url.PathEscape(info.Name()))
	if contentType := mime.TypeByExtension(filepath.Ext(info.Name())); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}

func (s *Server) archive(w http.ResponseWriter, r *http.Request) {
	requestPath := queryPath(r)
	principal := s.principal(r)
	if !principal.Policy.For(requestPath).Read {
		permissionDenied(w, principal)
		return
	}
	physical, virtual, err := s.browser.Resolve(requestPath)
	if err != nil {
		writeFileError(w, err)
		return
	}
	info, err := os.Stat(physical)
	if err != nil || !info.IsDir() {
		writeFileError(w, filebrowser.ErrInvalidPath)
		return
	}
	if s.config.Features.LeafDirectoryDownloadOnly && containsDirectory(physical) {
		writeError(w, http.StatusForbidden, "not_leaf_directory", "only leaf directories can be downloaded")
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(info.Name()+".zip"))
	rootName := info.Name()
	archive := zip.NewWriter(w)
	err = filepath.WalkDir(physical, func(current string, item fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if item.Type()&os.ModeSymlink != 0 {
			if item.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		relative, err := filepath.Rel(physical, current)
		if err != nil || relative == "." {
			return err
		}
		entryVirtual := path.Join(virtual, filepath.ToSlash(relative))
		if !principal.Policy.For(entryVirtual).Read {
			if item.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := item.Info()
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = path.Join(rootName, filepath.ToSlash(relative))
		if item.IsDir() {
			header.Name += "/"
			_, err = archive.CreateHeader(header)
			return err
		}
		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}
		file, err := os.Open(current)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(writer, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	closeErr := archive.Close()
	if err != nil || closeErr != nil {
		return
	}
}

func (s *Server) createDirectory(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Path string `json:"path"`
	}
	if !s.writeJSONRequest(w, r, &input, input.Path) {
		return
	}
	if !s.requirePermission(w, r, path.Dir(input.Path), "write") {
		return
	}
	if err := s.browser.CreateDir(input.Path); err != nil {
		writeFileError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"path": input.Path})
}

func (s *Server) createText(w http.ResponseWriter, r *http.Request) { s.writeText(w, r, true) }
func (s *Server) updateText(w http.ResponseWriter, r *http.Request) { s.writeText(w, r, false) }

func (s *Server) writeText(w http.ResponseWriter, r *http.Request, createOnly bool) {
	var input struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if !s.writeJSONRequest(w, r, &input, input.Path) {
		return
	}
	permissionPath := input.Path
	if createOnly {
		permissionPath = path.Dir(input.Path)
	}
	if !s.requirePermission(w, r, permissionPath, "write") {
		return
	}
	if err := s.browser.WriteFile(input.Path, strings.NewReader(input.Content), createOnly); err != nil {
		writeFileError(w, err)
		return
	}
	writeJSON(w, map[bool]int{true: http.StatusCreated, false: http.StatusOK}[createOnly], map[string]string{"path": input.Path})
}

func (s *Server) rename(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Path    string `json:"path"`
		NewName string `json:"newName"`
	}
	if !s.writeJSONRequest(w, r, &input, input.Path) {
		return
	}
	if !s.requirePermission(w, r, input.Path, "write") {
		return
	}
	newPath, err := s.browser.Rename(input.Path, input.NewName)
	if err != nil {
		writeFileError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"path": newPath})
}

func (s *Server) deleteFile(w http.ResponseWriter, r *http.Request) {
	if !requireOrigin(w, r) {
		return
	}
	requestPath := queryPath(r)
	if !s.requirePermission(w, r, requestPath, "delete") {
		return
	}
	if err := s.browser.Delete(requestPath); err != nil {
		writeFileError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) upload(w http.ResponseWriter, r *http.Request) {
	if !requireOrigin(w, r) {
		return
	}
	base := queryPath(r)
	if !s.requirePermission(w, r, base, "write") {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, s.config.Features.MaxUploadBytes+(32<<20))
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "upload_too_large", "upload exceeds the configured limit")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	files := r.MultipartForm.File["file"]
	paths := r.MultipartForm.Value["relativePath"]
	if len(files) == 0 {
		writeError(w, http.StatusBadRequest, "missing_file", "at least one file is required")
		return
	}
	created := make([]string, 0, len(files))
	for i, header := range files {
		relative := header.Filename
		if i < len(paths) && paths[i] != "" {
			relative = paths[i]
		}
		relative, ok := safeRelative(relative)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid_path", "invalid upload path")
			return
		}
		destination := path.Join(base, relative)
		if !s.principal(r).Policy.For(destination).Write {
			permissionDenied(w, s.principal(r))
			return
		}
		if err := s.ensureDirectories(path.Dir(destination)); err != nil {
			writeFileError(w, err)
			return
		}
		file, err := header.Open()
		if err != nil {
			writeFileError(w, err)
			return
		}
		err = s.browser.WriteFile(destination, file, true)
		file.Close()
		if err != nil {
			writeFileError(w, err)
			return
		}
		created = append(created, destination)
	}
	writeJSON(w, http.StatusCreated, map[string]any{"paths": created})
}

func (s *Server) ensureDirectories(virtual string) error {
	if virtual == "/" || s.browser.IsShareRoot(virtual) {
		return nil
	}
	if physical, _, err := s.browser.Resolve(virtual); err == nil {
		info, statErr := os.Stat(physical)
		if statErr != nil || !info.IsDir() {
			return filebrowser.ErrConflict
		}
		return nil
	}
	if err := s.ensureDirectories(path.Dir(virtual)); err != nil {
		return err
	}
	return s.browser.CreateDir(virtual)
}

func (s *Server) writeJSONRequest(w http.ResponseWriter, r *http.Request, target any, _ string) bool {
	if !requireOrigin(w, r) {
		return false
	}
	if err := decodeJSON(r, target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON request")
		return false
	}
	return true
}

func (s *Server) requirePermission(w http.ResponseWriter, r *http.Request, requestPath, operation string) bool {
	principal := s.principal(r)
	permissions := principal.Policy.For(requestPath)
	allowed := permissions.Read
	if operation == "write" {
		allowed = permissions.Write
	} else if operation == "delete" {
		allowed = permissions.Delete
	}
	if !allowed {
		permissionDenied(w, principal)
	}
	return allowed
}

func permissionDenied(w http.ResponseWriter, principal auth.Principal) {
	if !principal.Authenticated {
		writeError(w, http.StatusUnauthorized, "authentication_required", "authentication is required")
		return
	}
	writeError(w, http.StatusForbidden, "forbidden", "permission denied")
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 16<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return nil
}

func queryPath(r *http.Request) string {
	value := r.URL.Query().Get("path")
	if value == "" {
		return "/"
	}
	return value
}

func safeRelative(value string) (string, bool) {
	if strings.Contains(value, "\\") || strings.HasPrefix(value, "/") || strings.ContainsRune(value, '\x00') {
		return "", false
	}
	cleaned := path.Clean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", false
	}
	return cleaned, true
}

func containsDirectory(root string) bool {
	entries, err := os.ReadDir(root)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() && entry.Type()&os.ModeSymlink == 0 {
			return true
		}
	}
	return false
}
