package storage

import "sync"

// MemoryBackend is an in-memory implementation of
// Storage Backend.
type MemoryBackend struct {
	my   sync.RWMutex
	data map[string][]byte
}

// NewMemoryBackend create a new MemoryBackend with an
// empty data map. Returns a pointer to MemoryBackend.
func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{data: make(map[string][]byte)}
}

// TODO@mazidrehaan: Implement the StorageBackend Interface
