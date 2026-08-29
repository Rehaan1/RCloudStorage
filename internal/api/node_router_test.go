package api

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"rcloudstorage/internal/service"
	"rcloudstorage/internal/storage"
	"testing"
)

func TestNodeRouterServesPublicAndInternalRoutes(t *testing.T) {
	backend := storage.NewMemoryBackend()
	metadata := storage.NewMemoryMetadataStore()
	svc := service.New(backend, metadata, 4)

	router := NewNodeRouter(svc, backend)

	// The raw replication endpoint writes directly to StorageBackend.
	internalRequest := httptest.NewRequest(
		http.MethodPut,
		"/internal/objects/raw/chunks/0",
		bytes.NewReader([]byte("raw data")),
	)
	internalResponse := httptest.NewRecorder()

	router.ServeHTTP(internalResponse, internalRequest)

	if internalResponse.Code != http.StatusCreated {
		t.Fatalf(
			"internal PUT status = %d; want %d",
			internalResponse.Code,
			http.StatusCreated,
		)
	}

	// The public endpoint still goes through Service, which chunks files
	// and creates a manifest.
	publicRequest := httptest.NewRequest(
		http.MethodPut,
		"/objects/example.txt",
		bytes.NewReader([]byte("hello")),
	)
	publicResponse := httptest.NewRecorder()

	router.ServeHTTP(publicResponse, publicRequest)

	if publicResponse.Code != http.StatusCreated {
		t.Fatalf(
			"public PUT status = %d; want %d",
			publicResponse.Code,
			http.StatusCreated,
		)
	}

	getRequest := httptest.NewRequest(
		http.MethodGet,
		"/objects/example.txt",
		nil,
	)
	getResponse := httptest.NewRecorder()

	router.ServeHTTP(getResponse, getRequest)

	if getResponse.Code != http.StatusOK {
		t.Fatalf(
			"public GET status = %d; want %d",
			getResponse.Code,
			http.StatusOK,
		)
	}

	got, err := io.ReadAll(getResponse.Body)
	if err != nil {
		t.Fatalf("reading public GET body: %v", err)
	}

	if string(got) != "hello" {
		t.Errorf("public GET body = %q; want %q", got, "hello")
	}
}
