package api

import (
	"net/http"
	"rcloudstorage/internal/service"
	"rcloudstorage/internal/storage"
)

// NewRouter creates and returns an HTTP handler (ServeMux) configured with
// routes for object storage operations. It wires the provided `*service.Service`
// handlers to endpoint patterns:
//   - PUT /objects/{key}      -> handlePut
//   - GET /objects/{key}      -> handleGet
//   - DELETE /objects/{key}   -> handleDelete
//   - GET /objects            -> handleList
//
// The returned handler can be used as the root HTTP handler for the API server.
func NewRouter(svc *service.Service) http.Handler {
	mux := http.NewServeMux()
	registerPublicRoutes(mux, svc)

	return mux
}

// NewNodeRouter creates the API run by every storage node.
//
// It exposes both:
//
//   - /objects/...          public Phase 1 API, handled by Service
//   - /internal/objects/... raw replication API, handled by StorageBackend
//
// The coordinator uses only the internal API.
func NewNodeRouter(
	svc *service.Service,
	backend storage.StorageBackend,
) http.Handler {
	mux := http.NewServeMux()

	registerPublicRoutes(mux, svc)
	registerInternalRoutes(mux, backend)

	return mux
}

func registerPublicRoutes(mux *http.ServeMux, svc *service.Service) {
	mux.Handle("GET /objects/{key}", handleGet(svc))
	mux.HandleFunc("PUT /objects/{key}", handlePut(svc))
	mux.HandleFunc("DELETE /objects/{key}", handleDelete(svc))
	mux.HandleFunc("GET /objects", handleList(svc))
}
