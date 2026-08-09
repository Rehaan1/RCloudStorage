package service

import (
	"bytes" // >>> ADDED: needed for bytes.NewReader / bytes.Repeat, was missing
	"errors"
	"io"
	"testing" // >>> ADDED: needed for *testing.T, was missing

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

type corruptingBackend struct {
	storage.StorageBackend
	corruptKey string
}

func (c *corruptingBackend) Get(key string) (io.ReadCloser, error) {
	rc, err := c.StorageBackend.Get(key)
	if err != nil {
		return nil, err
	}

	if key != c.corruptKey {
		return rc, nil
	}

	data, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		return nil, err
	}

	if len(data) > 0 {
		data[0] ^= 0xFF
	}

	return io.NopCloser(bytes.NewReader(data)), nil
}

func TestService_PutGet_ChunkedRoundTrip(t *testing.T) {
	backend := storage.NewMemoryBackend()
	metaStore := storage.NewMemoryMetadataStore()
	svc := New(backend, metaStore, 4) // tiny chunk size to force many chunks

	data := []byte("this input is well over three chunks of test data")

	if err := svc.Put("bigfile", bytes.NewReader(data)); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}

	rc, manifest, err := svc.Get("bigfile")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
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

func TestService_Put_FailurePartway_NoManifest(t *testing.T) {
	backend := &failingBackend{
		StorageBackend: storage.NewMemoryBackend(),
		failOnCall:     3, // fail writing the 3rd chunk
	}
	metaStore := storage.NewMemoryMetadataStore()
	svc := New(backend, metaStore, 4)

	data := bytes.Repeat([]byte("x"), 20) // 5 chunks at chunk size 4

	if err := svc.Put("bigfile", bytes.NewReader(data)); err == nil {
		t.Fatal("expected Put to return an error, got nil")
	}

	if _, _, err := svc.Get("bigfile"); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("got error %v, want ErrNotFound", err)
	}
}

func TestService_Get_DetectsCorruptedChunk(t *testing.T) {
	wrapped := &corruptingBackend{
		StorageBackend: storage.NewMemoryBackend(),
		corruptKey:     "bigfile/chunks/0",
	}
	metaStore := storage.NewMemoryMetadataStore()
	svc := New(wrapped, metaStore, 4)

	data := []byte("this input is well over three chunks of test data")
	if err := svc.Put("bigfile", bytes.NewReader(data)); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}

	if _, _, err := svc.Get("bigfile"); err == nil {
		t.Fatal("expected Get to fail when a chunk is corrupted")
	}
}

