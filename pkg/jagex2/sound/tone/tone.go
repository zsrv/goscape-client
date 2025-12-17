package tone

import (
	"math"
	"math/rand"

	"goscape-client/pkg/jagex2/io"
	"goscape-client/pkg/jagex2/sound/envelope"
)

var (
	Buffer       []int
	Noise        []int
	Sin          []int
	TmpPhases    []int = make([]int, 5)
	TmpDelays    []int = make([]int, 5)
	TmpVolumes   []int = make([]int, 5)
	TmpSemitones []int = make([]int, 5)
	TmpStarts    []int = make([]int, 5)
)

type Tone struct {
	FrequencyBase     *envelope.Envelope
	AmplitudeBase     *envelope.Envelope
	FrequencyModRate  *envelope.Envelope
	FrequencyModRange *envelope.Envelope
	AmplitudeModRate  *envelope.Envelope
	AmplitudeModRange *envelope.Envelope
	Release           *envelope.Envelope
	Attack            *envelope.Envelope
	HarmonicVolume    []int
	HarmonicSemitone  []int
	HarmonicDelay     []int
	ReverbDelay       int
	ReverbVolume      int
	Length            int
	Start             int
}

func NewTone() *Tone {
	return &Tone{
		HarmonicVolume:   make([]int, 5),
		HarmonicSemitone: make([]int, 5),
		HarmonicDelay:    make([]int, 5),
		ReverbVolume:     100,
		Length:           500,
	}
}

func Init() {
	Noise = make([]int, 32768)
	for i := range 32768 {
		if rand.Float64() > 0.5 {
			Noise[i] = 1
		} else {
			Noise[i] = 0
		}
	}
	Sin = make([]int, 32768)
	for i := range 32768 {
		Sin[i] = int(math.Sin(float64(i)/5215.1903) * 16384.0)
	}
	Buffer = make([]int, 220500)
}

func (t *Tone) Generate(arg0, arg1 int) []int {
	for i := range arg0 {
		Buffer[i] = 0
	}
	if arg1 < 10 {
		return Buffer
	}
	var4 := float64(arg0) / (float64(arg1) + 0.0)
	t.FrequencyBase.Reset()
	t.AmplitudeBase.Reset()
	var6 := 0
	var7 := 0
	var8 := 0
	if t.FrequencyModRate != nil {
		t.FrequencyModRate.Reset()
		t.FrequencyModRange.Reset()
		var6 = int(float64(t.FrequencyModRate.End-t.FrequencyModRate.Start) * 32.768 / var4)
		var7 = int(float64(t.FrequencyModRate.Start) * 32.768 / var4)
	}
	var9 := 0
	var10 := 0
	var11 := 0
	if t.AmplitudeModRate != nil {
		t.AmplitudeModRate.Reset()
		t.AmplitudeModRange.Reset()
		var9 = int(float64(t.AmplitudeModRate.End-t.AmplitudeModRate.Start) * 32.768 / var4)
		var10 = int(float64(t.AmplitudeModRate.Start) * 32.768 / var4)
	}
	for i := range 5 {
		if t.HarmonicVolume[i] != 0 {
			TmpPhases[i] = 0
			TmpDelays[i] = int(float64(t.HarmonicDelay[i]) * var4)
			TmpVolumes[i] = (t.HarmonicVolume[i] << 14) / 100
			TmpSemitones[i] = int(float64(t.FrequencyBase.End-t.FrequencyBase.Start) * 32.768 * math.Pow(1.0057929410678534, float64(t.HarmonicSemitone[i])) / var4)
			TmpStarts[i] = int(float64(t.FrequencyBase.Start) * 32.768 / var4)
		}
	}
	var14 := 0
	var15 := 0
	for i := range arg0 {
		var14 = t.FrequencyBase.Evaluate(arg0)
		var15 = t.AmplitudeBase.Evaluate(arg0)
		if t.FrequencyModRate != nil {
			var16 := t.FrequencyModRate.Evaluate(arg0)
			var17 := t.FrequencyModRange.Evaluate(arg0)
			var14 += t.Generate2(var17, var8, t.FrequencyModRate.Form) >> 1
			var8 += (var16 * var6 >> 16) + var7
		}
		if t.AmplitudeModRate != nil {
			var16 := t.AmplitudeModRate.Evaluate(arg0)
			var17 := t.AmplitudeModRange.Evaluate(arg0)
			var15 = var15 * ((t.Generate2(var17, var11, t.AmplitudeModRate.Form) >> 1) + 32768) >> 15
			var11 += (var16 * var9 >> 16) + var10
		}
		for j := range 5 {
			if t.HarmonicVolume[j] != 0 {
				var17 := i + TmpDelays[j]
				if var17 < arg0 {
					Buffer[var17] += t.Generate2(var15*TmpVolumes[j]>>15, TmpPhases[j], t.FrequencyBase.Form)
					TmpPhases[j] += (var14 * TmpSemitones[j] >> 16) + TmpStarts[j]
				}
			}
		}
	}
	if t.Release != nil {
		t.Release.Reset()
		t.Attack.Reset()
		var14 = 0
		var21 := true
		for i := range arg0 {
			var18 := t.Release.Evaluate(arg0)
			var19 := t.Attack.Evaluate(arg0)
			if var21 {
				var15 = t.Release.Start + ((t.Release.End - t.Release.Start) * var18 >> 8)
			} else {
				var15 = t.Release.Start + ((t.Release.End - t.Release.Start) * var19 >> 8)
			}
			var14 += 256
			if var14 >= var15 {
				var14 = 0
				var21 = !var21
			}
			if var21 {
				Buffer[i] = 0
			}
		}
	}
	if t.ReverbDelay > 0 && t.ReverbVolume > 0 {
		var14 = int(float64(t.ReverbDelay) * var4)
		for i := var14; i < arg0; i++ {
			Buffer[i] += Buffer[i-var14] * t.ReverbVolume / 100
		}
	}
	for i := range arg0 {
		if Buffer[i] < -32768 {
			Buffer[i] = -32768
		}
		if Buffer[i] > 32767 {
			Buffer[i] = 32767
		}
	}
	return Buffer
}

