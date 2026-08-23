package filebrowser

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var (
	ErrInvalidPath = errors.New("invalid path")
	ErrConflict    = errors.New("path already exists")
	ErrVirtualRoot = errors.New("operation is not available on the virtual root")
)

type Share struct {
	Name string
	Path string
}

type Entry struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Kind        string    `json:"kind"`
	Size        int64     `json:"size"`
	ModifiedAt  time.Time `json:"modifiedAt"`
	HasChildren bool      `json:"hasChildren"`
	PreviewKind string    `json:"previewKind"`
}

type Listing struct {
	Path        string  `json:"path"`
	VirtualRoot bool    `json:"virtualRoot"`
	Entries     []Entry `json:"entries"`
}

type Browser struct {
	shares []Share
	root   string
}

func New(root string) (*Browser, error) {
	return NewShares([]Share{{Name: filepath.Base(filepath.Clean(root)), Path: root}})
}

func NewShares(shares []Share) (*Browser, error) {
	if len(shares) == 0 {
		return nil, fmt.Errorf("at least one share is required")
	}
	resolved := make([]Share, 0, len(shares))
	for _, share := range shares {
		absRoot, err := filepath.Abs(share.Path)
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
			return nil, fmt.Errorf("%s is not a directory", share.Path)
		}
		resolved = append(resolved, Share{Name: share.Name, Path: realRoot})
	}
	browser := &Browser{shares: resolved}
	if len(resolved) == 1 {
		browser.root = resolved[0].Path
	}
	return browser, nil
}

func (b *Browser) Root() string {
	if b.root != "" {
		return b.root
	}
	return "virtual root"
}

func (b *Browser) Shares() []Share {
	result := make([]Share, len(b.shares))
	copy(result, b.shares)
	return result
}

func (b *Browser) IsVirtualRoot(requestPath string) bool {
	return len(b.shares) > 1 && cleanVirtual(requestPath) == "/"
}

func (b *Browser) List(requestPath string) (Listing, error) {
	virtual := cleanVirtual(requestPath)
	if err := validatePath(requestPath); err != nil {
		return Listing{}, err
	}
	if b.IsVirtualRoot(virtual) {
		entries := make([]Entry, 0, len(b.shares))
		for _, share := range b.shares {
			info, err := os.Stat(share.Path)
			if err != nil {
				return Listing{}, err
			}
			entries = append(entries, Entry{
				Name: share.Name, Path: "/" + share.Name, Kind: "directory",
				Size: info.Size(), ModifiedAt: info.ModTime(), HasChildren: hasChildren(share.Path),
			})
		}
		sortEntries(entries)
		return Listing{Path: "/", VirtualRoot: true, Entries: entries}, nil
	}

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
		if item.Type()&os.ModeSymlink != 0 {
			continue
		}
		info, err := item.Info()
		if err != nil {
			return Listing{}, err
		}
		kind := "file"
		if item.IsDir() {
			kind = "directory"
		}
		entryPath := path.Join(virtual, item.Name())
		entries = append(entries, Entry{
			Name: item.Name(), Path: entryPath, Kind: kind, Size: info.Size(), ModifiedAt: info.ModTime(),
			HasChildren: item.IsDir() && hasChildren(filepath.Join(resolved, item.Name())),
			PreviewKind: previewKind(item.Name()),
		})
	}
	sortEntries(entries)
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

