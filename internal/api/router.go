package api

import (
	"net/http"
	"rcloudstorage/internal/service"
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

	mux.Handle("GET /objects/{key}", handleGet(svc))
	mux.HandleFunc("PUT /objects/{key}", handlePut(svc))

	// TODO@mazidrehaan: Complete the DELETE And PUT
	//mux.Handle("DELETE /objects/{key}", handleDelete(svc))
	//mux.HandleFunc("GET /objects", handleList(svc))

	return mux
}
