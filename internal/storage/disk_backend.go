package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var _ StorageBackend = (*DiskBackend)(nil)

type DiskBackend struct {
	root string
}

func NewDiskBackend(root string) (*DiskBackend, error) {
	abs, err := filepath.Abs(root)

	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, err
	}

	return &DiskBackend{root: abs}, nil
}

// pathFor maps an object key to a path under root.
// Keys must be relative. ".." and absolute keys are rejected so a
// client cannot write outside the data directory.
func (d *DiskBackend) pathFor(key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("storage: empty key")
	}

	if filepath.IsAbs(key) {
		return "", fmt.Errorf("storage: key must be relative: %q", key)
	}

	cleaned := filepath.Clean(key)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("storage: key escapes root: %q", key)
	}

	full := filepath.Join(d.root, cleaned)

	rel, err := filepath.Rel(d.root, full)
	if err != nil {
		return "", err
	}

	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("storage: key escapes root: %q", key)
	}

	// TODO@mazidrehaan: SymLink protection pending to be added.
	return full, nil
}

func (d *DiskBackend) Put(key string, r io.Reader) error {
	finalPath, err := d.pathFor(key)
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

	if _, err := io.Copy(tmp, r); err != nil {
		tmp.Close()
		return err
	}

	// NOTE@mazidrehaan: Sync is used to flush data from in memory 
	// to the disk.
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

	// NOTE@mazidrehaan: In windows, unlike linux, rename cannot replace an existing file.
	// so we have to remove the existing file first.
	if rmErr := os.Remove(finalPath); rmErr != nil && !os.IsNotExist(rmErr) {
		return err
	}
	return os.Rename(tmpName, finalPath)
}

func (d *DiskBackend) Get(key string) (io.ReadCloser, error) {
	path, err := d.pathFor(key)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return f, nil
}

func (d *DiskBackend) Delete(key string) error {
	path, err := d.pathFor(key)
	if err != nil {
		return err
	}

	err = os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (d *DiskBackend) List(prefix string) ([]string, error) {
	keys := make([]string, 0)
	err := filepath.WalkDir(d.root, func(path string, de os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if de.IsDir() {
			return nil
		}

		if strings.HasPrefix(de.Name(), ".tmp-") {
			return nil
		}

		rel, err := filepath.Rel(d.root, path)
		if err != nil {
			return err
		}

		key := filepath.ToSlash(rel)
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	
	sort.Strings(keys)
	return keys, nil
}