# WS2 — Wire protocol / login / REBUILD + LocChange merge + lazy model (rev-244) Implementation Plan

> **STATUS: PLAN (drafted 2026-06-03).** Design approved; plan-phase research
> complete (the five open questions are now RESOLVED below). WS2 is the last gate
> before a host smoke test. Follows WS1 (`WS1-MODEL-LOADER-DESIGN.md`, DONE) and
> WS3 (`WS3-MODELSOURCE-DESIGN.md`, DONE).

> **For agentic workers:** REQUIRED SUB-SKILL: use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement task-by-task with
> two-stage review, exactly as WS1. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Make the Go client **connect to and render against an Engine-TS 244
server** — the first host smoke test since rev-244 work began.

**Architecture:** Faithful 1:1 port of **Client-Java 244** (`01f16088`),
cross-checked against **Client-TS 244** (`1cfb57b`). Eight build-gated increments:
opcode constants → incoming rewire → outgoing rewire → login → REBUILD_NORMAL +
scene state machine → LocChange merge → lazy-model sweep (B1) → Component type-6
deferred model (B2).

**Tech Stack:** Go 1.26. Existing `io.Packet` (`P1Isaac`, `G1/G2/G4/G8`, `GSmartS`,
`RSAEnc`, `PJStr`), `datastruct.LinkList`, `dash3d/model` (`TryGet`/`Request`/
`NewModel1` from WS1), `config/*`, `jstring.ToBase37`. No new dependencies.

---

## References (read before each increment)

- **Java 244** = `Client-Java` `01f16088`. Read via
  `git -C $HOME/Code/github.com/LostCityRS/Client-Java show 01f16088:src/main/java/jagex2/<path>`.
  **Read via `git show`, NOT the working tree** (local debug edits).
- **Client-TS 244** = `Client-TS` `1cfb57b` — opcode enums (`src/io/ServerProt.ts`,
  `src/io/ClientProt.ts`) + modernized login/REBUILD. `git -C …/Client-TS show 1cfb57b:<path>`.
- **Engine-TS 244** = `Engine-TS` `9aadcec` — the target server.
- Build/test (sandbox), run on EACH `go` invocation (env prefix scopes to ONE command):
  `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOPATH=$HOME/go
  GOFLAGS=-mod=mod PATH=$HOME/go/go1.26.3/bin:$PATH go build/vet/test ./...`.
  golangci-lint: `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 run`
  (`GOLANGCI_LINT_CACHE=/tmp/claude-1000/golangci-cache`, **dangerouslyDisableSandbox: true**).
  **`gofmt -l .` is a SEPARATE CI gate** — run it on every change.
- **gofmt struct-field trap:** a STANDALONE comment line above a new `Client`
  struct field triggers a mass re-tab of ~130 fields. Use TRAILING `// Java:`
  comments on struct fields.
- **Trust `go build`/`go test` over IDE/gopls `<new-diagnostics>`** (systematically
  false in this tree). Faithful port: `// Java:` refs, `int8` for signed bytes,
  match Java control flow + operator precedence. Commit `--no-gpg-sign`.

---

## Resolved facts (plan-phase research, 2026-06-03)

1. **`CLIENTPROT_LOOKUP` is DEAD.** `grep CLIENTPROT_LOOKUP 01f16088` → one hit (the
   definition). Both Java (`Packet.pIsaac: data[pos++] = ptype + random.nextInt()`)
   and TS write the RAW opcode + ISAAC, no table indirection. **Do NOT port the
   lookup** — mark it a deob artifact. Port only `SERVERPROT_LENGTH`.
2. **`SERVERPROT_LENGTH` (Java, authoritative):** 256 ints (inlined in Inc 1).
   Differs from TS `ServerProtSizes` at index 242 (`RESET_ANIMS`): Java=`6`, TS=`0`.
   **Use the Java value (`6`).**
3. **Scene build trigger moved out of the opcode handlers.** 225 inlined
   `sceneState=2 + BuildScene` in opcode 184. 244 splits it: REBUILD_NORMAL (165)
   sets `sceneState=1` + `awaitingSync=true`; PLAYER_INFO (86) *clears*
   `awaitingSync`; a NEW polled `updateSceneState()`→`checkScene()` pair
   (`Client.java:3234-3291`) flips `sceneState→2` and calls `buildScene()`. Neither
   method nor `awaitingSync` exists in Go yet.
4. **REBUILD_NORMAL (165) allocates ALL FIVE scene arrays** (`Client.java:7741-7745`,
   incl. both `byte[][]` data arrays) AND carries the entity delta-shift (7772-7854,
   the logic the 225 Go opcode-237 had). It does NOT do the 225 CRC cache-load /
   OnDemand-reply / LoopRate dance — it uses `onDemand.getMapFile`+`request(3,…)`.
5. **244 zone-loc opcodes:** LOC_ADD_CHANGE=232, LOC_DEL=125, LOC_ANIM=155,
   LOC_MERGE=29. LOC_MERGE makes ONE `appendLoc(x,0,end+1,-1,0,layer,z,level,start+1)`
   call (not two merge entities) + attaches the loc model to the player.

### Scope refinements vs. the approved design
- **Chat crowns (`@cr1@`/`@cr2@`) → DEFERRED.** Emit is easy, but rendering strips
  the prefix and plots `imageModIcons[0|1]` sprites that the Go port does not load.
  Inc 4 ships login + `StaffModLevel` only; crowns become a flagged follow-up
  (needs `imageModIcons` sprite loading — its own small workstream).
- **Opcode 103 (recolor component model) has NO 244 equivalent** → removed in Inc 2
  (also erases one B2 ripple site). Confirm the 244 server never sends it.
- **New 244 opcodes** with no 225 Go branch: `MIDI_JINGLE=173`, `IF_OPENOVERLAY=158`
  → add branches in Inc 2. `opcode 192` (obfuscated dead-write) → not ported.

---

## Architectural gap tables

### A. Opcodes / framing
| Aspect | Go 225 (current) | 244 (target) |
|---|---|---|
| Incoming sizes | `io/protocol.go` `SERVERPROT_SIZES` (225 values) | 244 `SERVERPROT_LENGTH` values (re-derive verbatim; index 242=6) |
| Incoming dispatch | `if c.PacketType == N` cascade, magic numbers (`client.go:9316+`) | same shape, 244 numbers, named `SERVERPROT_*` |
| Outgoing opcodes | scattered `P1Isaac(N)`/`P1(N)` | named `CLIENTPROT_*` (raw, no scramble) |
| Zone opcodes | 59/76/42/23/… | 232/125/155/29/… |

### B. Login (Client.java:2590-2843; Go LoginFunc client.go:6617-6872)
| Aspect | Go 225 | 244 |
|---|---|---|
| Prefix | none (reads seed directly) | `out.p1(14); out.p1(loginServer)`; `loginServer=(int)(toBase37(user)>>16&0x1FL)`; `stream.write(2,0,out)` |
| Handshake | unconditional seed read | 8 dummy reads → `reply=read()` → seed exchange only `if reply==0` |
| Version | `Login.P1(225)` (6663) | `login.p1(244)` |
| Staff flag | `Rights bool` (264; 4 sites) | `staffmodlevel int` (2→0,18→1,19→2); sites gate `>=1`/`<1` |
| Replies | 2,18 | 2,18,19; +20 "Invalid loginserver"; default "Unexpected server response" |

### C. Scene rebuild (Client.java:7704-7858, 3234-3291; Go client.go 9417-9685)
| Aspect | Go 225 | 244 |
|---|---|---|
| Opcode | 80 (cache-save) + 237 (in-band CRC list + shift) | 165 REBUILD_NORMAL (zone center only) |
| Build trigger | inlined in opcode 184 (9420-9424) | polled `updateSceneState()`→`checkScene()` + `awaitingSync` gate |
| Map fetch | CRC cache-load + server request | `onDemand.request(3, getMapFile(z,x,0|1))` |

### D. Loc-update (LocChange.java; Client.java 3431,3539,8760,8788; Go locchange.go/locmergeentity.go)
| Aspect | Go 225 | 244 |
|---|---|---|
| Types | `LocChange`(Last*) + `LocMergeEntity`(LastCycle) | one `LocChange` (old*/new*+startTime/endTime) |
| Lists | `SpawnedLocations`+`MergedLocations` | one `locChanges` |
| Drivers | inlined in ReadZonePacket; `UpdateMergeLocs`; zone-exit restore | `appendLoc`/`storeLoc`/`updateLocChanges`/`clearLocChanges`+`World.changeLocAvailable` |

### E. Lazy model (config getters; Component.java 361-523; Go config/*)
| Aspect | Go (current) | 244 |
|---|---|---|
| Getters | eager `model.NewModel1(id)` | `model.TryGet(id)` (+`Model.request` precheck; nil on miss) |
| Component type-6 | eager `*model.Model` (`Model`/`ActiveModel`) | deferred `(modelType,model)`/`(activeModelType,activeModel)` ints + `loadModel(type,id)` |

