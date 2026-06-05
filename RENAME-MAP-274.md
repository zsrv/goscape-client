# RENAME-MAP-274.md — verified 254↔274 class map

Method: obf-key + name + structural agreement (two-of-three, structure
breaks ties), per PORT-DESIGN-274.md P1. Evidence: audit-274/ scratch
(untracked). Java pins: 254 = 2e62978, 274 = 32f3062.

## How the obf keys moved between revisions

The class-level obf keys shifted by a **uniform −1 ordinal** between 254 and
274 (254 `g`→274 `f`, `sb`→`rb`, `bc`→`ac`, `ac`→`zb`, …). Ordinal numbering
is `a..z` (0–25), then `ab,bb,…,zb` (suffix-`b` group, 26–51), then
`ac,bc,…,zc` (suffix-`c` group, 52+). 254 had two extra early classes
(`VertexNormal` slot and `InputTracking` slot collapsed plus the
`Wave`/`Filter` reshuffle in the `c`-group), and 274 compacted the numbering,
producing the global −1 step.

Verified mechanically over all 65 same-name pairs:
- **45** same-name pairs have differing keys that fit the −1 shift **exactly**
  (0 exceptions) — this is the set in `audit-274/same-name-key-mismatch.txt`.
- **20** same-name pairs keep the **same** key (early single-letter / config
  `c`-group classes that did not move position, e.g. `Client`, `GameShell` `a`,
  `IfType` `d`, `Protocol` `ic`, `Tone` `dc`, plus NOKEY `signlink`).

Because every 274 name maps back to a 254 key that is either identical or its
exact −1 predecessor, **no 274 class name carries a key that belonged to a
different 254 class** → there is no name-reuse trap this revision (unlike
rev-244/254). The −1 shift is therefore corroborating key-evidence for each
same-name pairing, not a confounder.

## Renames (Go action: file/package/type rename, P2)

| 254 class | 274 class | 254 key | 274 key | Verdict & evidence |
|---|---|---|---|---|
| Wave | JagFX | cc | cc | **CONFIRMED**. Same key `cc`. Fields identical: `tracks→synth`, `delay→delays`, `waveBytes`, `waveBuffer`, `Tone[10] tones`, `loopBegin`, `loopEnd`. Methods correspond: `unpack→init`, `generate(int,int)` (kept), `read→load`, `trim→optimiseStart`, `getWave`, `generate(int)→makeSound`. Java sound pkg both sides. |
| VertexNormal | PointNormal | o | n | **CONFIRMED**. Key fits −1 shift. Field-identical: `{x,y,z,w}` all `int` (obf members a/b/c/d both sides). dash3d pkg both sides. |
| Stats | Skill | oc | oc | **CONFIRMED**. Same key `oc`. `COUNT→count`=25, `NAMES→names` (identical 25-string skill array), `ENABLED→used` (identical 25-bool array). client pkg both sides. |
| SpotAnimType | SpotType | pc | pc | **CONFIRMED**. Same key `pc`. Identical field set: `count→numDefinitions`, `list`, `id`, `model`, `anim=-1`, `seq`, `recol_s[6]`, `recol_d[6]`, `resizeh=128`, `resizev=128`, `angle`, `ambient`, `contrast`, `modelCache=LruCache(30)`. Methods: `unpack→init`, `decode` (kept), `getTempModel→getTempModel2`. config pkg both sides. |
| VarBitType | VarbitType | qc | qc | **CONFIRMED**. Same key `qc`. Identical fields: `count→numDefinitions`, `list`, `debugname`, `basevar`, `startbit`, `endbit`. `unpack→init`. **Scope hint:** `decode(int,Packet)` → `decode(Packet,int)` — argument order swapped. config pkg both sides. |
| DoublyLinkable | Linkable2 | x | w | **CONFIRMED**. Key fits −1 shift. Both `extends Linkable`; identical members `next2`, `prev2` (obf f/g both sides), `unlink2()` (obf `b()V` both sides, byte-identical body). datastruct pkg both sides. |
| DoublyLinkList | LinkList2 | qb | pb | **CONFIRMED**. Key fits −1 shift. Identical shape: `sentinel=new <link2>()`, `cursor`, `push(<link2>)`, `head()`, `next()`, `size()`. **Scope hint:** `pop()→popFront()` rename; `next()` obf signature gained a dummy byte arg (`a(I)…`→`a(B)…`). datastruct pkg both sides. |
| InputTracking | *(removed)* | f | — | **REMOVED in 274** — see "Removed in 274". (Listed here for completeness; not a rename.) |

