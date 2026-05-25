//go:build js

package audio

import (
	"log"
	"sync"
	"sync/atomic"
	"syscall/js"
	"time"

	"github.com/sinshu/go-meltysynth/meltysynth"

	"github.com/zsrv/goscape-client/pkg/sign/signlink"
)

// fadeDuration matches the native 2s fade-out window.
const fadeDuration = 2 * time.Second

// fadeTimeConstant is the exponential time constant for setTargetAtTime,
// matching the native per-sample smoother's ~0.5s (gainSmoothingAlpha).
const fadeTimeConstant = 0.5

// PlayMIDI renders an in-memory MIDI track and plays it (replacing the
// current track). fade=true reproduces the native fade-out-then-start.
func PlayMIDI(midData []byte, fade bool) {
	d := getMidiDriver()
	if d == nil {
		return
	}
	d.playFromBytes(midData, fade)
}

// webMidiDriver owns the current music AudioBufferSourceNode + its fadeGain,
// a one-track render cache, and a generation counter so a rapid track change
// abandons an in-flight fade.
type webMidiDriver struct {
	soundFont *meltysynth.SoundFont
	loadOnce  sync.Once

	mu       sync.Mutex
	curSrc   js.Value // current AudioBufferSourceNode (or undefined)
	curFade  js.Value // current per-track fade GainNode (or undefined)
	cacheKey string   // identity of the cached rendered track
	cacheBuf js.Value // cached AudioBuffer for cacheKey

	gen atomic.Uint64
}

func newWebMidiDriver() *webMidiDriver { return &webMidiDriver{} }

func (d *webMidiDriver) ensureSoundFont() *meltysynth.SoundFont {
	d.loadOnce.Do(func() {
		sf, err := loadSoundFont()
		if err != nil {
			log.Printf("audio/midi: soundfont unavailable, music silent: %v", err)
			return
		}
		d.soundFont = sf
	})
	return d.soundFont
}

// handle dispatches a signlink Midi command (same protocol as native).
func (d *webMidiDriver) handle(cmd string) {
	switch cmd {
	case "stop":
		d.stop(signlink.ReadMidiFade() == 1)
	case "voladjust":
		d.applyMidiVolume()
	default:
		log.Printf("audio/midi: ignoring unexpected command %q", cmd)
	}
}

// applyMidiVolume sets the shared music gain from MidiVol (called on
// voladjust and every track change).
func (d *webMidiDriver) applyMidiVolume() {
	v := float64(volumeFromCentibels(signlink.ReadMidiVol()))
	musicGain.Get("gain").Set("value", v)
}

// renderToBuffer renders midData to an AudioBuffer, reusing the cache when the
// same track is re-issued (the game's NextMusicDelay restart).
// renderToBuffer returns the AudioBuffer for midData, reusing the one-track
// cache on a re-issue of the same track (instant, no render). A new track is
// synthesized via the chunked, yielding renderMidiToPCM. Runs on a background
// goroutine (see playFromBytes); the cache fields are guarded by mu since
// playFromBytes can be called from the game-loop goroutine while a prior
// render goroutine is still in flight.
func (d *webMidiDriver) renderToBuffer(midData []byte) js.Value {
	key := string(midData)
	d.mu.Lock()
	if d.cacheKey == key && d.cacheBuf.Truthy() {
		buf := d.cacheBuf
		d.mu.Unlock()
		return buf
	}
	d.mu.Unlock()

	sf := d.ensureSoundFont()
	if sf == nil {
		return js.Undefined()
	}
	left, right, err := renderMidiToPCM(sf, midData)
	if err != nil {
		log.Printf("audio/midi: render: %v", err)
		return js.Undefined()
	}
	buf := ac.Call("createBuffer", ChannelCount, len(left), SampleRate)
	buf.Call("copyToChannel", f32ToJSFloat32Array(left), 0)
	buf.Call("copyToChannel", f32ToJSFloat32Array(right), 1)
	d.mu.Lock()
	d.cacheKey = key
	d.cacheBuf = buf
	d.mu.Unlock()
	return buf
}

// playFromBytes renders the track in the BACKGROUND (synthesis is 100s of ms;
// renderMidiToPCM yields between chunks so the game loop keeps drawing instead
// of freezing) and then swaps it in. The previous track keeps playing from its
// static buffer throughout. gen is bumped now so a newer play/stop arriving
// during the render makes this goroutine abandon — a rapid area change won't
// swap in a stale track.
func (d *webMidiDriver) playFromBytes(midData []byte, fade bool) {
	d.applyMidiVolume()
	gen := d.gen.Add(1)
	go func() {
		buf := d.renderToBuffer(midData)
		if !buf.Truthy() || d.gen.Load() != gen {
			return
		}
		d.startTrack(buf, fade, gen)
	}()
}

// startTrack swaps buf in as the current track. With fade, the OLD source's
// fade gain ramps to 0 and is stopped after fadeDuration, THEN the new source
// starts at full — the native fade-out-then-start (no overlap), gen-guarded.
func (d *webMidiDriver) startTrack(buf js.Value, fade bool, gen uint64) {
	d.mu.Lock()
	oldSrc, oldFade := d.curSrc, d.curFade
	d.mu.Unlock()

	startNew := func() {
		fadeGain := ac.Call("createGain")
		fadeGain.Get("gain").Set("value", 1.0)
		fadeGain.Call("connect", musicGain)
		src := ac.Call("createBufferSource")
		src.Set("buffer", buf)
		src.Set("loop", false)
		src.Call("connect", fadeGain)
		src.Call("start")
		d.mu.Lock()
		d.curSrc, d.curFade = src, fadeGain
		d.mu.Unlock()
	}

	if !fade || !oldSrc.Truthy() {
		stopNodes(oldSrc, oldFade)
		startNew()
		return
	}

	now := ac.Get("currentTime").Float()
	oldFade.Get("gain").Call("setTargetAtTime", 0.0, now, fadeTimeConstant)
	go func() {
		time.Sleep(fadeDuration)
		if d.gen.Load() != gen {
			return
		}
		stopNodes(oldSrc, oldFade)
		startNew()
	}()
}

// stop silences music. With fade, ramp the current source's fade gain to 0
// then stop it after fadeDuration; without, stop immediately. gen-guarded.
func (d *webMidiDriver) stop(fade bool) {
	gen := d.gen.Add(1)
	d.mu.Lock()
	src, fadeGain := d.curSrc, d.curFade
	d.curSrc, d.curFade = js.Undefined(), js.Undefined()
	d.mu.Unlock()
	if !src.Truthy() {
		return
	}
	if !fade {
		stopNodes(src, fadeGain)
		return
	}
	now := ac.Get("currentTime").Float()
	fadeGain.Get("gain").Call("setTargetAtTime", 0.0, now, fadeTimeConstant)
	go func() {
		time.Sleep(fadeDuration)
		if d.gen.Load() != gen {
			return
		}
		stopNodes(src, fadeGain)
	}()
}

// stopNodes stops the source and disconnects both the source and its fade
// GainNode, so a replaced track leaves no dead nodes wired into musicGain.
func stopNodes(src, fadeGain js.Value) {
	if src.Truthy() {
		src.Call("stop")
		src.Call("disconnect")
	}
	if fadeGain.Truthy() {
		fadeGain.Call("disconnect")
	}
}
