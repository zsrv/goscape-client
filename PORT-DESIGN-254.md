# PORT-DESIGN-254.md — Rev-254 Port Design

Approved design for porting the RS2 release #254 Java client delta
(245.2 → 254) onto Go branch `rev-254`. Approved by user 2026-06-04.

Companion documents:

- `REFERENCES.md` (on `main`) — reference pins; the `## rev-254` section is
  added as part of kickoff.
- `PORTING-LESSONS.md` (on `main`) — cross-revision workflow + translation
  gotchas. §2 (porting workflow, both lineage-divergence subsections) and
  §5 (verification / full parity audit) govern this port.
- `RENAME-MAP.md` — verified class pairing; gains a `245.2 → 254` section
  in kickoff.
- `LOGIC-DELTA-SCOPE-254.md` — the verified delta scope. **Produced by the
  scope workflow (§3); does not exist until kickoff completes.**

## §0 Decisions log (user, 2026-06-04)

| Decision | Choice |
|---|---|
| Naming policy | **Hybrid**: adopt 254 class/file-level names everywhere via an up-front mechanical rename pass; where 254 regressed locals/params to `argN`, **keep** existing descriptive names, annotated `// Java: argN`. |
| Scope method | **Multi-agent workflow** (245.2 precedent `wf_b6bde418`): per-unit comparison, adversarial verification of every claimed delta. |
| Kickoff ordering | **Approach A — concurrent**: verify rename map by obf-key first, launch scope workflow in background, do branch/pins/rename pass in the foreground meanwhile. |
| First-session scope | Spec + plan + kickoff + scope workflow; workstream execution in later `/clear`-able sessions via resume notes. |

## §1 Scope & references

A full-parity port: faithful 1:1 translation of the 245.2 → 254 Java delta,
gate-green at every increment, closed by a full parity audit and a host smoke
test. Branch `rev-254` is cut from `rev-245.2` @ `05a7659244b9`.

Reference pins (recorded in `REFERENCES.md` on `main`; **always read via
`git show <pin>:…`**, never working trees — all four local repos sit on
`254-GOSCAPE` branches that may accumulate local edits; verified
`254 ≡ 254-GOSCAPE` in all four on 2026-06-04):

| Repo | Role | Branch | Pin |
|---|---|---|---|
| Client-Java | **primary** translation source | `254` | `2e629784c3dcb671ee3aab134f9cb91d614d8094` |
| Client-TS | secondary cross-check | `254` | `d340fc258c8d` |
| Engine-TS | smoke-test server | `254` | `2e3bcf439220` |
| Content | game content reference | `254` | `caee3f2eb3eb` |

(Engine-TS pin advanced `43e02957f355` → `2e3bcf439220` on 2026-06-10 — a
fast-forward on branch `254`; `254 ≡ 254-GOSCAPE` re-verified at the new pin.
The 2026-06-05 host smoke test ran against the prior pin `43e02957f355`.)

### Lineage characterization (verified 2026-06-04)

