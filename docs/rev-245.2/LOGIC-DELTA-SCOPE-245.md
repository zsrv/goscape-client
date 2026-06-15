# rev-245.2 Logic-Delta Scope (244 → 245.2)

Scope of the **game-logic delta** to apply on the `rev-245.2` branch (cut from the
complete rev-244 Go port). Unlike the 225→244 jump, **245.2 is the same
deobfuscation lineage as 244 — there is NO rename/restructure pass.** Every
`Client.java` method pairs 1:1 by name (144 methods, zero added/removed; the class
was merely reordered). The delta is therefore narrow and concentrated: a wholesale
**per-build wire-opcode renumbering**, a small set of **new config-format fields**,
a few **new handlers / hit-box / logic guards**, an **audio volume-scale migration**,
and a **Swing windowing modernization** (mostly no-op under the Go host shell).

> **Provenance.** This document was machine-generated on **2026-06-04** by a
> verified multi-agent line-by-line comparison of `Client-Java` `01f16088` (Java
> 244, base) vs `176a85f` (Java 245.2, target), per file/slice, with every claimed
> delta adversarially re-verified against both sources (read via `git show` only).
> Churn (obfuscator-key reshuffles, local/param renames, commutative operand
> reorders, `final`-drops, package-case moves, callee param reorders) was
> separated from genuine behavioural change. Sizes are S/M/L/XL.

## Executive summary

The raw 244→245.2 diff is large but **~99% churn**. Whole files are naming/reorder
only: `Pix3D`, `Pix32`, `Pix8`, `PixFont`, `Ground`, `World.java`, `WordFilter`,
`OnDemand`, `ObjType`, `CollisionMap` (modulo two dead guards), and all of
`datastruct`/`io` bundles except `Protocol`. The substantive work is **six
workstreams**:

1. **Wire protocol — opcode renumbering** (the dominant, pervasive change; outbound
   `pIsaac`, inbound `ptype`, zone dispatch, menu-option, and the `Protocol.java`
   lookup/length tables all renumber; one new fixed-length-4 server message).
2. **New inbound + interface logic** (`IF_SETSCROLLPOS` handler; `Component.swappable`
   + `activeOverColour` driving objDrag swap, drag eligibility, and type-3/4 render;
   `updateInterfaceContent` friend-range narrowing; chat/chatback hit-box widening).
3. **Config cache-format additions** (`Component.swappable`/`activeOverColour`,
   `LocType` opcode 74 `breakroutefinding`, `UnkType` field retype/additions).
4. **Audio volume-scale migration to centibels** (`setMidiVolume`/`setWaveVolume`
   constants `{128,96,64,32}` → `{0,-400,-800,-1200}`; signlink defaults `96→0`,
   `"none"→null`).
5. **signlink publisher reconciliation + package decision** (clientversion 244→245,
   reporterror URL, defaults; the 244 deob's wrapper-side consumer is *removed* in
   245.2 — the Go seam stays, this WS is publisher-side only).
6. **Windowing / coordinate model** (`GameShell`/`ViewBox` AWT→Swing: drop
   inset-subtraction, `getBaseComponent` returns shell, `setPreferredSize`,
   `initApplet` param-order swap, login client-version byte 244→245).

Plus a handful of **standalone small deltas** (`World3D.setDecorOffset` 5th-param
guard, `Model.decode` orientation restructure, `CollisionMap` dead-guard fidelity,
`Component.loadModel` cast timing, `ClientPlayer` hash-shift constants, NpcType
walkanim coupling, `getJagFile` error-text rework).

## Cross-cutting hazards (read first)

- **EVERY wire opcode renumbers between 244 and 245.2.** The Go port currently
  **hard-codes 244 numbers** in `pkg/jagex2/io/clientprot.go`, `serverprot.go`, and
  `protocol.go`. These are NOT `@ObfuscatedName` annotations — they are the literal
  bytes emitted/dispatched on the wire, so a naive carry-forward sends/mis-routes
  every packet. Re-derive **all** outbound (`pIsaac`), inbound (`ptype`), zone-
  dispatch, and menu-option opcodes from Java 245.2 + `Protocol.java`. **Collision
  trap:** some 245.2 values reuse old 244 values for a *different* message (e.g.
  zone `LOC_DEL=198` is 244's `MAP_ANIM` value; `OBJ_REVEAL` 244=69 collides with
  nothing in 245.2). A partial table = silent mis-routing.
