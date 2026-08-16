// Service is the common thin client that connects everything.
// It does not care if the caller is HTTP, gRPC, CLI, etc.
package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"rcloudstorage/internal/storage"
	"strings"
	"time"
)

// NOTE@mazidrehaan: Service does not have an
// interface because there will always ever
// be one Service. There will not be flavors of
// Service like MemoryService, DiskService, etc.

// Service is a thin layer that recieves request
// to store or get the data. It composes of a
// StorageBackend that handles the storing of data
// and the MetadataStore which handles the storing
// of metadata for the data based on the
// implementation. The caller only needs to provide
// the key and data and not worry about metadata
// store and backend and other lower level implementation.
type Service struct {
	Backend   storage.StorageBackend
	Metadata  storage.MetadataStore
	ChunkSize int64
}

func New(backend storage.StorageBackend, metadata storage.MetadataStore, chunkSize int64) *Service {
	return &Service{
		Backend:   backend,
		Metadata:  metadata,
		ChunkSize: chunkSize,
	}
}

// PutLarge stores data in chunks of s.ChunkSize, writing every chunk
// to the backend before ever writing the manifest. The manifest is
// written last and is what makes the object visible to GetLarge.
func (s *Service) Put(key string, r io.Reader) error {

	var chunks []ChunkRef
	// This allows hashing incremently
	overallHash := sha256.New()
	var totalSize int64
	var contentType string

	// Make a byte slice of size s.ChunkSize
	buf := make([]byte, s.ChunkSize)

	for i := 0; ; i++ {
		n, err := io.ReadFull(r, buf)

		// If more than 0 bytes are read
		if n > 0 {

			if i == 0 {
				contentType = http.DetectContentType(buf[:n])
			}

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
		ObjectKey:   key,
		TotalSize:   totalSize,
		ChunkSize:   s.ChunkSize,
		Chunks:      chunks,
		Checksum:    hex.EncodeToString(overallHash.Sum(nil)),
		ContentType: contentType,
	}

	manifestReader, err := manifestAsReader(manifest)
	if err != nil {
		return fmt.Errorf("encoding manifest: %w", err)
	}

	// Finally store the manifest for the key that has access
	// to the manifest which contains the list of ChunkRef
	// which contains the ordered list of all the Chunks and
	// its key to fetch from the Backend
	if err := s.Backend.Put(manifestKey(key), manifestReader); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	created := time.Now()

	// TODO@mazidrehaan: This assumes all errors are ErrNotFound,
	// but with disk storage it could be anything. Handle that
	// in the future.
	if existing, err := s.Metadata.Get(metadataKey(key)); err == nil {
		created = existing.CreatedAt
	}

	return s.Metadata.Put(metadataKey(key), storage.Metadata{
		CreatedAt:  created,
		ModifiedAt: time.Now(),
	})
}

// GetLarge returns a stream of a chunked object's bytes and its manifest.
func (s *Service) Get(key string) (io.ReadCloser, Manifest, error) {

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

		chunkBytes, err := io.ReadAll(record)
		record.Close()
		if err != nil {
			return nil, Manifest{}, fmt.Errorf("reading chunk %d: %w", i, err)
		}

		if err := s.verifyChunk(chunkBytes, ref.Checksum); err != nil {
			return nil, Manifest{}, err
		}

		chunkReader := io.NopCloser(bytes.NewReader(chunkBytes))

		readers[i] = chunkReader
		closers[i] = chunkReader
	}

	// TODO@mazidrehaan: Complete this later an use metadata for
	// other info
	meta, err := s.Metadata.Get(metadataKey(key))
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, Manifest{}, err
	}

	_ = meta

	return &multiReadCloser{
		Reader:  io.MultiReader(readers...),
		closers: closers,
	}, manifest, nil
}

func (s *Service) List(prefix string) ([]string, error) {
	keys, err := s.Backend.List(prefix)
	if err != nil {
		return nil, err
	}

	objects := make([]string, 0, len(keys))
	for _, key := range keys {
		if strings.HasSuffix(key, "/manifest") {
			objects = append(objects, strings.TrimSuffix(key, "/manifest"))
		}
	}

	return objects, nil
}

func (s *Service) Delete(key string) error {
	manifestRecord, err := s.Backend.Get(manifestKey(key))
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return s.Metadata.Delete(metadataKey(key))
		}
		return err
	}
	defer manifestRecord.Close()

	manifestBytes, err := io.ReadAll(manifestRecord)
	if err != nil {
		return fmt.Errorf("reading manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return fmt.Errorf("decoding manifest: %w", err)
	}

	// TODO@mazidrehaan: In future, perhaps we can think
	// of a better way to delete and let a GC in the BG
	// clear up unlinked chunks.
	for _, ref := range manifest.Chunks {
		if err := s.Backend.Delete(ref.Key); err != nil {
			return fmt.Errorf("deleting chunk %q: %w", ref.Key, err)
		}
	}

	if err := s.Backend.Delete(manifestKey(key)); err != nil {
		return fmt.Errorf("deleting manifest: %w", err)
	}

	if err := s.Metadata.Delete(metadataKey(key)); err != nil {
		return fmt.Errorf("deleting metadata: %w", err)
	}

	return nil
}
