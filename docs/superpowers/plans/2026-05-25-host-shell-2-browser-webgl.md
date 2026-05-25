# Host Shell Plan 2 — Browser (WebGL) Backend + Gio Deletion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Prerequisite:** Plan 1 (`2026-05-25-host-shell-1-seam-native.md`) is complete and
committed — the `platform` seam, native GLFW backend, `RunShell` loop, and the
compile-only `js` stub backend all exist on `rev-225`.

**Goal:** Replace the stub `js` backend with a real `syscall/js` + WebGL backend, give the browser build its own HTML harness (no `gogio`), and delete the vendored+patched Gio tree entirely — leaving a Gio-free repository that runs natively and in the browser via one shared loop.

**Architecture:** A WebGL backend talks to a `<canvas>` directly through `syscall/js`, using the same texture-quad-shader present path as the native GLES2 backend (WebGL1 ≈ GLES2). DOM `addEventListener` callbacks (`js.FuncOf`) feed the same neutral `platform.Event` queue the native backend produces. The `RunShell` loop's `time.Sleep` yields to the JS event loop (no `requestAnimationFrame`), per the TS-client precedent. The build switches from `gogio` to a plain `GOOS=js GOARCH=wasm go build` plus a committed `web/index.html` and the stock `wasm_exec.js`.

**Tech Stack:** Go 1.26 `syscall/js`, WebGL1, the Go toolchain's `lib/wasm/wasm_exec.js`.

**Reference spec:** `docs/superpowers/specs/2026-05-25-hand-rolled-host-shell-design.md`

**Sandbox note:** the wasm build, `go vet`, and `go mod tidy` run in the sandbox
(prefix `TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache`). The browser
*run* is host-only (a real browser). All commits use `git commit --no-gpg-sign`.

---

## File Structure

| File | Responsibility |
|---|---|
| `pkg/jagex2/platform/backend_webgl.go` | **New, `//go:build js`.** Real WebGL backend: canvas/context, texture/blit/shader, DOM input listeners → neutral events, `KeyboardEvent.code` → `Key` mapping. Provides `newJSBackend`. |
| `pkg/jagex2/platform/backend_js_stub.go` | **Delete.** Replaced by `backend_webgl.go` (same `//go:build js`, same `newJSBackend`). |
| `web/index.html` | **New.** HTML harness: a `<canvas id="app">`, loads `wasm_exec.js` + `main.wasm`, parses `?argv=` into `go.argv`. |
| `Makefile` | **Modify.** `wasm` target: plain `go build` + copy `wasm_exec.js` + `web/index.html`; rename `WASM_OUT` to `build/web`. |
| `cmd/wasmserve/main.go` | **Modify.** Update default `-dir` to `build/web` and the doc comments (no longer "gogio bundle"; files are `index.html`, `main.wasm`, `wasm_exec.js`). |
| `go.mod` | **Modify.** Remove `replace gioui.org => ./third_party/gioui.org` and `require gioui.org v0.10.0`. |
| `third_party/gioui.org/` | **Delete.** The entire vendored, patched Gio tree. |

---

## Task 1: WebGL backend

Replace the stub. Not unit-testable without a browser; verification is the wasm
build (sandbox) + a browser run (Task 4).

**Files:**
- Create: `pkg/jagex2/platform/backend_webgl.go` (`//go:build js`)
- Delete: `pkg/jagex2/platform/backend_js_stub.go`

- [ ] **Step 1: Delete the stub**

```bash
cd $HOME/Code/github.com/zsrv/goscape-client
git rm pkg/jagex2/platform/backend_js_stub.go
```

- [ ] **Step 2: Implement the WebGL backend**

