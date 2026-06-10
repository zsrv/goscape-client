# Reference Sources

The upstream sources each Go revision was ported **from**. Branch names move, so
the **commit hash is the real pin** — treat this file like a lockfile for the
port. To port a new revision, diff the new reference commit against the commit
recorded here for the revision you branch from (see the "Porting workflow"
section of `PORTING-LESSONS.md`).

Local working-copy paths are machine-specific and live in `CLAUDE.local.md`
(not committed); only the portable URL / branch / commit belong here.

## rev-225 — Go branch `rev-225`

| Repo | Role | URL | Branch | Pinned commit |
|---|---|---|---|---|
| Client-Java | **primary** — authoritative translation source; every Go change maps to a Java function | https://github.com/LostCityRS/Client-Java | `225-clean` | `cc3781de9e45265c52711dca850cd154f03c3a2c` |
| Client-TS | secondary cross-check for ambiguous Java→Go translations | https://github.com/LostCityRS/Client-TS | `225` | `8e0fca6d1b01cee8e1f23603ddc78cf009a6ce38` |
| Engine | engine reference | https://github.com/LostCityRS/Engine | `main` | `5b5584280d910511ac5635e1025b9fd2912a8264` |
| Engine-TS | engine reference (TypeScript) | https://github.com/LostCityRS/Engine-TS | `225` | `e1dea19f256c7ff1a89d47024c811c755ad2184d` |
| Content | game content reference | https://github.com/LostCityRS/Content | `225` | `9901aa27b60198afac49012f45f32e4eb4d5c012` |
| Server | protocol cross-check (reference only — no code reuse) | https://github.com/LostCityRS/Server | `main` | `326bb4a3b24fbf7a1bf503ec598a4c2cab118ee1` |

(Commits captured 2026-05-23. `Engine` / `Server` track moving `main` branches —
the pinned commit is what the rev-225 port corresponds to, regardless of where
those branches have since moved.)

## rev-244 — Go branch `rev-244`

| Repo | Role | URL | Branch | Pinned commit |
|---|---|---|---|---|
| Client-Java | **primary** — authoritative translation source; every Go change maps to a Java function | https://github.com/LostCityRS/Client-Java | `244` | `01f1608842acb12901f7e4f3df25553f641cc86e` |
| Client-TS | secondary cross-check for ambiguous Java→Go translations | https://github.com/LostCityRS/Client-TS | `244` | `1cfb57bff1a4a5dc9ca36cdbe76a302fed4fa532` |
| Engine-TS | engine reference (TypeScript) | https://github.com/LostCityRS/Engine-TS | `244` | `9aadcec4e9560b810b5e5eee31aadc67f3b206cd` |
| Content | game content reference | https://github.com/LostCityRS/Content | `244` | `e5d0282e03b383efd3b2a81e63090e703ffb5399` |

(Commits captured 2026-06-02. Go branch `rev-244` is cut from `rev-225`.
Engine (Java) and Server — pinned at `main` for rev-225 — were **not** supplied
for rev-244; record them here if/when a 244-specific need arises, otherwise the
rev-225 pins remain the last-known reference.)

> **⚠ Deob-lineage divergence — read before treating the diff as the work list.**
> The `244` branch is a *different deobfuscation lineage* from `225-clean`, not a
> linear continuation. A raw `git diff cc3781de..01f16088` is ~42 000 changed
> lines that are mostly **not** the 225→244 game delta:
> - Every `@ObfuscatedName` key was reassigned by the obfuscator (e.g. `NpcType`
>   `bc`→`gc`) — mechanical, behaviourally irrelevant (~3 800 lines).
> - Classes are renamed and the `dash3d/entity/` + `dash3d/type/` sub-packages
>   are flattened: `PathingEntity`→`ClientEntity`, `SpotAnimEntity`→`MapSpotAnim`,
>   `GroundObject`→`GroundDecor`, `Location`→`QuickGround`, `Wall`→`Sprite`,
>   `graphics/Model`→`dash3d/Model`, `deob/client.java`→`jagex2/client/Client.java`.
> - **The `225-clean` names that the Go rev-225 port mirrors do not exist in the
>   `244` tree**, and Client-TS `244` uses the *same new convention* as Java `244`.
>
> Consequence: the standard "Java diff = change-for-change Go work list" step in
> `PORTING-LESSONS.md` §2 cannot be applied mechanically here. Pair files with
> rename detection (`git diff -M20% -w`) and filter `@ObfuscatedName` churn to
> recover the real delta. Git's low-% rename pairings are unreliable — verify by
> reading both trees + Client-TS — and beware **name reuse** (`type/Wall`→`Sprite`
> while a *new, unrelated* `Wall` is added; same for `Ground`).
>
> **Agreed strategy (2026-06-02): adopt 244's names/structure.** The Go `rev-244`
> branch does a mechanical rename/restructure pass *first* — realigning to the
> Java/TS-244 vocabulary (`ClientEntity`, `Sprite`, `dash3d/Model`, flattened
> `dash3d` packages, monolith split into `jagex2/client/Client.java`) — and only
> *then* applies the 225→244 game-logic delta on top. The Go↔Java mapping stays
> 1:1 for this and later revisions. See `PORTING-LESSONS.md` §2 "When deob
> lineages diverge".

