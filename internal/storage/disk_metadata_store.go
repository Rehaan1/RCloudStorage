package storage

import (
	"encoding/json"
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

func (s *DiskMetadataStore) Put(key string, m Metadata) error {
	finalPath, err := s.pathFor(key)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(finalPath), ".tmp-*")
	if err != nil {
		return err
	}

	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	data, err := json.Marshal(m)
	if err != nil {
		tmp.Close()
		return err
	}

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}

	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpName, finalPath); err == nil {
		return nil
	}

	if rmErr := os.Remove(finalPath); rmErr != nil && !os.IsNotExist(rmErr) {
		return err
	}
	return os.Rename(tmpName, finalPath)
}

func (s *DiskMetadataStore) Get(key string) (Metadata, error) {
	var m Metadata

	path, err := s.pathFor(key)
	if err != nil {
		return Metadata{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Metadata{}, ErrNotFound
		}
		return Metadata{}, err
	}

	if err := json.Unmarshal(data, &m); err != nil {
		return Metadata{}, err
	}

	return m, nil
}

func (s *DiskMetadataStore) Delete(key string) error {
	path, err := s.pathFor(key)
	if err != nil {
		return err
	}

	err = os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
