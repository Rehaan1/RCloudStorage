package storage

import (
	"bytes"
	"errors"
	"io"
	"os"
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