## Package moves (Go action: directory move, P2)

| Class | 254 path | 274 path | Go decision |
|---|---|---|---|
| Protocol | jagex2/client | jagex2/io | **Move Go file `pkg/jagex2/client/protocol.go` → `pkg/jagex2/io`** (REVERT of 254's move; 274 puts it back in `io`, matching ≤245.2). Same key `ic`/`ic`. Cycle check: `protocol.go` has **no imports** (only two `var`/comment blocks: `SERVERPROT_SIZES`, the non-ported `CLIENTPROT_LOOKUP` comment) → no cycle possible; clean move. |
| ClientBuild | jagex2/dash3d | jagex2/client | **Move Go dir `pkg/jagex2/dash3d/clientbuild` → `pkg/jagex2/client/clientbuild`** (mirrors Java; a subdirectory is a separate Go package, so `client` importing it is fine — no cycle). Same key `c`/`c`. Cycle check: `pkg/jagex2/dash3d/clientbuild` does **not** import `pkg/jagex2/client` (grep: 0 hits); the only edge is `pkg/jagex2/client/client.go` → `dash3d/clientbuild` (a one-way parent→child dep that survives the move as parent→subpackage). No cycle. |

## New in 274 (logic-delta scope, not P2)

| Class | Evidence | Notes |
|---|---|---|
| Filter | Key `bc` (the slot freed when `Envelope` shifted `bc`→`ac`). No 254 sound class has its shape — 254 `sound/` has only `{Envelope, Tone, Wave}`, all paired; none declares `pairs`/`frequencies`/`ranges`/`coeff`/`radius`/`calculateCoeffs`. Fields: `pairs[2]`, `frequencies[2][2][4]`, `ranges[2][2][4]`, `unities[2]`, static `coeff[2][8]`/`coeffInt[2][8]`, `reduceCoeff(Int)`; methods `radius`, `frequency` (x2), `calculateCoeffs`, `load(Packet,Envelope)`. | **Logic-delta:** part of the JagFX/Tone audio-synth rework. Tone (274) gained `Filter filter` + `Envelope filterRange` fields and consumes `Filter.calculateCoeffs`/`Filter.coeffInt`/`Filter.reduceCoeffInt` in its generate path. Wire Filter into the Go `sound/tone` synth when porting JagFX. |

## Removed in 274 (logic-delta scope, not P2)

| Class | Evidence | Notes |
|---|---|---|
| InputTracking | No `InputTracking.java` in 274 (`ls-tree` 0 hits); tree-wide `grep` for `InputTracking` over 274 src = 0 hits; `Client.java` references dropped 5 (254) → 0 (274). | **Logic-delta, not a rename:** removing the Go package `pkg/jagex2/client/inputtracking` is a behavior change — it stops the client building/sending the input-tracking (anti-cheat telemetry) payload. Belongs to the logic-delta phase, not the P2 rename pass. The 254 obf slot `f` it occupied is part of why the global key numbering shifted −1. |

## Same-name identity sweep

**All 65 same-name pairs adjudicated — every one CONFIRMED (same entity). No
traps.** Corroboration per pair: same key (20 pairs) or exact −1 ordinal shift
(45 pairs); see "How the obf keys moved" for the mechanical proof. The 45
−1-shift pairs are exactly `audit-274/same-name-key-mismatch.txt`. Structural
spot-checks on `IfType`, `LocType`, `ClientStream`, `ViewBox`, `CollisionMap`,
`Tone` (same-key) and the high-risk list below all corresponded.

Two same-name pairs additionally changed **Go/Java package** (not a type
rename, but flagged for the P2 directory move):
- **WordFilter** (`sc`/`sc`, same key) and **WordPack** (`ac`→`zb`, −1 shift):
  Java pkg `jagex2/wordenc` → `jagex2/wordfilter` in 274. Go side: move
  `pkg/jagex2/wordenc/wordfilter` and `…/wordpack` to `pkg/jagex2/wordfilter/*`
  to mirror, or keep as an intentional deviation (small, isolated; decide in
  the P2 pass).

### Prior-trap high-risk classes — individually re-verified (keys both sides)

| Class | 254 key | 274 key | Shift | Verdict & structural note |
|---|---|---|---|---|
| Wall | r | q | −1 ✓ | **CONFIRMED**. `{y,x,z,angle1,angle2,model1,model2,typecode,typecode2,…}` identical. |
| World | s | r | −1 ✓ | **CONFIRMED** (same entity, the rev-254 `World3D`-lineage scene graph). Heavily restructured 254→274: `maxLevel→maxTileLevel`, added `groundh[][][]`, `occlusionCycle[][][]`; `changedLocCount→dynamicCount`, `changedLocs[5000]` reworked; static `topLevel→maxLevel`. **Scope hint: large logic-delta.** Keeps `lowMem`, `maxTileX/Z`, `minLevel`, statics `fillLeft/cycleNo/minX/maxX`. No name-reuse: 274 `r` ≡ 254 `s` (−1), and 254 `r`(=Wall) maps to 274 `q`(=Wall) — clean. |
| Sprite | q | p | −1 ✓ | **CONFIRMED**. `{level,y,x,z,model,minTileX/maxTileX/minTileZ/maxTileZ,distance,cycle,…}` identical. **Scope hint:** `angle→yaw`. |
| Ground | j | i | −1 ✓ | **CONFIRMED**. `{vertexX/Y/Z, …, flat=true, shape}` correspond. **Scope hint:** `triangleColourA/B/C→faceColourA/B/C`, `triangleVertexA/B/C→faceVertexA/B/C`, `triangleTextureIds→faceTexture`, `shape→overlayShape`. |
| GroundObject | l | k | −1 ✓ | **CONFIRMED**. `{y,x,z,…,typecode,height}` identical. **Scope hint:** `top/bottom/middle→topObj/bottomObj/middleObj`. |
| QuickGround | p | o | −1 ✓ | **CONFIRMED**. 4 corner colours + texture + `flat=true` + rgb. **Scope hint:** `swColour/seColour/neColour/nwColour→colourSW/SE/NE/NW`, `textureId→texture`, `rgb→minimapRgb`. |
| Decor | i | h | −1 ✓ | **CONFIRMED**. `{y,x,z,wshape,angle,model,typecode,typecode2}` identical. |
| GroundDecor | k | j | −1 ✓ | **CONFIRMED**. `{y,x,z,model,typecode,typecode2}` identical. |
| LocChange | ob | nb | −1 ✓ | **CONFIRMED**. `extends Linkable`; full field set `{level,layer,x,z,old/new Type/Angle/Shape,startTime,endTime=-1,…}` identical. |
| Square | w | v | −1 ✓ | **CONFIRMED**. `extends Linkable`; `{level,x,z,originalLevel,quickGround,ground,wall,decor,groundDecor,groundObject,…}` identical. **Scope hint:** `primaryCount→spriteCount`, `Sprite[5] sprite→int[5] spriteSpan`. |

No TRAP findings anywhere in the sweep.

## CLIENTPROT_SCRAMBLED liveness

**Declaration-only in 274 (verdict: existing non-port stands).**
`git grep -c CLIENTPROT_SCRAMBLED 32f3062` = 1 hit, all in
`jagex2/io/Protocol.java` line 9 (the 257-entry table literal); zero usages
tree-wide. Note 274 reverted the name back to `CLIENTPROT_SCRAMBLED` (it was
`CLIENTPROT_LOOKUP` at 254). The Go side already intentionally omits this dead
opcode-scramble table (`pkg/jagex2/client/protocol.go` comment, which cites
both names); that decision is unchanged for 274. **Not a logic-delta work
item.** When `protocol.go` moves to `pkg/jagex2/io` (P2), carry the comment
with it.
