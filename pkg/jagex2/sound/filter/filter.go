// Package filter ports Java 274's jagex2.sound.Filter — a per-Tone IIR
// (pole/zero pair cascade) filter applied by Tone.generate. The class is NEW
// in 274; nothing in ≤254 corresponds to it.
package filter

import (
	"math"

	"github.com/zsrv/goscape-client/pkg/jagex2/io"
	"github.com/zsrv/goscape-client/pkg/jagex2/sound/envelope"
)

// Package-level scratch shared by all Filter instances, mirroring Java's
// statics (Filter.java:23-32 @32f3062). Sized [2][8]: channel 0 holds the
// feedforward (zero) coefficients, channel 1 the feedback (pole)
// coefficients; up to 4 pairs → 8 coefficients each. Kept int32/float32 to
// mirror Java's int/float exactly — Tone widens to int64 at the product
// sites, as Java casts to long there.
var (
	Coeff          [2][8]float32 // Java: coeff
	CoeffInt       [2][8]int32   // Java: coeffInt
	ReduceCoeff    float32       // Java: reduceCoeff
	ReduceCoeffInt int32         // Java: reduceCoeffInt
)

// Filter holds the serialized pole/zero pair parameters for one Tone.
// Java: Filter (Filter.java @32f3062). Indexing: [channel][interp end][pair]
// where channel 0 = feedforward, 1 = feedback, and interp end 0/1 are the
// envelope-interpolated endpoints.
type Filter struct {
	Pairs       [2]int       // Java: pairs
	Frequencies [2][2][4]int // Java: frequencies
	Ranges      [2][2][4]int // Java: ranges
	Unities     [2]int       // Java: unities
}

func NewFilter() *Filter {
	return new(Filter)
}

// Radius interpolates the stored attenuation range for one pair and returns
// the pole/zero radius. Java: radius(int, int, float) (Filter.java:34-39
// @32f3062).
func (f *Filter) Radius(pair, ch int, scale float32) float32 {
	magnitude := float32(f.Ranges[ch][0][pair]) + scale*float32(f.Ranges[ch][1][pair]-f.Ranges[ch][0][pair]) // Java: var4
	attenuation := magnitude * 0.0015258789                                                                  // Java: var5
	return 1.0 - float32(math.Pow(10.0, float64(-attenuation/20.0)))
}

// AngularFrequency converts an octave offset (relative to C1, 32.703197 Hz)
// to an angular frequency normalised to the 11025 Hz half-rate. Java:
// frequency(float) (Filter.java:41-45 @32f3062) — first of two same-named
// overloads.
func (f *Filter) AngularFrequency(octave float32) float32 {
	hz := float32(math.Pow(2.0, float64(octave))) * 32.703197 // Java: var3
	return hz * 3.1415927 / 11025.0
}

// InterpolatedFrequency interpolates the stored frequency parameter for one
// pair and converts it via AngularFrequency. Java: frequency(int, int, float)
// (Filter.java:47-52 @32f3062) — second of two same-named overloads.
func (f *Filter) InterpolatedFrequency(ch, pair int, scale float32) float32 {
	value := float32(f.Frequencies[ch][0][pair]) + scale*float32(f.Frequencies[ch][1][pair]-f.Frequencies[ch][0][pair]) // Java: var5
	octave := value * 1.2207031e-4                                                                                      // Java: var6
	return f.AngularFrequency(octave)
}

// CalculateCoeffs expands channel ch's pole/zero pairs into the package-level
// polynomial coefficient scratch at interpolation position scale, returning
// the coefficient count (pairs*2). Channel 0 also refreshes the unity-gain
// reduction. Java: calculateCoeffs(int, float) (Filter.java:54-89 @32f3062).
func (f *Filter) CalculateCoeffs(ch int, scale float32) int {
	if ch == 0 {
		unity := float32(f.Unities[0]) + float32(f.Unities[1]-f.Unities[0])*scale // Java: var4
		gain := unity * 0.0030517578                                              // Java: var5
		ReduceCoeff = float32(math.Pow(0.1, float64(gain/20.0)))
		ReduceCoeffInt = int32(ReduceCoeff * 65536.0)
	}
	if f.Pairs[ch] == 0 {
		return 0
	}
	radius := f.Radius(0, ch, scale) // Java: var6
	Coeff[ch][0] = -2.0 * radius * float32(math.Cos(float64(f.InterpolatedFrequency(ch, 0, scale))))
	Coeff[ch][1] = radius * radius
	for pair := 1; pair < f.Pairs[ch]; pair++ { // Java: var7
		r := f.Radius(pair, ch, scale)                                                       // Java: var8
		a := -2.0 * r * float32(math.Cos(float64(f.InterpolatedFrequency(ch, pair, scale)))) // Java: var9
		b := r * r                                                                           // Java: var10
		Coeff[ch][pair*2+1] = Coeff[ch][pair*2-1] * b
		Coeff[ch][pair*2] = Coeff[ch][pair*2-1]*a + Coeff[ch][pair*2-2]*b
		for i := pair*2 - 1; i >= 2; i-- { // Java: var11
			Coeff[ch][i] += Coeff[ch][i-1]*a + Coeff[ch][i-2]*b
		}
		Coeff[ch][1] += Coeff[ch][0]*a + b
		Coeff[ch][0] += a
	}
	if ch == 0 {
		for i := 0; i < f.Pairs[0]*2; i++ { // Java: var12
			Coeff[0][i] *= ReduceCoeff
		}
	}
	for i := 0; i < f.Pairs[ch]*2; i++ { // Java: var13
		CoeffInt[ch][i] = int32(Coeff[ch][i] * 65536.0)
	}
	return f.Pairs[ch] * 2
}

// Load deserializes the filter parameters; when any second interpolation
// endpoint differs from the first, the trailing envelope points are read into
// rng (the owning Tone's filterRange). Java: load(Packet, Envelope)
// (Filter.java:91-122 @32f3062).
func (f *Filter) Load(buf *io.Packet, rng *envelope.Envelope) {
	header := buf.G1() // Java: var4 — pair counts packed as nibbles
	f.Pairs[0] = header >> 4
	f.Pairs[1] = header & 0xF
	if header == 0 {
		f.Unities[0] = 0
		f.Unities[1] = 0
		return
	}
	f.Unities[0] = buf.G2()
	f.Unities[1] = buf.G2()
	migrated := buf.G1() // Java: var6 — bitmask of pairs with distinct second endpoints
	for ch := range 2 {  // Java: var7
		for pair := 0; pair < f.Pairs[ch]; pair++ { // Java: var8
			f.Frequencies[ch][0][pair] = buf.G2()
			f.Ranges[ch][0][pair] = buf.G2()
		}
	}
	for ch := range 2 { // Java: var9
		for pair := 0; pair < f.Pairs[ch]; pair++ { // Java: var10
			if migrated&(1<<(ch*4)<<pair) == 0 {
				f.Frequencies[ch][1][pair] = f.Frequencies[ch][0][pair]
				f.Ranges[ch][1][pair] = f.Ranges[ch][0][pair]
			} else {
				f.Frequencies[ch][1][pair] = buf.G2()
				f.Ranges[ch][1][pair] = buf.G2()
			}
		}
	}
	if migrated != 0 || f.Unities[1] != f.Unities[0] {
		rng.LoadPoints(buf)
	}
}