func (t *Tone) Generate2(arg1, arg2, arg3 int) int {
	switch arg3 {
	case 1:
		if arg2&0x7FFF < 16384 {
			return arg1
		}
		return -arg1
	case 2:
		return Sin[arg2&0x7FFF] * arg1 >> 14
	case 3:
		return ((arg2 & 0x7FFF) * arg1 >> 14) - arg1
	case 4:
		return Noise[arg2/2607&0x7FFF] * arg1
	default:
		return 0
	}
}

func (t *Tone) Read(arg1 *io.Packet) {
	t.FrequencyBase = envelope.NewEnvelope()
	t.FrequencyBase.Read(arg1)
	t.AmplitudeBase = envelope.NewEnvelope()
	t.AmplitudeBase.Read(arg1)
	var3 := arg1.G1()
	if var3 != 0 {
		arg1.Pos--
		t.FrequencyModRate = envelope.NewEnvelope()
		t.FrequencyModRate.Read(arg1)
		t.FrequencyModRange = envelope.NewEnvelope()
		t.FrequencyModRange.Read(arg1)
	}
	var3 = arg1.G1()
	if var3 != 0 {
		arg1.Pos--
		t.AmplitudeModRate = envelope.NewEnvelope()
		t.AmplitudeModRate.Read(arg1)
		t.AmplitudeModRange = envelope.NewEnvelope()
		t.AmplitudeModRange.Read(arg1)
	}
	var3 = arg1.G1()
	if var3 != 0 {
		arg1.Pos--
		t.Release = envelope.NewEnvelope()
		t.Release.Read(arg1)
		t.Attack = envelope.NewEnvelope()
		t.Attack.Read(arg1)
	}
	for i := range 10 {
		var5 := arg1.GSmartS()
		if var5 == 0 {
			break
		}
		t.HarmonicVolume[i] = var5
		t.HarmonicSemitone[i] = arg1.GSmart()
		t.HarmonicDelay[i] = arg1.GSmartS()
	}
	t.ReverbDelay = arg1.GSmartS()
	t.ReverbVolume = arg1.GSmartS()
	t.Length = arg1.G2()
	t.Start = arg1.G2()
}
