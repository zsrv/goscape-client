# WS2 — Wire protocol / login / REBUILD + LocChange merge + lazy model (rev-244) Design

> **STATUS: DESIGN (drafted 2026-06-03).** Scope/architecture spec for WS2, the
> last gate before a host smoke test. Follows WS1 (`WS1-MODEL-LOADER-DESIGN.md`,
> DONE) and WS3 (`WS3-MODELSOURCE-DESIGN.md`, DONE). The detailed per-increment
> step checkboxes are produced by the implementation-plan phase (writing-plans)
> after this design is reviewed.

> **For agentic workers:** REQUIRED SUB-SKILL once the plan exists: use
> superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement task-by-task with 2-stage review,
> exactly as WS1.

**Goal:** Make the Go client **connect to and render against an Engine-TS 244
server**. WS1 built the on-demand loader and left the scene-rebuild fields
(`SceneMapLandFile/LocFile`) + archive-3 dispatch waiting; WS2 supplies the 244
wire protocol, login handshake, `REBUILD_NORMAL` handler, unified loc-update
(`LocChange`) semantics, and lazy model resolution so non-preloaded models
render. After WS2, a real 244 server connection + render is possible — the first
**host smoke test** since the rev-244 work began.

**Architecture:** Faithful 1:1 port of **Client-Java 244**, cross-checked against
**Client-TS 244** (the modernized non-applet reference, the same HTTP+WS
constraint as the Go port). Six concerns, each a build-gated increment:
1. **Opcode constants** — replace scattered rev-225 magic numbers with named
   `SERVERPROT_*` / `CLIENTPROT_*` constants at 244 numbers, derived verbatim
   from 244 `Protocol.java` + Client-TS `ServerProt`/`ClientProt` (user decision:
   named constants, for auditability of a stream-framing-critical renumber).
2. **Incoming dispatch** — the `if c.PacketType == N` cascade + `ReadZonePacket`
   point at the new constants/numbers.
3. **Outgoing renumber** — `P1Isaac(N)`/`P1(N)` writes use `CLIENTPROT_*`.
4. **Login handshake** — 244 prefix + version + `staffmodlevel`.
5. **REBUILD_NORMAL (165)** — client-side region grid + on-demand map requests;
   remove 225's in-band region-CRC push (opcodes 80/237).
6. **LocChange merge (3f)** + **lazy model** (WS1 follow-ups B1/B2, folded in per
   user decision so the smoke test renders non-preloaded models).

**Tech Stack:** Go 1.26. Existing `io.Packet` (Isaac-keyed `P1Isaac`, `G1/G2/G4`,
`GSmartS`), `datastruct.LinkList`, `dash3d/model` (`TryGet`/`Request` from WS1),
`config/*` types. No new dependencies.

---

## References (read before each increment)

- **Java 244** = `Client-Java` commit `01f16088` (branch 244). Read via
  `git -C $HOME/Code/github.com/LostCityRS/Client-Java show 01f16088:src/main/java/jagex2/<path>`.
  225-clean base = `cc3781de` (bug-vs-delta classification only). **Read via
  `git show`, not the working tree** — the working trees carry local debug edits.
- **Client-TS 244** = `Client-TS` commit `1cfb57b` — *the* reference for the named
  opcode enums (`src/io/ServerProt.ts`, `src/io/ClientProt.ts`) and the modernized
  login/REBUILD adaptation. Read via
  `git -C $HOME/Code/github.com/LostCityRS/Client-TS show 1cfb57b:<path>`.
- **Engine-TS 244** = `Engine-TS` `9aadcec` — the target server (serves the game
  WebSocket + `/ondemand.zip`). The protocol the Go client must match end-to-end.
- Build/test (sandbox): `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache
  GOPATH=$HOME/go GOFLAGS=-mod=mod PATH=$HOME/go/go1.26.3/bin:$PATH go
  build/vet/test ./...`. golangci-lint: `go run
  github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 run`
  (`GOLANGCI_LINT_CACHE=/tmp/claude-1000/golangci-cache`, **dangerouslyDisableSandbox: true**).
  `gofmt -l` is a *separate* CI gate — run it on every change. Avoid standalone
  comment lines above new `Client` struct fields (triggers a mass re-tab); use
  trailing `// Java:` comments.
