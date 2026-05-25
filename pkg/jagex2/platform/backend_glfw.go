//go:build !js

package platform

import (
	"fmt"
	"strings"

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

type glTexture struct {
	id   uint32
	w, h int
}

type glfwBackend struct {
	win    *glfw.Window
	w, h   int
	prog   uint32
	vbo    uint32
	events []Event
}

const vertexShaderSrc = `
attribute vec2 aPos;     // pixel coords
attribute vec2 aUV;
uniform vec2 uScreen;    // viewport size in pixels
varying vec2 vUV;
void main() {
	// pixel space -> clip space (flip Y so (0,0) is top-left)
	vec2 ndc = vec2(aPos.x / uScreen.x * 2.0 - 1.0,
	                1.0 - aPos.y / uScreen.y * 2.0);
	gl_Position = vec4(ndc, 0.0, 1.0);
	vUV = aUV;
}` + "\x00"

const fragmentShaderSrc = `
#ifdef GL_ES
precision mediump float;
#endif
varying vec2 vUV;
uniform sampler2D uTex;
void main() { gl_FragColor = texture2D(uTex, vUV); }
` + "\x00"

// newGLFWBackend creates the window and GL context. MUST be called on the
// LockOSThread'd main goroutine (see run_native.go).
func newGLFWBackend(width, height int, title string) *glfwBackend {
	if err := glfw.Init(); err != nil {
		panic(fmt.Sprintf("glfw init: %v", err))
	}
	glfw.WindowHint(glfw.ContextVersionMajor, 2)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.Resizable, glfw.False)
	win, err := glfw.CreateWindow(width, height, title, nil, nil)
	if err != nil {
		panic(fmt.Sprintf("glfw window: %v", err))
	}
	win.MakeContextCurrent()
	glfw.SwapInterval(0) // vsync OFF — the RunShell sleep pacing governs fps
	if err := gl.Init(); err != nil {
		panic(fmt.Sprintf("gl init: %v", err))
	}
	b := &glfwBackend{win: win, w: width, h: height}
	b.prog = buildProgram()
	gl.GenBuffers(1, &b.vbo)
	b.installCallbacks()
	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.BLEND)
	gl.ClearColor(0, 0, 0, 1)
	return b
}

func buildProgram() uint32 {
	vs := compileShader(vertexShaderSrc, gl.VERTEX_SHADER)
	fs := compileShader(fragmentShaderSrc, gl.FRAGMENT_SHADER)
	p := gl.CreateProgram()
	gl.AttachShader(p, vs)
	gl.AttachShader(p, fs)
	gl.LinkProgram(p)
	var ok int32
	gl.GetProgramiv(p, gl.LINK_STATUS, &ok)
	if ok == gl.FALSE {
		var n int32
		gl.GetProgramiv(p, gl.INFO_LOG_LENGTH, &n)
		log := strings.Repeat("\x00", int(n+1))
		gl.GetProgramInfoLog(p, n, nil, gl.Str(log))
		panic("link: " + log)
	}
	gl.DeleteShader(vs)
	gl.DeleteShader(fs)
	return p
}

func compileShader(src string, kind uint32) uint32 {
	s := gl.CreateShader(kind)
	csrc, free := gl.Strs(src)
	gl.ShaderSource(s, 1, csrc, nil)
	free()
	gl.CompileShader(s)
	var ok int32
	gl.GetShaderiv(s, gl.COMPILE_STATUS, &ok)
	if ok == gl.FALSE {
		var n int32
		gl.GetShaderiv(s, gl.INFO_LOG_LENGTH, &n)
		log := strings.Repeat("\x00", int(n+1))
		gl.GetShaderInfoLog(s, n, nil, gl.Str(log))
		panic("shader compile: " + log)
	}
	return s
}

func (b *glfwBackend) NewTexture(w, h int) Texture {
	var id uint32
	gl.GenTextures(1, &id)
	gl.BindTexture(gl.TEXTURE_2D, id)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(w), int32(h), 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)
	return &glTexture{id: id, w: w, h: h}
}

func (b *glfwBackend) UploadTexture(t Texture, rgba []byte) {
	tex := t.(*glTexture)
	gl.BindTexture(gl.TEXTURE_2D, tex.id)
	gl.TexSubImage2D(gl.TEXTURE_2D, 0, 0, 0, int32(tex.w), int32(tex.h),
		gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(rgba))
}

func (b *glfwBackend) BeginFrame() {
	w, h := b.win.GetFramebufferSize()
	b.w, b.h = w, h
	gl.Viewport(0, 0, int32(w), int32(h))
	gl.Clear(gl.COLOR_BUFFER_BIT)
	gl.UseProgram(b.prog)
	gl.Uniform2f(gl.GetUniformLocation(b.prog, gl.Str("uScreen\x00")), float32(w), float32(h))
}

func (b *glfwBackend) Blit(t Texture, x, y int) {
	tex := t.(*glTexture)
	x0, y0 := float32(x), float32(y)
	x1, y1 := float32(x+tex.w), float32(y+tex.h)
	// two triangles: pos.xy, uv.xy
	verts := []float32{
		x0, y0, 0, 0,
		x1, y0, 1, 0,
		x0, y1, 0, 1,
		x1, y0, 1, 0,
		x1, y1, 1, 1,
		x0, y1, 0, 1,
	}
	gl.BindBuffer(gl.ARRAY_BUFFER, b.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(verts)*4, gl.Ptr(verts), gl.DYNAMIC_DRAW)
	aPos := uint32(gl.GetAttribLocation(b.prog, gl.Str("aPos\x00")))
	aUV := uint32(gl.GetAttribLocation(b.prog, gl.Str("aUV\x00")))
	gl.EnableVertexAttribArray(aPos)
	gl.VertexAttribPointerWithOffset(aPos, 2, gl.FLOAT, false, 16, 0)
	gl.EnableVertexAttribArray(aUV)
	gl.VertexAttribPointerWithOffset(aUV, 2, gl.FLOAT, false, 16, 8)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, tex.id)
	gl.Uniform1i(gl.GetUniformLocation(b.prog, gl.Str("uTex\x00")), 0)
	gl.DrawArrays(gl.TRIANGLES, 0, 6)
}

