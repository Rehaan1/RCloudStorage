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
