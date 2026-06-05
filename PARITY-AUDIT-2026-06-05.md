# rev-254 Parity Audit — 2026-06-05

Exhaustive line-by-line, side-by-side function walk of the Go client (branch `rev-254`,
HEAD `47cb2db`) against the Java 254 reference (`Client-Java` git `2e62978`), modeled on
the rev-225, rev-244, and rev-245.2 audits (PARITY-AUDIT-2026-05-28.md,
PARITY-AUDIT-2026-06-03.md, PARITY-AUDIT-2026-06-04.md).

## Method

- 50 audit units covering **every** Java file under `src/main/java` at `2e62978`
  (large files chunked by declaration-line windows: Client.java ×13, Pix3D/World/Model ×3,
  ClientBuild/WordFilter ×2; the rest singly or bundled; plus one dedicated exhaustive
  wire-opcode unit covering Protocol.java's 257+256-entry tables and every
  pIsaac/ptype/zone/menu opcode, non-diff-based; plus 3 reverse-coverage units).
  **805 Java methods walked** statement by statement: packet read widths; operator
  grouping; branch polarity; loop bounds/direction; argument semantics traced through
  callee bodies (deob param scramble); 32-bit wrap / sign-extension / shift-mask
  semantics; side-effect ordering. Comments were not trusted; claimed equivalences were
  re-verified against cited code.
- Every blocker/bug/latent finding was independently **adversarially verified** by a
  skeptic agent primed with the known false-positive classes (deob arg scramble, the
  full compensated-pairs list, Java implicit shift-count masking,
  restructured-but-equivalent control flow, behavior-lives-elsewhere, documented seams).
  22 routed → **20 confirmed, 2 refuted**. The two wire-surface defects were each found
  *twice independently* (forward window walk + wire-opcode unit), so the 20
  confirmations describe **18 distinct defects**.
- Reverse coverage: all 176 Go files classified; **0 suspicious**
  (port / seam-justified / test / tooling; reverse-3 reconciled the whole repo).
- 72 agents, ~6.3M tokens, single workflow run (`wf_4d44c668-433`). Unit evidence
  reports in `audit-254/units/` (untracked scratch, like `audit-225/` and `audit-245/`).

## Verdict summary

| Final severity | Distinct | Confirmations |
|---|---|---|
| **Blocker** (crash / protocol or cache-format desync) | 2 | 3 |
| **Bug** (wrong behavior or rendering) | 3 | 4 |
| **Latent** (edge-value / not-yet-reachable) | 13 | 13 |
| Cosmetic (comments/naming/dead code; unverified) | 54 | — |
| Intentional (documented deviations/seams) | 11 | — |
| Methods missing in Go | 0 | — |
| Refuted by verification | 2 | — |

Fix order: blockers first (one is a network-reachable unrecovered crash on a routine
opcode, the other a crash on inventory Examine), then bugs, then latent.
This document is a snapshot — the fix pass lands on top of it; live status belongs in
resume notes, never here.

---

## Blockers (2 distinct / 3 confirmations)

### client-08-01. IF_SETANIM (ptype 95) reads unsigned g2 instead of signed g2b and omits the seq-reset block

- **Unit:** `client-08`  **Java:** `Client.java:7007-7019 @2e62978 (ptype 95 IF_SETANIM)`  **Go:** `pkg/jagex2/client/client.go:10982-10988 (SERVERPROT_IF_SETANIM handler)`
- Independently re-found by the wire-opcode unit as `wire-01` (cross-confirmation; both verdicts kept).

Java: `int var64 = this.in.g2b();` (SIGNED) then `var65.modelAnim = var64; if (var64 == -1) { var65.seqFrame = 0; var65.seqCycle = 0; }`. Go: `var4 := c.In.G2()` (UNSIGNED) then `iftype.List[var26].Anim = var4` with NO `if var4 == -1` reset block. TWO divergences: (a) g2b vs G2 signedness — wire 0xFFFF yields -1 in Java but 65535 in Go; (b) the seqFrame/seqCycle reset-to-0 on clear is absent. Both are 254 deltas: 245.2 (ptype 69) used g2() with no reset block (Java 176a85f:7152-7158; Go rev-245.2 client.go:10513), the rev-254 delta-port carried 245.2 forward unchanged. Packet size index 95 = 4 (protocol.go), and Go's G2+G2 also consumes 4 bytes, so no desync. Reachability: server sends IF_SETANIM with -1 to clear a component's animation (normal op). Go stores Anim=65535; UpdateInterfaceAnimation (client.go:8730-8739) then passes `var7.Anim != -1`, computes var9=65535, and indexes `seqtype.List[65535]` -> out-of-range panic caught by the TcpIn T2 recover -> logout (or wrong-anim render). IfType has SeqFrame/SeqCycle fields (iftype.go:31-32) so the reset is portable.

> **Verifier (confirmed, severity blocker):** Read Java ptype 95 IF_SETANIM at Client.java:7007-7019 @2e62978: `int var63 = this.in.g2(); int var64 = this.in.g2b(); IfType var65 = IfType.list[var63]; var65.modelAnim = var64; if (var64 == -1) { var65.seqFrame = 0; var65.seqCycle = 0; }`. So the SECOND read is SIGNED (g2b) and there IS a seq-reset-on-clear block. Read the Go handler at client.go:10982-10988 (SERVERPROT_IF_SETANIM, serverprot.go:24 = 95): `var26 := c.In.G2(); var4 := c.In.G2(); iftype.List[var26].Anim = var4` — second read is UNSIGNED G2 and there is NO reset block. Confirmed both divergences. (a) Signedness: packet.go:218-221 G2() returns unsigned ((hi<<8)+lo, 0xFFFF->65535); packet.go:223-229 G2B() exists and subtracts 65536 when >32767 (0xFFFF->-1). The handler calls the wrong method — a genuine mistranslation, G2B() is available and is the correct port of g2b(). (b) Missing reset: SeqFrame/SeqCycle fields exist (iftype.go:31-32) so the block is portable but absent. Field-name map verified: Java modelAnim == Go Anim (header read IfType.java:368-370 == iftype.go:295-297; Anim declared iftype.go:68). Reachability verified end-to-end: server sending -1 to clear an anim (normal op) makes Go store Anim=65535; UpdateInterfaceAnimation (client.go:8730-8739) guards `var7.Anim != -1` — exactly mirroring Java updateInterfaceAnimation `var8.modelAnim != -1` (Client.java:10161) — so 65535 passes the guard, line 8736 sets var9=65535, line 8738 `var9 != -1` passes, line 8739 indexes seqtype.List[65535] -> out-of-range panic. No protocol desync: SERVERPROT_SIZES[95]=4 (protocol.go:15) and G2+G2 consumes exactly 4 bytes. Checked all FP classes: not a deob arg-scramble (single linear read, no swapped args), not a compensated pair (no caller compensation — value is stored raw and consumed raw), not a name-inversion (modelAnim<->Anim is a same-polarity rename), not restructured-equivalent (concrete input 0xFFFF diverges: -1 vs 65535, and a side effect — the seqFrame/seqCycle reset — is dropped), not behavior-elsewhere (grep shows the only SERVERPROT_IF_SETANIM handler is here and it does not reset), not shift-mask, and the // Java cite (opcode 69 / Client.java:7152) is a stale 245.2-vintage comment but the behavioral evidence is the 254 Java at 7007-7019. Finding is real on both counts.
>
> **Refined:** IF_SETANIM (SERVERPROT opcode 95) is mistranslated on two counts in the Go port. Java Client.java:7007-7019 @2e62978: the second field is read SIGNED via `this.in.g2b()` into var64, assigned `var65.modelAnim = var64`, and followed by a clear-reset block `if (var64 == -1) { var65.seqFrame = 0; var65.seqCycle = 0; }`. Go client.go:10982-10988 reads it UNSIGNED via `c.In.G2()` (packet.go:218-221) and stores `iftype.List[var26].Anim = var4` with NO reset block. (a) Signedness: when the server sends 0xFFFF to clear an animation (a normal op), Java stores modelAnim = -1 but Go stores Anim = 65535. The correct port is `c.In.G2B()` which already exists at packet.go:223-229 and yields -1. (b) The seqFrame/seqCycle reset-on-clear is dropped; the target fields exist (iftype.go SeqFrame:31, SeqCycle:32) so the block is portable. Consequence (crash, reachable in normal play): UpdateInterfaceAnimation (client.go:8730-8739, mirroring Java updateInterfaceAnimation at Client.java:10161) guards `var7.Anim != -1`; the stored 65535 passes the guard, sets var9 = 65535, and indexes seqtype.List[65535] at client.go:8739 -> slice out-of-range panic (caught by the TcpIn recover -> logout). No byte-count desync: SERVERPROT_SIZES[95] = 4 (protocol.go:15) and G2+G2 consumes 4 bytes. Fix: change the second read to `c.In.G2B()` and add the `if var4 == -1 { iftype.List[var26].SeqFrame = 0; iftype.List[var26].SeqCycle = 0 }` reset block. Note: the // Java comment on the Go handler (opcode 69 / Client.java:7152) is a stale 245.2-vintage cite; the authoritative 254 source is Client.java:7007-7019.

---

### client-11-01. Inventory Examine pusher writes wrong menu params (paramC=stack-count, paramB unset); action-1328 dispatcher panics / reads wrong interface

- **Unit:** `client-11`  **Java:** `Client.java:10085-10090 (pusher) and Client.java:8763-8774 (dispatcher) @2e62978`  **Go:** `pkg/jagex2/client/client.go:1929-1933 (pusher) vs pkg/jagex2/client/client.go:5096-5104 (dispatcher)`

Half-applied WS7 change. Java handleComponentInput pushes the inventory Examine entry (action 1328) with menuParamA=var22.id (obj id), menuParamB=var17 (inventory slot), menuParamC=var12.id (inventory component id). The Java dispatcher useMenuOption@8763 reads them back as: ObjType.get(paramA), IfType.list[paramC], IfType.list[paramC].linkObjCount[paramB]. The Go DISPATCHER was correctly updated to this Java-254 convention (client.go:5097-5103: objtype.Get(var6); var19If := iftype.List[var4]; var19If.InvSlotObjCount[var3]) — matching LOGIC-DELTA-SCOPE-254.md:390-392. But the Go PUSHER (client.go:1931-1932) still uses the OLD convention: MenuParamA=var18.Index (OK), MenuParamC=var12.InvSlotObjCount[var23] (the STACK COUNT, not a component id), and never sets MenuParamB (leaves a stale value). git: original port (0717143) used this old pusher convention with old action 1773 and old dispatcher 'if (var4 >= 100000)'; commit 2281be6 renumbered 1773->1328 and rewrote the dispatcher to iftype.List[var4].InvSlotObjCount[var3] but did not update the pusher. Reachability: right-click 'Examine' on any inventory item (type-2 component slot with InvSlotObjId[slot]>0). On selecting Examine, UseMenuOption 1328 does iftype.List[var4] with var4=stack count: if count>=len(iftype.List) -> panic index-out-of-range (crash; large coin/arrow stacks easily exceed the few-thousand interface list); if count<len, returns an unrelated/nil interface; then var19If.InvSlotObjCount[var3] with var3=stale paramB dereferences a nil slice (InvSlotObjCount is nil for any non-type-2 component) -> panic. Even count=1 (non-stackable) does iftype.List[1].InvSlotObjCount[staleParamB] and crashes unless interface 1 happens to be a type-2 inventory with a long-enough count array. Net: Examine on a normal inventory item crashes the client. Fix: pusher should set MenuParamA=var18.Index, MenuParamB=var23 (slot), MenuParamC=var12.Id (component id) to match the already-correct dispatcher.

