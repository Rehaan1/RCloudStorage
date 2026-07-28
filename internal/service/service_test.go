package service

import (
	"bytes"
	"io"
	"testing"

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