func (b *Browser) CreateDir(requestPath string) error {
	target, _, err := b.resolveTarget(requestPath)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(target); err == nil {
		return ErrConflict
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Mkdir(target, 0o755)
}

func (b *Browser) WriteFile(requestPath string, reader io.Reader, createOnly bool) error {
	target, _, err := b.resolveTarget(requestPath)
	if err != nil {
		return err
	}
	if createOnly {
		if _, err := os.Lstat(target); err == nil {
			return ErrConflict
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	temp, err := os.CreateTemp(filepath.Dir(target), ".mdfs-write-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if _, err := io.Copy(temp, reader); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Chmod(0o644); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempName, target)
}

func (b *Browser) Rename(requestPath, newName string) (string, error) {
	if !validName(newName) {
		return "", ErrInvalidPath
	}
	resolved, virtual, err := b.resolve(requestPath)
	if err != nil {
		return "", err
	}
	if b.IsShareRoot(virtual) {
		return "", ErrVirtualRoot
	}
	target := filepath.Join(filepath.Dir(resolved), newName)
	if _, err := os.Lstat(target); err == nil {
		return "", ErrConflict
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err := os.Rename(resolved, target); err != nil {
		return "", err
	}
	return path.Join(path.Dir(virtual), newName), nil
}

func (b *Browser) Move(sourcePath, destinationPath string) error {
	source, sourceVirtual, err := b.resolve(sourcePath)
	if err != nil {
		return err
	}
	if b.IsShareRoot(sourceVirtual) {
		return ErrVirtualRoot
	}
	destination, _, err := b.resolveTarget(destinationPath)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(destination); err == nil {
		return ErrConflict
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(source, destination)
}

func (b *Browser) Delete(requestPath string) error {
	resolved, virtual, err := b.resolve(requestPath)
	if err != nil {
		return err
	}
	if virtual == "/" || b.IsShareRoot(virtual) {
		return ErrVirtualRoot
	}
	return os.RemoveAll(resolved)
}

func (b *Browser) Search(requestPath, query string, limit int, allowed func(string) bool) ([]Entry, bool, error) {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return []Entry{}, false, nil
	}
	var roots []struct{ physical, virtual string }
	if b.IsVirtualRoot(requestPath) {
		for _, share := range b.shares {
			roots = append(roots, struct{ physical, virtual string }{share.Path, "/" + share.Name})
		}
	} else {
		physical, virtual, err := b.resolve(requestPath)
		if err != nil {
			return nil, false, err
		}
		roots = append(roots, struct{ physical, virtual string }{physical, virtual})
	}

	results := make([]Entry, 0)
	truncated := false
	for _, root := range roots {
		err := filepath.WalkDir(root.physical, func(current string, item fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if current == root.physical {
				return nil
			}
			if item.Type()&os.ModeSymlink != 0 {
				if item.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			relative, err := filepath.Rel(root.physical, current)
			if err != nil {
				return err
			}
			virtual := path.Join(root.virtual, filepath.ToSlash(relative))
			if !allowed(virtual) {
				if item.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.Contains(strings.ToLower(item.Name()), query) {
				return nil
			}
			if len(results) >= limit {
				truncated = true
				return fs.SkipAll
			}
			info, err := item.Info()
			if err != nil {
				return err
			}
			kind := "file"
			if item.IsDir() {
				kind = "directory"
			}
			results = append(results, Entry{Name: item.Name(), Path: virtual, Kind: kind, Size: info.Size(), ModifiedAt: info.ModTime(), HasChildren: item.IsDir() && hasChildren(current), PreviewKind: previewKind(item.Name())})
			return nil
		})
		if err != nil && !errors.Is(err, fs.SkipAll) {
			return nil, false, err
		}
		if truncated {
			break
		}
	}
	sortEntries(results)
	return results, truncated, nil
}

func (b *Browser) Resolve(requestPath string) (string, string, error) { return b.resolve(requestPath) }
func (b *Browser) ResolveTarget(requestPath string) (string, string, error) {
	return b.resolveTarget(requestPath)
}

func (b *Browser) IsShareRoot(virtual string) bool {
	virtual = cleanVirtual(virtual)
	if len(b.shares) == 1 {
		return virtual == "/"
	}
	for _, share := range b.shares {
		if virtual == "/"+share.Name {
			return true
		}
	}
	return false
}

func (b *Browser) resolve(requestPath string) (string, string, error) {
	if err := validatePath(requestPath); err != nil {
		return "", "", err
	}
	virtual := cleanVirtual(requestPath)
	share, relative, err := b.shareFor(virtual)
	if err != nil {
		return "", "", err
	}
	candidate := filepath.Join(share.Path, filepath.FromSlash(relative))
	realPath, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", "", err
	}
	if !within(share.Path, realPath) {
		return "", "", ErrInvalidPath
	}
	return realPath, virtual, nil
}

func (b *Browser) resolveTarget(requestPath string) (string, string, error) {
	if err := validatePath(requestPath); err != nil {
		return "", "", err
	}
	virtual := cleanVirtual(requestPath)
	if virtual == "/" || b.IsShareRoot(virtual) {
		return "", "", ErrVirtualRoot
	}
	parentVirtual, name := path.Split(virtual)
	if !validName(name) {
		return "", "", ErrInvalidPath
	}
	parent, _, err := b.resolve(parentVirtual)
	if err != nil {
		return "", "", err
	}
	target := filepath.Join(parent, name)
	share, _, err := b.shareFor(virtual)
	if err != nil || !within(share.Path, target) {
		return "", "", ErrInvalidPath
	}
	return target, virtual, nil
}

func (b *Browser) shareFor(virtual string) (Share, string, error) {
	if len(b.shares) == 1 {
		return b.shares[0], strings.TrimPrefix(virtual, "/"), nil
	}
	if virtual == "/" {
		return Share{}, "", ErrVirtualRoot
	}
	parts := strings.Split(strings.TrimPrefix(virtual, "/"), "/")
	for _, share := range b.shares {
		if parts[0] == share.Name {
			return share, strings.Join(parts[1:], "/"), nil
		}
	}
	return Share{}, "", os.ErrNotExist
}

func validatePath(requestPath string) error {
	if strings.ContainsRune(requestPath, '\x00') || strings.Contains(requestPath, "\\") {
		return ErrInvalidPath
	}
	for _, segment := range strings.Split(requestPath, "/") {
		if segment == ".." {
			return ErrInvalidPath
		}
	}
	return nil
}

func validName(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, `/\\`) && !strings.ContainsRune(name, '\x00')
}

func cleanVirtual(value string) string { return path.Clean("/" + strings.TrimPrefix(value, "/")) }

func within(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func hasChildren(directory string) bool {
	items, err := os.ReadDir(directory)
	if err != nil {
		return false
	}
	for _, item := range items {
		if item.Type()&os.ModeSymlink == 0 {
			return true
		}
	}
	return false
}

func sortEntries(entries []Entry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Kind != entries[j].Kind {
			return entries[i].Kind == "directory"
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
}

func previewKind(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".txt", ".md", ".log", ".ini", ".yaml", ".yml", ".json", ".xml", ".toml", ".go", ".js", ".ts", ".vue", ".css", ".html", ".sh":
		return "text"
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".bmp", ".ico":
		return "image"
	case ".mp3", ".wav", ".ogg", ".flac", ".mp4", ".webm", ".mov", ".m4v":
		return "media"
	case ".pdf":
		return "document"
	}
	if mime.TypeByExtension(ext) != "" {
		return "browser"
	}
	return "none"
}