---

## Increment plan (each build/vet/test/gofmt/lint-gated + 1 commit)

Ordering keeps every commit green. The renumber is NOT compiler-checked, so Inc 1
gets an independent re-derivation+diff and Inc 2 a per-handler byte review; the
host smoke test is the runtime gate.

---

### Inc 1: Opcode constant tables (`SERVERPROT_*` / `CLIENTPROT_*`)

**Files:**
- Create: `pkg/jagex2/io/serverprot.go`
- Create: `pkg/jagex2/io/clientprot.go`
- Modify: `pkg/jagex2/io/protocol.go` (the `SERVERPROT_SIZES` array values)
- Test: `pkg/jagex2/io/protocol_test.go` (create/extend)

- [ ] **Step 1 — `serverprot.go` named constants.** Create the file with the 244
  ServerProt numbers (from `1cfb57b:src/io/ServerProt.ts`). Use SCREAMING_SNAKE
  matching the existing `SERVERPROT_SIZES` style:

```go
package io

// 244 server→client opcodes. Java: jagex2.io.Protocol (numbers) +
// Client-TS src/io/ServerProt.ts (names). Renumbered from rev-225.
const (
	SERVERPROT_IF_OPENCHAT                  = 189
	SERVERPROT_IF_OPENMAIN_SIDE             = 207
	SERVERPROT_IF_CLOSE                     = 214
	SERVERPROT_IF_SETTAB                    = 200
	SERVERPROT_IF_OPENMAIN                  = 10
	SERVERPROT_IF_OPENSIDE                  = 176
	SERVERPROT_IF_OPENOVERLAY               = 158 // NEW in 244 (no 225 Go branch)
	SERVERPROT_IF_SETTAB_ACTIVE             = 56
	SERVERPROT_IF_SETCOLOUR                 = 78
	SERVERPROT_IF_SETHIDE                   = 123
	SERVERPROT_IF_SETOBJECT                 = 164
	SERVERPROT_IF_SETMODEL                  = 245
	SERVERPROT_IF_SETANIM                   = 219
	SERVERPROT_IF_SETPLAYERHEAD             = 108
	SERVERPROT_IF_SETTEXT                   = 154
	SERVERPROT_IF_SETNPCHEAD                = 129
	SERVERPROT_IF_SETPOSITION               = 241
	SERVERPROT_TUT_FLASH                    = 168
	SERVERPROT_TUT_OPEN                     = 174
	SERVERPROT_UPDATE_INV_STOP_TRANSMIT     = 162
	SERVERPROT_UPDATE_INV_FULL              = 72
	SERVERPROT_UPDATE_INV_PARTIAL           = 132
	SERVERPROT_CAM_LOOKAT                    = 222
	SERVERPROT_CAM_SHAKE                     = 50
	SERVERPROT_CAM_MOVETO                    = 12
	SERVERPROT_CAM_RESET                     = 53
	SERVERPROT_NPC_INFO                      = 244
	SERVERPROT_PLAYER_INFO                   = 86
	SERVERPROT_FINISH_TRACKING               = 60
	SERVERPROT_ENABLE_TRACKING               = 22
	SERVERPROT_MESSAGE_GAME                  = 95
	SERVERPROT_UPDATE_IGNORELIST             = 7
	SERVERPROT_CHAT_FILTER_SETTINGS          = 9
	SERVERPROT_MESSAGE_PRIVATE               = 30
	SERVERPROT_UPDATE_FRIENDLIST             = 70
	SERVERPROT_UNSET_MAP_FLAG                = 62
	SERVERPROT_UPDATE_RUNWEIGHT              = 160
	SERVERPROT_HINT_ARROW                    = 49
	SERVERPROT_UPDATE_REBOOT_TIMER           = 85
	SERVERPROT_UPDATE_STAT                   = 24
	SERVERPROT_UPDATE_RUNENERGY              = 177
	SERVERPROT_RESET_ANIMS                   = 242
	SERVERPROT_UPDATE_PID                    = 210
	SERVERPROT_LAST_LOGIN_INFO               = 44
	SERVERPROT_LOGOUT                        = 17
	SERVERPROT_P_COUNTDIALOG                 = 152
	SERVERPROT_SET_MULTIWAY                  = 97
	SERVERPROT_REBUILD_NORMAL                = 165
	SERVERPROT_VARP_SMALL                    = 236
	SERVERPROT_VARP_LARGE                    = 226
	SERVERPROT_RESET_CLIENT_VARCACHE         = 87
	SERVERPROT_SYNTH_SOUND                   = 151
	SERVERPROT_MIDI_SONG                     = 240
	SERVERPROT_MIDI_JINGLE                   = 173 // NEW in 244 (no 225 Go branch)
	SERVERPROT_UPDATE_ZONE_PARTIAL_FOLLOWS   = 94
	SERVERPROT_UPDATE_ZONE_FULL_FOLLOWS      = 131
	SERVERPROT_UPDATE_ZONE_PARTIAL_ENCLOSED  = 233
	SERVERPROT_LOC_MERGE                     = 29
	SERVERPROT_LOC_ANIM                      = 155
	SERVERPROT_OBJ_DEL                       = 39
	SERVERPROT_OBJ_REVEAL                    = 69
	SERVERPROT_LOC_ADD_CHANGE                = 232
	SERVERPROT_MAP_PROJANIM                  = 137
	SERVERPROT_LOC_DEL                       = 125
	SERVERPROT_OBJ_COUNT                     = 209
	SERVERPROT_MAP_ANIM                      = 198
	SERVERPROT_OBJ_ADD                       = 234
)
```

- [ ] **Step 2 — `clientprot.go` named constants.** Create with the 244 ClientProt
  numbers (from `1cfb57b:src/io/ClientProt.ts`). Include a comment that
  `CLIENTPROT_LOOKUP` is a deob artifact, intentionally not ported:

```go
package io

// 244 client→server opcodes. Java: jagex2.io.Protocol + Client-TS src/io/ClientProt.ts.
// Written RAW through Packet.P1Isaac (opcode + ISAAC keystream). Java's
// CLIENTPROT_LOOKUP table is a deobfuscation artifact (never read at runtime by
// Java or TS) — intentionally not ported.
const (
	CLIENTPROT_NO_TIMEOUT             = 107
	CLIENTPROT_IDLE_TIMER             = 146
	CLIENTPROT_EVENT_TRACKING         = 217
	CLIENTPROT_ANTICHEAT_OPLOGIC1     = 47
	CLIENTPROT_ANTICHEAT_OPLOGIC2     = 218
	CLIENTPROT_ANTICHEAT_OPLOGIC3     = 37
	CLIENTPROT_ANTICHEAT_OPLOGIC4     = 34
	CLIENTPROT_ANTICHEAT_OPLOGIC5     = 7
	CLIENTPROT_ANTICHEAT_OPLOGIC6     = 177
	CLIENTPROT_ANTICHEAT_OPLOGIC7     = 50
	CLIENTPROT_ANTICHEAT_OPLOGIC8     = 100
	CLIENTPROT_ANTICHEAT_OPLOGIC9     = 169
	CLIENTPROT_ANTICHEAT_CYCLELOGIC1  = 46
	CLIENTPROT_ANTICHEAT_CYCLELOGIC2  = 148
	CLIENTPROT_ANTICHEAT_CYCLELOGIC3  = 144
	CLIENTPROT_ANTICHEAT_CYCLELOGIC4  = 41
	CLIENTPROT_ANTICHEAT_CYCLELOGIC5  = 232
	CLIENTPROT_ANTICHEAT_CYCLELOGIC6  = 215
	CLIENTPROT_OPOBJ1                 = 231
	CLIENTPROT_OPOBJ2                 = 110
	CLIENTPROT_OPOBJ3                 = 27
	CLIENTPROT_OPOBJ4                 = 17
	CLIENTPROT_OPOBJ5                 = 225
	CLIENTPROT_OPOBJT                 = 25
	CLIENTPROT_OPOBJU                 = 111
	CLIENTPROT_OPNPC1                 = 222
	CLIENTPROT_OPNPC2                 = 84
	CLIENTPROT_OPNPC3                 = 132
	CLIENTPROT_OPNPC4                 = 229
	CLIENTPROT_OPNPC5                 = 102
	CLIENTPROT_OPNPCT                 = 101
	CLIENTPROT_OPNPCU                 = 52
	CLIENTPROT_OPLOC1                 = 238
	CLIENTPROT_OPLOC2                 = 38
	CLIENTPROT_OPLOC3                 = 19
	CLIENTPROT_OPLOC4                 = 55
	CLIENTPROT_OPLOC5                 = 243
	CLIENTPROT_OPLOCT                 = 182
	CLIENTPROT_OPLOCU                 = 106
	CLIENTPROT_OPPLAYER1              = 211
	CLIENTPROT_OPPLAYER2              = 219
	CLIENTPROT_OPPLAYER3              = 64
	CLIENTPROT_OPPLAYER4              = 43
	CLIENTPROT_OPPLAYERT              = 73
	CLIENTPROT_OPPLAYERU              = 48
	CLIENTPROT_OPHELD1                = 228
	CLIENTPROT_OPHELD2                = 166
	CLIENTPROT_OPHELD3                = 221
	CLIENTPROT_OPHELD4                = 6
	CLIENTPROT_OPHELD5                = 133
	CLIENTPROT_OPHELDT                = 143
	CLIENTPROT_OPHELDU                = 58
	CLIENTPROT_INV_BUTTON1            = 153
	CLIENTPROT_INV_BUTTON2            = 193
	CLIENTPROT_INV_BUTTON3            = 158
	CLIENTPROT_INV_BUTTON4            = 204
	CLIENTPROT_INV_BUTTON5            = 212
	CLIENTPROT_IF_BUTTON              = 39
	CLIENTPROT_RESUME_PAUSEBUTTON     = 11
	CLIENTPROT_CLOSE_MODAL            = 187
	CLIENTPROT_RESUME_P_COUNTDIALOG   = 190
	CLIENTPROT_TUTORIAL_CLICKSIDE     = 233
	CLIENTPROT_MOVE_OPCLICK           = 167
	CLIENTPROT_REPORT_ABUSE           = 251
	CLIENTPROT_MOVE_MINIMAPCLICK      = 56
	CLIENTPROT_INV_BUTTOND            = 81
	CLIENTPROT_IGNORELIST_DEL         = 207
	CLIENTPROT_IGNORELIST_ADD         = 203
	CLIENTPROT_IF_PLAYERDESIGN        = 8
	CLIENTPROT_CHAT_SETMODE           = 98
	CLIENTPROT_MESSAGE_PRIVATE        = 170
	CLIENTPROT_FRIENDLIST_DEL         = 69
	CLIENTPROT_FRIENDLIST_ADD         = 9
	CLIENTPROT_CLIENT_CHEAT           = 76
	CLIENTPROT_MESSAGE_PUBLIC         = 171
	CLIENTPROT_MOVE_GAMECLICK         = 63
)
```

