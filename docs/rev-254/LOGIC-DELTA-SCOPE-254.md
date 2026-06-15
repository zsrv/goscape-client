# rev-254 Logic-Delta Scope (245.2 → 254)

Scope of the **game-logic delta** to apply on the `rev-254` branch (cut from the
complete rev-245.2 Go port @ `05a7659`). The mechanical class-rename pass
(`RENAME-MAP.md` Passes A–E: `clientbuild`/`world`/`iftype`/`JagFile`/protocol
move) is **already landed** — this document covers everything else.

> **Provenance.** Machine-generated 2026-06-04 by workflow `wf_fc8ded6b-5ca`
> (173 agents incl. resume; the first launch `wf_d3661243-a27` aborted on an
> args-passing defect and produced no findings). Method: per-unit comparison of
> `Client-Java` `176a85f` (245.2, base) vs `2e62978` (254, target), all reads
> via `git show` only; methods paired by name with 1:1 name-set checks,
> mismatches resolved by body-shape pairing; **every claimed real delta
> adversarially re-verified** against both pins. Result: 72/72 units covered,
> **97 real-delta claims → 77 confirmed + 15 amended (92 port-actionable) +
> 5 refuted**, 106 signature-only changes, churn separated (member obf-key
> reassignment, `argN` local regression, reflow, compensated reorders).
> Treat claims as strong hints, not gospel — verify against `@2e62978` while
> porting. Full agent output: session task `wa53nfw21` (untracked scratch).

## Executive summary

The raw diff (~34k lines) is ~95% churn, but the real delta is the largest
since 225→244. Beyond the usual **full wire-opcode renumber**, 254 brings
genuinely new systems: **varbits**, a **clientscript value VM with arithmetic
operators**, **server-driven player right-click options**, a **friend-server
connection state machine**, **player→NPC transmog**, **mouse/camera/focus
telemetry**, a **rewritten anticheat layer**, and a pervasive
**`animHasAlpha`→`AnimFrame.shareAlpha` alpha-sharing refactor**. A heavy
**method-rename layer** sits on top: Client.java pairs only 85 of ~145 methods
by name (60 base / 59 target resolved by body-shape pairing — see appendix).

Workstreams in dependency order:

1. **WS1 — Wire protocol: opcode renumber + framing + login [XL, critical
   path]** — nothing connects until this lands.
2. **WS2 — Config deltas (LocType dominates) [L]**
3. **WS3 — animHasAlpha → shareAlpha alpha system [M]** (cross-cutting with
   WS2; land together or immediately after)
4. **WS4 — Varbits + clientscript VM + Stats [M]**
5. **WS5 — New protocol features (player options, friend status, transmog,
   telemetry, anticheat) [XL]** (requires WS1)
6. **WS6 — Scene/render deltas [M]**
7. **WS7 — Stragglers + method-rename adoption [M]**

---

## WS1 — Wire protocol: opcode renumber + framing + login [XL]

**Every opcode table re-derived from `2e62978` by exhaustive enumeration; a
second agent independently re-derived the target tables and found ZERO
discrepancies.** Never carry 245.2 values — collisions abound (e.g. inbound
225 is 254's CAM_SHAKE; outbound 245 is now OPOBJU while 245.2's 245 was
CLOSE_MODAL).

- **`CLIENTPROT_LOOKUP`** (Protocol.java:9 @2e62978): 256 of 257 entries
  changed — same 0..255 multiset, new permutation. Replace verbatim.
  *(Go note: the rev-245.2 port intentionally does NOT carry this table —
  it is a dead deob artifact with zero readers, re-verify that's still true
  at 254 before porting; the `client/protocol.go` header documents this.)*
- **`SERVERPROT_LENGTH`** (Protocol.java:12 @2e62978): 107 of 257 entries
  changed; NOT a permutation (one extra `-1`, one extra `1`). Replace
  `SERVERPROT_SIZES` in `client/protocol.go` verbatim (full array below).
