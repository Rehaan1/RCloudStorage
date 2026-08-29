package api

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"rcloudstorage/internal/storage"
	"strings"
	"testing"
)

func TestInternalRouterPutAndGet(t *testing.T) {
	backend := storage.NewMemoryBackend()
	router := NewInternalRouter(backend)

	key := "photos/2026/sunset.jpg/chunks/0"
	want := []byte("a raw replicated chunk")

	putRequest := httptest.NewRequest(
		http.MethodPut,
		"/internal/objects/"+key,
		bytes.NewReader(want),
	)
	putResponse := httptest.NewRecorder()

	router.ServeHTTP(putResponse, putRequest)

	if putResponse.Code != http.StatusCreated {
		t.Fatalf("PUT status = %d; want %d", putResponse.Code, http.StatusCreated)
	}

	getRequest := httptest.NewRequest(
		http.MethodGet,
		"/internal/objects/"+key,
		nil,
	)
	getResponse := httptest.NewRecorder()

	router.ServeHTTP(getResponse, getRequest)

	if getResponse.Code != http.StatusOK {
		t.Fatalf("GET status = %d; want %d", getResponse.Code, http.StatusOK)
	}

	got, err := io.ReadAll(getResponse.Body)
	if err != nil {
		t.Fatalf("reading GET body: %v", err)
	}

	if !bytes.Equal(got, want) {
		t.Errorf("GET body = %q; want %q", got, want)
	}
}

func TestInternalRouterGetMissingObject(t *testing.T) {
	backend := storage.NewMemoryBackend()
	router := NewInternalRouter(backend)

	request := httptest.NewRequest(
		http.MethodGet,
		"/internal/objects/missing",
		nil,
	)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Errorf("GET status = %d; want %d", response.Code, http.StatusNotFound)
	}
}

func TestInternalRouterDelete(t *testing.T) {
	backend := storage.NewMemoryBackend()

	if err := backend.Put("remove-me", strings.NewReader("data")); err != nil {
		t.Fatalf("seeding backend: %v", err)
	}

	router := NewInternalRouter(backend)

	request := httptest.NewRequest(
		http.MethodDelete,
		"/internal/objects/remove-me",
		nil,
	)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("DELETE status = %d; want %d", response.Code, http.StatusNoContent)
	}

	_, err := backend.Get("remove-me")
	if err != storage.ErrNotFound {
		t.Errorf("object remains after delete; Get() error = %v", err)
	}
}

func TestInternalRouterList(t *testing.T) {
	backend := storage.NewMemoryBackend()

	for _, key := range []string{
		"photos/a.jpg",
		"photos/b.jpg",
		"documents/notes.txt",
	} {
		if err := backend.Put(key, strings.NewReader("data")); err != nil {
			t.Fatalf("seeding %q: %v", key, err)
		}
	}

	router := NewInternalRouter(backend)

	request := httptest.NewRequest(
		http.MethodGet,
		"/internal/objects?prefix=photos/",
		nil,
	)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("LIST status = %d; want %d", response.Code, http.StatusOK)
	}

	got := response.Body.String()
	want := "photos/a.jpg\nphotos/b.jpg\n"

	if got != want {
		t.Errorf("LIST body = %q; want %q", got, want)
	}
}
