package api

import (
	"errors"
	"net/http"
	"rcloudstorage/internal/service"
	"rcloudstorage/internal/storage"
	"strconv"
)

// handleGet is the api call for smaller files < ChunkSize
// using the metadataStore compared to the manifest store
// for larger files.
func handleGet(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")

		data, meta, err := svc.Get(key)

		switch {
		case errors.Is(err, storage.ErrNotFound):
			http.Error(w, "not found", http.StatusNotFound)
			return

		case err != nil:
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", meta.ContentType)
		w.Header().Set("Content-Length", strconv.FormatInt(meta.Size, 10))
		w.Write(data)
	}
}
