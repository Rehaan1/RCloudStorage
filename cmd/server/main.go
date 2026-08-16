package main

import (
	"flag"
	"log"
	"net/http"
	"rcloudstorage/internal/api"
	"rcloudstorage/internal/service"
	"rcloudstorage/internal/storage"
)

func main() {
	dataDir := flag.String("data-dir", "./data", "directory for object files")
	addr := flag.String("addr", ":8080", "HTTP listen address")
	
	flag.Parse()
	
	backend, err := storage.NewDiskBackend(*dataDir)
	if err != nil {
		log.Fatal(err)
	}
	
	metaStore := storage.NewMemoryMetadataStore()
	svc := service.New(backend, metaStore, 4*1024*1024)
	router := api.NewRouter(svc)
	
	log.Printf("listening on %s (data-dir=%s)", *addr, *dataDir)
	log.Fatal(http.ListenAndServe(*addr, router))
}