func (b *glfwBackend) EndFrame() {
	b.win.SwapBuffers()
}

// (Threading note: this backend must be constructed and driven on the
// LockOSThread'd main goroutine — see run_native.go. No runtime import here.)

func (b *glfwBackend) PollEvents() []Event {
	b.events = b.events[:0]
	glfw.PollEvents() // fires the callbacks below, appending to b.events
	out := make([]Event, len(b.events))
	copy(out, b.events)
	return out
}

func (b *glfwBackend) ShouldClose() bool { return b.win.ShouldClose() }
func (b *glfwBackend) Size() (int, int)  { return b.w, b.h }

func (b *glfwBackend) Destroy() {
	b.win.Destroy()
	glfw.Terminate()
}

func (b *glfwBackend) installCallbacks() {
	b.win.SetCursorPosCallback(func(_ *glfw.Window, x, y float64) {
		b.events = append(b.events, MouseMove{X: int(x), Y: int(y)})
	})
	b.win.SetMouseButtonCallback(func(_ *glfw.Window, btn glfw.MouseButton, action glfw.Action, _ glfw.ModifierKey) {
		x, y := b.win.GetCursorPos()
		button := 1
		if btn == glfw.MouseButtonRight {
			button = 2
		}
		b.events = append(b.events, MouseButton{X: int(x), Y: int(y), Button: button, Pressed: action == glfw.Press})
	})
	b.win.SetCursorEnterCallback(func(_ *glfw.Window, entered bool) {
		b.events = append(b.events, MouseCross{Entered: entered})
	})
	b.win.SetFocusCallback(func(_ *glfw.Window, focused bool) {
		b.events = append(b.events, FocusChange{Gained: focused})
	})
	b.win.SetCharCallback(func(_ *glfw.Window, r rune) {
		b.events = append(b.events, CharInput{Rune: r})
	})
	b.win.SetKeyCallback(func(_ *glfw.Window, key glfw.Key, _ int, action glfw.Action, mods glfw.ModifierKey) {
		if action == glfw.Repeat {
			return // Java had no auto-repeat for these sentinels
		}
		k, r := glfwKeyToNeutral(key)
		if k == KeyNone {
			return
		}
		b.events = append(b.events, KeyPress{
			Key:  k,
			Rune: r,
			Mods: glfwMods(mods),
			Down: action == glfw.Press,
		})
	})
}

func glfwMods(m glfw.ModifierKey) Mod {
	var out Mod
	if m&glfw.ModShift != 0 {
		out |= ModShift
	}
	if m&glfw.ModControl != 0 {
		out |= ModCtrl
	}
	if m&glfw.ModAlt != 0 {
		out |= ModAlt
	}
	if m&glfw.ModSuper != 0 {
		out |= ModSuper
	}
	return out
}

// glfwKeyToNeutral maps a GLFW key to a neutral Key. Letters/digits map to
// KeyRune with the rune set (charFor resolves Shift). Returns KeyNone for keys
// the game ignores.
func glfwKeyToNeutral(key glfw.Key) (Key, rune) {
	switch {
	case key >= glfw.KeyA && key <= glfw.KeyZ:
		return KeyRune, rune('A' + (key - glfw.KeyA))
	case key >= glfw.Key0 && key <= glfw.Key9:
		return KeyRune, rune('0' + (key - glfw.Key0))
	}
	switch key {
	case glfw.KeyLeft:
		return KeyLeft, 0
	case glfw.KeyRight:
		return KeyRight, 0
	case glfw.KeyUp:
		return KeyUp, 0
	case glfw.KeyDown:
		return KeyDown, 0
	case glfw.KeyEnter, glfw.KeyKPEnter:
		return KeyReturn, 0
	case glfw.KeyEscape:
		return KeyEscape, 0
	case glfw.KeyHome:
		return KeyHome, 0
	case glfw.KeyEnd:
		return KeyEnd, 0
	case glfw.KeyBackspace:
		return KeyBackspace, 0
	case glfw.KeyDelete:
		return KeyDelete, 0
	case glfw.KeyPageUp:
		return KeyPageUp, 0
	case glfw.KeyPageDown:
		return KeyPageDown, 0
	case glfw.KeyTab:
		return KeyTab, 0
	case glfw.KeyLeftControl, glfw.KeyRightControl:
		return KeyCtrl, 0
	case glfw.KeyLeftShift, glfw.KeyRightShift:
		return KeyShift, 0
	case glfw.KeyLeftAlt, glfw.KeyRightAlt:
		return KeyAlt, 0
	case glfw.KeyF1:
		return KeyF1, 0
	case glfw.KeyF2:
		return KeyF2, 0
	case glfw.KeyF3:
		return KeyF3, 0
	case glfw.KeyF4:
		return KeyF4, 0
	case glfw.KeyF5:
		return KeyF5, 0
	case glfw.KeyF6:
		return KeyF6, 0
	case glfw.KeyF7:
		return KeyF7, 0
	case glfw.KeyF8:
		return KeyF8, 0
	case glfw.KeyF9:
		return KeyF9, 0
	case glfw.KeyF10:
		return KeyF10, 0
	case glfw.KeyF11:
		return KeyF11, 0
	case glfw.KeyF12:
		return KeyF12, 0
	}
	return KeyNone, 0
}