- **Same lineage, no rename pass.** Do NOT re-run a RENAME-MAP exercise. Naming
  policy (user decision 2026-06-04): in methods *touched by a delta*, adopt 245.2's
  local/param names with `// Java:` refs; untouched methods keep current Go names.
- **Coupled multi-file edits must land together.** Two pairs are net-neutral at
  runtime but wrong if half-ported: (a) `Component.swappable`/`activeOverColour`
  field+decode in `config/component` ↔ their consumers in `client`; (b)
  `getNpcPos*` walkanim de-swap in `client` ↔ `NpcType` walkanim read-order swap +
  `readyanim→runanim` rename. Port both halves in the same commit.
- **The `Model(boolean,boolean,Model)` ctor trap (carried from 244 churn list).**
  Its signature text is identical 244↔245.2 but the two booleans' body meaning is
  swapped, compensated by swapped `LocType` call sites. If any 245.2-driven touch
  lands near this ctor, do not "fix" it in isolation.
- **The game cannot run headless.** Build/vet/`test -race`/gofmt/golangci-lint gate
  every increment; behavioural verification needs a host run against the smoke
  server (Engine-TS branch `245.2 @ 3c16994c`) after WS1+WS2 land.

## Confirmed-delta inventory (by subsystem)

### Wire protocol — outbound opcodes (`pIsaac` literals)
All payload widths/field order UNCHANGED; only the leading opcode literal differs.
- `useMenuOption` — **wholesale reassignment of ~50 opcodes** (43 named + 7 OPLOC via
  `interactWithLoc`): IF_BUTTON 39→177, OPHELDU 58→126, OPNPC1-5/U/T, OPOBJ1-5/U/T
  (OPOBJ4 17→17 unchanged), OPHELD1-5/T, INV_BUTTON1-5, OPPLAYER1-4/U/T,
  RESUME_PAUSEBUTTON 11→239, OPLOGIC1-9, OPLOC1-5/T/U. **[L]**
- `handleInputKey` — MESSAGE_PRIVATE 170→99, its CHAT_SETMODE 98→8, RESUME_P_COUNTDIALOG
  190→241, CLIENT_CHEAT 76→11, MESSAGE_PUBLIC 171→78, its CHAT_SETMODE 98→8. **[M]**
- `handleChatModeInput` — CHAT_SETMODE 98→8 (all three public/private/trade branches). **[S]**
- `addFriend` FRIENDLIST_ADD 9→116; `removeFriend` FRIENDLIST_DEL 69→61;
  `addIgnore` IGNORELIST_ADD 203→20; `removeIgnore` IGNORELIST_DEL 207→4. **[S each]**
- `closeInterfaces` CLOSE_MODAL 187→245; `drawGame` TUTORIAL_CLICKSIDE 233→243
  **and** ANTICHEAT_CYCLELOGIC2 148→223 (the comparer under-counted drawGame by one
  — both must port). **[S]**
- `handleInterfaceAction` IF_PLAYERDESIGN 8→150, REPORT_ABUSE 251→205. **[S]**
- `tryMove` MOVE_GAMECLICK 63→182, MOVE_MINIMAPCLICK 56→198, MOVE_OPCLICK 167→216. **[M]**
- Anticheat/keepalive: `handleTabInput` CYCLELOGIC1 46→136; `drawScene`/`updatePlayers`
  CYCLELOGIC2 148→223 & CYCLELOGIC6 215→112; `updateLocChanges` CYCLELOGIC5 232→63;
  `updateGame`/`buildScene` NO_TIMEOUT 107→206, IDLE_TIMER 146→102, EVENT_TRACKING
  217→19, INV_BUTTOND 81→7, CYCLELOGIC3 144→181, CYCLELOGIC4 41→94. **[S each]**

### Wire protocol — inbound dispatch + tables
- `readPacket` — **all ~60 inbound `ptype` opcodes renumbered** (per-build); handler
  bodies match 1:1 except the new handler below. Examples: UPDATE_FRIENDLIST 70→109,
  REBUILD_NORMAL 165→66, UPDATE_INV_PARTIAL 132→95, IF_SETOBJECT 164→153. **[L]**
