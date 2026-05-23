# Performance profiling

The client compiles in a SIGUSR1-triggered profile-capture mechanism.
Use it to produce a comprehensive performance baseline of any single
phase of gameplay — title screen, walking around, region change,
combat — for analysis with `go tool pprof` and `go tool trace`.

## Triggering a capture

On startup, the client logs its PID:

```
profiling: signal listener ready, send SIGUSR1 to pid 12345
```

From another terminal, send SIGUSR1 to capture:

```bash
kill -USR1 12345
```

After ~30 seconds the client writes a session directory under
`./profiles/<UTC-timestamp>/` (relative to the working directory at
process start). The absolute path is logged on each successful
capture, so you can copy it from the terminal output.

A second SIGUSR1 received during an active capture is logged and
ignored, not queued.

## What gets captured

Each session produces six artifacts:

| File | Tool |
|---|---|
| `cpu.prof` | `go tool pprof cpu.prof` |
| `heap.prof` | `go tool pprof heap.prof` |
| `goroutine.prof` | `go tool pprof goroutine.prof` |
| `mutex.prof` | `go tool pprof mutex.prof` |
| `block.prof` | `go tool pprof block.prof` |
| `trace.out` | `go tool trace trace.out` |

The CPU profile, runtime trace, and mutex/block contention sampling
all cover the same 30-second window, so cross-correlating across
artifacts is meaningful.

## Disk usage

Each session is roughly 100 MB on disk, dominated by `trace.out`
(30–80 MB). Clean up `./profiles/` periodically — the client never
removes old sessions.

## Design

See `docs/superpowers/specs/2026-05-22-perf-profiling-design.md`
for the full design, including the rationale for each profile type
and the best-effort error policy.
