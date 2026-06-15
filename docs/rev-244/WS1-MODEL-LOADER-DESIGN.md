# WS1 — On-demand cache + Model loader (rev-244) Implementation Plan

> **STATUS: DONE 2026-06-03.** All 6 increments landed on `rev-244`
> (`b2021f2`→`7ff9643`), each build/vet/test/gofmt/golangci-lint green and
> spec+quality reviewed; final holistic review confirmed clean end-to-end
> integration. Two structural follow-ups discovered and flagged in code (NOT
> done): the config-getter `NewModel1`→`TryGet` sweep and the Component type-6
> deferred-model refactor — both tie to on-demand lazy resolution, slated for
> WS2 or a dedicated increment. Host smoke test pending WS2 (login + REBUILD).

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Port the 244 on-demand loading subsystem so models, animations, MIDI,
and (per the WS1/WS2 seam) map/loc files load by numeric id from a `/ondemand.zip`
bundle, replacing 225's bulk-`Jagfile` model/anim unpack.

**Architecture:** Mirror **Client-TS 244**'s `OnDemand` *modernized* path (the
reference non-applet client, same HTTP+WS constraint as the Go port): one
`/ondemand.zip` is downloaded once, unzipped in memory, and each file served by
the key `"{archive+1}.{file}"`. The Java *socket* worker (byte-15 framing,
`FileStream` .idx/.dat store, parts, heartbeats) is **intentionally not ported**.
Per-vertex/per-face model decode is unchanged; only the *container* and the
*delivery/validation* layers change. Each increment is build/vet/test/
golangci-lint-gated and committed; host smoke test is deferred to after WS2.

**Tech Stack:** Go 1.26. Stdlib `archive/zip` + `compress/gzip` + `hash/crc32`
(IEEE poly == `java.util.zip.CRC32`). Existing `io.Packet`/`io.Jagfile`,
`datastruct.{LinkList,DoublyLinkList}`, and `client/sign/signlink` HTTP fetch.

---

## References (read before each increment)

- **Java 244** = `Client-Java` commit `01f16088` (branch 244). Read via
  `git -C $HOME/Code/github.com/LostCityRS/Client-Java show 01f16088:src/main/java/jagex2/<path>`.
  225-clean base = `cc3781de` (bug-vs-delta classification only).
- **Client-TS 244** = `Client-TS` commit `1cfb57b` — *the* reference for the
  modernized HTTP-bundle adaptation. `src/io/OnDemand.ts` is the gold standard.
  Read via `git -C $HOME/Code/github.com/LostCityRS/Client-TS show 1cfb57b:<path>`.
- **Engine-TS 244** = `Engine-TS` `9aadcec` — `src/web.ts:81-83` serves
  `/ondemand.zip` (built by `tools/pack/PackAll.ts` via `fflate.zipSync`) and
  `/build`. Confirms the bundle route + format the Go client targets.
- Build/test (sandbox): `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache
  GOPATH=$HOME/go GOFLAGS=-mod=mod PATH=$HOME/go/go1.26.3/bin:$PATH go
  build/vet/test ./...`. golangci-lint: `go run
  github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 run`
  (`GOLANGCI_LINT_CACHE=/tmp/claude-1000/golangci-cache`, **dangerouslyDisableSandbox: true**).
- Faithful 1:1 port: `// Java:` refs on renamed/non-obvious code, `int8` for
  signed bytes, match Java control flow. Commit `--no-gpg-sign`. Trust `go build`
  over IDE/gopls `<new-diagnostics>` (systematically stale in this tree).

## Architectural gap (Go-225 current vs Java/Client-TS-244 target)

| Aspect | Go 225 (current) | 244 (target) |
|---|---|---|
| Model archive | bulk `models` Jagfile (15 split streams `ob_*.dat`); `Model.Unpack(jagfile)` fills shared package streams | per-id self-contained blob; `Model.Unpack(id, data)` records offsets into a per-model `Metadata.Data`; `versionlist` archive holds version/crc/index tables |
| `Metadata` | offsets into shared streams (no `data`) | `+ Data []byte` (each model owns its blob) |
| Model decode (`NewModel1`) | reads shared `Point1..5/Face1..5/Vertex1/2/Axis`; `Axis.Pos = off*6` | reads local packets from `info.Data`; `axis.pos = off` (no `*6`) |
| AnimFrame | `Unpack(jagfile)` reads `frame_*.dat`; base looked up by per-frame id from `animbase.Instances` | `Unpack(data)` per-blob (8-byte trailer); **`AnimBase` embedded at blob tail**, built inline; no per-frame base-id read |
| AnimBase | `Unpack(jagfile)` reads `base_*.dat` | constructor `NewAnimBase(buf *Packet)` only (no archive) |
| Delivery | everything fetched as whole named archives at boot; maps via in-band world-server push (opcodes 132/220/237) | `/ondemand.zip` downloaded once; per-id files dispatched by `updateOnDemand()`; maps requested by id via `OnDemand.request(3, …)` |
| OnDemand | none | full state machine (`requests/queue/missing/pending/completed/prefetches`, `request/cycle/prefetch/prefetchPriority`, modernized `read()` from the zip) |
| `LocType` | `Models/Shapes` fields only | `+ CheckModel/CheckModelAll/Prefetch` (drive loc-model readiness) |
| `World` | `LoadLocations/AddLoc/Build` | `+ CheckLocations/PrefetchLocations` |

