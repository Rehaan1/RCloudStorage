package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"rcloudstorage/internal/storage"
)

// NewInternalRouter provides the coordinator-facing node API.
//
// It works directly with StorageBackend rather than service.Service because
// the coordinator replicates already-created chunks and manifests. Passing
// those records through Service again would chunk them a second time.
func NewInternalRouter(backend storage.StorageBackend) http.Handler {
	mux := http.NewServeMux()

	// {key...} is a catch-all wildcard. It is essential because raw backend
	// keys include slashes, for example: photo.jpg/chunks/0.
	mux.HandleFunc("PUT /internal/objects/{key...}", handleInternalPut(backend))
	mux.HandleFunc("GET /internal/objects/{key...}", handleInternalGet(backend))
	mux.HandleFunc("DELETE /internal/objects/{key...}", handleInternalDelete(backend))
	mux.HandleFunc("GET /internal/objects", handleInternalList(backend))

	return mux
}

func handleInternalPut(backend storage.StorageBackend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")

		if err := backend.Put(key, r.Body); err != nil {
			http.Error(w, "write failed", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

func handleInternalGet(backend storage.StorageBackend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")

		reader, err := backend.Get(key)

		if errors.Is(err, storage.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		if err != nil {
			http.Error(w, "read failed", http.StatusInternalServerError)
			return
		}
		defer reader.Close()

		if _, err := io.Copy(w, reader); err != nil {
			// The response could already be partially written, so there is no
			// safe HTTP error response left to send.
			return
		}
	}
}

func handleInternalDelete(backend storage.StorageBackend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")

		if err := backend.Delete(key); err != nil {
			http.Error(w, "delete failed", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func handleInternalList(backend storage.StorageBackend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		prefix := r.URL.Query().Get("prefix")

		keys, err := backend.List(prefix)
		if err != nil {
			http.Error(w, "list failed", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")

		for _, key := range keys {
			if _, err := fmt.Fprintln(w, key); err != nil {
				return
			}
		}
	}
}
