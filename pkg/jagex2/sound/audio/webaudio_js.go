//go:build js

package audio

import (
	"syscall/js"
	"unsafe"
)

// f32ToJSFloat32Array copies a Go []float32 into a new JS Float32Array via a
// byte view (one bulk CopyBytesToJS, no per-element boundary crossings).
func f32ToJSFloat32Array(s []float32) js.Value {
	if len(s) == 0 {
		return js.Global().Get("Float32Array").New(0)
	}
	b := unsafe.Slice((*byte)(unsafe.Pointer(&s[0])), len(s)*4)
	u8 := js.Global().Get("Uint8Array").New(len(b))
	js.CopyBytesToJS(u8, b)
	return js.Global().Get("Float32Array").New(u8.Get("buffer"))
}