## Transport decision (resolved)

`/ondemand.zip` bundle, mirroring Client-TS `modernized=true`. **Not** per-file
HTTP (Engine-TS exposes no per-file route) and **not** WS state-2 (only needed
for a server that streams per-file; Engine-TS serves the bundle). The Java
socket subsystem (`OnDemand.run/handleQueue→send/read` socket branch,
`FileStream`, `OnDemandRequest` wire framing) is **not ported** — mark sites
`// Java: … socket transport — intentionally not ported (see WS1 doc)`.

Bundle contract (from Client-TS): key `"{archive+1}.{file}"` (model 5 → `"1.5"`,
anim 3 → `"2.3"`, midi 0 → `"3.0"`, map file 100 → `"4.100"`); each entry is
`[gzip payload][2-byte big-endian version trailer]`. `validate()` CRCs
`src[0:len-2]` and reads the version from the last 2 bytes; `cycle()` strips the
trailer and gunzips before dispatch.

## Package / file structure

- **Create** `pkg/jagex2/io/ondemand/ondemand.go` — `OnDemand` struct (mirrors
  Client-TS modernized path), `OnDemandRequest`, the versionlist parse, getters,
  request/cycle/prefetch state machine, modernized `read()` (download+unzip+key).
  Depends only on `io`, `datastruct`, stdlib (`archive/zip`,`compress/gzip`,
  `hash/crc32`). Network fetch + persistent cache injected via interfaces (no
  import of `client/*` → no cycle).
- **Create** `pkg/jagex2/io/ondemand/ondemand_test.go` — pure-logic tests.
- **Modify** `pkg/jagex2/io/packet.go` — (none required; use `hash/crc32` in
  ondemand. The dead `CRCTable` stays as-is.)
- **Modify** `pkg/jagex2/io/ondemandprovider.go` (**create**) — `OnDemandProvider`
  interface `{ RequestModel(id int) }` so `model` can hold a provider without a
  cycle. (Java: `io/OnDemandProvider`.)
- **Modify** `pkg/jagex2/datastruct/doublylinklist.go` — add `cursor` field +
  `Head()/Next()/Size()`.
- **Modify** `pkg/jagex2/dash3d/metadata/metadata.go` — add `Data []byte`.
- **Modify** `pkg/jagex2/dash3d/model/model.go` — `Init/TryGet/Request/Unload(id)`,
  `Provider`, `Loaded`; `Unpack(id,data)` per-id; `NewModel1` from `info.Data`;
  drop shared stream vars.
- **Modify** `pkg/jagex2/dash3d/animframe/animframe.go` — `Init(capacity)`,
  `Unpack(data)`; `pkg/jagex2/dash3d/animbase/animbase.go` — `NewAnimBase(buf)`.
- **Modify** `pkg/jagex2/config/loctype/loctype.go` — `CheckModel/CheckModelAll/
  Prefetch`.
- **Modify** `pkg/jagex2/dash3d/world/world.go` — `CheckLocations/PrefetchLocations`.
- **Modify** `pkg/jagex2/client/client.go` — `Load()` boot rewire (versionlist +
  OnDemand init + request loops), `UpdateOnDemand()`, main-loop hook, `SaveMidi`
  archive-2 path.

## Dependency-ordered increments (each build/vet/test/lint-gated + commit)

Ordering keeps every commit green: the new `ondemand` package and primitives
land first (imported by nothing), then a single **cutover** increment swaps the
loader format and rewires `Load()` (the only point where the old `Model.Unpack`
call site must change), then maps/World, then carry-forwards.

---

### Inc 0: `DoublyLinkList` iteration primitives

**Files:**
- Modify: `pkg/jagex2/datastruct/doublylinklist.go`
- Test: `pkg/jagex2/datastruct/doublylinklist_test.go` (create)

Java `DoublyLinkList` (`datastruct/DoublyLinkList.java`) has `cursor` +
`head()/next()/size()`; Go's has only `Push/Pop`. `OnDemand.requests` needs all
three.

