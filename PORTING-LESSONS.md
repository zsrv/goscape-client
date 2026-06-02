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
