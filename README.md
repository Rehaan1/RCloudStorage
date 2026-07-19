# Personal Cloud Storage

A personal cloud storage system, built from scratch as a self-directed learning project — the goal isn't just to have working storage, but to understand every layer of it along the way.

## What this is

A single-user, web-accessible storage service: upload files from anywhere, download them from anywhere, and share individual files via public links with people who don't have an account. No multi-device sync, no multi-user access — kept deliberately simple in scope so the focus stays on understanding the system rather than building every feature a commercial product would have.

## Why

Rather than reaching for an existing solution (Nextcloud, Syncthing, S3, etc.), this project builds the pieces deliberately, one phase at a time, to actually learn the concepts behind cloud storage systems — storage abstraction, durability, availability trade-offs, and eventually security — rather than just consuming them as a black box.

## Approach

The project is structured in phases, moving from a fully mocked backend to real self-hosted hardware:

- **Phase 1 — V1 (current):** Core functionality (upload, download, list, delete, share links) built against a mocked/in-memory storage backend, sitting behind a storage abstraction interface so the backend can be swapped later without touching the rest of the system.
- **Phase 2 — Durability & Replication:** Swap the in-memory backend for a real disk-backed one with atomic (temp-file + rename) writes, then run multiple independent node processes coordinated with Dynamo-style quorum replication (N/W/R writes and reads), heartbeat-based failure detection, and anti-entropy recovery — so the system survives a node dying without losing data or needing manual intervention.
- **Phase 3:** Move off the laptop onto real hardware — a Raspberry Pi with attached external hard disks — replacing the dev-time disk backend behind the same interface.
- **Phase 4:** Encryption at rest, once the system is expected to hold sensitive files.

## Stack

Written in Go.

## Design principles

- **Write-once, read-many.** Files aren't expected to be edited after upload — this simplifies versioning and conflict handling.
- **Durability first.** Data loss is treated as unacceptable, so the storage abstraction and checksum-based integrity checks are in place from V1, ahead of real redundancy in Phase 3.
- **High availability, not a strict SLA.** Built for a personal project running on modest hardware — brief outages are an accepted trade-off rather than something engineered away.
- **Throughput over latency.** Optimized for transferring whole files (photos, phone-recorded video) rather than serving many small, latency-sensitive requests.
