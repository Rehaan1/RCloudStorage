package storage

import (
	"fmt"
	"path/filepath"
	"strings"
)

// safePath maps a relative key to a path under root.
// Keys must be relative. ".." and absolute keys are rejected so a
// client cannot write outside the data directory.
func safePath(root, key string) (string, error) {
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

	full := filepath.Join(root, cleaned)

	rel, err := filepath.Rel(root, full)
	if err != nil {
		return "", err
	}

	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("storage: key escapes root: %q", key)
	}

	// TODO@mazidrehaan: SymLink protection pending to be added.
	return full, nil
}