- [ ] **Step 3 — update `SERVERPROT_SIZES` to the 244 array.** In
  `pkg/jagex2/io/protocol.go`, replace the array VALUES with the Java
  `SERVERPROT_LENGTH` (256 ints, verbatim — note index 242 = `6`):

```go
var SERVERPROT_SIZES = []int{
	0, 0, 0, 0, 0, 0, 0, -2, 0, 3, 2, 0, 6, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 6, 0,
	0, 0, 0, 14, -1, 0, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 10, 0, 0, 0, 0, 6, 4, 0,
	0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 7, 9, 0, -2, 0, 0, 0, 0, 0, 4,
	0, 0, 0, 0, 0, 0, 2, -2, 0, 0, 0, 0, 0, 0, 0, 2, -1, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3, 0, 2, 0, 0, 0, 4, 0, 2, -2,
	0, 0, 0, 0, 15, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5, 0, 0, -2, 4, 0, 0, 2, 0,
	2, 0, 2, 0, 6, 4, 0, 0, 1, 0, 0, 0, 0, 4, 2, 0, 2, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 6, 0, 3, 0, 0, 0, 0, 0, 0, 4, 0, 7, 3, 0, 0, 0,
	0, 0, 0, 0, 0, 4, 0, 0, 6, 0, 0, 0, 6, 0, 0, 0, 0, 0, 4, -2, 5, 0, 3, 0, 0, 0, 2,
	6, 0, 0, -2, 4, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0, 0,
}
```
  (If `protocol.go` has a `// Java:` comment, update it to cite
  `Protocol.SERVERPROT_LENGTH (Protocol.java)`.)

- [ ] **Step 4 — test.** In `protocol_test.go`: assert `len(SERVERPROT_SIZES) == 256`;
  assert spot values that pin the renumber against the named consts:
  `SERVERPROT_SIZES[SERVERPROT_REBUILD_NORMAL]` (165) `== 0`,
  `SERVERPROT_SIZES[29]` (LOC_MERGE) `== 14`, `SERVERPROT_SIZES[242]` (RESET_ANIMS) `== 6`,
  `SERVERPROT_SIZES[SERVERPROT_NPC_INFO]` (244) `== 0`,
  `SERVERPROT_SIZES[SERVERPROT_MESSAGE_GAME]` (95) `== -2` (var-length).
  Run: `… go test ./pkg/jagex2/io/...` → PASS.
- [ ] **Step 5 — independent cross-check (review gate).** A second agent
  re-derives `SERVERPROT_LENGTH` + the two enums fresh from `01f16088`/`1cfb57b`
  and diffs against the committed files. Fix any mismatch before commit.
- [ ] **Step 6 — gate + commit.** build/vet/test/gofmt/lint green.
  `git commit --no-gpg-sign -m "feat(rev-244): SERVERPROT_*/CLIENTPROT_* opcode constants (WS2 Inc 1)"`

---

### Inc 2: Incoming dispatch rewire (main cascade + ReadZonePacket)

**Files:**
- Modify: `pkg/jagex2/client/client.go` (cascade ~9316-10350; `ReadZonePacket` ~1088-1388)

The bodies of most handlers DO NOT change — only the opcode literal in
`if c.PacketType == <225#>` becomes `== io.SERVERPROT_<NAME>` (= the 244 number).
The renumber map is the **225→244 correspondence table** below (semantic match).
Rows marked **Δ** are logic-deltas (change the body too); rows marked **✂** are
removed; **NEW** rows add a branch.

#### Main cascade renumber map (225# → 244 const)
| 225# | meaning | 244 const | 244# | note |
|---|---|---|---|---|
| 4 | message game | SERVERPROT_MESSAGE_GAME | 95 | |
| 41 | private message | SERVERPROT_MESSAGE_PRIVATE | 30 | |
| 1 | npc info | SERVERPROT_NPC_INFO | 244 | |
| 162 | zone partial enclosed | SERVERPROT_UPDATE_ZONE_PARTIAL_ENCLOSED | 233 | |
| 184 | player info + scene build | SERVERPROT_PLAYER_INFO | 86 | **Δ** (Inc 5: strip build, add `awaitingSync=false`) |
| 150 | varp small | SERVERPROT_VARP_SMALL | 236 | |
| 152 | friendlist | SERVERPROT_UPDATE_FRIENDLIST | 70 | |
| 43 | reboot timer | SERVERPROT_UPDATE_REBOOT_TIMER | 85 | |
| 80 | map land cache save | — | — | **✂** (→ Inc 5) |
| 237 | scene rebuild + shift | SERVERPROT_REBUILD_NORMAL | 165 | **Δ✂** (→ Inc 5) |
| 197 | if_setplayerhead | SERVERPROT_IF_SETPLAYERHEAD | 108 | **Δ** (Inc 8 deferred-model) |
| 25 | hint arrow | SERVERPROT_HINT_ARROW | 49 | |
| 54 | midi song | SERVERPROT_MIDI_SONG | 240 | **Δ** (now `g2 id`→`onDemand.request(2,id)`) |
| 142 | logout | SERVERPROT_LOGOUT | 17 | |
| 20 | map loc cache save | — | — | **✂** |
| 19 | unset map flag | SERVERPROT_UNSET_MAP_FLAG | 62 | |
| 139 | update pid | SERVERPROT_UPDATE_PID | 210 | **Δ** (+`membersAccount=g1`) |
| 28 | if_openmain_side | SERVERPROT_IF_OPENMAIN_SIDE | 207 | |
| 175 | varp large | SERVERPROT_VARP_LARGE | 226 | |
| 146 | if_setanim | SERVERPROT_IF_SETANIM | 219 | |
| 167 | if_settab | SERVERPROT_IF_SETTAB | 200 | |
| 220 | map loc data inbound | — | — | **✂** |
| 133 | finish tracking | SERVERPROT_FINISH_TRACKING | 60 | **Δ** (outbound reply 81→217 in Inc 3) |
| 98 | inv full | SERVERPROT_UPDATE_INV_FULL | 72 | |
| 226 | enable tracking | SERVERPROT_ENABLE_TRACKING | 22 | |
| 243 | p_countdialog | SERVERPROT_P_COUNTDIALOG | 152 | |
| 15 | inv stop transmit | SERVERPROT_UPDATE_INV_STOP_TRANSMIT | 162 | |
| 140 | last login info | SERVERPROT_LAST_LOGIN_INFO | 44 | |
| 126 | tut flash | SERVERPROT_TUT_FLASH | 168 | |
| 212 | midi push (bzip2) | — | — | **✂** (midi via OnDemand) |
| 254 | set multiway | SERVERPROT_SET_MULTIWAY | 97 | |
| 12 | synth sound | SERVERPROT_SYNTH_SOUND | 151 | |
| 204 | if_setnpchead | SERVERPROT_IF_SETNPCHEAD | 129 | **Δ** (Inc 8 deferred-model) |
| 7 | zone partial follows | SERVERPROT_UPDATE_ZONE_PARTIAL_FOLLOWS | 94 | |
| 103 | recolor component model | — | — | **✂** (no 244 match; confirm w/ server) |
| 32 | chat filter settings | SERVERPROT_CHAT_FILTER_SETTINGS | 9 | |
| 195 | if_openside | SERVERPROT_IF_OPENSIDE | 176 | |
| 14 | if_openchat | SERVERPROT_IF_OPENCHAT | 189 | |
| 209 | if_setposition | SERVERPROT_IF_SETPOSITION | 241 | |
| 3 | cam_moveto | SERVERPROT_CAM_MOVETO | 12 | |
| 135 | zone full follows | SERVERPROT_UPDATE_ZONE_FULL_FOLLOWS | 131 | **Δ** (loc body → Inc 6: `endTime=0`) |
| 132 | map land data inbound | — | — | **✂** |
| 193 | reset client varcache | SERVERPROT_RESET_CLIENT_VARCACHE | 87 | |
| 87 | if_setmodel | SERVERPROT_IF_SETMODEL | 245 | **Δ** (Inc 8) |
| 185 | tut open | SERVERPROT_TUT_OPEN | 174 | |
| 68 | run energy | SERVERPROT_UPDATE_RUNENERGY | 177 | |
| 74 | cam_lookat | SERVERPROT_CAM_LOOKAT | 222 | |
| 84 | if_settab_active | SERVERPROT_IF_SETTAB_ACTIVE | 56 | |
| 46 | if_setobject | SERVERPROT_IF_SETOBJECT | 164 | **Δ** (Inc 8) |
| 168 | if_openmain | SERVERPROT_IF_OPENMAIN | 10 | |
| 2 | if_setcolour | SERVERPROT_IF_SETCOLOUR | 78 | |
| 136 | reset anims | SERVERPROT_RESET_ANIMS | 242 | |
| 26 | if_sethide | SERVERPROT_IF_SETHIDE | 123 | |
| 21 | update ignorelist | SERVERPROT_UPDATE_IGNORELIST | 7 | |
| 239 | cam_reset | SERVERPROT_CAM_RESET | 53 | |
| 129 | if_close | SERVERPROT_IF_CLOSE | 214 | |
| 201 | if_settext | SERVERPROT_IF_SETTEXT | 154 | |
| 44 | update stat | SERVERPROT_UPDATE_STAT | 24 | |
| 22 | run weight | SERVERPROT_UPDATE_RUNWEIGHT | 160 | |
| 13 | cam_shake | SERVERPROT_CAM_SHAKE | 50 | |
| 213 | inv partial | SERVERPROT_UPDATE_INV_PARTIAL | 132 | |

