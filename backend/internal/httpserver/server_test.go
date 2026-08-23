package httpserver

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"mdfs/internal/config"
	"mdfs/internal/filebrowser"
)

func boolPointer(value bool) *bool { return &value }

func testServer(t *testing.T, permissions string) (http.Handler, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "hello.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	browser, err := filebrowser.New(root)
	if err != nil {
		t.Fatal(err)
	}
	assets := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<h1>MDFS</h1>")},
		"app.js":     &fstest.MapFile{Data: []byte("console.log('mdfs')")},
	}
	cfg := config.Config{
		Server:    config.Server{Listen: "127.0.0.1:8080", SessionTimeout: "1h"},
		Anonymous: config.Principal{Permissions: permissions},
		Users:     []config.User{{Name: "admin", Password: "secret", Permissions: "RWD"}},
		WebDAV:    config.WebDAV{Enabled: boolPointer(true), Prefix: "/webdav"},
		UI:        config.UI{Title: "MDFS", DefaultLanguage: "zh-CN"},
		Features:  config.Features{MaxUploadBytes: 1 << 20, MaxSearchResults: 100, ImagePreview: boolPointer(true)},
	}
	handler, err := New(browser, fs.FS(assets), cfg, "test")
	if err != nil {
		t.Fatal(err)
	}
	return handler, root
}

func TestFilesAndContentAPI(t *testing.T) {
	handler, _ := testServer(t, "R")
	request := httptest.NewRequest(http.MethodGet, "/api/v1/files?path=/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "hello.txt") {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/content?path=/hello.txt&disposition=inline", nil)
	request.Header.Set("Range", "bytes=1-3")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusPartialContent || response.Body.String() != "ell" {
		t.Fatalf("range response = %d %q", response.Code, response.Body.String())
	}
}

func TestLoginAndMutations(t *testing.T) {
	handler, root := testServer(t, "R")
	loginBody := strings.NewReader(`{"username":"admin","password":"secret"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/session", loginBody)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || len(response.Result().Cookies()) != 1 {
		t.Fatalf("login = %d %s", response.Code, response.Body.String())
	}
	cookie := response.Result().Cookies()[0]

	request = httptest.NewRequest(http.MethodPost, "/api/v1/directories", strings.NewReader(`{"path":"/docs"}`))
	request.AddCookie(cookie)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create directory = %d %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/text-files", strings.NewReader(`{"path":"/docs/note.txt","content":"note"}`))
	request.AddCookie(cookie)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create text = %d %s", response.Code, response.Body.String())
	}
	data, err := os.ReadFile(filepath.Join(root, "docs", "note.txt"))
	if err != nil || string(data) != "note" {
		t.Fatalf("created text = %q, %v", data, err)
	}

	request = httptest.NewRequest(http.MethodDelete, "/api/v1/files?path=/docs/note.txt", nil)
	request.AddCookie(cookie)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("delete = %d %s", response.Code, response.Body.String())
	}
}

func TestUploadAndArchive(t *testing.T) {
	handler, _ := testServer(t, "RWD")
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "upload.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.WriteString(part, "uploaded")
	_ = writer.WriteField("relativePath", "nested/upload.txt")
	_ = writer.Close()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/uploads?path=/", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("upload = %d %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/archive?path=/nested", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("archive = %d %s", response.Code, response.Body.String())
	}
	archive, err := zip.NewReader(bytes.NewReader(response.Body.Bytes()), int64(response.Body.Len()))
	if err != nil || len(archive.File) != 1 || !strings.HasSuffix(archive.File[0].Name, "upload.txt") {
		t.Fatalf("zip = %#v, %v", archive, err)
	}
}

func TestTraversalAndPermissions(t *testing.T) {
	handler, _ := testServer(t, "R")
	request := httptest.NewRequest(http.MethodGet, "/api/v1/files?path=%2F..%2Fsecret", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}

	request = httptest.NewRequest(http.MethodDelete, "/api/v1/files?path=/hello.txt", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestSearchWebDAVAndSPA(t *testing.T) {
	handler, _ := testServer(t, "RWD")
	request := httptest.NewRequest(http.MethodGet, "/api/v1/search?path=/&query=hello", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	var result struct {
		Entries []json.RawMessage `json:"entries"`
	}
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &result) != nil || len(result.Entries) != 1 {
		t.Fatalf("search = %d %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest("PROPFIND", "/webdav/", nil)
	request.Header.Set("Depth", "1")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusMultiStatus || !strings.Contains(response.Body.String(), "hello.txt") {
		t.Fatalf("webdav = %d %s", response.Code, response.Body.String())
	}

	for _, target := range []string{"/", "/nested/route"} {
		request = httptest.NewRequest(http.MethodGet, target, nil)
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "MDFS") {
			t.Fatalf("%s response = %d %s", target, response.Code, response.Body.String())
		}
	}
}

func TestWebDAVWriteMoveDelete(t *testing.T) {
	handler, root := testServer(t, "RWD")
	request := httptest.NewRequest(http.MethodPut, "/webdav/dav.txt", strings.NewReader("dav"))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated && response.Code != http.StatusNoContent {
		t.Fatalf("PUT = %d %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest("MOVE", "/webdav/dav.txt", nil)
	request.Header.Set("Destination", "http://example.com/webdav/moved.txt")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("MOVE = %d %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodDelete, "/webdav/moved.txt", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("DELETE = %d %s", response.Code, response.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "moved.txt")); !os.IsNotExist(err) {
		t.Fatalf("WebDAV file remains: %v", err)
	}
}

func TestRejectsCrossOriginMutation(t *testing.T) {
	handler, _ := testServer(t, "RWD")
	request := httptest.NewRequest(http.MethodPost, "/api/v1/directories", strings.NewReader(`{"path":"/blocked"}`))
	request.Header.Set("Origin", "https://evil.example")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}
