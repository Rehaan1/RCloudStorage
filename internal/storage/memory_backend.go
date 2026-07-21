package storage

import (
	"bytes"
	"io"
	"sort"
	"strings"
	"sync"
)

// MemoryBackend is an in-memory implementation of
// Storage Backend.
type MemoryBackend struct {
	mu   sync.RWMutex
	data map[string][]byte
}

// NewMemoryBackend create a new MemoryBackend with an
// empty data map. Returns a pointer to MemoryBackend.
func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{data: make(map[string][]byte)}
}

func (m *MemoryBackend) Put(key string, data io.Reader) error {
	// NOTE@mazidrehaan: We do not take lock here as
	// reading data (aka upload) might take time and we
	// dont want to block other readers meanwhile.
	buf, err := io.ReadAll(data)

	if err != nil {
		return err
	}

	// Take write lock
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = buf

	return nil
}

func (m *MemoryBackend) Get(key string) (io.ReadCloser, error) {

	// Take read lock
	m.mu.RLock()
	defer m.mu.RUnlock()

	buf, ok := m.data[key]

	if !ok {
		return nil, ErrNotFound
	}

	return io.NopCloser(bytes.NewReader(buf)), nil
}

func (m *MemoryBackend) Delete(key string) error {

	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data, key)

	return nil
}

func (m *MemoryBackend) List(prefix string) ([]string, error) {

	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]string, 0)

	for k := range m.data {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}

	sort.Strings(keys)

	return keys, nil
}