- `readZonePacket` — **all 10 zone opcodes renumbered** (LOC_ADD_CHANGE 232→119,
  LOC_DEL 125→198, LOC_ANIM 155→71, OBJ_ADD 234→94, OBJ_DEL 39→13, MAP_PROJANIM
  137→187, MAP_ANIM 198→141, OBJ_REVEAL 69→190, LOC_MERGE 29→188, OBJ_COUNT
  209→151) — value-collision trap noted above. **[L]**
- **NEW handler `IF_SETSCROLLPOS` (`ptype==226`)** — g2 comId, g2 pos; if Component
  found and type==0, clamp pos to `[0, scroll-height]`, set `scrollPosition`. 245.2
  has 61 handlers vs 244's 60. **[M]**
- `Protocol.java` `CLIENTPROT_LOOKUP[257]` fully reshuffled (same multiset; pure
  client→wire re-encoding) and `SERVERPROT_LENGTH[257]` reindexed **+ one new
  fixed-length-4 server message** (value 0 count 198→197, value 4 10→11; active
  opcode count 59→60). Adopt both 245.2 tables verbatim. **[M]**

### Interface / game-logic
- `updateGame` — **NEW objDrag branch** gated on `Component.swappable`: direct slot
  move (`invSlotObjId/Count[dst]=...[src]; ...[src]=-1/0`) prepended before the
  existing mode==1 walk / `swapObj` paths. **[L]** (depends on `Component.swappable`)
- `handleMouseInput` — obj-drag eligibility widened `if (draggable)` →
  `if (draggable || swappable)`. **[S]** (depends on `Component.swappable`)
- `updateInterfaceContent` — first clientCode guard upper bound 900→800; reassigns
  component codes 801..900 from friend-**name** display to friend-**world** display
  (`-=601` first branch vs `-=701` second). Verify against matching interface defs. **[M]**
- `drawInterface` — type-3 (rect) rewritten with hovered-flag + `executeInterfaceScript`
  + active/over-colour selection (`activeColour`/`activeOverColour`/`colour`/
  `overColour`) and `trans` (was `alpha`); type-4 (text) gains `activeOverColour`
  branch (the %1..%5 / `\n` loop restructure is churn). **[L]** (depends on Component fields)
- `handleInput` — chatback hit-box outer bound `mouseX<426`→`<496`; inner secondary
  branch re-guarded to `mouseY<434 && mouseX<426`. **[S]**
- `handlePrivateChatInput` — split-private-chat right-click region drops fixed
  `<516`, computes dynamic `width = stringWid("From:  "+sender+msg)+25` clamped to
  450, gates on `mouseX < width+4`; `@cr1@/@cr2@` strip becomes `if/else if`. **[M]**
- `updateLocChanges` — new tile-bounds guard `loc.x>=1 && loc.z>=1 && loc.x<=102 &&
  loc.z<=102` before applying the queued newType loc change. **[S]**
- `getJagFile` — download error-handling reworked: per-path `errorMessage`
  ("Connection/Null/Bounds/Unexpected error", "Checksum error: "+crc), retry text
  `errorMessage+" - Retrying in "+i`, split `catch(IOException/NPE/AIOOBE/Exception)`;
  a dead `StringBuffer("Length error: ")` build before EOF throw. Retry/backoff math
  unchanged. **[M]**

### Config cache-format
- `Component.unpack` — **NEW `swappable` g1()==1** in the type==2 block (between
  `usable` and `marginX`; shifts all later type-2 reads by 1 byte); **NEW
  `activeOverColour` g4()** in the type==3||4 block (after `overColour`; shifts by 4
  bytes). New fields `swappable` (bool) and `activeOverColour` (int); `alpha`→`trans`
  rename. **[S]** (load-bearing for the interface logic above)
- `LocType.decode` — **NEW opcode 74** `breakroutefinding=true`; new post-loop block
  `if (breakroutefinding) { blockwalk=false; blockrange=false; }`; `reset()` clears
  it; new field `breakroutefinding`. **[S]**
- `UnkType` — `mc.e` retyped `boolean true`→`int 1`; `mc.i` initializer `false`→`true`;
  two new booleans `mc.j`/`mc.k` (`=false`). UnkType is a deob stub (244 policy
  marked it not-ported — re-confirm whether any 245.2 path loads it). **[S]**

