# Porting Lessons (Java → Go, RuneScape client)

Cross-revision knowledge for porting a RuneScape Java client revision to Go.
This is the durable, repo-owned distillation of what makes these ports correct
and what bites if you translate naively. Read it before starting a new revision.

Companion files: `REFERENCES.md` (pinned upstream commits per revision),
and each revision branch's own `README.md` / `PORTING.md` (revision-specific
conventions and roadmap).

---

## 1. Philosophy

**Faithful 1:1 translation is the default.** Preserve Java function names,
parameter order, control flow, and even oddities (`var7`-style locals). Any Java
line should map to a small, identifiable region of Go so behaviour bugs can be
diff-checked against the reference. Adapt to Go's type system rigorously, but do
**not** refactor opportunistically — no renaming, no extracting helpers, no
"idiomatic" rewrites that would obscure the Java mapping. Idiomatic-Go *style*
lints are deliberately not enforced for this reason.

---

## 2. Porting workflow for a new revision

The Go port of revision N is (Go port of revision N-1) + (the translated Java
delta). That makes it a branch operation:

1. **Branch** `rev-N` from the nearest prior Go revision branch (e.g. `rev-225`).
2. **Diff the primary Java reference across the gap.** Look up the prior
   revision's pinned commit in `REFERENCES.md`, then
   `git -C Client-Java diff <prev-commit>..<new-commit>`. That diff is your work
   list.
3. **Translate each Java change** into the corresponding Go region, applying the
   gotcha rules in §3. The Go branch diff (`git diff rev-(N-1) rev-N`) should
   correspond change-for-change to the Java diff — this is your audit.
4. **Record** the new reference commits in `REFERENCES.md` under `## rev-N`.
5. Each revision branch is self-contained (its own code, tooling, CI). Do **not**
   share code packages across revisions — independent faithful translations.

### When deob lineages diverge (the diff is *not* a clean work list)

Step 2 assumes the two reference commits share a deobfuscation lineage. The
LostCityRS `Client-Java` repo keeps **one branch per game revision** (`225`,
`225-clean`, `244`, `254`, `274`, …), and these are *independent deob efforts*,
not a linear history. When the new revision's branch was deobfuscated by a
different hand (or to a different stage), a raw `prev..new` diff is dominated by
non-behavioural churn and is **useless as a literal work list**. First observed
porting **rev-244** (`225-clean` → `244`): ~42 000 changed lines, of which the
real game-logic delta was a minority. The churn breaks down as:

- **Reassigned `@ObfuscatedName` keys** — the obfuscator hands every class new
  short names per build (`NpcType` `bc`→`gc`). Mechanical, behaviourless.
- **Divergent human deob names + package moves** — `PathingEntity`→`ClientEntity`,
  `Wall`→`Sprite`, `Location`→`QuickGround`, `graphics/Model`→`dash3d/Model`,
  the `dash3d/entity/` and `dash3d/type/` sub-packages flattened.
- **Name *reuse* traps** — 244 renamed `type/Wall`→`Sprite` **and** added a *new,
  unrelated* class also called `Wall`. "244 `Wall`" ≠ "225 `Wall`". Never pair
  classes by name alone across a divergent lineage.

Workflow when this happens:

1. **Recover the real delta** with rename detection and obfuscation filtering:
   `git -C Client-Java diff -M20% -w <prev>..<new>` and ignore `@ObfuscatedName`-only
   hunks. Treat git's low-% rename pairings (and the ADD/DELETE lists) as *hints*
   to verify by reading both trees + the TS client — not as truth.
2. **Pick a naming policy and record it in `REFERENCES.md`.** The rev-244 choice
   was to **adopt the new revision's names/structure** in the Go branch: do a
   mechanical rename/restructure pass *first* (Go realigns to match the new
   primary reference), *then* apply the game-logic delta on top. This keeps the
   Go↔Java mapping 1:1 going forward at the cost of one large up-front rename.
   The alternative (keep the prior names and maintain a mapping table) trades
   rename churn for a permanently skewed Go↔Java diff.

### Same lineage, fresh deob (the diff is *still* not a clean work list)

The benign-looking middle case, first observed porting **rev-245.2**
(`244` → `245.2`): the new branch adopts the *same naming convention* (no
rename/restructure pass needed) but is a **fresh deob with a cleanup pass**,
so the raw diff (~27k lines for 244→245.2) is still ~99% churn. What hides in
it:

