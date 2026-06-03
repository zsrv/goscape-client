package audio

import (
	"time"

	"github.com/zsrv/goscape-client/pkg/jagex2/client/sign/signlink"
)

// This file ports the wrapper-side MIDI consumer that the LostCityRS 244
// deob reconstructs in SignLink (SignLink.java:359-425, "adapted from
// play_members.html's JS loop") — the half of the audio pipeline that ran
// outside the signed Java applet and was never in the 225 sources.
//
// Deviations from the Java reference (everything audible is faithful):
//   - Track bytes travel in-memory through the signlink slot
//     (signlink.SetMidiTrack) instead of the jingle<pos>.mid disk round-trip
//     (SignLink.java:327-337) — an applet process-boundary artifact.
//   - The wave branch of audioLoop (SignLink.java:427-475) is not ported
//     into this ticker: SFX play directly (and may overlap) via PlayWave,
//     instead of synchronously blocking the consumer thread for the clip's
//     duration — in Java a playing SFX stalls any in-flight MIDI fade.
//   - MidiPlayer's CC7/CC39/CC121 interception collapses to the linear
//     vol/256 post-gain — see linearVolume.

// midiSink is the per-backend playback primitive set the audioLoop drives,
// mirroring the MidiPlayer surface SignLink uses. Implemented by *midiDriver
// (native: oto + meltysynth) and *webMidiDriver (js: Web Audio prerender);
// nullSink stands in when no audio backend is available.
type midiSink interface {
	// play starts a track at linear gain vol/256, replacing any current
	// one. loop mirrors Java MidiPlayer.play's setLoopCount(loop==1 ? -1 : 0)
	// (MidiPlayer.java:36-44): the fade flag doubles as loop-forever.
	play(midData []byte, loop bool, vol int)
	// stop halts playback immediately (Java: MidiPlayer.stop(),
	// MidiPlayer.java:46-49).
	stop()
	// setVolume applies linear gain vol/256 to the current track
	// (Java: MidiPlayer.setVolume(0, volume), MidiPlayer.java:32-34).
	setVolume(vol int)
	// running reports whether a track is still playing
	// (Java: Sequencer.isRunning() via MidiPlayer.running()).
	running() bool
}

// nullSink drains commands when no backend is available (oto init failure,
// low-memory mode, missing Web Audio). Contrast with Java, where a command
// arriving after a failed MidiPlayer constructor would NPE the wrapper
// thread (SignLink.run has no try/catch around audioLoop) — draining
// harmlessly is strictly better.
type nullSink struct{}

func (nullSink) play([]byte, bool, int) {}
func (nullSink) stop()                  {}
func (nullSink) setVolume(int)          {}
func (nullSink) running() bool          { return false }

// audioLoopInterval is the consumer cadence AND the fade step rate: the Java
// wrapper thread sleeps 50ms per iteration (SignLink.java:197-200), so the
// ±8 fade steps tick every 50ms — a 600ms linear ramp at the default
// midivol 96.
const audioLoopInterval = 50 * time.Millisecond

// audioLoop holds the MIDI fade state machine. Java: SignLink's instance
// fields midiFadingIn/midiFadingOut/midiFadeVol (SignLink.java:360-362).
// Single-goroutine (the runAudioLoop ticker); the signlink slot and the
// sink do their own locking.
type audioLoop struct {
	sink midiSink

	midiFadingIn  bool // Java: midiFadingIn (SignLink.java:360)
	midiFadingOut bool // Java: midiFadingOut (SignLink.java:361)
	midiFadeVol   int  // Java: midiFadeVol (SignLink.java:362)
}

// runAudioLoop ticks the consumer for the process lifetime.
// Java: the tail of SignLink.run's while-loop — audioLoop();
// Thread.sleep(50L) (SignLink.java:195-200).
func runAudioLoop(sink midiSink) {
	l := &audioLoop{sink: sink}
	for {
		l.tick()
		time.Sleep(audioLoopInterval)
	}
}

// tick is one consumer iteration: fade step first, then command dispatch.
// Java: SignLink.audioLoop() (SignLink.java:391-425; the wave branch at
// :427-475 is handled by PlayWave — see the file comment).
func (l *audioLoop) tick() {
	midivol := signlink.ReadMidiVol()

	// Fade step — Java: SignLink.java:392-411.
	if l.midiFadingIn {
		l.midiFadeVol += 8
		if l.midiFadeVol > midivol {
			l.midiFadeVol = midivol
		}
		l.sink.setVolume(l.midiFadeVol)
		if l.midiFadeVol == midivol {
			l.midiFadingIn = false
		}
	} else if l.midiFadingOut {
		l.midiFadeVol -= 8
		if l.midiFadeVol < 0 {
			l.midiFadeVol = 0
		}
		l.sink.setVolume(l.midiFadeVol)
		if l.midiFadeVol == 0 {
			l.midiFadingOut = false
			l.midiFadingIn = true
		}
	}

	// Command dispatch — Java: SignLink.java:413-425. The slot clears only
	// when not fading out: a command latched mid-fade re-dispatches every
	// tick until the fade-out completes (for a track, playMidi then starts
	// it; for stop/voladjust the repeats are idempotent — a faithful quirk).
	cmd, data := signlink.PeekMidi()
	if cmd == "" {
		return
	}
	switch cmd {
	case "stop":
		l.sink.stop()
	case "voladjust":
		l.sink.setVolume(midivol)
	default: // "play"
		l.playMidi(data)
	}
	if !l.midiFadingOut {
		signlink.ClearMidi()
	}
}

// playMidi decides how a pending track starts: ignored (stays latched)
// during fade-out; triggers a fade-out first when fading is on and a track
// is already playing; starts at volume 0 (then ramps in) when arriving out
// of a fade-out; starts at full midivol otherwise — first plays and jingles
// skip the fade entirely.
// Java: SignLink.playMidi (SignLink.java:369-388). The Java catch around
// the play maps to the sink logging-and-returning on a parse failure.
func (l *audioLoop) playMidi(data []byte) {
	midifade := signlink.ReadMidiFade()
	midivol := signlink.ReadMidiVol()
	if l.midiFadingOut {
		return
	}
	if !l.midiFadingIn && midifade != 0 && l.sink.running() {
		l.midiFadingOut = true
		l.midiFadeVol = midivol
		return
	}
	if midifade != 0 && l.midiFadingIn {
		l.midiFadingOut = false
		l.midiFadeVol = 0
		l.sink.play(data, midifade == 1, l.midiFadeVol)
	} else {
		l.sink.play(data, midifade == 1, midivol)
	}
}
