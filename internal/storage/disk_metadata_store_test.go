package storage

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestDiskMetadataStore_PutGet(t *testing.T) {
	dir := t.TempDir()
	s, err := NewDiskMetadataStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	m := Metadata{CreatedAt: time.Now(), ModifiedAt: time.Now()}

	if err := s.Put("foo", m); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}

	got, err := s.Get("foo")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}

	if !got.CreatedAt.Equal(m.CreatedAt) || !got.ModifiedAt.Equal(m.ModifiedAt) {
		t.Errorf("got %+v, want %+v", got, m)
	}
}

func TestDiskMetadataStore_GetMissing(t *testing.T) {
	s, err := NewDiskMetadataStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.Get("missing")
	if err != ErrNotFound {
		t.Errorf("got error %v, want ErrNotFound", err)
	}
}

func TestDiskMetadataStore_Delete(t *testing.T) {
	s, err := NewDiskMetadataStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	_ = s.Put("foo", Metadata{})
	if err := s.Delete("foo"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	if _, err := s.Get("foo"); err != ErrNotFound {
		t.Errorf("got error %v, want ErrNotFound after delete", err)
	}
}
