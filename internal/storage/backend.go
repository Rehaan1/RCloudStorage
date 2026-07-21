package storage

import (
	"errors"
	"io"
)

// ErrorNotFound is returned when the key to perform the
// action on is not found
var ErrNotFound = errors.New("storage: key not found")

// StorageBackend is the contract every storage implementation
// satisfies.
//
// Design decisions:
//   - Put overwrites existing keys, making it idempotent.
//   - Delete is idempotent — deleting a missing key returns nil.
//   - List("") matches all keys.
type StorageBackend interface {

	// Put stores data under the key, reading until EOF.
	// It fully consumes or fully rejects - never leaves partial
	// object visible. Put on an existing key overwrites the
	// data.
	Put(key string, data io.Reader) error

	// Get returns a reader for the object at key. Caller is
	// responsible for closing the reader. Returns ErrNotFound
	// if key does not exist
	Get(key string) (io.ReadCloser, error)

	// Delete removes the object at key. It is idempotent: deleting
	// a key that does not exist is not an error and returns nil.
	Delete(key string) error

	// List returns all keys with the given prefix, in lexicographical
	// order. An empty prefix matches all keys. A no match is returned
	// as empty slice.
	List(prefix string) ([]string, error)
}