- Faithful 1:1 port: `// Java:` refs on renamed/non-obvious code, `int8` for
  signed bytes, match Java control flow + operator precedence. Commit
  `--no-gpg-sign`. **Trust `go build`/`go test` over IDE/gopls `<new-diagnostics>`**
  (systematically stale/false in this tree).

---

## Architectural gap (Go-225 current vs Java/Client-TS-244 target)

### A. Opcodes / framing
| Aspect | Go 225 (current) | 244 (target) |
|---|---|---|
| Incoming sizes | `io/protocol.go` `SERVERPROT_SIZES` (225 values) | `Protocol.java` `SERVERPROT_LENGTH` (244 values; re-derive verbatim) |
| Incoming dispatch | `if c.PacketType == N` cascade w/ magic numbers (`client.go:9316+`) | same shape, **244 numbers**, named `SERVERPROT_*` |
| Outgoing opcodes | scattered `P1Isaac(N)`/`P1(N)` magic numbers | named `CLIENTPROT_*` at 244 numbers |
| `CLIENTPROT_LOOKUP` | present, treated dead in 225 | **resolve**: does 244 scramble outgoing opcodes via this table, or is it still dead? (open question) |
| Zone opcodes | rev-225 numbers in `ReadZonePacket` (LOC_ADD 59, LOC_DEL 76, LOC_ANIM 42, LOC_MERGE 23, …) | 244 numbers (LOC_ADD_CHANGE 232, LOC_DEL 125, LOC_ANIM 155, …) |

### B. Login
| Aspect | Go 225 (current) | 244 (target) |
|---|---|---|
| Handshake prefix | none | `out.p1(14); out.p1(loginServer)` where `loginServer=(toBase37(user)>>16)&0x1F` (`Client.java:2602-2607`) |
| Version byte | `c.Login.P1(225)` (`client.go:6663`) | `login.p1(244)` (`Client.java:2641`) |
| Staff flag | `Rights bool` (`client.go:264`; reply 18→true, 2→false; 4 read sites: `2002,2018,5190,5320`) | `staffmodlevel int` (`Client.java:435`; reply 2→0, 18→1, **19→2**) |
| Reply codes | 2, 18 | 2, 18, **19**, **20** ("Invalid loginserver"), default "Unexpected server response" |
| Chat crowns | n/a | client-side `@cr1@`/`@cr2@` name prefixes by `staffmodlevel` (no wire change) |

### C. Scene rebuild
| Aspect | Go 225 (current) | 244 (target, `Client.java:7704-7803`) |
|---|---|---|
| Rebuild opcode | 80 (cache-save) + 237 (in-band region-CRC list + entity shift) | **165 REBUILD_NORMAL** |
| Packet body | `g2 zoneX,g2 zoneZ` + per-region `(g1 x,g1 z,g4 landCRC,g4 locCRC)` list | `g2 zoneX, g2 zoneZ` only |
| Region grid | derived from packet size | computed client-side: `x=(zone-6)/8 .. (zone+6)/8`, same for z |
| Map fetch | validate local cache vs CRC; request misses from world server | `onDemand.request(3, getMapFile(z,x,0|1))` per region; tutorial-island skip coords |
| Scene arrays | `SceneMapLandData/LocData/Index` alloc'd in opcode 237 | alloc `LandData/LocData/Index` **+ `SceneMapLandFile`/`SceneMapLocFile`** (WS1 added the latter two as nil fields read by archive-3 dispatch `client.go:8515-8532`) |
| State guard | `sceneState != 0` early-exit | `sceneState == 2` early-exit (verify state machine 1→2 + `BuildScene` trigger) |

### D. Loc-update (3f LocChange merge)
| Aspect | Go 225 (current) | 244 (target, `dash3d/LocChange.java`) |
|---|---|---|
| Types | `LocChange` (old via `Last*`) **+** `LocMergeEntity` (`LastCycle`) | single `LocChange` |
| Fields | `LocIndex/Angle/Shape` + `LastLocIndex/LastAngle/LastShape`; merge type has `LastCycle` | `oldType/oldAngle/oldShape` + `newType/newAngle/newShape` + `startTime`/`endTime` (countdown; `endTime=-1` = permanent) |
| Lists | `SpawnedLocations` + `MergedLocations` (two) | one `locChanges` LinkList |
| Drivers | inline in `ReadZonePacket`; expiry at `client.go:2868`; zone-exit restore at `:10110` | `appendLoc`/`storeLoc`/`updateLocChanges`/`clearLocChanges` (`Client.java:3431,3539,8760`); uses WS1 `World.changeLocAvailable` |

