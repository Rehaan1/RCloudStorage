package replication

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"rcloudstorage/internal/storage"
	"strings"
	"testing"
)

type testNode struct {
	backend *storage.MemoryBackend
	server  *httptest.Server
}

// newTestNode starts a real HTTP test server. It behaves like a storage node
// but keeps its data in MemoryBackend, so each test stays fast and isolated.
func newTestNode(t *testing.T) *testNode {
	t.Helper()

	backend := storage.NewMemoryBackend()
	mux := http.NewServeMux()

	mux.HandleFunc("PUT /internal/objects/{key}", func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")

		if err := backend.Put(key, r.Body); err != nil {
			http.Error(w, "write failed", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	})

	mux.HandleFunc("GET /internal/objects/{key}", func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")

		reader, err := backend.Get(key)
		if err == storage.ErrNotFound {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		if err != nil {
			http.Error(w, "read failed", http.StatusInternalServerError)
			return
		}
		defer reader.Close()

		if _, err := io.Copy(w, reader); err != nil {
			t.Errorf("copying test-node response: %v", err)
		}
	})

	mux.HandleFunc("DELETE /internal/objects/{key}", func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")

		if err := backend.Delete(key); err != nil {
			http.Error(w, "delete failed", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("GET /internal/objects", func(w http.ResponseWriter, r *http.Request) {
		prefix := r.URL.Query().Get("prefix")

		keys, err := backend.List(prefix)
		if err != nil {
			http.Error(w, "list failed", http.StatusInternalServerError)
			return
		}

		for _, key := range keys {
			_, _ = w.Write([]byte(key + "\n"))
		}
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return &testNode{
		backend: backend,
		server:  server,
	}
}

func (n *testNode) client() *NodeClient {
	return &NodeClient{
		Addr: n.server.URL,
		HTTP: n.server.Client(),
	}
}

func TestCoordinatorPutSucceedsWithWriteQuorum(t *testing.T) {
	node1 := newTestNode(t)
	node2 := newTestNode(t)
	node3 := newTestNode(t)

	coordinator, err := NewCoordinator(
		[]*NodeClient{
			node1.client(),
			node2.client(),
			node3.client(),
		},
		2, // W
		2, // R
	)
	if err != nil {
		t.Fatalf("NewCoordinator() error = %v", err)
	}

	data := []byte("this should reach every healthy replica")

	err = coordinator.Put("hello", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	for i, node := range []*testNode{node1, node2, node3} {
		reader, err := node.backend.Get("hello")
		if err != nil {
			t.Fatalf("node %d does not contain object: %v", i+1, err)
		}

		got, err := io.ReadAll(reader)
		reader.Close()

		if err != nil {
			t.Fatalf("reading node %d object: %v", i+1, err)
		}

		if !bytes.Equal(got, data) {
			t.Errorf("node %d stored %q; want %q", i+1, got, data)
		}
	}
}

func TestCoordinatorPutSucceedsWhenOneNodeFails(t *testing.T) {
	node1 := newTestNode(t)
	node2 := newTestNode(t)

	failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "node failed", http.StatusInternalServerError)
	}))
	t.Cleanup(failingServer.Close)

	coordinator, err := NewCoordinator(
		[]*NodeClient{
			node1.client(),
			node2.client(),
			{
				Addr: failingServer.URL,
				HTTP: failingServer.Client(),
			},
		},
		2, // W: node 1 and node 2 are sufficient
		2,
	)
	if err != nil {
		t.Fatalf("NewCoordinator() error = %v", err)
	}

	data := []byte("two good replicas are enough")

	err = coordinator.Put("quorum-write", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}
}

func TestCoordinatorPutFailsWithoutWriteQuorum(t *testing.T) {
	node := newTestNode(t)

	failingHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "node failed", http.StatusInternalServerError)
	})

	failingServer1 := httptest.NewServer(failingHandler)
	t.Cleanup(failingServer1.Close)

	failingServer2 := httptest.NewServer(failingHandler)
	t.Cleanup(failingServer2.Close)

	coordinator, err := NewCoordinator(
		[]*NodeClient{
			node.client(),
			{
				Addr: failingServer1.URL,
				HTTP: failingServer1.Client(),
			},
			{
				Addr: failingServer2.URL,
				HTTP: failingServer2.Client(),
			},
		},
		2, // W=2 but only one node can succeed
		2,
	)
	if err != nil {
		t.Fatalf("NewCoordinator() error = %v", err)
	}

	err = coordinator.Put("not-enough-nodes", strings.NewReader("data"))

	if err == nil {
		t.Fatal("Put() error = nil; want write quorum failure")
	}

	if !strings.Contains(err.Error(), "write quorum failed") {
		t.Errorf("Put() error = %q; want write quorum failure", err)
	}
}

