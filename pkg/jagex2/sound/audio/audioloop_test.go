package audio

import (
	"slices"
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/client/sign/signlink"
)

// fakeSink records midiSink calls so tests can assert the exact sequence the
// audioLoop state machine produces.
type playCall struct {
	data []byte
	loop bool
	vol  int
}

type fakeSink struct {
	playCalls []playCall
	stopCalls int
	volCalls  []int
	runningV  bool
}

func (f *fakeSink) play(d []byte, loop bool, vol int) {
	f.playCalls = append(f.playCalls, playCall{data: d, loop: loop, vol: vol})
	f.runningV = true
}
func (f *fakeSink) stop()           { f.stopCalls++; f.runningV = false }
func (f *fakeSink) setVolume(v int) { f.volCalls = append(f.volCalls, v) }
func (f *fakeSink) running() bool   { return f.runningV }

// resetSignlinkAudio restores the signlink audio fields the loop reads,
// before and after each test. The volume is published on the 245.2 centibel
// scale: -400 cB converts to the internal linear 96 (centibelToVol128) that
// the fade sequences below assume — the same audible level as the old 244
// default. (The true 245.2 default is 0 cB = internal 128 = full volume.)
func resetSignlinkAudio(t *testing.T) {
	t.Helper()
	reset := func() {
		signlink.ClearMidi()
		signlink.SetMidiFade(0)
		signlink.SetMidiVol(-400)
	}
	reset()
	t.Cleanup(reset)
}

// TestTrackChangeFadesOutThenIn pins the full Java fade cycle for a track
// change with midifade=1 while music is playing (SignLink.java:369-411):
// fade-out trigger (track latched), 12 ticks of -8 steps to 0, the new track
// starting at volume 0 and looping, then 12 ticks of +8 steps back to 96.
func TestTrackChangeFadesOutThenIn(t *testing.T) {
	resetSignlinkAudio(t)
	signlink.SetMidiFade(1)
	sink := &fakeSink{runningV: true}
	l := &audioLoop{sink: sink}

	signlink.SetMidiTrack([]byte{0xB})
	l.tick() // playMidi: running + fade → start fade-out; track stays latched
	if len(sink.playCalls) != 0 || !l.midiFadingOut {
		t.Fatalf("tick 1: want fade-out latched, got plays=%d fadingOut=%v",
			len(sink.playCalls), l.midiFadingOut)
	}
	if cmd, _ := signlink.PeekMidi(); cmd != "play" {
		t.Fatalf("track must stay latched during fade-out, got %q", cmd)
	}

	for range 12 { // 96/8 = 12 steps down; the 12th flips to fade-in + plays
		l.tick()
	}
	wantDown := []int{88, 80, 72, 64, 56, 48, 40, 32, 24, 16, 8, 0}
	if !slices.Equal(sink.volCalls, wantDown) {
		t.Fatalf("fade-out steps: got %v, want %v", sink.volCalls, wantDown)
	}
	if len(sink.playCalls) != 1 || sink.playCalls[0].vol != 0 || !sink.playCalls[0].loop {
		t.Fatalf("new track must start at vol 0 looping: %+v", sink.playCalls)
	}
	if cmd, _ := signlink.PeekMidi(); cmd != "" {
		t.Fatalf("slot must clear once the fade-out ends, got %q", cmd)
	}

	sink.volCalls = nil
	for range 12 {
		l.tick()
	}
	wantUp := []int{8, 16, 24, 32, 40, 48, 56, 64, 72, 80, 88, 96}
	if !slices.Equal(sink.volCalls, wantUp) {
		t.Fatalf("fade-in steps: got %v, want %v", sink.volCalls, wantUp)
	}
	if l.midiFadingIn || l.midiFadingOut {
		t.Fatalf("fade flags must settle: in=%v out=%v", l.midiFadingIn, l.midiFadingOut)
	}
}

// TestFirstPlayNoFadeStartsAtFullVolume: with nothing running, even a
// midifade=1 track starts immediately at full midivol (the fade only kicks
// in on a CHANGE — Java: the `midiPlayer.running()` condition).
func TestFirstPlayNoFadeStartsAtFullVolume(t *testing.T) {
	resetSignlinkAudio(t)
	signlink.SetMidiFade(1)
	sink := &fakeSink{runningV: false}
	l := &audioLoop{sink: sink}

	signlink.SetMidiTrack([]byte{0xA})
	l.tick()
	if len(sink.playCalls) != 1 {
		t.Fatalf("want immediate play, got %d calls", len(sink.playCalls))
	}
	got := sink.playCalls[0]
	if got.vol != 96 || !got.loop {
		t.Fatalf("first play: vol=%d loop=%v, want vol=96 loop=true", got.vol, got.loop)
	}
	if cmd, _ := signlink.PeekMidi(); cmd != "" {
		t.Fatalf("slot must clear, got %q", cmd)
	}
}

// TestJingleReplacesImmediately: midifade=0 (jingles) replaces the playing
// track at once, non-looping, full volume — no fade at all.
func TestJingleReplacesImmediately(t *testing.T) {
	resetSignlinkAudio(t)
	sink := &fakeSink{runningV: true}
	l := &audioLoop{sink: sink}

	signlink.SetMidiTrack([]byte{0xC})
	l.tick()
	if len(sink.playCalls) != 1 || sink.playCalls[0].loop || sink.playCalls[0].vol != 96 {
		t.Fatalf("jingle: %+v, want one immediate non-loop play at 96", sink.playCalls)
	}
}

