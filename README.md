# Modifications

Java's `jagex2.datastruct.HashTable` has been replaced by Go's built-in map.

## Browser (WebAssembly) build

The client can run in a browser via Gio's `js/wasm` target. The game data and
WebSocket server are expected to be served from the **same origin** as the page;
the client derives its server target from `window.location` automatically.

```bash
# 1. Build the wasm bundle into gio/client/ (needs gogio; pulled via go run).
make wasm

# 2. Serve it locally (maps .wasm to application/wasm, required for streaming).
make wasm-serve

# 3. Open the client, passing the non-host args via the ?argv= query parameter:
#    node-id port-offset lowmem|highmem free|members
#    http://localhost:8080/?argv=10 0 highmem members
```

Notes:
- The server **host/scheme are auto-derived** from the page origin (`ws://` over
  HTTP, `wss://` over HTTPS), so — unlike the desktop build — you do **not** pass
  a host argument.
- Storage is **in-memory only** in this build: the cache and client id do not
  survive a page reload (IndexedDB persistence is planned).
- Audio is not yet wired for the browser build.
