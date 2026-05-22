# Performance profiling infrastructure — design

**Date**: 2026-05-22
**Status**: Approved by user (brainstorming session)
**Author**: brainstorming session with Claude Code

## Purpose

Establish a comprehensive performance baseline for the goscape-client so
future work can be measured against a known-good reference point. The
artifact this design produces is **infrastructure** — a programmatic
profile-capture mechanism — not a one-off profiling run. Once landed, the
user can produce a fresh baseline at any time, and future regression
checks can diff against earlier captures.

## Scope

In scope:

- New `pkg/profiling/` package with a single public entry point.
- SIGUSR1-triggered, in-process capture of all four standard pprof
  profile types plus a runtime execution trace.
- Output organized on disk per session.
- Best-effort error handling — profiling never affects gameplay.
- Unit tests for the capture function.

Out of scope (deliberate exclusions, per YAGNI):

- HTTP `/debug/pprof` endpoint. Rejected because exposing a debug port
  during normal play adds attack surface for a feature the user does
  not want to use interactively.
- In-client keybind trigger (e.g. F12). Rejected because Gio key-event
  plumbing is unnecessary complexity when an external `kill -USR1`
  is just as ergonomic in the user's workflow.
- Build-tag gating. Rejected because mutex/block profiling has zero
  cost when the fractions are 0, and a stray SIGUSR1 in production is
  not a real concern for this client.
- Configurable CPU window duration via env var / flag. Rejected as
  unnecessary API surface; the hardcoded 30s can be changed in source
  later if the need arises.
- Resolving output dir to the binary's directory rather than CWD.
  Rejected because there is no shipping binary scenario yet.

## Architecture

```
┌───────────────────────────────────────────────────────────┐
│  cmd/client/main.go                                       │
│  profiling.Start()  ← single new call, before wg.Go's     │
└─────────────┬─────────────────────────────────────────────┘
              │
┌─────────────▼─────────────────────────────────────────────┐
│  pkg/profiling/profiling.go                               │
│                                                           │
│  func Start()                       (public, non-blocking)│
│    └─ goroutine: signal listener                          │
│         └─ on SIGUSR1: `go captureAll(...)`               │
│                                                           │
│  func captureAll(outBase string, cpuWindow time.Duration) │
│    (internal; parameterized for testability)              │
└───────────────────────────────────────────────────────────┘
```

### Public surface

Exactly one exported symbol: `func Start()`. It:

1. Creates a buffered channel `chan os.Signal` of size 1.
2. Calls `signal.Notify(ch, syscall.SIGUSR1)`.
3. Spawns one listener goroutine that loops on `<-ch`, launching a
   capture goroutine on each fire.
4. Returns immediately.

No `Stop()`. The process lives until shutdown; profiling lives with it.

### Concurrency model

Two layers of goroutine separation:

- **Listener goroutine** — does nothing but receive on the signal
  channel and dispatch. Must never block, because if the buffer is
  full when a signal arrives, the runtime drops the signal silently.
- **Capture goroutines** — one per SIGUSR1 fire, do all I/O.
- **`inFlight` atomic.Bool** — CAS'd at the top of `captureAll`.
  If a second SIGUSR1 fires during an active capture, the second
  goroutine sees the CAS fail and logs `"profiling: capture already
  in flight, ignoring SIGUSR1"` before returning. **Policy: ignore,
  do not queue.** Queuing would surprise users expecting each fire
  to start an immediate sample.

A `defer inFlight.Store(false)` at the top of `captureAll` ensures the
slot is released even if the capture panics or aborts mid-way.

## Capture mechanics

A single SIGUSR1 fire produces six artifacts in a fixed sequence:

