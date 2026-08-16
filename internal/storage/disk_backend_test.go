package storage

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"slices"
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

func TestDiskBackend_PutWritesFinalFile(t *testing.T) {
	d, err := NewDiskBackend(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	key := "photos/cat.jpg/chunks/0"
	want := []byte("hello disk")
	if err := d.Put(key, bytes.NewReader(want)); err != nil {
		t.Fatal(err)
	}

	path, err := d.pathFor(key)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDiskBackend_PutOverwrites(t *testing.T) {
	d, err := NewDiskBackend(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	key := "obj/chunks/0"
	if err := d.Put(key, bytes.NewReader([]byte("v1"))); err != nil {
		t.Fatal(err)
	}
	if err := d.Put(key, bytes.NewReader([]byte("v2"))); err != nil {
		t.Fatal(err)
	}

	path, err := d.pathFor(key)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "v2" {
		t.Fatalf("got %q, want v2", got)
	}
}

func TestDiskBackend_PutGet(t *testing.T) {
	d, err := NewDiskBackend(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	key := "photos/cat.jpg/chunks/0"
	want := []byte("hello disk")
	if err := d.Put(key, bytes.NewReader(want)); err != nil {
		t.Fatal(err)
	}

	r, err := d.Get(key)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDiskBackend_GetMissing(t *testing.T) {
	d, err := NewDiskBackend(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	_, err = d.Get("no/such/key")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestDiskBackend_Delete(t *testing.T) {
	d, err := NewDiskBackend(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if err := d.Put("bob", bytes.NewReader([]byte("Hello World"))); err != nil {
		t.Fatal(err)
	}

	if err := d.Delete("bob"); err != nil {
		t.Fatal(err)
	}
	
	_, err = d.Get("bob")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}

	// idempotent
	if err := d.Delete("bob"); err != nil {
		t.Fatalf("second Delete: %v", err)
	}
}

func TestDiskBackend_List(t *testing.T) {
	tests := []struct {
		name   string
		seed   []string
		prefix string
		want   []string
	}{
		{
			name:   "filters by prefix",
			seed:   []string{"photos/a.jpg", "photos/b.jpg", "docs/readme.txt"},
			prefix: "photos/",
			want:   []string{"photos/a.jpg", "photos/b.jpg"},
		},
		{
			name:   "empty prefix matches everything",
			seed:   []string{"c.txt", "a.txt", "b.txt"},
			prefix: "",
			want:   []string{"a.txt", "b.txt", "c.txt"},
		},
		{
			name:   "no matches returns empty",
			seed:   []string{"foo.txt"},
			prefix: "nomatch/",
			want:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := NewDiskBackend(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}

			for _, key := range tt.seed {
				if err := d.Put(key, bytes.NewReader([]byte("data"))); err != nil {
					t.Fatal(err)
				}
			}

			got, err := d.List(tt.prefix)
			if err != nil {
				t.Fatal(err)
			}

			if !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewDiskBackend_RemovesOrphanTemps(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "obj", "chunks")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	orphan := filepath.Join(nested, ".tmp-orphan")
	if err := os.WriteFile(orphan, []byte("leftover"), 0o644); err != nil {
		t.Fatal(err)
	}
	
	keep := filepath.Join(dir, "keep.txt")
	if err := os.WriteFile(keep, []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := NewDiskBackend(dir); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(orphan); !os.IsNotExist(err) {
		t.Fatalf("orphan temp still exists: %v", err)
	}
	
	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("real file was deleted: %v", err)
	}
}