### Audio (volume-scale migration)
- `updateVarp` clientcode==3 — `setMidiVolume` constants `{128,96,64,32}` →
  `{0,-400,-800,-1200}` (centibel scale); param order swapped `(int,boolean)`→
  `(boolean,int)`. **[M]**
- `updateVarp` clientcode==4 — `setWaveVolume` constants `{128,96,64,32}` →
  `{0,-400,-800,-1200}` (single int param; no reorder). **[M]**
- signlink defaults: `midivol 96→0`, `wavevol 96→0`, `midi "none"→null`,
  `wave "none"→null`. (See WS5.) **[S]**

### signlink / windowing / standalone
- `signlink` — `clientversion 244→245`; `reporterror244.cgi`→`reporterror245.cgi`;
  the 244 deob's wrapper-side **audio consumer is removed** (audioLoop/MidiPlayer/
  midiFading*/curPosition/Position enum — Go keeps its own seam). **[L scope, mostly
  no-port]**
- `GameShell`/`ViewBox` — AWT `Frame`→Swing `JFrame`; drop inset-subtraction in
  mousePressed/Dragged/Moved; `getBaseComponent` always `return this`; add
  `setPreferredSize`; `initApplet(height,width)`→`(width,height)` (caller flips
  `initApplet(503,765)`→`(765,503)`); ViewBox title now version-suffixed; remove
  `insets` field + `getGraphics` override. **[M]**
- `login` — client-version byte `p1(244)`→`p1(245)`. **[S]**
- `World3D.setDecorOffset` — new 5th param + `if (arg3 == -23232)` guard around the
  z-axis offset; all callers pass -23232 (net runtime == 244). **[M, faithful-port only]**
- `CollisionMap` — `addLoc` gains a trailing `boolean` + `if (arg6) return;` (all
  callers pass false → dead); `testWDecor` drops a `boolean` param + its NPE branch
  (sole caller always passed true → dead). Port for signature/structural fidelity. **[S]**
- `Model.decode` — orientation block restructured into independent `if(==1..4)`
  blocks; equivalent for valid data (1-4), diverges only on impossible 0/>4. **[S]**
- `Component.loadModel` — cache-key cast timing `(long)type<<16`→`(long)((arg0<<16)+arg1)`;
  equivalent for valid ids. **[S]**
- `ClientPlayer.getAnimatedModel` — held-item hash shifts `<<40→<<8` (left) and
  `<<48→<<16` (right). Real cache-key change. **[S]**
- `getNpcPosNewVis`/`getNpcPosExtended` — walkanim de-swap removed (direct assign)
  + `readyanim`←`type.runanim`; **coupled** with `NpcType` walkanim read-order swap +
  `readyanim→runanim` rename. Net-neutral only if both land together. **[S, coupled]**
- `Wave` — static `waveBytes`/`waveBuffer` initializers moved into `unpack()` (lazy);
  `delays`→`delay` rename. Go port already lazy — likely a no-op. **[S]**
- `main` banner — `"...release #" + 244` → `+ signlink.clientversion`. **[S]**

## Workstream decomposition

### WS1 — Wire protocol: opcode renumbering + tables  [L, critical path]
Re-derive ALL outbound/inbound/zone/menu-option opcodes from Java 245.2 and adopt
the 245.2 `CLIENTPROT_LOOKUP`/`SERVERPROT_LENGTH` tables verbatim. Update
`pkg/jagex2/io/{clientprot,serverprot,protocol}.go` plus any inline literals in
`client`. Add the new fixed-length-4 server message length and wire the new
`SERVERPROT` opcode for `IF_SETSCROLLPOS`. **Rationale:** pervasive and load-bearing;
nothing connects to a 245.2 server until this lands. Mechanical but high-volume and
collision-prone — extract programmatically from the deob `// LABEL` comments and
diff multisets to catch mis-mappings.

### WS2 — New inbound handler + interface/render logic  [M]
Port `IF_SETSCROLLPOS` (ptype 226) handler; `updateGame` swappable objDrag branch;
`handleMouseInput` drag-eligibility widening; `drawInterface` type-3/4 active/over-
colour + `trans`; `updateInterfaceContent` friend-range 900→800; `handleInput`
chatback hit-box; `handlePrivateChatInput` dynamic width; `updateLocChanges` bounds
guard. **Rationale:** the genuinely new game behaviour; depends on WS3 fields
(`swappable`, `activeOverColour`, `trans`). Land WS3 first or in the same series.

