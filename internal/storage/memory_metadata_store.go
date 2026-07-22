package storage

import "sync"

// MemoryMetadataStore is an in-memory implementation
// of MetadataStore.
type MemoryMetadataStore struct {
	mu    sync.RWMutex
	store map[string]Metadata
}

func NewMemoryMetadataStore() *MemoryMetadataStore {
	return &MemoryMetadataStore{
		store: make(map[string]Metadata),
	}
}

func (s *MemoryMetadataStore) Put(key string, m Metadata) error {

	s.mu.Lock()
	defer s.mu.Unlock()

	s.store[key] = m

	return nil
}

func (s *MemoryMetadataStore) Get(key string) (Metadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, ok := s.store[key]

	if !ok {
		return Metadata{}, ErrNotFound
	}

	return data, nil
}

func (s *MemoryMetadataStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.store, key)

	return nil
}