```go
//go:build js

package platform

import (
	"syscall/js"
)

type webglTexture struct {
	tex   js.Value // WebGLTexture
	w, h  int
	u8    js.Value // reusable Uint8Array sized w*h*4 for uploads
}

type webglBackend struct {
	doc    js.Value
	canvas js.Value
	gl     js.Value
	prog   js.Value
	vbo    js.Value
	aPos   int
	aUV    int
	uTex   js.Value
	uScr   js.Value
	w, h   int

	events []Event
	funcs  []js.Func // retained so the GC doesn't collect live listeners
}

const vertexShaderSrc = `
attribute vec2 aPos;
attribute vec2 aUV;
uniform vec2 uScreen;
varying vec2 vUV;
void main() {
	vec2 ndc = vec2(aPos.x / uScreen.x * 2.0 - 1.0,
	                1.0 - aPos.y / uScreen.y * 2.0);
	gl_Position = vec4(ndc, 0.0, 1.0);
	vUV = aUV;
}`

const fragmentShaderSrc = `
precision mediump float;
varying vec2 vUV;
uniform sampler2D uTex;
void main() { gl_FragColor = texture2D(uTex, vUV); }`

// newJSBackend creates the WebGL context on the page's <canvas id="app"> and
// installs DOM input listeners. The canvas exists before the wasm runs (index.html
// loads the module after the DOM). Same signature/name as the Plan 1 stub.
func newJSBackend(width, height int, title string) Backend {
	doc := js.Global().Get("document")
	canvas := doc.Call("getElementById", "app")
	if !canvas.Truthy() {
		panic("platform(js): no <canvas id=\"app\"> in the page")
	}
	canvas.Set("width", width)
	canvas.Set("height", height)
	doc.Set("title", title)

	gl := canvas.Call("getContext", "webgl")
	if !gl.Truthy() {
		gl = canvas.Call("getContext", "experimental-webgl")
	}
	if !gl.Truthy() {
		panic("platform(js): WebGL unavailable")
	}

	b := &webglBackend{doc: doc, canvas: canvas, gl: gl, w: width, h: height}
	b.prog = b.buildProgram()
	b.vbo = gl.Call("createBuffer")
	b.aPos = gl.Call("getAttribLocation", b.prog, "aPos").Int()
	b.aUV = gl.Call("getAttribLocation", b.prog, "aUV").Int()
	b.uTex = gl.Call("getUniformLocation", b.prog, "uTex")
	b.uScr = gl.Call("getUniformLocation", b.prog, "uScreen")
	gl.Call("disable", gl.Get("DEPTH_TEST"))
	gl.Call("disable", gl.Get("BLEND"))
	gl.Call("clearColor", 0, 0, 0, 1)
	b.installListeners()
	return b
}

func (b *webglBackend) buildProgram() js.Value {
	gl := b.gl
	vs := b.compile(vertexShaderSrc, gl.Get("VERTEX_SHADER"))
	fs := b.compile(fragmentShaderSrc, gl.Get("FRAGMENT_SHADER"))
	p := gl.Call("createProgram")
	gl.Call("attachShader", p, vs)
	gl.Call("attachShader", p, fs)
	gl.Call("linkProgram", p)
	if !gl.Call("getProgramParameter", p, gl.Get("LINK_STATUS")).Bool() {
		panic("platform(js): link: " + gl.Call("getProgramInfoLog", p).String())
	}
	return p
}

func (b *webglBackend) compile(src string, kind js.Value) js.Value {
	gl := b.gl
	s := gl.Call("createShader", kind)
	gl.Call("shaderSource", s, src)
	gl.Call("compileShader", s)
	if !gl.Call("getShaderParameter", s, gl.Get("COMPILE_STATUS")).Bool() {
		panic("platform(js): shader: " + gl.Call("getShaderInfoLog", s).String())
	}
	return s
}

func (b *webglBackend) NewTexture(w, h int) Texture {
	gl := b.gl
	tex := gl.Call("createTexture")
	gl.Call("bindTexture", gl.Get("TEXTURE_2D"), tex)
	gl.Call("texParameteri", gl.Get("TEXTURE_2D"), gl.Get("TEXTURE_MIN_FILTER"), gl.Get("NEAREST"))
	gl.Call("texParameteri", gl.Get("TEXTURE_2D"), gl.Get("TEXTURE_MAG_FILTER"), gl.Get("NEAREST"))
	gl.Call("texParameteri", gl.Get("TEXTURE_2D"), gl.Get("TEXTURE_WRAP_S"), gl.Get("CLAMP_TO_EDGE"))
	gl.Call("texParameteri", gl.Get("TEXTURE_2D"), gl.Get("TEXTURE_WRAP_T"), gl.Get("CLAMP_TO_EDGE"))
	gl.Call("texImage2D", gl.Get("TEXTURE_2D"), 0, gl.Get("RGBA"), w, h, 0,
		gl.Get("RGBA"), gl.Get("UNSIGNED_BYTE"), js.Null())
	u8 := js.Global().Get("Uint8Array").New(w * h * 4)
	return &webglTexture{tex: tex, w: w, h: h, u8: u8}
}

func (b *webglBackend) UploadTexture(t Texture, rgba []byte) {
	gl := b.gl
	tex := t.(*webglTexture)
	js.CopyBytesToJS(tex.u8, rgba)
	gl.Call("bindTexture", gl.Get("TEXTURE_2D"), tex.tex)
	gl.Call("texSubImage2D", gl.Get("TEXTURE_2D"), 0, 0, 0, tex.w, tex.h,
		gl.Get("RGBA"), gl.Get("UNSIGNED_BYTE"), tex.u8)
}

func (b *webglBackend) BeginFrame() {
	gl := b.gl
	gl.Call("viewport", 0, 0, b.w, b.h)
	gl.Call("clear", gl.Get("COLOR_BUFFER_BIT"))
	gl.Call("useProgram", b.prog)
	gl.Call("uniform2f", b.uScr, float64(b.w), float64(b.h))
}

func (b *webglBackend) Blit(t Texture, x, y int) {
	gl := b.gl
	tex := t.(*webglTexture)
	x0, y0 := float64(x), float64(y)
	x1, y1 := float64(x+tex.w), float64(y+tex.h)
	verts := []float64{
		x0, y0, 0, 0,
		x1, y0, 1, 0,
		x0, y1, 0, 1,
		x1, y0, 1, 0,
		x1, y1, 1, 1,
		x0, y1, 0, 1,
	}
	fa := js.Global().Get("Float32Array").New(len(verts))
	for i, v := range verts {
		fa.SetIndex(i, v)
	}
	gl.Call("bindBuffer", gl.Get("ARRAY_BUFFER"), b.vbo)
	gl.Call("bufferData", gl.Get("ARRAY_BUFFER"), fa, gl.Get("DYNAMIC_DRAW"))
	gl.Call("enableVertexAttribArray", b.aPos)
	gl.Call("vertexAttribPointer", b.aPos, 2, gl.Get("FLOAT"), false, 16, 0)
	gl.Call("enableVertexAttribArray", b.aUV)
	gl.Call("vertexAttribPointer", b.aUV, 2, gl.Get("FLOAT"), false, 16, 8)
	gl.Call("activeTexture", gl.Get("TEXTURE0"))
	gl.Call("bindTexture", gl.Get("TEXTURE_2D"), tex.tex)
	gl.Call("uniform1i", b.uTex, 0)
	gl.Call("drawArrays", gl.Get("TRIANGLES"), 0, 6)
}

// EndFrame is a no-op: WebGL commands flush when the loop goroutine next yields
// (time.Sleep), at which point the browser composites the canvas.
func (b *webglBackend) EndFrame() {}

func (b *webglBackend) PollEvents() []Event {
	out := b.events
	b.events = nil
	return out
}

func (b *webglBackend) ShouldClose() bool { return false }
func (b *webglBackend) Size() (int, int)  { return b.w, b.h }

func (b *webglBackend) Destroy() {
	for _, f := range b.funcs {
		f.Release()
	}
}

func (b *webglBackend) on(target js.Value, event string, fn func(e js.Value)) {
	f := js.FuncOf(func(_ js.Value, args []js.Value) any {
		fn(args[0])
		return nil
	})
	b.funcs = append(b.funcs, f)
	target.Call("addEventListener", event, f)
}

func (b *webglBackend) installListeners() {
	win := js.Global()
	// Mouse: coordinates are canvas-relative via offsetX/offsetY.
	b.on(b.canvas, "mousemove", func(e js.Value) {
		b.events = append(b.events, MouseMove{X: e.Get("offsetX").Int(), Y: e.Get("offsetY").Int()})
	})
	b.on(b.canvas, "mousedown", func(e js.Value) {
		button := 1
		if e.Get("button").Int() == 2 {
			button = 2
		}
		b.events = append(b.events, MouseButton{
			X: e.Get("offsetX").Int(), Y: e.Get("offsetY").Int(), Button: button, Pressed: true,
		})
	})
	b.on(b.canvas, "mouseup", func(e js.Value) {
		button := 1
		if e.Get("button").Int() == 2 {
			button = 2
		}
		b.events = append(b.events, MouseButton{
			X: e.Get("offsetX").Int(), Y: e.Get("offsetY").Int(), Button: button, Pressed: false,
		})
	})
	b.on(b.canvas, "mouseenter", func(e js.Value) { b.events = append(b.events, MouseCross{Entered: true}) })
	b.on(b.canvas, "mouseleave", func(e js.Value) { b.events = append(b.events, MouseCross{Entered: false}) })
	// Suppress the right-click context menu so button 2 reaches the game.
	b.on(b.canvas, "contextmenu", func(e js.Value) { e.Call("preventDefault") })
	// Focus.
	b.on(win, "focus", func(e js.Value) { b.events = append(b.events, FocusChange{Gained: true}) })
	b.on(win, "blur", func(e js.Value) { b.events = append(b.events, FocusChange{Gained: false}) })
	// Keys on the window: keydown emits KeyPress (+ CharInput for printable
	// keys); keyup emits KeyPress release.
	b.on(win, "keydown", func(e js.Value) {
		k, r := jsKeyToNeutral(e.Get("code").String())
		mods := jsMods(e)
		if k != KeyNone {
			e.Call("preventDefault")
			b.events = append(b.events, KeyPress{Key: k, Rune: r, Mods: mods, Down: true})
		}
		key := e.Get("key").String()
		if rs := []rune(key); len(rs) == 1 {
			b.events = append(b.events, CharInput{R: rs[0]})
		}
	})
	b.on(win, "keyup", func(e js.Value) {
		k, r := jsKeyToNeutral(e.Get("code").String())
		if k != KeyNone {
			b.events = append(b.events, KeyPress{Key: k, Rune: r, Mods: jsMods(e), Down: false})
		}
	})
}

func jsMods(e js.Value) Mod {
	var m Mod
	if e.Get("shiftKey").Bool() {
		m |= ModShift
	}
	if e.Get("ctrlKey").Bool() {
		m |= ModCtrl
	}
	if e.Get("altKey").Bool() {
		m |= ModAlt
	}
	if e.Get("metaKey").Bool() {
		m |= ModSuper
	}
	return m
}

// jsKeyToNeutral maps a KeyboardEvent.code to a neutral Key. Letters/digits map
// to KeyRune with the rune (charFor resolves Shift). Returns KeyNone for keys
// the game ignores.
func jsKeyToNeutral(code string) (Key, rune) {
	if len(code) == 4 && code[:3] == "Key" { // "KeyA".."KeyZ"
		return KeyRune, rune(code[3])
	}
	if len(code) == 6 && code[:5] == "Digit" { // "Digit0".."Digit9"
		return KeyRune, rune(code[5])
	}
	switch code {
	case "ArrowLeft":
		return KeyLeft, 0
	case "ArrowRight":
		return KeyRight, 0
	case "ArrowUp":
		return KeyUp, 0
	case "ArrowDown":
		return KeyDown, 0
	case "Enter", "NumpadEnter":
		return KeyReturn, 0
	case "Escape":
		return KeyEscape, 0
	case "Home":
		return KeyHome, 0
	case "End":
		return KeyEnd, 0
	case "Backspace":
		return KeyBackspace, 0
	case "Delete":
		return KeyDelete, 0
	case "PageUp":
		return KeyPageUp, 0
	case "PageDown":
		return KeyPageDown, 0
	case "Tab":
		return KeyTab, 0
	case "ControlLeft", "ControlRight":
		return KeyCtrl, 0
	case "ShiftLeft", "ShiftRight":
		return KeyShift, 0
	case "AltLeft", "AltRight":
		return KeyAlt, 0
	case "F1":
		return KeyF1, 0
	case "F2":
		return KeyF2, 0
	case "F3":
		return KeyF3, 0
	case "F4":
		return KeyF4, 0
	case "F5":
		return KeyF5, 0
	case "F6":
		return KeyF6, 0
	case "F7":
		return KeyF7, 0
	case "F8":
		return KeyF8, 0
	case "F9":
		return KeyF9, 0
	case "F10":
		return KeyF10, 0
	case "F11":
		return KeyF11, 0
	case "F12":
		return KeyF12, 0
	}
	return KeyNone, 0
}
```