- [ ] **Step 1 — add cursor + methods.** In `doublylinklist.go`, add `cursor
  *DoublyLinkable[T]` to the struct and:

```go
// Head returns the first node and seeds the cursor for Next, mirroring Java
// DoublyLinkList.head() (datastruct/DoublyLinkList.java).
func (l *DoublyLinkList[T]) Head() *DoublyLinkable[T] {
	n := l.head.next2
	if n == l.head {
		l.cursor = nil
		return nil
	}
	l.cursor = n.next2
	return n
}

// Next advances the cursor seeded by Head. Java DoublyLinkList.next().
func (l *DoublyLinkList[T]) Next() *DoublyLinkable[T] {
	n := l.cursor
	if n == l.head {
		l.cursor = nil
		return nil
	}
	l.cursor = n.next2
	return n
}

// Size counts live nodes. Java DoublyLinkList.size().
func (l *DoublyLinkList[T]) Size() int {
	count := 0
	for n := l.head.next2; n != l.head; n = n.next2 {
		count++
	}
	return count
}
```

- [ ] **Step 2 — test.** `doublylinklist_test.go`: push 3 nodes, assert
  `Size()==3`, iterate `Head()`/`Next()` collects them in FIFO order, `Pop()`
  then `Size()==2`.
- [ ] **Step 3 — gate + commit.** `go build/vet/test ./...` + lint green.
  `git commit --no-gpg-sign -m "feat(rev-244): DoublyLinkList Head/Next/Size (WS1)"`

---

### Inc 1: `ondemand` package — types, validate, versionlist parse, getters

**Files:**
- Create: `pkg/jagex2/io/ondemandprovider.go`
- Create: `pkg/jagex2/io/ondemand/ondemand.go`
- Test: `pkg/jagex2/io/ondemand/ondemand_test.go`

Java refs: `io/OnDemandProvider.java`, `io/OnDemandRequest.java`,
`io/OnDemand.java:135-281` (`unpack`,`getFileCount`,`getAnimCount`,`getMapFile`,
`prefetchMaps`,`hasMapLocFile`,`getModelFlags`,`shouldPrefetchMidi`),
`io/OnDemand.java:775-789` (`validate`). Cross-check Client-TS `OnDemand.ts`
constructor + getters + `validate`.

- [ ] **Step 1 — provider interface.** `io/ondemandprovider.go`:

```go
package io

// OnDemandProvider is the model loader's hook back to the cache subsystem.
// Java: jagex2.io.OnDemandProvider (a base class with a single requestModel).
type OnDemandProvider interface {
	RequestModel(id int)
}
```

- [ ] **Step 2 — request DTO + struct.** In `ondemand.go`, `OnDemandRequest`
  wraps into the datastruct lists. Java `OnDemandRequest extends DoublyLinkable
  extends Linkable` ⇒ one `*datastruct.DoublyLinkable[*OnDemandRequest]` per
  request gives it BOTH link-field pairs, so it can sit in a `LinkList` queue
  (via the embedded `*Linkable`) **and** the `requests` `DoublyLinkList`
  simultaneously — exactly as Java does.

```go
type OnDemandRequest struct {
	Archive int
	File    int
	Data    []byte
	Cycle   int
	Urgent  bool // Java default: true
	node    *datastruct.DoublyLinkable[*OnDemandRequest]
}
```

  Helper to build a node whose `.Value` points back at the request:
  `newRequest()` sets `Urgent=true`, allocates `node = datastruct.NewDoublyLinkable[*OnDemandRequest](r)`, returns `r`.
  Lists: `requests *datastruct.DoublyLinkList[*OnDemandRequest]` (push `r.node`,
  iterate via `Head/Next` reading `.Value`); `queue/missing/pending/completed/
  prefetches *datastruct.LinkList[*OnDemandRequest]` (push `r.node.Linkable`,
  pop returns `*Linkable` whose `.Value` is the request). Java `unlink2()` ⇒
  `r.node.Uncache()`; Java `unlink()` ⇒ `r.node.Unlink()`.

- [ ] **Step 3 — struct fields + seams.** Port the Client-TS field set
  (`versions [4][]int`, `crcs [4][]int`, `priorities [4][]byte`, `topPriority`,
  `models []byte`, `mapIndex/mapLand/mapLoc/mapMembers/animIndex/midiIndex
  []int`, the 6 lists, `zip map[string][]byte`, `topPriority`,
  `loadedPrefetchFiles/totalPrefetchFiles`, `message`). Add injected seams (no
  `client` import):