| Order | Artifact | Mechanism | Window |
|---|---|---|---|
| 1 | Enable mutex/block sampling | `runtime.SetMutexProfileFraction(1)`, `runtime.SetBlockProfileRate(1)` | instant |
| 2 | Start `cpu.prof` | `pprof.StartCPUProfile(w)` | 30 s |
| 3 | Start `trace.out` | `runtime/trace.Start(w)` | 30 s |
| 4 | Sleep 30 s | `time.Sleep(cpuWindow)` | — |
| 5 | Stop CPU profile | `pprof.StopCPUProfile()` | instant |
| 6 | Stop trace | `trace.Stop()` | instant |
| 7 | `runtime.GC()` then snapshot `heap.prof` | `pprof.Lookup("heap").WriteTo(w, 0)` | instant |
| 8 | Snapshot `goroutine.prof` | `pprof.Lookup("goroutine").WriteTo(w, 0)` | instant |
| 9 | Snapshot `mutex.prof` | `pprof.Lookup("mutex").WriteTo(w, 0)` | instant |
| 10 | Snapshot `block.prof` | `pprof.Lookup("block").WriteTo(w, 0)` | instant |
| 11 | Disable mutex/block sampling | reset both to 0 | instant |
| 12 | Log absolute path of output dir to stderr | `filepath.Abs(dir)` then `log.Printf` | — |

### Why this ordering matters

- **CPU + trace run concurrently** in steps 2–6. The stdlib supports
  this; they don't interfere.
- **Mutex/block sampling spans the same 30s window** as the CPU
  profile. This is intentional — contention data and CPU data should
  cover the same period of game activity.
- **`runtime.GC()` before heap snapshot** ensures the heap profile
  reflects post-mark state (in-use vs garbage classification is
  computed against the most recent GC cycle). The `net/http/pprof`
  heap handler does this internally; programmatic capture must do
  it explicitly.
- **Mutex/block fractions are reset to 0 after snapshot** so the
  rest of the process runs with zero contention-profiling overhead
  between sessions.

### CPU window duration

Hardcoded `const cpuWindow = 30 * time.Second` in `profiling.go`.
The internal `captureAll` takes the window as a parameter (for
testability with shorter windows), but the public `Start()` always
passes the constant.

## Output layout

Sessions land under `./profiles/` (relative to process CWD at start):

```
./profiles/
  20260522T143015Z/
    cpu.prof
    heap.prof
    goroutine.prof
    mutex.prof
    block.prof
    trace.out
  20260522T145402Z/
    ...
```

**Timestamp format**: `YYYYMMDDTHHMMSSZ` (ISO 8601 basic, UTC).
Lexicographically sortable, no colons (Windows-safe), no whitespace.

**Directory creation**: `os.MkdirAll(path, 0o755)`. Creates `profiles/`
on first capture, idempotent on later runs. Never deletes; cleanup is
manual.

**Per-file handling**: each artifact uses
`os.OpenFile(path, O_CREATE|O_WRONLY|O_TRUNC, 0o644)`, then
`defer f.Close()` **inside** the helper that writes that one file —
not via a single shared defer — so one file's failure can't leak
another file's handle.

**Size note**: a 30s `trace.out` for typical gameplay is roughly
30–80 MB. A session directory lands at ~100 MB total. The user should
clean `./profiles/` periodically; we will note this in the README
addition (see "Documentation" below).

## Error handling

Profiling is **strictly best-effort**. Failures must never crash,
slow, or otherwise affect normal gameplay.

| Failure | Behavior |
|---|---|
| `MkdirAll` fails (read-only fs, permissions) | Log, abort this capture, listener keeps running |
| Individual `OpenFile` fails | Log, skip that one artifact, continue with the others |
| `pprof.StartCPUProfile` returns error | Log, skip CPU + trace, still snapshot the four instant profiles |
| `runtime/trace` fails to start | Log, skip trace, CPU profile continues unaffected |
| Panic anywhere in capture | `recover()` at the top of `captureAll` logs the stack; `defer` still releases `inFlight`; listener keeps running |
| Process exits mid-capture | Whatever artifacts were already written are intact; the in-flight CPU + trace get truncated, which is acceptable |