> **Verifier (confirmed, severity blocker):** Traced both sides with line numbers. Param bindings in the dispatcher are identical Java/Go: var3=menuParamB, var4=menuParamC, var6=menuParamA (Java Client.java:8576-8579 @2e62978; Go client.go:4880-4883). 
> 
> JAVA convention (authoritative): pusher Client.java:10085-10090 sets the Examine(1328) entry menuParamA=var22.id (obj id), menuParamB=var17 (inventory slot), menuParamC=var12.id (component id) — exactly as every sibling entry in the same loop (10001-10003, 10017-10019, 10078-10080, etc.). Dispatcher Client.java:8763-8774 reads them back: ObjType.get(var6=paramA), IfType.list[var4=paramC], var19.linkObjCount[var3=paramB]. Internally consistent.
> 
> GO dispatcher (client.go:5096-5110) was correctly updated to this 254 convention: var17=objtype.Get(var6=paramA); var19If=iftype.List[var4=paramC]; var19If.InvSlotObjCount[var3=paramB]. Matches Java and matches LOGIC-DELTA-SCOPE-254.md:390-392 ("obj-examine count source now IfType.list[var4].linkObjCount[slot] with nil guard").
> 
> GO pusher (client.go:1929-1933) was NOT updated — it still uses the OLD convention: MenuParamA=var18.Index (obj id, OK), MenuParamC=var12.InvSlotObjCount[var23] (the STACK COUNT, should be var12.Id component id), and never assigns MenuParamB at all (Java sets var17=slot). Confirmed via git: 0717143 introduced this old-convention pusher with action 1773 and an old dispatcher "if var4 >= 100000"; commit 2281be6 renumbered 1773->1328 and rewrote the dispatcher to iftype.Instances[var4].InvSlotObjCount[var3] (later renamed List) but the diff shows it left MenuParamA/MenuParamC unchanged and never added MenuParamB — a half-applied WS7 change. The Examine entry at line 1850's else-branch is pushed UNCONDITIONALLY, so MenuParamB[MenuSize] retains a stale value from a prior menu entry/frame (slice is make([]int,500) reused each rebuild).
> 
> Verified the dispatcher's nil guard does NOT prevent the crash: iftype.List[var4] (client.go:5102) is indexed BEFORE the guard, and []*IfType out-of-range panics; if var4 (the item stack count) >= len(iftype.List) it panics outright (large coin/arrow stacks vastly exceed the few-thousand interface list). For small counts, var19If.InvSlotObjCount[var3] (client.go:5103, []int per iftype.go:30, allocated only for type-2 components at iftype.go:197/311, nil otherwise) is dereferenced when var19If!=nil — interface index 1 (count=1 non-stackable case) is not a type-2 inventory component, so InvSlotObjCount is nil -> nil-slice index panic. Either path crashes.
> 
> Reachability: right-click "Examine" on any inventory item (type-2 component, InvSlotObjId[slot]>0) and select it -> UseMenuOption(1328) -> guaranteed panic. Normal, frequent interaction. Ruled out FP classes: dispatcher matches Java exactly (not restructured-equivalent — the pusher genuinely diverges from the dispatcher's contract); AddChat(0,var18,"")==addChat("",0,var20) is the documented compensated pair, not part of this defect; examineIDSuffix is the documented devmode addition, irrelevant. Cites in the finding are all accurate.
>
> **Refined:** Inventory "Examine" (menu action 1328) crashes the client because the Go menu PUSHER feeds the dispatcher the wrong params. The dispatcher was correctly ported to the Java-254 convention but the matching pusher fix was skipped (half-applied WS7 commit 2281be6).
> 
> Go DISPATCHER pkg/jagex2/client/client.go:5096-5110 (correct, matches Java Client.java:8763-8774 @2e62978 and LOGIC-DELTA-SCOPE-254.md:390-392): with var6=MenuParamA, var4=MenuParamC, var3=MenuParamB (bindings client.go:4880-4883), it does var17=objtype.Get(var6); var19If=iftype.List[var4]; uses var19If.InvSlotObjCount[var3]. So it requires paramA=obj id, paramB=inventory slot, paramC=inventory component id.
> 
> Go PUSHER pkg/jagex2/client/client.go:1929-1933 (wrong): sets MenuAction=1328, MenuParamA=var18.Index (obj id, OK), MenuParamC=var12.InvSlotObjCount[var23] (the item STACK COUNT, not a component id), and never sets MenuParamB (leaves a stale value from a prior frame/entry; MenuParamB is make([]int,500) reused each rebuild). Java pusher Client.java:10085-10090 @2e62978 instead sets menuParamA=var22.id, menuParamB=var17 (slot), menuParamC=var12.id (component id), consistent with every sibling entry in the same loop.
> 
> Crash mechanics: in UseMenuOption, var4=MenuParamC=stack count. client.go:5102 does iftype.List[var4] (iftype.List is []*IfType, iftype.go:20) BEFORE any guard, so a stack count >= len(iftype.List) panics index-out-of-range (large coin/arrow stacks easily exceed the interface list). For small counts the nil guard passes and client.go:5103 dereferences var19If.InvSlotObjCount[var3] (InvSlotObjCount is []int allocated only for type-2 components, iftype.go:30,197,311) with var3=stale paramB -> nil-slice/out-of-range panic. Even count=1 (non-stackable) hits iftype.List[1], whose InvSlotObjCount is nil for a non-type-2 interface -> panic.
> 
> Reachability: Examine on any inventory item (type-2 component slot, InvSlotObjId[slot]>0). Blocker (client crash).
> 
> Fix: in the pusher (client.go:1931-1932) set MenuParamA=var18.Index, MenuParamB=var23 (slot), MenuParamC=var12.Id (component id) to match the already-correct dispatcher and Java 254.

---

### wire-01. IF_SETANIM (opcode 95) reads g2 (unsigned) instead of g2b (signed) and omits the anim==-1 reset

- **Unit:** `wire-opcodes`  **Java:** `Client-Java @2e62978 src/main/java/jagex2/client/Client.java:7007-7018 (read(): if ptype==95 { var63=in.g2(); var64=in.g2b(); var65=IfType.list[var63]; var65.modelAnim=var64; if (var64==-1){var65.seqFrame=0; var65.seqCycle=0;} })`  **Go:** `pkg/jagex2/client/client.go:10982-10987 (if c.PacketType==SERVERPROT_IF_SETANIM { var26:=c.In.G2(); var4:=c.In.G2(); iftype.List[var26].Anim=var4; ... }); compare anim-tick consumer at client.go:8730-8739`
- Same defect as `client-08-01` — independent confirmation by a second unit.

Java reads the interface anim id with g2b() (SIGNED 16-bit), so the server's stop-animation sentinel 0xFFFF decodes to -1; Java then resets seqFrame/seqCycle when var64==-1. Go reads it with G2() (UNSIGNED), so 0xFFFF becomes 65535, and the `==-1` reset branch is entirely absent. Reachability: any server-sent IF_SETANIM with anim=-1 (the normal way to stop an interface animation). Consequence: the next interface animation tick (client.go:8730 `var7.Anim != -1` is true for 65535, 8736 `var9 = var7.Anim`, 8738 `var9 != -1` true) indexes seqtype.List[65535] (List is make([]*SeqType, Count) at seqtype.go:74, Count << 65535) → index-out-of-range panic. The panic fires in the update/draw loop, OUTSIDE TcpIn's recover (client.go:10417, which only wraps packet parse/dispatch) → client crash. Even without OOB the seqFrame/seqCycle reset is silently dropped. Byte count is unchanged (g2 and g2b both consume 2 bytes) so there is no stream desync. Contrast: MIDI_SONG at client.go:10890 correctly does the 65535->-1 conversion; IF_SETANIM does not. Fix: read G2B() (or convert 65535->-1) and add the seqFrame=0/seqCycle=0 reset on -1. Width sequences for all other 59 single-opcode read() blocks plus all 10 zone opcodes were diffed and are identical; this is the only g2-vs-g2b mismatch.

> **Verifier (confirmed, severity blocker):** Read both sides with line numbers and traced the full path. JAVA (git show 2e62978:src/main/java/jagex2/client/Client.java:7007-7018): opcode 95 IF_SETANIM reads `var63 = in.g2()` (id, unsigned), `var64 = in.g2b()` (anim, SIGNED 16-bit), sets `IfType.list[var63].modelAnim = var64`, then `if (var64 == -1) { seqFrame=0; seqCycle=0; }`. GO (pkg/jagex2/client/client.go:10982-10987): SERVERPROT_IF_SETANIM (=95, serverprot.go:24) reads `var26 := c.In.G2()` (id), `var4 := c.In.G2()` (anim, UNSIGNED — the bug), sets `iftype.List[var26].Anim = var4`, with NO `== -1` reset branch. Verified packet.go:218-230: G2 returns unsigned (0xFFFF→65535) while G2B sign-extends (0xFFFF→-1), so the stop-animation sentinel 0xFFFF decodes to 65535 in Go instead of -1. Verified IfType field is `Anim` (iftype.go:68) — correct field, not a name inversion. Traced the consumer UpdateInterfaceAnimation (client.go:8718-8755): at 8730 `var7.Anim != -1` is true for 65535; 8736 `var9 = var7.Anim` = 65535; 8738 `var9 != -1` true; 8739 `seqtype.List[var9]` indexes a slice allocated `make([]*SeqType, Count)` (seqtype.go:74) where Count is a 2-byte config count (≪65535) → index-out-of-range panic. Verified the callers of UpdateInterfaceAnimation are in DrawGame() (4613, 4642) and OtherOverlays() (6692, 6696) — the update/render loop, NOT TcpIn — so the panic fires OUTSIDE the TcpIn recover at client.go:10417 (which only wraps the read/dispatch body) → client crash. Byte count is identical (g2 and g2b both consume 2 bytes; size table entry 95→4 in LOGIC-DELTA-SCOPE-254.md:165 unchanged) → no stream desync. Contrast confirmed: MIDI_SONG (client.go:10889-10893) and IF_SETTAB (10990-10995) both apply the explicit `if x == 65535 { x = -1 }` sentinel conversion; IF_SETANIM omits it. Checked LOGIC-DELTA-SCOPE-254.md and PORT-DESIGN-254.md — IF_SETANIM appears only in the opcode/size table, no documented deviation. Ruled out all FP classes: not deob arg scramble (id/anim flow direct and matches Java), not a compensated pair, not name inversion (Anim correct), not equivalent control flow (concrete input 0xFFFF diverges 65535 vs -1 AND the seqFrame/seqCycle reset side effect is dropped), not behavior-elsewhere (no conversion anywhere on this path), not shift-masking. Finder's cites are all accurate.
>
> **Refined:** CONFIRMED (blocker). IF_SETANIM (opcode 95) reads the interface animation id with the wrong width and omits the stop-animation reset. Java @2e62978 Client.java:7007-7018: `int var63 = in.g2(); int var64 = in.g2b(); IfType var65 = IfType.list[var63]; var65.modelAnim = var64; if (var64 == -1) { var65.seqFrame = 0; var65.seqCycle = 0; }` — anim is read with g2b() (SIGNED), so the server's 0xFFFF stop sentinel decodes to -1, and seqFrame/seqCycle are reset. Go pkg/jagex2/client/client.go:10982-10987: `if c.PacketType == SERVERPROT_IF_SETANIM { var26 := c.In.G2(); var4 := c.In.G2(); iftype.List[var26].Anim = var4; c.PacketType = -1; return true }` — anim is read with G2() (UNSIGNED, packet.go:218-221 vs G2B at 223-229), so 0xFFFF becomes 65535, and the `== -1` reset branch is entirely absent. Reachability: any server-sent IF_SETANIM with anim=-1 (the normal way to stop an interface animation). Consequence: on the next interface-animation tick in UpdateInterfaceAnimation (client.go:8730 `var7.Anim != -1` true for 65535 → 8736 `var9 = var7.Anim` = 65535 → 8738 `var9 != -1` true → 8739 `seqtype.List[65535]`), it indexes a slice of length Count (seqtype.go:74 `make([]*SeqType, Count)`, Count ≪ 65535) → index-out-of-range panic. That tick runs from DrawGame() (client.go:4613, 4642) and OtherOverlays() (6692, 6696) in the render/update loop, OUTSIDE the TcpIn recover (client.go:10417 wraps only packet read/dispatch) → unrecovered client crash. Even absent the OOB, the seqFrame/seqCycle reset is silently dropped. Byte count unchanged (g2/g2b both 2 bytes; size-table entry 95→4 unchanged in LOGIC-DELTA-SCOPE-254.md:165) → no stream desync. Contrast: MIDI_SONG (client.go:10889-10893) and IF_SETTAB (10990-10995) both apply `if x == 65535 { x = -1 }`; IF_SETANIM does not. Fix: read G2B() (or convert 65535→-1) for var4, and add `if var4 == -1 { iftype.List[var26].SeqFrame = 0; iftype.List[var26].SeqCycle = 0 }`. Not documented as a deviation in LOGIC-DELTA-SCOPE-254.md or PORT-DESIGN-254.md.


---

## Bugs (3 distinct / 4 confirmations)

### client-08-02. IF_OPENOVERLAY (ptype 85) omits the resetInterfaceAnimation call

- **Unit:** `client-08`  **Java:** `Client.java:7279-7288 @2e62978 (ptype 85 IF_OPENOVERLAY)`  **Go:** `pkg/jagex2/client/client.go:10917-10923 (SERVERPROT_IF_OPENOVERLAY handler)`

Java: `int var103 = this.in.g2b(); if (var103 >= 0) { this.resetInterfaceAnimation(var103); } this.mainOverlayLayerId = var103;`. Go reads `com := c.In.G2B()` and sets `c.MainOverlayLayerID = com` but NEVER calls `c.ResetInterfaceAnimation(com)` for com>=0. 254 delta: 245.2 (ptype 115) had no reset call (Java 176a85f:7982-7988; Go rev-245.2), the rev-254 port missed the added guard+call. g2b read and byte count (size index 85 = 2) are correct, so no desync. Reachability: when the server opens an overlay interface containing animated (type-6) components, Java resets their seqFrame/seqCycle to 0 before display; Go leaves stale animation state so animated overlay components render from the wrong frame. c.ResetInterfaceAnimation exists and is used by the sibling IF_OPENMAIN/IF_OPENSIDE/IF_OPENCHAT handlers (client.go:8776, 11150, 11170, 11334).

> **Verifier (confirmed, severity bug):** Read both sides with line numbers. Java (git show 2e62978:Client.java | nl -ba, lines 7279-7288): the ptype==85 IF_OPENOVERLAY handler does `int var103 = this.in.g2b(); if (var103 >= 0) { this.resetInterfaceAnimation(var103); } this.mainOverlayLayerId = var103;`. Go (client.go:10917-10923, SERVERPROT_IF_OPENOVERLAY handler): reads `com := c.In.G2B()` and sets `c.MainOverlayLayerID = com` with NO ResetInterfaceAnimation call and no >=0 guard. Confirmed opcode identity: serverprot.go:18 defines SERVERPROT_IF_OPENOVERLAY = 85, matching Java ptype 85 (the `// Java: opcode 115 ... Client.java:7982` comment is a stale 245.2 cite — cosmetic, value 85 is correct). Ruled out false-positive classes: (1) behavior-lives-elsewhere — grepped all MainOverlayLayerID uses (599/626/6691-6693/7445/10920); the setter at 10920 is a plain field assignment with no hidden reset; the only resets are -1 at 7445 (login) and the render reads at 6691-6693. No reset exists anywhere in the IF_OPENOVERLAY path in Go. (2) no desync — protocol.go:15 SERVERPROT_SIZES index 85 = 2 (computed), matching G2B (packet.go:223-230, advances Pos by 2, signed). Both Java g2b and Go G2B read exactly 2 bytes; the missing reset does not change byte count. (3) ResetInterfaceAnimation exists and works (client.go:8776-8786, recursively zeroes SeqFrame/SeqCycle for type-1 children) and is used by the sibling IF_OPENMAIN/IF_OPENSIDE/IF_OPENCHAT handlers (8781, 11150, 11170, 11334), confirming it is the correct call. Java siblings IF_OPENCHAT(141)/IF_OPENMAIN(197) at 6563/6579 call it unconditionally on a g2() read; IF_OPENOVERLAY(85) uses the >=0 guard because g2b is signed — both omitted in Go. Render impact is real: DrawScene (6691-6692) calls UpdateInterfaceAnimation(MainOverlayLayerID) each frame, which advances SeqFrame/SeqCycle from whatever stale value they hold, so animated (type-6) overlay components render from the wrong frame. This is a 254 delta (245.2 ptype 115 had no reset; the rev-254 port missed the added guard+call).
>
> **Refined:** IF_OPENOVERLAY (opcode 85) handler omits the resetInterfaceAnimation call. Java (Client.java:7279-7288 @2e62978): `if (this.ptype == 85) { int var103 = this.in.g2b(); if (var103 >= 0) { this.resetInterfaceAnimation(var103); } this.mainOverlayLayerId = var103; ... }`. Go (pkg/jagex2/client/client.go:10917-10923): reads `com := c.In.G2B()` and assigns `c.MainOverlayLayerID = com` but never calls `c.ResetInterfaceAnimation(com)` for com>=0. SERVERPROT_IF_OPENOVERLAY = 85 (serverprot.go:18) correctly matches Java ptype 85; the `// Java: opcode 115 (Client.java:7982)` comment is a stale 245.2 cite. No desync: SERVERPROT_SIZES[85] = 2 (protocol.go:15) matches the 2-byte signed G2B read (packet.go:223-230). C.ResetInterfaceAnimation exists (client.go:8776, zeroes SeqFrame/SeqCycle of type-1 children) and is used by sibling open-interface handlers (8781, 11150, 11170, 11334). Effect: when the server opens an overlay interface (rendered each frame via UpdateInterfaceAnimation at client.go:6692), animated overlay components keep stale seqFrame/seqCycle and render from the wrong animation frame. This is a rev-254 delta the port missed (245.2 had no reset). Fix: add `if com >= 0 { c.ResetInterfaceAnimation(com) }` before the MainOverlayLayerID assignment at client.go:10920.

---

### client-08-04. zonePacket MAP_PROJANIM (arg0==37) missing `* 4` on the two g1 projectile-height reads

- **Unit:** `client-08`  **Java:** `Client.java:7682-7694 @2e62978 (zonePacket MAP_PROJANIM)`  **Go:** `pkg/jagex2/client/client.go:1332-1344 (ZonePacket SERVERPROT_MAP_PROJANIM branch)`
- Independently re-found by the wire-opcode unit as `wire-02` (cross-confirmation; both verdicts kept).

Java: `int var48 = arg1.g1() * 4;` and `int var49 = arg1.g1() * 4;`. var48 is used as `getAvH(...) - var48` (the ctor srcY term); var49 is the ctor's dstHeight/OffsetY arg AND `getAvH(...) - var49` in setTarget. Go reads `var11 = arg1.G1()` and `var36 = arg1.G1()` with NO `* 4`, so both are 1/4 of Java. Arg mapping verified through NewClientProj (clientproj.go:39) and UpdateVelocity (clientproj.go:56): Go offsetY=var36->OffsetY(dstHeight); Go ctor srcY = GetHeightMapY(level,var5,var6)-var11; Go UpdateVelocity arg0 = GetHeightMapY(level,var7,var8)-var36. 254 delta: 245.2 Java had `int srcHeight = buf.g1(); int dstHeight = buf.g1();` with no `* 4` (Java 176a85f:8242-8243) and 245.2 Go was correct for 245.2; the 254 engine added the `* 4` and the rev-254 delta-port missed it (rev-245.2 and rev-254 Go both read plain G1). Byte count (15, scope doc / SERVERPROT_SIZES[37]) is unaffected so no desync. Reachability: every server projectile (spells, arrows, etc.) — source and destination Y heights are 1/4 the intended offset, so projectiles render at the wrong vertical position. Normal play.

> **Verifier (confirmed, severity bug):** Read Java zonePacket MAP_PROJANIM (arg0==37) at Client.java:7673-7696 @2e62978. Lines 7682-7683: `int var48 = arg1.g1() * 4;` and `int var49 = arg1.g1() * 4;`. Read the ctor call (7693): new ClientProj(minusedlevel, var47, var50+loopCycle, var51+loopCycle, getAvH(var55,...,var54)-var48, var49, var54, var52, var46, var55, var53). Mapped against the Java ClientProj ctor (dash3d/ClientProj.java:81-94 @2e62978; the path is dash3d/ClientProj.java not dash3d/entity/): arg4=`getAvH(...)-var48` -> srcY (line 86); arg5=`var49` -> dstHeight (line 92). The setTarget call (7694) passes `getAvH(var57,...,var56)-var49` as arg1, which feeds accelerationY at ClientProj.java:113.
> 
> Read Go ZonePacket SERVERPROT_MAP_PROJANIM at client.go:1324-1346. Reads: var11 = arg1.G1() (1332) and var36 = arg1.G1() (1333) — NO `* 4`. Traced consumers: ctor NewClientProj (clientproj.go:39) maps offsetY->OffsetY (line 51, ==Java dstHeight), srcY->SrcY (line 45). The Go ctor call (client.go:1343) passes offsetY=var36 and srcY=GetHeightMapY(...,var5,var6)-var11. UpdateVelocity (clientproj.go:56-76) arg0 -> AccelerationY (line 75); Go call (client.go:1344) passes arg0=GetHeightMapY(...,var7,var8)-var36.
> 
> Therefore Go var11 ≡ Java var48 (both feed srcY via `height - X`), and Go var36 ≡ Java var49 (both feed dstHeight/OffsetY and the setTarget/UpdateVelocity accelerationY term via `height - X`). Java multiplies both g1 reads by 4; Go does not. Confirmed via grep that var11 (declared client.go:1215) and var36 (client.go:1322) have no `* 4` applied anywhere between read and use; GetHeightMapY is an unrelated height lookup, not a compensator.
> 
> Corroboration of 254-delta claim: Java 245.2 (176a85f:8242-8243) read `srcHeight = buf.g1(); dstHeight = buf.g1();` with no `* 4`; the 254 engine (2e62978:7682-7683) added `* 4` and the rev-254 Go delta-port missed it.
> 
> False-positive classes checked and ruled out: (1) deob arg scramble — traced real flow through callee bodies, mapping holds; (2) not in compensated-pair/standing-decision DO-NOT-REPORT list; (3) no name inversion — field semantics match; (4) concrete divergence — Go heights are exactly 1/4 of Java for every projectile; (5) no `* 4` lives elsewhere (grepped whole branch); (6)/(7) N/A. Byte count is unaffected (15 bytes; `* 4` is a post-read scale, not an extra byte read) so no desync.
>
> **Refined:** CONFIRMED. zonePacket MAP_PROJANIM (arg0==37) drops the `* 4` scale on both g1 projectile-height reads in the Go port.
> 
> Java (Client.java:7682-7683 @2e62978):
>   int var48 = arg1.g1() * 4;
>   int var49 = arg1.g1() * 4;
> var48 feeds the ClientProj srcY argument as `getAvH(var55, minusedlevel, var54) - var48` (ctor call Client.java:7693 -> ClientProj.java:86). var49 is the ctor's dstHeight arg (Client.java:7693 arg5 -> ClientProj.java:92) and the setTarget arg1 term `getAvH(var57, minusedlevel, var56) - var49` (Client.java:7694 -> accelerationY at ClientProj.java:113).
> 
> Go (pkg/jagex2/client/client.go:1332-1333):
>   var11 = arg1.G1()
>   var36 = arg1.G1()
> with NO `* 4`. var11 feeds SrcY as `c.GetHeightMapY(c.CurrentLevel, var5, var6) - var11` (NewClientProj srcY arg, client.go:1343 -> clientproj.go:45). var36 is the OffsetY/dstHeight ctor arg (client.go:1343 -> clientproj.go:51) and the UpdateVelocity arg0 term `c.GetHeightMapY(c.CurrentLevel, var7, var8) - var36` (client.go:1344 -> AccelerationY at clientproj.go:75).
> 
> Mapping verified by tracing callee bodies: Go var11 ≡ Java var48, Go var36 ≡ Java var49. Result: both projectile source-Y and destination-Y height offsets in the Go port are 1/4 of the intended value, so every server-driven projectile (spells, arrows, etc.) renders at the wrong vertical position.
> 
> 254 delta: 245.2 Java (176a85f:8242-8243) had `srcHeight = buf.g1(); dstHeight = buf.g1();` (no `* 4`); the 254 engine added `* 4` and the rev-254 Go delta-port missed it.
> 
> No desync: byte count is 15 either way (the `* 4` is a post-read multiply, not an extra read; SERVERPROT_MAP_PROJANIM = 37 at serverprot.go:81).
> 
> Fix: client.go:1332-1333 should read `var11 = arg1.G1() * 4` and `var36 = arg1.G1() * 4`.

---

### pix3d-B-01. GouraudRaster divTable slope multiply lacks the int32 wrap Java relies on (analog of the FIXED textureRaster pix3d-3-03 site)

- **Unit:** `pix3d-B`  **Java:** `Pix3D.java:867 (gouraudRaster, lowDetail-true non-HClip branch): var11 = (arg7 - arg6) * divTable[var10] >> 15; with var10 = arg5 - arg4 >> 2 (line 865); var11 used as the colour-interp step arg6 += var11 in the unrolled colourTable[arg6>>8] loop.`  **Go:** `pix3d.go:885: var8 = ((arg7 - arg6) * DivTable[arg3]) >> 15 (arg3 = (arg5-arg4)>>2 at line 883), used as arg6 += var8.`

arg6/arg7 are gouraud colour endpoints. Magnitude trace: GouraudTriangle is called (Model.gouraudTriangle, e.g. Java Model.java:1806 / Go model.go:1808) with faceColourA/B/C = HSL16 colours 0..65535; gouraudTriangle sets var18 = arg6<<15 and hands gouraudRaster var18>>7 ≈ colour<<8, so an endpoint reaches up to 65535<<8 ≈ 1.677e7 ≈ 2^24, and the scanline-adjacent delta (arg7-arg6) can approach ±2^24 on steeply shaded faces. divTable[var10]=32768/var10, max 32768=2^15 when var10==1 (span 4..7 px). The product (arg7-arg6)*divTable[var10] reaches ~2^39 > 2^31. Java: this is a 32-bit int multiply that WRAPS to low 32 bits before the arithmetic >>15, so var11 takes a wrapped/possibly-negative step. Go: arg6, arg7, DivTable[...] are 64-bit int, so the product does NOT wrap; >>15 keeps the high bits and var8 differs from Java's step, then propagates via arg6 += var8 into colourTable[arg6>>8] -> wrong colour gradient on that scanline. This is the EXACT structural analog of the textureRaster slope at pix3d.go:2065, which the project DELIBERATELY wrapped: var15 = int(int32((arg8-arg7)*DivTable[var16])) >> 6, with comment '32-bit product wraps before the >>6 ... audit pix3d-3-03' (PARITY-AUDIT-2026-06-04 pix3d-3-03, severity bug, same colour<<8 x divTable>2^31 magnitude analysis). The gouraud sibling at line 885 was NOT given the matching int(int32(...)) wrap, so the texture family wraps like Java while the gouraud family does not — an inconsistency. Reachability: Jagged(lowDetail)==true path, non-HClip sub-branch (arg3>0, span>=4), colour endpoint delta large enough that |arg7-arg6|*divTable[var10] >= 2^31 (bright/high-contrast gouraud faces with short spans 4..7 -> small var10 -> large divTable). On those scanlines Java and Go pick different interpolation steps -> subtle intermittent colour banding; no crash. Classified latent (narrower trigger than pix3d-3-03: needs the lowDetail non-HClip sub-branch and short span), flagged for a consistency decision since the project chose to fix the identical class on the texture side. The HClip sub-branch (Java 846/Go 866) and lowDetail-false branch (Java 923/Go 952) use direct division (arg7-arg6)/(arg5-arg4) and are unaffected. flatTriangle's (argA-argB)<<16 gradient overflow is the accepted L45 rasterizer-family convention (X bounded by screen width ~765, Y clamped to boundBottom) and is NOT reported separately. flatTriangle is otherwise full parity across all 12 leaf rasterizing loops (var->arg accumulator/counter remapping traced).

> **Verifier (confirmed, severity bug):** Read Go GouraudRaster (pix3d.go:856-950): line 885 `var8 = ((arg7 - arg6) * DivTable[arg3]) >> 15` (arg3=(arg5-arg4)>>2 at 883), non-HClip lowDetail sub-branch, var8 used as `arg6 += var8` feeding ColourTable[arg6>>8] at 899/909/928/938. Java gouraudRaster (Pix3D.java:838-956 @2e62978): line 867 `var11 = (arg7 - arg6) * divTable[var10] >> 15` (var10 at 865). Bodies are line-identical; the only difference is operand width: Java operands are 32-bit int so the multiply wraps to low 32 bits BEFORE the arithmetic >>15; Go uses 64-bit int so it does not wrap. Confirmed finder's exact cites.
> 
> Magnitude trace: gouraudTriangle (Java 343 / Go 374) sets the colour accumulator var=arg<<15 and passes var>>7 (=arg<<8) as gouraudRaster's colour endpoints; callers Model.gouraudTriangle (Model.java:1806 / model.go:1808) pass faceColourA/B/C = HSL16 colours 0..65535. So an endpoint reaches up to 65535<<8 ≈ 16.7M ≈ 2^24; DivTable[i]=32768/i ≤ 32768 (pix3d.go:41); product ≈ 2^39 ≫ 2^31. Numeric test: Java int32-wrap vs Go 64-bit diverge whenever |arg7-arg6|·divTable ≥ 2^31, i.e. |colourΔ| ≥ 256 of a 0..65535 range on a 4-7px span (var10=1). Worst case javaStep=-256 vs goStep=16776960. The diverging step propagates via arg6 += var8 into ColourTable[arg6>>8] → different colour gradient. ColourTable size 65536; for in-range endpoints Go's un-wrapped step keeps arg6>>8 in-range (renders a smoother/different gradient, no crash); only the most extreme endpoint pair (cL=0,cR=65535) would index -1, but that AIOOBE exists in Java too, so it is not a Go-introduced crash.
> 
> This is the exact structural analog of textureRaster pix3d.go:2065, which the project DELIBERATELY wrapped: `var15 = int(int32((arg8-arg7)*DivTable[var16])) >> 6` with comment 'audit pix3d-3-03'. The gouraud sibling was NOT given the matching int(int32(...)) wrap — a real fidelity inconsistency. Checked LOGIC-DELTA-SCOPE-254.md / PORT-DESIGN-254.md: no documented exemption for this site (grep hits were unrelated menu-action table rows).
> 
> Reachability is the DEFAULT render path, not an edge: Jagged/lowDetail defaults to true (pix3d.go:15; Pix3D.java:13 lowDetail=true); it is only briefly set false for inventory icon models (objtype.go:418→499) which use HClip-direct-division anyway. Gouraud-shaded untextured faces are ubiquitous in the scene. Prior 245 audit (audit-245/units/pix3d-1.md:77) judged the gouraud 32-bit-wrap class 'bounded, not a finding' but ONLY analyzed the arg6<<15 accumulator (0x7FFF8000 < int32 max) and per-vertex deltas — it never analyzed this slope multiply, which is a distinct (endDiff)·divTable computation that DOES overflow. The texture twin pix3d-3-03 was finalized as severity BUG (PARITY-AUDIT-2026-06-04.md:112-120, verifier-upgraded from latent) using the identical shade-endpoint(~16.7M)×divTable(≤32768)>2^31 analysis. By parity, and because the gouraud path is the MORE common one (default lowDetail, untextured faces), the finder's 'latent' is an under-rating → bug.
> 
> Ruled out false-positive classes: not deob arg-scramble (traced actual arg flow through gouraudTriangle→gouraudRaster); not a compensated pair / standing decision (none listed for this site); not name-inversion; not restructured-equivalent (concrete diverging input values demonstrated numerically); not shift-count masking; cites are real code lines, not stale comments.
>
> **Refined:** CONFIRMED (severity bug). GouraudRaster's gouraud colour-interpolation slope step is computed without the int32 wrap that the authoritative Java client relies on, while the structurally identical textureRaster slope (pix3d.go:2065) was deliberately wrapped — a fidelity inconsistency within the rasterizer family.
> 
> Go: pkg/jagex2/graphics/pix3d/pix3d.go:885 `var8 = ((arg7 - arg6) * DivTable[arg3]) >> 15` (arg3 = (arg5-arg4)>>2 at line 883), in the lowDetail non-HClip sub-branch of GouraudRaster; var8 is the colour step applied as `arg6 += var8` (lines 910/939) and read as `ColourTable[arg6>>8]`. All operands are Go 64-bit int → no wrap.
> 
> Java: Pix3D.java:867 (@2e62978) `var11 = (arg7 - arg6) * divTable[var10] >> 15` (var10 = arg5-arg4 >> 2 at line 865). All operands are 32-bit int → the multiply wraps to its low 32 bits BEFORE the arithmetic >>15.
> 
> Magnitude: colour endpoints arg6/arg7 reach gouraudRaster as colour<<8 (gouraudTriangle Pix3D.java:343/Go pix3d.go:374 builds arg<<15, passes >>7; callers Model.java:1806 / model.go:1808 pass HSL16 faceColour 0..65535), so |arg7-arg6| reaches ~2^24; divTable[var10] ≤ 32768 (=32768/i, pix3d.go:41). The product reaches ~2^39 and exceeds 2^31 whenever |colourΔ| ≥ 256 (of a 0..65535 range) on a short 4..7px span (var10==1). At that point Java's wrapped step differs from Go's 64-bit step (e.g. javaStep=-256 vs goStep=16776960 for endpoints 0→65535), so `arg6 += step` accumulates differently and ColourTable[arg6>>8] selects a different colour → wrong/smoothed gradient versus the Java client. No Go crash for in-range endpoints (Go's un-wrapped step keeps arg6>>8 in [0,65535]); it simply renders a different gradient — i.e. a rendering-fidelity bug, not a blocker.
> 
> Reachability is the default scene path: pix3d.Jagged / Java lowDetail defaults true (pix3d.go:15; Pix3D.java:13), turned off only for inventory-icon models (objtype.go:418→499, which use the HClip direct-division branch anyway). Untextured gouraud-shaded faces with steep colour gradients on short scanlines are common in normal play.
> 
> Fix: mirror the texture sibling — wrap the 32-bit product before the shift, e.g. `var8 = int(int32((arg7 - arg6) * DivTable[arg3])) >> 15`, with a `// Java: 32-bit product wraps before >>15 (audit pix3d-B-01, analog of pix3d-3-03)` comment. Note the prior 245 verification note (audit-245/units/pix3d-1.md:77) that judged the gouraud 32-bit-wrap class 'bounded' covered only the arg6<<15 accumulator and per-vertex deltas, NOT this (endDiff)·divTable slope multiply — so it does not exempt this site. Severity raised from the finder's 'latent' to 'bug' by parity with PARITY-AUDIT-2026-06-04 pix3d-3-03 (finalized bug under the same magnitude analysis) and because the gouraud path is the more commonly exercised one.

---

### wire-02. MAP_PROJANIM (zone opcode 37) drops the *4 scale on the source-Y and destination-height offsets

- **Unit:** `wire-opcodes`  **Java:** `Client-Java @2e62978 src/main/java/jagex2/client/Client.java:7682-7683 (zonePacket op 37: int var48 = arg1.g1() * 4; int var49 = arg1.g1() * 4;) used at 7693 (new ClientProj(..., getAvH(var55,lvl,var54)-var48 /*arg4 srcY*/, var49 /*arg5 dstHeight*/, ...)) and 7694 (setTarget(var57, getAvH(var57,lvl,var56)-var49, ...)); ctor field map at dash3d/ClientProj.java:81-92 (srcY=arg4, dstHeight=arg5)`  **Go:** `pkg/jagex2/client/client.go:1332-1333 (var11 = arg1.G1(); var36 = arg1.G1()) used at 1343 (NewClientProj(var36 /*offsetY==Java dstHeight*/, ..., c.GetHeightMapY(...)-var11 /*srcY*/, ...)) and 1344 (var43.UpdateVelocity(c.GetHeightMapY(...)-var36, ...)); ctor at dash3d/entity/clientproj.go:39`
- Same defect as `client-08-04` — independent confirmation by a second unit.

Java multiplies BOTH g1 reads by 4: var48 (source-Y offset, subtracted to form ClientProj srcY) and var49 (destination height offset, used as ClientProj dstHeight/arg5 and subtracted in setTarget arg1). Go reads var11 and var36 as plain G1() with no *4. Arg mapping verified: Go NewClientProj signature (offsetY,peakPitch,srcZ,lastCycle,level,target,startCycle,arc,srcY,arg10,srcX) — call arg#1 offsetY=var36 maps to Java dstHeight (arg5=var49); call arg#9 srcY=GetHeightMapY-var11 maps to Java srcY (arg4=var48); UpdateVelocity arg0=GetHeightMapY-var36 maps to Java setTarget arg1=getAvH-var49. Consequence: projectile (spell/arrow) source-Y and target-height offsets are 1/4 of intended → projectiles render at the wrong vertical height. Reachable whenever the server sends MAP_PROJANIM with a nonzero height offset (common for spells/ranged). Read width is unchanged (still one g1 each) so no desync, but the rendered arc is wrong. Not documented in LOGIC-DELTA-SCOPE-254.md / PORT-DESIGN-254.md / RENAME-MAP.md. Fix: multiply both G1() reads by 4. The other ten zone-opcode read sequences and all field widths were diffed and match Java exactly.

> **Verifier (confirmed, severity bug):** I read both sides with line numbers and traced the full argument flow through the constructors and callees.
> 
> JAVA (git show 2e62978:src/main/java/jagex2/client/Client.java, zonePacket op 37 / MAP_PROJANIM, lines 7673-7696):
> - 7682: int var48 = arg1.g1() * 4;
> - 7683: int var49 = arg1.g1() * 4;
> - 7693: new ClientProj(minusedlevel, var47, var50+loopCycle, var51+loopCycle, getAvH(var55,lvl,var54) - var48, var49, var54, var52, var46, var55, var53)
> - 7694: var58.setTarget(var57, getAvH(var57,lvl,var56) - var49, var56, var50+loopCycle)
> ClientProj ctor (ClientProj.java:81-94): arg4->srcY (86), arg5->dstHeight (92). So srcY = getAvH - var48(=g1*4); dstHeight = var49(=g1*4); setTarget arg1 height = getAvH - var49(=g1*4).
> 
> GO (pkg/jagex2/client/client.go, SERVERPROT_MAP_PROJANIM block, lines 1322-1346):
> - 1332: var11 = arg1.G1()   <- no *4
> - 1333: var36 = arg1.G1()   <- no *4
> - 1343: NewClientProj(var36, var15, var6, var14+LoopCycle, CurrentLevel, var9, var37+LoopCycle, var16, GetHeightMapY(CurrentLevel,var5,var6)-var11, var10, var5)
> - 1344: var43.UpdateVelocity(GetHeightMapY(CurrentLevel,var7,var8)-var36, var8, var7, var37+LoopCycle)
> Go ctor sig (clientproj.go:39): NewClientProj(offsetY, peakPitch, srcZ, lastCycle, level, target, startCycle, arc, srcY, arg10, srcX); offsetY->OffsetY (51), srcY->SrcY (45). Field OffsetY == Java dstHeight (clientproj.go:18 OffsetY; RENAME-MAP.md:116 maps ClientProj offsetY to the same field).
> 
> Read-order mapping is 1:1 (var48->var11, var49->var36) and read widths are identical (1 byte g1 each, both sides), so no protocol desync; payload still 15 bytes (LOGIC-DELTA-SCOPE-254.md:177 MAP_PROJANIM(15) verified: 1+1+1+2+2+1+1+2+2+1+1=15).
> 
> Mismatch, field-by-field:
> - srcY: Java getAvH - var48(g1*4) vs Go GetHeightMapY - var11(g1, NO *4) -> Go is 1/4 short on the offset.
> - dstHeight: Java var49(g1*4) vs Go OffsetY=var36(g1, NO *4).
> - setTarget/UpdateVelocity height arg: Java getAvH - var49(g1*4) vs Go GetHeightMapY - var36(g1, NO *4).
> 
> False-positive checks: (1) Not a deob arg scramble — I traced ctor field stores on both sides; graphic/peak/target/arc all match (var47=var10, var52=var15, var46=var9, var53=var16). (2) GetHeightMapY(level,x,z)==getAvH(z,level,x) is a documented compensated pair, correctly applied here — not part of this finding. (3) No compensating *4 anywhere: var11/var36 are block-local (declared/assigned at 1322/1332/1333, used only at 1343/1344); grep of all var11/var36 occurrences shows the others are in unrelated scopes. (4) OffsetY consumed downstream at client.go:6809 & 6822 (UpdateVelocity(GetHeightMapY(...)-v.OffsetY, ...)) == Java dstHeight use at Client.java:5482 & 5494 (setTarget(..., getAvH(...) - var3.dstHeight, ...)) — confirms it is a vertical height offset and the 1/4 error propagates to every mobile-projectile re-target. (5) Not documented in LOGIC-DELTA-SCOPE-254.md / PORT-DESIGN-254.md / RENAME-MAP.md / README.md (only opcode+payload size documented).
> 
> The finder's cites are accurate. Confirmed.
>
> **Refined:** CONFIRMED. MAP_PROJANIM (zone opcode 37) drops the `* 4` scale on BOTH single-byte height reads, mis-scaling projectile source-Y and destination-height to 1/4 of intended.
> 
> Java (Client-Java @2e62978 src/main/java/jagex2/client/Client.java):
> - 7682: `int var48 = arg1.g1() * 4;`  (source-Y offset)
> - 7683: `int var49 = arg1.g1() * 4;`  (destination-height offset)
> - 7693: `new ClientProj(..., getAvH(var55,lvl,var54) - var48 /*arg4 srcY*/, var49 /*arg5 dstHeight*/, ...)`
> - 7694: `var58.setTarget(var57, getAvH(var57,lvl,var56) - var49, var56, ...)`
> - ClientProj.java:86 `srcY = arg4`, :92 `dstHeight = arg5`.
> 
> Go (pkg/jagex2/client/client.go, SERVERPROT_MAP_PROJANIM block):
> - 1332: `var11 = arg1.G1()`   — missing `* 4`
> - 1333: `var36 = arg1.G1()`   — missing `* 4`
> - 1343: `entity.NewClientProj(var36 /*OffsetY==Java dstHeight*/, ..., c.GetHeightMapY(c.CurrentLevel, var5, var6)-var11 /*SrcY*/, var10, var5)`
> - 1344: `var43.UpdateVelocity(c.GetHeightMapY(c.CurrentLevel, var7, var8)-var36, var8, var7, ...)`
> - ctor pkg/jagex2/dash3d/entity/clientproj.go:39 `NewClientProj(offsetY,...,srcY,arg10,srcX)`; offsetY->OffsetY (:51, == Java dstHeight), srcY->SrcY (:45).
> 
> Argument mapping fully traced and matches the finder: Go var11 == Java var48 (srcY offset), Go var36 == Java var49 (dstHeight). Read widths are identical (one g1/G1 each), so no protocol desync — payload stays 15 bytes. Effect: ClientProj.SrcY and ClientProj.OffsetY are each 1/4 of the Java value, so spawned projectiles render at the wrong vertical height and arc. The error also propagates to every mobile-projectile re-target: OffsetY is reused at client.go:6809 and 6822 (`UpdateVelocity(GetHeightMapY(...)-v.OffsetY, ...)`), matching Java Client.java:5482/5494 (`setTarget(..., getAvH(...) - var3.dstHeight, ...)`). Reachable in normal play whenever the server sends MAP_PROJANIM with a nonzero height offset (common for spells/ranged). Not documented in LOGIC-DELTA-SCOPE-254.md, PORT-DESIGN-254.md, RENAME-MAP.md, or README.md. No compensating `* 4` exists anywhere (var11/var36 are block-local to the projanim branch). Fix: multiply both G1() reads at client.go:1332-1333 by 4.


---

## Latent (13)

### client-04-01. @cr1@/@cr2@ crown-tag strip uses if/else-if; rev-254 Java uses two independent ifs

- **Unit:** `client-04`  **Java:** `Client.java:3490-3497 (handlePrivateChatInput) and Client.java:3546-3553 (handleChatMouseInput)`  **Go:** `pkg/jagex2/client/client.go:5581-5585 (HandlePrivateChatInput) and pkg/jagex2/client/client.go:2013-2017 (HandleChatMouseInput)`

Java strips the crown tag with TWO independent if blocks: `if (var5!=null && var5.startsWith("@cr1@")) var5=var5.substring(5);` then a separate `if (var5!=null && var5.startsWith("@cr2@")) var5=var5.substring(5);`. Go uses `if strings.HasPrefix(var10,"@cr1@"){var10=var10[5:]} else if strings.HasPrefix(var10,"@cr2@"){var10=var10[5:]}`. Divergence: for a sender beginning literally "@cr1@@cr2@...", Java strips both (after removing "@cr1@" the residue "@cr2@..." still matches the second if), yielding "..."; Go's else-if removes only "@cr1@", leaving "@cr2@...". The leftover "@cr2@" then flows into IsFriend(var10) (which fails) and into the Report-abuse/Add-ignore/Add-friend menu strings (e.g. "Add friend @whi@@cr2@Name"). Reachability: only when a message sender carries both crown prefixes in sequence; normal server output prefixes a name with at most one crown tag, hence latent. The Go comment claims "245.2 makes the two strips mutually exclusive (if/else if)" but the authoritative rev-254 Java at 2e62978 (lines 3490-3497, 3546-3553) uses two independent ifs, so Go does not match rev-254.

> **Verifier (confirmed, severity latent):** Read both sides with line numbers.
> 
> Java rev-254 (AUTHORITATIVE, 2e62978 src/main/java/jagex2/client/Client.java):
> - handlePrivateChatInput, lines 3490-3497: TWO independent ifs — `if (var5!=null && var5.startsWith("@cr1@")){var5=var5.substring(5); boolean var7=true;}` then `if (var5!=null && var5.startsWith("@cr2@")){var5=var5.substring(5); boolean var8=true;}`.
> - handleChatMouseInput, lines 3546-3553: same two independent ifs (var8/var10/var11).
> - Two further strip sites also use two independent ifs: 6005-6012 (split-PM DrawChatback render, sets byte var8=1 then 2) and 10692-10699 (main DrawChatback render, sets byte var8=1 then 2).
> 
> Go port (pkg/jagex2/client/client.go, rev-254 working tree):
> - HandlePrivateChatInput 5581-5585: `if HasPrefix(var10,"@cr1@"){var10=var10[5:]} else if HasPrefix(var10,"@cr2@"){var10=var10[5:]}` — if/else if. DIVERGES from Java 3490-3497.
> - HandleChatMouseInput 2013-2017: same if/else if. DIVERGES from Java 3546-3553.
> - DrawChatback main render 10308-10314: if/else if (also sets var11=1 in first arm, 2 in else-if arm). DIVERGES from Java 10692-10699 — finder MISSED this site.
> - DrawChatback split-PM render 1026-1033: TWO independent ifs (var11=1 then var11=2) — this one MATCHES Java 6005-6012, proving the if/else-if at the other three is an inconsistent port, not a blanket decision.
> 
> Concrete divergence for sender literal "@cr1@@cr2@Name": Java strips both crown tags -> "Name"; Go's else-if strips only "@cr1@" -> "@cr2@Name". That residue flows into IsFriend(var10) (mismatch) and the Report-abuse/Add-ignore/Add-friend menu strings. At the render site 10308 it additionally yields the wrong mod-icon (var11=1 mod-crown instead of 2 admin-crown). The Go `// Java: ...245.2 makes the two strips mutually exclusive (if/else if)` comments are accurate for 245.2: 176a85f Client.java:3753-3759 genuinely used `else if`. So the Go is a faithful 245.2 port that was NOT updated for the rev-254 change to two independent ifs.
> 
> Checked false-positive classes: (4) restructured-but-equivalent — refuted, a concrete input value ("@cr1@@cr2@Name") diverges; (5) behavior-lives-elsewhere — grepped all Go @cr1@ sites, all four accounted for; (2)/standing decisions — no entry for crown-tag in LOGIC-DELTA-SCOPE-254.md, PORT-DESIGN-254.md, RENAME-MAP.md, README.md; (7) the stale-comment angle is the cause, not the finding. Reachability: server prefixes a name with at most one crown tag in normal play, so only an unusual/crafted "@cr1@@cr2@..." sender triggers it -> latent. Confirmed, broadened to include the third divergent site at client.go:10308 and noting the matching site at 1026.
>
> **Refined:** CONFIRMED (latent). The Go port strips the @cr1@/@cr2@ crown-tag prefix with an `if/else if` (mutually exclusive) at THREE sites, but rev-254 Java (2e62978) uses TWO independent `if` blocks at all four of its strip sites. Divergence is concrete: for a sender beginning "@cr1@@cr2@Name", Java strips both -> "Name"; Go's else-if strips only "@cr1@" -> "@cr2@Name". The residue then flows into IsFriend() (fails) and the Report-abuse/Add-ignore/Add-friend menu strings, and at the render site corrupts the mod-icon index too.
> 
> Divergent sites (Go if/else-if vs Java two-independent-ifs):
> - pkg/jagex2/client/client.go:5581-5585 (HandlePrivateChatInput) vs Client.java:3490-3497 (handlePrivateChatInput).
> - pkg/jagex2/client/client.go:2013-2017 (HandleChatMouseInput) vs Client.java:3546-3553 (handleChatMouseInput).
> - pkg/jagex2/client/client.go:10308-10314 (DrawChatback main render) vs Client.java:10692-10699 — MISSED by the finder; here Go also sets var11=1 (mod crown) and skips the else-if instead of overwriting to var11=2 (admin crown), so "@cr1@@cr2@Name" gets the wrong crown icon in addition to the wrong name.
> 
> Already-correct site (no fix needed, for contrast): pkg/jagex2/client/client.go:1026-1033 (DrawChatback split-PM render) uses two independent ifs and matches Client.java:6005-6012 — proving the else-if at the other three sites is an inconsistent port, not a deliberate uniform decision.
> 
> Root cause: the Go `// Java: ...245.2 makes the two strips mutually exclusive (if/else if)` comments are accurate for the 245.2 reference (176a85f Client.java:3753-3759 used `else if`), but rev-254 changed all sites to two independent ifs; the Go port faithfully kept the 245.2 form and was not updated. No documented exception exists in LOGIC-DELTA-SCOPE-254.md, PORT-DESIGN-254.md, RENAME-MAP.md, or README.md.
> 
> Fix: change the three `else if` to a second independent `if` (matching the already-correct site at 1026-1033), and update the three stale `// Java: ...if/else if` comments to cite 2e62978 two-independent-ifs. Severity latent: normal server output prefixes a name with at most one crown tag, so only an unusual/crafted "@cr1@@cr2@..." sender reaches it.

---

### client-04-02. HandleChatMouseInput has Go-only `c.PacketSize += arg1` not present in Java

- **Unit:** `client-04`  **Java:** `Client.java:3534-3610 (handleChatMouseInput body has no packetSize statement; second param arg2 = super.mouseX-17 at caller Client.java:3443 is unused)`  **Go:** `pkg/jagex2/client/client.go:2074 (`c.PacketSize += arg1`); sole caller pkg/jagex2/client/client.go:6625 passes 0`

Go HandleChatMouseInput ends with `c.PacketSize += arg1`. rev-254 Java handleChatMouseInput (3535-3609) contains no packetSize manipulation; its second parameter is a dead/unused arg. The only Go caller passes 0, so `c.PacketSize += 0` is inert now. But c.PacketSize is the live network packet-length field (assigned from SERVERPROT_SIZES at client.go:10453; consumed by ReadFully/GetPlayer/GetNpcPos in the read loop). This is a leftover deob anti-tamper accumulator of the same class that was already DROPPED for handleChatModeInput (see comment at client.go:1952-1953 'the deob junk-byte slot and its PacketSize accumulator are dropped (audit client-04-06)'). Latent: inert at the current callsite, protocol-desync hazard if any future caller passes a nonzero second arg. Matches prior-branch finding client-04-02 in PARITY-AUDIT-2026-06-04.md, still present in rev-254.

> **Verifier (confirmed, severity latent):** Read both sides with line numbers. Java handleChatMouseInput(int arg0, int arg2) at Client.java:3535-3610 (git show 2e62978) — the method body ends at line 3609/3610 with NO packetSize statement; grep for `arg2` over lines 3535-3610 matches ONLY the signature line, so arg2 is a genuinely dead/unused parameter. The sole Java caller (Client.java:3443) passes `super.mouseX - 17` as the second arg, but since Java has no packetSize field, that value is harmlessly discarded. Go HandleChatMouseInput(arg0, arg1 int) at client.go:2000 ends with `c.PacketSize += arg1` at client.go:2074 — a Go-only statement with no Java counterpart. The sole Go caller (client.go:6625) passes 0, so `c.PacketSize += 0` is inert today. c.PacketSize is confirmed the live network packet-length field: decl client.go:371 (int); assigned from SERVERPROT_SIZES[c.PacketType] at client.go:10453, variable-length read at 10464/10476, consumed by ReadFully(c.In.Data, 0, c.PacketSize) at 10483 and the read loop (10479, 10601, GetNpcPos 10593, GetPlayer 10614). This is a leftover deob anti-tamper accumulator of exactly the same class as the one explicitly DROPPED for HandleChatSettingsInput (comment client.go:1952-1953 'the deob junk-byte slot and its PacketSize accumulator are dropped (audit client-04-06)'). FP-class checks: not a deob arg scramble (the extra param is genuinely inserted, value flow traced through caller and callee); not a compensated pair or standing decision (not on the DO-NOT-REPORT list); not behavior-lives-elsewhere (the statement exists only in Go); not stale-comment cosmetic (`c.PacketSize += arg1` is live executable code). The prior-branch PARITY-AUDIT-2026-06-04.md classified this specific chat instance as 'cosmetic' (line 380, 'Net-equivalent; record only'), but the structurally identical HandleScrollInput case (client-11-02, lines 180-184) was verifier-confirmed as latent with this same reasoning and a recommendation to drop the param+statement. Latent is the consistent, more accurate severity: real mistranslation, inert at the current callsite, protocol-desync hazard if any future caller ever passes a nonzero second arg.
>
> **Refined:** Go HandleChatMouseInput(arg0, arg1 int) (pkg/jagex2/client/client.go:2000) ends with a Go-only statement `c.PacketSize += arg1` (client.go:2074) that has no counterpart in rev-254 Java. Java handleChatMouseInput(int arg0, int arg2) (Client.java:3535-3610 @2e62978) contains no packetSize manipulation, and its second parameter arg2 is dead/unused (grep of the body matches only the signature line; the sole Java caller at Client.java:3443 passes `super.mouseX - 17`, harmlessly discarded since Java has no packetSize field). The sole Go caller (client.go:6625) passes 0, so `c.PacketSize += 0` is inert today. However c.PacketSize is the live network packet-length field (decl client.go:371; set from SERVERPROT_SIZES[c.PacketType] at client.go:10453 and the variable-length branches 10464/10476; consumed by ReadFully at client.go:10483 and by the read loop / GetNpcPos / GetPlayer). This is a leftover deob anti-tamper accumulator of the same class as the one explicitly dropped for HandleChatSettingsInput (see comment client.go:1952-1953, audit client-04-06) and as the verifier-confirmed-latent HandleScrollInput case (client-11-02). Latent: inert at the current callsite, but a protocol-desync hazard if any future caller passes a nonzero second arg. Fix: drop the extra param and the `c.PacketSize += arg1` statement (per the precedent set by HandleChatSettingsInput / HandleScrollInput). Note the prior-branch PARITY-AUDIT-2026-06-04.md (line 380) recorded the chat-specific instance as cosmetic; latent is the consistent classification given the identical HandleScrollInput case was rated latent.

---

### client-07-01. DrawPrivateMessages type-6 ("To") line uses original sender field instead of the crown-stripped local

- **Unit:** `client-07`  **Java:** `Client.java:6043-6051 (and strip at 6003-6012) @2e62978`  **Go:** `pkg/jagex2/client/client.go:1063-1066 (strip local at 1024-1033)`

Java drawPrivateMessages type-6 branch renders `"To " + var7 + ": " + messageText[var5]` where var7 is the SENDER LOCAL after the @cr1@/@cr2@ crown-tag strip (var7 = messageSender[var5]; then var7 = var7.substring(5) when it begins with @cr1@/@cr2@). Go DrawPrivateMessages type-6 branch (client.go:1063-1066) renders `"To "+c.MessageSender[i]+": "+c.MessageText[i]` using the ORIGINAL field c.MessageSender[i], NOT the stripped local var10 (Go's equivalent of Java var7; var10 := c.MessageSender[i] then stripped at client.go:1024-1033). The Go type-3/7 branch (client.go:1034,1047-1048) correctly uses the stripped var10, so the type-6 branch is an internal inconsistency / oversight, not a documented decision (no entry in LOGIC-DELTA-SCOPE-254 / PORT-DESIGN-254 / RENAME-MAP). Cross-check: Client-TS Client.ts:5212-5216 uses the stripped local `sender` for the 'To' line, matching Java. Divergence: a type-6 message whose sender carried a @cr1@/@cr2@ prefix would render the prefixed name in Go vs the stripped name in Java. Reachability LOW: type-6 messages are created only at Client.java:4315 with sender = formatDisplayName(fromBase37(socialName37)) (the recipient display name, never carrying a crown prefix), so the strip is a no-op for every type-6 message the client produces and rendered output is identical in normal play. Fix for faithful 1:1 parity: use var10 instead of c.MessageSender[i].

> **Verifier (confirmed, severity latent):** Read Java drawPrivateMessages at Client.java:5990-6053 @2e62978. The crown-strip (6003-6012) computes `var7 = messageSender[var5]` then `var7 = var7.substring(5)` for "@cr1@"/"@cr2@" prefixes. The type-3/7 branch (6027-6028) and the type-6 "To" branch (6045-6046) BOTH render `var7` (the stripped local).
> 
> Read Go DrawPrivateMessages at client.go:1010-1074. The strip (1024-1033) computes `var10 := c.MessageSender[i]` then `var10 = var10[5:]` — equivalent to Java var7. The type-3/7 branch (1047-1048) correctly uses `var10`. But the type-6 branch (1065-1066) renders `"To "+c.MessageSender[i]+": "+c.MessageText[i]` — the ORIGINAL field, NOT the stripped `var10`. This is a genuine deviation from Java; the type-3/7 branch in the same function proves the strip-local was intended.
> 
> Checked the false-positive classes: not a compensated pair (this is a single field reference, not call-site arg order), not a name inversion (MessageSender maps directly to Java messageSender — verified via AddChat at client.go:8758-8774 where `c.MessageSender[0] = arg3`, the compensated AddChat(int,text,sender) pair), not restructured-equivalent (var10 vs MessageSender[i] differ concretely when a crown prefix is present), not behavior-elsewhere. No entry in LOGIC-DELTA-SCOPE-254/PORT-DESIGN-254/RENAME-MAP documents this.
> 
> Verified reachability LOW: the only type-6 producer is Java Client.java:4315 `addChat(formatDisplayName(fromBase37(socialName37)), 6, ...)` and its Go twin client.go:2305 `c.AddChat(6, c.SocialInput, jstring.FormatName(jstring.FromBase37(c.SocialName37)))` — sender is the recipient display name, which never carries an @cr1@/@cr2@ prefix. So the strip is a no-op for every client-produced type-6 message and rendered output is identical in normal play. Divergence only manifests if a crown-prefixed sender ever populated MessageSender for a type-6 entry (not produced by this client). Confirmed as latent.
>
> **Refined:** DrawPrivateMessages type-6 ("To") branch uses the original sender field instead of the crown-stripped local. Go pkg/jagex2/client/client.go:1065-1066 renders `"To "+c.MessageSender[i]+": "+c.MessageText[i]`, referencing the original MessageSender field. Java Client.java:6045-6046 @2e62978 renders `"To " + var7 + ": " + this.messageText[var5]` where var7 is the @cr1@/@cr2@-stripped sender local (strip at Client.java:6003-6012; Go equivalent var10 at client.go:1024-1033). The Go type-3/7 branch (client.go:1047-1048) correctly uses the stripped var10, mirroring Java (Client.java:6027-6028), which makes the type-6 branch an internal inconsistency. Fix for 1:1 parity: use var10 instead of c.MessageSender[i] in both DrawString calls at client.go:1065-1066. Severity latent: the sole client-produced type-6 message (Go client.go:2305 / Java Client.java:4315) passes a crown-free recipient display name as the sender, so the strip is a no-op and rendered output is identical in normal play; the divergence is only reachable if a crown-prefixed name ever reaches MessageSender for a type-6 entry. Cross-check: Client-TS Client.ts:5212-5216 uses the stripped local for the "To" line, matching Java.

---

### client-08-03. MIDI_SONG (ptype 163) drops the `nextMusicDelay == 0` gate and adds an unconditional NextMusicDelay=0

- **Unit:** `client-08`  **Java:** `Client.java:7181-7195 @2e62978 (ptype 163 MIDI_SONG)`  **Go:** `pkg/jagex2/client/client.go:10889-10903 (SERVERPROT_MIDI_SONG handler)`

Java condition is 4-part: `if (var89 != this.nextMidiSong && this.midiActive && !lowMem && this.nextMusicDelay == 0)` and the handler does NOT reset nextMusicDelay. Go condition is 3-part: `if c.NextMidiSong != id && c.MidiActive && !LowMemory` (missing `&& NextMusicDelay == 0`) and ADDS `c.NextMusicDelay = 0` after `c.NextMidiSong = id`. 254 reference (2e62978:7187) is authoritative and clearly has the gate and no reset. The Go line predates 254 (rev-244 commit bf2fb8c; persisted through c5cb8ac rename); not documented in LOGIC-DELTA-SCOPE-254.md / PORT-DESIGN-254.md. Reachability: when a jingle is active (NextMusicDelay>0, set by MIDI_JINGLE at client.go:10912), a MIDI_SONG packet in Java does NOT swap the song (gate) but updates nextMidiSong so the deferred resume at client.go:8189-8192 plays the new song after the jingle. In Go the new song starts immediately over the jingle AND NextMusicDelay is zeroed, cancelling the jingle's deferred resume. Wrong audio behavior whenever MIDI_SONG arrives during a jingle delay.

> **Verifier (confirmed, severity latent):** Read Java 254 (2e62978) Client.java and Go client.go on both sides with line numbers.
> 
> Java MIDI_SONG handler, Client.java:7181-7195 (nl -ba): `if (this.ptype == 163)` reads var89=g2() (65535→-1), then `if (var89 != this.nextMidiSong && this.midiActive && !lowMem && this.nextMusicDelay == 0) { midiSong=var89; midiFading=true; onDemand.request(2, midiSong); }`, then `this.nextMidiSong = var89;` — a FOUR-part gate, and NO write to nextMusicDelay.
> 
> Go MIDI_SONG handler, client.go:10889-10903: `if c.PacketType == SERVERPROT_MIDI_SONG` reads id=G2() (65535→-1), then `if c.NextMidiSong != id && c.MidiActive && !LowMemory { c.MidiSong=id; c.MidiFading=true; c.OnDemand.Request(2, c.MidiSong) }`, then `c.NextMidiSong = id` and `c.NextMusicDelay = 0`. So Go is missing the `&& c.NextMusicDelay == 0` gate clause AND adds an unconditional `c.NextMusicDelay = 0`. Finder's cites are exact.
> 
> Reachability confirmed:
> - MIDI_JINGLE (Java 7196-7208 / Go 10905-10915) sets nextMusicDelay = delay (Java 7204 / Go 10912) when midiActive && !lowMem — so nextMusicDelay > 0 is a real state.
> - Deferred-resume loop (Java 3393-3403 / Go 8181-8194): while nextMusicDelay > 0, decrement by 20; when it hits 0 with midiActive && !lowMem, play midiSong = nextMidiSong over OnDemand archive 2. This is the mechanism the gate defers to.
> 
> Concrete divergent input: jingle with delay>0 in flight, then a MIDI_SONG packet with a new id arrives before the delay elapses. Java: gate false → does NOT swap now, updates nextMidiSong; the new song plays when the jingle delay reaches 0. Go: no gate → swaps immediately over the jingle, and `NextMusicDelay = 0` cancels the jingle's deferred resume. Different observable audio.
> 
> False-positive checks: grep of all nextMusicDelay sites (Java: 931 decl, 2652 logout reset, 3393-3403 loop, 7187 gate, 7204 jingle, 10241 vol-toggle; Go: 369 decl, 3771 logout reset, 4163 vol-toggle, 8181-8194 loop, 10900 the extra reset, 10912 jingle). No compensating mechanism re-creates the gate for Go; the only other NextMusicDelay=0 sites are logout (3771) and the volume-toggle path (4163, mirrors Java 10241) — both unrelated to MIDI_SONG. Not a compensated pair, name inversion, masked-shift, or equivalent restructuring. Not in LOGIC-DELTA-SCOPE-254.md (only an opcode-renumber table row 146) / PORT-DESIGN-254.md / RENAME-MAP.md. git -L confirms the divergent lines were introduced rev-244 (bf2fb8c, a rewrite — rev-225 used a different SetMidi name/CRC handler) and survived the c5cb8ac protocol-table rename; never matched 254 Java.
> 
> Severity: divergence is reachable only on an overlapping-packet timing window (jingle delay still counting down when a song-change packet arrives) driven by unusual server ordering, not normal play — latent rather than bug. (The 254 reference is authoritative per instructions; whether rev-244-vintage Java differed does not change the divergence vs 2e62978.)
>
> **Refined:** CONFIRMED. The Go SERVERPROT_MIDI_SONG handler diverges from Java 254 (ptype 163) in two compounding ways.
> 
> Java (2e62978 Client.java:7181-7195):
> ```
> if (this.ptype == 163) { // MIDI_SONG
>   int var89 = this.in.g2();
>   if (var89 == 65535) var89 = -1;
>   if (var89 != this.nextMidiSong && this.midiActive && !lowMem && this.nextMusicDelay == 0) {
>     this.midiSong = var89; this.midiFading = true; this.onDemand.request(2, this.midiSong);
>   }
>   this.nextMidiSong = var89; this.ptype = -1; return true;
> }
> ```
> Note the 4-part gate ending in `&& this.nextMusicDelay == 0`, and that nextMusicDelay is NOT modified.
> 
> Go (pkg/jagex2/client/client.go:10889-10903):
> ```
> if c.PacketType == SERVERPROT_MIDI_SONG {
>   id := c.In.G2()
>   if id == 65535 { id = -1 }
>   if c.NextMidiSong != id && c.MidiActive && !LowMemory {   // MISSING && c.NextMusicDelay == 0
>     c.MidiSong = id; c.MidiFading = true; c.OnDemand.Request(2, c.MidiSong)
>   }
>   c.NextMidiSong = id
>   c.NextMusicDelay = 0    // EXTRA: not present in Java
>   c.PacketType = -1; return true
> }
> ```
> 
> Behavioral impact: a MIDI_JINGLE packet (Java 7196-7208 / Go 10905-10915) sets nextMusicDelay = delay > 0 (Java 7204 / Go 10912). The deferred-resume loop (Java 3393-3403 / Go 8181-8194) counts that delay down and, on reaching 0, plays nextMidiSong over OnDemand archive 2 — resuming background music after the jingle finishes. When a MIDI_SONG packet with a new id arrives while a jingle delay is still counting down:
> - Java: the `nextMusicDelay == 0` gate is false, so the song does NOT swap immediately; it only records nextMidiSong, and the new song begins when the jingle's delay elapses.
> - Go: the gate is absent, so the new song starts immediately over the still-playing jingle, and `c.NextMusicDelay = 0` cancels the jingle's deferred resume entirely.
> 
> Fix: add `&& c.NextMusicDelay == 0` to the condition and remove the `c.NextMusicDelay = 0` line, matching Java 254.
> 
> Not documented in LOGIC-DELTA-SCOPE-254.md / PORT-DESIGN-254.md / RENAME-MAP.md (only an opcode-renumber table entry for MIDI_SONG). Divergence introduced rev-244 (bf2fb8c, a rewrite of the rev-225 SetMidi handler) and carried through c5cb8ac; it never matched the 254 Java. Severity downgraded from the finder's "bug" to "latent": reachable only on the specific overlapping-packet timing where a song-change arrives during an active jingle delay, i.e. unusual server packet ordering rather than a normal-play path.

---

### client-13-01. RenderFlames drops this.flameCycle++ per-iteration increment

- **Unit:** `client-13`  **Java:** `src/main/java/jagex2/client/Client.java:11091 (renderFlames loop body); field decl 727; only reader lag() at 4491`  **Go:** `pkg/jagex2/client/client.go:7201-7204 (RenderFlames loop body); no FlameCycle field in package`

Java renderFlames() executes this.flameCycle++ as the first statement of every while(this.flameActive) iteration (Client.java:11091). flameCycle (field decl Client.java:727) is a DISTINCT field from flameCycle0 (decl 784, the +=128 gradient counter that IS ported to Go FlameCycle0). The Go RenderFlames loop body (client.go:7201-7204) is UpdateFlames();UpdateFlames();DrawFlames();var4++ with no increment of any FlameCycle field, and no bare FlameCycle field exists anywhere in the package (grep-confirmed; only FlameCycle0 exists). flameCycle is written in exactly one place (this increment) and read in exactly one place: the lag() debug dump System.out.println("flame-cycle:"+this.flameCycle) at Client.java:4491. lag() is reachable only via the unported ::lag staff command and is itself not ported to Go. Therefore the dropped increment has no observable effect today; it becomes an always-prints-0 bug only if lag() is later ported. Reachability: none currently. Matches prior audit (PARITY-AUDIT-2026-06-03 L18).

> **Verifier (confirmed, severity latent):** Verified both sides with line numbers.
> 
> JAVA (read via git show 2e62978):
> - Client.java:11084-11110 renderFlames(): loop `while (this.flameActive)` whose FIRST statement is `this.flameCycle++;` (11091), then updateFlames(); updateFlames(); drawFlames(); var4++ ...
> - Field decls: `flameCycle` at 727 (client.ae) is DISTINCT from `flameCycle0` at 784 (client.Ze, the +=128 gradient counter).
> - Full reference set for `flameCycle` (git grep): exactly 3 sites — decl 727, reader `System.out.println("flame-cycle:" + this.flameCycle)` at 4491 inside lag() (4489), and the increment 11091. So written in 1 place (the increment), read in 1 place (lag dump).
> - lag() callers (git grep `.lag(`): exactly one, Client.java:4373, guarded by `staffmodlevel == 2` and `chatTyped.equals("::lag")` (4368-4374).
> 
> GO (read working tree):
> - client.go:7193-7217 RenderFlames(): loop `for c.FlameActive { c.UpdateFlames(); c.UpdateFlames(); c.DrawFlames(); var4++; ... }`. No increment of any flameCycle field anywhere in the loop body (7201-7215).
> - grep FlameCycle across pkg/: only FlameCycle0 exists (decl client.go:483; used 2850-2860 for the +=128 gradient). No bare FlameCycle field in the package — confirmed.
> - The ::lag command is intentionally not ported: client.go:2365-2381 handles ::clientdrop and ::prefetchmusic under StaffModLevel==2 but the comment at 2373-2380 states "Java also handles \"::lag\" here via the lag() stdout debug dump; lag() and its whole diagnostic family are intentionally not ported (audit client-05-04/01-02/02-06/13-02, gameshell-02): the drawCycle/flameCycle counters exist only to be printed here..."
> 
> ASSESSMENT vs false-positive classes: Not a deob arg scramble (single field, no args). Not a compensated pair / standing decision in the DO-NOT-REPORT list (flameCycle is not listed; only FlameCycle0 was ported). Not name-inversion (FlameCycle0 is the gradient counter, a genuinely different field — finder correctly distinguishes them). Not behavior-lives-elsewhere (grep of whole pkg confirms no increment site anywhere). Not restructured-equivalent: the increment is a real dropped side effect with no compensating write. The Go comment documents dropping the lag() READER and asserts the counters "exist only to be printed here," which is why the increment was dropped as collateral — so the divergence is consistent with a documented intentional decision, but the increment drop itself is a genuine mistranslation whose only saving grace is that its sole reader is unported.
> 
> Reachability TODAY: none. flameCycle's only reader (lag(), via ::lag) is intentionally not ported. So the dropped increment has zero observable effect now; it becomes an always-prints-0 bug only if lag() is later ported. This matches the finder's own classification and PARITY-AUDIT-2026-06-03 L18.
>
> **Refined:** CONFIRMED (latent). Java renderFlames() increments `this.flameCycle++` as the first statement of every `while (this.flameActive)` loop iteration (Client.java:11091). This field `flameCycle` (decl Client.java:727, client.ae) is distinct from `flameCycle0` (decl Client.java:784, client.Ze — the +=128 gradient counter that IS correctly ported to Go FlameCycle0). The Go RenderFlames loop body (client.go:7201-7215) runs UpdateFlames();UpdateFlames();DrawFlames();var4++ with NO increment of any flameCycle field, and no bare FlameCycle field exists in the client package (grep-confirmed; only FlameCycle0). flameCycle has exactly 3 sites in Java: decl (727), the increment (11091), and a single reader — `System.out.println("flame-cycle:" + this.flameCycle)` in lag() at Client.java:4491. lag() is called only from the `::lag` staff command path (Client.java:4368-4374, staffmodlevel==2). The Go port intentionally does not port lag() or the ::lag command; this is documented at client.go:2373-2380 ("lag() and its whole diagnostic family are intentionally not ported ... the drawCycle/flameCycle counters exist only to be printed here").
> 
> Net: the dropped increment is a genuine dropped side effect, but it is currently unreachable for any observable behavior because its sole reader (lag()) is itself intentionally not ported. It has zero effect on rendering/protocol today. It would surface as an always-prints-0 bug only if lag() / ::lag is ported in the future. Severity latent (real mistranslation, not-yet-exercised path). Note: an alternative classification of "intentional" is defensible given the documented lag()-family decision at client.go:2373-2380 implicitly covers the flameCycle counter; if the fix policy treats the documented lag()-family decision as authoritatively covering this increment, downgrade to intentional with that cite. As a pure latent divergence it is confirmed.

---

### client-shell-01. Escape key queues spurious 65535 (Java drops it as keyChar 27<30)

- **Unit:** `client-shell`  **Java:** `GameShell.java:379-438 (keyPressed: var3=getKeyChar(); if(var3<30)var3=0; no VK_ESCAPE override; if(var3>4) keyQueue push) + 440-480 (keyReleased) @2e62978`  **Go:** `pkg/jagex2/client/gameshell.go:266-371 (handleKey isSentinel gate incl. var3>=1000), :456-487 (charFor returns 65535 for non-KeyRune), :376-430 (awtFor has no Escape entry); pkg/jagex2/platform/backend_glfw.go:288-289 + backend_webgl.go:315-316 (Escape -> KeyEscape, not KeyNone)`

Both platform backends map Escape to neutral KeyEscape (not KeyNone), so an Escape press delivers KeyPress{KeyEscape} to handleKey. There awtFor(KeyEscape)=0 (Escape absent from the switch) and charFor(KeyEscape)=65535 (not KeyRune, gameshell.go:457-463). var2=0 means no override fires; var3 stays 65535; `65535<30` is false; the queue gate `isSentinel := var3==5||8||9||10||var3>=1000` is TRUE (65535>=1000) -> c.KeyQueue[KeyQueueWritePos]=65535 pushed. In Java AWT, KeyEvent.getKeyChar() for VK_ESCAPE returns control char 27 (0x1B), NOT CHAR_UNDEFINED; so Java keyPressed has var3=27, `27<30` zeroes it to 0, no var2 override matches (VK_ESCAPE=27 not in {37,39,38,40,17,8,127,9,10,112-123,36,35,33,34}), and `var3>4` is false -> Java pushes nothing and leaves actionKey untouched (complete no-op). The charFor comment (gameshell.go:458-463) justifying 65535 'exactly as Java does' is correct for arrows/Shift/Alt/F-keys/Ctrl/Home/End/PgUp/PgDn (AWT keyChar IS CHAR_UNDEFINED 65535 there, and overrides/queue behave identically) but WRONG for Escape, the unique reachable key whose AWT keyChar is a control char <30 with no var2 override. Reachable on every Escape keypress in normal play. No player-visible effect today: both keyQueue consumers range-check (chat input client.go:2352 needs var2<=122/<=126; login field client.go:2558-2569 checks CHARSET membership + {8,9,10,13}) and the queue is flushed each tick (KeyQueueReadPos=KeyQueueWritePos, gameshell.go:571). Secondary: when InputTracking.Enabled, Go records keyPressed(65535)/keyReleased(65535) for Escape whereas Java records keyPressed(0)/keyReleased(0), diverging the anticheat byte stream on every Escape. Latent (real reachable mistranslation, currently masked by range-checking consumers; fragile + divergent telemetry). Fix: map Escape to KeyNone (it is consumed nowhere in client/platform) or have charFor return 27 for KeyEscape so the <30 zeroing matches Java.

> **Verifier (confirmed, severity latent):** Verified both sides with line numbers.
> 
> GO PATH: (1) Platform backends map Escape -> KeyEscape (backend_glfw.go:288-289, backend_webgl.go:315-316), a non-KeyNone value, so it is NOT dropped at the `if k == KeyNone { return }` gate (backend_glfw.go:238-240) and a KeyPress{KeyEscape} reaches handleKey. (2) awtFor (gameshell.go:376-430) — read the full switch; there is no KeyEscape case, so var2 = 0. (3) charFor (gameshell.go:456-463) returns 65535 for any non-KeyRune key. (4) handleKey (gameshell.go:266-361): var3=65535, `65535<30` is false, no var2 override matches (var2=0), `isSentinel := var3==5||8||9||10||var3>=1000` is TRUE because 65535>=1000, so c.KeyQueue gets 65535 and inputtracking.KeyPressed(65535) runs (e.Key != KeyRune, gameshell.go:357).
> 
> JAVA PATH: keyPressed (GameShell.java:379-438 @2e62978), read verbatim. `int var3 = arg0.getKeyChar();` For VK_ESCAPE, AWT getKeyChar() returns control char 0x1B=27 (not CHAR_UNDEFINED). `if (var3 < 30) var3 = 0;` zeroes it. The override set is exactly keyCodes 37/39/38/40/17/8/127/9/10/112-123/36/35/33/34 — VK_ESCAPE=27 is absent, so no override fires. var3=0 -> `var3>0` false (no actionKey), `var3>4` false (NOTHING pushed to keyQueue), InputTracking.keyPressed(0). keyReleased (GameShell.java:440-480) symmetric: var3=0.
> 
> LOAD-BEARING AWT PREMISE (getKeyChar=27 for Escape): strongly corroborated by the Java code's own structure — the `var3<30` guard plus the explicit keyCode overrides for Backspace(8)/Delete(127)/Tab(9)/Enter(10) exist precisely because AWT delivers real control chars via getKeyChar() for those keys; CHAR_UNDEFINED=65535 is never <30. VK_ESCAPE (0x1B) is the canonical control-char key that has NO keyCode override, which is exactly the gap this finding identifies. This matches documented AWT behavior.
> 
> CONCRETE DIVERGENCE: Escape press -> Go queues 65535 + InputTracking byte; Java queues nothing + InputTracking byte 0. InputTracking.KeyPressed (inputtracking.go:170-198): 65535 hits `arg0 >= 1008 -> arg0 -= 992` = 64543, then OutBuffer.P1(64543) truncates to byte 64543 & 0xFF = 0x1F = 31. Java emits P1(0)=0. So the recorded anticheat byte stream diverges (31 vs 0) on every Escape press, and similarly on release (64543 trunc).
> 
> NO PLAYER-VISIBLE EFFECT TODAY: keyQueue consumers all range-check — client.go:2257 (outer loop requires alphanumeric/space), 2276/2325/2352 (chat/social/chatback require <=122/<=126/digit ranges), 2558-2563 (login requires CHARSET membership). 65535 matches none. Queue is flushed each tick (gameshell.go:571 KeyQueueReadPos=KeyQueueWritePos). Confirmed.
> 
> FALSE-POSITIVE CLASSES CLEARED: not a deob arg scramble; not a compensated/standing-decision pair (checked the DO-NOT-REPORT list — Escape not listed; nor in LOGIC-DELTA-SCOPE/PORT-DESIGN/RENAME-MAP); not a name inversion; control flow is NOT equivalent because a concrete input (Escape) diverges; behavior does not live elsewhere — grepped whole repo, Escape/KeyEscape is consumed nowhere in client/ (only the platform mapping and the test); not shift-masking. The Go charFor 65535 was a deliberate fix for Shift/Alt/F-keys (whose AWT keyChar IS CHAR_UNDEFINED=65535, per PARITY-AUDIT-2026-05-28:329), which is correct for those keys but introduced the Escape asymmetry because Escape's keyChar is 27, not 65535.
> 
> Severity latent is correct: real, reachable mistranslation on every Escape press, but masked from gameplay by range-checking consumers; only consequence today is divergent InputTracking telemetry bytes (an anticheat-stream mismatch, not a desync since it is fire-and-forget) and fragility to any future non-range-checking keyQueue consumer.
>
> **Refined:** CONFIRMED (latent). Escape key produces a spurious keyQueue entry and divergent InputTracking bytes vs Java.
> 
> Go path: Both platform backends map Escape to neutral KeyEscape (pkg/jagex2/platform/backend_glfw.go:288-289; pkg/jagex2/platform/backend_webgl.go:315-316), a non-KeyNone value, so it survives the KeyNone drop (backend_glfw.go:238-240) and a KeyPress{KeyEscape} reaches handleKey (pkg/jagex2/client/gameshell.go:266-371). awtFor has no KeyEscape case -> var2=0 (gameshell.go:376-430). charFor returns 65535 for non-KeyRune keys (gameshell.go:456-463). Therefore var3=65535: `65535<30` false; no var2 override; the gate `isSentinel := var3==5||8||9||10||var3>=1000` is TRUE (65535>=1000) so c.KeyQueue gets 65535 (gameshell.go:345-348); and inputtracking.KeyPressed(65535) runs (gameshell.go:357-359). In InputTracking (pkg/jagex2/client/inputtracking/inputtracking.go:170-198) 65535 hits `arg0 >= 1008 -> arg0 -= 992` = 64543, then OutBuffer.P1(64543) truncates to byte 0x1F = 31 (release path symmetric, inputtracking.go:201-229).
> 
> Java path: GameShell.java:379-438 @2e62978 (keyPressed). AWT KeyEvent.getKeyChar() for VK_ESCAPE returns control char 27 (0x1B), not CHAR_UNDEFINED, so var3=27; `if (var3 < 30) var3 = 0;` zeroes it; the override keyCodes are exactly {37,39,38,40,17,8,127,9,10,112-123,36,35,33,34} and VK_ESCAPE=27 is absent so no override fires; var3=0 -> actionKey untouched, `var3 > 4` false so NOTHING is pushed to keyQueue, and InputTracking.keyPressed(0) emits byte 0. keyReleased (GameShell.java:440-480) is symmetric (var3=0).
> 
> Net divergence on every Escape press/release: Go enqueues a spurious 65535 (vs Java nothing) and records InputTracking byte 31 (vs Java 0). The charFor 65535 sentinel is correct for arrows/Shift/Alt/Ctrl/F-keys (whose AWT keyChar genuinely is CHAR_UNDEFINED=65535) but WRONG for Escape, the one reachable key whose AWT keyChar is a control char <30 with no var2 keyCode override.
> 
> No player-visible effect today: every keyQueue consumer range-checks and rejects 65535 (chat client.go:2352 requires var2<=122/<=126; social 2276; chatback 2325; login 2558-2563 require CHARSET membership), and the queue is flushed each tick (gameshell.go:571). Impact is limited to the InputTracking/anticheat byte stream (telemetry mismatch, not a protocol desync) plus fragility to any future non-range-checking consumer. Escape is consumed nowhere else in client/ (grepped whole repo). Not a documented deviation (absent from DO-NOT-REPORT list, LOGIC-DELTA-SCOPE-254.md, PORT-DESIGN-254.md, RENAME-MAP.md).
> 
> Fix: map Escape to KeyNone in both backends (it is consumed nowhere), or have charFor return 27 for KeyEscape so Java's `<30` zeroing reproduces. The existing audit unit entry (audit-254/units/client-shell.md:55-59) already states this finding accurately; cites verified accurate (line ranges 266-371, 456-487, 376-430, 288-289, 315-316 all correct).

---

### pix3d-C-01. Perspective numerator products (arg12>>3)*var20 not int32-wrapped in TextureRaster

- **Unit:** `pix3d-C`  **Java:** `Client-Java Pix3D.java:2065-2067 (var81/82/83 high-detail) and 2239-2241 (var23/24/25 low-detail): var81 = arg9 + (arg12 >> 3) * var80; (32-bit int, wraps mod 2^32)`  **Go:** `pkg/jagex2/graphics/pix3d/pix3d.go:2088-2090 (LowDetail path) and 2273-2275 (high-detail path): var32 = arg9 + (arg12>>3)*var20; var33 = arg10 + (arg13>>3)*var20; var34 = arg11 + (arg14>>3)*var20 (64-bit int, no int32 guard)`

Java computes the perspective-correct texture numerators var81/var82/var83 (and the low-detail var23/var24/var25) as 32-bit int with wraparound. Go computes the same expressions (var32/var33/var34) in 64-bit int with no int(int32(...)) guard. These are full-value reads downstream: var19 = var34>>14 (high) or var34>>12 (low), then arg2 = var32/var19 and arg3 = var33/var19. A multiply/add that overflows 32 bits would wrap in Java but not in Go, changing the divisor var19 and the quotients arg2/arg3, hence the texture U/V coordinate and the rasterized pixel. Reachability: with valid in-range geometry these numerators stay under 2^31 (same magnitude class as the (arg8-arg7)*DivTable product that the prior 245 audit pix3d-3-03 classed latent-only and later int32-guarded), so Go and Java agree in normal play; divergence requires pathological arg9..arg14 inputs not produced by normal scene data. Recorded as latent (dominant 32-bit-wrap class) for completeness; not reachable in normal rendering. NOTE on the rest of the unit, all parity: flatRaster (dead arg3=literal-0 dropped; clipX->SafeWidth; post-increment blend trick reproduced); textureTriangle (all 9 int32-wrapped cross products map term-by-term under var20->var36/var21->var38/var22->var40/var23->var37/var24->var39/var25->var41; all 3 top-level branches + subcases + 24/24 TextureRaster call tuples verified position-by-position incl swap-sensitive branches; projectionY->CenterH3D, projectionX->CenterW3D); textureRaster (branch-order swap lowMem->LowDetail with per-path masks verified; (arg7>>23)&0x1F shift-count mask and 0x600000/0xC0000 low-bit masks faithful since only bits<=27 of arg7 are ever read; var15 int32-wrap fix present; >>> vs >> safe because texels masked &0xF8F8FF); static initializer (divTable/divTable2/sin/cos loops identical).

> **Verifier (confirmed, severity latent):** Read both sides with line numbers and traced argument flow through callers.
> 
> JAVA (Pix3D.java @ 2e62978, textureRaster decl at 2028):
> - !lowMem (high-detail) path, lines 2064-2067: `int var80 = arg5 - projectionX; int var81 = arg9 + (arg12 >> 3) * var80; int var82 = arg10 + (arg13 >> 3) * var80; int var83 = arg11 + (arg14 >> 3) * var80;` then `var84 = var83 >> 14; arg2 = var81/var84; arg3 = var82/var84`.
> - lowMem (low-detail) path, lines 2238-2241: `int var22 = arg5 - projectionX; int var23 = arg9 + (arg12 >> 3) * var22; ...` then `var26 = var25 >> 12; arg2 = var23/var26; arg3 = var24/var26`.
> All var80-83 / var22-25 are declared `int` → arithmetic wraps mod 2^32.
> 
> GO (pix3d.go, TextureRaster decl at 2041, all args plain `int`; var32/33/34 declared `int` at 2081-2083):
> - LowDetail block (Go tests LowDetail first; documented branch-order swap, audit pix3d-3): lines 2087-2090 `var20 = arg5 - CenterW3D; var32 = arg9 + (arg12>>3)*var20; var33 = arg10 + (arg13>>3)*var20; var34 = arg11 + (arg14>>3)*var20;` then `var19 = var34 >> 12` (matches Java lowMem >>12).
> - High-detail fall-through: lines 2272-2275 same expressions, then `var19 = var34 >> 14` (matches Java !lowMem >>14).
> No `int(int32(...))` guard on either pair. CenterW3D is plain `int` (defined 28, 102, 111). The finder's path labeling, branch-swap note, line cites, and downstream-read claims (var34>>14/>>12, arg2=var32/var19, arg3=var33/var19) are all exactly correct.
> 
> DISTINCT-SITE check: the prior 245 audit pix3d-3-03 (audit-245/units/pix3d-3.md:43-45) flagged only the `var15` step `(arg8-arg7)*DivTable[var16]>>6`; that one HAS since been int32-guarded (Go line 2065: `var15 = int(int32((arg8-arg7)*DivTable[var16])) >> 6`). The base perspective numerator products (var81/82/83 / var23/24/25) are a SEPARATE unguarded site in the same function not covered by that fix — confirmed by reading both the 245 audit notes and the live Go code.
> 
> MAGNITUDE/REACHABILITY: Traced arg12/13/14 origin. TextureTriangle call sites (pix3d.go:1521 etc.) pass `int(var21), int(var24), int(var27)` where var21/24/27 are int32-typed gradient cross products (declared int32 at 1454-1459). So arg12 is sign-extended from int32 → |arg12>>3| < 2^28. var20 = arg5 - CenterW3D where arg5 is a clipped scanline x (Java clamps arg5>=0 and arg6<=clipX in hclip path, lines 2037-2046) so |var20| is screen-width class (<2^11). Worst-case product ~2^39 can exceed 2^31, but in normal play arg12 (a perspective gradient) is far smaller; divergence requires pathological arg9..arg14 not produced by normal scene data. Same magnitude class as pix3d-3-03 which was classed latent.
> 
> FALSE-POSITIVE classes ruled out: not deob arg-scramble (traced actual flow); not a compensated pair / standing decision (not on the DO-NOT-REPORT list; this is an unguarded int32-wrap, the dominant latent class); not name-inversion; not restructured-equivalent (the missing wrap is a concrete numeric divergence, reproducible at edge inputs); not behavior-elsewhere (no guard anywhere in the path); not Java shift-mask (this is a multiply/add, not a shift count); not a stale comment.
> 
> Cannot reproduce divergence in normal play, but the mistranslation is real and reachable on edge/pathological inputs → latent is the correct, consistent severity.
>
> **Refined:** CONFIRMED (latent). TextureRaster perspective-numerator base products are computed in Go's 64-bit `int` with no `int(int32(...))` wrap, where Java computes them as 32-bit `int` that wraps mod 2^32.
> 
> Java (Client-Java @ 2e62978, src/main/java/jagex2/graphics/Pix3D.java):
> - !lowMem/high-detail path, lines 2065-2067: `var81 = arg9 + (arg12 >> 3) * var80; var82 = arg10 + (arg13 >> 3) * var80; var83 = arg11 + (arg14 >> 3) * var80;` (var80 = arg5 - projectionX, line 2064). Consumed at 2068-2071: `var84 = var83 >> 14; arg2 = var81/var84; arg3 = var82/var84`.
> - lowMem/low-detail path, lines 2239-2241: `var23 = arg9 + (arg12 >> 3) * var22; var24 = arg10 + (arg13 >> 3) * var22; var25 = arg11 + (arg14 >> 3) * var22;` (var22 = arg5 - projectionX, line 2238). Consumed at 2242-2245: `var26 = var25 >> 12; arg2 = var23/var26; arg3 = var24/var26`.
> 
> Go (pkg/jagex2/graphics/pix3d/pix3d.go):
> - LowDetail branch (Go tests LowDetail first — documented branch-order swap), lines 2088-2090: `var32 = arg9 + (arg12>>3)*var20; var33 = arg10 + (arg13>>3)*var20; var34 = arg11 + (arg14>>3)*var20;` (var20 = arg5 - CenterW3D, line 2087). Read at 2091-2095 via `var34 >> 12`.
> - High-detail fall-through, lines 2273-2275: identical expressions, read at 2276 via `var34 >> 14`.
> var32/var33/var34 are plain `int` (declared lines 2081-2083); all TextureRaster args and CenterW3D are plain `int`. No int32 guard on either pair.
> 
> This is a DISTINCT site from the already-fixed pix3d-3-03 (`var15` step, now guarded at Go line 2065 as `int(int32(...)) >> 6`). For full bug-for-bug fidelity the same `int(int32(...))` wrap should be applied to var32/var33/var34 at both sites, e.g. `var32 = int(int32(arg9 + (arg12>>3)*var20))`.
> 
> Reachability/severity: arg12/13/14 enter as int32-sign-extended gradient cross products (TextureTriangle passes `int(var21/var24/var27)` at pix3d.go:1521 etc., where those are int32 at 1454-1459), so |arg12>>3| < 2^28; var20 is screen-width class (arg5 clipped to [0, clipX] in the hclip prologue). The product exceeds 2^31 only on pathological geometry not produced by normal scene data, so Go and Java agree in normal play — latent (dominant 32-bit-wrap class), matching the latent classification of sibling pix3d-3-03.
> 
> Unit-wide parity claims spot-verified as accurate: int32 guard present on var15 (line 2065); shift-count mask `(arg7>>23)&0x1F` and low-bit masks `&0x600000`/`&0xC0000` faithful (per documented pix3d-3-01/3-02); textureTriangle cross products int32-typed at 1448-1459; LowDetail/high-detail branch-swap with correct per-path masks (>>12 vs >>14).

---

### model-A-02. loaded++ missing in NewModel2 and NewModel3

- **Unit:** `model-A`  **Java:** `Model.java:516 (Model(Model[],int): loaded++) and Model.java:617 (Model(int,Model[],boolean): loaded++) @2e62978`  **Go:** `pkg/jagex2/dash3d/model/model.go:516-617 (NewModel2 — no Loaded touch) and model.go:625-737 (NewModel3 — no Loaded touch); only increment is NewModel1 model.go:345`

Both Java multi-part merge constructors begin with `loaded++`; the Go ports NewModel2/NewModel3 never increment `Loaded`. Java increments the static decode counter in all four constructors; Go increments only in NewModel1. So `Loaded` is maintained inconsistently — it counts single-id decodes but not the two merge constructors. Reachability/severity: `Model.loaded`/`model.Loaded` is WRITE-ONLY — grep over Java tree @2e62978 finds no read of Model.loaded; grep over Go tree (incl. model_test.go) finds no read of model.Loaded. Dead counter today, so unobservable. Rated latent (not cosmetic) because the field IS ported and partially maintained; the divergence is a real mistranslation that would surface if any future code reads Loaded (e.g. a debug/stats overlay) — NewModel1-only counting under-reports by the number of multi-part models built. Fix: add `Loaded++` as the first statement of NewModel2 and NewModel3.

> **Verifier (confirmed, severity latent):** Java path moved to jagex2/dash3d/Model.java in rev-254 (finding's javaCite used the old graphics/ path but cited correct line numbers). Read Java @2e62978: `public static int loaded;` declared at Model.java:13, and `loaded++` is the FIRST statement of FIVE constructors — Model(int):372, Model(Model[],int):516, Model(int,Model[],boolean):617, Model(boolean,boolean,boolean,Model):729, Model(boolean,Model,boolean):783 (verified each via nl -ba windows). The finding says "all four constructors" but there are actually five; its two cited merge-ctor sites (516, 617) are correct. Go side (model.go): `Loaded` declared line 49, reset to 0 at line 118, and `Loaded++` appears ONLY at line 345 inside NewModel1. Read the heads of NewModel2 (line 516), NewModel3 (line 625), NewModel4 (line 739), NewModel5 (line 795) — every one opens with `var m Model` and never touches Loaded. So Go increments in 1 of 5 constructor analogues; Java increments in 5 of 5. Divergence reproduced concretely. Reachability: `git grep loaded 2e62978` over the Java tree shows the only non-write occurrences are unrelated (loadedTextures, loadedPrefetchFiles, a UI string) — Model.loaded is write-only. Whole-Go-repo grep (incl. *_test.go) for .Loaded finds only decl/reset/single-write — model.Loaded is write-only too. No design-doc decision about Loaded in LOGIC-DELTA-SCOPE-254.md / PORT-DESIGN-254.md / RENAME-MAP.md / README.md. Checked false-positive classes: not a compensated pair, not on the DO-NOT-REPORT list, not behavior-living-elsewhere (grep covered whole repo), not restructured-equivalent (the increment is simply absent). Field comment at model.go:48 ("Loaded counts every Model(int) decode") even documents the narrowed Go semantics, confirming the omission is real and not accidental-but-equivalent.
>
> **Refined:** Model.loaded decode-counter is maintained inconsistently in the Go port. Java @2e62978 increments the static counter `loaded` (declared jagex2/dash3d/Model.java:13) as the first statement of ALL FIVE Model constructors: Model(int) at Model.java:372, Model(Model[],int) at :516, Model(int,Model[],boolean) at :617, Model(boolean,boolean,boolean,Model) at :729, and Model(boolean,Model,boolean) at :783. The Go port (pkg/jagex2/dash3d/model/model.go) increments `Loaded` only in NewModel1 (model.go:345); the other four constructor analogues omit it — NewModel2 (model.go:516, == Java :516), NewModel3 (model.go:625, == Java :617), NewModel4 (model.go:739, == Java :729), and NewModel5 (model.go:795, == Java :783) all open with `var m Model` and never touch Loaded. Note: the finding scoped this to NewModel2/NewModel3 and described "all four constructors," but the Java side has five constructors and the omission additionally affects NewModel4/NewModel5 — the audit fix should add `Loaded++` as the first statement of NewModel2, NewModel3, NewModel4, AND NewModel5. Severity latent: Model.loaded/model.Loaded is write-only on both sides today (verified by grep over Java @2e62978 and the whole Go repo incl. tests — no reads), so the under-count is currently unobservable; but the field is ported, reset (model.go:118), and functionally incremented, so this is a genuine state-maintenance mistranslation that would surface as an undercount (by the number of multi-part/copy models built) the moment any future code reads Loaded (e.g. a debug/stats overlay). Not a documented deviation — no decision in LOGIC-DELTA-SCOPE-254.md / PORT-DESIGN-254.md / RENAME-MAP.md / README.md.

---

### wordfilter-B-01. firstFragmentId int32-wrap missing vs Go int64 accumulator (unreachable via sole call site)

- **Unit:** `wordfilter-B`  **Java:** `2e62978:src/main/java/jagex2/wordenc/WordFilter.java:1053-1072 (firstFragmentId); caller isBadFragment @1022-1050; sole reachable call @630 passes char[3]`  **Go:** `pkg/jagex2/wordenc/wordfilter/wordfilter.go:1228-1248 (FirstFragmentID); caller IsBadFragment @1192-1226; sole reachable call @717,731 passes make([]rune,3)`

Java firstFragmentId guards `if (arg0.length > 6) return 0;` then runs `var2 = var2*38 + <0..37>` over up to 6 chars. For length 4-6 the value can exceed 2^31-1 (max ~3.0e9 for 6 chars), so Java int arithmetic WRAPS at 32 bits, possibly yielding a negative/aliased id. Go FirstFragmentID uses `value := 0` (64-bit int) with the identical `value = value*38 + ...` and does NOT wrap, so Go can return a different id than Java for length-4..6 inputs. The id is then binary-searched against Fragments (G2-loaded, 0..65535) inside IsBadFragment, so a divergent id could in principle flip the bad-fragment verdict. This is the standard missing-int32()-wrap class. HOWEVER: the only caller of isBadFragment/firstFragmentId is filter/Filter2, which always passes a fixed 3-element array (Java `new char[3]` @618, Go `make([]rune,3)` @717) — verified by `git -C Client-Java grep firstFragmentId|isBadFragment 2e62978` and `grep -rn FirstFragmentID|IsBadFragment pkg/` (no other call sites). With length 3 the max value is 37*38^2+37*38+37 = 54871, well under 2^31, so the wrap-prone path (length 4-6) is unreachable in normal play; the `length>6` guard and wrap-prone branch are faithfully ported but dead. Severity latent: a real integer-width mistranslation, but the divergent path is currently unreachable. Defensive fix: wrap accumulator with int(int32(...)).

> **Verifier (confirmed, severity latent):** Read both sides with line numbers. Java firstFragmentId (2e62978 WordFilter.java:1053-1071) returns `int` (32-bit); the accumulator `var2 = var2*38 + <0..37>` therefore WRAPS at 32 bits for length-4..6 inputs (the `arg0.length > 6` guard @1054 caps length at 6; 6-char max ≈ 37*38^5 ≈ 3.0e9 > 2^31-1). Go FirstFragmentID (wordfilter.go:1228-1248) uses `value := 0` which is 64-bit platform int and does NOT wrap — identical formula otherwise. So the width mismatch is genuinely present: a faithful instance of the missing-int32()-wrap class.\n\nVerified the sole call site on BOTH sides. Java: `git grep -E firstFragmentId|isBadFragment 2e62978` returns only the def lines plus one caller @630 (`isBadFragment(var28)`), where var28 = `new char[3]` @618, populated only for `var29 < 3` @620. Go: `grep -rn FirstFragmentID|IsBadFragment` across the whole repo (incl. tests, all dirs) returns only the two defs plus one caller @731 (`IsBadFragment(frag)`), where `frag := make([]rune, 3)` @717, written only for `off < 3` @720. Confirmed `frag` is never appended to or reassigned (grep: only `make([]rune,3)` @717 and `frag[off]=` @721), so length is constant 3. No external/test callers exist.\n\nWith length 3, max value = 37*38^2 + 37*38 + 37 = 54871, far below 2^31-1, so no wrap can occur — for every input the function can actually receive, Java and Go return bit-identical ids, and the binary search against Fragments cannot diverge. The wrap-prone length-4..6 branch and the `length>6` guard are faithfully ported but dead. This matches FP-class #4 (restructured-but-equivalent) partially — but here the divergence is real in code, just unreachable, which is precisely the `latent` definition rather than refuted. Confirming as latent, not as a reachable bug.
>
> **Refined:** FirstFragmentID accumulator is 64-bit in Go vs 32-bit (wrapping) in Java — real width mismatch, currently unreachable (latent).\n\nGo: pkg/jagex2/wordenc/wordfilter/wordfilter.go:1228-1248 (FirstFragmentID) uses `value := 0` (platform int = 64-bit on target) with `value = value*38 + ...`. Java: 2e62978:src/main/java/jagex2/wordenc/WordFilter.java:1053-1071 (firstFragmentId) uses `int var2 = 0` (32-bit) with `var2 = var2*38 + ...`, which wraps at 2^31. Both guard `length > 6` (Go:1229, Java:1054). For length-4..6 inputs the value can exceed 2^31-1 (6-char max ≈ 3.0e9), so Java wraps to a different (possibly negative) id than Go; that id is then binary-searched against Fragments inside IsBadFragment (Go:1192-1226 / isBadFragment Java:1022-1050), so a divergent id could in principle flip the bad-fragment verdict.\n\nReachability (verified): the ONLY caller of IsBadFragment/isBadFragment is the filter inner loop, which on both sides passes a fixed length-3 array (Go `frag := make([]rune, 3)` @wordfilter.go:717, never resized/appended, written only for off<3 @720; Java `char[] var28 = new char[3]` @WordFilter.java:618, written only for var29<3 @620). The call sites are Go:731 and Java:630. Repo-wide grep confirms no other (incl. test/external) callers on either side. With length 3 the max accumulator value is 37*38^2+37*38+37 = 54871, well under 2^31, so the wrap-prone path (length 4-6) is dead in normal play and Java/Go are bit-identical for all reachable inputs.\n\nSeverity latent: genuine integer-width mistranslation faithfully ported but unreachable via the sole call site. Defensive fix to restore bug-for-bug fidelity on the dead path: wrap the accumulator update with int(int32(...)) in FirstFragmentID. Finder's cites all verified accurate.

---

### io-bzip2-02. Huffman code-length byte signedness: Go zero-extends (uint8) vs Java sign-extends (signed byte)

- **Unit:** `io-bzip2`  **Java:** `src/main/java/jagex2/io/BZip2.java:331-341 (min/max scan), :618 (perm fill ==), :630 (base count)`  **Go:** `pkg/jagex2/io/bzip2/bzip2.go:345-351 (min/max), :625 (perm fill), :637 (base count)`

Root cause: s.len[t][i] is Java byte[][] (signed -128..127) used via byte→int SIGN-extension, but Go s.Len is [][]byte (uint8) used via ZERO-extension. (a) Min/max: Java `byte minLen=32, maxLen=0; if(s.len[t][i]>maxLen)/(<minLen)` are signed byte comparisons; Go `minLen:=byte(32); maxLen:=byte(0); max(s.Len[t][i],maxLen)/min(...)` — Go min/max on byte are UNSIGNED. A negative-in-Java length (byte 128..255) would set minLen in Java but maxLen in Go. (b) createDecodeTables perm: Java `if(length[j]==i)` sign-extends; Go `if int(length[j])==i` zero-extends. (c) base count: Java `base[length[i]+1]++` sign-extends (byte 255→index 0) vs Go `base[length[i]+1]++` zero-extends (index 256). Divergence reachable ONLY when a stored code length is ≥128, i.e. the delta-coded curr at BZip2.java:315/bzip2.go:329 reached 128..255. Well-formed bzip2 caps code lengths at ≤20 (<BZ_MAX_CODE_LEN=23), all positive → sign/zero extension coincide, no divergence with valid cache data (server-controlled trusted input). Input condition: adversarial/malformed stream w/ code length ≥128. Not reachable in normal play. Faithful fix: type Len/length/minLen/maxLen as int8 or sign-extend on use.

> **Verifier (confirmed, severity latent):** Read both sides with line numbers.
> 
> TYPES:
> - Java BZip2State.java:117 `public byte[][] len = new byte[6][258]` (SIGNED byte), :129 `public int[] minLens = new int[6]`, :120/123/126 limit/base/perm = `int[6][258]`.
> - Go bzip2state.go:49 `Len [][]byte` (UNSIGNED uint8), :50-53 Limit/Base/Perm `[][]int` rows sized 258 (lines 78-87), MinLens `[]int`. BZ_MAX_CODE_LEN=23 (line 7).
> Type divergence confirmed: Java signed byte vs Go unsigned uint8 for the code-length storage.
> 
> (a) MIN/MAX SCAN — Java BZip2.java:331-341 `byte minLen=32; byte maxLen=0; if(s.len[t][i]>maxLen) maxLen=...; if(s.len[t][i]<minLen) minLen=...` are SIGNED byte comparisons. Go bzip2.go:345-351 `minLen:=byte(32); maxLen:=byte(0); maxLen=max(s.Len[t][i],maxLen); minLen=min(s.Len[t][i],minLen)` — Go max/min on byte are UNSIGNED. For a stored length whose byte value is 128..255 (Java sees -128..-1): Java sets minLen (e.g. -128<32), Go instead sets maxLen (e.g. 128>0) and leaves minLen at 32. Divergent. Confirmed.
> 
> (b) PERM FILL — Java BZip2.java:618 `if(length[j]==i)` sign-extends length (loop i runs minLen..maxLen which in Java can be negative). Go bzip2.go:625 `if int(length[j])==i` zero-extends (loop i is int over int(minLen)..int(maxLen), always >=0 in Go). Different perm composition for negative-in-Java lengths. Confirmed.
> 
> (c) BASE COUNT — Java BZip2.java:630 `base[length[i]+1]++` sign-extends: byte 255->-1->index 0 (no throw); byte 128..254 -> index -127..-1 -> Java ArrayIndexOutOfBoundsException. Go bzip2.go:637 `base[length[i]+1]++` zero-extends: byte 254->index 255, byte 255->index 256, all fit in the 258-sized row, no panic. Both Java (base int[6][258], :123) and Go (row 258, bzip2state.go:83) size rows 258 so positive indices up to ~128 are in-bounds. Confirmed divergence; Go is also more lenient (no AIOOBE where Java throws for 128..254).
> 
> REACHABILITY: BZip2.java:309 `curr=getBits(5,s)` (0..31), then ++/-- in the loop with NO clamp (verified :311-326), stored as (byte)curr at :315 / byte(curr) at Go:329. So curr can climb to >=128 only on malformed/adversarial bitstreams. bzip2.Read is called only from JagFile (jagfile.go:36,80) i.e. cache/on-demand archive decompression = server-controlled trusted input. Not reachable in normal play with well-formed bzip2 (code lengths capped <=20). Finding's scope accurate.
> 
> FALSE-POSITIVE CLASSES CHECKED: not a stale-cite/comment issue (real type + operation divergence); not shift-mask; not restructured-equivalent (concrete diverging input value byte>=128 exhibited); not behavior-elsewhere (single decompress impl); not a documented standing decision (bzip2 not in DO-NOT-REPORT list; checked the list). go min/max-on-byte being unsigned is the actual root cause, genuine.
>
> **Refined:** Confirmed (latent). BZip2 Huffman code-length storage diverges in signedness between Java (signed byte) and Go (unsigned uint8), causing different decode tables on malformed input.
> 
> Types: Java `BZip2State.len` is `byte[][]` (signed, BZip2State.java:117); Go `bzip2state.BZip2State.Len` is `[][]byte` (unsigned uint8, bzip2state.go:49). The decode-length is written as `(byte)curr` (BZip2.java:315) / `byte(curr)` (bzip2.go:329) with no clamp on curr (BZip2.java:309-326).
> 
> Three consumption sites zero-extend in Go where Java sign-extends:
> (a) Min/max scan — Java BZip2.java:331-341 uses `byte minLen=32, maxLen=0` with signed `>`/`<` comparisons; Go bzip2.go:345-351 uses `minLen:=byte(32); maxLen:=byte(0)` with `max(...)`/`min(...)`, which are UNSIGNED on Go byte. A length byte of 128..255 (Java: -128..-1) sets minLen in Java but instead sets maxLen in Go.
> (b) createDecodeTables perm fill — Java BZip2.java:618 `if(length[j]==i)` (sign-extended, i can be negative) vs Go bzip2.go:625 `if int(length[j])==i` (zero-extended, i>=0).
> (c) base count — Java BZip2.java:630 `base[length[i]+1]++` sign-extends (byte 255->index 0; bytes 128..254 -> negative index -> Java AIOOBE) vs Go bzip2.go:637 `base[length[i]+1]++` zero-extends (indices 129..256, all in the 258-sized row -> no panic). Both rows are sized 258 (Java BZip2State.java:123; Go bzip2state.go:83), so Go never panics and silently builds a different table.
> 
> Reachability: divergence requires a stored code length >= 128, i.e. curr climbing to 128..255 on a malformed/adversarial bzip2 stream. bzip2.Read is only called from JagFile (jagfile.go:36,80) on server/cache-controlled trusted archives; well-formed bzip2 caps code lengths at <=20 (all positive, sign==zero extension) so there is no divergence in normal play. Hence latent.
> 
> Faithful fix: type `Len`/`length`/`minLen`/`maxLen` as `int8` (and the perm/base index uses) to match Java's signed byte->int promotion, or sign-extend on each use.

---

### config-B-04. IfType.CacheModel cache key not int32-wrapped while LoadModel key is

- **Unit:** `config-B`  **Java:** `src/main/java/jagex2/config/IfType.java:467 & 491 (loadModel get/put key `(long)((arg0<<16)+arg1)`), :500 (cacheModel put key `(long)((arg3<<16)+arg1)`) @2e62978`  **Go:** `pkg/jagex2/config/iftype/iftype.go:387,407 (LoadModel `int64(int32((arg0<<16)+arg1))`) vs :416 (CacheModel `(int64(typ)<<16)+int64(id)`)`

Java computes both keys as `(x<<16)+y` in 32-bit int (wraps) then widens to long. Go's LoadModel faithfully reproduces the wrap with `int64(int32(...))` on both the get and put; Go's CacheModel instead does pure 64-bit arithmetic `int64(typ)<<16 + int64(id)` with no int32 truncation. If `(type<<16)+id` ever exceeded int32 positive range, CacheModel would store under a different key than LoadModel looks up, causing a cache miss (redundant rebuild, never a wrong render). Reachability: NONE at any server-reachable value — model types are 1..6 and ids fit in 16 bits, so `(type<<16)+id` <= ~458751 << INT32_MAX, making both key forms identical for all valid inputs. Recorded as latent for the key-computation inconsistency only; not a behavior bug in normal play.

> **Verifier (confirmed, severity latent):** Read Java IfType.java @2e62978 (loadModel lines 466-494, cacheModel 497-502) via git show | nl -ba, and Go pkg/jagex2/config/iftype/iftype.go:382-418.
> 
> Java side: arg0/arg1/arg3 are all `int` (32-bit). loadModel computes the key as `(long)((arg0 << 16) + arg1)` on both the get (IfType.java:467) and put (IfType.java:491); cacheModel computes `(long)((arg3 << 16) + arg1)` (IfType.java:500). In all three, the `(x<<16)+y` add executes in 32-bit int (wraps on overflow) and is only widened to long AFTER the add.
> 
> Go side: LoadModel faithfully reproduces the 32-bit wrap on both the get (iftype.go:387 `int64(int32((arg0 << 16) + arg1))`) and put (iftype.go:407 `int64(int32((arg0<<16)+arg1))`), with an explanatory comment at 384-386. But CacheModel (iftype.go:416) uses pure 64-bit arithmetic `(int64(typ)<<16)+int64(id)` — no int32 truncation. So the key-computation forms are genuinely inconsistent between LoadModel and CacheModel, and CacheModel deviates from Java's 32-bit wrap.
> 
> Reachability traced concretely: cacheModel/CacheModel has exactly one non-test call site each — Java Client.java:10363 `IfType.cacheModel(0, var13, 5)` (id=0, type=5) and Go client.go:5756 `iftype.CacheModel(var10, 0, 5)` (id=0, type=5). Both forms yield 327680, identical. More generally modelType is only ever set to 1..5 (Java IfType.java:358 and Go client.go ModelType= sites 1,2,3,4,5), and modelId max = `(255-1 << 8) + 255` = 65279 (IfType.java:359, g1() is an unsigned byte 0-255). Worst-case key `(5<<16)+65279` = 393215 << INT32_MAX (2147483647), so the int32 wrap is never triggered and both key expressions are arithmetically identical for the entire server-reachable domain. LruCache key type is int64 (lrucache.go:32,57), so no further truncation.
> 
> This matches false-positive class 4 review (restructured-but-equivalent) but is NOT dismissable: the two key expressions are not equivalent in general — they only happen to coincide for all reachable inputs. It is a faithfulness deviation (CacheModel should be `int64(int32((typ<<16)+id))` to mirror Java's wrap), reachable only on unreachable edge values, hence latent. Never a wrong render — at worst a redundant cache miss/rebuild. Finder's cites are all accurate (Java 467/491/500; Go 387/407/416).
>
> **Refined:** IfType.CacheModel does not int32-wrap its cache key, unlike LoadModel and unlike the Java source. Java (IfType.java:467 get, :491 put, :500 cacheModel put, @2e62978) computes every modelCache key as `(long)((x << 16) + y)` — the `(x<<16)+y` add runs in 32-bit `int` (wraps on overflow) and is widened to `long` only afterward. Go's LoadModel faithfully reproduces this on both get and put: `int64(int32((arg0 << 16) + arg1))` (pkg/jagex2/config/iftype/iftype.go:387, :407). Go's CacheModel instead computes the key in pure 64-bit with no truncation: `(int64(typ)<<16)+int64(id)` (iftype.go:416). The two forms diverge only when `(type<<16)+id` exceeds INT32_MAX positive range. That is unreachable: model types are 1..5 (set at client.go ModelType= sites and Java IfType.java:358) and modelIds are ≤65279 (`(var28-1 << 8) + g1()`, both unsigned bytes, IfType.java:359), so the worst-case key 393215 is far below INT32_MAX and both forms coincide for all valid inputs. The single live call pair (Java Client.java:10363 `cacheModel(0, var13, 5)`; Go client.go:5756 `CacheModel(var10, 0, 5)`) both yield key 327680. Impact: at most a cache miss → redundant model rebuild, never a wrong render or desync. Latent only. Suggested faithful fix: change iftype.go:416 to `ModelCache.Put(int64(int32((typ<<16)+id)), m)` to match Java's 32-bit wrap and stay consistent with LoadModel.

---

### dash3d-collision-01. CollisionMap constructor maps params to size fields in reversed order vs Java

- **Unit:** `dash3d-collision`  **Java:** `src/main/java/jagex2/dash3d/CollisionMap.java:23-28 (ctor: this.sizeX = arg2; this.sizeZ = arg1; this.flags = new int[this.sizeX][this.sizeZ])`  **Go:** `pkg/jagex2/dash3d/collisionmap.go:11-21 (NewCollisionMap(x,z): m.SizeX = x; m.SizeZ = z; Flags = make([][]int, m.SizeX) of make([]int, m.SizeZ))`

Java CollisionMap(int arg1, int arg2) assigns 1st param->sizeZ and 2nd param->sizeX (sizeX=arg2, sizeZ=arg1). Go NewCollisionMap(x, z) assigns 1st param->SizeX and 2nd param->SizeZ. The parameter-to-field mapping is transposed. NOT documented in RENAME-MAP.md or LOGIC-DELTA-SCOPE-254.md as an intentional swap. Reachability: the ONLY constructor call in each codebase is new CollisionMap(104,104) at Client.java:1559 and dash3d.NewCollisionMap(104,104) at client.go:6202. Since both args are 104, SizeX==SizeZ==104 regardless of order, so there is no observable difference today. It becomes a real divergence only if a non-square map is ever constructed (a!=b): Go would build an a-by-b grid, Java a b-by-a grid, making Reset's boundary marking and every Size-bound loop diverge. baseX/baseZ (Go OffsetX/OffsetZ) are never assigned in either codebase (the this.baseX/baseZ writes in Client.java are a different Client field), so offsets stay 0 and do not interact. Classified latent.

> **Verifier (confirmed, severity latent):** Read Java CollisionMap ctor via `git show 2e62978:.../CollisionMap.java | nl -ba` lines 23-28: `public CollisionMap(int arg1, int arg2) { this.sizeX = arg2; this.sizeZ = arg1; this.flags = new int[this.sizeX][this.sizeZ]; }` — so 1st param→sizeZ, 2nd param→sizeX, and Flags is allocated [sizeX][sizeZ]. Read Go ctor at pkg/jagex2/dash3d/collisionmap.go:11-21: `func NewCollisionMap(x, z int) { m.SizeX = x; m.SizeZ = z; m.Flags = make([][]int, m.SizeX); ... m.Flags[i] = make([]int, m.SizeZ) }` — 1st param→SizeX, 2nd param→SizeZ. The parameter-to-field mapping is genuinely transposed (verified by reading the assignment statements themselves, not param names — defeats the deob-arg-scramble false-positive class).
> 
> Checked all call sites: `git grep "new CollisionMap"` → only Client.java:1559 `new CollisionMap(104, 104)`; `grep NewCollisionMap` → only client.go:6202 `NewCollisionMap(104,104)` (production) and logout_test.go:25 `NewCollisionMap(0,0)` (test). No `CollisionMap{...}` struct literals exist. Both production call sites pass symmetric args with NO caller-side arg-order swap, so this is NOT a compensated pair (a compensated pair would swap at the caller to offset the ctor; here neither swaps and the ctor itself differs). Not in the DO-NOT-REPORT compensated-pair list.
> 
> Consequence trace: for input (a,b) with a!=b — Java yields sizeX=b, sizeZ=a, Flags shape [b][a]; Go yields SizeX=a, SizeZ=b, Flags shape [a][b]. Reset() (Java :30-41 / Go :23-33) is structurally identical (sizeX outer, sizeZ inner, boundary on x==0||z==0||x==sizeX-1||z==sizeZ-1), so with transposed field values it would mark a transposed boundary rectangle, and every Size-bound loop in AddLoc/DelLoc/TestLoc would diverge. Masked today because both maps are 104×104 (a==b).
> 
> baseX/baseZ (Go OffsetX/OffsetZ): Java CollisionMap.baseX/baseZ only ever read (arg - this.baseX) plus `= 0` field initializers, never written (git grep confirms). Go OffsetX/OffsetZ writes found by grep are all in loctype.go on a different struct (LocType), never on CollisionMap. So offsets stay 0 in both and don't interact — finding's offset claim is accurate.
> 
> Design docs: only CollisionMap mention in LOGIC-DELTA-SCOPE-254.md is :344 (addLoc trailing bool drop) — unrelated to the ctor. No documentation of the ctor transposition in RENAME-MAP.md, LOGIC-DELTA-SCOPE-254.md, PORT-DESIGN-254.md, or README.md. Not an intentional documented swap.
> 
> All false-positive classes ruled out. The finding's cites are accurate and the divergence is real but unreachable on current inputs.
>
> **Refined:** CONFIRMED (latent). CollisionMap constructor maps its two parameters to the size fields in the opposite order from Java.
> 
> Java (src/main/java/jagex2/dash3d/CollisionMap.java:23-28 @2e62978):
> ```
> public CollisionMap(int arg1, int arg2) {
>     this.sizeX = arg2;   // 2nd param -> sizeX
>     this.sizeZ = arg1;   // 1st param -> sizeZ
>     this.flags = new int[this.sizeX][this.sizeZ];
>     this.reset();
> }
> ```
> Go (pkg/jagex2/dash3d/collisionmap.go:11-21):
> ```
> func NewCollisionMap(x, z int) *CollisionMap {
>     m.SizeX = x          // 1st param -> SizeX
>     m.SizeZ = z          // 2nd param -> SizeZ
>     m.Flags = make([][]int, m.SizeX)   // outer = SizeX
>     for i := range m.Flags { m.Flags[i] = make([]int, m.SizeZ) } // inner = SizeZ
>     m.Reset()
> }
> ```
> Java binds (arg1→sizeZ, arg2→sizeX); Go binds (1st→SizeX, 2nd→SizeZ). The param-to-field mapping is transposed.
> 
> Not a compensated pair: the only constructor calls are Java Client.java:1559 `new CollisionMap(104, 104)` and Go client.go:6202 `dash3d.NewCollisionMap(104, 104)` (plus test-only logout_test.go:25 `NewCollisionMap(0, 0)`). No `CollisionMap{...}` struct literals exist. Both production call sites pass symmetric args with NO caller arg-order swap, so nothing offsets the ctor difference. Not in the DO-NOT-REPORT compensated-pair list and not documented in RENAME-MAP.md / LOGIC-DELTA-SCOPE-254.md / PORT-DESIGN-254.md / README.md (the only documented CollisionMap delta, LOGIC-DELTA-SCOPE-254.md:344, is the unrelated addLoc bool-param drop).
> 
> No observable difference today because every map is square (104×104), so SizeX==SizeZ regardless of order, and Reset()'s boundary marking plus all Size-bound loops produce identical results. It becomes a real divergence only if a non-square map (a != b) is ever constructed: Java would build a [b][a] Flags grid (sizeX=b, sizeZ=a) while Go builds [a][b] (SizeX=a, SizeZ=b), transposing Reset's boundary rectangle and desyncing AddLoc/DelLoc/TestLoc/AddWall bound checks. OffsetX/OffsetZ (Java baseX/baseZ) are never assigned in either codebase (the loctype.go OffsetX/OffsetZ writes are on a different struct), so they stay 0 and do not interact. Fix: swap the assignments in NewCollisionMap to `m.SizeX = z; m.SizeZ = x` (or swap the param names) so the 1st param maps to SizeZ and the 2nd to SizeX, matching Java. Severity latent: correct mistranslation, unreachable on current (always-square) inputs.

---

### signlink-02. wavesave/wavereplay/midisave return-value + 2000000-byte guard not reproduced by in-memory audio seam

- **Unit:** `signlink`  **Java:** `sign/signlink.java:255-268 (wavesave), 270-279 (wavereplay), 281-290 (midisave) @2e62978; caller Client.java:3356-3385 (soundsDoQueue), 3366/3373 @2e62978`  **Go:** `pkg/jagex2/client/client.go:5883-5889 (SaveWave always return true), :5891-5894 (ReplayWave always return true), :3534-3538 (saveMidi); pkg/jagex2/sound/audio/wave_native.go:36-65 (PlayWave/ReplayWave, no size cap, no busy slot)`

Java wavesave returns false when arg1>2000000 OR when savereq!=null (slot busy); wavereplay returns false when savereq!=null; midisave silently drops when arg1>2000000 || savereq!=null. Those return values drive control flow in soundsDoQueue (Client.java:3366 `if(this.saveWave(...)){...played...}else{var3=true;}` and 3373 `else if(!this.replayWave()){var3=true;}`): var3==true keeps the wave queued (waveDelay[var2]=-5, retried next cycle) instead of dequeuing it. The Go port relocates these into Client.SaveWave/ReplayWave which ALWAYS return true and call audio.PlayWave/ReplayWave (fire-and-forget into oto), and midisave -> signlink.SetMidiTrack (latest-wins single slot). Two concrete divergences: (1) the 2000000-byte size cap is dropped entirely (oversized samples always accepted) — reachable only on edge values since Wave.generate emits <=~441KB (wave_native.go:103) and game MIDI is well under 2MB, so latent. (2) the busy-slot retry path is gone — under rapid SFX bursts Java would re-queue, Go always plays/dequeues; this is an inherent consequence of the documented disk-rotation -> in-memory audio seam restructure (the savereq single slot no longer exists) and cannot be fixed without reintroducing the disk protocol. Recorded latent: the seam relocation itself is intentional/documented (LOGIC-DELTA-SCOPE-245 §WS5 keeps the Go audio seam), but the publisher-side reconciliation doc does not call out preserving the 2MB cap, so item (1) is the only piece not explicitly subsumed.

> **Verifier (confirmed, severity latent):** Read both sides with line numbers.
> 
> JAVA (@2e62978, all finder cites accurate):
> - sign/signlink.java:255-268 wavesave — returns false if arg1>2000000 OR savereq!=null; else rotates wavepos, sets savereq="sound"+wavepos+".wav", returns true.
> - :270-279 wavereplay — returns false if savereq!=null; else sets savereq, returns true.
> - :281-290 midisave — `if (arg1>2000000 || savereq!=null) return;` then rotates midipos, sets savereq="jingle"+midipos+".mid".
> - jagex2/client/Client.java:1450-1458 saveWave(arg0==null?true:signlink.wavesave)/replayWave wrappers; :1431-1433 saveMidi -> midifade + midisave; :3356-3392 soundsDoQueue: var3 retry flag set when saveWave (3366) / replayWave (3373) returns false -> waveDelay[var2]=-5 (re-queue), else dequeue+shift.
> 
> GO:
> - client.go:5883-5889 SaveWave: if arg0==nil return true; else audio.PlayWave(arg0[:arg1]); ALWAYS returns true. No 2000000 cap, no savereq busy check.
> - client.go:5891-5894 ReplayWave: audio.ReplayWave(); ALWAYS returns true.
> - client.go:3532-3539 SaveMidi: SetMidiFade + signlink.SetMidiTrack; NO size cap.
> - signlink.go:393-398 SetMidiTrack: single slot (Midi="play", MidiData=data), latest-wins; no busy check.
> - client.go:8139-8180 soundsDoQueue: faithfully reproduces var4/WaveDelay=-5/dequeue control flow, BUT since SaveWave/ReplayWave never return false, the busy-slot re-queue branch is unreachable for that reason.
> - wave_native.go:36-65 PlayWave/ReplayWave: fire-and-forget into oto, no size cap, no busy slot.
> 
> REACHABILITY: wave.go:32 WaveBytes=make([]byte,441_000); GetWave (wave.go:102-120) writes the 44-byte RIFF header + length into that fixed buffer, so var5.Pos <= ~441KB, never >2,000,000. Game MIDI tracks are well under 2MB. So the dropped cap is reachable only on abnormal/oversized cache input -> latent.
> 
> DOCS: LOGIC-DELTA-SCOPE-245.md:221-237 (WS4/WS5) + :286-300 DO-NOT-PORT — the Go in-memory audio seam (pkg/jagex2/sound/audio/, rev-244 WS5) is kept and the disk-rotation consumer is NOT ported. The savereq single slot exists ONLY for the sound<pos>.wav / jingle<pos>.mid disk protocol, so divergence (2) (busy-slot retry path gone) is an inherent, DOCUMENTED consequence of the seam. The 2000000-byte publisher guard is NOT explicitly called out as preserved in WS4/WS5 or any 254 doc; it is the only piece not subsumed.
> 
> False-positive classes checked: not arg-scramble (control flow traced through callee bodies); not a compensated pair / standing decision (audio seam is listed as seam-justified, but the 2MB cap is not); behavior-lives-elsewhere checked (grepped whole repo for the cap — absent in client.go, signlink.go, wave_native.go, wave.go); restructured-but-equivalent — divergence is concrete (a >2MB sample is rejected by Java, accepted by Go), only the input value to reach it is out of normal range.
> 
> The finder's reachability cite "wave_native.go:103" for the ~441KB figure is slightly off — the actual fixed-buffer bound is wave.go:32 (WaveBytes=make([]byte,441_000)); corrected below.
>
> **Refined:** CONFIRMED (latent). Java publisher guards in sign/signlink.java @2e62978 are not reproduced by the Go in-memory audio seam:
> - wavesave (signlink.java:255-268) returns false on arg1>2000000 OR savereq!=null;
> - wavereplay (signlink.java:270-279) returns false on savereq!=null;
> - midisave (signlink.java:281-290) silently drops on arg1>2000000 || savereq!=null.
> Those return values drive soundsDoQueue (Client.java:3366 `if(saveWave(...)){...}else{var3=true;}`, 3373 `else if(!replayWave()){var3=true;}`): var3==true keeps the wave queued (waveDelay[var2]=-5, retried next cycle) instead of dequeuing.
> 
> Go: Client.SaveWave (client.go:5883-5889) and ReplayWave (client.go:5891-5894) ALWAYS return true and call audio.PlayWave/ReplayWave (wave_native.go:36-65, fire-and-forget into oto, no size cap, no busy slot); SaveMidi (client.go:3532-3539) -> signlink.SetMidiTrack (signlink.go:393-398, single slot latest-wins, no size cap). The Go soundsDoQueue (client.go:8139-8180) faithfully ports the var4/WaveDelay=-5/dequeue logic, but the busy-slot re-queue branch is now unreachable.
> 
> Two divergences:
> (1) The 2,000,000-byte size cap is dropped entirely (oversized samples/MIDI always accepted). NOT explicitly subsumed by the documented seam decision. Reachable only on edge values: Wave.GetWave writes into the fixed WaveBytes=make([]byte,441_000) buffer (pkg/jagex2/sound/wave/wave.go:32, header+data at wave.go:102-120), so SFX var5.Pos <= ~441KB, and game MIDI is well under 2MB — hence latent, not a normal-play bug.
> (2) The savereq busy-slot retry path is gone. This is INTENTIONAL/DOCUMENTED: savereq exists only for the sound<pos>.wav / jingle<pos>.mid disk-rotation protocol, which LOGIC-DELTA-SCOPE-245.md:221-237 & :286-300 (WS4/WS5 + DO-NOT-PORT) replace with the kept Go in-memory audio seam; it cannot be reintroduced without the disk protocol.
> 
> Net: item (2) is documented/intentional; item (1) (the missing >2,000,000 publisher guard) is the only un-subsumed residue — a faithful-fidelity gap reachable only on abnormal oversized cache input. Severity latent. (Minor cite correction: the ~441KB bound is wave.go:32, not wave_native.go:103.)


---

## Refuted by verification (2)

### client-04-03. UpdateOrbitCamera (followCamera) has Go-only `c.PacketSize += arg0` not present in Java *(finder severity: latent)*

- **Unit:** `client-04`  **Java:** `Client.java:4123-4124 (followCamera: yaw assign then pitch assign, nothing between; method has no packetSize statement, no logical param)`  **Go:** `pkg/jagex2/client/client.go:6761 (`c.PacketSize += arg0`, between yaw line 6760 and pitch line 6762); sole caller pkg/jagex2/client/client.go:8334 passes 0`

> **Verifier (refuted):** REFUTED — documented intentional deob-param sink, not a latent bug; reclassify as cosmetic (harmless dead-write). Go UpdateOrbitCamera (pkg/jagex2/client/client.go:6725) inserts `c.PacketSize += arg0` (line 6761) between the yaw assign (6760) and pitch assign (6762), which rev-254 Java followCamera() does NOT contain (git show 2e62978:src/main/java/jagex2/client/Client.java: lines 4123 yaw / 4124 pitch are adjacent; signature line 4095 `public void followCamera()` takes no logical param — the `@ObfuscatedName("client.f(I)V")` int is obfuscation garbage stripped by the deob). The finding's factual observations are accurate, but this is the project's documented deob-garbage-param sink pattern, already audited and ruled intentional in the prior rev-245 audit (audit-245/units/client-05.md, finding client-05-02 [intentional] "Deob garbage int/byte params preserved as no-op `PacketSize += argN`", which names UpdateOrbitCamera + StartForceMovement explicitly and confirms call sites pass 0). The Go port reconstructs the obfuscated descriptor's dummy int as `arg0 int` and sinks it into PacketSize; the sole caller repo-wide is `c.UpdateOrbitCamera(0)` (client.go:8334) and Java's sole caller `this.followCamera()` (Client.java:2954) passes nothing. With the only caller hardcoding 0 and no alternate call path, the add is a provable no-op — not "latent reachable on edge values" (the finder itself concedes the write is inert). The same sink recurs for StartForceMovement (client.go:4280, callers 4231/4252 pass 0) and the menu builder (client.go:2074), all the same documented class. PacketSize is the live network length field (set client.go:10453, consumed 10483) but no nonzero value can reach this site, so the protocol-desync hazard is not realizable. Do not spend a fix cycle.

---

### client-11-02. doScrollbar scroll-position multiply not int32-wrapped (Go 64-bit vs Java 32-bit) *(finder severity: latent)*

- **Unit:** `client-11`  **Java:** `Client.java:9736-9742 @2e62978`  **Go:** `pkg/jagex2/client/client.go:7242-7246`

> **Verifier (refuted):** REFUTED. Finding client-11-02 claims doScrollbar's scroll-position multiply `(scrollSize - height) * var11 / var12` overflows int32 in Java but not in Go's 64-bit int. Cites verified: Java Client.java:9742 @2e62978 vs Go pkg/jagex2/client/client.go:7246; both fields are `int`; scrollSize/height are g2-capped at 65535. The arithmetic-width difference is real in source, but the overflow is mathematically unreachable. var11 is bounded by the branch entry guard (arg2 < arg7+height-16) to var11 < height, and the only caller that feeds server data (client.go:1777 / Java Client.java:9933) is gated by `scrollSize > height`, so 0 < (scrollSize - height) < scrollSize ≤ 65535. Hence the intermediate product (scrollSize-height)*var11 < (scrollSize-height)*height, whose maximum over height∈(0,scrollSize) is scrollSize²/4 = 65535²/4 ≈ 1.073e9 — well under the int32 max 2.147e9. The two factors are coupled (a large (scrollSize-height) forces a small height, which bounds var11), so they cannot both approach 65535 simultaneously; the finding's ~4.3e9 figure double-counts the ranges. The chat caller (client.go:4631 / Java Client.java:5130) uses height=77 and ChatScrollHeight in the low thousands, also far below overflow. Java and Go therefore always produce identical scrollPosition values; no divergence exists on any reachable input. (Note: a separate divide-by-zero/negative var12 concern exists but is identical in both languages and outside this finding.)


---

## Cosmetic (54) — recorded, not verified, fix opportunistically

- **client-01-01** Stale Java cite on SetMidiVolume doc comment
  - `src/main/java/jagex2/client/Client.java:1442-1448 @2e62978 (setMidiVolume(int arg1, boolean arg2))` ↔ `pkg/jagex2/client/client.go:3569 (doc comment) / body 3570-3575`
- **client-01-02** Stale Java cite on SetWaveVolume doc comment
  - `src/main/java/jagex2/client/Client.java:1460-1463 @2e62978 (setWaveVolume(int arg1) -> signlink.wavevol = arg1)` ↔ `pkg/jagex2/client/client.go:5897 (doc comment) / body 5898-5900`
- **client-02-02** refresh()->RefreshFunc rename lacks a // Java: cite at its definition
  - `Client.java:2028-2031 refresh() { this.redrawFrame = true; }` ↔ `client.go:6859-6861 RefreshFunc() { c.RedrawFrame = true }; caller gameshell.go:129`
- **client-03-01** Stale pIsaac(N) opcode values in GameLoop/MapBuild inline comments
  - `Client.java:2828 (pIsaac 142), 2908 (176), 2968 (144), 3028/3154/3172/3182/3186 (239) @2e62978` ↔ `pkg/jagex2/client/client.go:8197, 8286, 8347, 8409, 9497, 9519, 9529, 9532`
- **client-03-02** Stale Java line-number and arg-order cites in GameLoop/UpdateSceneState/wave-block comments
  - `Client.java:2822-2824, 3069-3090, 3362 @2e62978` ↔ `pkg/jagex2/client/client.go:8129-8137, 8150, 9396-9418`
- **client-04-04** Stale `// Java: pIsaac(8)` comments in HandleChatSettingsInput; actual rev-254 opcode is pIsaac(129)
  - `Client.java:4009, 4019, 4029 (this.out.pIsaac(129) // CHAT_SETMODE)` ↔ `pkg/jagex2/client/client.go:1962, 1971, 1980 (`c.Out.P1Isaac(CLIENTPROT_CHAT_SETMODE) // Java: pIsaac(8)`)`
- **client-04-05** Stale `// Java: pIsaac(245) Client.java:4350` comment in CloseModal; actual rev-254 is pIsaac(58) at Client.java:4050
  - `Client.java:4050 (this.out.pIsaac(58) // CLOSE_MODAL)` ↔ `pkg/jagex2/client/client.go:967 (`c.Out.P1Isaac(CLIENTPROT_CLOSE_MODAL) // Java: pIsaac(245) Client.java:4350`)`
- **client-05-01** HandleInputKey stale `// Java: pIsaac(NN)` comments cite 245.2 opcode numbers
  - `Client.java:4268 pIsaac(226), 4307 pIsaac(214), 4320 pIsaac(129), 4352 pIsaac(161), 4383 pIsaac(86), 4446 pIsaac(83) @2e62978` ↔ `pkg/jagex2/client/client.go:2297,2309,2343,2383,2445 (and 2252)`
- **client-05-02** UpdateClientPlayer/UpdateClientNpc stale `clearRoute` comments (Java 254 renamed to abortRoute)
  - `Client.java:4541 (player) and 4550 (localPlayer extra bounds) call arg1.abortRoute(); ClientEntity.java:256 declares abortRoute() @2e62978` ↔ `pkg/jagex2/client/client.go:4217, 4226 (UpdateClientPlayer), 4247 (UpdateClientNpc)`
- **client-05-03** LoadTitleImages allocates FlameBuffer3 before FlameBuffer2 (order reversed vs Java); LoadTitle comment now misleading
  - `Client.java:4997-5001 @2e62978: flameBuffer0, flameBuffer1, updateFlameBuffer(null), flameBuffer2 (5000), flameBuffer3 (5001)` ↔ `pkg/jagex2/client/client.go:3406-3410 (alloc), 3417 (go RenderFlames); comment at 7156-7159`
- **client-05-05** StartForceMovement has extra `c.PacketSize += arg1` not in Java exactMove2 (no-op, documented idiom)
  - `Client.java:4586-4611 exactMove2 (sig client.a(BLz;)V) — body never touches any psize field @2e62978` ↔ `pkg/jagex2/client/client.go:4280 `c.PacketSize += arg1`; callers at 4231 and 4252 both pass 0`
- **client-06-01** DrawGame: stale Java cite on TUTORIAL_CLICKSIDE pIsaac
  - `Client.java:5177 (this.out.pIsaac(201); inside redrawSideicons flashing-tab block 5174-5179)` ↔ `pkg/jagex2/client/client.go:4671`
- **client-06-02** TitleScreenDraw: stale Java cite for fileserver status line
  - `Client.java:5019 (fontPlain11.centreStringTag(7711145, var4/2, onDemand.message, var6, true))` ↔ `pkg/jagex2/client/client.go:3591-3592`
- **client-06-03** GetTopLevelCutscene (roofCheck2): stale/mixed-vintage doc cites
  - `Client.java:5577-5580 (roofCheck2(), ObfuscatedName client.i(Z)I)` ↔ `pkg/jagex2/client/client.go:1548-1554`
- **client-07-02** Stale `// Java: pIsaac(N)` opcode comments in TryMove (constants are correct)
  - `Client.java:6480,6485,6490 @2e62978` ↔ `pkg/jagex2/client/client.go:8648,8652,8656 (constants at clientprot.go:94,83,81)`
- **client-09-01** Stale Java identifier/line in GetPlayerExtended2 0x40-block comment (visible vs ready)
  - `Client.java:8201 (if (arg4.name != null && arg4.ready))` ↔ `pkg/jagex2/client/client.go:11655-11657`
- **client-10-01** Stale `// Java: pIsaac(N)` comment cites in UseMenuOption (values correct, comments wrong)
  - `Client.java:8568-9202 (useMenuOption) @2e62978 — each pIsaac(N) opcode literal` ↔ `pkg/jagex2/client/client.go:4872-5494 (UseMenuOption) — inline `// Java: pIsaac(N) Client.java:NNNN` comments`
- **client-12-01** Stale opcode cites in HandleInterfaceAction
  - `Client.java:10606 (pIsaac(13) IF_PLAYERDESIGN), 10623 (pIsaac(203) REPORT_ABUSE)` ↔ `pkg/jagex2/client/client.go:6052, 6072`
- **client-12-02** Stale 244-vintage Java line cites in DrawChatback
  - `Client.java:10664-10791 (drawChat); message loop 10686-10768; color 0x7e3200 @10763; name/typed block 10775-10782` ↔ `pkg/jagex2/client/client.go:10298, 10370, 10381`
- **client-12-03** Stale opcode cites in friend/ignore list mutators
  - `Client.java:10979 (pIsaac(9) FRIENDLIST_ADD), 11000 (pIsaac(84) FRIENDLIST_DEL), 11032 (pIsaac(189) IGNORELIST_ADD), 11049 (pIsaac(193) IGNORELIST_DEL)` ↔ `pkg/jagex2/client/client.go:7765, 8801, 1203, 10024`
- **client-13-02** FlameBuffer2 <-> FlameBuffer3 naming swap in updateFlames/drawFlames (net-correct)
  - `src/main/java/jagex2/client/Client.java:11122,11129,11134,11150 (updateFlames), 11251,11274 (drawFlames); alloc 5000-5001` ↔ `pkg/jagex2/client/client.go:2832,2842,2847,2862 (UpdateFlames), 1710,1738 (DrawFlames); decl 586-587; alloc 3409-3410`
- **client-13-03** titleFlamesMerge kept as old Go name Mix without // Java: cite
  - `src/main/java/jagex2/client/Client.java:11290-11294 (titleFlamesMerge, obf client.a(IZII)I; arg1 is a dead boolean)` ↔ `pkg/jagex2/client/client.go:6912-6915 (Mix); call sites 1671,1675,1681,1685`
- **client-shell-03** Stale GameShell.java line-number cites in gameshell.go comments
  - `Actual rev-254 lines @2e62978: pollKey 486-493; drawProgress 566-597; focusGained 495-502 / focusLost 504-509 (refresh() call at 498, not 517=windowClosing); keyPressed 379-438 / keyReleased 440-480; mousePressed 288-322 / mouseReleased 324-337 (nextMouseClickTime at 295); mouseDragged 357-366 / mouseMoved 368-377; mouseEntered 342-346 / mouseExited 348-355; run() var1 init at 144, fps at 210-212` ↔ `pkg/jagex2/client/gameshell.go:41 (PollKey "459-466"), :52-54 ("529-560"), :117 ("444-456"), :127 (refresh "517"), :143 ("342-396"), :164 ("263-300"), :182 (nextMouseClickTime "281" tagged @2e62978), :217 ("381-407"), :237 ("376-390"), :258/:305 ("338-439"/"338-397"/"399-439"), :514 ("136"), :575 ("186-188")`
- **client-misc-01** InputTracking variable name-swap vs Java old/out (functionally correct)
  - `src/main/java/jagex2/client/InputTracking.java:13,16 (old/out fields); activate @33-39; flush @48-56 returns out; stop @58-66 returns old; ensureCapacity @68-75` ↔ `pkg/jagex2/client/inputtracking/inputtracking.go:11-13 (OutBuffer/OldBuffer); SetEnabled @35-42; Flush @58-67 returns OldBuffer; Stop @69-78 returns OutBuffer; EnsureCapacity @83-89`
- **client-misc-02** viewbox.go header comment + NewViewBox signature describe 245.2 ViewBox, not the 254 reference
  - `src/main/java/jagex2/client/ViewBox.java:9 (extends java.awt.Frame), :14 (ViewBox(boolean arg0, int arg1, GameShell arg2, int arg3)), :16 (title "RS2 user client - release #"); GameShell.java:110 (new ViewBox(false, canvasHeight, this, canvasWidth))` ↔ `pkg/jagex2/client/viewbox.go:5-16 (header refs 245.2 JFrame @176a85f), :79 ("Java signature: ViewBox(int screenHeight, GameShell shell, int screenWidth)"), :80 (NewViewBox(arg0 int, arg2 *GameShell, arg3 int))`
- **pix3d-A-01** lowMem/lowDetail boolean field names swapped to LowDetail/Jagged (and other field renames)
  - `Pix3D.java:10 (lowMem ib.B), :13 (lowDetail ib.E); use-sites :129,:145,:212,:839; set-sites Client.java:1358/1366, ObjType.java:490/561` ↔ `pkg/jagex2/graphics/pix3d/pix3d.go:14 (LowDetail),:15 (Jagged); use-sites :135,:171,:244,:862; set-sites client.go:1649/8485, objtype.go:418/499`
- **world-B-01** Slightly imprecise // Java: line cite on the equal-distance euclidean tiebreak in DrawTile
  - `src/main/java/jagex2/dash3d/World.java:1468-1485 @2e62978 (sort loop opens at 1468; euclidean tiebreak block at 1477-1485)` ↔ `pkg/jagex2/dash3d/world/world.go:1577 (comment) covering code at 1576-1587`
- **world-C-01** Stale // Java: cite in PointInsideTriangle (insideTriangle)
  - `src/main/java/jagex2/dash3d/World.java:1794 (else if (arg1 > arg2 && arg1 > arg3 && arg1 > arg4) return false;)` ↔ `pkg/jagex2/dash3d/world/world.go:1948 (comment: // Java: World.java:1889 — all-three-greater early reject.)`
- **model-A-03** NewModel1 has nil guards + 'Error model' print not present in rev-254 Java Model(int)
  - `Model.java:371-513 @2e62978 (Model(int): `Metadata var3 = meta[arg0];` at 373 with NO nil check and NO 'Error model' print; `git grep "Error model" 2e62978 -- Model.java` returns empty)` ↔ `pkg/jagex2/dash3d/model/model.go:347-355 (if Metadata == nil { return &m }; if var3 == nil { fmt.Printf("Error model:%d not found!\n", arg1); return &m }); WS1-MODEL-LOADER-DESIGN.md:393-395`
- **model-A-04** Static field name drift mouseX/mouseY -> MouseX/MouseZ
  - `Model.java:220 (static int mouseX, fb.Db) and Model.java:223 (static int mouseY, fb.Eb) @2e62978` ↔ `pkg/jagex2/dash3d/model/model.go:43-44 (MouseX int; MouseZ int); used model.go:1556, 1635`
- **model-B-01** NewModel4 omits loaded++ (dead-write counter inconsistency)
  - `src/main/java/jagex2/dash3d/Model.java:729 (loaded++ in Model(boolean,boolean,boolean,Model)); field decl Model.java:13 public static int loaded` ↔ `pkg/jagex2/dash3d/model/model.go:739-740 (NewModel4 starts with `var m Model`, no Loaded++); package var Loaded model.go:49; only increment site model.go:345 (NewModel1)`
- **model-B-02** NewModel5 omits loaded++ (dead-write counter inconsistency)
  - `src/main/java/jagex2/dash3d/Model.java:783 (loaded++ in Model(boolean,Model,boolean)); field decl Model.java:13` ↔ `pkg/jagex2/dash3d/model/model.go:795-796 (NewModel5 starts with `var m Model`, no Loaded++)`
- **io-ondemand-01** Systematically stale // Java: line cites and obfuscated method descriptors in ondemand.go
  - `2e62978:src/main/java/jagex2/io/OnDemand.java (actual rev-254 lines/descriptors): getFileCount vb.a(IB)I @205; getAnimCount vb.a(Z)I @210; getMapFile vb.a(IIII)I @215; hasMapLocFile vb.b(IB)Z @240; getModelFlags vb.a(BI)I @250; shouldPrefetchMidi vb.b(IZ)Z @255; request vb.a(II)V @265-286; cycle vb.c()Lnb; @297-334; prefetchPriority vb.a(IIBI)V @337-350; prefetchMaps vb.a(IZ)V @229-237; validate vb.a([BIZI)Z @671-685; unpack @135-197; remaining @289-294` ↔ `pkg/jagex2/io/ondemand/ondemand.go:194,298,304,311,328,339,346,352,361,381,409,423,466,490,496,512`
- **io-bzip2-01** Read: stale/incorrect // Java: cite (BZip2.read deob/BZip2.java)
  - `src/main/java/jagex2/io/BZip2.java:11-31 (public decompress)` ↔ `pkg/jagex2/io/bzip2/bzip2.go:16`
- **io-bzip2-03** finish/decompress: k0 zero-extended (0..255) vs Java sign-extended byte (-128..127)
  - `src/main/java/jagex2/io/BZip2.java:85,92,142,146,150,556` ↔ `pkg/jagex2/io/bzip2/bzip2.go:92,99,149,153,157,561`
- **io-net-01** Write() doc comment misdescribes Java param roles (and a non-existent boolean)
  - `src/main/java/jagex2/io/ClientStream.java:101-120 (write(byte[] arg0,int arg1,int arg3): loop `var6 < arg3`, index `arg0[var6+arg1]`); call sites Client.java:2403,2441,3032 (all pass offset arg1=0)` ↔ `pkg/jagex2/io/clientstream/clientstream.go:316-351 (Write doc + body: `for var7 := range arg1`, index `arg0[var7+arg3]`); call sites client.go:7281,7340,8424 (all pass offset arg3=0)`
- **io-net-02** Incorrect 'per-byte traces' comment; close() 'Error closing stream' println dropped
  - `src/main/java/jagex2/io/ClientStream.java: System.out lines are ONLY 67 ("Error closing stream" in close()) and 170-176 (debug()); no per-byte traces and no 'InputStream CLOSE' marker anywhere in the file` ↔ `pkg/jagex2/io/clientstream/clientstream.go:118-123 (comment) and Close() at clientstream.go:127-142 (no println on conn.Close() error)`
- **config-A-01** Stale // Java: cites and wrong method name (getIcon) in ObjType GetSprite / CheckWearModel / CheckHeadModel
  - `ObjType.java @2e62978: getSprite @442-569 (NOT getIcon @474-622); checkWearModel @572-595; checkHeadModel @638-656` ↔ `pkg/jagex2/config/objtype/objtype.go:357 (header: "Java: ObjType.getIcon(...) (ObjType.java:474-622)"); inline cites in GetSprite body (e.g. :393 "lines 507-508 (01f16088)", :452 ":567-582", :470 ":583-591", :481 ":593-601", :491 ":602-604"); :509 "ObjType.java:625-650"; :582 "ObjType.java:697-717"`
- **config-A-02** Wrong field names in NpcType.Decode opcode 90/91/92 discard comment
  - `NpcType.java @2e62978:199-204 — opcode 90 -> this.field998 = g2(); opcode 91 -> field999 = g2(); opcode 92 -> field1000 = g2(); resizex/y/z (resizeh/resizev) are opcodes 97/98 (NpcType.java:209-212)` ↔ `pkg/jagex2/config/npctype/npctype.go:175-177 (comment "opcodes 90/91/92 write resizex/y/z") guarding case 90,91,92 -> arg1.G2() at :178-179; correct field names appear at struct comment :40-45`
- **config-B-01** IfType.SwapObj stale Java cite (244/245.2-vintage line numbers)
  - `src/main/java/jagex2/config/IfType.java:428-435 @2e62978 (swapObj decl @428)` ↔ `pkg/jagex2/config/iftype/iftype.go:104 (comment) / 105-108 (SwapObj)`
- **config-B-02** IfType.LoadModel stale cite (245.2 hash+lines) and stale method-name comment
  - `src/main/java/jagex2/config/IfType.java:466-494 @2e62978 (loadModel; ObjType.getInvModel call @485)` ↔ `pkg/jagex2/config/iftype/iftype.go:382 (header comment), 401 (inline comment)`
- **config-B-03** IfType.CacheModel and GetImage stale Java cites (245.2-vintage line numbers)
  - `src/main/java/jagex2/config/IfType.java:496-502 (cacheModel) and 504-519 (getImage; try/catch 509-515) @2e62978` ↔ `pkg/jagex2/config/iftype/iftype.go:412 (CacheModel comment), 426-429 (GetImage comment)`
- **config-C-02** SeqType opcode 6/7 field names swapped vs Java replaceheldleft/replaceheldright
  - `src/main/java/jagex2/config/SeqType.java:42,45,146,148; consumer src/main/java/jagex2/dash3d/ClientPlayer.java:251-258` ↔ `pkg/jagex2/config/seqtype/seqtype.go:24-25,60-61,145-147; consumer pkg/jagex2/dash3d/entity/playerentity/clientplayer.go:257-272`
- **dash3d-collision-02** Stale // Java: cite on TestWDecor (line number + commit hash both stale)
  - `src/main/java/jagex2/dash3d/CollisionMap.java:493 (testWDecor decl start @2e62978; line 491 is inside testWall)` ↔ `pkg/jagex2/dash3d/collisionmap.go:447-449 (comment: "Java: testWDecor (CollisionMap.java:491 @176a85f). 245.2 drops the 244 deob's dead boolean param + NPE guard...")`
- **dash3d-entity-A-01** Java field spotanimHeight (z.Z) renamed to SpotanimOffset with no // Java: mapping comment
  - `src/main/java/jagex2/dash3d/ClientEntity.java:169-170 (public int spotanimHeight; @ObfuscatedName z.Z); used ClientPlayer.java:179 var5.translate(0,0,-super.spotanimHeight)` ↔ `pkg/jagex2/dash3d/entity/clententity.go:63 (SpotanimOffset int — no // Java: comment); used pkg/jagex2/dash3d/entity/playerentity/clientplayer.go:176 var4.Translate(-e.SpotanimOffset,0,0)`
- **dash3d-typ-01** Decor doc comment cites wrong Java field names (angle1/angle2 vs wshape/angle)
  - `src/main/java/jagex2/dash3d/Decor.java:17-21 (i.d wshape, i.e angle)` ↔ `pkg/jagex2/dash3d/typ/decor.go:5-7 (header comment)`
- **dash3d-typ-02** Ground bug-preservation comment misquotes the original always-false condition
  - `src/main/java/jagex2/dash3d/Ground.java:274 (if (arg16 > arg16) var48 = arg16;)` ↔ `pkg/jagex2/dash3d/typ/ground.go:229-234 (comment + `if arg3 > arg3`)`
- **datastruct-01** Stale "// Java:" cite and wrong method name on ToAsterisks
  - `src/main/java/jagex2/datastruct/JString.java:106-112 (method `censor`, decl @106, body @107-111)` ↔ `pkg/jagex2/datastruct/jstring/jstring.go:135-139 (doc comment) -> ToAsterisks @140-146`
- **gfx-2d-01** Pix8.RGBAdjust compensated arg reorder (verification note, correct)
  - `src/main/java/jagex2/graphics/Pix8.java:134-159 (rgbAdjust); caller Client.java:1781 rgbAdjust(var50+var52, var49+var52, var51+var52)` ↔ `pkg/jagex2/graphics/pix8/pix8.go:140-166 (RGBAdjust); caller pkg/jagex2/client/client.go:6489 RGBAdjust(randomR+random, randomG+random, randomB+random)`
- **gfx-2d-02** Stale rev-vintage comment on Pix8.Plot
  - `src/main/java/jagex2/graphics/Pix8.java:202 (plot signature, 9 params, no dummy)` ↔ `pkg/jagex2/graphics/pix8/pix8.go:209-211 (comment) and 212-263 (Plot)`
- **sound-01** Stale diff-base commit cites in wave.go (@176a85f, not target @2e62978)
  - `Wave.java @2e62978: delay decl @13; unpack static @33-47; generate static @49-57` ↔ `pkg/jagex2/sound/wave/wave.go:10 (Delay), :28 (Unpack), :48 (Generate)`
- **sound-02** Misleading parameter name `delta` in Envelope.Evaluate (it is the sample count, not a delta)
  - `Envelope.java @2e62978:65-80 genNext(int arg0); arg0 used only @72 `threshold = (int)((double)shapeDelta[position] / 65536.0D * (double) arg0)`` ↔ `pkg/jagex2/sound/envelope/envelope.go:49 `func (e *Envelope) Evaluate(delta int) int`, used @56`
- **signlink-01** Stale // Java: line/vintage cites across signlink.go
  - `sign/signlink.java:101-170 (run), 217-229 (opensocket), 231-243 (openurl), 292-307 (reporterror) @2e62978` ↔ `pkg/sign/signlink/signlink.go:143 (Run cites 107-178), :271-273 (CacheLoad cites 249-254 + Thread.sleep(1L)), :297 (CacheSave cites 258-277), :321 (OpenSocket cites 279-291), :345 (OpenURL cites 293-305), :468/469/475 (ReportErrorFunc @176a85f), :82-92 (MidiVol/WaveVol @176a85f)`
- **signlink-03** FindCacheDir returns dir without Java's trailing slash
  - `sign/signlink.java:189 (return var3 + var1 + "/") @2e62978` ↔ `pkg/sign/signlink/storage_disk.go:100 (return path.Join(var3, var1, "/"))`

## Intentional (11) — documented deviations/seams, no action

- **client-01-03** saveWave/replayWave always return true under the wave-audio seam; Java retry/defer branch unreachable
  - `src/main/java/jagex2/client/Client.java:1450-1458 @2e62978 (saveWave returns signlink.wavesave(arg0,arg1); replayWave returns signlink.wavereplay() — both may return false; consumed at Client.java:3366/3373)` ↔ `pkg/jagex2/client/client.go:5883-5894 (SaveWave/ReplayWave unconditionally return true) and consuming loop client.go:8147-8176 (var4 retry flag)`
- **client-02-01** draw() omits drawCycle++ (write-only diagnostic counter, dropped lag() family)
  - `Client.java:1887 (drawCycle++ in draw()); field decl @838; sole read @4496 System.out.println("draw-cycle:"+drawCycle) inside lag()/::lag dump` ↔ `client.go:2488-2499 (Draw); rationale at client.go:2373-2380`
- **client-05-04** lag() diagnostic method not ported (documented host-shell seam)
  - `Client.java:4489-4503 lag() @2e62978 (stdout dump of flame/Od/loop/draw cycles, ptype, psize; stream.debug(); super.debug=true). Invoked via ::lag at handleInputKey 4372-4374.` ↔ `pkg/jagex2/client/client.go:2373-2380 (documented omission); no Go Lag func exists`
- **client-shell-02** GameShell debug timing-telemetry block (do-while !debug) not ported
  - `GameShell.java:30 (debug field), 214-224 (do{...}while(!debug) fallthrough + System.out.println ntime/otim/opos/fps/ratio/count/del/intex); driver Client.java:4502 (super.debug=true in lag()), Client.java:4372-4373 (::lag command) @2e62978` ↔ `pkg/jagex2/client/gameshell.go:521-582 (RunShell flat for-loop, no debug branch); pkg/jagex2/client/client.go:2373-2380 (::lag comment)`
- **pix3d-A-02** initWH parameter order swapped vs Java; all 3 callers compensate
  - `Pix3D.java:105-113 (initWH(arg0,arg2)); callers Client.java:1844/1846/1848 initWH(479,96)/(190,261)/(512,334)` ↔ `pix3d.go:106-113 (InitWH(arg0,arg1)); callers client.go:6561/6563/6565 InitWH(96,479)/(261,190)/(334,512)`
- **pix3d-A-03** clearTexels/initPool rewritten as a buffer-recycling pool (documented wasm-alloc deviation)
  - `Pix3D.java:115-121 (clearTexels), :123-137 (initPool); call path Client.java:3131 clearTexels then :3206 initPool(20), boot :1790` ↔ `pix3d.go:120-131 (ClearTexels), :133-159 (InitPool); call path client.go:9476 ClearTexels then :9556 InitPool(20), boot :6499; doc commit 2871c9a, docs/superpowers/plans/2026-05-25-wasm-alloc-quick-wins.md Task 2`
- **model-A-01** Five static fields absent from Go (empty, tmpVertexX/Y/Z, tmpFaceAlpha)
  - `Model.java:16,19,22,25,28 @2e62978 (static Model empty = new Model(); static int[] tmpVertexX/Y/Z = new int[2000]; static int[] tmpFaceAlpha = new int[2000])` ↔ `pkg/jagex2/dash3d/model/model.go:16-53 (package var block has no empty / tmpVertexX/Y/Z / tmpFaceAlpha); replacements at model.go:160 (seqAlphaBuf), 863-931 (growInts/ResetFromModel6), config/npctype/npctype.go:204-209 (reusable Model param)`
- **io-bzip2-04** decompress: es/N RUNA-RUNB accumulator 32-bit wrap not reproduced (Go 64-bit)
  - `src/main/java/jagex2/io/BZip2.java:396-406,434-438` ↔ `pkg/jagex2/io/bzip2/bzip2.go:404-415,442-446`
- **io-net-03** ClientStream.debug() not ported (consistent: sole caller lag()/::lag also unported)
  - `src/main/java/jagex2/io/ClientStream.java:168-179 (debug() prints dummy/tcycl/tnum/writer/ioerror/available); sole caller Client.java:4499-4500 inside lag() (Client.java:4488-4503), reached only via the ::lag staff command Client.java:4372-4374 (staffmodlevel==2)` ↔ `pkg/jagex2/io/clientstream/clientstream.go (no Debug method); pkg/jagex2/client/client.go (no lag() port — no 'flame-cycle'/'loop-cycle'/'============' strings, no '::lag' handler, no Stream.Debug() call)`
- **config-C-01** UnkType not ported (documented deob stub)
  - `src/main/java/jagex2/config/UnkType.java:1-43; src/main/java/jagex2/client/Client.java:2014 (UnkType.list = null)` ↔ `(no Go file); skip documented at pkg/jagex2/client/client.go:7885-7890 and LOGIC-DELTA-SCOPE-254.md:238`
- **datastruct-02** LruCache `search` field + dead double-pop branch not ported (deob artifact)
  - `src/main/java/jagex2/datastruct/LruCache.java:15 (field `search`), :47-62 (put), specifically :52 `if (var5 == this.search)` and the inner :53-55 second pop` ↔ `pkg/jagex2/datastruct/lrucache.go:12-21 (struct, no Search field), :57-71 (Put, no double-pop)`

## Reverse coverage

All 176 Go files in the repo enumerated and classified; **0 suspicious / unexplained**. reverse-3 swept the
whole repo as a reconciliation net (its counts overlap reverse-1/2 at the boundaries by design), and also
cross-checked the 122 files not referenced by any forward unit.

| Unit | Scope | port | seam | test | tooling | suspicious |
|---|---|---|---|---|---|---|
| reverse-1 | pkg/jagex2/client/**, pkg/sign/** | 13 | 16 | 21 | 0 | 0 |
| reverse-2 | pkg/jagex2/{config,dash3d,datastruct,graphics,io,sound,wordenc}/** | 62 | 14 | 34 | 1 | 0 |
| reverse-3 | cmd/**, pkg/platform/**, web/**, root + whole-repo reconcile | 23 | 42 | 60 | 13 | 0 |

