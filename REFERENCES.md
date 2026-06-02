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

## Future revisions

When porting revision *N*:

1. Add a `## rev-N` section below recording the reference commits used.
2. Branch the Go code `rev-N` from `rev-225` (or the nearest prior revision).
3. Diff the primary reference across the gap —
   `git -C Client-Java diff cc3781de..<rev-N commit>` — and apply the
   corresponding Go deltas on the `rev-N` branch, so the Go branch diff mirrors
   the Java revision diff.