- **Outbound opcodes** (83 distinct; `pIsaac(N)` literal IS the wire opcode):
  full table below. The 7 `OPLOC*` sites route through
  `interactWithLoc(...)`'s `pIsaac(arg4)` re-emission (Client.java:6312).
- **Inbound opcodes** (60 single-opcode handlers + 10-opcode zone set):
  full tables below. `tcpIn` (was `readPacket`) at Client.java:6516;
  `zonePacket` (was `readZonePacket`) at :7567; opcodes 70/88 share the
  `LOC_ADD_CHANGE || LOC_DEL` branch (88 ⇒ delete).
- **Login** (Client.java:2429+): version byte `p1(245)→p1(254)`; signlink
  `clientversion = 254` + `reporterror254.cgi` (signlink.java:62).
- **Login reply protocol changed** (Client.java:2444): reply 2 now reads two
  extra bytes — `staffmodlevel = stream.read(); mouseTracked = stream.read()==1`
  (new static field); replies 18/19 (staffmod variants) REMOVED; NEW replies
  **21** (profile-transfer countdown: reads g1 delay, counts down redrawing
  `titleScreenDraw(true)` — note the method gained a boolean param — then
  retries login) and **-1** (no response); default branch gains
  `System.out.println("response:" + var8)`.
- **NPC capacity doubled** (Client.java:55/61/2496/7456 @2e62978): `npcs` and
  `npcIds` arrays 8192→**16384** + BOTH clear loops.
- **NPC local-index field 13→14 bits** in `getNpcPosNewVis`
  (Client.java:8352): `gBit(14)`, sentinel `16383`.
- **New outbound `MAP_BUILD_COMPLETE` (134)** sent by `checkScene` after
  `mapBuild()` (Client.java:3121).
- **Inbound `SET_PLAYER_OP` (204)** and **`FRIENDLIST_LOADED` (255)** are new
  message types whose handlers belong to WS5 features; the dispatch entries
  land here.

### Outbound table (254, complete — opcode / label / site @2e62978)

