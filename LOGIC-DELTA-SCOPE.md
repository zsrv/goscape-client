# rev-244 Logic-Delta Scope (225-clean → 244)

Scope of the **game-logic delta** to apply *after* the rename pass (see
`RENAME-MAP.md`). Produced by a filtered-diff heat map across all mapped file
pairs, then six parallel subsystem analyses of `Client-Java` `cc3781de`
(225-clean, base) vs `01f16088` (244, target), each reading hunks to separate
real behaviour from deob-lineage naming noise.

## Executive summary

The raw 225→244 diff is ~42k lines, but the **real behavioural delta is
moderate and concentrated**. The overwhelming majority is non-behavioural:
reassigned `@ObfuscatedName` keys, divergent human names, package moves, and
opcode/constant *renumbering*. Whole subsystems are naming-only (`Pix3D`,
`BZip2`, `Isaac`, `Jagfile`, `Packet`, `CollisionMap`, `Occlude`, `Envelope`,
`Tone`, `Wave`, `InputTracking`).

The substantive work falls into **five workstreams** plus UI/render polish:

1. **On-demand cache + Model loader** (the spine — touches model/map/anim/midi loading)
2. **Wire protocol: opcode renumbering, login handshake, REBUILD_NORMAL**
3. **`ModelSource` base class + scene-graph hierarchy rework**
4. **Config cache-format opcode additions** (`Component`, `NpcType`, `ObjType`, `SeqType`, `VarpType`)
5. **Audio consumer reconciliation** (`SignLink.audioLoop` + `MidiPlayer`)

## Cross-cutting hazards (read first)

- **All opcode/constant NUMBERS are renumbered between the two lineages.** The
  Go port currently hard-codes rev-225 numbers for incoming/outgoing/zone
  packets and menu actions. **Every one must be re-derived from the 244 target**
  (`io/Protocol.java` arrays + the `Client` handler switch). This is pervasive,
  not a localized change.
- **OnDemand transport divergence (KEY DECISION).** The Go port already fetches
  cache assets over **HTTP** (`-ondemand-server`, `signlink.OpenURL`,
  `clientextras.OndemandBaseURL`) with 225's `signlink.CacheLoad/CacheSave`.
  244-Java's `OnDemand` uses a **socket** protocol (byte `15` handshake on
  `port+43594`, 4-byte requests, 6-byte response headers, gzip payloads,
  `.idx`/`.dat` sector store). **Decide before starting workstream 1** whether to
  (a) port 244's socket protocol faithfully, or (b) keep the Go HTTP transport
  and adopt only 244's higher-level *API* (load by numeric file index;
  prefetch/priority; per-model `unpack(byte[])`). The answer depends on what the
  **target 244 server speaks** — verify against the Go server (`zsrv/goscape`)
  / LostCity `Server` before writing socket code that may be dead on arrival.
- **The game cannot run headless** (no display in sandbox/CI). Build/vet/test/
  golangci-lint gate every increment; behavioural verification needs host runs
  at milestones (a real 244 server connection is possible only after workstream
  1+2 land).

## Risk-tiered subsystem inventory

### Workstream 1 — On-demand cache + Model loader  [XL, critical path]
- **NEW io classes** (in 225 this logic lived inline in `client`/`signlink`):
  - `OnDemandRequest` — DTO: `archive, file, data, cycle, urgent`.
  - `OnDemandProvider` — abstract base with `requestModel(int)`.
  - `FileStream` — `.idx`/`.dat` sector store: 520-byte sectors, 8-byte sector
    header (`file:u16, part:u16, next:u24, archive:u8`), 6-byte idx entry
    (`size:u24, sector:u24`); shared `static temp[520]` → **needs a mutex in Go**.
  - `OnDemand` (Runnable worker): `unpack(versionlist, Client)` parses
    `model/anim/midi/map` version+crc tables and indices; socket protocol per the
    KEY DECISION above; `cycle()` **gzip**-decompresses payloads (vs Jagfile's
    BZip2); `validate(src,crc,version)` (trailing 2-byte version + CRC32). Rich
    API: `getMapFile, request, prefetch, prefetchMaps, prefetchPriority,
    getModelFlags, getAnimCount, hasMapLocFile, cycle, remaining, unpack, ...`.
