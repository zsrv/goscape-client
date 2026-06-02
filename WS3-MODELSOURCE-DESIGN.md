# WS3 — ModelSource base + scene-graph hierarchy (rev-244)

Execution plan for logic-delta workstream 3 (see `LOGIC-DELTA-SCOPE.md`). This is
the hierarchy rework: 244 unifies every scene element onto a polymorphic
`ModelSource` and draws via `node.draw(...)`. It is **central** (rewires
`World3D`, the ~1500-line renderer), **interconnected** (no isolated slice), and
**not runtime-verifiable in the sandbox** (no display) — so it is planned here
and executed in build/vet/test/golangci-lint-gated sub-increments, with a host
smoke test deferred to after WS1+WS2.

## The architectural gap (Go-225 vs Java-244)

| Aspect | Go 225 (current) | Java 244 (target) |
|---|---|---|
| Scene model fields | concrete `*model.Model` (`Wall.ModelA/ModelB`, etc.) | `ModelSource` (interface/base) everywhere |
| Draw | `World3D` rasterizes inline: `wall.ModelA.Draw1(angle,sinP,cosP,sinY,cosY,x,y,z,bitset)` | `wall.model1.draw(... )` — polymorphic `ModelSource.draw` |
| Animation | `World3D.SetLocModel/SetWallModel/SetWallModels` mutate the stored `*Model` each frame; animated entities go through a `LocEntity` list | each scene field holds a self-animating `ModelSource` (`ClientLocAnim`) whose `getModel()` returns the current frame |
| `ModelSource` iface | `Draw() *model.Model` (returns the model; caller draws) — entity path only | base class: `vertexNormal[]`, `minY=1000`, virtual `getModel()`, concrete `draw(9)` = `getModel()`→cache minY→`model.draw(9)` |
| Entity base | `Entity`→renamed interface; entities have `Draw() *Model` | `ModelSource` class; `ClientEntity/ClientObj/ClientProj/MapSpotAnim/ClientLocAnim extend ModelSource`; `Model extends ModelSource` |

## The Go mapping decision (revised during 3a)

**`ModelSource` stays a Go interface, with its virtual method renamed
`Draw() *model.Model` → `GetModel() *model.Model`** (244's `getModel`). World3D
keeps its existing **resolve-the-model-then-draw** structure (`m := node.GetModel();
m.Draw1(args)`) rather than pushing a void `draw()` into the node.

Why NOT the void `Draw1(9-arg)` interface (the earlier plan): the loc/sprite draw
sites read the resolved model's Y-extent for the visibility cull *before* drawing
(`world3d.go:1530` reads `farthestModel.MaxY` for `LocVisible`; 244 line 1457 reads
`farthest.model.minY`). A void `Draw1` would lose that model reference. Keeping
`GetModel` + call-site draw is behaviourally equivalent to 244's `node.draw`
(which internally calls `getModel`), supports the Y-extent read naturally, and
preserves World3D's structure — lower risk than inverting the draw path.

Consequences for later sub-increments:
- `*model.Model` must satisfy `ModelSource` for the field merge (3c) — but 244
  `Model.getModel()` returns **null** (Model is drawn via its overridden `draw`,
  not `getModel`). So static `*Model` fields are NOT resolved via `GetModel`; the
  merged scene field stays drawable directly. Pin the exact Go shape in 3c: most
  likely the scene field is `ModelSource` and the call site does
  `if m, ok := field.(*model.Model); ok { m.Draw1(args) } else { field.GetModel()?.Draw1(args) }`,
  or `*Model` gets a `GetModel()` returning itself (diverges from Java but is the
  clean Go unification — decide in 3c against the draw loop).
- **244's cached `minY` on `ModelSource`** (set in its `draw`, read for
  visibility) is the mechanism behind the `minY` visibility arg. Port it where a
  site needs it (3c/3d), adding `MinY int` (default 1000) to the nodes. Verify the
  225-vs-244 visibility arg per site: Go-225 passes the resolved model's `MaxY`,
  244 passes the source's cached `minY` — confirm whether that's a real delta or
  deob-renamed field before changing it.
- **`instanceof Model` → Go type assertion** at `World3D` sites that read
  `model.VertexNormal`/`MinY` off a scene field once it's the interface type
  (normal-merge/lighting `world3d.go:678/680/685/751/754`, `addGroundObject`
  obj-raise): `if m, ok := field.(*model.Model); ok && m.VertexNormal != nil {…}`.