```go
// Downloader fetches an absolute path from the on-demand server (Client wires
// signlink.OpenURL). Cache is the per-file persistent store (Client wires the
// storage seam); a nil Cache means in-memory-only (bundle covers everything).
type Downloader interface{ Get(path string) ([]byte, error) }
type Cache interface {
	Read(archive, file int) []byte
	Write(archive, file int, data []byte)
}
```

  `OnDemand` holds `dl Downloader`, `cache Cache` (may be nil), and an
  `ingame func() bool` (Client sets it; replaces Java `app.ingame`). The Java
  `app.fileStreams[0] != null` cache gate ⇒ `od.cache != nil`.

- [ ] **Step 4 — `Unpack(versionlist Archive)`.** Faithful port of
  `OnDemand.java:135-216` (the loop reads `model/anim/midi/map_version` g2,
  `*_crc` g4, `model_index` bytes, `map_index` 7-byte records (g2 idx, g2 land,
  g2 loc, g1 members), `anim_index` g2, `midi_index` g1). **Drop** the trailing
  `app.startThread(this,2)` (socket). **Test seam:** instead of `*io.Jagfile`
  the param is a consumer-defined interface `type Archive interface { Read(name
  string, dst []byte) []byte }` — `*io.Jagfile` satisfies it structurally
  (unchanged), and tests pass a fake map-backed reader. (A `Jagfile`'s members
  are bzip2-compressed and the port has no bzip2 *encoder*, so a real synthetic
  Jagfile can't be built in a unit test; the interface is the minimal faithful
  adaptation — comment it `// Java: OnDemand.unpack(Jagfile, Client); the Jagfile
  is taken as a reader interface so the parse is unit-testable`.)

- [ ] **Step 5 — `New(versionlist, dl, cache, ingame)`** constructor: calls
  `Unpack(versionlist)`, stores seams, `running=true`.

- [ ] **Step 6 — getters.** Port verbatim (`OnDemand.java:223-281`):
  `GetFileCount(archive)`, `GetAnimCount()`, `GetMapFile(z,x,type)`
  **(⚠ Java body computes `map=(x<<8)+z` with `x`=2nd param; port the signature
  `GetMapFile(z, x, type)` and the `(x<<8)+z` formula verbatim — callers pass
  values, not labels)**, `HasMapLocFile(file)`, `GetModelFlags(id)`,
  `ShouldPrefetchMidi(id)`.

- [ ] **Step 7 — `Validate(src []byte, expectedCrc, expectedVersion int) bool`**
  (`OnDemand.java:775-789`):

```go
func Validate(src []byte, expectedCrc, expectedVersion int) bool {
	if src == nil || len(src) < 2 {
		return false
	}
	tp := len(src) - 2
	version := (int(src[tp])&0xFF)<<8 + int(src[tp+1])&0xFF
	crc := int(int32(crc32.ChecksumIEEE(src[:tp]))) // Java (int) CRC32.getValue()
	return expectedVersion == version && expectedCrc == crc
}
```

  Note the `int(int32(...))` to match Java's `(int)` narrowing of the `long` CRC
  (the stored `*_crc` table is read with `g4()` → signed int32). Confirm the
  versionlist CRC sign against a real `versionlist`/`/ondemand.zip` during the
  host check.

- [ ] **Step 8 — tests** (`ondemand_test.go`, pure):
  - `Validate`: build `payload + crc32(payload) + 2-byte version`; assert true;
    flip a version/crc byte → false; `len<2` → false.
  - `Unpack`: a fake `Archive` whose `Read(name,_)` returns in-memory member
    bytes built with `io.Packet` writers (`P2/P4/P1`) for tiny
    `*_version/*_crc/model_index/map_index/anim_index/midi_index` members;
    assert `GetFileCount`, `GetMapFile`, `GetModelFlags`, `ShouldPrefetchMidi`
    return the encoded values.
- [ ] **Step 9 — gate + commit.** Green. `git commit --no-gpg-sign -m
  "feat(rev-244): OnDemand types, versionlist parse, validate (WS1)"`

---

### Inc 2: `ondemand` state machine + modernized bundle `read()`

**Files:**
- Modify: `pkg/jagex2/io/ondemand/ondemand.go`
- Test: `pkg/jagex2/io/ondemand/ondemand_test.go`

Java refs: `OnDemand.java:282-413` (`requestModel/request/remaining/cycle/
prefetchPriority/clearPrefetches/prefetch`). Client-TS `OnDemand.ts`
`run/handleQueue/handlePending/handleExtras/read (modernized)/downloadZip`.

