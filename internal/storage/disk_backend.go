package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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