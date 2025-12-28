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
	Buffer = make([]int, 22050*10)
}

func (t *Tone) Generate(samples, length int) []int {
	for i := range samples {
		Buffer[i] = 0
	}

	if length < 10 {
		return Buffer
	}

	samplesPerStep := float64(samples) / (float64(length) + 0.0)

	t.FrequencyBase.Reset()
	t.AmplitudeBase.Reset()

	frequencyStart := 0
	frequencyDuration := 0
	frequencyPhase := 0

	if t.FrequencyModRate != nil {
		t.FrequencyModRate.Reset()
		t.FrequencyModRange.Reset()
		frequencyStart = int(float64(t.FrequencyModRate.End-t.FrequencyModRate.Start) * 32.768 / samplesPerStep)
		frequencyDuration = int(float64(t.FrequencyModRate.Start) * 32.768 / samplesPerStep)
	}

	amplitudeStart := 0
	amplitudeDuration := 0
	amplitudePhase := 0

	if t.AmplitudeModRate != nil {
		t.AmplitudeModRate.Reset()
		t.AmplitudeModRange.Reset()
		amplitudeStart = int(float64(t.AmplitudeModRate.End-t.AmplitudeModRate.Start) * 32.768 / samplesPerStep)
		amplitudeDuration = int(float64(t.AmplitudeModRate.Start) * 32.768 / samplesPerStep)
	}

	for i := range 5 {
		if t.HarmonicVolume[i] != 0 {
			TmpPhases[i] = 0
			TmpDelays[i] = int(float64(t.HarmonicDelay[i]) * samplesPerStep)
			TmpVolumes[i] = (t.HarmonicVolume[i] << 14) / 100
			TmpSemitones[i] = int(float64(t.FrequencyBase.End-t.FrequencyBase.Start) * 32.768 * math.Pow(1.0057929410678534, float64(t.HarmonicSemitone[i])) / samplesPerStep)
			TmpStarts[i] = int(float64(t.FrequencyBase.Start) * 32.768 / samplesPerStep)
		}
	}

	for sample := range samples {
		frequency := t.FrequencyBase.Evaluate(samples)
		amplitude := t.AmplitudeBase.Evaluate(samples)

		if t.FrequencyModRate != nil {
			rate := t.FrequencyModRate.Evaluate(samples)
			rng := t.FrequencyModRange.Evaluate(samples)
			frequency += t.Generate2(rng, frequencyPhase, t.FrequencyModRate.Form) >> 1
			frequencyPhase += ((rate * frequencyStart) >> 16) + frequencyDuration
		}

		if t.AmplitudeModRate != nil {
			rate := t.AmplitudeModRate.Evaluate(samples)
			rng := t.AmplitudeModRange.Evaluate(samples)
			amplitude = (amplitude * ((t.Generate2(rng, amplitudePhase, t.AmplitudeModRate.Form) >> 1) + 32768)) >> 15
			amplitudePhase += ((rate * amplitudeStart) >> 16) + amplitudeDuration
		}

		for harmonic := range 5 {
			if t.HarmonicVolume[harmonic] != 0 {
				pos := sample + TmpDelays[harmonic]
				if pos < samples {
					Buffer[pos] += t.Generate2((amplitude*TmpVolumes[harmonic])>>15, TmpPhases[harmonic], t.FrequencyBase.Form)
					TmpPhases[harmonic] += ((frequency * TmpSemitones[harmonic]) >> 16) + TmpStarts[harmonic]
				}
			}
		}
	}

	if t.Release != nil {
		t.Release.Reset()
		t.Attack.Reset()

		counter := 0
		muted := true

		for sample := range samples {
			releaseValue := t.Release.Evaluate(samples)
			attackValue := t.Attack.Evaluate(samples)

			threshold := 0
			if muted {
				threshold = t.Release.Start + (((t.Release.End - t.Release.Start) * releaseValue) >> 8)
			} else {
				threshold = t.Release.Start + (((t.Release.End - t.Release.Start) * attackValue) >> 8)
			}

			counter += 256
			if counter >= threshold {
				counter = 0
				muted = !muted
			}

			if muted {
				Buffer[sample] = 0
			}
		}
	}

	if t.ReverbDelay > 0 && t.ReverbVolume > 0 {
		start := int(float64(t.ReverbDelay) * samplesPerStep)
		for sample := start; sample < samples; sample++ {
			Buffer[sample] += Buffer[sample-start] * t.ReverbVolume / 100
		}
	}

	for sample := range samples {
		if Buffer[sample] < -32768 {
			Buffer[sample] = -32768
		}

		if Buffer[sample] > 32767 {
			Buffer[sample] = 32767
		}
	}

	return Buffer
}

// WaveFunc
func (t *Tone) Generate2(amplitude, phase, form int) int {
	switch form {
	case 1:
		if phase&0x7FFF < 16384 {
			return amplitude
		}
		return -amplitude
	case 2:
		return (Sin[phase&0x7FFF] * amplitude) >> 14
	case 3:
		return (((phase & 0x7FFF) * amplitude) >> 14) - amplitude
	case 4:
		return Noise[(phase/2607)&0x7FFF] * amplitude
	default:
		return 0
	}
}

// Unpack
func (t *Tone) Read(buf *io.Packet) {
	t.FrequencyBase = envelope.NewEnvelope()
	t.FrequencyBase.Read(buf)

	t.AmplitudeBase = envelope.NewEnvelope()
	t.AmplitudeBase.Read(buf)

	hasFrequencyMod := buf.G1()
	if hasFrequencyMod != 0 {
		buf.Pos--

		t.FrequencyModRate = envelope.NewEnvelope()
		t.FrequencyModRate.Read(buf)
		t.FrequencyModRange = envelope.NewEnvelope()
		t.FrequencyModRange.Read(buf)
	}

	hasAmplitudeMod := buf.G1()
	if hasAmplitudeMod != 0 {
		buf.Pos--

		t.AmplitudeModRate = envelope.NewEnvelope()
		t.AmplitudeModRate.Read(buf)
		t.AmplitudeModRange = envelope.NewEnvelope()
		t.AmplitudeModRange.Read(buf)
	}

	hasReleaseAttack := buf.G1()
	if hasReleaseAttack != 0 {
		buf.Pos--

		t.Release = envelope.NewEnvelope()
		t.Release.Read(buf)
		t.Attack = envelope.NewEnvelope()
		t.Attack.Read(buf)
	}

	for i := range 10 {
		volume := buf.GSmartS()
		if volume == 0 {
			break
		}

		t.HarmonicVolume[i] = volume
		t.HarmonicSemitone[i] = buf.GSmart()
		t.HarmonicDelay[i] = buf.GSmartS()
	}

	t.ReverbDelay = buf.GSmartS()
	t.ReverbVolume = buf.GSmartS()
	t.Length = buf.G2()
	t.Start = buf.G2()
}
