# RCloudStorage Architecture

## Current system

```mermaid
flowchart TB
    Client["Client<br/>(curl, web UI later)"]

    Client -->|"Public API<br/>PUT /objects/{key}<br/>GET /objects/{key}<br/>DELETE /objects/{key}"| CoordinatorAPI

    subgraph CoordinatorProcess["Coordinator process — :9000"]
        CoordinatorAPI["Public API router"]

        Service["Service<br/>• splits uploads into 4 MB chunks<br/>• creates manifest<br/>• verifies SHA-256 checksums on read"]

        Replication["Coordinator<br/>implements StorageBackend<br/>N=3, W=2, R=2"]

        CoordinatorMetadata["DiskMetadataStore<br/>CreatedAt / ModifiedAt<br/>local only"]

        CoordinatorAPI --> Service
        Service -->|"chunk + manifest operations"| Replication
        Service -->|"metadata operations"| CoordinatorMetadata
    end

    Replication -->|"Concurrent raw HTTP PUT/GET/DELETE<br/>/internal/objects/{key...}"| Node1
    Replication -->|"Concurrent raw HTTP PUT/GET/DELETE<br/>/internal/objects/{key...}"| Node2
    Replication -->|"Concurrent raw HTTP PUT/GET/DELETE<br/>/internal/objects/{key...}"| Node3

    subgraph Node1["Node 1 process — :9001"]
        Node1API["Internal raw-storage API"]
        Node1Disk["DiskBackend<br/>node1/<key>/chunks/0<br/>node1/<key>/manifest"]
        Node1API --> Node1Disk
    end

    subgraph Node2["Node 2 process — :9002"]
        Node2API["Internal raw-storage API"]
        Node2Disk["DiskBackend<br/>node2/<key>/chunks/0<br/>node2/<key>/manifest"]
        Node2API --> Node2Disk
    end

    subgraph Node3["Node 3 process — :9003"]
        Node3API["Internal raw-storage API"]
        Node3Disk["DiskBackend<br/>node3/<key>/chunks/0<br/>node3/<key>/manifest"]
        Node3API --> Node3Disk
    end

    Node1 -.->|"Public /objects API exists<br/>development/debug only;<br/>close off in deployment"| Node1Public["Do not use directly"]
    Node2 -.->|"Public /objects API exists<br/>development/debug only;<br/>close off in deployment"| Node2Public["Do not use directly"]
    Node3 -.->|"Public /objects API exists<br/>development/debug only;<br/>close off in deployment"| Node3Public["Do not use directly"]
```

## Upload path

```text
Client
  → Coordinator public API
  → Service splits the file into 4 MB chunks
  → Coordinator replicates every chunk to all three nodes
  → upload succeeds after W=2 successful node acknowledgements
  → Service writes the manifest last
  → coordinator stores local CreatedAt / ModifiedAt metadata
```

The manifest makes the complete file visible only after its chunks have been
written.

## Download path

```text
Client
  → Coordinator public API
  → Service asks Coordinator for manifest/chunks
  → Coordinator queries replicas concurrently
  → returns data only when R=2 nodes return identical bytes
  → Service verifies each chunk’s SHA-256 checksum
  → reconstructed file streams back to the client
```

## Current durability boundary

```text
Replicated:
  chunks + manifest → node 1, node 2, node 3

Not yet replicated:
  CreatedAt + ModifiedAt metadata → coordinator disk only
```

A node can fail without losing a successfully quorum-written file. Coordinator
metadata replication remains future work.
