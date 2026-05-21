# PORTING.md

A living roadmap for the Java → Go port of the RuneScape 2 (rev #225) client.

This document tracks **what is incomplete** in the port and the **order in
which the remaining work should be tackled**. It is updated as work lands.
Cross-reference with `README.md` (translation conventions) and `CLAUDE.md`
(architecture overview).

## 1. References

| Repo | Path | Role |
|---|---|---|
| Java client | `$HOME/Code/github.com/LostCityRS/Client-Java` (branch `225-clean`) | **Authoritative source** for translation. Every Go change should map to a Java function. |
| TypeScript client | `$HOME/Code/github.com/LostCityRS/Client-TS` | Secondary cross-check for ambiguous Java → Go translations. |
| Go server | `$HOME/Code/github.com/zsrv/goscape` | **Reference only**, no code reuse. Useful as a third source of truth for the wire protocol (it is the server side of `ClientStream`). |

## 2. Porting Philosophy

These rules govern how to decide between "faithful translation" and "idiomatic
Go" when they conflict:

1. **Faithful 1:1 translation is the default.** Preserve Java function names,
   parameter order, control flow, and even apparent oddities (e.g. `var7` style
   locals translated to `var7` in Go). The goal is that any Java line should
   map to a small, identifiable region of Go code so behavior bugs can be
   diff-checked against the reference.
2. **Adapt to Go's type system rigorously.** Apply the conversions documented
   in `README.md` and `CLAUDE.md` (`byte → int8`, etc.). Use parentheses where
   Java/Go operator precedence differs.
3. **Replace Java idioms with Go equivalents only when there is no other
   option:**
    - `synchronized` blocks → `sync.Mutex` only where state is actually shared
      across goroutines. Many `synchronized` methods in Java existed for
      defensive reasons that don't apply here. Annotate the choice with a
      comment referencing the Java source.
    - `Thread.start` / `Runnable` → `go func()` goroutine. Where Java does
      `shell.startThread(this, 2)` the Go side calls the equivalent run-loop
      method in a goroutine directly.
    - `try/catch` → idiomatic Go error handling, but only at boundaries where
      the Java code actually relies on the exception flow. Many translated
      `// TODO: try/catch` markers are vestigial and can be deleted with a
      one-line comment explaining why.
    - `Applet` API and `signlink.mainapp` checks → no-op, since the Go client
      is always a standalone binary.
4. **Don't refactor opportunistically.** No renaming, no extracting helpers,
   no replacing `Packet` with a more idiomatic buffer, no folding global
   `var` blocks into structs. Faithful structure is more valuable than
   incremental cleanliness while the port is still in progress.
5. **No code reuse from `goscape` (server).** Read it for protocol cross-checks,
   but do not import or copy. The server is a clean-sheet rewrite with a
   different architecture; mixing them now would obscure the Java mapping and
   couple two repositories that should evolve independently.

When in doubt, write the literal translation and leave a `// TODO: verify`
comment.

## 3. Build & Test

Per `CLAUDE.local.md`, all Go commands run with explicit temp dirs:

```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go build ./...
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./...
```

Git commits use `git commit --no-gpg-sign` (per global `CLAUDE.md`), and
because the GPG agent is sandboxed read-only, commits often have to be made
from a terminal outside Claude Code.

## 4. Gap Inventory

Severity legend:
- **🔴 blocker** — prevents a core gameplay flow (connecting, logging in,
  rendering the world).
- **🟡 important** — degrades a feature or correctness but the game can still
  reach a playable state without it.
- **⚪ cosmetic** — applet-only code, dead code, error-reporting niceties, or
  pure verification-style TODOs left over from the translation pass.

### 4.1 Whole files / packages missing

| Java source | Expected Go location | Severity | Notes |
|---|---|---|---|
| ~~`jagex2/io/ClientStream.java` (182 lines)~~ | ~~`pkg/jagex2/io/clientstream/clientstream.go`~~ | ~~🔴~~ | ~~TCP socket wrapper with buffered writer goroutine. Required by login and every in-game packet. See §5.1.~~ **Ported 2026-05-21.** |
| `jagex2/datastruct/HashTable.java` (47 lines) | `pkg/jagex2/datastruct/hashtable/hashtable.go` | 🟡 | Hash bucket used by several config/entity systems. Currently unported; absence may be masked by callers that haven't been ported either. |
| ~~`sign/signlink.java` `opensocket` (lines 279–291)~~ | ~~`pkg/sign/signlink/signlink.go` (in `Run()` loop + new `OpenSocket` func)~~ | ~~🔴~~ | ~~Function entirely stubbed (line 215 `// TODO: OpenSocket`). See §5.2.~~ **Ported 2026-05-21.** Direct `net.DialTimeout` (skipped the polling pattern); `SocketReq` global and `Run()` branch removed. |
| `jagex2/client/ViewBox.java` | `pkg/jagex2/client/viewbox.go` (14 lines, empty) | 🟡 | Stub struct only. AWT-derived; may be replaceable by a tiny Gio adapter rather than a literal port. |

Note: `deob/client.java` (10,643 lines) and `deob/ObfuscatedName.java` are
**not missing** — the deobfuscated client moved into `pkg/jagex2/client/client.go`
during the merge documented in commit `e136d42`, and `ObfuscatedName` is a Java
annotation with no runtime equivalent in Go.

### 4.2 Networking gaps in `pkg/jagex2/client/client.go`

All blockers in this section depend on §5.1 (ClientStream port) and §5.2
(`signlink.OpenSocket`) landing first.

| Line | Function | Gap | Severity |
|---|---|---|---|
| ~~483~~ | ~~`Client` struct~~ | ~~`//Stream ClientStream // TODO` — field commented out.~~ **Ported 2026-05-21.** | ~~🔴~~ |
| 4865 | `GetHost` | Returns hardcoded `"127.0.0.1"`. Should resolve from codebase / args. | 🔴 |
| ~~6178~~ | ~~`LoginFunc`~~ | ~~`// TODO: clientstream` — `c.stream = new ClientStream(...)` not wired.~~ **Ported 2026-05-21.** | ~~🔴~~ |
| ~~6214~~ | ~~`LoginFunc`~~ | ~~`// TODO: stream.write` — login handshake bytes never sent.~~ **Ported 2026-05-21.** | ~~🔴~~ |
| ~~6215~~ | ~~`LoginFunc`~~ | ~~`var7 := 0 // TODO: placeholder - var7 stream.read` — login response not read.~~ **Ported 2026-05-21.** | ~~🔴~~ |
| ~~6497~~ | ~~`Unload`~~ | ~~`// TODO: stream.close` — leaks the socket on shutdown.~~ **Ported 2026-05-21.** | ~~🟡~~ |
| ~~6615~~ | ~~(commented)~~ | ~~`//func (c *Client) OpenSocket(arg0 int)` — function not written.~~ **Ported 2026-05-21.** | ~~🔴~~ |
| ~~6938~~ | ~~(heartbeat write)~~ | ~~`// TODO: stream write` — periodic write not connected.~~ **Ported 2026-05-21.** | ~~🔴~~ |
| ~~7761~~ | ~~`TryReconnect`~~ | ~~`// TODO: c.stream` — local `var2 = this.stream` save before reconnect.~~ **Ported 2026-05-21.** | ~~🟡~~ |
| ~~7767~~ | ~~`TryReconnect`~~ | ~~`// TODO: c.stream.close()` — close pre-existing stream before retry.~~ **Ported 2026-05-21.** | ~~🟡~~ |
| ~~8623~~ | ~~`Read`~~ | ~~Entire function is `return false // TODO: stub - c.stream`. This is the main inbound packet dispatcher (~100 lines in Java).~~ **Ported 2026-05-21** in steps 7a-7f. | ~~🔴~~ |
| ~~3362–3363~~ | ~~`Logout`~~ | ~~`// TODO: c.Stream.Close()`, `// TODO: c.Stream = nil`.~~ **Ported 2026-05-21.** | ~~🟡~~ |

### 4.3 Other client.go gaps

| Line | Function | Gap | Severity |
|---|---|---|---|
| 595 | `Client` struct | `//MidiSync: // TODO` — field still commented. | 🟡 |
| 656 | `SetMidi` | `// TODO: synchronized` — likely needs a small `sync.Mutex` for the MIDI-loader handoff (single producer / consumer, so a channel may be cleaner). | 🟡 |
| 1533 | (MIDI run loop) | `// TODO: synchronized` — same handoff. | 🟡 |
| 1544 | (MIDI download) | `int(crc32.ChecksumIEEE(...)) // TODO: verify conversion` — verify against Java `CRC32.getValue()`. | ⚪ |
| 1558 | (MIDI download) | `var15.ReadAt(...)` semantics differ from Java `RandomAccessFile.read`; verify. | 🟡 |
| 1617, 1643, 1650, 1652, 1669 | `DrawFlames` | Pixel buffer copy/draw operations marked `// TODO: verify`. Visible flicker is the symptom if wrong. | 🟡 |
| 2450 | `GetJagFile` | CRC32 conversion `// TODO: verify`. | ⚪ |
| 2451 | `GetJagFile` | `// TODO: anything missing here?` — sanity check vs. Java needed. | 🟡 |
| 2460 | `GetJagFile` | `// TODO: try/except` — IO error from HTTP download. | 🟡 |
| 2741 | (minimap related) | `// TODO: GetBaseComponent()` — helper unported. | 🟡 |
| 3000 | (background loader) | `// TODO: startThread` — single goroutine spawn, low risk. | 🟡 |
| 3283, 3288 | `DrawTitleScreen` | `// TODO: verify` — title rendering parity. | ⚪ |
| 5310–5311 | `Load` | `//c.startthread(this, 2)` / `go c.Run()` — MIDI run-loop thread launch idiom. | ⚪ |
| 5354–5355 | `Load` | `// TODO: try/except - recover panic?` and `retry := 5 // TODO`. | 🟡 |
| 6049 | `OpenURL` | `// TODO: signlink.openurl for applets not included` — Go isn't an applet; remove the TODO. | ⚪ |
| 6101 | `RunFlames` | `// TODO: try/catch` — flame anim error handling. | ⚪ |
| ~~6172~~ | ~~`LoginFunc`~~ | ~~`// TODO: try/catch` — wrap login in a recover/log.~~ **Ported 2026-05-21** — handled inline at each I/O site (4 checks → `"Error connecting to server."`), matching Java's single outer `catch (IOException)` semantics. | ~~🟡~~ |
| 6976 | `GetCodeBase` | `// TODO: getcodebase signlink - signlink.mainapp` — replace with config-driven host. | 🟡 |
| 7224 | `UpdateInterfaceAnimation` | `// TODO: verify or` — operator precedence check. | ⚪ |
| 7747 | (commented) | `//func (c *Client) GetParameter(...)` — applet HTML parameters; not applicable to Go. Delete with a one-line comment. | ⚪ |
| 7863 | `BuildScene` | `// TODO: try/catch` — IO error from cache read. | ⚪ |
| 7986 | `ExecuteClientscript1` | `// TODO: try/catch`. | ⚪ |
| 8079 | `DrawError` | Entire function is `// TODO: stub`. Needed if connection/login fails visibly. | 🟡 |
| 8731 | `GetPlayerExtended2` | `// TODO: try/catch`. | ⚪ |

### 4.4 GameShell / input / pixmap gaps

| File:Line | Function | Gap | Severity |
|---|---|---|---|
| `client/gameshell.go:22–24` | `InitApplication` | `DrawArea` link to window component; window open timing. | 🟡 |
| `client/gameshell.go:38` | `InitApplication` | `go app.Main() // TODO: go?` — pattern for the Gio main goroutine. | 🟡 |
| ~~`client/gameshell.go:86`~~ | ~~(event loop)~~ | ~~`// TODO: listeners` — keyboard/mouse event listeners not attached.~~ **Ported 2026-05-21** (Phase 2 step 1). Gio events drained per-frame inside `draw()`'s FrameEvent via `event.Op` + `source.Event(c.inputFilters...)`. | ~~🔴~~ |
| ~~`client/gameshell.go:168`~~ | ~~`PollKey`~~ | ~~`return 0 // TODO: stub` — never returns a real key.~~ **Ported 2026-05-21**. Ring-buffer pop matching `GameShell.java:459-466`, returning -1 when empty. | ~~🔴~~ |
| `client/gameshell.go:172` | `DrawProgress` | `// TODO: stub` — progress UI during cache load. | 🟡 |
| ~~`client/inputtracking/inputtracking.go:20`~~ | ~~(package)~~ | ~~`// TODO: all funcs synchronized` — concurrent access from event goroutine and game loop.~~ **Ported 2026-05-21** (Phase 2 step 3 folded into step 1). Package-level `sync.Mutex` wraps every public function; internal helpers (`ensureCapacity`, `setDisabledLocked`) document the non-locking contract. | ~~🟡~~ |
| `client/viewbox.go:10–14` | `ViewBox` | Whole struct is a stub. | 🟡 |
| ~~`graphics/pixmap/pixmap.go:18`~~ | ~~(file)~~ | ~~`// TODO` — pipeline orientation comment, may be vestigial.~~ **Resolved 2026-05-21** (Phase 2 step 4). Vestigial — deleted. | ~~⚪~~ |
| ~~`graphics/pixmap/pixmap.go:45`~~ | ~~`NewPixMap`~~ | ~~`image.NewRGBA(...) // TODO: unused` — may be dead allocation now that `convertPixmapPixels` uses NRGBA (commit f1eca00).~~ **Resolved 2026-05-21** (Phase 2 step 4). Confirmed unread anywhere in `pkg/` or `cmd/`; `Image` field and its allocation deleted. | ~~⚪~~ |
| ~~`graphics/pixmap/pixmap.go:62–63`~~ | ~~`Draw`~~ | ~~Concurrent `ops.Ops` access concern. The `DrawMu` mutex was added later (`CLAUDE.md`) — verify these TODOs are now stale.~~ **Resolved 2026-05-21** (Phase 2 step 4). `DrawMu` renamed `OpsMu` and extended to cover the FrameEvent block (`event.Op` + drain + `e.Frame`), so every `c.Ops` access is serialized. | ~~🟡~~ |

### 4.5 Lower-priority verification TODOs

These are all `// TODO: verify`-style markers from the original translation
pass. None are blockers; each is a small, isolated correctness check against
the Java source. Tracked here so the list isn't lost.

- `pkg/jagex2/datastruct/lrucache.go:14` — `HashTable: make(map[int64]T, 0x400) // TODO: not limited to 0x400`
- `pkg/jagex2/datastruct/jstring/jstring.go:51` — return of fixed-size builder slice
- `pkg/jagex2/io/bzip2/bzip2.go` lines 15, 141, 384, 418, 515, 549, 578 — several `byte ↔ int` conversion markers
- `pkg/jagex2/graphics/pix32/pix32.go:37–72` — five markers, mostly applet `MediaTracker` / `PixelGrabber` and pixel type
- `pkg/jagex2/graphics/pix3d/pix3d.go:280` — color packing arithmetic
- `pkg/jagex2/graphics/model/model.go:259,264` — possibly-nil return where Java would return an empty Model
- `pkg/jagex2/wordenc/wordpack/wordpack.go:59,64` — slice bounds and `80` truncation
- `pkg/jagex2/config/component/component.go:182,224,233` — string parsing in `Unpack`
- `pkg/jagex2/dash3d/world/world.go:289,598` — coord swap and bit shift conversion
- `pkg/jagex2/dash3d/entity/pathingentity.go:153` — virtual-dispatch pattern (interface vs. embedded method)
- `pkg/jagex2/sound/wave/wave.go:138` — signed-byte bitwise AND
- `pkg/jagex2/config/loctype/loctype.go:213`, `pkg/jagex2/config/npctype/npctype.go:35` — `*string` vs `string` for optional ops

### 4.6 Signlink-specific notes

Verified state of `pkg/sign/signlink/signlink.go` (the Go side is started in a
goroutine from `cmd/client/main.go`, so `Run()` doesn't deadlock the caller —
that was a false alarm):

- ✅ `DNSLookup` matches Java semantics (sets `dns` to the request before the
  background goroutine resolves it).
- ✅ `CacheLoad`, `CacheSave`, `WaveSave`, `MidiSave` all match the Java
  polling protocol.
- ~~🔴 `OpenSocket` (Java `opensocket`) is completely absent; `Run()` clears
  `SocketReq` without ever creating a connection (lines 71–74).~~ **Ported
  2026-05-21** — `OpenSocket(port int) (net.Conn, error)` dials
  `clientextras.Host:port` directly (10s timeout); `SocketReq` and the
  `Run()` branch are gone.
- 🟡 `OpenURL` returns `[]byte` instead of a streaming reader. Acceptable as a
  simplification, but callers in `client.go` may need adjusting if any stream
  the bytes lazily.
- 🟡 Spin-waits in `CacheLoad`, `CacheSave`, `OpenURL` (`for X != "" { time.Sleep(...) }`)
  are functionally correct under the single-polling-goroutine model but
  unprotected by any memory-barrier primitive. A small `sync.Mutex` or
  channel-based request/response pattern would be more correct; this is
  worth doing once but doesn't block any single feature.
- ⚪ `// TODO: synchronized` markers correspond to Java's `synchronized`
  methods. Most are redundant because the polling goroutine is the only
  writer; document this and remove the markers where applicable.

## 5. Execution Plan

Phases run in dependency order. Each phase ends with `go build ./...` and
`go test ./...` clean, and a commit per logical step.

### Phase 1 — Networking transport (unblocks login)

**Goal:** A logged-in client can exchange packets with a local server.

1. ~~Port `jagex2/io/ClientStream.java` → `pkg/jagex2/io/clientstream/clientstream.go`.~~
   - ~~Wrap `net.Conn` with `bufio.Reader` for `read()`, `read(buf, off, len)`,
     `available()`. Use a 5000-byte ring buffer + writer goroutine for
     `write()`, mirroring Java's `buf/bufPos/bufLen` exactly.~~
   - ~~Provide `Close()` that cancels the writer goroutine via a context or
     channel and closes the conn. Avoid Java's `synchronized` + `notify()`
     pattern — use a buffered channel or `sync.Cond`.~~
   - ~~Tests: round-trip bytes via `net.Pipe()`.~~
   - **Done 2026-05-21.** Used `sync.Cond` (closer line-by-line to Java than
     a channel). `atomic.Bool` for `closed` so lockless reads in `Read` /
     `Available` don't race with `Close`. Tests cover round-trip, offset/length
     argument order, EOF semantics, `Close` unblocking a blocked reader,
     `Close` idempotency, write-after-close, and multi-write drain. `go test
     -race` clean.
2. ~~Implement `signlink.OpenSocket(port int) (net.Conn, error)`.~~
   - ~~Since Go isn't sandboxed like a signed applet, skip the request/response
     polling pattern entirely: dial directly and return the conn. Document the
     deviation from the Java protocol in a comment.~~
   - **Done 2026-05-21.** `net.DialTimeout("tcp", host:port, 10s)` against
     `clientextras.Host` (new field, defaults `"127.0.0.1"`). Removed
     `SocketReq` global, the `if SocketReq != 0` branch in `Run()`, and the
     `//Socket // TODO` placeholder. Tests cover round-trip + connect-refused.
3. ~~Add `Client.OpenSocket(port int) (net.Conn, error)` matching Java's
   `client.openSocket(int)` — currently always uses the signlink path since
   `signlink.mainapp` is always nil in Go.~~
   - **Done 2026-05-21.** Body delegates to `signlink.OpenSocket(port)`; the
     Java `signlink.mainapp == null` ternary collapses since Go is always
     standalone. No current Go callers (all `// TODO: clientstream` in
     `LoginFunc`/`TryReconnect`); wiring deferred to steps 4-7.
4. ~~Add `Stream *clientstream.ClientStream` field to `Client`. Uncomment
   `//Stream ClientStream // TODO` at `client.go:483`.~~ **Done 2026-05-21.**
   Field declared as `Stream *clientstream.ClientStream` (pointer — type
   carries goroutine + `sync.Cond`, must not be copied). `NewClient`
   unchanged; nil-pointer zero value matches Java's default-null reference.
5. ~~Wire `LoginFunc` (client.go:6170+) end-to-end: create the stream, write
   login bytes, read the response byte, dispatch the switch. Drop the
   placeholder `var7 := 0`.~~ **Done 2026-05-21.** Verified byte order on the
   wire matches Java `client.java:6786-6820` exactly: openSocket → new
   ClientStream → ReadFully(in.Data, 0, 8) → unpack serverSeed → build out +
   login buffers → Stream.Write(login.Data, login.Pos, 0) → Stream.Read()
   into `var7`. Java rev 225 does *not* send a leading username-hash byte
   (the resume prompt's caveat doesn't apply to this rev). Errors at any of
   the four I/O sites set `LoginMessage0=""` / `LoginMessage1="Error
   connecting to server."` and return, matching Java's outer
   `catch (IOException)`. The `// TODO: try/catch` marker at 6172 collapses
   into that inline handling and is gone.
6. ~~Wire `Logout`, `Unload`, `TryReconnect`, and the heartbeat write
   (client.go:6938) to actually call `c.Stream.Write` / `Close`.~~
   **Done 2026-05-21.** All four sites ported faithfully from Java
   `client.java`: `logout()` 3955-3977 (close + nil), `unload()` 7114-7122
   (close + nil — Java does nil it, contra the resume prompt's note),
   heartbeat block 7566-7580 (`p1isaac(108)` separate from the
   conditional `stream != null && out.pos > 0` write; success path resets
   `out.pos`/`heartbeatTimer`, error path calls `TryReconnect`), and
   `tryReconnect()` 8409-8431 (`var2 := c.Stream` snapshot, then close
   after `LoginFunc`; Go must nil-check `var2` since `var2.Close()` on a
   nil pointer panics, whereas Java's catch swallows the NPE). Java's
   two-branch catch (`IOException → tryReconnect`, `Exception → logout`)
   collapses in Go because `ClientStream.Write` returns a single untyped
   error.
7. ~~Port the inbound packet dispatcher in `Read()` (client.go:8623). This is
   the largest single function still stubbed (~100 lines in Java); break it
   into one PR per opcode group if it gets unwieldy.~~ **Done 2026-05-21**
   across commits `e6b9dff` (7a — framing + dispatcher skeleton), `304edd7`
   (7b — chat/PM), `a8aa26f` (7c — NPC/player updates), `d883e57` (7d — zone
   packet group), `7bbf0ab` (7e — pre-zone span), and 7f-i..7f-iv (post-zone
   span). All Java opcodes in `client.java:8810-10348` now have a Go case.
8. ~~Replace the `GetHost` stub at client.go:4865 with a config-driven host
   (CLI arg → `clientextras.Host`, falling back to `127.0.0.1`).~~
   **Done 2026-05-21.** `GetHost()` now returns `strings.ToLower(clientextras.Host)`,
   matching Java's `.toLowerCase()` in both branches of `getHost()`
   (client.java:5510, 5512). The companion `GetCodeBase()` was also
   threaded through `clientextras.Host` so the HTTP cache fetch (via
   `OpenURL`) hits the same server as the game socket — its prior
   hardcoded `127.0.0.1` would have diverged from any non-localhost
   `--host` override. CLI usage extended to an optional 5th arg `host`;
   omitting it keeps the existing `127.0.0.1` default from
   `clientextras.go:13`. **Phase 1 (networking transport) complete.**

### Phase 2 — Input wiring (unblocks playable UI)

1. ~~Port `gameshell.go:86` event listeners — Gio key/mouse events → the existing
   `inputtracking` package.~~ **Done 2026-05-21.** Gio's modern (post-2024-02)
   pull-per-frame API: `event.Op(&c.Ops, c)` registers the Client pointer as a
   tag inside the `FrameEvent` case, then a loop drains `e.Source.Event(...)`
   against `c.inputFilters` (one `pointer.Filter` plus a `key.Filter` per
   named/letter/digit key, all with `Optional: ModShift|ModCtrl|ModAlt|ModSuper|ModCommand`
   so events fire regardless of modifier state). Java's separate `mousePressed/
   mouseReleased/mouseMoved/mouseDragged/mouseEntered/mouseExited/keyPressed/
   keyReleased` methods collapse into `handlePointer` and `handleKey` switching
   on `pointer.Kind` / `key.State`. The Java→Go key translation lives in
   `keyNameToAwt` (Gio `key.Name` → AWT keyCode for the 25 codes Java's
   override sequence checks) and `keyCharFor` (synthesizes Java's keyChar from
   `key.Event.Name` + `Modifiers`, since Gio reports only uppercase letter
   names and a modifier bitset). Java's `mouseMoved(y, x)` argument swap is
   preserved.
2. ~~Implement `PollKey` (`gameshell.go:168`) returning the next queued key.~~
   **Done 2026-05-21.** Ring-buffer pop, `-1` when empty, mirroring
   `GameShell.java:459-466`.
3. ~~Add a small `sync.Mutex` to `inputtracking` to remove the `// TODO: synchronized`.~~
   **Done 2026-05-21.** Folded into step 1 — single package-level `mu sync.Mutex`
   wraps every public function. Internal helpers (`EnsureCapacity`,
   `setDisabledLocked`) document their non-locking contract so the
   non-reentrant `sync.Mutex` doesn't deadlock when `Stop` → `setDisabledLocked`
   under the same lock.
4. ~~Resolve the pixmap concurrency TODOs (`pixmap.go:62–63`) — confirm `DrawMu`
   coverage and delete stale markers.~~ **Done 2026-05-21.** `DrawMu` renamed to
   `OpsMu` (it now guards more than `PixMap.Draw`) and its critical section
   extended in `gameshell.go`'s `FrameEvent` case to cover `event.Op(&c.Ops, c)`,
   the `e.Source.Event(...)` drain, and `e.Frame(&c.Ops)`. The dead
   `PixMap.Image *image.RGBA` field and its allocation were also removed.
   Verification: `go build ./...` clean; `go test ./... -race` passes
   (no pixmap tests exercise the path; runtime smoke-test of the title
   screen deferred — sandbox has no display server).

   Note: Client-side input fields (`MouseX/Y`, `MouseButton`, `MouseClick*`,
   `ActionKey`, `KeyQueue`, `KeyQueueReadPos/WritePos`, `IdleCycles`) are
   written from the Gio goroutine and read unsynchronized from the game loop —
   matching Java's AWT-EDT/game-thread split. `go test -race` is clean because
   no test exercises that code path; runtime smoke-test will need its own pass
   (sandbox lacks a display server).

### Phase 3 — Audio handoff

1. Replace MIDI-sync TODOs (`client.go:656`, `client.go:1533`, and field at
   `client.go:595`) with a single-element buffered channel between the
   loader and the playback goroutine. Document this as the Go equivalent of
   the Java `synchronized` / `wait` / `notify` pattern.

### Phase 4 — Missing utility types

1. Port `jagex2/datastruct/HashTable.java` → `pkg/jagex2/datastruct/hashtable/`.
   Audit callers in `client.go` and config types; some may currently use the
   Go built-in `map` where Java used `HashTable` and need updating.
2. Decide whether `ViewBox` is worth a literal port or whether it can be
   replaced by a Gio-native equivalent (the Java version is AWT-derived).
   Document the decision inline.

### Phase 5 — Error reporting & cleanup

1. Implement `DrawError` (`client.go:8079`).
2. Sweep `// TODO: try/catch` markers: convert to idiomatic Go errors at
   boundaries, delete the rest with a one-line comment per Java site.
3. Strip applet-only TODOs (`signlink.openurl for applets`, `GetParameter`,
   `signlink.mainapp` checks) and document in code comments why they don't
   apply.
4. Work through §4.5 verification TODOs in batches by package.

### Phase 6 — Hardening

1. Spin-wait → channel/mutex conversion in `signlink`.
2. Race-detector run: `go test -race ./...`.
3. Optional: add Java-side cross-check tests for any complex algorithm
   (model rendering, bzip2, ISAAC) that already has TODOs flagged.

## 6. Conventions for Updating This File

- Mark items 🔴 / 🟡 / ⚪ as you go; strike them through (`~~text~~`) when
  landed, then prune in a periodic cleanup.
- When a new gap is discovered that doesn't fit an existing bucket, add it
  to §4 with a file:line reference and a one-line description.
- When the phase plan in §5 changes (e.g. a step turns out to depend on
  something not anticipated), revise the order rather than leaving stale
  steps in place.
- Commit message convention: prefix porting work with `port:` (e.g.
  `port: ClientStream + signlink.OpenSocket`), separate from `Bug fixes` /
  `Renaming` style commits already in the log.
