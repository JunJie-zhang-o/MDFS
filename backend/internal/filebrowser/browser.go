package filebrowser

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var ErrInvalidPath = errors.New("invalid path")

type Entry struct {
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	Type       string    `json:"type"`
	Size       int64     `json:"size"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

type Listing struct {
	Path    string  `json:"path"`
	Entries []Entry `json:"entries"`
}

type Browser struct {
	root string
}

func New(root string) (*Browser, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	realRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(realRoot)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", root)
	}
	return &Browser{root: realRoot}, nil
}

func (b *Browser) Root() string {
	return b.root
}

func (b *Browser) List(requestPath string) (Listing, error) {
	resolved, virtual, err := b.resolve(requestPath)
	if err != nil {
		return Listing{}, err
	}
	items, err := os.ReadDir(resolved)
	if err != nil {
		return Listing{}, err
	}

	entries := make([]Entry, 0, len(items))
	for _, item := range items {
		info, err := item.Info()
		if err != nil {
			return Listing{}, err
		}
		kind := "file"
		if item.IsDir() {
			kind = "directory"
		}
		entries = append(entries, Entry{
			Name:       item.Name(),
			Path:       path.Join(virtual, item.Name()),
			Type:       kind,
			Size:       info.Size(),
			ModifiedAt: info.ModTime(),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Type != entries[j].Type {
			return entries[i].Type == "directory"
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	return Listing{Path: virtual, Entries: entries}, nil
}

func (b *Browser) Open(requestPath string) (*os.File, os.FileInfo, error) {
	resolved, _, err := b.resolve(requestPath)
	if err != nil {
		return nil, nil, err
	}
	file, err := os.Open(resolved)
	if err != nil {
		return nil, nil, err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, err
	}
	if !info.Mode().IsRegular() {
		file.Close()
		return nil, nil, ErrInvalidPath
	}
	return file, info, nil
}

func (b *Browser) resolve(requestPath string) (string, string, error) {
	if strings.ContainsRune(requestPath, '\x00') || strings.Contains(requestPath, "\\") {
		return "", "", ErrInvalidPath
	}
	for _, segment := range strings.Split(requestPath, "/") {
		if segment == ".." {
			return "", "", ErrInvalidPath
		}
	}

	virtual := path.Clean("/" + strings.TrimPrefix(requestPath, "/"))
	relative := strings.TrimPrefix(virtual, "/")
	candidate := filepath.Join(b.root, filepath.FromSlash(relative))
	realPath, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", "", err
	}
	rel, err := filepath.Rel(b.root, realPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", ErrInvalidPath
	}
	return realPath, virtual, nil
}
