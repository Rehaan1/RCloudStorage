package api

import (
	"errors"
	"io"
	"net/http"
	"rcloudstorage/internal/service"
	"rcloudstorage/internal/storage"
	"strconv"
)

// handleGet is the api call for getting smaller files < ChunkSize
// using the metadataStore compared to the manifest store
// for larger files.
func handleGet(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")

		data, manifest, err := svc.Get(key)

		switch {
		case errors.Is(err, storage.ErrNotFound):
			http.Error(w, "not found", http.StatusNotFound)
			return

		case err != nil:
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", manifest.ContentType)
		w.Header().Set("Content-Length", strconv.FormatInt(manifest.TotalSize, 10))
		if _, err := io.Copy(w, data); err != nil {
			// headers are already sent at this point, so we can't call
			// http.Error here — just log it server-side
			// (add a logger later; for now this is a known gap)
			return
		}
	}
}

// handlePut is the api call for adding smaller files < ChunkSize
// using the metadataStore compared to the manifest store
// for larger files.
func handlePut(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")

		if err := svc.Put(key, r.Body); err != nil {
			http.Error(w, "write failed", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}
