# rev-245.2 Parity Audit — 2026-06-04

Exhaustive line-by-line, side-by-side function walk of the Go client (branch `rev-245.2`,
HEAD `b6574b6`) against the Java 245.2 reference (`Client-Java` git `176a85f`), modeled on
the rev-225 and rev-244 audits (PARITY-AUDIT-2026-05-28.md, PARITY-AUDIT-2026-06-03.md).

## Method

- 54 audit units covering **every** Java file under `src/main/java` at `176a85f`
  (large files chunked by declaration-line windows: Client.java ×13, Pix3D/World3D/Model ×3,
  World/WordFilter ×2; the rest singly or bundled; plus one dedicated exhaustive wire-opcode
  unit covering Protocol.java's 257+256-entry tables and every pIsaac/ptype/zone/menu opcode,
  non-diff-based). **674 Java methods walked** statement by statement: packet read
  widths; operator grouping; branch polarity; loop bounds/direction; argument semantics traced
  through callee bodies (deob param scramble); 32-bit wrap / sign-extension / shift-mask
  semantics; side-effect ordering. Comments were not trusted; claimed equivalences were
  re-verified against cited code.
- Every blocker/bug/latent finding was independently **adversarially verified** by a skeptic
  agent primed with the known false-positive classes (deob arg scramble, compensated pairs —
  Model ctor boolean swap / NpcType walkanim, Java implicit shift-count masking,
  restructured-but-equivalent control flow, behavior-lives-elsewhere, documented seams).
  30 routed → **29 confirmed, 1 refuted**.
- Reverse coverage: all Go/web files classified; **0 suspicious**
  (70 port / 42 seam-justified / 59 test / 2 tooling).
- 89 agents, ~6.4M tokens, single workflow run (`wf_33a2621d-be8`). Unit evidence reports in
  `audit-245/units/` (untracked scratch, like `audit-225/`).

## Verdict summary

| Final severity | Count |
|---|---|
| **Blocker** (crash / protocol or cache-format desync) | 1 |
| **Bug** (wrong behavior or rendering) | 6 |
| **Latent** (edge-value / not-yet-reachable) | 22 |
| Cosmetic (comments/naming/dead code) | 120 |
| Intentional (documented deviations/seams) | 136 |
| Methods missing in Go | 0 |
| Refuted by verification | 1 |

Fix order: blocker first (network-reachable panic), then bugs, then latent.
This document is a snapshot — the fix pass lands on top of it; live status belongs in
resume notes, never here.

---

## Blockers (1)

### client-07-01. GetHeightMapY drops the tile-bounds guard (return 0 → out-of-range)

- **Unit:** `client-07`  **Java:** `Client.java:6442-6460`  **Go:** `pkg/jagex2/client/client.go:2095-2107`

Java getHeightmapY guards `if (tileX<0||tileZ<0||tileX>103||tileZ>103) return 0;` (tileX=sceneX>>7, tileZ=sceneZ>>7) before indexing levelTileFlags[1][tileX][tileZ] and levelHeightmap[realLevel][tileX/+1][tileZ/+1]. Go GetHeightMapY(level,sceneX,sceneZ) omits that guard entirely. Arrays are LevelTileFlags 4x104x104 and LevelHeightMap 4x105x105, so an out-of-scene coord makes Go read garbage or panic with index-out-of-range where Java returns 0. ProjectFromGround2 callers clamp to 128..13056 (tiles 1..102), but the ~18 other callers (cutscene Camera/Orbit at 8553/8592/10713/10791, projectile/spotanim coords at 1302/1303/1316) are not all clamped. Real mistranslation; crash reachable only on edge/server-driven coords, hence latent.

> **Verifier (confirmed, severity blocker):** Read Java 176a85f Client.java:6441-6459 (guard present) and 8046-8059 CAM_MOVETO calling getHeightmapY(cameraZ,level,cameraX) with cameraX=cutsceneSrcLocalTileX*128+64, g1()=0..255. Read Go client.go:2095-2107 (no guard; var5=arg1>>7, var6=arg3>>7, indexes LevelTileFlags[1][var5][var6]). Array sizes confirmed at client.go:6006-6019 (TileFlags 104, HeightMap 105). G1 (packet.go:203) returns &0xFF=0..255. Cutscene callers 10703-10713/10781-10791 and ApplyCutscene 8548/8590 pass unclamped tile fields; grep shows no clamp anywhere. Projectile/spotanim callers (1297,1313) ARE guarded <104, so finder overstated those, but cutscene path is genuinely unguarded. Prior audit B08 already noted this missing guard (rated blocker). Refutation classes (arg-scramble, behavior-elsewhere, intentional) all ruled out. Crash from server input ⇒ blocker.
>
> **Refined:** Go GetHeightMapY (client.go:2095-2107) omits Java getHeightmapY's tile-bounds guard `if (tileX<0||tileZ<0||tileX>103||tileZ>103) return 0;` (Java 6445-6447). With LevelTileFlags sized 4x104x104 and LevelHeightMap 4x105x105, an out-of-scene coord that Java returns 0 for instead panics index-out-of-range in Go. Reachable from network: CAM_MOVETO/CAM_LOOKAT (client.go:10703,10781) set CutsceneSrc/DstLocalTileX/Z = G1() (0..255), then pass tile*128+64 to GetHeightMapY (10713,10791,8548,8590) with no clamp; tile>103 → tileX>103 → panic.

---

## Bugs (6)

### client-04-01. handleTabInput else-if chain ported as independent ifs; overlapping tab hit-boxes select the wrong tab

- **Unit:** `client-04`  **Java:** `Client.java:4203-4276`  **Go:** `pkg/jagex2/client/client.go:8648-8717`

Java handleTabInput is one if/else-if chain so at most one tab branch fires and the FIRST matching x/y range wins. Go ports every branch as a separate `if` with no `else`. Adjacent tab x-ranges overlap (tab0 539-573 vs tab1 569-599 share [569,573]; tab1/tab2 [597,599]; tab2/tab3 [625,627]; bottom row likewise) and y-ranges overlap (169-205 vs 168-205). A click in a seam with both tab interfaces present matches two branches: Java keeps the first (e.g. selectedTab=0), Go runs both assignments and the last wins (selectedTab=1). Consequence: clicking a tab's left edge selects the next tab, showing the wrong sidebar panel. Reachable in normal play. Fix: chain with else if / switch.

> **Verifier (confirmed, severity bug):** Read Java via git show 176a85f Client.java:4222-4276: explicit `if {...} else if {...} else if {...}` chain, so first matching range wins and only one branch runs. Read Go client.go:8644-8724: each tab is a standalone `if` with no `else`; last matching branch's SelectedTab assignment overrides earlier ones. Verified x-range overlaps from the literals themselves: tab0 ≤573 vs tab1 ≥569; tab1 ≤599 vs tab2 ≥597; tab2 ≤627 vs tab3 ≥625, etc.; y-ranges 168/169..205 overlap. No compensating clamp after the chain (only CycleLogic1 increment). None of the false-positive classes apply: no args, whole method present, no seam/shift/naming involvement. Control-flow restructure is non-equivalent because ranges overlap.
>
> **Refined:** Go HandleTabInput (client.go:8648-8717) flattens Java's single if/else-if chain (Client.java:4222-4276) into 14 independent `if`s. Adjacent tab hit-boxes overlap by ~3px in x (e.g. tab0 [539,573] vs tab1 [569,599] share 569-573) with overlapping y (168/169-205). In a seam where both tab interfaces are populated, Java keeps the first match; Go runs both assignments so the last wins, selecting the wrong tab/sidebar. Reachable in normal play. Fix: use else-if.

### client-07-02. DrawMenu uses stale rev-225 menuArea hover offsets

- **Unit:** `client-07`  **Java:** `Client.java:6712-6722`  **Go:** `pkg/jagex2/client/client.go:5414-5424`

Java 245.2 drawMenu subtracts (4,4)/(553,205)/(17,357) from mouseX/mouseY for menuArea 0/1/2 to hit-test the hovered option. Go DrawMenu subtracts (8,11)/(562,231)/(22,375) — the rev-225 (789x532) layout values, never updated to 245.2 (765x503). The Go file's own sibling code uses the correct 245.2 offsets: HandleMouseInput (client.go:8486-8497) subtracts 4/4, 553/205, 17/357; the layout comment at 6406-6407 and area hit-tests/blits at 6419/6434/4763/4764 all use 553/205 and 17/357. Consequence: when a right-click menu opens over the sidebar (menuArea 1) or chatbox (menuArea 2), the mouse-over highlight is misaligned and does not track the cursor. Reachable in normal play.

> **Verifier (confirmed, severity bug):** Read Java drawMenu (6712-6722): offsets 4/4, 553/205, 17/357; confirmed identical in sibling HandleMouseInput block (4115-4170) and in showContextMenu menu anchoring (9060-9100, e.g. menuY=mouseClickY-205 for area1, -357 for area2). Go DrawMenu (client.go:5414-5424) uses 8/11, 562/231, 22/375. Go ShowContextMenu (6811-6852) anchors area1 at MouseClickY-205 and area2 at MouseClickY-357 — matching Java/245.2 — and Go HandleMouseInput (8486-8497) subtracts 553/205, 17/357. So DrawMenu's offsets are stale and inconsistent with the Go file's own anchoring. Not a deob/seam/compensated-pair/shift-mask case; constants are plain literals computed inline. Highlight misaligned; selection unaffected. Confirmed bug, reachable.
>
> **Refined:** Go DrawMenu (client.go:5414-5424) subtracts stale rev-225 offsets (8,11)/(562,231)/(22,375) from mouseX/Y to position the hover highlight, but Java 245.2 drawMenu (Client.java:6712-6722) uses (4,4)/(553,205)/(17,357). The Go file's own ShowContextMenu anchors menus at origins 553/205 and 17/357, and HandleMouseInput hit-tests with 553/205, 17/357 — so DrawMenu is inconsistent with its own siblings. Result: right-click menu hover highlight is misaligned and doesn't track the cursor (all three areas). Reachable in normal play; clicks still select correctly.

### client-10-01. useMenuOption var5==188 omits redrawSidebar before early return

- **Unit:** `client-10`  **Java:** `Client.java:9273-9281`  **Go:** `pkg/jagex2/client/client.go:4994-5002`

var5==188 (select inventory item for 'Use'). Java sets objSelected/objSelectedSlot/objSelectedInterface/objInterface/objSelectedName, spellSelected=0, then this.redrawSidebar=true, then return. The Go block performs the same field writes and SpellSelected=0 but omits c.RedrawSidebar=true and returns. Because the return is early, the method-tail c.RedrawSidebar=true (client.go:5362) is also skipped. Consequence: when a player selects an inventory item for use, the sidebar is not flagged for redraw, so the selected-item highlight may not refresh that frame. Reachable in normal play; can be masked by other redraw triggers.

> **Verifier (confirmed, severity bug):** Read Java 176a85f Client.java: var5==188 block (9273-9281) has redrawSidebar=true before return; method tail (9742-9744) is objSelected=0; spellSelected=0; redrawSidebar=true. Go client.go:4994-5002 omits c.RedrawSidebar=true and returns early, skipping the matching tail at 5360-5362. Verified consequence: DrawSidebar (11021-11045) gates pixel re-render on RedrawSidebar (blit always runs but uses stale back buffer); the white selected-item outline at 3765 depends on ObjSelected and is only emitted via DrawInterface during that gated render. Checked false-positive classes: no compensating site (grep of RedrawSidebar shows no var5==188 path), no documented seam/deviation in registries. Real, reachable. Often masked by other RedrawSidebar triggers, so bug (mild) not blocker.
>
> **Refined:** In useMenuOption var5==188 (select inventory item for "Use"), Java (Client.java:9273-9281) sets the obj-selected fields, spellSelected=0, redrawSidebar=true, then returns. Go (client.go:4994-5002) sets the same fields and SpellSelected=0 but omits c.RedrawSidebar=true; its early return also skips the method-tail RedrawSidebar=true (client.go:5362). DrawSidebar (client.go:11028) only re-renders sidebar pixels when RedrawSidebar is set, and the white selected-item outline (client.go:3765, gated on ObjSelected) is drawn there — so the selection highlight won't refresh that frame.

### client-10-02. useMenuOption var5==930 omits unconditional redrawSidebar

- **Unit:** `client-10`  **Java:** `Client.java:9672-9693`  **Go:** `pkg/jagex2/client/client.go:5148-5168`

var5==930 (select targeted spell). Java sets spellSelected=1, activeSpellId, activeSpellFlags, objSelected=0, then UNCONDITIONALLY this.redrawSidebar=true, builds spellCaption, then (if flags==16) sets redrawSidebar/selectedTab/redrawSideicons, then returns. Go sets the same fields and the flags==16 branch, but is MISSING the unconditional c.RedrawSidebar=true after ObjSelected=0. For spells with flags!=16, the sidebar is not flagged for redraw, and the early return skips the method-tail c.RedrawSidebar=true. Consequence: selecting most targeted spells (flags!=16) may not refresh the sidebar/spell highlight that frame. Reachable in normal play.

> **Verifier (confirmed, severity bug):** Read Java 176a85f Client.java:9672-9693: `objSelected=0; this.redrawSidebar=true;` then spellCaption, then `if(activeSpellFlags==16){redrawSidebar/selectedTab/redrawSideicons}` then return. Read Go client.go:5148-5168: identical field sets but NO unconditional RedrawSidebar after ObjSelected=0; only the flags==16 branch (5163-5167) sets it; block returns at 5168. Confirmed the 930 block has `return` so it never reaches the method-tail RedrawSidebar at 5362 (the separate G21 fix). RedrawSidebar is a real field used widely, so not a rename. Java-244 (01f1608) also had the unconditional line, so this is a long-standing port omission, not a 245.2 delta and not in any intentional-deviation registry. Real one-frame stale-sidebar bug for spells with mask!=16; self-corrects on next redraw, so bug not blocker.
>
> **Refined:** UseMenuOption var5==930 (select targeted spell) omits the unconditional c.RedrawSidebar=true. Java (Client.java:9672-9693) sets spellSelected/activeSpellId/activeSpellFlags/objSelected, then UNCONDITIONALLY redrawSidebar=true, builds spellCaption, and only re-sets it inside the flags==16 branch before returning. Go (client.go:5148-5168) sets the fields and the flags==16 branch but lacks the unconditional set; the block returns, so the method-tail RedrawSidebar at 5362 is never reached. Targeted spells with mask!=16 don't flag a sidebar redraw that frame.

### client-11-01. drawInterface item-drag autoscroll targets the child inventory (var14) instead of the parent container (arg3)

- **Unit:** `client-11`  **Java:** `Client.java:10001-10021`  **Go:** `pkg/jagex2/client/client.go:3786-3807`

Java scrolls the PARENT container (drawInterface param arg0) when an item being dragged reaches the clip edge: arg0.scrollPosition/-=var27, arg0.scroll, arg0.height. The Go parent parameter is arg3, but Go uses the type-2 child var14 for ScrollPosition/Scroll/Height. Client-TS @bd29ce0 (Client.ts:9484-9508) confirms the parent (com.scrollPosition) is correct; the Go comment even says 'parent scrollable' while the code references the child. For a type-2 inventory var14.Scroll is typically 0 and var14.Height is the row count, so the parent never scrolls and the lower-edge guard compares against a negative bound. Consequence: drag-to-edge autoscroll is broken in scrollable inventories (bank, shop, trade). Fix: use arg3 in both autoscroll blocks.