`254` is a **fresh deob with no shared git history** with `245.2` (root
`0f3a5df feat: Initial deob`; head commit "Synced names with webclient
research"). Same `jagex2/` package layout, **but**:

- **Class-level renames** from webclient research, including a chained
  name-reuse trap confirmed by obf-key forensics:
  - `World` (obf `c`, builder, 1203 ln) → **`ClientBuild`** (obf `c`, 1190 ln)
  - `World3D` (obf `s`, scene graph, 2187 ln) → **`World`** (obf `s`, 2190 ln)
  - "254 `World`" ≠ "245.2 `World`". Git's `M World.java` pairs two
    different classes; never pair by filename.
  - Also: `Component` → `IfType` (508↔520 ln), `Jagfile` → `JagFile`,
    `io/Protocol` → `client/Protocol` (package move).
- **Local/param name regression**: many methods regressed to `arg0`-style
  names (245.2 `LinkList.push(Linkable node)` → 254 `push(Linkable arg0)`).
  The churn direction is *worse* names this time, hence the hybrid policy.
- **Member-level `@ObfuscatedName` keys reassigned** (e.g. `pb.d`→`pb.c`) —
  member keys are per-build and **cannot** be used for cross-build pairing;
  only **class-level** keys are stable pairing anchors.
- **Genuinely new surface** (delta work, not rename churn): `VarBitType`
  (varbits arrive in 254), `client/Stats` (16-line skill-name table),
  `IfType` evolution, and an expected full wire-opcode renumbering across
  the 9-revision gap.
- Raw `git diff 176a85f..2e62978` ≈ 34k lines, ~unchanged by `-w` — the
  churn is renames, not whitespace. The raw diff is **not** a work list.

## §2 Kickoff (first session, concurrent)

1. **Obf-key class pairing.** Mechanically extract class-level
   `@ObfuscatedName("X")` keys from every file in both trees (`176a85f`,
   `2e62978`); pair by key. Output: `## 245.2 → 254` section in
   `RENAME-MAP.md` listing every rename, move, addition, deletion. Escalate
   (read both bodies + Client-TS) any class whose key is missing or
   ambiguous. This map is the trap-proof input to everything below.
2. **Launch the scope workflow** (§3) in the background, parameterized by
   the verified pairing.
3. **Foreground, while the workflow runs:**
   - Pin `## rev-254` in `REFERENCES.md` on `main` (lineage note, naming
     policy, trap warnings — mirroring the rev-244/245.2 sections).
   - **Mechanical Go rename pass** on `rev-254`, behavior-preserving, in
     collision-safe order:
     1. `pkg/jagex2/dash3d/world` → `pkg/jagex2/dash3d/clientbuild` (frees
        the `world` name),
     2. `pkg/jagex2/dash3d/world3d` → `pkg/jagex2/dash3d/world`,
     3. `pkg/jagex2/config/component` → `pkg/jagex2/config/iftype`
        (type `Component` → `IfType`),
     4. `io.Jagfile` → `io.JagFile` (type rename, package unchanged),
     5. protocol constants (`io/protocol.go`) move into `pkg/jagex2/client`,
        mirroring Java's `io/Protocol` → `client/Protocol` move —
        `client/client.go` is the table's only consumer (verified
        2026-06-04), so no import cycle arises,
     6. update the `CLAUDE.md` package table and any doc references.
   - Full gates after each step: build / vet / `test -race` / gofmt /
     golangci-lint; small commits, `--no-gpg-sign`.
   - New 254 classes (`VarBitType`, `Stats`) are **not** kickoff — they are
     delta work assigned to workstreams by the scope doc.
4. **When the workflow completes:** synthesize and commit
   `LOGIC-DELTA-SCOPE-254.md` on `rev-254`; write the resume note in
   `.claude/resume/`; update memory.

## §3 Scope workflow (delta distillation)

Mirror of 245.2's `wf_b6bde418` (67 agents), adapted to 254's churn profile:

- **Fan-out per paired unit** from the obf-key map. Each agent compares the
  245.2 unit ↔ 254 unit via `git show`, pairing methods **by name** after a
  1:1 name-set check (method names are mostly stable even where locals
  regressed). Name-set mismatches escalate to signature/body-shape pairing —
  never silently dropped.
- **Classification** of every difference: real delta / signature-only /
  churn (`argN` renames, member obf keys, blank-line reflow).
- **Adversarial verification**: every claimed real delta is independently
  re-checked by a second agent prompted to refute it. (245.2 precedent: 1 of
  118 claims refuted — the pass earns its cost.) Scope-doc claims are
  treated as strong hints during execution, not gospel.