### WS3 — Config cache-format additions  [S]
`Component.swappable` (+g1 read, type-2), `activeOverColour` (+g4 read, type-3/4),
`alpha→trans` rename in `pkg/jagex2/config/component`; `LocType` opcode 74
`breakroutefinding` + post-loop block + reset in `pkg/jagex2/config/loctype`;
`UnkType` field changes (confirm load path first). **Rationale:** small, low-risk,
testable against cache files; the data-side dependency for WS2.

### WS4 — Audio volume-scale migration  [S–M]
Migrate `setMidiVolume`/`setWaveVolume` constants to the centibel scale
`{0,-400,-800,-1200}` and apply the signlink default changes (96→0, "none"→null) in
the Go publisher path. **Reconcile against the existing Go audio seam**
(`pkg/jagex2/sound/audio/`, WS5 of rev-244) — the Go consumer already implements a
linear vol model, so this is a constant/mapping change at the publisher + a check
that the seam interprets the new centibel inputs correctly, NOT a re-port.
**Rationale:** isolated; verify the gain mapping end-to-end on host.

### WS5 — signlink publisher reconciliation + package decision  [S–M]
Publisher-side only: `clientversion 244→245`, `reporterror` URL literal, the
midivol/wavevol/midi/wave default changes (shared with WS4), `main` banner now reads
`signlink.clientversion`. **The 244 deob's wrapper-side consumer removal is NOT
ported** — the Go seam stays (documented). Decide the package move:
`pkg/jagex2/client/sign/signlink` → `pkg/sign/signlink` to mirror 245.2's authentic
top-level `sign/signlink.java` (optional, cosmetic; defer unless cheap).
**Rationale:** trivial constants; the package-move decision is the only judgement call.

### WS6 — Windowing / coordinate model (GameShell/ViewBox) + login byte  [M]
Drop inset-subtraction in the three mouse handlers; make `getBaseComponent` return
the shell; add `setPreferredSize`-equivalent; honour `initApplet` param-order swap;
login client-version byte 244→245; ViewBox title version suffix. **Most of this is
no-op under the Go GLFW host shell** (already heavily diverged) — map, don't
line-port; the load-bearing item is the login byte and confirming the Go input path
takes canvas-relative coords (it already does under GLFW). **Rationale:** small real
surface inside a large AWT→Swing diff; isolate the login byte if needed for WS1
connectivity.

### WS7 — Standalone small deltas (faithful-port fidelity)  [S, batched]
`World3D.setDecorOffset` 5th-param guard; `CollisionMap` addLoc/testWDecor dead-guard
fidelity; `Model.decode` orientation restructure; `Component.loadModel` cast timing;
`ClientPlayer` hash shifts (<<40→<<8, <<48→<<16); `getNpcPos*`↔`NpcType` walkanim
coupling; `getJagFile` error-text rework; `Wave` lazy-init (likely no-op). **Rationale:**
each is independently small; batch into one or two commits. The ClientPlayer hash
shifts and the NpcType coupling are the only ones with observable runtime effect.

## Suggested execution order (dependency-ordered)

```
WS3 Config fields ──┐  (small, testable vs cache; data-side dep for WS2)
                    ├─→ WS2 New handler + interface/render logic
WS1 Opcodes/tables ─┘─→ (connectivity) ─→ host smoke test (Engine-TS 245.2 @3c16994c)
WS6 login byte ─────┘   (needed for the handshake)

WS4 Audio scale ────────  (independent; reconcile vs Go seam)
WS5 signlink publisher ──  (independent; shares the 96→0 defaults with WS4)
WS7 Standalone deltas ───  (independent; faithful-port batch, any time)
```

1. **WS1 Opcodes/tables** + **WS6 login byte** — required for any 245.2-server
   connection; do these first so a host smoke test becomes possible.
2. **WS3 Config fields** — small warm-up; data-side prerequisite for WS2.
3. **WS2 Interface/render logic** — depends on WS3 (`swappable`/`activeOverColour`/
   `trans`); land after or with WS3.
4. **WS4 Audio**, **WS5 signlink**, **WS7 standalone** — independent; slot anywhere.
   WS4 and WS5 share the signlink default changes (96→0, "none"→null) — coordinate
   so they don't double-edit.