- **Method reordering** — 245.2's `Client.java` was wholesale reordered,
  defeating raw line diffing. Pair methods **by name** (verify the name sets
  match 1:1 first; 144 ↔ 144 for 245.2) and compare bodies per pair.
- **Local/param renames *and reorders*** (`move(boolean,int,int)` →
  `move(int x, int z, boolean jump)`) — re-derive argument semantics from
  callee bodies, never from names or positions alone.
- **Wholesale wire-opcode renumbering** — every outbound/inbound/zone opcode
  literal renumbers per build, and new values *collide* with old values of
  *different* messages, so a partial carry-forward mis-routes silently.
  Re-derive **all** opcode tables from the new reference; verify by full
  enumeration, never by diff.
- Compensated pairs (a ctor's parameter semantics swap + its call sites
  swapping to match) are net-neutral but wrong if half-ported — land both
  halves in one commit.

Distill the real delta into a scope document first (the rev-245.2 method was
a method-paired multi-agent comparison with adversarial verification of every
claimed delta), then execute it as workstreams. Treat the scope doc's claims
as strong hints, not gospel — one of its claims was refuted during execution.

---

## 3. Java → Go translation gotchas

Each is a real bug class hit during the rev-225 port. Symbol-first; verify
against the Java source before "fixing" anything a linter flags.

### Numeric types & sign
- **`byte` fields → `int8`, not `byte`.** Java `byte` is signed; `int(field)`
  sign-extends. Go `byte` (uint8) zero-extends, silently changing values. Declare
  byte-typed fields as `int8` so promotion matches Java.
- **Type map:** `byte`→`int8`, `short`→`int16`, `int`→`int32`, `long`→`int64`,
  `char`→`uint16` (see the per-revision README translation table).
- **int64-vs-int32 truncation is the dominant latent class — three audits in a
  row** (225, 244, 245.2). Java `int` arithmetic wraps at 32 bits; Go locals
  typed `int` don't. Wrap with `int(int32(expr))` at multiply/accumulate/shift
  sites that can exceed 2^31, or type a cluster of locals `int32` when wraps
  chain through accumulators (Go `int32` arithmetic wraps mod 2^32 exactly like
  Java `int`; sign-extend back with `int(...)` at the hand-off). Watch `g4()`
  reads especially: Java's is a *signed* 32-bit value, and `>>`/comparison
  semantics on it differ once bit 31 is set.

### Operators & precedence
- **Operator precedence differs.** Java vs Go disagree on shift-vs-`&` and
  additive-vs-bitwise grouping. Bare-translating `a & b << c` or `a + b | c` can
  silently misgroup. Add explicit parens to match Java's evaluation order.
- **Shift-count masking.** Java implicitly masks shift counts (& 31 / & 63); Go
  panics on negative or oversized counts. Port `y >> (x >> N)` patterns by adding
  `& 0x1F` to the inner shift. (Caused a live `TextureRaster` crash.)
- **Boolean `|=` / `&=` are bitwise (unconditional) in Java.** Go has no boolean
  `|=`. Naively writing `x = x || rhs()` short-circuits and drops `rhs()`'s side
  effects. Port as `if rhs() { x = true }`, evaluating `rhs()` unconditionally.

### null / nil & casts
- **`== null` ↔ `== nil`: keep the polarity.** Flipping to `!= nil` compiles
  cleanly and is a silent logic inversion. (Caused the DrawInterface blank bug.)
- **Hoisted subtype casts.** Java's parent-typed reference with inline child
  casts must NOT become a hoisted `v.(*Child)` at the top; scope the type
  assertion to the branch where the concrete type is actually known.

### Strings & chars
- **`s + (char) i` → `s + string(rune(i))`**, never `strconv.Itoa(i)` (which
  yields `"97"` not `"a"`).
- **`String.length`/`charAt`/`substring` are UTF-16 code-unit based; Go
  `len(s)`/`s[i]` are byte-based.** Identical for ASCII; diverges on non-ASCII.
  Most parsed strings here are ASCII (filenames, config codes) so byte indexing
  is safe — but verify per site.

### Control flow & loops
- **Java for-loops that mutate the index** (`for (i=…; …; …) { … i += k; }`) must
  port to a C-style Go `for`, never `for i := range n` — the range form silently
  drops the mutation.
- **In-line increment side effects** (`a[i++]` inside a larger expression)
  evaluate differently; split them out explicitly.

### Concurrency
- **`synchronized` is a real hazard, not boilerplate.** Audit each against the
  actual goroutine layout. Many were defensive and need no lock here; some guard
  genuine cross-goroutine state (e.g. the MIDI loader handoff → `sync.Mutex`).
