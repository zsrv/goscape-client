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

## Future revisions

When porting revision *N*:

1. Add a `## rev-N` section below recording the reference commits used.
2. Branch the Go code `rev-N` from `rev-225` (or the nearest prior revision).
3. Diff the primary reference across the gap —
   `git -C Client-Java diff cc3781de..<rev-N commit>` — and apply the
   corresponding Go deltas on the `rev-N` branch, so the Go branch diff mirrors
   the Java revision diff.
