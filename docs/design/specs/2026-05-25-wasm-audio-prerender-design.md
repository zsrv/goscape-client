# Decouple wasm audio from the main thread (pre-render to Web Audio)

Status: design approved 2026-05-25. Branch: rev-225.

## Problem

On `js/wasm`, Go runs all goroutines on a single thread. oto v3's `AudioWorklet`
(`OtoWorkletProcessor`) plays from a small buffer and, when it runs low,
`postMessage`s the **main thread** to synthesize and send more — oto's
main-thread `port.onmessage` calls `mux.ReadFloat32s`, which pulls our
`midiSource.Read` (meltysynth). So **audio synthesis is main-thread and
request-driven**: any synchronous CPU burst on the main thread (cache/jagfile
decompression at startup, scene build, model normal/lighting bake, etc.) blocks
the worklet's data request, the worklet outputs silence, and the music skips.

Per-loop `platform.Yield()` was tried (commits de799df..cc99c8c) and reverted
(91991ce): single-threaded wasm starves audio during ANY heavy burst, and they
occur in many code paths, so yielding was non-convergent whack-a-mole.

Native (GLFW + oto) is unaffected — oto feeds from a real OS audio thread in
parallel with the game loop. **This change is wasm-only.**

## Approach

Stop depending on the main thread *during playback*. Pre-render each MIDI track
to a complete PCM buffer once, hand it to Web Audio as a static
`AudioBufferSourceNode`, and let the browser's audio thread play it
independently. Go touches audio only at track-change and command time, never
per-sample during playback, so no main-thread burst can starve it.

Rejected alternative: a second Go wasm instance in a Web Worker synthesizing
into a `SharedArrayBuffer` ring (true streaming, low memory) — far more complex
(two wasm instances, SAB, COOP/COEP cross-origin-isolation server headers, a
message protocol) for no listener-perceptible benefit here. The per-track
pre-render burst is harmless because the *previous* track keeps playing from its
buffer during it.

Scope: **music + SFX**, both via one Web Audio `AudioContext`. **oto is removed
from the js build entirely.** Native keeps oto unchanged.

## Architecture & file boundary

Build-tag the audio *output sink* by platform; keep everything else shared.

- **Shared (untagged):** public API (`Start`, `PlayMIDI`, `PlayWave`,
  `ReplayWave`, `DisableForLowMemory`), the signlink command watcher
  (`runMidiWatcher` → `ConsumeMidi` → command dispatch: `stop`, `voladjust`),
  soundfont loading (`soundfont.go`), WAV parsing (`wave8MonoToStereoInt16` and
  friends), and `volumeFromCentibels`. meltysynth is used by both.
- **`//go:build !js` (native):** the current oto path, unchanged in behavior.
- **`//go:build js` (wasm):** the new Web Audio sink (this spec).

Proposed files (final names to be settled in the plan): split the driver impls
into `*_native.go` / `*_js.go` (e.g. `midi_native.go`/`midi_js.go`,
`wave_native.go`/`wave_js.go`); shared logic stays in untagged files. The public
function signatures are identical across builds, so callers
(`client.SaveMidi` → `PlayMIDI`, SFX → `PlayWave`) are unchanged.

oto imports (`github.com/ebitengine/oto/v3`) must not be reachable from a `js`
build after this change; verify `GOOS=js go build` doesn't pull oto.

## Web Audio graph (js)

One `AudioContext` created in `Start`, resumed on the first user gesture
(`touchend`/`keyup`/`mouseup`) — same autoplay-policy handling oto did.

```
musicSource (AudioBufferSourceNode, loop=false) -> fadeGain -> musicVol -> destination
sfxSource   (AudioBufferSourceNode, one-shot)                 -> sfxVol  -> destination
```

- `fadeGain`: per-track-transition gain used only for the fade-out ramp.
- `musicVol`, `sfxVol`: GainNodes set from `MidiVol` / `WaveVol` centibels.

## Music path

`PlayMIDI(data []byte, fade bool)` (wasm impl):