### E. Lazy model resolution (WS1 follow-ups B1/B2)
| Aspect | Go (current) | 244 (target) |
|---|---|---|
| Config getters | eager `model.NewModel1(id)` (works only for boot-preloaded/flagged) | `model.TryGet(id)` (requests + returns nil on miss) in `ObjType.getModel`, `NpcType.getSequencedModel`, `LocType`, `SpotAnimType`, `IdkType`, `Component.loadModel`; callers handle nil |
| Component type-6 | eager `*model.Model` (`Component.Model`/`ActiveModel`, built at decode) | deferred `(modelType,model)` / `(activeModelType,activeModel)` int pairs resolved lazily via `loadModel(type,id)` |

---

## Increment plan (dependency-ordered; each build/vet/test/gofmt/lint-gated + commit)

Ordering: the constant table lands first (referenced by everything); then the
incoming/outgoing rewire; then login; then REBUILD (needs incoming numbers + the
WS1 OnDemand) and LocChange (needs zone numbers); then the lazy-model work last
(rendering-facing, ripples to draw sites). A half-renumbered client cannot frame
the stream, so correctness between increments is verified by table cross-check +
per-handler byte review, **not** by build (it compiles regardless); the closing
host smoke test is the runtime gate.

| # | Increment | Scope | Java refs | Depends |
|---|---|---|---|---|
| **1** | Opcode constant tables | Create `io/serverprot.go` + `io/clientprot.go` (named consts at 244 numbers from `Protocol.java` + TS `ServerProt`/`ClientProt`); set `SERVERPROT_SIZES` to 244 lengths. Constants defined, not all wired yet. Independent cross-check (2nd agent re-derives + diffs). | `io/Protocol.java`; TS `ServerProt.ts`/`ClientProt.ts` | — |
| **2** | Incoming dispatch rewire | Repoint the `if c.PacketType==N` cascade (`client.go:9316+`) + `ReadZonePacket` to the constants/244 numbers. Per-handler byte-level spot-check (player/NPC info reads expected unchanged — confirm `getPlayerLocal/OldVis/NewVis/Extended`, `getNpcPos`, extended-mask bits). | `Client.java` `readPacket` (~7223) | 1 |
| **3** | Outgoing renumber | Replace `P1Isaac(N)`/`P1(N)` writes with `CLIENTPROT_*`; resolve `CLIENTPROT_LOOKUP` scramble (apply iff 244 does). | `Client.java` out sites; `Protocol.java` | 1 |
| **4** | Login handshake | `p1(14)+p1(loginServer)` prefix; version 244; `Rights bool`→`StaffModLevel int` (update 4 read sites); reply codes 19/20 + default msg; `@cr1@`/`@cr2@` crowns. | `Client.java:2590-2680` (login); crown sites | 1 |
| **5** | REBUILD_NORMAL (165) | New handler: `g2 zoneX/zoneZ`, region-grid loop, alloc the 5 scene arrays (incl. WS1's `SceneMapLandFile/LocFile`), `request(3, getMapFile(...))`, tutorial-island skip, entity delta-shift; set/verify scene state machine; **remove** 225 opcodes 80/237 + in-band region-CRC path. | `Client.java:7704-7803`; TS `Client.ts:6940` | 2 |
| **6** | LocChange merge (3f) | Fuse `LocAddEntity`+`LocMergeEntity`→one `LocChange` (`old*/new*`+`startTime/endTime`); single `locChanges` list; port `appendLoc`/`storeLoc`/`updateLocChanges`/`clearLocChanges`; delete `LocMergeEntity`; rewire loc zone opcodes; gate via WS1 `World.changeLocAvailable`. | `dash3d/LocChange.java`; `Client.java:3431,3539,8760` | 2 |
| **7** | B1 — `NewModel1`→`TryGet` sweep | Config getters lazy-load via `model.TryGet` (nil on miss); callers + draw sites handle nil. Breadcrumb at `objtype.go` near `NewModel1(t.Model)`. | `ObjType/NpcType/LocType/SpotAnimType/IdkType/Component` getModel | 5,6 |
| **8** | B2 — Component type-6 deferred model | type-6 model fields → `(modelType,model)`/`(activeModelType,activeModel)` int pairs + `loadModel`; rewire `Component.Model`/`ActiveModel` read sites. Breadcrumb at `component.go:62-63`. | `Component.java` type-6 + `loadModel` | 7 |
| **—** | **Milestone** | **Host smoke test** vs Engine-TS 244: connect, log in, render terrain + locs + npc/obj/player models. (Host-only; no headless.) | — | all |

Inc 2+3 may merge if review stays tractable. Inc 4 (login) is essentially
standalone — it rides a separate raw handshake stream, not the CLIENTPROT
game-opcode stream — so it may be sequenced anywhere after Inc 1 (e.g. as an
early warm-up); it is listed here after the renumber only by convention.

---

## Verification

- Each increment: `go build/vet/test ./...` + `gofmt -l` + golangci-lint, all
  green; commit small (one increment per commit, `--no-gpg-sign`).
- **Renumber is not compiler-checked.** Inc 1's table gets an **independent
  re-derivation + diff** (a second agent reads `Protocol.java`/`ServerProt.ts`
  fresh and compares to the committed constants). Inc 2/3/5/6 get per-handler
  byte-level review against the Java `git show`.
- **Highest-risk translations** (re-read Java, add `// Java:` refs): the
  `SERVERPROT_LENGTH` array contents; the `loginServer` shift/mask
  (`(toBase37>>16)&0x1F`) and `int64` width; the `Rights`→`StaffModLevel`
  comparison rewrites (`if c.Rights` → `>= 1`?); the 165 region-grid integer
  division `(zone±6)/8`; the `getMapFile(z,x,type)` param-label trap (WS1 already
  hit this); the LocChange countdown vs the old cycle-based expiry; per-handler
  extended-info bit reads.
- **Final runtime gate:** host smoke test vs Engine-TS 244. Logged outcome (what
  rendered / what didn't) recorded in the doc status + a resume note.

## Open questions (resolve in the implementation-plan phase, by reading Java/TS)

1. **`CLIENTPROT_LOOKUP`** — does 244 apply it to outgoing opcodes (scramble), or
   is it dead as in 225? Derive from `Protocol.java` + the `Client.java` out
   sites. Determines Inc 3's shape.
2. **Exact `SERVERPROT_LENGTH`** — transcribe verbatim from 244 `Protocol.java`
   (the WS2 exploration eyeballed, did not diff).
3. **Scene state machine** — confirm where 244 sets `sceneState` 1→2 and what
   triggers `BuildScene` (225 used opcode 184 at `client.go:9421`); confirm
   whether 165 allocates `sceneMapLandData/LocData` (exploration self-contradicted
   vs its own quoted Java lines 7741-7742 — re-read).
4. **244 zone-loc opcodes + `appendLoc`/`storeLoc` signatures** — pin exact
   numbers (LOC_ADD_CHANGE 232 / LOC_DEL 125 / LOC_ANIM 155 per exploration) and
   the merge semantics (`startTime`/`endTime` from the LOC_MERGE packet).
5. **Entity delta-shift in 165** — confirm 244 REBUILD_NORMAL carries the same
   NPC/player coordinate shift the 225 opcode-237 handler did, or whether it moved.

## Scope notes

- **In scope:** the six concerns above + their removals (225 opcodes 80/237
  in-band map push; `LocMergeEntity`).
- **Out of scope / deferred:** WS5 audio reconciliation; UI/render polish; the
  small WS3 cleanups (MergeNormals y-bound, ClientProj.GetModel diff, cosmetic
  244 renames) — tracked in the post-WS1 resume, independent of connect+render.
- **Faithful-port carry:** Java socket OnDemand stays not-ported (WS1 decision);
  WS2 touches only the game WebSocket/TCP opcode stream + login handshake stream.
