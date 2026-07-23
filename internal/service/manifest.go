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

// PutLarge stores data in chunks of s.ChunkSize, writing every chunk
// to the backend before ever writing the manifest. The manifest is
// written last and is what makes the object visible to GetLarge.
func (s *Service) PutLarge(key string, r io.Reader) error {

	var chunks []ChunkRef
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