## rev-245.2 — Go branch `rev-245.2`

| Repo | Role | URL | Branch | Pinned commit |
|---|---|---|---|---|
| Client-Java | **primary** — authoritative translation source; every Go change maps to a Java function | https://github.com/LostCityRS/Client-Java | `245.2` | `176a85f7b423111c878a476e1ead048745e377c0` |
| Client-TS | secondary cross-check for ambiguous Java→Go translations | https://github.com/LostCityRS/Client-TS | `245.2` | `bd29ce0127e1810a0b5bba43bc143461ce0ee4a1` |
| Engine-TS | engine reference (TypeScript) + smoke-test server | https://github.com/LostCityRS/Engine-TS | `245.2` | `3c16994ca4ba51b4e04f88316c1f7395b0c4bb8a` |
| Content | game content reference | https://github.com/LostCityRS/Content | `245.2` | `cbcfe6706ef9f4093e5b8e4c9cfee93577346993` |

(Commits captured 2026-06-04. Go branch `rev-245.2` is cut from `rev-244`.
`clientversion = 245`; the branch is named `245.2` upstream and the Go branch
matches it. As with rev-244, the Client-Java working tree sits on a
`245.2-GOSCAPE` branch that may accumulate local edits — **always read
references via `git show 176a85f:…`**, never the working tree.)

> **Lineage note — same convention, fresh deob with a cleanup pass.** The
> `245.2` branch is a *new* 13-commit deob (`feat: Initial deob`…) with **no
> shared git history** with `244`, but it **adopts the same naming convention**
> (`ClientEntity`, `primarySeqId`, same `jagex2/` layout; 72 vs 73 files) — so
> no rename/restructure pass is needed this time. The raw
> `git diff -M20% -w 01f16088..176a85f` is still ~27 000 +/- lines, dominated
> by: reassigned `@ObfuscatedName` keys (~2 600 lines), **`Client.java` method
> reordering** (`chore: Reordered Client class methods` — defeats raw line
> diffing; pair methods by name), local/param renames and *reorders*
> (`var5`→`dx`; `move(boolean teleport, int, int)`→`move(int x, int z, boolean
> jump)`), and blank-line reflow. Real deltas confirmed inside the churn:
> `postanim_mode`→`postanim_move`, "New frame logic", packet identification,
> `clientversion = 245`.
>
> **signlink reshape:** `jagex2/client/sign/SignLink.java` (477 ln) +
> `MidiPlayer.java` → single top-level `sign/signlink.java` (308 ln, the
> authentic original package name). The 244 deob's wrapper-side audio-consumer
> reconstruction (audioLoop/MidiPlayer) is **absent** in 245.2, and
> `midivol/wavevol` lost their `= 96` initializers — the Go consumer
> (`audio/audioloop.go`, WS5) becomes a documented Go-side seam to reconcile
> against the new publisher shape, not a line-port target.
>
> **Naming policy (user decision 2026-06-04): adopt-in-touched-methods.**
> Methods rewritten for the 245.2 delta adopt the new local/param names (with
> `// Java:` refs); untouched methods keep their current names. The Go↔Java
> mapping converges as the delta lands, without a churn-only rename sweep.

## rev-254 — Go branch `rev-254`

| Repo | Role | URL | Branch | Pinned commit |
|---|---|---|---|---|
| Client-Java | **primary** — authoritative translation source; every Go change maps to a Java function | https://github.com/LostCityRS/Client-Java | `254` | `2e629784c3dcb671ee3aab134f9cb91d614d8094` |
| Client-TS | secondary cross-check for ambiguous Java→Go translations | https://github.com/LostCityRS/Client-TS | `254` | `d340fc258c8d3becb1b7680793415621b40064e2` |
| Engine-TS | engine reference (TypeScript) + smoke-test server | https://github.com/LostCityRS/Engine-TS | `254` | `2e3bcf4392200e84dd15ce67008c5d41fa4537aa` |
| Content | game content reference | https://github.com/LostCityRS/Content | `254` | `caee3f2eb3eb3df60126e2be88c436dc2dc98e43` |