- **`Model` loader change [H]:** bulk `unpack(Jagfile)` (15 split streams) →
  per-id `unpack(int, byte[])` of a self-contained blob with an 18-byte trailer
  header (`vtxCount g2, faceCount g2, texFaces g1, flags g1×5, 4×g2 stream lens`)
  + lazy `tryGet(int)`/`request(int)` calling `provider.requestModel(id)`.
  **Per-vertex/per-face decode logic is identical** — only the container changes.
- **`World` prefetch [M]:** new `checkLocations`, `prefetchLocations(Packet,
  OnDemand)`, `changeLocAvailable` driving loc-model readiness; depend on new
  `LocType.checkModel/checkModelAll/prefetch`.
- **`Client` wiring [H]:** `updateOnDemand()`; map/model/anim/midi load by numeric
  index; removes 225's in-band map push (`packetType==1/80`).

### Workstream 2 — Wire protocol  [L]
- **`Protocol.java` [H]:** `CLIENTPROT_LOOKUP` + `SERVERPROT_LENGTH` arrays
  reordered (same size vocabulary, new indices). Port = copy verbatim from 244.
- **Login handshake [H]:** new prefix `p1(14) + p1(loginServer)` where
  `loginServer = (toBase37(user) >> 16) & 0x1F`; read 8 sync + a `reply`; new
  reply codes **19** (→staffmodlevel 2), **20** ("Invalid loginserver"), default
  "Unexpected server response"; version byte 225→244; `rights`(bool) →
  `staffmodlevel`(int 0/1/2; reply 2→0, 18→1, 19→2).
- **REBUILD_NORMAL [H]:** re-encoded — reads `g2 zoneX, g2 zoneZ`, computes the
  region grid client-side, fetches maps via OnDemand (replaces 225's in-band
  region-CRC list + world-server map request).
- **Player/NPC info bit-streams UNCHANGED** (same getPlayerLocal/OldVis/NewVis/
  Extended + getNpcPos decomposition; extended-info mask bits identical). Still,
  do a per-handler byte-level spot-check on movement/extended reads during port.
- **Chat crowns [L]:** client-side `@cr1@`/`@cr2@` name prefixes by staffmodlevel
  (no wire change; the staffModLevel byte already existed in 225).

### Workstream 3 — ModelSource base + scene hierarchy  [L]
- **NEW `ModelSource extends DoublyLinkable`**: `vertexNormal[]`, `minY=1000`,
  virtual `getModel()` (null), concrete `draw(...)` that calls `getModel()`,
  caches `minY`, delegates to `model.draw(...)`. **Inverts** the old contract
  (`Entity.draw()` *returned* a Model; now subclasses implement `getModel()`).
- **Re-parent:** `ClientEntity extends ModelSource` (was `Entity`); `ClientObj`
  and `ClientLocAnim` move `Linkable → ModelSource`; `ClientProj`, `MapSpotAnim`
  `extends ModelSource`. `Model` now `extends ModelSource`.
- **Scene field types `Model → ModelSource`** across `Sprite`/`Wall`/`Decor`/
  `GroundDecor`/`GroundObject` (and `Location.model`+`Location.entity` collapse
  into one `Sprite.model`).
- **`World.addLoc` [H]:** builds self-animating `ClientLocAnim` nodes inline
  (passed as the `ModelSource`) instead of registering a `LocEntity` into a
  `LinkList`. `ClientLocAnim.getModel()` advances the seq against `loopCycle`.
