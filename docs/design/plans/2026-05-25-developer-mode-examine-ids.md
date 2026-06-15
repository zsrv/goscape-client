# Developer Mode — Examine Menu IDs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When the `DEVELOPER_MODE` env var is `true`, the four Examine right-click menu options append the target's config-type id in green parentheses (e.g. `Examine @cya@Tree @whi@(@gre@1276@whi@)`).

**Architecture:** A single package-level bool in the `client` package, read once at package load from `os.Getenv("DEVELOPER_MODE")`. A one-line helper `examineIDSuffix(id int)` returns the markup suffix (or `""` when off). The flag and helper live in a new small file `pkg/jagex2/client/devmode.go` so the large `client.go` import block is untouched; the four Examine sites in `client.go` each gain a `+ examineIDSuffix(...)` call. Faithful port of LostCityRS TS-client commits `15260cc` / `b585f4c`; no Java #225 equivalent exists.

**Tech Stack:** Go 1.26, standard library only (`os`, `strconv`). Tests via `go test`. Spec: `docs/superpowers/specs/2026-05-25-developer-mode-examine-ids-design.md`.

> **Environment note (sandbox):** prefix every Go command with
> `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache`.
> Commit with `git commit --no-gpg-sign` (GPG signing fails in-sandbox).

---

## File Structure

- **Create** `pkg/jagex2/client/devmode.go` — `developerMode` flag + `examineIDSuffix` helper. Package `client`. Imports `os`, `strconv`.
- **Create** `pkg/jagex2/client/devmode_test.go` — unit test for `examineIDSuffix` (both states).
- **Modify** `pkg/jagex2/client/client.go` — four Examine menu sites gain a `+ examineIDSuffix(...)` call (no import change; `examineIDSuffix` is same-package).
- **Modify** `README.md` — add a "Developer mode" section.
- **Modify** `CLAUDE.md` — add a developer-mode note to the Build & Run section.

---

## Task 1: Flag + helper (TDD)

**Files:**
- Create: `pkg/jagex2/client/devmode.go`
- Test: `pkg/jagex2/client/devmode_test.go`

- [ ] **Step 1: Write the failing test**

Create `pkg/jagex2/client/devmode_test.go`:

```go
package client

import "testing"

func TestExamineIDSuffix(t *testing.T) {
	// Save and restore the package flag so the test is independent of the
	// DEVELOPER_MODE env var the binary was launched with.
	orig := developerMode
	defer func() { developerMode = orig }()

	t.Run("off returns empty", func(t *testing.T) {
		developerMode = false
		if got := examineIDSuffix(1276); got != "" {
			t.Errorf("examineIDSuffix(1276) off = %q, want %q", got, "")
		}
	})

	t.Run("on returns green id markup", func(t *testing.T) {
		developerMode = true
		want := " @whi@(@gre@1276@whi@)"
		if got := examineIDSuffix(1276); got != want {
			t.Errorf("examineIDSuffix(1276) on = %q, want %q", got, want)
		}
	})
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/jagex2/client/ -run TestExamineIDSuffix`
Expected: FAIL — compile error `undefined: developerMode` and `undefined: examineIDSuffix`.

- [ ] **Step 3: Write the minimal implementation**

Create `pkg/jagex2/client/devmode.go`:

```go
package client

import (
	"os"
	"strconv"
)

// developerMode mirrors the TS client's dev-client flag. When true, Examine
// menu options append the config-type id. Enabled with the DEVELOPER_MODE=true
// environment variable. Java: no equivalent — LostCityRS TS-client addition
// (commits 15260cc "feat: Dev client option" / b585f4c "fix: DEV_CLIENT env flag").
var developerMode = os.Getenv("DEVELOPER_MODE") == "true"

// examineIDSuffix returns the developer-mode id annotation appended to Examine
// menu text, e.g. " @whi@(@gre@1276@whi@)", or "" when developer mode is off.
// The id is the config type's .Index field (this port has no separate .Id).
func examineIDSuffix(id int) string {
	if !developerMode {
		return ""
	}
	return " @whi@(@gre@" + strconv.Itoa(id) + "@whi@)"
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/jagex2/client/ -run TestExamineIDSuffix -v`
Expected: PASS — both subtests `off returns empty` and `on returns green id markup`.

- [ ] **Step 5: Commit**

```bash
git add pkg/jagex2/client/devmode.go pkg/jagex2/client/devmode_test.go
git commit --no-gpg-sign -m "feat(client): DEVELOPER_MODE flag + examineIDSuffix helper

Java: no equivalent (LostCityRS TS-client addition, 15260cc/b585f4c).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: Wire the four Examine sites

**Files:**
- Modify: `pkg/jagex2/client/client.go` (4 sites: ~lines 1887, 2152, 8749, 8837)

> The two `"Examine @lre@" + var18.Name` lines are textually identical, so each
> edit below includes the following `c.MenuAction` line as a uniqueness anchor.
> Match on the surrounding code, not the line number. Append
> `+ examineIDSuffix(...)` to the `MenuOption` line only.

- [ ] **Step 1: Inventory obj site (MenuAction 1773)**

Edit `pkg/jagex2/client/client.go` — find:

```go
									c.MenuOption[c.MenuSize] = "Examine @lre@" + var18.Name
									c.MenuAction[c.MenuSize] = 1773
