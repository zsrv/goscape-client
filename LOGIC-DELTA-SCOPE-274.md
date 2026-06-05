# rev-274 Logic-Delta Scope (254 → 274)

Scope of the **game-logic delta** to apply on the `rev-274` branch (cut from
the complete rev-254 Go port @ `a94d3fc`). The mechanical rename pass
(`RENAME-MAP-274.md`, P2: nine commits `6446fdc..fa4f6cc` — wordfilter/JagFX/
PointNormal/SpotType/VarbitType/Linkable2/Skill/protocol-move/clientbuild-move)
is **already landed** — this document covers everything else.

> **Provenance.** Machine-generated 2026-06-05 by workflow `wf_a1fe73b6-796`
> (two passes: 155 + 159 agents; the first verify wave was rate-limited and
> re-ran on resume). Method per `PORT-DESIGN-274.md` §3: 50 compare units
> (32 class units, Client.java preamble + 11 method-paired chunks from a
> 148-pair descriptor-corroborated pairing table, 3 opcode-table units with
> embedded mechanical completeness checklists, constants/Filter/InputTracking
> units) over Java pins `2e62978` (254) vs `32f3062` (274), reads via
> `git show` only; **every claimed non-cosmetic delta adversarially
> re-verified** against both pins. Result: 856 deltas → 461 verified →
> **315 confirmed / 146 refuted** (refutations = compensated deob churn, see
> §Churn policy). 14 trivial units (Protocol/Skill/signlink/Decor/etc.) were
> adjudicated inline in the main session by full normalized diff.
> Cross-run conflict resolutions: (1) inbound `IF_SETNPCHEAD` is `3→142`
> (a run-1 row claimed 3→129; refuted, then re-derived mechanically AND
> independently by the run-2 inbound agent); (2) `routeMove`'s walkanim_l/r
> swap is **compensated** by the NpcType obf-slot swap (net wire-neutral —
> a run-2 verifier confirmed it in isolation; the NpcType unit proves the
> compensation); (3) a run-1 `client-chunk-4` result was degenerate and was
> fully re-derived in run 2. Raw output + digests: `audit-274/p3/`
> (untracked scratch — run1-full.json / run2-full.json / digest-run*.md).
> Treat claims as strong hints, not gospel — verify against `@32f3062`
> while porting; cite `// Java: ... @32f3062`.

## Executive summary

The 254→274 gap (20 revisions) is dominated by a **fresh-deob churn layer**
(annotation/`final`/`this.`-drop, local renames, and ~100 *compensated*
parameter-order shuffles that the adversarial pass killed), over a real delta
that is mid-sized but touches every subsystem:

- **Full wire-opcode renumber** on all three surfaces — 76 outbound messages
  (exactly ONE value unchanged: ANTICHEAT_CYCLELOGIC5 = 100), ~60 inbound,
  10 zone ops — plus a new login handshake (`p1(255)+p2(274)`), a rewritten
  checksum fetch with self-validation, and a **new JAGGRAB socket transport**
  fallback (port 43595).
- **InputTracking is deleted wholesale** (class + 15 call sites + 3 opcodes).
- A genuinely new **sound IIR filter** (`Filter.java`, 123 lines) wired into
  `Tone`, plus a 3-site `midiFading` polarity flip.
- A **PixFont rework**: 256-glyph direct char indexing replaces the 94-glyph
  `CHAR_LOOKUP` scheme; new `*_full` font archives and a `wide` ctor flag.
- A new **minimapState machine** (new inbound op 194, blackout draw mode,
  click gating).
- Scattered real logic: hillskew decor rotation in the scene builder, a new
  player-options loop, an extra NPC jump bit on the wire, skillLevel in
  player appearance, OnDemand failure accounting, WordFilter changes,
  focus-loss key clearing, palette-collapse guard in Pix3D.
- A heavy **rename layer** with EIGHT field/method-level name-reuse traps
  (class-level pairing was clean; the traps live one level down — §Traps).

Suggested P4 workstream order: **WS1 → WS2 → WS5 → WS7 → WS6 → WS3 → WS4 →
WS8** (WS1 is the critical path — nothing connects without it; WS2 is small
and unblocks GameShell; the rest are largely independent).

## WS1 — Wire protocol: opcode renumber + login + transport [XL, critical path]

**Outbound (client→server).** All 76 messages renumbered except
ANTICHEAT_CYCLELOGIC5 (100). Payload bodies are byte-identical at both pins
for every message (verified per-site; pairing anchored on the
revision-invariant menu-action selector constants `var5 == NNN` and 254 deob
comments). `pIsaac` → `p1Enc` is the Packet writer rename (same
ISAAC-encrypted byte). Sites: `@NNNN` = Client.java line at that pin.
**Beware intra/cross-rev value reuse** (e.g. 254 op9=FRIENDLIST_ADD but 274
op9=IF_BUTTON) — never port by value, always by message.

EVENT_TRACKING (254 op 142, ×2 sites) is **removed** with InputTracking (WS2).
The one variable-opcode helper (254 `pIsaac(arg4)` ≡ 274 `p1Enc(arg3)`,
the interactWithLoc-style op helpers) survives unchanged.