## Ordered sub-increments (each build/vet/test/lint-gated)

1. **3a — Rename the `ModelSource` virtual `Draw()` → `GetModel()`.** ✅ DONE.
   Interface + 4 implementers (`ClientNpc`, `ClientPlayer`, `ClientProj`,
   `MapSpotAnim`) + the 2 call sites (`world3d.go:1268`, `:1530`). Pure rename,
   zero behaviour change (build/vet/test/golangci-lint green). World3D keeps
   resolve-then-draw; the `Draw1`-polymorphism + `minY`-caching were intentionally
   NOT adopted here (see revised mapping decision) — deferred to 3c/3d.
2. **3b — Re-parent `ClientObj` onto `ModelSource`.** ✅ DONE.
   Added `ClientObj.GetModel()` = `objtype.Get(index).GetInterfaceModel(count)`
   (Go's count-aware world model == 244 `ObjType.getModel(count)`). `*ClientObj`
   now satisfies `ModelSource` (compile-asserted). Currently dormant — wired into
   the scene in 3c/3d. Build/vet/test/golangci-lint green.
   **`ClientLocAnim` re-parent MOVED to 3c/3d:** unlike `ClientObj` (fields already
   match 244), the Go `ClientLocAnim` (renamed `LocEntity`) has the rev-225 fields
   `Level/Type/X/Z`; 244 needs `shape/angle/heightmapSW/SE/NE/NW` + a new
   constructor — which changes `World.addLoc`'s call signature. So `ClientLocAnim`'s
   field restructure + `GetModel` (frame-advance vs `LoopCycle`: cap delta 100 for
   looping seqs, `seq.getFrameDuration`/`loops`/`numFrames`, then
   `LocType.Get(index).getModel(shape, angle, heightmaps, transformId)`) lands with
   the `World.addLoc` rework (3d), where its constructor call site changes.
3. **3c — Retype scene fields `*model.Model → entity.ModelSource`** in
   `typ.{Wall(ModelA/B→model1/2), Sprite(model), Decor(model), GroundDecor(model),
   GroundObject(top/bottom/middle)}`. Assigning a `*Model` still works (it
   satisfies the interface). Add the `instanceof`→type-assertion guards at the
   VertexNormal/MinY read sites (see above). `Location.model`+`Location.entity`
   collapse to the single `Sprite.model`.
4. **3d — `World.addLoc` rework + delete `SetLocModel/SetWallModel/SetWallModels`.**
   When `loc.anim != -1`, build a `ClientLocAnim` and pass it as the `ModelSource`
   to `addGroundDecor/addLoc/addWall/addDecor` (drop the `LinkList` param + the
   `LocEntity`-list registration). Static locs pass the `*Model` as before.
5. **3e — `World3D` method renames + new getters** (mostly cosmetic, but in scope):
   `addObjStack→addGroundObject`, `getWallBitset→getWallTypecode`, …; add
   `getWall/getDecor/getSprite/getGroundDecor` (needed by the dynamic-loc /
   `LocChange` apply path). Verify the draw loop after the field retype.
6. **3f — `LocChange` merge** (`LocAddEntity`+`LocMergeEntity` → one `LocChange`):
   fields `level, layer, x, z, oldType, oldAngle, oldShape, newType, newAngle,
   newShape, startTime, endTime=-1`. Update Client call sites (`client.go:1140`,
   `:1300/1302`) and the `MergedLocations`/`SpawnedLocations` lists. **Entangled
   with Client's loc-update logic** — coordinate with WS2/Client; may land with
   the Client loc-update port rather than here.

## Risk / verification
- **No headless runtime check.** Each sub-increment must be build/vet/test/
  golangci-lint green AND diff-verified against the 244 Java. The real behavioural
  check is a host smoke test after WS1+WS2 (when the client can connect/render).
- **3a is the keystone** — once the interface is `Draw1`-shaped and Model
  satisfies it, 3c (field retype) and the draw loop fall out naturally.
- Verify during impl: `ClientProj.getModel` Model-ctor/`scale` arg order
  (flagged in scope), and `ClientEntity` move/step parity (gameplay-sensitive).
- Field-name alignment of the scene structs (e.g. `Wall.ModelA→Model1`) can ride
  with 3c or be a later cosmetic pass — they work either way once retyped.