> **Verifier (confirmed, severity bug):** Java drawInterface(Component arg0,...) (Client.java:9933) makes arg0 the parent; autoscroll blocks (read via git show, ~10001-10021) use arg0.scrollPosition/scroll/height. Go signature (client.go:3702) maps Java arg0→Go arg3 (callee body uses arg3.Type/ChildID/ChildX — confirmed by reading, not name-pairing). var14 is the iterated CHILD (client.go:3719). Lines 3786-3807 reference var14, not arg3 — wrong target. TS @bd29ce0 (Client.ts:9392 drawInterface(com,...); 9498-9508) confirms com (parent) is correct; child is separate (line 9430 passes child.scrollPosition). PARITY-AUDIT G24 added this block intending the parent. Reachable in normal play (scrollable inventories); no crash → bug, not blocker. No documented-seam/intentional exemption.
>
> **Refined:** In DrawInterface's type-2 inventory item-drag branch, the two drag-to-edge autoscroll blocks (client.go:3786-3807) operate on the child component var14 (.ScrollPosition/.Scroll/.Height) instead of the parent container arg3. Java (Client.java) and TS (com.scrollPosition) both scroll the PARENT (arg0/com). The Go comment even says "parent scrollable." Result: dragging an item to the top/bottom clip edge of a scrollable inventory (bank/shop/trade) does not auto-scroll the parent as intended.

### pix3d-3-03. var15 interpolation step computed without explicit int32 truncation

- **Unit:** `pix3d-3`  **Java:** `Pix3D.java:2053 (var17 = (arg8-arg7)*divTable[var16] >> 6)`  **Go:** `pkg/jagex2/graphics/pix3d/pix3d.go:2059`

Java var17=(arg8-arg7)*divTable[var16]>>6 is a 32-bit multiply that wraps at 2^32. Go uses 64-bit int: var15=((arg8-arg7)*DivTable[var16])>>6 with no int(int32(...)) guard. For real raster inputs (arg7/arg8 are texture-u positions var62>>8/var59>>8; divTable[var16] <= 32768) the product stays far below 2^31 so Go and Java agree. Flagged because unguarded multiply/accumulate is the dominant latent class from prior audits and would diverge only at out-of-range edge inputs unreachable in normal play. Same observation applies to arg7<<=0x9 (go:2055/2064) and var15<<=0xC (go:2054), both bounded.

> **Verifier (confirmed, severity bug):** Read Java (git show 176a85f:.../Pix3D.java:2028-2160): `int var17`, `int[] divTable`, so var17=(arg8-arg7)*divTable[var16]>>6 is a wrapping int32 multiply. Go pix3d.go:2037-2174 uses int64. TS bd29ce0:Pix3D.ts:2157-2208,1426-1539 shows arg7/arg8=shadeA/shadeB (shade, not texture-u — finder mislabeled). MulColourLightness (model.go:1425-1441) returns up to 16-bit HSL≈65535; <<8 at raster → ~16.7M; ×divTable(≤32768) ⇒ >2^31. My numeric test (overflow product 6.55e9) showed javaStep=-31817728 vs goStep=102400000, delta=2^27 (not 2^32-multiple), flipping (arg7>>23)&0x1F (iters=1:28vs12; 5:13vs29; 95:23vs7). So pix3d-3-01's "bits 23-27 preserved" holds only when steps match; here it fails. Reachable on bright large textured polys; subtle intermittent shade banding, no crash (mask prevents negative shift). Finder under-rated as latent/unreachable.
>
> **Refined:** Pix3D.java:2054 computes the shade interpolation step `var17 = (arg8-arg7)*divTable[var16] >> 6` as a 32-bit int multiply that wraps at 2^31; Go pix3d.go:2059 uses 64-bit int with no int32 guard. arg7/arg8 are shade endpoints (TS shadeA/shadeB), reaching ~16.7M (FaceColour up to 65535, scaled <<8); times divTable (≤32768) overflows 2^31 on bright wide textured triangles. When it overflows, the step difference (≈2^27) propagates via arg7+=var15 into the masked brightness shift `(arg7>>23)&0x1F`, changing rendered texel brightness. NOT neutralized by the &0x1F mask.

---

## Latent (22)

### client-02-17. getJagFile(): 6-byte header uses single Read instead of readFully

- **Unit:** `client-02`  **Java:** `Client.java:2326-2327`  **Go:** `pkg/jagex2/client/client.go:2588-2602`

Java reads the JAG header with stream.readFully(header,0,6) (loops until 6 bytes). Go does n,err:=reader.Read(header); if n<6 { error+retry }. A single Read returning <6 on a streaming reader would force a full re-download where Java keeps reading. SAFE today because OpenURL always returns an in-memory *bytes.Reader (one Read returns all available, >=6 unless the whole body is <6 bytes), and inconsistent with Load()'s crc read which uses io2.ReadFull. Latent only: fragile to any future streaming transport; no live bug.

> **Verifier (confirmed, severity latent):** Read Go client.go:2588-2620 (single reader.Read, n<6 → loadingError + continue restarts the for data==nil loop, re-issuing OpenURL) and Java Client.java getJagFile via git show 176a85f (stream.readFully(header,0,6), which DataInputStream loops to fill 6 bytes). Confirmed OpenURL (client.go:6854-6868) does io2.ReadAll then bytes.NewReader, so a *bytes.Reader Read returns all available bytes in one call — n<6 branch unreachable for bodies ≥6 bytes. Confirmed the adjacent Load CRC path (client.go:5977) uses io2.ReadFull, proving the faithful translation exists and this site diverges. No deob/seam/shift-mask refutation applies; transport-seam reasoning is exactly why it's latent not live. Finder's "forces full re-download vs Java keeps reading" is accurate.
>
> **Refined:** getJagFile reads the 6-byte JAG header with a single reader.Read + "if n<6 → error & restart whole download" (client.go:2599-2610), whereas Java uses stream.readFully(header,0,6) (Client.java:2317-2318 @176a85f), which loops until 6 bytes. Harmless today: OpenURL returns an in-memory *bytes.Reader (client.go:6854-6868, io2.ReadAll→bytes.NewReader) so one Read yields all bytes (≥6 unless body<6). Inconsistent with sibling Load CRC read using io2.ReadFull (client.go:5977). Real mistranslation, latent: fragile to any streaming transport.

### client-08-01. LAST_LOGIN_INFO stores g4 lastAddress as positive int64, not Java signed int32

- **Unit:** `client-08`  **Java:** `Client.java:7363-7369`  **Go:** `pkg/jagex2/client/client.go:10569-10575`

Java g4() returns 32-bit signed int, so an IP whose high byte >=128 stores negative into lastAddress; Go G4() returns 64-bit int with no sign truncation, storing a positive ~2-4e9 value. All three consumers are sign-agnostic: ==0 (5633), !=0 (10574), and FormatIPv4(int32(c.LastAddress)) (10575) which re-truncates to int32 and reproduces Java's dotted-quad exactly. No reachable behavior diverges today; the stored field merely holds a non-Java numeric value. Latent — would surface only if a future site reads LastAddress as a signed quantity. Fix: c.LastAddress = int(int32(c.In.G4())).

> **Verifier (confirmed, severity latent):** Reproduced both sides. Java Packet.g4() (Packet.java:216-219) returns int (32-bit); (data&0xFF)<<24 sets bit31 → negative; lastAddress is int (Client.java:584). Go G4() (packet.go:238-241) returns 64-bit int; same arithmetic stays positive; LastAddress is int (client.go:413). Stored values differ when high octet>=128. Consumers: client.go:5633 ==0, 10574 !=0 (both sign-agnostic), 10575 FormatIPv4(int32(LastAddress)) — int32() wraps positive→same 32-bit pattern as Java; FormatIPv4 (jstring.go:80-82) (ip>>24)&0xFF matches Java formatIPv4 (JString.java:69-70). Java's only other read (Client.java:10920 ==0) maps to Go 5633. No reachable behavior diverges; real but masked mistranslation = latent.
>
> **Refined:** Java g4() returns a 32-bit signed int, so lastAddress goes negative when the IP's high octet >= 128; Go G4() returns 64-bit int, so LastAddress stores the positive (~2-4e9) equivalent. All consumers are sign-agnostic: ==0 (client.go:5633), !=0 (10574), and FormatIPv4(int32(LastAddress)) (10575) where int32() re-wraps to Java's bit pattern and &0xFF masks make the dotted-quad identical. No reachable divergence today. Latent mistranslation; fix: LastAddress = int(int32(c.In.G4())).

### client-08-02. Inventory g4 item counts stored unwrapped (UPDATE_INV_FULL / UPDATE_INV_PARTIAL)

- **Unit:** `client-08`  **Java:** `Client.java:7314,8290`  **Go:** `pkg/jagex2/client/client.go:10530,11005`

When the g1 count marker is 255, Java reads var=g4() (signed int) into invSlotObjCount; for a count with bit 31 set (>2^31-1) Java stores negative. Go reads var7=c.In.G4() (64-bit) and stores it positive without int32 wrap. Stack counts are capped well below 2^31 in normal play, so the divergence is unreachable. Latent only. Fix: int(int32(c.In.G4())) at both sites if exact parity for pathological counts is required.

> **Verifier (confirmed, severity latent):** Read Java 176a85f Client.java:7673-7688 (UPDATE_INV_FULL) and 7818-7831 (UPDATE_INV_PARTIAL): var73/var103 is int, reassigned `= this.in.g4()`, stored into `int[] invSlotObjCount` (Component.java:23). Packet.java:216-219 g4() returns int; `(data&0xFF)<<24` overflows to negative when high byte ≥0x80. Go client.go:10529-10532, 11004-11009 mirror logic; packet.go:238-241 G4() returns 64-bit int, `(int(byte)&0xFF)<<24` max 0xFF000000 stays positive, stored into []int (component.go:29). Counts flow to FormatObjCount(int) with no int32 re-wrap (client.go:3977). Divergence confirmed but item stacks cap at 2^31-1, so unreachable. Finder's Java line cites (7314/8290) are wrong (cosmetic); Go cites correct. Latent stands.
>
> **Refined:** In UPDATE_INV_FULL/UPDATE_INV_PARTIAL, when the g1 count marker is 255 the count is read via g4. Java g4() returns a 32-bit int whose `(byte&0xFF)<<24` overflows into the sign bit, storing a negative value for counts with bit 31 set; Go G4() returns a 64-bit int that stays positive, storing the unwrapped value into []int InvSlotObjCount. Real divergence, but unreachable: legitimate stack counts are capped at 2^31-1, so a g4 with bit 31 set is never sent. Fix: int(int32(c.In.G4())) at both sites for exact parity.

### client-09-01. g4()>>16 sign divergence in player spotanim height

- **Unit:** `client-09`  **Java:** `Client.java:8997-9009`  **Go:** `pkg/jagex2/client/client.go:11184-11187`

Mask 0x100 (spotanim): Java reads var26=g4() (signed 32-bit int) and sets spotanimHeight=var26>>16; an arithmetic shift sign-extends when bit 31 is set. Go reads var6=G4() (io/packet.go:238) which assembles into a non-negative 64-bit Go int (range 0..0xFFFFFFFF), so var6>>16 never sign-extends. They diverge only when var6&0x80000000 is set, i.e. spotanimHeight>=32768 — unreachable with realistic spotanim heights, hence edge-value latent. Consequence: a wrong (large positive) spotanim render height for malformed/extreme values. Fix shape: int(int32(var6))>>16.

> **Verifier (confirmed, severity latent):** Java (git show 176a85f): Client.java:8792-8795 `int var26 = arg4.g4(); arg3.spotanimHeight = var26 >> 16;`; Packet.java:216-219 g4 returns signed int; ClientEntity.java:167 `public int spotanimHeight;`. Go: client.go:11185-11187 `var6 = arg3.G4(); arg4.SpotanimOffset = var6 >> 16`; packet.go:238-240 G4 masks each byte &0xFF into Go int (always non-negative); clententity.go:59 SpotanimOffset is Go int; consumer clientnpc.go:42 Translate(-e.SpotanimOffset,0,0) uses it directly with no int32 truncation. Mechanism is the documented "Go int 64-bit vs Java arithmetic >> sign-extension" class; only reachable for height>=32768 (g4 bit31 set), so latent. SpotanimOffset/spotanimHeight naming differs but slot/shift/packet-position match — no arg-scramble, no compensating pair. Evidence file's cited line 8997-9009 is drifted (actual 8792-8795) — cosmetic.
>
> **Refined:** In getPlayerExtendedInfo mask 0x100, Java reads var26=g4() (signed 32-bit int) and sets spotanimHeight=var26>>16; arithmetic shift sign-extends when bit 31 is set, yielding a negative height (~-1..-32768). Go reads var6=G4() which masks each byte (&0xFF) and assembles into a non-negative 64-bit int (0..0xFFFFFFFF), so var6>>16 is always positive. They diverge only when the g4 high bit is set (spotanim height >=32768), giving a wrong (large positive) render offset. Unreachable in normal play. Fix: int(int32(var6))>>16.

### client-09-02. g4()>>16 sign divergence in npc spotanim height

- **Unit:** `client-09`  **Java:** `Client.java:9006-9007`  **Go:** `pkg/jagex2/client/client.go:1117-1120`

Mask 0x40 (npc spotanim): Java var15=g4() then spotanimHeight=var15>>16 (arithmetic, sign-extends for negative). Go var8=G4() yields a non-negative 64-bit int so var8>>16 does not sign-extend. Diverges only when the high half (var8&0x80000000) is set, i.e. height>=32768 — edge-value latent, same class and same fix shape (int(int32(var8))>>16) as client-09-01. Same dominant int64-vs-int32 latent class noted in prior audits.

> **Verifier (confirmed, severity latent):** Read Java Client.java @176a85f (npc mask 0x40 region): int var15 = arg1.g4(); var6.spotanimHeight = var15 >> 16. Java Packet.g4() (line 218) builds value with byte<<24 into a signed 32-bit int, so bit-31 makes it negative and >>16 is arithmetic. Read Go client.go:1117-1120: var8 = arg0.G4(); SpotanimOffset = var8 >> 16. Go G4() (packet.go:240) uses (int(byte)&0xFF)<<24 into a 64-bit int, always 0..0xFFFFFFFF; var8 declared int at line 1045 (64-bit); SpotanimOffset is int (clententity.go:59). No compensating cast; not in any intentional/deferred registry. Identical mechanism to confirmed client-09-01. Divergence only for height>=32768, not normal play -> latent.
>
> **Refined:** In GetNpcPosExtended mask 0x40 (client.go:1119-1120), Go reads spotanim height via var8 = G4(); SpotanimOffset = var8 >> 16. Go G4() (packet.go:240) returns a non-negative 64-bit int and var8 is a 64-bit int, so >>16 never sign-extends. Java (Client.java:9006-9007) uses signed 32-bit g4() (Packet.java:218); var15 >> 16 sign-extends when bit 31 is set. They diverge only when the encoded height word >= 32768, unreachable in normal play. Fix: int(int32(var8)) >> 16.

### client-11-02. handleScrollInput retains a stale `c.PacketSize += arg1` deob-padding statement (extra param) absent from 245.2 Java

- **Unit:** `client-11`  **Java:** `Client.java:10288-10330`  **Go:** `pkg/jagex2/client/client.go:6985`

Java handleScrollInput has 8 logical params and no packetSize statement (no packetSize field exists in Client.java @176a85f). Go HandleScrollInput has 9 params and runs `c.PacketSize += arg1`. Both call sites (client.go:1751, 4546) pass 0, so it is inert today. But c.PacketSize is the LIVE network packet-size field (set from SERVERPROT_SIZES, read by the read loop at client.go:9959, consumed by ReadFully/GetPlayer/GetNpcPos). This is a leftover from the rev-225 deob anti-tamper accumulator; it is a fidelity deviation and a latent protocol-desync hazard if any caller ever passes a nonzero 2nd arg. Recommend dropping the extra param + statement (as was done for DrawScene per the comment at client.go:1523).

