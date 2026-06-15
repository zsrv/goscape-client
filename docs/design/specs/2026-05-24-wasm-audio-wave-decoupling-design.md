# WASM Sub-project 3: Browser Audio (wave SFX decoupling) — Design

**Date:** 2026-05-24
**Status:** Approved (pending implementation)
**Branch:** rev-225

## Goal

Make wave **sound effects** play in the browser. After investigation, this is
the *only* audio gap: MIDI music, the SF2 soundfont, and the browser autoplay
policy already work on `js/wasm` (see *Background*). The wave path is the
exception — it reads SFX from a signlink scratch file via `os.ReadFile`
(`wave.go:49`), which fails in the browser and also persists transient SFX into
the cache.

The fix mirrors the MIDI refactor already in the codebase: play SFX from the
in-memory WAV bytes the client already holds, via a new `audio.PlayWave([]byte)`
/ `audio.ReplayWave()`, retiring the signlink scratch-path + watcher + disk
read. SFX then play from memory on **both** native and browser.

This is **sub-project 3 of the 4-part WASM-parity roadmap**; #1 (boot) and #2
(IndexedDB) are complete and live-validated.

## Background: what already works (and why)

- **Autoplay is handled by oto.** oto v3's wasm driver (`driver_js.go:168-192`)
  installs its own `document` listeners for `touchend`/`keyup`/`mouseup`, calls
  `AudioContext.resume()` on the first gesture, and only then closes the `ready`
  channel. So `ensureContext`'s `<-ready` blocks the audio goroutine until the
  user's first interaction — exactly right for the browser (music/SFX start on
  first click). **No `Resume()` wiring is needed from us.**
- **SF2 already loads without a filesystem.** `soundfont.go:loadSoundFont()`
  uses `signlink.CacheLoad` + `signlink.OpenURL` (not `os.ReadFile`), so it
  fetches `SCC1_Florestan.sf2` from the origin (via the dev proxy) and caches it
  in IndexedDB. (The server must serve that file — same as Client-TS.)
- **MIDI already plays from bytes.** `audio.PlayMIDI([]byte, fade)` + pure-Go
  `meltysynth`; `c.SaveMidi` calls it directly (`client.go:3322`).

The wave path is the lone holdout because it still uses the
write-to-scratch-file → `os.ReadFile` round-trip the MIDI path already abandoned.

## Non-goals

- Touching MIDI, SF2, autoplay, or the oto context lifecycle (all working).
- A browser-only code split: the fix is one shared path for native + browser
  (consistent with the MIDI refactor), not a `//go:build js` divergence.
- Removing the now-vestigial `signlink.WaveSave`/`WaveReplay`/`ConsumeWave`/`Wave`
  API — left in place (dead but harmless), exactly as `signlink.MidiSave` was
  left after the MIDI refactor. Minimizes signlink churn.
- Cleaning up the dead `os.ReadFile` in `midi.go:185` (`play(path)`, a defensive
  fallback never on the live path) — noted, not done here.

## Background: the current wave flow

```
c.SaveWave(data, len)                       client.go:5317
  └─ signlink.WaveSave(data, len)           sets SaveBuf/SaveReq="sound<n>.wav", WavePlay
       └─ Run(): store.save("sound<n>.wav", buf)   ← persists transient SFX to cache (browser: pollutes IndexedDB)
                 Wave = path.Join(cacheDir, "sound<n>.wav")  ← "" cacheDir in browser
  runWaveWatcher → ConsumeWave() → playWaveFile(path) → os.ReadFile(path)  ← FAILS in browser (no FS)
```

The WAV bytes originate at `client.go:6986` (`wave.Generate(...)` → `var5.Data`,
`var5.Pos`), are handed to `c.SaveWave(var5.Data, var5.Pos)` (`client.go:6990`),
and the result gates the `LastWaveID` de-dup (`true`) vs a retry (`false`).

## Design

### New audio API (`pkg/jagex2/sound/audio/wave.go`)

```go
// PlayWave plays a one-shot SFX from in-memory WAV bytes (the format produced
// by sound/wave.GetWave). It caches the bytes for ReplayWave, then plays them
// if the audio context is ready. SFX are fire-and-forget: if the context is not
// ready (pre-gesture, low-memory, or init failed) the clip is dropped, never
// queued or blocked — playing it later would surprise the player with a stale
// burst, and the call runs on the game-update path which must not block.
func PlayWave(data []byte)

// ReplayWave replays the most recently PlayWave'd clip (Java replaywave). No-op
// if nothing has played yet.
func ReplayWave()
```

- **Readiness (lock-free).** `ensureContext` holds `otoMu` across its blocking
  `<-ready`, so `PlayWave` must NOT take `otoMu` (it would block until the first
  gesture). Add a lock-free publish:

  ```go
  var readyCtx atomic.Pointer[oto.Context] // nil until the context is ready
  ```

  `Start` stores it once the context is ready (after `ensureContext` returns,
  before spawning watchers). `PlayWave`/`ReplayWave` do `ctx := readyCtx.Load();
  if ctx == nil { return }` — a single atomic load, no contention with init.