```

Replace with:

```go
									c.MenuOption[c.MenuSize] = "Examine @lre@" + var18.Name + examineIDSuffix(var18.Index)
									c.MenuAction[c.MenuSize] = 1773
```

- [ ] **Step 2: NPC site (MenuAction 1607)**

Find:

```go
		c.MenuOption[c.MenuSize] = "Examine @yel@" + var6
		c.MenuAction[c.MenuSize] = 1607
```

Replace with:

```go
		c.MenuOption[c.MenuSize] = "Examine @yel@" + var6 + examineIDSuffix(int(arg0.Index))
		c.MenuAction[c.MenuSize] = 1607
```

(`arg0` is the npc type; its `.Index` is `int64`, hence the `int(...)` conversion.)

- [ ] **Step 3: Loc / scenery site (MenuAction 1175)**

Find:

```go
						c.MenuOption[c.MenuSize] = "Examine @cya@" + var9.Name
						c.MenuAction[c.MenuSize] = 1175
```

Replace with:

```go
						c.MenuOption[c.MenuSize] = "Examine @cya@" + var9.Name + examineIDSuffix(var9.Index)
						c.MenuAction[c.MenuSize] = 1175
```

- [ ] **Step 4: Ground obj site (MenuAction 1102)**

Find:

```go
								c.MenuOption[c.MenuSize] = "Examine @lre@" + var18.Name
								c.MenuAction[c.MenuSize] = 1102
```

Replace with:

```go
								c.MenuOption[c.MenuSize] = "Examine @lre@" + var18.Name + examineIDSuffix(var18.Index)
								c.MenuAction[c.MenuSize] = 1102
```

- [ ] **Step 5: Build, vet, and lint**

Run:
```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go build ./...
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go vet ./pkg/jagex2/client/
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOLANGCI_LINT_CACHE=/tmp/claude-1000/golangci-cache \
  go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 run --max-issues-per-linter=0 --max-same-issues=0 ./pkg/jagex2/client/
```
Expected: build and vet succeed with no output; lint reports no new issues for the four changed lines / `devmode.go`.

If `int(arg0.Index)` fails to compile because `arg0.Index` is already `int`, drop the `int(...)` wrapper at the NPC site. (Spec records it as `int64`; trust the compiler.)

- [ ] **Step 6: Re-run the unit test (sanity)**

Run: `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/jagex2/client/ -run TestExamineIDSuffix`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add pkg/jagex2/client/client.go
git commit --no-gpg-sign -m "feat(client): append config-type id to Examine menus in developer mode

Wires examineIDSuffix into the obj/npc/loc/ground-obj Examine options.
Java: no equivalent (LostCityRS TS-client addition, 15260cc/b585f4c).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: Documentation

**Files:**
- Modify: `README.md` (after the WebAssembly "Notes" block, before `##### TO DO`)
- Modify: `CLAUDE.md` (Build & Run section)

- [ ] **Step 1: Add a Developer mode section to README.md**

Find:

```markdown
- Audio is not yet wired for the browser build.

##### TO DO
```

Replace with:

```markdown
- Audio is not yet wired for the browser build.

## Developer mode

Set `DEVELOPER_MODE=true` to append the config-type id to the four Examine
right-click options (inventory item, ground item, scenery/loc, npc):

```bash
DEVELOPER_MODE=true go run ./cmd/client 10 0 highmem members
```

An Examine option then renders as e.g. `Examine Tree (1276)`. Only the literal
string `true` enables it. This is a developer aid ported from the LostCityRS
TS client; it has no effect on game logic.

##### TO DO
```

- [ ] **Step 2: Add a developer-mode note to CLAUDE.md Build & Run**

Find:

```markdown
# Run (requires 4 args: node-id, port-offset, lowmem|highmem, free|members)
go run ./cmd/client 10 0 highmem members
```

Replace with:

```markdown
# Run (requires 4 args: node-id, port-offset, lowmem|highmem, free|members)
go run ./cmd/client 10 0 highmem members

# Run with developer mode (Examine menus show config-type ids)
DEVELOPER_MODE=true go run ./cmd/client 10 0 highmem members
```

- [ ] **Step 3: Commit**

```bash
git add README.md CLAUDE.md
git commit --no-gpg-sign -m "docs: document DEVELOPER_MODE developer mode

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Final verification

- [ ] **Full build + test sweep**

Run:
```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go build ./...
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/jagex2/client/
```
Expected: build clean; client package tests pass.

- [ ] **Manual smoke (host-only, optional)** — outside the sandbox, run
  `DEVELOPER_MODE=true go run ./cmd/client 10 0 highmem members`, log in, and
  right-click a tree / npc / ground item / inventory item; confirm the green id
  appears in the Examine option. (The sandbox cannot open a display, so this is
  a host step, not an in-session verification.)
