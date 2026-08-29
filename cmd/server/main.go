package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"rcloudstorage/internal/api"
	"rcloudstorage/internal/replication"
	"rcloudstorage/internal/service"
	"rcloudstorage/internal/storage"
	"strings"
)

func main() {
	role := flag.String("role", "node", "server role: node or coordinator")
	dataDir := flag.String("data-dir", "./data", "directory for object files")
	addr := flag.String("addr", ":8080", "HTTP listen address")
	nodesFlag := flag.String("nodes", "", "comma-separated node URLs; required for coordinator role")
	writeQuorum := flag.Int("w", 2, "required successful node writes")
	readQuorum := flag.Int("r", 2, "required successful node reads")

	flag.Parse()

	switch *role {
	case "node":
		runNode(*addr, *dataDir)

	case "coordinator":
		runCoordinator(
			*addr,
			*dataDir,
			*nodesFlag,
			*writeQuorum,
			*readQuorum,
		)

	default:
		log.Fatalf("unknown role %q: expected node or coordinator", *role)
	}
}

// runNode starts one independent storage node.
//
// Each node owns a separate DiskBackend and DiskMetadataStore, located in its
// own data directory. It exposes both public and coordinator-only HTTP routes.
func runNode(addr, dataDir string) {
	backend, err := storage.NewDiskBackend(dataDir)
	if err != nil {
		log.Fatal(err)
	}

	metadata, err := storage.NewDiskMetadataStore(dataDir)
	if err != nil {
		log.Fatal(err)
	}

	svc := service.New(backend, metadata, 4*1024*1024)
	router := api.NewNodeRouter(svc, backend)

	log.Printf("node listening on %s (data-dir=%s)", addr, dataDir)
	log.Fatal(http.ListenAndServe(addr, router))
}

// runCoordinator starts the public entry point.
//
// It uses Coordinator as Service's StorageBackend. Consequently, when Service
// writes a chunk or manifest, Coordinator replicates that raw record to the
// node processes and waits for W acknowledgements.
//
// Metadata remains local to the coordinator for now. It is not part of the
// replicated data path in this first quorum-replication module.
func runCoordinator(addr, dataDir, nodesFlag string, w, r int) {
	nodes, err := parseNodes(nodesFlag)
	if err != nil {
		log.Fatal(err)
	}

	// NOTE@mazidrehaan: Instead of using the Backend directly
	// we are using the coordinator that coordinates the replication of data to multiple nodes.
	backend, err := replication.NewCoordinator(nodes, w, r)
	if err != nil {
		log.Fatal(err)
	}

	metadata, err := storage.NewDiskMetadataStore(dataDir)
	if err != nil {
		log.Fatal(err)
	}

	svc := service.New(backend, metadata, 4*1024*1024)
	router := api.NewRouter(svc)

	log.Printf(
		"coordinator listening on %s (metadata-dir=%s, nodes=%d, W=%d, R=%d)",
		addr,
		dataDir,
		len(nodes),
		w,
		r,
	)

	log.Fatal(http.ListenAndServe(addr, router))
}

func parseNodes(value string) ([]*replication.NodeClient, error) {
	if strings.TrimSpace(value) == "" {
		return nil, fmt.Errorf(
			"-nodes is required when running with -role=coordinator",
		)
	}

	parts := strings.Split(value, ",")
	nodes := make([]*replication.NodeClient, 0, len(parts))

	for _, part := range parts {
		addr := strings.TrimSpace(part)

		if addr == "" {
			return nil, fmt.Errorf("node URL cannot be empty")
		}

		if !strings.HasPrefix(addr, "http://") &&
			!strings.HasPrefix(addr, "https://") {
			return nil, fmt.Errorf(
				"node URL %q must start with http:// or https://",
				addr,
			)
		}

		nodes = append(nodes, replication.NewNodeClient(addr))
	}

	return nodes, nil
}
