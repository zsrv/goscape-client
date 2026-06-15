# rev-244 Parity Audit — 2026-06-03

Exhaustive line-by-line, side-by-side function walk of the Go client (branch `rev-244`,
HEAD `3a0b67b`) against the Java 244 reference (`Client-Java` git `01f16088`), modeled on
the rev-225 audit (PARITY-AUDIT-2026-05-28.md).

## Method

- 50 audit units covering **every** Java file under `src/main/java` at `01f16088`
  (large files chunked by declaration-line windows: Client.java ×13, Pix3D/World3D/Model ×3,
  World/WordFilter ×2; the rest singly or bundled). **683 Java methods walked** statement by
  statement: packet read widths, operator grouping, branch polarity, loop bounds, argument
  semantics traced through callee bodies (deob param scramble), 32-bit wrap / sign-extension /
  shift-mask semantics, side-effect ordering. Comments were not trusted; claimed equivalences
  were re-verified against cited code.
- Every blocker/bug/latent finding was independently **adversarially verified** by a skeptic
  agent primed with the known false-positive classes (deob arg scramble, cross-lineage name
  inversions such as Java-244 `Model.minY` ≡ Go `Model.MaxY`, restructured-but-equivalent
  control flow, behavior-lives-elsewhere, documented seams). 136 routed → **132 confirmed, 4 refuted**.
- Reverse coverage: all 112 Go files classified; **0 suspicious** (every Go-original file is
  seam-justified: platform/transport/storage/audio-backend/profiling/build tooling).
- 188 agents, ~9.9M tokens, single workflow run (`wf_210d4603-cf8`).

## Verdict summary

| Final severity | Count |
|---|---|
| **Blocker** (crash / protocol or cache-format desync) | 11 |
| **Bug** (wrong behavior or rendering) | 62 |
| **Latent** (edge-value / not-yet-reachable) | 50 |
| Cosmetic (comments/naming/dead code) | 64 |
| Deferred (explicitly unimplemented) | 20 + ledger below |
| Intentional (documented deviations) | 126 |
| Methods missing in Go | 9 |
| Refuted by verification | 4 |

Fix order: blockers first (each can desync the session), then bugs, then latent.
Duplicate findings across adjacent units were not auto-merged — triage may collapse a few.

---

## Blockers (11)

### B01. Unpack nulls ModelCache (no Java equivalent) -> nil-panic on every later interface model load

- **Unit:** `Component`  **Java:** `Component.java:119,204,443`  **Go:** `pkg/jagex2/config/component/component.go:96,104-107,358,374,381,383`

Java Component.modelCache is a static field initialized ONCE at class load (line 119 `new LruCache(30)`) and is NEVER nulled anywhere in Component (no unload method; grep confirms only line 119 assigns it). Java unpack only nulls imageCache (line 443 `imageCache = null;`). The Go Unpack additionally creates ModelCache inside Unpack (line 96) AND nulls it on the EOF-return path (line 106 `ModelCache = nil`). After Unpack returns at boot, component.ModelCache is nil and nothing re-creates it (grep shows component.go:96 is the only producer outside the test file). Any subsequent type-6 interface (NPC head / player head / item model) hits LoadModel/GetModel/CacheModel which dereference ModelCache (ModelCache.Get line 358, ModelCache.Put line 374, ModelCache.Clear line 381 in CacheModel called from client.go:5341) and will nil-panic. NpcType/ObjType correctly null their ModelCache only inside an Unload() method (NpcType.java:130 is in unload(), not unpack), so this Go site is a mis-port of that pattern into the wrong lifecycle hook.

```
Java: line 443 `imageCache = null;` (modelCache untouched). Go: lines 105-106 `ImageCache = nil` then `ModelCache = nil`. Fix: drop line 106. ImageCache=nil is correct; ModelCache=nil is the bug.
```

> **Verifier (confirmed, severity blocker):** Confirmed real after attempting refutation on every false-positive axis.

Java (git 01f16088 Component.java): line 119 `public static LruCache modelCache = new LruCache(30);` is a STATIC FIELD INITIALIZER — created once at class load, and grep over the whole file shows it is NEVER reassigned or nulled (only .get/.put/.clear at 493/511/519/522). unpack() touches ONLY imageCache: creates it at line 204 (`imageCache = new LruCache(50000)`) and nulls it at line 443 (`imageCache = null;`) just before returning. modelCache is deliberately left alive.

Go (component.go): Unpack creates ModelCache at line 96 AND nulls it at line 106 on the EOF-return path. I verified this EOF path is the NORMAL exit […]
>
> **Refined:** Go component.Unpack (pkg/jagex2/config/component/component.go) nulls ModelCache at line 106 on its normal EOF-exit path, with no Java equivalent. Java Component.modelCache is a static field initializer (Component.java:119 `new LruCache(30)`) created once at class load and never nulled; Java unpack only nulls imageCache (line 443). The Go EOF-return (lines 104-107) is the normal end of Unpack (translating Java's `while (data.pos < data.data.length)` loop-exit + post-loop `imageCache = null`), so  […]

### B02. Inlined swapObj call site drops rev-244 bankArrangeMode reorder loop and trailing p1(mode) INV_BUTTOND byte

- **Unit:** `Component`  **Java:** `Client.java:3010-3043 (Component.swapObj Component.java:447-455)`  **Go:** `pkg/jagex2/client/client.go:7505-7517`

Component.swapObj is not present as a Go method; its swap math is inlined at client.go:7507-7512 and the swap itself is correct (swaps InvSlotObjId[Hovered]<->[Drag] and InvSlotObjCount likewise; symmetric, matches swapObj(objDragSlot,hoveredSlot) = Java line 3034). However the Java 244 caller wraps swapObj in a bankArrangeMode mechanism: it computes `byte mode` from `bankArrangeMode==1 && com.clientCode==206` (clamped to 0 if invSlotObjId[hoveredSlot]<=0), and when mode==1 runs `while (src != dst) { swapObj(src, src-1)/swapObj(src, src+1) }` instead of the single swap, then writes a 5th protocol byte `out.p1(mode)`. The Go path has no BankArrangeMode field anywhere (grep returns nothing), always performs the single swap, and writes only P2,P2,P2 (lines 7514-7516) with no trailing P1(mode). This is a protocol desync of CLIENTPROT_INV_BUTTOND (Java writes p2,p2,p2,p1 = 7 bytes payload; Go writes p2,p2,p2 = 6) plus missing bank-rearrange behavior. Scoped here because swapObj is the Component-unit method; the full fix lives in the Client unit.

```
Java 3038-3042: `this.out.pIsaac(81); p2(objDragInterfaceId); p2(objDragSlot); p2(hoveredSlot); p1(mode);`. Go 7513-7516: `P1Isaac(INV_BUTTOND); P2(ObjDragInterfaceID); P2(ObjDragSlot); P2(HoveredSlot)` — no P1(mode), no mode/while-reorder logic.
```

> **Verifier (confirmed, severity blocker):** VERIFIED REAL — protocol desync. Confirmed every claim against Java 01f16088 and the Go tree.\n\nJava (Client.java:3007-3042): at the obj-drag-drop commit, the caller computes `byte mode` = 1 iff `bankArrangeMode==1 && com.clientCode==206`, clamped to 0 if `invSlotObjId[hoveredSlot]<=0`. If mode==1 it runs a `while(src!=dst) swapObj(src, src±1)` shuffle; else a single `swapObj(objDragSlot, hoveredSlot)`. Then it ALWAYS writes INV_BUTTOND as `pIsaac(81); p2(objDragInterfaceId); p2(objDragSlot); p2(hoveredSlot); p1(mode)` = opcode + 7 payload bytes. swapObj (config/Component.java:447) just swaps invSlotObjId/invSlotObjCount at src<->dst.\n\nGo (client.go:7505-7517): swap math is inlined and co […]
>
> **Refined:** Go's inlined INV_BUTTOND send (client.go:7505-7517) omits the rev-244 bank-arrange mechanism entirely. Missing pieces: (1) the `BankArrangeMode` field and its server-setting case (Java setClientSetting clientCode==9, Client.java:11424) — Go's equivalent at client.go:3919-3928 handles only codes 4/5/6/8; (2) the `byte mode` computation and the `while(src!=dst) swapObj(src,src±1)` reorder loop (Java Client.java:3010-3033); and (3) the trailing `out.p1(mode)` byte after the three p2 values (Java Cl […]

### B03. SeqType.Decode eager AnimFrame delay-resolution dereferences a nil *AnimFrame at decode time (Java getFrameDuration is lazy + null-guarded)

- **Unit:** `ConfigSmall`  **Java:** `SeqType.java:77-93,118 (getFrameDuration; decode reads delay via g2 at line 118)`  **Go:** `pkg/jagex2/config/seqtype/seqtype.go:117-128`

Java-244 moved the delay==0 fallback OUT of decode into a lazy getFrameDuration(frame): it calls AnimFrame.get(frames[frame]) which returns instances[id] (possibly null), guards `if (transform != null)`, and otherwise falls back to duration 1. The Go port resolves this EAGERLY inside Decode's opcode-1 loop: `if t.Delay[i] == 0 { t.Delay[i] = animframe.Instances[t.Frames[i]].Delay }`, with NO nil guard on the AnimFrame entry. Load order makes this unsafe: animframe.Init (client.go:5943) only allocates the slice (all entries nil); seqtype.Unpack runs at client.go:6060; animframe.Unpack is ONLY called later at client.go:8781 inside the OnDemand request-completion handler (req.Archive==1), exactly mirroring Java (AnimFrame.unpack at Client.java:2443 in the async handler, after SeqType.unpack at Client.java:1891). Therefore at decode time every animframe.Instances[t.Frames[i]] is nil. Any seq frame whose wire delay is 0 (the normal case the whole getFrameDuration mechanism exists to handle) makes `.Delay` a nil-pointer dereference -> boot panic. Severity is data-dependent (only triggers if seq.dat contains a zero-delay frame) but the lazy design in Java implies zero-delay frames are routine, so this is a crash-class desync.

```
Go: `if t.Delay[i] == 0 { t.Delay[i] = animframe.Instances[t.Frames[i]].Delay }` (seqtype.go:123-124, no nil check). Java: `AnimFrame transform = AnimFrame.get(this.frames[frame]); if (transform != null) { duration = this.delay[frame] = transform.delay; }` (SeqType.java:82-85). AnimFrame.get: `if (instances == null) return null; else return instances[id];`. Load order: animframe.Unpack only at client.go:8781 (OnDemand handler), seqtype.Unpack at client.go:6060.
```

> **Verifier (confirmed, severity blocker):** I could not refute this finding; all of its load-bearing claims check out against both code sites.

DECISIVE EVIDENCE:

1. Java-244 decode does NOT resolve the zero-delay fallback. `SeqType.decode` opcode-1 loop (SeqType.java) reads `this.delay[i] = buf.g2();` and stops there — no AnimFrame touch. The fallback was relocated to the lazy getter `getFrameDuration(frame)`, which does: `AnimFrame transform = AnimFrame.get(this.frames[frame]); if (transform != null) { duration = this.delay[frame] = transform.delay; }` — null-guarded. `AnimFrame.get` returns null if `instances == null` (and returns `instances[id]`, which may itself be null). So Java never dereferences a null AnimFrame.

2. The Go p […]

### B04. Missing bankArrangeMode field + dropped INV_BUTTOND mode logic causes client→server protocol desync

- **Unit:** `client/Client#1`  **Java:** `Client.java:426,3010-3041,11425`  **Go:** `pkg/jagex2/client/client.go:7505-7517`

Java field bankArrangeMode (client.Nh, Client.java:426) is absent from the Go Client struct. It is LIVE: set by the SET_VARC-style handler at Java:11425 (clientCode 9) and read in the item-drag-drop logic at Java:3013 to compute `byte mode` (1 = bank insert-shift, 0 = simple swap). The Go drag-drop site (client.go:7505-7517) drops the entire mode computation — it always performs a simple swap — AND omits the trailing `out.p1(mode)` after the three p2 writes. The 244 server (Engine-TS) declares INV_BUTTOND as ClientGameProt(245, 81, 7): a FIXED 7-byte packet whose decoder reads g2+g2+g2+g1 (InvButtonDDecoder.ts). The Go client sends only 6 bytes (p2+p2+p2). Because opcode 81 is fixed-length 7, the server consumes one byte of the following packet as `mode`, then misframes every subsequent opcode → full inbound desync. Also, the bank arrange-by-insert behaviour is silently wrong even before desync.

```
Java 3034-3038: `this.out.pIsaac(81); this.out.p2(this.objDragInterfaceId); this.out.p2(this.objDragSlot); this.out.p2(this.hoveredSlot); this.out.p1(mode);`  Go 7513-7516: `c.Out.P1Isaac(io.CLIENTPROT_INV_BUTTOND); c.Out.P2(c.ObjDragInterfaceID); c.Out.P2(c.ObjDragSlot); c.Out.P2(c.HoveredSlot)` (no P1, no mode branch). Server: `new ClientGameProt(245, 81, 7)` + decoder `g2;g2;g2;g1`.
```

> **Verifier (confirmed, severity blocker):** Confirmed end-to-end against Java-244 client, Go client, and the actual 244 server (Engine-TS branch 244-GOSCAPE).

THREE defects, all verified:
1. FIELD MISSING: Java `bankArrangeMode` (client.Nh, Client.java:426) has no counterpart in the Go Client struct — grep for BankArrangeMode/bankArrangeMode returns zero hits.
2. SETTER DROPPED: Java's SET_VARC-style handler has `else if (clientCode == 9) { this.bankArrangeMode = value; }` (Client.java:11424-11425). The Go translation (client.go:3901-3928) ends at `if var3 == 8` and entirely omits the `var3 == 9` branch — so the value is never stored even if the field existed.
3. SEND-SITE DROPPED MODE + TRAILING BYTE: Java drag-drop (Client.java:301 […]
>
> **Refined:** Three linked defects drop the INV_BUTTOND `mode` byte and cause a client→server protocol desync against the 244 server. (1) Java field `bankArrangeMode` (Client.java:426) is absent from the Go Client struct. (2) Its setter — Java `else if (clientCode == 9) { this.bankArrangeMode = value; }` (Client.java:11424-11425) — is omitted from the Go SET_VARC-style handler, which ends at `var3 == 8` (client.go:3928). (3) The Go drag-drop site (client.go:7505-7517) drops the entire `mode` computation (Java […]

### B05. getNpcPosExtended missing 244 DAMAGE_STACK (mask 0x1) block — protocol desync on second hitmark

- **Unit:** `client/Client#10`  **Java:** `Client.java:9456-9467`  **Go:** `pkg/jagex2/client/client.go:1000-1066`

Java 244 getNpcPosExtended reads a NEW first block `if ((mask & 0x1) == 1)` that consumes 4 bytes (g1 damage, g1 damageType, g1 health, g1 totalHealth) and calls npc.hit(). The Go GetNpcPosExtended has NO 0x1 handling at all — it jumps straight from reading the mask (var7) to checking 0x2. The 244 server (Engine-TS) DOES emit this bit: Npc.ts:484-491 sets `masks |= NpcInfoProt.DAMAGE2` for the 2nd hitmark slot. When any NPC takes two simultaneous hitmarks in one tick, the server sets bit 0x1, the Go client reads zero bytes for it, every subsequent block in the npc stream is misaligned, and getNpcPos's `if arg0.Pos != psize` check (client.go:3152) panics with 'size mismatch in getnpcpos' → disconnect. The Go entity model is also missing the 4-slot hit queue this requires.

```
Go: `var7 := arg0.G1(); var8 := 0; if var7&0x2 == 2 {` (no 0x1 case). Java: `int mask = buf.g1(); if ((mask & 0x1) == 1) { // DAMAGE_STACK ... buf.g1(); buf.g1(); npc.hit(...); npc.health=buf.g1(); npc.totalHealth=buf.g1(); }`. Server Npc.ts:487 `this.masks |= NpcInfoProt.DAMAGE2;`
```

> **Verifier (confirmed, severity blocker):** VERIFIED REAL after attempting refutation. Evidence:

1. Java-244 reference confirmed verbatim (Client.java:9456-9467): getNpcPosExtended reads `if ((mask & 0x1) == 1)` DAMAGE_STACK as the FIRST block, consuming 4 bytes (g1 damage, g1 damageType, g1 health, g1 totalHealth) + npc.hit(). This is the 8-branch 244 protocol.

2. Go code (client.go:1000-1066, full function re-read) has NO 0x1 handling whatsoever: line 1004 `var7 := arg0.G1()` then line 1006 jumps straight to `if var7&0x2 == 2`. Only 7 branches exist (0x2/0x4/0x8/0x10/0x20/0x40/0x80) — the 225-lineage protocol. The repo's own audit-225/client-CG2.md:108 explicitly documents this as "all 7 bit-flag branches", confirming the Go port  […]
>
> **Refined:** Go GetNpcPosExtended (pkg/jagex2/client/client.go:1000-1066) omits the 244 DAMAGE_STACK block that Java-244 reads first: `if ((mask & 0x1) == 1)` consuming 4 bytes (g1 damage, g1 damageType, g1 health, g1 totalHealth) + npc.hit(). The Go function reads the mask (var7) then jumps directly to the 0x2 ANIM branch, implementing only the 7-branch 225-lineage protocol. The target 244 server (Engine-TS branch 244-GOSCAPE, package.json requires @2004scape/rsbuf ^244.1.0) emits this bit: Npc.ts:487 sets  […]

### B06. getPlayerExtendedInfo missing 244 DAMAGE_STACK (mask 0x400) block + early-return at 0x200 — protocol desync on second hitmark

- **Unit:** `client/Client#10`  **Java:** `Client.java:9280-9302`  **Go:** `pkg/jagex2/client/client.go:10720-10733`

Java 244 getPlayerExtendedInfo handles mask bit 0x400 (DAMAGE_STACK, 4 bytes: g1 damage, g1 type, g1 health, g1 totalHealth) AFTER the 0x200 EXACTMOVE block. The Go GetPlayerExtended2 ports the 225 form where 0x200 is the last block and uses `if arg2&0x200 != 512 { return }` (an early return), so it can neither reach nor read a 0x400 block. Player.ts:1880-1887 on the 244 server sets `masks |= PlayerInfoProt.DAMAGE2` for the 2nd hitmark slot. When a player takes two simultaneous hitmarks, the server sets 0x400, the Go reads zero bytes for it, and player-info stream parsing desyncs.

```
Go: `if arg2&0x200 != 512 { return }` then EXACTMOVE fields, then function ends — no 0x400. Java: after EXACTMOVE block `if ((mask & 0x400) == 1024) { // DAMAGE_STACK int damage = buf.g1(); int damageType = buf.g1(); player.hit(...); player.health = buf.g1(); player.totalHealth = buf.g1(); }`. Server Player.ts:1887 `this.masks |= PlayerInfoProt.DAMAGE2;`
```

> **Verifier (confirmed, severity blocker):** Verified against the authoritative 244 reference and could not refute the finding.

DECISIVE EVIDENCE:
1. Java-244 reference (Client-Java @ 01f1608, confirmed genuine 244 via `System.out.println("RS2 user client - release #" + 244)`) getPlayerExtendedInfo (Client.java:9280-9302) handles, AFTER the 0x200 EXACTMOVE block, a 0x400 DAMAGE_STACK block: `if ((mask & 0x400) == 1024) { int damage = buf.g1(); int damageType = buf.g1(); player.hit(...); player.health = buf.g1(); player.totalHealth = buf.g1(); }` — 4 bytes consumed.

2. Go GetPlayerExtended2 (pkg/jagex2/client/client.go:10609-10733) ends with the 225-lineage EXACTMOVE form: `if arg2&0x200 != 512 { return }` followed by the ForceMove fi […]
>
> **Refined:** Confirmed as written. Go GetPlayerExtended2 (client.go:10609-10733) omits the 244 DAMAGE_STACK (mask 0x400) block and uses the 225-form `if arg2&0x200 != 512 { return }` early-return, after which the function ends — so it can neither reach nor consume the 4 trailing bytes (g1 damage, g1 type, g1 health, g1 totalHealth) that Java-244 reads at Client.java:9296-9302. The mask bit reaches the function because the caller GetPlayerExtended1 (client.go:3847-3848) reads the high mask byte on 0x80. The 2 […]

### B07. INV_BUTTOND packet missing trailing p1(mode) byte + entire bank-arrange swap path unported (protocol desync)

- **Unit:** `client/Client#3`  **Java:** `Client.java:3009-3041`  **Go:** `pkg/jagex2/client/client.go:7505-7517`

Java updateGame's obj-drag drop computes a byte `mode` (mode=1 when bankArrangeMode==1 && com.clientCode==206, forced to 0 when com.invSlotObjId[hoveredSlot]<=0), performs an insert-style shift rearrange via com.swapObj when mode==1, and writes INV_BUTTOND as p2(interfaceId) p2(dragSlot) p2(hoveredSlot) p1(mode) — FOUR fields. The Go only does the simple swap (mode==0 path) and writes THREE fields, omitting the final P1(mode). With a 244 server this desyncs the client->server stream on any inventory/bank item drag. bankArrangeMode (set by a varp at Java:11425) and Component.swapObj/clientCode==206 do not exist in the Go port at all.

```
Java: `byte mode = 0; if (this.bankArrangeMode == 1 && com.clientCode == 206) mode = 1; ... this.out.p2(this.hoveredSlot); this.out.p1(mode);` — Go ends at `c.Out.P2(c.HoveredSlot)` with no P1(mode) and no mode/bankArrange logic.
```

> **Verifier (confirmed, severity blocker):** CONFIRMED on every axis; could not refute.

WIRE FORMAT (the core claim): Java-244 Client.java:3032-3041 writes INV_BUTTOND as pIsaac(81) + p2(interfaceId) + p2(dragSlot) + p2(hoveredSlot) + p1(mode) = 7 payload bytes. Go client.go:7513-7516 writes P1Isaac(81) + P2 + P2 + P2 = only 6 payload bytes, with no P1(mode). The trailing byte is omitted.

DESYNC SEVERITY VERIFIED against the actual 244 server (Engine-TS, the documented 244 transport target per memory): ClientGameProt.ts:82 declares `INV_BUTTOND = new ClientGameProt(245, 81, 7)` — a positive (fixed) length of 7. InvButtonDDecoder.ts reads exactly g2 g2 g2 g1 (=7 bytes). Because the framing is fixed-length, the server reads 7 bytes for […]
>
> **Refined:** Confirmed as written. Go client.go:7513-7516 sends INV_BUTTOND (opcode 81) with only three p2 fields (interfaceId, dragSlot, hoveredSlot) = 6 payload bytes, omitting the trailing p1(mode) the Java-244 client writes (Client.java:3041). The 244 server (Engine-TS ClientGameProt INV_BUTTOND fixed length 7; InvButtonDDecoder reads g2 g2 g2 g1) expects 7 bytes, so the missing byte misframes all subsequent client->server packets — a hard desync on any inventory/bank item drag. Additionally unported and […]

### B08. PushProjectiles omits the entity-coordinate bounds guard before updateVelocity

- **Unit:** `client/Client#7`  **Java:** `Client.java:6038-6056`  **Go:** `pkg/jagex2/client/client.go:6332-6348`

Java 244 only calls proj.updateVelocity when the target entity is inside the scene: `if (npc != null && npc.x >= 0 && npc.x < 13312 && npc.z >= 0 && npc.z < 13312)` (and the same for the player branch). The Go checks only `if var3 != nil` / `if var5 != nil`, omitting the `>= 0 && < 13312` range checks for both X and Z. For an entity at out-of-range coords this would feed bad positions into the projectile homing/velocity update (and GetHeightMapY clamps to 0 for off-grid tiles, producing a different trajectory than the Java skip). Only triggers for out-of-bounds target coordinates, hence latent.

```
Java: `if (npc != null && npc.x >= 0 && npc.x < 13312 && npc.z >= 0 && npc.z < 13312) { proj.updateVelocity(...); }`. Go: `if var3 != nil { v.UpdateVelocity(...) }` (no coord range test). Same omission in the player branch (Java `player.x >= 0 && player.x < 13312 ...` vs Go `if var5 != nil`).
```

> **Verifier (confirmed, severity blocker):** CONFIRMED REAL after attempting refutation.\n\nDecisive evidence:\n1. Java PushProjectiles (Client.java:6040, 6054) gates updateVelocity behind `npc.x >= 0 && npc.x < 13312 && npc.z >= 0 && npc.z < 13312` (and the identical player guard at 6054). Go PushProjectiles (client.go:6334 and 6346) only checks `if var3 != nil` / `if var5 != nil`. The range checks are genuinely absent — the constant 13312 appears NOWHERE in the Go codebase (grep across pkg/ returns nothing).\n\n2. Ruled out the deob arg-scramble false-positive: Java `updateVelocity(npc.z, getHeightmapY(...)-offsetY, npc.x, loopCycle)` vs Go `UpdateVelocity(GetHeightMapY(...)-OffsetY, var3.Z, var3.X, LoopCycle)`. UpdateVelocity (clien […]
>
> **Refined:** PushProjectiles omits the entity-coordinate bounds guard before UpdateVelocity. Java 244 (Client.java:6040, 6054) only calls updateVelocity when the target entity is on-grid: `if (npc != null && npc.x >= 0 && npc.x < 13312 && npc.z >= 0 && npc.z < 13312)` (and the same for the player branch). Go (client.go:6334, 6346) checks only `if var3 != nil` / `if var5 != nil`, dropping the `>= 0 && < 13312` range checks for both X and Z. Crucially, the Go GetHeightMapY (client.go:2063-2075) ALSO lacks the  […]

### B09. Opcode 192 handler missing entirely — server packet triggers spurious logout (protocol desync)

- **Unit:** `client/Client#8`  **Java:** `Client.java:7377-7382`  **Go:** `pkg/jagex2/client/client.go:9492-10563`

Java readPacket() handles opcode 192 with `if (this.ptype == 192) { this.field1264 = 255; this.ptype = -1; return true; }`. SERVERPROT_LENGTH[192]=0 (a valid zero-length packet). The Go Read() has NO branch for opcode 192 (no SERVERPROT constant value 192 exists in pkg/jagex2/io/serverprot.go, and no handler in Read()). When the server sends opcode 192, Go falls through the entire if-chain to the catch-all `signlink.ReportErrorFunc("T1 - ..."); c.Logout(); return true` at client.go:10561-10562, disconnecting the client. The entire field1264 chat-flash feature (set to 255 here, decremented in the update loop at Java:2935-2936, drawn as a yellow horizontal-line flash in DrawGame at Java:6560-6565) is absent from the Go port (grep for field1264/Field1264/chatFlash across pkg/ returns nothing).

```
Java: `if (this.ptype == 192) { this.field1264 = 255; this.ptype = -1; return true; }`  |  Go has no `== 192`/`SERVERPROT_*=192` anywhere; opcode 192 reaches `signlink.ReportErrorFunc(fmt.Sprintf("T1 - %d,%d ...")); c.Logout()` (client.go:10561-10562). SERVERPROT_LENGTH[192]==0 confirmed identical in protocol.go and Protocol.java.
```

> **Verifier (confirmed, severity blocker):** CONFIRMED after exhausting all refutation avenues.

Java side (authoritative ref git show 01f16088): Client.java:7377-7382 handles opcode 192 with `if (this.ptype == 192) { this.field1264 = 255; this.ptype = -1; return true; }`. The field1264 chat-flash feature is fully real in the 244 reference: declared at 563, reset at 2692, decremented in the update loop at 2935-2936, and drawn as a yellow translucent hline flash in DrawGame at 6560-6565. The Java catch-all (8456-8457) is exactly `SignLink.reporterror("T1 - ...") ; this.logout()`.

Go side: opcode 192 has NO handler. Verified exhaustively — serverprot.go SERVERPROT_SIZES (257 entries) and the entire pkg/ tree contain no constant with val […]
>
> **Refined:** Opcode 192 (zero-length, SERVERPROT_LENGTH[192]=0) is unhandled in Go. Java Client.java:7377-7382 consumes it (sets field1264=255 then ptype=-1, returns true). In Go's Read() (client.go:9492+) no branch matches PacketType==192, so the packet reads cleanly (size 0) and falls through to the catch-all at client.go:9561-9562 (`signlink.ReportErrorFunc("T1 - ...")` + `c.Logout()`), forcibly disconnecting the client to the title screen. The associated field1264 chat-flash feature (set 255 here, decrem […]

### B10. ApplyTransform(int) drops the null-frame guard — nil-pointer panic where Java safely no-ops

- **Unit:** `dash3d/Model#2`  **Java:** `Model.java:1155-1160`  **Go:** `pkg/jagex2/dash3d/model/model.go:1062-1063`

Java fetches the frame via AnimFrame.get(id) and returns early if it is null (frame not yet loaded over the lazy on-demand channel). The Go port indexes animframe.Instances[arg1] directly and immediately dereferences .Base with NO nil check. animframe.Instances is a fixed-size slice pre-allocated with all-nil entries (animframe.go Init: make([]*AnimFrame, capacity+1)) and only loaded frame ids are populated (Instances[id] = frame in Unpack), so a not-yet-loaded transform id yields nil and Go panics on var3.Base. This is the proven ==null-vs-!=nil / missing-guard bug class.

```
Java: `AnimFrame frame = AnimFrame.get(id); if (frame == null) { return; } AnimBase base = frame.base;`  Go: `var3 := animframe.Instances[arg1]\n\tvar4 := var3.Base` (no nil check between).
```

> **Verifier (confirmed, severity blocker):** Verified against Java ref 01f16088 (src/main/java/jagex2/dash3d/Model.java, not the /graphics/ path in the finding's javaRef — minor path typo, code matches) and the Go repo.

Java applyTransform(int id) (Model.java:1149-1169): `if (this.labelVertices == null || id == -1) return; AnimFrame frame = AnimFrame.get(id); if (frame == null) { return; } AnimBase base = frame.base; ...`. AnimFrame.get(id) (AnimFrame.java:153-159) returns instances[id] (or null if instances itself is null). So a not-yet-loaded id yields null and the `if (frame == null) return;` makes it a safe no-op.

Go ApplyTransform(arg1) (pkg/jagex2/dash3d/model/model.go:1058-1071): `if m.LabelVertices == nil || arg1 == -1 { retu […]
>
> **Refined:** Confirmed. Java applyTransform(int) guards `if (frame == null) return;` after AnimFrame.get(id); the Go port (model/model.go:1062-1063) indexes animframe.Instances[arg1] and dereferences var3.Base with no nil check. animframe.Instances is a nil-filled fixed slice populated only for on-demand-loaded ids, so a not-yet-loaded transform id is nil and Go panics where Java no-ops. The javaRef path in the finding cites jagex2/graphics/Model.java but the actual file is jagex2/dash3d/Model.java:1149-1169 […]

### B11. ApplyTransforms drops both null-frame guards (var5==null return; var6==null applyTransform fallback)

- **Unit:** `dash3d/Model#2`  **Java:** `Model.java:1180-1190`  **Go:** `pkg/jagex2/dash3d/model/model.go:1081-1083`

Java: after var5 = AnimFrame.get(arg1) it does `if (var5 == null) return;`, and after var6 = AnimFrame.get(arg2) it does `if (var6 == null) { applyTransform(arg1); return; }`. The Go port does `var5 := animframe.Instances[arg2]; var6 := animframe.Instances[arg0]; var7 := var5.Base` with neither guard. If the primary frame slot is nil, Go panics on var5.Base. If only the secondary frame slot is nil, Go does NOT fall back to applying just the primary transform (Java's behavior) and instead panics later in the second loop on var6.Length / var6.Groups. Both a crash and a wrong-behavior divergence (the secondary-missing fallback is logic, not just a crash guard).

```
Java: `if (var5 == null) { return; } AnimFrame var6 = AnimFrame.get(arg2); if (var6 == null) { this.applyTransform(arg1); return; } AnimBase var7 = var5.base;`  Go: `var5 := animframe.Instances[arg2]\n\tvar6 := animframe.Instances[arg0]\n\tvar7 := var5.Base` (no guards).
```

> **Verifier (confirmed, severity blocker):** Verified against Java 01f16088 Model.java applyTransforms(arg1,arg2,arg3) and Go model.go:1073-1116 ApplyTransforms(arg0,arg2,arg3). The Go parameter names are deob-scrambled (arg2 = Java's primary arg1; arg0 = Java's secondary arg2), but I traced each value through both bodies and the role mapping is consistent: Go's var5 := Instances[arg2] is the primary frame (Java var5 = get(arg1)), var6 := Instances[arg0] is the secondary (Java var6 = get(arg2)), var7 := var5.Base, first loop over var5, second over var6 — all match. So the value flow is correct and NOT a swapped-arg false positive.\n\nThe substantive defect holds: Java has two null-frame guards that Go omits entirely. (1) `var5 = AnimFr […]
>
> **Refined:** Confirmed accurate. Go ApplyTransforms (model.go:1073-1116) drops both Java null-frame guards from Model.java applyTransforms: (a) the `if (var5 == null) return;` after fetching the primary frame, and (b) the `if (var6 == null) { applyTransform(primary); return; }` secondary-missing fallback. var5/var6 come from animframe.Instances ([]*AnimFrame), which can hold nil entries for unloaded/sparse frame IDs, so var5.Base / var6.Length / var6.Groups can nil-panic, and the primary-only fallback behavi […]

---

## Bugs (62)

### G01. Player model LruCache sized 200 in Go vs 260 in Java-244 (stale rev-225 value)

- **Unit:** `ClientPlayer`  **Java:** `ClientPlayer.java:47`  **Go:** `pkg/jagex2/dash3d/entity/playerentity/clientplayer.go:21`

Java-244 declares `public static LruCache modelCache = new LruCache(260);`. The Go port initializes `ModelCache = datastruct.NewLruCache[*model.Model](200)`. This 200 was inherited from the rev-225 lineage when the file was created by the rename commit (f0c75f0) and never updated for the 244 deob, which raised the cap to 260. A 23% smaller cache evicts complete player-appearance composites more aggressively. This directly interacts with the 244 lazy-model barrier: GetSequencedModel falls back to `ModelCacheKey` (the last complete composite) while parts reload (lines 252-257); if that composite has been evicted, the method returns nil and the player renders absent/flickers. With more players than the cache holds, Go will drop composites Java would retain, producing observable rendering divergence (more frequent reloads / blank players) and a different eviction order.

```
Java: `public static LruCache modelCache = new LruCache(260);`  |  Go: `ModelCache = datastruct.NewLruCache[*model.Model](200)`
```

> **Verifier (confirmed, severity bug):** CONFIRMED, could not refute. Java-244 ClientPlayer.java (at src/main/java/jagex2/dash3d/ClientPlayer.java, not the path in the finding's javaRef) declares `public static LruCache modelCache = new LruCache(260);`. Go pkg/jagex2/dash3d/entity/playerentity/clientplayer.go:21 initializes `ModelCache = datastruct.NewLruCache[*model.Model](200)`.

Ruled out false positives:
1. Not a rename/deob trap: LruCache(int size) in both lineages assigns size directly to capacity AND available (Java constructor: `this.capacity = size; this.available = size;`). Go NewLruCache takes the same single size arg. So 260 vs 200 is a genuine capacity delta, not a renamed-field comparison.
2. Not parameter scramble: s […]
>
> **Refined:** Java-244 (src/main/java/jagex2/dash3d/ClientPlayer.java) declares `public static LruCache modelCache = new LruCache(260);`. The Go port at pkg/jagex2/dash3d/entity/playerentity/clientplayer.go:21 uses `NewLruCache[*model.Model](200)`. The 200 is the stale rev-225 value (225-clean PlayerEntity.java:75 = `new LruCache(200)`), carried into the 244 port (origin commit 3b5229da, survived the f0c75f0 rename) without raising the cap to 260. LruCache's constructor sets capacity = available = size in bot […]

### G02. move/step animation-cancel guard uses wrong field AND wrong comparison (Priority<=1 instead of PostanimMode==1) — missed 244 logic delta

- **Unit:** `EntityA`  **Java:** `ClientEntity.java:160-162 (move), 220-222 (step)`  **Go:** `pkg/jagex2/dash3d/entity/clententity.go:83, 137`

Java-244 cancels the primary seq with `SeqType.types[primarySeqId].postanim_mode == 1`. The Go port still carries the rev-225 expression `seqtype.Instances[e.PrimarySeqID].Priority <= 1`. This is BOTH the wrong field (Priority vs PostanimMode — verified distinct: Go SeqType.PostanimMode is opcode 10, defaults -1 then 0/2; Priority is opcode 5, defaults 5) AND the wrong comparison (<=1 vs ==1). Confirmed via 225-clean PathingEntity.java:167/228 (`priority <= 1`) vs 244 ClientEntity.java (`postanim_mode == 1`). Result: animations are cancelled on movement for the wrong set of sequences. LOGIC-DELTA-SCOPE.md:226 lists `ClientEntity move/step bodies` as an unresolved open question, so this delta is acknowledged-but-unported.

```
Java: `if (this.primarySeqId != -1 && SeqType.types[this.primarySeqId].postanim_mode == 1)`  |  Go: `if e.PrimarySeqID != -1 && seqtype.Instances[e.PrimarySeqID].Priority <= 1`
```

> **Verifier (confirmed, severity bug):** CONFIRMED real after exhausting refutation paths.

Code sites (re-read in full):
- Java-244 src/main/java/jagex2/dash3d/ClientEntity.java: move() line 174 and step() line 236 both guard with `if (this.primarySeqId != -1 && SeqType.types[this.primarySeqId].postanim_mode == 1)`.
- Go pkg/jagex2/dash3d/entity/clententity.go: Teleport (=move) line 83 and MoveAlongRoute (=step) line 137 both guard with `if e.PrimarySeqID != -1 && seqtype.Instances[e.PrimarySeqID].Priority <= 1`.

Ruled out cross-lineage rename trap (the key false-positive risk): traced both fields through Go seqtype.go and Java-244 SeqType.java decode. Go `Priority` decodes from opcode 5, default 5 (seqtype.go:23,50,142); Java-24 […]
>
> **Refined:** Java-244 ClientEntity.move (line 174) and ClientEntity.step (line 236) cancel the primary seq with `SeqType.types[primarySeqId].postanim_mode == 1`. The Go port (clententity.go:83 Teleport=move, :137 MoveAlongRoute=step) still carries the rev-225 expression `seqtype.Instances[e.PrimarySeqID].Priority <= 1`. Verified via seqtype.go and Java-244 SeqType.java that Priority (opcode 5, default 5) and PostanimMode (opcode 10, default -1→0/2) are distinct, faithfully-mapped fields — so this is both the […]

### G03. ClientEntity.hit() method and 4-slot damage arrays (damage[4]/damageType[4]/damageCycle[4]) not ported — Go keeps rev-225 scalar single-hitmark model

- **Unit:** `EntityA`  **Java:** `ClientEntity.java:91-99 (fields), 245-254 (hit method)`  **Go:** `pkg/jagex2/dash3d/entity/clententity.go:22-23 (scalar Damage/DamageType), no Hit/DamageCycle`

Java-244 ClientEntity declares `int[] damage = new int[4]`, `int[] damageType = new int[4]`, `int[] damageCycle = new int[4]`, and a `hit(int type, int amount)` method that scans for the first free slot (`damageCycle[i] <= Client.loopCycle`), writes the slot, and sets a 70-cycle expiry (`damageCycle[i] = loopCycle + 70`). The Go ClientEntity has scalar `Damage int` / `DamageType int`, NO `DamageCycle` field, and NO `Hit` method (verified: grep finds no DamageCycle / Hit / hit in pkg/). This is a genuine 244 addition (225-clean PathingEntity had scalar damage/damageType and no hit()). The 244 renderer (Client.java:6322-6340) iterates the 4 slots with per-slot cycle expiry; the Go scalar model can display only one hitmark with no timing. Java callers `player.hit(...)`/`npc.hit(...)` (Client.java:9204/9298/9463/9521) are replaced in Go by direct scalar writes (client.go:1034-1035, 10651-10652). Rendering/protocol parity gap; the missing struct shape + method belong to this file.

```
Java: `public int[] damage = new int[4]; ... public final void hit(int arg0, int arg2) { for (int var4=0;var4<4;var4++){ if (this.damageCycle[var4] <= Client.loopCycle){ this.damage[var4]=arg2; this.damageType[var4]=arg0; this.damageCycle[var4]=Client.loopCycle+70; return; }}}`  |  Go: only `Damage int` / `DamageType int`, no array, no Hit()
```

> **Verifier (confirmed, severity bug):** Confirmed real after re-reading both sites in full and ruling out false-positive causes.

Java-244 (read via git show 01f16088): src/main/java/jagex2/dash3d/ClientEntity.java declares three 4-element arrays at lines 98/101/104 (`public int[] damage = new int[4]`, `damageType`, `damageCycle`) and a `hit(int arg0, int arg2)` method at lines 264-272 that scans for the first slot where `damageCycle[var4] <= Client.loopCycle`, writes damage/damageType, and sets `damageCycle[var4] = loopCycle + 70`. (The finding's line citations are slightly off — fields are 98-104 not 91-99, hit() is 264-272 not 245-254 — but the constructs exist exactly as described.)

The Java-244 renderer (Client.java:6321-634 […]
>
> **Refined:** Confirmed. Java-244 ClientEntity (dash3d/ClientEntity.java:98-104) adds `int[] damage/damageType/damageCycle = new int[4]` and a `hit(int type, int amount)` method (lines 264-272, NOT 245-254 as cited) that finds the first slot with `damageCycle[i] <= loopCycle`, writes it, and sets a 70-cycle expiry. The Java-244 renderer (Client.java:6321-6342) loops all 4 slots, each gated by `damageCycle[i] > loopCycle`, applying per-slot positional offsets (i==1/2/3) so up to 4 hitmarks display at once. Go  […]

### G04. mouseExited drops idleCycles=0 / mouseX=-1 / mouseY=-1 (and gates everything behind InputTracking.Enabled)

- **Unit:** `GameShell`  **Java:** `GameShell.java:382-390`  **Go:** `pkg/jagex2/client/gameshell.go:198-207`

Java mouseExited does `this.idleCycles = 0; this.mouseX = -1; this.mouseY = -1;` BEFORE the InputTracking call. The Go handleMouseCross for the leave case only calls inputtracking.MouseExited(), and it early-returns when `!inputtracking.Enabled` — so in the normal (tracking-disabled) state, leaving the window performs NO state update at all. Consequences: (1) MouseX/MouseY are never reset to -1, so after the cursor leaves the window the client keeps hovering/menuing at the last in-window position (stale hover highlights, tooltips, and click-region tests on a phantom position). (2) idleCycles is not reset on exit, diverging the idle/logout timing. This is always reachable (every mouse-leave) and independent of InputTracking. mouseEntered (Java 376-380) is correctly side-effect-free, so only the exit path is wrong.

```
Java: `this.idleCycles = 0; this.mouseX = -1; this.mouseY = -1; if (InputTracking.enabled) InputTracking.mouseExited();`  vs  Go: `if !inputtracking.Enabled { return }; if e.Entered { MouseEntered() } else { MouseExited() }` — no IdleCycles/MouseX/MouseY writes.
```

> **Verifier (confirmed, severity bug):** Confirmed against both sites. Java GameShell.java:382-390 (`mouseExited`): unconditionally `this.idleCycles = 0; this.mouseX = -1; this.mouseY = -1;` THEN `if (InputTracking.enabled) InputTracking.mouseExited();`. The three field writes are outside the tracking gate. Go `handleMouseCross` (gameshell.go:198-207) early-returns when `!inputtracking.Enabled`, and even when enabled the leave branch only calls `inputtracking.MouseExited()`.

Ruled out the common false positives:
- Behavior elsewhere: grep for `MouseX = -1`/`MouseY = -1` across pkg/jagex2/client returns nothing — no other site resets these on leave. `inputtracking.MouseExited()` (inputtracking.go) only writes opcode 13 to the track […]
>
> **Refined:** handleMouseCross (gameshell.go:198-207) omits Java mouseExited's pre-gate side effects (GameShell.java:383-385): `idleCycles=0; mouseX=-1; mouseY=-1`. Because the Go handler early-returns on `!inputtracking.Enabled` (the default state) and the leave branch only calls inputtracking.MouseExited(), no reset ever happens on window-leave. Result: MouseX/MouseY stay at the last in-window position (phantom hover/menu/click-region behavior — the dominant symptom) and IdleCycles is not reset (idle-timing […]

### G05. inputtracking.KeyPressed fires TWICE per printable key (KeyRune KeyPress + CharInput both record it)

- **Unit:** `GameShell`  **Java:** `GameShell.java:428-480`  **Go:** `pkg/jagex2/client/gameshell.go:129-140,217-320`

A letter/digit generates BOTH a platform KeyPress{Key:KeyRune} and a CharInput (backend_glfw.go:230-247 / backend_webgl.go:255-266). handleCharInput calls inputtracking.KeyPressed(rune) (gameshell.go:137-139) AND handleKey calls inputtracking.KeyPressed(var3) unconditionally for the KeyRune press (gameshell.go:306-308, var3 = lowercased rune via charFor). Java has exactly one keyPressed event per physical key and records it once. When InputTracking is enabled (reachable: client.go:10113 inputtracking.SetEnabled()), every printable keystroke is recorded twice in OutBuffer, corrupting the server-facing input-tracking byte stream (also double-advances LastTime/delta accounting). KeyQueue is NOT doubled (handleKey's isSentinel gate excludes letters), so only the InputTracking record is wrong. Releases are single (no CharInput on release), so only the press doubles.

```
handleCharInput: `if inputtracking.Enabled { inputtracking.KeyPressed(var3) }` ; handleKey (Down branch): `if inputtracking.Enabled { inputtracking.KeyPressed(var3) }` — both run for the same printable key.
```

> **Verifier (confirmed, severity bug):** Confirmed and could not refute. A single printable keystroke produces TWO platform events in both backends: KeyPress{Key:KeyRune} and a separate CharInput. backend_glfw.go installCallbacks (lines 230-247) registers SetCharCallback (emits CharInput{Rune}) AND SetKeyCallback (emits KeyPress; glfwKeyToNeutral maps A-Z/0-9 to KeyRune). backend_webgl.go keydown (lines 257-268) emits both KeyPress and CharInput for a printable key. PollEvents drains all events with no de-dup; dispatchInputEvent (gameshell.go:84-99) routes KeyPress->handleKey and CharInput->handleCharInput, so both run for one keystroke.\n\nTracing the letter 'a': handleKey computes var2=awtFor(KeyRune)=0 (KeyRune is absent from aw […]
>
> **Refined:** For each printable key (letter/digit), the GLFW and WebGL backends emit BOTH a KeyPress{Key:KeyRune} and a separate CharInput event. dispatchInputEvent routes these to handleKey and handleCharInput respectively. When InputTracking is enabled (server opcode 22 -> inputtracking.SetEnabled, client.go:10113), handleKey calls inputtracking.KeyPressed(var3) unconditionally (gameshell.go:306-308, var3 = lowercased rune from charFor, var2=awtFor(KeyRune)=0 so no override fires) AND handleCharInput calls […]

### G06. getIcon NOT migrated to 244 lineage: missing outlineRgb param, zoom scaling, colored-outline pass, conditional shadow/cache

- **Unit:** `ObjType`  **Java:** `ObjType.java:473-622`  **Go:** `pkg/jagex2/config/objtype/objtype.go:346-450`

The 244 Java getIcon has signature getIcon(int outlineRgb, int count, int id) — a 3-parameter version. The Go GetIcon(arg0, arg2) is the rev-225 2-parameter version and was never migrated. Confirmed by diffing 225-clean vs 01f16088 (line 645 old vs 688 new). The 244 behavior the Go is missing: (1) cache lookup is gated by `if (outlineRgb == 0)` (Java:475) — Go always reads cache (go:347); (2) zoom is scaled `zoom = zoom*1.5` when outlineRgb==-1 and `zoom*1.04` when outlineRgb>0 (Java:540-544) — Go uses raw obj.Zoom2D (go:401-402), so the sinPitch/cosPitch fed into drawSimple differ; (3) when outlineRgb>0, a colored-outline pass paints icon.pixels = outlineRgb around the silhouette (Java:567-582) — entirely absent in Go; (4) the 0x302020 (3153952) shadow pass is gated `else if (outlineRgb == 0)` (Java:583-591) — Go runs it UNCONDITIONALLY (go:419-425); (5) `iconCache.put` is gated `if (outlineRgb == 0)` (Java:603) — Go always puts (go:436). The sole caller (Client-Java Client.java:10581) passes outline=16777215 for the selected inventory slot, so the white selection-highlight outline is never rendered by the Go port. The Go caller (client.go:3659) drops the outline arg and instead uses the 225-era DrawAlpha(128,...) highlight (client.go:3673-3675).

```
Java ObjType.java:474 `public static final Pix32 getIcon(int outlineRgb, int count, int id)` / :540 `if (outlineRgb == -1) { zoom = (int)((double)zoom*1.5D); } else if (outlineRgb > 0) { zoom = (int)((double)zoom*1.04D); }` / :567 `if (outlineRgb > 0) { ... icon.pixels[y*32+x] = outlineRgb; ... } else if (outlineRgb == 0) { ... = 0x302020; }`. Go go:346 `func GetIcon(arg0, arg2 int) *pix32.Pix32` (no outline param); go:419-425 unconditional shadow pass; go:436 unconditional `IconCache.Put`. Caller Client.java:10581 `ObjType.getIcon(outline, child.invSlotObjCount[slot], id)` with outline=16777215 when selected (Client.java:10577).
```

> **Verifier (confirmed, severity bug):** Confirmed against 01f16088. The Java-244 getIcon has a 3-parameter signature `getIcon(int outlineRgb, int count, int id)` (ObjType.java:473-622); the Go `GetIcon(arg0, arg2)` (objtype.go:346-450) is a 2-parameter function. I diffed the Go against the 225-clean Java `getIcon(int arg0, int arg2)` and they match verbatim (identical var3..var22 naming and structure), proving the Go is the un-migrated rev-225 version — not a deob rename or restructured-but-equivalent form.

All five behavioral deltas the finding lists are real, verified line-by-line:
(1) Cache lookup: Java-244 gates the iconCache.get + unlink + early-return inside `if (outlineRgb == 0)` (Java:475-486); Go always reads/early-retur […]
>
> **Refined:** getIcon was not migrated from the rev-225 2-parameter form to the 244 3-parameter form `getIcon(int outlineRgb, int count, int id)`. The Go GetIcon(arg0, arg2) is a verbatim port of 225-clean's getIcon(int arg0, int arg2). Missing 244 behavior: (1) cache get/early-return is gated by outlineRgb==0 (Go always reads cache); (2) zoom is scaled *1.5 when outlineRgb==-1 and *1.04 when outlineRgb>0 (Go uses raw Zoom2D); (3) when outlineRgb>0 a colored-outline pass paints the silhouette with outlineRgb  […]

### G07. getIcon cert overlay uses Crop (scaling) instead of 244 plotSprite (1:1 blit), and wrong recursive getIcon zoom arg

- **Unit:** `ObjType`  **Java:** `ObjType.java:593-601`  **Go:** `pkg/jagex2/config/objtype/objtype.go:426-435`

For certificate (note) items, 244 Java blits the linked item's icon over the note background via `linkedIcon.plotSprite(0, 0)` — a straight 1:1 sprite copy (ObjType.java:598; Pix32.plotSprite at Pix32.java:226). The Go calls `var20.Crop(22, 5, 22, 5)` (go:432), which is the rev-225 scaling routine (Go Crop calls Pix32.Scale, pix32.go:402). The 244 Java Pix32 has NO crop method at all — this is leftover 225 code. Additionally, 244 fetches the linked icon with `getIcon(-1, 10, obj.certlink)` (Java:514) which applies the 1.5x zoom (outlineRgb==-1 branch), whereas the Go fetches `GetIcon(var4.CertLink, 10)` (go:427) with no zoom scaling. The Go's stale comment at go:349-351 cites `Pix32.java:302-353` for a crop try/catch that does not exist in the 244 Java Pix32 (that range is transPlotSprite in 244).

```
Java ObjType.java:598 `linkedIcon.plotSprite(0, 0);` and :514 `linkedIcon = getIcon(-1, 10, obj.certlink);`. Go go:432 `var20.Crop(22, 5, 22, 5)` and go:427 `var20 := GetIcon(var4.CertLink, 10)`. 244 Pix32 method list has plotSprite (226) but no `crop`; Go Crop (pix32.go:348) ends `p.Scale(...)` (pix32.go:402).
```

> **Verifier (confirmed, severity bug):** CONFIRMED after attempting refutation. The Go GetIcon at pkg/jagex2/config/objtype/objtype.go:346-450 is a bug-for-bug port of the 225-clean Java getIcon(int arg0, int arg2) (verified via `git show 225-clean:...ObjType.java`: signature line 390, `getIcon(var4.certlink, 10)` line 457, `var20.crop(22,5,22,5)` line 462), NOT the 244 getIcon(int outlineRgb, int count, int id).\n\nThe 244 Java (01f16088 ObjType.java:474 signature, :514 recursive call, :593-601 cert block) differs in three verified ways:\n1. Cert/note overlay: 244 uses `linkedIcon.plotSprite(0, 0)` (ObjType.java:598) — a 1:1 blit (Pix32.java:226 plotSprite calls plot() with no scale ratios). Go uses `var20.Crop(22, 5, 22, 5)` (obj […]
>
> **Refined:** Go GetIcon (objtype.go:346-450) is a faithful port of 225-clean getIcon(arg0=id, arg2=count), not the 244 getIcon(outlineRgb, count, id). Three 244 divergences: (1) cert overlay — 244 ObjType.java:598 `linkedIcon.plotSprite(0,0)` is a 1:1 blit (Pix32.java:226), but Go objtype.go:432 `var20.Crop(22,5,22,5)` scales via Pix32.Scale (pix32.go:402); 244 Pix32 has no crop method. (2) 244 fetches the linked icon with `getIcon(-1, 10, certlink)` (ObjType.java:514) applying the outlineRgb==-1 1.5x zoom ( […]

### G08. iconCache capacity is 200 (225 value) but 244 specifies 100

- **Unit:** `ObjType`  **Java:** `ObjType.java:87-88`  **Go:** `pkg/jagex2/config/objtype/objtype.go:23`

244 Java declares `iconCache = new LruCache(100)`. The Go declares `IconCache = datastruct.NewLruCache[*pix32.Pix32](200)` — the rev-225 capacity. Confirmed by the 225→244 diff (225 had 200, 244 changed to 100). This doubles the icon cache size relative to 244, changing eviction timing and resident memory. modelCache(50) matches Java correctly.

```
Java ObjType.java:88 `public static LruCache iconCache = new LruCache(100);` (225-clean line 85 was `new LruCache(200)`). Go go:23 `IconCache  = datastruct.NewLruCache[*pix32.Pix32](200)`.
```

> **Verifier (confirmed, severity bug):** Confirmed against both cited sites. Java 244 (git show 01f16088:src/main/java/jagex2/config/ObjType.java line 88): `public static LruCache iconCache = new LruCache(100);`. Go (pkg/jagex2/config/objtype/objtype.go:23): `IconCache = datastruct.NewLruCache[*pix32.Pix32](200)`. The capacity is 200 in Go vs 100 in Java 244.

Ruled out all false-positive causes:
- Not a deob arg-scramble: LruCache's only constructor parameter is `size`, which is assigned directly to `capacity` (Java LruCache.java:29-30 `this.capacity = size;`; Go NewLruCache[T any](size int32)). The literal 100/200 is the exact same field in both languages.
- Not a cross-lineage rename inversion: the sibling modelCache ports corre […]
>
> **Refined:** 244 Java declares `iconCache = new LruCache(100)` (ObjType.java:88, git 01f16088). Go declares `IconCache = datastruct.NewLruCache[*pix32.Pix32](200)` (objtype.go:23) — the carried-over rev-225 capacity. LruCache's constructor arg is assigned directly to `capacity` in both languages, so this is a direct 100-vs-200 mismatch. The sibling modelCache(50) matches Java correctly, confirming the intended pattern is a literal copy. Result: the icon cache holds double the entries vs 244, changing LRU evi […]

### G09. Go-extra Crop/Scale methods have no Java-244 source; live caller diverges from 244 (uses Crop instead of plotSprite)

- **Unit:** `Pix32`  **Java:** `ObjType.java:593-601 (244); Pix32.java:302-355,357-379 (225-clean only)`  **Go:** `pkg/jagex2/graphics/pix32/pix32.go:348-431 (Crop+Scale); caller pkg/jagex2/config/objtype/objtype.go:432`

Java 244 Pix32 has NO crop() or scale() methods — they exist only in the 225-clean lineage (225 Pix32.java:302-379) and were dropped in the 244 deob. The Go port carries both over (Crop at pix32.go:348, Scale at pix32.go:405). Crop is live-called by objtype.go:432 `var20.Crop(22, 5, 22, 5)` for the certlink icon overlay, but Java 244 ObjType.getIcon does this overlay with `linkedIcon.owi=32; linkedIcon.ohi=32; linkedIcon.plotSprite(0,0); ...` (ObjType.java:593-601) — a plain plotSprite, NOT a scaled crop. The Go inventory/icon path therefore renders the cert overlay via the wrong (225) algorithm. The Pix32-side defect is the presence of the no-244-source methods; the actual visible wrong-render is in the ObjType caller (out of this unit) but is enabled by these carried-over methods.

```
Java 244 ObjType.java:593-601: `linkedIcon.owi = 32; linkedIcon.ohi = 32; linkedIcon.plotSprite(0, 0); linkedIcon.owi = w; linkedIcon.ohi = h;`  vs Go objtype.go:430-432: `var20.OWi = 32; var20.OHi = 32; var20.Crop(22, 5, 22, 5)`. Java 244 Pix32 (01f16088) method list ends at copyPixelsMasked; grep for `void crop|void scale` in 244 returns nothing (only 225-clean has them).
```

> **Verifier (confirmed, severity bug):** Verified against Java 244 (01f16088) and 225-clean. (1) Java 244 Pix32 has NO crop/scale methods — its method list ends at copyPixelsMasked (line 496) and a token grep for crop|scale returns 0. (2) crop (Pix32.java:302) and scale (Pix32.java:357) exist only in the 225-clean lineage. (3) The Go port carries both over: Crop at pix32.go:348 and Scale at pix32.go:405; the sole caller of Pix32.Scale is Crop itself (pix32.go:402), and the sole live caller of Crop is objtype.go:432 (plus one unit test). (4) The live caller diverges: Java 244 ObjType.getIcon (lines 593-601) does the certtemplate overlay via `linkedIcon.owi=32; linkedIcon.ohi=32; linkedIcon.plotSprite(0,0); ...`, while Go objtype.go: […]
>
> **Refined:** Go Pix32 carries over Crop (pix32.go:348) and Scale (pix32.go:405), which have no Java-244 source — they exist only in the 225-clean lineage (225-clean Pix32.java:302 crop, :357 scale) and were dropped in the 244 deob (244 Pix32 method list ends at copyPixelsMasked, line 496). Crop's only live caller is objtype.go:432 `var20.Crop(22, 5, 22, 5)` in the certtemplate icon-overlay block; Scale's only caller is Crop itself. Java 244 ObjType.getIcon (Client-Java 01f16088 ObjType.java:593-601) performs […]

### G10. DrawStringTaggable drops the @str@ strikeout line entirely (missing strikeout field + final hline)

- **Unit:** `PixFont`  **Java:** `PixFont.java:173-202`  **Go:** `pkg/jagex2/graphics/pixfont/pixfont.go:225-247`

Java drawStringTag sets this.strikeout=false at entry, saves var7=arg1 (start x), and after the glyph loop, if this.strikeout, draws Pix2D.hline(8388608, (int)((double)this.height*0.7D)+var8, arg1-var7, var7) — a dark-red strikethrough across the rendered text. The Go PixFont struct has no Strikeout field, DrawStringTaggable saves no start-x, and never emits the hline. So content containing an @str@ tag (a real, reachable game-content escape; DrawStringTaggable is the primary interface/chat text renderer) renders without the strikethrough. Combined with finding #2, the @str@ tag instead silently turns the text black.

```
Java: this.strikeout=false; int var7=arg1; ... if(this.strikeout){Pix2D.hline(8388608,(int)((double)this.height*0.7D)+var8,arg1-var7,var7);}  | Go DrawStringTaggable has neither a strikeout flag, a var7 save, nor any pix2d.HLine call.
```

> **Verifier (confirmed, severity bug):** Verified against Java 01f16088:src/main/java/jagex2/graphics/PixFont.java and Go pkg/jagex2/graphics/pixfont/pixfont.go.

CONFIRMED FACTS:
1. Java declares `public boolean strikeout = false;` (PixFont.java:37). Java drawStringTag (line 174) sets `this.strikeout = false;` at entry, saves `int var7 = arg1;` (original start x), and after the glyph loop emits `if (this.strikeout) { Pix2D.hline(8388608, (int)((double)this.height*0.7D)+var8, arg1-var7, var7); }` (lines 199-200) — a dark-red (0x800000) strikethrough across the rendered text width.
2. The flag is genuinely functional, not a deob dead-write: evaluateTag sets `this.strikeout = true;` for the "str" tag (line 273). This is a real engine […]
>
> **Refined:** Confirmed as written. Java PixFont.drawStringTag (PixFont.java:37,174-200) maintains a `strikeout` boolean (set false at entry, set true by evaluateTag's "str" branch at line 273) and, after the glyph loop, emits a dark-red strikethrough via Pix2D.hline(8388608, (int)(height*0.7)+var8, arg1-var7, var7) where var7 is the saved start x. Go's PixFont struct has no Strikeout field, DrawStringTaggable (pixfont.go:225-247) never saves the start x and never calls pix2d.HLine (which exists with a matchi […]

### G11. EvaluateTag default returns 0 (black) and callers assign it unconditionally; Java returns -1 and guards with if(!=-1)

- **Unit:** `PixFont`  **Java:** `PixFont.java:183-186,214-217,235-277`  **Go:** `pkg/jagex2/graphics/pixfont/pixfont.go:234,267,285-324`

Java evaluateTag returns -1 for any unrecognized 3-char tag (and for "str", which only sets the strikeout side-effect). Both Java callers do `int v = evaluateTag(...); if (v != -1) { color = v; }` — they leave the current color unchanged on an unknown tag. Go EvaluateTag's default case returns 0 (black), and DrawStringTaggable (line 234: `arg5 = p.EvaluateTag(...)`) and DrawStringTooltip (line 267: `arg4 = p.EvaluateTag(...)`) assign the result UNCONDITIONALLY with no -1 guard. Result: any unrecognized @xxx@ tag — including @str@ — sets the text color to black instead of preserving the prior color. This is wrong rendering for any tagged string whose tag is not in the recognized set.

```
Java: int var10=this.evaluateTag(arg5.substring(var9+1,var9+4)); if(var10!=-1){arg0=var10;}  | Go: arg5 = p.EvaluateTag(string(runes[i+1 : i+4]))  (no `!= -1` / sentinel check). Go EvaluateTag default: `default: return 0`.
```

> **Verifier (confirmed, severity bug):** Confirmed against Java 01f16088 (PixFont.java) and the Go port. Java evaluateTag (PixFont.java:236-277) returns -1 for any unrecognized 3-char tag, and for "str" it sets this.strikeout=true as a side effect then still falls through to `return -1`. Both and only Java callers — drawStringTag (PixFont.java:183-185) and drawStringAntiMacro (PixFont.java:214-216) — guard the reassignment: `int v = this.evaluateTag(...); if (v != -1) { color = v; }`, so the active color is left unchanged on an unknown tag.

Go EvaluateTag (pixfont.go:285-324) has a `default: return 0` (black, identical to the "bla" case) and no -1 sentinel. Its two callers assign unconditionally with no guard: DrawStringTaggable l […]
>
> **Refined:** Go EvaluateTag (pkg/jagex2/graphics/pixfont/pixfont.go:285-324) returns 0 (black) from its default case for any unrecognized 3-char @xxx@ tag, whereas Java evaluateTag (PixFont.java:236-277) returns -1. Both Java callers (drawStringTag PixFont.java:183-185; drawStringAntiMacro PixFont.java:214-216) guard the color reassignment with `if (v != -1)`, preserving the current color on an unknown tag. The two Go callers — DrawStringTaggable (line 234) and DrawStringTooltip (line 267) — assign the Evalu […]

### G12. DrawStringTooltip / DrawStringTaggable jitter uses Go math/rand, not java.util.Random — anti-macro pattern diverges

- **Unit:** `PixFont`  **Java:** `PixFont.java:204-233`  **Go:** `pkg/jagex2/graphics/pixfont/pixfont.go:252-283`

Java drawStringAntiMacro seeds this.rand.setSeed((long)arg1) (java.util.Random, a specific 48-bit LCG) and derives both the per-call base alpha var8=(rand.nextInt()&0x1F)+192 and the per-char +1 horizontal jitter (rand.nextInt()&0x3)==0. The Go port uses math/rand (rand.New(rand.NewSource(int64(arg0))) / Seed), a completely different PRNG. Identical seeds therefore produce a different sequence, so the rendered jitter/alpha pattern of anti-macro tooltips does not match the Java client. (The &0x1F and &0x3 masks read only low bits, so Go's always-non-negative Int() does not by itself flip those, but the underlying stream is unrelated.) For exact parity a java.util.Random LCG port is required.

```
Java: this.rand.setSeed((long)arg1); int var8=(this.rand.nextInt()&0x1F)+192; ... if((this.rand.nextInt()&0x3)==0){arg2++;}  | Go: p.Random=rand.New(rand.NewSource(int64(arg0))) / p.Random.Seed(int64(arg0)); var8:=(p.Random.Int()&0x1F)+192; ... if p.Random.Int()&0x3==0 {arg6++}
```

> **Verifier (confirmed, severity bug):** Confirmed real and could not refute it. Java PixFont.java:34 declares `public Random rand = new Random()` (java.util.Random — a 48-bit LCG with multiplier 0x5DEECE66D, increment 0xB). drawStringAntiMacro (PixFont.java:204-233) does `this.rand.setSeed((long) arg1)`, then `int var8 = (this.rand.nextInt() & 0x1F) + 192` for the per-call base alpha (192-223), and `if ((this.rand.nextInt() & 0x3) == 0) arg2++` for the per-char +1 horizontal jitter. The Go port (pixfont.go:46,252-283) backs `p.Random` with `math/rand` (`rand.New(rand.NewSource(int64(arg0)))` / `Seed(int64(arg0))`), whose default source is an Additive Lagged Fibonacci Generator — a completely different algorithm from java.util.Rand […]
>
> **Refined:** PixFont anti-macro jitter PRNG mismatch: Java uses java.util.Random (48-bit LCG, PixFont.java:34,204-233) seeded by setSeed((long)arg1); the Go port (pixfont.go:46,252-283) uses math/rand (ALFG source). Same seed yields an unrelated stream, so per-call base alpha var8=(nextInt()&0x1F)+192 and per-char +1 jitter (nextInt()&0x3)==0 differ from the Java client. No java.util.Random LCG port exists in the Go repo; exact parity requires porting one. Low visual-impact (anti-macro tooltip jitter still r […]

### G13. ReportErrorFunc omits the ':' -> '_' replacement Java does first

- **Unit:** `SignLink+Midi`  **Java:** `SignLink.java:346-353`  **Go:** `pkg/jagex2/client/sign/signlink/signlink.go:421-433`

Java reporterror() sanitizes the error string with FOUR replacements in order: ':' then '@' then '&' then '#' (lines 347-350). The Go ReportErrorFunc only performs three: '@', '&', '#'. The ':' -> '_' replacement is missing entirely, so any error message containing a colon (and these messages frequently do — e.g. the glcfb/T1 messages at client.go:8627/10561 use comma/space but other call sites like client.go:9517 forward arbitrary exception strings that often contain ':') is sent to the reporterror CGI with a raw colon instead of '_'. ReportErrorFunc is live (called from ~12 sites in client.go), so this is observable wrong query-string output, not dead code.

```
Java: String safe = err.replace(':', '_'); safe = safe.replace('@','_'); safe = safe.replace('&','_'); safe = safe.replace('#','_');  |  Go: var3 := strings.ReplaceAll(e, "@", "_"); var4 := strings.ReplaceAll(var3, "&", "_"); var5 := strings.ReplaceAll(var4, "#", "_")  — no ':' step
```

> **Verifier (confirmed, severity bug):** Confirmed real and not refutable. Java SignLink.reporterror (01f16088 SignLink.java:347-350) sanitizes the error string with FOUR ordered char replacements: `':' -> '_'`, then `'@'`, `'&'`, `'#'`. Go ReportErrorFunc (signlink.go:421-423) performs only the last three (`@`, `&`, `#`); the leading `':' -> '_'` step is entirely missing.

Ruled out every common false-positive cause:
- Not done elsewhere: OpenURL (signlink.go:318-332) passes arg0 through verbatim with zero string sanitization; no upstream or downstream colon replacement compensates.
- Not deob-artifact / dead code: ReportErrorFunc is live, called from ~10 sites in client.go (3154, 3160, 3274, 7933, 7939, 8422, 8627, 9517, 10561).  […]
>
> **Refined:** Java SignLink.reporterror (SignLink.java:347-350) applies four ordered char replacements to the error string before building the reporterror CGI query: ':'→'_', '@'→'_', '&'→'_', '#'→'_'. Go ReportErrorFunc (pkg/jagex2/client/sign/signlink/signlink.go:421-423) performs only the last three and omits the leading ':'→'_'. OpenURL does no compensating sanitization. ReportErrorFunc is live (~10 client.go call sites); several pass colon-containing messages by construction (e.g. client.go:3154 and 7933 […]

### G14. reporterror CGI URL hardcodes version 225, Java-244 uses 244

- **Unit:** `SignLink+Midi`  **Java:** `SignLink.java:352`  **Go:** `pkg/jagex2/client/sign/signlink/signlink.go:433`

Java-244 builds the report URL as "reporterror" + 244 + ".cgi?..." (the literal 244, matching clientversion=244 at line 53). The Go port still builds "reporterror" + strconv.Itoa(225) + ".cgi?...". This is a leftover from the rev-225 lineage; the request path desyncs from the 244 server endpoint. ReportErrorFunc is actively called from many client.go sites, so the wrong endpoint is hit at runtime. (The reporterror.cgi host/feature is itself a signed-applet artifact under the deviation registry, but the embedded version number is a concrete 225-vs-244 value mismatch independent of the transport relaxation.)

```
Java: openurl("reporterror" + 244 + ".cgi?error=" + errorname + " " + safe)  |  Go: OpenURL("reporterror" + strconv.Itoa(225) + ".cgi?error=" + ErrorName + " " + var5)
```

> **Verifier (confirmed, severity bug):** Verified against both sources. Java-244 SignLink.java:352 (via git show 01f16088) builds the URL as "reporterror" + 244 + ".cgi?error=" + errorname + " " + safe, with the literal 244 matching clientversion = 244 (line 53). Go pkg/jagex2/client/sign/signlink/signlink.go:433 builds "reporterror" + strconv.Itoa(225) + ".cgi?error=" + ErrorName + " " + var5, hardcoding 225. The Go package also still declares ClientVersion int = 225 (line 71) where Java-244 has 244.

Ruled out the false-positive paths:
- Not a cross-lineage rename or restructure: the values are plain integer literals embedded in the same URL string in the same function; the @/&/# -> _ sanitization and readLine/close subsumption a […]
>
> **Refined:** In pkg/jagex2/client/sign/signlink/signlink.go, the reporterror URL hardcodes the rev-225 version: line 433 builds "reporterror" + strconv.Itoa(225) + ".cgi?...", and line 71 declares ClientVersion int = 225. Java-244 (SignLink.java:53,352) uses clientversion = 244 and "reporterror" + 244 + ".cgi?...". rev-244 scope doc LOGIC-DELTA-SCOPE.md:168 explicitly calls for "clientversion 244, reporterror244.cgi". Both the URL literal (line 433) and the ClientVersion global (line 71) should be 244. Real- […]

### G15. Friend-list arrays sized 100 instead of Java's 200 (and matching cap also halved)

- **Unit:** `client/Client#1`  **Java:** `Client.java:515,671,710,7407`  **Go:** `pkg/jagex2/client/client.go:607,565,638,9742`

Java allocates friendName37=new long[200], friendWorld=new int[200], friendName=new String[200], and the friend-add handler guards with `friendCount < 200` (Client.java:7407). The Go NewClient allocates all three at length 100 (FriendName37/FriendWorld/FriendName) and the opcode-70 handler guards with `c.FriendCount < 100` (client.go:9742). The size and the bound were changed together so there is no out-of-bounds crash, but the Go client silently caps the friend list at 100 entries whereas Java permits 200. A members account with 101–200 friends would have friends 101+ silently dropped from the in-client list/sort/display.

```
Java: `friendName37 = new long[200]` / `friendWorld = new int[200]` / `friendName = new String[200]`; `if (displayName != null && this.friendCount < 200)`.  Go: `FriendName37: make([]int64, 100)`, `FriendWorld: make([]int, 100)`, `FriendName: make([]string, 100)`; `if !matched && c.FriendCount < 100 {`.
```

> **Verifier (confirmed, severity bug):** VERIFIED against both lineages. Java-244 (01f16088) Client.java declares friendName37=new long[200] (515), friendWorld=new int[200] (671), friendName=new String[200] (710), and the opcode-70 UPDATE_FRIENDLIST handler guards with `friendCount < 200` (7407). Go NewClient allocates all three at length 100 (client.go:565/607/638) and the SERVERPROT_UPDATE_FRIENDLIST handler guards `c.FriendCount < 100` (client.go:9742). Confirmed verbatim at every cited line.

I tried to refute this as a cross-lineage rename/restructure false positive. It is not: the Go code is a faithful port of 225-clean (deob/client.java), which independently uses friendWorld=new int[100] (170), friendName37=new long[100] (32 […]
>
> **Refined:** Friend-list arrays and bounds match 225-clean (length 100, guards <100/>=100) but Java-244 doubled them to 200 with a tiered free(100)/members(200) cap. Three sites need updating to reach 244 parity: (1) array allocations FriendWorld/FriendName37/FriendName to length 200 (client.go:565/607/638); (2) opcode-70 UPDATE_FRIENDLIST handler guard to `< 200` (client.go:9742); (3) AddFriend (client.go:7147) to the tiered cap — `FriendCount >= 100 && MembersAccount != 1` else `FriendCount >= 200`, with m […]

### G16. Missing warnMembersInNonMembers field — packet-44 g1 dropped, members-warning welcome text + clientCode selection wrong

- **Unit:** `client/Client#1`  **Java:** `Client.java:881,7281,7288,11601-11634`  **Go:** `pkg/jagex2/client/client.go:10138-10157,5412-5449`

Java field warnMembersInNonMembers (client.rf, Client.java:881) is absent in Go. It is read as the 5th value of LAST_LOGIN_INFO (opcode 44): Java:7281 `warnMembersInNonMembers = in.g1()`. The Go packet-44 handler (client.go:10138-10142) reads only g4+g2+g1+g2 = 9 bytes and never reads the trailing g1. Packet 44 is fixed-length 10 (SERVERPROT_LENGTH[44]=10) and the Go framing resets In.Pos=0 per packet (client.go:9570), so this does NOT desync the stream, but it has two behavioural effects: (1) the clientCode selection at Go:10147 only checks `DaysSinceRecoveriesChanged != 201`, dropping Java's `|| warnMembersInNonMembers == 1` (Java:7288); (2) the welcome-screen text for clientCodes 652/653/654 (Java:11601/11622/11634) that shows the 'This is a non-members world… member benefits unavailable' warning is unreachable — Go (client.go:5415/5433/5443) always sets Text="" for DaysSinceRecoveriesChanged==201.

```
Java 7281: `this.warnMembersInNonMembers = this.in.g1();`  Java 7288: `if (this.daysSinceRecoveriesChanged != 201 || this.warnMembersInNonMembers == 1)`.  Go 10142: `c.UnreadMessages = c.In.G2()` is the last read (no trailing G1); Go 10147: `if c.DaysSinceRecoveriesChanged != 201 {`. Go 5414-5415: `case 201: arg1.Text = ""`.
```

> **Verifier (confirmed, severity bug):** Verified all four claims against Java (01f16088) and Go. (1) Java field warnMembersInNonMembers (client.rf, Client.java:881) is genuinely absent in Go — grep for warnMembers/MembersInNon/NonMembers across pkg/ returns zero hits under any name. (2) Java packet-44 reads 5 values: g4+g2+g1+g2+g1 (Client.java:7277-7281); Go (client.go:10139-10142) reads only g4+g2+g1+g2 = 9 bytes, omitting the trailing g1. SERVERPROT_LENGTH[44] = 10 in both Java and Go tables (confirmed index 44 = 10), and Go framing sets In.Pos=0 and reads exactly PacketSize bytes per packet (client.go:9566-9570), so the unread byte does NOT desync the stream — the finding correctly states this. (3) clientCode selection: Java:7 […]
>
> **Refined:** Confirmed as written. Go is missing the warnMembersInNonMembers field (Java client.rf, Client.java:881). The Go packet-44 handler (client.go:10139-10142) reads g4+g2+g1+g2 = 9 of 10 bytes and never reads the trailing g1 that Java assigns to warnMembersInNonMembers (Client.java:7281). Because packet 44 is fixed-length 10 and Go framing resets In.Pos=0 and reads exactly PacketSize bytes per packet (client.go:9566-9570), the dropped read does NOT desync the stream. Two behavioral defects result: (1 […]

### G17. Live deob field field1264 (welcome/disconnect screen-flash) not ported

- **Unit:** `client/Client#1`  **Java:** `Client.java:563,2935-2936,6560-6565,7378`  **Go:** `pkg/jagex2/client/client.go (absent)`

Java field field1264 (client.lc, Client.java:563) is a LIVE field: set to 255 by inbound opcode 192 (Java:7378), decremented by 2 each game cycle (Java:2935-2936), and drives a yellow hlineTrans screen-flash overlay near the viewport bottom while >0 (Java:6560-6565). The Go port has no such field and no opcode-192 handler (no `PacketType == 192` branch, no SERVERPROT constant 192). The packet-192 handler and draw/update sites are outside the field-chunk line range, but the field itself is a chunk-1 omission; the net effect is the disconnect/welcome flash effect is silently dropped. Latent because reachability depends on the server sending opcode 192.

```
Java 7378: `this.field1264 = 255;`  Java 6560-6565: `if (this.field1264 > 0) { int offset = 302 - (int) Math.abs(Math.sin((double) this.field1264 / 10.0D) * 10.0D); ... Pix2D.hlineTrans(offset + i, w, 16776960, 256 - w / 2, this.field1264); }`. No counterpart in client.go.
```

> **Verifier (confirmed, severity bug):** Confirmed across both reference clients and the Go port. Java: field1264 (client.lc) is live — declared Client.java:563, decremented by 2 per cycle in updateGame() (2935-2936), drives the yellow Pix2D.hlineTrans flash overlay while >0 (6560-6565), and set to 255 by the ptype==192 handler (7378). Client-TS mirrors all four sites (field1264, -=2, drawHorizontalLineAlpha, ptype==192 -> 255), proving it is a faithful feature, not a deob artifact. Go port: grep finds no field1264, no opcode-192 branch among the 58 PacketType== handlers, and pix2d has only opaque HLine (no translucent hline variant). SERVERPROT_SIZES[192]=0 in Go matches Java SERVERPROT_LENGTH[192]=0, so the packet is correctly re […]
>
> **Refined:** Java field field1264 (client.lc, Client.java:563) is a LIVE field: set to 255 by inbound opcode 192 (Client.java:7378, zero-payload), decremented by 2 each game cycle (updateGame, 2935-2936), and drives a yellow Pix2D.hlineTrans translucent screen-flash overlay near the viewport bottom while >0 (6560-6565). Both reference clients implement it (Client-TS Client.ts:500/1496/1735-1736/5327-5332/6390-6391). The Go port has no field1264, no opcode-192 branch in the readPacket dispatcher, and pix2d la […]

### G18. Player/NPC ANIM block (mask 0x2) ported from 225, not 244 — missing duplicatebehavior branch, wrong priority test, missing preanimRouteLength write

- **Unit:** `client/Client#10`  **Java:** `Client.java:9153-9181 (player), 9468-9498 (npc)`  **Go:** `pkg/jagex2/client/client.go:10620-10636 (player), 1006-1022 (npc)`

The Go ANIM handling is byte-for-byte the 225 form (verified vs 225-clean deob/client.java:10460-10465 / 1565). 244 rewrote it: (1) adds a first branch `if (primarySeqId == seqId && seqId != -1)` reading SeqType.duplicatebehavior (replaceMode): mode 1 restarts frame/cycle/delay, mode 2 only zeroes loop; (2) the else-branch uses `priority >= priority` (>=) whereas Go uses `priority > priority || priority == 0` (so equal non-zero priorities replace in 244 but NOT in Go); (3) the else-branch writes `player.preanimRouteLength = player.routeLength` (Go has no such write and the Go entity has no PreanimRouteLength field at all). Byte width is unchanged (g2+g1) so this is wrong behavior, not a desync. The seqtype.DuplicateBehavior field already exists in Go (config/seqtype) but is unused here.

```
Go: `if var6 == -1 || arg4.PrimarySeqID == -1 || seqtype.Instances[var6].Priority > seqtype.Instances[arg4.PrimarySeqID].Priority || seqtype.Instances[arg4.PrimarySeqID].Priority == 0 {`. Java 244: `int replaceMode = SeqType.types[seqId].duplicatebehavior; if (replaceMode == 1) {...} else if (replaceMode == 2) {...} } else if (seqId == -1 || player.primarySeqId == -1 || SeqType.types[seqId].priority >= SeqType.types[player.primarySeqId].priority) { ... player.preanimRouteLength = player.routeLength; }`
```

> **Verifier (confirmed, severity bug):** Verified all three sub-claims against Java 244 (01f16088) and 225-clean (deob/client.java), and confirmed the Go code is a byte-for-byte 225 port.

SITE 1 — player, Go pkg/jagex2/client/client.go:10620-10636 (GetPlayerExtended2). Go condition: `if var6 == -1 || arg4.PrimarySeqID == -1 || seqtype.Instances[var6].Priority > seqtype.Instances[arg4.PrimarySeqID].Priority || seqtype.Instances[arg4.PrimarySeqID].Priority == 0`. This is IDENTICAL to 225-clean (deob/client.java:10463-10470: `priority > priority || priority == 0`).

SITE 2 — npc, Go client.go:1006-1022 (GetNpcPosExtended). Same form, matches 225-clean (1565 area).

Java 244 (Client.java:9159-9178 player, 9479-9498 npc) was rewritten: […]
>
> **Refined:** Confirmed as written. The Go ANIM block (mask 0x2) for player (client.go:10620-10636) and npc (client.go:1006-1022) is a byte-for-byte 225-clean port. Java 244 (Client.java:9159-9178 / 9479-9498) rewrote it with: (1) a first branch on SeqType.duplicatebehavior (mode 1 = restart frame/cycle/delay/loop, mode 2 = zero loop only) when the same non-(-1) seq re-arrives; (2) the else-branch priority test changed from `> || ==0` (225/Go) to `>=` (244), so equal non-zero priorities now replace; (3) the e […]

### G19. Player/NPC DAMAGE block (mask 0x10): wrong combatCycle constant (+400 vs 244's +300) and scalar damage write instead of hit() queue

- **Unit:** `client/Client#10`  **Java:** `Client.java:9203-9209 (player), 9520-9528 (npc)`  **Go:** `pkg/jagex2/client/client.go:10650-10655 (player), 1033-1038 (npc)`

244 DAMAGE block calls `player.hit(damageType, damage)` (4-slot damage queue, ClientEntity.java:264-273) and sets `combatCycle = loopCycle + 300`. The Go ports the 225 form: directly assigns `arg4.Damage`/`arg4.DamageType` (scalar fields, the 225 model) and uses `LoopCycle + 400`. Byte width matches (g1x4), so this is wrong behavior: combat-bar timeout is 100 cycles too long, and only one hitsplat is recorded/displayed instead of the queued multi-splat the 244 entity model supports. Verified the Go entity lacks the Damage[]/DamageType[]/DamageCycle[] arrays.

```
Go: `arg4.Damage = arg3.G1(); arg4.DamageType = arg3.G1(); arg4.CombatCycle = clientextras.LoopCycle + 400; arg4.Health = arg3.G1(); arg4.TotalHealth = arg3.G1()`. Java 244: `player.hit(damageType, damage); player.combatCycle = loopCycle + 300; player.health = buf.g1(); player.totalHealth = buf.g1();`
```

> **Verifier (confirmed, severity bug):** Verified both cited sites against Java-244 (commit 01f16088).

CONFIRMED (decisive evidence):
- Java-244 player DAMAGE block (Client.java:9203-9209) and npc DAMAGE block (9520-9528) both call `entity.hit(damageType, damage)` and set `combatCycle = loopCycle + 300`. The Go (client.go:10650-10655 player, 1033-1038 npc) instead does scalar `arg4.Damage = G1(); arg4.DamageType = G1(); arg4.CombatCycle = LoopCycle + 400`.
- Java-244 ClientEntity (dash3d/ClientEntity.java:98-107, 264-273) defines `int[] damage = new int[4]`, `int[] damageType = new int[4]`, `int[] damageCycle = new int[4]`, and `hit()` which scans the 4 slots, writing into the first free one with `damageCycle[slot] = loopCycle + 7 […]
>
> **Refined:** 244 DAMAGE block (player mask 0x10 / npc mask 0x10) calls `entity.hit(damageType, damage)` into a 4-slot damage queue (ClientEntity.java:98-107, 264-273: `damage[4]`, `damageType[4]`, `damageCycle[4]`, each slot timed `loopCycle+70`) and sets `combatCycle = loopCycle + 300`. The render loop (Client.java:6321-6341) draws up to 4 simultaneous hitmarks at distinct screen offsets. The Go ports the 225 form: scalar `arg4.Damage`/`arg4.DamageType` assignment and `CombatCycle = LoopCycle + 400`; the Go […]

### G20. Player CHAT block (mask 0x40): missing 244 crown prefixes (@cr1@/@cr2@) and player.visible guard

- **Unit:** `client/Client#10`  **Java:** `Client.java:9215-9260`  **Go:** `pkg/jagex2/client/client.go:10661-10705`

Two 244 deltas missed (Go matches 225-clean deob/client.java:10495-10530 exactly): (1) The visibility guard is `if (player.name != null && player.visible)` in 244 but Go has only `if arg4.Name != ""` — the `&& arg4.Visible` term is absent, so chat from non-visible players is processed when 244 suppresses it (Go ClientPlayer HAS a Visible field). (2) The addMessage routing in 244 prepends staff crowns: `if (type==2||type==3) addMessage(filtered, "@cr2@"+name, 1); else if (type==1) addMessage(filtered, "@cr1@"+name, 1); else addMessage(filtered, name, 2);`. Go has only `if var15 > 1 { AddMessage(1, var18, arg4.Name) } else { AddMessage(2, var18, arg4.Name) }` — no crown tags, and the type==2/3 vs type==1 distinction collapsed. (LOGIC-DELTA-SCOPE.md:96 lists crowns as a known deferred [L] item.)

```
Go: `if arg4.Name != "" {` and `if var15 > 1 { c.AddMessage(1, var18, arg4.Name) } else { c.AddMessage(2, var18, arg4.Name) }`. Java 244: `if (player.name != null && player.visible) {` and `if (type == 2 || type == 3) { addMessage(filtered, "@cr2@" + player.name, 1); } else if (type == 1) { addMessage(filtered, "@cr1@" + player.name, 1); } else { addMessage(filtered, player.name, 2); }`
```

> **Verifier (confirmed, severity bug):** Both sub-claims are confirmed real 244 deltas the Go port is missing. I ruled out the deob parameter-scramble false-positive: Go/225 addMessage is (type,text,sender) while 244 addMessage is (text,sender,type), but I traced both bodies — each stores type→messageType[0], text→messageText[0], sender→messageSender[0], so `type` semantics are identical; the routing comparison is valid.\n\nDELTA 1 (visible guard): Java 244 Client.java:9223 guards CHAT with `if (player.name != null && player.visible)`; Go client.go:10666 has only `if arg4.Name != \"\"`, dropping the `&& arg4.Visible` term. The Go ClientPlayer DOES have a Visible field (playerentity/clientplayer.go:28, with IsVisible() returning e.V […]
>
> **Refined:** Player CHAT block (mask 0x40) in Go (client.go:10661-10705) misses two 244 logic deltas vs Java 244 (Client.java:9215-9260): (1) the visibility guard is only `if arg4.Name != \"\"` but 244 requires `if (player.name != null && player.visible)` — the `&& arg4.Visible` term is missing (Go ClientPlayer has the Visible field), so chat from non-visible players is processed; this delta is NOT in LOGIC-DELTA-SCOPE.md. (2) The addMessage routing differs: Go does `if var15>1 {AddMessage(1,var18,name)} els […]

### G21. useMenuOption missing 244 tail `redrawSidebar = true`

- **Unit:** `client/Client#10`  **Java:** `Client.java:10314-10318`  **Go:** `pkg/jagex2/client/client.go:5143-5145`

244 useMenuOption ends with three statements: `this.objSelected = 0; this.spellSelected = 0; this.redrawSidebar = true;`. The Go ends with only `c.ObjSelected = 0; c.SpellSelected = 0` — the `c.RedrawSidebar = true` is missing (Go matches 225-clean deob/client.java:5480-5481 which had only two). After any menu interaction that falls through to the tail, 244 forces a sidebar redraw to clear the obj/spell selection highlight; the Go leaves stale selection state on the sidebar until some other path triggers a redraw.

```
Go: `c.ObjSelected = 0\n\tc.SpellSelected = 0\n}` (function ends). Java 244: `this.objSelected = 0;\n\t\tthis.spellSelected = 0;\n\t\tthis.redrawSidebar = true;\n\t}`
```

> **Verifier (confirmed, severity bug):** Verified against both source sites. Java-244 useMenuOption (Client.java function starting at 9669; tail at 10314-10318) ends with THREE statements: `this.objSelected = 0; this.spellSelected = 0; this.redrawSidebar = true;`. The Go UseMenuOption (client.go, function starting at line 4570, tail at 5143-5145) ends with only `c.ObjSelected = 0` and `c.SpellSelected = 0` — `c.RedrawSidebar = true` is absent.

Ruled out false-positive causes:
- Not a rename/cross-lineage issue: the Go field `RedrawSidebar` genuinely exists and is set in this very function on lines 5062 and 5108, and elsewhere (891, 1092, 4298, etc.). Java-244 field `redrawSidebar` (declared line 539) maps directly to Go `RedrawSid […]
>
> **Refined:** useMenuOption (Go: UseMenuOption, client.go:4570-5145) is missing the 244 tail statement `c.RedrawSidebar = true`. Java-244 (Client.java:10314-10318) ends the function with `this.objSelected = 0; this.spellSelected = 0; this.redrawSidebar = true;`; the Go ends with only the first two assignments at client.go:5143-5144. The RedrawSidebar field exists and is used elsewhere in the same function, so this is a true omission, not a rename. Callers do not unconditionally set RedrawSidebar after the cal […]

### G22. useMenuOption action 660 (walk-here) scene.click uses 225 offsets (-11/-8) not 244 (-4/-4)

- **Unit:** `client/Client#10`  **Java:** `Client.java:10188-10193`  **Go:** `pkg/jagex2/client/client.go:4770-4775`

244 action 660 calls `this.scene.click(c - 4, b - 4)` (menu visible) or `this.scene.click(mouseClickY - 4, mouseClickX - 4)`. The Go uses `c.Scene.Click(var4-11, var3-8)` and `c.Scene.Click(c.MouseClickY-11, c.MouseClickX-8)` — the 225 viewport offsets (verified deob/client.java:5109/5111). Same root cause as the showContextMenu finding: the 244 viewport origin moved to (4,4). The tile clicked when walking is off by (7 in X, 7 in Z) relative to the cursor.

```
Go: `c.Scene.Click(var4-11, var3-8)` / `c.Scene.Click(c.MouseClickY-11, c.MouseClickX-8)`. Java 244: `this.scene.click(c - 4, b - 4);` / `this.scene.click(super.mouseClickY - 4, super.mouseClickX - 4);`
```

> **Verifier (confirmed, severity bug):** Verified real. Traced argument semantics through both callee bodies to rule out a deob parameter scramble: Go var3=MenuParamB=Java b, var4=MenuParamC=Java c, var5=MenuAction=action (client.go:4578-4580 vs Client.java:9681-9683). The "Walk here" menu entry stores MenuParamB=mouseX, MenuParamC=mouseY identically in both (client.go:9131-9132 vs Java HandleViewportOptions). Scene.Click(arg1->mouseY, arg2->mouseX) is byte-identical in both ports (world3d.go:1029 vs World3D.java:1032) and the offset genuinely lands on the stored click coords that drive next-frame tile picking — not absorbed anywhere downstream.

Java 244 action 660: `scene.click(c - 4, b - 4)` (visible) and `scene.click(mouseClick […]
>
> **Refined:** 244 action 660 (walk-here) calls `scene.click(c - 4, b - 4)` (menu visible) / `scene.click(mouseClickY - 4, mouseClickX - 4)` (else). The Go uses `c.Scene.Click(var4-11, var3-8)` / `c.Scene.Click(c.MouseClickY-11, c.MouseClickX-8)` — the stale 225 viewport offsets. In 244 the 3D viewport origin moved to (4,4) (areaViewport.draw(graphics, 4, 4)), so all mouse->viewport translations subtract 4. The clicked tile is off by 7 px in the Y/Z axis (11-4) and 4 px in the X axis (8-4), not 7/7 as original […]

### G23. Cross-reference: ClientEntity.Teleport (Java move) omits preanimRouteLength=0 reset and uses Priority<=1 for postanim_mode==1

- **Unit:** `client/Client#10`  **Java:** `ClientEntity.java:173-201 (move), 205-251 (step)`  **Go:** `pkg/jagex2/dash3d/entity/clententity.go:82-110 (Teleport), 112-151 (MoveAlongRoute)`

Encountered while verifying the move/step call sites in my unit (getPlayerNewVis/OldVis, getNpcPos*). These methods are in ClientEntity (another auditor's unit) but feed directly into my handlers. Java 244 move() resets THREE fields `routeLength=0; preanimRouteLength=0; seqDelayMove=0;` (ClientEntity.java:195-197); Go Teleport resets only `PathLength=0; SeqTrigger=0` — no PreanimRouteLength reset (the field does not exist on the Go entity). Also Java guards the seq-cancel with `SeqType.types[primarySeqId].postanim_mode == 1` while Go uses `seqtype.Instances[...].Priority <= 1`; these are different quantities unless the 244 lineage maps postanim_mode onto the Go Priority field. Reported as latent for the owning auditor to confirm; the parameter scrambling itself (Java step(dir,run) vs Go MoveAlongRoute(run,dir)) IS correctly compensated at every call site in my unit.

```
Go Teleport tail: `e.PathLength = 0; e.SeqTrigger = 0; e.PathTileX[0] = arg2; ...` (no preanim reset). Java move tail: `this.routeLength = 0; this.preanimRouteLength = 0; this.seqDelayMove = 0; this.routeTileX[0] = arg1; ...`
```

> **Verifier (confirmed, severity bug):** CONFIRMED REAL after attempting refutation on all common false-positive axes (rename traps, field-mapping, deob scramble, behavior-exists-elsewhere).

Note: the Java file lives at src/main/java/jagex2/dash3d/ClientEntity.java (not the .../entity/ path in the finding's javaRef), but the move()/step() bodies match the cited content exactly.

Claim 1 (missing preanimRouteLength=0 reset) - REAL:
- Java 244 move() tail (01f16088 ClientEntity.java) resets THREE fields: routeLength=0; preanimRouteLength=0; seqDelayMove=0. Go Teleport (clententity.go:104-105) resets only PathLength=0; SeqTrigger=0.
- Ruled out a rename trap: SeqTrigger maps to seqDelayMove, NOT preanimRouteLength. Proof: Java update […]
>
> **Refined:** ClientEntity.Teleport (Java 244 move) and MoveAlongRoute (Java 244 step) carry stale 225-lineage logic instead of the 244 delta: (1) the primary-sequence cancel guard uses `seqtype.Instances[PrimarySeqID].Priority <= 1` where Java 244 uses `SeqType.types[primarySeqId].postanim_mode == 1` - these are distinct SeqType fields (Go Priority=opcode5/default5; PostanimMode=opcode10/default-1), so they cancel different animation sets; (2) Java 244 move() resets routeLength=0, preanimRouteLength=0, seqDe […]

### G24. DrawInterface type-2 inventory drag autoscroll block entirely missing

- **Unit:** `client/Client#11`  **Java:** `Client.java:10601-10629`  **Go:** `pkg/jagex2/client/client.go:3660-3673`

In the type-2 (inventory) drag branch, Java draws the dragged icon then runs two autoscroll adjustments: when slotY+dy < Pix2D.top it scrolls the parent up (com.scrollPosition -= autoscroll; objGrabY += autoscroll) and when slotY+dy+32 > Pix2D.bottom it scrolls down, each clamped to sceneDelta*10 and to the remaining scroll range. The Go port stops after DrawAlpha(128, var18+var33, var32+var21) and omits BOTH autoscroll blocks. Dragging an inventory item to the top/bottom edge of a scrollable interface will no longer auto-scroll. SceneDelta and ObjGrabY both exist in the Go struct, so this is a dropped translation, not an unportable feature.

```
Java: `icon.transPlotSprite(slotX + dx, 128, slotY + dy); if (slotY + dy < Pix2D.top && com.scrollPosition > 0) { int autoscroll = (Pix2D.top - slotY - dy) * this.sceneDelta / 3; ... com.scrollPosition -= autoscroll; this.objGrabY += autoscroll; } if (slotY + dy + 32 > Pix2D.bottom && ...) { ... }`  Go: `var23.DrawAlpha(128, var18+var33, var32+var21)
} else if c.SelectedArea != 0 ...` (no autoscroll)
```

> **Verifier (confirmed, severity bug):** Confirmed against the reference. In Java drawInterface (Client.java:10601-10629), inside the type-2 inventory drag branch (the `objDragArea != 0 && objDragSlot == slot && objDragInterfaceId == child.id` case), after `icon.transPlotSprite(slotX + dx, 128, slotY + dy)` there are TWO autoscroll blocks: (1) `if (slotY + dy < Pix2D.top && com.scrollPosition > 0)` scrolls the parent up — `autoscroll = (Pix2D.top - slotY - dy) * sceneDelta / 3`, clamped to `sceneDelta*10` and to `com.scrollPosition`, then `com.scrollPosition -= autoscroll; objGrabY += autoscroll`; and (2) `if (slotY + dy + 32 > Pix2D.bottom && com.scrollPosition < com.scroll - com.height)` scrolls down symmetrically, `com.scrollPos […]

### G25. DrawInterface type-3 alpha-blended rectangle branches missing (translucent components render opaque)

- **Unit:** `client/Client#11`  **Java:** `Client.java:10657-10670`  **Go:** `pkg/jagex2/client/client.go:3832-3836`

Java type-3 has four cases keyed on child.alpha: alpha==0 -> opaque fillRect/drawRect; alpha!=0 -> alpha-blended Pix2D.fillRectTrans / Pix2D.drawRectTrans with 256-(alpha&0xFF). The Go port reduces type-3 to just `else if var14.Fill { FillRect } else { DrawRect }` with NO alpha check and NO translucent path. Component.Alpha (int8) IS parsed from the wire (component.go:38,125) but is never consulted at draw time, and pix2d has no FillRectTrans/DrawRectTrans at all (grep returns nothing). Result: every type-3 component with alpha != 0 (intended semi-transparent overlay) renders fully opaque.

```
Java: `if (child.alpha == 0) { if (child.fill) Pix2D.fillRect(...); else Pix2D.drawRect(...); } else if (child.fill) { Pix2D.fillRectTrans(childY, 256 - (child.alpha & 0xFF), child.height, child.width, child.colour, childX); } else { Pix2D.drawRectTrans(...256 - (child.alpha & 0xFF)); }`  Go: `} else if var14.Fill { pix2d.FillRect(...) } else { pix2d.DrawRect(...) }`
```

> **Verifier (confirmed, severity bug):** Confirmed real after exhausting refutation paths. Java Client.java:10650-10670 (the `child.type == 3` branch) has four cases keyed on the parsed wire field `child.alpha` (byte, Component.java:53,227): alpha==0 -> opaque Pix2D.fillRect/drawRect; alpha!=0 -> Pix2D.fillRectTrans / Pix2D.drawRectTrans with intensity 256-(alpha&0xFF). The Go type-3 branch (client.go:3832-3836, reached via the `} else {` after `if Type==2 {} else if Type != 3 {}`) contains only `if var14.Fill { pix2d.FillRect } else { pix2d.DrawRect }` — the opaque path with NO `alpha == 0` guard and NO translucent branch. Verification of false-positive axes: (1) not a lineage/name inversion — traced control flow and call args, th […]

### G26. DrawInterface type-4 text: Pix2D.width2d==479 chatback colour swap missing

- **Unit:** `client/Client#11`  **Java:** `Client.java:10737-10745`  **Go:** `pkg/jagex2/client/client.go:3709-3713`

When rendering type-4 text into the 479-wide chatback PixMap, Java remaps colour 0xffff00 -> 0x0000ff and 0x00c000 -> 0xffffff (so yellow/green text stays legible on the parchment chatback). This block sits immediately after the buttonType==6 'Please wait...' handling. The Go port omits it entirely; text colours in the chatback interface will be wrong. Present in both Java 244 and TS client (Client.ts:9538-9545).

```
Java: `if (Pix2D.width2d == 479) { if (colour == 0xffff00) { colour = 0x0000ff; } if (colour == 0x00c000) { colour = 0xffffff; } }`  Go: goes straight from `if var14.ButtonType == 6 && c.PressedContinueOption {...}` to `var32 = var26 + var15.Height` with no 479 block.
```

> **Verifier (confirmed, severity bug):** Confirmed real and could not refute. Java Client.java:10737-10745 (in the type-4 text branch of drawInterface) remaps the local `colour` variable when `Pix2D.width2d == 479`: 0xffff00 (yellow) -> 0x0000ff (blue) and 0x00c000 (green) -> 0xffffff (white). This block sits immediately after the `child.buttonType == 6 && pressedContinueOption` "Please wait..." handler and immediately before the `lineY = font.height + childY` line. The remapped `colour` is then passed to centreStringTag/drawStringTag.

In the Go port (pkg/jagex2/client/client.go), the type-4 branch's colour variable is `var16` (declared at ~3636, set from var14.Colour/OverColour/ActiveColour). After the `var14.ButtonType == 6 && c […]
>
> **Refined:** When rendering type-4 interface text while the 479-wide chatback PixMap is bound (pix2d.Width2D == 479), Java remaps the text colour 0xffff00 -> 0x0000ff and 0x00c000 -> 0xffffff so yellow/green config text stays legible on the parchment chatback. This block (Client.java:10737-10745) sits between the buttonType==6 "Please wait..." handler and the lineY computation. The Go port (pkg/jagex2/client/client.go: between lines 3712 and 3713, with the colour held in var16) omits it entirely; affected ch […]

### G27. DrawInterface type-2: selection-highlight outline argument dropped in getIcon call

- **Unit:** `client/Client#11`  **Java:** `Client.java:10575-10580`  **Go:** `pkg/jagex2/client/client.go:3654-3659`

Java computes `int outline = 0; if (objSelected == 1 && objSelectedSlot == slot && objSelectedInterface == child.id) outline = 16777215;` and passes it as the first arg of `getIcon(outline, count, id)`; getIcon draws a white outline around the icon when outline>0 (and applies a 1.04x zoom). The Go call site at 3659 computes no outline and calls `objtype.GetIcon(var22, ...)` (id, count) — a 2-arg signature with no outlineRgb. The root is that objtype.GetIcon (objtype.go:346, another auditor's unit) only implements Java's outlineRgb==0 path: it omits the white selection outline AND the cert-template recursion uses GetIcon(CertLink, 10) instead of Java's getIcon(-1, 10, certlink) so the cert sub-icon loses its 1.5x zoom. Net visible defect from this call site: a selected/being-used inventory item no longer gets its white highlight outline.

```
Java: `int outline = 0; if (this.objSelected == 1 && this.objSelectedSlot == slot && this.objSelectedInterface == child.id) { outline = 16777215; } Pix32 icon = ObjType.getIcon(outline, child.invSlotObjCount[slot], id);`  Go: `var22 = var14.InvSlotObjId[var27] - 1; ... var23 := objtype.GetIcon(var22, var14.InvSlotObjCount[var27])`
```

> **Verifier (confirmed, severity bug):** Confirmed against the Java-244 reference at 01f16088 (the exact commit the Go doc-comments cite). Java `ObjType.getIcon` is a 3-arg method `getIcon(int outlineRgb, int count, int id)` with three outlineRgb-dependent behaviors: (a) cache get/put gated on outlineRgb==0; (b) zoom multiplier — outlineRgb==-1 -> x1.5, outlineRgb>0 -> x1.04; (c) an `if (outlineRgb > 0)` block that paints the colored selection-highlight outline. The Go `GetIcon` (objtype.go:346) is only 2-arg `GetIcon(arg0=id, arg2=count)` and implements ONLY the outlineRgb==0 subset: no zoom adjustment, no colored-outline block, unconditional cache use.\n\nCall sites: Java Client.java:10575-10580 computes `int outline = 0; if (obj […]
>
> **Refined:** DrawInterface type-2 (client.go:3654-3659) drops Java-244's selection-highlight outline argument. Java (Client.java:10575-10580) computes `outline = 16777215` when the inventory slot is the currently selected/being-used item (objSelected==1 && objSelectedSlot==slot && objSelectedInterface==child.id) and passes it as the first arg of the 3-arg `ObjType.getIcon(outlineRgb, count, id)`; getIcon paints a white selection outline when outlineRgb>0 (with a 1.04x zoom). The Go call site computes no outl […]

### G28. UpdateVarp missing clientCode==9 (bankArrangeMode) write — 244 bank-arrange feature broken

- **Unit:** `client/Client#12`  **Java:** `Client.java:11424-11425`  **Go:** `pkg/jagex2/client/client.go:3919-3928`

Java updateVarp has `else if (clientCode == 9) { this.bankArrangeMode = value; }`. The Go UpdateVarp handles clientCode 1,3,4,5,6,8 but has NO branch for clientCode==9. There is no BankArrangeMode field anywhere in the Go port (grep returns nothing). bankArrangeMode is NEW in 244 (absent from 225-clean) and is consumed in drawInterface (Java Client.java:3013: `if (this.bankArrangeMode == 1 && com.clientCode == 206)`). With the varp write missing, the value stays 0 and the bank swap/insert arrangement mode can never activate. This is a 244 migration gap, not a deob-name divergence.

```
Java: `} else if (clientCode == 9) {\n\t\t\tthis.bankArrangeMode = value;\n\t\t}`. Go ends at: `if var3 == 8 { c.SplitPrivateChat = var4; c.RedrawChatback = true }` — no `var3 == 9`. `grep -rn BankArrange` across client.go + clientextras returns nothing (exit 1).
```

> **Verifier (confirmed, severity bug):** Verified all claims directly. Java 244 updateVarp (Client.java:11349 `updateVarp(int id)`, branch at 11424-11425) has `} else if (clientCode == 9) { this.bankArrangeMode = value; }`. The Go port UpdateVarp (pkg/jagex2/client/client.go:3854-3929) is the direct port (same clientcode read, same handling of codes 1/3/4/5/6/8) and ends at `if var3 == 8 { ... RedrawChatback = true }` with NO `var3 == 9` branch. `grep -rni bankArrange pkg/` returns nothing (exit 1) — confirmed no BankArrangeMode field exists anywhere in the Go port. Confirmed it is a 244-new feature: 225-clean lives at src/main/java/deob/client.java and has no bankArrange. The value is consumed in Java drawInterface (Client.java:30 […]
>
> **Refined:** Confirmed: Go UpdateVarp (pkg/jagex2/client/client.go:3854-3929) omits the Java 244 `clientCode == 9 → bankArrangeMode = value` branch (Client.java:11424-11425), and no BankArrangeMode field exists in the Go port. Beyond the cited gap, the consumer site in Go drawInterface/mouse-handling (client.go:7505-7516) is also still the pre-244 plain swap — it lacks the `bankArrangeMode == 1 && clientCode == 206` insert-mode gate, the swapObj shift loop, and the trailing `out.p1(mode)` protocol byte prese […]

### G29. HandleSocialMenuOption stuck on 225 logic — missing 244 clientCode 701-900 friend range

- **Unit:** `client/Client#12`  **Java:** `Client.java:11255-11273`  **Go:** `pkg/jagex2/client/client.go:8386-8408`

Java 244 condition is `((clientCode >= 1 && clientCode <= 200) || (clientCode >= 701 && clientCode <= 900))` with a 4-way adjust (>=801 -> -701, >=701 -> -601, >=101 -> -101, else --). The Go only checks `var3 >= 1 && var3 <= 200` with a 2-way adjust (>=101 -> -101, else --). The entire 701-900 friend-slot range (used in 244 for friends 100..199) is dropped, so right-click Remove/Message menu options are never generated for those friend components. This is the rev-225 version not migrated to 244.

```
Go: `if var3 >= 1 && var3 <= 200 { if var3 >= 101 { var3 -= 101 } else { var3-- } ... }`. Java: `if ((clientCode >= 1 && clientCode <= 200) || (clientCode >= 701 && clientCode <= 900)) { if (clientCode >= 801) { clientCode -= 701; } else if (clientCode >= 701) { clientCode -= 601; } else if (clientCode >= 101) { clientCode -= 101; } else { clientCode--; } ... }`
```

> **Verifier (confirmed, severity bug):** Confirmed by direct side-by-side comparison of both cited sites. Java 244 handleSocialMenuOption (Client.java:11251-11281) gates on `((clientCode >= 1 && clientCode <= 200) || (clientCode >= 701 && clientCode <= 900))` with a 4-way adjust: >=801 -> -701, >=701 -> -601, >=101 -> -101, else --. Go HandleSocialMenuOption (client.go:8386-8408) only checks `var3 >= 1 && var3 <= 200` with the 2-way adjust (>=101 -> -101, else --). The 701-900 range and its two adjust branches are entirely absent.

Ruled out false-positive causes:
- Not a deob name trap: the friend-array index math is identical in meaning across both. Java 244 maps 1..100/101..200 -> indices 0..99 and 701..800/801..900 -> indices 1 […]
>
> **Refined:** Accurate as written. HandleSocialMenuOption (pkg/jagex2/client/client.go:8386-8408) is the rev-225 version: it gates on `var3 >= 1 && var3 <= 200` with a 2-way adjust, missing the Java-244 (Client.java:11251-11281) expanded condition `(clientCode 1..200) || (clientCode 701..900)` and its 4-way adjust (>=801 -> -701, >=701 -> -601, >=101 -> -101, else --). The 701-900 friend-slot range (friends 100..199 in 244's 200-entry friend list) is dropped, so Remove/Message menu options are never generated […]

### G30. UpdateInterfaceContent stuck on 225 logic — missing 244 clientCode 701-900 / 801-900 friend ranges

- **Unit:** `client/Client#12`  **Java:** `Client.java:11434-11468`  **Go:** `pkg/jagex2/client/client.go:5260-5284`

Java 244 friend-name branch is `((clientCode >= 1 && clientCode <= 100) || (clientCode >= 701 && clientCode <= 900))` with `if (clientCode > 700) clientCode -= 601; else clientCode--`. Java 244 friend-world branch is `clientCode >= 101 && clientCode <= 200 || !(clientCode < 801 || clientCode > 900)` (i.e. 101-200 OR 801-900) with `if (clientCode > 800) clientCode -= 701; else clientCode -= 101`. The Go only handles `var3 >= 1 && var3 <= 100` (name, `var3--`) and `var3 >= 101 && var3 <= 200` (world, `var3 -= 101`). The 701-900 and 801-900 ranges are absent, so friend list slots beyond the first 100 never get name/world text or buttonType set. Paired with the HandleSocialMenuOption gap above.

```
Go: `if var3 >= 1 && var3 <= 100 { var3-- ... } else if var3 >= 101 && var3 <= 200 { var3 -= 101 ... }`. Java: `if ((clientCode >= 1 && clientCode <= 100) || (clientCode >= 701 && clientCode <= 900)) { if (clientCode > 700) { clientCode -= 601; } else { clientCode--; } ... } else if (clientCode >= 101 && clientCode <= 200 || !(clientCode < 801 || clientCode > 900)) { if (clientCode > 800) { clientCode -= 701; } else { clientCode -= 101; } ... }`
```

> **Verifier (confirmed, severity bug):** Confirmed against Java-244 via `git show 01f16088:.../Client.java`. The 244 `updateInterfaceContent` (lines 11434-11468) friend-name branch is `((clientCode >= 1 && clientCode <= 100) || (clientCode >= 701 && clientCode <= 900))` with `if (clientCode > 700) clientCode -= 601; else clientCode--`, and the friend-world branch is `clientCode >= 101 && clientCode <= 200 || !(clientCode < 801 || clientCode > 900)` with `if (clientCode > 800) clientCode -= 701; else clientCode -= 101`. The description and evidence reproduce these conditions and arithmetic exactly (NOT(<801||>900) == 801-900 is a correct read).\n\nThe Go `UpdateInterfaceContent` (client.go:5258-5284) only handles `var3 >= 1 && var3  […]
>
> **Refined:** Accurate as written. Optional context: the gap is part of the broader 244 friend-list expansion (100→200) that the Go port has not adopted — Go's FriendName/FriendWorld are still make(...,100) and FriendCount is capped at 100 (client.go:565,638,7147,9742) — so a full 244 fix must also bump those arrays/caps to 200 alongside adding the 701-900/801-900 branches to UpdateInterfaceContent.

### G31. DrawChatback type-8 (duel) chat message rendered in wrong colour

- **Unit:** `client/Client#12`  **Java:** `Client.java:11916`  **Go:** `pkg/jagex2/client/client.go:9468`

Java 244 draws type-8 (duel) messages with colour `0x7e3200` (=8270336). The Go uses `13350793` (=0xCBB789), a completely different colour. Type-8 messages are NEW in 244 (absent from 225-clean), so this branch was added during migration with an incorrect constant. Cross-checked against the TS client: Colors.ts `DUEL_MESSAGE = 0x7e3200`. Duel/trade-accept chat lines will render in the wrong colour.

```
Java: `font.drawString(sender + " " + this.messageText[i], 0x7e3200, y, 4)` (0x7e3200=8270336). Go: `var2.DrawString(4, var6, 13350793, c.MessageSender[i]+" "+c.MessageText[i])` (13350793=0xCBB789). TS Colors.ts:27 `DUEL_MESSAGE = 0x7e3200`.
```

> **Verifier (confirmed, severity bug):** Confirmed real after tracing both call sites and argument semantics.

Java 244 (Client.java:11916, read via git show 01f16088): the type-8 (duel/trade-accept) branch calls `font.drawString(sender + " " + this.messageText[i], 0x7e3200, y, 4)`. PixFont.drawString signature is `drawString(String arg0, int arg1, int arg3, int arg4)` where arg1 is the colour (forwarded to plotLetter's final colour arg), arg3 is y, arg4 is x. So colour = 0x7e3200 = 8270336.

Go (client.go:9468): `var2.DrawString(4, var6, 13350793, c.MessageSender[i]+" "+c.MessageText[i])`. Go DrawString signature is `DrawString(arg0 int, arg1 int, arg3 int, arg4 string)` where arg0 is x, arg1 is y, arg3 is the colour (forwarded to […]
>
> **Refined:** Go client.go:9468 renders type-8 (duel/trade-accept) chatback messages with colour 13350793 (0xCBB789, light tan) instead of Java 244's 0x7e3200 (8270336, dark brown). The colour argument maps correctly between Java drawString(text, colour=arg1, y, x) and Go DrawString(x, y, colour=arg3, text), so this is a genuine constant mismatch, not an arg-order artifact. All other chatback branches pass colours through as raw RGB unchanged, and TS Colors.ts confirms DUEL_MESSAGE = 0x7e3200. Fix: change 133 […]

### G32. DrawChatback username line: missing 244 localPlayer.name preference + width computed from a different string than displayed

- **Unit:** `client/Client#12`  **Java:** `Client.java:11933-11941`  **Go:** `pkg/jagex2/client/client.go:9478-9479`

Java 244 picks the chat input prompt name as `localPlayer.name` when `localPlayer != null && localPlayer.name != null`, falling back to `formatDisplayName(this.username)` only when null, and uses that SAME `username` variable for both the drawn text and the chatTyped x-offset width. The Go unconditionally uses `FormatName(c.Username)` for the drawn name (never localPlayer.name), AND computes the chatTyped x-offset from `StringWidth(c.Username+": ")` — the RAW (un-FormatName'd) Username — so the width is measured from a different string than what is displayed. FormatName transforms underscore->space and capitalizes (JString.formatDisplayName), so the displayed text and the measured width diverge whenever the name has underscores/lowercase, misaligning the typed-text cursor.

```
Java: `String username; if (localPlayer == null || localPlayer.name == null) { username = JString.formatDisplayName(this.username); } else { username = localPlayer.name; } font.drawString(username + ":", 0, 90, 4); font.drawString(this.chatTyped + "*", 255, 90, font.stringWid(username + ": ") + 6);`. Go: `var2.DrawString(4, 90, 0, jstring.FormatName(c.Username)+":"); var2.DrawString(var2.StringWidth(c.Username+": ")+6, 90, 0xFF, c.ChatTyped+"*")` — display uses FormatName(Username), width uses raw Username, localPlayer.name never consulted.
```

> **Verifier (confirmed, severity bug):** Both cited sites confirmed in full context, and the arg orderings are correct (not a deob scramble).

JAVA (Client.java:11933-11941, read via git show 01f16088): builds `String username` = `localPlayer.name` when `localPlayer != null && localPlayer.name != null`, else `JString.formatDisplayName(this.username)`. Then `font.drawString(username + ":", 0, 90, 4)` and `font.drawString(this.chatTyped + "*", 255, 90, font.stringWid(username + ": ") + 6)`. Java drawString signature is (text, color, y, x): both the drawn name AND the x-offset width are computed from the SAME formatted `username` variable.

GO (client.go:9478-9479): `var2.DrawString(4, 90, 0, jstring.FormatName(c.Username)+":")` then  […]
>
> **Refined:** Confirmed real. Decisive defect: the chatTyped x-offset is computed from StringWidth(c.Username+": ") using the RAW username, while the displayed prompt uses FormatName(c.Username) — so the measured width and the drawn glyphs diverge (FormatName changes '_'->' ' and capitalizes, altering proportional-font widths), misaligning the typed-text cursor. Java measures the width from the same formatted string it draws. Fix: measure StringWidth from the same formatted string used for drawing (and, for f […]

### G33. AddFriend full-list cap and message do not match 244 (single 100-cap + wrong text; members 200-tier dropped)

- **Unit:** `client/Client#13`  **Java:** `Client.java:12149-12189`  **Go:** `pkg/jagex2/client/client.go:7143-7174`

Java 244 addFriend uses a two-tier cap: `if (friendCount >= 100 && membersAccount != 1)` rejects free users at 100, and `else if (friendCount >= 200)` rejects members at 200, both with the message "Your friendlist is full. Max of 100 for free users, and 200 for members". The Go port has a single `if c.FriendCount >= 100` check with NO membersAccount test, so members are wrongly capped at 100 slots, and it prints the wrong text "Your friends list is full. Max of 100 hit" (a paraphrase of the ignore-list message), not the 244 friend message. The MembersAccount field exists (client.go:537, set at client.go:10017) but is unused here. The remainder of the method (already-on-list / on-ignore-list checks, localPlayer.name guard, FRIENDLIST_ADD pIsaac(9)+p8) is correct.

```
Java: `if (this.friendCount >= 100 && this.membersAccount != 1) { this.addMessage("Your friendlist is full. Max of 100 for free users, and 200 for members",...); return; } else if (this.friendCount >= 200) {...}`  ||  Go: `if c.FriendCount >= 100 { c.AddMessage(0, "Your friends list is full. Max of 100 hit", ""); return }`
```

> **Verifier (confirmed, severity bug):** Confirmed against Java 244 (git 01f16088 Client.java:12152-12159) and Go (pkg/jagex2/client/client.go:7143-7174).

Java 244 addFriend uses a two-tier full-list cap:
  if (friendCount >= 100 && membersAccount != 1) { addMessage("Your friendlist is full. Max of 100 for free users, and 200 for members", "", 0); return; }
  else if (friendCount >= 200) { addMessage(same text, "", 0); return; }
So free accounts (membersAccount != 1) are capped at 100, members (membersAccount == 1) at 200.

The Go port has a single check with no members test:
  if c.FriendCount >= 100 { c.AddMessage(0, "Your friends list is full. Max of 100 hit", ""); return }
This (a) wrongly caps members at 100 instead of 200, a […]
>
> **Refined:** Go AddFriend (pkg/jagex2/client/client.go:7147-7150) uses a single `if c.FriendCount >= 100` full-list cap with the wrong message text "Your friends list is full. Max of 100 hit". Java 244 (Client.java:12154-12159) uses a two-tier cap: free users (membersAccount != 1) rejected at 100 and members rejected at 200, both with "Your friendlist is full. Max of 100 for free users, and 200 for members". The Go port drops the membersAccount tier (the field exists at client.go:537 and is set from UPDATE_P […]

### G34. load() starts a 225-lineage MIDI worker thread + scape_main request that does not exist in the 244 client

- **Unit:** `client/Client#2`  **Java:** `Client.java:1489-1499 (load() start), Client.java:1601-1609 (the only midi handling in load: midiSong=0 / onDemand.request(2,...))`  **Go:** `pkg/jagex2/client/client.go:5642-5652`

The Java 244 load() does NOT start any MIDI worker thread and does NOT request 'scape_main'. In 244, ALL MIDI playback is routed through OnDemand archive 2 (Client.java:1601-1603 inside load issues onDemand.request(2, midiSong=0); Client.java:2444-2445 plays it via saveMidi(midiFading, req.data)). There is no setMidi(int,String,int), no midiCrc/midiSize/currentMidi fields, and no scape_main jingle anywhere in 244 (verified by grep across the whole file). The Go load() retains the 225-era block: it sets StartMidiThread/MidiThreadActive, launches `go c.RunMidi()`, and calls `c.SetMidi(12345678, "scape_main", 40000)`. RunMidi (client.go:1548-1599) then tries to download scape_main_12345678.mid over HTTP via OpenURL — a request a 244 server does not serve — and runs alongside the faithful OnDemand midiSong=0 request a few lines later (client.go:5952-5959). This is a functional divergence (extra goroutine + failing network fetch + title-music behavior absent from 244).

```
Go: `if !LowMemory { c.StartMidiThread = true; c.MidiThreadActive = true; go c.RunMidi(); c.SetMidi(12345678, "scape_main", 40000) }`. Java 244 load() has no equivalent; its only midi line is `this.midiSong = 0; this.midiFading = false; this.onDemand.request(2, this.midiSong);` which the Go ALSO has at client.go:5953-5955.
```

> **Verifier (confirmed, severity bug):** VERIFIED REAL. Read both sites in full plus callees and grep-swept the 244 reference (01f16088, confirmed on branch 244).

Java 244 load() (Client.java:1489-1700 via git show 01f16088): contains NO MIDI worker-thread start and NO setMidi/scape_main. Its ONLY MIDI handling inside load() is the OnDemand block at 1601-1603: `this.midiSong = 0; this.midiFading = false; this.onDemand.request(2, this.midiSong);` (then a remaining()/updateOnDemand() drain loop). Playback happens later via the archive-2 dispatch at 2444-2445: `req.archive == 2 && this.midiSong == req.file ... this.saveMidi(this.midiFading, req.data)`.

Grep across the entire 244 Client.java confirms: no `setMidi(int,String,int)` (on […]
>
> **Refined:** Go Client.Load() (pkg/jagex2/client/client.go:5642-5652) retains a 225-lineage MIDI-worker block — StartMidiThread/MidiThreadActive=true, `go c.RunMidi()`, and `c.SetMidi(12345678, "scape_main", 40000)` — that has NO equivalent in Java 244 load() (Client.java:1489-1700, ref 01f16088). In 244, load()'s only MIDI is `midiSong=0; midiFading=false; onDemand.request(2, midiSong)` (Client.java:1601-1603), played later via the archive-2 dispatch `saveMidi(midiFading, req.data)` (Client.java:2444-2445); […]

### G35. field1264 viewport flash effect entirely unported (login reset + updateGame decrement among the missing sites)

- **Unit:** `client/Client#3`  **Java:** `Client.java:2692,2935-2937`  **Go:** `pkg/jagex2/client/client.go:6828-6919,7382-7389`

Java field1264 is a yellow sine-modulated horizontal-line flash drawn over the viewport (set to 255 by server opcode 192 at Java:7378, decremented `-= 2` per cycle in updateGame at 2935-2937, reset to 0 on login at 2692, rendered at 6560-6565). The Go has no field1264 field anywhere, no decrement in UpdateGame, no reset in LoginFunc, no opcode-192 handler, and no render. The effect is completely absent.

```
Java updateGame: `if (this.field1264 > 0) { this.field1264 -= 2; }` — absent between Go's IdleTimeout decrement (7387) and the readPacket loop (7389). Go grep for Field1264 / the sine-line render returns nothing.
```

> **Verifier (confirmed, severity bug):** Verified all five Java sites verbatim via `git show 01f16088:.../Client.java`: line 563 declares `public int field1264`; line 7378 (server opcode 192 handler, in the readPacket dispatch beside opcode 70 UPDATE_FRIENDLIST) sets `field1264 = 255`; lines 2935-2936 (updateGame) decrement `if (field1264 > 0) { field1264 -= 2; }`; line 2692 (login reset) sets `field1264 = 0`; lines 6560-6565 RENDER it as a fading yellow (0xFFFF00) sine-modulated horizontal-line overlay via `Pix2D.hlineTrans(offset+i, w, 16776960, 256 - w/2, field1264)`. I read Pix2D.hlineTrans (line 208): the final arg is `alpha`, and the body blends the colour into the framebuffer with `invAlpha = 256 - alpha` — so field1264 is g […]
>
> **Refined:** Java field1264 implements a server-triggered viewport flash: opcode 192 (Client.java:7378) sets field1264=255; updateGame (2935-2936) decrements it by 2 per cycle; login (2692) resets it to 0; and the render path (6560-6565) draws a fading yellow (0xFFFF00) sine-modulated horizontal-line overlay using field1264 as the hlineTrans alpha (Pix2D.java:208). The Go port has none of these: no Field1264 field, no opcode-192 handler, no updateGame decrement (client.go:7382-7393), no login reset (client.g […]

### G36. PrepareGameScreen uses rev-225 (789-wide) area PixMap dimensions, not 244's classic 765 layout

- **Unit:** `client/Client#3`  **Java:** `Client.java:2909-2920`  **Go:** `pkg/jagex2/client/client.go:3524-3536`

Java 244 prepareGame allocates AreaMapback(172,156), AreaBackbase1(496,50), AreaBackbase2(269,37), AreaBackhmid1(249,45). The Go allocates AreaMapback(168,160), AreaBackbase1(501,61), AreaBackbase2(288,40), AreaBackhmid1(269,66) — carried over from the rev-225 wider layout (blamed to the original 2025-12-24 deob commit). The corresponding Go draw positions (AreaMapback.Draw(561,5), AreaBackbase1.Draw(0,471), AreaBackbase2.Draw(501,492)) also differ from Java (550,4 / 0,453 / 496,466). The Go's in-game chrome is internally consistent at the 789-wide layout but diverges port-wide from 244's 765 layout; with 244 cache sprites (backbase1/mapback Pix8 native sizes) the chrome would render misaligned. Pre-existing port-wide UI-layout decision, not introduced in this method.

```
Java `new PixMap(172, 156, ...)` / `new PixMap(496, 50, ...)`; Go `pixmap.NewPixMap(168, 160)` / `pixmap.NewPixMap(501, 61)`.
```

> **Verifier (confirmed, severity bug):** Confirmed accurate on every checkable point against the Java-244 reference (git 01f16088).

Allocations (Java prepareGame Client.java:2909-2920 vs Go PrepareGameScreen client.go:3524-3536):
- AreaMapback: Java new PixMap(172,156) vs Go NewPixMap(168,160)
- AreaBackbase1: Java(496,50) vs Go(501,61)
- AreaBackbase2: Java(269,37) vs Go(288,40)
- AreaBackhmid1: Java(249,45) vs Go(269,66)
The shared areas (AreaChatback 479,96 / AreaSidebar 190,261 / AreaViewport 512,334) match exactly; only the four "back" chrome areas diverge.

Draw positions also diverge exactly as cited:
- AreaMapback: Java draw(550,4) [Client.java:5582,5666] vs Go Draw(561,5) [client.go:4290,4361,4545]
- AreaBackbase1: Java(0 […]
>
> **Refined:** Confirmed: Go PrepareGameScreen (client.go:3524-3536) allocates the four in-game "back" chrome PixMaps at rev-225 (789-wide-frame) sizes — AreaMapback(168,160), AreaBackbase1(501,61), AreaBackbase2(288,40), AreaBackhmid1(269,66) — whereas Java-244 prepareGame (Client.java:2909-2920) uses the classic 765-wide-frame sizes 172,156 / 496,50 / 269,37 / 249,45. The corresponding Go draw positions (AreaMapback.Draw 561,5; AreaBackbase1.Draw 0,471; AreaBackbase2.Draw 501,492; AreaBackhmid1.Draw 520,165) […]

### G37. buildScene drops the entire post-build tail: surrounding-zone map prefetch, onDemand.clearPrefetches(), and the lowMem Model.unload loop

- **Unit:** `client/Client#4`  **Java:** `Client.java:3383-3428`  **Go:** `pkg/jagex2/client/client.go:8748-8751`

Java buildScene, after clearLocChanges()/the catch, runs LocType.modelCacheStatic.clear(); a lowMem+cache_dat Model.unload(i) sweep over getFileCount(0)/getModelFlags(i) (flags&0x79==0); System.gc(); Pix3D.initPool(20); this.onDemand.clearPrefetches(); then a left/right/bottom/top edge loop (with withinTutorialIsland override to 49/50/49/50) that calls onDemand.prefetch(3, getMapFile(z,x,0)) and prefetch(3, getMapFile(z,x,1)) for every perimeter zone. The Go BuildScene ends at loctype.ModelCacheStatic.Clear() + pix3d.InitPool(20) and STOPS. The Model.unload loop, clearPrefetches(), and the whole edge-prefetch loop are absent. OnDemand.Prefetch/ClearPrefetches are never called anywhere in client.go (verified by grep), and OnDemand.GetFileCount/GetModelFlags/model.UnloadOne all exist — so this is a port gap, not an API gap. Effect: adjacent map squares are not prefetched as the player approaches square edges, so the cache won't be warm when the scene rebuilds, producing visible 'Loading - please wait' stalls the prefetch was meant to hide; lowMem model eviction also never happens.

```
Java tail: `this.onDemand.clearPrefetches(); int left=(sceneCenterZoneX-6)/8-1; ... for(x=left;x<=right;x++) for(z=bottom;z<=top;z++) if(left==x||...) { int land=onDemand.getMapFile(z,x,0); if(land!=-1) onDemand.prefetch(3,land); ... }`. Go BuildScene last lines: `c.ClearLocChanges() // Java:3379 ; loctype.ModelCacheStatic.Clear() ; pix3d.InitPool(20) }` — nothing after.
```

> **Verifier (confirmed, severity bug):** Verified against Java buildScene() (Client.java:3294, tail at 3383-3428) and Go BuildScene() (client.go:8674-8751).

Java tail after the try/catch contains, in order: (1) LocType.modelCacheStatic.clear(); (2) a lowMem && cache_dat!=null sweep over getFileCount(0) calling Model.unload(i) where (getModelFlags(i)&0x79)==0; (3) System.gc(); (4) Pix3D.initPool(20); (5) this.onDemand.clearPrefetches(); (6) a perimeter edge loop (with withinTutorialIsland override to left=49/right=50/bottom=49/top=50) calling onDemand.prefetch(3, getMapFile(z,x,0)) and prefetch(3, getMapFile(z,x,1)) for each perimeter zone.

Go BuildScene ends at: c.ClearLocChanges() ; loctype.ModelCacheStatic.Clear() ; pix3d.InitP […]
>
> **Refined:** Go BuildScene (pkg/jagex2/client/client.go:8748-8751) omits the entire Java buildScene post-build tail (Client.java:3383-3428): the lowMem+cache Model.unload(i) eviction sweep (flags&0x79==0), System.gc(), onDemand.clearPrefetches(), and the surrounding-zone perimeter map prefetch loop (left=(sceneCenterZoneX-6)/8-1 .. right=(sceneCenterZoneX+6)/8+1, with withinTutorialIsland override to 49/50/49/50, calling onDemand.prefetch(3, getMapFile(z,x,0/1)) for each perimeter zone). The Go function ends […]

### G38. updateAudio deferred-music resume uses stale 225-style SetMidi(MidiCRC,CurrentMidi,MidiSize) instead of 244's onDemand.request(2, nextMidiSong)

- **Unit:** `client/Client#4`  **Java:** `Client.java:3621-3633`  **Go:** `pkg/jagex2/client/client.go:7441-7449`

Java updateAudio's nextMusicDelay block, when the delay reaches 0 and midiActive && !lowMem, does `this.midiSong = this.nextMidiSong; this.midiFading = false; this.onDemand.request(2, this.midiSong);` — i.e. re-requests the background song by numeric id over OnDemand archive 2. The Go inlined copy instead calls `c.SetMidi(c.MidiCRC, c.CurrentMidi, c.MidiSize)`, the rev-225 name/CRC/length mechanism. The 244 server packet handlers (SERVERPROT_MIDI_SONG opcode 240 at Go 9967-9980, SERVERPROT_MIDI_JINGLE opcode 173 at 9983-9993) were correctly migrated to `OnDemand.Request(2, MidiSong)` and never populate MidiCRC/CurrentMidi/MidiSize. So when a jingle with a delay finishes, updateAudio calls SetMidi with empty CurrentMidi, which immediately returns on its `if name=="" return` guard — the background song never resumes. This updateAudio site is a 225 leftover the WS2 protocol migration missed.

```
Java: `this.midiSong=this.nextMidiSong; this.midiFading=false; this.onDemand.request(2,this.midiSong);`. Go: `if c.NextMusicDelay==0 && c.MidiActive && !LowMemory { c.SetMidi(c.MidiCRC, c.CurrentMidi, c.MidiSize) }`. SetMidi: `if name=="" { return }` (client.go:698). MidiCRC/CurrentMidi/MidiSize are never set by the 244 MIDI opcode handlers.
```

> **Verifier (confirmed, severity bug):** Confirmed real after attempting refutation on all common false-positive vectors.

DECISIVE EVIDENCE:
1. Java-244 updateAudio (Client.java:3628-3631) deferred-music resume does: `this.midiSong = this.nextMidiSong; this.midiFading = false; this.onDemand.request(2, this.midiSong);` — re-requests the deferred song by numeric id over OnDemand archive 2.
2. Go (client.go:7446-7448) instead calls `c.SetMidi(c.MidiCRC, c.CurrentMidi, c.MidiSize)` — the rev-225 name/CRC/length mechanism.
3. Java-244 has NO setMidi method and NO midiCrc/currentMidi/midiSize fields whatsoever (grep for `setMidi(`, `midiCrc`, `currentMidi`, `midiSize` in Client.java @01f16088 = zero matches). These are pure 225-lineage  […]
>
> **Refined:** updateAudio's deferred-music resume (client.go:7446-7448) is a rev-225 leftover the WS2 audio migration missed: it calls `c.SetMidi(c.MidiCRC, c.CurrentMidi, c.MidiSize)` where Java-244 (Client.java:3628-3631) does `midiSong = nextMidiSong; midiFading = false; onDemand.request(2, midiSong)`. In the Go port MidiCRC/MidiSize are never written (stay 0) and CurrentMidi is only ever set to "" (client.go:3597); the 244 MIDI opcode handlers (240 @9967, 173 @9983) populate MidiSong/MidiFading/OnDemand.R […]

### G39. UI region geometry uses pre-244 (225-era) screen layout: viewport/sidebar/chat input regions and area draw origins are offset from Java 244

- **Unit:** `client/Client#4`  **Java:** `Client.java:3649,3663,3678,3216,5565-5573,5581`  **Go:** `pkg/jagex2/client/client.go:6144,6155,6167,6148,6157,6171`

handleInput's three hit-test regions and their HandleInterfaceInput base origins use the wrong, non-244 constants. Java 244: viewport region (mouseX>4 && mouseY>4 && mouseX<516 && mouseY<338) with base (4,4); sidebar (553,205,743,466) base (553,205); chat (17,357,426,453) base (17,357), and areaViewport.draw at (4,4), areaBackvmid1 at (516,4), areaBackvmid2 at (516,205), areaBackright2 at (743,205), areaBackleft1 at (0,4). Go uses viewport (8,11,520,345) base (8,11); sidebar (562,231,752,492) base (562,231); chat (22,375,431,471) base (22,375); and AreaViewport.Draw(8,11), AreaBackvmid1.Draw(520,11), AreaBackvmid2.Draw(520,231), AreaBackright2.Draw(752,231), AreaBackleft1.Draw(0,11). The Go input and draw constants are internally self-consistent (both at the shifted origin), so the UI is coherent but shifted from 244 by non-uniform offsets (viewport +4,+7; sidebar +9,+26; chat +5,+18). These constants originate from the original 225 port (git blame: commits 0717143/3b5229d 'Client partial') and were never reconciled to the 244 geometry. Cross-cutting: also affects handlePrivateChatInput bounds (8/520/arg2-11 vs Java 4/516/mouseY-4) and the handleChatMouseInput caller passing MouseY-375 vs Java mouseY-357.

```
Java handleInput: `if (super.mouseX>4 && super.mouseY>4 && super.mouseX<516 && super.mouseY<338)` ... `handleInterfaceInput(mouseX,4,mouseY,4,...)`; areaViewport.draw(super.graphics,4,4). Go: `if c.MouseX>8 && c.MouseY>11 && c.MouseX<520 && c.MouseY<345` ... `HandleInterfaceInput(c.MouseY,c.MouseX,11,...,8,0)`; AreaViewport.Draw(8,11).
```

> **Verifier (confirmed, severity bug):** CONFIRMED real after attempting refutation. The Go UI region/draw constants do not match the Java 244 reference and instead exactly reproduce the 225-clean deob lineage.

Decisive evidence (all traced through full enclosing functions, accounting for the deob argument scramble):

1) Java 244 (01f16088) handleInput (Client.java:3649-3682): viewport hit-test `mouseX>4 && mouseY>4 && mouseX<516 && mouseY<338` with base (4,4); sidebar `553,205,743,466` base (553,205); chat `17,357,426,453` base (17,357). Area draws: areaViewport.draw(...,4,4) (3216), areaBackleft1.draw(...,0,4), areaBackright2.draw(...,743,205), areaBackvmid1.draw(...,516,4), areaBackvmid2.draw(...,516,205) (5565-5571). Canvas =  […]
>
> **Refined:** handleInput's three hit-test regions, their HandleInterfaceInput base origins, the HandlePrivateChatInput bounds, the HandleChatMouseInput y-offset, and the area-draw origins all use 225-era (789x532) constants instead of Java 244's 765x503 layout. Java 244 (Client.java:3649-3682, 3216, 5565-5571): viewport region mouseX>4&&mouseY>4&&mouseX<516&&mouseY<338 base (4,4); sidebar 553,205,743,466 base (553,205); chat 17,357,426,453 base (17,357); areaViewport.draw(4,4); areaBackvmid1(516,4); areaBack […]

### G40. ApplyCutscene passes raw tile indices to GetHeightMapY instead of world coords (camera height computed at wrong tile)

- **Unit:** `client/Client#5`  **Java:** `Client.java:4596,4642 (applyCutscene); cross-site 8322 (CAM_MOVETO)`  **Go:** `pkg/jagex2/client/client.go:8213,8252 (ApplyCutscene); cross-site 10280`

Java getHeightmapY(sceneZ, level, sceneX) right-shifts its scene args >>7 to derive the tile, so callers must pass WORLD coords (tile*128+64). In applyCutscene Java computes x=cutsceneSrcLocalTileX*128+64, z=...*128+64, then calls getHeightmapY(z, level, x) — i.e. the world coords. The Go GetHeightMapY(arg0=level, arg1=sceneX, arg3=sceneZ) likewise does arg1>>7 / arg3>>7. But ApplyCutscene calls GetHeightMapY(c.CurrentLevel, c.CutsceneSrcLocalTileX, c.CutsceneSrcLocalTileZ) — passing the raw G1() tile indices (0-255) instead of var2/var3 (the already-computed world coords on lines 8211-8212). For any tile index < 128, idx>>7==0, so the heightmap is sampled at near-origin, giving a wrong cutscene camera Y. Both the src (8213) and dst (8252) calls are wrong; should be GetHeightMapY(c.CurrentLevel, var2, var3). The identical defect also exists in the CAM_MOVETO handler at line 10280 (should pass c.CameraX/c.CameraZ, not the tile fields) — same root error in another auditor's packet-switch unit.

```
Go 8211-8213: `var2 := c.CutsceneSrcLocalTileX*128 + 64` / `var3 := ...*128 + 64` / `var4 := c.GetHeightMapY(c.CurrentLevel, c.CutsceneSrcLocalTileX, c.CutsceneSrcLocalTileZ) - c.CutsceneSrcHeight` (var2/var3 unused for the height lookup). Java 4596: `int y = this.getHeightmapY(z, this.currentLevel, x) - this.cutsceneSrcHeight;` where x=...*128+64, z=...*128+64. Go GetHeightMapY body (2064-2065): `var5 := arg1 >> 7` / `var6 := arg3 >> 7`.
```

> **Verifier (confirmed, severity bug):** CONFIRMED genuine bug after tracing both callee bodies and the call convention.

Java getHeightmapY(sceneZ, level, sceneX) at Client.java:6486 derives the tile via sceneX>>7 / sceneZ>>7, so callers must pass WORLD coords (tile*128+64). Java applyCutscene (Client.java:4475) computes `x = cutsceneSrcLocalTileX*128+64`, `z = cutsceneSrcLocalTileZ*128+64`, then calls `getHeightmapY(z, currentLevel, x)` — i.e. world coords.

Go GetHeightMapY(arg0, arg1, arg3) at client.go:2063 does `var5 := arg1>>7`, `var6 := arg3>>7`, indexing LevelHeightMap[var7][var5][var6]. Param mapping: arg0=level, arg1=Java sceneX, arg3=Java sceneZ. So the Go-equivalent of Java's `getHeightmapY(z, level, x)` is `GetHeightM […]
>
> **Refined:** In ApplyCutscene (client.go:8213 and 8252), c.GetHeightMapY is called with the raw G1 tile indices (c.CutsceneSrcLocalTileX/Z, c.CutsceneDstLocalTileX/Z; range 0-104) instead of the world coordinates var2/var3 (=tile*128+64) that the function expects and that are computed one/two lines above. GetHeightMapY (client.go:2063) right-shifts its 2nd and 3rd args by 7 to derive the tile (mirroring Java getHeightmapY's sceneX>>7 / sceneZ>>7), so for any tile index < 128 the shift yields 0 and the camera […]

### G41. UpdateMovement uses 225-era walkmerge gate; the new 244 preanim_move/postanim_mode + preanimRouteLength movement-block logic is not ported

- **Unit:** `client/Client#5`  **Java:** `Client.java:4998-5010, 5117-5121 (updateMovement)`  **Go:** `pkg/jagex2/client/client.go:4043-4049, 4138-4140 (UpdateMovement)`

Java-244 updateMovement gates movement during the primary sequence using preanimRouteLength + the new SeqType fields preanim_move / postanim_mode: `if (e.preanimRouteLength > 0 && var3.preanim_move == 0) { e.seqDelayMove++; return; }` and `if (e.preanimRouteLength <= 0 && var3.postanim_mode == 0) { e.seqDelayMove++; return; }`. The Go still has the OLD 225 logic: `if var3.WalkMerge == nil { arg1.SeqTrigger++; return }`. Although seqtype.PreanimMove/PostanimMode were ported in WS4, UpdateMovement does not consume them, and the ClientEntity has no preanimRouteLength field at all. Consequently the tail decrement `if (e.preanimRouteLength > 0) e.preanimRouteLength--` (Java 5119-5121) is also missing from the Go (4138-4140 only does PathLength--). Result: animations that should block movement (244 behavior) won't, producing wrong entity movement/animation timing.

```
Go 4043-4049: `if arg1.PrimarySeqID != -1 && arg1.PrimarySeqDelay == 0 { var3 := seqtype.Instances[arg1.PrimarySeqID]; if var3.WalkMerge == nil { arg1.SeqTrigger++; return } }`. Java 4998-5010 uses preanimRouteLength + var3.preanim_move/postanim_mode. grep confirms no PreanimRouteLength field anywhere in pkg/jagex2/dash3d/entity or client.go.
```

> **Verifier (confirmed, severity bug):** Confirmed real after ruling out the equivalence false-positive. Go UpdateMovement (client.go:4043-4049) uses the 225 single-condition gate `if var3.WalkMerge == nil { arg1.SeqTrigger++; return }`. Java-244 updateMovement (Client.java:5004-5010) uses a two-branch gate: `if (e.preanimRouteLength > 0 && var3.preanim_move == 0) block` and `if (e.preanimRouteLength <= 0 && var3.postanim_mode == 0) block`.

Refutation attempt (equivalence): SeqType.postEncode derives preanim_move/postanim_mode from walkmerge when -1 (walkmerge==null → 0/0; walkmerge!=null → 2/2), and the Go parser ports this defaulting (seqtype.go:91-104). For seqs that DON'T set the new decode opcodes, the 225 `WalkMerge==nil` ga […]
>
> **Refined:** Go UpdateMovement (pkg/jagex2/client/client.go:4043-4049) still uses the rev-225 movement-block gate `if var3.WalkMerge == nil { arg1.SeqTrigger++; return }`, whereas Java-244 updateMovement (Client.java:5004-5010) gates on preanimRouteLength plus the new SeqType fields preanim_move/postanim_mode: `if (preanimRouteLength > 0 && preanim_move == 0) block` and `if (preanimRouteLength <= 0 && postanim_mode == 0) block`. These coincide only for seqs that leave decode opcodes 9 (preanim_move) and 10 ( […]

### G42. HandleInputKey ::command handling drops staffmodlevel==2 gate, omits CLIENT_CHEAT for ::clientdrop, and drops ::lag / ::prefetchmusic

- **Unit:** `client/Client#5`  **Java:** `Client.java:4701-4718 (handleInputKey)`  **Go:** `pkg/jagex2/client/client.go:2278-2283 (HandleInputKey)`

Java-244 structures this as two SEPARATE statements: (1) `if (staffmodlevel == 2) { if ::clientdrop tryReconnect() else if ::lag lag() else if ::prefetchmusic <prefetch loop> }` then (2) an unconditional `if (chatTyped.startsWith("::")) { pIsaac(76 CLIENT_CHEAT); p1(len-1); pjstr(substring(2)) }`. So in 244, ANY ::-prefixed text always sends CLIENT_CHEAT, and only staff (level 2) additionally fire the local commands. The Go uses a single if/else-if chain: `if c.ChatTyped == "::clientdrop" { c.TryReconnect() } else if strings.HasPrefix(c.ChatTyped, "::") { CLIENT_CHEAT }`. Divergences: (a) ::clientdrop is no longer gated by staffmodlevel==2 (any user can force a reconnect); (b) because of the else-if, ::clientdrop no longer ALSO sends CLIENT_CHEAT to the server (Java does — separate non-else if) — a protocol behavior change; (c) ::lag and ::prefetchmusic local commands are entirely absent (::lag falls through to plain CLIENT_CHEAT, never calling lag()). The doc-comment justifies the simplification via the 225 super.frame host-check, but the authoritative 244 reference has no host check — it has the staffmodlevel==2 guard.

```
Go 2278-2283: `if c.ChatTyped == "::clientdrop" { c.TryReconnect() } else if strings.HasPrefix(c.ChatTyped, "::") { c.Out.P1Isaac(io.CLIENTPROT_CLIENT_CHEAT) ... }`. Java 4701-4711: `if (this.staffmodlevel == 2) { if (...equals("::clientdrop")) tryReconnect(); else if (..."::lag") lag(); else if (..."::prefetchmusic") for(...) onDemand.prefetchPriority(2,i,(byte)1); }` then 4713 `if (this.chatTyped.startsWith("::")) { pIsaac(76); ... }`.
```

> **Verifier (confirmed, severity bug):** Verified against authoritative Java-244 at 01f16088 (Client.java:4699-4719). Java-244 structures this as TWO statements: (1) `if (staffmodlevel == 2) { if ::clientdrop tryReconnect(); else if ::lag lag(); else if ::prefetchmusic <for getFileCount(2): prefetchPriority(2,i,1)> }` then (2) a SEPARATE non-else `if (chatTyped.startsWith("::")) { pIsaac(76 CLIENT_CHEAT); p1(len-1); pjstr(substring(2)) }`. Go (client.go:2278-2283) collapses this into one if/else-if chain: `if ChatTyped=="::clientdrop" { TryReconnect() } else if HasPrefix("::") { CLIENT_CHEAT... }`.\n\nConfirmed divergences, none refutable as deob-scramble/rename/equivalent-control-flow:\n(a) Dropped staffmodlevel==2 gate — Go lets  […]
>
> **Refined:** Confirmed exactly as described. Java-244 (Client.java:4700-4716) gates ::clientdrop/::lag/::prefetchmusic behind `staffmodlevel == 2` (lines 4701-4711) and then unconditionally sends CLIENT_CHEAT for any `::`-prefixed text via a SEPARATE `if (chatTyped.startsWith(\"::\"))` (line 4713). Go (client.go:2278-2283) instead uses a single if/else-if: `if ChatTyped==\"::clientdrop\" TryReconnect() else if HasPrefix(\"::\") CLIENT_CHEAT`. Three divergences: (a) ::clientdrop ungated — any user can force r […]

### G43. updateSequences: missing rev-244 preanim_move early-trigger block (still 225-lineage)

- **Unit:** `client/Client#6`  **Java:** `Client.java:5229-5235`  **Go:** `pkg/jagex2/client/client.go:4202-4256`

Java-244 updateSequences contains a block that, when primarySeqId != -1 && primarySeqDelay <= 1 and the SeqType.preanim_move == 1 with preanimRouteLength > 0 and forceMoveEndCycle <= loopCycle && forceMoveStartCycle < loopCycle, sets primarySeqDelay = 1 and RETURNS early (deferring the primary-seq advance). The Go UpdateSequences has no such block. Verified against 225-clean (deob/client.java updateSequences at ~4596-4652) that this block does not exist in 225 — i.e. Go is a faithful 225 port that was NOT migrated to the 244 logic. The Go ClientEntity also entirely lacks the preanimRouteLength field (pkg/jagex2/dash3d/entity/clententity.go has no preanim* field), so the block cannot currently be ported without adding that field (which 244 also sets at player/npc decode, Java Client.java:9177, 9498). TS client confirms the 244 structure (Client-TS Client.ts:4002-4009). Behavioral effect: forced-movement animations during pathing advance their primary sequence one tick early vs the 244 reference.

```
Java-244: `if (e.primarySeqId != -1 && e.primarySeqDelay <= 1) { SeqType seq = ...; if (seq.preanim_move == 1 && e.preanimRouteLength > 0 && e.forceMoveEndCycle <= loopCycle && e.forceMoveStartCycle < loopCycle) { e.primarySeqDelay = 1; return; } }`. Go has no equivalent; jumps from the secondary-seq block straight to the primary-seq advance block.
```

> **Verifier (confirmed, severity bug):** Confirmed real after attempting to refute. Decisive evidence:

1. Java-244 Client.java:5227-5235 (updateSequences) contains the early-trigger block exactly as described: `if (e.primarySeqId != -1 && e.primarySeqDelay <= 1) { SeqType seq = SeqType.types[e.primarySeqId]; if (seq.preanim_move == 1 && e.preanimRouteLength > 0 && e.forceMoveEndCycle <= loopCycle && e.forceMoveStartCycle < loopCycle) { e.primarySeqDelay = 1; return; } }`.

2. Go UpdateSequences (pkg/jagex2/client/client.go:4202-4256) has no equivalent block — it goes straight from the secondary-seq block to the `PrimarySeqID != -1 && PrimarySeqDelay == 0` primary-advance block.

3. Ruled out cross-lineage rename / behavior-elsewhe […]

### G44. Title-screen family (loadTitle/loadTitleBackground/drawTitleScreen) still uses 225 layout, not rev-244

- **Unit:** `client/Client#6`  **Java:** `Client.java:5268-5557`  **Go:** `pkg/jagex2/client/client.go:6597-6664,9034-9107,3418-3504`

The whole title-screen rendering family is a faithful 225-lineage port and was not migrated to 244's layout. Concrete 244 deltas NOT reflected in Go: (1) PixMap dimensions in loadTitle — 244 imageTitle2=509x171, imageTitle3=360x132, imageTitle5=202x238, imageTitle6=203x238, imageTitle7=74x94, imageTitle8=75x94; Go uses 533x186/360x146/214x267/215x267/86x79/87x79 (the 225 values, verified at 225-clean deob/client.java:6669-6691). (2) loadTitleBackground quickPlotSprite/blit offsets — 244 uses -637/-202/-371/-562 etc.; Go uses -661/-214/-386/-574 (225 values, verified deob/client.java:8784-8835). (3) logo placement — 244 `plotSprite(382 - wi/2 - 128, 18)` (hard 382); Go uses ScreenWidth/2-Wi/2-128 (225). (4) drawTitleScreen final draw coords — 244 imageTitle4.draw(202,171)/imageTitle3.draw(202,371)/imageTitle6.draw(562,265)/imageTitle8.draw(562,171); Go uses 214/186, 214/386, 574/265, 574/186 (225). (5) titleScreenState==0 — 244 draws an extra fontPlain11.centreStringTag(w/2, onDemand.message, h/2+80, 0x75a9a9) status line that Go (and 225) omit. Effect: title/login screen is laid out for rev-225 asset geometry, not rev-244 — visibly wrong tiling and a missing fileserver status message on the welcome screen for a true 244 client.

```
Go LoadTitle: `c.ImageTitle2 = pixmap.NewPixMap(533, 186)` vs Java-244 `new PixMap(509, 171)`. Go DrawTitleScreen state 0 has only `DrawStringTaggableCenter(var2/2, ..., "Welcome to RuneScape")` with no onDemand.message line; Java-244 line 5485 draws `this.onDemand.message` first.
```

> **Verifier (confirmed, severity bug):** All five sub-deltas verified against authoritative Java-244 (git 01f16088) and cross-checked against 225-clean deob to prove genuine 225-vs-244 divergence (not name-reuse, not arg-order scramble).

(1) loadTitle PixMap dims — Java-244 (Client.java:5288-5306): imageTitle2=509x171, imageTitle3=360x132, imageTitle5=202x238, imageTitle6=203x238, imageTitle7=74x94, imageTitle8=75x94. Go LoadTitle (client.go:6645-6657): 533x186, 360x146, 214x267, 215x267, 86x79, 87x79 — byte-identical to 225-clean deob/client.java:6673-6686. Confirmed.

(2) loadTitleBackground blit offsets — Java-244 (5316-5343): imageTitle1=-637, imageTitle3=-202/-371, imageTitle6=-562/-265, etc., and mirror side 382/-255/254/180 […]
>
> **Refined:** Confirmed as written. The entire title-screen rendering family (LoadTitle client.go:6597-6664, LoadTitleBackground 9034-9107, DrawTitleScreen 3418-3504) is a faithful 225-lineage port and was not migrated to rev-244 geometry. All five enumerated deltas (PixMap dims, blit offsets, logo x=ScreenWidth/2 vs hard 382, final draw coords, and the missing fontPlain11 onDemand.message status line in titleScreenState==0) are verified against Java-244 (git 01f16088, Client.java:5268-5557) and shown to matc […]

### G45. drawGame still uses 225 area-tile/redstone/sideicon coordinates, not rev-244

- **Unit:** `client/Client#6`  **Java:** `Client.java:5561-5836`  **Go:** `pkg/jagex2/client/client.go:4258-4509`

DrawGame is a faithful 225-lineage port (verified coord-for-coord against 225-clean deob/client.java:4655-4845). Java-244 uses a different screen layout: e.g. areaBackleft1.draw at y=4 (Go/225 y=11), areaBackright1 at x=722 (Go/225 x=729), areaViewport.draw(4,4) (Go/225 x=8,y=11), areaMapback at (550,4) (Go/225 561,5), and the redstone/sideicon plotSprite coordinates differ (244 imageRedstone1.plotSprite(22,10) vs Go/225 (30,29); 244 imageSideicons[0].plotSprite(29,13) vs Go/225 (34,35)). The opcodes WERE migrated to 244 (TUTORIAL_CLICKSIDE=233 matches Java-244:5677). Logic structure matches; only the pixel geometry is 225. Effect: in-game HUD chrome, sidebar tabs and minimap are positioned for the 225 frame, not 244.

```
Go: `c.AreaBackleft1.Draw(0, 11)` / `c.AreaViewport.Draw(8, 11)` / `c.AreaMapback.Draw(561, 5)`; Java-244: `areaBackleft1.draw(super.graphics, 0, 4)` / `areaViewport.draw(super.graphics, 4, 4)` / `areaMapback.draw(super.graphics, 550, 4)`.
```

> **Verifier (confirmed, severity bug):** Verified coord-for-coord against both references. Java-244 (git show 01f16088:src/main/java/jagex2/client/Client.java, drawGame ~L5561) and 225-clean (git show 225-clean:src/main/java/deob/client.java, drawGame ~L4655) confirm the Go DrawGame (pkg/jagex2/client/client.go:4258-4509) uses 225 geometry throughout, not 244.

Decisive evidence (note the 225-clean deob draw signature is draw(y, graphics, x) — y first, x last — which the Go port renders as Draw(x, y)):
- AreaBackleft1: 225-clean draw(11,_,0) → Go Draw(0,11) MATCHES 225; Java-244 draw(_,0,4)
- AreaBackright1: 225-clean (y5,x729) → Go Draw(729,5); Java-244 (722,4)
- AreaViewport: 225-clean (y11,x8) → Go Draw(8,11); Java-244 (4,4)
- A […]

### G46. drawScene uses 225 npc/player push structure (single pushNpcs) instead of 244 two-pass pushNpcs(true/false)

- **Unit:** `client/Client#6`  **Java:** `Client.java:5839-5846`  **Go:** `pkg/jagex2/client/client.go:1450-1456`

Java-244 drawScene pushes entities as: pushNpcs(true); pushPlayers(); pushNpcs(false); pushProjectiles(); pushSpotanims() — splitting NPC submission into an always-on-top pass before players and a normal pass after. The Go DrawScene uses the 225 ordering pushPlayers(); PushNPCs() (single call, no bool) followed by `c.PacketSize += arg0` (a 225-only anticheat-padding line absent from 244's signature). Verified faithful to 225-clean (deob/client.java:2018-2025). The 244 two-pass split affects draw/occlusion ordering of always-on-top NPCs relative to players. Note Go correctly omits pushLocs() here (244 dropped it after the LocChange merge; 225 had it at deob/client.java:2026) and uses 244 opcode 148 for cyclelogic2, so this method is a 225/244 hybrid.

```
Go: `c.PushPlayers(); c.PushNPCs(); c.PacketSize += arg0`. Java-244: `this.pushNpcs(true); this.pushPlayers(); this.pushNpcs(false);` (no packetSize line; method signature is `drawScene()` not `drawScene(int)`).
```

> **Verifier (confirmed, severity bug):** Confirmed at every layer. Java-244 drawScene() (Client.java:5841-5844) does a two-pass NPC submission: pushNpcs(true); pushPlayers(); pushNpcs(false). pushNpcs(boolean alwaysontop) (Client.java:6002-6007) filters each NPC with `npc.type.alwaysontop != alwaysontop`, so the true-pass submits only always-on-top NPCs BEFORE players and the false-pass submits the rest AFTER players.

Go DrawScene (client.go:1452-1453) instead does PushPlayers(); PushNPCs() — the 225 ordering — followed by the 225-only `c.PacketSize += arg0` padding (Go's method is DrawScene(int) vs 244's parameterless drawScene()). Go PushNPCs (client.go:3390-3408) is a SINGLE pass with NO `AlwaysOnTop` filter: it submits every v […]
>
> **Refined:** Go DrawScene (pkg/jagex2/client/client.go:1450-1456) uses the 225 NPC/player push structure instead of the 244 two-pass split. Two fixes are required: (1) DrawScene must call PushNPCs(true); PushPlayers(); PushNPCs(false) to match Java-244 drawScene() (Client.java:5841-5844), submitting always-on-top NPCs before players and the rest after; and (2) PushNPCs (client.go:3390) must take a bool parameter and add the filter `if var3.Type.AlwaysOnTop != alwaysontop { continue }` per Java-244 pushNpcs(b […]

### G47. drawScene 3D mouse-pick offset uses 225 viewport origin (mouseX-8 / mouseY-11) not 244 (-4 / -4)

- **Unit:** `client/Client#6`  **Java:** `Client.java:5928-5929`  **Go:** `pkg/jagex2/client/client.go:1531-1532`

Java-244 sets Model.mouseX = super.mouseX - 4 and Model.mouseY = super.mouseY - 4 (244 viewport is at 4,4). Go sets model.MouseX = c.MouseX - 8 and model.MouseZ = c.MouseY - 11 (225 viewport at 8,11). This is consistent with the 225 viewport geometry used throughout DrawGame/DrawScene. Effect: world hover/click picking is offset by (4,7) pixels vs a true 244 client, because the pick origin matches the 225 viewport position. Part of the same 225-layout-not-migrated cluster.

```
Go: `model.MouseX = c.MouseX - 8` / `model.MouseZ = c.MouseY - 11`; Java-244: `Model.mouseX = super.mouseX - 4` / `Model.mouseY = super.mouseY - 4`. (Go field name MouseZ vs Java mouseY is the proven 244/225 name divergence, not a bug.)
```

> **Verifier (confirmed, severity bug):** Confirmed by direct comparison of both code sites in full enclosing-function context.

Java-244 drawScene (Client.java:5926-5938):
  Model.mouseX = super.mouseX - 4;
  Model.mouseY = super.mouseY - 4;
  ...
  this.areaViewport.draw(super.graphics, 4, 4);
And every other areaViewport.draw in Java-244 (lines 3216, 3240, 5581, 7732) also uses (4, 4). The viewport buffer is `new PixMap(512, 334)`.

Go drawScene (client.go:1531-1540):
  model.MouseX = c.MouseX - 8
  model.MouseZ = c.MouseY - 11
  ...
  c.AreaViewport.Draw(8, 11)
Both the other Draw site (client.go:4289) also uses (8, 11). These are the 225-lineage viewport-origin constants.

Decisive corroboration ruling out a "consistent/intenti […]
>
> **Refined:** drawScene sets the 3D mouse-pick origin to the 225 viewport position (model.MouseX = c.MouseX - 8, model.MouseZ = c.MouseY - 11) where Java-244 uses the 244 viewport origin (Model.mouseX = super.mouseX - 4, Model.mouseY = super.mouseY - 4). The companion c.AreaViewport.Draw(8, 11) (client.go:1540 and 4289) likewise uses 225's (8, 11) where Java-244 uses (4, 4). Notably the viewport BUFFER was already migrated to 244 (AreaViewport = pixmap.NewPixMap(512, 334) at client.go:3529, matching Java-244  […]

### G48. PushNPCs drops the 244 alwaysontop two-pass split (missing parameter and filter)

- **Unit:** `client/Client#7`  **Java:** `Client.java:6002-6028, callers 5842/5844`  **Go:** `pkg/jagex2/client/client.go:3390-3408, caller 1453`

Java 244 declares pushNpcs(boolean alwaysontop) and the visibility guard includes `|| npc.type.alwaysontop != alwaysontop`, so drawScene calls it TWICE: pushNpcs(true) before pushPlayers and pushNpcs(false) after. This renders alwaysontop NPCs in a different scene-insertion pass (depth ordering). The Go PushNPCs() takes no parameter, omits the `npc.type.alwaysontop != alwaysontop` clause entirely, and is invoked exactly ONCE from DrawScene. NpcType.AlwaysOnTop exists in Go (npctype.go:47, set by a 244 opcode) but is never consulted here. Result: alwaysontop NPCs are drawn in the wrong order relative to players, and the second pass is missing.

```
Java guard: `if (npc == null || !npc.isVisible() || npc.type.alwaysontop != alwaysontop) { continue; }`; Go guard: `if var3 != nil && var3.IsVisible() {` (no alwaysontop term). Java drawScene: `this.pushNpcs(true); this.pushPlayers(); this.pushNpcs(false);` vs Go DrawScene: `c.PushPlayers(); c.PushNPCs();`
```

> **Verifier (confirmed, severity bug):** Verified against Java-244 (git show 01f16088:Client.java) and the Go rev-244 branch; could not refute.

Java-244 pushNpcs declares `pushNpcs(boolean alwaysontop)` (line 6002) with the visibility guard `if (npc == null || !npc.isVisible() || npc.type.alwaysontop != alwaysontop) continue;` (line 6007), and drawScene (lines 5842-5844) calls it as a two-pass split: `pushNpcs(true); pushPlayers(); pushNpcs(false);`.

Go PushNPCs (client.go:3390) takes no parameter and its guard is only `if var3 != nil && var3.IsVisible()` — the `npc.type.alwaysontop != alwaysontop` clause is entirely absent. DrawScene (client.go:1452-1453) invokes it exactly once as `c.PushPlayers(); c.PushNPCs();`.

Ruled out fa […]
>
> **Refined:** PushNPCs drops the 244 alwaysontop two-pass split. Java-244 declares pushNpcs(boolean alwaysontop) with the guard term `|| npc.type.alwaysontop != alwaysontop` and drawScene calls pushNpcs(true) before pushPlayers and pushNpcs(false) after, so alwaysontop NPCs are inserted into the scene before players and the rest after. The Go PushNPCs() (client.go:3390) takes no parameter, omits the alwaysontop filter, and is called once (client.go:1453) as PushPlayers() then PushNPCs(). NpcType.AlwaysOnTop ( […]

### G49. Draw2DEntityElements: NPC headicon (244 NpcType.headicon) never rendered

- **Unit:** `client/Client#7`  **Java:** `Client.java:6239-6248`  **Go:** `pkg/jagex2/client/client.go:720-747`

Java 244 NPC branch draws imageHeadicon[npc.headicon] (plotSprite at projectX-12, projectY-30) when `npc.headicon >= 0 && npc.headicon < imageHeadicon.length`, in addition to the hintType==1 marker. The Go collapses the player/NPC dispatch into one `if i < c.PlayerCount { ... } else if c.HintType == 1 && ... {`, so for the NPC case it ONLY runs the hintType==1 logic and never renders the NPC headicon. NpcType.HeadIcon exists in Go (npctype.go:48, default -1, set by 244 opcode at npctype.go:185) but Draw2DEntityElements never reads it.

```
Java: `NpcType npc = ((ClientNpc) entity).type; if (npc.headicon >= 0 && npc.headicon < this.imageHeadicon.length) { this.projectFromEntity(entity.height + 15, entity); if (this.projectX > -1) { this.imageHeadicon[npc.headicon].plotSprite(this.projectX - 12, this.projectY - 30); } }`. Go NPC branch has no such block — jumps straight to `} else if c.HintType == 1 && ...`.
```

> **Verifier (confirmed, severity bug):** Confirmed against Java 01f16088. In Client.java the NPC arm (`index >= playerCount`) contains TWO independent `if` blocks: (1) `if (npc.headicon >= 0 && npc.headicon < imageHeadicon.length) { projectFromEntity(height+15); if (projectX>-1) imageHeadicon[npc.headicon].plotSprite(projectX-12, projectY-30); }` and (2) the separate `if (hintType==1 && ...)` marker. The Go (client.go:711-747) collapses the dispatch into `if i < c.PlayerCount {...} else if c.HintType == 1 && c.HintNPC == ... {...}`, so the NPC arm executes ONLY the hintType==1 marker and the npc.headicon block is entirely absent.\n\nRefutation attempts all failed: (a) Not a name inversion — Java NpcType.headicon (NpcType.java:97/23 […]
>
> **Refined:** In Draw2DEntityElements (client.go:708-747) the player/NPC dispatch is collapsed into `if i < c.PlayerCount {...} else if c.HintType == 1 && ... {...}`. The Java NPC arm (Client.java:6240-6256) has TWO independent blocks: a headicon block `if (npc.headicon >= 0 && npc.headicon < imageHeadicon.length) { projectFromEntity(height+15); if (projectX>-1) imageHeadicon[npc.headicon].plotSprite(projectX-12, projectY-30); }` plus the separate `if (hintType==1 ...)` marker. The Go has no headicon block; i […]

### G50. Draw2DEntityElements still uses 225 single-hitmark/health model instead of 244 four-slot damage arrays

- **Unit:** `client/Client#7`  **Java:** `Client.java:6308-6342 (+ ClientEntity.java:98-107,266-269)`  **Go:** `pkg/jagex2/client/client.go:769-785 (+ clententity.go:22-24)`

Java 244 ClientEntity has damage[4], damageType[4], damageCycle[4] and combatCycle. The health bar triggers on `entity.combatCycle > loopCycle`; hitmarks loop `for (i=0;i<4;i++) if (entity.damageCycle[i] > loopCycle)` and position up to 4 marks (i==1: y-=20; i==2: x-=15,y-=10; i==3: x+=15,y-=10) using imageHitmark[damageType[i]] and damage[i]. The Go ClientEntity has only scalar Damage/DamageType and triggers health bar on `CombatCycle > LoopCycle+100` and a SINGLE hitmark on `CombatCycle > LoopCycle+330` (the 225-era constants). The 4-slot 244 hitmark model is entirely unported here (and the underlying ClientEntity fields are still 225-shaped), so simultaneous/positioned hitmarks and the correct health-bar timing are wrong.

```
Java: `if (entity.combatCycle > loopCycle) {...health...} for (int i = 0; i < 4; i++) { if (entity.damageCycle[i] > loopCycle) { ... this.imageHitmark[entity.damageType[i]].plotSprite(...); ... String.valueOf(entity.damage[i]) ... } }`. Go: `if pe.CombatCycle > clientextras.LoopCycle+100 {...health...} if pe.CombatCycle > clientextras.LoopCycle+330 { c.ImageHitmarks[pe.DamageType].PlotSprite(...); strconv.Itoa(pe.Damage) }`. Go clententity.go has `Damage int; DamageType int; CombatCycle int` (no [4] arrays).
```

> **Verifier (confirmed, severity bug):** Confirmed against Java-244 reference (git 01f16088) and Go working tree; could not refute.

DECISIVE EVIDENCE:

1. Java-244 ClientEntity (dash3d/ClientEntity.java:98-107) declares `int[] damage = new int[4]`, `int[] damageType = new int[4]`, `int[] damageCycle = new int[4]`, and a SEPARATE `int combatCycle = -1000`. Go ClientEntity (clententity.go:22-24) has only scalar `Damage int; DamageType int; CombatCycle int` (plus Health/TotalHealth). The 4-slot fields are not ported at all — confirmed no `damageCycle`, `damage[]`, or `hit()` exist anywhere in pkg/jagex2 (grep clean).

2. Java draw (Client.java:6308-6342): health bar triggers on `entity.combatCycle > loopCycle`; hitmarks are a `for (i […]

### G51. Opcode 44 LAST_LOGIN_INFO under-reads warnMembersInNonMembers (g1) and uses wrong clientCode condition

- **Unit:** `client/Client#8`  **Java:** `Client.java:7275-7304`  **Go:** `pkg/jagex2/client/client.go:10138-10160`

Java reads FIVE fields: lastAddress=g4, daysSinceLogin=g2, daysSinceRecoveriesChanged=g1, unreadMessageCount=g2, warnMembersInNonMembers=g1 (total 10 bytes = SERVERPROT_LENGTH[44]). Go reads only FOUR: LastAddress=G4, DaysSinceLastLogin=G2, DaysSinceRecoveriesChanged=G1, UnreadMessages=G2 (9 bytes; the trailing g1 is never consumed and the field does not exist in Go — grep WarnMembersInNonMembers returns nothing). Java's welcome-screen selection is `if (daysSinceRecoveriesChanged != 201 || warnMembersInNonMembers == 1) clientCode = 655;` whereas Go is only `if c.DaysSinceRecoveriesChanged != 201 { var47 = 655 }`. So when daysSinceRecoveriesChanged==201 AND warnMembersInNonMembers==1, Java shows interface clientCode 655; Go wrongly shows 650. The 1-byte under-read does NOT desync subsequent packets (framing resets In.Pos=0 and reads psize fresh each packet), so this is a bug (wrong welcome interface), not a blocker.

```
Java: `this.warnMembersInNonMembers = this.in.g1(); ... if (this.daysSinceRecoveriesChanged != 201 || this.warnMembersInNonMembers == 1) { clientCode = 655; }`  |  Go: `c.UnreadMessages = c.In.G2()  // (no 5th g1 read) ... if c.DaysSinceRecoveriesChanged != 201 { var47 = 655 }`
```

> **Verifier (confirmed, severity bug):** All four core claims of the finding are verified against the Java-244 reference (git 01f16088) and the Go source.

1. UNDER-READ CONFIRMED. Java opcode-44 handler (Client.java:7275-7281) reads five fields: lastAddress=g4 (4B), daysSinceLogin=g2 (2B), daysSinceRecoveriesChanged=g1 (1B), unreadMessageCount=g2 (2B), warnMembersInNonMembers=g1 (1B) = 10 bytes. Go (client.go:10139-10142) reads only LastAddress=G4, DaysSinceLastLogin=G2, DaysSinceRecoveriesChanged=G1, UnreadMessages=G2 = 9 bytes; the trailing g1 is never consumed. Both SERVERPROT length tables agree index 44 = 10 (Java Protocol.java:12 and Go protocol.go:10), so Go under-reads the declared frame by exactly 1 byte.

2. FIELD ABSENT […]
>
> **Refined:** Opcode 44 LAST_LOGIN_INFO: Go under-reads the frame by 1 byte (omits the 5th field warnMembersInNonMembers = g1; reads 9 of the declared 10 bytes, SERVERPROT length[44]=10) and the field does not exist in Go at all. The interface-selection condition drops `|| warnMembersInNonMembers == 1` (Go only tests DaysSinceRecoveriesChanged != 201), so when daysSinceRecoveriesChanged==201 && warnMembersInNonMembers==1 Java picks clientCode 655 but Go picks 650. The same missing field also makes the Go welc […]

### G52. Opcode 30 MESSAGE_PRIVATE collapses three staff-level branches into two — wrong message type and missing crown prefix

- **Unit:** `client/Client#8`  **Java:** `Client.java:8396-8404`  **Go:** `pkg/jagex2/client/client.go:9656-9660`

Java has THREE branches on staffModLevel: (==2||==3) -> addMessage(filtered, "@cr2@"+name, 7); (==1) -> addMessage(filtered, "@cr1@"+name, 7); else -> addMessage(filtered, name, 3). Go has only TWO: `if var6 > 1 { AddMessage(7, var38, FormatName(...)) } else { AddMessage(3, var38, FormatName(...)) }`. Two defects: (1) staffModLevel==1 (a moderator, var6==1) falls into Go's else and is shown as a normal player PM (type 3, no crown) instead of type 7 with the "@cr1@" crown; (2) for staffModLevel 2/3 Go uses the correct type 7 but DROPS the "@cr2@" crown prefix on the sender. The crown tags are never prepended in Go.

```
Java: `if (staffModLevel == 2 || staffModLevel == 3) { addMessage(filtered, "@cr2@" + ...name, 7); } else if (staffModLevel == 1) { addMessage(filtered, "@cr1@" + ...name, 7); } else { addMessage(filtered, ...name, 3); }`  |  Go: `if var6 > 1 { c.AddMessage(7, var38, jstring.FormatName(...)) } else { c.AddMessage(3, var38, jstring.FormatName(...)) }`
```

> **Verifier (confirmed, severity bug):** Confirmed against Java 01f16088 Client.java:8396-8404 and Go pkg/jagex2/client/client.go:9656-9660. Java has THREE branches on staffModLevel; Go has only TWO (`if var6 > 1 ... else ...`).

Argument mapping is consistent (not a deob swap): Java `addMessage(String text, String sender, int type)` (Client.java:12110); Go `AddMessage(arg0 int type, arg1 string text, arg3 string sender)` (client.go:7985). Go reorders type-first. Go's AddMessage body is a faithful 1:1 port that stores `sender` raw into MessageSender[0] with no crown re-addition (verified 7985-8001). FormatName/formatDisplayName only does underscore-to-space + capitalization, no crown logic (verified both bodies). So the crown is ge […]
>
> **Refined:** Opcode 30 MESSAGE_PRIVATE: Go collapses Java's three staffModLevel branches into two (`if var6 > 1 ... else ...`). Confirmed bug. (1) staffModLevel==1 (moderator) falls into Go's else and is tagged message type 3 instead of Java's type 7; type 3 is subject to the PrivateChatSetting filter at client.go:1982/5228 whereas type 7 bypasses it, so the mod's PM can be wrongly suppressed and report-abuse menu eligibility differs. (2) The "@cr2@"/"@cr1@" crown prefix that Java prepends to the sender is n […]

### G53. Opcode 12 CAM_MOVETO passes tile coords (not scene coords) to GetHeightMapY — wrong camera Y

- **Unit:** `client/Client#8`  **Java:** `Client.java:8308-8325`  **Go:** `pkg/jagex2/client/client.go:10270-10283`

Java: cameraX = cutsceneSrcLocalTileX*128+64; cameraZ = cutsceneSrcLocalTileZ*128+64; cameraY = getHeightmapY(cameraZ, currentLevel, cameraX) - cutsceneSrcHeight — i.e. it passes the SCENE coordinates (tile*128+64). getHeightmapY does sceneX>>7 / sceneZ>>7 to recover the tile and sceneX&0x7F for sub-tile interpolation. Go computes c.CameraX/c.CameraZ = ...*128+64 but then calls `c.GetHeightMapY(c.CurrentLevel, c.CutsceneSrcLocalTileX, c.CutsceneSrcLocalTileZ)` passing the raw TILE coords. Inside GetHeightMapY (client.go:2063-2074) arg1>>7 of a tile value 0-127 yields 0 or 1, so the height is sampled at tile (0|1,0|1) instead of the real cutscene tile — wrong camera Y during CAM_MOVETO cutscenes. Should pass c.CameraX, c.CameraZ. (Same defect exists in ApplyCutscene client.go:8213,8252 and in opcode 222 below.)

```
Java: `this.cameraY = this.getHeightmapY(this.cameraZ, this.currentLevel, this.cameraX) - this.cutsceneSrcHeight;` (cameraZ/cameraX are *128+64)  |  Go: `c.CameraY = c.GetHeightMapY(c.CurrentLevel, c.CutsceneSrcLocalTileX, c.CutsceneSrcLocalTileZ) - c.CutsceneSrcHeight`
```

> **Verifier (confirmed, severity bug):** Confirmed and could not refute. Traced argument semantics through both callee bodies. Java getHeightmapY(sceneZ, level, sceneX) does sceneX>>7 / sceneZ>>7 to recover the tile and sceneX&0x7F for sub-tile interpolation, so it requires SCENE coords (tile*128+64). Go GetHeightMapY(arg0=level, arg1, arg3) is identical: var5=arg1>>7, var6=arg3>>7, &0x7F interp — i.e. Go's convention is (level, sceneX, sceneZ). The correct Go callsites confirm this: line 1439 passes c.CameraX/c.CameraZ, line 6300 passes c.OrbitCameraX/Z, line 8601 passes arg0*128+64/arg1*128+64. The four cutscene callsites are wrong: opcode 12 (client.go:10280), opcode 222 (10356), and ApplyCutscene (8213, 8252) all pass the RAW t […]
>
> **Refined:** Opcode 12 (CAM_MOVETO, client.go:10280), opcode 222 (CAM_LOOKAT, client.go:10356), and ApplyCutscene (client.go:8213 src, 8252 dst) all call c.GetHeightMapY with raw cutscene TILE coords (CutsceneSrc/DstLocalTileX/Z, 0-127) instead of the SCENE coords (tile*128+64). GetHeightMapY(level, sceneX, sceneZ) internally does sceneX>>7/sceneZ>>7 to recover the tile and &0x7F for sub-tile interpolation, so it must receive scene coords. Java passes the *128+64 scene values (Client.java:8321, 8132, 4478, 4 […]

### G54. Opcode 222 CAM_LOOKAT passes tile coords (not scene coords) to GetHeightMapY — wrong target Y / camera pitch

- **Unit:** `client/Client#8`  **Java:** `Client.java:8120-8146`  **Go:** `pkg/jagex2/client/client.go:10346-10371`

Java: sceneX = cutsceneDstLocalTileX*128+64; sceneZ = cutsceneDstLocalTileZ*128+64; sceneY = getHeightmapY(sceneZ, currentLevel, sceneX) - cutsceneDstHeight (SCENE coords). Go computes var26 = CutsceneDstLocalTileX*128+64 and var4 = CutsceneDstLocalTileZ*128+64 but calls `c.GetHeightMapY(c.CurrentLevel, c.CutsceneDstLocalTileX, c.CutsceneDstLocalTileZ)` with raw TILE coords instead of var26/var4. The resulting sceneY (var5) is wrong, so dy=var7=var5-CameraY is wrong, so cameraPitch = atan2(dy, distance) is wrong for the cutscene look-at. Should pass var26, var4.

```
Java: `int sceneY = this.getHeightmapY(sceneZ, this.currentLevel, sceneX) - this.cutsceneDstHeight;`  |  Go: `var5 := c.GetHeightMapY(c.CurrentLevel, c.CutsceneDstLocalTileX, c.CutsceneDstLocalTileZ) - c.CutsceneDstHeight`
```

> **Verifier (confirmed, severity bug):** CONFIRMED REAL after attempting refutation via the deob-parameter-scramble and cross-lineage-rename checks.

Argument-order trace (ruling out parameter scramble): Java `getHeightmapY(int sceneZ, int level, int sceneX)` indexes the heightmap as `levelHeightmap[realLevel][tileX=sceneX>>7][tileZ=sceneZ>>7]`. Go `GetHeightMapY(arg0, arg1, arg3)` body does `var5=arg1>>7`, `var6=arg3>>7`, level-check on `arg0`, and indexes `LevelHeightMap[var7][var5][var6]`. Since Go's first index var5 must equal Java's first index tileX(=sceneX>>7), the Go convention is unambiguously `GetHeightMapY(level, sceneX, sceneZ)` — i.e. Go reorders to put level first and X second, Z third. Both params 2 and 3 are SCENE c […]
>
> **Refined:** Opcode 222 CAM_LOOKAT (client.go:10356) calls c.GetHeightMapY(c.CurrentLevel, c.CutsceneDstLocalTileX, c.CutsceneDstLocalTileZ) with raw tile coords. Go's GetHeightMapY signature is (level, sceneX, sceneZ) with both coordinate args expected in SCENE scale (tile*128+64), as proven by the projectile/MAP_ANIM/AddGroundObject call sites. The function already computes the correct scene coords as var26 (=CutsceneDstLocalTileX*128+64, sceneX) and var4 (=CutsceneDstLocalTileZ*128+64, sceneZ) and uses th […]

### G55. Loc/decor occlusion cull caches the WRONG model bound (m.MinY instead of m.MaxY) — Java-244 ModelSource.minY == Go Model.MaxY, an uncompensated lineage inversion

- **Unit:** `dash3d/World3D#2`  **Java:** `World3D.java:1365,1563,1632; ModelSource.java:18-22; Model.java:1002-1003,1006-1007`  **Go:** `pkg/jagex2/dash3d/world3d/world3d.go:331,422,1289,1364,1367,1394,1402,1578,1580,1651,1654,1681,1689; typ/decor.go:17-20; typ/sprite.go:25-29`

Java-244 culls locs/wall-decorations on `ModelSource.minY`, which `ModelSource.draw()` sets to `model.minY` (ModelSource.java:18-22). In Model.java `minY` is `super.minY`, accumulated as `max(-y)` (lines 1002-1003) = the model's HEIGHT ABOVE ORIGIN. The Go field-inversion (proven for this lineage) makes Java `model.minY` == Go `Model.MaxY`, NOT Go `Model.MinY`. The verified-correct rev-225 Go port passed `model.MaxY`/`farthestModel.MaxY` to Visible()/LocVisible() (git show rev-225:.../world3d.go lines 1336,1533,1597). The rev-244 port introduced a cached `Sprite.MinY`/`Decor.MinY` field but seeds and refreshes it from `m.MinY` (= Java `model.maxY` = `max(+y)` = DEPTH BELOW ORIGIN, the opposite bound) at world3d.go:331,422,1289,1367,1394,1402,1580,1654,1681,1689, and the culls read that field at 1364 (decor front), 1578 (loc), 1651 (decor back). Visible()/LocVisible()/Occluded() are byte-identical to rev-225 (no compensating swap), and they subtract the param from the heightmap (`heightmap - arg5`) to test the model's TOP, which requires the height-above-origin quantity. Result: locs and wall decorations are occlusion-culled against the wrong vertical extent, causing them to wrongly appear/disappear behind occluders. Fix: seed/refresh the cached field from `m.MaxY` (not `m.MinY`) at all 10 set-sites.

```
Java World3D.java:1563 `if (!this.locVisible(originalLevel, ..., farthest.model.minY))`; ModelSource.java:18 `public int minY = 1000;` and:21 `this.minY = model.minY;`; Model.java:1002 `if (-y > super.minY) { super.minY = -y; }`. Go world3d.go:1578 `if !w.LocVisible(originalLevel, ..., farthest.MinY)` then :1580 `farthest.MinY = m.MinY`. rev-225 (correct) world3d.go:1533 `if !w.LocVisible(originalLevel, ..., farthestModel.MaxY)`. Go model.go:966 `m.MaxY = max(-var4, m.MaxY)` / :967 `m.MinY = max(var4, m.MinY)` confirms MaxY==Java minY.
```

> **Verifier (confirmed, severity bug):** CONFIRMED after attempting refutation. The lineage inversion is proven at the field level: Java-244 Model.super.minY accumulates max(-y) (height above origin) and Model.maxY accumulates max(+y) (depth below origin). Go model.go:966-967 sets m.MaxY = max(-var4) and m.MinY = max(var4), so Java model.minY == Go m.MaxY. This is independently cross-checked by the depth formulas: Go MinDepth uses MaxY*MaxY / MaxDepth uses MinY*MinY (model.go:972-973), mirroring Java minDepth(super.minY)/maxDepth(maxY) exactly.

The occlusion cull requires the height-above-origin bound. Java World3D.java:1365,1563,1632 pass decor.model.minY / farthest.model.minY (where Decor.model and Sprite.model are typed ModelSo […]
>
> **Refined:** Confirmed as written. Decisive evidence: (1) Java model.minY == Go Model.MaxY proven via both accumulation (Go m.MaxY=max(-y) ↔ Java super.minY=max(-y)) and depth formulas (Go MinDepth uses MaxY, MaxDepth uses MinY — mirroring Java). (2) rev-225 correctly passed Model.MaxY to Visible/LocVisible. (3) rev-244 seeds/refreshes the cached Sprite.MinY/Decor.MinY from m.MinY (the opposite, depth-below-origin bound) and the culls read it. (4) Visible/LocVisible/Occluded are byte-identical to rev-225 and […]

### G56. ALLOWLIST is missing two entries ("woop", "woops") — filter under-allows

- **Unit:** `wordenc/WordFilter#1`  **Java:** `WordFilter.java:29`  **Go:** `pkg/jagex2/wordenc/wordfilter/wordfilter.go:16`

Java ALLOWLIST has 7 entries: {"cook", "cook's", "cooks", "seeks", "sheet", "woop", "woops"}. The Go ALLOWLIST has only 5: {"cook", "cook's", "cooks", "seeks", "sheet"}. The allowlist loop in Filter restores these substrings after the bad/domain/tld/fragment passes have masked them; with "woop"/"woops" absent, those words stay censored where Java would un-censor them. This is the field-initializer owned by the chunk containing line 1, and it is a direct behavioral divergence.

```
Java: new String[] { "cook", "cook's", "cooks", "seeks", "sheet", "woop", "woops" };  Go: ALLOWLIST []string = []string{"cook", "cook's", "cooks", "seeks", "sheet"}
```

> **Verifier (confirmed, severity bug):** Confirmed by reading both sites directly. Java WordFilter.java:29 (via git show 01f16088): `ALLOWLIST = new String[] { "cook", "cook's", "cooks", "seeks", "sheet", "woop", "woops" }` — 7 entries. Go wordfilter.go:16: `ALLOWLIST []string = []string{"cook", "cook's", "cooks", "seeks", "sheet"}` — 5 entries; "woop" and "woops" are absent.

The consumer loop is faithful in both: in Java's filter() (lines 166-174) and Go's Filter() (lines 186-198), the code iterates the full length of ALLOWLIST and, for each entry found in the lowercased input, writes the original characters back into `output`, un-masking any '*' censoring applied by the preceding FilterTLD/FilterBad/FilterDomains/FilterFragments […]
>
> **Refined:** Go ALLOWLIST (pkg/jagex2/wordenc/wordfilter/wordfilter.go:16) has 5 entries {"cook","cook's","cooks","seeks","sheet"}; Java WordFilter.java:29 has 7, adding "woop" and "woops". The Filter() allowlist-restore loop iterates the full ALLOWLIST length and un-masks each matching substring after the bad/domain/tld/fragment passes; with the two entries absent, "woop"/"woops" stay censored in Go where Java restores them. No other Go site supplements the allowlist. Fix: add "woop", "woops" to the Go slic […]

### G57. getEmulatedSize 'b' branch drops the (b=='i' && c=='3') two-char alternative

- **Unit:** `wordenc/WordFilter#1`  **Java:** `WordFilter.java:852-861`  **Go:** `pkg/jagex2/wordenc/wordfilter/wordfilter.go:852-860`

Java a=='b' branch: when b is not '6'/'8', it returns 2 if (b=='1'&&c=='3') OR (b=='i'&&c=='3'). Go only checks (b=='1'&&c=='3') and otherwise returns 0. The 'i3' leetspeak emulation of 'b' is silently lost, so fragments matched via 'i3'->b will not be censored. (NOTE: declaration is at Java line 833, after this chunk's 1-650 range; primary ownership is WordFilter#2, reported here because it was traced through call sites.)

```
Java: if ((b != '1' || c != '3') && (b != 'i' || c != '3')) { return 0; } return 2;  Go: if b == '1' && c == '3' { return 2 } return 0
```

> **Verifier (confirmed, severity bug):** Confirmed real after ruling out the parameter-scramble false positive. Java getEmulatedSize is declared (char a, char c, char b) and called getEmulatedSize(fragment[fragOff], c, b); Go is declared (c, a, b rune) and called GetEmulatedSize(c, fragment[fragOff], b). Tracing both bodies: in both, the fragment letter maps to the body variable tested as `a`, the lookahead char maps to `c`, and the current input char maps to `b`. So the signature reorder is a cosmetic deob scramble, NOT a semantic swap — the `a == 'b'` branches in both languages compare the same operands.

Java a=='b' branch (852-861): inner cond `(b != '1' || c != '3') && (b != 'i' || c != '3')` returns 0 only when NOT(b=='1'&&c= […]
>
> **Refined:** In GetEmulatedSize (pkg/jagex2/wordenc/wordfilter/wordfilter.go:852-860), the a=='b' branch returns 2 only for (b=='1' && c=='3'), dropping Java's second alternative (b=='i' && c=='3') (WordFilter.java:852-861). The argument-order difference between the Java declaration (a,c,b) and Go (c,a,b) is a cosmetic deob scramble; call sites map operands identically, so the comparison is valid. Consequence: the leetspeak 'i3' -> 'b' emulation is lost, so a banned fragment containing 'b' written as 'i3' is […]

### G58. getEmulatedSize 'd' branch drops the (b=='i' && c==')') two-char alternative

- **Unit:** `wordenc/WordFilter#1`  **Java:** `WordFilter.java:869-874`  **Go:** `pkg/jagex2/wordenc/wordfilter/wordfilter.go:869-874`

Java a=='d' branch returns 2 if (b=='['&&c==')') OR (b=='i'&&c==')'). Go only checks (b=='['&&c==')'). The 'i)'->d emulation is lost. (Declaration at Java line 833 falls after the 1-650 range; primarily owned by WordFilter#2, surfaced via the FilterDomain/Filter2 trace.)

```
Java: if ((b != '[' || c != ')') && (b != 'i' || c != ')')) { return 0; } return 2;  Go: if b == '[' && c == ')' { return 2 } return 0
```

> **Verifier (confirmed, severity bug):** Confirmed real after ruling out the deob param-scramble trap. Java declares getEmulatedSize(char a, char c, char b); Go declares GetEmulatedSize(c, a, b rune) — different positional order. I traced both call sites: Java getEmulatedSize(fragment[fragOff], c, b) maps to (a=fragment[fragOff], c=c, b=b); Go GetEmulatedSize(c, fragment[fragOff], b) maps to (c=c, a=fragment[fragOff], b=b). The mapping is identical (a=fragment[fragOff], c=c, b=b), so the named variables a/b/c carry the same semantics in both bodies and a name-for-name branch comparison is valid.

The 'd' branch (Java WordFilter.java:869-874): `if ((b != '[' || c != ')') && (b != 'i' || c != ')')) { return 0; } return 2;`. By De Mor […]
>
> **Refined:** getEmulatedSize 'd' branch (Go wordfilter.go:869-874) drops the (b=='i' && c==')') two-char emulation alternative. Java returns 2 when (b=='[' && c==')') OR (b=='i' && c==')'); Go returns 2 only for the '[' case and 0 otherwise. Argument-position scramble between Java (a,c,b) and Go (c,a,b) is handled correctly at both call sites, so the named-variable comparison is valid and the dropped 'i)'->d rule is a genuine word-filter behavior bug.

### G59. getEmulatedSize 'g' branch drops 'q' as an emulation of 'g'

- **Unit:** `wordenc/WordFilter#1`  **Java:** `WordFilter.java:893-898`  **Go:** `pkg/jagex2/wordenc/wordfilter/wordfilter.go:893-898`

Java a=='g' branch returns 1 when b is '9', '6', OR 'q'. Go only allows '9' and '6'; for b=='q' it returns 0. The 'q'->g emulation is lost. (Declaration at Java line 833 is past the 1-650 range; primary owner is WordFilter#2.)

```
Java: if (b != '9' && b != '6' && b != 'q') { return 0; } return 1;  Go: if b != '9' && b != '6' { return 0 } return 1
```

> **Verifier (confirmed, severity bug):** Confirmed real after ruling out the deob parameter-scramble false-positive. The Go decl reorders params to GetEmulatedSize(c, a, b rune) vs Java getEmulatedSize(char a, char c, char b), but the call sites compensate correctly: Java line 639 getEmulatedSize(fragment[fragOff], c, b) maps a=fragment[fragOff], c=c, b=b; Go line 644 GetEmulatedSize(c, fragment[fragOff], b) maps c=c, a=fragment[fragOff], b=b. So the `b` operand tested against the literals is the SAME semantic value in both languages — no swap masking the diff.

Decisive evidence, a=='g' branch:
- Java (WordFilter.java:893-898): `if (b != '9' && b != '6' && b != 'q') { return 0; } return 1;` → returns 1 for b in {'9','6','q'}.
- Go […]
>
> **Refined:** In GetEmulatedSize, the a=='g' branch omits the b=='q' case present in Java (WordFilter.java:893-898). Java returns 1 (a 1-char emulation of 'g') when b is '9', '6', OR 'q'; Go (wordfilter.go:894) only checks '9' and '6', so b=='q' returns 0. Note the Go signature reorders params to (c, a, b); call sites at wordfilter.go:644/662 compensate so the `b` operand is identical to Java's — the missing 'q' literal is the sole defect. Fix: `if b != '9' && b != '6' && b != 'q' { return 0 }`.

### G60. filterFragments (Filter2 numeral/alpha masking) omits the alphaIndex adjustment

- **Unit:** `wordenc/WordFilter#1`  **Java:** `WordFilter.java:756-771`  **Go:** `pkg/jagex2/wordenc/wordfilter/wordfilter.go:760-775`

In the final 'if (bad)' block of filter(char[],char[],byte[][]), Java tracks alphaIndex (the last index where isAlpha was true) and, before the numeralCount<=alphaCount test, does `if (alphaIndex > -1) numeralCount -= end - alphaIndex + 1;`. The Go Filter2 has neither alphaIndex nor the `numeralCount -= end - alphaIndex + 1` subtraction. This changes whether a matched span gets masked when it contains a mix of digits and letters, so some words Java censors will pass through Go unmasked (and vice-versa). Note Java's filter() declaration is at line 607 (in range); this masking block runs to the closing brace past line 650, so it is owned by this chunk.

```
Java: int alphaIndex = -1; ... else if (isAlpha(chars[i])) { alphaCount++; alphaIndex = i; } ... if (alphaIndex > -1) { numeralCount -= end - alphaIndex + 1; } if (numeralCount <= alphaCount) {...}.  Go (lines 761-771): only numeralCount/alphaCount counters, then `if numeralCount <= alphaCount` — no alphaIndex, no subtraction.
```

> **Verifier (confirmed, severity bug):** Confirmed against Java-244 (commit 01f16088, WordFilter.java) and the Go port. Java's filter(char[],char[],byte[][]) (declared line 607) contains, in its final `if (bad)` masking block: `int alphaIndex = -1;` (758), `alphaIndex = i;` inside the `else if (isAlpha(...))` branch (765), and `if (alphaIndex > -1) { numeralCount -= end - alphaIndex + 1; }` (769-771) applied BEFORE the `if (numeralCount <= alphaCount)` masking gate (773). The Go port Filter2 (wordfilter.go:612, masking block 760-776) has the bare numeralCount/alphaCount counters and the `if numeralCount <= alphaCount` gate but NO alphaIndex variable and NO `numeralCount -= end - alphaIndex + 1` subtraction (verified by grep: alphaI […]
>
> **Refined:** In the final `if (bad)` masking block of Filter2 (Go port of Java filter(char[],char[],byte[][]), wordfilter.go:760-776 vs WordFilter.java:756-773), the Go code omits the alphaIndex tracking and the `numeralCount -= end - alphaIndex + 1` adjustment that Java applies before the `numeralCount <= alphaCount` masking gate. Java tracks alphaIndex as the last index where isAlpha was true and, when alphaIndex > -1, subtracts `end - alphaIndex + 1` from numeralCount (always reducing it by >= 2). Because […]

### G61. getEmulatedSize case 'b' drops the (b=='i' && c=='3') -> 2 alternative present in 244

- **Unit:** `wordenc/WordFilter#2`  **Java:** `WordFilter.java:852-862`  **Go:** `pkg/jagex2/wordenc/wordfilter/wordfilter.go:852-860`

Java-244 for a=='b' returns 2 when NOT ('6'/'8') and ((b=='1'&&c=='3') OR (b=='i'&&c=='3')). The Go only handles (b=='1'&&c=='3') and returns 0 for (b=='i'&&c=='3'). Param mapping verified through both bodies and the call sites (Java getEmulatedSize(fragment[fragOff],c,b); Go GetEmulatedSize(c,fragment[fragOff],b)): Java a/b/c map 1:1 to Go a/b/c. The Go faithfully ports rev-225 (which had only the '1'/'3' alternative) but was not re-synced to the 244 reference, so the 'i'/'3' leet-substitution slips past the word filter.

```
Java: if ((b != '1' || c != '3') && (b != 'i' || c != '3')) { return 0; } return 2;  |  Go: if b == '1' && c == '3' { return 2 } return 0
```

> **Verifier (confirmed, severity bug):** Confirmed real after ruling out all common false-positive causes.

Param mapping (the key false-positive vector) is NOT scrambled. Java signature `getEmulatedSize(char a, char c, char b)` called as `getEmulatedSize(fragment[fragOff], c, b)` binds a=fragment[fragOff], c=c, b=b. Go signature `GetEmulatedSize(c, a, b rune)` called as `GetEmulatedSize(c, fragment[fragOff], b)` binds c=c, a=fragment[fragOff], b=b. Both call sites (Java WordFilter.java:639/656; Go wordfilter.go:644-645/662-663) pass the same positional values, so a/b/c align 1:1 between the two bodies.

The a=='b' branch differs:
- Java-244 (WordFilter.java:852-862): `if (b != '6' && b != '8') { if ((b != '1' || c != '3') && (b != […]
>
> **Refined:** In GetEmulatedSize (wordfilter.go:852-860), the a=='b' case omits the 244 alternative that returns 2 for (b=='i' && c=='3'). Java-244 (WordFilter.java:852-862) returns 2 when, given b is not '6'/'8', either (b=='1'&&c=='3') OR (b=='i'&&c=='3'); the Go only handles the '1'/'3' case and returns 0 for 'i'/'3'. Param mapping verified through both call sites and both bodies (Java getEmulatedSize(a,c,b) called (fragment[fragOff],c,b); Go GetEmulatedSize(c,a,b) called (c,fragment[fragOff],b)) — a/b/c a […]

### G62. getEmulatedSize case 'd' drops the (b=='i' && c==')') -> 2 alternative present in 244

- **Unit:** `wordenc/WordFilter#2`  **Java:** `WordFilter.java:864-869`  **Go:** `pkg/jagex2/wordenc/wordfilter/wordfilter.go:869-874`

Java-244 for a=='d' returns 2 when (b=='['&&c==')') OR (b=='i'&&c==')'). The Go only checks (b=='['&&c==')') and returns 0 for (b=='i'&&c==')'). Same root cause as the 'b' case: Go ported rev-225's single alternative and was not updated to 244's two-alternative form. Causes the 'i)' leet form of 'd' to be missed by the filter.

```
Java: if ((b != '[' || c != ')') && (b != 'i' || c != ')')) { return 0; } return 2;  |  Go: if b == '[' && c == ')' { return 2 } return 0
```

> **Verifier (confirmed, severity bug):** Confirmed real. Java-244 WordFilter.java (git 01f16088) a=='d' block: `if ((b != '[' || c != ')') && (b != 'i' || c != ')')) { return 0; } return 2;` — by De Morgan this returns 2 when (b=='[' && c==')') OR (b=='i' && c==')'). Go (pkg/jagex2/wordenc/wordfilter/wordfilter.go:869-874) only handles the first alternative: `if b == '[' && c == ')' { return 2 } return 0`, so it returns 0 for (b=='i' && c==')'), dropping the second leet form.\n\nParameter-scramble false positive ruled out: Java signature is getEmulatedSize(char a, char c, char b) called as getEmulatedSize(fragment[fragOff], c, b); Go signature is GetEmulatedSize(c, a, b rune) called as GetEmulatedSize(c, fragment[fragOff], b). Trac […]
>
> **Refined:** Java-244 getEmulatedSize for a=='d' returns 2 when (b=='[' && c==')') OR (b=='i' && c==')') (source: `if ((b != '[' || c != ')') && (b != 'i' || c != ')')) return 0; return 2;`). The Go GetEmulatedSize a=='d' branch only checks (b=='[' && c==')') and returns 0 for (b=='i' && c==')'), dropping 244's second leet alternative. Root cause: Go ported 225-clean's single-alternative 'd' form and was not updated to the 244 two-alternative form (same class as the 'b' case). Parameter roles verified consis […]

---

## Latent (50)

### L01. min/max Huffman-length scan uses unsigned byte compare vs Java's signed byte compare

- **Unit:** `BZip2`  **Java:** `BZip2.java:328-340`  **Go:** `pkg/jagex2/io/bzip2/bzip2.go:344-355`

Java computes minLen/maxLen with `if (s.len[t][i] > maxLen)` / `< minLen` where len[t][i], minLen, maxLen are all signed `byte` (promoted to int with sign-extension) — a SIGNED comparison. The Go port declares minLen/maxLen as `byte` (uint8) and uses the builtin `max`/`min`, which on uint8 are UNSIGNED comparisons. For valid bzip2 streams the code lengths are always 0..23 (<= BZ_MAX_CODE_LEN), so signed and unsigned compares agree and output is identical. The divergence only manifests for malformed/adversarial input where a decoded length reaches >=128 (sign bit set): Java would treat it as negative (e.g. -128), driving minLen negative and causing createDecodeTables to index arrays with a negative start (ArrayIndexOutOfBounds), whereas Go treats it as 128..255 and silently builds a different/garbage table. Behavior diverges only on invalid data; latent, not reachable from the legitimate cache.

```
Java: `byte minLen = 32; byte maxLen = 0; ... if (s.len[t][i] > maxLen) maxLen = s.len[t][i]; if (s.len[t][i] < minLen) minLen = s.len[t][i];`  Go: `minLen := byte(32); maxLen := byte(0); ... maxLen = max(s.Len[t][i], maxLen); minLen = min(s.Len[t][i], minLen)` (unsigned).
```

> **Verifier (confirmed, severity latent):** Verified both sites in full context. Java (01f16088:src/main/java/jagex2/io/BZip2.java:326-340): `byte minLen=32; byte maxLen=0; ... if (s.len[t][i] > maxLen) maxLen=...; if (s.len[t][i] < minLen) minLen=...;` where BZip2State.len is `byte[][]` (line 120) — Java promotes both byte operands to int with sign-extension, a SIGNED comparison. Go (pkg/jagex2/io/bzip2/bzip2.go:344-355): `minLen := byte(32); maxLen := byte(0); ... maxLen = max(s.Len[t][i], maxLen); minLen = min(s.Len[t][i], minLen)` where bzip2state.Len is `[][]byte` (uint8) — the Go max/min builtins on byte operands perform UNSIGNED comparison. The signedness divergence is real and distinct.

Refutation attempts all failed: (1) not […]
>
> **Refined:** Java's min/max Huffman-length scan (BZip2.java:326-340) uses SIGNED byte comparisons (`byte minLen/maxLen`, operands sign-extended to int), while the Go port (bzip2.go:344-355) declares `minLen/maxLen` as `byte` (uint8) and uses the `max`/`min` builtins, yielding UNSIGNED comparisons. For valid bzip2 streams all code lengths are 0..23, so signed and unsigned compares agree and output is byte-identical; the live game cache only contains trusted data, so the divergence is unreachable in practice ( […]

### L02. total_in/total_out 32-bit wrap counters never wrap in Go (64-bit int), so hi32 carry path is dead

- **Unit:** `BZip2`  **Java:** `BZip2.java:585-588,154-158`  **Go:** `pkg/jagex2/io/bzip2/bzip2.go:596-599,162-166`

In getBits Java does `s.total_in_lo32++; if (s.total_in_lo32 == 0) s.total_in_hi32++;` relying on 32-bit `int` wrapping at 2^32 to trigger the hi32 carry. The Go fields TotalInLo32/TotalInHi32 are Go `int` (64-bit), so TotalInLo32 never returns to 0 at 2^32 and TotalInHi32 is never bumped; likewise in Finish the `s.TotalOutLo32 += ...; if (s.TotalOutLo32 < var) s.TotalOutHi32++` overflow-detect-by-wrap never fires for realistic output sizes. These counters are write-only — a repo-wide grep confirms nothing outside bzip2 reads TotalInLo32/Hi32/TotalOutLo32/Hi32 — and input streams are well under 2^32 bytes, so the divergence is unreachable and unobservable. Flagged per the int32-truncation bug class.

```
Java `int total_in_lo32;` wraps; Go `TotalInLo32 int` (64-bit) — `if s.TotalInLo32 == 0 { s.TotalInHi32++ }` cannot trigger at 2^32. No external readers found via grep of TotalIn*/TotalOut* outside the bzip2 package.
```

> **Verifier (confirmed, severity latent):** Verified both cited sites in full context against Java-244 (git show 01f16088).

FACTS CONFIRMED:
- Java BZip2State (line 48/51/63/66): total_in_lo32/hi32 and total_out_lo32/hi32 are Java `int` (32-bit).
- Go bzip2state.go:23-29: TotalInLo32/Hi32/TotalOutLo32/Hi32 are Go `int` (64-bit on the amd64 target per CLAUDE.local.md).
- getBits: Java `s.total_in_lo32++; if (s.total_in_lo32 == 0) s.total_in_hi32++;` (BZip2.java:585-587). Go bzip2.go:596-599 is a literal port `s.TotalInLo32++; if s.TotalInLo32 == 0 { s.TotalInHi32++ }`. Java relies on 32-bit wrap-to-zero at 2^32 to trigger the carry; Go's 64-bit int never returns to 0 at 2^32, so TotalInHi32 is never bumped.
- Finish: identical structu […]
>
> **Refined:** Accurate as written. Java's total_in_lo32/total_out_lo32 are 32-bit ints whose wrap-to-zero (getBits) / overflow-below-prior (Finish) triggers the hi32 carry; the Go fields are 64-bit `int`, so the carry into TotalInHi32/TotalOutHi32 is dead. Confirmed write-only in both languages (init, increment+carry only) with zero external readers repo-wide, and only reachable past a 2^32-byte (4 GiB) single-stream boundary that never occurs for game cache data. Unobservable; severity latent.

### L03. Available() never surfaces the read-side error; surfaces one step later at ReadFully

- **Unit:** `ClientStream`  **Java:** `ClientStream.java:87-90`  **Go:** `pkg/jagex2/io/clientstream/clientstream.go:240-247,279-284`

Java available() can throw IOException if the underlying InputStream is in an error state, which the dispatcher's try/catch (Client.java:7228 area) would route to reconnect. Go Available() always returns (count, nil) — it never propagates cs.rErr. After a socket error/EOF, Available() returns the bytes still buffered in rbuf, then 0, never an error. The reconnect outcome is preserved because ReadFully DOES surface cs.rErr once the ring drains (line 283), which the caller routes to TryReconnect (client.go:9523-9526, 9532, 9549, 9560, 9572). So the error still triggers reconnect, just one packet-read step later than Java. Edge-only timing divergence, not a wrong outcome; no crash or desync. The caller already checks `if err != nil` on Available so wiring a future error return would be safe.

```
Go Available(): `return cs.availableLocked(), nil` (always nil err). Caller client.go:9522-9526 checks err but Available can never set it; the error instead arrives via ReadFully -> cs.rErr -> TryReconnect.
```

> **Verifier (confirmed, severity latent):** Verified against the real Java reference and Go code; could not refute.

1. Java available() (ClientStream.java:87-90) is `return this.dummy ? 0 : this.in.available();` with signature `public int available() throws IOException`. InputStream.available() genuinely throws IOException on a hard socket-error state, so the Java method can throw.

2. Java readPacket() (Client.java:7223-7232) calls `this.stream.available()` as the FIRST statement inside the try block opened at line 7228. Its catch (IOException ignore) at line 8458 calls `this.tryReconnect()`. So in Java a read-side IOException surfaced by available() triggers reconnect immediately, before any packet is read.

3. Go Available() (clie […]

### L04. Relocated minY cache is per-node in Go vs shared-on-ModelSource in Java — stale-by-one-frame only if a ModelSource is shared across nodes or refreshed via a foreign draw path

- **Unit:** `Dash3dSmall`  **Java:** `ModelSource.java:15-22; World3D.java:1563-1564,1365,1632`  **Go:** `pkg/jagex2/dash3d/typ/sprite.go:29, pkg/jagex2/dash3d/typ/decor.go:20, pkg/jagex2/dash3d/world3d/world3d.go:1578-1582,1364-1367`

In Java the minY cache lives on the ModelSource, so ANY draw() of that ModelSource (from any reference/path) refreshes the value all readers see. In Go the cache is per-scene-node (Sprite.MinY/Decor.MinY) and is only refreshed by that node's own draw path. If the same ModelSource object were shared by multiple scene nodes, or refreshed through a draw path other than the node's own cull-draw, Go could read a one-frame-stale minY where Java would see the just-updated value. In the current scene graph static models are seeded at add-time and animated wrappers are per-instance, so this is not reachable today; flagged as latent for the shared-source edge.

```
Java ModelSource.draw (ModelSource.java:19) `this.minY = model.minY;` mutates the shared base object; Go refresh sites are node-local: world3d.go:1580 `farthest.MinY = m.MinY`, world3d.go:1367/1394/1402/1654/1681/1689 `decor.MinY = m.MinY`.
```

> **Verifier (confirmed, severity latent):** I traced both code paths in full and could not refute the finding's core architectural claim; its evidence and reachability assessment are accurate.

Decisive evidence:

1. Cache location genuinely differs. In Java, minY is an inherited field on ModelSource (ModelSource.java:17 `public int minY = 1000`). World3D reads `decor.model.minY` / `farthest.model.minY` (World3D.java:1365,1563,1632) — i.e. from the shared source object — and ModelSource.draw refreshes it on that same object (`this.minY = model.minY`, line 19). Java addLoc/addDecor do NOT seed minY (they only assign `var15.model = arg8`). In Go the cache is per-scene-node: Sprite.MinY (sprite.go:29) and Decor.MinY (decor.go:20); the cu […]

### L05. ToAsterisks/ToSentenceCase/FormatName/ToBase37/HashCode use rune iteration vs Java UTF-16 code units (non-BMP divergence only)

- **Unit:** `Datastruct`  **Java:** `JString.java:21 (toBase37 charAt), 67 (hashCode charAt), 86-103 (formatDisplayName), 110-123 (toSentenceCase), 117-126 (censor)`  **Go:** `pkg/jagex2/datastruct/jstring/jstring.go:13-41,65-78,85-105,107-127,135-141`

Java String.length()/charAt() operate on UTF-16 code units; the Go ports convert to []rune (code points) or `for range`. For any BMP character (everything the RuneScape username/chat/password alphabet allows — ASCII plus at most BMP), one rune == one UTF-16 unit, so behavior is identical. They diverge only for supplementary (non-BMP) characters, which would be a single rune in Go but two UTF-16 units in Java — affecting loop counts, the 12-char cap in toBase37, and star counts. Unreachable for validated inputs. Documented in each function's comment. Reporting as latent per type-width semantics check.

```
`runes := []rune(s); for i := 0; i < len(runes) && i < 12; i++` (jstring.go:20-23) vs Java `for (i=0; i<s.length() && i<12; i++) s.charAt(i)` — diverges only when a char is a surrogate pair.
```

> **Verifier (confirmed, severity latent):** Verified both sites in full. Java JString uses String.length()/charAt() (UTF-16 code units); the Go ports use []rune(s) (ToBase37, HashCode, FormatName, ToSentenceCase) or `for range s` (ToAsterisks), both of which iterate Unicode code points. These are identical for every BMP character (1 code point == 1 UTF-16 unit) and diverge ONLY for supplementary/non-BMP characters (1 rune in Go vs a 2-unit surrogate pair in Java). The finding's central claim is therefore correct.

Reachability: decisively unreachable for validated inputs. Interactive fields (Username/Password/SocialInput/ChatTyped/ReportAbuseInput) are gated at client.go:2438-2447 by membership in CHARSET = "ABC...xyz0-9!\"£$%^&*()... […]
>
> **Refined:** Java JString.toBase37/hashCode/formatDisplayName/toSentenceCase/censor iterate UTF-16 code units (length()/charAt()); the Go ports iterate Unicode code points ([]rune for ToBase37/HashCode/FormatName/ToSentenceCase; `for range s` for ToAsterisks). These are identical for all BMP characters and diverge only for supplementary (non-BMP) characters, which are one rune in Go but a two-unit surrogate pair in Java — affecting loop counts, the 12-char cap in ToBase37, and star count in ToAsterisks. Conf […]

### L06. ClientEntity.clearRoute() method not ported

- **Unit:** `EntityA`  **Java:** `ClientEntity.java:236-239`  **Go:** `pkg/jagex2/dash3d/entity/clententity.go (absent)`

Java-244 ClientEntity has `clearRoute() { this.routeLength = 0; this.preanimRouteLength = 0; }`, called from Client.java:4915/4925/9290. No `ClearRoute` method exists in the Go entity package (verified by grep). Because the Go struct also lacks `preanimRouteLength`, even an inlined equivalent at the call sites would drop the `preanimRouteLength=0` half. Tied to the unported preanim_move delta; the method itself belongs to this file.

```
Java: `public final void clearRoute() { this.routeLength = 0; this.preanimRouteLength = 0; }` — no Go counterpart in entity pkg
```

> **Verifier (confirmed, severity latent):** VERIFIED REAL (could not refute). Java-244 ClientEntity.clearRoute() exists and is `{ this.routeLength = 0; this.preanimRouteLength = 0; }` — found at src/main/java/jagex2/dash3d/ClientEntity.java:252-256 (the finding's path `dash3d/entity/ClientEntity.java:236-239` is wrong — the file lives at dash3d/ClientEntity.java and the method is at 252-256 — but the body and the three call sites Client.java:4915/4925/9290 all verify exactly). No `ClearRoute` method exists anywhere in the Go repo (grep across pkg/ and cmd/ returns nothing). The behavior is inlined at the three corresponding Go sites: client.go:3949, 3958 (UpdateClientPlayer/UpdateClientNpc, = Java updateEntity 4915/4925) and client.go […]
>
> **Refined:** Java-244 ClientEntity.clearRoute() (src/main/java/jagex2/dash3d/ClientEntity.java:252-256, NOT the dash3d/entity/...:236-239 path cited) is `{ routeLength = 0; preanimRouteLength = 0; }`, called from Client.java:4915/4925/9290. No ClearRoute method exists in the Go entity package; the behavior is inlined as bare `PathLength = 0` at the three Go equivalents (client.go:3949, 3958, 10730). The Go ClientEntity struct has no preanimRouteLength field, so the `preanimRouteLength = 0` half is dropped at […]

### L07. Missing `if (delta > deltime) delta = deltime` clamp in the run loop

- **Unit:** `GameShell`  **Java:** `GameShell.java:189-191`  **Go:** `pkg/jagex2/client/gameshell.go:494-496`

Java clamps delta to deltime between the `ratio > 256` block and the `otim[opos] = ntime` assignment. The Go RunShell jumps straight from `if var4 > 256 { ... var5 = ... }` to `c.OTim[var3] = var1` with no `var5 = min(var5, c.DelTime)`. In all normally-reachable branches var5 stays <= DelTime (the var4>256 branch yields DelTime - (elapsed/10) <= DelTime; the carry/init paths stay <= DelTime by induction), so the omission is latent. It becomes observable only under backward clock skew: when var1 < OTim[var3], `(var1-OTim)/10` is negative and var5 = DelTime - negative > DelTime, causing over-accumulation of OTim[] and a too-long sleep that Java would have clamped.

```
Java: `if (delta > this.deltime) { delta = this.deltime; }` immediately after the ratio>256 block; Go has no equivalent line before `c.OTim[var3] = var1`.
```

> **Verifier (confirmed, severity latent):** Confirmed real. Java GameShell.java:189-191 contains `if (delta > this.deltime) { delta = this.deltime; }` immediately after the `ratio > 256` block and before `this.otim[opos] = ntime`. Go RunShell (pkg/jagex2/client/gameshell.go) jumps from the `if var4 > 256 { var4 = 256; var5 = int(int64(c.DelTime) - (var1-c.OTim[var3])/10) }` block (lines 492-495) straight to `c.OTim[var3] = var1` (line 496) with no `var5 = min(var5, c.DelTime)` equivalent.

Variable mapping verified: delta->var5, deltime->c.DelTime (both int, Java line 29 / Go client.go:116), otim->c.OTim ([]int64 / long[]), opos->var3, ntime->var1, ratio->var4, mindel->c.MinDel. Computations match bit-for-bit (both 64-bit arithmetic t […]

### L08. op array sized [5] but codes 30-38 index 0..8 — faithful out-of-bounds on codes 35-38

- **Unit:** `LocType`  **Java:** `LocType.java:289-297`  **Go:** `pkg/jagex2/config/loctype/loctype.go:215-226`

Both sides allocate the op array with length 5 (Java new String[5], Go make([]string,5)) then index op[code-30] for codes 30..38, i.e. indices 0..8. Codes 35-38 would index 5..8 and throw ArrayIndexOutOfBoundsException (Java) / panic (Go). This is a bug-for-bug FAITHFUL port — the cache data never sends codes 35-38, so neither side actually crashes. Flagged as latent only because the shared bound is wrong for unexpected data; no port-side divergence.

```
Java: this.op = new String[5]; ... this.op[code - 30] = buf.gjstr(); Go: loc.Op = make([]string, 5); ... loc.Op[var4-30] = arg1.GJStr(). Identical [5]/index-by-(code-30) on both.
```

> **Verifier (confirmed, severity latent):** Confirmed correct on every claim by reading both full decode functions.

Java (LocType.java, decode): the branch `} else if (code >= 30 && code < 39) {` handles codes 30..38 (code 39 is handled separately above as `this.contrast = buf.g1b()`). It allocates `this.op = new String[5]` (length 5, valid indices 0..4) and writes `this.op[code - 30] = buf.gjstr()`. For codes 35,36,37,38 the index `code-30` is 5,6,7,8 — out of bounds for a length-5 array → ArrayIndexOutOfBoundsException.

Go (loctype.go:215-226, Decode): `case 30, 31, 32, 33, 34, 35, 36, 37, 38:` (case 39 handled separately above as `loc.Contrast = arg1.G1B()`, exactly matching Java's `< 39`). It allocates `loc.Op = make([]string, 5 […]
>
> **Refined:** Faithful bug-for-bug port of an upstream out-of-bounds bound. Java `LocType.decode` (branch `code >= 30 && code < 39`) and Go `LocType.Decode` (`case 30..38`) both handle config codes 30..38 (code 39 = contrast is handled separately), allocate a length-5 container (`new String[5]` / `make([]string, 5)`), and index it with `code-30`. Codes 35-38 yield indices 5..8, overflowing the length-5 container → ArrayIndexOutOfBoundsException (Java) / slice index panic (Go). The Go port faithfully replicate […]

### L09. ToCertificate first-char vowel test uses Name[0] (byte) vs Java charAt(0) (UTF-16 unit)

- **Unit:** `ObjType`  **Java:** `ObjType.java:393-398`  **Go:** `pkg/jagex2/config/objtype/objtype.go:298-307`

Java `char c = link.name.charAt(0)` reads a UTF-16 code unit; Go `var5 := var3.Name[0]` reads the first UTF-8 byte. For ASCII item names (all real data) both yield the same char and the vowel test matches. A non-ASCII first byte (e.g. 0xC2) can never equal the A/E/I/O/U set, so the 'a'/'an' choice is identical and safe. Documented in the Go comment (go:299-302). Flagged latent per the audit's String.charAt vs byte-index rule.

```
Java :394 `char c = link.name.charAt(0);`. Go go:303 `var5 := var3.Name[0]` with explanatory comment go:299-302.
```

> **Verifier (confirmed, severity latent):** Both sites confirmed verbatim. Java ObjType.genCert (shown via git 01f16088): `char c = link.name.charAt(0);` then `if (c == 'A' || c == 'E' || c == 'I' || c == 'O' || c == 'U') article = "an";`. Go ObjType.ToCertificate (objtype.go:298-306): `var5 := var3.Name[0]` with the same A/E/I/O/U test, where Name is a Go `string` (line 29: `Name string`), so `Name[0]` is a byte index (first UTF-8 byte), NOT a UTF-16 code unit like charAt(0). The finding's translation-pattern claim is accurate (String.charAt vs byte index, the documented audit rule).

However the divergence is provably unreachable: (a) for any ASCII codepoint <0x80 the single UTF-8 byte exactly equals the codepoint equals the UTF-16  […]

### L10. Cycle() gunzip lacks Java's 65000-byte decompressed-size cap

- **Unit:** `OnDemand`  **Java:** `OnDemand.java:96,344-368`  **Go:** `pkg/jagex2/io/ondemand/ondemand.go:420-449`

Java cycle() decompresses into a fixed 65000-byte scratch buffer (vb.N = new byte[65000]) and throws RuntimeException("buffer overflow!") if the decompressed output would exceed 65000 bytes; req.data is then sized exactly to the decoded length (<=65000). The Go Cycle() uses io.ReadAll(gz) with no upper bound, so a model/anim/map/midi file whose decompressed size exceeds 65000 bytes is accepted by Go but rejected (throws) by Java. This is a divergence only for over-cap files (none in the shipped cache today), hence latent, but it is a real behavioral difference not covered by the transport deviation registry.

```
Java: `public byte[] data = new byte[65000];` then `if (this.data.length == pos) { throw new RuntimeException("buffer overflow!"); }`. Go: `decoded, err := io.ReadAll(gz) ... r.Data = decoded` (no size limit).
```

> **Verifier (confirmed, severity latent):** Verified both sites directly. Java path is jagex2/io/OnDemand.java (finding's javaRef said jagex2/client but the code matches the cited lines). Java cycle() (01f16088, lines 343-368) decompresses GZIPInputStream into the fixed scratch buffer `this.data = new byte[65000]` (field decl line 96), and inside the read loop checks `if (this.data.length == pos) { throw new RuntimeException("buffer overflow!"); }`, then sizes `req.data = new byte[pos]` to the exact decoded length (<=65000). Go Cycle() (ondemand.go:420-449) does `decoded, err := io.ReadAll(gz)` then `r.Data = decoded` with no upper bound. So for any file whose decompressed size exceeds 65000 bytes, Java throws (and since the caller up […]
>
> **Refined:** Java OnDemand.cycle() (jagex2/io/OnDemand.java @ 01f16088, field decl line 96, loop lines 343-368) decompresses into a fixed 65000-byte scratch buffer (`public byte[] data = new byte[65000]`) and throws `RuntimeException("buffer overflow!")` if the decompressed output reaches 65000 bytes with more to read; `req.data` is then sized exactly to the decoded length (<=65000). The caller updateOnDemand() (Client.java:2424; Go client.go:8761) does not catch this, so the throw propagates up the game loo […]

### L11. ClientVersion field initialized to 225, Java-244 declares clientversion = 244

- **Unit:** `SignLink+Midi`  **Java:** `SignLink.java:53`  **Go:** `pkg/jagex2/client/sign/signlink/signlink.go:71`

Java declares `public static final int clientversion = 244;`. The Go counterpart is `ClientVersion int = 225`. Value mismatch carried over from the rev-225 lineage. Grep shows ClientVersion has no live reader in the Go tree today (the login handshake hardcodes its own version constant elsewhere), so this is currently a dead/stale value rather than a protocol desync — but it is a concrete field-initializer divergence and would feed a desync if any code starts reading it. Reported as bug (wrong constant) trending latent (no current reader).

```
Java: public static final int clientversion = 244;  |  Go: ClientVersion int = 225
```

> **Verifier (confirmed, severity latent):** Both cited sites are confirmed verbatim. Java-244 SignLink.java:53 declares `public static final int clientversion = 244;`; Go signlink/signlink.go:71 declares `ClientVersion int = 225`. This is a genuine field-initializer value divergence (225 carried over from the rev-225 lineage instead of 244).

I attempted to refute it on several axes and could not:
- No rename/lineage inversion: same field name and role in both trees, just a wrong literal.
- Not an intentional documented deviation: no `// Java: ... Intentionally not ported` or transport/platform note marks it; it is simply stale.
- Not an int-width issue: plain constant.

However, the finding's own characterization that this is current […]
>
> **Refined:** Field-initializer divergence: Java-244 SignLink.java:53 declares `clientversion = 244`; Go signlink.go:71 declares `ClientVersion int = 225` (stale from the rev-225 lineage). Confirmed dead in BOTH lineages — `ClientVersion`/`clientversion` has no reader in either tree (only its declaration), and the login handshake sends its protocol version as an independent hardcoded literal (Go client.go:6796 `c.Login.P1(244)` == Java Client.java:2641 `this.login.p1(244)`). Therefore no current protocol/logi […]

### L12. Per-step intermediate multiplies in Tone.Generate/Generate2 overflow Java int32 but not Go 64-bit int (divergence for large synth data)

- **Unit:** `Sound`  **Java:** `Tone.java:152,159,166,167,229,231,233`  **Go:** `pkg/jagex2/sound/tone/tone.go:128-129,135-136,143-144,205,207,209`

Several `int * int` products that feed a `>>` shift are 32-bit multiplies in Java that wrap mod 2^32, but are 64-bit in Go and do not wrap. The frequency/amplitude modulation accumulators (`frequencyStart * rate >> 16`, `amplitudeStart * rate >> 16`), the harmonic phase step (`fMulti[h] * frequency >> 16`), and the waveFunc/Generate2 form-2/3 products (`Sin[phase&0x7FFF] * amplitude`, `(phase&0x7FFF) * amplitude`) all diverge once a single product exceeds 2^31. With typical game tone data (harmonic volumes <=100) the products stay within int32, but harmonic volume is decoded via gsmarts with NO range validation, and frequency-mod start/end come from g4 (full 32-bit), so adversarial or unusual cache data reaches overflow in ONE step (not the multi-sample accumulation the existing comment reasons about). Java would produce a wrapped (sign-flipped) sample; Go produces the un-wrapped value — a different waveform. To match Java exactly these multiplies need int32(...) wraps. NOTE: the existing audit comment at tone.go:11-19 justifies only the accumulator growth and Buffer-clamp bounds; it does not address these single-step intermediate products, so its safety claim is incomplete.

```
Java: `frequencyPhase += (frequencyStart * rate >> 16) + frequencyDuration;` (Tone.java:152) and `return sine[phase & 0x7FFF] * amplitude >> 14;` (Tone.java:229). Go: `frequencyPhase += ((rate * frequencyStart) >> 16) + frequencyDuration` (tone.go:129), `return (Sin[phase&0x7FFF] * amplitude) >> 14` (tone.go:205). Demonstrated: with HarmonicVolume=1000, ampBase peak 65535, the form-2 product `16384*327675 = 5368627200` overflows int32 (Java wraps to ~1.07e9 region with sign flip; Go keeps 5.37e9 before >>14). fMulti(100000)*frequency(65535): Java32 = -2036434592, Go64 = 6553500000; after >>16 Java=-31074 vs Go=99998.
```

> **Verifier (confirmed, severity latent):** CONFIRMED after tracing both function bodies in full and verifying the arithmetic. Java Tone.generate/waveFunc use 32-bit int[] and int locals throughout; the multiplies that feed each `>>` shift (`frequencyStart*rate`, `amplitudeStart*rate`, `fMulti[h]*frequency`, `fAmp[h]*amplitude`, and inside waveFunc `sine[..]*amplitude`, `(phase&0x7FFF)*amplitude`, `noise[..]*amplitude`) all wrap mod 2^32. The Go port (tone.go) uses native 64-bit `int` for every one of these with no int32() wrap at any site (verified Go G4 at packet.go:238-241 and Evaluate at envelope.go:49-64 do not pre-wrap to int32). I traced input ranges: Envelope.genNext/Evaluate returns ~16-bit values (shapePeak from g2, returned […]
>
> **Refined:** Real latent int32-vs-int64 divergence. In Java Tone.generate (Tone.java:152,159,166,167) and waveFunc (Tone.java:229,231,233) the per-step intermediate products feeding each `>>` shift are 32-bit `int` multiplies that wrap mod 2^32; the Go port (tone.go:129,136,143-144,205,207,209) computes them in native 64-bit `int` with no int32() wrap. Each product (frequencyStart*rate, amplitudeStart*rate, fMulti[h]*frequency, fAmp[h]*amplitude, sine[phase&0x7FFF]*amplitude, (phase&0x7FFF)*amplitude, noise[ […]

### L13. WaveBytes/WaveBuffer static field initializers relocated into Unpack(); nil in low-memory mode

- **Unit:** `Sound`  **Java:** `Wave.java:16-19,30-44`  **Go:** `pkg/jagex2/sound/wave/wave.go:8-13,27-29`

Java declares `waveBytes = new byte[44100*10]` and `waveBuffer = new Packet(waveBytes)` as static field initializers that run at class load unconditionally. The Go port allocates them inside `Unpack()` (wave.go:28-29). The client only calls `wave.Unpack` when `!LowMemory` (client.go:6069-6074), so in low-memory mode `WaveBytes`/`WaveBuffer` remain nil in Go whereas Java always has them allocated. This is currently safe only because `Generate(loopCount, id)` short-circuits on `Tracks[id] == nil` (wave.go:45) and Tracks is never populated in low-mem, so `GetWave`/`Generate(method)` (the only WaveBytes/WaveBuffer consumers) are unreachable. If any future code path reached GetWave without a loaded track in low-mem, Go would nil-panic where Java would not. Behaviorally equivalent today but a structural deviation from Java's eager static init.

```
Java static: `public static byte[] waveBytes = new byte[44100 * 10]; public static Packet waveBuffer = new Packet(waveBytes);` (Wave.java:16-19). Go: allocation moved into Unpack: `WaveBytes = make([]byte, 441_000); WaveBuffer = io.NewPacket(WaveBytes)` (wave.go:28-29), and client gates `wave.Unpack` behind `if !LowMemory` (client.go:6069).
```

> **Verifier (confirmed, severity latent):** Verified against both sources. Java Wave.java:16-19 declares `public static byte[] waveBytes = new byte[44100*10]` and `public static Packet waveBuffer = new Packet(waveBytes)` as static field initializers that run unconditionally at class load. The Go port (wave.go:11-12) declares `WaveBytes []byte` / `WaveBuffer *io.Packet` as nil package vars and allocates them only inside `Unpack()` at wave.go:28-29. Both clients gate `Wave.unpack`/`wave.Unpack` behind low-memory: Java Client.java:1901 `if (!lowMem)` and Go client.go:6069 `if !LowMemory`. So in low-memory mode Go leaves WaveBytes/WaveBuffer nil while Java always has them allocated — a genuine structural deviation from Java's eager static […]
>
> **Refined:** Accurate as written. Minor nuance: even in Java the eager static allocation is not load-bearing for the client's only reachable consumer path, because `getWave` is reached via `tracks[id].getWave` and the same `tracks[id] == null` guard (Wave.java:31 / wave.go:45) protects both languages when no track is loaded. The Go nil-buffer exposure therefore requires a future call that constructs a Wave and invokes GetWave/Generate(method) directly while bypassing Tracks in low-mem mode — a narrower trigg […]

### L14. HandleScrollInput has an extra parameter and a spurious `c.PacketSize += arg1` side effect with no Java source

- **Unit:** `client/Client#11`  **Java:** `Client.java:10883-10923`  **Go:** `pkg/jagex2/client/client.go:6692-6699`

Java handleScrollInput takes 8 params (com, top, redraw, mouseY, height, scrollHeight, mouseX, left) and never touches any packet/network state. The Go HandleScrollInput declares a 9th int param arg1 (between mouseX-equiv arg0 and mouseY-equiv arg2) and executes `c.PacketSize += arg1` at the top — a write to network-buffer accounting state that does not exist in Java. Both call sites (client.go:1732 and 4329) pass 0 for arg1, so today it is a harmless no-op (PacketSize += 0). It is dead/incorrect code that would corrupt PacketSize if any future caller passed nonzero. The rest of the body (the three mouse-region branches, gripSize/gripY/maxY math, scrollInputPadding handling) matches Java exactly.

```
Go: `func (c *Client) HandleScrollInput(arg0, arg1, arg2, arg3, arg4 int, arg5 bool, arg6 int, arg7 int, arg8 *component.Component) { ... c.ScrollGrabbed = false; c.PacketSize += arg1; ...`  Java has no equivalent statement and only 8 parameters.
```

> **Verifier (confirmed, severity latent):** Verified against Java-244 (git 01f16088 Client.java:10883-10923), the TS client, and the obfuscated descriptor. All four factual claims hold:

1. EXTRA PARAM: Java-244 handleScrollInput declares 8 params (Component com, int top, boolean redraw, int mouseY, int height, int scrollHeight, int mouseX, int left). The Go HandleScrollInput (client.go:6692) declares 9: arg0..arg8 with an extra int `arg1` wedged between mouseX-equiv (arg0) and mouseY-equiv (arg2). The TS client (Client.ts:9753) also has only 8 params, no extra int. The obfuscated descriptor `client.a(Ld;IZIIIIII)V` contains 7 ints, whereas the deob Java source uses only 6 ints (top, mouseY, height, scrollHeight, mouseX, left) — i.e.  […]
>
> **Refined:** Go HandleScrollInput (pkg/jagex2/client/client.go:6692) carries a deob-bytecode artifact absent from Java-244 (Client.java:10883): a 9th parameter `arg1` (the obfuscated descriptor's extra 7th int, dropped by both the Java-244 deob and the TS port) plus a dead `c.PacketSize += arg1` write at line 6699. PacketSize is real protocol-length accounting state. Both and only call sites (client.go:1732 and 4329) pass literal 0, so it is a verified no-op today; it would corrupt PacketSize and desync the  […]

### L15. ExecuteClientscript1 opcode 13 left shift count not masked

- **Unit:** `client/Client#11`  **Java:** `Client.java:11027-11031`  **Go:** `pkg/jagex2/client/client.go:8940-8949`

Java opcode 13 computes `register += (varp & 0x1 << lsb) == 0 ? 0 : 1` where lsb comes from script data; Java implicitly masks the int shift count to 5 bits (lsb & 0x1F) and never panics. The Go port computes `var12 & (0x1 << var9)` with no mask. For non-negative var9 >= 32 Go yields 0 (matching Java only when lsb&0x1F also lands on a zero bit — not generally equal), and for a NEGATIVE var9 Go panics. The surrounding deferred recover() would convert that panic to the -1 sentinel, whereas Java would return a masked, in-range value. Grouping/precedence itself is correct (Go parenthesizes the shift to match Java's shift-before-AND). lsb is realistically 0..31 from cache data, so this only bites on malformed/edge script data.

```
Java: `register += (varp & 0x1 << lsb) == 0 ? 0 : 1;`  Go: `if var12&(0x1<<var9) == 0 { var5 += 0 } else { var5 += 1 }` (no `& 0x1F` on var9)
```

> **Verifier (confirmed, severity latent):** Confirmed real, severity latent. Decisive evidence:

1. Java (Client.java:11027-11031): `register += (varp & 0x1 << lsb) == 0 ? 0 : 1;` where `lsb = script[pc++]` is an int. Per JLS 15.19, an `int` left-shift masks the shift count to its low 5 bits (`lsb & 0x1F`), so Java NEVER throws and always tests bit `(lsb & 0x1F)`.

2. Go (client.go:8940-8949): `var9 = var4[var6]; ... if var12&(0x1<<var9) == 0 {...}`. Both `var9` and `var12` are plain `int` (declared `:= 0`). Go does NOT mask non-constant shift counts, so for `var9 >= 32` Go does not wrap — it shifts the bit to position `var9`. Since varps mirror Java 32-bit ints (`Varps []int`, but values are int32-range so bits 32+ are 0), `0x1 << va […]
>
> **Refined:** Java opcode 13 (Client.java:11027-11031) computes `register += (varp & 0x1 << lsb) == 0 ? 0 : 1`, where `lsb = script[pc++]`. Java's int left-shift implicitly masks the shift count to 5 bits (`lsb & 0x1F`, JLS 15.19), so it tests bit `lsb & 0x1F` and never throws. The Go port (client.go:8945) computes `var12 & (0x1 << var9)` with no `& 0x1F` mask. var9 is loaded from the component script array via the UNSIGNED Packet.G2() (range 0..65535), so it can be >= 32 but is never negative; the negative-s […]

### L16. HandleInterfaceInput: buttonType branches and type==2 inventory branch are independent ifs, not a mutually-exclusive else-if chain

- **Unit:** `client/Client#12`  **Java:** `Client.java:11069-11111`  **Go:** `pkg/jagex2/client/client.go:1734-1902`

Java structures the post-`type==0` logic as one `else if` chain: buttonType==1 ... else if buttonType==6 ... else if type==2 (inventory). They are mutually exclusive. The Go renders each as a separate `if` inside the single `else` block, so for a component with type==2 AND a buttonType in 1..6 the Go would execute BOTH a buttonType menu-entry block and the full inventory block, whereas Java executes only the buttonType block and skips the inventory rendering. The buttonType==1..6 blocks are themselves mutually exclusive (single int), so the only reachable divergence requires cache data where a type-2 component also carries buttonType 1..6 — not produced by standard data, hence latent.

```
Java: `} else if (child.buttonType == 1 ...) { ... } else if (child.buttonType == 2 ...) ... } else if (child.type == 2) { ... }`. Go: `} else { if var12.ButtonType == 1 ... {} if var12.ButtonType == 2 ... {} ... if var12.Type == 2 { ... } }` (each a separate `if`, client.go:1735,1747,1757,1763,1769,1775,1781).
```

> **Verifier (confirmed, severity latent):** Verified both sites in full. Java Client.java:11062-11111 is a single mutually-exclusive `if (child.type == 0) ... else if (child.buttonType == 1 ...) ... else if (child.buttonType == 6 ...) ... else if (child.type == 2) { inventory }` chain: the first matching arm wins and the rest are skipped. The Go (client.go:1729-1903) renders the non-type==0 case as `} else { if ButtonType==1 {...} if ButtonType==2 {...} ... if ButtonType==6 {...} if Type==2 { inventory } }` — seven independent `if`s. I confirmed there is no `continue`/`return`/`else if` inside the Go buttonType blocks (only the outer `} else {`), so the `if var12.Type == 2` inventory block at line 1781 is reached unconditionally regar […]
>
> **Refined:** Confirmed as written. Java Client.java:11062-11111 structures the post-`type==0` logic as one mutually-exclusive `else if` chain (buttonType==1 ... else if buttonType==6 ... else if type==2 inventory); Go client.go:1734-1902 renders each as an independent `if` inside a single `else` block. For a component with type==2 AND buttonType in 1..6 whose buttonType guard passes (mouse over the component), Java executes only the buttonType menu-entry block and skips inventory rendering, while Go executes […]

### L17. UpdateVarp clientCode==3 drops the Java `&& !lowMem` guard on midi (re)activation

- **Unit:** `client/Client#12`  **Java:** `Client.java:11390`  **Go:** `pkg/jagex2/client/client.go:3892`

Java gates the midi state-change block with `if (this.midiActive != lastMidiActive && !lowMem)`. The Go has only `if c.MidiActive != var5`. The `LowMemory` global exists in the Go port (client.go:86). In low-memory mode Java skips the SetMidi/StopMidi + nextMusicDelay reset entirely; the Go would still attempt it. The inner request body itself (Java `midiSong=nextMidiSong; midiFading=false; onDemand.request(2, midiSong)` vs Go `SetMidi(MidiCRC, CurrentMidi, MidiSize)`) is the OnDemand/WS5 audio transport seam (registry-covered), but the `!lowMem` guard is plain logic and should be present. Only affects low-mem clients, hence latent.

```
Java: `if (this.midiActive != lastMidiActive && !lowMem) { ... this.nextMusicDelay = 0; }`. Go: `if c.MidiActive != var5 { if c.MidiActive { c.SetMidi(...) } else { c.StopMidi() }; c.NextMusicDelay = 0 }` — no `&& !LowMemory`.
```

> **Verifier (confirmed, severity latent):** Confirmed against Java-244 (git 01f16088, Client.java:11390) and Go (client.go:3892). In the clientCode==3 (Go var3==3) midi block, Java gates the state-change sub-block with `if (this.midiActive != lastMidiActive && !lowMem)`; the Go has only `if c.MidiActive != var5` with no `&& !LowMemory`.

Refutation attempts, all failed:
1. Global existence: `LowMemory` is a package-level bool in client.go:86, set by SetLowMem (mirroring Java setLowMemory) and already used with the idiomatic `!LowMemory` pattern in many places, including midi contexts (client.go:7446 `c.NextMusicDelay == 0 && c.MidiActive && !LowMemory`, 9972, 9986). Java's `lowMem` is the matching plain `public static boolean` (Client […]
>
> **Refined:** In UpdateVarp's clientCode==3 (Go var3==3) midi block, the Go drops Java's `&& !lowMem` conjunct on the state-change guard. Java (Client.java:11390): `if (this.midiActive != lastMidiActive && !lowMem) { ...; this.nextMusicDelay = 0; }`. Go (client.go:3892): `if c.MidiActive != var5 { if c.MidiActive { c.SetMidi(...) } else { c.StopMidi() }; c.NextMusicDelay = 0 }` with no `&& !LowMemory`. The `LowMemory` global exists (client.go:86) and is used as `!LowMemory` elsewhere; SetMidi/StopMidi do not  […]

### L18. RunFlames drops `this.flameCycle++` per-iteration increment

- **Unit:** `client/Client#13`  **Java:** `Client.java:12308-12346`  **Go:** `pkg/jagex2/client/client.go:6666-6690`

Java runFlames increments `this.flameCycle` at the top of every while-iteration (Client.java:12317). The Go RunFlames has no corresponding FlameCycle field/increment (Go has only FlameGradientCycle0/1, which are the distinct gradient counters). The only reader of Java flameCycle is the debug dump at Client.java:4833 (`System.out.println("flame-cycle:"+this.flameCycle)`). That debug dump is not ported to Go, so the dropped increment currently has no observable effect; it becomes a real bug only if that debug dump is later ported (it would always print 0). The remainder of the loop (double updateFlames, drawFlames, 10-cycle interval recalibration `interval = 40 - ((int)(now-last)/10 - interval)` clamped to >=5) matches.

```
Java loop body starts with `this.flameCycle++;` then `this.updateFlames(); this.updateFlames(); this.drawFlames();`  ||  Go loop body: `c.UpdateFlames(); c.UpdateFlames(); c.DrawFlames(); var4++` — no flameCycle field anywhere in the package
```

> **Verifier (confirmed, severity latent):** Confirmed real after ruling out the name-reuse trap. Java has TWO distinct fields: `flameCycle0` (@ObfuscatedName client.Ed, line 845) and `flameCycle` (@ObfuscatedName client.Ce, line 863). These are different fields. The Go `FlameCycle0` field correctly ports Java `flameCycle0` — the gradient counter that does `+= 128` in updateFlames (Java:12374 / Go client.go:2707), gated by buffer length. The finding is about the OTHER field, bare `flameCycle` (client.Ce).\n\nTracing `flameCycle`: it is written in exactly ONE place — `this.flameCycle++` at the top of every runFlames while-iteration (Java:12317) — and read in exactly ONE place — the `lag()` debug dump `System.out.println(\"flame-cycle:\" […]
>
> **Refined:** Java runFlames increments `this.flameCycle` (@ObfuscatedName client.Ce, field decl Client.java:863) at the top of every while-iteration (Client.java:12317). This is a DISTINCT field from `flameCycle0` (client.Ed, Client.java:845), which is the gradient counter correctly ported to Go `FlameCycle0` (+=128 in UpdateFlames, client.go:2707). Go RunFlames (client.go:6666-6690) has no field or increment corresponding to the bare `flameCycle`. The only reader of Java `flameCycle` is the `lag()` debug du […]

### L19. IsFriend early-return uses empty-string instead of Java null, conflating non-null empty username

- **Unit:** `client/Client#13`  **Java:** `Client.java:12132-12146`  **Go:** `pkg/jagex2/client/client.go:10592-10605`

Java isFriend returns false only when `username == null`. The Go port returns false when `arg1 == ""`. A Java non-null empty string would NOT take the early return and would fall through to `return "".equalsIgnoreCase(localPlayer.name)`; the Go port short-circuits that path. This is the standard project null-string→"" mapping and is only reachable if a non-null empty username is ever passed, so latent. The friend-name EqualsIgnoreCase loop and the trailing localPlayer.name comparison are otherwise faithful.

```
Java: `if (username == null) { return false; }`  ||  Go: `if arg1 == "" { return false }`
```

> **Verifier (confirmed, severity latent):** Verified both sites in full context. Java Client.java:12132-12146 isFriend(String username) short-circuits ONLY on `username == null`; a non-null empty string `""` falls through the friendName EqualsIgnoreCase loop and reaches `return "".equalsIgnoreCase(localPlayer.name)`. Go client.go:10592-10605 IsFriend short-circuits on `arg1 == ""`, skipping both the loop and the trailing LocalPlayer.Name comparison. This is the project's canonical Java null -> Go "" mapping (Go strings cannot be nil), so the two agree for the genuine-null case. They diverge in exactly one observable scenario: a non-null EMPTY username is passed AND localPlayer.name == "" — Java returns true (`"".equalsIgnoreCase("")`) […]
>
> **Refined:** Java Client.java:12132-12146 isFriend returns false only when `username == null`; a non-null empty string falls through to the friendName loop and then `return "".equalsIgnoreCase(localPlayer.name)`. Go client.go:10593 returns false on `arg1 == ""`, short-circuiting both the loop and the LocalPlayer.Name comparison. This is the standard project null->"" mapping; the two agree for genuine null and diverge only when a non-null empty username is passed while LocalPlayer.Name is also "" (Java true,  […]

### L20. SaveWave drops Java wavesave's len>2000000 size cap (always plays)

- **Unit:** `client/Client#2`  **Java:** `Client.java:1467-1474 (saveWave) -> SignLink.java:299-313 (wavesave: returns false if len>2000000 or savereq!=null)`  **Go:** `pkg/jagex2/client/client.go:5453-5459`

Java saveWave returns the result of SignLink.wavesave, which returns false (skip + retry) when len>2000000. The Go SaveWave plays unconditionally and always returns true, so an oversized (>2MB) generated wave that Java would refuse to play is attempted in Go. The savereq!=null branch returning false is the signed-applet wrapper-consumer backpressure path which is registry-covered (wrapper not ported); but the len>2000000 guard is an independent size cap, not a wrapper concern, and is silently dropped. Practically unreachable (game SFX are small), hence latent.

```
Go: `if arg0 == nil { return true }; audio.PlayWave(arg0[:arg1]); return true`. Java wavesave: `if (len > 2000000) return false;`
```

> **Verifier (confirmed, severity latent):** Mechanically the finding is accurate. Java Client.saveWave (Client.java:1467-1474) returns SignLink.wavesave (SignLink.java:299-313), which returns false when len > 2000000. At the call site updateAudio (Client.java:3580-3618 / Go client.go:7400-7425), a false return sets replay/var4=true: the wave is deferred (delay=-5) and retried, lastWaveId is NOT updated, and it is never actually played. The Go SaveWave (client.go:5453-5459) calls audio.PlayWave directly and always returns true, so the len>2000000 size guard is genuinely absent from the wave path. The finding correctly separates the two wavesave branches: the savereq!=null branch is the signed-applet polling-consumer backpressure path ( […]
>
> **Refined:** Java Client.saveWave delegates to SignLink.wavesave, which returns false (causing a deferred retry without updating lastWaveId, i.e. the wave is not played) when len > 2000000. Go's SaveWave (client.go:5453-5459) calls audio.PlayWave directly and unconditionally returns true, dropping the len>2000000 size cap (the savereq!=null backpressure branch is the not-ported wrapper-consumer path and is correctly excluded). However, the value passed (var5.Pos from wave.Generate) is bounded by the fixed 44 […]

### L21. GetJagFile header read uses bare Read (not readFully) — partial-read divergence

- **Unit:** `client/Client#2`  **Java:** `Client.java getJagFile: `stream.readFully(header, 0, 6)``  **Go:** `pkg/jagex2/client/client.go:2530-2541`

Java reads the 6-byte archive header with readFully (blocks until all 6 bytes). The Go uses a single `reader.Read(header)` and errors/retries if n<6. Against the in-memory bytes.Reader returned by OpenURL a single Read yields all 6 bytes, so this is currently equivalent; but if OpenURL ever returned a true stream a short read would wrongly trigger the retry/backoff path. The sibling CRC fetch in load() correctly uses io2.ReadFull (client.go:5716), so the inconsistency is real. Callee of load() (GetJagFile, outside owned range) — noted for completeness.

```
Go: `n, err := reader.Read(header); if n < 6 { ...loadingError()... }`. Java: `stream.readFully(header, 0, 6)`.
```

> **Verifier (confirmed, severity latent):** Verified against Java getJagFile (Client.java line 2336: `stream.readFully(header, 0, 6)`) and Go GetJagFile (client.go:2530-2541). The divergence is real: Java's `readFully` blocks until exactly 6 bytes are read (or throws EOFException), whereas Go does a single `reader.Read(header)` and treats `n < 6` as an error that triggers `loadingError()` + `continue` (the retry/backoff path). These are NOT semantically equivalent for a partial-read source: a short-but-nonzero read (n in 1..5, err==nil) would make Go discard the partial header and restart the whole fetch, while Java would keep reading to fill the buffer.\n\nWhy it is only latent: `OpenURL` (client.go:6568-6583) reads the entire HTTP b […]
>
> **Refined:** Java getJagFile reads the 6-byte archive header with `stream.readFully(header, 0, 6)` (blocks until all 6 bytes arrive or throws EOF). Go GetJagFile (client.go:2530-2541) uses a single `reader.Read(header)` and treats `n < 6` as an error that triggers loadingError()+continue (full retry/backoff). Against the fully-buffered `*bytes.Reader` returned by OpenURL (which io2.ReadAll's the whole body), a single Read yields all 6 bytes, so this is currently equivalent. The divergence is latent: if OpenU […]

### L22. updateGame: missing `if (this.ingame) return;` after login() in title-screen state 2

- **Unit:** `client/Client#3`  **Java:** `Client.java:2541-2545`  **Go:** `pkg/jagex2/client/client.go:2424-2426`

In UpdateTitle's TitleScreenState==2 login-button branch, Java calls login() then `if (this.ingame) return;`. The Go calls c.LoginFunc(...) but omits the early return. After a successful login (InGame becomes true) the Go falls through to the cancel-button check and then enters the infinite PollKey() loop, processing leftover queued keys into username/password instead of returning immediately.

```
Java: `this.login(this.username, this.password, false); if (this.ingame) { return; }` vs Go: `c.LoginFunc(c.Username, c.Password, false)` (no InGame check follows).
```

> **Verifier (confirmed, severity latent):** Confirmed real divergence. Java updateTitle TitleScreenState==2 login-button branch (Client.java:2541-2545) is `this.login(this.username, this.password, false); if (this.ingame) { return; }`. The Go port (pkg/jagex2/client/client.go:2424-2426) calls `c.LoginFunc(c.Username, c.Password, false)` with NO `if c.InGame { return }` after it.

Traced the semantics to rule out false-positive:
- LoginFunc (client.go:6725) is the faithful port of Java login() and sets c.InGame = true on a successful login (client.go:6836 and 6982, mirroring Java Client.java:2680/2813). So after a successful click-login the Go state matches Java's (ingame=true) but the control flow diverges.
- In Java the function retu […]
>
> **Refined:** In UpdateTitle TitleScreenState==2 login-button branch, Java calls login() then `if (this.ingame) { return; }` (Client.java:2541-2545). The Go (pkg/jagex2/client/client.go:2424-2426) calls c.LoginFunc(...) but omits the early return. LoginFunc sets c.InGame=true on success. Without the return, Go falls through the cancel-button check (harmless — different X range) and into the `for { PollKey() }` loop (client.go:2433), which drains the shared key queue into c.Username/c.Password. Those fields ar […]

### L23. login reply==2: ViewportOverlayInterfaceID not reset to -1 (stale overlay leaks across re-login)

- **Unit:** `client/Client#3`  **Java:** `Client.java:2748`  **Go:** `pkg/jagex2/client/client.go:6889-6893`

Java login's successful-login reset sets `this.viewportOverlayInterfaceId = -1`. The Go reply==2 block resets StickyChat/Chat/Viewport/Sidebar interface ids but omits ViewportOverlayInterfaceID. The field is initialized to -1 only in NewClient; Logout does not reset it either. After a server sets it (opcode 158 at client.go:9998) then the player logs out and logs back in, the stale overlay interface id persists and renders.

```
Java 2748 `this.viewportOverlayInterfaceId = -1;`; Go 6889-6893 jumps StickyChatInterfaceID..PressedContinueOption with no ViewportOverlayInterfaceID = -1.
```

> **Verifier (confirmed, severity latent):** The core parity claim is confirmed. Java's successful-login reset (Client.java:2742-2748) clears six interface ids in order: stickyChatInterfaceId, chatInterfaceId, viewportInterfaceId, sidebarInterfaceId, viewportOverlayInterfaceId, pressedContinueOption. The Go reply==2 block (client.go:6889-6893) resets StickyChatInterfaceID, ChatInterfaceID, ViewportInterfaceID, SidebarInterfaceID, then jumps straight to PressedContinueOption — the `ViewportOverlayInterfaceID = -1` line is genuinely omitted (not relocated; an exhaustive grep shows only 3 references in the whole repo: decl at 538, NewClient init at 562, opcode-158 setter at 9998). I verified Logout (client.go:3576-3599) does not reset it  […]
>
> **Refined:** Java login reply==2 reset (Client.java:2748) sets viewportOverlayInterfaceId = -1; the Go reply==2 block (client.go:6889-6893) omits the corresponding ViewportOverlayInterfaceID = -1, resetting only StickyChat/Chat/Viewport/Sidebar then PressedContinueOption. The field is initialized to -1 only in NewClient (client.go:562); neither Go Logout (3576-3599) nor Java logout (2847) resets it (parity-correct — both reset only on login). Opcode 158 / IF_OPENOVERLAY sets it (client.go:9998 / Java 7573).  […]

### L24. updateGame: updateLocChanges() called out of order (after entity updates instead of before)

- **Unit:** `client/Client#3`  **Java:** `Client.java:2942-2963`  **Go:** `pkg/jagex2/client/client.go:7397-7464`

Java order inside the ingame block: updateSceneState() (2942), updateLocChanges() (2943), updateAudio() (2944), input flush, idleNetCycles, updatePlayers/updateNpcs/updateEntityChats. The Go calls UpdateLocChanges() at line 7464 AFTER UpdatePlayers/UpdateNpcs/UpdateEntityChats, rather than immediately after UpdateSceneState. The loc-change processing runs one phase later within the cycle; ordering relative to entity updates differs from Java.

```
Java: `this.updateSceneState(); this.updateLocChanges(); this.updateAudio();` then later `this.updatePlayers(); this.updateNpcs(); this.updateEntityChats();`. Go: UpdateSceneState (7398) ... UpdatePlayers (7461), UpdateNpcs (7462), UpdateEntityChats (7463), UpdateLocChanges (7464).
```

> **Verifier (confirmed, severity latent):** CONFIRMED REAL (could not refute), but severity downgraded from "bug" to "latent".

Decisive ordering evidence:
- Java 244 (git show 01f16088 Client.java:2942-2963) in-game block: updateSceneState() (2943) -> updateLocChanges() (2944) -> updateAudio() (2945) -> input flush -> idleNetCycles -> updatePlayers() (2961), updateNpcs() (2962), updateEntityChats() (2963) -> sceneDelta++. So updateLocChanges runs near the TOP, immediately after updateSceneState and before audio/entity updates.
- Go (pkg/jagex2/client/client.go): UpdateSceneState() (7398) -> inlined audio/wave loop (7400-7449) -> input flush (7450) -> IdleNetCycles (7457) -> UpdatePlayers (7461), UpdateNpcs (7462), UpdateEntityChats ( […]
>
> **Refined:** UpdateGame in-game block: Go calls UpdateLocChanges() last (pkg/jagex2/client/client.go:7464, after UpdatePlayers/UpdateNpcs/UpdateEntityChats), whereas Java 244 calls updateLocChanges() near the top, immediately after updateSceneState() and before updateAudio()/entity updates (Client.java:2944). Root cause: the 225 deob ran updateMergeLocs() after entity updates; commit a850b34 renamed UpdateMergeLocs()->UpdateLocChanges() in place during the 244 LocChange-merge port without relocating the call […]

### L25. login reply==15: missing sceneLoadStartTime reset

- **Unit:** `client/Client#3`  **Java:** `Client.java:2807`  **Go:** `pkg/jagex2/client/client.go:6981-6994`

Java login reply==15 (reconnect-without-reinit) sets `this.sceneLoadStartTime = System.currentTimeMillis();` as its last statement. The Go reply==15 block sets InGame..MenuVisible but omits c.SceneLoadStartTime = time.Now().UnixMilli(). SceneLoadStartTime drives the 360000ms scene-load timeout/report check in UpdateSceneState (client.go:8616-8628); on the reply==15 path it stays stale, so the timeout window is measured from a prior load.

```
Java 2807 `this.sceneLoadStartTime = System.currentTimeMillis();` has no counterpart in Go 6981-6994 (block ends at `c.MenuVisible = false; return`).
```

> **Verifier (confirmed, severity latent):** Verified against Java ref 01f16088. Java's reply==15 block (Client.java:2812-2825) ends with `this.sceneLoadStartTime = System.currentTimeMillis();` as its final statement, immediately after `this.menuVisible = false;`. The Go reply==15 block (client.go:6981-6994) ports every other statement faithfully (InGame, Out.Pos, In.Pos, PacketType/LastPacketType0-2, PacketSize, IdleNetCycles, SystemUpdateTimer, MenuSize, MenuVisible) but the block ends at `c.MenuVisible = false; return` with no `c.SceneLoadStartTime = time.Now().UnixMilli()`.

Ruled out false-positive causes:
- No cross-lineage name inversion: the Go field is declared at client.go:533 as `SceneLoadStartTime int64 // Java: sceneLoadSt […]

### L26. Logout does not reset NextMidiSong/MidiSong to -1

- **Unit:** `client/Client#3`  **Java:** `Client.java:2872-2873`  **Go:** `pkg/jagex2/client/client.go:3596-3598`

Java logout sets `this.nextMidiSong = -1; this.midiSong = -1; this.nextMusicDelay = 0;`. The Go Logout sets `c.CurrentMidi = ""; c.NextMusicDelay = 0` but does NOT reset the int c.NextMidiSong / c.MidiSong (which the 244 protocol path actually uses, opcode 240 at client.go:9967-9978 guards on `c.NextMidiSong != id`). After logout these retain their last values; on the next login a MIDI_SONG packet carrying the previously-playing id would be suppressed by the `NextMidiSong != id` guard and the track would not restart.

```
Java `this.nextMidiSong = -1; this.midiSong = -1;` — Go Logout has neither (only `c.CurrentMidi = ""`).
```

> **Verifier (confirmed, severity latent):** Verified against Java-244 (git show 01f16088) and the Go tree. The divergence is real and not refutable:

1. Java logout (Client.java:2871-2874) does: `stopMidi(); this.nextMidiSong = -1; this.midiSong = -1; this.nextMusicDelay = 0;`. Confirmed via grep at lines 2872-2873.
2. Go Logout (client.go:3596-3598) does: `c.StopMidi(); c.CurrentMidi = ""; c.NextMusicDelay = 0`. It omits BOTH `c.NextMidiSong = -1` and `c.MidiSong = -1`.
3. `CurrentMidi` is a 225-lineage string field with NO Java-244 analog — grep of the Java-244 source finds no `currentMidi` field at all. So `c.CurrentMidi = ""` does NOT substitute for the missing int resets; it belongs to the separate named-MIDI/SetMidi mechanism.
4 […]
>
> **Refined:** Java logout (Client.java:2871-2874) resets `nextMidiSong = -1; midiSong = -1; nextMusicDelay = 0` after stopMidi(). The Go Logout (client.go:3596-3598) resets only `c.NextMusicDelay = 0` and `c.CurrentMidi = ""` — but `CurrentMidi` is a 225-lineage string field with no Java-244 analog (the 244 source has no `currentMidi`). The int fields `c.NextMidiSong`/`c.MidiSong`, which the 244 protocol actually uses (opcode 240 at client.go:9972 guards on `c.NextMidiSong != id` before `c.OnDemand.Request(2, […]

### L27. GetJagFile: no post-download CRC re-validation, no loops counter / 'Game updated' message

- **Unit:** `client/Client#3`  **Java:** `Client.java:2349-2399`  **Go:** `pkg/jagex2/client/client.go:2553-2578`

Java getJagFile re-runs CRC32 on freshly downloaded data and, on mismatch, nulls data + increments `loops`, retrying; if loops>=3 it shows 'Game updated - please reload page' and forces the retry counter to 10. The Go verifies CRC only on cache-loaded data (2505-2510) but after a successful chunked download it falls straight to CacheSave/NewJagfile with NO CRC check, no loops counter, and no 'Game updated' message. A corrupted/truncated-but-EOF-clean download would be cached and used unverified. Part of the HTTP/storage transport seam.

```
Java post-download: `this.crc32.update(data); int checksum=(int)this.crc32.getValue(); if (crc != checksum) { data = null; loops++; }` and `if (loops >= 3) { this.drawProgress(progress, "Game updated - please reload page"); i = 10; }` — both absent in Go.
```

> **Verifier (confirmed, severity latent):** VERIFIED REAL. I read the full Java getJagFile (Client.java:2302-2422 via git 01f16088) and the full Go GetJagFile (client.go:2488-2580) plus callees.

Java post-download path (2376-2387): after the chunked read and cache write, Java runs `this.crc32.reset(); this.crc32.update(data); int checksum=(int)this.crc32.getValue(); if (crc != checksum) { data = null; loops++; }`. `int loops=0` is declared at 2327 before the `while(data==null)` loop. The retry-sleep block (2393-2402) does `if (loops >= 3) { this.drawProgress(progress, "Game updated - please reload page"); i = 10; }`.

Go (client.go:2553-2579): after the chunked-read loop completes without error (2573), control falls straight to `sign […]
>
> **Refined:** GetJagFile (client.go:2488-2580) omits Java's post-download integrity logic from getJagFile (Client.java:2302-2422). After a clean chunked download, Go calls CacheSave + NewJagfile directly with no CRC re-validation; Java instead recomputes CRC32 on the downloaded data and on mismatch nulls it + increments a `loops` counter to force a re-download, and shows "Game updated - please reload page" once loops>=3. Net effect: a same-length-corrupted download (one that survives HTTP/TCP/TLS and isn't ca […]

### L28. DrawError uses independent if blocks instead of Java's if/else-if chain

- **Unit:** `client/Client#3`  **Java:** `Client.java:2236-2299`  **Go:** `pkg/jagex2/client/client.go:8992-8031`

Java drawError is `if (errorLoading) {...} else if (errorHost) {...} else if (errorStarted) {...}` — exactly one branch runs. The Go runs `if c.ErrorLoading {...}` then `if c.ErrorHost {...}` then `if !c.ErrorStarted { draw; return }` then the errorStarted text. For mutually-exclusive flags (the only states reachable in practice) this is behaviorally equivalent. If two error flags were ever simultaneously true, Java shows only the first; the Go would overdraw both. Also Java fills 765x503 (g.fillRect) vs Go fills ScreenWidth/ScreenHeight (789x532 overlay) and uses errorfont/FontBold12 instead of AWT Helvetica — both documented platform-seam deviations.

```
Java `if (this.errorLoading) {...} else if (this.errorHost) {...} else if (this.errorStarted) {...}` vs Go three separate `if` statements at 8992/9011/9018.
```

> **Verifier (confirmed, severity latent):** Verified both sites in full. Java drawError (Client.java 2241/2269/2278) is a strict if/else-if/else-if chain over errorLoading/errorHost/errorStarted — exactly one branch (or none) runs. Go DrawError (client.go 8992/9011/9018) uses three independent statements: `if c.ErrorLoading {...}` (no return), `if c.ErrorHost {...}` (no return), `if !c.ErrorStarted { Draw; return }`, then the errorStarted text falls through. The finding's structural claim is factually accurate.

Traced every reachable flag state: for zero or one flag set, the Go is behaviorally identical to Java (Loading-only -> loading text; Host-only -> host text; Started-only -> falls through to started text; none -> blank). The on […]
>
> **Refined:** Go DrawError (client.go 8992/9011/9018) replaces Java's mutually-exclusive if/else-if/else-if chain (Client.java 2241/2269/2278) with three independent if blocks. Behaviorally equivalent for every reachable state because the three error flags are mutually exclusive by construction in both implementations: Java sets errorStarted/errorHost in early returns before the try whose catch sets errorLoading; the Go port additionally never sets ErrorHost (host allowlist intentionally unenforced) and sets  […]

### L29. handleChatMouseInput omits @cr1@/@cr2@ sender-prefix stripping before isFriend() and menu text

- **Unit:** `client/Client#4`  **Java:** `Client.java:3793-3806`  **Go:** `pkg/jagex2/client/client.go:1956-2017`

Same class as the handlePrivateChatInput finding. Java handleChatMouseInput strips @cr1@/@cr2@ from sender (sender=sender.substring(5)) and uses the stripped value in isFriend(sender), the localPlayer.name self-check (!sender.equals(localPlayer.name)), and all menu strings for the public (type 1/2) and unsplit-private (type 3/7) branches. Go uses c.MessageSender[i] raw everywhere (e.g. `c.MessageSender[i] != c.LocalPlayer.Name` at 1967, `c.IsFriend(c.MessageSender[i])` at 1966/1982). Modded senders therefore mis-match friend/self checks and produce menu text containing the literal crown tag.

```
Java: `if(sender!=null && sender.startsWith("@cr1@")){sender=sender.substring(5);...} ... !sender.equals(localPlayer.name) ... "Add friend @whi@"+sender`. Go (1967): `c.MessageSender[i] != c.LocalPlayer.Name`; (1976): `"Add friend @whi@" + c.MessageSender[i]` — raw.
```

> **Verifier (confirmed, severity latent):** Factually the finding is accurate. Java handleChatMouseInput (Client.java:3792-3802, confirmed via git show 01f16088) strips @cr1@/@cr2@ into a local `sender` and then uses the STRIPPED value in isFriend(sender), the !sender.equals(localPlayer.name) self-check, and every menu string for type 1/2/3/7. Go HandleChatMouseInput (client.go:1954-2020) performs NO stripping and uses c.MessageSender[i] raw at every site: 1966/1982/1998/2009 (IsFriend), 1967 (self-check), and 1969/1973/1976/etc (menu text). IsFriend in both languages does a direct equalsIgnoreCase/EqualFold with no internal stripping (Java 12133-12145; Go 10592-10605), so the prefix must be stripped at the call site — which Go does n […]
>
> **Refined:** Go HandleChatMouseInput (client.go:1954-2020) omits the @cr1@/@cr2@ sender-prefix stripping that Java Client.java:3792-3802 performs before isFriend(), the localPlayer.name self-check, and all menu strings — a genuine divergence. However this is currently LATENT, not an active bug: the crown-prefix producer logic is entirely unported and explicitly documented as deferred (client.go:5873-5875). No Go call site ever writes a @cr1@/@cr2@-prefixed string into MessageSender (the three Java producer s […]

### L30. HandleInputKey chat typing drops the 244 '::'-prefix upper-bound extension to char 126

- **Unit:** `client/Client#5`  **Java:** `Client.java:4690 (handleInputKey)`  **Go:** `pkg/jagex2/client/client.go:2261 (HandleInputKey)`

Java-244 allows characters up to 126 (not just 122) into chatTyped when the input is a ::command: `if (key >= 32 && (key <= 122 || (this.chatTyped.startsWith("::") && key <= 126)) && this.chatTyped.length() < 80)`. The Go condition is only `if var2 >= 32 && var2 <= 122 && len(c.ChatTyped) < 80` — the `|| (startsWith("::") && key <= 126)` clause was dropped. Consequently chars 123-126 ({ | } ~) can never be typed, even inside a cheat/admin command that needs them.

```
Go 2261: `if var2 >= 32 && var2 <= 122 && len(c.ChatTyped) < 80 {`. Java 4690: `if (key >= 32 && (key <= 122 || (this.chatTyped.startsWith("::") && key <= 126)) && this.chatTyped.length() < 80) {`.
```

> **Verifier (confirmed, severity latent):** Confirmed by direct side-by-side comparison at the exact cited lines.

Go pkg/jagex2/client/client.go:2261 (inside the `else if c.ChatInterfaceID == -1` branch of HandleInputKey):
  `if var2 >= 32 && var2 <= 122 && len(c.ChatTyped) < 80 {`
Java-244 Client.java:4690 (inside `else if (this.chatInterfaceId == -1)`, verified via `git show 01f16088`):
  `if (key >= 32 && (key <= 122 || (this.chatTyped.startsWith("::") && key <= 126)) && this.chatTyped.length() < 80) {`

The `|| (this.chatTyped.startsWith("::") && key <= 126)` sub-clause is genuinely absent in Go.

Refutation attempts all failed:
- Variable semantics match: Go `var2 = c.PollKey()` == Java `key = this.pollKey()`; no deob-arg scramb […]
>
> **Refined:** In HandleInputKey's `c.ChatInterfaceID == -1` branch (pkg/jagex2/client/client.go:2261), the condition admitting characters into c.ChatTyped is `var2 >= 32 && var2 <= 122 && len(c.ChatTyped) < 80`, missing Java-244's `|| (this.chatTyped.startsWith("::") && key <= 126)` sub-clause (Client.java:4690). As a result, characters 123-126 ({ | } ~) cannot be typed even inside a `::` cheat/admin command. Impact is confined to those 4 edge-value characters and only when the buffer already starts with `::` […]

### L31. pushPlayers tile-occupied skip missing 244's `&& i != -1` guard (local player not exempted)

- **Unit:** `client/Client#6`  **Java:** `Client.java:5983`  **Go:** `pkg/jagex2/client/client.go:2046-2048`

Java-244 guards the tile-occupied early-continue with `&& i != -1`, exempting the local player (i == -1) from being skipped when its scene tile was already marked this cycle. Go (faithful to 225-clean deob/client.java:2618) omits `&& i != -1`. In practice the local player is the FIRST loop iteration (i == -1) so TileLastOccupiedCycle has not yet been set to the current sceneCycle for its tile, meaning the skip almost never fires for the local player — hence latent rather than an active bug. It would only diverge if some other code pre-marked the local player's tile in the same cycle.

```
Go: `if c.TileLastOccupiedCycle[var5][var6] == c.SceneCycle { continue }`; Java-244: `if (this.tileLastOccupiedCycle[stx][stz] == this.sceneCycle && i != -1) { continue; }`.
```

> **Verifier (confirmed, severity latent):** VERIFIED REAL, latent. Java-244 Client.java:5983 guards the tile-occupied early-continue in pushPlayers with `&& i != -1`, exempting the local player (i == -1). Go client.go:2046-2048 omits it: `if c.TileLastOccupiedCycle[var5][var6] == c.SceneCycle { continue }`. The Go is a faithful port of its 225-clean reference (verified directly: 225-clean client.java:2618 reads `if (this.tileLastOccupiedCycle[var5][var6] == this.sceneCycle) { continue; }` with NO `&& i != -1`), so this is a genuine Java-225 vs Java-244 semantic divergence, not a Go porting error.

Ruled out false-positive causes: not a deob arg scramble (single-array indexed compare), not a cross-lineage name inversion (computations m […]
>
> **Refined:** Java-244 (Client.java:5983) guards the pushPlayers tile-occupied early-continue with `&& i != -1`, exempting the local player (i == -1). Go (client.go:2046-2048) omits it, faithfully matching 225-clean (client.java:2618), where the guard does not exist. This is a real Java-225-vs-244 divergence, currently LATENT in Go: the guard's purpose only matters because Java-244 also reordered drawScene to call pushNpcs(true) BEFORE pushPlayers() (Client.java:5842-5843), letting an always-on-top NPC pre-ma […]

### L32. DrawTooltip/DrawMenu call PixFont tag helpers whose EvaluateTag returns 0 (black) for unknown tags instead of -1 (keep color)

- **Unit:** `client/Client#7`  **Java:** `PixFont.java:181-206 (drawStringAntiMacro), 174-202 (drawStringTag)`  **Go:** `pkg/jagex2/graphics/pixfont/pixfont.go:267 (DrawStringTooltip), 234 (DrawStringTaggable), EvaluateTag 285-323`

This is a callee defect outside my unit (PixFont) but it changes my methods' output. Java's evaluateTag returns -1 for an unrecognized @xxx@ tag and the caller keeps the previous color (`if (var11 != -1) arg5 = var11;`). The Go EvaluateTag default branch returns 0 (black), and DrawStringTooltip/DrawStringTaggable assign `arg4 = p.EvaluateTag(...)` / `arg5 = p.EvaluateTag(...)` unconditionally. So an unknown tag in a tooltip (DrawTooltip) or menu option (DrawMenu) would turn the rest of the string black in Go vs unchanged in Java. Also DrawStringTaggable omits Java's strikeout/hline tail. Reported here because the call sites are in my unit; the fix belongs to the pixfont auditor.

```
Go EvaluateTag: `default: return 0`. Go DrawStringTooltip: `arg4 = p.EvaluateTag(string(runes[i+1 : i+4]))` (unconditional). Java: `int var11 = this.evaluateTag(...); if (var11 != -1) { arg5 = var11; }`.
```

> **Verifier (confirmed, severity latent):** Confirmed both halves of the finding against the Java-244 reference (git show 01f16088:.../PixFont.java) and the Go code.

PRIMARY DEFECT (unknown-tag color handling):
- Java evaluateTag (lines ~236-276): the final else branch sets `strikeout=true` if arg1.equals("str") and then `return -1;`. So any unrecognized 3-char tag yields -1.
- Both Java callers GUARD the assignment: drawStringTag has `int var10 = evaluateTag(...); if (var10 != -1) { arg0 = var10; }`; drawStringAntiMacro has `int var11 = evaluateTag(...); if (var11 != -1) { arg5 = var11; }`. An unknown @xxx@ tag therefore leaves the current color unchanged.
- Go EvaluateTag (pixfont.go:321-322) has `default: return 0` — black, no -1  […]
>
> **Refined:** PixFont.EvaluateTag returns 0 (black) for unrecognized @xxx@ tags whereas Java's evaluateTag returns -1, and the client call sites DrawMenu (client.go:5213 via DrawStringTaggable) and DrawTooltip (client.go:7677 via DrawStringTooltip) assign the result unconditionally — so an unknown tag blackens the rest of a menu option or tooltip in Go vs. keeping the prior color in Java. Java guards with `if (var != -1) color = var;`. Additionally, Go's EvaluateTag/DrawStringTaggable omit Java's `@str@` stri […]

### L33. Opcode 70 UPDATE_FRIENDLIST capped at 100 friends instead of Java's 200

- **Unit:** `client/Client#8`  **Java:** `Client.java:7407-7413`  **Go:** `pkg/jagex2/client/client.go:9742-9748`

Java caps the friend list with `if (displayName != null && this.friendCount < 200)` and declares friendName37/friendWorld/friendName as new[200] (Client.java:515,671,710). Go uses `if !matched && c.FriendCount < 100` and the arrays are make([]..,100) (client.go:565,607,638). A player with more than 100 friends will not have friends 101-200 added in the Go client. The in-unit symptom is the `< 100` cap at the handler; the root cause is the 100-sized arrays (declarations outside this unit's line range). FriendName37 is correctly []int64.

```
Java: `if (displayName != null && this.friendCount < 200) {` with `friendName37 = new long[200]`  |  Go: `if !matched && c.FriendCount < 100 {` with `FriendName37: make([]int64, 100)`
```

> **Verifier (confirmed, severity latent):** Confirmed real after attempting refutation. Java-244 opcode 70 (UPDATE_FRIENDLIST) handler at Client.java:7407 caps with `if (displayName != null && this.friendCount < 200)`, and the backing arrays are declared length 200: friendName37 = new long[200] (515), friendWorld = new int[200] (671), friendName = new String[200] (710). The Go opcode 70 handler at client.go:9742 uses `if !matched && c.FriendCount < 100`, and the arrays are make([]int64,100) (607), make([]int,100) (565), make([]string,100) (638). No 200-sized allocation or reallocation exists anywhere in client.go (grep for `, 200)` and `FriendName* =` shows only the title image, the nil-resets, and the 100-sized makes). FriendName37 i […]
>
> **Refined:** Go does not implement the Java-244 feature where members accounts get a 200-friend list (free accounts stay at 100). Java-244 opcode 70 handler caps at `friendCount < 200` (Client.java:7407) with friend arrays sized 200 (515/671/710); Go caps the opcode 70 handler at `c.FriendCount < 100` (client.go:9742) with arrays make(...,100) (565/607/638), and the addFriend cap is a flat `>= 100` (client.go:7147) versus Java's `>= 100 && membersAccount != 1` else `>= 200` (Client.java:12154). MembersAccoun […]

### L34. Opcode 226 VARP_LARGE g4 value not int32-wrapped — sign divergence for high-bit varps

- **Unit:** `client/Client#8`  **Java:** `Client.java:7614-7615`  **Go:** `pkg/jagex2/client/client.go:10042-10047`

Java `int value = this.in.g4();` returns a 32-bit signed int; when the high byte has bit7 set the value is negative. Go `var4 := c.In.G4()` returns a 64-bit int holding the unsigned bit pattern [0,2^32) (Go G4 does not sign-extend — packet.go:238-240). varps/varCache are int[2000] in Java (signed) and []int in Go. For a varp whose g4 value has the sign bit set, Java stores a negative value and Go stores a large positive value, so c.Varps[varp] and downstream updateVarp/component logic can diverge. Unreachable for ordinary small varp values; latent. Fix per project convention: `var4 := int(int32(c.In.G4()))`. (UPDATE_STAT opcode 24 xp=g4 is bounded <2^31 so not affected; MESSAGE_PRIVATE messageId=g4 is only used for self-consistent dedup so harmless.)

```
Java: `int value = this.in.g4();` (g4 returns 32-bit int, sign bit -> negative)  |  Go: `var4 := c.In.G4()` (returns positive [0,2^32) for the same bytes); `c.Varps[var26] = var4`
```

> **Verifier (confirmed, severity latent):** CONFIRMED real and correctly classified as latent. Decisive evidence traced through all four links:

1. Java `g4()` (Packet.java:237-240) composes bytes in 32-bit signed int arithmetic: `(this.data[pos-1] & 0xFF) + ... + ((this.data[pos-4] & 0xFF) << 24)`. When the top byte's bit7 is set, `<<24` lands in the sign bit and the result is NEGATIVE. `Client.java:7615` stores this directly: `int value = this.in.g4();`.

2. Go `G4()` (packet.go:237-241) does the identical byte composition but in Go `int` (64-bit on linux/amd64), so `<<24` never reaches a sign bit — the result is always in [0, 2^32), NON-negative. `client.go:10043` `var4 := c.In.G4()` stores the positive value. No int32() wrap exist […]
>
> **Refined:** Opcode 226 VARP_LARGE: Go stores the g4 value without an int32 wrap (client.go:10043 `var4 := c.In.G4()`), so a value with the high bit set is stored as a large positive int in [0,2^32) (Go G4 returns 64-bit non-sign-extended, packet.go:237-241), whereas Java's g4 (Packet.java:237-240) returns a negative signed 32-bit int (Client.java:7615). Varps/VarCache are int[2000] in Java vs []int in Go, so the stored value's sign diverges. Observable via ExecuteClientscript1 (client.go:8899/8907/8941) whe […]

### L35. NewModel1 orientation loop: faceVertex writes were moved INSIDE the per-orientation if-blocks; Java writes them unconditionally every face

- **Unit:** `dash3d/Model#1`  **Java:** `Model.java:522-553`  **Go:** `pkg/jagex2/dash3d/model/model.go:461-498`

In Java's vertex-orientation loop the three assignments `this.faceVertexA[f]=a; this.faceVertexB[f]=b; this.faceVertexC[f]=c;` sit AFTER the `if(orientation==1)...else if(==4)` chain and execute UNCONDITIONALLY for every face. So when an orientation byte is not 1/2/3/4 (e.g. 0 or >=5), Java still writes the carried-over a/b/c (the previous face's values). The Go port hoisted the three writes into each `if var14==N { ... }` block, so for an orientation not in {1,2,3,4} Go writes nothing and the face's FaceVertexA/B/C remain 0 instead of the carried-over values. The `vertex2.G1()` orientation read is still unconditional, so the packet cursor stays in sync (not a cache desync) — only the face index array values diverge. For all valid model data orientation is always 1-4, so the difference is unreachable in practice; latent for malformed/edge blobs.

```
Java: `} else if (orientation == 4) { ... } \n this.faceVertexA[f] = a; this.faceVertexB[f] = b; this.faceVertexC[f] = c;` (writes outside chain). Go: each branch ends with its own `m.FaceVertexA[i]=var9; m.FaceVertexB[i]=...; m.FaceVertexC[i]=var11` and there is NO unconditional write after the four `if` blocks.
```

> **Verifier (confirmed, severity latent):** Confirmed by reading both full functions. Java Model.java:522-553 (NewModel1 orientation loop) reads `int orientation = vertex2.g1()` unconditionally, runs an `if(orientation==1)...else if(==2)...else if(==3)...else if(==4)` chain, and then executes the three writes `this.faceVertexA[f]=a; this.faceVertexB[f]=b; this.faceVertexC[f]=c;` UNCONDITIONALLY after the chain. So for an orientation byte not in {1,2,3,4}, Java still stores the carried-over a/b/c (previous face's values, or 0 on the first face).\n\nGo model.go:461-498 reads `var14 = vertex2.G1()` unconditionally, but converts the chain into four separate `if var14==N {...}` blocks and HOISTS the three writes (`m.FaceVertexA/B/C[i]=...` […]

### L36. Bounds/normals squared-magnitude sums computed in Go 64-bit int can diverge from Java 32-bit wrap for huge models

- **Unit:** `dash3d/Model#2`  **Java:** `Model.java:1017-1018,1088-1089,1407`  **Go:** `pkg/jagex2/dash3d/model/model.go:972-973,1010-1011,1294`

Expressions like `radius*radius + minY*minY` (CalculateBoundsCylinder/Y/AABB) and `lightSrcX*lightSrcX + lightSrcY*lightSrcY + lightSrcZ*lightSrcZ` (CalculateNormals) are evaluated in Java as 32-bit int (overflow wraps) and then widened to double for Math.sqrt. The Go port computes the sum in 64-bit int before the float64() conversion. For ordinary model dimensions (radius well under 46340) neither overflows and results are identical; only pathologically large coordinate magnitudes would make Java wrap negative while Go does not. Edge-only; matches the known int32-vs-int64 truncation class. No int32() wrap is present, consistent with surrounding ported code.

```
Go: `int(math.Sqrt(float64(m.Radius*m.Radius+m.MaxY*m.MaxY)) + 0.99)` — 64-bit int sum. Java: `(int)(Math.sqrt(this.radius * this.radius + super.minY * super.minY) + 0.99D)` — 32-bit int sum then widen.
```

> **Verifier (confirmed, severity latent):** Verified all three cited sites against Java-244 (git 01f16088:src/main/java/jagex2/dash3d/Model.java) and Go (pkg/jagex2/dash3d/model/model.go).

Code-site correspondence is accurate:
- calculateBoundsCylinder Java ~1017-1018: `(int)(Math.sqrt(this.radius*this.radius + super.minY*super.minY)+0.99D)` ↔ Go 972-973 `int(math.Sqrt(float64(m.Radius*m.Radius+m.MaxY*m.MaxY))+0.99)`.
- calculateBoundsAABB Java ~1088-1089 ↔ Go 1010-1011.
- calculateNormals Java 1407 `(int)Math.sqrt(lightSrcX*lightSrcX + lightSrcY*lightSrcY + lightSrcZ*lightSrcZ)` ↔ Go 1294 `int(math.Sqrt(float64(arg2*arg2 + arg3*arg3 + arg4*arg4)))`.

Width difference confirmed: in Java `radius`, `minY`, `maxY`, vertex coords, and `l […]

### L37. 3D draw arithmetic uses Go 64-bit int without int32() wrap (project-wide convention)

- **Unit:** `dash3d/Model#3`  **Java:** `Model.java:1582-1601, 1638-1687, 1762-1788`  **Go:** `pkg/jagex2/dash3d/model/model.go:1442-1463, 1557-1568, 1827-1885`

Products such as sin*coord and (vvX-x)*reciprocal*shift are 32-bit-wrapping in Java but use Go `int` (64-bit) here, so an overflow that wraps in Java would not wrap in Go. Verified that model.go AND the sibling pix3d.go rasterizer both use ZERO int32() wraps — this is a deliberate, established project-wide convention for the bounded 3D pipeline (model radius/coords keep products within int32 in normal gameplay). Flagged for completeness per the type-width directive; not a per-method regression since this range matches the codebase convention exactly. Only reachable for pathological coordinate magnitudes.

```
e.g. Go `var23 = (var21*var14 + var20*var15) >> 16` with no int32() vs Java `int var23 = var14 * var21 + var15 * var20 >> 16` (32-bit wrap). `grep -c int32( ` returns 0 in both model.go and pix3d.go.
```

> **Verifier (confirmed, severity latent):** Verified all factual claims against Java rev-244 (src/main/java/jagex2/dash3d/Model.java via git show 01f16088) and the Go port (pkg/jagex2/dash3d/model/model.go).

CONFIRMED FACTS:
1. Java declares these intermediaries as 32-bit `int` (e.g. drawSimple Model.java:1582 `int var18 = arg5*var16 + arg6*var17 >> 16`; draw 1638-1687; clipFace interpolation 1762-1788). The Go port uses `int` (64-bit on amd64) with no int32() wrap — model.go:1442/1564/1828 etc.
2. `grep -c 'int32('` = 0 in both model.go and pix3d.go (pix3d lives at graphics/pix3d/pix3d.go). The evidence example `var23 = (var21*var14 + var20*var15) >> 16` matches Java `int var23 = var14*var21 + var15*var20 >> 16`.
3. The sin/cos tabl […]
>
> **Refined:** 3D draw/clip arithmetic in dash3d/model/model.go (drawSimple, draw, clipFace) uses Go 64-bit `int` for products that are 32-bit `int` in Java rev-244, so a hypothetical overflow that wraps in Java would not wrap in Go. Verified the divergence is real but unreachable in normal play: model.go's draw functions only ever receive bounded inputs — camera-relative scene coords (~±13000), near/far-clipped depths (>=50/<3500), screen-space coords (~±5000), and offsets that were already int32()-wrapped at […]

### L38. typecode/BitSet built as 64-bit int with no int32 wrap (addLoc + standalone AddLoc)

- **Unit:** `dash3d/World#1`  **Java:** `World.java:341-344`  **Go:** `pkg/jagex2/dash3d/world/world.go:295-298,1029-1032`

Java builds typecode as a 32-bit int: `(locId<<14)+(z<<7)+x+0x40000000`, then `typecode += Integer.MIN_VALUE` for inactive locs, relying on 32-bit overflow wraparound. Go uses native 64-bit int (`typeCode += math.MinInt32`) with no int32() wrap. For ACTIVE locs with locId<32768 the active value stays positive and <2^31, so it is bit-identical to Java (verified: max active typecode ~1.61e9). The divergence is confined to (a) inactive locs, where Java's value is a 32-bit negative (bits 29,30 = 0b11 via wrap) while Go produces a 64-bit negative with different high bits, and (b) locId>=32768 where `locId<<14` overflows bit 28. GetLocBitSet only returns the bitset when `(BitSet>>29)&0x3 == 2`, which both lineages satisfy only for active locs (identical), and fail for inactive locs (both !=2) — so no behavioral difference is reachable with the real cache. Latent because all typecode equality/identity comparisons in the port are self-consistent (same 64-bit construction everywhere) and locId<32768 in practice.

```
Java: `int typecode = (locId << 14) + (z << 7) + x + 0x40000000; if (!loc.active) { typecode += Integer.MIN_VALUE; }` (World.java:341-344). Go: `typeCode := x + (z << 7) + (locId << 14) + 0x40000000; if !loc.Active { typeCode += math.MinInt32 }` (world.go:295-298). Consumer: `(var7.BitSet>>29)&0x3 == 2` (world3d.go:534).
```

> **Verifier (confirmed, severity latent):** VERDICT: real=true, severity=latent. The finding accurately identifies a genuine code-level divergence: both typecode constructions in the Go port — World.AddLoc (pkg/jagex2/dash3d/world/world.go:295-298) and the standalone AddLoc (world.go:1029-1032) — build the value in native 64-bit int (`x + (z<<7) + (locId<<14) + 0x40000000`, then `+= math.MinInt32` for inactive locs) with NO `int32()` wrap, whereas Java (World.java:341-344) uses a 32-bit int relying on overflow wraparound. That omitted wrap is a real, accurate observation.

However, no behavioral divergence is reachable — confirming the latent severity — and the finding's verdict is right for a STRONGER reason than it states. I refuted […]
>
> **Refined:** Java builds typecode as a 32-bit int (`(locId<<14)+(z<<7)+x+0x40000000`, then `+= Integer.MIN_VALUE` for inactive locs, relying on 32-bit overflow). Go builds it in native 64-bit int (`x+(z<<7)+(locId<<14)+0x40000000`; `+= math.MinInt32`) with no int32() wrap, at both world.go:295-298 and world.go:1029-1032. This is LATENT with NO reachable behavioral difference: (a) for ACTIVE locs the value is positive and < 2^31 (max 1610612735), bit-identical to Java; (b) for INACTIVE locs the Go value is in […]

### L39. overlayShape+1 computed in int8 before widening (build overlay-tile path)

- **Unit:** `dash3d/World#1`  **Java:** `World.java:164`  **Go:** `pkg/jagex2/dash3d/world/world.go:715`

Java `int var89 = this.overlayShape[...] + 1` promotes the signed byte to int (sign-extend) THEN adds 1 in 32-bit int arithmetic. Go writes `var39 = int(w.LevelTileOverlayShape[i][j][l] + 1)`: because the `1` is an untyped constant it adopts the int8 element type, so `+1` happens in int8 and only then is widened to int. For element value 127 Java yields 128 while Go wraps to -128. In practice overlayShape originates from `(var13-2)/4` with var13<=49 (max 11), so the value range is 0..12 and the result is identical. Latent: only differs for unreachable int8-overflow inputs; correct fix is `int(...)+1`.

```
Java: `int var89 = this.overlayShape[var5][var58][var67] + 1;` (World.java:164). Go: `var39 = int(w.LevelTileOverlayShape[i][j][l] + 1)` (world.go:715).
```

> **Verifier (confirmed, severity latent):** Verified both sites. Java (World.java:806, inside the BuildSquares/setTile overlay path — NOT line 164 as the javaRef field claims): `int var89 = this.overlayShape[var5][var58][var67] + 1;` where overlayShape is byte[][][] (field decl line 41). Java promotes the signed byte to int (sign-extend) BEFORE adding 1, so the add is 32-bit: value 127 -> 128. Go (world.go:715): `var39 = int(w.LevelTileOverlayShape[i][j][l] + 1)` where LevelTileOverlayShape is [][][]int8 (decl line 61). The untyped constant 1 adopts the int8 element type, so int8(127)+int8(1) overflows to -128, then int(-128) = -128. The two genuinely diverge for element value 127 (128 vs -128). Could not refute the semantic claim.

R […]
>
> **Refined:** Java `int var89 = this.overlayShape[var5][var58][var67] + 1;` (World.java:806, not :164 as cited) promotes the signed byte to int (sign-extend) THEN adds 1 in 32-bit arithmetic. Go `var39 = int(w.LevelTileOverlayShape[i][j][l] + 1)` (world.go:715): the untyped constant 1 adopts the int8 element type, so +1 happens in int8 and wraps before widening — value 127 yields Java 128 vs Go -128. In practice overlayShape is only ever written as `(var13-2)/4` with var13 gated to 2..49 (decodeGround), so it […]

### L40. noise() 32-bit multiply chain runs at 64-bit width in Go but is provably identical due to & MaxInt32 low-31-bit masking

- **Unit:** `dash3d/World#2`  **Java:** `World.java:1035-1040`  **Go:** `pkg/jagex2/dash3d/world/world.go:947-952`

Java noise() is a 32-bit-wrapping PRNG: var4 = (var3*var3*15731+789221)*var3 + 1376312589 & Integer.MAX_VALUE, then var4 >> 19 & 0xFF. Java int overflows/wraps mod 2^32 in the multiply chain. Go's int is 64-bit, so var4*(var4*var4*15731+789221) is computed exactly without 32-bit wrap. I flagged this as a potential int32-overflow class trap and tested it: because the final result only extracts bits 19-26 after & math.MaxInt32 (0x7FFFFFFF masks to the low 31 bits) and the low N bits of a product depend only on the low N bits of the operands, the Go 64-bit value and the Java 32-bit-wrapped value always agree in the low 31 bits. Verified 0 mismatches over 5,000,000 full-range int32 random inputs AND the dense -3000..3000 grid. NOT a bug; recorded as latent so future readers know the missing int32() wraps are intentionally safe here. Note also Go swaps the local var names relative to Java (Go var4 = Java var3 = (var2<<13)^var2; Go var3 = Java var4 = masked product) — cosmetic naming inversion only, values correct.

```
Java: int var4 = (var3 * var3 * 15731 + 789221) * var3 + 1376312589 & Integer.MAX_VALUE; return var4 >> 19 & 0xFF;  Go: var3 := (var4*(var4*var4*15731+789221) + 1376312589) & math.MaxInt32; return (var3 >> 19) & 0xFF  -> brute-force test: total mismatches: 0
```

> **Verifier (confirmed, severity latent):** Both code sites confirmed. Java World.java noise() (244 ref, lines as cited): `int var2 = arg1*57 + arg0; int var3 = var2<<13 ^ var2; int var4 = (var3*var3*15731+789221)*var3 + 1376312589 & Integer.MAX_VALUE; return var4>>19 & 0xFF;`. Go pkg/jagex2/dash3d/world/world.go:947-951: `var2 := arg0 + arg1*57; var4 := (var2<<13)^var2; var3 := (var4*(var4*var4*15731+789221)+1376312589) & math.MaxInt32; return (var3>>19)&0xFF`.

Structural equivalence verified: (1) var2 add is commutative (identical); (2) the local-name inversion is real — Go's var4 == Java's var3 (the `<<13 ^` seed), Go's var3 == Java's var4 (masked product); (3) Java `(s*s*15731+789221)*s` == Go `s*(s*s*15731+789221)` by commutativ […]
>
> **Refined:** Confirmed accurate as written. noise() differs only in arithmetic width (Java 32-bit wrap vs Go 64-bit) and in local var naming (Go var4 == Java var3, Go var3 == Java var4 — cosmetic inversion). The output extracts bits 19-26 after & MaxInt32 (low 31 bits); since low-bit arithmetic is wrap-invariant, the Go 64-bit and Java 32-bit-wrapped results are provably identical. Independently verified: 0 mismatches over the dense -3000..3000 grid, 5M random full-int32 inputs, and high-bit-biased/corner in […]

### L41. AddLoc2 has a Go-only nil-model guard absent from Java addLoc/13

- **Unit:** `dash3d/World3D#1`  **Java:** `World3D.java:473-484`  **Go:** `pkg/jagex2/dash3d/world3d/world3d.go:395-398`

Java's 13-arg addLoc (addLoc2) body starts directly with the bounds loop and has NO null check on the ModelSource (arg8). The Go AddLoc2 adds an early `if arg8 == nil { return false }`. All three current callers (AddLoc1, AddTemporary1, AddTemporary2) null-check the model and return true before invoking AddLoc2, so arg8 is never nil in practice and the guard is unreachable today. It is still a divergence: if AddLoc2 were ever reached with a nil model, Java proceeds (sets sprite.model=null) and returns true, whereas Go returns false. Different return value and side effects in that path.

```
Go line 396: `if arg8 == nil { return false }` vs Java addLoc whose first statement is `for (int var14 = arg1; var14 < arg1 + arg3; var14++) {` with no arg8==null test anywhere.
```

> **Verifier (confirmed, severity latent):** Verified both sites in full. Java's 13-arg addLoc (@ObfuscatedName "s.a(IIIIIIIILy;IZIB)Z", the AddLoc2 analogue) has NO null check on the ModelSource (arg8): its first statement is the bounds loop `for (int var14 = arg1; ...)`, and it unconditionally assigns `var15.model = arg8` and returns true. Go's AddLoc2 (world3d.go:396-398) adds an early `if arg8 == nil { return false }`. This is a genuine Go-only guard with two divergent effects on the nil path: return value (Java true vs Go false) and side effects (Java still constructs the Sprite, populates tiles, and pushes to changedLocs/TemporaryLocs; Go bails before any of that).\n\nReachability: I enumerated all callers. There are exactly thre […]

### L42. BuildModels light-magnitude computed in 64-bit (Go) vs 32-bit wrapping (Java)

- **Unit:** `dash3d/World3D#1`  **Java:** `World3D.java:721`  **Go:** `pkg/jagex2/dash3d/world3d/world3d.go:663`

Java computes lightSrcX*lightSrcX + lightSrcY*lightSrcY + lgihtSrcZ*lgihtSrcZ in 32-bit int (wraps past 2^31) before Math.sqrt; Go computes arg2*arg2 + arg0*arg0 + arg4*arg4 in 64-bit then casts to float64, never wrapping. For realistic light vectors the sum is far below 2^31 so results are identical; divergence only for pathological magnitudes that never occur. Argument role map (lightSrcX/Y/Z <-> arg4/arg0/arg2; applyLighting first arg arg1=Java arg4) verified consistent.

```
Go: `lightMagnitude := int(math.Sqrt(float64(arg2*arg2 + arg0*arg0 + arg4*arg4)))` vs Java: `int lightMagnitude = (int) Math.sqrt(lightSrcX * lightSrcX + lightSrcY * lightSrcY + lgihtSrcZ * lgihtSrcZ);`
```

> **Verifier (confirmed, severity latent):** Verified both sites in full. Java World3D.java:721 computes `int lightMagnitude = (int) Math.sqrt(lightSrcX*lightSrcX + lightSrcY*lightSrcY + lgihtSrcZ*lgihtSrcZ)` where the sum is evaluated in 32-bit int (can overflow/wrap past 2^31). Go world3d.go:663 computes `int(math.Sqrt(float64(arg2*arg2 + arg0*arg0 + arg4*arg4)))` where the sum is evaluated in 64-bit int then cast to float64 (no wrap). That width difference is real.

Refutation attempts all fail to dismiss it but confirm it is harmless: (1) The deob parameter scramble is the key trap and I traced it. Go param order is (arg0,arg1,arg2,lightAttenuation,arg4); Java is (lightSrcY,lightSrcX,lgihtSrcZ,arg4,lightAttenuation). The single cal […]
>
> **Refined:** Java World3D.java:721 computes the light-magnitude squared-sum in 32-bit int (wraps past 2^31) before Math.sqrt; Go world3d.go:663 computes it in 64-bit int then casts to float64 (never wraps). Argument-role mapping is correct: the Go call site BuildModels(-10,64,-50,768,-50) reorders args to match Go's param order so the three squared light components (-50,-10,-50) equal Java's, both summing to 5100. The sole caller passes compile-time constants whose squared sum (5100) is far below 2^31, so ne […]

### L43. MinY is per-node in Go but shared per-ModelSource in Java

- **Unit:** `dash3d/World3D#1`  **Java:** `ModelSource.java:12`  **Go:** `pkg/jagex2/dash3d/world3d/world3d.go:411-423`

Java stores minY on the ModelSource object, so multiple scene nodes referencing the SAME ModelSource share one minY (a draw via one updates the value read by the other). Go's MinY lives on each Sprite/Decor node, so they are independent. Only matters if a single mutable ClientLocAnim instance is referenced by more than one scene node; for static Models minY is constant so per-node copies are always equal, and in normal scene building each loc owns its own anim, making this unreachable in practice. Flagged as an edge-value latent divergence from the ModelSource-seam port.

```
Java: `class ModelSource { public int minY = 1000; }` (shared object field). Go seeds `var22.MinY = m.MinY` onto the Sprite node and `decor.MinY = m.MinY` onto the Decor node, never onto a shared model object.
```

> **Verifier (confirmed, severity latent):** VERIFIED REAL, severity confirmed as latent.

Structural premise confirmed: Java `ModelSource.minY` is a field on the shared ModelSource object (ModelSource.java:12, default 1000), and the Java scene-node classes Sprite.java and Decor.java have NO minY field of their own — they only hold a `ModelSource model` reference. The cull reads `decor.model.minY` (World3D.java:1365,1632) and `farthest.model.minY` (:1563), and `ModelSource.draw` writes `this.minY = model.minY` back onto that same shared object each draw. In Go, `MinY` is a per-node field on typ.Sprite (sprite.go:29) and typ.Decor (decor.go:20); world3d.go seeds `var22.MinY = m.MinY` / `decor.MinY = m.MinY` and updates them per-node at  […]
>
> **Refined:** Java stores minY on the shared ModelSource object (Sprite/Decor have no minY field), so multiple scene nodes referencing the SAME ModelSource share one minY; Go stores MinY per-node on each Sprite/Decor. This only diverges if a single mutable ClientLocAnim is referenced by 2+ nodes, which never happens: every scene-build branch (Java World.java, Go world/world.go) constructs a fresh ClientLocAnim, and static Model bounds are constant — so the per-node copies never diverge in practice. Correctly  […]

### L44. PointInsideTriangle product-of-cross-products comparison lacks int32 wrap (Go 64-bit vs Java 32-bit sign)

- **Unit:** `dash3d/World3D#3`  **Java:** `World3D.java:1896-1899`  **Go:** `pkg/jagex2/dash3d/world3d/world3d.go:1940-1943`

Java computes var9/var10/var11 as 32-bit cross products of coordinate differences, then tests `var9 * var11 > 0 && var10 * var11 > 0`. Each cross product fits in int32 (~10^7 max), but their pairwise PRODUCT (~10^14) overflows 31 bits, so in Java it wraps mod 2^32 and only the wrapped sign is used. Go's `var9*var11` / `var11*var10` are 64-bit and do not wrap, so for projected triangles large enough that the product exceeds 2^31 the sign — and thus the inside/outside result — can differ from Java. Effect is limited to tile mouse-pick (clickTileX/Z reporting via DrawTileUnderlay/DrawTileOverlay); it does not alter rendering. Reachability is low because per-tile triangles are small on screen; the divergence only triggers for degenerate/huge projected triangles. No int32() wraps exist anywhere in world3d.go, so this is the canonical int64-vs-int32 latent class noted in prior audits.

```
Java: `return var9 * var11 > 0 && var10 * var11 > 0;` (32-bit wrap). Go: `return var9*var11 > 0 && var11*var10 > 0` (64-bit, no wrap). var9=(arg1-arg2)*(arg6-arg5)-(arg0-arg5)*(arg3-arg2) etc. — each ~10^7, product ~10^14 > 2^31.
```

> **Verifier (confirmed, severity latent):** Confirmed and could not refute. Java World3D.pointInsideTriangle (01f16088 World3D.java:1886-1900) declares all eight params `int` (32-bit; mouseX/mouseY are `static int`), and returns `var9 * var11 > 0 && var10 * var11 > 0` where var9/var10/var11 are cross products of coordinate differences. That product arithmetic is 32-bit and wraps mod 2^32 — only the wrapped sign is observed. Go PointInsideTriangle (world3d.go:1921-1944) declares the same params as `int`, which I verified is 64-bit on this amd64 platform (bits.UintSize==64), so `var9*var11` / `var11*var10` are computed in 64-bit and do not wrap. Ruled out false-positive causes: (1) no int32() wrap or upstream clamp exists anywhere in wo […]
>
> **Refined:** PointInsideTriangle (world3d.go:1943) computes `var9*var11` and `var11*var10` in Go's 64-bit `int`, whereas Java's pointInsideTriangle does the same products in 32-bit `int`, wrapping mod 2^32 and using only the wrapped sign. var9/var10/var11 are cross products of differences of screen-projected vertex coordinates (and MouseX/MouseY); for triangles whose vertices project to large off-screen coordinates (var24 near the 50 near-plane clamp, no value clamp before projection) the cross products and  […]

### L45. GouraudTriangle gradient/accumulator <<16 / <<15 math uses native int (64-bit) with no int32 wrap

- **Unit:** `graphics/Pix3D#1`  **Java:** `Pix3D.java:369-383, 394-396`  **Go:** `pkg/jagex2/graphics/pix3d/pix3d.go:378-391, 398-410`

Java computes the X gradients as `(arg4-arg3) << 16 / (arg1-arg0)` and color gradients as `<< 15`, plus per-edge accumulators `argN <<= 0x10` / `<<= 0xF` then `-= varK * argM`, all in 32-bit wrapping int. Go performs the identical expressions in 64-bit int with no int32() wrap. For normal on/near-screen coordinates and color values these never exceed 2^31 so results are bit-identical; only extreme degenerate off-screen coordinates (>|2^31| after the <<16) would diverge (Java wraps, Go does not — and Java's wrapped output is itself garbage). This is the rasterizer-family latent overflow class. Confirmed the whole pix3d.go file has zero int32() wraps, i.e. this is an accepted project-wide convention rather than a per-function omission. Grouping (Java additive `-` binds tighter than `<<`) is correctly parenthesized in Go as `((arg4-arg3)<<16)/(arg1-arg0)`.

```
Go: `var9 = ((arg4 - arg3) << 16) / (arg1 - arg0)` ; `arg3 <<= 0x10` ; `arg5 -= var13 * arg0` — vs Java int32 `var9 = (arg4 - arg3 << 16) / (arg1 - arg0)` etc.
```

> **Verifier (confirmed, severity latent):** Verified both sites in full context. Java Pix3D.gouraudTriangle (01f16088, lines 366-410+) computes X gradients as `(arg4 - arg3 << 16) / (arg1 - arg0)` and color gradients as `(arg7 - arg6 << 15) / (arg1 - arg0)` in 32-bit wrapping int, plus per-edge accumulators `arg3 <<= 16` / `arg6 <<= 15` and `-= argN * varK`. Go pix3d.go:374-410 declares `func GouraudTriangle(arg0..arg8 int)` and locals `var9..var14 := 0`, all plain `int` (64-bit on amd64), with no int32() wrap anywhere. grep -c "int32(" pix3d.go = 0, confirming the claim that the entire file omits int32 wraps (a project-wide rasterizer-family convention, not a per-function slip).

Grouping is correct: Java `<<` binds looser than `-`,  […]

### L46. textureTriangle UV-gradient accumulator products are 64-bit in Go but 32-bit-wrapping in Java

- **Unit:** `graphics/Pix3D#2`  **Java:** `Pix3D.java:1450-1458,1502-1504`  **Go:** `pkg/jagex2/graphics/pix3d/pix3d.go:1450-1458,1502-1504`

The texture barycentric-gradient terms are computed as e.g. Java `var26 = arg12 * var23 - arg9 * var24 << 14` (= Go `var20 := (var37*arg12 - var39*arg9) << 14`) and similarly `<< 8` / `<< 5` for var27..var34. These, plus the per-row seed `var20 += var22 * var35` (Java `var48 = var28*var47 + var26`, var35=arg0-centerY), can exceed 31 bits for large/degenerate triangle and texture coordinates. Java `int` truncates each multiply/shift/add to 32 bits (wraps, and the wrapped sign bit then participates in the later `>> 8`/`>> 16` arithmetic shifts in TextureRaster), whereas Go `int` is 64-bit and never wraps. For valid in-range geometry the values fit in 32 bits and results are identical; divergence only occurs at edge magnitudes. No `int32()` wrap exists anywhere in pix3d.go, so this is a deliberate, file-wide latent class rather than a one-off omission.

```
Go 1450: `var20 := (var37*arg12 - var39*arg9) << 14`  vs Java 1450: `int var26 = arg12 * var23 - arg9 * var24 << 14;` — Go has no int32() wrap, so the <<14 result and downstream `var20 += var22*var35` do not wrap to 32 bits as Java does.
```

> **Verifier (confirmed, severity latent):** Verified against Java-244 (git 01f16088) and Go pix3d.go. The variable-name mapping is a cross-lineage scramble but the algebra is identical: Go line 1450 `var20 := (var37*arg12 - var39*arg9) << 14` == Java `int var26 = arg12 * var23 - arg9 * var24 << 14` (Java var23=Go var37=arg11-arg9, var24=Go var39=arg14-arg12). Operator precedence matches — Java binds `*`/`-` tighter than `<<`, and the Go explicit parens reproduce that. The per-row seed Go line 1502 `var20 += var22*var35` (var35 = arg0-CenterH3D) == Java `var48 = var28*var47 + var26` (var47 = arg0-centerY). 

Decisive points:
1) Java computes these gradient terms and accumulators as 32-bit `int` (wraps on overflow); Go computes them as  […]

### L47. Java logical shift >>> ported as Go arithmetic shift >> in textureRaster texel reads (safe for masked texels)

- **Unit:** `graphics/Pix3D#3`  **Java:** `Pix3D.java:2120, 2249, 2294, 2347`  **Go:** `pkg/jagex2/graphics/pix3d/pix3d.go:2126,2185,2300,2360`

Java reads texels with a LOGICAL right shift: arg1[...] >>> var92. The Go ports this as arg1[...] >> var23 on a []int (signed) slice, which is an ARITHMETIC shift. For these two operations to diverge the texel value would need bit 31 set (negative). In GetTexels every texel is masked with & 0xF8F8FF (pix3d.go:247/252-254/270/275-277), so all texel values are <= 0xF8F8FF < 2^31 and the sign bit is never set. Arithmetic and logical shift therefore produce identical results for all reachable values. This is latent (only an issue if texels could ever be negative, which the mask prevents) and not currently a bug. No int32() wrap is needed because the operand is bounded.

```
Java: `arg0[var19++] = arg1[(var91 >> 7) + (arg3 & 0x3F80)] >>> var92;`  Go: `arg0[arg4] = arg1[(arg3&0x3F80)+(arg2>>7)] >> var23`  texel mask: `var1[i] &= 0xF8F8FF` (pix3d.go:270)
```

> **Verifier (confirmed, severity latent):** Verified against both sources. Java textureRaster reads texels with a LOGICAL right shift `arg1[...] >>> var92` at the cited sites (Pix3D.java:2120, 2249, 2294, 2347 plus their unrolled siblings — all confirmed via grep to use `>>>`). Go ports these as `arg1[...] >> var23` (pix3d.go:2126, 2185, 2300, 2360 and siblings), and arg1 is declared `[]int` at the TextureRaster signature (pix3d.go:2037), so Go's `>>` is an ARITHMETIC shift. The texel-read `>>>` is distinct from the shift-count source `var92 = var18 >> 23` (Java plain `>>`), which is the subject of a separate already-handled finding (Go's `& 0x1F` mask); I confirmed the two are different sites.\n\nThe divergence between arithmetic and […]

### L48. FirstFragmentID can overflow Java int32; Go 64-bit int does not wrap

- **Unit:** `wordenc/WordFilter#1`  **Java:** `WordFilter.java:1222-1232`  **Go:** `pkg/jagex2/wordenc/wordfilter/wordfilter.go:1212-1232`

value = value*38 + ... accumulates over up to 6 chars (returns 0 for len>6). Worst case (six digit-9 chars) reaches 3,010,936,383 which overflows Java int (wraps to -1284030913) but stays positive in Go's 64-bit int, so `i` differs and the IsBadFragment binary search diverges. In the live path the only caller builds a length-3 frag (Go line 715 / Java line 51 `char[3]`), max value 54871, so this is unreachable today — latent. A faithful port would wrap with int32(). (Declaration at Java line 1212 is past the 1-650 range; primary owner WordFilter#2.)

```
Java: value = value * 38 + c - 'a' + 1; (and +27 / +28 variants) with `int value` 32-bit wrapping.  Go: value = value*38 + int(c) - 'a' + 1  using 64-bit int — no wrap.
```

> **Verifier (confirmed, severity latent):** Verified both sites in full context. Java WordFilter.firstFragmentId (git 01f16088, lines 1221-1233) accumulates `int value = value*38 + ...` over up to 6 chars in a 32-bit Java int; the Go port (pkg/jagex2/wordenc/wordfilter/wordfilter.go:1212-1232) uses `value := 0` which infers 64-bit int on the target platforms, so it does not wrap.

Numeric claims confirmed exactly via computation: max per-char contribution is 37 (digit '9' => c-'0'+28), so a length-6 all-'9' fragment yields 37*(38^5+...+1) = 3,010,936,383, which exceeds int32 max (2,147,483,647). Java wraps this to -1,284,030,913; Go keeps +3,010,936,383. The len-3 max is 54871, matching the finding.

Divergence mechanism is sound: the […]
>
> **Refined:** FirstFragmentID overflow divergence is genuine but latent. Java's `int value` (32-bit) wraps; Go's inferred 64-bit `int` does not. For a length-6 all-'9' fragment, Java yields -1,284,030,913 while Go yields +3,010,936,383. Since Fragments[] entries are all in [0,65535] (read via g2/G2), the sign/magnitude difference makes the IsBadFragment binary search walk opposite directions and potentially return different verdicts. However, the sole caller (wordfilter.go:729 / WordFilter.java:724) builds a  […]

### L49. getEmulatedSize case 'g' drops the b=='q' -> 1 alternative present in 244

- **Unit:** `wordenc/WordFilter#2`  **Java:** `WordFilter.java:885-890`  **Go:** `pkg/jagex2/wordenc/wordfilter/wordfilter.go:893-898`

Java-244 for a=='g' returns 1 when b is '9','6', OR 'q'. The Go omits 'q' (b != '9' && b != '6' only). Go matches rev-225 (which lacked 'q') but not the 244 reference, so 'q'-for-'g' substitution is not emulated and can bypass the filter.

```
Java: if (b != '9' && b != '6' && b != 'q') { return 0; } return 1;  |  Go: if b != '9' && b != '6' { return 0 } return 1
```

> **Verifier (confirmed, severity latent):** Confirmed real. Java-244 WordFilter.getEmulatedSize (decl line 833, a=='g' branch at lines 899-905) reads: `if (b != '9' && b != '6' && b != 'q') { return 0; } return 1;`. The Go GetEmulatedSize (wordfilter.go:893-898) reads: `if b != '9' && b != '6' { return 0 } return 1` — the `&& b != 'q'` conjunct is dropped. Consequently for a=='g', b=='q' the 244 reference returns 1 (treats 'q' as an emulated 'g') while Go returns 0.

Ruled out the false-positive classes:
- Deob parameter scramble: Java decl is getEmulatedSize(char a, char c, char b); Go decl is GetEmulatedSize(c, a, b rune). I traced the call sites: Java calls getEmulatedSize(fragment[fragOff], c, b); Go calls GetEmulatedSize(c, fragm […]
>
> **Refined:** Java-244 WordFilter.getEmulatedSize, branch a=='g' (WordFilter.java:899-905), returns 1 when b is '9','6', OR 'q': `if (b != '9' && b != '6' && b != 'q') return 0; return 1;`. The Go port (pkg/jagex2/wordenc/wordfilter/wordfilter.go:893-898) drops the `&& b != 'q'` conjunct, so b=='q' falls through to return 0. Go matches the older rev-225 source (which lacked 'q') but not the 244 reference; the g->q substitution is no longer emulated, slightly weakening chat-filter evasion detection. Add `&& b  […]

### L50. firstFragmentId base-38 value not int32-wrapped (length-6 fragments only); non-observable in lookup

- **Unit:** `wordenc/WordFilter#2`  **Java:** `WordFilter.java:1227-1239`  **Go:** `pkg/jagex2/wordenc/wordfilter/wordfilter.go:1217-1231`

value = value*38 + ... over up to 6 chars maxes at 38^6 ~ 3.01e9, exceeding int32 max, so Java wraps negative for some length-6 inputs while Go does not. The Fragments table is read via G2() (values 0..65535), so any value > 65535 (i.e. all length>=4 fragments) cannot match a table entry on either side; Java's wrapped-negative value also can't match a non-negative table entry. The binary-search branch direction differs but, since i never equals a table entry in this range, both implementations terminate at the same 'not bad' result. Recorded as latent for completeness; no observable divergence given the G2 table.

```
Java int value (32-bit): value = value * 38 + c - 'a' + 1;  |  Go: value = value*38 + int(c) - 'a' + 1  | Fragments read as G2(): wordfilter.go:74 Fragments[i] = buf.G2()
```

> **Verifier (confirmed, severity latent):** CONFIRMED, could not refute. Decisive evidence:

1. Type widths verified: Java firstFragmentId (WordFilter.java:1222-1239) accumulates `value` in a 32-bit `int`. Go FirstFragmentID (wordfilter.go:1212-1232) returns Go `int` (64-bit on amd64) with no int32() wrap anywhere. For length-6 inputs the base-38 sum reaches up to 3,010,936,383 (computed exactly), which exceeds int32 max (2,147,483,647), so Java wraps to a negative value while Go stays positive. The width divergence is real.

2. Non-observability confirmed, and even stronger than the finding states. The Fragments table is loaded via G2() (packet.go:219-222: `((b<<8)+b)&0xFF masked`, range 0..65535; Java g2() identical), so every entry […]
>
> **Refined:** Java firstFragmentId computes the base-38 fragment id in a 32-bit `int`; for hypothetical length-6 inputs the accumulation maxes at 38^6-1 = 3,010,936,383, exceeding int32 max, so Java wraps negative while Go (return type `int`, 64-bit, no int32 wrap) stays positive. This is non-observable for two independent reasons: (1) the Fragments table is read via G2() (values 0..65535) and the binary search returns true only on an exact match, so any value >65535 (Go) or wrapped-negative (Java) misses eve […]

---

## Methods missing in Go (9)

- **hit(int type, int amount)** (`EntityA`, `ClientEntity.java:245-254`) — 4-slot damage system; absent in Go (rev-225 scalar model retained). See bug finding for detail.
- **clearRoute()** (`EntityA`, `ClientEntity.java:236-239`) — Resets routeLength + preanimRouteLength; no Go method. preanimRouteLength field itself also missing.
- **Pix2D.fillRectTrans(int y, int alpha, int height, int width, int colour, int x)** (`Pix2D+Pix8`, `Pix2D.java:88-129`) — Alpha-blended filled rectangle. Entirely absent from pkg/jagex2/graphics/pix2d/pix2d.go and the whole Go tree (grep for FillRectTrans returns nothing). USED by Java Client.java:10659 for type-3 interface children when child.alpha != 0. The Go DrawInterface (client.go:3832-3836) only ever calls the opaque FillRect/DrawRect path and ignores Component.Alpha, so the alpha branch is silently dropped.
- **Pix2D.drawRectTrans(int height, int colour, int x, int y, int width, int alpha)** (`Pix2D+Pix8`, `Pix2D.java:158-167`) — Alpha-blended rectangle outline (hlineTrans top/bottom + vlineTrans sides when height>=3). Absent from Go. USED by Java Client.java:10661 for type-3 non-fill interface children with alpha != 0.
- **Pix2D.hlineTrans(int y, int width, int colour, int x, int alpha)** (`Pix2D+Pix8`, `Pix2D.java:222-256`) — Alpha-blended horizontal line. Absent from Go. USED by Java Client.java:6565 (friend-server name flash in draw3DEntityElements, field1264 block) which is itself unported in the Go client.
- **Pix2D.vlineTrans(int x, int y, int alpha, int height, int colour)** (`Pix2D+Pix8`, `Pix2D.java:282-309`) — Alpha-blended vertical line. Absent from Go. Called by Java drawRectTrans for the rectangle sides.
- **bankArrangeMode (field, client.Nh)** (`client/Client#1`, `Client.java:426`) — Live field; drives INV_BUTTOND mode byte. See blocker finding.
- **warnMembersInNonMembers (field, client.rf)** (`client/Client#1`, `Client.java:881`) — Live field read from opcode-44 g1; drives non-members welcome warnings. See bug finding.
- **field1264 (field, client.lc)** (`client/Client#1`, `Client.java:563`) — Live field; opcode-192 screen-flash overlay. See latent finding.

---

## Refuted findings (4)

Kept for the record — verifier refuted with evidence:

- **handlePrivateChatInput omits @cr1@/@cr2@ sender-prefix stripping before isFriend() and menu text** (`client/Client#4`, Java `Client.java:3737-3757` / Go `pkg/jagex2/client/client.go:5228-5241`): REFUTED as a live bug; it is part of the documented deferred chat-crowns deviation.

What is TRUE: The Go HandlePrivateChatInput (pkg/jagex2/client/client.go:5228-5241) genuinely lacks the `sender = sender.substring(5)` strip that Java Client.java:3737-3757 performs, and it passes c.MessageSender[i] raw into IsFriend and all three menu strings. AddMessage (client.go:7985-8001) stores arg3 verbatim […]
- **useMenuOption action 930 spellCaption substring is byte-based vs Java UTF-16 codeunit-based** (`client/Client#10`, Java `Client.java:10166-10176` / Go `pkg/jagex2/client/client.go:4937-4944`): Both code sites match the finding's quotes. Go (pkg/jagex2/client/client.go:4937-4944) splits ActionVerb on " " via strings.Index/Contains (byte indices) + byte-slice substrings; Java (Client.java:10166-10176) splits targetVerb via indexOf/substring (UTF-16 code-unit indices). The strings come from gjstr (Java: new String(byte[]) Latin-1 -> UTF-16; Go: latin1ToUTF8 -> UTF-8), with no ASCII-only in […]
- **Shift-count mask & 0x1F on (arg7>>23) correctly reproduces Java implicit shift-count masking — verified, not a defect** (`graphics/Pix3D#3`, Java `Pix3D.java:2117,2161,2244,2291,2335` / Go `pkg/jagex2/graphics/pix3d/pix3d.go:2122,2174,2296,2348,2425`): I re-read both full TextureRaster bodies (Go pkg/jagex2/graphics/pix3d/pix3d.go:2037-2436; Java textureRaster via git show 01f16088:src/main/java/jagex2/graphics/Pix3D.java:2051-2260+) and mapped the branches: Go LowDetail == Java lowMem. In Java the accumulator is var18 (=arg7<<9, += var17), shift count var92/var34 = var18>>23 used in `arg1[...] >>> varN`; Go reuses arg7 (=arg7<<9 in place, += va […]
- **filterFragments digit-run value accumulation can diverge under Java int32 overflow** (`wordenc/WordFilter#2`, Java `WordFilter.java:1102-1110` / Go `pkg/jagex2/wordenc/wordfilter/wordfilter.go:1106-1115`): REFUTED. The finding claims that over a >=10-digit run, Java int32 `value` could wrap to <=255 (taking count++) while Go int64 stays large (taking count=0), diverging the count. This overlooks the second conjunct of the guard.

The guard is `value <= 255 && end - index <= 8` (Java) / `value <= 0xFF && end-index <= 8` (Go) — identical in both. The two conditions are ANDed.

Overflow reachability: t […]

---

## Cosmetic (64)

| Unit | Title | Java | Go | Note |
|---|---|---|---|---|
| `BZip2` | Stale/inaccurate doc comment on Decompress outer loop (wrong structure description and wrong line refs) | `BZip2.java:206-207,563-564` | `pkg/jagex2/io/bzip2/bzip2.go:214-218` | The comment claims Java's decompress has an `outer while(true)` wrapping an `inner while(var27)` and cites `BZip2.java:180-468`. The actual Java 244 source has a SINGLE `while (reading)` loop (lines 207-563) with no outer while(tr… |
| `BZip2` | Entry-point doc comment cites Java method name 'read' but the 244 method is 'decompress' | `BZip2.java:11-12` | `pkg/jagex2/io/bzip2/bzip2.go:16,20-21` | The Go entry function is named `Read` and its doc comment says `Java: BZip2.read (deob/BZip2.java)`. In the 244 deob the public entry is `decompress(byte[] decompressed, int length, byte[] stream, int avail_in, int next_in)`, not … |
| `ClientPlayer` | Hash-shift comment shows masked `<< 8` as if it were the Java source (actually `<< 40`) | `ClientPlayer.java:273,278` | `pkg/jagex2/dash3d/entity/playerentity/clientplayer.go:217-220,224` | The Go code correctly reproduces Java's int-shift masking: Java line 273 is `hash += leftHandValue - this.appearance[5] << 40` and line 278 is `hash += rightHandValue - this.appearance[3] << 48`. Because the shift operand is a 32-… |
| `ClientStream` | NewClientStream doc-comment reverses the Java constructor parameter order | `ClientStream.java:44` | `pkg/jagex2/io/clientstream/clientstream.go:91-92` | The Go doc comment claims it mirrors `ClientStream(GameShell, Socket)`, but the actual Java constructor signature is `ClientStream(Socket socket, GameShell shell)` — socket first, shell second. The order in the comment is reversed… |
| `ClientStream` | NewClientStream doc-comment cites wrong startThread priority constant (2 vs 3) | `ClientStream.java:136` | `pkg/jagex2/io/clientstream/clientstream.go:93` | The Go comment says the Java side spawned the writer thread via `shell.startThread(this, 2)`, but the only startThread call in ClientStream is `this.shell.startThread(this, 3)` (priority 3, from write(), not the constructor). The … |
| `Component` | Stale/incorrect Java line refs and 225-lineage field names in unused-field (field98/field99) comments | `Component.java:146,182,278,279` | `pkg/jagex2/config/component/component.go:74-77,173-175` | Two comments document the Type==1 dead-write fields. The struct comment (lines 74-77) cites `Component.java:130,160 declares unusedShort1/unusedBoolean1`; the decode comment (lines 173-175) cites `Component.java:264-265 ... unused… |
| `Component` | GetImage try/catch comment cites wrong Java line range | `Component.java:535-541` | `pkg/jagex2/config/component/component.go:393` | The defer/recover comment cites `Component.java:433-439` for the try/catch returning null on Pix32 construction failure. The actual try/catch is at Component.java:535-541 (`try { image = new Pix32(...); imageCache.put(...); return… |
| `ConfigSmall` | SeqType comment falsely claims AnimFrame is loaded before decode | `Client.java:1891,2443 (SeqType.unpack precedes AnimFrame.unpack in async handler)` | `pkg/jagex2/config/seqtype/seqtype.go:118-125,37-44` | The Decode comment states 'current load order has AnimFrame ready at decode' and GetFrameDuration's comment says decode 'already resolves that fallback eagerly into Delay[], so the resolved value is identical'. Both are false: Ani… |
| `ConfigSmall` | VarpType.Unpack omits Java's post-loop 'varptype load mismatch' packet-alignment diagnostic | `VarpType.java:75-77` | `pkg/jagex2/config/varptype/varptype.go:31-43` | Java VarpType.unpack ends with `if (dat.data.length != dat.pos) { System.out.println("varptype load mismatch"); }`, a diagnostic that fires if the decode loop did not consume the full buffer. The Go Unpack has no equivalent. Purel… |
| `Dash3dSmall` | AnimFrame.get() and AnimFrame.unload() static methods not ported as functions; callers inline direct Instances[] access and the get() null-guard is dropped at the (out-of-unit) call sites | `AnimFrame.java:147-159` | `pkg/jagex2/dash3d/animframe/animframe.go:8-27 (no Get/Unload), pkg/jagex2/client/client.go:7297, pkg/jagex2/dash3d/model/model.go:1062,1081-1082, pkg/jagex2/config/seqtype/seqtype.go:124` | Java AnimFrame.get(int id) returns null when instances==null else instances[id]; AnimFrame.unload() sets instances=null. The Go port has neither function. unload() is faithfully inlined as `animframe.Instances = nil` (client.go:72… |
| `Datastruct` | ToAsterisks doc comment describes a byte-based pitfall the code does not actually take | `JString.java:117-126 (censor: temp = temp + "*" per i < s.length() UTF-16 units)` | `pkg/jagex2/datastruct/jstring/jstring.go:129-141` | The doc comment warns that Go's `len(s)` is byte-based and would over-count stars for non-ASCII like '£'. But the implementation uses `for range s`, which iterates RUNES (Unicode code points), not bytes — so the comment's stated p… |
| `EntityB` | ClientProj/MapSpotAnim index Seq.Delay[] directly instead of calling GetFrameDuration like Java (and ClientLocAnim) | `ClientProj.java:135-136; MapSpotAnim.java:52-55` | `pkg/jagex2/dash3d/entity/clientproj.go:89-90; pkg/jagex2/dash3d/entity/mapspotanim.go:38-41` | Java ClientProj.update and MapSpotAnim.update call `this.type.seq.getFrameDuration(frame)`, whose body lazily resolves a zero delay from the frame's AnimFrame and applies a `==0 -> 1` fallback. The Go ports index `e.SpotAnim.Seq.D… |
| `EntityB` | ClientProj.Update adds the two displacement terms in swapped order vs Java | `ClientProj.java:127` | `pkg/jagex2/dash3d/entity/clientproj.go:81` | Java: `field518 += field523*0.5*arg1*arg1 + arg1*field522` (acceleration term first, velocity term second). Go: `e.Y += e.VelocityY*float64(arg1) + e.AccelerationY*0.5*float64(arg1)*float64(arg1)` (velocity term first, acceleratio… |
| `GameShell` | handleMouseMove comment claims 'Java does the (y,x) swap' — Java does NOT swap; the swap compensates the Go InputTracking's scrambled params | `GameShell.java:405-407,423-425` | `pkg/jagex2/client/gameshell.go:182-194` | The comment says Java's mouseDragged/mouseMoved call `InputTracking.mouseMoved(y, x)` with a swap and that the Go port 'preserves that swap.' Verified Java actually calls `InputTracking.mouseMoved(x, y)` (x=horizontal, y=vertical)… |
| `Ground` | Stale Java line-ref in always-false-bug comment (cites :274, actual is :289) | `Ground.java:289-290` | `pkg/jagex2/dash3d/typ/ground.go:226-232` | The Go comment documenting the preserved always-false `if (arg3 > arg3)` branch cites `Ground.java:274`. Line 274 in the Java source is an unrelated `} else {` token. The actual `if (arg19 > arg19) { var37 = arg19; }` lives at Gro… |
| `Ground` | Stale Java line-refs on dead /14 scaling comments (cite :289/:290, actual is :304/:305) | `Ground.java:304-305` | `pkg/jagex2/dash3d/typ/ground.go:237-238` | The two trailing dead-scaling lines `var36 /= 14` and `var37 /= 14` carry comments `Java: Ground.java:289` and `Java: Ground.java:290`. Those Java lines are actually the always-false bug (`if (arg19 > arg19)` and `var37 = arg19;`)… |
| `InputTracking` | False 'mouseMoved(y, x) swap' comments — Java has no swap; the Go double-swap is a correctness trap for maintainers | `GameShell.java:406,424 + InputTracking.java:136-176` | `pkg/jagex2/client/gameshell.go:182-193 + pkg/jagex2/client/inputtracking/inputtracking.go:129-167` | Java GameShell.mouseDragged and GameShell.mouseMoved both call InputTracking.mouseMoved(x, y) — first arg = x (horizontal), second = y (vertical), NO swap. The Go gameshell comment (gameshell.go:184-186 and inline at :192) claims … |
| `InputTracking` | Stale Gio/threading rationale in InputTracking mutex doc-comment | `InputTracking.java:34-46 (synchronized methods)` | `pkg/jagex2/client/inputtracking/inputtracking.go:24-31` | The mu doc-comment justifies the mutex by claiming 'The Go port runs Gio's app.Main() goroutine as the producer and Client.Run() as the consumer, so the same race exists.' Gio was removed from the repo (now GLFW/WebGL via the plat… |
| `LocType` | CheckModel/CheckModelAll doc-comments cite slightly off-by-one Java line ranges | `LocType.java:335-354` | `pkg/jagex2/config/loctype/loctype.go:378-421` | The Go CheckModel comment says "lines 336-354" (Java method is 335-354, 336 is the body open) and CheckModelAll says "lines 357-371" (Java method spans 356-371, 356 is the @ObfuscatedName/signature). The cited bodies are correct; … |
| `NpcType` | Go field comment block references wrong Java line numbers for resize opcodes | `NpcType.java:72-79,210-215` | `pkg/jagex2/config/npctype/npctype.go:36-40,164-166` | Two Go comments cite stale Java line refs from the 225 lineage. Comment at npctype.go:36-37 says 'NpcType.java:73-79 declares resizex/resizey/resizez' — in the 244 source those are field1008/field1009/field1010 at lines 72-79 (not… |
| `NpcType` | GStrByte doc-comment cites wrong Java method name (gstrbyte) for opcode-3 desc reader | `src/main/java/jagex2/io/Packet.java:258-267` | `pkg/jagex2/io/packet.go:271-280` | NpcType opcode 3 correctly calls arg1.GStrByte() to read the raw desc bytes, matching Java `this.desc = buf.gjstrraw()`. However the GStrByte doc-comment in packet.go labels the Java method as 'gstrbyte (Packet.java:243-252)' wher… |
| `ObjType` | Decode opcode 30-34 uses "" instead of Java null for 'hidden' op slots | `ObjType.java:317-319` | `pkg/jagex2/config/objtype/objtype.go:225-229` | Java sets `op[code-30] = null` when the op string equalsIgnoreCase('hidden'); Go sets `t.Op[var3-30] = ""`. This is the codebase-wide null-vs-empty-string convention for []string op arrays (documented in the Go comment referencing… |
| `OnDemand` | protocol.go comment uses stale rev#225 name 'CLIENTPROT_SCRAMBLED' / 'rev#225' for the 244 dead table 'CLIENTPROT_LOOKUP' | `Protocol.java:8-9 (CLIENTPROT_LOOKUP, @ObfuscatedName ic.a)` | `pkg/jagex2/io/protocol.go:3-7` | The Go comment documenting the intentionally-not-ported 256-entry client-opcode table calls it 'Protocol.CLIENTPROT_SCRAMBLED (Protocol.java:8-9)' and says it has 'zero references anywhere in the rev#225 Java client'. In the 244 l… |
| `Packet+Isaac+Jagfile` | CRCTable is a correctly-documented dead field | `Packet.java:21-38` | `pkg/jagex2/io/packet.go:17-50` | Java's crctable static [256] int is computed in the static initializer but never read anywhere in the client (real CRC uses java.util.zip.CRC32). The Go init() reproduces the same CRC32-reflected table. The Go comment accurately s… |
| `Pix2D+Pix8` | Pix8.Plot has a synthetic 10th parameter (arg5) and inert early-return guard with no Java source | `Pix8.java:218-264 (9-param plot, no unused int)` | `pkg/jagex2/graphics/pix8/pix8.go:210-215` | Java Pix8.plot takes exactly 9 parameters, none of which is an unused integer flag. The Go Pix8.Plot adds a 10th parameter 'arg5 int' (between rows arg4 and cols arg6) plus a guard 'if arg5 != 0 { return }'. PlotSprite is the sole… |
| `Pix2D+Pix8` | Pix8 struct omits Pix2D / DoublyLinkable inheritance (verified unused) | `Pix8.java:11 'public class Pix8 extends Pix2D'; Pix2D.java:7 'extends DoublyLinkable'` | `pkg/jagex2/graphics/pix8/pix8.go:8-19; pix2d.go:16-18 (commented-out struct)` | Java Pix8 extends Pix2D which extends DoublyLinkable (LRU-cache linkage). The Go Pix8 is a standalone struct and pix2d.go leaves the Pix2D struct commented out. Verified via git grep on 01f16088 that no code ever uses Pix8/Pix2D a… |
| `Pix2D+Pix8` | Pix2D.cls / Pix8 ctor length use Width*Height vs Java Height*Width (commutative, no effect) | `Pix2D.java:120 'int length = height2d * width2d;'; Pix8.java:71 'int len = this.hi * this.wi;'` | `pkg/jagex2/graphics/pix2d/pix2d.go:68 'length := Width2D * Height2D'; pkg/jagex2/graphics/pix8/pix8.go:49 'length := p.Wi * p.Hi'` | Operand order of the dimension product is swapped relative to Java in Pix2D.Clear and the Pix8 constructor's pixel length. Multiplication is commutative and these dimensions never overflow int32 at any real resolution, so results … |
| `Pix32` | Crop/Scale Java line-ref comments cite 225 lineage lines that don't exist in 244 | `Pix32.java:302-353,357-378 (225-clean lines, NOT present in 244)` | `pkg/jagex2/graphics/pix32/pix32.go:349-351,406-408` | The Crop and Scale doc-comments cite `Pix32.java:302-353` and `Pix32.java:357-378` as if these were 244 references, but per the project ref-case convention 244 is `Client.java`-style and these line ranges only exist in the 225-cle… |
| `Pix32` | DrawRotatedMasked comment cites Pix32.java:442-468 but 244 method is at 377-410 | `Pix32.java:377-410 (244 drawRotatedMasked)` | `pkg/jagex2/graphics/pix32/pix32.go:500-502; test pix32_test.go:11` | The DrawRotatedMasked doc-comment and the TestDrawRotatedMaskedRecoversOnOutOfBounds comment both cite `Pix32.java:442-468` for the try/catch swallow, but the actual 244 drawRotatedMasked spans lines 377-410 (the try/catch is at 3… |
| `Pix32` | DrawAlpha is the 225-named clone of Java 244 transPlotSprite; behaviorally correct but Go has no method named TransPlotSprite | `Pix32.java:317-355 (244 transPlotSprite)` | `pkg/jagex2/graphics/pix32/pix32.go:433-477 (DrawAlpha)` | Java 244 added a public method `transPlotSprite(arg0=x, arg1=alpha, arg3=y)` (Pix32.java:318) that clips and calls transPlot. The Go port implements this as `DrawAlpha(arg0=alpha, x, y)` (225 lineage name). I traced every local an… |
| `PixFont` | DrawStringTooltip comment claims @str@ side-effect path matches, but Go EvaluateTag has no strikeout side-effect at all | `PixFont.java:204-233,272-275` | `pkg/jagex2/graphics/pixfont/pixfont.go:249-283,321-323` | The Go DrawStringTooltip doc-comment cites PixFont.java:181-206 (a 225-lineage line range) as if faithfully ported, but the method silently omits Java's evaluateTag("str") side-effect (this.strikeout=true). In drawStringAntiMacro … |
| `SignLink+Midi` | GetUID returns int(uint32+1) instead of int(int32+1) — sign-extension divergence for high-bit uid.dat values | `SignLink.java:244-252` | `pkg/jagex2/client/sign/signlink/storage_disk.go:117-118` | Java reads the uid via DataInputStream.readInt() into a signed `int uid` and returns `uid + 1` in 32-bit signed arithmetic; if the stored word has the high bit set, Java returns a NEGATIVE int. Go reads it as uint32 (binary.BigEnd… |
| `SignLink+Midi` | storeid clamp + dynamic '.file_store_' + storeid target hardcoded to .file_store_32 | `SignLink.java:19,206-210` | `pkg/jagex2/client/sign/signlink/storage_disk.go:63` | Java findcachedir clamps storeid into [32,34] (default 32) and builds target = ".file_store_" + storeid. Go hardcodes ".file_store_32" with a comment-free literal. Since storeid is never reassigned anywhere in the codebase (defaul… |
| `Sound` | Doc-comment markers on Go methods use 225-lineage names but bodies match 244 Java (name-only) | `Envelope.java:43,59,68; Tone.java:79,98,225,240; Wave.java:31,47,57,73,104,125` | `pkg/jagex2/sound/envelope/envelope.go:23,39,48; pkg/jagex2/sound/tone/tone.go:196,215; pkg/jagex2/sound/wave/wave.go (method names)` | Go methods are renamed relative to the 244 Java names (Envelope.unpack->Read, genInit->Reset, genNext->Evaluate; Tone.waveFunc->Generate2, unpack->Read; Wave.read->Read, trim->Trim, generate->Generate). Each carries a one-line mar… |
| `WordPack+PixMap` | Unpack drops Java's `(char)` 16-bit truncation on the uppercase adjustment; harmless because the guard constrains the range | `WordPack.java:47-50` | `pkg/jagex2/wordenc/wordpack/wordpack.go:49-52` | Java: `charBuffer[i] = (char)(charBuffer[i] + -32);` — the `(char)` cast truncates to 16 bits. Go: `CharBuffer[i] = CharBuffer[i] + -32` on a `[]rune` (int32) with no mask. This is safe ONLY because the surrounding guard `c >= 'a'… |
| `client/Client#1` | Dead/duplicate Go field MessageIDs (no Java counterpart) | `Client.java:596` | `pkg/jagex2/client/client.go:223,593` | Java has a single messageIds=new int[100] (Client.java:596, private-message dedup). The Go port declares TWO fields: MessageIds (line 357, allocated 555) — the correct counterpart used at client.go:9626/9652 — and MessageIDs (capi… |
| `client/Client#10` | LOGIC-DELTA-SCOPE.md falsely claims player/NPC extended-info bit-streams are UNCHANGED between 225 and 244 | `Client.java:9135-9302, 9451-9565` | `LOGIC-DELTA-SCOPE.md:93-95` | The scoping doc asserts 'Player/NPC info bit-streams UNCHANGED (same getPlayerLocal/OldVis/NewVis/Extended + getNpcPos decomposition; extended-info mask bits identical).' This is incorrect: 244 added DAMAGE_STACK mask bits (npc 0x… |
| `client/Client#2` | Startup banner prints 'release #225' instead of 244 | `Client.java:1281 (main): System.out.println("RS2 user client - release #" + 244)` | `cmd/client/main.go:47` | The Java 244 main() prints 'RS2 user client - release #244'. The Go main() prints `"RS2 user client - release #" + strconv.Itoa(225)`. On the rev-244 branch audited against the 244 reference this is a stale version literal; the ba… |
| `client/Client#2` | SetMidiVolume / SaveMidi carry overload-disambiguator dummy params folded into PacketSize | `Client.java:1447-1465 (saveMidi a(Z[BZ)V; setMidiVolume a(IZZ)V)` | `pkg/jagex2/client/client.go:3410-3416 (SetMidiVolume), 3366-3388 (SaveMidi)` | Java setMidiVolume's bytecode signature a(IZZ)V has a trailing dummy boolean (overload-disambiguator) absent from the deob source body (int volume, boolean active). The Go SetMidiVolume(arg0 int, arg1 int, arg2 bool) maps arg1=vol… |
| `client/Client#3` | Draw() omits drawCycle++ counter | `Client.java:2008` | `pkg/jagex2/client/client.go:2378-2389` | Java draw() increments `drawCycle++` after the error-screen early-return. The Go Draw() omits it; drawCycle exists nowhere in the Go port. drawCycle is consumed only by a debug `System.out.println("draw-cycle:"+drawCycle)` (Java:4… |
| `client/Client#4` | handleChatMouseInput trailing PacketSize += arg1 uses 0 from caller instead of Java's garbage mouseX value (harmless dead-write) | `Client.java:3782` | `pkg/jagex2/client/client.go:2019,6169` | The Go method ends with `c.PacketSize += arg1`, the deob garbage-parameter dead-write pattern (PacketSize is an anti-tamper accumulator, reset at 6843, never meaningfully read). In Java the unused/garbage param is mouseX, and the … |
| `client/Client#5` | UpdateOrbitCamera drops the Java SignLink.reporterror try/catch (registry-covered signed-applet artifact) | `Client.java:4585-4587 (updateOrbitCamera catch)` | `pkg/jagex2/client/client.go:6262-6324 (UpdateOrbitCamera)` | Java wraps updateOrbitCamera's body in try/catch that calls SignLink.reporterror("glfc_ex ...") then rethrows RuntimeException("eek"). The Go has no try/catch; a panic would propagate (matching the rethrow). reporterror is registr… |
| `client/Client#6` | updateSequences: spotanim block reordered after primary-seq block (225 ordering) | `Client.java:5211-5227` | `pkg/jagex2/client/client.go:4239-4255` | In Java-244 the spotanim-frame-advance block runs BEFORE the primarySeq blocks (positions: secondary, spotanim, preanim-trigger, primary-advance, delay--). The Go port (matching 225-clean ordering: secondary, primary-advance, dela… |
| `client/Client#6` | drawScene retains dead 225 `PacketSize += arg0` (always 0 from caller) | `Client.java:5839` | `pkg/jagex2/client/client.go:1454` | DrawScene keeps the 225 signature `DrawScene(arg0 int)` and executes `c.PacketSize += arg0`. The sole caller (DrawGame line 4304) passes 0, so this is a no-op. Java-244 drawScene takes no parameter and has no such accumulation. Ha… |
| `client/Client#6` | updateSequences indexes SeqType.Delay[] directly instead of getFrameDuration() — comment claims eager-resolved equivalence | `Client.java:5200,5219-5220,5241-5242` | `pkg/jagex2/client/client.go:4208,4220-4221,4247-4248` | Java-244 reads frame duration via seq.getFrameDuration(frame); Go indexes var3.Delay[frame] directly. The SeqType comment (pkg/jagex2/config/seqtype/seqtype.go:37-43) asserts the Go decode eagerly resolves getFrameDuration's fallb… |
| `client/Client#7` | GetTopLevelCutscene carries a disambiguator parameter consumed only by PacketSize accounting | `Client.java:6117-6121` | `pkg/jagex2/client/client.go:1438-1448` | GetTopLevelCutscene(arg0 int) adds `c.PacketSize += arg0` — the obfuscated `(I)` overload-disambiguator argument pattern, value-neutral to the computation (matches the deob-artifact convention). The actual logic (y - cameraY >= 80… |
| `client/Client#8` | Stale doc-comment line refs inside Read() cite a different revision's line numbers | `Client.java:7223-8470` | `pkg/jagex2/client/client.go:9496-9518` | The Read() header comment says `Java: read() (client.java:9316-10384)` and the recover blocks cite `client.java:10374-10382` and `client.java:10054-10067`. In the audited 244 reference (01f16088) readPacket() spans 7223-8470, the … |
| `client/Client#9` | getPlayerPos size/null mismatch aborts with panic(msg) instead of Java's RuntimeException("eek") | `Client.java:8950-8961` | `pkg/jagex2/client/client.go:7931-7941` | Java getPlayerPos throws new RuntimeException("eek") after reporterror on both the size-mismatch and null-entry checks. Go GetPlayer calls signlink.ReportErrorFunc(msg) then panic(msg) with the descriptive message rather than "eek… |
| `dash3d/Model#1` | NewModel1 comment cites a Java println('Error model:...not found!') that does not exist in rev-244 | `Model.java:389-394` | `pkg/jagex2/dash3d/model/model.go:351` | The comment `// Java: System.out.println("Error model:" + arg1 + " not found!") — colon, no spaces.` asserts a Java source line that does NOT exist in rev-244 Model.java (git grep for 'not found' in 01f16088 Model.java returns not… |
| `dash3d/Model#1` | NewModel3 doc-comment states the wrong Java signature and wrong line for the overload it ports | `Model.java:665` | `pkg/jagex2/dash3d/model/model.go:614-617` | The Go comment says NewModel3 `ports Java's 'Model(Model[] arg0, byte arg1, int arg2)' (Model.java:666). Java's arg1 is a deobfuscator-preserved overload disambiguator`. The actual rev-244 declaration at line 665 is `Model(boolean… |
| `dash3d/Model#1` | NewModel1 texture-axis comment 'NO * 6' is misleading vs Unpack which does multiply texturedFaceCount by 6 | `Model.java:339-340` | `pkg/jagex2/dash3d/model/model.go:500-502` | The comment at the axis-read site reads `// Java: rev-244 stores FaceTextureAxisOffset as a byte offset directly — NO `* 6` (225 stored a texture-face count and multiplied at use).` This is easy to misread as 'Unpack omits the *6'… |
| `dash3d/Model#2` | NewModel4 omits Java's loaded++ (dead-write counter, never read) | `Model.java:778` | `pkg/jagex2/dash3d/model/model.go:732-737` | Java constructor Model(Model, boolean, boolean, boolean) increments static `loaded` as its first statement. Go NewModel4 does not increment model.Loaded. Model.loaded is write-only in Java (incremented in 5 constructors at J390/J5… |
| `dash3d/Model#2` | NewModel5 omits Java's loaded++ (dead-write counter, never read) | `Model.java:832` | `pkg/jagex2/dash3d/model/model.go:788-793` | Java constructor Model(boolean, boolean, Model) increments static `loaded` as its first statement. Go NewModel5 does not. Same dead-write counter as the NewModel4 finding; no behavioral impact since loaded is never read. |
| `dash3d/Model#2` | Loaded doc-comment claims it counts only Model(int) decodes, but Java increments in all 5 constructors | `Model.java:390,565,666,778,832` | `pkg/jagex2/dash3d/model/model.go:48-49` | The Go comment reads `// Loaded counts every Model(int) decode. Java: Model.loaded.` In Java, loaded++ fires in five constructors (Model(int) at J390 plus the four merge/copy constructors), not just the int decode. Because the Go … |
| `dash3d/Model#3` | nolint:staticcheck QF1007 suppression on var23 declaration in Draw1 | `Model.java:1646-1649` | `pkg/jagex2/dash3d/model/model.go:1508-1511` | `var23 := false` then a conditional `if var11-var22 <= 50 { var23 = true }`, with a //nolint:staticcheck QF1007 comment, deliberately mirroring the Java `boolean var23 = false; if (...) var23 = true;` structure rather than collaps… |
| `dash3d/World#1` | Misleading inline comment '// Java: (byte) g1b' on loadGround overlay read | `World.java:167` | `pkg/jagex2/dash3d/world/world.go:194` | The Go comment claims Java applies a `(byte)` cast to the g1b() result, but Java has no cast there: `this.overlayType[...] = var7.g1b();` — g1b() already returns a byte and is assigned directly. The Go code itself is correct (G1B … |
| `dash3d/World3D#2` | Decor/Sprite cached-MinY doc-comments describe a faithful port but the implementation caches the inverted Go field | `ModelSource.java:18-22; Model.java:1002-1007` | `pkg/jagex2/dash3d/typ/decor.go:17-20; pkg/jagex2/dash3d/typ/sprite.go:25-29; pkg/jagex2/dash3d/world3d/world3d.go:1361-1363,1575-1577,327-329,418-420` | The comments (decor.go:17-19 'mirrors rev-244 ModelSource.minY ... rev-244 swapped the rev-225 resolved model.maxY arg for cached minY'; world3d.go:1575-1577 'culls on the source's cached minY ... vs rev-225's resolved model.maxY'… |
| `dash3d/World3D#2` | World3D.Click caller in Client uses -11/-8 viewport offsets where Java-244 uses -4/-4 (cross-reference, caller is outside this unit) | `Client.java:10190-10192` | `pkg/jagex2/client/client.go:4772,4774` | Java `this.scene.click(super.mouseClickY - 4, super.mouseClickX - 4)` subtracts 4 from both; the Go caller `c.Scene.Click(c.MouseClickY-11, c.MouseClickX-8)` subtracts 11/8. World3D.Click itself (go 1029-1035) faithfully maps Java… |
| `dash3d/World3D#3` | PointInsideTriangle doc-comment cites off-by-one / lineage-ambiguous Java line ref | `World3D.java:1890` | `pkg/jagex2/dash3d/world3d/world3d.go:1925-1930` | The doc-comment documenting the previously-fixed `arg1 > arg4` guard cites `Java: World3D.java:1889`. The actual 244 declaration of pointInsideTriangle is at line 1886 and the third-greater guard `arg1 > arg2 && arg1 > arg3 && arg… |
| `graphics/Pix3D#1` | Unload double-nulls DivTable and never nulls DivTable2 — faithful reproduction of a Java deob bug | `Pix3D.java:79-93` | `pkg/jagex2/graphics/pix3d/pix3d.go:81-95` | Java unload() contains `divTable = null;` twice (lines 80-81) and never nulls divTable2 — almost certainly the second line was meant to be divTable2. Go faithfully reproduces both the duplicate `DivTable = nil` (lines 82-83) and t… |
| `graphics/Pix3D#3` | textureRaster Go refactor folds Java var91/var92/var33/var34 into arg2/var23 and inverts branch ordering vs Java — verified semantically equivalent | `Pix3D.java:2051-2431` | `pkg/jagex2/graphics/pix3d/pix3d.go:2037-2436` | The Go TextureRaster is a heavy structural rewrite, not a line-for-line port: (1) Java's first/high-detail branch is `if (!lowMem) { ... >>14/0x600000/0x3F80/>>7 ... return; }` with the low-mem path as fallthrough using >>12/0xC00… |
| `wordenc/WordFilter#1` | Dead local `match2` (type==3 double-write) dropped in Go FilterTLD2 | `WordFilter.java:456-461` | `pkg/jagex2/wordenc/wordfilter/wordfilter.go:471` | Java filterTld computes `boolean match2` (true when type==3 && status0>2 && status1>0, else false) but never reads it — a deob dead-write. Go FilterTLD2 omits it. This is a faithful drop of a pure deob artifact (no behavioral effe… |
| `wordenc/WordFilter#1` | filter() timing locals (System.currentTimeMillis start/end) dropped silently | `WordFilter.java:152,179-180` | `pkg/jagex2/wordenc/wordfilter/wordfilter.go:168-204` | Java filter(String) records `long start = System.currentTimeMillis();` and `long end = ...;` but never uses them (dead profiling locals). Go Filter omits both. No behavioral impact; faithful drop of a dead-write, noted for complet… |
| `wordenc/WordFilter#2` | GetEmulatedSize / GetEmulatedDomainCharSize / ComboMatches reorder parameters vs Java signature (call sites compensate; verified safe) | `WordFilter.java:784,812,833` | `pkg/jagex2/wordenc/wordfilter/wordfilter.go:784,811,836` | The Go signatures reorder the scrambled Java deob params (Java getEmulatedSize(a,c,b) vs Go GetEmulatedSize(c,a,b); Java getEmulatedDomainCharSize(c,a,b) vs Go GetEmulatedDomainCharSize(c,b,a); Java comboMatches(combos,a,b) vs Go … |

---

## Deferred work (consolidated ledger)

### From audit findings

- **Teleport (move) omits the preanimRouteLength=0 reset; the field does not exist in Go** (`EntityA`, Java `ClientEntity.java:195-197` / Go `pkg/jagex2/dash3d/entity/clententity.go:104-105`) — Java `move` resets THREE fields on the teleport path: `routeLength=0; preanimRouteLength=0; seqDelayMove=0`. The Go `Teleport` resets only `PathLength=0` and `SeqTrigger=0`. SeqTrigger is the Go counterpart of `seqDelayMove` (verified by usage: client.go:4046 `SeqTrigger++`, 4106 `SeqTrigger>0 && PathLength>1`, 4108 `SeqTrigger--`, 4171 — matching …
- **Four alpha-blend Pix2D methods missing; type-3 interface alpha rendering dropped** (`Pix2D+Pix8`, Java `Pix2D.java:88-129,158-167,222-256,282-309 (callers Client.java:6565,10659,10661)` / Go `pkg/jagex2/graphics/pix2d/pix2d.go:1-140 (DrawInterface client.go:3832-3836)`) — fillRectTrans, drawRectTrans, hlineTrans, vlineTrans are entirely absent from the Go port (grep across pkg/ and cmd/ finds no FillRectTrans/DrawRectTrans/HLineTrans/VLineTrans). These are live methods in Java-244: Client.java:10659-10661 dispatches type-3 interface rectangle children with alpha != 0 to fillRectTrans/drawRectTrans, and Client.java:6…
- **draw3DEntityElements friend-flash (field1264) block using hlineTrans not ported** (`Pix2D+Pix8`, Java `Client.java:6560-6567` / Go `pkg/jagex2/client/client.go (no counterpart; grep Field1264 finds nothing)`) — The Java draw3DEntityElements block 'if (this.field1264 > 0) { ... Pix2D.hlineTrans(offset + i, w, 16776960, 256 - w / 2, this.field1264); }' (a yellow flashing banner) is not ported. There is no Field1264 field and no hlineTrans loop in the Go client. This is downstream of the missing hlineTrans method and is unimplemented functionality.
- **MidiPlayer.java entirely reimplemented (meltysynth/oto), not faithfully ported — WS5 audio reconciliation** (`SignLink+Midi`, Java `MidiPlayer.java:1-181` / Go `pkg/jagex2/sound/audio/midi_native.go:1-554`) — The whole MidiPlayer class — constructor (javax Synthesizer/Sequencer/Receiver wiring), setSoundfont (unload/loadAllInstruments), setVolume(velocity,volume), play (setLoopCount loop==1?-1:0), stop, running, resetVolume, setTick (the six 16-channel controller-reset sweeps: CC123/CC120/CC121/CC0/CC32/PC0), sendMessage (ShortMessage), closeImpl, setVo…
- **audioLoop() midiFadingIn/Out +8/-8 step state machine not ported (WS5)** (`SignLink+Midi`, Java `SignLink.java:369-476` / Go `pkg/jagex2/sound/audio/midi_native.go:282-342`) — Java run() calls audioLoop() every 50ms iteration (line 195). audioLoop implements: playMidi() with the midiFadingOut/midiFadingIn guard and the midifade!=0 && midiPlayer.running() fade-out trigger; the per-cycle midiFadeVol += 8 / -= 8 ramp clamped to [0, midivol] driving midiPlayer.setVolume(0, midiFadeVol); the midi=="stop"/"voladjust"/path disp…
- **playMidi() fade state machine (midiFadingIn/Out/midiFadeVol) not ported (WS5)** (`SignLink+Midi`, Java `SignLink.java:369-388` / Go `pkg/jagex2/client/client.go:3366-3387`) — Java playMidi(music) has a 3-way fade guard: return if midiFadingOut; if !midiFadingIn && midifade!=0 && midiPlayer.running() begin a fade-out (set midiFadingOut, midiFadeVol=midivol, return); otherwise play with either midiFadeVol or midivol depending on midiFadingIn. Go's c.SaveMidi/audio.PlayMIDI passes a boolean `fade` straight to the driver's …
- **MidiVol/WaveVol default 96 not ported; Go defaults 0 under a different (centibel) volume model** (`SignLink+Midi`, Java `SignLink.java:59,71` / Go `pkg/jagex2/client/sign/signlink/signlink.go:73,77`) — Java initializes midivol = 96 and wavevol = 96 (the MidiPlayer velocity/volume scale, 0..127-ish). Go declares MidiVol and WaveVol with no initializer (default 0). Because the Go audio layer reinterprets these fields as CENTIBELS (volumeFromCentibels: 0 = full volume, negative = quieter — format.go:15), the default-0 is 'full volume', not silence, …
- **Audio consumer/playback (WS5) not yet wired to synthesized wave output** (`Sound`, Java `Wave.java:47-54 (generate -> Packet for playback)` / Go `pkg/jagex2/sound/wave/wave.go:44-50; pkg/jagex2/sound/audio/wave_native.go; wave_js.go`) — The synthesis math (Tone/Wave/Envelope) IS fully ported and audited. The downstream audio playback path (WS5 audio work) is the open item per the unit note. `wave.Generate` correctly returns the RIFF Packet; the audio/ package (platform-specific wave_native.go/wave_js.go) consumes WAV bytes. This is consumer-side wiring, classified deferred per the…
- **showContextMenu uses 225 viewport-chrome offsets, not 244 — right-click menu mispositioned in all three areas** (`client/Client#10`, Java `Client.java:9580-9650` / Go `pkg/jagex2/client/client.go:6503-6565`) — Every hit-region bound and subtraction offset in the Go matches 225-clean (verified deob/client.java:6585+) not 244. 244 area-0 region is `mouseClickX>4 && >4 && <516 && <338` with `x = mouseClickX-4-width/2; y = mouseClickY-4`; Go uses `>8 && >11 && <520 && <345` with `-8`/`-11`. Area-1: 244 `>553 && >205 && <743 && <466` / `-553`/`-205`; Go `>562…
- **DrawInterface type-2 slot visibility test hardcodes clip bounds and inverts strictness** (`client/Client#11`, Java `Client.java:10574` / Go `pkg/jagex2/client/client.go:3658`) — Java gates per-slot icon rendering on `slotX > Pix2D.left - 32 && slotX < Pix2D.right && slotY > Pix2D.top - 32 && slotY < Pix2D.bottom` using the CURRENT dynamic clip rectangle (which DrawInterface itself reset to com.width+x / com.height+y / y / x at entry). The Go port hardcodes `var18 >= -32 && var18 <= 512 && var32 >= -32 && var32 <= 334`, whi…
- **DrawChatback missing 244 mod-crown (@cr1@/@cr2@) icon rendering and prefix stripping** (`client/Client#12`, Java `Client.java:11840-11848,11859-11865,11880-11886` / Go `pkg/jagex2/client/client.go:9416-9472`) — Java 244 drawChat strips a leading `@cr1@`/`@cr2@` tag from the sender, sets modicon=1/2, and for message types 1/2 and 3/7 plots imageModIcons[0]/[1] (the mod/admin crown) before the sender name, advancing x by 14. The Go message-rendering loop has none of this: no prefix stripping, no modicon plotting. The crown sprites ARE loaded (ImageModIcons)…
- **ImageMapedge loaded but drawMinimapArrow consumer deferred to UI-polish pass** (`client/Client#2`, Java `Client.java:1755-1756 (mapedge sprite load; consumer in minimap-arrow rendering)` / Go `pkg/jagex2/client/client.go:5784-5788`) — load() loads the new-in-244 'mapedge' minimap hint-arrow edge sprite (c.ImageMapedge = NewPix323(jagMedia,'mapedge',0); .Trim()) for parity, but the comment states its drawMinimapArrow consumer is deferred to the UI-polish pass. The sprite is loaded and trimmed correctly; only the downstream rendering is unimplemented.
- **mod_icons crown sprites loaded but @cr1@/@cr2@ crown rendering deferred** (`client/Client#2`, Java `Client.java:1828-1830 (mod_icons load)` / Go `pkg/jagex2/client/client.go:5873-5878`) — load() loads the new-in-244 mod/admin chat-crown sprites (imageModIcons[0..1] = NewPix8(jagMedia,'mod_icons',i)) for parity, but the comment states the @cr1@/@cr2@ crown rendering that consumes them is still deferred. Sprite load itself is faithful.
- **HandleInputKey omits @cr1@/@cr2@ mod-crown name prefix for local public chat (244 chat-crowns delta)** (`client/Client#5`, Java `Client.java:4796-4802 (handleInputKey)` / Go `pkg/jagex2/client/client.go:2356 (HandleInputKey)`) — Java-244 prefixes the local player's outgoing public-chat name with a staff crown: `if (staffmodlevel == 2) addMessage(msg, "@cr2@" + name, 2); else if (staffmodlevel == 1) addMessage(msg, "@cr1@" + name, 2); else addMessage(msg, name, 2);`. The Go does a single `c.AddMessage(2, c.LocalPlayer.Chat, c.LocalPlayer.Name)` with no crown prefix. This is…
- **UI hit-test coordinates (mouse/minimap/tab/chat-mode) and the spell-cancel box are still on the 225 fixed layout, not the 244 interface reshape** (`client/Client#5`, Java `Client.java:4048 (mouse spell box), 4169-4170 (minimap), 4178-4262 (tabs), 4287-4316 (chat mode), 4097-4106/4138-4147 (menuArea offsets)` / Go `pkg/jagex2/client/client.go:8103/8149-8190 (HandleMouseInput), 8065-8066 (HandleMinimapInput), 8308-8377 (HandleTabInput), 1912-1939 (HandleChatSettingsInput)`) — Every UI hit-test region in these four handlers uses the rev-225 layout coordinates, which differ from Java-244. Examples: minimap origin Go `MouseClickX-21-561 / Y-9-5` vs Java `mouseClickX-25-550 / y-5-4`; tab[0] Go X549-583,Y195-231 vs Java X539-573,Y169-205; bottom tabs Go Y492-528 vs Java Y466-503; chat-mode public box Go X8-108,Y490-522 vs Ja…
- **lag() debug method not ported and ::lag / ::prefetchmusic command paths absent** (`client/Client#5`, Java `Client.java:4831-4845 (lag)` / Go `pkg/jagex2/client/client.go (no counterpart)`) — The lag() debug dump (System.out.println of flameCycle, onDemand.cycle, loopCycle, drawCycle, ptype, psize; stream.debug(); super.debug=true) is not ported to Go. It is only reachable via the staff `::lag` command, which is also absent from HandleInputKey (see the ::command finding). Self-consistent omission of a debug-only path; reported as missin…
- **Draw3DEntityElements: viewportOverlayInterfaceId rendering not implemented** (`client/Client#7`, Java `Client.java:6555-6558` / Go `pkg/jagex2/client/client.go:6214-6226 (field decl 538)`) — Java 244 renders the viewport overlay interface (`if (viewportOverlayInterfaceId != -1) { updateInterfaceAnimation(sceneDelta, viewportOverlayInterfaceId); drawInterface(0,0,Component.types[viewportOverlayInterfaceId],0); }`) BEFORE the main viewport interface. The Go Draw3DEntityElements has no such block; the field ViewportOverlayInterfaceID is s…
- **Draw3DEntityElements: field1264 screen-flash band (244 feature) entirely unported** (`client/Client#7`, Java `Client.java:6560-6567 (field 563; set 7378; tick 2935-2936; reset 2692)` / Go `pkg/jagex2/client/client.go:6214-6261`) — Java 244 draws a pulsing yellow horizontal-line band when field1264 > 0: `int offset = 302 - (int) Math.abs(Math.sin(field1264/10.0)*10.0); for (i=0;i<30;i++){ int w=(30-i)*16; Pix2D.hlineTrans(offset+i, w, 16776960, 256-w/2, field1264); }`. The Go has no field1264 field, no decrement, no opcode-192 handler, and no hlineTrans (alpha hline) primitiv…
- **DrawPrivateMessages: @cr1@/@cr2@ mod-crown rendering and sender prefix-strip not implemented** (`client/Client#7`, Java `Client.java:6634-6661` / Go `pkg/jagex2/client/client.go:969-977 (load site 5873-5878)`) — Java 244 split-private chat strips an `@cr1@`/`@cr2@` prefix from the sender (sender=sender.substring(5); modlevel=1/2), draws the literal 'From' label, then plots imageModIcons[0] (pmod) or [1] (jmod) crown before the sender, advancing x by 14. The Go draws a single concatenated string `"From "+sender+": "+text` with no prefix strip and no crown. …
- **ClientEntity.Teleport (callee of getPlayerLocal op==3) drops one of Java move()'s three field resets — preanimRouteLength=0 appears unported** (`client/Client#9`, Java `ClientEntity.java:195-197 (move clears routeLength, preanimRouteLength, seqDelayMove); read sites Client.java:5004,5009,5119` / Go `pkg/jagex2/dash3d/entity/clententity.go:104-105 (Teleport clears PathLength + SeqTrigger only); struct fields lines 27-31`) — getPlayerLocal op==3 (Go GetPlayerLocal line 9378) calls LocalPlayer.Teleport, the port of Java ClientEntity.move(teleport,x,z). Java move()'s teleport branch resets THREE fields: routeLength=0, preanimRouteLength=0, seqDelayMove=0 (ClientEntity.java:195-197). Go Teleport resets only two: PathLength=0 (=routeLength) and SeqTrigger=0 (=seqDelayMove)…

### From the marker sweep (code + planning docs)

- `pkg/jagex2/client/client.go:5873-5875; pkg/jagex2/client/client.go:5417 (unrelated msg, ignore); WS2-PROTOCOL-DESIGN.md:84-85; WS2-PROTOCOL-DESIGN.md:590; WS2-PROTOCOL-DESIGN.md:1176-1179; WS2-PROTOCOL-DESIGN.md:1192-1193; WS2-PROTOCOL-DESIGN.md:1198` — **still deferred / DEFERRED (flagged) / chat crowns @cr1@/@cr2@**: Chat-crown sprites (mod/admin @cr1@/@cr2@) are loaded into ImageModIcons for parity but the actual crown rendering that strips and draws them in chat lines is not implemented.
- `pkg/jagex2/client/client.go:5784-5787` — **deferred to the UI-polish pass (see DrawMinimap)**: The new-in-244 'mapedge' minimap hint-arrow sprite is loaded but its drawMinimapArrow consumer is not implemented (deferred to UI polish).
- `pkg/jagex2/client/client.go:538; pkg/jagex2/client/client.go:9995-10000; WS2-PROTOCOL-DESIGN.md:1177-1178; WS2-PROTOCOL-DESIGN.md:1193` — **WS2-followup: render viewportOverlay in DrawScene; IF_OPENOVERLAY render**: The IF_OPENOVERLAY (opcode 158) handler stores ViewportOverlayInterfaceID but DrawScene never renders the viewport overlay interface (cosmetic; rendering follow-up).
- `LOGIC-DELTA-SCOPE.md:201; LOGIC-DELTA-SCOPE.md:213; WS1-MODEL-LOADER-DESIGN.md:434-435; WS1-MODEL-LOADER-DESIGN.md:537; WS2-PROTOCOL-DESIGN.md:1193` — **WS5 / Deferred to WS5: real MIDI/wave sink**: WS5 (Audio) is an independent, not-yet-started workstream for the 244 audio path; the WS1 SaveMidi dispatch is wired but the broader 244 audio re-port remains scheduled.
- `LOGIC-DELTA-SCOPE.md:202; LOGIC-DELTA-SCOPE.md:213; WS2-PROTOCOL-DESIGN.md:1193` — **UI/render polish / UI polish**: A general 'UI/render polish' workstream is listed as independent remaining work (slot anytime), covering cosmetic rendering items like the mapedge arrow and crowns.
- `LOGIC-DELTA-SCOPE.md:199; LOGIC-DELTA-SCOPE.md:214; WS1-MODEL-LOADER-DESIGN.md:8-9; WS1-MODEL-LOADER-DESIGN.md:24; WS1-MODEL-LOADER-DESIGN.md:524-525; WS2-PROTOCOL-DESIGN.md:3-18; WS2-PROTOCOL-DESIGN.md:140; WS2-PROTOCOL-DESIGN.md:1147; WS2-PROTOCOL-DESIGN.md:1184; WS3-MODELSOURCE-DESIGN.md:9; WS3-MODELSOURCE-DESIGN.md:101` — **host smoke test deferred / pending host smoke test (host-only runtime gate)**: The host-only smoke test against a 244 server (login + REBUILD + render models) is the remaining runtime verification gate, deferred because it cannot run in the sandbox. **PASSED 2026-06-03** post-fix-pass (client @7e9b28e+docs, server Engine-TS `244-GOSCAPE` @9aadcec4): connect → login → REBUILD → scene render → walk/interact, on the migrated 765x503 layout.
- `WS2-PROTOCOL-DESIGN.md:1178-1179; WS2-PROTOCOL-DESIGN.md:1192-1193` — **opcode-103 confirm-then-remove**: Server opcode 103 was removed in WS2 Inc 2 on the assumption the 244 server never sends it. **CONFIRMED statically 2026-06-03** against Engine-TS `244-GOSCAPE` @9aadcec4: opcode 103 = `IF_SETRECOL` (`ServerGameProt.ts:17`), and that declaration is the *only* occurrence of the constant in the entire server source — no `ServerGameMessage` subclass, no codec, no script command references it, and the encoder-registry architecture has no raw-opcode write path. The server is structurally incapable of emitting it.
- `WS2-PROTOCOL-DESIGN.md:1193` — **WS3 cleanups**: Miscellaneous WS3 ModelSource/scene cleanups are noted as remaining (not in WS2 scope).
- `pkg/jagex2/client/viewbox.go:41-45; pkg/jagex2/client/viewbox.go:50-51; PORTING.md:117` — **Deferred cleanup (intentionally NOT done); intentional stub**: ViewBox is a sentinel stub with no live consumer; deleting the file and the now-vestigial Client.Frame *ViewBox field is a deferred dedicated-pass refactor.
- `WS1-MODEL-LOADER-DESIGN.md:21-24; WS1-MODEL-LOADER-DESIGN.md:72-73; WS1-MODEL-LOADER-DESIGN.md:534-538; LOGIC-DELTA-SCOPE.md:46-62; pkg/jagex2/io/ondemand/ondemand.go:8; pkg/jagex2/io/ondemand/ondemand.go:42; pkg/jagex2/io/ondemand/ondemand.go:165; pkg/jagex2/io/ondemand/ondemand.go:173; pkg/jagex2/io/ondemand/ondemand.go:195; pkg/jagex2/io/ondemand/ondemand.go:523; pkg/jagex2/io/ondemand/ondemand.go:678; pkg/jagex2/io/ondemand/ondemand.go:722; pkg/jagex2/client/sign/signlink/signlink.go:92` — **intentionally not ported (WS1) / socket transport not ported**: The Java socket-based OnDemand subsystem (FileStream .idx/.dat store, OnDemandRequest wire framing, stall/heartbeat loops, signlink cacheload/cachesave/reporterror) is intentionally not ported because the Go client uses HTTP/WS transport.
- `cmd/client/main.go:25; pkg/jagex2/client/viewbox.go:33; pkg/jagex2/client/client.go:6736` — **is not ported (port-offset)**: Java's single port-offset-over-fixed-base argument is intentionally not ported; the Go client takes full scheme://host:port for world-server / ondemand-server from CLI flags instead.
- `pkg/jagex2/client/client.go:8479; PORTING.md:167` — **GetParameter ... intentionally not ported**: Java's applet GetParameter (HTML <param>) is intentionally not ported; the Go client reads config from CLI args / clientextras.
- `pkg/jagex2/client/client.go:134; pkg/jagex2/client/client.go:6916; pkg/jagex2/client/client.go:7286-7287` — **Intentionally not ported (deobfuscator artifact / field1382 / class61)**: Deobfuscator-emitted artifacts (a dead deob array, the field1382 dead-write reset, and the class61 single-static-array stub) are intentionally not ported.
- `pkg/jagex2/io/protocol.go:4` — **intentionally NOT ported (scramble table)**: An unreferenced Java protocol scramble table is intentionally not ported (zero references anywhere).
- `pkg/jagex2/io/clientprot.go:6; WS2-PROTOCOL-DESIGN.md:234; WS2-PROTOCOL-DESIGN.md:242` — **intentionally not ported (CLIENTPROT_LOOKUP)**: The CLIENTPROT_LOOKUP client-protocol entry is a deob artifact unused in Java or TS and is intentionally not ported.
- `pkg/jagex2/io/clientstream/clientstream.go:119` — **intentionally NOT ported (io-net #19)**: Java ClientStream per-byte stdout debug-spam logging is intentionally not ported.
- `pkg/jagex2/config/objtype/objtype.go:571; pkg/jagex2/config/objtype/objtype.go:468-469 (nolint); pkg/jagex2/config/objtype/objtype.go:535 (nolint); pkg/jagex2/config/component/component.go:330; pkg/jagex2/config/component/component_test.go:35; pkg/jagex2/config/component/component_test.go:66; WS2-PROTOCOL-DESIGN.md:24; WS2-PROTOCOL-DESIGN.md:132; WS2-PROTOCOL-DESIGN.md:1046; LOGIC-DELTA-SCOPE.md:131-136` — **not yet available / deferred (modelType,id) pair / type-6 deferred model**: Component type-6 lazy/deferred model resolution: the model id is stored as a deferred (type,id) pair and resolved on demand rather than eagerly; this is the implemented WS2-Inc8 deferred-model design (lazy-resolution behavior, not unfinished).
- `PORTING.md:6; PORTING.md:42-79; PORTING.md:193-200; PORTING.md:242-245; PORTING.md:362-451` — **remaining work / verification-style TODOs / // TODO: verify**: PORTING.md is the authoritative roadmap; nearly all its enumerated TODO/verify/stub items are struck-through as completed (rev-225 era), leaving only the doc-level convention that residual verification-style TODOs may exist from the translation pass.
- `pkg/jagex2/config/objtype/objtype.go:468; pkg/jagex2/config/objtype/objtype.go:535; pkg/jagex2/datastruct/lrucache.go:37; pkg/jagex2/client/gameshell.go:467; pkg/jagex2/dash3d/typ/ground.go:230; pkg/jagex2/dash3d/typ/ground.go:237; pkg/jagex2/dash3d/typ/ground.go:238; pkg/jagex2/dash3d/model/model.go:302; pkg/jagex2/dash3d/model/model.go:1508; pkg/jagex2/io/clientstream/clientstream.go:329; pkg/jagex2/io/bzip2/bzip2.go:228-239; pkg/jagex2/io/bzip2/bzip2.go:576; pkg/jagex2/wordenc/wordpack/wordpack.go:75; pkg/jagex2/wordenc/wordfilter/wordfilter.go:271; pkg/jagex2/wordenc/wordfilter/wordfilter.go:416; pkg/jagex2/wordenc/wordfilter/wordfilter.go:632; pkg/jagex2/client/client.go:2420; pkg/jagex2/client/client.go:3460; pkg/jagex2/client/client.go:3479; pkg/jagex2/client/client.go:10601` — **//nolint citing Java refs (faithful-port fidelity)**: Numerous //nolint suppressions document deliberate bug-for-bug faithful ports of Java idioms (dead-write increments, always-false conditions, ineffectual stream-advance reads, split flag shapes, range-over-rune char semantics); these are intentional fidelity choices, not unfinished work, but flagged here as deferred-cleanup candidates per the convention.

---

## Intentional deviations (126)

Registry-covered (transport / platform / thread-model / deob-artifact / scratch-model /
logging seams) or explicitly documented. Listed for completeness:

| Unit | Title | Java | Go |
|---|---|---|---|
| `BZip2` | BZip2State.cftabCopy dead field omitted from Go state struct | `BZip2State.java:95-96` | `pkg/jagex2/io/bzip2state/bzip2state.go:19-57` |
| `BZip2` | Dead local nblockMAX dropped in Go Decompress | `BZip2.java:348` | `pkg/jagex2/io/bzip2/bzip2.go:357-358` |
| `BZip2` | Java `synchronized(state)` mapped to a package-level sync.Mutex; thread-model deviation | `BZip2.java:13-14,28-30` | `pkg/jagex2/io/bzip2/bzip2.go:10-18,22-23` |
| `BZip2` | Faithful-port nolint markers on discarded getUnsignedChar reads, single-shot outer loop, and load-bearing int32 cast | `BZip2.java:186-195,591` | `pkg/jagex2/io/bzip2/bzip2.go:228-239,576,602-605` |
| `ClientPlayer` | Per-instance seqModel scratch replaces shared static Model.empty (wasm alloc optimization) | `ClientPlayer.java:368-369` | `pkg/jagex2/dash3d/entity/playerentity/clientplayer.go:45,307-311` |
| `ClientStream` | Per-byte System.out.println debug instrumentation not ported | `ClientStream.java:54,82,97-132,190-201` | `pkg/jagex2/io/clientstream/clientstream.go:117-122` |
| `ClientStream` | ClientStream.debug() method has no Go counterpart (caller Client.lag() also unported) | `ClientStream.java:190-201` | `pkg/jagex2/io/clientstream/clientstream.go (absent)` |
| `ClientStream` | SO_TIMEOUT relocated from socket to consumer-side wait; Go-original timeout machinery | `ClientStream.java:47,83-90,93-107` | `pkg/jagex2/io/clientstream/clientstream.go:50-57,207-229,264-292` |
| `ClientStream` | Inbound reader goroutine + rbuf ring is a Go-original Available() seam | `ClientStream.java:87-90,93-107` | `pkg/jagex2/io/clientstream/clientstream.go:143-197,237-247` |
| `ClientStream` | out.flush() drop in writer run() — net.Conn is unbuffered | `ClientStream.java:179-185` | `pkg/jagex2/io/clientstream/clientstream.go:392-394` |
| `CollisionMap` | TestWDecor drops the boolean arg3 parameter and its `throw new NullPointerException()` guard | `CollisionMap.java:494-497` | `pkg/jagex2/dash3d/collisionmap.go:445-448` |
| `Component` | field98/field99 Type==1 reads preserved as discards; storage omitted (deob-artifact policy) | `Component.java:277-280` | `pkg/jagex2/config/component/component.go:172-178` |
| `Component` | iop empty-string nulling uses "" instead of Java null (project string convention) | `Component.java:311-317,409-414` | `pkg/jagex2/config/component/component.go:206-216,300-306` |
| `Component` | GetModel/LoadModel take localPlayer as a parameter instead of reading Client.localPlayer global | `Component.java:503` | `pkg/jagex2/config/component/component.go:329,357,367` |
| `ConfigSmall` | UnkType (mc) intentionally not ported — pure deobfuscation artifact, never instantiated or read | `UnkType.java:1-35; Client.java:2163` | `(no Go counterpart file)` |
| `ConfigSmall` | SeqType opcode 6/7 fields RightHand/LeftHand are name-swapped vs Java replaceheldleft/replaceheldright but used consistently by value | `SeqType.java:42,44,135-138; ClientPlayer.java:271-277` | `pkg/jagex2/config/seqtype/seqtype.go:24-25,143-146; pkg/jagex2/dash3d/entity/playerentity/clientplayer.go:215-225` |
| `Dash3dSmall` | Sprite.MinY / Decor.MinY are Go-added cache fields relocating Java ModelSource.minY off the interface | `ModelSource.java:9-22; Sprite.java:7-50; Decor.java:5-31` | `pkg/jagex2/dash3d/typ/sprite.go:29-34, pkg/jagex2/dash3d/typ/decor.go:17-25, pkg/jagex2/dash3d/entity/modelsource.go:5-7` |
| `Datastruct` | LruCache redesigned from Java HashTable+DoublyLinkList dual-membership to Go map + history list | `LruCache.java:25-40 (fields), 43-77 (get/put/clear)` | `pkg/jagex2/datastruct/lrucache.go:12-97` |
| `Datastruct` | LruCache.Delete is a Go-original method with documented INTENTIONAL divergence from Java unlink-leak behavior | `LruCache.java has no delete; ObjType.java:476-479 (iconCache.get(id); icon.unlink())` | `pkg/jagex2/datastruct/lrucache.go:63-86` |
| `Datastruct` | LruCache.Put does not guard duplicate-key inserts (Java HashTable.put unlinked the prior bucket node) | `HashTable.java:38-48 (put unlinks prior node); LruCache.java:43-60` | `pkg/jagex2/datastruct/lrucache.go:40-61` |
| `Datastruct` | LruCache drops Java's notFound/found statistics counters and the dead `search` field | `LruCache.java:9-15 (notFound, found, search), 31-32, 38, 41, 54-58` | `pkg/jagex2/datastruct/lrucache.go:12-26` |
| `Datastruct` | Standalone hashtable package is a faithful HashTable.java port with NO production callers (Go-side extra) | `HashTable.java:1-52; Linkable.java:1-21` | `pkg/jagex2/datastruct/hashtable/hashtable.go:1-104` |
| `EntityA` | ClientNpc.seqModel per-frame scratch-model reuse (ResetFromModel6) — wasm allocation optimization | `ClientNpc.java:59-77 (getAnimatedModel — allocates fresh each call)` | `pkg/jagex2/dash3d/entity/clientnpc.go:14, 72-75, 81-84` |
| `EntityA` | PathableEntity interface + Pathing() adapter — Go-idiom replacement for Java parent-reference dispatch | `ClientNpc extends ClientEntity / ClientPlayer extends ClientEntity (inheritance)` | `pkg/jagex2/dash3d/entity/clententity.go:160-174` |
| `EntityA` | ClientObj / ClientEntity do not embed ModelSource base (vertexNormal[]/minY) — WS3 interface design | `ModelSource.java:9-25 (vertexNormal[], minY=1000, draw caches minY); ClientObj.java/ClientEntity extends ModelSource` | `pkg/jagex2/dash3d/entity/modelsource.go:5-7 (interface), clientobj.go:8-11, clententity.go:5-58` |
| `EntityB` | ClientProj dead-write fields field502/field503 omitted in Go without a marker comment | `ClientProj.java:9-13` | `pkg/jagex2/dash3d/entity/clientproj.go:11-36` |
| `EntityB` | ModelSource translated from a Java class (DoublyLinkable subclass with minY/vertexNormal/draw) to a Go interface; behavior redistributed to scene nodes | `ModelSource.java:6-28` | `pkg/jagex2/dash3d/entity/modelsource.go:5-21; pkg/jagex2/dash3d/model/model.go:163-175; pkg/jagex2/dash3d/world3d/world3d.go:1288-1290,1366-1368,1579-1581,678-694` |
| `EntityB` | ModelSourceOf helper is a Go-original (typed-nil interface guard), documented with Java rationale | `ModelSource.java:24-27` | `pkg/jagex2/dash3d/entity/modelsource.go:9-21` |
| `EntityB` | LocChange is the completed merge of rev-225 LocChange + LocMergeEntity; LocMergeEntity type no longer exists | `LocChange.java:6-43` | `pkg/jagex2/dash3d/entity/locchange.go:9-28` |
| `GameShell` | Screen size 789x532 (Go) vs Java standalone 765x503 from initApplication(503,765) | `GameShell.java:116-123` | `pkg/jagex2/client/gameshell.go:447-449` |
| `GameShell` | MouseTracking class not ported (dead deob artifact: unstarted thread, never-read sample arrays) | `MouseTracking.java:1-47` | `pkg/jagex2/client/ (no counterpart)` |
| `GameShell` | GameShell debug field + per-frame timing-diagnostics println block not ported | `GameShell.java:41,234-245` | `pkg/jagex2/client/gameshell.go:514-516 (and absent Debug field)` |
| `GameShell` | fps field and its per-frame computation dropped (dead deob residue) | `GameShell.java:38,228-230` | `pkg/jagex2/client/client.go:119-122, gameshell.go:514-516` |
| `GameShell` | start()/stop()/destroy() AWT applet-lifecycle methods not ported | `GameShell.java:274-297` | `pkg/jagex2/client/gameshell.go (no counterpart); RunShell loop uses platform.Active.ShouldClose()` |
| `GameShell` | initApplication/initApplet/getBaseComponent/startThread/update(Graphics)/paint(Graphics)/window* not ported (platform + thread seam) | `GameShell.java:115-132,299-315,534-608,552-572` | `pkg/jagex2/client/gameshell.go:447-449 (initScreenSize replaces InitApplication); viewbox.go` |
| `GameShell` | mouseClickTime/lastMouseClickTime fields dropped; click-latch restructured (no LastMouseClick* fields) | `GameShell.java:109-113,214-225,317-356` | `pkg/jagex2/client/gameshell.go:145-180,507-512` |
| `Ground` | Pure-deob dead fields shape0P1/shape0P2/shape0P3 omitted without a documenting comment | `Ground.java:68-75` | `pkg/jagex2/dash3d/typ/ground.go:3-11` |
| `InputTracking` | trackedCount field and its 9 increment sites dropped as deob residue | `InputTracking.java:22,87,114,147,184,218,252,273,294,315` | `pkg/jagex2/client/inputtracking/inputtracking.go:15-18` |
| `LocType` | ModelCacheDynamic capacity 256 instead of Java's LruCache(30) | `LocType.java:80` | `pkg/jagex2/config/loctype/loctype.go:23-33` |
| `LocType` | Op absence marker is "" (empty string) in Go vs null in Java | `LocType.java:294-297` | `pkg/jagex2/config/loctype/loctype.go:215-226` |
| `NpcType` | Opcodes 90/91/92 wire reads kept as discards; resizex/y/z fields omitted (deob artifact) | `NpcType.java:72-79,210-215` | `pkg/jagex2/config/npctype/npctype.go:36-40,164-168` |
| `NpcType` | GetSequencedModel uses caller-supplied scratch model instead of static Model.empty | `NpcType.java:273-274` | `pkg/jagex2/config/npctype/npctype.go:190,223-224` |
| `NpcType` | Op "hidden" sentinel stored as empty string instead of Java null | `NpcType.java:190-193` | `pkg/jagex2/config/npctype/npctype.go:144-149` |
| `ObjType` | GetIcon 2-arg signature is a registry-style deviation only insofar as the whole 244 getIcon rewrite was out of WS4 (decode-only) scope | `ObjType.java:473-622` | `pkg/jagex2/config/objtype/objtype.go:346-450` |
| `ObjType` | code9/code10 deob-artifact fields omitted; wire reads preserved as discards | `ObjType.java:70,73,231-232,291-294` | `pkg/jagex2/config/objtype/objtype.go:39-43,197-203` |
| `OnDemand` | Socket transport (send body, socket read(), heartbeat, partial-part reassembly, run() resend/waitCycles tail) not ported | `OnDemand.java:414-515 (run tail),661-772 (socket read),792-838 (send)` | `pkg/jagex2/io/ondemand/ondemand.go:525-547,679-724` |
| `OnDemand` | unpack() startThread + app/Client wiring, CRC32 field, socketOpenTime/in/out/socket fields, buf scratch, app.ingame not ported | `OnDemand.java:57-132,212-215` | `pkg/jagex2/io/ondemand/ondemand.go:94-189,196-285` |
| `OnDemand` | FileStream (jagex2.io.FileStream) entirely not ported; replaced by Cache seam | `FileStream.java:1-252 (read, write x2, seek, sector reassembly)` | `pkg/jagex2/io/ondemand/ondemand.go:71-76 (Cache interface)` |
| `OnDemand` | OnDemandRequest.Cycle field present but unused (socket stall-detection only) | `OnDemandRequest.java:18-19; OnDemand.java:453-470,684` | `pkg/jagex2/io/ondemand/ondemand.go:42-43` |
| `OnDemand` | cmd/client/ondemand.go parseOndemandServer is Go-original CLI helper (no Java source) | `n/a (Java used applet params + Client.portOffset+43594)` | `cmd/client/ondemand.go:16-48` |
| `Packet+Isaac+Jagfile` | RSAEnc/javaBytesFromBigInt comment describes a hypothetical bug, not an actual one; code is faithful | `Packet.java:321-335` | `pkg/jagex2/io/packet.go:377-446` |
| `Packet+Isaac+Jagfile` | Packet pool replaced by sync.Pool (bounded LIFO + counter caps dropped) | `Packet.java:43-118` | `pkg/jagex2/io/packet.go:25-99` |
| `Packet+Isaac+Jagfile` | PJStr/GJStr/Latin1ToUTF8 transcoding seam for UTF-8 Go strings | `Packet.java:185-190,249-255` | `pkg/jagex2/io/packet.go:183-190,256-312` |
| `Packet+Isaac+Jagfile` | latin1ToUTF8 / Latin1ToUTF8 are Go-original helpers with no Java counterpart | `Packet.java:249-255 (gjstr only)` | `pkg/jagex2/io/packet.go:284-312` |
| `Pix2D+Pix8` | Pix2D.Reset() is a Go-original helper with no Java equivalent | `Pix2D.java (no counterpart)` | `pkg/jagex2/graphics/pix2d/pix2d.go:27-42` |
| `Pix32` | NewPix322 replaces AWT Toolkit/MediaTracker/PixelGrabber with image.Decode (platform seam) | `Pix32.java:43-63` | `pkg/jagex2/graphics/pix32/pix32.go:37-79` |
| `Pix32` | Crop/Scale/DrawRotatedMasked/DrawRotated use defer recover() to mirror Java try/catch swallow | `Pix32.java:378-409 (drawRotatedMasked), 415-450 (drawRotated); 225 crop/scale try/catch` | `pkg/jagex2/graphics/pix32/pix32.go:352-356,409-413,503,542` |
| `PixFont` | StringWidth / GlyphIndex clamp out-of-Latin-1 runes instead of indexing out of bounds like Java | `PixFont.java:126-140,148-149` | `pkg/jagex2/graphics/pixfont/pixfont.go:23-32,156-181` |
| `SignLink+Midi` | active lifecycle / threadliveid re-entrancy guard / !active early-returns not ported | `SignLink.java:75,84-110,113-114,139-140,340` | `pkg/jagex2/client/sign/signlink/signlink.go:86-107` |
| `SignLink+Midi` | opensocket socketreq busy-wait replaced by direct dial + WS branch + 10s timeout | `SignLink.java:255-270` | `pkg/jagex2/client/sign/signlink/signlink.go:305-313` |
| `SignLink+Midi` | openurl getCodeBase()-relative stream replaced by direct HTTP GET against OndemandBaseURL | `SignLink.java:185-193,272-287` | `pkg/jagex2/client/sign/signlink/signlink.go:181-208,318-332` |
| `SignLink+Midi` | RandomAccessFile cache_dat/cache_idx[5] + 52428800-byte prune not ported (storage seam) | `SignLink.java:21-23,124-137` | `pkg/jagex2/client/sign/signlink/storage_disk.go:24-59` |
| `SignLink+Midi` | wavesave/wavereplay/midisave scratch-file protocol replaced by direct in-memory PlayWave/PlayMIDI | `SignLink.java:299-337,164-184` | `pkg/jagex2/sound/audio/wave_native.go:36-65,pkg/jagex2/sound/audio/midi_native.go:119-125` |
| `SignLink+Midi` | startthread(Runnable, priority) replaced by direct goroutines, priority dropped | `SignLink.java:294-297,149-155` | `pkg/jagex2/client/client.go:3166-3187` |
| `SignLink+Midi` | reporterror !active guard relaxed; reporterror.cgi host is a signed-applet artifact | `SignLink.java:339-357` | `pkg/jagex2/client/sign/signlink/signlink.go:416-438` |
| `SignLink+Midi` | DNSLookup forward/reverse resolution expanded beyond Java's getByName().getHostName() | `SignLink.java:156-162,289-292` | `pkg/jagex2/client/sign/signlink/signlink.go:134-165,334-339` |
| `SignLink+Midi` | CacheLoad/CacheSave/StartPriv use mu/cond/slotMu handoff instead of Java Thread.sleep busy-waits | `SignLink.java:255-337,105-110` | `pkg/jagex2/client/sign/signlink/signlink.go:99-107,251-332` |
| `Sound` | Per-frame scratch buffers shared as package-level slices (Java static int[] mirror) | `Tone.java:55-76` | `pkg/jagex2/sound/tone/tone.go:11-29` |
| `WordPack+PixMap` | Pack has an extra `terminate bool` parameter absent from Java; all callers pass true so behavior is identical | `WordPack.java:60-101` | `pkg/jagex2/wordenc/wordpack/wordpack.go:64,103-105` |
| `WordPack+PixMap` | PixMap AWT ImageProducer/ImageConsumer pipeline not ported (platform seam) | `PixMap.java:5-13,33-51,64-100` | `pkg/jagex2/graphics/pixmap/pixmap.go:15-69` |
| `WordPack+PixMap` | Draw uses hashPixels change-detection to skip unchanged GPU re-uploads; Java re-pushes pixels every frame | `PixMap.java:58-62,90-96` | `pkg/jagex2/graphics/pixmap/pixmap.go:52-86` |
| `WordPack+PixMap` | writePixmapPixels expands Java DirectColorModel 0x00RRGGBB to RGBA 0xRRGGBBFF for the texture seam | `PixMap.java:37,93` | `pkg/jagex2/graphics/pixmap/pixmap.go:88-104` |
| `client/Client#1` | FileStream disk-cache field fileStreams[5] not ported (registry: zip bundle + signlink storage seam) | `Client.java:587` | `pkg/jagex2/client/client.go:5991,8622-8624` |
| `client/Client#1` | Debug-only counters drawCycle/flameCycle and reporterror progress fields not ported | `Client.java:854,863,1184,989,4831-4838,1977` | `pkg/jagex2/client/client.go:5678-5686,10735` |
| `client/Client#1` | mouseTracking field replaced by inputtracking package; portOffset replaced by CLI flags | `Client.java:48,31` | `pkg/jagex2/client/inputtracking` |
| `client/Client#11` | AddNPCOptions Examine option appends examineIDSuffix (developer-mode id annotation) | `Client.java:10405-10410` | `pkg/jagex2/client/client.go:2149` |
| `client/Client#11` | ExecuteClientscript1 uses deferred recover() to mirror Java catch(Exception){return -1} | `Client.java:11033-11035` | `pkg/jagex2/client/client.go:8858-8862` |
| `client/Client#12` | Area-draw screen coordinates use a shifted layout vs Java 244 (789x532 window adaptation) | `Client.java:11802,11950,12053; 3216,5565,5570` | `pkg/jagex2/client/client.go:10589,9489,9394,1540,4279,4284` |
| `client/Client#12` | DrawSidebar/DrawChatback/DrawMinimap wrapped in dirty-flag guards with always-run blit (immediate-mode upload seam) | `Client.java:11785-11806,11808-11954` | `pkg/jagex2/client/client.go:10566-10590,9387-9490` |
| `client/Client#12` | Examine menu text appends developer-mode id suffix (Go-original dev feature) | `Client.java:11230` | `pkg/jagex2/client/client.go:1884` |
| `client/Client#12` | UpdateVarp midi/wave volumes use decibel-hundredths scale feeding signlink instead of Java 0-128 linear | `Client.java:11375-11388,11402-11416` | `pkg/jagex2/client/client.go:3874-3917,3410-3416` |
| `client/Client#13` | DrawFlames omits the two imageTitleN.draw(graphics,...) GPU blits (platform seam) | `Client.java:12519,12563` | `pkg/jagex2/client/client.go:1609-1709` |
| `client/Client#13` | DrawFlames takes flameMu lock absent in Java (thread-model deviation) | `Client.java:12464` | `pkg/jagex2/client/client.go:1619-1620` |
| `client/Client#13` | UnloadTitle keeps title images/flame buffers alive instead of nilling them | `Client.java:12275-12305` | `pkg/jagex2/client/client.go:2582-2601` |
| `client/Client#2` | SaveWave/ReplayWave always return true (wrapper-consumer backpressure absent) | `Client.java:1467-1479; SignLink.java:299-325 (wavesave/wavereplay return false while savereq!=null)` | `pkg/jagex2/client/client.go:5453-5464` |
| `client/Client#2` | load() reorders media-sprite/texture unpack ahead of OnDemand request loops vs Java | `Client.java:1599-1810 (Java: OnDemand setup + all request loops run BEFORE 'Unpacking media' sprite loads)` | `pkg/jagex2/client/client.go:5768-6057` |
| `client/Client#2` | load() progress percentages and messages diverge from Java 244 | `Client.java getJagFile/drawProgress calls (title 25, config 30, interface 35, media 40, textures 45, wordenc 50, sounds 55, 'Connecting to update server' 60, 'Unpacking media' 80, 'Unpacking textures' 83)` | `pkg/jagex2/client/client.go:5728-6078` |
| `client/Client#2` | Boot-time map prefetch block skipped in bundle mode (HasCache()==false) | `Client.java:1662-1690 (gated `if (this.fileStreams[0] != null)`; 12 onDemand.request(3, getMapFile(...)) calls)` | `pkg/jagex2/client/client.go:5993-6016` |
| `client/Client#2` | Applet entry/threading/param methods (init, run, startThread, getParameter, getBaseComponent) not ported | `Client.java:1317-1345 (init/run), 1381-1414 (getParameter/getBaseComponent), 1434-1445 (startThread)` | `cmd/client/main.go:19-146; pkg/jagex2/client/client.go:2835-2867 (getBaseComponent non-port note); client.go:8479 (GetParameter non-port note)` |
| `client/Client#2` | getCodeBase/getHost/openUrl/openSocket rerouted through HTTP/WS transport + CLI host seam | `Client.java:1365-1431 (getCodeBase/getParameter/getHost/getBaseComponent/openUrl/openSocket)` | `pkg/jagex2/client/client.go:5176-5182 (GetHost), 6568-6583 (OpenURL), 7307-7309 (OpenSocket), 7696-7707 (GetCodeBase)` |
| `client/Client#2` | load() host allowlist (errorHost) intentionally not enforced | `Client.java:5962-5987 region (deob load host check: getHost() must end in jagex.com/runescape.com/192.168.1.x/127.0.0.1 else errorHost=true,return)` | `pkg/jagex2/client/client.go:5661-5671` |
| `client/Client#2` | load() FileStream disk cache (fileStreams[0..4]) not allocated; GetJagFile uses signlink storage seam | `Client.java:5? (load: `if (SignLink.cache_dat != null) for i<5 fileStreams[i] = new FileStream(i+1, ...)`); getJagFile(...,int file,...) reads fileStreams[0].read(file)` | `pkg/jagex2/client/client.go:5637-5659 (no FileStream init); client.go:2488-2514 (GetJagFile uses signlink.CacheLoad(name))` |
| `client/Client#2` | load() omits Thread.sleep(100) pacing in OnDemand drain loops (no worker thread) | `Client.java:1601-1690 (each `while (onDemand.remaining()>0)` loop calls updateOnDemand() then Thread.sleep(100L))` | `pkg/jagex2/client/client.go:5956-6015` |
| `client/Client#3` | updateGame inlines the 244 updateAudio music block as a rev-225 SetMidi(name) call instead of midiSong=nextMidiSong/onDemand.request(2,...) | `Client.java:3628-3631` | `pkg/jagex2/client/client.go:7446-7448` |
| `client/Client#3` | Unload: omits onDemand.stop() + onDemand = null; adds MidiThreadActive=false; skips mouseTracking/hasFocus/drawArea nilling | `Client.java:2040-2041,2035-2038,2169` | `pkg/jagex2/client/client.go:7176-7298` |
| `client/Client#3` | login reply==2: dead-write fields field1402/field1403/field1252 + mouseTracking.length=0 + hasFocus=true not ported (no marker comment) | `Client.java:2675-2679` | `pkg/jagex2/client/client.go:6835-6836` |
| `client/Client#3` | login reply==2: oplogic10 = 0 not ported (dead-write deob field, no marker comment) | `Client.java:2773` | `pkg/jagex2/client/client.go:6906-6915` |
| `client/Client#3` | DrawProgress omits lastProgressPercent/lastProgressMessage writes + redrawFrame guard + uses 789-layout title coords | `Client.java:2189-2230` | `pkg/jagex2/client/client.go:10735-10785` |
| `client/Client#3` | UpdateOnDemand adds nil guard + inline OnDemand.Run() (thread-model seam) | `Client.java:2425-2474` | `pkg/jagex2/client/client.go:8761-8806` |
| `client/Client#4` | Examine menu options append developer-mode config-id suffix (Go-original dev feature) | `Client.java:3917,4022` | `pkg/jagex2/client/client.go:9178,9266` |
| `client/Client#4` | Platform-seam substitutions in tryReconnect/updateSceneState/buildScene (areaViewport.draw -> present, signlink.reporterror, System.gc dropped) | `Client.java:3216,3240,3248,3318` | `pkg/jagex2/client/client.go:8491,8614,8627,8674-8751` |
| `client/Client#5` | updateEntity(e,size) split into UpdateClientPlayer/UpdateClientNpc; unused size param is a faithful dead arg | `Client.java:4907-4940 (updateEntity), 4894-4905 (updateNpcs), 4849-4892 (updatePlayers)` | `pkg/jagex2/client/client.go:3941-3990 (UpdateClientPlayer/UpdateClientNpc), 9287-9299, 3971` |
| `client/Client#6` | Thread-dispatch deviation: flame goroutine started via `go c.RunFlames()` instead of startThread(this,2) | `Client.java:5466-5470` | `pkg/jagex2/client/client.go:3256-3261,6633-6638` |
| `client/Client#6` | Immediate-mode GPU-upload deviation: DrawGame/DrawTitleScreen hoist static-tile Draws out of redrawFrame guard | `Client.java:5562-5584,5547-5556` | `pkg/jagex2/client/client.go:4258-4302,3485-3503` |
| `client/Client#6` | RunFlames Java try/catch swallow replaced by natural panic propagation; PixMap has no Component arg | `Client.java:5282-5306 (getBaseComponent), 5475 onward (super.graphics draws)` | `pkg/jagex2/client/client.go:6641-6657,6666-6671` |
| `client/Client#7` | Whole-UI viewport-origin layout shift: Go uses (8,11)/(562,231)/(22,375) where 244 Java uses (4,4)/(553,205)/(17,357) | `Client.java:6549-6552 (cross), 6779-6788 (menu areas), 3216 (areaViewport.draw 4,4)` | `pkg/jagex2/client/client.go:6218-6221 (cross), 5196-5206 (menu areas), 1540/4289 (AreaViewport.Draw 8,11)` |
| `client/Client#8` | catch(IOException)/catch(Exception) modeled via inline TryReconnect + deferred recover() | `Client.java:7223-8470` | `pkg/jagex2/client/client.go:9505-9521,9646-9661` |
| `client/Client#8` | REBUILD_NORMAL map fetch via OnDemand HTTP/WS request(3,...) on the loop goroutine; AreaViewport.draw replaced by present() | `Client.java:7704-7857` | `pkg/jagex2/client/client.go:9786-9919` |
| `client/Client#9` | sortObjStacks eagerly builds obj models (GetInterfaceModel) instead of passing live ClientObj as ModelSource | `Client.java:8889-8932 (addGroundObject gets ClientObj middleObj/topObj/bottomObj as ModelSource); ClientObj.java:15-19 (getModel = ObjType.get(index).getModel(count))` | `pkg/jagex2/client/client.go:8549-8602; dash3d/world3d/world3d.go:259-287` |
| `client/Client#9` | readZonePacket LOC_ANIM retargets scene-node ModelSource to fresh ClientLocAnim (244 lineage), not 225 LocList swap | `Client.java:8496-8548` | `pkg/jagex2/client/client.go:1125-1172` |
| `dash3d/Model#1` | NewModel1 adds nil-guards on Metadata and meta[id] that rev-244 Java Model(int) does not have | `Model.java:389-393` | `pkg/jagex2/dash3d/model/model.go:345-354` |
| `dash3d/Model#1` | Static fields Model.empty, tmpVertexX/Y/Z, tmpFaceAlpha not ported | `Model.java:16-28` | `pkg/jagex2/dash3d/model/model.go:16-53` |
| `dash3d/Model#1` | Go-side extra GetModel() method (ModelSource interface adapter) | `Model.java:10` | `pkg/jagex2/dash3d/model/model.go:163-175` |
| `dash3d/Model#1` | Unpack final 'pos += dataLengthZ' dead-write preserved with nolint | `Model.java:347-348` | `pkg/jagex2/dash3d/model/model.go:299-302` |
| `dash3d/Model#2` | set() static scratch buffers (tmpVertexX/Y/Z, tmpFaceAlpha) replaced by per-instance owned pools in ResetFromModel6 | `Model.java:903-961` | `pkg/jagex2/dash3d/model/model.go:858-924` |
| `dash3d/Model#3` | DrawSimple/Draw1 recover() swallows panic mirroring Java try/catch | `Model.java:1572-1575, 1688-1691` | `pkg/jagex2/dash3d/model/model.go:1470-1476, 1583-1589` |
| `dash3d/World#2` | addLoc rotation/typecode byte: Java (byte) signed truncation vs Go byte (uint8), reconverted to int8 at scene boundary | `World.java:473` | `pkg/jagex2/dash3d/world/world.go:1034` |
| `dash3d/World#2` | addLoc dead local var19 (Java boolean var19 = false) intentionally not ported | `World.java:474` | `pkg/jagex2/dash3d/world/world.go:1019-1137` |
| `dash3d/World3D#1` | Sprite.MinY/Decor.MinY cached field + seeding replaces Java-244 ModelSource.minY base-class field | `ModelSource.java:9-22; World3D.java:398-417,1365,1632` | `pkg/jagex2/dash3d/world3d/world3d.go:330-332,421-423` |
| `graphics/Pix3D#1` | ClearTexels reclaims-and-keeps the pool instead of nulling it (Java clearTexels frees texelPool=null) | `Pix3D.java:117-124` | `pkg/jagex2/graphics/pix3d/pix3d.go:120-131` |
| `graphics/Pix3D#1` | InitPool adds a reuse fast-path absent from Java initPool | `Pix3D.java:126-143` | `pkg/jagex2/graphics/pix3d/pix3d.go:133-159` |
| `graphics/Pix3D#1` | UnpackTextures wraps body in func(){ defer recover() } to mirror Java try/catch swallow | `Pix3D.java:145-163` | `pkg/jagex2/graphics/pix3d/pix3d.go:161-178` |
| `graphics/Pix3D#1` | GetTexels palette index uses unsigned []byte (cannot crash) vs Java signed byte[] OOB-throw | `Pix3D.java:219-236` | `pkg/jagex2/graphics/pix3d/pix3d.go:235-266` |
| `graphics/Pix3D#1` | Go-original Reset() function has no Java counterpart and is unused | `(none — no reset in Pix3D.java)` | `pkg/jagex2/graphics/pix3d/pix3d.go:59-79` |
| `graphics/Pix3D#2` | Dead overload-disambiguator params dropped from Go gouraudRaster/flatRaster signatures | `Pix3D.java:860,1386` | `pkg/jagex2/graphics/pix3d/pix3d.go:856,1379` |
| `wordenc/WordFilter#2` | filterFragments drops Java's dead-write local 'compare' (pure deob artifact) | `WordFilter.java:1075` | `pkg/jagex2/wordenc/wordfilter/wordfilter.go:1077-1080` |

---

## Coverage attestation

### Audit units (all 50 completed; no silent truncation — each unit attests full range coverage)

| Unit | Java methods audited | Findings |
|---|---|---|
| `client/Client#1` | 1 | 8 |
| `client/Client#2` | 20 | 16 |
| `client/Client#3` | 13 | 17 |
| `client/Client#4` | 12 | 8 |
| `client/Client#5` | 16 | 9 |
| `client/Client#6` | 9 | 12 |
| `client/Client#7` | 20 | 10 |
| `client/Client#8` | 1 | 10 |
| `client/Client#9` | 7 | 4 |
| `client/Client#10` | 11 | 11 |
| `client/Client#11` | 11 | 9 |
| `client/Client#12` | 11 | 12 |
| `client/Client#13` | 14 | 6 |
| `graphics/Pix3D#1` | 12 | 7 |
| `graphics/Pix3D#2` | 4 | 2 |
| `graphics/Pix3D#3` | 2 | 3 |
| `dash3d/World3D#1` | 37 | 4 |
| `dash3d/World3D#2` | 7 | 3 |
| `dash3d/World3D#3` | 10 | 2 |
| `dash3d/Model#1` | 11 | 8 |
| `dash3d/Model#2` | 19 | 7 |
| `dash3d/Model#3` | 8 | 3 |
| `dash3d/World#1` | 8 | 3 |
| `dash3d/World#2` | 11 | 3 |
| `wordenc/WordFilter#1` | 23 | 8 |
| `wordenc/WordFilter#2` | 15 | 7 |
| `OnDemand` | 23 | 7 |
| `ObjType` | 14 | 7 |
| `GameShell` | 35 | 11 |
| `BZip2` | 8 | 8 |
| `CollisionMap` | 13 | 1 |
| `Component` | 7 | 7 |
| `Pix32` | 16 | 6 |
| `LocType` | 9 | 4 |
| `SignLink+Midi` | 34 | 18 |
| `ClientPlayer` | 5 | 3 |
| `PixFont` | 15 | 5 |
| `Packet+Isaac+Jagfile` | 40 | 5 |
| `NpcType` | 7 | 5 |
| `InputTracking` | 14 | 3 |
| `Ground` | 1 | 3 |
| `Sound` | 13 | 5 |
| `Pix2D+Pix8` | 20 | 6 |
| `EntityA` | 9 | 7 |
| `EntityB` | 12 | 6 |
| `ConfigSmall` | 18 | 5 |
| `Datastruct` | 31 | 7 |
| `Dash3dSmall` | 7 | 3 |
| `WordPack+PixMap` | 11 | 5 |
| `ClientStream` | 8 | 8 |

### Go-side reverse coverage (112 files, 0 suspicious)

- maps-to-java: 70 files
- go-original-justified: 42 files:
  - `cmd/client/main.go` — Standalone CLI entry seam (build/launch tooling). Replaces the Java applet's positional-args + getCodeBase() boot. Parses -node-id/-mem/-world-type/-world-server/-ondemand-server f…
  - `cmd/client/ondemand.go` — Transport/config seam: parseOndemandServer validates the -ondemand-server [http|https]://host:port URL for the cache server base. Pure flag parsing; no game logic. The OnDemand tra…
  - `cmd/client/profile_js.go` — Dev tooling seam, gated behind //go:build js && goscapedebug. Installs browser-console pprof/heap hooks (goscapeMemStats/goscapeDumpAllocs). Absent from release artifacts; cannot a…
  - `cmd/client/worldserver.go` — Transport seam: parseWorldServer validates the -world-server [tcp|ws|wss]://host:port URL and selects clientextras.TransportKind. Pure URL parsing; the WS scheme is a documented Go…
  - `cmd/wasmserve/main.go` — Build/dev tooling: a local static+reverse-proxy server for browser testing (serves main.wasm with application/wasm, proxies cache fetches to the data backend for single-origin). No…
  - `pkg/jagex2/client/codebase_js.go` — Codebase/transport seam (//go:build js): codeBaseURL returns window.location.origin for same-origin browser cache fetches, mirroring Java getCodeBase() document base. Pure environm…
  - `pkg/jagex2/client/codebase_native.go` — Codebase/transport seam (//go:build !js): codeBaseURL returns clientextras.OndemandBaseURL (default http://127.0.0.1:8888, mirroring client.java:7624). Endpoint configuration only;…
  - `pkg/jagex2/client/devmode.go` — Dev tooling seam: developerMode/examineIDSuffix append config-type ids to Examine menu text when DEVELOPER_MODE=true. Header documents it as the LostCityRS TS-client DEV_CLIENT add…
  - `pkg/jagex2/client/sign/signlink/signlink_socket.go` — Transport seam: resolveWSTarget derives the WebSocket target from window.location (build-neutral, unit-tested). Pure transport-target computation mirroring Client-TS ClientStream.t…
  - `pkg/jagex2/client/sign/signlink/signlink_socket_js.go` — Transport seam (//go:build js): dialTCP fails loudly because browsers lack raw sockets. Connection plumbing only.
  - `pkg/jagex2/client/sign/signlink/signlink_socket_native.go` — Transport seam (//go:build !js): dialTCP wraps net.DialTimeout for the Java-parity raw-socket transport. Connection plumbing only.
  - `pkg/jagex2/client/sign/signlink/signlink_url_js.go` — Transport seam (//go:build js): urlBase returns window.location.origin for same-origin OpenURL fetches. Environment derivation only.
  - `pkg/jagex2/client/sign/signlink/signlink_url_native.go` — Transport seam (//go:build !js): urlBase returns clientextras.OndemandBaseURL for native OpenURL fetches (mirrors client.java:7624). Endpoint config only.
  - `pkg/jagex2/client/sign/signlink/signlink_ws.go` — Transport seam: buildWSURL/WebSocket dial via coder/websocket for the ws|wss game transport. Documented Go-original standalone extension (Java applet was raw-socket only); pure con…
  - `pkg/jagex2/client/sign/signlink/storage.go` — Storage seam: cacheStore interface abstracting signlink persistence (disk native / mem+IndexedDB browser). Interface declaration only; no game logic.
  - `pkg/jagex2/client/sign/signlink/storage_disk.go` — Storage seam (//go:build !js): diskStore preserves the Java file-store (.file_store_32) behavior. Persistence backend; faithfully mirrors Java FileStore semantics, no game logic.
  - `pkg/jagex2/client/sign/signlink/storage_idb_js.go` — Storage seam (//go:build js): IndexedDB-backed cacheStore so browser cache survives reloads. Persistence backend; no game logic.
  - `pkg/jagex2/client/sign/signlink/storage_js.go` — Storage seam (//go:build js): newCacheStore selects the IndexedDB store (falls back to mem). One-line factory; no game logic.
  - `pkg/jagex2/client/sign/signlink/storage_mem.go` — Storage seam: in-RAM cacheStore for the browser build; browserUID=1337 matches the TS reference client's fixed uid (Client-TS Client.ts:1729). Persistence backend / parity constant…
  - `pkg/jagex2/graphics/bootfont/bootfont.go` — Boot-font rendering seam: wraps x/image basicfont Face7x13 for boot-phase text (DrawProgress) before the RuneScape pixfont cache loads. Renders monospace text only; no game logic, …
  - `pkg/jagex2/graphics/errorfont/errorfont.go` — Error-font rendering seam: x/image gobold opentype face approximating the Helvetica BOLD Java used on DrawError screens, available even before pixfont loads (fixes nil *PixFont der…
  - `pkg/jagex2/platform/backend_glfw.go` — Platform seam (//go:build !js): GLFW+go-gl window/GL-texture backend. Replaces Java AWT/ViewBox windowing; GPU/window plumbing only, no game logic (per platform.go seam doc).
  - `pkg/jagex2/platform/backend_webgl.go` — Platform seam (//go:build js): syscall/js + WebGL backend (texSubImage2D uploads). Browser windowing/GPU plumbing only; no game logic.
  - `pkg/jagex2/platform/platform.go` — Platform seam: toolkit-neutral Backend interface + neutral input events replacing Gio/AWT. Interface/seam declaration; no game logic.
  - `pkg/jagex2/platform/platformtest/backend.go` — Test/dev tooling: no-op platform.Backend so PixMap unit tests run without a GPU context. Test infrastructure only.
  - `pkg/jagex2/platform/run_js.go` — Platform seam (//go:build js): Main builds the browser backend and runs the loop goroutine yielding to the JS event loop. Loop/threading plumbing; no game logic.
  - `pkg/jagex2/platform/run_native.go` — Platform seam (//go:build !js): Main locks the OS thread and runs the loop on it for GLFW/GL thread-affinity. Loop/threading plumbing; no game logic.
  - `pkg/jagex2/sound/audio/audio_js.go` — Audio backend seam (//go:build js): Web Audio playback consumer. Supplies the signlink-audio consumer half that the LostCityRS Java repo never ported (it lived in the signed-applet…
  - `pkg/jagex2/sound/audio/audio_native.go` — Audio backend seam (//go:build !js): native (oto) playback consumer of signlink.midi/wave. Header documents it as the missing signlink-wrapper consumer half. Output plumbing; no ga…
  - `pkg/jagex2/sound/audio/format.go` — Audio backend seam: sample-rate/channel constants + volumeFromCentibels (10^(dB/20), matching TS tinymidipcm). Playback-format helpers, not game logic; the synthesis lives in the p…
  - `pkg/jagex2/sound/audio/midi_js.go` — Audio backend seam (//go:build js): MIDI playback via meltysynth + Web Audio. Output-device consumer plumbing; no game logic.
  - `pkg/jagex2/sound/audio/midi_native.go` — Audio backend seam (//go:build !js): MIDI playback via meltysynth + oto. Output plumbing; no game logic.
  - `pkg/jagex2/sound/audio/render.go` — Audio backend seam: renderFrameCount/release-tail helpers for pre-rendering tracks to PCM buffers. Playback-pipeline timing; no game logic.
  - `pkg/jagex2/sound/audio/soundfont.go` — Audio backend seam: loads the SF2 soundfont (via signlink cache) for the meltysynth MIDI consumer. Asset-loading plumbing for the playback backend; no game logic.
  - `pkg/jagex2/sound/audio/wave_js.go` — Audio backend seam (//go:build js): SFX playback consumer of signlink.wave via Web Audio. Output plumbing; no game logic.
  - `pkg/jagex2/sound/audio/wave_native.go` — Audio backend seam (//go:build !js): SFX playback consumer of signlink.wave via oto. Output plumbing; no game logic.
  - `pkg/jagex2/sound/audio/wavparse.go` — Audio backend seam: parseWave8Mono validates the RIFF/WAV header emitted by the ported sound/wave.GetWave and extracts raw 8-bit PCM for the playback device. Container parsing for …
  - `pkg/jagex2/sound/audio/webaudio_js.go` — Audio backend seam (//go:build js): f32ToJSFloat32Array bulk-copy helper for Web Audio buffer uploads. JS-interop plumbing; no game logic.
  - `pkg/profiling/profiling.go` — Profiling seam: in-process CPU/heap profile capture (platform-neutral half). Per design doc 2026-05-22; pure perf tooling, no game logic.
  - `pkg/profiling/profiling_signal.go` — Profiling seam (//go:build unix): SIGUSR1-triggered profile capture. Perf tooling; no game logic.
  - `pkg/profiling/profiling_stub.go` — Profiling seam (//go:build !unix): no-op Start for js/wasm and Windows. Perf-tooling stub; no game logic.
  - `pkg/util/build/build.go` — Build tooling: ldflags-injected version/revision/branch/build metadata. Build-info reporting only; no game logic.