- [ ] **Step 1 — request/remaining.** `RequestModel(id)` → `Request(0,id)`.
  `Request(archive,file)` (`OnDemand.java:287-314`): bounds + `versions[a][f]==0`
  guard; dedup-scan `requests` via `Head/Next`; else build request
  (`Urgent=true`), `queue.AddTail(r.node.Linkable)`, `requests.Push(r.node)`.
  `Remaining()` → `requests.Size()`.
- [ ] **Step 2 — `Cycle()`** (Java `cycle()` / Client-TS `loop()`): pop
  `completed`; `r.node.Uncache()` (unlink from `requests`); if `Data==nil` return
  r; else strip trailer + gunzip: `gz, _ := gzip.NewReader(bytes.NewReader(
  r.Data[:len(r.Data)-2])); r.Data, _ = io.ReadAll(gz)`. (Client-TS:
  `gunzipSync(req.data.slice(0, len-2))`. Java reads into a 65000 scratch buffer;
  the gunzip-to-fresh-slice form is equivalent and is what Client-TS does.)
- [ ] **Step 3 — prefetch.** `PrefetchPriority(archive,file,priority byte)`
  (Java:371-389): `cache==nil || versions==0` guard; `Validate(cache.Read(
  archive+1,file), …)` → already-have skip; else set `priorities`, bump
  `topPriority`, `totalPrefetchFiles++`. `ClearPrefetches()`; `Prefetch(archive,
  file)` (Java:398-413): guards then `prefetches.AddTail`.
- [ ] **Step 4 — `Run()`** (Client-TS `run()`): the 100-iter
  `handleQueue/handlePending/handleExtras/read` pump + the pending-cycle resend
  loop. **Omit** the socket-only tails (`stream.close`, heartbeat write) — guard
  with `od.cache`/modernized constant. Port `handleQueue` (validate against
  `cache`, push to `completed`/`missing`), `handlePending` (drain `missing`→
  `pending`, `send`), `handleExtras` (priority drain). `send()` modernized ⇒
  no-op (handled by `read`).
- [ ] **Step 5 — modernized `read()`** (Client-TS `read()` modernized branch):
  take head of `pending`; `downloadZip()` (lazy: `od.dl.Get("/ondemand.zip")`
  once, `archive/zip` into `od.zip` keyed `"{a+1}.{f}"`); set `current.Data =
  zip[key]`; nil → `unlink`; else `cache?.Write(a+1,f,data)`; the **archive-3 →
  93 promotion** for non-urgent loc files (`if !urgent && archive==3 { urgent=
  true; archive=93 }`); push `completed` if urgent else `unlink`.
  `downloadZip()` uses `archive/zip.NewReader(bytes.NewReader(buf), int64(len))`
  and reads every entry into `od.zip`.
- [ ] **Step 6 — tests:** `Request` dedup (same id twice → `Remaining()==1`);
  `Cycle` gunzip+trailer-strip on a synthetic `gzip(payload)+2-byte`; bundle key
  lookup via a fake `Downloader` returning an in-memory zip (build with
  `archive/zip.Writer`, one entry `"1.5"`), drive `Request(0,5)`→`Run()`→
  `Cycle()`, assert decoded `payload`.
- [ ] **Step 7 — gate + commit.** Green. `git commit --no-gpg-sign -m
  "feat(rev-244): OnDemand request/cycle/prefetch + /ondemand.zip read (WS1)"`

---

### Inc 3 (CUTOVER): Model + AnimFrame/AnimBase per-id format; `Load()` rewire (archives 0/1/2)

This is the one increment where the old `Model.Unpack(jagfile)` call site must
change; it is green only as a whole. Sub-step it, then build once at the end.

**Files:**
- Modify: `pkg/jagex2/dash3d/metadata/metadata.go`
- Modify: `pkg/jagex2/dash3d/model/model.go`
- Modify: `pkg/jagex2/dash3d/animbase/animbase.go`
- Modify: `pkg/jagex2/dash3d/animframe/animframe.go`
- Modify: `pkg/jagex2/client/client.go`
- Test: `pkg/jagex2/dash3d/model/model_test.go`, `…/animframe/animframe_test.go`

- [ ] **Step 1 — `Metadata.Data`.** Add `Data []byte` to `metadata.Metadata`
  (Java `Metadata.data`, `dash3d/Metadata.java`).
- [ ] **Step 2 — Model statics.** In `model.go` add (Java `Model.java:14,58-61,
  256-260,352,357-385`):
  `Loaded int`; `Provider io.OnDemandProvider`;
  `func Init(count int, provider io.OnDemandProvider) { Metadata = make([]*metadata.Metadata, count); Provider = provider }`;
  `func TryGet(id int) *Model` (meta nil→nil; `meta[id]`==nil→`Provider.RequestModel(id)`, nil; else `NewModel1(id)`);
  `func Request(id int) bool` (meta nil→false; `meta[id]`==nil→`Provider.RequestModel(id)`,false; else true);
  `func UnloadOne(id int) { Metadata[id] = nil }` (Java `unload(int)`).
