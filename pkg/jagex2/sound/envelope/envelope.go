package envelope

import "github.com/zsrv/goscape-client/pkg/jagex2/io"

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

// Java: load (Envelope.java:43-48 @32f3062); was unpack in ≤254.
func (e *Envelope) Load(buf *io.Packet) {
	e.Form = buf.G1()
	e.Start = buf.G4()
	e.End = buf.G4()
	e.LoadPoints(buf)
}

// LoadPoints reads the shape point list. NEW split in 274 (pure extraction
// from load, no logic change) so Filter.load can re-read the filterRange
// points mid-stream. Java: loadPoints (Envelope.java:50-59 @32f3062).
func (e *Envelope) LoadPoints(buf *io.Packet) {
	e.Length = buf.G1()
	e.ShapeDelta = make([]int, e.Length)
	e.ShapePeak = make([]int, e.Length)
	for i := range e.Length {
		e.ShapeDelta[i] = buf.G2()
		e.ShapePeak[i] = buf.G2()
	}
}

// GenInit
func (e *Envelope) Reset() {
	e.Threshold = 0
	e.Position = 0
	e.Delta = 0
	e.Amplitude = 0
	e.Ticks = 0
}

// GenNext
func (e *Envelope) Evaluate(delta int) int {
	if e.Ticks >= e.Threshold {
		e.Amplitude = e.ShapePeak[e.Position] << 15
		e.Position++
		if e.Position >= e.Length {
			e.Position = e.Length - 1
		}
		e.Threshold = int(float64(e.ShapeDelta[e.Position]) / 65536.0 * float64(delta))
		if e.Threshold > e.Ticks {
			e.Delta = ((e.ShapePeak[e.Position] << 15) - e.Amplitude) / (e.Threshold - e.Ticks)
		}
	}
	e.Amplitude += e.Delta
	e.Ticks++
	return (e.Amplitude - e.Delta) >> 15
}
