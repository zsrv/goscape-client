# rev-274 Parity Audit — 2026-06-06

Exhaustive line-by-line, side-by-side function walk of the Go client (branch `rev-274`,
HEAD `bedf95f`) against the Java 274 reference (`Client-Java` git `32f3062`), modeled on
the rev-225, rev-244, rev-245.2, and rev-254 audits (PARITY-AUDIT-2026-05-28.md,
PARITY-AUDIT-2026-06-03.md, PARITY-AUDIT-2026-06-04.md, PARITY-AUDIT-2026-06-05.md).

## Method

- 53 audit units covering **every** Java file under `src/main/java` at `32f3062`
  (large files chunked by declaration-line windows: Client.java ×12 — a preamble/fields
  unit walking all ~414 field declarations plus the 11 method chunks adjudicated in
  `audit-274/p3/client-chunks.json` — Pix3D/World/Model ×3, ClientBuild/WordFilter ×2;
  the rest singly or bundled; plus one dedicated exhaustive wire-opcode unit covering
  Protocol.java's 257-entry size table and every inbound/outbound/zone opcode,
  non-diff-based; plus 3 reverse-coverage units).
  **~967 Java methods/branches walked** statement by statement: packet read widths and
  signedness; operator grouping; branch polarity; loop bounds/direction; argument
  semantics traced through callee bodies (deob param scramble); 32-bit wrap /
  sign-extension / shift-mask semantics; side-effect ordering. Comments were not
  trusted; claimed equivalences were re-verified against cited code.
- **Audit-process defect found and corrected:** the original `client-08` agent read
  `chunks[7]` 1-based and silently re-audited `client-07`'s 11 methods (caught by
  cross-checking per-unit counts against the P3 chunk table). The duplication is kept
  as an independent cross-confirmation of that chunk's clean verdict; the real
  `chunks[7]` — `tcpIn/0` (Client.java:7746–8724) and `clientButton/1` (8725–8844) —
  was covered by two corrective units (`client-08b-A`: 41 branches with case labels
  ≤8290; `client-08b-B`: 20 branches ≥8291 + error tail + clientButton). The corrective
  pass found the audit's only confirmed **bug** (below), and re-verified that the
  254-audit blocker shape (IF_SETANIM `g2b` + `==-1` seq reset) is present and correct
  in Go (client.go:11231-11240).
- Every blocker/bug/latent finding was independently **adversarially verified** by a
  skeptic agent primed with the known false-positive classes (the eight member-level
  274 name-reuse traps, the ~146 refuted churn claims from LOGIC-DELTA-SCOPE-274.md,
  deob arg scramble / compensated pairs, Java implicit shift-count masking,
  restructured-but-equivalent control flow, behavior-lives-elsewhere, documented
  seams). 12 routed → **11 confirmed, 1 refuted**. UPDATE_STAT's unwrapped g4 was
  found *twice independently* (wire-opcode unit + tcpIn body walk), so the 11
  confirmations describe **10 distinct findings**; two of those were downgraded to
  cosmetic by their verifiers (final severities below).
- Reverse coverage: all Go files under `pkg/` and `cmd/` (incl. tests) classified;
  **0 suspicious** (port / seam-justified / test / tooling). Two orphan packages
  outside the assigned dirs (`pkg/profiling/`, `pkg/util/build/`) were reconciled by
  reverse-3 as Go-side tooling seams.
- 65 agents, ~6.3M tokens, two workflow runs (`wf_72511a61-585` + corrective
  `wf_85368072-e22`). Unit evidence reports in `audit-274/units/` (untracked scratch,
  like `audit-225/`, `audit-245/`, `audit-254/`); raw run payloads in `audit-274/p5/`.

## Verdict summary