#### Zone-packet renumber (ReadZonePacket; both the main-cascade dispatch guard AND the sub-opcode checks)
| 225# | meaning | 244 const | 244# |
|---|---|---|---|
| 59 | loc add/change | SERVERPROT_LOC_ADD_CHANGE | 232 |
| 76 | loc del | SERVERPROT_LOC_DEL | 125 |
| 42 | loc anim | SERVERPROT_LOC_ANIM | 155 |
| 223 | obj add | SERVERPROT_OBJ_ADD | 234 |
| 49 | obj del | SERVERPROT_OBJ_DEL | 39 |
| 69 | map projanim | SERVERPROT_MAP_PROJANIM | 137 |
| 191 | map anim | SERVERPROT_MAP_ANIM | 198 |
| 50 | obj reveal | SERVERPROT_OBJ_REVEAL | 69 |
| 23 | loc merge | SERVERPROT_LOC_MERGE | 29 |
| 151 | obj count | SERVERPROT_OBJ_COUNT | 209 |

The 225 main cascade dispatches the zone group via one combined
`if c.PacketType == 151 || …== 59 { c.ReadZonePacket(...) }`. Update that guard to
the 244 numbers `{209,29,69,198,137,39,234,155,125,232}` and the sub-opcode `==`
checks inside `ReadZonePacket` likewise. **Collision trap: match by body, never by
number** — e.g. 225 zone `69`=projanim but 244 `69`=OBJ_REVEAL.

- [ ] **Step 1 — renumber the main cascade.** For every non-✂/non-NEW row, change
  the `if c.PacketType == <225#>` literal to `if c.PacketType == io.SERVERPROT_<NAME>`.
  Add `import` of the `io` const if not already named-imported (it's the same `io`
  package already used). Leave the Δ bodies as-is for now (their body changes land
  in Inc 5/6/8) EXCEPT the two simple Δ below.
- [ ] **Step 2 — simple Δ bodies.** Two body changes that belong here (small, local):
  - **MIDI_SONG (240, was 54):** replace the in-band filename/crc/size read +
    `SetMidi(...)` with `id := c.In.G2(); if id == 65535 { id = -1 }` then request
    via OnDemand: `c.OnDemand.Request(2, id)` (mirror `Client.java` MIDI_SONG —
    quote it via `git show` to get exact midi-fade/`currentMidi` handling). The
    archive-2 dispatch already exists (WS1).
  - **UPDATE_PID (210, was 139):** after `c.LocalPID = c.In.G2()` add the new
    `membersAccount := c.In.G1()` read (store in a new `Client` field
    `MembersAccount int` with a TRAILING `// Java: membersAccount (Client.java)`
    comment, or discard if unused downstream — check the Java).
- [ ] **Step 3 — remove the ✂ rows.** Delete the handler branches for 80, 20, 220,
  132, 212, 103. (237 and the 184 build-split are handled in Inc 5; leave the 184
  branch renumbered-to-86 with its body intact for now — Inc 5 edits it.)
- [ ] **Step 4 — add NEW branches.** Port from `Client.java` (`git show`):
  - **MIDI_JINGLE (173):** quote the Java handler; it reads a jingle id + delay and
    `onDemand.request(2, id)` (request via OnDemand archive 2). Add a faithful Go
    branch.
  - **IF_OPENOVERLAY (158):** Java `viewportOverlayInterfaceId = g2b()`. Add a new
    `Client` field `ViewportOverlayInterfaceID int` (trailing `// Java:` comment) +
    the branch. Wire its read where 244 reads it in `drawScene` if trivial; else a
    `// WS-followup` note (overlay render is cosmetic, not connect-critical).
- [ ] **Step 5 — renumber ReadZonePacket.** Update the dispatch guard + sub-opcode
  `==` checks to the 244 numbers (zone table above). Leave loc-branch BODIES for
  Inc 6 (they reference SpawnedLocations/MergedLocations which Inc 6 replaces).
