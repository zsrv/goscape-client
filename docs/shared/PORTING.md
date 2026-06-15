# PORTING.md

A living roadmap for the Java → Go port of the RuneScape 2 (rev #225) client.

This document tracks **what is incomplete** in the port and the **order in
which the remaining work should be tackled**. It is updated as work lands.
Cross-reference with `README.md` (translation conventions) and `CLAUDE.md`
(architecture overview).

## 1. References

`$HOME/...` paths below are local checkout locations; the public upstreams (with
per-revision branch/commit pins) are listed in [`REFERENCES.md`](REFERENCES.md).

| Repo | Upstream · local checkout | Role |
|---|---|---|
| Java client | <https://github.com/LostCityRS/Client-Java> · `$HOME/Code/github.com/LostCityRS/Client-Java` (branch `225-clean`) | **Authoritative source** for translation. Every Go change should map to a Java function. |
| TypeScript client | <https://github.com/LostCityRS/Client-TS> · `$HOME/Code/github.com/LostCityRS/Client-TS` | Secondary cross-check for ambiguous Java → Go translations. |
| Go server | <https://github.com/zsrv/goscape> · `$HOME/Code/github.com/zsrv/goscape` | **Reference only**, no code reuse. Useful as a third source of truth for the wire protocol (it is the server side of `ClientStream`). |

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

> **Architecture note (updated 2026-06-02) — two refactors postdate most of the
> completed (struck-through) items in §4 and §5.** Those items faithfully record
> *how the work was done at the time*; they are kept as an audit trail, but the
> mechanisms they describe have since been replaced. Current reality:
>
> 1. **Windowing/rendering — Gio removed (commit `56d8b31`, 2026-05-25).** The
>    client no longer uses Gio. Every reference below to `gioui.org/app`,
>    `app.Window`, `app.Main()`, `FrameEvent`, the `Ops` op-list, `c.Ops`,
>    `OpsMu`/`DrawMu`, `event.Op`, or `e.Source.Event(...)` is **historical**.
>    Rendering now goes through the `platform` seam (see `CLAUDE.md`): native =
>    GLFW + go-gl, browser = syscall/js + WebGL. `pixmap.PixMap.Draw(x, y)` blits
>    via the active `platform` backend's texture upload, on the loop goroutine;
>    out-of-band repaints go through `Client.present(draw func())`
>    (`gameshell.go`). `OpsMu` is gone — the only surviving lock from that area is
>    `Client.flameMu` (`client.go:154`), which guards the `ImageTitle0/1`
>    title-flame buffers between the `RunFlames` writer goroutine and the
>    render-loop readers. So §4.4 and Phase 2's input/pixmap descriptions reflect
>    a since-deleted backend; the *behavior* they ported (event draining,
>    per-frame blit, flame-buffer locking) survives, only the Gio plumbing changed.
> 2. **Client invocation — positional args → named flags (merged to `rev-225` at
>    `339d3ab`, 2026-06-01).** Any "optional Nth positional arg" wording below —
>    notably the `host` arg in Phase 1 step 8 — is superseded by five named flags:
>    `-node-id`, `-mem`, `-world-type`, `-world-server [tcp|ws|wss]://host:port`,
>    and `-ondemand-server [http|https]://host:port`. `clientextras.Host` still
>    exists, now alongside `clientextras.WorldPort`, `OndemandBaseURL`, `WSPath`,
>    and `Transport`. `parseHostArg`/`hostarg.go` were deleted in favour of
>    `parseWorldServer`/`parseOndemandServer`. Design:
>    `docs/superpowers/specs/2026-06-01-cli-flags-design.md`.

### 4.1 Whole files / packages missing

| Java source | Expected Go location | Severity | Notes |
|---|---|---|---|
| ~~`jagex2/io/ClientStream.java` (182 lines)~~ | ~~`pkg/jagex2/io/clientstream/clientstream.go`~~ | ~~🔴~~ | ~~TCP socket wrapper with buffered writer goroutine. Required by login and every in-game packet. See §5.1.~~ **Ported 2026-05-21.** |
| ~~`jagex2/datastruct/HashTable.java` (47 lines)~~ | ~~`pkg/jagex2/datastruct/hashtable/hashtable.go`~~ | ~~🟡~~ | ~~Hash bucket used by several config/entity systems. Currently unported; absence may be masked by callers that haven't been ported either.~~ **Ported 2026-05-21** in `e616bf8`. Faithful port + `Linkable` struct + 7 tests. Caller audit: the only Java call site is `LruCache.java:15`; the Go `LruCache` already uses the built-in `map[int64]T`, so the new package is currently unused — kept for completeness and any future literal-port callers. |
| ~~`sign/signlink.java` `opensocket` (lines 279–291)~~ | ~~`pkg/sign/signlink/signlink.go` (in `Run()` loop + new `OpenSocket` func)~~ | ~~🔴~~ | ~~Function entirely stubbed (line 215 `// TODO: OpenSocket`). See §5.2.~~ **Ported 2026-05-21.** Direct `net.DialTimeout` (skipped the polling pattern); `SocketReq` global and `Run()` branch removed. |
| ~~`jagex2/client/ViewBox.java`~~ | ~~`pkg/jagex2/client/viewbox.go` (14 lines, empty)~~ | ~~🟡~~ | ~~Stub struct only. AWT-derived; may be replaceable by a tiny Gio adapter rather than a literal port.~~ **Decided 2026-05-21** in `4ef21b6`: skip literal port. Java `ViewBox extends java.awt.Frame` exists only for AWT window-chrome plumbing — all three responsibilities (window creation, `(4, 24)` chrome translation, paint forwarding) are already handled inline by Gio's `app.Window` in `gameshell.go`. Stub retained only as a non-nil sentinel for the `Client.Frame != nil` check at `client.go:2235` (gates the `::clientdrop` debug command). Future cleanup TODO documented in-file. |

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
| ~~595~~ | ~~`Client` struct~~ | ~~`//MidiSync: // TODO` — field still commented.~~ **Ported 2026-05-21**: replaced `MidiSync any` with `MidiSyncMu sync.Mutex`, mirroring Java's `midiSync = new Object()` dedicated lock object. | ~~🟡~~ |
| ~~656~~ | ~~`SetMidi`~~ | ~~`// TODO: synchronized` — likely needs a small `sync.Mutex` for the MIDI-loader handoff (single producer / consumer, so a channel may be cleaner).~~ **Ported 2026-05-21**: locks `MidiSyncMu` around the name/CRC/length triplet assignment, matching Java `synchronized (midiSync)`. | ~~🟡~~ |
| ~~1533~~ | ~~(MIDI run loop)~~ | ~~`// TODO: synchronized` — same handoff.~~ **Ported 2026-05-21**: locks `MidiSyncMu` around the read-and-clear of the triplet, slow work (CacheLoad/OpenURL/bzip2) intentionally outside the lock per Java. | ~~🟡~~ |
| ~~1544~~ | ~~(MIDI download)~~ | ~~`int(crc32.ChecksumIEEE(...)) // TODO: verify conversion` — verify against Java `CRC32.getValue()`.~~ **Verified 2026-05-21**. Re-audited: marker already removed in an earlier session; Java `(int)CRC32.getValue()` and Go `int(crc32.ChecksumIEEE(...))` truncate identically because the comparand at the adjacent site is also int-cast. | ~~⚪~~ |
| ~~1558~~ | ~~(MIDI download)~~ | ~~`var15.ReadAt(...)` semantics differ from Java `RandomAccessFile.read`; verify.~~ **Ported 2026-05-21** in `dd305fa`: switched to `var15.Read(var14[i:var4])`, matching Java `DataInputStream.read(buf, off, len)`. Title music download + decompress verified end-to-end. | ~~🟡~~ |
| ~~1617, 1643, 1650, 1652, 1669~~ | ~~`DrawFlames`~~ | ~~Pixel buffer copy/draw operations marked `// TODO: verify`. Visible flicker is the symptom if wrong.~~ **Verified** in `2f853c2` — buffer index math, stride, mix formula all match Java's `drawFlames()`. | ~~🟡~~ |
| ~~2450~~ | ~~`GetJagFile`~~ | ~~CRC32 conversion `// TODO: verify`.~~ **Verified 2026-05-21**. Same `int(crc32.ChecksumIEEE(...))` equivalence as line 1544; comparand is also int-cast. | ~~⚪~~ |
| ~~2451~~ | ~~`GetJagFile`~~ | ~~`// TODO: anything missing here?` — sanity check vs. Java needed.~~ **Verified 2026-05-21**. Loading-error path handled by `loadingError()` per `66260c1`; surrounding code is complete vs. Java. | ~~🟡~~ |
| ~~2460~~ | ~~`GetJagFile`~~ | ~~`// TODO: try/except` — IO error from HTTP download.~~ **Ported 2026-05-21** in `66260c1`. Chunk-read error path now calls `loadingError()` (which sets `c.LoadError = true` and retries with exponential backoff: 5/10/20/40/60s), matching Java's outer `IOException` retry loop. Prior code returned `nil` and silently dropped the retry semantics. | ~~🟡~~ |
| ~~2741~~ | ~~(minimap related)~~ | ~~`// TODO: GetBaseComponent()` — helper unported.~~ **Decided 2026-05-21** in `78f3d32`: intentional non-port. Java's `getBaseComponent()` returns the AWT applet root used (a) as the first arg to PixMap's AWT constructor (Go's `pixmap.NewPixMap(w,h)` needs no Component) and (b) as a target for `drawError`/`drawProgress` AWT blits (Go routes everything through one `pixmap.PixMap` uploaded by Gio). Decision block in code mirrors `viewbox.go`'s precedent. | ~~🟡~~ |
| ~~3000~~ | ~~(background loader)~~ | ~~`// TODO: startThread` — single goroutine spawn, low risk.~~ **Decided 2026-05-21** in `66451eb`: intentional non-port. Java's `startThread(Runnable, int)` dispatches to `Applet.startThread` or `signlink.startthread`; all Go call sites (`client.go ~3093` flames, `~5338` MIDI, `clientstream.NewClientStream`) already use `go method()` directly. Priority arg has no Go equivalent and is intentionally dropped. | ~~🟡~~ |
| ~~3283, 3288~~ | ~~`DrawTitleScreen`~~ | ~~`// TODO: verify` — title rendering parity.~~ **Verified** in `2f853c2`. Alpha/blit operations match Java's `drawTitleScreen()`. | ~~⚪~~ |
| ~~5310–5311~~ | ~~`Load`~~ | ~~`//c.startthread(this, 2)` / `go c.Run()` — MIDI run-loop thread launch idiom.~~ **Verified + tightened** in `e9d348c`. Conversion was already present; comment updated to cite `deob/client.java:5952`, confirm via `Client.Run`'s dispatch (`StartMidiThread=true`/`FlamesThread=false`) that this goroutine reaches `RunMidi`, and call out that `MidiSyncMu` preserves Java's `synchronized (midiSync)` contract. | ~~⚪~~ |
| ~~5354–5355~~ | ~~`Load`~~ | ~~`// TODO: try/except - recover panic?` and `retry := 5 // TODO`.~~ **Ported 2026-05-21** in `7292254`. Java's outer `catch (Exception) { errorLoading = true; }` ported as a deferred `recover()` that sets `c.ErrorLoading = true`. | ~~🟡~~ |
| ~~6049~~ | ~~`OpenURL`~~ | ~~`// TODO: signlink.openurl for applets not included` — Go isn't an applet; remove the TODO.~~ **Done** in `fbe31bf`. One-line `// Java:` reference comment present at the current site. | ~~⚪~~ |
| ~~6101~~ | ~~`RunFlames`~~ | ~~`// TODO: try/catch` — flame anim error handling.~~ **Ported 2026-05-21** in `a237414`. Java's empty `catch (Exception) {}` deleted; Go panics propagate (the right idiom for "unrecoverable", not a swallowing log). One-line `// Java:` comment retained for archaeology. | ~~⚪~~ |
| ~~6172~~ | ~~`LoginFunc`~~ | ~~`// TODO: try/catch` — wrap login in a recover/log.~~ **Ported 2026-05-21** — handled inline at each I/O site (4 checks → `"Error connecting to server."`), matching Java's single outer `catch (IOException)` semantics. | ~~🟡~~ |
| ~~6976~~ | ~~`GetCodeBase`~~ | ~~`// TODO: getcodebase signlink - signlink.mainapp` — replace with config-driven host.~~ **Done 2026-05-21** (pre-existing). Returns `http://<clientextras.Host>:<port>/`, used by `OpenURL`. Same shape as `GetHost`. | ~~🟡~~ |
| ~~7224~~ | ~~`UpdateInterfaceAnimation`~~ | ~~`// TODO: verify or` — operator precedence check.~~ **Fixed** in `41017f0`: Java's `var4 \|= recurse(...)` is a *bitwise-OR-assign on boolean* that does NOT short-circuit. Naive port to `var4 = var4 \|\| recurse(...)` short-circuits and silently drops `recurse()`'s side effects. Now ported as `if recurse {var4=true}`, evaluating `recurse()` unconditionally. **New gotcha class** — saved as porting memory. | ~~⚪~~ |
| ~~7747~~ | ~~(commented)~~ | ~~`//func (c *Client) GetParameter(...)` — applet HTML parameters; not applicable to Go. Delete with a one-line comment.~~ **Done** in `fbe31bf`. One-line `// GetParameter ... intentionally not ported` comment present. | ~~⚪~~ |
| ~~7863~~ | ~~`BuildScene`~~ | ~~`// TODO: try/catch` — IO error from cache read.~~ **Ported 2026-05-21** in `a237414`. Java's empty `catch (Exception) {}` deleted; tail cleanup runs unconditionally as in Java's `finally`-equivalent control flow. | ~~⚪~~ |
| ~~7986~~ | ~~`ExecuteClientscript1`~~ | ~~`// TODO: try/catch`.~~ **Ported 2026-05-21** in `7292254`. Java's `catch (Exception) { return -1; }` ported as named return `result int` + deferred `recover()` setting `result = -1`. | ~~⚪~~ |
| ~~8079~~ | ~~`DrawError`~~ | ~~Entire function is `// TODO: stub`. Needed if connection/login fails visibly.~~ **Ported 2026-05-21** in `978e471`. State-changing parts (`SetFrameRate(1)`, three `FlameActive = false` assignments) and the early-return ordering across `ErrorLoading`/`ErrorHost`/`ErrorStarted` match Java exactly. Text rasterization left as inline `// Java:` comments because Go has no `getBaseComponent().getGraphics()` equivalent yet (same gap as `DrawProgressGameShell`); follow-up when that helper is ported. | ~~🟡~~ |
| ~~8731~~ | ~~`GetPlayerExtended2`~~ | ~~`// TODO: try/catch`.~~ **Ported 2026-05-21** in `a237414`. Java's `catch (Exception) { signlink.reporterror("cde2"); }` applet-diagnostic deleted; Go panics propagate. Also bonus-deleted the parallel marker in `Read`'s opcode-4 case (`signlink.reporterror("cde1")`). | ~~⚪~~ |

### 4.4 GameShell / input / pixmap gaps

> ⚠️ Gio/`OpsMu`/`app.Window`/`FrameEvent` mentions in this section are
> **historical** — see the Architecture note at the top of §4 (Gio was removed
> 2026-05-25; rendering is now the `platform` seam, the surviving lock is
> `Client.flameMu`).

| File:Line | Function | Gap | Severity |
|---|---|---|---|
| ~~`client/gameshell.go:22–24`~~ | ~~`InitApplication`~~ | ~~`DrawArea` link to window component; window open timing.~~ **Documented 2026-05-21** in `8d1d4cc`. Replaced TODO/commented sites with a single `// Java:` block explaining the AWT Frame/Graphics auto-paint loop has no Gio equivalent; the port does an explicit per-FrameEvent PixMap blit (mirrors `viewbox.go`'s precedent). | ~~🟡~~ |
| ~~`client/gameshell.go:38`~~ | ~~`InitApplication`~~ | ~~`go app.Main() // TODO: go?` — pattern for the Gio main goroutine.~~ **Documented 2026-05-21** in `8d1d4cc`. Investigated via Gio docs (`gioui.org/app`): `app.Main()` should run on the OS main thread (required on macOS, looser on Linux/X11). Per `cmd/client/main.go`, `InitApplication` is itself launched from a goroutine, so `app.Main()` is already off the OS main thread regardless of the `go` prefix. Linux-only assumption documented inline; macOS portability fix would require reshaping `cmd/client/main.go` — tracked as a follow-up below. | ~~🟡~~ |
| ~~`client/gameshell.go:86`~~ | ~~(event loop)~~ | ~~`// TODO: listeners` — keyboard/mouse event listeners not attached.~~ **Ported 2026-05-21** (Phase 2 step 1). Gio events drained per-frame inside `draw()`'s FrameEvent via `event.Op` + `source.Event(c.inputFilters...)`. | ~~🔴~~ |
| ~~`client/gameshell.go:168`~~ | ~~`PollKey`~~ | ~~`return 0 // TODO: stub` — never returns a real key.~~ **Ported 2026-05-21**. Ring-buffer pop matching `GameShell.java:459-466`, returning -1 when empty. | ~~🔴~~ |
| ~~`client/gameshell.go:172`~~ | ~~`DrawProgress`~~ | ~~`// TODO: stub` — progress UI during cache load.~~ **Documented 2026-05-21** in `8d1d4cc`. Java's `drawProgress` paints a 304×34 progress bar via the base AWT Component; no Java-visible state mutations to preserve. Function body left empty with a `// Java:` reference comment recording the full Java drawing sequence (mirrors `DrawError`'s `978e471` pattern), awaiting a canvas decision. | ~~🟡~~ |
| ~~`client/inputtracking/inputtracking.go:20`~~ | ~~(package)~~ | ~~`// TODO: all funcs synchronized` — concurrent access from event goroutine and game loop.~~ **Ported 2026-05-21** (Phase 2 step 3 folded into step 1). Package-level `sync.Mutex` wraps every public function; internal helpers (`ensureCapacity`, `setDisabledLocked`) document the non-locking contract. | ~~🟡~~ |
| ~~`client/viewbox.go:10–14`~~ | ~~`ViewBox`~~ | ~~Whole struct is a stub.~~ **Decided** (intentional non-port). File-header comment lays out the architectural decision: Gio's `app.Window` subsumes every responsibility of Java's `ViewBox` (subclass of `java.awt.Frame`). Stub retained as a `c.Frame != nil` sentinel; cleaner long-term shape (`*app.Window` field directly) noted as a follow-up. | ~~🟡~~ |
| ~~`graphics/pixmap/pixmap.go:18`~~ | ~~(file)~~ | ~~`// TODO` — pipeline orientation comment, may be vestigial.~~ **Resolved 2026-05-21** (Phase 2 step 4). Vestigial — deleted. | ~~⚪~~ |
| ~~`graphics/pixmap/pixmap.go:45`~~ | ~~`NewPixMap`~~ | ~~`image.NewRGBA(...) // TODO: unused` — may be dead allocation now that `convertPixmapPixels` uses NRGBA (commit f1eca00).~~ **Resolved 2026-05-21** (Phase 2 step 4). Confirmed unread anywhere in `pkg/` or `cmd/`; `Image` field and its allocation deleted. | ~~⚪~~ |
| ~~`graphics/pixmap/pixmap.go:62–63`~~ | ~~`Draw`~~ | ~~Concurrent `ops.Ops` access concern. The `DrawMu` mutex was added later (`CLAUDE.md`) — verify these TODOs are now stale.~~ **Resolved 2026-05-21** (Phase 2 step 4). `DrawMu` renamed `OpsMu` and extended to cover the FrameEvent block (`event.Op` + drain + `e.Frame`), so every `c.Ops` access is serialized. | ~~🟡~~ |

### 4.5 Lower-priority verification TODOs

**All items resolved 2026-05-21.** Sweep across prior commits
(`4888742`, `d027e39`, `3a64d06`) plus this session's parallel-track audit
(`3853e8c`, `9cb798f`, `979ed72`, `0a84002`). Per-item verdicts retained for
posterity.

- ~~`pkg/jagex2/datastruct/lrucache.go:14` — `HashTable: make(map[int64]T, 0x400) // TODO: not limited to 0x400`~~ **Verified** in `d027e39`. Go's `make` size is a hint and the map grows freely — exactly matches Java's chained-bucket `HashTable(1024)` semantics. Marker replaced with `// Java: HashTable(1024) bucket count — Go map auto-grows beyond hint`.
- ~~`pkg/jagex2/datastruct/jstring/jstring.go:51` — return of fixed-size builder slice~~ **Verified** in `d027e39`. Fixed size 12 is correct: `toBase37` caps at 12 chars. `Builder[12-length:12-length+length]` is byte-equivalent to Java's `new String(builder, 12-var3, var3)`.
- ~~`pkg/jagex2/io/bzip2/bzip2.go` lines 15, 141, 384, 418, 515, 549, 578 — several `byte ↔ int` conversion markers~~ **Verified** in `4888742`; round-trip tests added in `3853e8c` (`bzip2_test.go` with hello-world and repeated-runs payloads to exercise BZ_GET_FAST_C). Re-audit confirmed every site: gPos stays in [0..50] so sign-extension cannot fire on shifts; tPos byte storage convention is internally consistent; redundant `& 0xFF` on `byte` (Go's `uint8`) is harmless.
- ~~`pkg/jagex2/graphics/pix32/pix32.go:37–72` — five markers, mostly applet `MediaTracker` / `PixelGrabber` and pixel type~~ **Fixed + verified** in `9cb798f`. One real bug: nil-deref on JPEG decode error — now returns `&Pix32{}` matching Java's zero-fields-on-exception. Four others (`MediaTracker`/`PixelGrabber` applet APIs replaced by `image.Decode`, `[]int` pixel type, ARGB packing) verified deviation-acceptable and annotated.
- ~~`pkg/jagex2/graphics/pix3d/pix3d.go:280` — color packing arithmetic~~ **Verified**. No marker present in file; Java had explicit parens around each shifted term and Go matches structurally.
- ~~`pkg/jagex2/graphics/model/model.go:259,264` — possibly-nil return where Java would return an empty Model~~ **Verified**. Java's `Model(int)` constructor returns the partially-constructed object on null inputs; Go's `return &m` matches. All callers cache the pointer without nil-checking, so an empty Model is correct.
- ~~`pkg/jagex2/wordenc/wordpack/wordpack.go:59,64` — slice bounds and `80` truncation~~ **Verified** + tested in `0a84002`. `TestPackTruncatesAt80` exercises 80-char and 100-char paths; rune-based truncation matches Java's UTF-16-code-unit `substring(0, 80)` for the ASCII chat inputs the encoder accepts.
- ~~`pkg/jagex2/config/component/component.go:182,224,233` — string parsing in `Unpack`~~ **Verified** in `3a64d06`. Parsed strings are ASCII sprite filenames (`"name,index"`); Java UTF-16 indexing and Go byte indexing are identical here.
- ~~`pkg/jagex2/dash3d/world/world.go:289,598` — coord swap and bit shift conversion~~ **Verified** + annotated in `979ed72`. Line 289: `addLoc` arg/index mapping cross-checked against `World.java:274-285`. Line 598: shademap `>>` on `[][]byte` is unsigned in Go but Java stores non-negative values ≤ 50, so observationally identical.
- ~~`pkg/jagex2/dash3d/entity/pathingentity.go:153` — virtual-dispatch pattern (interface vs. embedded method)~~ **Verified**. Existing `PathableEntity` interface + embedded base implements the polymorphism correctly (`NpcEntity.isVisible` and `PlayerEntity.isVisible` overrides preserved).
- ~~`pkg/jagex2/sound/wave/wave.go:138` — signed-byte bitwise AND~~ **Verified**. Java's signed `-128` and Go's unsigned `0x80` are the same byte; existing comment already explains.
- ~~`pkg/jagex2/config/loctype/loctype.go:213`, `pkg/jagex2/config/npctype/npctype.go:35` — `*string` vs `string` for optional ops~~ **Verified** in `3a64d06`. Go uses `Op[i] = ""` as the hidden-op sentinel; all readers check `!= ""`, semantically equivalent to Java's `null`. No `*string` migration needed.

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
- ~~🟡 Spin-waits in `CacheLoad`, `CacheSave`, `OpenURL` (`for X != "" { time.Sleep(...) }`)
  are functionally correct under the single-polling-goroutine model but
  unprotected by any memory-barrier primitive. A small `sync.Mutex` or
  channel-based request/response pattern would be more correct; this is
  worth doing once but doesn't block any single feature.~~ **Done 2026-05-21**
  in `031341b`. Replaced with `sync.Cond` (line-by-line closer to Java's
  `synchronized` + `notify` than channels) plus an outer `slotMu` to
  serialize callers per slot (needed because Go's `sync.Cond.Wait` drops the
  underlying mutex, unlike Java's monitor which retains it across `wait()`).
  Concurrent stress test (`signlink_stress_test.go`, 7 goroutines × 100 iters)
  added in `15d4678`. `go test -race ./...` clean.
- ~~⚪ `// TODO: synchronized` markers correspond to Java's `synchronized`
  methods. Most are redundant because the polling goroutine is the only
  writer; document this and remove the markers where applicable.~~ **No-op
  2026-05-21**: zero `// TODO: synchronized` markers exist in `signlink.go`
  (Track 4 agent grepped to verify).

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

> ⚠️ The Gio specifics below (`event.Op`, `e.Source.Event`, `pointer.Filter`,
> `key.Filter`, `OpsMu`) are **historical** — Gio was removed 2026-05-25. The
> ported *behavior* (per-frame event draining into `inputtracking`, `PollKey`
> ring buffer) is current; the backend is now the `platform` seam. See the
> Architecture note at the top of §4.

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

1. ~~Replace MIDI-sync TODOs (`client.go:656`, `client.go:1533`, and field at
   `client.go:595`) with a single-element buffered channel between the
   loader and the playback goroutine. Document this as the Go equivalent of
   the Java `synchronized` / `wait` / `notify` pattern.~~ **Done 2026-05-21**
   — see the three struck-through §4.3 entries at lines 595, 656, 1533.
   Implementation diverged from the plan: a `sync.Mutex` (`MidiSyncMu`) was
   chosen over a buffered channel because Java's `synchronized (midiSync)`
   holds the monitor across the assignment of *three* fields
   (name/CRC/length) — a channel would have coupled producer/consumer too
   tightly for that triplet handoff, while a mutex preserves Java's
   "atomic read-and-clear of the triplet" semantics line-for-line. Slow
   work (`CacheLoad`/`OpenURL`/`bzip2`) is intentionally outside the lock
   per Java. **Phase 3 complete.**

### Phase 4 — Missing utility types

1. ~~Port `jagex2/datastruct/HashTable.java` → `pkg/jagex2/datastruct/hashtable/`.
   Audit callers in `client.go` and config types; some may currently use the
   Go built-in `map` where Java used `HashTable` and need updating.~~
   **Done 2026-05-21** in `e616bf8` + `223b68c`. Faithful port + `Linkable`
   struct + 7 tests. Caller audit: only Java call site is `LruCache.java:15`;
   Go `LruCache` already uses built-in `map`, so the new package has no
   production callers yet. Future literal-port callers can import it.
2. ~~Decide whether `ViewBox` is worth a literal port or whether it can be
   replaced by a Gio-native equivalent (the Java version is AWT-derived).
   Document the decision inline.~~ **Decided 2026-05-21** in `4ef21b6`:
   Gio-native (skip literal port). Inline rationale documented in
   `viewbox.go`. Stub retained as a non-nil sentinel for one debug-command
   gate. **Phase 4 complete.**

### Phase 5 — Error reporting & cleanup

1. ~~Implement `DrawError` (`client.go:8079`).~~ **Done 2026-05-21** in
   `978e471`. State-changing parts ported; text rasterization deferred until
   `GetBaseComponent` is ported (same gap as `DrawProgressGameShell`). See
   §4.3 entry for details.
2. ~~Sweep `// TODO: try/catch` markers: convert to idiomatic Go errors at
   boundaries, delete the rest with a one-line comment per Java site.~~
   **Done 2026-05-21** across `66260c1` (`GetJagFile` retry semantics fix),
   `7292254` (`Load` recover + `ExecuteClientscript1` named-return), and
   `a237414` (vestigial deletes at `RunFlames`, `BuildScene`,
   `GetPlayerExtended2`, and `Read` opcode 4). All `// TODO: try/catch` and
   `// TODO: try/except` markers in `client.go` are gone.
3. ~~Strip applet-only TODOs (`signlink.openurl for applets`, `GetParameter`,
   `signlink.mainapp` checks) and document in code comments why they don't
   apply.~~ **Done 2026-05-21** in `fbe31bf`. Three sites in `client.go`
   (OpenURL applet-branch comment, Load goroutine launch, commented-out
   `GetParameter`). Other applet-only sites (`GetHost`, `OpenSocket`,
   `GetCodeBase`) already had clean documentation from Phase 1.
4. ~~Work through §4.5 verification TODOs in batches by package.~~ **Done
   2026-05-21**. See §4.5 — all 12 items resolved across prior commits
   (`4888742`, `d027e39`, `3a64d06`) and this session's parallel-track audit
   (`3853e8c`, `9cb798f`, `979ed72`, `0a84002`). One real bug fixed
   (`pix32.go` nil-deref on JPEG decode error); rest verified-correct with
   inline Java reference annotations.

### Phase 6 — Hardening

1. ~~Spin-wait → channel/mutex conversion in `signlink`.~~ **Done 2026-05-21**
   in `031341b` — `sync.Cond` + `slotMu`. See §4.6 for details.
2. ~~Race-detector run: `go test -race ./...`.~~ **Done 2026-05-21** — clean
   across all packages including the new `signlink_stress_test.go` (7
   goroutines × 100 iters).
3. ~~Optional: add Java-side cross-check tests for any complex algorithm
   (model rendering, bzip2, ISAAC) that already has TODOs flagged.~~
   **Partially done 2026-05-21** — `bzip2_test.go` (`3853e8c`) covers
   round-trip decompression of two real bzip2 payloads including the
   BZ_GET_FAST_C run-length path. Model rendering and ISAAC cross-checks
   not done; both can be added on demand. **Phase 6 complete.**

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
