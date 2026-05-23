# Logging Standardization — Design

**Date:** 2026-05-23
**Status:** Approved (brainstorming complete)
**Scope:** Tier 3 (Go-native operational logging) only.

## Problem

Diagnostic output in the codebase is inconsistent. A survey of `pkg/` and
`cmd/` (excluding tests) found:

- 42 `fmt.Print*` calls — most write to **stdout**, with ad-hoc or absent
  subsystem prefixes.
- 8 stdlib `log.*` calls — write to **stderr** (7 of them in `pkg/profiling`,
  already prefixed `profiling:`).
- 0 `slog`, 0 builtin `println`, 0 direct `os.Stderr`/`os.Stdout`.

The inconsistency that matters: **operational diagnostics are split across
stdout and stderr**, so a consumer of the client's stdout (e.g. the startup
banner) also captures interleaved error chatter.

## Constraint: bug-for-bug fidelity

This is a faithful Java→Go port (`PORTING.md` §2 rule 1 + 4). Many output
calls are **exact 1:1 translations of Java `System.out.println`** and must
not be reformatted — doing so would break the 1:1 Java mapping the project
relies on for diff-audits. The work is therefore scoped to output that has
**no Java equivalent** (machinery the port added).

## Classification

### Tier 1 — Faithful Java `System.out.println` ports — EXCLUDED (do not touch)

Verified character-for-character against `Client-Java` (`225-clean`):

| Go site | Java source |
|---|---|
| `config/flotype/flotype.go:69` | `FloType.java:83` |
| `config/idktype/idktype.go:71` | `IdkType.java:75` |
| `config/seqtype/seqtype.go:109` | `SeqType.java:124` |
| `config/spotanimtype/spotanimtype.go:89` | `SpotAnimType.java:106` |
| `config/varptype/varptype.go:70` | `VarpType.java:95` |
| `dash3d/world3d/world3d.go:2145` | `World3D.java:2174` |
| `graphics/model/model.go:334` | `Model.java:414` |
| `graphics/pix32/pix32.go:45` | `Pix32.java:65` |
| `io/bzip2/bzip2.go:247` | `BZip2.java:203` |
| `sign/signlink/signlink.go:516` | `signlink.java:361` (`reporterror`) |

### Tier 2 — CLI program output — EXCLUDED (do not touch)

`cmd/client/main.go` lines 19, 21, 27, 32, 41, 50 — startup banner, usage
text, and arg-parse errors. This is user-facing stdout UX, not logging; the
banner mirrors Java's startup line.

### Tier 3 — Go-native operational logging — CONVERT (27 sites)

No Java equivalent (panic-recovery handlers, the audio subsystem, signlink
FS/net error reporting). These are the standardization target.

| Subsystem | Sites | Prefix |
|---|---|---|
| `client` | client.go: 60, 1578, 2530, 2538, 2543, 2565, 5559, 5582, 5590 (9) | `client: ` |
| `audio` | audio.go:58; midi.go:183, 209, 217, 389; soundfont.go:35; wave.go:51, 56 (8) | keep existing `audio:` / `audio/midi:` / `audio/wave:` |
| `signlink` | signlink.go: 160, 173, 210, 215, 237, 249, 254, 276, 531 (9) | `signlink: ` |
| `gameshell` | gameshell.go:46 (1) | `gameshell: ` |

### Already conforms — leave as-is

The 7 `pkg/profiling/profiling.go` `log.Printf` calls (lines 154, 162, 169,
177, 190, 196, 234). They are the de-facto template for this convention.

## Convention

1. **Mechanism:** stdlib `log` — `log.Printf` for the 26 non-fatal sites,
   `log.Fatalf` for `gameshell.go:46` (which must still `os.Exit(1)`).
   No new package, no logger globals, no `slog`.
2. **Destination:** stderr (the std logger's default). This is the
   substantive change — it moves the 27 currently-stdout `fmt.Printf` lines
   to stderr.
3. **Prefix:** inline `"<subsystem>: "` at the start of each format string.
   `client`/`signlink`/`gameshell` gain a prefix; `audio` already has good
   per-file prefixes and keeps them. `client` keeps its descriptive context
   after the prefix (e.g. `client: RunMidi error: %v`).
4. **Newlines:** drop the trailing `\n` from every converted format string —
   `log` appends its own line terminator. (Failing to do this produces a
   double-newline.)
5. **Flags:** keep `log`'s defaults (date+time). No `log.SetFlags` /
   `log.SetOutput` call is added. Consistent with `profiling`, and
   timestamps aid diagnosis of real-play audio/network/cache failures.

## Out of scope (YAGNI)

- No log levels / `slog` / structured key-value output.
- No internal logging package or per-subsystem `*log.Logger` instances.
- No change to `profiling` (already conforms).
- No touching Tier 1 or Tier 2.

## Verification

1. `go build ./...` — clean.
2. `go vet ./...` — clean.
3. `go test -race ./...` — clean.
4. `grep` confirms every Tier 1 + Tier 2 line (including `reporterror` at
   `signlink.go:516`) is byte-identical to before.
5. `grep` confirms no converted `log.Printf`/`log.Fatalf` format string ends
   in `\n`.
6. Confirm `grep -rn 'fmt.Print' pkg cmd | grep -v _test` returns exactly the
   16 excluded `fmt.Print*` sites (10 Tier 1 — the 9 config/graphics ports
   **plus** `signlink.go:516` `reporterror` — and 6 Tier 2). I.e. all 26
   Tier 3 `fmt.Print*` calls are gone. (The 27th Tier 3 site,
   `gameshell.go:46`, is a `log.Fatal` → `log.Fatalf` conversion, not a
   `fmt.Print*`.)

## Expected diff size

~27 single-line edits across 7 files (`client.go`, `audio.go`, `midi.go`,
`soundfont.go`, `wave.go`, `signlink.go`, `gameshell.go`) plus `"log"` /
`"fmt"` import adjustments where a file loses its last `fmt.Print` or gains
its first `log` use.