- [ ] **Step 3 — `Unpack(id int, data []byte)`** replacing
  `Unpack(*io.Jagfile)`. Faithful port of `Model.java:262-351`: `data==nil` →
  empty Metadata (counts 0). Else `buf.Pos = len(data)-18`; read the 18-byte
  trailer (`vertexCount g2, faceCount g2, texturedFaceCount g1, hasInfo g1,
  priority g1, hasAlpha g1, hasFaceLabels g1, hasVertexLabels g1, dataLengthX g2,
  dataLengthY g2, dataLengthZ g2, dataLengthFaceOrientations g2`); store
  `info.Data = data`; walk `pos` to set every `*Offset` exactly as Java
  (priority==255 sentinel ⇒ `FacePrioritiesOffset = -priority-1`; absent labels/
  info/alpha ⇒ `-1`). **Delete** the shared package stream vars `Head, Face1..5,
  Point1..5, Vertex1, Vertex2, Axis` and their nil-clears in `Unload()`/`Reset()`
  (they were the 225 split-stream buffers; 244 uses local packets from
  `info.Data`).
- [ ] **Step 4 — `NewModel1(id)` from `info.Data`** (Java `Model.java:389-502`).
  Keep the existing vertex/face decode **verbatim**, but: build all packets as
  `io.NewPacket(info.Data)` locals with `.Pos = info.<Offset>` (Point1..5,
  Face1..5, Vertex1, Vertex2, Axis), and set **`axis.Pos = info.FaceTextureAxisOffset`
  (no `* 6`)** — 244 stores the byte offset directly (225 stored a textured-face
  count and multiplied at use). Add `Loaded++` at entry (Java
  `Model(int):loaded++`). Drop the `Metadata==nil`/`meta[id]==nil` `fmt.Printf`
  guard? Java keeps it (`Error model:%d not found!`) — keep it.
- [ ] **Step 5 — `AnimBase` constructor** (Java `AnimBase.java`). Replace
  `Unpack(*io.Jagfile)` with `NewAnimBase(buf *io.Packet) *AnimBase`: `size=g1`,
  `types[size]` each g1, `labels[size][count]` each g1. Remove the package-level
  `Instances` + `Unpack` (244 has no anim-base archive). (`Length` field == Java
  `size`.)
- [ ] **Step 6 — `AnimFrame.Init/Unpack(data)`** (Java `AnimFrame.java:34-…`).
  `Init(capacity int) { Instances = make([]*AnimFrame, capacity+1) }`. Replace
  `Unpack(*io.Jagfile)` with `Unpack(data []byte)`: 8-byte trailer
  (`headLength/tran1Length/tran2Length/delLength` g2); local `head/tran1/tran2/
  del` packets (`head.Pos=0`, `pos += headLength+2`, then tran1/tran2/del);
  **`base := animbase.NewAnimBase(baseBuf)`** where `baseBuf.Pos = pos` after del;
  then the `total = head.g2()` frame loop **without** the 225 per-frame base-id
  `g2` read — every frame shares the single embedded `base`. Keep the
  group/flags/`gsmart` body verbatim (Go field names `Groups/X/Y/Z/Length` ==
  Java `ti/tx/ty/tz/size`).
- [ ] **Step 7 — `Client.Load()` rewire.** In `client.go` (~5678, 5859-5862):
  - Replace `jagModels := GetJagFile("3d graphics", JagChecksum[5], "models", 40)`
    with the versionlist fetch at the 244 position:
    `jagVersionList := c.GetJagFile("update list", c.JagChecksum[5], "versionlist", 60)`.
  - **Delete** `model.Unpack(jagModels); animbase.Unpack(jagModels);
    animframe.Unpack(jagModels)`.
  - After config unpack (so types exist), build OnDemand + init + boot loops,
    porting `Client.java:1594-1660` (archives 0/1/2 only here; maps = Inc 4):
    `c.OnDemand = ondemand.New(jagVersionList, dl, cache, func() bool { return c.Ingame })`
    where `dl` wraps `c.OpenURL` and `cache` is nil for now (bundle-only);
    `animframe.Init(c.OnDemand.GetAnimCount())`;
    `model.Init(c.OnDemand.GetFileCount(0), c.OnDemand)`;
    initial MIDI request (`!LowMemory`) + the **anim** request-all loop + the
    **flagged-model** request loop, each pumping `c.UpdateOnDemand()` until
    `Remaining()==0` (Go: call `c.OnDemand.Run()` then `c.UpdateOnDemand()` in the
    loop; drop `Thread.sleep`). Then the background `prefetchPriority` model loop
    (Java:1698-1726) + `shouldPrefetchMidi` loop (1730-1736). `prefetchMaps` = Inc 4.
