package filebrowser

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestListAndOpen(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "z.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("world"), 0o644); err != nil {
		t.Fatal(err)
	}

	browser, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	listing, err := browser.List("/")
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Entries) != 3 || listing.Entries[0].Name != "docs" || listing.Entries[1].Name != "a.txt" {
		t.Fatalf("unexpected listing: %#v", listing)
	}

	file, info, err := browser.Open("/z.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if info.Size() != 5 {
		t.Fatalf("file size = %d, want 5", info.Size())
	}
}

func TestRejectsTraversalAndEscapingSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "outside")); err != nil {
		t.Fatal(err)
	}
	browser, err := New(root)
	if err != nil {
		t.Fatal(err)
	}

	for _, candidate := range []string{"/../secret.txt", "../../secret.txt", "/outside/secret.txt", "C:\\secret.txt"} {
		if _, _, err := browser.resolve(candidate); !errors.Is(err, ErrInvalidPath) {
			t.Errorf("resolve(%q) error = %v, want ErrInvalidPath", candidate, err)
		}
	}
}