```
4   ANTICHEAT_CYCLELOGIC3   3321        134 MAP_BUILD_COMPLETE      3121
6   MOVE_GAMECLICK          6480        141 OPOBJ1                  8856
8   EVENT_APPLET_FOCUS      2813,2819   142 EVENT_TRACKING          2828,6971
9   FRIENDLIST_ADD          10979       143 OPNPC1                  8740
13  IF_PLAYERDESIGN         10606       144 IDLE_TIMER              2968
17  OPPLAYER2               9081        146 RESUME_PAUSEBUTTON      9039
18  OPPLAYER3               9063        147 OPLOC5                  9128*
26  OPLOCT                  8745*       160 INV_BUTTON4             8956
28  ANTICHEAT_OPLOGIC1      8708        161 RESUME_P_COUNTDIALOG    4352
33  OPLOC1                  9100*       162 ANTICHEAT_OPLOGIC9      8609
36  ANTICHEAT_CYCLELOGIC6   5407        163 OPHELD4                 8613
47  OPOBJ4                  8866        176 INV_BUTTOND             2908
51  ANTICHEAT_CYCLELOGIC1   5505        178 OPOBJ3                  8830
56  ANTICHEAT_OPLOGIC3      8836        181 INV_BUTTON1             8968
58  CLOSE_MODAL             4050        182 ANTICHEAT_CYCLELOGIC7   2927
59  INV_BUTTON3             8976        187 ANTICHEAT_OPLOGIC7      8852
62  INV_BUTTON5             8980        189 IGNORELIST_ADD          11032
67  OPOBJ2                  8844        192 OPPLAYER1               8901,9073
68  OPPLAYERT               9111        193 IGNORELIST_DEL          11049
69  OPNPC3                  8724        195 OPNPC2                  8732
70  INV_BUTTON2             8972        200 OPHELDU                 8647
72  OPPLAYER4               8891,9059   201 TUTORIAL_CLICKSIDE      5177
74  OPHELD5                 8625        202 OPOBJT                  8813
77  ANTICHEAT_OPLOGIC2      9183        203 REPORT_ABUSE            10623
80  OPHELD3                 8629        206 ANTICHEAT_OPLOGIC8      8862
83  MESSAGE_PUBLIC          4446        213 OPLOC2                  8712*
84  FRIENDLIST_DEL          11000       214 MESSAGE_PRIVATE         4307
86  CLIENT_CHEAT            4383        220 MOVE_MINIMAPCLICK       6485
87  OPLOC4                  8801*       225 ANTICHEAT_CYCLELOGIC2   6270
91  EVENT_CAMERA_POSITION   2806        226 ANTICHEAT_CYCLELOGIC4   4268
97  OPOBJ5                  8840        228 OPHELD2                 8617
98  OPLOC3                  9187*       230 OPPLAYER5               9077
100 ANTICHEAT_CYCLELOGIC5   5956        231 OPNPCT                  9158
102 OPHELDT                 8588        232 EVENT_MOUSE_MOVE        2711
113 OPPLAYERU               9172        233 ANTICHEAT_OPLOGIC5      8887,9055
118 OPNPC5                  8736        234 EVENT_MOUSE_CLICK       2793
119 OPNPCU                  9023        239 NO_TIMEOUT              3028,3154,…
121 ANTICHEAT_OPLOGIC4      8897,9069   240 OPLOCU                  8913*
122 OPNPC4                  8728        243 OPHELD1                 8621
127 MOVE_OPCLICK            6490        244 IF_BUTTON               8673,8751,9088
129 CHAT_SETMODE            4009,…      245 OPOBJU                  9006
131 ANTICHEAT_OPLOGIC6      8964
```
(* = OPLOC family, emitted via `interactWithLoc`'s `pIsaac(arg4)`.)

### Inbound table (254, complete — opcode / label / length)

```
0   CAM_LOOKAT            6     141 IF_OPENCHAT             2
3   IF_SETNPCHEAD         4     143 UPDATE_REBOOT_TIMER     2
5   P_COUNTDIALOG         0     146 LAST_LOGIN_INFO         10
14  IF_SETSCROLLPOS       4     159 UPDATE_ZONE_FULL_FOLLOWS 2
21  LOGOUT                0     161 IF_SETPLAYERHEAD        2
24  CHAT_FILTER_SETTINGS  3     163 MIDI_SONG               2
25  SYNTH_SOUND           5     164 UPDATE_RUNWEIGHT        2
27  IF_SETPOSITION        6     167 CAM_RESET               0
28  UPDATE_INV_FULL       var-g2 168 UPDATE_INV_STOP_TRANSMIT 2
29  FINISH_TRACKING       0     170 UPDATE_INV_PARTIAL      var-g2
38  IF_SETCOLOUR          4     173 UPDATE_ZONE_PARTIAL_FOLLOWS 2
41  IF_SETTEXT            var-g2 174 IF_CLOSE               0
55  CAM_MOVETO            6     186 VARP_SMALL              3
58  TUT_FLASH             1     187 IF_OPENSIDE             2
60  MESSAGE_PRIVATE       var-g1 196 VARP_LARGE             6
61  UPDATE_ZONE_PARTIAL_ENCLOSED var-g2 197 IF_OPENMAIN     2
63  UPDATE_IGNORELIST     var-g2 203 RESET_ANIMS            0
64  HINT_ARROW            6     204 SET_PLAYER_OP           var-g1
73  MESSAGE_GAME          var-g1 209 REBUILD_NORMAL         4
75  SET_MULTIWAY          1     211 IF_SETMODEL             4
85  IF_OPENOVERLAY        2     213 UPDATE_PID              3
87  PLAYER_INFO           var-g2 222 IF_SETOBJECT           6
91  IF_SETTAB             3     225 CAM_SHAKE               4
94  UPDATE_RUNENERGY      1     227 IF_SETHIDE              3
95  IF_SETANIM            4     239 TUT_OPEN                2
108 UNSET_MAP_FLAG        0     242 MIDI_JINGLE             4
111 UPDATE_FRIENDLIST     9     249 IF_OPENMAIN_SIDE        4
123 NPC_INFO              var-g2 251 ENABLE_TRACKING        0
136 UPDATE_STAT           6     255 FRIENDLIST_LOADED       1
138 IF_SETTAB_ACTIVE      1
140 RESET_CLIENT_VARCACHE 0
```

### Zone-update set (shares the inbound number space; dispatched to `zonePacket`)

```
8 OBJ_REVEAL(7)  30 LOC_ANIM(4)  37 MAP_PROJANIM(15)  70 LOC_ADD_CHANGE(4)
88 LOC_DEL(2)  98 OBJ_COUNT(7)  114 MAP_ANIM(6)  115 OBJ_DEL(3)
120 OBJ_ADD(5)  218 LOC_MERGE(14)
```

### SERVERPROT_SIZES (254, 257 entries, verbatim)

```
6,0,0,4,0,0,0,0,7,0,0,0,0,0,4,0,0,0,0,0,0,0,0,0,3,5,0,6,-2,0,4,0,0,0,0,0,0,
15,4,0,0,-2,0,0,0,0,0,0,0,0,0,0,0,0,0,6,0,0,1,0,-1,-2,0,-2,6,0,0,0,0,0,4,0,
0,-1,0,1,0,0,0,0,0,0,0,0,0,2,0,-2,2,0,0,3,0,0,1,4,0,0,7,0,0,0,0,0,0,0,0,0,
0,0,0,9,0,0,6,3,0,0,0,0,5,0,0,-2,0,0,0,6,0,0,0,0,0,0,0,0,6,0,1,0,0,2,0,2,
0,0,10,0,0,0,0,0,0,0,0,0,0,0,0,2,0,2,0,2,2,0,0,0,2,0,-2,0,0,2,0,0,0,0,0,0,
0,0,0,0,0,0,3,2,0,0,0,0,0,0,0,0,6,2,0,0,0,0,0,0,-1,0,0,0,0,4,0,4,0,3,0,0,
0,0,14,0,0,0,6,0,0,4,0,3,0,0,0,0,0,0,0,0,0,0,0,2,0,0,4,0,0,0,0,0,0,4,0,0,
0,0,0,1,0
```

---

## WS2 — Config deltas [L]

**LocType (11 deltas — the dominant config rework; LocType.java @2e62978):**
- `getModel` split into thin wrapper + new **`buildModel`** (:419, [L]):
  two branches — NEW `shapes==null` models-only path (key
  `((id<<6)+angle)+((frame+1)<<32)`, multi-model merge via new static
  `temp Model[4]` scratch, mirror +65536 / rotate180) and the classic
  shapes path. Common transform uses the reordered Model ctor with
  `AnimFrame.shareAlpha(frame)` (WS3). ⚠ **Compensated pair:** the
  `resize(resizez,resizex,resizey)` call-site reorder is compensated by a
  reordered `Model.resize` BODY — port both halves together or neither.
  Renames: `Model.tryGet→load`, `rotateY180→rotate180`, `rotateY90→rotate90`,
  `offset→translate`, `createLabelReferences→prepareAnim`,
  `applyFrame→animate`, `modelCacheStatic→mc1`, `modelCacheDynamic→mc2`.
- decode opcode **5** added (models-only array, `shapes=null`) (:255).
- decode opcode **75** `raiseobject` (+default `blockwalk?1:0` at end-of-decode;
  gates objRaise in buildModel, replacing the `blockwalk` gate) (:123).
- decode opcode **25 (`animHasAlpha`) REMOVED** (field deleted; WS3) (:505).
- end-of-stream finalization moved INTO `code==0`; `shapes` may stay null;
  NEW `active` rule `models!=null && (shapes==null || shapes[0]==10)` (:227).
- `ignoreCache` static + its getModel fast-path REMOVED (Go `Reset` global
  + its guards) (@176a85f:14).
- `checkModel` rewritten (shapes==null path, `Model.request→requestDownload`,
  -1 guards dropped) (:351); `checkModelAll` drops -1 guard (:373).
- `prefetch→prefetchModelAll` (:386): ⚠ the OnDemand.prefetch "arg swap" is
  compensated churn AND **the Go call order is already correct** — rename the
  method + drop the dead `-1` guard ONLY; do NOT touch the Go arg order.

**NpcType:** decode drops opcode 16 (`animHasAlpha`), adds **103 `turnspeed`**
(g2, field default 32) (:236, :98); `getTempModel` alpha from
`shareAlpha(f1)&shareAlpha(f2)` (:323; the `maskAnimate` arg-swap claim was
REFUTED — compensated).

**ObjType:** opcode 9 + bool `field1044` deleted; `field1045→field1034`
(opcode 10 semantics unchanged) (:300); `getWearModelNoCheck` branch
restructure (net-equivalent; use `translate`) (:586); the `resize` arg-order
claim was REFUTED (compensated pair, same as LocType).

**SpotAnimType:** `animHasAlpha` field + decode opcode 3 REMOVED (WS3 covers
the 4 caller sites).

**UnkType:** new 7th boolean field `field1109` (mc.l), default false (:42).

**IfType:** `getTempModel` face-alpha-share flag `true` →
`shareAlpha(arg1)&shareAlpha(arg3)` (:535; ctor positional reorder is
signature-only).

---

## WS3 — animHasAlpha → shareAlpha alpha system [M, cross-cutting]

The per-config `animHasAlpha` flag is deleted EVERYWHERE; alpha-sharing is
now derived per-frame via new **`AnimFrame.shareAlpha(id) == (id == -1)`**
(AnimFrame.java:117 @2e62978).

- Add `animframe.ShareAlpha(id int) bool` (one-liner).
- AnimFrame also gains `opaque []bool` (init all-true; `unpack` clears on
  base-type-5 transforms; `unload` doesn't clear it) — **DEAD-WRITE artifact,
  never read anywhere at `2e62978`** (verified by tree-wide grep). Per policy:
  intentionally NOT ported, mark with a comment.
- Rewire all 8 caller sites from `!xxx.AnimHasAlpha` to the shareAlpha form,
  with the reordered Model ctor `(shareAlpha, shareVertex=false,
  shareColour=true, src)` / `Model.set(flag, model)`:
  `clientnpc.go:41` (resolved frame id `seq.frames[spotanimFrame]` — NOT the
  index), `clientplayer.go:153` (+ `Model.set` site, flag =
  `shareAlpha(f1)&shareAlpha(f2)` replacing constant `true`),
  `clientproj.go:104` (full getTempModel restructure: frame id computed
  up-front, `animate` gated on `frame != -1`, `rotateX→rotateXAxis`),
  `mapspotanim.go:58`, plus NpcType/LocType/IfType decode+build sites (WS2).
- Delete `AnimHasAlpha` from loctype/npctype/spotanimtype (fields, resets,
  decode cases).

---

## WS4 — Varbits + clientscript VM + Stats [M]

- **`config/varbittype`** (new package; VarBitType.java @2e62978): statics
  `count`/`list`; fields `debugname`, `basevar`, `startbit`, `endbit`;
  `unpack(jag)` reads `varbit.dat` (g2 count, per-entry decode, trailing
  "varbit load mismatch" println); `decode`: op 0 return; op 1
  basevar=g2/startbit=g1/endbit=g1; op 10 debugname=gstr; else error println.
- **`Client.load`**: add `VarBitType.unpack(config)` after VarpType
  (Client.java:1800) and `startThread(mouseTracking, 10)` at tail (:1853;
  consumed by WS5 telemetry).
- **`getIfVar`** (was `executeClientScript`; Client.java:9783, [L]): now an
  operator state machine (op15 subtract / op16 divide-with-≠0-guard / op17
  multiply / default add); NEW opcodes: **14** varbit read
  (`varps[basevar] >> startbit & BITMASK[endbit-startbit]` — verify the Go
  BITMASK table location), **18/19** local-player world tile X/Z, **20**
  inline literal; ops 4/10 gain a members gate
  (`id < ObjType.count && (!members || membersWorld)`); op 9 skill loop is
  now table-driven via **Stats**.
- **`client/stats.go`** (new; Stats.java @2e62978): `COUNT=25`,
  `NAMES[25]` (verbatim incl. five `"-unused-"` slots), `ENABLED[25]`
  (false at 18 slayer, 19, 21–24).

---

## WS5 — New protocol features [XL, requires WS1]

- **gameLoop telemetry** (was `updateGame`; Client.java:2696, [XL]): new
  fields `sendCamera`, `focused`, `sendCameraDelay`, `mouseTrackedX/Y`,
  `mouseTrackedDelta`, `prevMousePressTime`; (1) EVENT_MOUSE_MOVE (232)
  packed-delta serialization under `synchronized(mouseTracking.lock)`
  (clamps y≤502/x≤764, `var=y*765+x`, sentinel 524287, 2/3/4-byte
  encodings); (2) EVENT_MOUSE_CLICK (234) `(timeDelta<<20|button<<19|y*765+x)`;
  (3) camera-key send (91) on 20-cycle gate; (4) focus change (8) p1(1/0).
  ⚠ `synchronized` → real mutex audit per the goroutine layout.
- **Anticheat C1–C7 wholesale rewrite** ([XL]; Client.java:2924,3318,4265,
  5407,5502,5953,6267): new counters (`field1294/1339/1354/1511/1587/1596/
  1285`), new thresholds/opcodes/payloads/host methods — C1 51@addProjectiles
  thr1174; C2 225@interactWithLoc thr1086 (randomized payload); C3 4 p1(50)
  thr112; C4 226 p1(232) thr192 @handleInputKey; C5 100 thr57 @otherOverlays
  under crossMode==2; C6 36 p1(62) thr122 @addPlayers; C7 NEW 182 thr62
  @gameLoop. Replace the 245.2 anticheat entirely.
- **Player options** (SET_PLAYER_OP 204): new `playerOptions [5]string` +
  `playerOptionsPushDown [5]bool` + login-time reset;
  `addPlayerOptions` rewritten array-driven (slot menuActions 639/499/27/
  387/185, attack pushdown 2000 by combat-level compare or pushdown flag)
  (Client.java:9297).
- **Friend-server state machine**: new `friendListStatus` int +
  FRIENDLIST_LOADED (255) handler; `updateInterfaceContent` gains status
  branches ("Loading friend list"/"Connecting to friendserver"/"Please
  wait...") and forces friend count 0 unless status==2 (:10281);
  `handleInterfaceAction` wraps clientCode 201/202 in `status==2` (:10513).
- **Transmog** (ClientPlayer): new `transmog *NpcType` field;
  `setAppearance` (was `read`) parses `appearance[0]==65535` → extra g2 NPC
  id + break (:88); `getTempModel2` (was `getAnimatedModel`) short-circuits
  to `transmog.getTempModel(frame, nil, -1)` (:244).
- **getNpcPosNewVis/Extended**: copy `turnspeed` (and on 0x20 transform,
  also re-copy `size`) from NpcType onto the entity (:8364, :8451).
- **handleInputKey preamble**: `field1339++ > 192 → pIsaac(226) p1(232)`
  (:4264); five opcode renumbers in-body (WS1 table).
- **`mouseTracked`** static (login reply 2) — pairs with the telemetry block.

---

## WS6 — Scene / render deltas [M]

- **ClientBuild** (was Go `world` pkg): `fullbright` static + all three gates
  in `finishBuild` REMOVED — hue/lightness random-walk, `shareLight`, and
  occluder pass now unconditional (ClientBuild.java:570-897). (The
  `buildModels→shareLight` literal swap claim was REFUTED — compensated.)
- **World** (was Go `world3d`): `pushDown` also decrements carried Sprites'
  `level` (typecode>>29&3==2 ∧ min-tile match) (:255); `setDecorOffset` drops
  the `-23232` sentinel 5th param, ALWAYS offsets Z (:563); `fill` adds an
  equal-distance euclidean tiebreak (`(x-cx)²+(z-cz)²`, farther wins) (:1468).
- **CollisionMap.addLoc**: trailing `arg6 bool` + its early-return DROPPED
  (:201) — remove from the Go signature and call sites.
- **ClientEntity**: `height` default 0→**200**; new `turnspeed = 32` (:79,:85).
- **addPlayers split** (was `pushPlayers`; Client.java:5400): boolean param —
  true=local only, false=others; caller order now
  `addPlayers(T) addNpcs(T) addPlayers(F) addNpcs(F) addProjectiles
  addMapAnim` — changes z-order of always-on-top entities.
- **getSpecialArea** (was `updateWorldLocation`; :6057): gutted ~46→12 lines —
  wilderness level, bank/arena region scan, `overrideChat` all REMOVED;
  remaining region logic writes `worldLocationState` (1 in two regions else 0).
- **otherOverlays** (was `draw3DEntityElements`; :5945): `field1504`
  hitsplat-flash overlay + wilderness/arena HUD REMOVED (unconditional
  `imageHeadicons[1]` at 472,296); folds in anticheat C5.
- **handleTabInput** (:3886 @176a85f): trailing CYCLELOGIC1 block removed;
  ⚠ else-if → independent-if is **NOT churn** here — overlapping tab rects
  make it last-match-wins (1477 boundary pixels differ); port as independent
  ifs.

---

## WS7 — Stragglers + rename adoption [M]

- **GameShell**: hook `update()→loop()` (+ Client override + call site)
  (GameShell.java:538); `shutdown` gains `frame != nil` guard around
  sleep+exit ONLY (state=-2 + unload stay unconditional; the boolean param is
  a dead deob arg — don't port it) (:229). Go `gameshell.go:16` currently
  exits unconditionally.
- **JString.fromBase37**: static 12-char builder → per-call local buffer
  (removes the Go package-level `Builder` global — also fixes a latent race)
  (JString.java:40). (`censor` StringBuffer claim REFUTED — equivalent.)
- **Packet**: `gjstr→gstr`, `gjstrraw→gstrbyte` (renames only);
  `base64enctab` char[64] is a never-read deob artifact — intentionally NOT
  ported (add marker) (Packet.java:36).
- **WordFilter**: ALLOWLIST + `"faq"` (8th entry) (WordFilter.java:26).
- **Menu-action id renumber** (internal dispatch constants, must stay
  pairwise-consistent): npc use-with 900→829, ops →242/209/309/852/793,
  examine 1607→1714, spell 265→240; player use-with 367→275, spell 651→131,
  walk-override 660→718; loc/obj viewport set (handleViewportOptions :3613,
  [L]) — loc ops →625/721/743/357/1071, examine 1175→1381, spell 55→899,
  use-with 450→810; obj use-with 217→111, ops →139/778/617/224/662, examine
  1102→1152, spell 965→370; obj-drag whitelist replaced (12 entries:
  {582,113,555,331,354,694,962,795,681,100,102,1328}) (:3789); social/chat
  sets (Add friend 406→605 ×3 sites incl. `isAddFriendOption`, Add ignore
  436→47, Report 34→524, trade/duel 903→507/363→957, split-chat
  2034→2524/2436→2047/2406→2605, Remove 557→513, Message 679→902, room-Remove
  556→884); Cancel sentinel 1252→1106 (inert but port for fidelity).
- **useMenuOption**: obj-examine count source now
  `IfType.list[var4].linkObjCount[slot]` with nil guard (was raw param)
  (:8763).
- **handleInterfaceAction**: pIsaac 150→13 (IF_PLAYERDESIGN), 205→203
  (REPORT_ABUSE) (:10606,:10623).
- **Client method renames** (adopt per hybrid policy as methods are touched;
  body-shape-verified pairings — full lists in appendix): `readPacket→tcpIn`,
  `readZonePacket→zonePacket`, `updateGame→gameLoop`, `buildScene→mapBuild`,
  `drawTitle→titleScreenDraw(bool)`, `addMessage→addChat`,
  `removeFriend/Ignore→delFriend/delIgnore`, `closeInterfaces→closeModal`,
  `executeClientScript→getIfVar`, `executeInterfaceScript→getIfActive`,
  `handleInterfaceInput→handleComponentInput`, `updateOnDemand→onDemandLoop`,
  `runFlames→renderFlames`, `pushNpcs/Players/Projectiles/Spotanims→
  addNpcs/addPlayers/addProjectiles/addMapAnim`,
  `updateWorldLocation→getSpecialArea`, `draw3DEntityElements→otherOverlays`,
  `getModel→getTempModel` family, `isVisible→isReady`, `move→teleport`,
  `step→moveCode`, `clearRoute→abortRoute`, `getFrameLength→getDuration`,
  `frameCount→numFrames`, `types→list` (config registries),
  `Pix3D.init2D/init3D→init/initWH`, `Pix2D/Pix32 bind→setPixels` +
  draw-family renames, `gjstr→gstr`, `allowCharacter→isCharAllowed`,
  `filterdomain→filterDomain`, `selectedTab→sideTab`,
  `idleTimeout→pendingLogout`, `idleNetCycles→packetCycle`, layer-id field
  renames (`viewportInterfaceId→mainLayerId`, `chatInterfaceId→chatLayerId`,
  `sidebarInterfaceId→sideLayerId`, `stickyChatInterfaceId→tutLayerId`,
  `viewportOverlayInterfaceId→mainOverlayLayerId`), `vislevel→combatLevel`,
  `scene→world` / `World3D→World` / `Component→IfType` (landed in kickoff).

---

## Refuted claims (verified NON-deltas — do NOT "fix")

1. NpcType `maskAnimate` arg swap — compensated param-mapping churn.
2. ObjType/LocType `resize` arg reorder — compensated by reordered
   `Model.resize` body (port both halves or neither).
3. JString `censor` StringBuffer — output-identical.
4. ClientBuild `shareLight(64,-10)` literal swap — compensated.
5. `updateMousePicking` X/Y swap — compensated (param-name permutation).

## Standing decisions carried forward (do not re-fix)

tone.go int-not-int32; LruCache dup-key caller constraint; loctype op[]
35–38 faithful overflow; Model ctor compensated pair; pre-varp FULL volume.

---

## Appendix — Client.java method pairing (85 by name, 60/59 body-paired)

Base-only (renamed away or removed): addLoc addMessage appendLoc
applyCutscene buildScene clearCache clearLocChanges closeInterfaces
createMinimap draw2DEntityElements draw3DEntityElements drawGame
drawInterface drawMinimapArrow drawMinimapLoc drawScene drawTileHint
drawTitle executeClientScript executeInterfaceScript getHeightmapY
getTopLevel getTopLevelCutscene handleInterfaceInput handleScrollInput mix
orbitCamera projectFromEntity projectFromGround pushNpcs pushPlayers
pushProjectiles pushSpotanims readPacket readZonePacket removeFriend
removeIgnore runFlames sortObjStacks startForceMovement storeLoc update
updateAudio updateEntity updateEntityChats updateFacingDirection
updateForceMovement updateGame updateInterfaceAnimation updateLocChanges
updateMovement updateNpcs updateOnDemand updateOrbitCamera updatePlayers
updateSceneState updateSequences updateTextures updateTitle
updateWorldLocation

Target-only (renamed to or new): addChat addMapAnim addNpcs addPlayers
addProjectiles animateLayer camFollow checkMinimap cinemaCamera clearCaches
closeModal coordArrow delFriend delIgnore doScrollbar drawDetail drawLayer
drawMinimapHint entityAnim entityFace entityOverlays exactMove1 exactMove2
followCamera gameDraw gameDrawMain gameLoop getAvH getIfActive getIfVar
getOverlayPos getSpecialArea handleComponentInput locChangeCreate
locChangeDoQueue locChangePostBuildCorrect locChangeSetOld locChangeUnchecked
loop mapBuild minimapBuildBuffer moveEntity moveNpcs movePlayers onDemandLoop
otherOverlays renderFlames roofCheck roofCheck2 routeMove showObject
soundsDoQueue tcpIn textureRunAnims timeoutChat titleFlamesMerge
titleScreenDraw titleScreenLoop zonePacket
