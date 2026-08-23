package filebrowser

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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

func TestMultipleSharesAndMutations(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	browser, err := NewShares([]Share{{Name: "one", Path: first}, {Name: "two", Path: second}})
	if err != nil {
		t.Fatal(err)
	}
	listing, err := browser.List("/")
	if err != nil || !listing.VirtualRoot || len(listing.Entries) != 2 {
		t.Fatalf("listing = %#v, %v", listing, err)
	}
	if err := browser.CreateDir("/one/docs"); err != nil {
		t.Fatal(err)
	}
	if err := browser.WriteFile("/one/docs/a.txt", strings.NewReader("hello"), true); err != nil {
		t.Fatal(err)
	}
	if _, err := browser.Rename("/one/docs/a.txt", "b.txt"); err != nil {
		t.Fatal(err)
	}
	if err := browser.Delete("/one/docs/b.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(first, "docs", "b.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("file still exists: %v", err)
	}
}
