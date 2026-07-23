# Decisions Log

Every decision here changed the shape of the system in some way — not a changelog of what was built, but a record of what was chosen, what else was on the table, and why. New entries go at the top.

Template for new entries:

```
## <Short Title>
**Date:**
**Decision:**
**Why:**
**Alternatives considered:**
**Revisit if:**
```

---

## Chunk Size: 4MB

**Date:** 2026-07-23
**Decision:** Use a 4MB chunk size for Service.ChunkSize.
**Why:** Per REQUIREMENTS.md, the system is expected to store mostly documents and photos, with video being rare. Photos and documents are typically well under a few MB each, so a 4MB chunk means most objects fit in a single chunk (no chunking overhead at all) or split into just a small handful, keeping manifests short and read/write amplification low. This follows the module's guidance: smaller chunks suit smaller, more numerous files; larger chunks (16–64MB) suit large, sequential files like video, which isn't the common case here.
**Alternatives considered:** 16–64MB chunks are better suited to video-heavy workloads, but would mean most photos/documents never get split at all, making the chunking/manifest path effectively untested by real usage and pointlessly coarse for the actual data. Sub-1MB chunks would exercise chunking more but adds unnecessary manifest overhead (more ChunkRef entries, more backend keys) for files that don't need it.
**Revisit if:** Usage patterns shift toward significantly larger files (e.g., video becomes a primary use case), or Phase 2 replication overhead per chunk turns out to make 4MB chunks too fine-grained across nodes.

## Language: Go

**Date:** Phase 1
**Decision:** Whole system in Go.
**Why:**

- Node coordination (Phase 2) needs fan-out writes with clean timeouts and no leaked goroutines on a slow/dead node. Goroutines + channels + context handle this natively — not a pattern I'd be bolting on.
- Deploy target is a Raspberry Pi. Go cross-compiles to one static binary (`GOOS=linux GOARCH=arm64 go build`), no runtime to install on the Pi.
- Scale is tiny (500 files/yr, one user), so GC overhead vs Rust/C doesn't matter. What matters is not shipping a race condition into the disk/replication path — `-race` and no manual memory management get me there faster.
- Stdlib (net/http, os/io, encoding/json) covers V1/V2, no dependency tree to manage.
  **Alternatives considered:**
- Rust — safer concurrency guarantees, but I'd spend more time fighting the borrow checker than learning the actual distributed-systems concepts this project is for.
- Python/Node — event-loop concurrency isn't real parallelism, and deployment means shipping a runtime instead of one binary.
  **Revisit if:** performance becomes an actual bottleneck, or this stops being solo work.