5. **Host smoke test** against Engine-TS branch `245.2 @ 3c16994c` after WS1+WS6
   (connectivity) and WS2+WS3 (interface behaviour).

Each increment: faithful 1:1 port + build/vet/`test -race`/gofmt/golangci-lint gate;
commit small with `--no-gpg-sign`; end with the full parity audit per
`PORTING-LESSONS.md` §5.

## DO-NOT-PORT items
- **The 244 deob's wrapper-side audio consumer removal** — `signlink` audioLoop /
  `MidiPlayer` / midiFading* / curPosition / Position enum were *added* by the 244
  deob and *removed* in 245.2. The Go port already implements its own audio seam
  (`pkg/jagex2/sound/audio/audioloop.go`, rev-244 WS5). Keep the Go seam; do NOT
  delete it to match 245.2's removal. WS5 is publisher-side only.
- **Pure obfuscator churn** — `@ObfuscatedName` key reshuffles, local/param renames,
  `final`-drops, commutative operand reorders, hex↔decimal literal reformatting,
  callee param reorders (recorded only for the naming policy). Field renumbering
  (`field1264→field1504`, flameBuffer2/3 role swap) is name churn.
- **Dead/removed deob placeholders** — `Client.field1538` (removed dead boolean);
  `WordFilter.filterDomains` added dead `boolean var6`; the `CollisionMap`/`Model`
  dead branches are ported for *structural* fidelity only (they never fire).
- **`UnkType`** — was a not-ported stub at 244; re-confirm 245.2 still never loads it
  before deciding whether the field changes need porting at all.
- **`MidiPlayer.java`** — entire file; never existed as a Go file (the consumer is
  the Go seam).
- **AWT/Swing windowing internals** — `ViewBox` JFrame/BorderLayout/pack/insets,
  `getGraphics` override removal: no-op under the GLFW host shell; map the
  *coordinate consequences* (canvas-relative mouse, `getBaseComponent`) only.

## Naming-policy statement
Same deob lineage as 244 — **no rename pass.** Per user decision (2026-06-04): in any
method **touched by a delta**, adopt 245.2's local/param names with a `// Java: <name>
(<file>:<lines>)` reference comment; **untouched** methods keep their current Go
names. Callee param-reorders/renames (Packet.pdata, addMessage, Model API, Pix2D/
PixFont helpers, World3D, etc.) are adopted only where a touched method's call sites
already need editing — they carry no behaviour and must not trigger a sweep of
untouched callers.

## Open questions to resolve during the port
- **`updateInterfaceContent` 900→800** — verify against the matching 245.2 interface
  definitions that codes 801..900 are genuinely meant as friend-**world** slots (the
  change is real in source; confirm the server/cache agrees so the reassignment is
  correct, not a regression).
- **`SERVERPROT_LENGTH` new length-4 message** — identify which opcode it is and what
  handler consumes it; a length-table-only port without the matching handler would
  desync the inbound reader the first time the server sends it.
- **`IF_SETSCROLLPOS` (ptype 226)** — confirm the Go `Component` has a `scrollPosition`
  field and a scroll-height accessor to clamp against; add if missing.
- **Audio centibel scale ↔ Go seam** — does the existing Go consumer expect the
  244 linear `vol/256` model or can it take the 245.2 centibel inputs directly? The
  publisher now writes `{0,-400,-800,-1200}` into `signlink.midivol`; confirm the
  Go gain mapping produces correct loudness (the rev-244 seam used vol/128 per the
  TS reference — re-derive for 245.2 and host-verify).
- **signlink package move** — adopt `pkg/sign/signlink` (authentic 245.2 path) or
  keep `pkg/jagex2/client/sign/signlink`? Cosmetic; decide on cost vs the import
  fan-out across the `client` package.
- **`UnkType` load path** — does any 245.2 code actually load `UnkType`? If still a
  pure stub, the field retype/additions are not-ported.
- **`Component.swappable` consumer breadth** — verify the only two consumers are
  `updateGame` objDrag and `handleMouseInput` drag eligibility (no third site that a
  partial port would miss).
- **Host smoke test** — defer the WS2 interface checks (objDrag swap, type-3/4 over-
  colours, scrollbar) to after a 245.2 server connection; they cannot be exercised
  headless.