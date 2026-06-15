# PORT-DESIGN-274.md — Rev-274 Port Design

Approved 2026-06-05. Branch `rev-274`, cut from `rev-254` @ `a94d3fc`.
Companion documents (produced by later phases): `RENAME-MAP-274.md`,
`LOGIC-DELTA-SCOPE-274.md`, `PARITY-AUDIT-<date>.md`.

## §0 Decisions log (user, 2026-06-05)

1. **Approach: established playbook (254-style).** Rename map → mechanical
   rename pass → scope recovery → subsystem workstreams → full parity audit →
   fix pass → host smoke. Rejected alternatives: fresh full translation
   (discards four revisions of verified Go work plus everything not in the
   Java reference — platform seam, WebSocket transport, wasm, CLI flags);
   direct diff-port without a scope document (fails on fresh-deob gaps per
   `PORTING-LESSONS.md` §2 — churn dominance, method reordering, opcode
   collisions).
2. **Naming policy: hybrid, same as 254.** Adopt 274 class/file-level names
   and package moves via an up-front mechanical rename pass; keep existing
   descriptive Go locals/params where 274 still uses `argN` (~886 sites,
   annotated `// Java: argN` where touched); adopt 274 local names where
   equal-or-better. All `// Java:` cites in touched code move to `@32f3062`.

## §1 Scope & references

Port the Go client from rev-254 parity to **rev-274 parity**, faithful 1:1
against the pinned Java reference. Definition of done matches prior
revisions: all workstreams landed, parity audit findings fixed, host smoke
passed vs Engine-TS 274.

Pins (to be recorded in `REFERENCES.md` on `main`; **always read references
via `git show <pin>:…`**, never the working tree — all four local repos sit
on `274-GOSCAPE` working branches):

| Repo | Role | Branch | Pinned commit |
|---|---|---|---|
| Client-Java | **primary** — authoritative translation source | `274` | `32f30626156783de9f142306eb73a2243909dacf` |
| Client-TS | secondary cross-check | `274` | `b67894260fb06ae6162ed3a8adab506abcd7faa9` |
| Engine-TS | engine reference + smoke-test server | `274` | `4c0d036f940c8c7e11b0ed714dcf40c88f9de200` |
| Content | game content reference | `274` | `85d62c8bdb9005cc02a784a31337c7df052d6469` |

(Captured 2026-06-05; `274 ≡ 274-GOSCAPE` verified 0/0 ahead-behind in all
four repos. Client-Java `274` quiet since 2026-03-20.)

### Lineage characterization (verified 2026-06-05)

- **Fresh deob, disjoint history.** No merge-base between `254-GOSCAPE` and
  `274-GOSCAPE`; the 274 log starts at "Regenerated deob". Raw tree diff
  ~20,212(+)/20,417(−) lines across 83 files — dominated by churn, NOT a
  work list.
- **74 Java files on each side, but not the same 74.** Suspected renames
  (P1 verifies by obf-key + structural adjudication, never name alone):
  `Wave→JagFX` (118-line diff under rename), `VertexNormal→PointNormal`,
  `Stats→Skill`, `SpotAnimType→SpotType`, `VarBitType→VarbitType`,
  `DoublyLinkable→Linkable2`, `DoublyLinkList→LinkList2`; package
  `wordenc→wordfilter`; `ClientBuild` moved `dash3d→client`.
- **Genuinely new:** `sound/Filter.java` — synth IIR filter (rev-270-era
  audio upgrade), pairs with the `JagFX` rework and a 288-line `Tone` diff.
- **Genuinely removed:** `InputTracking` — file gone, 0 references in 274's
  `Client.java`.
- **`Client.java` shrank** 350,364→323,443 bytes across a 20-revision gap
  (19,457-line diff) — the delta is not purely additive; expect removals
  and reverts (deltas are not monotonic — the 254 crown-strip lesson).
- **`argN` density unchanged** (884→886 in `Client.java`) — the regressed
  locals persist, hence the hybrid naming policy.
- **Full wire-opcode renumbering expected** across the gap, with collisions
  against old values of different messages.

## §2 Kickoff (first session(s))

- **P0 Setup** — branch cut (done with this commit); record pins in
  `REFERENCES.md` under `## rev-274` on `main` with a lineage note; delete
  the untracked `audit-225/`, `audit-245/`, `audit-254/` scratch dirs
  (sanctioned by the rev-254 close-out note).
- **P1 Rename map** — `RENAME-MAP-274.md`: pair all 74×74 classes by
  obf-key + name + structural adjudication (254 proved obf keys also rotate
  at class level — never pair by key alone or name alone). Verify the
  suspected-rename list, the `ClientBuild` package move, `Filter` as
  genuinely-new, `InputTracking` as genuinely-removed. Resolve the
  `ClientBuild` Go-package placement (see §6 risk 1).
- **P2 Mechanical rename pass** — hybrid policy, no behavior change, all
  gates green per commit. Go package moves: `wordenc→wordfilter`,
  `vertexnormal→pointnormal`, `wave→jagfx`, `Stats→Skill`, `SpotType`,
  `VarbitType`, datastruct renames, etc. **Post-sed string-literal grep is
  mandatory** (word-boundary seds corrupt string literals — the "…world."
  login-message lesson): after every rename sed, grep quoted strings for
  the NEW name and diff literals vs the pinned reference. Branch docs
  (`CLAUDE.md` revision line + package-layout table, `README.md`) update
  alongside the renames they describe, in the same commits.

