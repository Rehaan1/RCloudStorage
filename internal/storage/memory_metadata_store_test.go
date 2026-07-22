package storage_test

import (
	"rcloudstorage/internal/storage"
	"testing"
	"time"
)

func TestMemoryMetadataStore_PutGet(t *testing.T) {
	s := storage.NewMemoryMetadataStore()
	m := storage.Metadata{
		Key:         "foo",
		Size:        42,
		ContentType: "text/plain",
		CreatedAt:   time.Now(),
		ModifiedAt:  time.Now(),
	}

	if err := s.Put("foo", m); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}

	got, err := s.Get("foo")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got != m {
		t.Errorf("got %+v, want %+v", got, m)
	}
}

func TestMemoryMetadataStore_GetMissing(t *testing.T) {
	s := storage.NewMemoryMetadataStore()
	_, err := s.Get("missing")
	if err != storage.ErrNotFound {
		t.Errorf("got error %v, want ErrNotFound", err)
	}
}

func TestMemoryMetadataStore_Delete(t *testing.T) {
	s := storage.NewMemoryMetadataStore()
	_ = s.Put("foo", storage.Metadata{Key: "foo"})
	if err := s.Delete("foo"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if _, err := s.Get("foo"); err != storage.ErrNotFound {
		t.Errorf("got error %v, want ErrNotFound after delete", err)
	}
}
