# rev-244 Rename / Restructure Map (225-clean → 244)

Working checklist for the **rename-first** pass of the rev-244 port. Per the
agreed strategy (see `REFERENCES.md` on `main`, and `PORTING-LESSONS.md` §2
"When deob lineages diverge"), the Go `rev-244` branch first realigns its class
**names and structure** to the Java/TS-244 vocabulary, *then* applies the
225→244 game-logic delta on top. This keeps the Go↔Java mapping 1:1 going
forward.

**This map is verified by class field-structure**, not by git's rename detection
(`git diff -M` mis-paired half of these — see the traps below) and not by name.
Each row was confirmed by comparing field declarations in
`Client-Java cc3781de` (225-clean) vs `01f16088` (244), cross-checked against
`Client-TS @1cfb57bf`.

> **Do the rename pass as its own commit(s), with zero behavioural change.**
> Get `go build ./... && go vet ./... && go test -race ./...` green *before*
> touching any game logic. Only then is the 225→244 delta diff readable.

---

## Progress

Rename-pass increments landed on `rev-244` (each build+vet+test+gofmt+golangci-lint
green, zero behavioural change vs the rev-225 baseline):

- [x] **§B graphics → dash3d package moves** — `b7700fd`. Moved as Go sub-packages
  of `dash3d/` (not flattened into one package; directory move + import rewrite).
- [x] **§C entity type renames** — `f0c75f0`. `PathingEntity→ClientEntity`, …,
  `Entity→ModelSource` (type only). Includes `New<Type>`/`Update<Type>` compounds.
- [x] **§D scene/tile type renames (trap table)** — `15640ed`. `Location→Sprite`,
  `Ground→Square`, `TileOverlay→Ground`, `TileUnderlay→QuickGround`, ordered to
  avoid the `Ground` name collision.