- [ ] **Step 8 — `UpdateOnDemand()` (archives 0/1/2)** porting
  `Client.java:2425-2447`: loop `req := c.OnDemand.Cycle()`; `nil`→return;
  `archive==0` → `model.Unpack(req.File, req.Data)` + the `getModelFlags & 0x62`
  redraw-sidebar/chatback bits; `archive==1 && Data!=nil` → `animframe.Unpack(
  req.Data)`; `archive==2 && midiSong==File && Data!=nil` → `c.SaveMidi(
  c.MidiFading, req.Data)`. (Archives 3/93 added in Inc 4.) Wire `SaveMidi` to
  the existing Go MIDI sink if present, else a documented `// WS5` deferral
  (`SignLink.midisave` analog) — do **not** pull audio playback forward.
- [ ] **Step 9 — main-loop hook.** Call `c.OnDemand.Run()` + `c.UpdateOnDemand()`
  once per frame in the game-loop update (Java calls `updateOnDemand()` at
  `Client.java:1997`). Find the Go per-frame update (the `RunShell`/`Update`
  equivalent) and add the calls guarded by `c.OnDemand != nil`.
- [ ] **Step 10 — tests.**
  - `model_test.go`: synthesize a minimal model blob (≥1 vertex, 1 face, trailer)
    → `Unpack(0, blob)` → `NewModel1(0)`; assert `VertexCount/FaceCount` and a
    known vertex/face value. (Adapt the existing `model_test.go` which currently
    drives the bulk `Unpack`.)
  - `animframe_test.go`: synthesize a one-frame anim blob with embedded base →
    `Init(1); Unpack(blob)`; assert `Instances[id].Length` and a transform value.
- [ ] **Step 11 — gate + commit.** `go build/vet/test ./...` + lint green.
  `git commit --no-gpg-sign -m "feat(rev-244): per-id Model/AnimFrame loader +
  OnDemand boot cutover (WS1)"`

---

### Inc 4: Maps + World/LocType prefetch (archives 3/93)

**Files:**
- Modify: `pkg/jagex2/config/loctype/loctype.go`
- Modify: `pkg/jagex2/dash3d/world/world.go`
- Modify: `pkg/jagex2/client/client.go`
- Test: `pkg/jagex2/config/loctype/loctype_test.go`, `…/dash3d/world/world_test.go`

Java refs: `LocType.java:336-384`, `World.java` `checkLocations/
prefetchLocations`, `Client.java:1662-1690` (boot maps), `:2448-2469`
(updateOnDemand archive 3/93).

- [ ] **Step 1 — `LocType` methods** (verbatim port): `CheckModel(shape) bool`,
  `CheckModelAll() bool` (`ready &= model.Request(m & 0xFFFF)` over `Models`),
  `Prefetch(od *ondemand.OnDemand)` (`od.Prefetch(0, m & 0xFFFF)`). ⚠ `loctype`
  importing `ondemand` + `model`: confirm no cycle (`ondemand` does not import
  `loctype`; `model` does not import `loctype`). If a cycle appears, take the
  provider as the `io.OnDemandProvider`-style interface instead.
- [ ] **Step 2 — `World` funcs** (verbatim port of `checkLocations(xOffset,
  zOffset, src)` and `prefetchLocations(buf, od)`; both walk `gsmarts` delta loc
  ids; `checkLocations` culls to `0<stx,stz<103` and `shape!=22||…` then
  `loc.CheckModelAll()`; `prefetchLocations` calls `loc.Prefetch(od)`). Use
  `io.Packet.GSmartS`.
- [ ] **Step 3 — boot map requests** (Java:1662-1690), gated `if c.OnDemand has
  cache`/per Client-TS modernized gate: the 12 `Request(3, GetMapFile(z,x,t))`
  calls (tutorial-island coords — port the literal args verbatim) + pump loop,
  then `c.OnDemand.PrefetchMaps(MembersWorld)`.
- [ ] **Step 4 — `UpdateOnDemand` archives 3/93** (Java:2448-2469): `archive==3
  && sceneState==1` → match `req.File` against `SceneMapLandFile[i]`/
  `SceneMapLocFile[i]`, store `req.Data` (nil ⇒ set file id -1); `archive==93 &&
  HasMapLocFile(File)` → `world.PrefetchLocations(io.NewPacket(req.Data),
  c.OnDemand)`. (The REBUILD_NORMAL opcode handler that *sets* `sceneState=1` and
  fills `SceneMapLandFile/LocFile` stays **WS2** — add a `// WS2:` note where it
  will hook.)