| Final severity | Distinct | Confirmations |
|---|---|---|
| **Blocker** (crash / protocol or cache-format desync) | 0 | — |
| **Bug** (wrong behavior or rendering) | 1 | 1 |
| **Latent** (edge-value / not-yet-reachable) | 7 | 8 |
| Verified cosmetic (confirmed real, downgraded by verifier) | 2 | 2 |
| Cosmetic (comments/naming/dead code; unverified) | 18 | — |
| Intentional (documented deviations/seams) | 25 | — |
| Methods missing in Go | 0 | — |
| Refuted by verification | 1 | — |

Fix order: the bug first (one-line fix, reachable wrong render), then the two verified
cosmetics (user-visible text/UI staleness), then the latent int32-wrap family.
This document is a snapshot — the fix pass lands on top of it; live status belongs in
resume notes, never here.

---

## Bugs (1)

### client-08b-01. FRIENDLIST_LOADED (ptype 185) drops `redrawSidebar = true`

- **Unit:** `client-08b-B`  **Java:** `Client.java:8648-8652 @32f3062`  **Go:** `pkg/jagex2/client/client.go:11547-11551 (SERVERPROT_FRIENDLIST_LOADED handler)`

Java: `if (ptype == 185) { friendServerStatus = in.g1(); redrawSidebar = true; ptype = -1; return true; }`.
Go: `c.FriendListStatus = c.In.G1(); c.PacketType = -1; return true` — the
`RedrawSidebar = true` write is absent. This is Java's sole on-wire write of
`friendServerStatus` (0 none / 1 connecting / 2 loaded), and the unconditional dirty
flag is what repaints the friends panel on connection-state transitions. Reachable
network path; the panel shows a stale "Connecting to friendserver" (or fails to show
it) until some unrelated event next dirties the sidebar.

