package storage_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"rcloudstorage/internal/storage"
	"slices"
	"sync"
	"testing"
)

func TestMemoryBackend_PutGet(t *testing.T) {

	tests := []struct {
		name string
		key  string
		data []byte
	}{
		{"simple", "hello.txt", []byte("hello world")},
		{"empty value", "empty.txt", []byte{}},
		{"binary data", "bin.dat", []byte{0x00, 0xFF, 0x01}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := storage.NewMemoryBackend()
			if err := b.Put(tt.key, bytes.NewReader(tt.data)); err != nil {
				t.Fatalf("Put failed: %v", err)
			}

			r, err := b.Get(tt.key)
			if err != nil {
				t.Fatalf("Get failed: %v", err)
			}

			defer r.Close()
			got, _ := io.ReadAll(r)

			if !bytes.Equal(got, tt.data) {
				t.Errorf("got %v, want %v", got, tt.data)
			}
		})
	}
}

func TestMemoryBackend_Get_MissingKey(t *testing.T) {

	b := storage.NewMemoryBackend()

	_, err := b.Get("does-not-exist")

	if !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestMemoryBackend_Delete(t *testing.T) {

	tests := []struct {
		name      string
		deleteKey string
		wantData  []byte // nil means: expect Get("bob") to fail with ErrNotFound
	}{
		{"deletes existing key", "bob", nil},
		{"idempotent on non-existent key, leaves bob untouched", "abc", []byte("Hello World")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := storage.NewMemoryBackend()

			if err := b.Put("bob", bytes.NewReader([]byte("Hello World"))); err != nil {
				t.Fatalf("Put failed: %v", err)
			}

			if err := b.Delete(tt.deleteKey); err != nil {
				t.Fatalf("Delete failed: %v", err)
			}

			r, err := b.Get("bob")

			if tt.wantData == nil {
				if !errors.Is(err, storage.ErrNotFound) {
					t.Errorf("got err %v, want ErrNotFound", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Get failed: %v", err)
			}
			defer r.Close()

			got, _ := io.ReadAll(r)
			if !bytes.Equal(got, tt.wantData) {
				t.Errorf("got %v, want %v", got, tt.wantData)
			}
		})
	}
}

func TestMemoryBackend_List(t *testing.T) {

	tests := []struct {
		name   string
		seed   []string // keys to Put before listing
		prefix string
		want   []string // expected result, already in sorted order
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
			name:   "results are sorted lexicographically",
			seed:   []string{"z.txt", "m.txt", "a.txt"},
			prefix: "",
			want:   []string{"a.txt", "m.txt", "z.txt"},
		},
		{
			name:   "no matches returns empty, not an error",
			seed:   []string{"foo.txt", "bar.txt"},
			prefix: "nomatch/",
			want:   []string{},
		},
		{
			name:   "empty backend",
			seed:   []string{},
			prefix: "",
			want:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := storage.NewMemoryBackend()

			for _, key := range tt.seed {
				if err := b.Put(key, bytes.NewReader([]byte("data"))); err != nil {
					t.Fatalf("Put(%q) failed: %v", key, err)
				}
			}

			got, err := b.List(tt.prefix)
			if err != nil {
				t.Fatalf("List failed: %v", err)
			}

			if !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemoryBackend_ConcurrentAccess(t *testing.T) {
	b := storage.NewMemoryBackend()

	const numPutters = 50
	const numGetters = 50

	var wg sync.WaitGroup

	// Putters: each writes to its own distinct key.
	for i := 0; i < numPutters; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", i)
			data := fmt.Appendf(nil, "value-%d", i)
			if err := b.Put(key, bytes.NewReader(data)); err != nil {
				t.Errorf("Put(%q) failed: %v", key, err)
			}
		}(i)
	}

	// Getters: read keys concurrently with the putters above.
	// A miss (ErrNotFound) is expected and fine — we're not
	// asserting ordering, just that nothing races or panics.
	for i := 0; i < numGetters; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", i)
			r, err := b.Get(key)
			if err != nil {
				if !errors.Is(err, storage.ErrNotFound) {
					t.Errorf("Get(%q) unexpected error: %v", key, err)
				}
				return
			}
			defer r.Close()
			if _, err := io.ReadAll(r); err != nil {
				t.Errorf("Get(%q) read failed: %v", key, err)
			}
		}(i)
	}

	wg.Wait()
}
