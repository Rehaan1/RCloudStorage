package storage

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDiskMetadataStore_PathFor_Sidecar(t *testing.T) {
	dir := t.TempDir()
	s, err := NewDiskMetadataStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.pathFor("readme.md/metadata")
	if err != nil {
		t.Fatal(err)
	}

	wantSuffix := filepath.Join("readme.md", "metadata")
	if !strings.HasSuffix(got, wantSuffix) {
		t.Fatalf("got %q, want suffix %q", got, wantSuffix)
	}
	if !strings.HasPrefix(got, s.root) {
		t.Fatalf("path %q is not under root %q", got, s.root)
	}
}

func TestDiskMetadataStore_PathFor_RejectsEscape(t *testing.T) {
	s, err := NewDiskMetadataStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.pathFor("../outside"); err == nil {
		t.Fatal("expected error for .. key")
	}
}
