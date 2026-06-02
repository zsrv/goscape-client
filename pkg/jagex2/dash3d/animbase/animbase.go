package animbase

import "github.com/zsrv/goscape-client/pkg/jagex2/io"

type AnimBase struct {
	Length int
	Types  []int
	Labels [][]int
}

// NewAnimBase ports Java's AnimBase(Packet) (rev-244). In 244 the animation
// base is embedded per-anim blob and decoded straight from a Packet, so there
// is no longer a shared base archive (the 225 base_head/base_type/base_label
// streams and the package-level Instances slice are gone).
// Java: AnimBase.AnimBase(Packet).
func NewAnimBase(buf *io.Packet) *AnimBase {
	size := buf.G1()

	types := make([]int, size)
	labels := make([][]int, size)

	for i := range size {
		types[i] = buf.G1()
	}

	for i := range size {
		count := buf.G1()
		labels[i] = make([]int, count)

		for j := range count {
			labels[i][j] = buf.G1()
		}
	}

	return &AnimBase{Length: size, Types: types, Labels: labels}
}
