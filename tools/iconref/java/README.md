# iconref/java — Java-244 reference harness for the item-icon rasterizer

This directory renders the **golden pixels** under `cmd/icondump/testdata/`
directly from the **pinned, byte-for-byte** LostCityRS/Client-Java render code.
The Go item-icon port (`cmd/icondump`, `pkg/jagex2/graphics/pix3d`) diffs its
output against these goldens, so the goldens are the single source of truth:
they come from the real client math, never a re-implementation.

It is the rev-244 analogue of goscape-client rev-274's TypeScript harness
`tools/iconref/dump.ts` and rev-254's `IconDump.java`. The pipeline, init order,
triangle cases, category buckets and sample-selection algorithm are ported from
those so the branches' goldens are produced the same way.

## Provenance

- Pinned client: **LostCityRS/Client-Java**, commit **`01f16088`** (the
  Server244-ref javaclient pin). Nothing in the client is vendored here; the
  build checks out that commit into a throwaway git worktree.
- Reference cache (read-only): `LostCityRS/Server244-ref/unpack-ref/cache/`
  (`main_file_cache.dat` + `idx0..idx4`).
- debugname↔id map: `LostCityRS/Server244-ref/content/pack/obj.pack`
  (2894 entries; `ObjType.count` decodes to 2894, matching).
- The textures jag is archive 0, file 6 → `Pix3D.unpackTextures`.

## Build

```
./build.sh          # writes goldens into ../../cmd/icondump/testdata
./build.sh --keep   # keep the temp worktree/classes for inspection
```

`build.sh` checks the pinned client out into a `mktemp` worktree, applies the one
documented patch, compiles the **whole** client + `IconDump.java` with a stock
JDK `javac` (Java 25), and runs the harness headless.

## Stubs / patches

- **Stubs: none.** The pinned client compiles as-is against a stock JDK (Java 25);
  the icon closure (`ObjType`, `Model`, `Metadata`, `Pix3D`, `Pix2D`, `Pix32`,
  `Jagfile`, `Packet`, `FileStream`, `OnDemandProvider`, `Pix8`, …) is
  self-contained. `java.awt`/`java.applet` are part of the JDK; the harness runs
  headless (`-Djava.awt.headless=true`) and never constructs a GUI component, so
  no AWT stubbing is needed. The only non-file "surface" is a no-op
  `OnDemandProvider` passed to `Model.init`; it is never invoked because all
  archive-1 models are pre-unpacked before any `Model.tryGet`.

- **Patch: exactly one line.** `Pix3D.initColourTable` jitters brightness by
  `Math.random() * 0.03D - 0.015D`. A golden must be reproducible bit-for-bit, so
  `build.sh` rewrites that line in the throwaway worktree copy to
  `double var3 = arg1;` (brightness 0.8 used verbatim). This mirrors dump.ts
  pinning `Math.random => 0.5` (`0.5*0.03-0.015 == 0`) and is the one intentional
  deviation from a literal client run — it removes randomness, never adds
  behaviour. The Go port builds its palette with the un-jittered 0.8 brightness
  (`pix3d.InitColourTableDeterministic(0.8)`) to match `palette.bin`.

## Client-API deltas vs rev-254's harness (01f16088 vs 2e629784)

The 244 pin is one lineage step earlier than 254; the icon closure is
logic-identical but several client symbols were renamed / re-ordered later, so
this harness calls them by their 244 names:

| use | rev-254 (2e629784) | rev-244 (01f16088) |
|-----|--------------------|----------------------|
| archive class | `JagFile` | `Jagfile` |
| 2D clip init | `Pix3D.init()` | `Pix3D.init2D()` |
| bind target | `Pix2D.setPixels(W, px, H)` | `Pix2D.bind(W, px, H)` *(width, data, height)* |
| low-detail flag | `Pix3D.lowDetail` | `Pix3D.jagged` |
| render icon | `ObjType.getSprite(0,1,id)` | `ObjType.getIcon(0,1,id)` |
| decode model | `Model.unpack(data, id)` | `Model.unpack(id, data)` |
| fetch model | `Model.load(id)` | `Model.tryGet(id)` |
| sprite pixels | `Pix32.data` | `Pix32.pixels` |

