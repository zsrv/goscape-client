package envelope

import "goscape-client/pkg/jagex2/io"

type Envelope struct {
	Length     int
	ShapeDelta []int
	ShapePeak  []int
	Start      int
	End        int
	Form       int
	Threshold  int
	Position   int
	Delta      int
	Amplitude  int
	Ticks      int
}

func NewEnvelope() *Envelope {
	return new(Envelope)
}

func (e *Envelope) Read(arg1 *io.Packet) {
	e.Form = arg1.G1()
	e.Start = arg1.G4()
	e.End = arg1.G4()
	e.Length = arg1.G1()
	e.ShapeDelta = make([]int, e.Length)
	e.ShapePeak = make([]int, e.Length)
	for i := range e.Length {
		e.ShapeDelta[i] = arg1.G2()
		e.ShapePeak[i] = arg1.G2()
	}
}

func (e *Envelope) Reset() {
	e.Threshold = 0
	e.Position = 0
	e.Delta = 0
	e.Amplitude = 0
	e.Ticks = 0
}

func (e *Envelope) Evaluate(arg1 int) int {
	if e.Ticks >= e.Threshold {
		e.Amplitude = e.ShapePeak[e.Position] << 15
		e.Position++
		if e.Position >= e.Length {
			e.Position = e.Length - 1
		}
		e.Threshold = int(float64(e.ShapeDelta[e.Position]) / 65536.0 * float64(arg1))
		if e.Threshold > e.Ticks {
			e.Delta = ((e.ShapePeak[e.Position] << 15) - e.Amplitude) / (e.Threshold - e.Ticks)
		}
	}
	e.Amplitude += e.Delta
	e.Ticks++
	return (e.Amplitude - e.Delta) >> 15
}