1. **Render full track to PCM.** Parse with `meltysynth.NewMidiFile`, build a
   `Synthesizer` (reverb/chorus disabled, as now) and `MidiFileSequencer`
   (`Play(file, loop=false)`). Render the whole track into 2× `[]float32`
   (left/right) by repeatedly calling the sequencer's render until the sequence
   ends, plus a short release tail so trailing notes don't cut off. (Exact
   end-of-sequence detection / tail length is an implementation detail to pin
   down against the meltysynth API in the plan.)
2. **Build an `AudioBuffer`** (2 channels, 22050 Hz) and copy the float32 PCM in.
3. **Cache** the rendered buffer for the current track so the game's re-issue of
   the *same* track (the NextMusicDelay restart) **replays without
   re-rendering**. A new track renders a fresh buffer and replaces the cache
   (old buffer dropped for GC).
4. **Play** via a fresh `AudioBufferSourceNode(loop=false)` → `fadeGain` →
   `musicVol` → destination. (Source nodes are one-shot; each play/restart makes
   a new node from the cached buffer.)

Music does not loop at the synth or node level — `loop=false`, and the game
re-issues `SetMidi` to restart, identical to current behavior.

## Fades / stop / volume — faithful to current behavior

`fadeDuration = 2s`; fade curve is the current exponential (τ ≈ 0.5s).

- **fade = true (track change):** ramp the *outgoing* track's `fadeGain` to 0 via
  `setTargetAtTime(0, ctx.currentTime, ~0.5)`; after `fadeDuration`, stop the old
  source and start the new at full gain. This reproduces the current
  **fade-out-then-start (no overlap)** semantics. Guard with the existing `gen`
  counter (atomic): a fade abandons itself if a newer command superseded it, so
  rapid area changes can't leave a stale ramp clobbering the new track.
- **fade = false:** stop the old source, start the new at full immediately
  (the `snapGain(1.0)` equivalent — no unintended fade-in).
- **stop(fade):** with fade, ramp to 0 then stop; without, stop immediately.
  There is no oto internal buffer to flush — stopping the source IS the flush.
- **Volume:** `musicVol` set from `volumeFromCentibels(MidiVol)`, re-applied on
  every track change and on `voladjust` (covers MidiVol changing while music was
  disabled, then re-enabled). `sfxVol` from `WaveVol`.

## SFX path

`PlayWave(data []byte)` (wasm impl): decode the 22050 Hz mono 8-bit WAV to
float32, build a **mono (1-channel)** `AudioBuffer` (the browser up-mixes to the
output device; no manual mono→stereo duplication needed), and play a one-shot
`AudioBufferSourceNode` → `sfxVol` → destination. `ReplayWave` replays the last
decoded buffer. No GC-finalizer anchoring is needed (that was an oto Player
concern); the node fires and the browser reclaims it.

## Why this fixes the root cause

Every burst we chased (cache decompression, scene build, normal/lighting bake,
first render) runs on the main thread. Under this design, none of them touch the
audio data path during playback — the browser audio thread plays a static buffer
— so they cannot cause an underrun, regardless of which code path or how long
the burst is. It is convergent where per-loop yielding was not.

## Tradeoffs / risks / non-goals

- **Memory:** one cached rendered track (~5–25 MB depending on track length);
  freed on track change. Acceptable.
- **Per-track render burst:** a one-time main-thread cost per *new* track
  (~100s of ms). The previous track keeps playing during it (no skip). The very
  first track has a brief start delay (nothing is playing yet to skip).
  Chunking the render across frames is a possible later refinement, NOT v1.
- **meltysynth render-to-completion:** must detect sequence end and render a
  short tail; getting this wrong cuts off or over-renders the track.
- **No** `SharedArrayBuffer`, Worker, or COOP/COEP server-header changes.
- Native audio is out of scope and must remain byte-for-byte behavior-identical.

## Testing

- Native path unchanged → existing `audio` package tests still apply.
- Shared MIDI→PCM render is unit-testable on the native test build (meltysynth is
  cross-platform): render a small MIDI fixture, assert a non-empty PCM buffer of
  the expected approximate length. This guards the render logic without a browser.
- The js Web Audio graph (`AudioContext`, nodes, ramps, autoplay resume) cannot
  be unit-tested in the sandbox; it requires manual browser verification:
  music plays and loops via game re-issue, area-change crossfade matches the
  current feel, volume slider works, SFX play, and — the whole point — **no skip
  during cache download / scene load** with music playing.