- **`Thread.start` + flag-based `Runnable.run` dispatch races in Go.** `go x.Run()`
  is async, so dispatch flags can mutate between the `go` and the goroutine
  actually starting. Call the target method directly. (Caused a boot crash.)
- **`InputStream.available()` ≠ `bufio.Reader.Buffered()`.** Java reports
  OS-buffered bytes; `bufio` is lazy and returns 0 until a read fills it. A naive
  port reads 0 forever.

### Rendering (host-shell / `platform` seam specifics)
The renderer is toolkit-neutral behind the `platform.Backend` seam (GLFW+go-gl
native, syscall/js+WebGL browser) — there is **no Gio** and no retained op list;
drawing is immediate-mode (one textured-quad `Blit` per `PixMap`, mirroring Java
`Graphics.drawImage` / TS `putImageData`). The earlier "Gio" framing of these
lessons is obsolete, but the invariants survived the rewrite:
- **Don't gate the blit on the AWT retained-back-buffer flag.** Java
  `if (redraw) { …; pixmap.draw() }` gates *both* repaint and blit. Here the
  `Blit` must re-issue **every** frame; only the CPU pixel repaint **and** its
  GPU upload (`UploadTexture`) are gated — by a content hash (`hashPixels`), not
  the Java redraw flag. See `pixmap.PixMap.Draw`.
- **Re-upload in place; never re-create the texture per frame.** The texture is
  allocated once (`NewTexture`) and refreshed via `glTexSubImage2D` /
  `texSubImage2D` (`UploadTexture`). Re-creating it each upload leaks GPU memory
  in the browser — this was the wasm leak fix. For pixmaps that change almost
  every frame (the 3D viewport) set `AlwaysUpload` to skip the whole-buffer hash;
  in-place upload means unconditional upload is safe.
- **Stage into a reusable `*image.RGBA`.** Pack `0x00RRGGBB` Java pixels into one
  caller-owned RGBA buffer (one wide big-endian store per pixel) to avoid a
  per-frame allocation. Java's `DirectColorModel` has no alpha mask, so every
  pixel is opaque and premultiplied RGBA equals straight NRGBA byte-for-byte.

### Things intentionally NOT ported
- **Deobfuscation artifacts** — empty placeholder classes, dead-write fields —
  are intentionally skipped. Mark the site `// Java: … Intentionally not ported`.
- **Applet API / `signlink.mainapp` checks** — the Go client is always
  standalone; these collapse to the non-applet branch.
- **The signlink *consumer* half** (MIDI/Wave readers) lived in the signed-applet
  wrapper process and was never in the LostCityRS Java repo. It was supplied
  Go-native (see the `audio` package). Cross-check the TS client to tell a
  "wrapper-side gap" from a genuine dead field.

---

## 4. Comment & reference conventions

- **Reference the Java by symbol, optionally with line numbers as a hint:**
  `// Java: Client.drawProgress (deob/client.java:6256)`. The **symbol** is the
  durable anchor; **line numbers are ephemeral** — they drift from unrelated
  edits within a file (observed: a one-line import addition shifted a reference
  by one) and are near-useless across revisions. Fix the line number when you
  touch the code; don't invest in keeping them precise.
- **No per-comment revision tags.** The branch *is* the revision context, and
  `REFERENCES.md` pins the exact Java commit — together they make every bare
  `// Java:` comment unambiguous without tagging hundreds of sites.
- **Renaming:** local vars and nondescript Java function names *may* be renamed,
  but every rename needs a `// Java: <original-name> (file:lines)` comment.

---

## 5. Verification

- **Check the Java source before "fixing" a gopls/staticcheck diagnostic.**
  Bug-for-bug fidelity is the stance; many diagnostics flag intentional fidelity.
- **Formatting:** plain `gofmt` (not `gofmt -s`) — `-s` simplifications rewrite
  constructs that mirror Java. British spellings in faithful `System.out.println`
  ports (`"unrecognised"`, `"RANDOMISED"`) are intentional; don't let a spell
  linter "fix" them.
- **Gates:** `go build ./...`, `go vet ./...`, `go test -race ./...`, and
  `golangci-lint` (the `standard` set). The race detector matters — the
  AWT-EDT/game-thread split ports to goroutines. Lint policy: fix findings in
  *new* code, but a faithful port that trips a lint keeps a per-line
  `//nolint:<linter>` with a `// Java:` reference rather than being rewritten.
