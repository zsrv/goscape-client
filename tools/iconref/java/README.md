# iconref/java — Java-225 reference harness for the item-icon rasterizer

This directory renders the **golden pixels** under `cmd/icondump/testdata/`
directly from the **pinned, byte-for-byte** LostCityRS/Client-Java render code.
The Go item-icon port (`cmd/icondump`, `pkg/jagex2/graphics/pix3d`) diffs its
output against these goldens, so the goldens are the single source of truth:
they come from the real client math, never a re-implementation.

It is the rev-225 analogue of the rev-254 Java harness, adapted to the 225-clean
client lineage (the **jag-container era**): archives are raw JAG files loaded
directly — there is **no** `main_file_cache` and **no** OnDemand layer.

## Provenance

- Pinned client: **LostCityRS/Client-Java**, commit **`cc3781de`**, branch
  **`225-clean`** — a different deobfuscation lineage from the 244/254/274 pins
  (`Model`/`Metadata` live under `jagex2/graphics`, the archive class is
  `Jagfile`, the vertex-normal helper is `VertexNormal`). Nothing in the client
  is vendored here; `build.sh` checks that commit out into a throwaway git
  worktree.
- Client jag archives (read-only): `LostCityRS/Server225_2/engine/data/pack/client/`
  — the three files named **`config`**, **`models`**, **`textures`** (raw jag
  archives, no `.jag` extension). These are the 225 pack pipeline's client-jag
  output. There is no `.dat/.idx` cache at 225.
- debugname↔id map: `LostCityRS/Server225_2/content/pack/obj.pack`
  (2886 entries; `ObjType.count` decodes to **2886**, matching).
- Textures: 50 named entries ("0".."49") in the textures jag → `Pix3D.unpackTextures`.

## Build

```
./build.sh          # writes goldens into ../../cmd/icondump/testdata
./build.sh --keep   # keep the temp worktree/classes for inspection
```

`build.sh` checks the pinned client out into a `mktemp` worktree, applies the one
documented patch, compiles the **whole** client + `IconDump.java` with a stock JDK
`javac` (Java 25), and runs the harness headless.

## Stubs / patches

- **Stubs: none.** The pinned client compiles as-is against a stock JDK (Java 25);
  the icon closure (`ObjType`, `Model`, `Metadata`, `Pix3D`, `Pix2D`, `Pix32`,
  `Jagfile`, `Packet`, `Pix8`, …) is self-contained. `java.awt`/`java.applet` are
  part of the JDK; the harness runs headless (`-Djava.awt.headless=true`) and
  never constructs a GUI component (2 `java.applet` deprecation warnings, 0
  errors). At 225 the whole models jag is unpacked once — there is no OnDemand
  provider surface at all.

- **Patch: exactly one line.** `Pix3D.setBrightness` jitters brightness by
  `Math.random() * 0.03D - 0.015D`. A golden must be reproducible bit-for-bit, so
  `build.sh` rewrites that line in the throwaway worktree copy to
  `double var28 = arg1;` (brightness 0.8 used verbatim). Note the jitter local is
  **`var28`** at 225-clean (it was `var3` at the 254/274 pins). This is the one
  intentional deviation from a literal client run — it removes randomness, never
  adds behaviour. The Go port builds its palette with the un-jittered 0.8
  brightness (`pix3d.InitColourTableDeterministic(0.8)`) to match `palette.bin`.

## Init order

Mirrors the client startup (Client-Java @cc3781de `client.java`):
`Pix3D.unpackTextures` → `Pix3D.setBrightness(0.8)` → `Pix3D.initPool(20)` →
`Model.unpack(jag)` (whole archive) → `ObjType.unpack(jag)`.

## Detail-flag name mapping (225-clean → Go port)

The 225-clean client has two detail flags whose names the Go port renamed:

| Java (cc3781de)  | Go port      | controls                                   |
|------------------|--------------|--------------------------------------------|
| `Pix3D.lowDetail`| `LowMem`     | texture detail: `shrink()` vs `crop()`, 16384 vs 65536 texel pool |
| `Pix3D.jagged`   | `LowDetail`  | raster detail (toggled inside `getIcon`)   |

The harness sets `Pix3D.lowDetail = false` before `unpackTextures` — this is the
texture-detail flag, and mirrors the Go tool forcing `pix3d.LowMem = false` (the
128×128 texel path the goldens were dumped with). `getIcon` itself toggles the
raster-detail flag (`jagged` / `LowDetail`).

## Loading path (no cache reader)

At 225 there is no classic `.dat/.idx` cache and no gzip-per-file: each archive is
a raw jag file. `new Jagfile(Files.readAllBytes(path))` decompresses internally
(BZip2 whole-archive or per-file). The Go tool does the same via
`io.NewJagfile(os.ReadFile(...))` — much simpler than the later revisions'
FileStream/OnDemand path.

## Outputs (written to `../../cmd/icondump/testdata/`)

- `palette.bin` — 65536 × int32 **little-endian** (`Pix3D.colourTable`), 262144
  bytes.
- `tri/*.rgba` + `tri/manifest.json` — 8 synthetic single-triangle goldens
  (gouraud small/large/degenerate/alpha-128, flat, and three textured cases). The
  manifest records every input so each case is exactly replayable. Java argument
  order is y-major-first; the Go port mirrors it, so the manifest's named args map
  1:1 into both.
- `icons225/*.rgba` + `*.png` + `sample.json` — the curated item icons rendered
  via `ObjType.getIcon(id, 1)` (225 order: `id, count` — **no** outline param; the
  colored-outline feature arrived at 244).

## 225-vs-254 golden notes

- `palette.bin` and **all 8** tri goldens (**including the three textured cases**)
  are **byte-identical** to rev-254's (`cmp` clean). Two reasons: (1) the HSL
  colour-table build and every rasterizer are unchanged across these lineages;
  (2) neither 225-clean nor 254 has the transparent-collapse guard on the texture
  palette (that guard is a 274-only addition), and the synthetic textured
  triangles use small coords that never overflow int32. They are still
  regenerated branch-native here, never copied.

## Sample composition (`icons225/sample.json`)

Same algorithm as the 254 harness: seed the two mandatory items `bronze_dagger`
(225 id 1205) and `knife` (946), then sweep the obj space filling each category
bucket to 5. The sample lands at **34** icons covering all **7** categories:

```
untextured_gouraud=17, flat_faces=12, textured_faces=5, recoloured=14,
certtemplate=5, countobj_stack=5, alpha_faces=5
```

**`resized` and `ambient_contrast` are omitted — structurally, not merely empty.**
The 225-clean `ObjType` has **no** `resizex/resizey/resizez` and **no**
`ambient/contrast` fields (grep of `ObjType.java` @cc3781de for
`resize|ambient|contrast` returns nothing); those fields were added in later
revisions, so those two categories cannot be detected at all. Every other
category is populated (the harness prints no zero-coverage NOTE).