- [ ] **Step 3: Verify the wasm build**

```bash
cd $HOME/Code/github.com/zsrv/goscape-client
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOOS=js GOARCH=wasm go build ./cmd/client 2>&1 | grep -v 'stat cache'; echo "wasm ${PIPESTATUS[0]}"
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOOS=js GOARCH=wasm go vet ./pkg/jagex2/platform/ 2>&1 | grep -v 'stat cache'; echo "vet ${PIPESTATUS[0]}"
```
Expected: both exit 0.

- [ ] **Step 4: Commit**

```bash
git add pkg/jagex2/platform/backend_webgl.go
git rm --cached pkg/jagex2/platform/backend_js_stub.go 2>/dev/null || true
git commit --no-gpg-sign -m "feat(platform): browser syscall/js + WebGL backend (replaces stub)"
```

---

## Task 2: HTML harness + build target (drop gogio)

**Files:**
- Create: `web/index.html`
- Modify: `Makefile`, `cmd/wasmserve/main.go`

- [ ] **Step 1: Create `web/index.html`**

```html
<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<title>Jagex</title>
	<style>
		html, body { margin: 0; background: #000; }
		#app { display: block; image-rendering: pixelated; }
	</style>
</head>
<body>
	<canvas id="app"></canvas>
	<script src="wasm_exec.js"></script>
	<script>
		// argv: program name + the client's CLI args, taken from ?argv=...
		// (space-separated), defaulting to a sensible local run. wasmserve's
		// log prints an example URL. window.location drives the WebSocket /
		// cache origin (see signlink.ConfigureTransport, codebase_js.go).
		const params = new URLSearchParams(window.location.search);
		const argv = (params.get("argv") || "10 0 highmem members").split(/\s+/).filter(Boolean);
		const go = new Go();
		go.argv = ["client", ...argv];
		WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject)
			.then((res) => go.run(res.instance))
			.catch((err) => { document.body.innerText = "load error: " + err; });
	</script>
</body>
</html>
```