All are pure renames / argument-order swaps — no render-math change. The Go port
(`pix3d.Init2D`, `pix2d.SetPixels`, `pix3d.LowDetail`, `objtype.GetIcon`,
`model.Unpack`, `Pix32.Pixels`, `io.Jagfile`) already uses its own 244 names;
the golden-test/tool call sites are adapted to match.

## Init order

Mirrors the client startup: `ObjType.unpack` → `Pix3D.unpackTextures` →
`Pix3D.initColourTable(0.8)` → `Pix3D.initPool(20)` → `Model.init` +
pre-unpack of every archive-1 model.

## Cache reader

`IconDump.Cache` is a ~40-line classic `.dat/.idx` reader (520-byte sectors,
6-byte idx entries) ported from dump.ts. Archive 0 (config jags) is stored raw;
archives 1-4 (models/anims/midi/maps) are gzip-per-file — `GZIPInputStream` reads
the gzip member and stops, ignoring the 2-byte version trailer. (Go's own
`pkg/jagex2/io.FileStream`, added alongside this harness, reads the same layout
for the runtime `cmd/icondump` tool.)

## Outputs (written to `../../cmd/icondump/testdata/`)

- `palette.bin` — 65536 × int32 **little-endian** (`Pix3D.colourTable`), 262144
  bytes.
- `tri/*.rgba` + `tri/manifest.json` — 8 synthetic single-triangle goldens
  (gouraud small/large/degenerate/alpha-128, flat, and three textured cases). The
  manifest records every input so each case is exactly replayable. Java argument
  order is y-major-first; the Go port mirrors it, so the manifest's named args map
  1:1 into both.
- `icons244/*.rgba` + `*.png` + `sample.json` — the curated item icons rendered
  via `ObjType.getIcon(0, 1, id)` (244 order: `outlineRgb, count, id`).

## 244-vs-254/274 golden notes

- `palette.bin` and **all 8 tri goldens** — gouraud, flat **and the three
  textured cases** — are **byte-identical** to rev-254's. Two reasons: (1) Java
  244's `initColourTable` has **no** transparent-collapse guard on the texture
  palette (`if ((texPal & 0xF8F8FF) == 0 && j != 0) texPal = 1;` is a 274-only
  addition at Pix3D.java:340-342 @32f30626, absent at both 01f16088 and
  2e629784); and (2) the synthetic textured triangles use small coordinates that
  never overflow int32, so the running-numerator wrap does not bite. These
  goldens are still regenerated by this harness (never copied) so any future
  guard-sensitive item is covered branch-native.
- **P6 int32-wrap gap (latent, not reached):** rev-244's Go `GouraudRaster` /
  `TextureRaster` / `TextureTriangle` use plain 64-bit `int` arithmetic, whereas
  the Java-244 raster is 32-bit (wraps mod 2³²). The explicit `int32()` wrap
  fidelity fixes rev-254/274 carry are *later-rev additions* and are deliberately
  **not** back-ported here (no-forward-port policy — faithful to 01f16088). No
  sample icon and no tri case reaches an int32-overflowing numerator, so every
  golden renders identically to the 32-bit reference (full run 2894/2894 = 100%,
  all goldens GREEN). The gap surfaces only on extreme geometry / overflow; if a
  future sample item ever reaches it, the golden diff will flag it.

## Sample composition (`icons244/sample.json`)

Same algorithm as dump.ts: seed the two mandatory items `bronze_dagger` (244 id
1205) and `knife` (946), then sweep the obj space filling each category bucket to
5. Of the 9 categories, **two have no members in the 244 obj space** —
`resized` (no obj sets a non-128 resize) and `ambient_contrast` (its carriers, the
eadgar/viking quest items, are post-244 additions) — so the sample lands at 34
icons covering the 7 populated categories (untextured_gouraud, flat_faces,
textured_faces, recoloured, certtemplate, countobj_stack, alpha_faces). The
selected ids/debugnames are identical to rev-254's 34-item sample.