func TestCoordinatorGetSucceedsWithReadQuorum(t *testing.T) {
	node1 := newTestNode(t)
	node2 := newTestNode(t)
	node3 := newTestNode(t)

	want := []byte("replicated chunk data")

	for i, node := range []*testNode{node1, node2, node3} {
		if err := node.backend.Put("chunk-0", bytes.NewReader(want)); err != nil {
			t.Fatalf("seeding node %d: %v", i+1, err)
		}
	}

	coordinator, err := NewCoordinator(
		[]*NodeClient{
			node1.client(),
			node2.client(),
			node3.client(),
		},
		2,
		2, // R
	)
	if err != nil {
		t.Fatalf("NewCoordinator() error = %v", err)
	}

	reader, err := coordinator.Get("chunk-0")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer reader.Close()

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading coordinator result: %v", err)
	}

	if !bytes.Equal(got, want) {
		t.Errorf("Get() = %q; want %q", got, want)
	}
}

func TestCoordinatorDeleteSucceedsWithWriteQuorum(t *testing.T) {
	node1 := newTestNode(t)
	node2 := newTestNode(t)
	node3 := newTestNode(t)

	for i, node := range []*testNode{node1, node2, node3} {
		if err := node.backend.Put("remove-me", strings.NewReader("data")); err != nil {
			t.Fatalf("seeding node %d: %v", i+1, err)
		}
	}

	coordinator, err := NewCoordinator(
		[]*NodeClient{
			node1.client(),
			node2.client(),
			node3.client(),
		},
		2,
		2,
	)
	if err != nil {
		t.Fatalf("NewCoordinator() error = %v", err)
	}

	if err := coordinator.Delete("remove-me"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	for i, node := range []*testNode{node1, node2, node3} {
		_, err := node.backend.Get("remove-me")

		if err != storage.ErrNotFound {
			t.Errorf("node %d still has object; Get() error = %v", i+1, err)
		}
	}
}

func TestCoordinatorGetReturnsDataSupportedByReadQuorum(t *testing.T) {
	node1 := newTestNode(t)
	node2 := newTestNode(t)
	node3 := newTestNode(t)

	current := []byte("current version")
	stale := []byte("old version")

	// This represents a successful W=2 write: nodes 1 and 2 have the
	// newest data, while node 3 missed it and still has old data.
	for i, node := range []*testNode{node1, node2} {
		if err := node.backend.Put("manifest", bytes.NewReader(current)); err != nil {
			t.Fatalf("seeding current data on node %d: %v", i+1, err)
		}
	}

	if err := node3.backend.Put("manifest", bytes.NewReader(stale)); err != nil {
		t.Fatalf("seeding stale data: %v", err)
	}

	coordinator, err := NewCoordinator(
		[]*NodeClient{
			node1.client(),
			node2.client(),
			node3.client(),
		},
		2,
		2,
	)
	if err != nil {
		t.Fatalf("NewCoordinator() error = %v", err)
	}

	reader, err := coordinator.Get("manifest")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer reader.Close()

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading Get() result: %v", err)
	}

	if !bytes.Equal(got, current) {
		t.Errorf("Get() = %q; want current quorum value %q", got, current)
	}
}

func TestCoordinatorGetFailsWhenNoValueHasReadQuorum(t *testing.T) {
	node1 := newTestNode(t)
	node2 := newTestNode(t)
	node3 := newTestNode(t)

	for i, item := range []struct {
		node *testNode
		data string
	}{
		{node1, "version one"},
		{node2, "version two"},
		{node3, "version three"},
	} {
		if err := item.node.backend.Put("manifest", strings.NewReader(item.data)); err != nil {
			t.Fatalf("seeding node %d: %v", i+1, err)
		}
	}

	coordinator, err := NewCoordinator(
		[]*NodeClient{
			node1.client(),
			node2.client(),
			node3.client(),
		},
		2,
		2,
	)
	if err != nil {
		t.Fatalf("NewCoordinator() error = %v", err)
	}

	_, err = coordinator.Get("manifest")

	if err == nil {
		t.Fatal("Get() error = nil; want read quorum failure")
	}

	if !strings.Contains(err.Error(), "read quorum failed") {
		t.Errorf("Get() error = %q; want read quorum failure", err)
	}
}