- [ ] **Step 2: Rewrite the Makefile `wasm` target (no gogio)**

Replace the `WASM_OUT` definition and the `wasm` / `wasm-serve` targets:

```makefile
# Browser build directory (plain go build output: index.html, main.wasm, wasm_exec.js).
WASM_OUT := build/web

wasm: ## Build the browser (js/wasm) client into build/web/
	mkdir -p $(WASM_OUT)
	GOOS=js GOARCH=wasm go build -o $(WASM_OUT)/main.wasm $(CMD)
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" $(WASM_OUT)/wasm_exec.js
	cp web/index.html $(WASM_OUT)/index.html

wasm-serve: ## Serve the browser build at http://localhost:8080 (run `make wasm` first)
	go run ./cmd/wasmserve -dir $(WASM_OUT)
```

- [ ] **Step 3: Update `cmd/wasmserve/main.go` default dir + comments**

Change the `-dir` default from `gio/client` to `build/web`:

```go
	dir := flag.String("dir", "build/web", "directory with the wasm build (index.html, main.wasm, wasm_exec.js)")
```

Update the package doc comment's first line and the `servesFromBundle` comment to
say "the hand-rolled wasm build (index.html, main.wasm, wasm_exec.js)" instead of
"the gogio js/wasm build (index.html, main.wasm, wasm.js)". The serving logic is
unchanged — it already serves `/`→index.html and any real file in `-dir`, and
proxies the rest. Adjust `cmd/wasmserve/main_test.go` if it asserts the old
default dir or filenames (`wasm.js` → `wasm_exec.js`).

