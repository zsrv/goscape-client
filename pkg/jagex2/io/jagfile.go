package io

import (
	"fmt"
	"strings"
)

type Jagfile struct {
	Field734         int8
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
	j.Load(true, arg0)
	return &j
}

func (jf *Jagfile) Load(arg0 bool, dataIn []byte) {
	var3 := NewPacket(dataIn)
	decompressedLength := var3.G3()
	compressedLength := var3.G3()
	if compressedLength == decompressedLength {
		jf.Buffer = dataIn
		jf.Unpacked = false
	} else {
		decompressedData, err := BZip2Decompress(dataIn, int(decompressedLength), false, true)
		if err != nil {
			fmt.Println("BZip2Decompress error:", err)
			return
		}
		jf.Buffer = decompressedData
		var3 = NewPacket(jf.Buffer)
		jf.Unpacked = true
	}
	jf.FileCount = int(var3.G2())
	jf.FileHash = make([]int, jf.FileCount)
	jf.FileUnpackedSize = make([]int, jf.FileCount)
	jf.FilePackedSize = make([]int, jf.FileCount)
	jf.FileOffset = make([]int, jf.FileCount)
	if !arg0 {
		return
	}
	var8 := var3.Pos + jf.FileCount*10
	for i := int(0); i < jf.FileCount; i++ {
		jf.FileHash[i] = var3.G4()
		jf.FileUnpackedSize[i] = var3.G3()
		jf.FilePackedSize[i] = var3.G3()
		jf.FileOffset[i] = var8
		var8 += jf.FilePackedSize[i]
	}
}

func (jf *Jagfile) Read(arg0 string, arg1 []byte) []byte {
	var4 := int(0)
	var8 := strings.ToUpper(arg0)
	for i := 0; i < len(var8); i++ {
		var4 = var4*61 + int(var8[i]) - 32
	}
	for i := 0; i < int(jf.FileCount); i++ {
		if jf.FileHash[i] == var4 {
			if arg1 == nil {
				arg1 = make([]byte, jf.FileUnpackedSize[i])
			}
			if jf.Unpacked {
				for j := 0; j < int(jf.FileUnpackedSize[i]); j++ {
					arg1[j] = jf.Buffer[int(jf.FileOffset[i])+j]
				}
			} else {
				var err error
				arg1, err = BZip2Decompress(jf.Buffer, int(jf.FileUnpackedSize[i]), false, true)
				if err != nil {
					fmt.Println("BZip2Decompress error:", err)
					return nil
				}
			}
			return arg1
		}
	}
	return nil
}
