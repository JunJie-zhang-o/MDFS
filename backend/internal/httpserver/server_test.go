package httpserver

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"mdfs/internal/filebrowser"
)

func testServer(t *testing.T) http.Handler {
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
	return New(browser, fs.FS(assets))
}

func TestFilesAPI(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/files?path=/", nil)
	response := httptest.NewRecorder()
	testServer(t).ServeHTTP(response, request)

	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "hello.txt") {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestDownloadAPI(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/download?path=/hello.txt", nil)
	response := httptest.NewRecorder()
	testServer(t).ServeHTTP(response, request)

	if response.Code != http.StatusOK || response.Body.String() != "hello" {
		t.Fatalf("response = %d %q", response.Code, response.Body.String())
	}
}

func TestTraversalIsRejected(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/files?path=%2F..%2Fsecret", nil)
	response := httptest.NewRecorder()
	testServer(t).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestSPAHandler(t *testing.T) {
	for _, target := range []string{"/", "/nested/route"} {
		request := httptest.NewRequest(http.MethodGet, target, nil)
		response := httptest.NewRecorder()
		testServer(t).ServeHTTP(response, request)
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "MDFS") {
			t.Fatalf("%s response = %d %s", target, response.Code, response.Body.String())
		}
	}
}