// TestStopAndVoladjust: both sentinels dispatch on the next tick and clear
// when not fading out. Stop is immediate (stopMidi zeroes midifade first).
func TestStopAndVoladjust(t *testing.T) {
	resetSignlinkAudio(t)
	sink := &fakeSink{}
	l := &audioLoop{sink: sink}

	signlink.SetMidiCommand("stop")
	l.tick()
	if sink.stopCalls != 1 {
		t.Fatalf("stop: got %d calls, want 1", sink.stopCalls)
	}
	if cmd, _ := signlink.PeekMidi(); cmd != "" {
		t.Fatalf("stop must clear when not fading, got %q", cmd)
	}

	signlink.SetMidiVol(-800) // centibels; internal 64
	signlink.SetMidiCommand("voladjust")
	l.tick()
	if !slices.Equal(sink.volCalls, []int{64}) {
		t.Fatalf("voladjust: got %v, want [64]", sink.volCalls)
	}
}

// TestCommandClobbersLatchedTrack pins two coupled Java behaviors: (1) the
// single slot — a stop issued during a fade-out replaces the latched track,
// which then never plays; (2) the latch — the stop re-dispatches every tick
// until the fade-out completes, and only then clears.
func TestCommandClobbersLatchedTrack(t *testing.T) {
	resetSignlinkAudio(t)
	signlink.SetMidiFade(1)
	sink := &fakeSink{runningV: true}
	l := &audioLoop{sink: sink}

	signlink.SetMidiTrack([]byte{0xB})
	l.tick() // latch track, enter fade-out

	signlink.SetMidiFade(0) // Java stopMidi: midifade = 0, then midi = "stop"
	signlink.SetMidiCommand("stop")
	l.tick()
	l.tick()
	if sink.stopCalls < 2 {
		t.Fatalf("latched stop must re-run during fade-out: got %d", sink.stopCalls)
	}
	if cmd, _ := signlink.PeekMidi(); cmd != "stop" {
		t.Fatalf("stop must stay latched while fading out, got %q", cmd)
	}
	if len(sink.playCalls) != 0 {
		t.Fatalf("clobbered track must never play, got %+v", sink.playCalls)
	}

	for range 12 { // drain the rest of the fade-out
		l.tick()
	}
	if cmd, _ := signlink.PeekMidi(); cmd != "" {
		t.Fatalf("slot must clear after the fade-out completes, got %q", cmd)
	}
}

// TestVoladjustLatchedDuringFadeOut pins the subtlest faithful quirk: a
// voladjust arriving mid-fade-out stays latched (the slot only clears when
// not fading out) and re-dispatches every tick — so each tick calls
// setVolume twice: the fade step (Java SignLink.java:407) then the latched
// voladjust at full midivol (:417). The volume audibly seesaws until the
// fade-out completes; only then does the slot clear.
func TestVoladjustLatchedDuringFadeOut(t *testing.T) {
	resetSignlinkAudio(t)
	signlink.SetMidiFade(1)
	sink := &fakeSink{runningV: true}
	l := &audioLoop{sink: sink}

	signlink.SetMidiTrack([]byte{0xB})
	l.tick() // latch track, enter fade-out

	signlink.SetMidiCommand("voladjust") // clobbers the latched track
	l.tick()
	l.tick()
	wantSeesaw := []int{88, 96, 80, 96} // fade step, voladjust, fade step, voladjust
	if !slices.Equal(sink.volCalls, wantSeesaw) {
		t.Fatalf("seesaw: got %v, want %v", sink.volCalls, wantSeesaw)
	}
	if cmd, _ := signlink.PeekMidi(); cmd != "voladjust" {
		t.Fatalf("voladjust must stay latched while fading out, got %q", cmd)
	}

	for range 10 { // drain the fade-out (72..0 = 10 more steps)
		l.tick()
	}
	if cmd, _ := signlink.PeekMidi(); cmd != "" {
		t.Fatalf("slot must clear after the fade-out completes, got %q", cmd)
	}
}

// TestLinearVolume pins the vol/128 mapping (see linearVolume's doc for the
// calibration rationale): the 244 ladder 128/96/64/32 maps to gains
// 1/0.75/0.5/0.25 — unity at the slider max, matching the TS reference
// client (tinymidipcm.js:313, audio.js:64).
func TestLinearVolume(t *testing.T) {
	cases := []struct {
		vol  int
		want float64
	}{
		{0, 0}, {-8, 0}, {32, 0.25}, {64, 0.5}, {96, 0.75},
		{128, 1}, {256, 1}, {300, 1},
	}
	for _, c := range cases {
		if got := linearVolume(c.vol); got != c.want {
			t.Errorf("linearVolume(%d) = %v, want %v", c.vol, got, c.want)
		}
	}
}

// TestCentibelToVol128 pins the 245.2 centibel→internal-linear conversion
// (see centibelToVol128's doc): the published ladder 0/-400/-800/-1200
// restores the 244 ladder 128/96/64/32 exactly, the 245.2 zero default is
// full volume, and out-of-ladder values clamp to [0,128].
func TestCentibelToVol128(t *testing.T) {
	cases := []struct {
		cb   int
		want int
	}{
		{0, 128}, {-400, 96}, {-800, 64}, {-1200, 32}, // the published ladder
		{-1600, 0},             // exactly silent
		{-2000, 0}, {-9999, 0}, // below the scale: clamp to silence
		{100, 128}, {400, 128}, // above 0 cB never occurs: clamp to unity
	}
	for _, c := range cases {
		if got := centibelToVol128(c.cb); got != c.want {
			t.Errorf("centibelToVol128(%d) = %d, want %d", c.cb, got, c.want)
		}
	}
}