## §3 Scope workflow (delta distillation)

**P3 Scope recovery** — produce `LOGIC-DELTA-SCOPE-274.md` from a
method-paired multi-agent comparison of Java `2e62978` (254) vs `32f3062`
(274), with adversarial verification of every claimed delta (the rev-245.2
method, also used for 254):

- Pair methods **by name within paired classes** (after P1's class map);
  verify name-set match per class first; compare bodies per pair.
- Filter `@ObfuscatedName` churn, blank-line reflow, and local renames;
  treat undo-hunks as real delta, never as noise.
- Re-derive **all** opcode tables (outbound/inbound/zone) from the 274
  reference by full enumeration — never by diff, never carried forward.
- Treat scope-doc claims as strong hints, not gospel — prior revisions
  refuted at least one claim during execution.
- The 20-revision gap (vs 9 for 254) means a larger real delta; the scope
  doc groups it into workstreams.

## §4 Execution model (later sessions)

- **P4 Workstreams** — subsystem-grouped, defined by P3. Predicted seeds:
  sound engine (Filter + JagFX + Tone + envelope ripples, incl. the
  signlink audio seam), InputTracking removal, protocol/opcode renumbering,
  config types, renderer/model/world, client logic, shell/misc.
- Sessions stay discrete /clear-able chunks with date-prefixed resume notes
  in `.claude/resume/`, handed off by path.
- Faithful 1:1 translation per `PORTING.md` §2 philosophy and
  `PORTING-LESSONS.md` §3 gotchas; cite `Client.java:NNNN @32f3062`.
- Compensated pairs land both halves in one commit; when one end of any
  protocol-like pair changes, grep for the other end before committing
  (pusher↔dispatcher lesson). Standing compensated pairs
  (`AddChat`/`ZonePacket` arg orders) remain documented, not "fixed".
- Per-commit gates: `go build ./...`, `go vet ./...`, `go test -race ./...`,
  `gofmt`, `golangci-lint`, wasm build (`GOOS=js GOARCH=wasm go build`) —
  all with the `TMPDIR=/tmp/claude-1000 GOPATH=/tmp/claude-1000/go
  GOCACHE=/tmp/claude-1000/go-cache` prefix. Commits small,
  `--no-gpg-sign`, prefixed `feat(rev-274):` / `fix(rev-274):` /
  `docs(rev-274):`.
- gopls/IDE diagnostics are distrusted on this branch per standing notes —
  verify only with real build/test.

## §5 Verification & close-out

- **P5 Full parity audit** — the standard closing phase
  (`PORTING-LESSONS.md` §5): method-paired Go↔Java(@`32f3062`) comparison
  across all units plus reverse coverage of every Go file. Audit doc is a
  SNAPSHOT — the moment the fix pass starts, never re-fix from the doc.
- **P6 Fix pass** — audit findings fixed in small commits, gates green per
  commit; intentional deviations and cosmetic findings left open by policy,
  listed in the audit doc.
- **P7 Host smoke** — user-run on host vs Engine-TS 274
  (`4c0d036f`); watch items derived from what P4 changed (sound engine,
  opcode surface, UI). Lessons fold into `PORTING-LESSONS.md` on `main`;
  close-out resume note written; memory updated.

### Success criteria

1. Go branch diff `rev-254..rev-274` corresponds change-for-change to the
   real Java logic delta (the property the audit diffs against).
2. Parity audit complete with all confirmed blockers/bugs fixed.
3. Host smoke test passes against Engine-TS 274.
4. All gates green at every commit along the way.

## §6 Risks & mitigations

1. **`ClientBuild` package move → Go import cycle.** Java moved it into the
   `client` package; Go's `client` is the top-level aggregate (the reason
   `clientextras` exists). Mitigation: P1 examines the actual import graph;
   likely keep `clientbuild` a separate Go package, documented as an
   intentional deviation per the `clientextras` precedent.
2. **Name-reuse traps.** Both prior fresh-deob gaps had them (244 `Wall`,
   254 `World`). Mitigation: P1 pairs by obf-key + name agreement with
   structural adjudication; chained renames get explicit map entries.
3. **Opcode collisions.** New values collide with old values of different
   messages; partial carry-forward mis-routes silently. Mitigation: full
   re-derivation by enumeration (P3), both-ends grep for paired sites (P4).
4. **Non-monotonic deltas.** 274 may revert 254 shapes (Client.java shrank).
   Mitigation: treat undo-hunks as real delta; re-cite `// Java:` comments
   whenever the ground moves.
5. **Sed-induced string corruption** in the P2 rename pass. Mitigation:
   mandatory post-sed literal grep + diff vs reference.
6. **Large same-name diffs of unknown character** — `BZip2` (989 lines,
   was stable across prior revs), `OnDemand` (706), `CollisionMap` (617).
   Could be churn, could be real. Mitigation: P3 adjudicates per method;
   no assumption either way.
7. **Branch-dependent constants** — screen dims (765×503 on rev-244+ vs
   789×532 on rev-225), `clientversion`, window title. Mitigation: P3
   enumerates constants explicitly; never transposed from another branch.
8. **Audio seam drift.** The Go audio consumer (`audio/audioloop.go`) is a
   documented Go-side reconstruction seam, not a line-port target; the
   JagFX/Filter rework may reshape the publisher side again. Mitigation:
   scope the sound workstream against the 274 `sign/signlink.java` publisher
   shape first; TS client is the calibration arbiter where reconstructions
   disagree (245.2 precedent).
