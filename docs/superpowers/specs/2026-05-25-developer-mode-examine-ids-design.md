# Developer mode — Examine menu config-type IDs

**Date:** 2026-05-25
**Status:** Design approved, ready for implementation plan
**Branch:** rev-225

## Summary

Port the LostCityRS TypeScript client's "dev client" feature to the Go client.
When developer mode is enabled, the four **Examine** right-click menu options
append the config-type id of the target in green parentheses, so a developer can
read an obj/loc/npc/item id straight from the menu instead of looking it up.

Example rendered option: `Examine @cya@Tree @whi@(@gre@1276@whi@)`

This is a faithful port of TS commits `15260cc` ("feat: Dev client option") and
`b585f4c` ("fix: DEV_CLIENT env flag"). There is **no Java #225 equivalent** —
it is a TS-client addition, so no Java reference exists to mirror.

## Motivation

The Go client currently has no developer/debug affordance for surfacing content
ids. The TS sibling port added a tiny, zero-fidelity-impact feature that shows
ids in the Examine menus behind a build flag. It is the one genuinely portable,
on-mission improvement found in the TS-vs-Go comparison (most other TS changes
are fixes to TS-port-specific regressions or browser/mobile-only concerns).

## Toggle mechanism

A single package-level bool in the `client` package, evaluated **once** at
package load from an environment variable:

```go
// developerMode mirrors the TS client's dev-client flag. When true, Examine
// menu options append the config-type id. Java: no equivalent (LostCityRS
// TS-client addition, commits 15260cc/b585f4c).
var developerMode = os.Getenv("DEVELOPER_MODE") == "true"
```

- **Env var name:** `DEVELOPER_MODE`. Enabled with
  `DEVELOPER_MODE=true go run ./cmd/client 10 0 highmem members`.
- **Exact `== "true"` match**, matching the TS `b585f4c` fix — only the literal
  string `true` enables it; `1`/`yes`/empty leave it off.
- **Read once** via the `var` initializer (no per-frame `os.Getenv`), mirroring
  the TS bundler `define` that inlines the value at build time.
- **Lives in `client`** (not `clientextras`): the only consumers are the menu
  sites in `client.go`, so there is no circular-import concern.
- **Cross-platform with no platform code:** `os.Getenv` works on native and on
  the WASM build. In a browser the env is empty unless the JS glue injects it,
  so developer mode safely defaults **off** on WASM with no `//go:build`
  branching. Enabling it on WASM later is a JS-glue concern, not a code change.

## The shared helper

To keep the markup in one place and leave the four authentic
`"Examine @xxx@" + name` lines visually intact, add one helper in `client.go`:

```go
// examineIDSuffix returns the developer-mode id annotation appended to Examine
// menu text, e.g. " @whi@(@gre@1276@whi@)", or "" when developer mode is off.
func examineIDSuffix(id int) string {
	if !developerMode {
		return ""
	}
	return " @whi@(@gre@" + strconv.Itoa(id) + "@whi@)"
}
```

The helper takes a plain `int` rather than a config-type object, so it stays
decoupled from the four different types and the markup is defined exactly once.
(`strconv` is the import to confirm/add in `client.go`.)

## The four edit sites

All four are in `pkg/jagex2/client/client.go`. Each changes from a single string
assignment to that same assignment with `+ examineIDSuffix(<index>)` appended.
The displayed id is the config type's **`.Index`** field — this Go port has no
`.Id` field; `.Index` is the content id (`objtype.Get(arg0)` sets
`var2.Index = arg0`), and is what the TS `obj.id`/`loc.id`/`npc.id` maps to.

| Site | Line | Action | Current expression | Append |
|---|---|---|---|---|
| Inventory obj | 1887 | 1773 | `"Examine @lre@" + var18.Name` | `examineIDSuffix(var18.Index)` |
| NPC | 2152 | 1607 | `"Examine @yel@" + var6` | `examineIDSuffix(int(arg0.Index))` |
| Loc / scenery | 8749 | 1175 | `"Examine @cya@" + var9.Name` | `examineIDSuffix(var9.Index)` |
| Ground obj | 8837 | 1102 | `"Examine @lre@" + var18.Name` | `examineIDSuffix(var18.Index)` |

Notes:
- **`NpcType.Index` is `int64`** (the other config types use `int`), so the NPC
  site wraps it in `int(...)`. Config ids fit comfortably in `int`; no precision
  loss.
- Line numbers are current-state references; the implementer should match on the
  surrounding code, not the line number alone.

## Markup reference

The color tokens are existing pixfont control codes already used throughout the
menu code: `@whi@` = white, `@gre@` = green, `@lre@`/`@cya@`/`@yel@` = the
authentic per-type Examine colors. The suffix wraps the id as
` @whi@(@gre@<id>@whi@)` so the parentheses render white and the id green.

## Verification

- **Build / vet / lint:** `go build ./...`, `go vet ./...`, and the golangci-lint
  `standard` gate stay clean. This is new code, held to the gate per project
  policy.
- **Unit test:** one focused test on `examineIDSuffix` covering both states —
  off returns `""`; on returns `" @whi@(@gre@1276@whi@)"`. (The test toggles the
  `developerMode` package var directly.) The four call sites are trivial
  concatenations and need no separate tests.
- **Manual smoke (host-only, optional):** run with `DEVELOPER_MODE=true`,
  right-click a tree / npc / ground item / inventory item, confirm the green id
  renders. Live-window runs are host-only (the sandbox cannot open a display).
- **Docs:** add a short "Developer mode" note to `README.md` (and the run section
  of `CLAUDE.md`) documenting `DEVELOPER_MODE=true`.

## Scope

**In scope:** exactly the four Examine menu sites above, the `developerMode`
flag, the `examineIDSuffix` helper, a unit test, and the docs note.

**Out of scope (deliberately):** interface/component-id overlays, a varp
inspector, and any extra HUD. None exist in the TS client; each is a larger,
net-new design. They are deferred to a future "developer tools" effort, which
would build on the `developerMode` gate this spec establishes.
