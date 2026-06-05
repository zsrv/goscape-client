package jagfx

import (
	"github.com/zsrv/goscape-client/pkg/jagex2/io"
	"github.com/zsrv/goscape-client/pkg/jagex2/sound/tone"
)

var (
	// Java: synth (JagFX.java:10 @32f3062) — was tracks in ≤254; Go keeps
	// the descriptive 254-era name.
	Tracks []*JagFX = make([]*JagFX, 1000)
	// Java: delays (JagFX.java:13 @32f3062) — 274 reverts 245.2/254's delay
	// back to 244's delays; Go keeps the singular.
	Delay      []int = make([]int, 1000)
	WaveBytes  []byte
	WaveBuffer *io.Packet
)

type JagFX struct {
	Tones     []*tone.Tone
	LoopBegin int
	LoopEnd   int
}

func NewJagFX() *JagFX {
	return &JagFX{
		Tones: make([]*tone.Tone, 10),
	}
}

// Java: init (JagFX.java:31-45 @32f3062) — was unpack in ≤254; 245.2 moved
// the static waveBytes/waveBuffer initializers in here (lazy), where the Go
// port already allocated them.
func Init(buf *io.Packet) {
	WaveBytes = make([]byte, 441_000)
	WaveBuffer = io.NewPacket(WaveBytes)

	tone.Init()

	for {
		id := buf.G2()
		if id == 0xFFFF {
			return
		}
		Tracks[id] = NewJagFX()
		Tracks[id].Load(buf)
		Delay[id] = Tracks[id].OptimiseStart()
	}
}

// Java: generate (JagFX.java:47-54 @32f3062) — 274 transposes the param
// roles back to (id, loops); 254 was (loops, id), which the Go port never
// adopted (it kept 245.2's (id, loops) as a documented deviation), so Go now
// matches 274 as-is.
func Generate(id, loops int) *io.Packet {
	if Tracks[id] == nil {
		return nil
	}
	sound := Tracks[id] // Java: sound
	return sound.GetWave(loops)
}

// Java: load (JagFX.java:60-72 @32f3062); was read in ≤254.
func (w *JagFX) Load(buf *io.Packet) {
	for tn := range 10 {
		hasTone := buf.G1()
		if hasTone != 0 {
			buf.Pos--
			w.Tones[tn] = tone.NewTone()
			w.Tones[tn].Load(buf)
		}
	}

	w.LoopBegin = buf.G2()
	w.LoopEnd = buf.G2()
}

// Java: optimiseStart (JagFX.java:74-98 @32f3062); was trim in ≤254.
func (w *JagFX) OptimiseStart() int {
	start := 9999999
	for tn := range 10 {
		if w.Tones[tn] != nil && w.Tones[tn].Start/20 < start {
			start = w.Tones[tn].Start / 20
		}
	}

	if w.LoopBegin < w.LoopEnd && w.LoopBegin/20 < start {
		start = w.LoopBegin / 20
	}

	if start == 9999999 || start == 0 {
		return 0
	}

	for tn := range 10 {
		if w.Tones[tn] != nil {
			w.Tones[tn].Start -= start * 20
		}
	}

	if w.LoopBegin < w.LoopEnd {
		w.LoopBegin -= start * 20
		w.LoopEnd -= start * 20
	}

	return start
}

// Java: getWave (JagFX.java:100-119 @32f3062).
func (w *JagFX) GetWave(loopCount int) *io.Packet {
	length := w.MakeSound(loopCount)
	WaveBuffer.Pos = 0
	WaveBuffer.P4(0x52494646)   // "RIFF" ChunkID
	WaveBuffer.IP4(length + 36) // ChunkSize
	WaveBuffer.P4(0x57415645)   // "WAVE" format
	WaveBuffer.P4(0x666d7420)   // "fmt " chunk id
	WaveBuffer.IP4(16)          // chunk size
	WaveBuffer.IP2(1)           // audio format
	WaveBuffer.IP2(1)           // num channels
	WaveBuffer.IP4(22050)       // sample rate
	WaveBuffer.IP4(22050)       // byte rate
	WaveBuffer.IP2(1)           // block align
	WaveBuffer.IP2(8)           // bits per sample
	WaveBuffer.P4(0x64617461)   // "data"
	WaveBuffer.IP4(length)
	WaveBuffer.Pos += length
	return WaveBuffer
}

// Java: makeSound (JagFX.java:121-169 @32f3062); was generate in ≤254
// (overload-collided with the static; 274's fresh deob splits the names).
func (w *JagFX) MakeSound(loopCount int) int {
	duration := 0
	for tn := range 10 {
		if w.Tones[tn] != nil && w.Tones[tn].Length+w.Tones[tn].Start > duration {
			duration = w.Tones[tn].Length + w.Tones[tn].Start
		}
	}

	if duration == 0 {
		return 0
	}

	sampleCount := duration * 22050 / 1000
	loopStart := w.LoopBegin * 22050 / 1000
	loopStop := w.LoopEnd * 22050 / 1000

	if loopStart < 0 || loopStart > sampleCount || loopStop < 0 || loopStop > sampleCount || loopStart >= loopStop {
		loopCount = 0
	}

	totalSampleCount := sampleCount + (loopStop-loopStart)*(loopCount-1)
	for sample := 44; sample < totalSampleCount+44; sample++ {
		WaveBytes[sample] = 0x80 // Java: waveBytes[i] = -128 (signed byte); 0x80 is the unsigned equivalent
	}

	for tn := range 10 {
		if w.Tones[tn] != nil {
			toneSampleCount := w.Tones[tn].Length * 22050 / 1000
			start := w.Tones[tn].Start * 22050 / 1000
			samples := w.Tones[tn].Generate(toneSampleCount, w.Tones[tn].Length)
			for sample := range toneSampleCount {
				WaveBytes[sample+start+44] += byte(samples[sample] >> 8)
			}
		}
	}

	if loopCount > 1 {
		loopStart += 44
		loopStop += 44
		sampleCount += 44
		totalSampleCount += 44

		endOffset := totalSampleCount - sampleCount
		for sample := sampleCount - 1; sample >= loopStop; sample-- {
			WaveBytes[sample+endOffset] = WaveBytes[sample]
		}

		for loop := 1; loop < loopCount; loop++ {
			offset := (loopStop - loopStart) * loop

			for sample := loopStart; sample < loopStop; sample++ {
				WaveBytes[sample+offset] = WaveBytes[sample]
			}
		}

		totalSampleCount -= 44
	}

	return totalSampleCount
}
