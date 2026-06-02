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

## The Go mapping decision

**`ModelSource` becomes a Go interface `{ Draw1(angle, sinPitch, cosPitch, sinYaw,
cosYaw, x, y, z, bitset int) }`** — i.e. 244's 9-arg `draw`. Rationale:
- `*model.Model` **already implements `Draw1`** (the rasterizer) → satisfies the
  interface with zero changes. (Mirrors `Model extends ModelSource` + Model
  overriding `draw`.)
- Animated/entity nodes implement `Draw1` as: `m := <getModel-logic>; if m != nil
  { e.MinY = m.MinY; m.Draw1(args) }` (mirrors the inherited `ModelSource.draw`
  calling the virtual `getModel`). Go has no base-method-calls-virtual dispatch,
  so the `getModel`+delegate is written into each node's `Draw1` (or a shared
  helper takes a `func() *Model`).
- Base state (`vertexNormal`, `minY`): add `MinY int` (default 1000) and, where a
  node needs it, `VertexNormal []*vertexnormal.VertexNormal` to each implementer
  (Go has no shared base fields without embedding; embedding a `modelSourceBase`
  struct is optional sugar — decide at impl time, prefer plain fields if few).
- **`instanceof Model` → Go type assertion.** Where `World3D` reads
  `model.VertexNormal`/`MinY` directly off a scene field (normal-merge, lighting,
  obj-raise), the field is now the interface, so guard with
  `if m, ok := field.(*model.Model); ok && m.VertexNormal != nil { ... }` —
  exactly 244's `instanceof Model` guards. Known sites: `world3d.go:678/680/685`
  (MergeLocNormals/ApplyLighting on `ModelA/ModelB.VertexNormal`), `:751/754`, the
  `addGroundObject` obj-raise, and any `.MinY` reads.

## Ordered sub-increments (each build/vet/test/lint-gated)

1. **3a — Reshape the `ModelSource` interface to `Draw1(9 args)`.**
   - Change `entity.ModelSource` from `Draw() *model.Model` to the 9-arg `Draw1`.
   - `*model.Model` already satisfies it. Update the current implementers
     (`ClientNpc`, `ClientPlayer`, `ClientProj`, `MapSpotAnim`): rename their
     `Draw()`-returns-model to an unexported `getModel()` helper and add `Draw1`
     = `m := e.getModel(); if m != nil { e.MinY = m.MinY; m.Draw1(args) }`.
   - Update the 2 entity-path call sites that do `x.Entity.Draw()` then rasterize
     (`world3d.go:1268`, `:1530`) to call `x.Entity.Draw1(args)` directly.
   - Add `MinY` (default 1000) to those entity structs.
2. **3b — Re-parent `ClientObj` and `ClientLocAnim` onto `ModelSource`.**
   - Give each a `Draw1` (+ `getModel`): `ClientObj.getModel()` = `objType.GetModel(count)`;
     `ClientLocAnim.getModel()` = advance seq vs `LoopCycle`, return
     `locType.GetModel(shape, angle, heightmaps, transformId)`. Port
     `ClientLocAnim.getModel` from 244 `ClientLocAnim.java` (frame-advance, cap
     delta 100 for looping seqs, `seq.getFrameDuration`/`loops`/`numFrames`).
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
