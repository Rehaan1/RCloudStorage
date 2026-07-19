# Requirements — Personal Cloud Storage

## Users & Access

- Single user, or multiple (e.g. family)?
  - Single for V1

- Multi-device sync needed, or just upload-from-anywhere / download-from-anywhere?
  - V1 does not need multi-device sync

- Do you need to share a file/link with someone who doesn't have an account?
  - Yes

## Data Shape

- Largest single file you realistically need to store? (Photos? A VM image? A movie?)
  - Video Not too large, recorded on Phone
- Typical file count you expect to accumulate in year one?
  - 500
- Mostly write-once-read-many, or files that get overwritten/updated often?
  - Write-once read-many

## Durability & Availability

- What's the actual cost of losing a file? ("annoying" vs "irreplaceable")
  - irreplaceable
- Is "my laptop and the server are both on fire" a scenario you're designing for,
  or is "the server's disk dies" the realistic worst case?
  - Imagine the worst case where everything can go down
- Does it need to be reachable 24/7, or is "usually up" fine for a personal project?
  - 24/7 stated as the target. Treat as "high availability, personal-project tolerance for brief outages"

## Performance

- Rough upload/download speed you'd be unhappy going below?
  - Not a hard target for V1. Realistic ceiling on a Pi is roughly
    what its network interface and USB-attached disks support
- Is latency (time to first byte) or throughput (large file transfer speed) more important to you?
  - Throughput. Focused on transferring whole files

## Security

- Is this ever going to hold anything sensitive (financial docs, ID scans)?
  If yes — encryption at rest becomes a Phase 4 priority, not optional.
  - Yes
- Just you accessing it, or will it ever be exposed to anyone else's access?
  - Might change in future only if granted access

## Constraints

- Budget for real hardware/hosting once you get to Phase 3?
  - Not yet decided
- What hardware do you actually already own that could serve as a node?
  - Planning to use: Raspberry Pi + external hard disk(s) (not yet purchased)