- [ ] **Step 4: Add `build/` to `.gitignore`**

```bash
cd $HOME/Code/github.com/zsrv/goscape-client
grep -qxF 'build/' .gitignore || echo 'build/' >> .gitignore
```

- [ ] **Step 5: Verify wasmserve still builds + its tests pass**

```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go build ./cmd/wasmserve 2>&1 | grep -v 'stat cache'; echo "build ${PIPESTATUS[0]}"
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./cmd/wasmserve/ 2>&1 | grep -v 'stat cache'; echo "test ${PIPESTATUS[0]}"
```
Expected: build 0, test ok.

- [ ] **Step 6: Commit**

```bash
git add web/index.html Makefile cmd/wasmserve/main.go cmd/wasmserve/main_test.go .gitignore
git commit --no-gpg-sign -m "build(wasm): hand-rolled HTML harness + plain go build (drop gogio)"
```

---

## Task 3: Delete the vendored Gio tree

**Files:**
- Delete: `third_party/gioui.org/`
- Modify: `go.mod` (+ `go.sum`)

- [ ] **Step 1: Confirm nothing imports gioui anymore**

```bash
cd $HOME/Code/github.com/zsrv/goscape-client
grep -rn "gioui.org" --include='*.go' . | grep -v '/third_party/gioui.org/' || echo "no gioui imports outside the vendored tree"
```
Expected: "no gioui imports outside the vendored tree". (If anything prints, it is
a missed reference from Plan 1 — fix it before deleting.)

