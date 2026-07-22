package service

import (
	"bytes"
	"io"
	"net/http"
	"rcloudstorage/internal/storage"
	"time"
)

// NOTE@mazidrehaan: Service does not have an
// interface because there will always ever
// be one Service. There will not be flavors of
// Service like MemoryService, DiskService, etc.

// Service composes of a StorageBackend that handles
// the storing of data based on the implementation
// (memory, disk, etc) and the MetadataStore which
// handles the storing of metadata for the data
// based on the implementation.
type Service struct {
	Backend  storage.StorageBackend
	Metadata storage.MetadataStore
}

func New(backend storage.StorageBackend, metadata storage.MetadataStore) *Service {
	return &Service{
		Backend:  backend,
		Metadata: metadata,
	}
}

func (s *Service) Put(key string, data []byte) error {

	now := time.Now()

	sniffLen := 512
	if len(data) < sniffLen {
		sniffLen = len(data)
	}

	contentType := http.DetectContentType(data[:sniffLen])

	meta := storage.Metadata{
		Key:         key,
		Size:        int64(len(data)),
		ContentType: contentType,
		CreatedAt:   now,
		ModifiedAt:  now,
	}

	// If data already exists, preserve original CreatedAt.
	if existing, err := s.Metadata.Get(key); err == nil {
		meta.CreatedAt = existing.CreatedAt
	}

	if err := s.Backend.Put(key, bytes.NewReader(data)); err != nil {
		return err
	}

	return s.Metadata.Put(key, meta)
}

func (s *Service) Get(key string) ([]byte, storage.Metadata, error) {

	meta, err := s.Metadata.Get(key)

	if err != nil {
		return nil, storage.Metadata{}, err
	}

	rc, err := s.Backend.Get(key)

	if err != nil {
		return nil, storage.Metadata{}, err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, storage.Metadata{}, err
	}
	return data, meta, nil
}
