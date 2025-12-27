package io

import (
	"strings"

	"goscape-client/pkg/jagex2/io/bzip2"
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

func NewJagfile(arg0 []byte) *Jagfile {
	var j Jagfile
	j.Load(arg0)
	return &j
}

func (jf *Jagfile) Load(arg1 []byte) {
	var3 := NewPacket(arg1)
	var4 := var3.G3()
	var5 := var3.G3()
	if var5 == var4 {
		jf.Buffer = arg1
		jf.Unpacked = false
	} else {
		var6 := make([]byte, var4)
		bzip2.Read(var6, var4, arg1, var5, 6)
		jf.Buffer = var6
		var3 = NewPacket(jf.Buffer)
		jf.Unpacked = true
	}
	jf.FileCount = var3.G2()
	jf.FileHash = make([]int, jf.FileCount)
	jf.FileUnpackedSize = make([]int, jf.FileCount)
	jf.FilePackedSize = make([]int, jf.FileCount)
	jf.FileOffset = make([]int, jf.FileCount)
	var8 := var3.Pos + jf.FileCount*10
	for i := 0; i < jf.FileCount; i++ {
		jf.FileHash[i] = var3.G4()
		jf.FileUnpackedSize[i] = var3.G3()
		jf.FilePackedSize[i] = var3.G3()
		jf.FileOffset[i] = var8
		var8 += jf.FilePackedSize[i]
	}
}

func (jf *Jagfile) Read(arg0 string, arg1 []byte) []byte {
	var4 := int32(0)
	var8 := strings.ToUpper(arg0)
	for i := 0; i < len(var8); i++ {
		var4 = var4*61 + int32(var8[i]) - 32
	}
	for i := 0; i < jf.FileCount; i++ {
		if int32(jf.FileHash[i]) == var4 {
			if arg1 == nil {
				arg1 = make([]byte, jf.FileUnpackedSize[i])
			}
			if jf.Unpacked {
				for j := 0; j < jf.FileUnpackedSize[i]; j++ {
					arg1[j] = jf.Buffer[jf.FileOffset[i]+j]
				}
			} else {
				bzip2.Read(arg1, jf.FileUnpackedSize[i], jf.Buffer, jf.FilePackedSize[i], jf.FileOffset[i])
			}
			return arg1
		}
	}
	return nil
}
