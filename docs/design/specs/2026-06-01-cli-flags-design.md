# CLI flags instead of positional arguments — design

Date: 2026-06-01
Status: approved (pending implementation plan)

## Problem

`cmd/client/main.go` parses startup configuration from positional `os.Args`:

```
go run ./cmd/client <node-id> <lowmem|highmem> <free|members> [host|ws://…|wss://…]
```

This has three weaknesses:

1. **Positional and order-sensitive** — easy to misorder, no self-documentation.
2. **Hardcoded ports** — the game socket always dials `43594` (`client.go:6475`)
   and the asset/data server is fixed at port `8888`. Neither is configurable.
3. **A latent inconsistency in the data-server URL.** There are two independent
   notions of the data server that disagree today:
   - `client.codeBaseURL()` (native) → `http://<Host>:8888` (uses the configured host)
   - `signlink.dataServerURL` → hardcoded `http://127.0.0.1:8888` (ignores host;
     this is the one `signlink.OpenURL` actually fetches against)

## Goal

Replace positional args with named flags, make both server endpoints fully
configurable (scheme + host + port), and unify the two data-server sites behind
a single configured value.

## Non-goals

- No change to the `js`/wasm boot path's behavior. The browser build never runs
  `main.go`; it derives Host/port/transport from `window.location` in
  `signlink.ConfigureTransport` (`signlink_socket_js.go`). The flags are a
  native-only concern and must keep the shared `clientextras` globals settable
  from both entry points.
- No new flag library. Go stdlib `flag` (already used in `cmd/wasmserve`) is the
  idiomatic, zero-dependency choice for five flags.
- Backward compatibility with the positional form is **not** preserved; this is
  a hard switch to flags.

## Flag set

Native entry only (`cmd/client/main.go`):

| Flag | Type | Default | Notes |
|---|---|---|---|
| `-node-id` | int | `10` | was `os.Args[1]` |
| `-mem` | string | `high` | `high`\|`low`; any other value is an error |
| `-world-type` | string | `members` | `free`\|`members`; any other value is an error |
| `-world-server` | string | `tcp://127.0.0.1:43594` | `[tcp\|ws\|wss]://host:port`; scheme **and** port both required when supplied |
| `-ondemand-server` | string | `http://127.0.0.1:8888` | `[http\|https]://host:port`; scheme and port required when supplied |

Decisions baked in:

- **Single-value flags** for the two mutually-exclusive choices (`-mem`,
  `-world-type`) rather than bool pairs — one source of truth per choice, no
  "both set" validation, closer to Java's original string args.
- **`-world-type`** (not `-world`) to avoid visual confusion with `-world-server`.
- **Optional flags with localhost defaults**, but if a server flag is supplied
  it must carry an explicit scheme and explicit port. A bare host (no scheme) or
  a scheme with no port is an error. The defaults are full URLs, so they flow
  through the same parser and validate identically to user input.

Example invocations:

```
go run ./cmd/client                              # all defaults (localhost)
go run ./cmd/client -node-id 10 -mem high -world-type members
go run ./cmd/client -world-server tcp://gs.example.com:40000 \
                    -ondemand-server http://cache.example.com:8888
go run ./cmd/client -world-server wss://play.example.com:443/ws
```

Error cases (must exit non-zero with a clear message):

```
-mem hugemem                              # invalid -mem value
-world-server gs.example.com              # missing scheme
-world-server tcp://gs.example.com        # missing port
-world-server ftp://gs.example.com:21     # unsupported scheme
-world-server tcp://gs.example.com:40000/path   # tcp has no path
-ondemand-server http://cache.example.com # missing port
```

## Parsing & validation (`cmd/client/`)

- **Rename/extend** `parseHostArg` → `parseWorldServer(arg) (kind clientextras.TransportKind, host string, port int, path string, err error)`:
  - Scheme mandatory (must contain `://`); a bare host is an error.
  - Scheme switch: `tcp`→`TransportTCP`, `ws`→`TransportWS`, `wss`→`TransportWSS`,
    anything else → error.
  - `host` = `u.Hostname()`; empty → error.
  - Port required: `u.Port()` non-empty, parses to 1–65535; otherwise error.
  - Path: `u.EscapedPath()`. For `tcp`, a non-root path (`""`/`"/"`) is an error
    (TCP has no path). For `ws`/`wss`, empty → `"/"`.