- [ ] **Step 6 — per-handler byte review (review gate).** A reviewer confirms each
  renumbered handler's PAYLOAD reads still match the 244 `SERVERPROT_LENGTH` size
  and the Java body (esp. player/npc info `getPlayerPos`/`getNpcPos` + extended-info
  bit reads — expected unchanged; UPDATE_PID's extra byte; the zone guard).
- [ ] **Step 7 — gate + commit.** build/vet/test/gofmt/lint green.
  `git commit --no-gpg-sign -m "feat(rev-244): renumber incoming opcodes 225→244 + zone dispatch (WS2 Inc 2)"`

---

### Inc 3: Outgoing opcode renumber (`CLIENTPROT_*`)

**Files:**
- Modify: `pkg/jagex2/client/client.go` (scattered `P1Isaac(N)`/`P1(N)` outgoing writes)

- [ ] **Step 1 — map + replace.** Find every outgoing opcode write (`grep -n
  "P1Isaac(" pkg/jagex2/client/client.go` + the login `P1` writes are NOT opcodes —
  skip those). For each, identify the action and replace the magic number with the
  `io.CLIENTPROT_<NAME>` const at the 244 number. Cross-reference the Java
  `pIsaac(...)` call sites (`git show 01f16088:…Client.java`) by body to get the
  correct 244 opcode per action (e.g. movement → MOVE_GAMECLICK 63 / MOVE_MINIMAPCLICK
  56; idle → IDLE_TIMER 146 / NO_TIMEOUT 107; the FINISH_TRACking outbound reply
  217; OP* menu actions; INV_BUTTON*; chat MESSAGE_PUBLIC 171 / CLIENT_CHEAT 76;
  friend/ignore list ops). **No scramble** — write the raw const through `P1Isaac`.
- [ ] **Step 2 — FINISH_TRACKING outbound.** The 225 handler at the old opcode-133
  (now 60) writes outbound `81`; change that outbound write to
  `io.CLIENTPROT_EVENT_TRACKING` (217).
- [ ] **Step 3 — review gate.** Reviewer confirms each outbound opcode matches the
  Java `pIsaac` body by action; flag any 225 outbound with no clean 244 match.
- [ ] **Step 4 — gate + commit.** Green.
  `git commit --no-gpg-sign -m "feat(rev-244): renumber outgoing opcodes 225→244 (WS2 Inc 3)"`

---

### Inc 4: Login handshake + `StaffModLevel`

**Files:**
- Modify: `pkg/jagex2/client/client.go` (`LoginFunc` 6617-6872; `Rights` field 264; read sites 2002,2018,5190,5320)

Java contract (`Client.java:2590-2658`, the part Go is missing): write the
`p1(14)+p1(loginServer)` prefix, `stream.write(2,0,out)`, 8 dummy reads, read a
first `reply`, and ONLY `if reply==0` do the seed exchange (currently Go does the
seed read unconditionally at the top).

- [ ] **Step 1 — `Rights bool` → `StaffModLevel int`.** Change the field decl
  (`client.go:264`) from `Rights bool` to `StaffModLevel int` (keep it in the
  existing alignment block; trailing comment only). Update the 4 read sites:
  - `2002`, `2018`, `5190`: `if c.Rights {` → `if c.StaffModLevel >= 1 {`
  - `5320`: `if !c.Rights {` → `if c.StaffModLevel < 1 {`
- [ ] **Step 2 — restructure `LoginFunc` handshake.** Edit `LoginFunc` to match
  Java 2600-2657. After `c.Stream = clientstream.NewClientStream(conn)`:

```go
// Java: Client.login (Client.java:2602-2619). Prefix + 8 dummy reads + reply gate.
username37 := jstring.ToBase37(arg0)
loginServer := int(username37 >> 16 & 0x1F)
c.Out.Pos = 0
c.Out.P1(14)
c.Out.P1(loginServer)
if err := c.Stream.Write(c.Out.Data, c.Out.Pos, 0); err != nil { /* connect-error path */ }
for i := 0; i < 8; i++ {
	if _, err := c.Stream.Read(); err != nil { /* connect-error path */ }
}
reply, err := c.Stream.Read()
if err != nil { /* connect-error path */ }
if reply == 0 {
	if err := c.Stream.ReadFully(c.In.Data, 0, 8); err != nil { /* connect-error path */ }
	c.In.Pos = 0
	c.ServerSeed = c.In.G8()
	// ... existing seed[]/RSA/login-block build, but with c.Login.P1(244) ...
	reply, err = c.Stream.Read()
	if err != nil { /* connect-error path */ }
}
```
  Move the existing seed/RSA/login-block code (6645-6682) INTO the `if reply==0`
  block, and change `c.Login.P1(225)` → `c.Login.P1(244)`. The subsequent reply
  switch operates on the (possibly re-read) `reply`. Match the Java `Stream.write`
  arg order to the Go `Stream.Write(data, len, off)` signature already used at 6678.
- [ ] **Step 3 — reply switch.** Replace the `if var7 == 2 || var7 == 18` block
  (6694-6699) with:

```go
if reply == 2 || reply == 18 || reply == 19 {
	c.StaffModLevel = 0
	if reply == 18 {
		c.StaffModLevel = 1
	} else if reply == 19 {
		c.StaffModLevel = 2
	}
	inputtracking.SetDisabled()
	// ... existing game-state reset + c.PrepareGameScreen(); return ...
}
```
  Add two new reply branches before the trailing fall-through (Java 2832-2838):
  `if reply == 20 { c.LoginMessage0 = "Invalid loginserver requested"; c.LoginMessage1 = "Please try using a different world."; return }`
  and a final `else` default `"Unexpected server response" / "Please try using a different world."`.
  (The existing 3-17 branches stay; renumber none — these are login replies, not opcodes.)
- [ ] **Step 4 — test (pure logic).** Add a small test for the loginServer
  computation in a testable helper, OR a `jstring`-level assertion:
  `loginServer = int(jstring.ToBase37("zezima") >> 16 & 0x1F)` returns a value in
  `[0,31]`. (The handshake itself needs a live stream → build-gated + review.)
- [ ] **Step 5 — review gate.** Reviewer diffs the restructured `LoginFunc` against
  `Client.java:2590-2843` byte-for-byte (prefix, 8 reads, reply gate, version 244,
  staffmodlevel mapping, reply 20/default).
- [ ] **Step 6 — gate + commit.** Green.
  `git commit --no-gpg-sign -m "feat(rev-244): 244 login handshake + StaffModLevel (WS2 Inc 4)"`

> **DEFERRED (flagged):** chat crowns `@cr1@`/`@cr2@` (emit in `AddMessage` by
> `StaffModLevel`; render strips prefix + plots `imageModIcons[0|1]`). Needs
> `imageModIcons` sprite loading (absent in Go) — its own small follow-up.

---

### Inc 5: REBUILD_NORMAL (165) + scene state machine

**Files:**
- Modify: `pkg/jagex2/client/client.go` (remove 80/237 ~9520-9685; add 165 handler;
  split PLAYER_INFO (86); add `UpdateSceneState`/`CheckScene`; new fields; main-loop hook)

New `Client` fields (trailing `// Java:` comments to avoid the gofmt re-tab):
`AwaitingSync bool` (Java `awaitingSync`), `WithinTutorialIsland bool` (Java
`withinTutorialIsland`), `SceneLoadStartTime int64` (Java `sceneLoadStartTime`).
Verify `MapLastBaseX/Z` already exist (used at 9612-9615 — yes).

- [ ] **Step 1 — add the new fields** to the `Client` struct (in the existing
  alignment block, trailing comments).
- [ ] **Step 2 — port REBUILD_NORMAL (165).** Add a new `if c.PacketType ==
  io.SERVERPROT_REBUILD_NORMAL {` branch, a faithful port of `Client.java:7704-7858`.
  Key Go translation (use the existing Go field/method names; `c.OnDemand.GetMapFile(z,x,t)`
  + `c.OnDemand.Request(3, file)` from WS1; integer division `(zone±6)/8` matches Java):

```go
// Java: Client.readPacket REBUILD_NORMAL (Client.java:7704-7858).
if c.PacketType == io.SERVERPROT_REBUILD_NORMAL {
	zoneX := c.In.G2()
	zoneZ := c.In.G2()
	if c.SceneCenterZoneX == zoneX && c.SceneCenterZoneZ == zoneZ && c.SceneState == 2 {
		c.PacketType = -1
		return true
	}
	c.SceneCenterZoneX = zoneX
	c.SceneCenterZoneZ = zoneZ
	c.SceneBaseTileX = (c.SceneCenterZoneX - 6) * 8
	c.SceneBaseTileZ = (c.SceneCenterZoneZ - 6) * 8
	c.WithinTutorialIsland = false
	if (c.SceneCenterZoneX/8 == 48 || c.SceneCenterZoneX/8 == 49) && c.SceneCenterZoneZ/8 == 48 {
		c.WithinTutorialIsland = true
	} else if c.SceneCenterZoneX/8 == 48 && c.SceneCenterZoneZ/8 == 148 {
		c.WithinTutorialIsland = true
	}
	c.SceneState = 1
	c.SceneLoadStartTime = <now-millis> // match Go's existing time source; Java System.currentTimeMillis()
	c.AreaViewport.Bind()
	c.FontPlain12.CentreString(151, 0, "Loading - please wait.", 257)
	c.FontPlain12.CentreString(150, 0xFFFFFF, "Loading - please wait.", 256)
	c.presentLoadingMessage()
	regions := 0
	for x := (c.SceneCenterZoneX - 6) / 8; x <= (c.SceneCenterZoneX+6)/8; x++ {
		for z := (c.SceneCenterZoneZ - 6) / 8; z <= (c.SceneCenterZoneZ+6)/8; z++ {
			regions++
		}
	}
	c.SceneMapLandData = make([][]byte, regions)
	c.SceneMapLocData = make([][]byte, regions)
	c.SceneMapIndex = make([]int, regions)
	c.SceneMapLandFile = make([]int, regions)
	c.SceneMapLocFile = make([]int, regions)
	mapCount := 0
	for x := (c.SceneCenterZoneX - 6) / 8; x <= (c.SceneCenterZoneX+6)/8; x++ {
		for z := (c.SceneCenterZoneZ - 6) / 8; z <= (c.SceneCenterZoneZ+6)/8; z++ {
			c.SceneMapIndex[mapCount] = (x << 8) + z
			if c.WithinTutorialIsland && (z == 49 || z == 149 || z == 147 || x == 50 || (x == 49 && z == 47)) {
				c.SceneMapLandFile[mapCount] = -1
				c.SceneMapLocFile[mapCount] = -1
			} else {
				landFile := c.OnDemand.GetMapFile(z, x, 0)
				c.SceneMapLandFile[mapCount] = landFile
				if landFile != -1 {
					c.OnDemand.Request(3, landFile)
				}
				locFile := c.OnDemand.GetMapFile(z, x, 1)
				c.SceneMapLocFile[mapCount] = locFile
				if locFile != -1 {
					c.OnDemand.Request(3, locFile)
				}
			}
			mapCount++
		}
	}
	// --- entity delta-shift: faithful port of Client.java:7772-7854 ---
	//   (dx/dz from SceneBaseTile - MapLastBase; NPC/player route+world shift;
	//    AwaitingSync=true; 4-layer LevelObjStacks directional shift; LocChange
	//    x/z shift+unlink; FlagSceneTile shift; Cutscene=false). REUSE the existing
	//    225 opcode-237 shift code (client.go:9612-9682) VERBATIM — it is identical
	//    to 244 7772-7854; lift it into this handler. NOTE the LocChange loop uses
	//    the post-Inc-6 single list (c.LocChanges); until Inc 6 lands, keep it
	//    pointing at c.SpawnedLocations and update in Inc 6.
	c.AwaitingSync = true
	// ... lifted shift code ...
	c.Cutscene = false
	c.PacketType = -1
	return true
}
```
  (For `<now-millis>`: use whatever millis source the Go port already uses for
  timeouts; grep for an existing `time.Now().UnixMilli()`/`signlink` clock. If none,
  add a minimal `nowMillis()` helper. `SceneLoadStartTime` is only read by the
  360000ms timeout warning in `UpdateSceneState`.)
- [ ] **Step 3 — remove 225 opcodes 80 + 237.** Delete both branches (the shift code
  is now inside 165).
- [ ] **Step 4 — port `CheckScene` + `UpdateSceneState`.** Faithful port of
  `Client.java:3234-3291`. New methods on `*Client`:

```go
// Java: Client.checkScene (Client.java:3259-3291).
func (c *Client) CheckScene() int {
	for i := range len(c.SceneMapLandData) {
		if c.SceneMapLandData[i] == nil && c.SceneMapLandFile[i] != -1 {
			return -1
		}
		if c.SceneMapLocData[i] == nil && c.SceneMapLocFile[i] != -1 {
			return -2
		}
	}
	ready := true
	for i := range len(c.SceneMapLandData) {
		data := c.SceneMapLocData[i]
		if data != nil {
			x := (c.SceneMapIndex[i]>>8)*64 - c.SceneBaseTileX
			z := (c.SceneMapIndex[i]&0xFF)*64 - c.SceneBaseTileZ
			ready = ready && world.CheckLocations(x, z, io.NewPacket(data), c.OnDemand)
		}
	}
	if !ready {
		return -3
	} else if c.AwaitingSync {
		return -4
	}
	c.SceneState = 2
	world.LevelBuilt = c.CurrentLevel
	c.BuildScene()
	return 0
}

// Java: Client.updateSceneState (Client.java:3234-3251).
func (c *Client) UpdateSceneState() {
	if LowMemory && c.SceneState == 2 && world.LevelBuilt != c.CurrentLevel {
		c.AreaViewport.Bind()
		c.FontPlain12.CentreString(151, 0, "Loading - please wait.", 257)
		c.FontPlain12.CentreString(150, 0xFFFFFF, "Loading - please wait.", 256)
		c.presentLoadingMessage()
		c.SceneState = 1
		c.SceneLoadStartTime = <now-millis>
	}
	if c.SceneState == 1 {
		status := c.CheckScene()
		if status != 0 && <now-millis>-c.SceneLoadStartTime > 360000 {
			signlink.ReportError(<args per Java 3248>)
			c.SceneLoadStartTime = <now-millis>
		}
	}
}
```
  (Confirm the exact `World.CheckLocations` signature WS1 added — the agent saw
  `CheckLocations(xOffset, zOffset, src)`; pass `c.OnDemand` if its signature needs
  it. The low-mem block at the top mirrors Java `updateSceneState` 3236-3243; if
  the Go opcode-184 already had this block, move it here.)
- [ ] **Step 5 — split PLAYER_INFO (86).** In the renumbered-to-86 branch (was 184),
  REMOVE the inlined `if c.SceneState == 1 { SceneState=2; BuildScene }` + the
  low-mem/minimap block (9420-9436), and ADD `c.AwaitingSync = false` after
  `c.GetPlayer(...)` (Java 8002-8009). The minimap re-create + low-mem rebuild move
  into `UpdateSceneState`/`CheckScene` (already there per Java).
- [ ] **Step 6 — main-loop hook.** Call `c.UpdateSceneState()` once per frame in the
  game-loop update (Java calls it from the per-cycle update; find where WS1 hooked
  `c.UpdateOnDemand()`/`c.OnDemand.Run()` and call `UpdateSceneState` adjacent,
  guarded by `c.OnDemand != nil && c.SceneMapLandData != nil`).
- [ ] **Step 7 — gate + commit.** build/vet/test/gofmt/lint green.
  `git commit --no-gpg-sign -m "feat(rev-244): REBUILD_NORMAL 165 + scene state machine (WS2 Inc 5)"`

---

### Inc 6: LocChange merge (3f)

**Files:**
- Modify: `pkg/jagex2/dash3d/entity/locchange.go`
- Delete: `pkg/jagex2/dash3d/entity/locmergeentity.go`
- Modify: `pkg/jagex2/dash3d/world/world.go` (add `ChangeLocAvailable`)
- Modify: `pkg/jagex2/client/client.go` (fields; AppendLoc/StoreLoc/UpdateLocChanges/
  ClearLocChanges; ReadZonePacket loc branches; UPDATE_ZONE_FULL_FOLLOWS 131; the
  165 LocChange shift loop from Inc 5; remove `UpdateMergeLocs` + zone-exit restore)
- Test: `pkg/jagex2/dash3d/entity/locchange_test.go`, `…/world/world_test.go` (extend)

- [ ] **Step 1 — rewrite `LocChange`** (Java `dash3d/LocChange.java`). 12 fields,
  `EndTime` defaults -1 (set in a constructor since Go zero-values to 0):

```go
package entity

// Java: jagex2.dash3d.LocChange. Merge of the rev-225 LocChange (old via Last*)
// and LocMergeEntity (LastCycle). endTime=-1 means a permanent change.
type LocChange struct {
	Level    int // Java: level (was Plane)
	Layer    int
	X        int
	Z        int
	OldType  int // Java: oldType (was LastLocIndex)
	OldAngle int
	OldShape int
	NewType  int // Java: newType (was LocIndex)
	NewAngle int
	NewShape int
	StartTime int
	EndTime   int
}

func NewLocChange() *LocChange {
	return &LocChange{EndTime: -1}
}
```
  Delete `locmergeentity.go`.
- [ ] **Step 2 — `World.ChangeLocAvailable`** (Java `World.java:1096-1105`):

```go
// Java: World.changeLocAvailable (World.java:1096-1105).
func ChangeLocAvailable(id, shape int) bool {
	loc := loctype.Get(id)
	if shape == 11 {
		shape = 10
	}
	if shape >= 5 && shape <= 8 {
		shape = 4
	}
	return loc.CheckModel(shape)
}
```
  (`LocType.CheckModel` exists from WS1.)
- [ ] **Step 3 — Client fields.** Replace the two list fields with one:
  `LocChanges *datastruct.LinkList[*entity.LocChange]` (rename `SpawnedLocations`;
  drop `MergedLocations`). Update init (547/585) + lifecycle sites (6745/6753/7049/
  8413) to the single list. `LocChanges` replaces both.
- [ ] **Step 4 — `AppendLoc` + `StoreLoc`** (Java `Client.java:8760-8814`). Add
  methods on `*Client` (note Java param order; map to Go scene getters — the Go
  uses `GetWallBitSet`/`GetWallDecorationBitSet`/`GetLocBitSet`/`GetGroundDecorationBitSet`
  + `GetInfo`, per the current inlined code at client.go:1127-1144):

```go
// Java: Client.appendLoc (Client.java:8760-8784).
func (c *Client) AppendLoc(x, shape, endTime, typ, angle, layer, z, level, startTime int) {
	var loc *entity.LocChange
	for n := c.LocChanges.Head(); n != nil; n = c.LocChanges.Next() {
		v := n.Value
		if v.Level == level && v.X == x && v.Z == z && v.Layer == layer {
			loc = v
			break
		}
	}
	if loc == nil {
		loc = entity.NewLocChange()
		loc.Level = level
		loc.Layer = layer
		loc.X = x
		loc.Z = z
		c.StoreLoc(loc)
		c.LocChanges.AddTail(datastruct.NewLinkable(loc))
	}
	loc.NewType = typ
	loc.NewShape = shape
	loc.NewAngle = angle
	loc.StartTime = startTime
	loc.EndTime = endTime
}

// Java: Client.storeLoc (Client.java:8788-8814). Captures old* from the scene.
func (c *Client) StoreLoc(loc *entity.LocChange) {
	typecode := 0
	otherId, otherShape, otherAngle := -1, 0, 0
	switch loc.Layer {
	case 0:
		typecode = c.Scene.GetWallBitSet(loc.Level, loc.X, loc.Z)
	case 1:
		typecode = c.Scene.GetWallDecorationBitSet(loc.Level, loc.Z, loc.X)
	case 2:
		typecode = c.Scene.GetLocBitSet(loc.Level, loc.X, loc.Z)
	case 3:
		typecode = c.Scene.GetGroundDecorationBitSet(loc.Level, loc.X, loc.Z)
	}
	if typecode != 0 {
		info := c.Scene.GetInfo(loc.Level, loc.X, loc.Z, typecode)
		otherId = (typecode >> 14) & 0x7FFF
		otherShape = info & 0x1F
		otherAngle = info >> 6
	}
	loc.OldType = otherId
	loc.OldShape = otherShape
	loc.OldAngle = otherAngle
}
```
  (Java `getWallTypecode` etc. == the Go `Get*BitSet` names — WS3 kept the rev-225
  Go names. `getDecorTypecode(z,level,x)` arg order → Go `GetWallDecorationBitSet(level,z,x)`;
  verify the existing call at client.go:1131 for the exact Go arg order.)
- [ ] **Step 5 — `UpdateLocChanges`** (Java `Client.java:3539-3568`), replacing
  `UpdateMergeLocs` (client.go:2864). **Watch the Go `AddLoc` arg order**
  `(angle, x, z, layer, id, shape, plane)` vs Java `addLoc(id, x, angle, shape, level, z, layer)`:

```go
// Java: Client.updateLocChanges (Client.java:3539-3568).
func (c *Client) UpdateLocChanges() {
	if c.SceneState != 2 {
		return
	}
	for n := c.LocChanges.Head(); n != nil; n = c.LocChanges.Next() {
		loc := n.Value
		if loc.EndTime > 0 {
			loc.EndTime--
		}
		if loc.EndTime != 0 {
			if loc.StartTime > 0 {
				loc.StartTime--
			}
			if loc.StartTime == 0 && (loc.NewType < 0 || world.ChangeLocAvailable(loc.NewType, loc.NewShape)) {
				c.AddLoc(loc.NewAngle, loc.X, loc.Z, loc.Layer, loc.NewType, loc.NewShape, loc.Level)
				loc.StartTime = -1
				if loc.NewType == loc.OldType && loc.OldType == -1 {
					n.Unlink()
				} else if loc.NewType == loc.OldType && loc.NewAngle == loc.OldAngle && loc.NewShape == loc.OldShape {
					n.Unlink()
				}
			}
		} else if loc.OldType < 0 || world.ChangeLocAvailable(loc.OldType, loc.OldShape) {
			c.AddLoc(loc.OldAngle, loc.X, loc.Z, loc.Layer, loc.OldType, loc.OldShape, loc.Level)
			n.Unlink()
		}
	}
	CycleLogic5++ // Java: cyclelogic5++ (3570)
}
```
  (Confirm `CycleLogic5` exists; the old `UpdateMergeLocs` had it.) Update the
  per-frame caller from `UpdateMergeLocs()` to `UpdateLocChanges()`.
- [ ] **Step 6 — `ClearLocChanges`** (Java `Client.java:3431-3442`), replacing the
  zone-exit restore at client.go:10110:

```go
// Java: Client.clearLocChanges (Client.java:3431-3442). Re-arm permanent changes
// against the freshly-loaded scene; drop timed ones.
func (c *Client) ClearLocChanges() {
	for n := c.LocChanges.Head(); n != nil; n = c.LocChanges.Next() {
		loc := n.Value
		if loc.EndTime == -1 {
			loc.StartTime = 0
			c.StoreLoc(loc)
		} else {
			n.Unlink()
		}
	}
}
```
  Find where the 225 zone-exit loop (10110) was driven from (the opcode-129/zone-load
  path) and call `c.ClearLocChanges()` there instead (Java calls it from the
  zone-load handler ~3379).
- [ ] **Step 7 — rewire ReadZonePacket loc branches.** Replace the inlined
  LOC_ADD/LOC_DEL body (client.go:1100-1159) with the Java form (`Client.java:8476-8496`):

```go
if arg2 == io.SERVERPROT_LOC_ADD_CHANGE || arg2 == io.SERVERPROT_LOC_DEL {
	pos := arg1.G1()
	x := ((pos >> 4) & 0x7) + c.BaseX
	z := (pos & 0x7) + c.BaseZ
	info := arg1.G1()
	shape := info >> 2
	angle := info & 0x3
	layer := c.LOC_SHAPE_TO_LAYER[shape]
	id := -1
	if arg2 != io.SERVERPROT_LOC_DEL {
		id = arg1.G2()
	}
	if x >= 0 && z >= 0 && x < 104 && z < 104 {
		c.AppendLoc(x, shape, -1, id, angle, layer, z, c.CurrentLevel, 0)
	}
}
```
  Replace the LOC_MERGE body (client.go:1303-1362) with the Java form
  (`Client.java:8661-8730`): ONE `AppendLoc(x, 0, end+1, -1, 0, layer, z, level, start+1)`
  call (NOT two merge entities) + the player loc-model attach
  (`LocStartCycle/LocStopCycle/LocModel/LocOffset*`/min-max tile box). Keep the
  existing player-attach code (1330-1361); replace only the two `NewLocMergeEntity`
  pushes (1326-1329) with the single `AppendLoc`. Read `start := arg1.G2()` and
  `end := arg1.G2()` as raw values; pass `start+1`/`end+1` to AppendLoc and
  `clientextras.LoopCycle+start`/`+end` to the player cycles (per Java 8697-8700).
- [ ] **Step 8 — UPDATE_ZONE_FULL_FOLLOWS (131) loc body.** The 225 handler (was
  135) re-added spawned locs via AddLoc then unlinked; 244 instead sets
  `loc.EndTime = 0` on matching LocChanges in the cleared 8×8 region (Java
  `Client.java` UPDATE_ZONE_FULL_FOLLOWS). Quote the Java and port that loop.
- [ ] **Step 9 — fix the Inc-5 165 shift loop.** Point the LocChange shift loop
  inside the 165 handler at `c.LocChanges` (it was a placeholder on
  `SpawnedLocations` in Inc 5).
- [ ] **Step 10 — tests.**
  - `locchange_test.go`: `NewLocChange()` has `EndTime == -1`, others 0.
  - A pure `UpdateLocChanges` countdown harness is hard without a Client/scene; if
    feasible, extract the countdown math into a testable pure helper, else
    build-gate + review. At minimum, a `world_test.go` for `ChangeLocAvailable`
    shape-normalization (11→10, 5-8→4) over a fake LocType recorder.
- [ ] **Step 11 — gate + commit.** Green.
  `git commit --no-gpg-sign -m "feat(rev-244): LocChange merge (LocAdd+LocMerge→one) (WS2 Inc 6)"`

---

### Inc 7: B1 — config-getter `NewModel1`→`TryGet` sweep

**Files:**
- Modify: `pkg/jagex2/config/objtype/objtype.go` (GetInterfaceModel 311-342; GetWornModel
  443-483; GetHeadModel 485-509; ADD `GetInvModel`)
- Modify: `pkg/jagex2/config/npctype/npctype.go` (GetSequencedModel 190-230; GetHeadModel 232-252)
- Modify: `pkg/jagex2/config/loctype/loctype.go` (GetModel ~293)
- Modify: `pkg/jagex2/config/spotanimtype/spotanimtype.go` (GetModel ~99)
- Modify: `pkg/jagex2/config/idktype/idktype.go` (GetModel ~82; GetHeadModel ~101)
- Modify draw sites that don't nil-check (e.g. `objtype.go:387` GetIcon)
- Test: extend the relevant `*_test.go`

- [ ] **Step 1 — single-model cache-backed getters: `NewModel1`→`TryGet` + early-nil.**
  In `ObjType.GetInterfaceModel`, `SpotAnimType.GetModel`, and the `LocType.GetModel`
  static-cache branch (293): replace `model.NewModel1(id)` with `model.TryGet(id)`
  and add the Java early-exit right after:
```go
m := model.TryGet(id)
if m == nil {
	return nil
}
```
  Remove the eager-getter breadcrumb at `objtype.go:327`.
- [ ] **Step 2 — multi-model getters: `NewModel1`→`TryGet` (no per-id nil check).**
  In `ObjType.GetWornModel`/`GetHeadModel`, `IdkType.GetModel`/`GetHeadModel`:
  swap each `NewModel1`→`TryGet` (Java doesn't null-check these; the caller's
  `check*` gates them). Match Java `getWearModel`/`getHeadModel`/`IdkType` bodies.
- [ ] **Step 3 — NpcType: add the missing `Model.request` precheck.**
  `NpcType.GetSequencedModel` (190) + `GetHeadModel` (232) currently lack Java's
  precheck. Add it (Java `NpcType.java:241-250`/302-310), inverted-flag semantics:
```go
ready := false
for i := 0; i < len(t.Models); i++ {
	if !model.Request(t.Models[i]) {
		ready = true
	}
}
if ready {
	return nil
}
```
  then build via `model.TryGet`.
- [ ] **Step 4 — add `ObjType.GetInvModel`** (Java `ObjType.getInvModel` 445-468;
  type-4 path used by Component `loadModel` in Inc 8). It's `GetInterfaceModel`
  without the scale/normals/cache — just countobj redirect + `TryGet` + early-nil +
  recolour. Add as a new method returning `*model.Model`.
- [ ] **Step 5 — nil-guard draw sites.** Audit getter callers that don't nil-check
  (notably `objtype.GetIcon` at 387 → guard the `GetInterfaceModel(1)` result before
  `DrawSimple`). LocModel + Component draw sites already nil-check.
- [ ] **Step 6 — test.** Extend `objtype`/`model` tests: with `model.Metadata`
  un-init (so `TryGet` returns nil), a getter returns nil instead of the
  `"Error model:N not found!"` empty-model path. (Spot-check one getter.)
- [ ] **Step 7 — gate + commit.** Green.
  `git commit --no-gpg-sign -m "feat(rev-244): config getters NewModel1→TryGet lazy load (WS2 Inc 7/B1)"`

---

### Inc 8: B2 — Component type-6 deferred model

**Files:**
- Modify: `pkg/jagex2/config/component/component.go` (fields 62-63; decode 251-259;
  GetModel 317-340; package GetModel 362-370; ModelCache 91)
- Modify: `pkg/jagex2/client/client.go` (write sites 9689, 10001, 10157, 10220, 5289)
- Test: `pkg/jagex2/config/component/component_test.go` (extend)

- [ ] **Step 1 — fields → 4 ints.** Replace `Model *model.Model` / `ActiveModel
  *model.Model` (62-63) with `ModelType int`, `Model int`, `ActiveModelType int`,
  `ActiveModel int` (defaults 0; matches Java/TS). Remove the breadcrumbs.
- [ ] **Step 2 — type-6 decode** (Java `Component.java:361-372`): set the int pairs
  instead of building eagerly:
```go
if var8.Type == 6 {
	var7 = var4.G1()
	if var7 != 0 {
		var8.ModelType = 1
		var8.Model = ((var7 - 1) << 8) + var4.G1()
	}
	var7 = var4.G1()
	if var7 != 0 {
		var8.ActiveModelType = 1
		var8.ActiveModel = ((var7 - 1) << 8) + var4.G1()
	}
	// ... Anim/ActiveAnim/Zoom/Xan/Yan unchanged ...
}
```
- [ ] **Step 3 — `LoadModel(type, id, localPlayer)`** (Java `Component.loadModel`
  492-515). Thread `localPlayer` as a param (TS approach — avoids a
  config→entity import cycle). Cache key `(int64(type)<<16)+id`:
```go
// Java: Component.loadModel (Component.java:492-515).
func (c *Component) LoadModel(typ, id int, localPlayer *playerentity.ClientPlayer) *model.Model {
	key := (int64(typ) << 16) + int64(id)
	if m := ModelCache.Get(key); m != nil {
		return m
	}
	var m *model.Model
	switch typ {
	case 1:
		m = model.TryGet(id)
	case 2:
		m = npctype.Get(id).GetHeadModel()
	case 3:
		m = localPlayer.GetHeadModel()
	case 4:
		m = objtype.Get(id).GetInvModel(50)
	case 5:
		m = nil
	}
	if m != nil {
		ModelCache.Put(key, m)
	}
	return m
}
```
  (If importing `npctype`/`objtype`/`playerentity` into `component` creates a
  cycle, pass resolver funcs instead — check the import graph; the Go ConfigType
  packages may already be leaf-ward. `objtype.GetInvModel` is from Inc 7.)
- [ ] **Step 4 — method `GetModel`** (317-340): resolve via `LoadModel` + thread the
  `localPlayer` param through:
```go
func (c *Component) GetModel(arg0, arg1 int, arg2 bool, localPlayer *playerentity.ClientPlayer) *model.Model {
	var m *model.Model
	if arg2 {
		m = c.LoadModel(c.ActiveModelType, c.ActiveModel, localPlayer)
	} else {
		m = c.LoadModel(c.ModelType, c.Model, localPlayer)
	}
	if m == nil {
		return nil
	}
	// ... existing transform/normals tail unchanged ...
}
```
- [ ] **Step 5 — `CacheModel`** (Java 518-523) + fix `ModelCache` key/size. Add
  `func CacheModel(m *model.Model, id, typ int)` (clear; put if non-nil and type!=4,
  key `(int64(typ)<<16)+id`). Change `ModelCache` size 50000→30 (91) and re-key the
  package helper by `(type<<16)+id` (or fold the old package `GetModel` into
  `LoadModel`).
- [ ] **Step 6 — client.go write sites** (now renumbered by Inc 2; set int pairs):
  - opcode 245 IF_SETMODEL (was 87, 10157): `comp.ModelType = 1; comp.Model = id`
    (drop `model.NewModel1`).
  - opcode 129 IF_SETNPCHEAD (was 204, 10001): `comp.ModelType = 2; comp.Model = npcId`.
  - opcode 108 IF_SETPLAYERHEAD (was 197, 9689): `comp.ModelType = 3; comp.Model =
    <encoded>` (quote Java `Client.java:7995-7996` for the exact encoded value).
  - opcode 164 IF_SETOBJECT (was 46, 10220): `comp.ModelType = 4; comp.Model = objId`
    (drop the eager `GetInterfaceModel(50)`).
  - clientCode 327 player-design (5289): `comp.ModelType = 5; comp.Model = 0;
    component.CacheModel(builtModel, 0, 5)`.
  - (opcode 103 recolor was REMOVED in Inc 2 — no ripple here.)
  Thread `c.LocalPlayer` into the `GetModel(...)` draw call (client.go:3756/3759).
- [ ] **Step 7 — test.** `component_test.go`: a type-6 decode sets `ModelType==1` +
  the encoded `Model` id (no eager build); `LoadModel(5, …)` returns nil; `LoadModel`
  caches by `(type<<16)+id`.
- [ ] **Step 8 — gate + commit.** Green.
  `git commit --no-gpg-sign -m "feat(rev-244): Component type-6 deferred model + loadModel (WS2 Inc 8/B2)"`

---

## Milestone — host smoke test (post-Inc 8)

Run the real client against an Engine-TS 244 server (host-only; no headless).
Verify: connect → login (StaffModLevel) → REBUILD_NORMAL loads maps via OnDemand →
scene builds → terrain + locs + npc/obj/player models render. Record the outcome
(what rendered / what didn't) in this doc's status header + a resume note.

## Risk / verification
- **No runtime gate in sandbox.** Every increment build/vet/test/gofmt/lint-green +
  committed; correctness of the renumber is gated by Inc 1's independent cross-check
  + Inc 2/3's per-handler byte review; the host smoke test is the runtime gate.
- **Highest-risk translations** (re-read Java, add `// Java:` refs): `SERVERPROT_LENGTH`
  contents (index 242=6); the renumber correspondence (match by body, not number —
  collision traps); the login `reply==0` wrapper + `loginServer` shift/mask; the 165
  region-grid `/8` integer division + the delta-shift lift; the `CheckScene`/
  `awaitingSync` sequencing; the LocChange `AddLoc` arg-order mismatch + `start+1`/
  `end+1`; `ChangeLocAvailable` shape normalization; B1 nil-handling ripple; B2
  import-cycle (thread `localPlayer`, don't import).
- **Deferred (noted, not in WS2):** chat crowns + `imageModIcons`; opcode-103
  confirm-then-remove; IF_OPENOVERLAY render; WS5 audio; UI polish; WS3 cleanups.

## Self-review notes
- **Spec coverage:** opcode renumber ✔(Inc1/2/3), login+staffmodlevel ✔(Inc4),
  REBUILD+state-machine ✔(Inc5), LocChange merge ✔(Inc6), B1 ✔(Inc7), B2 ✔(Inc8).
  Crowns moved to deferred (flagged) — design called them in-scope; descoped due to
  the `imageModIcons` dependency surfaced in research.
- **Type consistency:** const names `SERVERPROT_*`/`CLIENTPROT_*` used identically
  in Inc1/2/3/5/6; `LocChange` fields (`Level/Layer/X/Z/Old*/New*/StartTime/EndTime`)
  consistent across Inc5(shift)/Inc6; `AppendLoc/StoreLoc/UpdateLocChanges/
  ClearLocChanges/ChangeLocAvailable` consistent; `CheckScene/UpdateSceneState/
  AwaitingSync/WithinTutorialIsland/SceneLoadStartTime` consistent Inc5↔Inc6;
  `LoadModel/CacheModel/ModelType/ActiveModelType` consistent Inc8; `TryGet`/
  `Request`/`GetInvModel` consistent Inc7↔Inc8.
- **Confirm at impl time:** the millis source for `SceneLoadStartTime`; `World.CheckLocations`
  exact signature (WS1); `CycleLogic5` existence; the Go `Get*BitSet` arg orders in
  `StoreLoc`; the config→entity import graph for B2 (`localPlayer` threading); the
  MIDI_SONG/MIDI_JINGLE exact Java bodies; opcode-108 encoded player-head model value.
