package blob

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type FilesystemStore struct {
	root string
}

func NewFilesystemStore(root string) (*FilesystemStore, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		root = "./data/blobs"
	}
	cleanRoot := filepath.Clean(root)
	if err := os.MkdirAll(cleanRoot, 0o750); err != nil {
		return nil, err
	}
	return &FilesystemStore{root: cleanRoot}, nil
}

func (s *FilesystemStore) Put(_ context.Context, key, _ string, body []byte) error {
	path, err := s.resolvePath(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, body, 0o640); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func (s *FilesystemStore) Get(_ context.Context, key string) ([]byte, error) {
	path, err := s.resolvePath(key)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrObjectNotFound
		}
		return nil, err
	}
	return data, nil
}

func (s *FilesystemStore) Delete(_ context.Context, key string) error {
	path, err := s.resolvePath(key)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s *FilesystemStore) resolvePath(key string) (string, error) {
	key = strings.TrimSpace(key)
	key = strings.TrimPrefix(filepath.Clean("/"+key), "/")
	if key == "" || key == "." {
		return "", errors.New("invalid blob key")
	}
	path := filepath.Join(s.root, key)
	rel, err := filepath.Rel(s.root, path)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(rel, "..") {
		return "", errors.New("invalid blob key path")
	}
	return path, nil
}