- [ ] **Step 2: Remove the tree and the go.mod directives**

```bash
git rm -r third_party/gioui.org
```

Edit `go.mod`: delete the line `replace gioui.org => ./third_party/gioui.org` and
the `gioui.org v0.10.0` line in the `require` block. Then tidy:

```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go mod tidy 2>&1 | grep -v 'stat cache'; echo "tidy ${PIPESTATUS[0]}"
```
Expected: tidy exit 0; `go.mod`/`go.sum` no longer mention `gioui.org`. (go-gl
requires from Plan 1 remain.)

- [ ] **Step 3: Build both targets + full test/vet**

```bash
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOOS=js GOARCH=wasm go build ./cmd/client 2>&1 | grep -v 'stat cache'; echo "wasm ${PIPESTATUS[0]}"
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go build ./pkg/... 2>&1 | grep -v 'stat cache'; echo "pkgs ${PIPESTATUS[0]}"
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/... ./cmd/... 2>&1 | grep -vE 'stat cache|no test files|^ok|cached'; echo "test ${PIPESTATUS[0]} (empty=pass)"
```
Expected: wasm 0, pkgs 0 (native-neutral packages; the cgo backend builds on the
host), tests pass.

> Sandbox caveat: `go build ./...` (including `./cmd/client` native) needs cgo +
> GL headers and will fail in the sandbox at the GLFW backend — that is expected.
> The host CI native-build job (added in Plan 1) covers it. In the sandbox verify
> the wasm build + `./pkg/...` + tests, as above.

