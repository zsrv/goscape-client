//go:build js

package platform

import (
	"syscall/js"
)

type webglTexture struct {
	tex  js.Value // WebGLTexture
	w, h int
	u8   js.Value // reusable Uint8Array sized w*h*4 for uploads
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
			b.events = append(b.events, CharInput{Rune: rs[0]})
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
