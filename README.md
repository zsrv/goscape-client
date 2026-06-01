# Modifications

Java's `jagex2.datastruct.HashTable` has been replaced by Go's built-in map.

## Browser (WebAssembly) build

The client can run in a browser via its `js/wasm` target (syscall/js + WebGL). The game data and
WebSocket server are expected to be served from the **same origin** as the page;
the client derives its server target from `window.location` automatically.

```bash
# 1. Build the wasm bundle into build/web/ (plain `go build`, no gogio).
make wasm

# 2. Serve it locally (maps .wasm to application/wasm, required for streaming).
make wasm-serve

# 3. Open the client, passing the non-host args via the ?argv= query parameter
#    (same -flag syntax as the desktop build; the server target is auto-derived):
#    -node-id 10 -mem high -world-type members
#    http://localhost:8080/?argv=-node-id 10 -mem high -world-type members
```

Notes:
- The server **host/scheme are auto-derived** from the page origin (`ws://` over
  HTTP, `wss://` over HTTPS), so — unlike the desktop build — you do **not** pass
  a host argument.
- Storage is **in-memory only** in this build: the cache and client id do not
  survive a page reload (IndexedDB persistence is planned).
- Audio is not yet wired for the browser build.
