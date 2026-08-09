package main

import (
	"log"
	"net/http"
	"rcloudstorage/internal/api"
	"rcloudstorage/internal/service"
	"rcloudstorage/internal/storage"
)

func main() {
	backend := storage.NewMemoryBackend()
	metaStore := storage.NewMemoryMetadataStore()

	// ChunkSize is 0 as it is unused until we add
	// support for streaming of large files
	svc := service.New(backend, metaStore, 4*1024*1024)

	router := api.NewRouter(svc)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
