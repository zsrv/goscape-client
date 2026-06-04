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

---

# rev-254 Rename Map (245.2 → 254)

Verified by class-level `@ObfuscatedName` key pairing between `176a85f`
(245.2) and `2e62978` (254), 2026-06-04 — NOT by filename and NOT by git
rename detection. Member-level keys are per-build and unusable for pairing.

## Progress (Go rename pass)

Each increment lands gate-green (build+vet+test -race+gofmt+golangci-lint),
zero behavioral change. The "245.2 class" column below is **historical** once
the corresponding box is checked.

- [x] **Pass A** `dash3d/world` → `dash3d/clientbuild` (type `World`→`ClientBuild`)
- [x] **Pass B** `dash3d/world3d` → `dash3d/world` (type `World3D`→`World`; MUST follow A)
- [x] **Pass C** `config/component` → `config/iftype` (type `Component`→`IfType`)
- [x] **Pass D** `io.Jagfile` → `io.JagFile` (case only)
- [ ] **Pass E** `io/protocol.go` → `client/protocol.go`

Extraction: 72 `.java` files at 176a85f, 74 at 2e62978. All 70 keyed classes
in 245.2 joined to exactly one class in 254. Two keys (`rc`, `sc`) are new in
254 (key-rotation artifacts — the deobfuscator re-keyed SpotAnimType, VarpType,
and WordFilter when inserting two brand-new classes). See the key-rotation note
below.

## Key-rotation warning

Between 245.2 and 254 the deobfuscator inserted two new classes and rotated
three existing class keys one step "up" the alphabet to make room:

| Class | 245.2 key | 254 key | Status |
|---|---|---|---|
| `config/SpotAnimType` | `oc` | `pc` | unchanged class, new key |
| `config/VarpType` | `pc` | `rc` | unchanged class, new key |
| `wordenc/WordFilter` | `qc` | `sc` | unchanged class, new key |
| `client/Stats` | (new) | `oc` | NEW class — key `oc` reassigned to it |
| `config/VarBitType` | (new) | `qc` | NEW class — key `qc` reassigned to it |

A naive `join` on keys `oc`, `pc`, and `qc` produces three FALSE pairings
(`SpotAnimType←→Stats`, `VarpType←→SpotAnimType`, `WordFilter←→VarBitType`).
These were caught by reading both class heads — the joined pairs are structurally
dissimilar. The correct treatment: all three existing classes are unchanged-name;
the two new keys are brand-new classes.

## Class renames and moves (Java) → Go actions

| Obf key | 245.2 class | 254 class | Go action |
|---|---|---|---|
| `c` | `dash3d/World` (scene builder) | `dash3d/ClientBuild` | pkg `dash3d/world` → `dash3d/clientbuild`; type `World`→`ClientBuild` |
| `s` | `dash3d/World3D` (scene graph) | `dash3d/World` | pkg `dash3d/world3d` → `dash3d/world`; type `World3D`→`World` |
| `d` | `config/Component` | `config/IfType` | pkg `config/component` → `config/iftype`; type `Component`→`IfType` |
| `yb` | `io/Jagfile` (case) | `io/JagFile` | type `Jagfile`→`JagFile` (case only; `io/jagfile.go` → `io/jagfile.go` filename unchanged, Go type renamed) |
| `ic` | `io/Protocol` | `client/Protocol` | move `io/protocol.go` → `client/protocol.go`; package import path changes |

> **Name-reuse trap:** "254 `World`" (key `s`, the scene graph) is NOT the same
> as "245.2 `World`" (key `c`, the scene builder). Git's `M World.java` would
> pair two different classes. The chained Go rename MUST land
> `world`→`clientbuild` before `world3d`→`world`.

## New in 254 (delta work, NOT rename churn)

| Obf key (254) | Class | Notes |
|---|---|---|
| `oc` | `client/Stats` | 3-field skill-name/enabled constants table (25 skills). New `client/stats` pkg or file. |
| `qc` | `config/VarBitType` | VarBit config type (basevar, startbit, endbit). New `config/varbittype` pkg. |

## Deleted in 254

None — all 70 keyed classes from 245.2 appear in 254 (some with rotated keys,
none removed). The two path-paired unkeyed files (`deob/ObfuscatedName.java`,
`sign/signlink.java`) are present in both trees with the same names — but
**both have content deltas** (verified by blob hash 2026-06-04); notably
`signlink.java` carries `clientversion = 245→254` and the matching
`reporterror254.cgi` URL — the critical-path login-version delta.

## Unchanged-name pairings

All remaining 65 classes (keyed) pair 1:1 with the same path in both pins.
No further renames or moves beyond the 5 listed above.

Classes paired by path (no class-level obf key — annotation absent):
- `deob/ObfuscatedName.java` — annotation definition itself; not ported (Go has no obf annotations)
- `sign/signlink.java` — present in both, no class-level annotation; no rename (content delta: clientversion 245→254, see above)

## Spot-checks performed

Five same-name pairings verified by reading class heads in both pins and
confirming field structure matches:
- `pb` = `datastruct/LinkList` — `Linkable sentinel` field present in both; confirmed same class
- `mb` = `io/Packet` — `extends DoublyLinkable`, same BigInteger import; confirmed same class
- `gc` = `config/NpcType` — config type with `LruCache`, model field; confirmed same class
- `t` = `datastruct/LruCache` — `int notFound` field in both; confirmed same class
- `u` = `datastruct/HashTable` — `int bucketCount` in both; confirmed same class

All 5 anchor cross-name pairings verified by reading both class heads:
- `c`: `World` (245.2) has `@ObfuscatedName("c")` static scene-builder fields;
  `ClientBuild` (254) has `@ObfuscatedName("c")` same structure — confirmed
- `s`: `World3D` (245.2) → `World` (254) — same scene graph class
- `d`: `Component` (245.2) → `IfType` (254) — same large UI config class (same imports, same role)
- `yb`: `Jagfile` → `JagFile` — same JAG archive class (identical `byte[] data`, `int fileCount` fields)
- `ic`: `io/Protocol` → `client/Protocol` — same protocol-lookup-table class (same array declarations, package-moved)

Key-rotation pairings that look like renames but are NOT:
- `oc`: `config/SpotAnimType` (245.2) vs `client/Stats` (254) — structurally dissimilar;
  SpotAnimType has dozens of fields, Stats has 3 constants. FALSE join — key rotated.
- `pc`: `config/VarpType` (245.2) vs `config/SpotAnimType` (254) — false join
- `qc`: `wordenc/WordFilter` (245.2) vs `config/VarBitType` (254) — false join
