package io

import (
	"strings"

	"github.com/zsrv/goscape-client/pkg/jagex2/io/bzip2"
)

type Jagfile struct {
	Buffer           []byte
	FileCount        int
	FileHash         []int
	FileUnpackedSize []int
	FilePackedSize   []int
	FileOffset       []int
	Unpacked         bool
}

func NewJagfile(src []byte) *Jagfile {
	var j Jagfile
	j.Load(src)
	return &j
}

// Unpack
func (jf *Jagfile) Load(src []byte) {
	data := NewPacket(src)
	unpackedSize := data.G3()
	packedSize := data.G3()

	if packedSize == unpackedSize {
		jf.Buffer = src
		jf.Unpacked = false
	} else {
		temp := make([]byte, unpackedSize)
		bzip2.Read(temp, unpackedSize, src, packedSize, 6)
		jf.Buffer = temp

		data = NewPacket(jf.Buffer)
		jf.Unpacked = true
	}

	jf.FileCount = data.G2()
	jf.FileHash = make([]int, jf.FileCount)
	jf.FileUnpackedSize = make([]int, jf.FileCount)
	jf.FilePackedSize = make([]int, jf.FileCount)
	jf.FileOffset = make([]int, jf.FileCount)

	pos := data.Pos + jf.FileCount*10
	for i := range jf.FileCount {
		jf.FileHash[i] = data.G4()
		jf.FileUnpackedSize[i] = data.G3()
		jf.FilePackedSize[i] = data.G3()
		jf.FileOffset[i] = pos
		pos += jf.FilePackedSize[i]
	}
}

func (jf *Jagfile) Read(name string, dst []byte) []byte {
	// Java: Jagfile.read walks arg1.toUpperCase().charAt(i) (UTF-16 code units).
	// Archive entry names are ASCII-only by convention, but iterating runes
	// matches Java's semantics for any BMP input.
	hash := int32(0)
	upper := []rune(strings.ToUpper(name))
	for i := range len(upper) {
		hash = hash*61 + int32(upper[i]) - 32
	}

	for i := range jf.FileCount {
		if int32(jf.FileHash[i]) == hash {
			if dst == nil {
				dst = make([]byte, jf.FileUnpackedSize[i])
			}

			if jf.Unpacked {
				for j := range jf.FileUnpackedSize[i] {
					dst[j] = jf.Buffer[jf.FileOffset[i]+j]
				}
			} else {
				bzip2.Read(dst, jf.FileUnpackedSize[i], jf.Buffer, jf.FilePackedSize[i], jf.FileOffset[i])
			}

			return dst
		}
	}

	return nil
}
