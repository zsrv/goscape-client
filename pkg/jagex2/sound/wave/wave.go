package wave

import (
	"github.com/zsrv/goscape-client/pkg/jagex2/io"
	"github.com/zsrv/goscape-client/pkg/jagex2/sound/tone"
)

var (
	Tracks     []*Wave = make([]*Wave, 1000)
	Delays     []int   = make([]int, 1000)
	WaveBytes  []byte
	WaveBuffer *io.Packet
)

type Wave struct {
	Tones     []*tone.Tone
	LoopBegin int
	LoopEnd   int
}

func NewWave() *Wave {
	return &Wave{
		Tones: make([]*tone.Tone, 10),
	}
}

func Unpack(buf *io.Packet) {
	WaveBytes = make([]byte, 441_000)
	WaveBuffer = io.NewPacket(WaveBytes)

	tone.Init()

	for {
		id := buf.G2()
		if id == 0xFFFF {
			return
		}
		Tracks[id] = NewWave()
		Tracks[id].Read(buf)
		Delays[id] = Tracks[id].Trim()
	}
}

func Generate(loopCount, id int) *io.Packet {
	if Tracks[id] == nil {
		return nil
	}
	wave := Tracks[id]
	return wave.GetWave(loopCount)
}

func (w *Wave) Read(buf *io.Packet) {
	for tn := range 10 {
		hasTone := buf.G1()
		if hasTone != 0 {
			buf.Pos--
			w.Tones[tn] = tone.NewTone()
			w.Tones[tn].Read(buf)
		}
	}

	w.LoopBegin = buf.G2()
	w.LoopEnd = buf.G2()
}

func (w *Wave) Trim() int {
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

func (w *Wave) GetWave(loopCount int) *io.Packet {
	length := w.Generate(loopCount)
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

func (w *Wave) Generate(loopCount int) int {
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