- **Opcode surface handled specially**: re-derive **all** outbound/inbound/
  zone opcode tables from `2e62978` deob labels; multiset-diff against the
  245.2 tables. Never carry values forward — new values collide with old
  values of different messages (documented 245.2 trap: zone `LOC_DEL=198`
  was 244's `MAP_ANIM`).
- **Synthesis**: `LOGIC-DELTA-SCOPE-254.md` — verified deltas grouped into
  workstreams with size tags and a dependency order. Expected axes (the
  final list comes from the data, not this guess): opcodes/protocol +
  login version byte; varbits (`VarBitType` + Client varp/varbit handling);
  IfType/interface logic; config-field deltas; scene/loc-normal changes;
  OnDemand; audio/wordenc/io stragglers.

## §4 Execution model (later sessions)

- Workstreams from the scope doc, executed in dependency order in discrete
  `/clear`-able sessions; each session ends with a dated resume note in
  `.claude/resume/` reporting the path only.
- Critical path expectation: **opcodes + login version byte** land first;
  nothing connects until they do.
- Conventions carried forward:
  - Faithful 1:1 translation; every change cites `// Java: <File>.java:NNN`
    with refs read `@2e62978`.
  - **Hybrid naming**: 254 class names everywhere (from kickoff); keep
    descriptive locals where 254 regressed to `argN` (annotate); adopt 254
    local names where they are equal-or-better.
  - "Intentionally not ported" markers for pure deob artifacts.
  - Standing decisions from prior revisions are **not** re-fixed (tone.go
    int typing, LruCache dup-key constraint, loctype op[] overflow, Model
    ctor compensated pair, pre-varp FULL volume — see memory/close-outs).
  - Build/test prefix: `TMPDIR=/tmp/claude-1000
    GOCACHE=/tmp/claude-1000/go-cache go …`; commits `--no-gpg-sign`.

## §5 Verification & close-out

- **Per increment**: build / vet / `test -race` / gofmt / golangci-lint;
  existing test suites stay green throughout.
- **End of port** (standard phases, one per revision):
  1. Full parity audit per `PORTING-LESSONS.md` §5 (multi-agent,
     method-paired, adversarially verified) → audit doc
     `PARITY-AUDIT-<date>.md`.
  2. Fix pass for all confirmed findings (audit doc is a snapshot; resume
     notes are the source of truth afterwards).
  3. Cleanup pass (245.2 C1–C5 precedent: re-derive stale `// Java:`
     comments, drop junk params, add missing markers).
- **Host smoke test** vs Engine-TS `254` @ `43e02957f355` (user runs the
  server on the host): connectivity/login at client version 254, window
  title, interface render, audio, plus spot-checks of audit fixes.
- **Close-out**: dated close-out resume note; memory updated; `rev-254`
  self-contained (no shared code packages across revision branches).

### Success criteria

1. All scope-doc workstreams landed, gate-green.
2. Full parity audit run and all confirmed findings fixed or recorded as
   standing decisions.
3. Host smoke test passed against Engine-TS 254.
4. `REFERENCES.md`, `RENAME-MAP.md`, resume notes, and memory updated.

## §6 Risks & mitigations

| Risk | Mitigation |
|---|---|
| Name-reuse traps (`World`≠`World`) corrupt pairing | Class-level obf-key pairing verified **before** any rename or comparison; filenames never used as pairing keys. |
| Opcode collision (new value = old value of different message) | Full table re-derivation + multiset diff; never partial carry-forward. |
| `argN` regression defeats method pairing | Pair methods by name with 1:1 name-set check; escalate mismatches to signature/body-shape pairing. |
| Scope workflow false positives/negatives | Adversarial verification per claim; claims treated as hints during execution; the end-of-port parity audit is the backstop. |
| Concurrent kickoff renames diverge from scope findings | Rename map is verified independently (obf keys) before the workflow launches; workflow output keyed to the same map. |
| Reading reference working trees (local edits) | All four repos pinned; refs read via `git show <pin>:…` only. |
| Chained Go package rename collision | Strict order: `world`→`clientbuild` commits before `world3d`→`world`. |
