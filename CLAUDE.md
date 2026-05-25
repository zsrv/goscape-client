# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go port of the RuneScape 2 (release #225) Java client. The codebase is a direct translation from Java to Go, preserving the original game logic while adapting to Go's idioms and type system.

## Build & Run

```bash
# Build
go build ./...

# Run (requires 4 args: node-id, port-offset, lowmem|highmem, free|members)
go run ./cmd/client 10 0 highmem members

# Run all tests
go test ./...

# Run a single test
go test ./pkg/jagex2/io/... -run TestIsaac
go test ./pkg/sign/signlink/... -run TestStartPriv
```

## Architecture

The `main` goroutine flow (from `main.go`):
1. `signlink.StartPriv()` — runs in a goroutine; handles filesystem, networking, DNS, MIDI, and wave audio via a polling loop
2. `platform.Main(w, h, title, fn)` (from `main.go`) — owns the threading model and builds the window backend via the `platform` seam (native: GLFW + go-gl; browser: syscall/js + WebGL). Inside its loop closure it calls `client.NewClient()` then `client.RunShell()`, which is the main game loop

### Package Layout (`pkg/jagex2/`)

| Package | Purpose |
|---|---|
| `client/` | Top-level `Client` struct and `GameShell`; owns the main game loop, network I/O, and rendering orchestration |
| `client/clientextras/` | Variables split out of `client` to avoid circular imports |
| `client/inputtracking/` | Mouse/keyboard input state |
| `config/{component,flotype,loctype,npctype,objtype,seqtype,spotanimtype,varptype,idktype}` | Game config/definition types loaded from cache files |
| `dash3d/` | Global scene variables for 3D rendering |
| `dash3d/world/` | Scene/tile building; converts cache data into a renderable `World3d` scene |
| `dash3d/world3d/` | The scene graph (tiles, entities, occlusion) |
| `dash3d/entity/` | Entity types: `PathingEntity`, `NpcEntity`, `PlayerEntity`, `LocEntity`, `ObjStackEntity`, `ProjectileEntity`, `SpotAnimEntity` |
| `graphics/model/` | 3D model data and rasterization |
| `graphics/pix2d/` | 2D pixel operations (line drawing, fill) |
| `graphics/pix3d/` | 3D rasterizer (triangle fill, texture mapping, sin/cos tables) |
| `graphics/pix8/` | 8-bit indexed-color pixel buffer |
| `graphics/pix32/` | 32-bit RGBA pixel buffer |
| `graphics/pixfont/` | Bitmap font rendering |
| `graphics/pixmap/` | CPU-side pixel buffer bridging the game renderer to GPU upload (via the `platform` backend) |
| `graphics/animbase/` & `animframe/` | Skeletal animation base/frame data |
| `graphics/metadata/` | Model metadata |
| `graphics/vertexnormal/` | Vertex normal smoothing |
| `datastruct/` | Generic `LruCache[T]`, doubly-linked list, `JString` |
| `io/` | `Packet` (binary reader/writer), `Jagfile` (JAG archive), ISAAC CSPRNG, `bzip2` decompressor, network protocol constants |
| `sound/wave/` | PCM wave audio |
| `sound/envelope/` & `tone/` | MIDI-style sound envelope/tone synthesis |
| `wordenc/wordfilter/` & `wordpack/` | Chat word filter and word packing |
| `sign/signlink/` | Filesystem/network bridge originally for the signed Java applet; handles cache directory, HTTP downloads, DNS, and audio requests |

### Key Java→Go Translation Notes (from README.md)

- `byte` → `int8`, `short` → `int16`, `int` → `int32`, `long` → `int64`, `char` → `uint16`
- Java `static` class vars → Go package-level vars (many packages have large `var` blocks)
- Java `HashTable` → Go built-in `map`
- Java class hierarchy → Go struct embedding + interfaces
- **Operator precedence**: Java and Go differ — parentheses are added where needed to match Java's evaluation order. Check [Java precedence](https://docs.oracle.com/javase/tutorial/java/nutsandbolts/operators.html) vs [Go precedence](https://go.dev/ref/spec#Operator_precedence) when translating expressions.
- **In-line increment side effects**: `i++` in Java within a larger expression differs from Go — watch for this in translated code.

### Rendering Pipeline

The game renders to CPU-side pixel buffers (`pix2d`, `pix8`, `pix32`) and the final composited frame is held in `pixmap.PixMap`. `PixMap.Draw` blits to the GPU each frame via the active `platform` backend's texture upload (native: OpenGL through go-gl; browser: WebGL `texSubImage2D`). The `DrawMu` mutex in `pixmap` guards concurrent access during frame uploads.

### Global State Pattern

Most packages use package-level `var` blocks for state (mirroring Java statics). The `Client` struct in `client/` is the main aggregate, but rendering subsystems (`pix3d`, `model`, `world`) also carry significant global state. This is intentional — it mirrors the original Java architecture.