> **Verifier (confirmed, severity bug):** Java 32f3062 Client.java:8648-8652: opcode
> 185 sets `friendServerStatus = in.g1(); redrawSidebar = true;`. Go
> client.go:11547-11551 (SERVERPROT_FRIENDLIST_LOADED=185, serverprot.go:53) sets
> FriendListStatus=In.G1() but omits RedrawSidebar=true. FriendListStatus (decl :470)
> drives friend-panel slot text (:5709-5770: "Loading friend list" / "Connecting to
> friendserver" vs the list). DrawSidebar (:11787) gates the pixel rebuild on
> RedrawSidebar; the blit re-issues stale pixels every frame (comment :11784-11787).
> No downstream trigger ties a status-only transition to a redraw. Reachable network
> path → stale friends panel until next unrelated dirty event. Self-heals on next
> sidebar dirty, but transient wrong render. Severity bug.

---

## Latent (7 distinct / 8 confirmations)

### wire-01 ≡ client-08b-02. UPDATE_STAT (ptype 105) g4 XP read not wrapped to int32

- **Units:** `wire` + `client-08b-B` (found twice independently; both verdicts kept)
- **Java:** `Client.java:8471-8485 @32f3062`  **Go:** `pkg/jagex2/client/client.go:11717-11726`

Java `int var104 = in.g4()` is signed 32-bit (bit-31 XP goes negative; the level loop
`var104 >= levelExperience[var106]` then never fires). Go `var4 := c.In.G4()` returns
the unsigned bit pattern in a 64-bit int with **no** `int(int32(...))` wrap — the cast
that sibling handlers VARP_LARGE (11217), LAST_LOGIN_INFO (11297), and both
UPDATE_INV g4 fields (11264/11764) all apply. For XP ≥ 2^31 Java stores negative and
stalls the level computation; Go stores ~4.2e9 positive and sets level 99. Unreachable
in practice: era XP caps at 200,000,000 (< 2^31). Fix: `int(int32(c.In.G4()))`.

> **Verifier (wire route — confirmed latent):** Verified at pin … Go client.go:11718
> omits the int32 cast that siblings 11217/11265/11298/11764 all apply … wire XP caps
> at 200,000,000 (<2^31), so unreachable. Confirmed only-unwrapped ordered-compare g4
> inbound (10782 is equality-only msgId). Latent stands.
> **Verifier (tcpIn route — confirmed latent):** statXP/levelExperience are int[]
> (32-bit, Client.java:592/166) … diverging the level loop + display read-back (9959).
> Not a name/reorder/dead-arg FP. Latent: era XP caps ~200M.

### client-01-01. GetIfVar accumulates in Go 64-bit int vs Java int32

- **Unit:** `client-01`  **Java:** `Client.java:2252-2376 (getIfVar)`  **Go:** `pkg/jagex2/client/client.go:9925-10085 (GetIfVar)`

Java performs the op4/op10 multiply-accumulate (`var5 *= var9`, `var9 +=
linkObjNumber`) and the op7 `varps[..]*100/46875` in 32-bit int that wraps; Go does the
identical ops in 64-bit int. Concrete diverging vector (verifier): varp = 30,000,000 →
op7 yields 64,000 in Go vs Java's `30000000*100` wrapping to −1,294,967,296, /46875 =
−27,625. Varps are server-settable to any int32 via VARP_LARGE, so the vector needs an
adversarial out-of-range varp (>~21.5M) — not normal gameplay. Shift ops 13/14 are
correctly `&0x1F`-masked on both sides (10029/10046).

### client-11-01. saveWave nil-vs-oversize guard ordering inverted

- **Unit:** `client-11`  **Java:** `Client.java:11206-11210`  **Go:** `pkg/jagex2/client/client.go:5960-5965`

Java: `return arg0 == null ? true : signlink.wavesave(arg0, arg1)` — the
size>2,000,000 reject lives inside wavesave and is only reached for non-null buffers.
Go checks `arg1 > 2000000 → false` **before** `arg0 == nil → true`. Sole diverging
input: `arg0==nil && arg1>2000000` (Java true, Go false). Unreachable on the live
path: soundsDoQueue always passes JagFX.generate's non-nil 441,000-byte buffer.

### pix3d-C-01. textureRaster running perspective numerators not re-wrapped to int32

- **Unit:** `pix3d-C`  **Java:** `Pix3D.java:2101-2105, 2143-2145, 2273-2280`  **Go:** `pkg/jagex2/graphics/pix3d/pix3d.go:2304-2308, 2350-2367, 2117-2123, 2179-2185`

Java keeps the u/v/w texture numerators as 32-bit int — every `+`/`+=` wraps mod 2^32.
Go wraps only the initial cross-products (`var32/33/34 = int(int32(...))`) and then
lets `arg9/arg10/arg11` grow unwrapped in 64-bit (`arg9 = var32 + arg12`; in-loop
`arg9 += arg12`), in both the high- and low-detail paths. Verifier reproduced a
concrete divergence (var34=2e9, arg14=5e8 ⇒ divisor sign flip ⇒ texel column diverges).
Hot path (every textured triangle) but divergent only on numerator overflow — extreme
geometry/coords. Fix mirrors the existing wraps: `int(int32(arg9 + arg12))` for all
three accumulators in both detail paths.

### model-B-01. Rotation multiply-accumulate uses Go 64-bit int vs Java 32-bit

- **Unit:** `model-B`  **Java:** `Model.java:1208-1232 (animate2 arg0==2), 1289 (rotateXAxis)`  **Go:** `pkg/jagex2/dash3d/model/model.go:1220-1234, 1291`

Java computes `pointA*sin ± pointB*cos` in 32-bit int before `>>16`; Go in 64-bit.
With sin/cos ≤65536 and realistic (gsmart-accumulated) model coords <~16k, the products
stay below int32 max — identical on any real model; diverges only on
adversarial/corrupt coordinates. Parenthesization itself is correct on both sides.
Same family as the textureRaster finding (and the 254 audit's dominant latent class).

### ondemand-01. priorities array byte signedness: unsigned Go vs signed Java

- **Unit:** `ondemand`  **Java:** `OnDemand.java:27, 350, 559`  **Go:** `pkg/jagex2/io/ondemand/ondemand.go:104, 496, 675`

Java `byte[][] priorities` is signed; comparisons sign-extend. Go `[4][]byte` is
unsigned; `int(priority)` zero-extends. Diverges for any priority ≥128 — but every
caller passes literals ≤10 (verified at all five call sites both sides). Type-
faithfulness divergence only; the int8 declaration rule from
feedback_porting_byte_field_sign_extension applies if ever touched.

### datastruct-01. ToAsterisks counts runes, not UTF-16 code units

- **Unit:** `datastruct`  **Java:** `JString.java:108-110 (getRepeatedCharacter)`  **Go:** `pkg/jagex2/datastruct/jstring/jstring.go:142 (ToAsterisks)`

Java appends one `*` per UTF-16 code unit (`arg0.length()`); Go's `for range s`
iterates runes. Identical for all BMP input; a supplementary-plane char yields 2 stars
in Java vs 1 in Go. Unreachable: the input path is gated by the BMP/ASCII CHARSET
membership check (client.go:2578). Classic UTF-16-vs-bytes class
(feedback_porting_string_byte_vs_char), recorded for completeness.

---

## Verified cosmetic (2) — confirmed real by skeptics, severity downgraded

### client-02-01. clientCode 654 members-warning text typo "unavailabe" + false `[sic]` comment

- **Unit:** `client-02`  **Java:** `Client.java:2570 @32f3062`  **Go:** `pkg/jagex2/client/client.go:5942`

Java-274 spells `"…member benefits are unavailable whilst here."`; Go renders
`"unavailabe"`. The adjacent Go comment claims `[sic] in the source`, but the pin
spells it correctly — the comment is wrong, the typo is ours (likely inherited from an
earlier rev's source that did misspell it). Reachable UI path (clientCode 654,
daysSinceRecoveriesChanged==201, warnMembersInNonMembers==1). Fix the string and drop
the false comment.

### ondemand-02. Title-screen "Loading extra files" status line never clears

- **Unit:** `ondemand`  **Java:** `OnDemand.java run() post-loop else: { packetCycle = 0; message = ""; }`  **Go:** `pkg/jagex2/io/ondemand/ondemand.go:552-574 (Run)`

Java blanks the welcome-screen status line whenever no urgent request is pending — the
only place `message` is cleared. Go's Run() (restructured for the WS1 socket seam)
drops the clear; `od.message` is only ever set to `"Loading extra files - N%"`
(ondemand.go:661, 690) and consumed at client.go:3652. After prefetch drains, the line
sticks at "100%" where Java shows blank. UI staleness only — no functional, protocol,
or cache effect.

---

## Refuted by verification (1)

### reverse-2-01. GouraudTriangle vertex-colour edge slopes lack int32 wrap *(finder severity: latent)*

Finder misread the signature: `gouraudTriangle(Y0,Y1,Y2,X0,X1,X2,colA,colB,colC)`
(confirmed by Model.java:1809) — arg3/arg4/arg5 are bounded screen-X coordinates, not
HSL indices. Only arg6..arg8 are colours 0..65535 and their slopes use `<<15`:
max `65535<<15 = 2,147,450,880 < 2^31−1` — no overflow possible. The `<<16` terms
operate solely on X coords. The divTable wrap at pix3d.go:899 is needed because it is
a *product* (delta×32768) — a different magnitude class. Not a divergence.

---

## Cosmetic (18) — recorded, not verified, fix opportunistically

| # | Unit | Item |
|---|---|---|
| 1 | client-00 | GameShell `debug` field absent from Go (loop-control flag inside the host-shell seam; GameShell.java:30/213/222) |
| 2 | client-01 | Stale `// Java:` comments in MapBuild cite pIsaac(206)/clearLocChanges(3379); constants/calls correct (client.go:9726-9771) |
| 3 | client-05 | getNpcPosOldVis too-many-npcs panic payload differs (Go panics with the reported message; Java throws `RuntimeException("eek")`) |
| 4 | client-06 | getPlayerPosOldVis panic payload differs from Java `"eek"` (client.go:3478-3482) |
| 5 | client-07 | getPlayerPos panic payload differs from Java `"eek"` (client.go:8929-8937) |
| 6 | client-08 | duplicate of #5 (same site, found via the duplicated chunk walk) |
| 7 | client-10 | UpdateOrbitCamera/StartForceMovement carry pre-274 deob dummy param doing `PacketSize += 0`; 274 dropped the param (client.go:6802, 4347; all call sites pass 0) |
| 8 | client-11 | getNpcPos panic payload differs from Java `"eek"` (client.go:3354, 3360) |
| 9 | pix3d-A | Go `LowDetail`/`Jagged` name Java `lowMem`/`lowDetail` respectively — consistent at every site (pix3d.go:14-15) |
| 10 | model-A | unpack has extra trailing dead write `pos += dataLengthZ` not in Java; adjacent comment claims "verbatim" — comment wrong (model.go:308-311) |
| 11 | model-B | Go `MaxY`/`MinY` hold Java `minY`/`maxY` respectively — swap applied consistently at every read/write incl. Draw1/worldRender (model.go:982-997, 1549, 1554) |
| 12 | clientbuild-A | SpreadHeight doc-comment cites 244-lineage `spreadHeight`; 274 name is `fadeAdjacent` (clientbuild.go:131-137) |
| 13 | typ | Ground ctor trailing **dead** block reflowed (max-accumulator form) + Go-only `/14` divisions not in Java-274; all in provably dead code, stale `// Java: Ground.java:289/290` cites (ground.go:228-245) |
| 14 | animmeta | Descriptive 254-lineage renames on AnimFrame/AnimBase/Metadata fields and methods (ShareAlpha, Length/Types, offset names) |
| 15 | pixfontmap | PixMap.Bind keeps 254 name; 274 renamed bind→setPixels (trap 1e family; whole-family rename deferred) |
| 16 | bzip2 | Dead `field788` (int[257]) and dead local `var47` correctly omitted (recorded for the ledger) |
| 17 | sound | Envelope genInit/genNext deferred-renamed Reset/Evaluate (bodies identical) |
| 18 | wire | Stale opcode numbers in Go interactWithLoc `// Java:` comments (208/87 vs actual 103/157); emitted constants verified correct |

(The corrective tcpIn walk also noted a stale `pIsaac(150)` comment at the opcode-125
emit site — same primer-7 comment-rot class, not separately tabled.)

## Intentional (25) — documented deviations/seams, no action

| # | Unit | Item |
|---|---|---|
| 1 | client-01 | combatColourCode subtraction order inverted, compensated at both call sites (primer 2) |
| 2 | client-05 | Boot 12-map prefetch always skipped: OnDemand wired with nil cache (`HasCache()` false) — signlink/cache seam |
| 3 | client-05 | Host allowlist + FileStream cache-open loop not ported (standalone-client + signlink seam) |
| 4 | client-05 | messageBox/lostCon drop redrawFrame early-return + flameActive gate (immediate-mode GPU upload seam) |
| 5 | client-06 | loadTitleImages fl_icon alternate-rune branch dropped (getParameter applet seam; default path identical) |
| 6 | client-06 | getHost drops applet frame-null "runescape.com" fallback (CLI host seam) |
| 7 | client-07 | maindraw `drawCycle++` not ported (read only by unported lag(); flameCycle++ class) |
| 8 | client-07 | drawError reconstructs AWT Graphics via overlay PixMap + errorfont seam (strings/coords match) |
| 9 | client-08 | Stale 254-era pIsaac comments in AddFriend/AddPlayers; constants correct (13, 188) |
| 10 | client-09 | lag() debug-dump not ported (Go-side debug seam; ::lag still reaches server) |
| 11 | client-09 | getBaseComponent() not ported (AWT Component / platform seam) |
| 12 | client-10 | showLoadError applet showDocument browser-nav not portable; println + permanent halt preserved |
| 13 | client-08b-A | REBUILD_NORMAL `areaViewport.setPixels()` rendered as Go `AreaViewport.Bind()` (trap 1e, known deferred rename; body identical) |
| 14 | pix3d-A | ClearTexels/InitPool buffer-conserving reuse replaces Java null+realloc (unobservable; GetTexels fully overwrites) |
| 15 | model-A | ModelSource minY=1000 default not on Go Model.MinY (every draw path recomputes; the default that matters is reproduced at typ/decor.go:24, typ/sprite.go:33) |
| 16 | signlink | CacheLoad/CacheSave/LoadReq/GetHash have no Java-274 counterpart (documented storage seam) |
| 17 | config-small | TRAP 1h verified: FloType Luminance/Chroma map to Java chroma/underlayHue by role, end-to-end through ClientBuild blending |
| 18 | config-small | UnkType.java is a dead deob artifact (sole use: `UnkType.list = null`; never instantiated); no Go package is correct |
| 19 | config-small | VarpType/VarbitType drop never-read deob fields; every wire read preserved as discard (byte-alignment identical) |
| 20 | animmeta | AnimFrame `opaque[]` + type-5 clear not ported (zero readers tree-wide; dead-write artifact) |
| 21 | animmeta | AnimFrame.unload inlined as `animframe.List = nil` at the one call site (same effect) |
| 22 | datastruct | LruCache.Delete diverges from Java unlink() leak-then-reclaim (icon-cache eviction timing; standing Go LruCache decision) |
| 23 | bzip2 | K0/k1 unsigned in Go vs signed-byte Java — equality-pattern compares + low-8-bit outputs, net-neutral |
| 24 | streams | FileStream .dat/.idx sector store not ported — replaced by ondemand.Cache + signlink storage seam (wired nil; see intentional #2) |
| 25 | streams | ClientStream.debug() stdout diagnostics not ported (documented io-net #19 decision) |

## Reverse coverage

All Go files under `pkg/` and `cmd/` (116 source + 60 test) classified by three
reverse units; reverse-3 reconciled the full repo and adjudicated the two orphans
(`pkg/profiling/` SIGUSR1 pprof harness, `pkg/util/build/` link-time version metadata)
as Go-side tooling. No dead Go-only logic, no leftover-254 behavior that 274 removed,
0 suspicious files. The config decoders and ClientBuild load methods (highest-risk
reverse Packet-read surface) were additionally walked forward by reverse-1.

## Notable clean verifications (for the P7 smoke ledger)

- IF_SETANIM (the 254 blocker shape): `g2b` signed read + `== -1` seq reset present
  and correct (client.go:11231-11240).
- WS8 deltas re-verified at the pin: WordFilter trailing-digit trailing-TERM sign flip,
  `else{stride=1}`, 10-entry allowlist; chat-accept 32..122; AddWorldOptions
  co-located-player loop.
- gsmart/gsmarts name-swap (trap 1b) verified by body at **every** call site across
  Model/ClientBuild/Tone/animmeta — packetio additionally proved both bodies by
  executable simulation.
- Packet data↔pos role swap (trap 1a) mapped correctly everywhere, including the
  VarpType/VarbitType `Pos != len(Data)` mismatch checks and Pix32/AnimFrame cursors.
- NpcType walkanim_l/r obf-slot swap + compensating decode order + routeMove
  turn-branch flips verified net-neutral end-to-end (trap 1g).
- SERVERPROT size table byte-identical (257 entries, machine-diffed); all 69 inbound
  dispatch values, 10 zone opcodes, every outbound opcode/payload/ISAAC site, and the
  login handshake verified exact.
- moveDecor typo-fix site at the pin commit itself (ClientBuild.java/World.java)
  verified ported.

## Artifacts

- Unit evidence: `audit-274/units/*.md` (53 files, untracked scratch).
- Raw run payloads + per-unit notes: `audit-274/p5/run1-full.json` (main, 51 units),
  `audit-274/p5/fixup-full.json` (corrective client-08b units).
- Scope/pairing inputs: `LOGIC-DELTA-SCOPE-274.md`, `audit-274/p3/client-chunks.json`
  (seed correction confirmed in-run: getIntString↔inf, formatObjCountTagged↔niceNumber,
  formatObjCount↔invNumber).
