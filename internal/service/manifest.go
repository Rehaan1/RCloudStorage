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
	ObjectKey string
	TotalSize int64
	ChunkSize int64
	Chunks    []ChunkRef
	Checksum  string
}

func manifestKey(key string) string { return key + "/manifest" }

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

// NOTE@mazidrehaan: We are keeping PutLarge and GetLarge in manifest.go
// instead of service.go as this uses manifest approach and its related
// to the Manifest and ChunkRef both of which exists in this file.
// In Go we should keep what references and is used together.

// PutLarge stores data in chunks of s.ChunkSize, writing every chunk
// to the backend before ever writing the manifest. The manifest is
// written last and is what makes the object visible to GetLarge.
func (s *Service) PutLarge(key string, r io.Reader) error {

	var chunks []ChunkRef
	// This allows hashing incremently
	overallHash := sha256.New()

	var totalSize int64

	// Make a byte slice of size s.ChunkSize
	buf := make([]byte, s.ChunkSize)

	for i := 0; ; i++ {
		n, err := io.ReadFull(r, buf)

		// If more than 0 bytes are read
		if n > 0 {

			chunkCheckSum := sha256.Sum256(buf[:n])

			// Chunk i of Key key
			ck := chunkKey(key, i)

			// Put the data associated with the chunkKey in the backend
			if putErr := s.Backend.Put(ck, bytes.NewReader(buf[:n])); putErr != nil {
				return fmt.Errorf("writing chunk %d: %w", i, putErr)
			}

			// Update the chunk key in the chunks and the reference
			chunks = append(chunks, ChunkRef{
				Index:    i,
				Key:      ck,
				Size:     int64(n),
				Checksum: hex.EncodeToString(chunkCheckSum[:]),
			})

			overallHash.Write(buf[:n])
			totalSize += int64(n)
		}

		// ErrUnexepectedEOF is used as if last chunk is not exact
		// size of ChunkSize it would return this while reading
		// the bytes into n.
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}

		if err != nil {
			return fmt.Errorf("reading input at chunk %d: %w", i, err)
		}
	}

	manifest := Manifest{
		ObjectKey: key,
		TotalSize: totalSize,
		ChunkSize: s.ChunkSize,
		Chunks:    chunks,
		Checksum:  hex.EncodeToString(overallHash.Sum(nil)),
	}

	manifestReader, err := manifestAsReader(manifest)
	if err != nil {
		return fmt.Errorf("encoding manifest: %w", err)
	}

	// Finally store the manifest for the key that has access
	// to the manifest which contains the list of ChunkRef
	// which contains the ordered list of all the Chunks and
	// its key to fetch from the Backend
	return s.Backend.Put(manifestKey(key), manifestReader)
}

// GetLarge returns a stream of a chunked object's bytes and its manifest.
func (s *Service) GetLarge(key string) (io.ReadCloser, Manifest, error) {

	// Get the manifest
	manifestKey := manifestKey(key)

	manifestRecord, err := s.Backend.Get(manifestKey)
	if err != nil {
		return nil, Manifest{}, err
	}
	defer manifestRecord.Close()

	manifestBytes, err := io.ReadAll(manifestRecord)
	if err != nil {
		return nil, Manifest{}, fmt.Errorf("reading manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, Manifest{}, fmt.Errorf("decoding manifest: %w", err)
	}

	readers := make([]io.Reader, len(manifest.Chunks))
	closers := make([]io.Closer, len(manifest.Chunks))

	// Get Readers for chunks in the manifest
	for i, ref := range manifest.Chunks {

		record, err := s.Backend.Get(ref.Key)
		if err != nil {
			return nil, Manifest{}, fmt.Errorf("fetching chunk %d: %w", i, err)
		}
		readers[i] = record
		closers[i] = record
	}

	return &multiReadCloser{
		Reader:  io.MultiReader(readers...),
		closers: closers,
	}, manifest, nil
}
