package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"rcloudstorage/internal/service"
	"rcloudstorage/internal/storage"
)

func newTestService() *service.Service {
	backend := storage.NewMemoryBackend()
	metaStore := storage.NewMemoryMetadataStore()
	return service.New(backend, metaStore, 4*1024*1024)
}

func TestHandlePut(t *testing.T) {
	svc := newTestService()

	req := httptest.NewRequest("PUT", "/objects/test.txt", strings.NewReader("hello world"))
	req.SetPathValue("key", "test.txt")
	w := httptest.NewRecorder()

	handlePut(svc)(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("got status %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestHandleGet(t *testing.T) {
	svc := newTestService()

	// Seed data through the service directly — this test is
	// about handleGet, not handlePut, so we don't want a Put
	// bug (if one existed) to make this test fail for the
	// wrong reason.
	if err := svc.Put("test.txt", strings.NewReader("hello world")); err != nil {
		t.Fatalf("seed Put failed: %v", err)
	}

	req := httptest.NewRequest("GET", "/objects/test.txt", nil)
	req.SetPathValue("key", "test.txt")
	w := httptest.NewRecorder()

	handleGet(svc)(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", w.Code, http.StatusOK)
	}
	if w.Body.String() != "hello world" {
		t.Errorf("got body %q, want %q", w.Body.String(), "hello world")
	}
}

func TestHandleGet_MissingKey(t *testing.T) {
	svc := newTestService()

	req := httptest.NewRequest("GET", "/objects/nope.txt", nil)
	req.SetPathValue("key", "nope.txt")
	w := httptest.NewRecorder()

	handleGet(svc)(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("got status %d, want %d", w.Code, http.StatusNotFound)
	}
}
