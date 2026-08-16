package storage

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestPathFor_StaysUnderRoot(t *testing.T) {
	dir := t.TempDir()
	d, err := NewDiskBackend(dir)
	if err != nil {
		t.Fatal(err)
	}

	got, err := d.pathFor("photos/cat.jpg/chunks/0")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, d.root) {
		t.Fatalf("path %q is not under root %q", got, d.root)
	}
	wantSuffix := filepath.Join("photos", "cat.jpg", "chunks", "0")
	if !strings.HasSuffix(got, wantSuffix) {
		t.Fatalf("got %q, want suffix %q", got, wantSuffix)
	}
}

func TestPathFor_RejectsEscape(t *testing.T) {
	dir := t.TempDir()
	d, err := NewDiskBackend(dir)
	if err != nil {
		t.Fatal(err)
	}

	_, err = d.pathFor("../outside")
	if err == nil {
		t.Fatal("expected error for .. key")
	}
}