package httpserver

import (
	"context"
	"errors"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"golang.org/x/net/webdav"

	"mdfs/internal/filebrowser"
)

type davFS struct{ browser *filebrowser.Browser }

func (f davFS) Mkdir(_ context.Context, name string, perm os.FileMode) error {
	if err := f.browser.CreateDir(name); err != nil {
		return davError(err)
	}
	physical, _, err := f.browser.Resolve(name)
	if err == nil {
		_ = os.Chmod(physical, perm)
	}
	return nil
}

func (f davFS) OpenFile(_ context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	if f.browser.IsVirtualRoot(name) {
		if flag&(os.O_WRONLY|os.O_RDWR|os.O_CREATE|os.O_TRUNC) != 0 {
			return nil, os.ErrPermission
		}
		return newVirtualRoot(f.browser.Shares()), nil
	}
	physical, _, err := f.browser.Resolve(name)
	if err != nil && errors.Is(err, os.ErrNotExist) && flag&os.O_CREATE != 0 {
		physical, _, err = f.browser.ResolveTarget(name)
	}
	if err != nil {
		return nil, davError(err)
	}
	file, err := os.OpenFile(physical, flag, perm)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (f davFS) RemoveAll(_ context.Context, name string) error {
	return davError(f.browser.Delete(name))
}

func (f davFS) Rename(_ context.Context, oldName, newName string) error {
	return davError(f.browser.Move(oldName, newName))
}

func (f davFS) Stat(_ context.Context, name string) (os.FileInfo, error) {
	if f.browser.IsVirtualRoot(name) {
		return virtualInfo{name: "/", modified: time.Unix(0, 0)}, nil
	}
	physical, _, err := f.browser.Resolve(name)
	if err != nil {
		return nil, davError(err)
	}
	return os.Stat(physical)
}

func davError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, filebrowser.ErrConflict):
		return os.ErrExist
	case errors.Is(err, filebrowser.ErrInvalidPath), errors.Is(err, filebrowser.ErrVirtualRoot):
		return os.ErrPermission
	default:
		return err
	}
}

type virtualRootFile struct {
	shares []filebrowser.Share
	offset int
}

func newVirtualRoot(shares []filebrowser.Share) *virtualRootFile {
	return &virtualRootFile{shares: shares}
}

func (f *virtualRootFile) Close() error              { return nil }
func (f *virtualRootFile) Read([]byte) (int, error)  { return 0, io.EOF }
func (f *virtualRootFile) Write([]byte) (int, error) { return 0, os.ErrPermission }
func (f *virtualRootFile) Stat() (os.FileInfo, error) {
	return virtualInfo{name: "/", modified: time.Unix(0, 0)}, nil
}
func (f *virtualRootFile) Seek(int64, int) (int64, error) { return 0, nil }
func (f *virtualRootFile) Readdir(count int) ([]os.FileInfo, error) {
	if f.offset >= len(f.shares) {
		return nil, io.EOF
	}
	end := len(f.shares)
	if count > 0 && f.offset+count < end {
		end = f.offset + count
	}
	result := make([]os.FileInfo, 0, end-f.offset)
	for _, share := range f.shares[f.offset:end] {
		info, err := os.Stat(share.Path)
		if err != nil {
			return nil, err
		}
		result = append(result, virtualInfo{name: share.Name, size: info.Size(), modified: info.ModTime()})
	}
	f.offset = end
	return result, nil
}

type virtualInfo struct {
	name     string
	size     int64
	modified time.Time
}

func (i virtualInfo) Name() string {
	trimmed := strings.TrimSuffix(i.name, "/")
	if trimmed == "" {
		return "/"
	}
	return path.Base(trimmed)
}
func (i virtualInfo) Size() int64        { return i.size }
func (i virtualInfo) Mode() os.FileMode  { return os.ModeDir | 0o555 }
func (i virtualInfo) ModTime() time.Time { return i.modified }
func (i virtualInfo) IsDir() bool        { return true }
func (i virtualInfo) Sys() any           { return nil }
