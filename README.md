# Personal Cloud Storage

A personal cloud storage system, built from scratch as a self-directed learning project — the goal isn't just to have working storage, but to understand every layer of it along the way.

## What this is

A single-user, web-accessible storage service: upload files from anywhere, download them from anywhere, and share individual files via public links with people who don't have an account. No multi-device sync, no multi-user access — kept deliberately simple in scope so the focus stays on understanding the system rather than building every feature a commercial product would have.

## Why

Rather than reaching for an existing solution (Nextcloud, Syncthing, S3, etc.), this project builds the pieces deliberately, one phase at a time, to actually learn the concepts behind cloud storage systems — storage abstraction, durability, availability trade-offs, and eventually security — rather than just consuming them as a black box.

## Approach

The project is structured in phases, moving from a fully mocked backend to real self-hosted hardware:

- **Phase 1 — V1:** Core functionality (upload, download, list, delete, share links) built against a mocked/in-memory storage backend, sitting behind a storage abstraction interface so the backend can be swapped later without touching the rest of the system.
- **Phase 2 — Durability & Replication:** Swap the in-memory backend for a real disk-backed one with atomic (temp-file + rename) writes, then run multiple independent node processes coordinated with Dynamo-style quorum replication (N/W/R writes and reads), heartbeat-based failure detection, and anti-entropy recovery — so the system survives a node dying without losing data or needing manual intervention.
- **Phase 3:** Move off the laptop onto real hardware — a Raspberry Pi with attached external hard disks — replacing the dev-time disk backend behind the same interface.
- **Phase 4:** Encryption at rest, once the system is expected to hold sensitive files.

## API Documentation

For detailed API endpoint documentation and testing, see: [RCloudStorage API Docs](https://rcloudstorage.docs.buildwithfern.com/)

## Stack

Written in Go.

## Current implementation

The project has completed the first part of Phase 2: a local, multi-process
quorum-replication prototype.

- Files are split into 4 MB chunks. A manifest records the ordered chunks,
  content type, size, and SHA-256 checksums.
- `DiskBackend` persists objects with temp-file plus rename writes, while
  `DiskMetadataStore` persists creation and modification timestamps.
- Three independent node processes can store the same chunks and manifests in
  separate data directories.
- A coordinator implements `StorageBackend` and sends each chunk to every node
  over HTTP. With `N=3`, `W=2`, and `R=2`, an upload succeeds when any two
  nodes acknowledge it; a read requires two successful node responses.
- Chunk checksums are verified when files are read.

The coordinator is the public entry point. Nodes also currently expose the
public API for local debugging, but clients should not use it: a direct node
write bypasses replication. Node-to-coordinator replication uses the separate
`/internal/objects/...` raw-storage API.

> **Deployment note:** The public `/objects/...` routes on storage nodes exist
> only because each node is currently run as a complete Phase 1 server, which
> is helpful while developing and debugging locally. They must be closed off
> before a real deployment (for example, by exposing only the internal API on
> nodes or by placing nodes on a private network/firewall). Otherwise a client
> could write directly to one node and bypass quorum replication. The
> coordinator should be the only public-facing process.

Metadata currently lives only in the coordinator data directory. The file
chunks and manifest are replicated, but `CreatedAt` and `ModifiedAt` are not
yet replicated; this is known future work.

## Run a three-node cluster locally

Run the commands below from the repository root in four separate PowerShell
terminals. Use the coordinator at port `9000` for normal uploads and downloads.

Start node 1:

```powershell
go run ./cmd/server -role node -addr :9001 -data-dir ./node1
```

Start node 2:

```powershell
go run ./cmd/server -role node -addr :9002 -data-dir ./node2
```

Start node 3:

```powershell
go run ./cmd/server -role node -addr :9003 -data-dir ./node3
```

Start the coordinator:

```powershell
go run ./cmd/server -role coordinator -addr :9000 -data-dir ./coordinator -nodes 'http://localhost:9001,http://localhost:9002,http://localhost:9003' -w 2 -r 2
```

Upload a file through the coordinator:

```powershell
curl.exe -i -X PUT --data-binary "@README.md" http://localhost:9000/objects/readme-copy
```

Download it again:

```powershell
curl.exe -i http://localhost:9000/objects/readme-copy
```

Run the automated checks:

```powershell
go test ./...
go test -race ./...
```

## Design principles

- **Write-once, read-many.** Files aren't expected to be edited after upload — this simplifies versioning and conflict handling.
- **Durability first.** Data loss is treated as unacceptable, so the storage abstraction and checksum-based integrity checks are in place from V1, ahead of real redundancy in Phase 3.
- **High availability, not a strict SLA.** Built for a personal project running on modest hardware — brief outages are an accepted trade-off rather than something engineered away.
- **Throughput over latency.** Optimized for transferring whole files (photos, phone-recorded video) rather than serving many small, latency-sensitive requests.
