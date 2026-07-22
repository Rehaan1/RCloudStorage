package storage

import "time"

// Metadata describes an object stored via StorageBackend.
// It is persisted seperately from the objectsb bytes -
// see MetadataStore.
type Metadata struct {
	Key         string
	Size        int64
	ContentType string
	Checksum    string
	CreatedAt   time.Time
	ModifiedAt  time.Time
}

// MetadataStore persists Metadata, keyed the same way as
// StorageBackend. Metadata and Object store are intentionally
// seperate as both have different access pattern with Metadata
// being small and accessed frequently while object bytes are
// large and read less often.
type MetadataStore interface {

	// Put stores metadata under the key. Put on an existing
	// key overwrites the key.
	Put(key string, m Metadata) error

	// Get gets the metadata associated with the key. If the
	// key is not found it returns ErrNotFound.
	Get(key string) (Metadata, error)

	// Delete deletes the metadata associated with the key.
	// It is idempotent in nature, deleting a key that does
	// not exist does nothing.
	Delete(key string) error
}