- [ ] **Step 4: Confirm the leak-fix artifacts are gone**

```bash
grep -rn "MutableImageOp\|OpsMu\|gioui.org" --include='*.go' --include='go.mod' . || echo "Gio + leak-patch fully removed"
```
Expected: "Gio + leak-patch fully removed".

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum
git commit --no-gpg-sign -m "chore: delete vendored+patched Gio; repository is now Gio-free"
```

---

## Task 4: Full verification + browser run handoff

- [ ] **Step 1: Sandbox gates**

```bash
cd $HOME/Code/github.com/zsrv/goscape-client
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOOS=js GOARCH=wasm go build ./cmd/client 2>&1 | grep -v 'stat cache'; echo "wasm ${PIPESTATUS[0]}"
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache go test ./pkg/... ./cmd/... 2>&1 | grep -vE 'stat cache|no test files|^ok|cached'; echo "test ${PIPESTATUS[0]} (empty=pass)"
TMPDIR=/tmp/claude-1000 GOCACHE=/tmp/claude-1000/go-cache GOOS=js GOARCH=wasm go vet ./pkg/jagex2/platform/ 2>&1 | grep -v 'stat cache'; echo "vet ${PIPESTATUS[0]}"
```
Expected: wasm 0, tests pass, vet 0.

- [ ] **Step 2: Host gate — browser build + run (handoff to user)**

On the host, with the cache-data backend running on :8888:
```bash
make wasm && make wasm-serve
# open http://localhost:8080/?argv=10%200%20highmem%20members
```
Acceptance gates (the spec's §Testing):
- Title screen renders; **flames animate** (proves change-detection re-upload via
  WebGL `texSubImage2D` + the flameMu hand-off).
- **Memory plateaus** on the title screen — no climb to GB (the in-place upload
  means no per-frame texture churn; this is the leak's permanent fix, now without
  any Gio patch).
- Typing (login/chat) and mouse (click, right-click menus) work.
- Login → gameplay viewport renders smoothly; memory stays flat.
- Audio plays (oto resumes on the first user gesture automatically).

- [ ] **Step 3: Host gate — native still works**

```bash
go build ./... && go run ./cmd/client 10 0 highmem members
```
Acceptance: unchanged from Plan 1 Task 10 (window, flames, input, flat memory).
Confirms deleting Gio didn't disturb the native path.

- [ ] **Step 4: `make lint` on the host before any push.**

- [ ] **Step 5: Mark Plan 2 complete. The host shell is hand-rolled and Gio-free on both targets.**

---

## Self-Review notes (run before execution)

- `newJSBackend` keeps the exact signature the Plan 1 `run_js.go` glue calls
  (`func newJSBackend(width, height int, title string) Backend`), so deleting the
  stub and adding the WebGL file is a drop-in swap.
- The WebGL present path uses the same shader source, attribute names (`aPos`,
  `aUV`, `uScreen`, `uTex`), quad winding, and nearest-neighbor filtering as the
  native GLES2 backend — visual parity by construction.
- Key mapping (`jsKeyToNeutral`) and native mapping (`glfwKeyToNeutral`) both
  target the same neutral `Key`/`KeyRune` set consumed by the shared
  `awtFor`/`charFor`, so input behaves identically on both targets.
- `texSubImage2D`-in-place + the change-detection hash means no per-frame texture
  creation — the original wasm leak cannot recur, and there is no Gio patch left
  to maintain.
