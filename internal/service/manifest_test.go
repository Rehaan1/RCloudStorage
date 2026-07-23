package service

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"rcloudstorage/internal/storage"
)

// failingBackend wraps a real StorageBackend and fails the Nth call to
// Put, to simulate a mid-write crash without needing real process death.
type failingBackend struct {
	// NOTE@mazidrehaan: This embeds StorageBackend. When struct embeds a type,
	// Go automatically promoes the embedded value's method onto
	// the outer struct. It gets Get, Delete, List and only Put
	// is different cause we defined our own Put below.
	// Go only promotes unnamed fields.
	storage.StorageBackend
	failOnCall int
	calls      int
}

func (f *failingBackend) Put(key string, data io.Reader) error {
	f.calls++
	if f.calls == f.failOnCall {
		return errors.New("simulated write failure")
	}
	return f.StorageBackend.Put(key, data)
}

func TestService_PutLarge_GetLarge_RoundTrip(t *testing.T) {
	backend := storage.NewMemoryBackend()
	metaStore := storage.NewMemoryMetadataStore()
	svc := New(backend, metaStore, 4) // tiny chunk size to force many chunks

	data := []byte("this input is well over three chunks of test data")
	if err := svc.PutLarge("bigfile", bytes.NewReader(data)); err != nil {
		t.Fatalf("PutLarge returned error: %v", err)
	}

	rc, manifest, err := svc.GetLarge("bigfile")
	if err != nil {
		t.Fatalf("GetLarge returned error: %v", err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("reading object: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("got %q, want %q", got, data)
	}
	if len(manifest.Chunks) < 3 {
		t.Errorf("expected at least 3 chunks, got %d", len(manifest.Chunks))
	}
	if manifest.TotalSize != int64(len(data)) {
		t.Errorf("got TotalSize %d, want %d", manifest.TotalSize, len(data))
	}
}

func TestService_PutLarge_FailurePartway_NoManifest(t *testing.T) {
	backend := &failingBackend{
		StorageBackend: storage.NewMemoryBackend(),
		failOnCall:     3, // fail writing the 3rd chunk
	}
	metaStore := storage.NewMemoryMetadataStore()
	svc := New(backend, metaStore, 4)

	data := bytes.Repeat([]byte("x"), 20) // 5 chunks at chunk size 4
	if err := svc.PutLarge("bigfile", bytes.NewReader(data)); err == nil {
		t.Fatal("expected PutLarge to return an error, got nil")
	}

	if _, _, err := svc.GetLarge("bigfile"); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("got error %v, want ErrNotFound", err)
	}
}