- The live game window can't run headless (no display in CI/sandbox); pre-window
  boot and in-process machinery are still observable via the real binary.

### The full parity audit (one per revision port — a standard phase, not an extra)

All three completed ports ended with an exhaustive line-by-line audit of the Go
branch against the pinned Java reference (rev-225: `PARITY-AUDIT-2026-05-28.md`,
rev-244: `PARITY-AUDIT-2026-06-03.md`, rev-245.2: `PARITY-AUDIT-2026-06-04.md`,
each at its rev branch root), and every one found real bugs that had survived
every gate **and** smoke testing — 14 bugs (225) vs 11 blockers + 62 bugs (244)
vs 1 blocker + 6 bugs + 22 latent (245.2). The 4–5× jump at 244 is the
divergent deob lineage (§2): rename churn multiplies the miss rate, so the
dirtier the diff, the more the audit pays. The 245.2 numbers show the floor is
NOT zero even for a clean same-lineage delta port whose base was audited the
day before: its one blocker (a dropped bounds guard, panic reachable only via
server-driven cutscene coords) could not have been caught by any smoke test —
only full coverage finds the crash that needs unusual server input. Most 245.2
bugs were in code the delta *touched or sat adjacent to* (stale rev-225
constants, an if/else-if chain flattened during an earlier translation), i.e.
full-coverage re-walks also catch what earlier audits missed.

**When.** After the logic delta has fully landed; ideally after the first host
smoke test passes (245.2 ran audit-then-smoke instead, with the single smoke
run validating both the port and the fix pass — workable, but order fixes
before smoke either way). The smoke test proves the happy path boots; the
audit catches the mistranslations that don't crash the login path.

**Method** (copy the structure of the prior audit docs):

1. **Forward coverage — every Java file, no sampling.** One audit unit per Java
   file at the pinned reference commit; chunk big files by declaration-line
   windows (244: `Client.java` ×13, `Pix3D`/`World3D`/`Model` ×3, the rest
   singly or bundled). Walk every method statement by statement against its Go
   counterpart.
2. **Per-statement checklist** — the recurring miss classes: packet read
   widths; operator grouping (Java vs Go precedence, §3); branch polarity
   (`== null`/`!= nil` flips); loop bounds and direction; argument semantics
   traced through *callee bodies* (deob parameter names lie — the "arg
   scramble"); 32-bit wrap, sign-extension, and shift-mask semantics;
   side-effect ordering. Do not trust comments — re-verify every "equivalent
   because…" claim against the cited code.
3. **Adversarial verification.** Route every blocker/bug/latent finding to an
   independent skeptic primed with the known false-positive classes: deob arg
   scramble, cross-lineage name *inversions* (Java-244 `Model.minY` ≡ Go
   `Model.MaxY`), restructured-but-equivalent control flow,
   behaviour-lives-elsewhere, and documented seams. In the 244 run this killed
   4 of 136 findings before they could burn fix time.
4. **Reverse coverage.** Classify every Go file: each must be either the port
   of a Java file or a seam-justified Go original (platform / transport /
   storage / audio backend / profiling / build tooling). Zero unexplained
   files.
5. **Severity triage**, one bucket per finding: **Blocker** (crash, protocol or
   cache-format desync) → **Bug** (wrong behaviour or rendering) → **Latent**
   (edge-value / not-yet-reachable) → Cosmetic → Deferred (explicitly
   unimplemented; consolidate into a ledger) → Intentional (documented
   deviations). Fix in that order — any blocker can desync the session.

**Fix pass.** Group the confirmed findings; one gate-green commit per group
(build / vet / `test -race` / gofmt / golangci-lint). Then **re-run the host
smoke test** — the fix pass is itself a port change (244: 16 commits
`7b2d63e`..`7e9b28e`, post-fix smoke green same day).

**The audit doc is a snapshot, not a tracker.** Commit it as
`PARITY-AUDIT-<date>.md` at the rev branch root and don't edit findings
afterwards; the fix pass lands *on top of it*, so its deferred ledger goes
stale immediately. Keep live status in resume notes / a fix tracker, and never
re-fix from the audit doc without checking HEAD first — the 244 close-out
caught a ledger item still marked "deferred" that had in fact already landed.

**Scale honestly.** The 244 audit ran as a single multi-agent workflow
(50 units / 683 Java methods walked / every finding adversarially verified);
245.2's was 54 units / 674 methods / 89 agents with the same shape.
The *method* — full coverage, per-statement checks, skeptic pass, reverse
coverage — is the requirement; the tooling that delivers it is not.
