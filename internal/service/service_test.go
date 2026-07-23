package service

import (
	"testing"

	"rcloudstorage/internal/storage"
)

func TestService_PutGet_RoundTrip(t *testing.T) {
	backend := storage.NewMemoryBackend()
	metaStore := storage.NewMemoryMetadataStore()
	svc := New(backend, metaStore, 4*1024*1024)

	data := []byte("hello, world")
	if err := svc.Put("greeting", data); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}

	gotData, gotMeta, err := svc.Get("greeting")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}

	if string(gotData) != string(data) {
		t.Errorf("got data %q, want %q", gotData, data)
	}
	if gotMeta.Key != "greeting" {
		t.Errorf("got Key %q, want %q", gotMeta.Key, "greeting")
	}
	if gotMeta.Size != int64(len(data)) {
		t.Errorf("got Size %d, want %d", gotMeta.Size, len(data))
	}
	if gotMeta.ContentType == "" {
		t.Errorf("got empty ContentType, want a sniffed value")
	}
	if gotMeta.CreatedAt.IsZero() {
		t.Errorf("got zero CreatedAt")
	}
	if gotMeta.ModifiedAt.IsZero() {
		t.Errorf("got zero ModifiedAt")
	}
}