> **Verifier (confirmed, severity latent):** Read Java Client.java:10280-10310 via git show 176a85f: 8 params, body has no packetSize statement; `grep -i packetSize` over the whole Java file returns nothing — no such field exists. Read Go client.go:6978-7009: 9 params, line 6985 `c.PacketSize += arg1`. PacketSize is the genuine network size field — decl line 337, assigned from SERVERPROT_SIZES (9959) and variable-length (9970/9982), consumed by ReadFully (9989), GetNpcPos (10093), GetPlayer (10114). Both call sites (1751, 4546) pass literal 0 for the inserted arg, so it is a no-op now. Not a deob-name-scramble (re-derived pairing; the extra param is genuinely inserted, matching the documented DrawScene/GetTopLevelCutscene padding pattern at 1523/1509). Confirmed latent: real mistranslation, inert today, protocol-desync hazard if a nonzero arg is ever passed.
>
> **Refined:** Go HandleScrollInput (client.go:6978) has 9 params vs Java's 8 (Client.java:10280): an extra int arg1 plus `c.PacketSize += arg1` (line 6985) — a leftover rev-225 deob anti-tamper accumulator. Java @176a85f has no packetSize field at all (zero grep hits). PacketSize is the live network packet-length field (decl 337; set 9959/9970/9982; consumed by ReadFully/GetNpcPos/GetPlayer). Both callers (1751, 4546) pass 0, so inert today, but any nonzero 2nd arg would desync the protocol read. Drop the param+statement per the DrawScene precedent (1523).

### gameshell-07. focusGained omits refresh()->redrawFrame=true

- **Unit:** `gameshell`  **Java:** `GameShell.java:513-521 + Client.java:2160-2162`  **Go:** `pkg/jagex2/client/gameshell.go:112-123`

Java focusGained does redrawScreen=true plus refresh(); Client.refresh() sets redrawFrame=true. Go handleFocus sets only c.Refresh (=redrawScreen), never c.RedrawFrame, and there is no Client.Refresh() method. c.Refresh is consumed only by the boot loading bar (DrawProgressGameShell), so during gameplay the focus-gained branch has no effect, whereas Java's redrawFrame at client.go:4509 forces RedrawSidebar/Chatback/SideIcons/PrivacySettings rebuild. Consequence: focus-regain during play skips that forced rebuild. Reachable (backends emit FocusChange). Impact muted by the every-frame PixMap re-upload and sub-panel dirty flags, hence latent.

> **Verifier (confirmed, severity latent):** Read Java GameShell.java:512-521 (focusGained: redrawScreen=true + refresh()), GameShell.java:571 (empty refresh base), Client.java:2159-2162 (refresh override sets redrawFrame=true), Client.java drawGame redrawFrame consumption. Read Go gameshell.go:58-123: handleFocus sets only c.Refresh; DrawProgressGameShell is its sole consumer. Grep confirms RefreshFunc (client.go:6642) sets RedrawFrame but has zero callers. client.go:4509-4519 forces sub-panel rebuild gated on c.RedrawFrame. Backends emit FocusChange (backend_glfw.go:228, backend_webgl.go:253). Discrepancy independently reproduced. Finder's "no Client.Refresh() method" is slightly wrong (RefreshFunc exists, uncalled) but the substantive claim holds. Static-chrome half is a documented immediate-mode seam (client.go:4475-4480), so only the forced rebuild is lost; muted by per-frame upload and dirty flags. Latent.
>
> **Refined:** Go handleFocus (gameshell.go:112-123) on focus-gain sets only c.Refresh (= Java redrawScreen, used solely by the boot loading bar), but never c.RedrawFrame. Java focusGained (GameShell.java:512-521) also calls refresh() → Client.refresh() (Client.java:2159-2162) sets redrawFrame=true, which in drawGame forces a full RedrawSidebar/Chatback/SideIcons/PrivacySettings rebuild (preserved in Go at client.go:4509-4519 but gated on c.RedrawFrame). RefreshFunc() exists (client.go:6642) but has zero callers. So focus-regain during play skips the forced sub-panel rebuild; reachable (both backends emit FocusChange). Impact muted by per-frame PixMap re-upload and sub-panel dirty flags.

### pix3d-2-01. textureTriangle gradient products computed in 64-bit, not Java/JS 32-bit-wrapped

- **Unit:** `pix3d-2`  **Java:** `Pix3D.java:1430-1456,1484-1486`  **Go:** `pix3d.go:1450-1458,1501-1504,1592-1594,1687-1690,1777-1780,1872-1875,1962-1965`

Java computes the nine texture-gradient cross-products as 32-bit int, e.g. `var23*arg12 - var24*arg9 << 14`, where the product-difference reaches ~1e6 and the `<<14` then exceeds 2^31, so Java wraps mod 2^32. TS reference matches (JS `<<` applies ToInt32 to operand and result; u/v/w carry no `|0` exactly because `<<` truncates). Go `int` is 64-bit and the Go expressions `(var37*arg12 - var39*arg9) << 14` (plus per-scanline `var20 += var22*var35` and `var20 += var22`) carry no `int(int32(...))`, so they are NOT truncated. These become textureRaster args 9/10/11, recovered there via `arg11>>14` and divided for perspective-correct UV (sibling unit pix3d-3). When the `<<14` term overflows int32, Java's `(W<<14 wrapped)>>14` differs from Go's full-width value, shifting the divisor and the sampled texel -> potential wrong texturing. Latent (not bug): host render smoke passed with textured terrain/models, the form survived prior int32-focused audits, and no concrete wrong-pixel repro was produced; overflow may stay benign for typical view-space coordinate magnitudes. Fix: wrap gradients and step accumulations with int(int32(...)). Coupled with pix3d-3's textureRaster recovery shifts.

