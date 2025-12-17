package wave

import (
	"goscape-client/pkg/jagex2/io"
	"goscape-client/pkg/jagex2/sound/tone"
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

func Unpack(arg0 *io.Packet) {
	WaveBytes = make([]byte, 441_000)
	WaveBuffer = io.NewPacket(WaveBytes)
	tone.Init()
	for {
		var2 := arg0.G2()
		if var2 == 65535 {
			return
		}
		Tracks[var2] = NewWave()
		Tracks[var2].Read(arg0)
		Delays[var2] = Tracks[var2].Trim()
	}
}

func Generate(arg1, arg2 int) *io.Packet {
	if Tracks[arg2] == nil {
		return nil
	}
	var3 := Tracks[arg2]
	return var3.GetWave(arg1)
}

func (w *Wave) Read(arg1 *io.Packet) {
	for i := range 10 {
		var4 := arg1.G1()
		if var4 != 0 {
			arg1.Pos--
			w.Tones[i] = tone.NewTone()
			w.Tones[i].Read(arg1)
		}
	}
	w.LoopBegin = arg1.G2()
	w.LoopEnd = arg1.G2()
}

func (w *Wave) Trim() int {
	var2 := 9999999
	for i := range 10 {
		if w.Tones[i] != nil && w.Tones[i].Start/20 < var2 {
			var2 = w.Tones[i].Start / 20
		}
	}
	if w.LoopBegin < w.LoopEnd && w.LoopBegin/20 < var2 {
		var2 = w.LoopBegin / 20
	}
	if var2 == 9999999 || var2 == 0 {
		return 0
	}
	for i := range 10 {
		if w.Tones[i] != nil {
			w.Tones[i].Start -= var2 * 20
		}
	}
	if w.LoopBegin < w.LoopEnd {
		w.LoopBegin -= var2 * 20
		w.LoopEnd -= var2 * 20
	}
	return var2
}

func (w *Wave) GetWave(arg1 int) *io.Packet {
	var3 := w.Generate(arg1)
	WaveBuffer.Pos = 0
	WaveBuffer.P4(1380533830)
	WaveBuffer.IP4(var3 + 36)
	WaveBuffer.P4(1463899717)
	WaveBuffer.P4(1718449184)
	WaveBuffer.IP4(16)
	WaveBuffer.IP2(1)
	WaveBuffer.IP2(1)
	WaveBuffer.IP4(22050)
	WaveBuffer.IP4(22050)
	WaveBuffer.IP2(1)
	WaveBuffer.IP2(8)
	WaveBuffer.P4(1684108385)
	WaveBuffer.IP4(var3)
	WaveBuffer.Pos += var3
	return WaveBuffer
}

func (w *Wave) Generate(arg0 int) int {
	var2 := 0
	for i := range 10 {
		if w.Tones[i] != nil && w.Tones[i].Length+w.Tones[i].Start > var2 {
			var2 = w.Tones[i].Length + w.Tones[i].Start
		}
	}
	if var2 == 0 {
		return 0
	}
	var4 := var2 * 22050 / 1000
	var5 := w.LoopBegin * 22050 / 1000
	var6 := w.LoopEnd * 22050 / 1000
	if var5 < 0 || var5 > var4 || var6 < 0 || var6 > var4 || var5 >= var6 {
		arg0 = 0
	}
	var7 := var4 + (var6-var5)*(arg0-1)
	for i := 44; i < var7+44; i++ {
		WaveBytes[i] = -128 & 0xFF // TODO: AND is mine, verify behavior
	}
	var10 := 0
	var11 := 0
	for i := range 10 {
		if w.Tones[i] != nil {
			var10 = w.Tones[i].Length * 22050 / 1000
			var11 = w.Tones[i].Start * 22050 / 1000
			var12 := w.Tones[i].Generate(var10, w.Tones[i].Length)
			for j := range var10 {
				WaveBytes[j+var11+44] += byte(var12[j] >> 8)
			}
		}
	}
	if arg0 > 1 {
		var5 += 44
		var6 += 44
		var4 += 44
		var7 += 44
		var10 = var7 - var4
		for i := var4 - 1; i >= var6; i-- {
			WaveBytes[i+var10] = WaveBytes[i]
		}
		for i := 1; i < arg0; i++ {
			var10 = (var6 - var5) * i
			for j := var5; j < var6; j++ {
				WaveBytes[j+var10] = WaveBytes[j]
			}
		}
		var7 -= 44
	}
	return var7
}