- **`World3D` [H]:** renders via `node.draw(...)` (lazy `getModel()`); **deletes**
  `setLocModel/setWallModel/setWallModels/setWallDecorationModel/
  setGroundDecorationModel`; method renames (`addObjStack→addGroundObject`, …,
  `getWallBitset→getWallTypecode`); new getters (`getWall/getDecor/getSprite/
  getGroundDecor`); casts `(Model) x.model` guarded by `instanceof`/`vertexNormal
  != null` to skip animated sources. **Occlusion algorithm unchanged.**
- **`LocChange` = MERGE of `LocAddEntity` + `LocMergeEntity`** (fields
  `old*/new*` from LocAdd + `startTime/endTime=-1` from LocMerge's `lastCycle`).
  `LocMergeEntity` does not exist in 244 — consolidate the two Go types into one.

### Workstream 4 — Config cache-format opcodes  [S–M, independently testable]
- **`Component` [H]:** NEW `alpha` byte (`g1`) read **between `height` and
  `overlayer`** in the header — shifts all following bytes; port at the exact
  position. type-6 model fields become deferred int ids (no wire change).
- **`NpcType` [S]:** +opcodes **99** (`alwaysontop=true`), **100** (`ambient=g1b`),
  **101** (`contrast=g1b*5`), **102** (`headicon=g2`); normals now use
  `ambient+64, contrast+850`.
- **`ObjType` [S]:** +opcodes **110/111/112** (`resizex/y/z=g2`), **113**
  (`ambient=g1b`), **114** (`contrast=g1b*5`); `reset()` defaults `resize=128`.
- **`SeqType` [M]:** opcode **8** repurposed (`replaycount`→`maxloops`, same
  width); +opcodes **9** (`preanim_move`), **10** (`postanim_mode`), **11**
  (`duplicatebehavior`); per-frame delay-fallback moved out of decode into a lazy
  `getFrameDuration(frame)` (removes the AnimFrame dependency at decode time —
  watch load order); post-loop pre/post-anim defaulting (−1 → 0 or 2).
- **`VarpType` [H-easy-to-miss]:** +opcode **11** (`code11=true`); opcode **8**
  now sets **both** `code8` AND `code11`.
- **No format change (skip):** `FloType`, `IdkType`, `LocType`, `SpotAnimType`
  (rename/helper-only; `LocType` gains `checkModel/checkModelAll` used by WS1).
- **`UnkType` — do NOT port:** empty deob stub, never loaded at 244 (only
  `UnkType.types = null` in unload). Mark `// Java: UnkType — empty stub … intentionally not ported`.

### Workstream 5 — Audio consumer reconciliation  [L–XL]
- 244 ports the **previously-missing wrapper-side audio consumer** (the known Go
  gap, `feedback_porting_signlink_wrapper_gap`): `SignLink.audioLoop()` (MIDI
  fade state machine ±8 toward `midivol`; `midi`/`wave` string protocol:
  `"stop"`/`"voladjust"`/file; WAV via `javax.sound.sampled` + optional PAN) and
  **NEW `MidiPlayer`** (`javax.sound.midi` Sequencer/Synthesizer; volume rescale
  on CC7/CC39/CC121; `play(seq, loop, volume)`).
- **Reconciliation, not line-port:** the Go port already has its own audio
  backends (oto native / Web Audio prerender on wasm). Map the fade logic +
  `midi`/`wave` string protocol + volume curves onto the existing sinks. XL if
  adopting fade/volume semantics faithfully; L if only the protocol/state machine.
- **Also in `SignLink` [M]:** direct `RandomAccessFile` cache (`main_file_cache.
  dat`/`.idx0..4`, deletes .dat if >50MB); configurable `storeid` (32–34 →
  `.file_store_<id>`); removed async `cacheload/cachesave`; defaults `midivol/
  wavevol=96`, `midi/wave="none"`; `clientversion 244`, `reporterror244.cgi`.

### UI / rendering polish  [S–M, mostly independent]
- **`Pix2D` [M]:** new `fillRectTrans/drawRectTrans/hlineTrans/vlineTrans/cls`;
  `setClipping` now clamps right/bottom + sets `centerY2d`.
- **`Pix32` [S]:** new `drawRotated` (sin/cos 16.16 sprite rotate), `rgbAdjust`.
- **`PixFont` [S]:** new `strikeout` + `<str>` markup tag; `evaluateTag` returns
  −1 (was 0) on not-found — verify nothing relied on 0.
- **`GameShell` [M]:** double-click latching (`lastMouseClick*`, `mouseClickTime`)
  feeds gameplay; BUTTON3/insets/focus mostly no-ops under GLFW but the
  loop-read fields must exist. Already heavily diverged in the Go host-shell —
  map, don't line-port.
- **`ViewBox` [M]:** dynamic `getInsets()` — no-op under GLFW; keep the `insets`
  field if `GameShell` reads it.
- **`WordFilter` [M]:** ~90% naming; real deltas: `getEmulatedSize` new leetspeak
  rules (`b→i3`, `d→i)`, `g→q`), `filter()` `alphaIndex` scoring tweak, ALLOWLIST
  +`woop/woops`. Dead `match2` in `filterTld` — skip. Word *tables* are external
  cache files (may shift with the 244 data revision independently).
- **`WordPack` [S]:** `pack()` lost its `boolean flush` guard (always flushes
  trailing nibble) — verify the Go caller always passed `true`.
- **`datastruct` [S–M]:** `LruCache` `put()` arg order swapped + new `search`
  sentinel double-evict + hit/miss counters (call-site fan-out); `DoublyLinkList`
  new `cursor`/`head()/next()/size()` iterator.
- **`ClientStream` [S]:** writer thread slot `2→3` (priority); new `debug()`.
- **`MouseTracking` [S]:** NEW ~50-line polling goroutine sampling `Client.
  MouseX/Y` into 500-entry ring buffers (NOT a split of InputTracking).

## Proposed phased plan (dependency-ordered)

```
WS4 Config opcodes ──┐  (independent, low-risk warm-up; testable vs cache files)
WS3 ModelSource/scene┤─→ WS1 OnDemand + Model loader ─→ WS2 Protocol/login/REBUILD ─→ host smoke test
                     │        (needs the transport DECISION first)
WS5 Audio ───────────┘  (independent)
UI/render polish ───────  (independent; any time)
```

1. **WS4 Config opcodes** — small, self-contained, verifiable against a 244 cache
   without a server. Good first increment to validate the cache-format approach.
2. **WS3 ModelSource + scene hierarchy** — structural, no protocol; can land
   before OnDemand. Introduces the `getModel()`/`draw()` polymorphism WS1 needs.
3. **Resolve the OnDemand transport DECISION**, then **WS1** (the spine).
4. **WS2 Protocol/login/REBUILD** — REBUILD depends on WS1; opcode renumbering
   can proceed in parallel once the 244 numbers are extracted.
5. **WS5 Audio** and **UI/render polish** — independent; slot anywhere.
6. **Host smoke test** against a 244 server after WS1+WS2.

Each increment: keep faithful-port discipline + build/vet/test/golangci-lint
gate; commit small.

## Open questions to resolve during the port
- **OnDemand transport** (socket vs keep-HTTP) — see KEY DECISION; gate WS1 on it.
- `ClientProj.getModel`: the renamed 244 `Model` ctor/`scale` arg order vs 225 —
  confirm it's a compensating reorder, not a behaviour change.
- `ClientEntity` `move`/`step` bodies — gameplay-sensitive movement; body-level diff.
- `WordPack.pack` — did the Go caller always pass `flush=true`? If not, real change.
- Player/NPC extended-info bit reads — per-handler byte-level confirmation.
- 244 word/data cache files may differ from 225 independently of source.