> **Verifier (confirmed, severity latent):** Read Java Pix3D.java@176a85f:1421-1495 — gradients var26..var34 and accumulators var48..var50 are 32-bit `int` with `<<14`/`<<8`/`<<5` and `+= var28*var47`. Read Go pix3d.go:1441-1610 — identical structure, native 64-bit `int`, zero int32() wraps; name-map (Java var20-25→Go var36-41, var26-34→Go var20-28) verified correct, not an arg-scramble. Numeric test: cross-product diff=512000 → Go `>>14`=512000 vs Java32 `>>14`=-12288 (overflow at diff>131072, reachable for terrain deltas~128 × view-coords~4000). Not refuted by any FP class: no wrap lives elsewhere (PARITY-AUDIT L37 grep -c int32(=0 in pix3d.go). This is a precise extension of verifier-CONFIRMED L37 (3D-pipeline 64-bit convention, also latent). Per the "overflow plausibly exceeds 32 bits → real" rule, confirmed latent.
>
> **Refined:** textureTriangle's nine texture-gradient cross-products (Java Pix3D.java:1430-1438 `var23*arg12-var24*arg9 << 14` etc., all 32-bit int) and the per-scanline accumulators (`var48 = var26 + var28*var47`, `var48 += var28`, lines 1484-1495) wrap mod 2^32 in Java. The Go port (pix3d.go:1450-1458 gradients, 1501-1504/1523-1525/1534-1536 accumulators) computes these in native 64-bit int with no int(int32(...)) wrap. When the cross-product diff exceeds ~131072 (>>14 reaches 2^31), Java's wrapped `(W<<14)>>14` differs from Go's full-width recovery in textureRaster, shifting the divisor and sampled texel. Reachable for terrain/model view-space magnitudes; no visible-wrong-pixel repro produced.

### model-3-03. Lighting/colour multiplies are 32-bit-wrap-sensitive (shared renderer class)

- **Unit:** `model-3`  **Java:** `Model.java:1416,1462`  **Go:** `pkg/jagex2/dash3d/model/model.go:1389,1401,1434`

applyLighting computes arg2*n.x+arg3*n.y+arg4*n.z and mulColourLightness computes arg1*(arg0&0x7F) in Java 32-bit int; Go computes the same in 64-bit int. For in-range data (normals ~256*W, light <=768, lightness clamped) products stay well under 2^31 and Java/Go agree. A pathological input (huge accumulated normal W or un-clamped lightness near a near-zero denominator) could overflow Java int and wrap negative while Go would not, diverging the resulting face colour. This is the inherent renderer int32-truncation class flagged by both prior audits, not introduced by this port; no int(int32()) applied here, consistent with the rest of the rasterizer. Only reachable with out-of-range model/lighting data; not seen in normal play.

> **Verifier (confirmed, severity latent):** Read Java via git show 176a85f:src/main/java/jagex2/dash3d/Model.java (applyLighting + mulColourLightness) and Go model.go:1389/1401/1434 — arithmetic is byte-for-byte identical, differing only in operand width (VertexNormal x/y/z/w are int in both; Java int=32-bit, Go int=64-bit). grep confirms 0 int32( in model.go. Checked false-positive classes: pairing derived from callee body (not deob names); no behavior-elsewhere; >>7 is a constant in-range shift (no mask issue). This is the documented int32-truncation class (PARITY-AUDIT-2026-06-03 L37: model.go/pix3d use zero int32 wraps by deliberate convention); the over-refute guard warns against killing this class which produced real findings (L12,L34,L36-38). Finder correctly bounds reachability and severity. Latent is accurate.
>
> **Refined:** Confirmed real int32-vs-int64 width divergence. Java applyLighting (Model.java, dash3d pkg) computes arg2*n.x+arg3*n.y+arg4*n.z over 32-bit int and mulColourLightness computes arg1*(arg0&0x7F)>>7 in 32-bit; Go (model.go:1389,1401,1434) computes the identical expressions in 64-bit int with no int32() wrap (VertexNormal fields are int=64-bit in Go vs int=32-bit Java). For in-range model/lighting data products stay under 2^31 and results match; only pathological out-of-range cache data wraps Java negative while Go does not. Edge-only, not reachable in normal play.

### world-1-01. build() drops Java's if(fullbright) random-offset zeroing branch

- **Unit:** `world-1`  **Java:** `World.java:589-607`  **Go:** `pkg/jagex2/dash3d/world/world.go:566-571`

Java build() sets randomHueOffset=0 and randomLightnessOffset=0 when fullbright is true, else increments+clamps them. Go's Build() unconditionally executes only the else arm (increment + clamp via max/min) with no FullBright guard. Those offsets feed the per-tile underlay HSL at world.go:689-690 ((var38+RandomHueOffset)&0xFF and var40+=RandomLightnessOffset), which runs regardless of FullBright, so under FullBright Go would apply drifting hue/lightness variation where Java applies zero -> wrong terrain tint. CURRENTLY UNREACHABLE: fullbright/FullBright is never assigned true in either codebase (only the default-false declaration), so the branch is dead; hence latent, not bug. The other two FullBright uses (if !FullBright BuildModels; if FullBright return) are ported correctly at world.go:749 and 759.

> **Verifier (confirmed, severity latent):** Read Java git show 176a85f World.java:589-607: explicit `if(fullbright){randomHueOffset=0; randomLightnessOffset=0;}else{...increment+clamp...}`. Go world.go:566-571 has no FullBright guard, only the else-arm via max/min — confirmed missing branch myself. Offsets consumed unconditionally at world.go:689-690 inside `if var26>0` block, no FullBright gate. Grep shows FullBright only declared (line 21), reset false (49), read at 749/759 — never set true; Java fullbright (line 20) likewise never assigned true (only reads 589/765/775). So divergence is real but dead. No false-positive class fits: not a seam, pairing/constants (-8/8,-16/16) match, missing statement exists nowhere else. Latent confirmed.
>
> **Refined:** Go World.Build() (world.go:566-571) unconditionally runs Java's else arm (increment + clamp RandomHueOffset/RandomLightnessOffset), dropping Java World.java:589-591's `if(fullbright){randomHueOffset=0; randomLightnessOffset=0;}` branch. Those offsets feed per-tile underlay HSL (world.go:689-690), executed regardless of FullBright. Under fullbright Go applies drifting tint where Java applies zero. But FullBright is never assigned true in either codebase, so the branch is dead -> latent.

### ondemand-01. Cycle() panics on a zip entry shorter than 2 bytes (Java/TS degrade gracefully)

- **Unit:** `ondemand`  **Java:** `OnDemand.java:323-369`  **Go:** `pkg/jagex2/io/ondemand/ondemand.go:427-456`

Cycle() does gz.NewReader(bytes.NewReader(r.Data[:len(r.Data)-2])) with no length guard. r.Data is set in read() from a /ondemand.zip entry (od.zip[key]); a present entry of length 0 or 1 is a non-nil slice, so the `if r.Data==nil` guard misses it and r.Data[:len-2] uses a negative high bound -> runtime panic 'slice bounds out of range'. Java gunzips the full buffer and surfaces a catchable IOException->RuntimeException; TS slice(0,length-2) clamps without throwing. Consequence: a malformed/empty bundle entry crashes the game-loop goroutine where Java/TS continue. Latent (requires a malformed bundle entry); fix with a `len(r.Data) >= 2` guard.

> **Verifier (confirmed, severity latent):** Read Go ondemand.go:427-456 (Cycle), 686-726 (read), 365-369 (Validate has the matching len<2 guard Cycle lacks). Empirically reproduced: an empty zip entry yields non-nil []byte{}, and [:len-2] panics. Traced caller chain: UpdateOnDemand (client.go:9146-9155) → Update (9205, no recover) → RunShell loop (gameshell.go:549, no recover). Java OnDemand.java:323-369 gunzips the whole buffer (no [:len-2]), throwing RuntimeException; GameShell.java:151-230 run() does NOT catch update() — so Java also crashes, just via a different mechanism. Finding's 'Java degrades gracefully' is imprecise, but the core claim (missing length guard → unguarded negative-index panic that == nil misses, defeating the stated graceful-drop intent) is correct and confirmed. TS (OnDemand.ts:245) slice(0,-2) clamps, no index error.
>
> **Refined:** Cycle() (ondemand.go:440) does r.Data[:len(r.Data)-2] with no length guard. read() (line 700) sets Data directly from od.zip[key]; a present /ondemand.zip entry of length 0 or 1 is a non-nil slice (io.ReadAll returns []byte{}), so the r.Data==nil guard at 436 misses it and [:len-2] panics 'slice bounds out of range [:-2]' (reproduced). The panic is unrecovered through UpdateOnDemand→Update→RunShell loop, crashing the loop goroutine. The code's own 442-443 "drops corrupt entry" recovery never runs (panic precedes gzip.NewReader). Fix: add len(r.Data) >= 2 guard. Reachable only via a malformed bundle entry — latent.

### ondemand-07. Validate: expectedCrc from G4 not narrowed to int32, so CRC compare fails when bit 31 is set

- **Unit:** `ondemand`  **Java:** `OnDemand.java:775-789; Packet.java:216-219`  **Go:** `pkg/jagex2/io/ondemand/ondemand.go:365-373,471,569; packet.go:238-241`

Java validate() compares two signed 32-bit ints: computed crc=(int)CRC32.getValue() and expectedCrc, the latter from g4() which also wraps negative when byte0>=0x80. They match for high-bit CRCs. Go narrows only the computed side: crc=int(int32(crc32.ChecksumIEEE(...))) (e.g. -559038737) but expectedCrc is od.crcs[..] from Packet.G4() which on Go's 64-bit int stays unsigned-positive (e.g. 3735928559). So for any CRC with bit 31 set the two never compare equal (verified), where Java matches — Validate falsely reports invalid for ~50% of files, forcing needless re-downloads. Latent only because the Cache seam is wired nil (ondemand.New(...,nil)) so Validate currently only runs with data==nil (returns false on len<2 first); becomes a live bug once a storage Cache is wired. Fix: narrow both sides (int(int32(expectedCrc))==crc) or compare as uint32.

> **Verifier (confirmed, severity latent):** Read Java OnDemand.validate (176a85f:OnDemand.java ~775-789): crc=(int)crc32.getValue(); returns crc==expectedCrc — both signed int32. Java g4 (Packet.java:216-219) builds an int with <<24, overflowing negative when top bit set. Go G4 (packet.go:238-241) uses 64-bit int → positive. Go Validate (ondemand.go:371) narrows only computed side. crcs populated from buf.G4() (ondemand.go:224, field []int line 98). Numerically reproduced: Go computed -559038737 vs expected 3735928559 → false; Java both -559038737 → true. Reachability: New(...,nil) at client.go:6203; handleQueue (line 565-569) passes data=nil when cache nil → Validate returns at len<2 guard (line 366). No false-positive class fits: genuine 64-bit-vs-32-bit overflow divergence on a value that does exceed 32 bits.
>
> **Refined:** Go OnDemand.Validate (ondemand.go:371-372) narrows only the computed CRC via int(int32(...)) (signed, e.g. -559038737), but expectedCrc comes from Packet.G4 (packet.go:238-241) which on Go's 64-bit int stays unsigned-positive (e.g. 3735928559). Java's g4 returns a signed int32 (top-bit overflow), and validate compares two signed ints, so they match. Go never matches for any CRC with bit 31 set (~50% of files), falsely reporting invalid. Latent: cache is wired nil (client.go:6203), so Validate currently only sees data==nil and returns at the len<2 guard before comparing.

### objtype-02. ToCertificate desc uses Go UTF-8 bytes vs Java platform-charset getBytes

- **Unit:** `objtype`  **Java:** `ObjType.java:398`  **Go:** `pkg/jagex2/config/objtype/objtype.go:307`

Java builds the note description and calls String.getBytes() (platform default charset; client names decode as Latin-1). Go uses []byte(string) which is UTF-8. For a name with a non-ASCII char (e.g. pound sign), Java emits one Latin-1 byte 0xA3 while Go emits UTF-8 0xC2 0xA3, so desc byte length/content diverges. RuneScape item names are ASCII so this is not reachable in normal play; same edge class as the already-documented charAt(0)/byte-index seam at objtype.go:299-302. Recorded as latent edge value.

> **Verifier (confirmed, severity latent):** Read Java git show 176a85f ObjType.java:389-398: desc = ("Swap this note..." + link.name + ".").getBytes() — no charset arg, default (Latin-1) charset → one byte per char. Go objtype.go:307: t.Desc = []byte(...) — UTF-8. Verified Name origin: Java gjstr (Packet.java:233) new String(default charset); Go GJStr (packet.go:256-262) transcodes Latin-1→UTF-8, so Go Name holds multibyte UTF-8 for £. Verified consumer asymmetry: Java Client.java:9174 new String(desc); Go client.go:4944 io.Latin1ToUTF8(Desc). Normal disk path (objtype.go:180 GStrByte) stores raw bytes verbatim and matches; only the synthesized cert desc diverges. Trigger needs a non-ASCII item name, which RS 225 lacks. Genuine mistranslation, edge/unreachable → latent.
>
> **Refined:** ObjType.ToCertificate (objtype.go:307) builds the note desc via []byte(string), emitting UTF-8. Java (ObjType.java:398) calls String.getBytes() with the platform/Latin-1 charset. For a linked name with a non-ASCII char (e.g. £), Java stores one byte 0xA3 while Go stores UTF-8 0xC2 0xA3, diverging Desc length/content. It compounds: the consumer (client.go:4944) runs Latin1ToUTF8 on the UTF-8 bytes, yielding garbled "Â£" vs Java's "£". RS 225-era item names are pure ASCII, so unreachable in normal play. Same edge class as the documented charAt(0) seam at objtype.go:299-302.

### bzip2-01. finish: total_out_lo32 32-bit carry detection broken by 64-bit Go int

- **Unit:** `bzip2`  **Java:** `BZip2.java:160-163`  **Go:** `pkg/jagex2/io/bzip2/bzip2.go:162-166`

Java total_out_lo32 is a 32-bit int; `if (total_out_lo32 < var23) total_out_hi32++` detects wraparound past 2^31 to carry into the high word. Go declares TotalOutLo32 as int (64-bit), so the addition never wraps at 2^31 and the carry branch never fires. This is the dominant 32-bit-wrap mistranslation class. Consequence: NONE reachable — TotalOutLo32/Hi32 are write-only (never read by output logic; Read returns length-AvailOut), and for cache blocks (<2^31 bytes) lo32 never wraps anyway. Faithful fix would type these fields int32 in bzip2state.

> **Verifier (confirmed, severity latent):** Read Java BZip2State.java:60,63 — `public int total_out_lo32/hi32` (32-bit). Read BZip2.java:155-163 — `int var23 = s.total_out_lo32; s.total_out_lo32 += avail_out_INIT - cs_avail_out; if (s.total_out_lo32 < var23) s.total_out_hi32++`. Read Go bzip2state.go:28-29 — `TotalOutLo32/Hi32 int` (64-bit on amd64), and bzip2.go:162-166 mirroring the idiom. Tree-wide grep for TotalOutLo32/Hi32 shows only: zero-init (34-35), the carry idiom (162-165) — no consumer. Read returns `length - State.AvailOut` (line 38), independent of TotalOut. So the divergence is genuine (carry semantics differ) but has no reachable effect. Latent, not bug. Fix is int32 typing.
>
> **Refined:** In BZip2.Finish, Java's `total_out_lo32` is a 32-bit `int` whose carry idiom `if (total_out_lo32 < var23) total_out_hi32++` detects 2^31 signed wraparound. Go declares TotalOutLo32/Hi32 as `int` (64-bit), so the addition never wraps at 2^31 and the carry branch fires only on true 64-bit overflow. Real 32-bit-wrap mistranslation, but unreachable: the fields are write-only (Read returns length-AvailOut; grep shows no other readers) and cache blocks never approach 2^31 bytes. Faithful fix: type both fields int32.

### bzip2-02. getBits: total_in_lo32 32-bit carry detection broken by 64-bit Go int

- **Unit:** `bzip2`  **Java:** `BZip2.java:432-435`  **Go:** `pkg/jagex2/io/bzip2/bzip2.go:596-599`

Java `total_in_lo32++; if (total_in_lo32 == 0) total_in_hi32++` detects 32-bit unsigned wrap (2^32-1 -> 0). Go increments TotalInLo32 (64-bit int) which never reaches 0 by wrap, so TotalInHi32 is never bumped. Same 32-bit-wrap class as bzip2-01. Consequence: NONE reachable — both Total* fields are write-only and lo32 cannot reach 2^32 for real inputs. Faithful fix is int32 typing.

> **Verifier (confirmed, severity latent):** Read Java getBits (BZip2.java:582-599 via git show 176a85f): `s.total_in_lo32++; if (s.total_in_lo32 == 0) s.total_in_hi32++;` with fields declared `public int` (BZip2State.java:45,48) — 32-bit, so wrap-to-zero fires the carry. Go bzip2.go:596-599 is a literal port but bzip2state.go:23-24 declares both as Go `int` (64-bit amd64), so the carry is dead. Repo-wide grep shows TotalInLo32/Hi32 only written (init+increment), never read — write-only. This exactly matches the already-confirmed PARITY-AUDIT-2026-06-03 L02 (lines 1302-1320, severity latent). Not a seam/intentional deviation; the Go-int-64-bit class the rules warn against over-refuting. Java line cite 432-435 in the finding is drifted (actual 590-592) — cosmetic only.
>
> **Refined:** getBits' 32-bit carry idiom is mistranslated. Java total_in_lo32 is a 32-bit int whose wrap-to-zero at 2^32 triggers total_in_hi32++ (BZip2State.java:45; BZip2.java:590-592). Go's TotalInLo32 is 64-bit int (bzip2state.go:23), so the increment never returns to 0 by wrap and TotalInHi32 (bzip2.go:596-599) is never bumped — dead carry. Both fields are write-only telemetry with zero readers repo-wide, and total_in cannot reach 4 GiB for cache data, so unobservable. Faithful fix is int32 typing.

### pix32-02. TransPlot uses Go logical >> where Java uses arithmetic >> on a sign-bit-set blend result

- **Unit:** `pix32`  **Java:** `Pix32.java:387`  **Go:** `pkg/jagex2/graphics/pix32/pix32.go:410`

Java groups the blend as (M_rb + M_g) >> 8 where M_rb = (...) & 0xFF00FF00. When the blended red byte >=0x80, M_rb has bit 31 set so the int is negative and Java's arithmetic >> sign-extends, giving top byte 0xFF (0xFFRRGGBB). Go computes the same bit pattern in 64-bit then shifts logically, giving top byte 0x00 (0x00RRGGBB). The multiply/add do not 32-bit-wrap (sum <= 0xFFFFFF00) so only bits[31:24] of the stored pixel differ. No observable consequence: pixmap.writePixmapPixels discards the top byte via uint32(argb)<<8|0xFF (pixmap.go:102), and any re-blend masks only 0xFF00FF/0xFF00. Real arithmetic-vs-logical-shift mistranslation, edge-value, unreachable as a visible bug.

> **Verifier (confirmed, severity latent):** Read Java Pix32.java:373-389 (git show 176a85f) and Go pix32.go:400-417. Reproduced exhaustively (/tmp test, alpha 0..256): low-24 RGB always matches; full sum max 0x7FFF7F00 fits 31 bits so no carry corrupts RGB; only the top byte diverges (Java sign-extends to 0xFF when red>=0x80, Go gives 0x00). Verified no reachable consequence: writePixmapPixels (pixmap.go:102) does uint32(argb)<<8|0xFF, discarding the top byte at output; grep found no >>24 or &0xFF000000 anywhere in graphics/; RGBAdjust (pix32.go:133-147) masks &0xFF; hashPixels includes the top byte but only for Go-only change-detection and cannot cause false-negative uploads since RGB changes still flip the hash. Genuine mistranslation, edge-value, latent.
>
> **Refined:** Pix32.transPlot (Java Pix32.java:387) blends as ((M_rb)&0xFF00FF00 + (M_g)&0xFF0000) >> 8 in 32-bit signed int. When the blended red byte >=0x80, M_rb has bit31 set so the int is negative and Java's arithmetic >> sign-extends to top byte 0xFF (0xFFRRGGBB). Go pix32.go:410 computes the same bits in 64-bit int, so the logical >>8 leaves 0x00 (0x00RRGGBB). Only result bits[31:24] differ; RGB identical. Real arithmetic-vs-logical shift mistranslation, edge-value, no visible consequence.

### loctype-02. op[] array size 5 vs codes 30-38 — faithful Java overflow reproduced

- **Unit:** `loctype`  **Java:** `LocType.java:288-294`  **Go:** `pkg/jagex2/config/loctype/loctype.go:209-220`

Java allocates op = new String[5] but the branch handles codes 30-38, indexing op[code-30] = 0..8. Codes 35-38 give indices 5-8 beyond the size-5 array, throwing ArrayIndexOutOfBounds in Java. Go reproduces this exactly: make([]string, 5) indexed Op[code-30], so codes 35-38 would panic identically. This is a pre-existing Java defect ported bug-for-bug. Unreachable in normal play (the cache never emits codes 35-38). Recorded for completeness; both implementations behave identically on malformed config.

> **Verifier (confirmed, severity latent):** Read Java LocType.java (git show 176a85f) lines 278-285: `else if (code >= 30 && code < 39)` allocates `this.op = new String[5]` and writes `this.op[code - 30]`; code 39 handled separately as contrast above. Read Go loctype.go:209-220: `case 30,31,...,38`, `loc.Op = make([]string, 5)`, `loc.Op[code-30]`; case 39 = Contrast at line 207-208. For codes 35-38, index = 5-8 overflows the length-5 container on both sides — identical fault (Java throws, Go panics). No false-positive class applies: single-position write (no arg scramble), case-list equals the Java range (no control-flow divergence), it's a slice-index overflow not arithmetic (Go 64-bit int irrelevant). Already recorded latent in PARITY-AUDIT-2026-06-03.md:1404-1416.
>
> **Refined:** Faithful bug-for-bug port of an upstream out-of-bounds bound. Java LocType.decode (`code >= 30 && code < 39`) and Go LocType.Decode (`case 30..38`) both handle config codes 30-38 (code 39 = contrast handled separately), allocate a length-5 container (`new String[5]` / `make([]string, 5)`), and index it with `code-30`. Codes 35-38 yield indices 5-8, overflowing the length-5 container → ArrayIndexOutOfBoundsException (Java) / slice index panic (Go). Unreachable with normal cache data; no port-side divergence.

### pixfont-01. plotLetterTransInner: missing int32 clamp on alpha-blend result (no observable effect)

- **Unit:** `pixfont`  **Java:** `PixFont.java:424-425,437`  **Go:** `pkg/jagex2/graphics/pixfont/pixfont.go:488,496`

Java int arithmetic is 32-bit: (colour&0xFF00FF)*alpha can reach 0xFEFF010D whose bit 31 is set, so after & 0xFF00FF00 the term is a negative int32 and (A+B)>>8 sign-extends, leaving 0xFF in bits 24-31 of the stored pixel. Go int is 64-bit so the same product stays positive (no real 2^32 wrap: max < 2^32), >>8 does not sign-extend, and bits 24-31 are 0x00. Operator grouping is otherwise correct ((A+B)>>8). Consequence: NONE observable. Verified by exhaustive emulation that the low 24 RGB bits are identical for all inputs; the high byte is masked off when re-read as dstRgb for overlapping glyphs (&0xFF00FF/&0xFF00) and discarded at texture upload (writePixmapPixels: uint32(argb)<<8|0xFF shifts bits 24-31 out). Latent because the int32-clamp gap would surface only if the pixel buffer ever consumed the high byte as alpha.

> **Verifier (confirmed, severity latent):** Read Java via git show 176a85f PixFont.java:423-438 and Go pixfont.go:487-504. Operator grouping matches: both compute (TERM1+TERM2)>>8 (Java + binds tighter than >>; Go explicitly parenthesized). Emulated Java int32 vs Go int64 over all 0x1000000 colours x 257 alphas: diffLow24=0, only high byte differs (e.g. colour=ff0000 alpha=129 java=ff800000 go=00800000). Per-pixel path emulated over 4.3B (alpha,dst) pairs with Java dst carrying worst-case 0xFF high byte: low24Diff=0. Confirmed consumers discard high byte: writePixmapPixels (pixmap.go:102) does uint32(argb)<<8|0xFF, shifting bits 24-31 out; grep found no >>24 alpha read in graphics/. Real int32-clamp mistranslation, no observable effect — latent.
>
> **Refined:** In PixFont.plotLetterTransInner the Go port (pixfont.go:488,496) computes the alpha-blend in 64-bit int, so the masked product stays positive and >>8 yields 0x00 in pixel bits 24-31. Java (PixFont.java:424,437) uses 32-bit int: when the masked product's bit 31 is set, the arithmetic >>8 sign-extends, leaving 0xFF there. Exhaustively verified the low 24 RGB bits are identical for all inputs; only the high byte diverges, and that byte is masked off on glyph re-read (&0xFF00FF/&0xFF00) and shifted out at upload (uint32(argb)<<8). No observable effect.

### packet-01. Alloc returns nil for type not in {0,1,2}; Java defaults unknown types to a 30000-byte buffer

- **Unit:** `packet`  **Java:** `Packet.java:49-80`  **Go:** `pkg/jagex2/io/packet.go:66-87`

Java alloc(int type) maps type 0->byte[100], 1->byte[5000], and ELSE (catch-all, any type>=2)->byte[30000]. Go's packetPool handles exactly {0,1,2} and returns nil for anything else, so Alloc(typ>=3) returns nil and the first wire op nil-derefs. Type 2 itself is handled identically (30000) in both. Only the Java catch-all for type 3+ is lost. All current call sites (Client out/in/login, InputTracking oldBuffer) pass type 1, so this is unreachable in normal play. Latent: a real mistranslation on an edge value not currently exercised.

> **Verifier (confirmed, severity latent):** Read Java 176a85f Packet.java:49-79: cache pop for 0/1/2, then `if type==0→byte[100] else if type==1→byte[5000] else→byte[30000]`. The else is a catch-all for type 2 and 3+. Read Go packet.go:66-87: packetPool switch returns &CacheMax for case 2, nil for default; Alloc returns nil when pool==nil. Thus type 2 matches (both 30000), only type≥3 diverges (Java 30000 vs Go nil→nil-deref). Grepped all Alloc call sites: client.go:557/565/573 and inputtracking.go:37/85 all pass type 1; Java equivalents (Client.java:173/245/485, InputTracking.java:35/74) also all type 1. No site uses 2 or 3+. Confirmed real but unreachable; finder's type-2 caveat is correct. Not in any documented deviation.
>
> **Refined:** Java Packet.alloc(int type) uses a catch-all else: type 0→100, type 1→5000, else→30000 (so any type≥2, including 3+, gets a 30000-byte buffer). Go Alloc/packetPool returns a pool only for {0,1,2} and nil for default (type≥3); Alloc then returns nil, so a type≥3 alloc nil-derefs on first use. Type 2 is identical (30000) in both. All call sites pass type 1, so unreachable in normal play.

### signlink-02. GetUID uint32+1 widening diverges from Java int32 wrap at high half

- **Unit:** `signlink`  **Java:** `sign/signlink.java:209-211`  **Go:** `pkg/sign/signlink/storage_disk.go:128-129`

Java getuid does int var4=readInt(); return var4+1, wrapping at 32 bits (signed). Go does var6:=binary.BigEndian.Uint32(...); return int(var6+1), widening uint32 to 64-bit int. Identical for all realistic values (uid.dat written from (int)(Math.random()*9.9999999E7) in [0,~1e8) where +1 cannot overflow; 0xFFFFFFFF+1 wraps to 0 in both). Diverges only for an externally-supplied var6>=0x80000000 not ending 0xFFFFFFFF: e.g. 0x7FFFFFFF gives Java -2147483648 vs Go +2147483648. Unreachable in normal play (file always self-generated, small positive; uid is opaque). Latent. Fix if strict shared-cache byte parity needed: return int(int32(var6+1)).

> **Verifier (confirmed, severity latent):** Read Java via git show 176a85f:src/main/java/sign/signlink.java:208-211 — int var4=var3.readInt(); return var4+1 (signed int wrap). Read Go storage_disk.go:128-129 — var6:=binary.BigEndian.Uint32(var5); return int(var6+1). var6+1 is uint32 arithmetic (wraps mod 2^32 unsigned), then int(...) widens zero-extended to 64 bits. Confirmed divergence at the signed-high half (0x7FFFFFFF case). No false-positive class applies: single local (no arg scramble), sole implementation (not behavior-elsewhere), arithmetic genuinely differs (not restructured-equivalent). The storage seam is documented but doesn't excuse the int32/uint32 mismatch. Reachability: uid.dat is always self-generated small-positive [0,~1e8), high bit never set, so divergence needs a hand-crafted file; uid is opaque to the server. Real mistranslation, edge-only — latent, not blocker (not a cache key).
>
> **Refined:** GetUID computes int(var6+1) where var6 is uint32: the +1 wraps mod 2^32 unsigned, then int(...) zero-extends to 64-bit Go int. Java getuid does signed int var4+1, wrapping mod 2^32 signed. They agree for all small positive uids (the only values uid.dat ever holds: (int)(Math.random()*9.9999999E7)∈[0,~1e8)) and for 0xFFFFFFFF (both 0). They diverge only for an externally-supplied var6>=0x80000000 not all-ones, e.g. 0x7FFFFFFF→Java -2147483648 vs Go +2147483648. Unreachable in normal play; uid is opaque. Latent. Fix: return int(int32(var6+1)).

### tone-envelope-01. int (64-bit) vs Java int (32-bit) wrap at synth multiply/accumulate sites

- **Unit:** `tone-envelope`  **Java:** `Tone.java:163-200 (generate harmonic/mod/phase loops)`  **Go:** `pkg/jagex2/sound/tone/tone.go:128-144 (and phase accumulators 129/136)`

Java synthesis uses 32-bit int[] buffers and int locals that wrap at 2^31; Go uses 64-bit int throughout (Buffer/Noise/Sin + all locals), so it does not wrap. The intermediate product amplitude*fAmp[j] (envelope amp ~ShapePeak<<15, fAmp up to (255<<14)/100~=41779) and the phase accumulators fPos/frequencyPhase/amplitudePhase can exceed int32 for large-but-legal peak / many-sample inputs, where Java wraps and Go does not. Phase consumers mask &0x7FFF so divergence only shows when a value crosses 2^31; for standard RS2 sound-cache data (small peaks, samples<=220500) products stay within int32 and outputs match. Documented decision in tone.go:11-19 (Theme C invariant, document-don't-retype). Consequence: no audible divergence in normal play; edge-value only.

> **Verifier (confirmed, severity latent):** Independently read Java (git show 176a85f Tone.java/Envelope.java) and Go (tone.go, envelope.go, packet.go). Confirmed Java declares buf/noise/sine and fPos/fAmp/etc as int[] and all generate/waveFunc locals as int (32-bit); Go uses []int/int (64-bit amd64) with no int32() wraps (lines 128-144,205-209). Traced inputs: harmonicVolume via GSmartS up to 32767→fAmp~5.4M; envelope amplitude is shapePeak(G2,max65535)<<15~2.1e9; intermediate amplitude*fAmp before >>15 vastly exceeds 2^31 for large values; Envelope Start/End from G4 full 32-bit. No false-positive mechanism applies: type widths genuinely differ, no compensating wrap. Matches prior confirmation PARITY-AUDIT-2026-06-03.md:1477-1479 and the documented Theme C decision tone.go:11-19. Latent: not reachable with realistic cache data.
>
> **Refined:** Java Tone.generate/waveFunc (Tone.java:55-76,98-200,226) use 32-bit int[] buffers (buf/noise/sine/fPos/fAmp/fMulti) and 32-bit int locals; the products feeding each shift (amplitude*fAmp>>15, frequency*fMulti>>16, rate*start>>16, sine/noise/(phase&0x7FFF)*amplitude) wrap mod 2^32. Go tone.go (lines 121-213) computes all in 64-bit int with no int32() wrap (verified 0 wraps). Real mistranslation, but with normal RS2 cache data (small peaks, samples<=220500) products stay within int32 so outputs match; only large/adversarial cache values cross 2^31 and diverge. Edge-only, no audible difference in normal play.

### datastruct-07. LruCache Java HashTable->Go map: Put does not guard duplicate keys

- **Unit:** `datastruct`  **Java:** `LruCache.java:40-60`  **Go:** `pkg/jagex2/datastruct/lrucache.go:47-61`

Java put delegates to HashTable.put which node.unlink()s the node from any prior bucket then re-links, so a re-put of an existing key is idempotent (one node). Go Put always allocates a fresh DoublyLinkable and overwrites HashTable[key], orphaning the prior node in History and double-decrementing Available. All 8 call sites follow Get-then-Put-only-on-miss (component.CacheModel Clears first), so no live duplicate-key Put exists. Documented in code. Latent mistranslation guarded by caller discipline; if a future caller Puts a hot key without a preceding miss, History leaks and Available undercounts, prematurely evicting.

> **Verifier (confirmed, severity latent):** Read Java LruCache.java put (`table.put(node,key); history.push(node)`) and HashTable.java:34-43 (`if(node.prev!=null) node.unlink()` then re-link) via git show 176a85f — confirms node-level bucket idempotency. Read Go lrucache.go:47-61: unconditional NewDoublyLinkable + HashTable[key]=node + History.Push + Available--, no existing-key check — divergence reproduced. Verified all flagged sites guard: component.go:382/402 (Get/return then Put), :409 Clear-before-Put; objtype.go:342; loctype.go:291-292/352 and :304-305/313 (Put inside `if Get==nil`); clientplayer.go:235/236/305 (Put-key var2 == Get-key var2, inside `if var15==nil`). No live duplicate-key Put. Finder's idempotency reasoning and latent classification are accurate; no false-positive class applies.
>
> **Refined:** Go LruCache.Put (lrucache.go:47-61) always allocates a fresh DoublyLinkable, overwrites HashTable[key], pushes to History, and decrements Available, with no duplicate-key guard. Java's put delegates to HashTable.put (HashTable.java:34-43) which `if(node.prev!=null) node.unlink()`s the prior bucket entry before re-linking, making a re-put of the same node structurally idempotent. A duplicate-key Go Put would orphan the previous node in History and double-decrement Available, causing premature eviction. All 8 call sites follow Get-then-Put-on-miss (component.CacheModel Clears first), so no live trigger exists. Documented in-code. Real mistranslation; latent.

---

## Refuted findings (1)

### world-2-04. static AddLoc typecode stored as 64-bit int (int32-wrap edge for locId>=65536)

- **Unit:** `world-2`  **Java:** `World.java:1044-1047`  **Go:** `pkg/jagex2/dash3d/world/world.go:1033-1036`

Java computes typecode var18 = x + (z<<7) + (locId<<14) + 1073741824, plus Integer.MIN_VALUE if inactive, as a 32-bit int that wraps. Go computes the same as 64-bit int (BitSet fields are Go int). For locId>=65536, locId<<14 reaches bit 30 and Java wraps into a different 32-bit pattern than Go's clean 64-bit value, potentially altering (BitSet>>29)&0x3 type field and (BitSet>>14)&0x7FFF locId decode. For realistic locId (<2^16, the loc.dat count this revision) value stays <2^31, the +MinInt32 inactive flag only flips bit 31, and both decode masks discard the diverging high bits; GetInfo equality key round-trips since stored and queried values both come from Go. Same pattern in instance AddLoc (world.go:295-298) and client.go:9528 decode. Net-correct in-range; flagged latent for the out-of-range locId edge only.

> **Verifier (REFUTED):** Read Java (git show 176a85f:.../dash3d/World.java:1037-1056): `int var18 = arg5 + (arg4<<7) + (arg6<<14) + 1073741824`, field types int (Sprite/Wall/Decor/GroundDecor.java typecode = int). Read Go world.go:1023-1036 (static) and 288-298 (instance): identical formula, BitSet=int (64-bit). Arg pairing verified via callee body (heightMap[level][x][z] = arg0[arg7][arg5][arg4]) — not a deob scramble. False-positive class: "Go int 64-bit overflow only real if values exceed 32 bits at that site." They cannot: Count=G2() (packet.go:221, max 65535), Offsets sized to Count (loctype.go:83), Get indexes Offsets[locId] so locId<Count<=65535. My calc (/tmp/claude-1000/calc.py) shows at locId=65535,x=z=127 active value = 2147483647 = INT_MAX, no wrap; decode identical Java/Go at all reachable locId. Out-of-range edge would panic in Get before encoding.
>
> **Refined:** Refuted. Java World.java:1044 computes typecode as 32-bit int (var18 = z + (x<<7) + (locId<<14) + 2^30, +Integer.MIN_VALUE if inactive); Go world.go:1033/295 computes it as 64-bit int. The width difference is real but the claimed wrap edge (locId>=65536) is structurally unreachable: loc Count is read via G2() (max 65535, loctype.go:82), so locId<=65534. At the absolute ceiling (locId=65535) the active typecode equals exactly Integer.MAX_VALUE — no int32 overflow. Java and Go produce bit-identical values, and (>>29)&3 type / (>>14)&0x7FFF locId decode round-trip identically, for every reachable input. No divergence, not even latent.

---

## Cosmetic (120)

| Id | Title | Java | Go |
|---|---|---|---|
| `client-01-01` | Dead duplicate field MessageIDs (only allocated, never used) | `Client.java:293` | `pkg/jagex2/client/client.go:231,593` |
| `client-01-02` | drawCycle counter and its lag() diagnostic dropped | `Client.java:971,1989,4819` | `pkg/jagex2/client/client.go:2427` |
| `client-01-03` | Field1264 named/commented after wrong Java field (behavior is field1504, correct) | `Client.java:860,8005,2939,6512` | `pkg/jagex2/client/client.go:182-185` |
| `client-01-04` | GetHost body diverges from Java standalone literal but is unused | `Client.java:1380-1389` | `pkg/jagex2/client/client.go:5394-5400` |
| `client-02-01` | load(): boot progress percentages diverge from Java | `Client.java:1500-1923` | `pkg/jagex2/client/client.go:5989-6418` |
| `client-02-02` | load(): crc-loop text/percent 'Connecting to web server'/20 -> 'Connecting to fileserver'/10 | `Client.java:1522` | `pkg/jagex2/client/client.go:5966` |
| `client-02-03` | load(): missing drawProgress('Connecting to update server', 60) | `Client.java:1578` | `pkg/jagex2/client/client.go:6196` |
| `client-02-04` | load(): media/textures unpack moved before OnDemand request loops | `Client.java:1724-1885` | `pkg/jagex2/client/client.go:6029-6199` |
| `client-02-06` | draw(): missing drawCycle++ | `Client.java:1989` | `pkg/jagex2/client/client.go:2427-2438` |
| `client-02-07` | unload(): missing onDemand.stop() + onDemand=null | `Client.java:2030-2031` | `pkg/jagex2/client/client.go:7469-7591` |
| `client-02-10` | unload(): UnkType skip comment cites stale 244 name 'class61' | `Client.java:2143` | `pkg/jagex2/client/client.go:7577-7580` |
| `client-02-12` | drawProgress(): missing lastProgressPercent/lastProgressMessage assignments | `Client.java:2179-2180` | `pkg/jagex2/client/client.go:11224-11226` |
| `client-03-01` | updateGame pIsaac comments cite stale 244 opcode numbers | `Client.java:2953,3048,3070,3122,3189,3196` | `pkg/jagex2/client/client.go:7757,7846,7864,7905,7966,7971` |
| `client-03-02` | Field/local name drift vs 245.2 Java in untouched methods | `Client.java:860,2696,2938` | `pkg/jagex2/client/client.go:185,7134,7681` |
| `client-04-02` | handleChatMouseInput unused mouseX param and dead mod write dropped; junk arg1 carried as PacketSize+=0 | `Client.java:3801-3873` | `pkg/jagex2/client/client.go:1973-2048` |
| `client-04-03` | handlePrivateChatInput junk byte param repurposed to pass mouseY explicitly | `Client.java:3743-3799` | `pkg/jagex2/client/client.go:5435-5492` |
| `client-04-04` | updateAudio inlined into UpdateGame; SaveWave arg order is a verified callee-scramble | `Client.java:3595-3633` | `pkg/jagex2/client/client.go:7699-7754` |
| `client-04-06` | handleChatModeInput renamed to HandleChatSettingsInput with junk-param PacketSize+=arg0 | `Client.java:4288-4347` | `pkg/jagex2/client/client.go:1926-1971` |
| `client-05-01` | Stale 244-era // Java: opcode/line comments throughout unit | `Client.java:4348-4360,4561-4814,4867-4890` | `client.go:905,2240,2252,2286,2322,2384,2409,9693` |
| `client-05-03` | updateOrbitCamera Java try/catch + reporterror(glfc_ex) not ported | `Client.java:4348-4445` | `client.go:6545-6607` |
| `client-05-04` | HandleInputKey ::lag local stdout dump dropped (server CLIENT_CHEAT retained) | `Client.java:4690-4716` | `client.go:2304-2324` |
| `client-06-04` | LoadTitleImages allocates FlameBuffer3 before FlameBuffer2 | `Client.java:5440-5442` | `pkg/jagex2/client/client.go:3344-3345` |
| `client-06-05` | GetTopLevelCutscene carries a dead arg0 dummy param (PacketSize += arg0) | `Client.java:6076-6081` | `pkg/jagex2/client/client.go:1507-1517` |
| `client-07-03` | Draw3DEntityElements field1504 ported under name Field1264 | `Client.java:6512-6517` | `pkg/jagex2/client/client.go:6499-6504` |
| `client-07-04` | TryMove move-opcode comments cite stale pIsaac numbers | `Client.java tryMove (window 6179-7100)` | `pkg/jagex2/client/client.go:8209,8213,8217` |
| `client-08-03` | FINISH_TRACKING outbound comment says pIsaac(217); actual opcode is 19 | `Client.java:7207` | `pkg/jagex2/client/client.go:10512` |
| `client-08-04` | Per-handler // Java: opcode N comments use 244-era opcode numbers | `Client.java:7101-8126` | `pkg/jagex2/client/client.go:9998-11014` |
| `client-10-04` | getCombatLevelTag renamed to GetCombatLevelColorTag | `Client.java:9916-9932` | `pkg/jagex2/client/client.go:5365-5390` |
| `client-10-05` | Stale // Java: pIsaac(N)/interactWithLoc(N) comments in UseMenuOption | `Client.java:9122-9746` | `pkg/jagex2/client/client.go:4787-5363` |
| `client-11-04` | drawInterface type-6 uses Go pix3d.CenterW3D/CenterH3D names for Java Pix3D.centerX/centerY | `Client.java:10141-10165` | `pkg/jagex2/client/client.go:3940-3967` |
| `client-11-05` | handleScrollInput adds early `return` after setting RedrawSidebar inside the if/else-if chain | `Client.java:10288-10330` | `pkg/jagex2/client/client.go:6990,6996` |
| `client-12-01` | Stale `// Java: pIsaac(NN)` opcode comments cite wrong (225-era) literals | `Client.java:11740,11759,12186,12209,12247,12267` | `client.go:5873,5893,7465,8362,1162,9508` |
| `client-13-02` | runFlames drops this.flameCycle++ debug counter | `Client.java:11576` | `pkg/jagex2/client/client.go:6952-6974` |
| `gameshell-02` | run-loop debug diagnostic block dropped, undocumented at site | `GameShell.java:228-240` | `pkg/jagex2/client/gameshell.go:554-558` |
| `gameshell-03` | hasFocus field not ported | `GameShell.java:56,514,525` | `pkg/jagex2/client/gameshell.go:112-123` |
| `gameshell-06` | handleMouseMove comment misstates Java arg order | `GameShell.java:390,403` | `pkg/jagex2/client/gameshell.go:199-211` |
| `viewbox-input-02` | InputTracking buffer field names inverted vs Java (OutBuffer<->OldBuffer) | `InputTracking.java:13-16,38-77` | `pkg/jagex2/client/inputtracking/inputtracking.go:12-13,34-88` |
| `viewbox-input-05` | EnsureCapacity exported but only internally used (no lock) | `InputTracking.java:79-86` | `pkg/jagex2/client/inputtracking/inputtracking.go:79-88` |
| `viewbox-input-06` | Stale 'Gio' reference in mu doc comment | `InputTracking.java:34` | `pkg/jagex2/client/inputtracking/inputtracking.go:24-31` |
| `viewbox-input-07` | Dropped mouseTracking Client.java sites lack intentional-skip marker | `Client.java:1958,2016-2018,2681` | `pkg/jagex2/client/client.go:7114-7128` |
| `pix3d-1-04` | unload duplicates divTable=null and never nulls divTable2 — faithful port | `Pix3D.java:79-94` | `pkg/jagex2/graphics/pix3d/pix3d.go:81-95` |
| `pix3d-1-05` | adopted/renamed identifiers per 245.2 naming policy | `Pix3D.java:9-342` | `pkg/jagex2/graphics/pix3d/pix3d.go:13-372` |
| `pix3d-3-04` | Local variable renumbering and LowDetail/!lowMem branch-order swap | `Pix3D.java:2028-2410` | `pkg/jagex2/graphics/pix3d/pix3d.go:2037-2435` |
| `world3d-1-01` | AddLoc2 adds an unreachable `arg8 == nil` guard not present in Java | `World3D.java:473-490` | `pkg/jagex2/dash3d/world3d/world3d.go:395-398` |
| `world3d-1-02` | RemoveLoc1 drops Java's dead `boolean var3` local | `World3D.java:543` | `pkg/jagex2/dash3d/world3d/world3d.go:474-497` |
| `world3d-2-04` | drawTile reuses loop var9 as the bridge-tile temp | `World3D.java:1253-1280` | `pkg/jagex2/dash3d/world3d/world3d.go:1273-1297` |
| `world3d-3-03` | PointInsideTriangle carries a stale historical-bug comment | `World3D.java:1788-1804` | `pkg/jagex2/dash3d/world3d/world3d.go:1924-1947` |
| `model-1-03` | Field mouseY renamed to MouseZ | `Model.java:223` | `model/model.go:44` |
| `model-1-04` | picking->Pickable rename; superclass minY flattened into struct | `Model.java:52` | `model/model.go:136,149` |
| `model-1-06` | NewModel3 doc comment cites a stale 244-era signature and line number | `Model.java:629` | `model/model.go:618-621` |
| `model-2-01` | Bounds field naming inverted vs Java (Go MaxY=Java super.minY, Go MinY=Java this.maxY) | `Model.java:943-1024` | `pkg/jagex2/dash3d/model/model.go:962-1016` |
| `model-2-02` | Dead-write deob local var10002 in createLabelReferences not ported | `Model.java:1026-1075` | `pkg/jagex2/dash3d/model/model.go:1018-1060` |
| `model-2-03` | Deob parameter-name remap in applyFrame/applyFrames vs Go ApplyTransform/ApplyTransforms | `Model.java:1095-1145` | `pkg/jagex2/dash3d/model/model.go:1082-1135` |
| `model-3-01` | MouseZ holds the mouse screen-Y coordinate (naming) | `Model.java:1582,1645` | `pkg/jagex2/dash3d/model/model.go:1552,1631` |
| `model-3-02` | Field picking renamed to Pickable | `Model.java:52,1583` | `pkg/jagex2/dash3d/model/model.go:136,1554` |
| `world-2-02` | ChangeLocAvailable param order reversed vs Java (net-correct) | `World.java:1025-1034` | `pkg/jagex2/dash3d/world/world.go:1012-1021` |
| `world-2-03` | static AddLoc Go param names mislead (arg7=base level, level=adjusted heightmap level) | `World.java:1037-1043` | `pkg/jagex2/dash3d/world/world.go:1023-1027` |
| `wordfilter-1-01` | Filter omits dead System.currentTimeMillis() timing locals | `WordFilter.java:141,162` | `pkg/jagex2/wordenc/wordfilter/wordfilter.go:169` |
| `wordfilter-1-02` | FilterDomains omits dead boolean var6 | `WordFilter.java:207` | `pkg/jagex2/wordenc/wordfilter/wordfilter.go:243` |
| `wordfilter-1-03` | FilterDomain/FilterTLD2 carry a benign extra match := false | `WordFilter.java:216,345` | `pkg/jagex2/wordenc/wordfilter/wordfilter.go:270,414` |
| `wordfilter-2-wordpack-01` | WordPack.Pack has an extra `terminate bool` parameter absent from Java 245.2 | `WordPack.java:61,88-90` | `pkg/jagex2/wordenc/wordpack/wordpack.go:64,103-105` |
| `wordfilter-2-wordpack-03` | getEmulatedSize/getEmulatedDomainCharSize use De-Morgan-inverted condition forms | `WordFilter.java:728-905,707-726` | `pkg/jagex2/wordenc/wordfilter/wordfilter.go:846-1073,821-844` |
| `wordfilter-2-wordpack-04` | filterFragments drops Java's unused local `boolean var2` | `WordFilter.java:920` | `pkg/jagex2/wordenc/wordfilter/wordfilter.go:1092-1139` |
| `ondemand-04` | Field obfuscated-name comments shifted one slot vs 245.2 Java; header says rev-244 | `OnDemand.java:18-130` | `pkg/jagex2/io/ondemand/ondemand.go:94-166` |
| `objtype-04` | IconCache capacity comment references 244/225 lineage | `ObjType.java (modelCache=50, iconCache=100)` | `pkg/jagex2/config/objtype/objtype.go:22-23` |
| `objtype-05` | Field/local naming differs from Java deob names | `ObjType.java (whole file)` | `pkg/jagex2/config/objtype/objtype.go (whole file)` |
| `bzip2-03` | finish/decompress: k0 zero-extended (0..255) vs Java sign-extended byte (-128..127) | `BZip2.java:39,85,91,92,142,146,150,556` | `pkg/jagex2/io/bzip2/bzip2.go:46,92,98,99,149,153,157,561` |
| `bzip2-04` | decompress: gPos inferred as Go byte vs Java int | `BZip2.java:362` | `pkg/jagex2/io/bzip2/bzip2.go:391` |
| `bzip2-06` | decompress: dead local nblockMAX omitted | `BZip2.java:350` | `pkg/jagex2/io/bzip2/bzip2.go:357` |
| `collisionmap-01` | Constructor assigns SizeX/SizeZ from swapped param order vs Java | `CollisionMap.java:23-27` | `pkg/jagex2/dash3d/collisionmap.go:11-21` |
| `pix32-01` | Sprite-blit param order swapped vs Java signature (uniform Go convention) | `Pix32.java:232,328,470` | `pkg/jagex2/graphics/pix32/pix32.go:247,353,496` |
| `pix32-05` | 244-era deob local names retained in untouched methods | `Pix32.java:113-147,210-230,374-392,516-562` | `pkg/jagex2/graphics/pix32/pix32.go:129,218,400,535` |
| `loctype-04` | Static ignoreCache renamed Reset in Go; never written (dead both sides) | `LocType.java:14,416,422` | `pkg/jagex2/config/loctype/loctype.go:13,288,369` |
| `loctype-07` | Model bounds field-name inversion (MinY<->maxY); ObjRaise=MaxY is correct | `LocType.java:520, Model.java:968-981` | `pkg/jagex2/config/loctype/loctype.go:350, pkg/jagex2/dash3d/model/model.go:980-989` |
| `component-01` | getModel parameter reorder vs Java (int,bool,int) | `Component.java:430-456` | `pkg/jagex2/config/component/component.go:350-375` |
| `component-02` | loadModel localPlayer injection replaces Client.localPlayer static | `Component.java:467` | `pkg/jagex2/config/component/component.go:392-394` |
| `component-06` | unpack local variable renaming/reuse and range loops | `Component.java:209-417` | `pkg/jagex2/config/component/component.go:109-347` |
| `component-07` | comment line-number drift in Go method headers | `Component.java:430-456` | `pkg/jagex2/config/component/component.go:349,377-381` |
| `clientplayer-03` | Java additive-vs-shift precedence correctly parenthesized in Go | `ClientPlayer.java:277,282` | `pkg/jagex2/dash3d/entity/playerentity/clientplayer.go:226,230` |
| `clientplayer-04` | SeqType RightHand/LeftHand name-vs-Java inversion is internally consistent | `ClientPlayer.java:274-283` | `pkg/jagex2/dash3d/entity/playerentity/clientplayer.go:215-231` |
| `clientplayer-06` | offset/Translate and resize/Scale arg permutations verified net-correct | `ClientPlayer.java:197,217,238` | `pkg/jagex2/dash3d/entity/playerentity/clientplayer.go:154,173,196,160` |
| `clientplayer-07` | applyFrames/ApplyTransforms arg scramble verified net-correct | `ClientPlayer.java:371` | `pkg/jagex2/dash3d/entity/playerentity/clientplayer.go:319` |
| `clientplayer-08` | height = model.minY mapped to Height = MaxY (Model min/max naming inversion) | `ClientPlayer.java:184` | `pkg/jagex2/dash3d/entity/playerentity/clientplayer.go:144` |
| `clientplayer-09` | else-if slot-replace collapsed to two if statements (slot 3 vs 5 mutually exclusive) | `ClientPlayer.java:294-298,322-326` | `pkg/jagex2/dash3d/entity/playerentity/clientplayer.go:244-249,271-276` |
| `clientplayer-10` | field/method renames with no behavior change | `ClientPlayer.java:111-148` | `pkg/jagex2/dash3d/entity/playerentity/clientplayer.go:63-133` |
| `npctype-02` | Go range loops vs Java C-style for in Unpack/Get/Decode | `NpcType.java:98-110 (unpack), decode loops` | `pkg/jagex2/config/npctype/npctype.go:83-90,124,161,168` |
| `packet-04` | GStrByte doc comment cites Java name 'gstrbyte'; actual method is 'gjstrraw' | `Packet.java:237-247` | `pkg/jagex2/io/packet.go:265-282` |
| `packet-05` | CRCTable is a dead field, never read in Java or Go | `Packet.java:22,303-322` | `pkg/jagex2/io/packet.go:18-23,36-50` |
| `packet-06` | CRC init stores 64-bit non-negative entries vs Java signed 32-bit; low 32 bits equal | `Packet.java:303-322` | `pkg/jagex2/io/packet.go:36-50` |
| `signlink-01` | FindCacheDir drops Java's trailing slash on returned dir | `sign/signlink.java:189-191` | `pkg/sign/signlink/storage_disk.go:100` |
| `signlink-08` | Consumer comments cite 244 deob SignLink.java filename/lines | `sign/signlink.java (245.2 has no consumer half)` | `pkg/jagex2/sound/audio/audioloop.go:9-27,62-164` |
| `tone-envelope-02` | Method/field renames vs Java deob names (policy-compliant, // Java: markers) | `Tone.java/Envelope.java (genInit, genNext, unpack, waveFunc, fPos/fDel/fAmp/fMulti/fOffset, buf/noise/sine)` | `pkg/jagex2/sound/tone/tone.go:20-29,196-216; pkg/jagex2/sound/envelope/envelope.go:23-49` |
| `ground-01` | Dead deob arrays shape0P1/P2/P3 not ported and unmarked | `Ground.java:69-75` | `pkg/jagex2/dash3d/typ/ground.go:3-11` |
| `pix2d-pix8-pixmap-01` | Pix2D draw-primitive params reordered vs Java signatures (uniform convention) | `Pix2D.java:40-290` | `pkg/jagex2/graphics/pix2d/pix2d.go:20-241` |
| `pix2d-pix8-pixmap-02` | Pix2D cls() renamed Clear() | `Pix2D.java:85-91` | `pkg/jagex2/graphics/pix2d/pix2d.go:67-72` |
| `pix2d-pix8-pixmap-04` | Pix8.Plot carries an extra dead parameter + dead guard | `Pix8.java:225-272` | `pkg/jagex2/graphics/pix8/pix8.go:210-264` |
| `pix2d-pix8-pixmap-05` | Pix8 method renames (halveSize/trim/hflip/vflip/rgbAdjust) | `Pix8.java:79-177` | `pkg/jagex2/graphics/pix8/pix8.go:67-166` |
| `pix2d-pix8-pixmap-06` | Pix8 plotSprite(x,y) ported as PlotSprite(y,x) (uniform convention) | `Pix8.java:179-223` | `pkg/jagex2/graphics/pix8/pix8.go:169-207` |
| `cliententity-01` | Field rename needsForwardDrawPadding -> SeqStretches | `ClientEntity.java:30-31` | `pkg/jagex2/dash3d/entity/clententity.go:12` |
| `cliententity-02` | Walk/seq/route anim field renames (244-era names) | `ClientEntity.java:23-275` | `pkg/jagex2/dash3d/entity/clententity.go:8-91` |
| `cliententity-07` | ObjType.getModel ported under name GetInterfaceModel | `ClientObj.java:14-18, ObjType.java:404-441` | `pkg/jagex2/dash3d/entity/clientobj.go:22-24, objtype.go:311-344` |
| `cliententity-08` | SeqType.postanim_move ported as PostanimMode | `ClientEntity.java:159-160,234-235; SeqType.java:54,141` | `pkg/jagex2/dash3d/entity/clententity.go:116,175; seqtype.go:31,154` |
| `streams-01` | Stale startThread(this, 2) in NewClientStream doc comment (Java is ,3) | `ClientStream.java:134` | `pkg/jagex2/io/clientstream/clientstream.go:93` |
| `flotype-seqtype-01` | FloType.getHsl ported as SetColour (244-era name) | `FloType.java:94-189` | `pkg/jagex2/config/flotype/flotype.go:74-152` |
| `flotype-seqtype-02` | SeqType field names use 244-era names (loops/maxloops/replaceheld*) | `SeqType.java:23-57,127-149` | `pkg/jagex2/config/seqtype/seqtype.go:15-33,131-149` |
| `flotype-seqtype-03` | SeqType.postanim_move (245.2) ported under 244-era name PostanimMode | `SeqType.java:54,141,167-172` | `pkg/jagex2/config/seqtype/seqtype.go:31,107-113,153-154` |
| `flotype-seqtype-04` | getFrameLength assignment-in-expression flattened in Go | `SeqType.java:79-92` | `pkg/jagex2/config/seqtype/seqtype.go:40-53` |
| `wave-02` | NewWave constructor seam (no Java equivalent) | `Wave.java:24,27-29` | `pkg/jagex2/sound/wave/wave.go:22-26` |
| `anim-meta-01` | AnimFrame.unload inlined at caller, no package function | `AnimFrame.java:147-150` | `pkg/jagex2/client/client.go:7589` |
| `proj-spotanim-02` | Dead local boolean var3=false not ported | `ClientProj.java:114` | `pkg/jagex2/dash3d/entity/clientproj.go:77-96` |
| `proj-spotanim-05` | ClientProj field renames vs Java | `ClientProj.java:9-86` | `pkg/jagex2/dash3d/entity/clientproj.go:11-36` |
| `isaac-jagfile-04` | Isaac.generate collapses y>>8>>2 to y>>10 | `Isaac.java:69` | `pkg/jagex2/io/isaac.go:59-60` |
| `config-small-02` | VarpType.unpack omits field1158 reset, field1159 alloc, and 'varptype load mismatch' println | `VarpType.java:62-66` | `pkg/jagex2/config/varptype/varptype.go:31-43` |
| `datastruct-02` | LinkList method renames push->AddTail, pop->RemoveHead | `LinkList.java:21-49` | `pkg/jagex2/datastruct/linklist.go:17-44` |
| `datastruct-03` | Linkable.key relocated to DoublyLinkable.Key in Go | `Linkable.java:8-24` | `pkg/jagex2/datastruct/linkable.go:3-20` |
| `datastruct-04` | DoublyLinkList sentinel named head; unlink2->Uncache | `DoublyLinkList.java:8-73` | `pkg/jagex2/datastruct/doublylinklist.go:8-66` |
| `datastruct-05` | LruCache hit/miss counters (notFound/found) dropped | `LruCache.java:8-37` | `pkg/jagex2/datastruct/lrucache.go:28-38` |
| `datastruct-09` | Standalone hashtable package is a faithful but unused port | `HashTable.java:1-51` | `pkg/jagex2/datastruct/hashtable/hashtable.go:1-104` |
| `typ-bundle-03` | QuickGround field263=true default initializer not replicated in Go struct | `QuickGround.java:31` | `pkg/jagex2/dash3d/typ/quickground.go:9-22` |
| `opcodes-03` | Stale // Java: pIsaac(N) inline comments cite rev-244 deob numbers | `Client.java (245.2 emit sites)` | `client.go:905,4833,8209-8217 and all P1Isaac sites` |

---

## Intentional deviations (136)

| Id | Title | Java | Go |
|---|---|---|---|
| `client-01-05` | Dead-write fields not ported (field1537, field1528, field1215, oplogic10) | `Client.java:467,1001,53,647` | `pkg/jagex2/client/client.go:111-540` |
| `client-01-06` | Wave audio facade routes around signlink to Go-native consumer; SaveWave arg swap | `Client.java:1453-1469` | `pkg/jagex2/client/client.go:5717-5734` |
| `client-01-07` | main() applet->CLI flag seam; portOffset dropped; dims verified | `Client.java:1270-1305` | `cmd/client/main.go:19-162` |
| `client-02-05` | load(): host allowlist + FileStream init dropped; recover() replaces reporterror | `Client.java:1485-1965` | `pkg/jagex2/client/client.go:5921-5945` |
| `client-02-08` | unload(): mouseTracking teardown not ported (deob artifact) | `Client.java:2024-2027` | `pkg/jagex2/client/client.go:7469-7591` |
| `client-02-09` | unload(): super.drawArea=null and UnkType.types=null not ported (deob) | `Client.java:2150,2143` | `pkg/jagex2/client/client.go:7572-7585` |
| `client-02-11` | unload(): System.gc() not ported | `Client.java:2168` | `pkg/jagex2/client/client.go:7591` |
| `client-02-13` | drawProgress(): title-tile upload always re-issued; FlameActive skip removed | `Client.java:2210-2225` | `pkg/jagex2/client/client.go:11247-11274` |
| `client-02-14` | drawError(): AWT Graphics -> overlay PixMap + errorfont; else-if -> sequential if | `Client.java:2230-2299` | `pkg/jagex2/client/client.go:9359-9418` |
| `client-02-15` | getJagFile(): FileStream read/write -> signlink storage seam; arg2 index dropped | `Client.java:2311-2362` | `pkg/jagex2/client/client.go:2552-2654` |
| `client-02-16` | getJagFile(): NPE/AIOOBE/generic '!reporterror -> return null' paths absent | `Client.java:2385-2402` | `pkg/jagex2/client/client.go:2566-2664` |
| `client-03-03` | updateGame inlines updateAudio and guards updateSceneState | `Client.java:2943-2945` | `pkg/jagex2/client/client.go:7689-7754` |
| `client-03-04` | login success block omits dead MouseTracking/focus fields | `Client.java:2679-2683` | `pkg/jagex2/client/client.go:7121-7122` |
| `client-03-05` | System.gc dropped; PixMap/present/cache_dat/fileStreams seams | `Client.java:2867,3316,3406,3248,3393,2907-2920` | `pkg/jagex2/client/client.go:9107,8837,8973,9099,3607-3635` |
| `client-04-05` | handleViewportOptions Examine options append examineIDSuffix (DEVELOPER_MODE extra) | `Client.java:3875-4040` | `pkg/jagex2/client/client.go:9566,9654` |
| `client-05-02` | Deob garbage int/byte params preserved as no-op PacketSize += argN | `Client.java:4396,4988` | `client.go:6573,4208` |
| `client-05-05` | lag() debug method not ported | `Client.java:4816-4831` | `client.go (absent)` |
| `client-06-01` | LoadTitle hoists area-nil-out + restarts flame goroutine on re-enter | `Client.java:5237-5281` | `pkg/jagex2/client/client.go:6897-6925` |
| `client-06-02` | LoadTitle drops super.drawArea = null | `Client.java:5242` | `pkg/jagex2/client/client.go:125-128` |
| `client-06-03` | DrawTitleScreen/DrawGame blit ImageTitle0/1 + drop redrawFrame guards (immediate-mode + flameMu seam) | `Client.java:5444-5534` | `pkg/jagex2/client/client.go:3589-3599` |
| `client-06-06` | DrawGame moves chrome/SideIcons/Privacy GPU blits out of dirty guards | `Client.java:5534-5805` | `pkg/jagex2/client/client.go:4496-4724` |
| `client-10-03` | AddNPCOptions Examine appends DEVELOPER_MODE id suffix | `Client.java:9831` | `pkg/jagex2/client/client.go:2181` |
| `client-11-06` | executeClientScript opcode-9 range-over-int mutation verified equivalent (not the range-mutation trap) | `Client.java:10460-10468` | `pkg/jagex2/client/client.go:9300-9306` |
| `client-13-01` | unloadTitle keeps title/flame fields alive instead of nilling them | `Client.java:11542-11564` | `pkg/jagex2/client/client.go:2666-2685` |
| `client-13-03` | runFlames runs on its own goroutine via direct call; timing math faithful | `Client.java:11569-11597` | `pkg/jagex2/client/client.go:6952-6974,3352` |
| `client-13-04` | FlameActive/FlameThread are unsynchronized cross-goroutine bools (faithful to Java) | `Client.java:11543-11546,11570,11597` | `pkg/jagex2/client/client.go:238,304,2669-2673,6953,6973` |
| `client-13-05` | FlameBuffer2 <-> FlameBuffer3 naming swap (net-correct pair) | `Client.java:11600-11656,11736,11759` | `pkg/jagex2/client/client.go:2768-2823,1684,1712,3344-3345` |
| `client-13-06` | drawFlames omits imageTitle0/1.draw blits (PixMap per-frame seam) | `Client.java:11748,11773` | `pkg/jagex2/client/client.go:1628-1726` |
| `client-13-07` | Blend/mix high byte sign-extension divergence, masked away by writePixmapPixels | `Client.java:11743,11766,11777` | `pkg/jagex2/client/client.go:1693,1716,6697,pkg/jagex2/graphics/pixmap/pixmap.go:103` |
| `client-13-08` | mix parameter order reordered (weight pos 0->1); all four call sites compensate | `Client.java:11775-11778` | `pkg/jagex2/client/client.go:6695-6698,1646-1660` |
| `gameshell-01` | fps field + per-frame computation dropped | `GameShell.java:38,224` | `pkg/jagex2/client/client.go:118-121, pkg/jagex2/client/gameshell.go:555-557` |
| `gameshell-04` | nextMouseClick* double-buffer mapped away — verified net-correct | `GameShell.java:209-213,316-338` | `pkg/jagex2/client/gameshell.go:160-195,548-553` |
| `gameshell-05` | applet lifecycle start()/stop()/destroy() not ported | `GameShell.java:268-291,538-540` | `pkg/jagex2/client/gameshell.go:501,563-565` |
| `viewbox-input-01` | ViewBox not literally ported; window/title owned by platform seam | `ViewBox.java:1-37` | `pkg/jagex2/client/viewbox.go:1-85` |
| `viewbox-input-03` | trackedCount dead-write field dropped | `InputTracking.java:22,85,110,140,176,209,242,262,282,302` | `pkg/jagex2/client/inputtracking/inputtracking.go:14-18` |
| `viewbox-input-04` | MouseTracking class not ported; verified still dead in 245.2 | `MouseTracking.java:1-46` | `(no Go counterpart)` |
| `pix3d-1-01` | init3D parameter order swapped, compensated at call sites | `Pix3D.java:106-113` | `pkg/jagex2/graphics/pix3d/pix3d.go:106-113` |
| `pix3d-1-02` | clearTexels/initPool rewritten to reuse texel pool (wasm-alloc deviation) | `Pix3D.java:115-138` | `pkg/jagex2/graphics/pix3d/pix3d.go:120-159` |
| `pix3d-1-03` | getTexels palette index zero-extends vs Java signed-byte sign-extension | `Pix3D.java:189-250` | `pkg/jagex2/graphics/pix3d/pix3d.go:211-281` |
| `pix3d-1-06` | Reset() test-only helper, no Java counterpart | `(none)` | `pkg/jagex2/graphics/pix3d/pix3d.go:59-79` |
| `pix3d-2-S1` | gouraudRaster/flatRaster drop dead Java parameters (callers pass 0) | `Pix3D.java:839,1365` | `pix3d.go:856,1379` |
| `pix3d-3-01` | Shift-count & 0x1F mask on arg7 >> 23 reproduces Java implicit shift masking | `Pix3D.java:2095,2150,2241,2297,2407 (var18>>23 → var92/var34)` | `pkg/jagex2/graphics/pix3d/pix3d.go:2122,2174,2296,2348,2425` |
| `pix3d-3-02` | Low-bit masks arg7 & 0x600000 / (arg7>>3)&0xC0000 unaffected by 64-bit width | `Pix3D.java:2094,2137,2240,2296,2406 (var91/var33 = arg2 + (var18&...))` | `pkg/jagex2/graphics/pix3d/pix3d.go:2110,2173,2295,2347,2424` |
| `world3d-1-03` | buildModels gates vertexNormal via `(*model.Model)` type assertion (ModelSource seam) | `World3D.java:729-749` | `pkg/jagex2/dash3d/world3d/world3d.go:679-708` |
| `world3d-1-04` | MinY seeded from a static model in AddLoc2/SetWallDecoration (rev-244 ModelSource.minY seam) | `World3D.java:473-531 (addLoc), 405-426 (addDecor)` | `pkg/jagex2/dash3d/world3d/world3d.go:421-423, 330-332` |
| `world3d-1-05` | Loc typecode `>>29 & 0x3` computed on 64-bit Go int (verified bit-equivalent) | `World3D.java:609,652,685,704 (>>29&0x3); World.java:312-314 (bitset build)` | `pkg/jagex2/dash3d/world3d/world3d.go:537,588,628; pkg/jagex2/dash3d/world/world.go:295-297` |
| `world3d-2-01` | init parameter reorder (int[] hoisted to arg0) | `World3D.java:936-998` | `pkg/jagex2/dash3d/world3d/world3d.go:941-1013` |
| `world3d-2-02` | testPoint horizontal sin/cos arg-swap, coupled with init caller | `World3D.java:1000-1013` | `pkg/jagex2/dash3d/world3d/world3d.go:1015-1030` |
| `world3d-2-03` | click parameter swap, coupled with caller | `World3D.java:1015-1022` | `pkg/jagex2/dash3d/world3d/world3d.go:1032-1038` |
| `world3d-2-05` | drawTile cull reads cached node MinY and draws via GetModel() (rev-244 seam) | `World3D.java:1328,1490,1546` | `pkg/jagex2/dash3d/world3d/world3d.go:1367,1581,1654` |
| `world3d-3-01` | drawGround/DrawTileOverlay yaw sin/cos arg scramble, compensated at call sites | `World3D.java:1717-1773` | `pkg/jagex2/dash3d/world3d/world3d.go:1855-1911` |
| `world3d-3-02` | mulLightness parameter-name swap, compensated by swapped call-site arg order | `World3D.java:1776-1786` | `pkg/jagex2/dash3d/world3d/world3d.go:1913-1922` |
| `model-1-01` | Model.empty static reuse target not ported (replaced by ResetFromModel6 pool) | `Model.java:16` | `model/model.go:876,932; config/npctype/npctype.go:230` |
| `model-1-02` | tmpVertexX/Y/Z and tmpFaceAlpha static scratch buffers not ported | `Model.java:19-28` | `model/model.go:160,862,876` |
| `model-1-05` | NewModel1 adds nil-guards and a diagnostic print absent from Java Model(int) | `Model.java:373-377` | `model/model.go:346-354` |
| `model-2-04` | set(Model,boolean): global static scratch buffers replaced by per-entity owned reuse (ResetFromModel6) | `Model.java:867-918` | `pkg/jagex2/dash3d/model/model.go:876-936` |
| `model-2-05` | NewModel6/ResetFromModel6 docstrings cite non-existent Java ctor `Model(Model,boolean)` | `Model.java:867` | `pkg/jagex2/dash3d/model/model.go:875,931` |
| `world-2-01` | Noise int32-wrap neutralized by final & MaxInt32 mask | `World.java:963-969` | `pkg/jagex2/dash3d/world/world.go:951-956` |
| `wordfilter-1-04` | FilterTLD2 omits Java dead-write var10000 block | `WordFilter.java:407-412` | `pkg/jagex2/wordenc/wordfilter/wordfilter.go:404` |
| `wordfilter-2-wordpack-02` | WordPack.Pack truncation uses rune (code-point) length where Java uses UTF-16 code-unit length | `WordPack.java:62-66,73-74` | `pkg/jagex2/wordenc/wordpack/wordpack.go:68-72,75-82` |
| `ondemand-02` | Cycle() slices the 2-byte version trailer before gunzip; Java gunzips the full buffer | `OnDemand.java:340-360` | `pkg/jagex2/io/ondemand/ondemand.go:440` |
| `ondemand-03` | Cycle() drops a corrupt entry (Data=nil) instead of throwing; no 65000-byte cap | `OnDemand.java:344-366` | `pkg/jagex2/io/ondemand/ondemand.go:441-453` |
| `ondemand-05` | GetMapFile parameter order reordered (z,x,type) vs Java (type,x,z); callers compensate | `OnDemand.java:231-245; Client.java:1649-1665,3427-3432` | `pkg/jagex2/io/ondemand/ondemand.go:311-323; client.go:6257-6268,9125,10257` |
| `ondemand-06` | request() uses Java's `>` bounds (faithful off-by-one) | `OnDemand.java:287-314` | `pkg/jagex2/io/ondemand/ondemand.go:386-404` |
| `ondemand-08` | handleExtras loop-condition refactor is semantically equivalent to Java | `OnDemand.java:415-... (handleExtras while loop)` | `pkg/jagex2/io/ondemand/ondemand.go:620-679` |
| `objtype-01` | code9/code10 fields not ported as struct fields; wire reads preserved | `ObjType.java:64-73,291-294` | `pkg/jagex2/config/objtype/objtype.go:39-43,200-203` |
| `objtype-03` | Op hidden sentinel uses empty string instead of Java null | `ObjType.java:316-319` | `pkg/jagex2/config/objtype/objtype.go:224-229` |
| `bzip2-05` | decompress: es/N RUNA-RUNB accumulator 32-bit wrap not reproduced (not reachable) | `BZip2.java:381-388` | `pkg/jagex2/io/bzip2/bzip2.go:404-415` |
| `bzip2-07` | BZip2State: dead field cftabCopy omitted; dead size constants preserved | `BZip2State.java:62 (tb.G cftabCopy)` | `pkg/jagex2/io/bzip2state/bzip2state.go:3-57` |
| `collisionmap-02` | addLoc arg6 dead-guard carried for fidelity | `CollisionMap.java:168-193` | `pkg/jagex2/dash3d/collisionmap.go:145-171` |
| `collisionmap-03` | testWDecor 245.2 dead boolean param + NPE guard dropped (never ported) | `CollisionMap.java:491-541` | `pkg/jagex2/dash3d/collisionmap.go:450-511` |
| `pix32-03` | TransPlot blend parenthesized to reproduce Java mixed-precedence grouping | `Pix32.java:387` | `pkg/jagex2/graphics/pix32/pix32.go:410` |
| `pix32-04` | JPEG decode via image/jpeg seam replaces AWT Toolkit/MediaTracker/PixelGrabber | `Pix32.java:41-61` | `pkg/jagex2/graphics/pix32/pix32.go:37-79` |
| `loctype-01` | Opcode 74 breakroutefinding added at 245.2 — present and correct | `LocType.java:320-321,341-344` | `pkg/jagex2/config/loctype/loctype.go:253-254,270-273` |
| `loctype-03` | getModel Model-ctor boolean-swap pair verified mutually compensating | `LocType.java:489,420-423,531` | `pkg/jagex2/config/loctype/loctype.go:327,353-354,372-373` |
| `loctype-05` | modelCacheDynamic capacity 30->256 (wasm alloc-churn deviation) | `LocType.java:88` | `pkg/jagex2/config/loctype/loctype.go:25-32` |
| `loctype-06` | OnDemand.Prefetch param order swapped, compensated at call site | `LocType.java:393, OnDemand.java:391` | `pkg/jagex2/config/loctype/loctype.go:444, pkg/jagex2/io/ondemand/ondemand.go:492` |
| `component-03` | getImage try/catch -> defer/recover | `Component.java:493-514` | `pkg/jagex2/config/component/component.go:415-433` |
| `component-04` | Type==1 field95/field96 dead-write residue not stored | `Component.java:276-277` | `pkg/jagex2/config/component/component.go:187-193` |
| `component-05` | IOps empty-string null convention | `Component.java:303-306,363-366` | `pkg/jagex2/config/component/component.go:229-234,322-327` |
| `clientplayer-01` | static Model.empty reuse -> per-player seqModel field (wasm-alloc seam) | `ClientPlayer.java:367-368` | `pkg/jagex2/dash3d/entity/playerentity/clientplayer.go:313-317` |
| `clientplayer-02` | 245.2 hash shift <<8/<<16 vs 244 <<40/<<48 is a true no-op (int operand) - comment claim verified | `ClientPlayer.java:275-283` | `pkg/jagex2/dash3d/entity/playerentity/clientplayer.go:215-231` |
| `clientplayer-05` | NewModel4 spotanim ctor boolean reorder compensated at call site | `ClientPlayer.java:196` | `pkg/jagex2/dash3d/entity/playerentity/clientplayer.go:153` |
| `npctype-01` | field1010/1011/1012 (op 90/91/92) omitted, reads kept as discards | `NpcType.java:191-196 (decode op 90/91/92)` | `pkg/jagex2/config/npctype/npctype.go:174-175` |
| `npctype-03` | op[i]=null hidden-sentinel mapped to Go empty string | `NpcType.java:160-168 (decode op 30-39)` | `pkg/jagex2/config/npctype/npctype.go:147-156` |
| `npctype-04` | walkanim opcode-17 read order coupled with client getNpcPos* direct assign (pair verified) | `NpcType.java decode op17 + Client.java:8921-8925,9006-9010` | `pkg/jagex2/config/npctype/npctype.go:139-146 + client.go:5754-5758,1111-1115` |
| `npctype-05` | LruCache.Put(key,value) vs Java put(value,key) arg order | `NpcType.java:282 (getModel modelCache.put(model, this.id))` | `pkg/jagex2/config/npctype/npctype.go:228` |
| `packet-02` | sync.Pool replaces bounded LinkList caches; per-type caps (1000/250/50) and counts dropped | `Packet.java:49-97` | `pkg/jagex2/io/packet.go:25-99` |
| `packet-03` | PJStr/GJStr use a Latin-1<->UTF-8 transcode seam vs Java getBytes/new String | `Packet.java:165-170,229-235` | `pkg/jagex2/io/packet.go:183-190,256-263,287-312` |
| `signlink-03` | OpenSocket returns dial error, not Java 'could not open socket' text | `sign/signlink.java:217-229` | `pkg/sign/signlink/signlink.go:333-341` |
| `signlink-04` | Dropped publisher fields/cases and FileStream cache | `sign/signlink.java:105-170,250-253` | `pkg/sign/signlink/signlink.go:144-241` |
| `signlink-05` | active / threadliveid lifecycle re-entrancy guards not ported | `sign/signlink.java:75-99,117-118` | `pkg/sign/signlink/signlink.go:114-135` |
| `signlink-06` | reporterror drops the !active half of the guard | `sign/signlink.java:292-307` | `pkg/sign/signlink/signlink.go:464-491` |
| `signlink-07` | Wave/Midi disk round-trip replaced by in-memory consumer slot | `sign/signlink.java:139-156,255-290` | `pkg/sign/signlink/signlink.go:373-462` |
| `signlink-09` | mu/cond/slotMu replace Java synchronized busy-wait | `sign/signlink.java:217-253,231-243` | `pkg/sign/signlink/signlink.go:20-55,279-360` |
| `ground-02` | Always-false if (arg11>arg11) min/max bug preserved | `Ground.java:274` | `pkg/jagex2/dash3d/typ/ground.go:226-232` |
| `ground-03` | Dead final /14 scaling preserved | `Ground.java:289-290` | `pkg/jagex2/dash3d/typ/ground.go:237-238` |
| `pix2d-pix8-pixmap-03` | Go-only pix2d.Reset() test helper | `Pix2D.java (n/a)` | `pkg/jagex2/graphics/pix2d/pix2d.go:31-42` |
| `pix2d-pix8-pixmap-07` | PixMap(height,Component,width) -> NewPixMap(width,height) AWT->platform seam | `PixMap.java:29-48` | `pkg/jagex2/graphics/pixmap/pixmap.go:36-45` |
| `pix2d-pix8-pixmap-08` | PixMap.draw(x,Graphics,y) -> Draw(x,y) upload/blit seam | `PixMap.java:55-58` | `pkg/jagex2/graphics/pixmap/pixmap.go:54-69` |
| `pix2d-pix8-pixmap-09` | AWT ImageProducer/ImageObserver methods not ported | `PixMap.java:60-97` | `pkg/jagex2/graphics/pixmap/pixmap.go (none)` |
| `cliententity-03` | move() param reorder verified by callee body | `ClientEntity.java:158-198` | `pkg/jagex2/dash3d/entity/clententity.go:113-146` |
| `cliententity-04` | Pathing()/PathableEntity Go interface seam | `ClientEntity.java (n/a)` | `pkg/jagex2/dash3d/entity/clententity.go:198-212` |
| `cliententity-05` | minY -> MaxY inverted-lineage naming for height | `ClientNpc.java:24` | `pkg/jagex2/dash3d/entity/clientnpc.go:35-37` |
| `cliententity-06` | Model.empty static temp -> per-npc seqModel reuse + reordered ctor/transform helper pairs | `ClientNpc.java:59-77, NpcType.java:237-296` | `pkg/jagex2/dash3d/entity/clientnpc.go:27-85, npctype.go:197-249` |
| `cliententity-09` | hit()/Hit() uses Go range-N loop; param roles verified | `ClientEntity.java:264-272` | `pkg/jagex2/dash3d/entity/clententity.go:95-104` |
| `streams-02` | ClientStream.debug() not ported (debug-only diagnostic) | `ClientStream.java:186-197` | `pkg/jagex2/io/clientstream/clientstream.go (no counterpart)` |
| `streams-03` | FileStream (whole class) not ported — storage seam | `FileStream.java:1-250` | `pkg/sign/signlink/storage_disk.go + ondemand Cache interface (seam replacement)` |
| `wave-01` | byte[] silence/accumulation ported to Go []byte (uint8) — bit-identical | `Wave.java:148,160` | `pkg/jagex2/sound/wave/wave.go:144,153` |
| `wave-03` | static generate param order (id, loops) — documented 245.2 WS7 swap | `Wave.java:52` | `pkg/jagex2/sound/wave/wave.go:50` |
| `anim-meta-02` | ModelSource class -> Go interface + inline draw seam | `ModelSource.java:1-28` | `pkg/jagex2/dash3d/entity/modelsource.go:5-7; pkg/jagex2/dash3d/world3d/world3d.go:1284-1296` |
| `anim-meta-03` | LocChange extends Linkable -> LinkList wrapper seam | `LocChange.java:1-44` | `pkg/jagex2/dash3d/entity/locchange.go:9-28; pkg/jagex2/client/client.go:234,594,7390` |
| `proj-spotanim-01` | ClientProj.updateVelocity parameter remap is caller-coordinated | `ClientProj.java:97-110` | `pkg/jagex2/dash3d/entity/clientproj.go:55-75` |
| `proj-spotanim-03` | ClientProj.update seq block ported as early-return | `ClientProj.java:122-132` | `pkg/jagex2/dash3d/entity/clientproj.go:85-95` |
| `proj-spotanim-04` | Model(boolean,Model,boolean,boolean) ctor pairing still mutually compensating | `ClientProj.java:142 / Model.java:741` | `pkg/jagex2/dash3d/entity/clientproj.go:104 / model.go:736` |
| `proj-spotanim-06` | resize/Scale axis-permuted arg pairing | `Model.java:1318-1324` | `pkg/jagex2/dash3d/model/model.go:1304-1310` |
| `proj-spotanim-07` | MapSpotAnim.update nested do/while translated to Go do-while idiom | `MapSpotAnim.java:48-63` | `pkg/jagex2/dash3d/entity/mapspotanim.go:33-48` |
| `proj-spotanim-08` | ClientLocAnim corner-height permutation consistent end-to-end | `World.java:305-322 / ClientLocAnim.java:48-66` | `pkg/jagex2/dash3d/world/world.go:287-313 / clientlocanim.go:42-96` |
| `proj-spotanim-09` | ClientLocAnim.getModel advance subtracts frameLength without +1 | `ClientLocAnim.java:68-87` | `pkg/jagex2/dash3d/entity/clientlocanim.go:65-96` |
| `proj-spotanim-10` | Projectile double/float64 parity preserved; operator grouping matches | `ClientProj.java:97-135` | `pkg/jagex2/dash3d/entity/clientproj.go:55-96` |
| `isaac-jagfile-01` | Jagfile header size variables: un-scrambled deob naming (net-equivalent) | `Jagfile.java:35-46` | `pkg/jagex2/io/jagfile.go:26-41` |
| `isaac-jagfile-02` | Jagfile.read hash uses int32 truncation against unsigned-stored fileHash | `Jagfile.java:67-72` | `pkg/jagex2/io/jagfile.go:63-70` |
| `isaac-jagfile-03` | Jagfile.read iterates runes vs Java UTF-16 charAt | `Jagfile.java:64-66` | `pkg/jagex2/io/jagfile.go:64-67` |
| `config-small-01` | VarpType dead-field opcodes decoded as discards | `VarpType.java:73-115` | `pkg/jagex2/config/varptype/varptype.go:45-77` |
| `config-small-03` | UnkType not ported (dead deob artifact, still dead in 245.2) | `UnkType.java:1-40` | `(none)` |
| `datastruct-01` | JString iterates Go runes, not Java UTF-16 code units | `JString.java:17-127` | `pkg/jagex2/datastruct/jstring/jstring.go:13-141` |
| `datastruct-06` | LruCache.search dead-write field + sentinel==search branch dropped | `LruCache.java:15,46-57` | `pkg/jagex2/datastruct/lrucache.go:47-61` |
| `datastruct-08` | LruCache.Delete is a documented behavior change vs Java cached.unlink() | `ObjType.java:476-480` | `pkg/jagex2/datastruct/lrucache.go:63-86` |
| `typ-bundle-01` | Square IS a Linkable in Java; Go uses an owned DrawQueueNode (comment says addTail, Java uses push) | `Square.java:7; LinkList.java:20-50` | `pkg/jagex2/dash3d/typ/square.go:30-46` |
| `typ-bundle-02` | Sprite/Decor carry a Go-side MinY mirroring Java ModelSource.minY | `ModelSource.java:9-18; World3D.java:1328,1490,1546` | `pkg/jagex2/dash3d/typ/sprite.go:29-33; decor.go:20-24; world3d.go:331,422,1367,1581` |
| `opcodes-01` | SERVERPROT_LENGTH table byte-identical (257 entries, all -1/-2 markers) | `Protocol.java:12; Client.java:7116` | `protocol.go:14; client.go:9959,9962-9984` |
| `opcodes-02` | All 69 inbound + 76 outbound opcode constants match Java 245.2 by name and value | `Client.java:7099-8127,8127-8370,9136-9740` | `serverprot.go:9-82; clientprot.go:9-86` |
| `opcodes-04` | CLIENTPROT_LOOKUP not ported — dead deob artifact still dead at 245.2 | `Protocol.java:9` | `protocol.go:3-9` |
| `opcodes-05` | Menu-option opcode selection identical across 45 discriminators | `Client.java:9136-9740` | `client.go:4570-5360` |
| `opcodes-06` | InteractWithLoc arg remap consistent; opcode emitted in correct position | `Client.java:6866,6897-6900,9188-9736` | `client.go:6739,6770-6773,4833-5259` |
| `opcodes-07` | pIsaac and inbound opcode decode 32-bit + precedence handling sound | `Packet.java:108-110; Client.java:7106` | `packet.go:101-104; client.go:9957` |

---

## Coverage attestation

- Forward: 54/54 units completed; 674 Java methods walked; 0 Java methods
  missing in Go without documented justification (FileStream, MouseTracking, UnkType and
  deob/ObfuscatedName remain verified intentional non-ports in 245.2).
- Reverse: every Go/web file classified port / seam / test / tooling; 0 suspicious files.
- Wire opcodes: CLIENTPROT_LOOKUP (257 entries) and SERVERPROT_LENGTH (256 entries) verified
  element-by-element; all outbound/inbound/zone/menu opcode literals enumerated against Java
  245.2 (see `audit-245/units/opcodes.md`).
