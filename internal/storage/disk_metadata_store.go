package storage

import (
	"os"
	"path/filepath"
)

type DiskMetadataStore struct {
	root string
}

func NewDiskMetadataStore(root string) (*DiskMetadataStore, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, err
	}

	return &DiskMetadataStore{root: abs}, nil
}

func (s *DiskMetadataStore) pathFor(key string) (string, error) {
	return safePath(s.root, key)
}
