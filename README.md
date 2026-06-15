# goscape-client

A Go port of the **RuneScape 2** client (client revisions 225–274), translated
from the original Java client. The goal is a faithful, line-by-line
reimplementation of the original game logic in idiomatic Go, runnable natively
(desktop, via GLFW + OpenGL) and in the browser (WebAssembly + WebGL).

> **Disclaimer.** goscape-client is an unofficial, fan-made preservation and
> study project. It is **not affiliated with, endorsed by, or associated with
> Jagex Ltd.** "RuneScape" is a trademark of Jagex. This repository contains
> **only original source code** — no game assets, cache, or content are included
> or distributed. To run the client you must supply your own compatible server
> and game cache.

## Repository layout

This repository uses a **branch-per-revision** layout. The default `main` branch
holds **documentation only** — the runnable client lives on the `rev-*` branches,
each targeting a specific RS2 client revision:

| Branch       | Client revision | Notes                          |
|--------------|-----------------|--------------------------------|
| `rev-225`    | 225             | Base port                      |
| `rev-244`    | 244             |                                |
| `rev-245.2`  | 245.2           |                                |
| `rev-254`    | 254             |                                |
| `rev-274`    | 274             | **Latest — start here**        |

Pick a revision branch to build and run; `rev-274` is the most recent.

## Requirements

- **Go 1.26 or newer.**
- **A C toolchain and OpenGL/GLFW system libraries** for the native build (the
  renderer uses cgo via `go-gl/glfw` and `go-gl/gl`). On Linux that means a C
  compiler plus the X11/Wayland and OpenGL development packages; GLFW also
  supports macOS and Windows. The browser build needs none of these.
- **A compatible RS2 server and game cache.** The client connects to an
  RS2-protocol server over TCP (or WebSocket) and downloads cache data from an
  on-demand HTTP server. A compatible server implementation is
  [LostCityRS / Engine-TS](https://github.com/LostCityRS/Engine-TS). The
  defaults below assume such a server running locally.

## Build & run (native desktop)

```bash
git clone https://github.com/zsrv/goscape-client.git
cd goscape-client
git switch rev-274          # choose a revision branch

go build ./...              # compile

# Run against a local server using all defaults:
go run ./cmd/client

# Or specify everything explicitly:
go run ./cmd/client -node-id 10 -mem high -world-type members \
    -world-server tcp://127.0.0.1:43594 -ondemand-server http://127.0.0.1:8888
```

### Command-line flags

| Flag               | Default                     | Description                                             |
|--------------------|-----------------------------|---------------------------------------------------------|
| `-node-id`         | `10`                        | Server node id.                                         |
| `-mem`             | `high`                      | Memory mode: `high` or `low`.                           |
| `-world-type`      | `members`                   | World type: `free` or `members`.                        |
| `-world-server`    | `tcp://127.0.0.1:43594`     | World server URL. Scheme `tcp://`, `ws://`, or `wss://`.|
| `-ondemand-server` | `http://127.0.0.1:8888`     | On-demand (cache) server URL. Scheme `http://` or `https://`. |
| `-store-id` †      | `32`                        | Disk cache directory id (`.file_store_<id>`, clamped to 32–34). |
| `-version`         | `false`                     | Print build version information and exit.               |

† `-store-id` is available on `rev-244` and later; it does **not** exist on
`rev-225`. All other flags (and their defaults) apply to every revision branch.

To connect over WebSocket (e.g. for a remote or browser-style deployment):

```bash
go run ./cmd/client -world-server wss://play.example.com:443/ws
```

### Developer mode

Set `DEVELOPER_MODE=true` to surface config-type ids in the in-game Examine
menus:

```bash
DEVELOPER_MODE=true go run ./cmd/client -mem high -world-type members
```

## Build & run (browser / WebAssembly)

The client can run in a browser via its `js/wasm` target (syscall/js + WebGL).
Game data and the WebSocket server are served from the **same origin** as the
page; the client derives its server target from `window.location` automatically.

```bash
git switch rev-274

# 1. Build the wasm bundle into build/web/ (plain `go build`, no extra tooling).
make wasm

# 2. Serve it locally (maps .wasm to application/wasm, required for streaming).
make wasm-serve

# 3. Open the client, passing the non-host args via the ?argv= query parameter
#    (same -flag syntax as the desktop build; the server target is auto-derived):
#    http://localhost:8080/?argv=-node-id 10 -mem high -world-type members
```

Notes on the browser build:
- The server **host/scheme are auto-derived** from the page origin (`ws://` over
  HTTP, `wss://` over HTTPS), so — unlike the desktop build — you do **not** pass
  a server argument.
- Storage is **in-memory only**: the cache and client id do not survive a page
  reload (IndexedDB persistence is planned).
- Audio is not yet wired for the browser build.

## Documentation

Project documentation lives under [`docs/`](docs/) on this branch:

- [`docs/shared/PORTING.md`](docs/shared/PORTING.md) — the porting roadmap and
  Java→Go translation conventions.
- [`docs/shared/REFERENCES.md`](docs/shared/REFERENCES.md) — upstream reference
  repositories (Java/TypeScript clients, server) with pinned commits.
- [`docs/shared/PORTING-LESSONS.md`](docs/shared/PORTING-LESSONS.md) — cross-revision
  lessons learned.
- `docs/rev-*/` — per-revision parity audits, rename maps, and design notes.
- [`docs/design/`](docs/design/) — feature design specs and implementation plans.

## Contributing

This is a translation project: the original Java client is the authoritative
reference, and every Go change is expected to map to a corresponding piece of
Java. Each `rev-*` branch also carries a `CLAUDE.md` with an architecture
overview and the package layout. Start with [`docs/shared/PORTING.md`](docs/shared/PORTING.md)
for the conventions before making changes.