Decision: dash3d sub-packages (`entity`, `typ`, `model`, …) are **kept**, not
flattened into one `dash3d` package — a flatten would reintroduce import cycles
(Go's import graph, not class identity). Class names match 244; package
boundaries follow what compiles.

**Not done (intentionally out of the zero-behaviour rename pass):**
- [ ] §E `sign/signlink` → `client/sign` — optional, low value (name already
  matches conceptually). Defer or skip.
- [ ] §F new classes (`ModelSource` members, `MouseTracking`, `UnkType`,
  `OnDemand*`, `FileStream`) — these add behaviour → **logic-delta phase**.
- [ ] Field renames (`underlay→quickGround`, `locs→sprite`, model field type
  `Model→ModelSource`, `bitset→typecode`, `info→typecode2`), the
  `ModelSource`/`DoublyLinkable` hierarchy rework, and the
  `LocMergeEntity→LocChange` consolidation — all **logic-delta phase** (they
  overlap with the 225→244 field-level changes anyway). `LocMergeEntity` is
  intentionally still present.
- [ ] Helper-method/data names that reference the floor concept, not the renamed
  scene type, are intentionally kept: `DrawTileOverlay`, `DrawTileUnderlay`,
  `LevelTileOverlay*`, `LevelTileUnderlay*`.

---

## ⚠ Name-reuse traps — read first

The 244 deob **reuses names for different classes**. Mapping by filename would
silently fuse unrelated classes. The verified identities:

| Name in 244 | Is NOT the old class of that name. It is actually… |
|---|---|
| `Ground` (244) | the old **`TileOverlay`** (triangulated overlay: vertex/triangle arrays + shape) |
| `Square` (244) | the old **`Ground`** (the per-tile aggregate, `extends Linkable`) |
| `Sprite` (244) | the old **`Location`** (per-tile loc: model + grid bounds + distance/cycle) — *nothing to do with 2D sprites* |
| `QuickGround` (244) | the old **`TileUnderlay`** (4 corner colours + texture + rgb) |
| `Wall` (244) | **does** equal the old `Wall` (two faces) — git's `Wall→Sprite` pairing was wrong |

So `225 Ground → 244 Square`, while a brand-new, unrelated `244 Ground` exists
(= `225 TileOverlay`). Rename in an order that never lets two classes hold the
same name simultaneously (see Execution order).

---

## A. Unchanged — no rename (verify only)

Same name, same package, same role. No action beyond the eventual logic delta.

- `config/`: `Component, FloType, IdkType, LocType, NpcType, ObjType, SeqType, SpotAnimType, VarpType`
- `datastruct/`: `DoublyLinkList, DoublyLinkable, HashTable*, JString, LinkList, Linkable, LruCache`
- `dash3d/`: `CollisionMap, Occlude, World, World3D`
- `graphics/`: `Pix2D, Pix32, Pix3D, Pix8, PixFont, PixMap`
- `io/`: `BZip2, BZip2State, ClientStream, Isaac, Jagfile, Packet, Protocol`
- `sound/`: `Envelope, Tone, Wave`
- `wordenc/`: `WordFilter, WordPack`
- `client/`: `GameShell, InputTracking, ViewBox`

\* `HashTable` is replaced by Go `map` in the port (README.md) — n/a.

---

## B. Package move only (graphics → dash3d), name unchanged

244 moves these out of `graphics` into the flat `dash3d` package.

| 225-clean (Java) | 244 (Java) | Go source now | Go action |
|---|---|---|---|
| `graphics/AnimBase` | `dash3d/AnimBase` | `graphics/animbase/` | move pkg → `dash3d/animbase` (or `dash3d`) |
| `graphics/AnimFrame` | `dash3d/AnimFrame` | `graphics/animframe/` | move pkg |
| `graphics/Metadata` | `dash3d/Metadata` | `graphics/metadata/` | move pkg |
| `graphics/Model` | `dash3d/Model` | `graphics/model/` | move pkg (widely imported — verify cycles) |
| `graphics/VertexNormal` | `dash3d/VertexNormal` | `graphics/vertexnormal/` | move pkg |

---

## C. `dash3d/entity/*` → flattened `dash3d/*`, renamed

| 225-clean (Java) | 244 (Java) | Go source now | Notes |
|---|---|---|---|
| `entity/Entity` | `ModelSource` | `dash3d/entity/entity.go` | **base reworked**: was `extends Linkable` w/ only `draw()`; 244 `ModelSource extends DoublyLinkable` adds `vertexNormal[]`, `minY`, `getModel()`. See §F. |
| `entity/PathingEntity` | `ClientEntity` | `dash3d/entity/pathingentity.go` | `extends ModelSource`; `seqStandId→readyanim`, `seqWalkId→walkanim`, etc. |
| `entity/NpcEntity` | `ClientNpc` | `dash3d/entity/npcentity.go` | `extends ClientEntity` |
| `entity/PlayerEntity` | `ClientPlayer` | `dash3d/entity/playerentity/` | `extends ClientEntity`; `headicons→headicon`, `appearances→appearance`, `colors→colour`, `combatLevel→vislevel`, `appearanceHashcode→hash` |
| `entity/ObjStackEntity` | `ClientObj` | `dash3d/entity/objstackentity.go` | base changes `Linkable → ModelSource` |
| `entity/ProjectileEntity` | `ClientProj` | `dash3d/entity/projectileentity.go` | 244 is **less deobbed** (`field502…`); structure matches (SpotAnimType, level, src coords, offsetY, start/end cycle). git listed it "deleted" — it is a rename. |
| `entity/SpotAnimEntity` | `MapSpotAnim` | `dash3d/entity/spotanimentity.go` | exact field match |
| `entity/LocEntity` | `ClientLocAnim` | `dash3d/entity/locentity.go` | animated loc (`SeqType seq`, `seqFrame/seqCycle`, `index`); gains heightmap corners |
| `entity/LocAddEntity` | `LocChange` | `dash3d/entity/locaddentity.go` | `extends Linkable`; `locIndex/angle/shape` + `last*→old*/new*` |
| `entity/LocMergeEntity` | **(merged into `LocChange`)** | `dash3d/entity/locmergeentity.go` | 244 has no separate merge class; `LocChange` carries both the add (`old*/new*`) and the merge (`startTime` ≈ old `lastCycle`) data. **Two Go files collapse into one — verify call sites.** |

---

## D. `dash3d/type/*` → flattened `dash3d/*` (THE TRAP TABLE)

Verified by structure. Go package is `dash3d/typ/` (`type` is a Go keyword).

| 225-clean (Java) | 244 (Java) | Go source now | Identity evidence |
|---|---|---|---|
| `type/Decor` | `Decor` | `dash3d/typ/decor.go` | y/x/z + 2 angles + 1 model (same role) |
| `type/GroundDecor` | `GroundDecor` | `dash3d/typ/grounddecor.go` | y/x/z + 1 model |
| `type/GroundObject` | `GroundObject` | `dash3d/typ/groundobject.go` | y/x/z + top/bottom/middle models |
| `type/Wall` | `Wall` | `dash3d/typ/wall.go` | y/x/z + two faces (`typeA/B,modelA/B` → `angle1/2,model1/2`) |
| `type/Location` | **`Sprite`** | `dash3d/typ/location.go` | level + model + grid bounds + distance/cycle |
| `type/Ground` | **`Square`** | `dash3d/typ/ground.go` | `extends Linkable`, per-tile aggregate holding all parts |
| `type/TileOverlay` | **`Ground`** | `dash3d/typ/tileoverlay.go` | vertex/triangle arrays + shape + over/underlay colour |
| `type/TileUnderlay` | **`QuickGround`** | `dash3d/typ/tileunderlay.go` | 4 corner colours + textureId + rgb (`neColour` confirms) |

**Pervasive field renames in this group:** model-field type `Model → ModelSource`
(tile parts now hold the `ModelSource` base, not `Model`); `bitset → typecode`;
`info (byte) → typecode2 (byte)`. `Square` (was `Ground`): `underlay→quickGround`,
`overlay→ground`, `locs[]→sprite[]`, `bridge→linkedSquare`, `occludeLevel→originalLevel`.

---

## E. Monolith / signlink relocation

| 225-clean (Java) | 244 (Java) | Go status |
|---|---|---|
| `deob/client.java` | `jagex2/client/Client.java` | **Already done in Go** — port never had `deob/client`; it is `client/client.go`. No-op. |
| `sign/signlink.java` | `jagex2/client/sign/SignLink.java` | Go has `sign/signlink`. Optional move to `client/sign`; low priority, name already matches conceptually. |

---

## F. New in 244 — create (no 225-clean counterpart)

| 244 (Java) | Origin / note |
|---|---|
| `dash3d/ModelSource` | New drawable base; absorbs the old `entity/Entity` role (see §C). Introduce first — entities/tiles depend on it. |
| `client/MouseTracking` | Split out of `InputTracking` (225 had one class; 244 has both). Cross-ref Go `client/inputtracking`. |
| `client/sign/MidiPlayer` | MIDI playback. The Go port already supplies this **Go-native** (`sound/audio`, signlink consumer gap) — use the Java as reference, do **not** blind-port over the working native impl. |
| `config/UnkType` | New config/definition type. New `config/unktype` package. |
| `io/OnDemand`, `io/OnDemandProvider`, `io/OnDemandRequest` | On-demand cache streaming extracted from the 225 client monolith / signlink. Cross-ref `Client-TS io/`. |
| `io/FileStream` | New IO helper. Cross-ref `Client-TS io/Database`/`io/`. |

---

## G. Genuinely deleted in 244

| 225-clean (Java) | Disposition |
|---|---|
| `deob/class61` | Deob artifact — never ported (project policy). |
| `deob/ObfuscatedName` | Annotation-only; not ported (Go has no obf annotations). |
| `entity/Entity` | Folded into `ModelSource` (§C / §F), not a true deletion. |
| `entity/LocMergeEntity` | Folded into `LocChange` (§C). |

---

## Execution order (rename pass)

1. **Pure package moves** (§B graphics→dash3d): mechanical, no renames. Build green.
2. **Introduce `ModelSource`** (§F) by reworking `entity/Entity` → base with
   `vertexNormal[]`, `minY`, `getModel()`. Repoint subclasses' embedding.
3. **Entity flatten + rename** (§C). Collapse `LocAddEntity`+`LocMergeEntity` → `LocChange`.
4. **Type flatten + rename** (§D) — **trap order**: do the *renames* in a sequence
   that avoids name collisions, e.g. `Ground→Square` and `TileOverlay→Ground`
   must not both be named `Ground` at once (rename `Ground→Square` first, then
   `TileOverlay→Ground`). Same caution if any new `Wall`/`Ground` is added later.
5. **(Optional)** signlink → `client/sign` (§E).
6. **Stub new classes** (§F): `ModelSource` done in step 2; add `MouseTracking`,
   `UnkType`, `OnDemand*`, `FileStream` as the logic delta requires them.
7. **Gate:** `go build`, `go vet`, `go test -race`, `golangci-lint` all green
   with **no behavioural change** vs rev-225, then start the 225→244 game delta.

### Go package-placement caveat
244 flattens everything into one `jagex2/dash3d` package. Go's port uses
sub-packages (`dash3d/entity`, `dash3d/typ`, `graphics/model`, …) partly to break
**import cycles** (cf. `client/clientextras`). Adopt the 244 **names** regardless;
for **package flattening**, prefer mirroring Java but treat it as build-verified —
if collapsing `entity`/`typ`/`model` into `dash3d` introduces an import cycle,
keep the rename and leave the type in a renamed sub-package. The class *identity*
map above is the durable part; final package boundaries follow what compiles.