The `recover()` at the capture-goroutine entry is the single most
important safety: this is the only code in the project running from
a signal-derived goroutine, and an uncaught panic there would
otherwise take down the working game session.

The "skip-one-artifact, continue with others" policy is a deliberate
departure from error-fast Go style. Profiles are independent
diagnostic outputs, not a single transaction — five of six is
dramatically more useful than zero.

## Testing strategy

One test file: `pkg/profiling/profiling_test.go`.

Tests:

1. **`captureAll` produces the expected file set in a temp dir.**
   Pass `t.TempDir()` as the output base and a 100ms window. Assert
   all six files exist and have non-zero size after the call returns.

2. **`captureAll` is reentrancy-safe.** Spawn two `captureAll` calls
   concurrently from the test; assert exactly one session directory
   is created. The second call should observe the `inFlight` CAS
   failure and return without writing anything.

3. **Filename timestamp format is sortable.** Generate two timestamps
   spaced by a few ms; assert lexicographic string ordering matches
   time ordering.

4. **Mutex/block fractions are reset after capture.** Read the
   previous fraction with `runtime.SetMutexProfileFraction(-1)`
   (which returns prior value without changing it) before and after
   the capture call. Assert both are 0 after capture completes,
   regardless of starting value.

**Not tested**: the `signal.Notify` registration itself. Sending a
real signal to the test process is brittle and the four-line
registration function provides essentially no value beyond what
visual inspection guarantees.

## Files to be created or modified

**Created**:

- `pkg/profiling/profiling.go` — the package implementation
- `pkg/profiling/profiling_test.go` — unit tests
- `docs/superpowers/specs/2026-05-22-perf-profiling-design.md` — this document

**Modified**:

- `cmd/client/main.go` — one import, one call to `profiling.Start()`
  placed after argument parsing and before the `wg.Go` launches
- `README.md` — short section noting:
  - How to trigger a capture: `kill -USR1 <PID>` (find the PID with
    `pgrep -f 'goscape-client|cmd/client'` or note it from the log
    line we add at `Start()`, see below)
  - Output location: `./profiles/<timestamp>/` (absolute path logged
    on each successful capture)
  - Expected output size (~100 MB per session)
  - How to view: `go tool pprof <file>`, `go tool trace trace.out`

`Start()` should additionally log its own PID once at registration
time (`log.Printf("profiling: signal listener ready, send SIGUSR1 to pid %d", os.Getpid())`)
so the user does not have to look up the PID separately.

## Conventions

- Use Go 1.26 (project Go version per `CLAUDE.local.md`).
- Invoke `use-modern-go` skill before writing the implementation.
- Prefix any `go` command invocations with
  `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=/tmp/claude-1000/go`.
- Commit with `git commit --no-gpg-sign`.
- `go.mod` / `go.sum` are intentionally untracked — do not commit
  changes to them. The profiling package uses only stdlib, so no
  module changes are expected anyway.

## Verification plan

After implementation:

1. `go build ./...` clean.
2. `go vet ./...` clean.
3. `go test -race ./pkg/profiling/...` passes.
4. `go test -race ./...` still passes (no regression elsewhere).
5. User runs the client locally, fires `kill -USR1`, confirms the
   session directory appears with all six files. Spot-check one
   profile via `go tool pprof ./profiles/<ts>/cpu.prof` to confirm
   it opens cleanly.
6. User fires SIGUSR1 again during the 30s window; confirms log
   message and no second directory created.

## Risks and open questions

- **Cross-platform note**: SIGUSR1 is POSIX. If the project ever
  targets Windows, this trigger needs replacement. Not a concern
  now — Linux is the development target.
- **`runtime/trace` overhead during the 30s window** is small but
  non-zero (1–3% CPU, per Go team's own docs). The user should be
  aware that a captured baseline reflects "client + light tracing,"
  not "client alone." For most purposes this is irrelevant; for
  micro-benchmarking it matters and we could split the trace into
  a separate signal (SIGUSR2) later if it becomes a real issue.