- [ ] **Step 5 — tests:** `LocType.CheckModelAll` with `Models=[-1, knownId]`
  (no provider set ⇒ Request returns false for the missing id; assert false);
  `World.CheckLocations`/`PrefetchLocations` over a synthetic delta-encoded loc
  buffer (assert it consumes to the terminating 0 without panic and calls
  prefetch the expected number of times via a fake recorder).
- [ ] **Step 6 — gate + commit.** Green. `git commit --no-gpg-sign -m
  "feat(rev-244): map/loc on-demand + World/LocType prefetch (WS1)"`

---

### Inc 5 (carry-forwards): model-build wiring of WS4-decoded fields

**Files:** `pkg/jagex2/config/objtype/objtype.go`,
`pkg/jagex2/config/npctype/npctype.go`, `pkg/jagex2/config/component/component.go`,
`pkg/jagex2/config/seqtype/seqtype.go` (verify only).

Resolve the TODOs left by WS4 (now that models build): `objtype.go:269-272` /
`npctype.go:178-179` — feed the decoded `Ambient`/`Contrast`/`resize` fields into
the model-build `CalculateNormals(Ambient+64, Contrast+768|+850, …)` and
resize/scale (the literal `CalculateNormals(64, 768|850, …)` calls at
`objtype.go:337`/`npctype.go:211` currently ignore the per-type fields);
`component.go` type-6 deferred model ids; confirm `seqtype.GetFrameDuration` is
used at the WS3 anim read-sites. **Diff each against 244 source** before
changing; this is small but easy to get wrong.

- [ ] **Step 1** — read 244 `ObjType.getInterfaceModel/getWornModel`,
  `NpcType.getSequencedModel`, `Component` type-6 build, confirm exact
  `+64/+768/+850` and resize usage; wire the decoded fields.
- [ ] **Step 2** — gate + commit `feat(rev-244): wire WS4 ambient/contrast/resize
  into model build (WS1 carry-forwards)`.

---

## Risk / verification

- **No runtime gate in sandbox** (no display, no 244 server). Every increment is
  build/vet/test/golangci-lint-green + committed. The real check (models render)
  needs WS2 (login + REBUILD + world-server map delivery) — host smoke test
  deferred to post-WS2, exactly as WS3.
- **Highest-risk translations** (re-read Java, add `// Java:` refs): the
  `NewModel1` axis offset (`*6` removal), the 18-byte trailer field order, the
  `cycle()` trailer-strip+gunzip, `Validate` CRC sign (`int(int32(...))`),
  `GetMapFile` param-label trap, the AnimFrame embedded-base (no per-frame
  base-id read), and the list double-membership idiom.
- **Cycle/import risks:** `ondemand` must not import `client`/`model`/`loctype`
  (seams + `io.OnDemandProvider` keep it leaf-ward). `loctype`→`ondemand`/`model`
  edge checked in Inc 4 Step 1.
- **Deferred to WS2** (note in code, do not port here): REBUILD_NORMAL (244
  opcode 165) handler + zone-grid region request; removal of 225 in-band map push
  (opcodes 132/220/237); all opcode renumbering; login staffmodlevel. **Deferred
  to WS5:** real MIDI/wave sink for the archive-2 bytes (only the dispatch is
  ported here).

## Self-review notes

- **Spec coverage** vs `LOGIC-DELTA-SCOPE.md` WS1: versionlist+validate ✔(Inc1),
  Model loader ✔(Inc3), World prefetch ✔(Inc4), Client wiring
  ✔(Inc3/4); socket subsystem explicitly not ported ✔. Carry-forwards ✔(Inc5).
- **Type consistency:** `OnDemand` methods named identically across Inc1/2/3/4
  (`Request/RequestModel/Cycle/Run/Prefetch/PrefetchPriority/PrefetchMaps/
  GetFileCount/GetAnimCount/GetMapFile/GetModelFlags/ShouldPrefetchMidi/
  HasMapLocFile/Remaining`); model statics `Init/Unpack/NewModel1/TryGet/Request/
  UnloadOne/Loaded/Provider`; `metadata.Data`; `animframe.Init/Unpack`;
  `animbase.NewAnimBase`.
- **Open item to confirm at impl time:** exact Go per-frame update site for the
  main-loop `Run()`+`UpdateOnDemand()` hook (Inc3 Step9); the existing Go MIDI
  sink name for `SaveMidi` (Inc3 Step8); whether `loctype`→`ondemand` is
  cycle-free (Inc4 Step1).
