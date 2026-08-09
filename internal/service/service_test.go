package service

import (
	"bytes"
	"io"
	"testing"
	"time"

	"rcloudstorage/internal/storage"
)

func TestService_PutGet_RoundTrip(t *testing.T) {
	backend := storage.NewMemoryBackend()
	metaStore := storage.NewMemoryMetadataStore()
	svc := New(backend, metaStore, 4*1024*1024)

	data := []byte("hello, world")

	if err := svc.Put("greeting", bytes.NewReader(data)); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}

	rc, manifest, err := svc.Get("greeting")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	defer rc.Close() // >>> NEW: must close the reader now

	gotData, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("reading data: %v", err)
	}

	if string(gotData) != string(data) {
		t.Errorf("got data %q, want %q", gotData, data)
	}

	if manifest.ObjectKey != "greeting" {
		t.Errorf("got ObjectKey %q, want %q", manifest.ObjectKey, "greeting")
	}

	if manifest.TotalSize != int64(len(data)) {
		t.Errorf("got TotalSize %d, want %d", manifest.TotalSize, len(data))
	}

	if manifest.ContentType == "" {
		t.Errorf("got empty ContentType, want a sniffed value")
	}
}

func TestService_Get_SucceedsWhenMetadataIsMissing(t *testing.T) {
	backend := storage.NewMemoryBackend()
	metaStore := storage.NewMemoryMetadataStore()
	svc := New(backend, metaStore, 4*1024*1024)

	data := []byte("hello, world")
	if err := svc.Put("greeting", bytes.NewReader(data)); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}

	if err := metaStore.Delete("greeting"); err != nil {
		t.Fatalf("Delete metadata returned error: %v", err)
	}

	rc, manifest, err := svc.Get("greeting")
	if err != nil {
		t.Fatalf("Get returned error after metadata deletion: %v", err)
	}
	defer rc.Close()

	gotData, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("reading data: %v", err)
	}

	if string(gotData) != string(data) {
		t.Errorf("got data %q, want %q", gotData, data)
	}

	if manifest.TotalSize != int64(len(data)) {
		t.Errorf("got TotalSize %d, want %d", manifest.TotalSize, len(data))
	}
}

func TestService_Put_PreservesCreatedAtOnOverwrite(t *testing.T) {
	backend := storage.NewMemoryBackend()
	metaStore := storage.NewMemoryMetadataStore()
	svc := New(backend, metaStore, 4*1024*1024)

	if err := svc.Put("greeting", bytes.NewReader([]byte("first"))); err != nil {
		t.Fatalf("first Put returned error: %v", err)
	}

	firstMeta, err := metaStore.Get("greeting")
	if err != nil {
		t.Fatalf("metadata Get returned error: %v", err)
	}

	time.Sleep(1 * time.Millisecond)

	if err := svc.Put("greeting", bytes.NewReader([]byte("second"))); err != nil {
		t.Fatalf("second Put returned error: %v", err)
	}

	secondMeta, err := metaStore.Get("greeting")
	if err != nil {
		t.Fatalf("metadata Get returned error after overwrite: %v", err)
	}

	if !firstMeta.CreatedAt.Equal(secondMeta.CreatedAt) {
		t.Errorf("CreatedAt changed after overwrite: got %v, want %v", secondMeta.CreatedAt, firstMeta.CreatedAt)
	}

	if !secondMeta.ModifiedAt.After(firstMeta.ModifiedAt) {
		t.Errorf("ModifiedAt was not updated on overwrite: first %v, second %v", firstMeta.ModifiedAt, secondMeta.ModifiedAt)
	}
}