- **New** `parseOndemandServer(arg) (baseURL string, err error)`:
  - Scheme `http` or `https` (mandatory), else error.
  - Host non-empty; port required.
  - Non-root path is an error (the cache base is `scheme://host:port`; signlink
    appends the rest).
  - Returns a normalized `scheme://host:port`.
- `-mem` / `-world-type` validated with a simple `switch`.

## Wiring into shared globals (`pkg/jagex2/client/clientextras`)

This step also fixes the existing data-server inconsistency.

- **Replace `clientextras.WSPort` with `clientextras.WorldPort int` (default `43594`)** —
  the authoritative game-server port for *all* transports. Removes the hardcoded
  `43594` and the "default-port + WS override" layering.
  - `LoginFunc` (`client.go:6475`): `OpenSocket(43594)` → `OpenSocket(clientextras.WorldPort)`.
  - `buildWSURL` (`signlink_ws.go`) simplifies from
    `(kind, host, defaultPort, overridePort, path)` to `(kind, host, port, path)`;
    the `overridePort` branch is removed.
  - `signlink_socket_js.go` `ConfigureTransport` sets `WorldPort` instead of
    `WSPort` (browser behavior unchanged — it still derives the port from the
    page origin).
- **Add `clientextras.OndemandBaseURL string` (default `http://127.0.0.1:8888`)** —
  the native data-server base, read by both former sites:
  - `client.codeBaseURL()` (native, `codebase_native.go`) returns `OndemandBaseURL`
    instead of synthesizing `http://<Host>:8888`.
  - `signlink.urlBase()` (native, `signlink_url_native.go`) returns
    `OndemandBaseURL`; the package-local `dataServerURL` var is removed. The
    signlink `OpenURL` test redirects `clientextras.OndemandBaseURL` to its
    `httptest` server instead of `dataServerURL`.
  - **`js` builds are untouched**: `codebase_js.go` and `signlink_url_js.go` keep
    deriving from `window.location`; `OndemandBaseURL` is native-only.

`main.go` flow: parse flags → set `client.NodeID`, call `SetLowMem`/`SetHighMem`,
set `client.MembersWorld`, set `clientextras.{Host, Transport, WorldPort, WSPath,
OndemandBaseURL}` from the parsed server flags → `signlink.ConfigureTransport()`
(still a native no-op) → unchanged from there.

## Faithful-port comments to update

The repo documents every deviation from the Java client. These comments become
false under this change and must be rewritten to reference the new flags:

- `main.go:29-37` — the `port-offset` deviation block claims the build "always
  uses the base ports (8888 / 43594)". Now untrue; both are configurable.
- `client.go:6472-6474` — narrates `openSocket(portOffset + 43594)` / "always
  dials the base port 43594".
- `codebase_native.go:12-15` — narrates `http://<host>:8888`.
- `signlink_url_native.go:5-11` — narrates the hardcoded `http://127.0.0.1:8888`.

## Docs to update

- `CLAUDE.md` "Build & Run" — replace positional examples with flag examples;
  drop the `port-offset` note or rephrase it (the port is now a real flag).
- `README.md:19-20` — replace the positional usage comment.

## Tests

- `hostarg_test.go` → `worldserver_test.go`: `tcp`/`ws`/`wss` happy paths;
  missing scheme (error); missing port (error); bad/out-of-range port (error);
  unsupported scheme (error); `tcp://host:port/path` (error); `ws`/`wss` empty
  path → `/`.
- New `ondemand_test.go`: `http`/`https` happy paths; missing scheme (error);
  missing port (error); non-root path (error); normalization output.
- `signlink_ws_test.go`: update to the 4-arg `buildWSURL` and `WorldPort`.
- signlink `OpenURL` test: set `clientextras.OndemandBaseURL` instead of
  `dataServerURL`.

## Risks

- Forgetting to set `WorldPort` on a code path that dials would TCP-connect to
  port 0. Mitigated by the `43594` default on the var and by both entry points
  (native `main.go`, browser `ConfigureTransport`) writing it explicitly.
- The `js` build must continue to compile and behave identically; the
  `WSPort`→`WorldPort` rename touches a `//go:build js` file, so the wasm build
  must be compiled as part of verification.