- **Last-wave cache.** `var waveMu sync.Mutex; var lastWave []byte`. `PlayWave`
  stores a copy of `data` under `waveMu` (defensive copy — `var5.Data` is a
  reused buffer). `ReplayWave` reads the copy under `waveMu` and plays it.

- **Playback (reused).** The conversion (`wave8MonoToStereoInt16`) and the
  one-shot, GC-anchored Player spawn (the `ctx.NewPlayer` + `SetVolume` +
  `IsPlaying`-poll anchor) move verbatim from the current `playWaveFile` into a
  shared internal `playWaveBytes(ctx *oto.Context, data []byte)`. Volume is read
  from `signlink.ReadWaveVol()` as today. `wave8MonoToStereoInt16` and
  `byteSliceReader` are unchanged.

### Wiring

`pkg/jagex2/sound/audio/audio.go`:
- Add `readyCtx atomic.Pointer[oto.Context]` (import `sync/atomic`).
- In `Start`, after `ensureContext()` succeeds: `readyCtx.Store(ctx)` (before
  `registerMidiDriver`).
- Remove `go runWaveWatcher(ctx)` (`audio.go:76`).

`pkg/jagex2/sound/audio/wave.go`:
- Remove `runWaveWatcher` and `playWaveFile` (the `os.ReadFile` path). Drop the
  now-unused `os` import. Keep the `signlink` import (`ReadWaveVol`).

`pkg/jagex2/client/client.go`:
```go
func (c *Client) SaveWave(arg0 []byte, arg1 int) bool {
	if arg0 == nil {
		return true
	}
	audio.PlayWave(arg0[:arg1])
	return true
}

func (c *Client) ReplayWave() bool {
	audio.ReplayWave()
	return true
}
```
Always returning `true` is correct: there is no longer a busy save-slot to
back-pressure on, so the wave is consumed (its `LastWaveID` recorded) rather than
retried. A clip dropped pre-gesture is simply skipped — the browser wouldn't
have played it anyway. (`arg0[:arg1]` matches the existing
`saveBuf[0:saveLen]` slice invariant; `var5.Data` is sized for `var5.Pos`.)

### Data flow (after)

```
c.SaveWave(var5.Data, var5.Pos)
  └─ audio.PlayWave(bytes)
       ├─ cache bytes (lastWave) for ReplayWave
       └─ ctx := readyCtx.Load(); if ctx != nil → playWaveBytes(ctx, bytes)
                                   else            → drop (pre-gesture/low-mem/failed)
```
No signlink scratch path, no `os.ReadFile`, no `store.save` → no cache pollution.

## Error handling

`PlayWave`/`ReplayWave` are best-effort and never crash the game (the call site
at `client.go:6983` already wraps them in `recover`): nil context → drop;
unparseable WAV (`wave8MonoToStereoInt16` returns false) → log + drop. Matches
current SFX behavior.

## Testing

- **Native unit tests** (`pkg/jagex2/sound/audio/wave_test.go`, new):
  - `wave8MonoToStereoInt16`: valid 22050/mono/8-bit RIFF → correct interleaved
    stereo int16 (midpoint `0x80`→0, `0xFF`→+32512, `0x00`→-32768); rejects wrong
    channels/rate/bits/format and truncated headers. (Currently untested; pure
    and high-value.)
  - Replay cache: `PlayWave(validWav)` with no ready context (test env:
    `readyCtx` nil → playback dropped) still caches `lastWave`; assert via an
    unexported accessor that the cached copy equals the input and is a distinct
    backing array (defensive copy). This exercises the caching path without an
    audio device.
- **No unit test for actual oto playback** — needs an audio device; browser-only,
  consistent with the rest of the package.
- **Manual browser smoke:** trigger an in-game SFX after the first click; confirm
  it is audible, that MIDI music also starts on first interaction, and that **no
  `sound<n>.wav` keys appear** in the `goscape` IndexedDB (Application panel).
- **Gates:** native `go build ./... && go test ./...`; `GOOS=js GOARCH=wasm go
  build ./cmd/client`; gofmt.

## Files changed

| File | Change |
|---|---|
| `pkg/jagex2/sound/audio/wave.go` | `PlayWave`/`ReplayWave`/`playWaveBytes` + `lastWave` cache; remove `runWaveWatcher`/`playWaveFile`; drop `os` import |
| `pkg/jagex2/sound/audio/audio.go` | `readyCtx atomic.Pointer`; `readyCtx.Store` in `Start`; remove `go runWaveWatcher` |
| `pkg/jagex2/client/client.go` | `SaveWave`/`ReplayWave` route to `audio.PlayWave`/`ReplayWave` |
| `pkg/jagex2/sound/audio/wave_test.go` | **new** — conversion + replay-cache tests |

## Open items for the implementation plan

- Confirm `wave.Generate`/`var5.Pos` never exceeds `len(var5.Data)` (the slice
  `arg0[:arg1]` invariant) — it matches the pre-existing `saveBuf[0:saveLen]`
  path, so this is a sanity check, not expected to change.
- Exact placement of the unexported test accessor for `lastWave` (a tiny
  `lastWaveForTest()` in wave.go guarded by `waveMu`, or a same-package white-box
  read).