(Commits captured 2026-06-04; Engine-TS pin advanced 2026-06-10 — a
fast-forward of branch `254` from `43e02957f355` to `2e3bcf439220`, with
`254 ≡ 254-GOSCAPE` re-verified at the new pin; the 2026-06-05 host smoke
test ran against the prior pin. Go branch `rev-254` is cut from `rev-245.2`
@ `05a7659`. All four local repos sit on `254-GOSCAPE` working branches —
verified `254 ≡ 254-GOSCAPE` at capture time, but **always read references
via `git show 2e62978:…`**, never the working tree.)

> **Lineage note — fresh deob, webclient-research names, argN regression.**
> `254` shares no git history with `245.2`. Same `jagex2/` layout, but with
> **class-level renames** incl. a chained name-reuse trap (obf-key verified):
> `World`→`ClientBuild` (obf `c`) and `World3D`→`World` (obf `s`) — "254
> `World`" ≠ "245.2 `World`"; plus `Component`→`IfType`, `Jagfile`→`JagFile`,
> `io/Protocol`→`client/Protocol`. Many locals/params REGRESSED to `argN`.
> `@ObfuscatedName` keys are reassigned at member level AND can rotate at
> class level (three 254 keys re-bind to different classes: `oc`/`pc`/`qc`) —
> pair classes by key+name agreement with structural adjudication, never by
> key alone or name alone. New surface: `config/VarBitType`, `client/Stats`.
> Full opcode renumbering expected across the 9-revision gap. See
> `RENAME-MAP.md` and `PORT-DESIGN-254.md` on `rev-254`.
>
> **Naming policy (user decision 2026-06-04): hybrid.** Go adopts 254
> class/file-level names via an up-front mechanical rename pass; where 254
> regressed locals/params to `argN`, Go keeps existing descriptive names
> (annotated `// Java: argN`); adopt 254 local names where equal-or-better.

## rev-274 — Go branch `rev-274`

| Repo | Role | URL | Branch | Pinned commit |
|---|---|---|---|---|
| Client-Java | **primary** — authoritative translation source; every Go change maps to a Java function | https://github.com/LostCityRS/Client-Java | `274` | `32f30626156783de9f142306eb73a2243909dacf` |
| Client-TS | secondary cross-check for ambiguous Java→Go translations | https://github.com/LostCityRS/Client-TS | `274` | `b67894260fb06ae6162ed3a8adab506abcd7faa9` |
| Engine-TS | engine reference (TypeScript) + smoke-test server | https://github.com/LostCityRS/Engine-TS | `274` | `4c0d036f940c8c7e11b0ed714dcf40c88f9de200` |
| Content | game content reference | https://github.com/LostCityRS/Content | `274` | `85d62c8bdb9005cc02a784a31337c7df052d6469` |

(Commits captured 2026-06-05. Go branch `rev-274` is cut from `rev-254` @
`a94d3fc`. All four local repos sit on `274-GOSCAPE` working branches —
verified `274 ≡ 274-GOSCAPE` (0/0 ahead-behind) at capture time, but
**always read references via `git show 32f3062:…`**, never the working
tree. Client-Java `274` has been quiet since 2026-03-20.)

> **Lineage note — fresh deob, disjoint history (third occurrence).** `274`
> shares no git history with `254` ("Regenerated deob" in its log); the raw
> tree diff is ~20.2k(+)/20.4k(−) lines across 83 files, dominated by churn.
> Same `jagex2/` layout; path-level candidates (obf-key verification in
> `RENAME-MAP-274.md` on `rev-274`): renames `Wave→JagFX`,
> `VertexNormal→PointNormal`, `Stats→Skill`, `SpotAnimType→SpotType`,
> `VarBitType→VarbitType`, `DoublyLinkable→Linkable2`,
> `DoublyLinkList→LinkList2`, package `wordenc→wordfilter`; moves
> `ClientBuild` `dash3d→client` and `Protocol` `client→io` (**a revert of
> 254's move** — deltas are not monotonic). New: `sound/Filter.java` (synth
> IIR filter). Removed: `client/InputTracking.java` (0 refs in 274
> `Client.java`). `Client.java` shrank 350,364→323,443 bytes across the
> 20-revision gap; full wire-opcode renumbering expected. See
> `PORT-DESIGN-274.md` on `rev-274`.
>
> **Naming policy (user decision 2026-06-05): hybrid, same as 254.** Go
> adopts 274 class/file-level names via an up-front mechanical rename pass;
> where 274 keeps `argN` locals (~886 in `Client.java`, unchanged from
> 254), Go keeps existing descriptive names (annotated `// Java: argN`);
> adopt 274 local names where equal-or-better.

## Future revisions

When porting revision *N*:

1. Add a `## rev-N` section below recording the reference commits used.
2. Branch the Go code `rev-N` from `rev-225` (or the nearest prior revision).
3. Diff the primary reference across the gap —
   `git -C Client-Java diff cc3781de..<rev-N commit>` — and apply the
   corresponding Go deltas on the `rev-N` branch, so the Go branch diff mirrors
   the Java revision diff.
