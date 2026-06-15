# Embedding the WASM client in another Go module

This guide is for projects — such as the `goscape` server — that want to build
and serve the browser (`js/wasm`) goscape client from their own repository,
rather than running it standalone out of this one.

The client is designed to make this cheap: `cmd/client` is an ordinary `main`
package, the `js/wasm` build needs **no cgo or C toolchain** (the native
GLFW/OpenGL backend is excluded under `//go:build !js`), and the host page is
same-origin by construction.

## The three artifacts

A working browser client is exactly three files served from one directory:

| File | Source |
|------|--------|
| `main.wasm` | `GOOS=js GOARCH=wasm go build … ./cmd/client` |
| `wasm_exec.js` | `"$(go env GOROOT)/lib/wasm/wasm_exec.js"` — the Go runtime glue |
| `index.html` | the host page — embedded as the Go package `github.com/zsrv/goscape-client/web` (on the `rev-*` branches) |

> **Critical:** `wasm_exec.js` is **specific to the Go toolchain version that
> compiled `main.wasm`**. Always copy it from the *same* `GOROOT` used for the
> build. This is the main reason to **build the wasm in the consuming repo**
> (one toolchain, always in sync) rather than shipping a prebuilt `main.wasm`.

## Recommended approach: build from the module

1. Add the client to your module, pinned to the revision you want. Because every
   client revision shares the module path `github.com/zsrv/goscape-client`, you
   select a revision by pinning its commit (the tip of a `rev-*` branch):

   ```bash
   go get github.com/zsrv/goscape-client@<commit-on-rev-274>
   ```

2. Build the three artifacts into your static-asset directory (here `web/`):

   ```bash
   OUT=web
   GOOS=js GOARCH=wasm go build -ldflags "-s -w" \
       -o "$OUT/main.wasm" github.com/zsrv/goscape-client/cmd/client
   cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" "$OUT/wasm_exec.js"
   ```

3. Provide the host page. Either copy the reference page out of the embedded
   `web` package at build time, or serve it directly from Go:

   ```go
   import goscapeweb "github.com/zsrv/goscape-client/web"

   // Option A — write it next to the other artifacts:
   os.WriteFile(filepath.Join(out, "index.html"), goscapeweb.IndexHTML, 0o644)

   // Option B — serve all host assets straight from the embedded FS:
   http.Handle("/play/", http.StripPrefix("/play/",
       http.FileServer(http.FS(goscapeweb.Assets))))
   ```

   `web.IndexHTML` is the reference page; `web.Assets` is an `embed.FS` of all
   host assets. `wasm_exec.js` is deliberately *not* embedded (see the note
   above) — supply it from your own toolchain.

## Runtime contract

- **Arguments.** The page reads space-separated client flags from the `?argv=`
  query parameter (program name is added automatically), defaulting to
  `-node-id 10 -mem high -world-type members`. Example:
  `https://your-host/play/?argv=-node-id 10 -mem high -world-type members`.
  See the [`main` README](../README.md) for the full flag reference.
- **Server origin.** Unlike the desktop build, the browser build does **not**
  take a server-host argument: it derives the WebSocket and cache origin from
  `window.location`. Serving the client from the same origin as your game/world
  endpoints means **no transport configuration** is required.
- **Cache transport.** The on-demand cache is streamed over the world-server
  socket (see the on-demand transport notes in the [`main` README](../README.md));
  pair the client revision with a server that speaks the matching protocol.

## Notes

- The `js/wasm` build does not compile the native cgo dependencies (GLFW,
  go-gl, oto), so a CI runner with only the Go toolchain can produce `main.wasm`.
- Bringing `github.com/zsrv/goscape-client` into your `go.mod` does pull the
  native dependencies into your module graph (they appear in `go.sum`) even
  though they are not compiled for `js`. This is harmless module-graph weight.
- The browser build persists its cache across reloads via IndexedDB (in-memory
  fallback when unavailable). Audio plays via the Web Audio API (sound starts
  after the first user gesture, per browser autoplay policy). See the
  browser-build notes in the [`main` README](../README.md).