| message | 254 | 274 | payload (274) | notes |
|---|---|---|---|---|
| ANTICHEAT_CYCLELOGIC3 | 4 | 52 | p1(50) | 254 @3321, 274 @6351. cyclelogic3>112. Opcode 4->52. |
| MOVE_GAMECLICK | 6 | 207 | identical | 254 @6480, 274 @2961. Walk path (arg==0). Opcode 6->207. |
| EVENT_APPLET_FOCUS | 8 | 73 | identical | 254 @2813(gain)/@2819(lost), 274 @9451(gain)/@9456(lost). Multiplicity 2 each. Opcode 8->73. |
| FRIENDLIST_ADD | 9 | 13 | p8(userhash) | 254 @10979 (addFriend), 274 @7440 (addFriend). Opcode 9->13. (254 op9 reassigned to IF_BUTTON in 274; FRIENDLI |
| IF_PLAYERDESIGN | 13 | 125 | identical | 254 @10606, 274 @8819. Character-design confirm (var3==326). Opcode 13->125. |
| OPPLAYER2 | 17 | 166 | identical | 254 @9081, 274 @4426. menu var5==499. Opcode 17->166. |
| OPPLAYER3 | 18 | 196 | identical | 254 @9063, 274 @4423. menu var5==27. Opcode 18->196. |
| OPLOCT | 26 | 213 | identical (caller appends p2(targetComId)) | VARIABLE SITE. menu var5==899. 254 @8745 arg=26 +@8747 p2(activeSpellId); 274 @4752 arg=213 +@4753 p2(targetCo |
| ANTICHEAT_OPLOGIC1 | 28 | 219 | p4(0) | 254 @8708, 274 @4315. Before OPLOC2 (var5==721) when oplogic1>=139. Opcode 28->219. |
| OPLOC1 | 33 | 215 | identical | VARIABLE SITE. menu var5==625. 254 @9100 arg=33; 274 @4756 arg=215. Opcode 33->215. |
| ANTICHEAT_CYCLELOGIC6 | 36 | 188 | p1(62) | 254 @5407, 274 @7581. addPlayers, cyclelogic6>122 at flag tile. Opcode 36->188. (274 op36=OPPLAYERU differs.) |
| OPOBJ4 | 47 | 62 | identical | 254 @8866, 274 @4602. menu var5==224. Opcode 47->62. (254 op62=INV_BUTTON5 vs 274 op62=OPOBJ4 — cross-rev valu |
| ANTICHEAT_CYCLELOGIC1 | 51 | 12 | identical | 254 @5505, 274 @4023. cyclelogic1>1174. Identical obfuscation body. Opcode 51->12. (274 op51=CLOSE_MODAL diffe |
| ANTICHEAT_OPLOGIC3 | 56 | 41 | p4(0) | 254 @8836, 274 @4581. Inside OPOBJ5 (var5==662) when oplogic3>=118. Opcode 56->41. |
| CLOSE_MODAL | 58 | 51 | (none) | 254 @4050, 274 @3354. closeModal() entry. Empty. Opcode 58->51. |
| INV_BUTTON3 | 59 | 239 | identical | 254 @8976, 274 @4465. menu var5==555. Opcode 59->239. (274 op239=INV_BUTTON3 vs 254 op239=NO_TIMEOUT.) |
| INV_BUTTON5 | 62 | 46 | identical | 254 @8980, 274 @4468. menu var5==354. Opcode 62->46. |
| OPOBJ2 | 67 | 169 | identical | 254 @8844, 274 @4608. menu var5==778. Opcode 67->169. |
| OPPLAYERT | 68 | 240 | identical | 254 @9111, 274 @4662. menu var5==131. Cast-on-player. Opcode 68->240. |
| OPNPC3 | 69 | 223 | identical | 254 @8724, 274 @4440. menu var5==309. Opcode 69->223. |
| INV_BUTTON2 | 70 | 82 | identical | 254 @8972, 274 @4462. menu var5==113. Opcode 70->82. |
| OPPLAYER4 | 72 | 98 | identical | 254 x2 @8891 (var5==507, p2(playerIds[i])) & @9059 (var5==387, p2(var6)), 274 x2 @4634 & @4409. Multiplicity 2 |
| OPHELD5 | 74 | 42 | identical | 254 @8625, 274 @4699. menu var5==100. Opcode 74->42. |
| ANTICHEAT_OPLOGIC2 | 77 | 201 | p2(37954) | 254 @9183, 274 @4300. Before OPLOC3 (var5==743) when oplogic2>=124. Opcode 77->201. |
| OPHELD3 | 80 | 123 | identical | 254 @8629, 274 @4708. menu var5==795. Opcode 80->123. (op80 also used by ANTICHEAT_OPLOGIC4 in 274 — intra-rev |
| MESSAGE_PUBLIC | 83 | 253 | identical | 254 @4446, 274 @10370. Var-length psize1. Opcode 83->253. |
| FRIENDLIST_DEL | 84 | 106 | p8(userhash) | 254 @11000 (delFriend), 274 @4950 (delFriend, verified). Opcode 84->106. |
| CLIENT_CHEAT | 86 | 224 | identical | 254 @4383, 274 @10308. ::command. Opcode 86->224. (274 op86=MOVE_MINIMAPCLICK is a different message.) |
| OPLOC4 | 87 | 157 | identical | VARIABLE SITE. menu var5==357. 254 @8801 arg=87; 274 @4725 arg=157. Opcode 87->157. |
| EVENT_CAMERA_POSITION | 91 | 53 | identical | 254 @2806, 274 @9445. Opcode 91->53. |
| OPOBJ5 | 97 | 117 | identical | 254 @8840, 274 @4584. menu var5==662. Opcode 97->117. |
| OPLOC3 | 98 | 187 | identical | VARIABLE SITE. menu var5==743. 254 @9187 arg=98; 274 @4297 arg=187. Opcode 98->187. |
| ANTICHEAT_CYCLELOGIC5 | 100 | 100 | (none) | 254 @5956, 274 @1553. otherOverlays, crossMode==2, field>57. Empty. Opcode UNCHANGED 100->100 — the ONLY same- |
| OPHELDT | 102 | 135 | identical | 254 @8588, 274 @4339. menu var5==563. Use-item-on-spell. Opcode 102->135. |
| OPPLAYERU | 113 | 36 | identical | 254 @9172, 274 @4290. menu var5==275. Use-item-on-player. Opcode 113->36. |
| OPNPC5 | 118 | 189 | identical | 254 @8736, 274 @4449. menu var5==793. Opcode 118->189. (274 op189=IGNORELIST_ADD differs.) |
| OPNPCU | 119 | 150 | identical | 254 @9023, 274 @4260. menu var5==829. Use-item-on-npc. Opcode 119->150. |
| ANTICHEAT_OPLOGIC4 | 121 | 80 | p1(131) | 254 x2 @8897 & @9069, 274 x2 @4639 & @4417. Inside OPPLAYER1 (var5==957 and var5==639) when oplogic4>=52. Mult |
| OPNPC4 | 122 | 147 | identical | 254 @8728, 274 @4446. menu var5==852. Opcode 122->147. (274 op147=OPLOC5 differs.) |
| MOVE_OPCLICK | 127 | 138 | identical | 254 @6490, 274 @2969. arg==2. Opcode 127->138. |
| CHAT_SETMODE | 129 | 154 | identical | 254 x5: @4009/@4019/@4029 (chatModeLoop 3 buttons) + @4320 (auto after private msg) + @4470 (auto after public |
| ANTICHEAT_OPLOGIC6 | 131 | 250 | p2(6118) | 254 @8964, 274 @4475. Inside INV_BUTTON1 (var5==582) when oplogic6>=133. Opcode 131->250. |
| MAP_BUILD_COMPLETE | 134 | 214 | (none) | 254 @3121, 274 @3283. After mapBuild(), method returns 0. Empty. Opcode 134->214. |
| OPOBJ1 | 141 | 247 | identical | 254 @8856, 274 @4594. menu var5==139. Opcode 141->247. |
| OPNPC1 | 143 | 236 | identical | 254 @8740, 274 @4443. menu var5==242. Opcode 143->236. |
| IDLE_TIMER | 144 | 209 | (none) | 254 @2968, 274 @9593. idleTimer>4500. Empty. Opcode 144->209. |
| RESUME_PAUSEBUTTON | 146 | 72 | p2(comId) | 254 @9039, 274 @4495. menu var5==997, !resumedPauseButton. Opcode 146->72. (274 op72=RESUME_PAUSEBUTTON vs 254 |
| OPLOC5 | 147 | 127 | identical | VARIABLE SITE. menu var5==1071. 254 @9128 arg=147; 274 @4615 arg=127. Opcode 147->127. |
| INV_BUTTON4 | 160 | 179 | identical | 254 @8956, 274 @4459. menu var5==331. Opcode 160->179. |
| RESUME_P_COUNTDIALOG | 161 | 102 | p4(value) | 254 @4352, 274 @10278. Numeric chatback reply. Opcode 161->102. (274 op102 here differs from 254 op102=OPHELDT |
| ANTICHEAT_OPLOGIC9 | 162 | 24 | p3(13018169) | 254 @8609, 274 @4693. Inside OPHELD4 branch (var5==681) when oplogic9>=116. Opcode 162->24. |
| OPHELD4 | 163 | 216 | identical | 254 @8613, 274 @4696. menu var5==681. Opcode 163->216. |
| INV_BUTTOND (drag) | 176 | 93 | identical | 254 @2908, 274 @9535. Inventory drag-and-drop. Opcode 176->93. |
| OPOBJ3 | 178 | 108 | identical | 254 @8830, 274 @4605. menu var5==617. Opcode 178->108. |
| INV_BUTTON1 | 181 | 74 | identical | 254 @8968, 274 @4478. menu var5==582. Opcode 181->74. (274 op181=OPNPCT differs.) |
| ANTICHEAT_CYCLELOGIC7 | 182 | 89 | (none) | 254 @2927, 274 @9553. cyclelogic7>62. Empty. Opcode 182->89. |
| ANTICHEAT_OPLOGIC7 | 187 | 25 | p4(0) | 254 @8852, 274 @4591. Inside OPOBJ1 (var5==139) when oplogic7>=123. Opcode 187->25. (274 op187=OPLOC3 differs. |
| IGNORELIST_ADD | 189 | 255 | p8(userhash) | 254 @11032 (addIgnore), 274 @7104 (addIgnore). Opcode 189->255. (254 op189=IGNORELIST_ADD vs 274 op189=OPNPC5. |
| OPPLAYER1 | 192 | 109 | identical | 254 x2 @8901 (var5==957) & @9073 (var5==639), 274 x2 @4642 & @4420. Multiplicity 2 each. Opcode 192->109. |
| IGNORELIST_DEL | 193 | 101 | p8(userhash) | 254 @11049 (delIgnore), 274 @9679 (delIgnore). Opcode 193->101. |
| OPNPC2 | 195 | 233 | identical | 254 @8732, 274 @4452. menu var5==209. Opcode 195->233. (274 op233=ANTICHEAT_OPLOGIC5 differs.) |
| OPHELDU | 200 | 136 | identical | 254 @8647, 274 @4668. menu var5==398. Use-item-on-item. Opcode 200->136. |
| TUTORIAL_CLICKSIDE | 201 | 94 | p1(sideTab) | 254 @5177, 274 @10968. Flashing tutorial tab clicked. Opcode 201->94. (274 op201=ANTICHEAT_OPLOGIC2 differs.) |
| OPOBJT | 202 | 91 | identical | 254 @8813, 274 @4746. menu var5==370. Cast-on-ground-obj. Opcode 202->91. (274 op91=OPOBJT vs 254 op91=EVENT_C |
| REPORT_ABUSE | 203 | 137 | identical | 254 @10623, 274 @8835. var3 in 601..612. Opcode 203->137. |
| ANTICHEAT_OPLOGIC8 | 206 | 0 | p1(19) | 254 @8862, 274 @4599. Inside OPOBJ4 (var5==224) when oplogic8>=75. Opcode 206->0. |
| OPLOC2 | 213 | 103 | identical | VARIABLE SITE. menu var5==721. 254 caller @8712 arg=213; 274 caller @4303 arg=103. Opcode 213->103. Preceded b |
| MESSAGE_PRIVATE | 214 | 139 | identical | 254 @4307, 274 @10235. Var-length psize1. p8 recipient + packed text. Opcode 214->139. |
| MOVE_MINIMAPCLICK | 220 | 86 | identical | 254 @6485 (tail @3908-3918), 274 @2965 (tail @10178-10188). arg==1. Opcode 220->86. |
| ANTICHEAT_CYCLELOGIC2 | 225 | 149 | identical | 254 @6270 (interactWithLoc, field1285>1086), 274 @3834 (interactWithLoc, cyclelogic2>1086). Identical body. Op |
| ANTICHEAT_CYCLELOGIC4 | 226 | 230 | p1(232) | 254 @4268, 274 @10197. handleInputKey, cyclelogic4>192. Opcode 226->230. |
| OPHELD2 | 228 | 2 | identical | 254 @8617, 274 @4705. menu var5==962. Opcode 228->2. |
| OPPLAYER5 | 230 | 174 | identical | 254 @9077, 274 @4412. menu var5==185. Opcode 230->174. (254 op230=OPPLAYER5 vs 274 op230=ANTICHEAT_CYCLELOGIC4 |
| OPNPCT | 231 | 181 | identical | 254 @9158, 274 @4519. menu var5==240. Cast-on-npc. Opcode 231->181. |
| EVENT_MOUSE_MOVE | 232 | 222 | identical | 254 @2711, 274 @9352. Var-length psize1 wraps a loop of p2(small delta)/p3/p4(large) per sample. Only opcode 2 |
| ANTICHEAT_OPLOGIC5 | 233 | 235 | p1(154) | 254 x2 @8887 & @9055, 274 x2 @4631 & @4406. Inside OPPLAYER4 (var5==507 and var5==387) when oplogic5>=66. Mult |
| EVENT_MOUSE_CLICK | 234 | 20 | identical | 254 @2793, 274 @9433. Single packed p4. Opcode 234->20. |
| NO_TIMEOUT | 239 | 120 | (none) | 254 x5: @3028 (main-loop noTimeoutCycle>50) + @3154/@3172/@3182/@3186 (4 in mapBuild). 274 x5: @9652 (main loo |
| OPLOCU | 240 | 60 | identical | VARIABLE SITE. menu var5==810. 254 @8913 arg=240 +@8915-8917; 274 @4367 arg=60 +@4368-4370. Opcode 240->60. (2 |
| OPHELD1 | 243 | 185 | identical | 254 @8621, 274 @4702. menu var5==694. Opcode 243->185. (274 op185=OPPLAYER5 differs.) |
| IF_BUTTON | 244 | 9 | p2(comId) | 254 x3: @8673 (var5==231), @8751 (var5==225), @9088 (var5==435). 274 x3: @4565 (231), @4777 (225), @4357 (435) |
| OPOBJU | 245 | 39 | identical | 254 @9006, 274 @4767. menu var5==111. Use-item-on-ground-obj. Opcode 245->39. |
| ONLY-254: EVENT_TRACKING op142 (x2): p2(packet.pos)+pdata(tracking bytes). Sites @2828 (main loop, after InputTracking.flush()) and @6971 (inside server pkt-typ | | | | |

**Inbound (server→client).** 60 paired handlers (incl. the zone-umbrella
dispatch), 2 removed (FINISH_TRACKING 29, ENABLE_TRACKING 251 — WS2), 1 new:

- **SET_MINIMAP_STATE (274 op 194, size 1, NEW)** — `minimapState = in.g1()`
  (WS5). No 254 counterpart anywhere.
- `IF_SETNPCHEAD` is **3→142** and `IF_SETMODEL` is **211→129** (adjudicated
  twice: a same-shape pair distinguished by `model1Type=2` vs `=1`).
- Handler bodies are logic-identical modulo the rename layer EXCEPT
  MESSAGE_GAME (`worldLocationState`→`chatDisabled` rename only) and the
  handlers feeding new fields noted in their workstreams.
- `SERVERPROT_LENGTH` → `SERVERPROT_SIZE`: replace the Go `SERVERPROT_SIZES`
  table **wholesale** from `@32f3062 jagex2/io/Protocol.java` (values fully
  renumbered; never patch incrementally). `CLIENTPROT_SCRAMBLED` (renamed
  back from 254's `CLIENTPROT_LOOKUP`, values renumbered) remains
  declaration-only dead — **existing Go non-port stands**, carry the comment.

| message | 254 | 274 | payload (274) | notes |
|---|---|---|---|---|
| CAM_LOOKAT | 0 | 233 | g1, g1, g2, g1, g1 | cutscene/cinemaCam=true; if accel>=100 compute cameraPitch/Yaw via atan2*325.949 with clamp 128..383. 254@6979 |
| IF_SETNPCHEAD | 3 | 142 | g2, g2 | modelType=2; modelId=npc. 254@7106 (modelType/modelId), 274@7997 (model1Type/model1Id). size both=4. |
| P_COUNTDIALOG | 5 | 210 | none | showSocialInput/socialInputOpen=false; chatbackInputOpen/dialogInputOpen=true; chatbackInput/dialogInput=""; r |
| IF_SETSCROLLPOS | 14 | 54 | g2, g2 | For IfType type==0, clamp pos to [0, scrollSize-height] then set scrollPosition/scrollPos. 254@7089 (scrollSiz |
| LOGOUT | 21 | 88 | none | calls logout(); returns FALSE (not true). 254@7175, 274@7927. size both=0. |
| CHAT_FILTER_SETTINGS | 24 | 114 | g1, g1, g1 | chatPublicMode/chatPrivateMode/chatTradeMode; redrawPrivacySettings+redrawChatback. 254@7379, 274@8036. Identi |
| SYNTH_SOUND | 25 | 34 | g2, g1, g2 | If waveEnabled && !lowMem && waveCount<50: append to waveIds/waveLoops/waveDelay (delay + Wave.delay[id] / Jag |
| IF_SETPOSITION | 27 | 77 | g2, g2b, g2b | IfType.list[com].x=g2b, .y=g2b. 254@6671, 274@8638. Identical. size both=6. |
| UPDATE_INV_FULL | 28 | 106 | g2, g1, count x {g2, g1/g4}, pad | redrawSidebar=true first; count items, g1 num with 255->g4 escape; trailing slots zeroed. 254@6698 (linkObjCou |
| IF_SETCOLOUR | 38 | 183 | g2, g2 | Decodes 5-5-5 (r=v>>10&0x1F,g=v>>5&0x1F,b=v&0x1F) -> colour=(r<<19)+(g<<11)+(b<<3). 254@6793, 274@8659. Identi |
| IF_SETTEXT | 41 | 44 | g2, gjstr | IfType.list[com].text=text; if layerId matches active tab interface redrawSidebar. 254@7368 (tabInterfaceId[si |
| CAM_MOVETO | 55 | 200 | g1, g1, g2, g1, g1 | cutscene/cinemaCam=true; if accel>=100 snap cameraX/Z/Y via getAvH. 254@6943 (cutsceneSrc*/getAvH(z,level,x)), |
| TUT_FLASH | 58 | 90 | g1 | flashingTab/tutFlashingTab=g1; if ==sideTab toggle sideTab 3<->1 and redrawSidebar. 254@7029, 274@8232. size b |
| MESSAGE_PRIVATE | 60 | 235 | g8, g4, g1, WordPack(psize-13) | Dedup via messageIds/privateMessageIds ring (mod 100); ignore-list check when status<=1; WordPack.unpack + Wor |
| UPDATE_ZONE_PARTIAL_ENCLOSED | 61 | 195 | g1, g1, while{g1; zonePacket} | Sets baseX/zoneUpdateX then loops embedded zone packets delegating to zonePacket (covered by zonePacket unit). |
| UPDATE_IGNORELIST | 63 | 3 | psize/8 x g8 | ignoreCount=psize/8; loop g8 into ignoreName37[] (254@6662) / ignoreUserhash[] (274@7989). size both=-2 (g2 le |
| HINT_ARROW | 64 | 156 | same | Identical branch tree mapping type 2-6 to hintOffsetX/Z then collapsing to type=2. 254@6804, 274@7944. size bo |
| MESSAGE_GAME | 73 | 161 | gjstr | Endswith :tradereq:/:duelreq: -> ignore-list check + addChat type 4/8; else addChat type 0. 254@7333 (worldLoc |
| SET_MULTIWAY | 75 | 207 | g1 | inMultizone=g1. 254@6937, 274@7790. size both=1. |
| IF_OPENOVERLAY | 85 | 240 | g2b | if com>=0 resetInterfaceAnimation/ifAnimReset; mainOverlayLayerId/mainOverlayId=com. 254@7279, 274@8064. size  |
| PLAYER_INFO | 87 | 167 | getPlayerPos(psize) | getPlayerPos then awaitingSync/awaitingPlayerInfo=false. 254@7289 getPlayerPos(in,psize), 274@7883 getPlayerPo |
| IF_SETTAB | 91 | 215 | g2, g1 | 65535->-1; tabInterfaceId/sideOverlayId[tab]=com; redrawSidebar+icons. 254@7162 (g2 com then g1 tab; tabInterf |
| UPDATE_RUNENERGY | 94 | 83 | g1 | if sideTab==12 redrawSidebar; runenergy=g1. 254@7020, 274@8261. size both=1. |
| IF_SETANIM | 95 | 134 | g2, g2b | IfType.modelAnim=anim; if anim==-1 reset seqFrame/seqCycle (254@7007) / animFrame/animCycle (274@7932). Field  |
| ZONE_UPDATE_UMBRELLA (delegates to zon | 98 | 95 | 274 ops 95/176/219/85/107/52/81/48/173/138 -> zonePacket(ptype,in) | Single multi-opcode dispatch branch forwarding loc/obj/zone-tile updates to zonePacket. delegates to zonePacke |
| UNSET_MAP_FLAG | 108 | 115 | none | flagSceneTileX/minimapFlagX = 0. 254@7389, 274@7817. Field rename only. size both=0. |
| UPDATE_FRIENDLIST | 111 | 247 | g8, g1 | Update or append friend (cap 200); logged in/out chat (type 5); bubble-sort friends by world vs nodeId. 254@72 |
| NPC_INFO | 123 | 197 | getNpcPos(psize) | Delegates to getNpcPos bit-stream reader. 254@7135 getNpcPos(in,psize), 274@8654 getNpcPos(psize,in) (arg orde |
| UPDATE_STAT | 136 | 105 | g1, g4, g1 | redrawSidebar; statXP/statEffectiveLevel set; statBaseLevel recomputed via levelExperience[0..97] loop (baseLe |
| IF_SETTAB_ACTIVE | 138 | 241 | g1 | sideTab=g1; redrawSidebar+redrawSideicons. 254@6911, 274@7795. size both=1. |
| RESET_CLIENT_VARCACHE | 140 | 190 | none | For all varps: if differs from varCache/varServ, copy + updateVarp/clientVar + redrawSidebar. 254@7150, 274@82 |
| IF_OPENCHAT | 141 | 166 | g2 (component) | Opens chat layer. 254@6560 sets chatLayerId/redrawChatback, clears sideLayerId, mainLayerId=-1, pressedContinu |
| UPDATE_REBOOT_TIMER | 143 | 89 | g2 (x30) | systemUpdateTimer/rebootTimer = g2*30. 254@7083, 274@8245. Field rename only. size both=2. |
| LAST_LOGIN_INFO | 146 | 91 | g4, g2, g1, g2, g1 | Same 10-byte fixed read + dnslookup + closeModal + clientCode 650/655 selection scanning IfType.list for layer |
| UPDATE_ZONE_FULL_FOLLOWS | 159 | 153 | g1, g1 | Zone reset: nulls objStacks/groundObj in 8x8 + showObject, and zeroes endTime on locChanges within zone at min |
| IF_SETPLAYERHEAD | 161 | 192 | g2 | modelType=3; modelId packed from localPlayer.colour[0],colour[4],appearance[0],appearance[8],appearance[11] vi |
| MIDI_SONG | 163 | 23 | g2 | If new song && midiActive && !lowMem && nextMusicDelay==0: midiSong set, midiFading=true, onDemand.request(2,s |
| UPDATE_RUNWEIGHT | 164 | 67 | g2b | if sideTab==12 redrawSidebar; runweight=g2b. 254@7115, 274@7981. size both=2. |
| CAM_RESET | 167 | 101 | none | cutscene/cinemaCam=false; for var<5 cameraModifierEnabled[]/camShake[]=false. 254@6598 cutscene+cameraModifier |
| UPDATE_INV_STOP_TRANSMIT | 168 | 227 | g2 | Clears linkObjType[] for the inventory interface (loop sets -1 then 0 - faithful redundant double-write). 254@ |
| UPDATE_INV_PARTIAL | 170 | 172 | g2, while{g1, g2, g1/g4} | redrawSidebar; per-slot update with bounds check slot in [0,len); 255->g4 escape on num. 254@6891 (linkObjCoun |
| UPDATE_ZONE_PARTIAL_FOLLOWS | 173 | 32 | g1, g1 | Just sets baseX/baseZ (zoneUpdateX/Z); subsequent zone packets arrive separately. INLINE. 254@6959, 274@8561.  |
| IF_CLOSE | 174 | 171 | none | Close all: clear side/chat/chatbackInput, mainLayerId/mainModalId=-1, pressedContinueOption/resumedPauseButton |
| VARP_SMALL | 186 | 203 | g2, g1b | varCache/varServ[id]=signed byte; if differs from varps/var[id] -> set + updateVarp/clientVar + redrawSidebar  |
| IF_OPENSIDE | 187 | 16 | g2 | Opens side modal; clears chat+chatbackInput, sets sideLayerId/sideModalId, redrawSidebar+redrawSideicons, main |
| VARP_LARGE | 196 | 245 | g2, g4 | varCache/varServ[id]=val; if differs from varps/var[id] -> set + updateVarp/clientVar + redrawSidebar + (if tu |
| IF_OPENMAIN | 197 | 211 | g2 (component) | Opens main modal. 254@6576 sets mainLayerId, clears side/chat/chatbackInput, resetInterfaceAnimation, pressedC |
| RESET_ANIMS | 203 | 47 | none | Set primarySeqId/primaryAnim=-1 for all players and npcs. 254@7296 (players/npcs, primarySeqId). 274@8213 (pla |
| SET_PLAYER_OP | 204 | 17 | g1, g1, gjstr | slot 1..5 -> playerOptions[slot-1] (254@6607) / playerOp[var-1] (274@8547); 'null'(ci) -> nil; pushDown/priori |
| REBUILD_NORMAL | 209 | 231 | g2, g2 | Scene rebuild: early-return if same center & sceneState==2; compute base tiles, tutorialIsland flags, request  |
| IF_SETMODEL | 211 | 129 | g2, g2 | modelType=1; modelId=model. 254@7141 (modelType/modelId), 274@8593 (model1Type/model1Id). size both=4. |
| UPDATE_PID | 213 | 133 | g2, g1 | localPid/selfSlot=g2; membersAccount=g1. 254@6884, 274@8587. Field rename localPid->selfSlot only. size both=3 |
| IF_SETOBJECT | 222 | 28 | g2, g2, g2 | modelType=4; pulls ObjType.get/list(obj) for xan2d/yan2d, modelZoom = zoom2d*100/var. 254@6648 (modelType/mode |
| CAM_SHAKE | 225 | 64 | g1, g1, g1, g1 | cameraModifierEnabled[type]=true + jitter/wobbleScale/wobbleSpeed + cycle=0. 254@6870 (cameraModifier* arrays) |
| IF_SETHIDE | 227 | 10 | g2, g1 | IfType.list[com].hidden/hide = (g1==1). 254@7311, 274@8464. Field rename hidden->hide. size both=3 (254) / 3 ( |
| TUT_OPEN | 239 | 130 | g2b | tutLayerId/tutComId = g2b; redrawChatback. 254@6785, 274@7876. size both=2. |
| MIDI_JINGLE | 242 | 15 | g2, g2 | If midiActive && !lowMem: midiSong set, midiFading=false, request(2), nextMusicDelay=delay. 254@7196, 274@8501 |
| IF_OPENMAIN_SIDE | 249 | 158 | g2, g2 | clears chat+chatbackInput, sets mainModalId+sideModalId, redrawSidebar+icons, resumedPauseButton=false. 274@80 |
| FRIENDLIST_LOADED | 255 | 185 | g1 | friendListStatus/friendServerStatus=g1; redrawSidebar. 254@6919, 274@8648. Field rename only. size both=1. |
| ONLY-254: 29 FINISH_TRACKING (254@6966): InputTracking.stop() -> if non-null pIsaac(142)/EVENT_TRACKING flush of recorded input to out. NO counterpart in 274 di | | | | |
| ONLY-254: 251 ENABLE_TRACKING (254@7523): InputTracking.activate(). NO counterpart in 274 (feature dropped). size254 idx251=0. | | | | |
| ONLY-274: 194 SET_MAP_STATE (274@8227): minimapState = in.g1(). NO counterpart in 254 dispatcher; the minimapState field does not exist in the 254 client. New i | | | | |

**Zone packets.** 10 ops renumbered, handler logic verified identical at both
pins (incl. LOC_ANIM's 103-bound + `groundh` corner reads — present in BOTH
revs, retiring the design-doc suspicion). 274 restructures LOC_MERGE(176) +
OBJ_COUNT(95) into a trailing `else { if(176){...} if(95){...} }` block —
mirror the structure, semantics equivalent.

| message | 254 | 274 | payload (274) | notes |
|---|---|---|---|---|
| OBJ_REVEAL | 8 | 219 | Identical wire (g1,g2,g2,g2) and identical guard [0,104) plus `receiver != selfS | op8->219. The self-exclusion guard semantics identical (don't double-show your own reveal). No logic delta bey |
| LOC_ANIM | 30 | 48 | Identical wire shape (g1,g1,g2) and identical 103-bound guard and identical 4-co | op30->48. This is the 'op-48 reads groundh' item: BOTH revs read groundh; 274 only permutes the get*/ctor arg  |
| MAP_PROJANIM | 37 | 107 | Identical wire shape and field reads in SAME order (g1,g1b,g1b,g2b,g2,g1*4,g1*4, | op37->107. The g1*4 height scaling (srcHeight,dstHeight) is preserved in BOTH. getAvH signature was reordered  |
| LOC_ADD_CHANGE / LOC_DEL | 70 | 138 | Identical wire shape: g1 tile, g1 type/rot, then op==173 -> -1 else g2. Same [0, | ADD and DEL share one branch via `op==70//88` (254) / `op==138//173` (274); DEL variant supplies type=-1 inste |
| OBJ_COUNT | 98 | 95 | Identical wire (g1,g2,g2,g2) and [0,104) guard, same id&0x7FFF + count match + r | op98->95. In 274 this is the second `if(op==95)` inside the trailing else-block (alongside 176). No logic delt |
| MAP_ANIM | 114 | 85 | Identical wire (g1,g2,g1,g2) and [0,104) guard and *128+64. DELTAS: MapSpotAnim  | op114->85. The 254 `delay` arg (var64=g2) maps to 274 var61=g2 in first ctor slot. getAvH(...)-heightOffset pr |
| OBJ_DEL | 115 | 52 | Identical wire (g1,g2) and [0,104) guard, same &0x7FFF id match + unlink + null- | op115->52. No logic delta beyond field rename. 254 site 7654-7674; 274 site 5662-5681. |
| OBJ_ADD | 120 | 81 | Identical wire (g1,g2,g2) and [0,104) guard. Same ClientObj push + showObject. O | op120->81. No logic delta beyond objStacks->groundObj. 254 site 7636-7653; 274 site 5645-5661. |
| LOC_MERGE | 218 | 176 | Identical wire shape (g1,g1,g2,g2,g2,g2,g1b x4) and SAME read order. DELTAS: own | op218->176. This is the entity-merge / morph-onto-player message (the rev-244 LocChange/LocMergeEntity merge t |

**Login handshake** (`login()`):
- 254 wrote size `out.pos+36+1+1` then `p1(254)`; 274 writes size
  `+36+1+1+2` then **`p1(255)` sentinel + `p2(274)`** (version > 255).
- Response-21 string: "transfered" → "**transferred**".
- Success path (code 2): InputTracking.deactivate() dropped (WS2);
  `minimapState = 0` reset added (WS5); **oplogic10 removed** — reset only
  oplogic1..9 (the field is gone tree-wide in 274).

**getJagChecksums() (NEW method — split out of 254 `load()` lines 1516-1541
and substantially rewritten, 274:11211-11269):**
- crc URL gains version suffix: `"crc"+rand+"-"+274`.
- Buffer/read 36 → **40 bytes**: 9 checksums + a 10th `g4` **validation
  word**: `acc=1234; for i in 0..8: acc=(acc<<1)+jagChecksum[i]`; mismatch →
  "checksum problem", `jagChecksum[8]=0`, retry.
- Error handling widened: EOFException → "EOF problem", IOException →
  "connection problem", Exception → "logic problem" guarded by
  `if (!signlink.reporterror) return;`.
- Retry counter: ≥10 failures → "Game updated - please reload page";
  backoff still doubles capped at 60s; **each failed cycle toggles
  `jaggrabEnabled`**.

**JAGGRAB transport (NEW):** fields `jaggrabSocket` + `jaggrabEnabled`
(default false). `openUrl()` gains a leading branch: when enabled,
close/reopen `jaggrabSocket = openSocket(43595)`, SoTimeout 10s, write
`"JAGGRAB /" + path + "\n\n"`, return stream on the socket; else the old
HTTP path. `getJagFile()` toggles `jaggrabEnabled` at the end of each failed
retry cycle. Go seam: route through signlink.OpenSocket so the
TCP/WebSocket transport seam is preserved.

**Other wire-level deltas:**
- `getNpcPosNewVis`: reads an **extra `gBit(1)` jump flag** before teleport
  and passes it as the jump arg (37 bits/iteration vs 36) — real wire change.
- `ClientPlayer.setAppearance`: **new `skillLevel = g2()`** between the
  combatLevel `g1()` and `ready = true` — appearance blob grows 2 bytes
  (consumed by WS8 addPlayerOptions caption).
- `ClientEntity.addHitmark` now takes the cycle as a parameter
  (callers pass `loopCycle`) instead of reading the Client global.
- `OnDemand`: socket-reopen backoff 5000→**4000 ms**; new `failCount` field
  (`= -10000` on successful send, `++` in the IOException catch); read by
  `maininit`'s wait loops — `failCount > 3` → `showLoadError("ondemand")`.
  `handleQueue(int)` → `handleQueue()` (the `arg0 != 2` guard is deleted).
- `showLoadError(String)` (NEW): print + `getAppletContext().showDocument(
  loaderror_<name>.html)` + sleep-forever; with NEW applet-shim
  `getAppletContext()` override delegating via `signlink.mainapp`. Go:
  structural mirror — log + halt; no applet context natively.
- tcpIn outer catch: `printStackTrace()` removed (revert toward pre-254
  shape) before building the "T2 -" report.

## WS2 — InputTracking removal + shell deltas [S/M]

Java 274 deletes `jagex2/client/InputTracking` entirely (0 tree-wide refs).
Go actions (sites per the inputtracking-removed + GameShell units):
- Delete `pkg/jagex2/client/inputtracking/` and its imports.
- Client strips: login success (`client.go:3775`), logout (`:7401`),
  gameLoop flush + EVENT_TRACKING send (`:8221-8227`), tcpIn FINISH_TRACKING
  (`:11040-11050`) and ENABLE_TRACKING (`:11073-11077`) handlers, and the
  `CLIENTPROT_EVENT_TRACKING` constant. (Go-side opcode comments carry
  earlier-rev values 165/28 — pre-existing skew, irrelevant: all gone.)
- GameShell strips: telemetry blocks in mousePressed (both arms),
  mouseReleased, mouseEntered, mouseExited, mouseDragged/Moved, keyPressed,
  keyReleased, focusGained, focusLost.
- GameShell real deltas: **focusLost gains a loop zeroing all
  `keyHeld[0..127]`** (stuck-key fix — gameplay-relevant);
  `shutdown(boolean)` → `shutdown()` (3 call sites); mousePressed's
  `NoSuchMethodError`/isMetaDown fallback try/catch removed.
- MouseTracking is a DIFFERENT class and **remains** (verified unchanged
  modulo churn).

## WS3 — Sound: Filter + Tone + midiFading [M]

- **`Filter.java` (NEW, 123 lines)** → new Go file in `pkg/jagex2/sound`:
  struct fields `pairs[2]`, `frequencies[2][2][4]`, `ranges[2][2][4]`,
  `unities[2]`; package-level scratch `coeff[2][8]` float32 /
  `coeffInt[2][8]` int32 / `reduceCoeff`/`reduceCoeffInt`; methods
  `radius`, `frequency` (two overloads — give distinct Go names),
  `calculateCoeffs(ch, phase) int`, `load(Packet, Envelope)` (nibble header,
  unities g2×2, migration-mask bit test `var6 & (1<<(ch*4)<<pair)`).
  Float math: keep float32 with float64 only inside math.Pow/cos calls,
  mirroring Java float semantics.
- **Tone**: new fields `filter`/`filterRange`; generate() gains the IIR pass
  between the reverb block and the final clamp loop (guard
  `pairs[0]>0 || pairs[1]>0`; int64 the `*(long)>>16` products);
  `load()` tail constructs both and calls `filter.load(p, filterRange)`;
  `unpack→load` rename; `waveFunc` 2nd/3rd params swapped (compensated —
  see churn policy); `hermonicSemitone`→`harmonicSemitone` (typo fix).
- **Envelope**: `unpack` → `load` + the point-reading half split into NEW
  `loadPoints(Packet)` (pure extraction, no logic change).
- **JagFX**: renames landed in P2; static `generate(a,b)` **transposed its
  parameter roles** (274 indexes `synth[arg0]`, passes `arg1` to getWave;
  254 was the reverse) — fix the Go static Generate accordingly (real,
  confirmed).
- **midiFading polarity flip — three consistent sites + initializer**:
  field init `false`→`true`; `maininit` music-start sets `true` (and NEW:
  parses applet param `"music"` into midiSong, default 0);
  `soundsDoQueue` next-track branch `false`→`true`; `clientVar` varp-3
  re-enable branch `false`→`true`. MIDI_SONG (fading=true) / MIDI_JINGLE
  (fading=false) handler semantics unchanged. Go watch item: the
  jingle→song deferral behavior validated in the 254 smoke test rides on
  this flag — re-verify in the 274 smoke.
- `setMidiVolume(int,boolean)` → `(boolean,int)` (real param-order change,
  4 call sites) — confirmed.

## WS4 — PixFont rework + title/UI [M]

- **PixFont** (largest single-class rework, all confirmed):
  per-char arrays `[94]`/`[95]` → **`[256]`**; `drawWidth[256]` and
  `CHAR_LOOKUP` + its static init **deleted**; ctor →
  `PixFont(JagFile, boolean wide, String name)` (drop the dummy byte);
  glyph loop reads **256** entries; font height accumulates only from
  glyphs `< 128`; space advance = `charAdvance[73]` when wide else
  `charAdvance[105]` (space is now ASCII 32, not glyph 94);
  all draw/measure methods index per-char arrays **directly by char** with
  `' '` as the skip sentinel; `evaluateTag` → `updateState`.
- **Font archives renamed**: `p11/p12/b12/q8` → `p11_full/p12_full/
  b12_full/q8_full`; q8 loads with `wide=true`, others false (maininit).
- `loadTitleImages`: NEW `fl_icon` applet-param branch — when nonzero, load
  runes at index `(i & 3) + 12` instead of `i`.
- `titleScreenDraw(boolean)` → `titleScreenDraw()`; the Login/Cancel
  buttons at loginscreen==2 are now **always drawn** (the `!arg0` guard is
  gone); "New user" → "**New User**".
- ViewBox window title → literal `"Jagex"` (version string gone; Go window
  title plumbing); System.out banner uses literal 274; `maininit` opens
  with NEW `messageBox("Starting up", 20)`.
- `drawProgress` → `messageBox` (rename); GameShell's own copy ditto.

## WS5 — minimapState machine + map-array growth [S]

- New `minimapState` int field on Client. Writes: inbound op 194 (g1);
  reset to 0 in login success and `lostCon()`. Reads: `minimapDraw` — NEW
  leading branch when `== 2`: zero `Pix2D.pixels` wherever
  `mapback.data == 0` (blackout), draw only the compass, rebind, return;
  `minimapLoop` (ex handleMinimapInput) — early-return unless `== 0`
  (click-to-walk gated).
- `mapscene` (Pix8) and `mapfunction` (Pix32) arrays **50 → 100**, with the
  two unpack loops and the rgbAdjust post-loop in `maininit` raised to 100.

## WS6 — Renderer / model / scene build [M]

- **Model.unload** gains a byte param: trailing 6 statics
  (tmpPriority11FaceDepth, tmpPriorityDepthSum, sin/cos/colour/divTable2)
  null only when arg==1; caller passes 1 (and the no-arg call in
  Client.unload passes (byte)1 — dummy there).
- **Model.translate arg→axis convention changed**: 274 maps
  (arg0→X, arg1→Y, arg3→Z); 254 mapped (arg1→X, arg3→Y, arg2→Z) — i.e. the
  Y/Z source positions swapped. ALL call sites were adapted in lockstep
  (spotanim height now the 2nd arg, wear offsets the 2nd arg, LocType
  offsets natural (x,y,z) order). Go action: keep Go's semantic
  `Translate(x, y, z)` signature; **re-verify every call site** passes
  (x, y, z) per the 274 source.
- **Pix3D.initColourTable** NEW guard: after gammaCorrect, if
  `(texPal[i][j] & 0xF8F8FF) == 0 && j != 0` force the entry to 1 (prevents
  non-transparent palette entries collapsing into the transparent key).
- **ClientBuild hillskew (NEW behavior)**: in `addLoc` AND
  `changeLocUnchecked`, decor shapes 4–8 now first rotate the four corner
  heights by `rotation` when `locType.hillskew` — port to both Go sites.
- **World.renderGround** gains a mode param (literal 7 at both call sites)
  with `if (mode != 7) return;` between the vertex and face loops — port
  faithfully (guard never fires in practice).
- `perlinNoise`/`finishBuild`/`loadGround` additive reassociations are
  numerically identical — mirror shape only when touching those lines.
- World/scene rename families land with this workstream (see §Traps for
  maxLevel/groundh/squares).

## WS7 — Config types + cache API [M]

- **LruCache API**: `get` → `find` (HashTable ditto); `put(value, key)` →
  `put(key, value)`. The arg-order half is a *compensated pair* (all
  callers swapped in lockstep) — default per standing policy: adopt the 274
  order in Go in ONE commit covering LruCache + every caller (ObjType
  modelcache/spriteCache, NpcType, LocType mc1/mc2, IfType modelCache/
  spriteCache, SpotType, ClientPlayer.modelCache), or document as
  intentional deviation; do NOT mix.
- LinkList: `addHead`→`pushFront`, `pop`→`popFront` (+LinkList2 ditto);
  **LinkList.clear gains an early-return empty guard** (real, port it).
- **VarpType**: op8 field `boolean` → **int** (op8 sets 1); **NEW op12**
  (`field1191 = g4()`, default -1, new field); **NEW op13** (sets the op8
  field to 2); `decode(int,Packet)` → `decode(Packet,int)`.
- **NpcType.getTempModel**: real signature change — mask array moved last
  `(primaryFrame, secondaryFrame, walkmerge[])`; ClientNpc call sites adapt.
- ObjType/LocType/FloType/IdkType/SeqType/SpotType/IfType: `unpack`→`init`,
  `get`→`list`, plus per-type rename families (see per-unit data); IfType
  `init` takes the data archive FIRST `(dataArchive, imageArchive, fonts)`;
  SeqType `getDuration`→`getDelay`; IdkType `type`→`part`,
  `models`→`model`.
- ObjType.getSprite: id moved to first param (id, flag, count) — confirmed
  real reorder; Go callers (drawInterface) adapt.

## WS8 — Client logic stragglers [M]

- `handleInputKey`: chat-input accept condition **drops the
  `startsWith("::") && c <= 126` extension** — only `32 <= c <= 122` now.
- `addWorldOptions` (ex handleViewportOptions): NEW inner loop over
  players co-located with a clicked size-1 NPC, adding their player
  options.
- `addPlayerOptions`: NEW caption branch — `skillLevel == 0` → combat-level
  tag (`combatColourCode`), else `name + " (skill-N)"`.
- **WordFilter** (3 confirmed): whitelist 8→10 (+`noob`, `noobs`); the
  trailing-digit discount flips `numCount -= (end - lastAlpha + 1)` →
  `- 1)`; rejected-match branch gains `else { step = 1 }` resetting the
  scan stride.
- `moveEntity(entity, size)` → `moveEntity(entity)` (param dropped;
  moveNpcs/movePlayers callers).
- `tryReconnect` → `lostCon` (rename; + the WS5 minimapState reset).
- Field reconciliation: `objSelected` → `useMode` and `worldLocationState`
  → `chatDisabled` are RENAMES (preamble's "removed/new" net-count is
  reconciled by body evidence); `objSelectedInterface/objInterface` →
  `objSelectedComId/objComId`.
- `Client.unload`/`clearCaches`/`drawError`: renames only.
- The `"wordenc"` cache-archive request literal at `client.go:6204` is
  **unchanged in 274** (`getJagFile("chat system", 7, "wordenc", ...)`).
- signlink: `cachesearch` paths gain `"c:/rscache"`, `"/rscache"` (12→14);
  `reporterror274.cgi`; `clientversion` is a dead uninitialized field in
  274 (deob folded the literal) — Go keeps its named constant, value 274,
  documented as intentional.

## Naming layer & traps

The fresh deob renames hundreds of fields/methods (full families in the
per-unit data: `audit-274/p3/digest-run2.md`). Apply renames opportunistically
with each workstream's files, updating `// Java:` cites to `@32f3062`.
**Eight name-reuse traps** (class-level pairing was clean — these are
member-level; blind seds WILL corrupt):

1. **Packet `data` ↔ `pos` SWAPPED**: 254 `data`=byte[] buffer /
   `pos`=cursor; 274 `pos`=buffer / `data`=cursor. Bytecode-identical
   everywhere. Go KEEPS `Data`=buffer / `Pos`=cursor as an intentional
   deviation with a dual-form cite (a name-following port would be
   catastrophic).
2. **Packet `gsmart` ↔ `gsmarts` method names SWAPPED** (bodies unchanged:
   274 `gsmarts` = g1-64/g2-49152; 274 `gsmart` = g1/g2-32768). Verify every
   call site by BODY not name (Tone/ClientBuild/Model call sites flip in
   lockstep).
3. **World**: 254 instance `maxLevel` → 274 `maxTileLevel`, while 254 static
   `topLevel` → 274 **`maxLevel`** (same name, different variable). And 254
   `groundHeight` (int heights) → 274 **`groundh`**, while 254 `groundh`
   (Square grid) → 274 `squares`.
4. **GameShell `canvasWidth`/`canvasHeight` → `sHei`/`sWid` INVERTED**:
   `sHei` holds the width value, `sWid` the height. Names lie; numeric
   roles unchanged — keep Go roles, cite both names.
5. **PixMap**: 254 `bind` → 274 `setPixels`, while 254 `setPixels` → 274
   `consumerSetPixels` (the name `setPixels` moves between methods).
6. **flameBuffer2 ↔ flameBuffer3 swapped roles** across
   updateFlames/drawFlames/unloadTitle (consistent; net-neutral).
7. **exactMove cycle fields name-swapped** (254 `exactMoveEndCycle` ≡ 274
   `exactMoveStart`; 254 `exactMoveStartCycle` ≡ 274 `exactMoveEnd`) —
   compensated at every read/write site; and **NpcType walkanim_l/r
   obf-slots swapped** with compensating decode order AND compensating
   routeMove turn-branch references (net wire/behavior-neutral; if adopting
   274 names, swap decode order + routeMove together).
8. **Ground minimap colour labels flip**: `underlayColour` →
   `minimapOverlay`, `overlayColour` → `minimapUnderlay`; FloType
   `luminance`→`chroma` while old `chroma`→`underlayHue` (the name `chroma`
   moves to a different quantity).

## Churn policy — refuted claims (do NOT port)

146 claims were adversarially refuted; classes (full list per-unit in
`digest-run*.md` with verifier notes):

- **Compensated positional reorders** (~100): callee param order + every
  call site changed in lockstep (World set*/del*/get*/shareLight*/ctor,
  CollisionMap blockGround/testWall/testLoc/del*, Pix2D/Pix32/PixFont/
  PixMap draw-API orders, Model/ClientLocAnim/ClientProj/MapSpotAnim/
  ViewBox/FileStream/ClientStream ctors, getAvH/tryMove/camFollow/
  drawScrollbar/addNpcOptions in Client, rsaenc RSAE/RSAN, addChat,
  WordPack pack/unpack, gdata/pdata). Go keeps its descriptive signatures —
  **no action** beyond verifying values when porting touched bodies.
  Real (non-compensated) param changes ARE in scope and listed in their
  workstreams (PixFont ctor, setMidiVolume, NpcType.getTempModel,
  titleScreenDraw, shutdown, handleQueue, moveEntity, Model.unload/
  translate, addHitmark, ObjType.getSprite, IfType.init).
- **Dead obfuscation args**: Pix2D.setClipping's leading literal 5 (unread),
  FileStream ctor's 29615 literal (unread dummy), WordFilter's dummy
  byte/int guards (filterBad 6, formatUpperCases 7, getDomainAtFilterStatus
  -8, replaceUpperCases -51), Model.load's (II) descriptor artifact.
- **Equivalent reflow/arithmetic**: early-return inversions, additive
  reassociations (followCamera, drawFlames, perlinNoise, JString ±32),
  while/for reshapes, double-`while(true)` wrappers.
- **Compensated renames**: exactMove start/end, walkanim l/r, Pix8/Pix32
  rgbAdjust param relabels (callers compensate; deltas are i.i.d. random
  anyway), PixMap.draw (x,g,y) orders, drawString param renumbering.
- **BZip2 and OnDemandProvider: ZERO deltas** — entire diffs are churn
  (BZip2's 274 deob even regressed to fieldNNN/methodNNN names; keep Go's
  descriptive names with dual cites).

## Standing decisions carried forward (do not re-fix)

- `AddChat(int,s,s) ≡ addChat(...)` and `ZonePacket(Packet,int) ≡
  zonePacket(int,Packet)` compensated pairs remain as documented.
- `CLIENTPROT_SCRAMBLED` dead table: not ported (comment updated in P2).
- flameCycle++ and savereq busy-slot guards: still not ported (254
  decisions, unchanged by 274).
- Go LruCache dup-key caller constraint, tone.go int-not-int32, loctype
  op[] 35-38 faithful overflow: unchanged.
- signlink consumer half remains the documented Go-side reconstruction
  seam (audio loop untouched by this delta except via JagFX/Tone synth).

## Appendix — Client.java method pairing & checklists

148 method pairs (full table: `audit-274/p3/client-chunks.json`): 102 by
name+arity, 43 renames adjudicated by descriptors + body, 3 NEW in 274.
Corrections made by agents during execution: the number-formatter trio was
rotated in the seed table — actual pairing `getIntString→inf`,
`formatObjCountTagged→niceNumber`, `formatObjCount→invNumber`. All other
tentative pairs confirmed (incl. handleInput→buildMinimenu,
drawProgress→messageBox, updateFlameBuffer→generateFlameCoolingMap).
NEW-in-274 adjudications: `getAppletContext` genuinely new (applet shim);
`showLoadError` genuinely new; `getJagChecksums` split-out + rewritten
(WS1). Removed methods: none (besides the InputTracking class).

Mechanical opcode checklists (every value accounted for in the tables
above): outbound 254 = 93 literal sites/76 values + 1 variable site;
outbound 274 = 90/76 + 1; inbound 70 vs 69 values; zone 10 vs 10.

Artifacts (untracked scratch, `audit-274/p3/`): run1-full.json,
run2-full.json (raw agent output incl. verifier notes), digest-run1.md,
digest-run2.md (per-unit findings), client-chunks.json (pairing table),
inline-adjudications.md, synthesis-notes.md, opcodes-raw.txt.
