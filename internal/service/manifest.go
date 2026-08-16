package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
)

// ChunkRef points at one chunk of a chunked object's data,
// stored under its own key in the backend
type ChunkRef struct {
	Index    int
	Key      string
	Size     int64
	Checksum string
}

// Manifest is the single record that makes a chunked object visible.
// Get checks for a manifest first with no manifest meaning no object.
type Manifest struct {
	ObjectKey   string
	TotalSize   int64
	ChunkSize   int64
	Chunks      []ChunkRef
	Checksum    string
	ContentType string
}

func manifestKey(key string) string { return key + "/manifest" }

func metadataKey(key string) string { return key + "/metadata" }

func chunkKey(objectKey string, index int) string {
	return fmt.Sprintf("%s/chunks/%d", objectKey, index)
}

func manifestAsReader(m Manifest) (io.Reader, error) {

	// Convert the manifest into json bytes to store
	// in the backend
	data, err := json.Marshal(m)

	if err != nil {
		return nil, err
	}

	return bytes.NewReader(data), nil
}

func (s *Service) verifyChunk(data []byte, want string) error {
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if got != want {
		return fmt.Errorf("chunk integrity check faled: got %s want %s", got, want)
	}
	return nil
}

// multiReadCloser adapts io.MultiReader's plain io.Reader into an
// io.ReadCloser, closing every underlying chunk reader on Close.
type multiReadCloser struct {
	io.Reader
	closers []io.Closer
}

// Close creates a custom Close function for multiReadCloser
func (m *multiReadCloser) Close() error {
	var firstErr error
	for _, c := range m.closers {
		if err := c.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
