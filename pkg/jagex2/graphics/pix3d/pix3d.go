package pix3d

import (
	"math"
	"math/rand/v2"
	"strconv"

	"goscape-client/pkg/jagex2/graphics/pix2d"
	"goscape-client/pkg/jagex2/graphics/pix8"
	"goscape-client/pkg/jagex2/io"
)

var (
	LowDetail          bool         = true
	Jagged             bool         = true
	DivTable           []int        = make([]int, 512)
	DivTable2          []int        = make([]int, 2048)
	SinTable           []int        = make([]int, 2048)
	CosTable           []int        = make([]int, 2048)
	Textures           []*pix8.Pix8 = make([]*pix8.Pix8, 50)
	TextureTranslucent []bool       = make([]bool, 50)
	AverageTextureRGB  []int        = make([]int, 50)
	ActiveTexels       [][]int      = make([][]int, 50)
	TextureCycle       []int        = make([]int, 50)
	ColourTable        []int        = make([]int, 65536)
	TexturePalette     [][]int      = make([][]int, 50)
	Trans              int
	CenterW3D          int
	CenterH3D          int
	TextureCount       int
	PoolSize           int
	Cycle              int
	HClip              bool
	Opaque             bool
	LineOffset         []int
	TexelPool          [][]int
)

func init() {
	for i := 1; i < 512; i++ {
		DivTable[i] = 32768 / i
	}
	for i := 1; i < 2048; i++ {
		DivTable2[i] = 65536 / i
	}
	for i := 0; i < 2048; i++ {
		SinTable[i] = int(math.Sin(float64(i)*0.0030679615) * 65536.0)
		CosTable[i] = int(math.Cos(float64(i)*0.0030679615) * 65536.0)
	}
}

func Unload() {
	DivTable = nil
	DivTable = nil
	SinTable = nil
	CosTable = nil
	LineOffset = nil
	Textures = nil
	TextureTranslucent = nil
	AverageTextureRGB = nil
	TexelPool = nil
	ActiveTexels = nil
	TextureCycle = nil
	ColourTable = nil
	TexturePalette = nil
}

func Init2D() {
	LineOffset = make([]int, pix2d.Height2D)
	for i := range pix2d.Height2D {
		LineOffset[i] = pix2d.Width2D * i
	}
	CenterW3D = pix2d.Width2D / 2
	CenterH3D = pix2d.Height2D / 2
}

func Init3D(arg0 int, arg1 int) {
	LineOffset = make([]int, arg0)
	for i := range arg0 {
		LineOffset[i] = arg1 * i
	}
	CenterW3D = arg1 / 2
	CenterH3D = arg0 / 2
}

func ClearTexels() {
	TexelPool = nil
	for i := range 50 {
		ActiveTexels[i] = nil
	}
}

func InitPool(size int) {
	if TexelPool != nil {
		return
	}
	PoolSize = size
	if LowDetail {
		TexelPool = make([][]int, PoolSize)
		for i := range TexelPool {
			TexelPool[i] = make([]int, 16384)
		}
	} else {
		TexelPool = make([][]int, PoolSize)
		for i := range TexelPool {
			TexelPool[i] = make([]int, 65536)
		}
	}
	for i := range 50 {
		ActiveTexels[i] = nil
	}
}

func UnpackTextures(jag *io.Jagfile) {
	TextureCount = 0
	for i := range 50 {
		Textures[i] = pix8.NewPix8(jag, strconv.Itoa(i), 0)
		if LowDetail && Textures[i].CropW == 128 {
			Textures[i].Shrink()
		} else {
			Textures[i].Crop()
		}
		TextureCount++
	}
}

func GetAverageTextureRGB(arg1 int) int {
	if AverageTextureRGB[arg1] != 0 {
		return AverageTextureRGB[arg1]
	}
	var2 := 0
	var3 := 0
	var4 := 0
	var5 := len(TexturePalette[arg1])
	for i := range var5 {
		var2 += (TexturePalette[arg1][i] >> 16) & 0xFF
		var3 += (TexturePalette[arg1][i] >> 8) & 0xFF
		var4 += TexturePalette[arg1][i] & 0xFF
	}
	var7 := ((var2 / var5) << 16) + ((var3 / var5) << 8) + var4/var5
	var7 = SetGamma(var7, 1.4)
	if var7 == 0 {
		var7 = 1
	}
	AverageTextureRGB[arg1] = var7
	return var7
}

func PushTexture(arg0 int) {
	if ActiveTexels[arg0] != nil {
		TexelPool[PoolSize] = ActiveTexels[arg0]
		PoolSize++
		ActiveTexels[arg0] = nil
	}
}

func GetTexels(arg0 int) []int {
	TextureCycle[arg0] = Cycle
	Cycle++
	if ActiveTexels[arg0] != nil {
		return ActiveTexels[arg0]
	}
	var var1 []int
	if PoolSize > 0 {
		PoolSize--
		var1 = TexelPool[PoolSize]
		TexelPool[PoolSize] = nil
	} else {
		var2 := 0
		var3 := -1
		for i := range TextureCount {
			if ActiveTexels[i] != nil && (TextureCycle[i] < var2 || var3 == -1) {
				var2 = TextureCycle[i]
				var3 = i
			}
		}
		var1 = ActiveTexels[var3]
		ActiveTexels[var3] = nil
	}
	ActiveTexels[arg0] = var1
	var6 := Textures[arg0]
	var7 := TexturePalette[arg0]
	if LowDetail {
		TextureTranslucent[arg0] = false
		for i := range 4096 {
			var1[i] = var7[var6.Pixels[i]] & 0xF8F8FF
			var5 := var1[i]
			if var5 == 0 {
				TextureTranslucent[arg0] = true
			}
			var1[i+4096] = (var5 - (var5 >> 3)) & 0xF8F8FF
			var1[i+8192] = (var5 - (var5 >> 2)) & 0xF8F8FF
			var1[i+12288] = (var5 - (var5 >> 2) - (var5 >> 3)) & 0xF8F8FF
		}
	} else {
		if var6.Width == 64 {
			for i := range 128 {
				for j := range 128 {
					var1[j+(i<<7)] = var7[var6.Pixels[(j>>1)+((i>>1)<<6)]]
				}
			}
		} else {
			for i := range 16384 {
				var1[i] = var7[var6.Pixels[i]]
			}
		}
		TextureTranslucent[arg0] = false
		for i := range 16384 {
			var1[i] &= 0xF8F8FF
			var8 := var1[i]
			if var8 == 0 {
				TextureTranslucent[arg0] = true
			}
			var1[i+16384] = (var8 - (var8 >> 3)) & 0xF8F8FF
			var1[i+32768] = (var8 - (var8 >> 2)) & 0xF8F8FF
			var1[i+49152] = (var8 - (var8 >> 2) - (var8 >> 3)) & 0xF8F8FF
		}
	}
	return var1
}

func SetBrightness(arg1 float64) {
	var28 := arg1 + (rand.Float64()*0.03 - 0.015)
	var3 := 0
	for i := range 512 {
		var5 := float64(i/8)/64.0 + 0.0078125
		var7 := float64(i&0x7)/8.0 + 0.0625
		for j := range 128 {
			var10 := float64(j) / 128.0
			var12 := var10
			var14 := var10
			var16 := var10
			if var7 != 0.0 {
				var18 := 0.0
				if var10 < 0.5 {
					var18 = var10 * (var7 + 1.0)
				} else {
					var18 = var10 + var7 - var10*var7
				}
				var20 := var10*2.0 - var18
				var22 := var5 + 0.3333333333333333
				if var22 > 1.0 {
					var22--
				}
				var26 := var5 - 0.3333333333333333
				if var26 < 0.0 {
					var26++
				}
				if var22*6.0 < 1.0 {
					var12 = var20 + (var18-var20)*6.0*var22
				} else if var22*2.0 < 1.0 {
					var12 = var18
				} else if var22*3.0 < 2.0 {
					var12 = var20 + (var18-var20)*(0.6666666666666666-var22)*6.0
				} else {
					var12 = var20
				}
				if var5*6.0 < 1.0 {
					var14 = var20 + (var18-var20)*6.0*var5
				} else if var5*2.0 < 1.0 {
					var14 = var18
				} else if var5*3.0 < 2.0 {
					var14 = var20 + (var18-var20)*(0.6666666666666666-var5)*6.0
				} else {
					var14 = var20
				}
				if var26*6.0 < 1.0 {
					var16 = var20 + (var18-var20)*6.0*var26
				} else if var26*2.0 < 1.0 {
					var16 = var18
				} else if var26*3.0 < 2.0 {
					var16 = var20 + (var18-var20)*(0.6666666666666666-var26)*6.0
				} else {
					var16 = var20
				}
			}
			var32 := var12 * 256.0
			var19 := var14 * 256.0
			var33 := var16 * 256.0
			var21 := (int(var32) << 16) + (int(var19) << 8) + int(var33) // TODO: verify
			var34 := SetGamma(var21, var28)
			ColourTable[var3] = var34
			var3++
		}
	}
	for i := range 50 {
		if Textures[i] != nil {
			var6 := Textures[i].Palette
			TexturePalette[i] = make([]int, len(var6))
			for j := range len(var6) {
				TexturePalette[i][j] = SetGamma(var6[j], var28)
			}
		}
	}
	for i := range 50 {
		PushTexture(i)
	}
}

func SetGamma(arg0 int, arg1 float64) int {
	var3 := float64((arg0 >> 16) / 256.0)
	var5 := float64(((arg0 >> 8) & 0xFF) / 256.0)
	var7 := float64((arg0 & 0xFF) / 256.0)
	var12 := math.Pow(var3, arg1)
	var13 := math.Pow(var5, arg1)
	var14 := math.Pow(var7, arg1)
	var9 := int(var12 * 256.0)
	var10 := int(var13 * 256.0)
	var11 := int(var14 * 256.0)
	return (var9 << 16) + (var10 << 8) + var11
}

func GouraudTriangle(arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7, arg8 int) {
	var9 := 0
	var10 := 0
	if arg1 != arg0 {
		var9 = ((arg4 - arg3) << 16) / (arg1 - arg0)
		var10 = ((arg7 - arg6) << 15) / (arg1 - arg0)
	}
	var11 := 0
	var12 := 0
	if arg2 != arg1 {
		var11 = ((arg5 - arg4) << 16) / (arg2 - arg1)
		var12 = ((arg8 - arg7) << 15) / (arg2 - arg1)
	}
	var13 := 0
	var14 := 0
	if arg2 != arg0 {
		var13 = ((arg3 - arg5) << 16) / (arg0 - arg2)
		var14 = ((arg6 - arg8) << 15) / (arg0 - arg2)
	}
	if arg0 <= arg1 && arg0 <= arg2 {
		if arg0 < pix2d.BoundBottom {
			if arg1 > pix2d.BoundBottom {
				arg1 = pix2d.BoundBottom
			}
			if arg2 > pix2d.BoundBottom {
				arg2 = pix2d.BoundBottom
			}
			if arg1 < arg2 {
				arg3 <<= 0x10
				arg5 = arg3
				arg6 <<= 0xF
				arg8 = arg6
				if arg0 < 0 {
					arg5 -= var13 * arg0
					arg3 -= var9 * arg0
					arg8 -= var14 * arg0
					arg6 -= var10 * arg0
					arg0 = 0
				}
				arg4 <<= 0x10
				arg7 <<= 0xF
				if arg1 < 0 {
					arg4 -= var11 * arg1
					arg7 -= var12 * arg1
					arg1 = 0
				}
				if arg0 != arg1 && var13 < var9 || arg0 == arg1 && var13 > var11 {
					arg2 -= arg1
					arg1 -= arg0
					arg0 = LineOffset[arg0]
					for {
						arg1--
						if arg1 < 0 {
							for {
								arg2--
								if arg2 < 0 {
									return
								}
								GouraudRaster(pix2d.Data, arg0, arg5>>16, arg4>>16, arg8>>7, arg7>>7)
								arg5 += var13
								arg4 += var11
								arg8 += var14
								arg7 += var12
								arg0 += pix2d.Width2D
							}
						}
						GouraudRaster(pix2d.Data, arg0, arg5>>16, arg3>>16, arg8>>7, arg6>>7)
						arg5 += var13
						arg3 += var9
						arg8 += var14
						arg6 += var10
						arg0 += pix2d.Width2D
					}
				} else {
					arg2 -= arg1
					arg1 -= arg0
					arg0 = LineOffset[arg0]
					for {
						arg1--
						if arg1 < 0 {
							for {
								arg2--
								if arg2 < 0 {
									return
								}
								GouraudRaster(pix2d.Data, arg0, arg4>>16, arg5>>16, arg7>>7, arg8>>7)
								arg5 += var13
								arg4 += var11
								arg8 += var14
								arg7 += var12
								arg0 += pix2d.Width2D
							}
						}
						GouraudRaster(pix2d.Data, arg0, arg3>>16, arg5>>16, arg6>>7, arg8>>7)
						arg5 += var13
						arg3 += var9
						arg8 += var14
						arg6 += var10
						arg0 += pix2d.Width2D
					}
				}
			} else {
				arg3 <<= 0x10
				arg4 = arg3
				arg6 <<= 0xF
				arg7 = arg6
				if arg0 < 0 {
					arg4 -= var13 * arg0
					arg3 -= var9 * arg0
					arg7 -= var14 * arg0
					arg6 -= var10 * arg0
					arg0 = 0
				}
				arg5 <<= 0x10
				arg8 <<= 0xF
				if arg2 < 0 {
					arg5 -= var11 * arg2
					arg8 -= var12 * arg2
					arg2 = 0
				}
				if arg0 != arg2 && var13 < var9 || arg0 == arg2 && var11 > var9 {
					arg1 -= arg2
					arg2 -= arg0
					arg0 = LineOffset[arg0]
					for {
						arg2--
						if arg2 < 0 {
							for {
								arg1--
								if arg1 < 0 {
									return
								}
								GouraudRaster(pix2d.Data, arg0, arg5>>16, arg3>>16, arg8>>7, arg6>>7)
								arg5 += var11
								arg3 += var9
								arg8 += var12
								arg6 += var10
								arg0 += pix2d.Width2D
							}
						}
						GouraudRaster(pix2d.Data, arg0, arg4>>16, arg3>>16, arg7>>7, arg6>>7)
						arg4 += var13
						arg3 += var9
						arg7 += var14
						arg6 += var10
						arg0 += pix2d.Width2D
					}
				} else {
					arg1 -= arg2
					arg2 -= arg0
					arg0 = LineOffset[arg0]
					for {
						arg2--
						if arg2 < 0 {
							for {
								arg1--
								if arg1 < 0 {
									return
								}
								GouraudRaster(pix2d.Data, arg0, arg3>>16, arg5>>16, arg6>>7, arg8>>7)
								arg5 += var11
								arg3 += var9
								arg8 += var12
								arg6 += var10
								arg0 += pix2d.Width2D
							}
						}
						GouraudRaster(pix2d.Data, arg0, arg3>>16, arg4>>16, arg6>>7, arg7>>7)
						arg4 += var13
						arg3 += var9
						arg7 += var14
						arg6 += var10
						arg0 += pix2d.Width2D
					}
				}
			}
		}
	} else if arg1 <= arg2 {
		if arg1 < pix2d.BoundBottom {
			if arg2 > pix2d.BoundBottom {
				arg2 = pix2d.BoundBottom
			}
			if arg0 > pix2d.BoundBottom {
				arg0 = pix2d.BoundBottom
			}
			if arg2 < arg0 {
				arg4 <<= 0x10
				arg3 = arg4
				arg7 <<= 0xF
				arg6 = arg7
				if arg1 < 0 {
					arg3 -= var9 * arg1
					arg4 -= var11 * arg1
					arg6 -= var10 * arg1
					arg7 -= var12 * arg1
					arg1 = 0
				}
				arg5 <<= 0x10
				arg8 <<= 0xF
				if arg2 < 0 {
					arg5 -= var13 * arg2
					arg8 -= var14 * arg2
					arg2 = 0
				}
				if arg1 != arg2 && var9 < var11 || arg1 == arg2 && var9 > var13 {
					arg0 -= arg2
					arg2 -= arg1
					arg1 = LineOffset[arg1]
					for {
						arg2--
						if arg2 < 0 {
							for {
								arg0--
								if arg0 < 0 {
									return
								}
								GouraudRaster(pix2d.Data, arg1, arg3>>16, arg5>>16, arg6>>7, arg8>>7)
								arg3 += var9
								arg5 += var13
								arg6 += var10
								arg8 += var14
								arg1 += pix2d.Width2D
							}
						}
						GouraudRaster(pix2d.Data, arg1, arg3>>16, arg4>>16, arg6>>7, arg7>>7)
						arg3 += var9
						arg4 += var11
						arg6 += var10
						arg7 += var12
						arg1 += pix2d.Width2D
					}
				} else {
					arg0 -= arg2
					arg2 -= arg1
					arg1 = LineOffset[arg1]
					for {
						arg2--
						if arg2 < 0 {
							for {
								arg0--
								if arg0 < 0 {
									return
								}
								GouraudRaster(pix2d.Data, arg1, arg5>>16, arg3>>16, arg8>>7, arg6>>7)
								arg3 += var9
								arg5 += var13
								arg6 += var10
								arg8 += var14
								arg1 += pix2d.Width2D
							}
						}
						GouraudRaster(pix2d.Data, arg1, arg4>>16, arg3>>16, arg7>>7, arg6>>7)
						arg3 += var9
						arg4 += var11
						arg6 += var10
						arg7 += var12
						arg1 += pix2d.Width2D
					}
				}
			} else {
				arg4 <<= 0x10
				arg5 = arg4
				arg7 <<= 0xF
				arg8 = arg7
				if arg1 < 0 {
					arg5 -= var9 * arg1
					arg4 -= var11 * arg1
					arg8 -= var10 * arg1
					arg7 -= var12 * arg1
					arg1 = 0
				}
				arg3 <<= 0x10
				arg6 <<= 0xF
				if arg0 < 0 {
					arg3 -= var13 * arg0
					arg6 -= var14 * arg0
					arg0 = 0
				}
				if var9 < var11 {
					arg2 -= arg0
					arg0 -= arg1
					arg1 = LineOffset[arg1]
					for {
						arg0--
						if arg0 < 0 {
							for {
								arg2--
								if arg2 < 0 {
									return
								}
								GouraudRaster(pix2d.Data, arg1, arg3>>16, arg4>>16, arg6>>7, arg7>>7)
								arg3 += var13
								arg4 += var11
								arg6 += var14
								arg7 += var12
								arg1 += pix2d.Width2D
							}
						}
						GouraudRaster(pix2d.Data, arg1, arg5>>16, arg4>>16, arg8>>7, arg7>>7)
						arg5 += var9
						arg4 += var11
						arg8 += var10
						arg7 += var12
						arg1 += pix2d.Width2D
					}
				} else {
					arg2 -= arg0
					arg0 -= arg1
					arg1 = LineOffset[arg1]
					for {
						arg0--
						if arg0 < 0 {
							for {
								arg2--
								if arg2 < 0 {
									return
								}
								GouraudRaster(pix2d.Data, arg1, arg4>>16, arg3>>16, arg7>>7, arg6>>7)
								arg3 += var13
								arg4 += var11
								arg6 += var14
								arg7 += var12
								arg1 += pix2d.Width2D
							}
						}
						GouraudRaster(pix2d.Data, arg1, arg4>>16, arg5>>16, arg7>>7, arg8>>7)
						arg5 += var9
						arg4 += var11
						arg8 += var10
						arg7 += var12
						arg1 += pix2d.Width2D
					}
				}
			}
		}
	} else if arg2 < pix2d.BoundBottom {
		if arg0 > pix2d.BoundBottom {
			arg0 = pix2d.BoundBottom
		}
		if arg1 > pix2d.BoundBottom {
			arg1 = pix2d.BoundBottom
		}
		if arg0 < arg1 {
			arg5 <<= 0x10
			arg4 = arg5
			arg8 <<= 0xF
			arg7 = arg8
			if arg2 < 0 {
				arg4 -= var11 * arg2
				arg5 -= var13 * arg2
				arg7 -= var12 * arg2
				arg8 -= var14 * arg2
				arg2 = 0
			}
			arg3 <<= 0x10
			arg6 <<= 0xF
			if arg0 < 0 {
				arg3 -= var9 * arg0
				arg6 -= var10 * arg0
				arg0 = 0
			}
			if var11 < var13 {
				arg1 -= arg0
				arg0 -= arg2
				arg2 = LineOffset[arg2]
				for {
					arg0--
					if arg0 < 0 {
						for {
							arg1--
							if arg1 < 0 {
								return
							}
							GouraudRaster(pix2d.Data, arg2, arg4>>16, arg3>>16, arg7>>7, arg6>>7)
							arg4 += var11
							arg3 += var9
							arg7 += var12
							arg6 += var10
							arg2 += pix2d.Width2D
						}
					}
					GouraudRaster(pix2d.Data, arg2, arg4>>16, arg5>>16, arg7>>7, arg8>>7)
					arg4 += var11
					arg5 += var13
					arg7 += var12
					arg8 += var14
					arg2 += pix2d.Width2D
				}
			} else {
				arg1 -= arg0
				arg0 -= arg2
				arg2 = LineOffset[arg2]
				for {
					arg0--
					if arg0 < 0 {
						for {
							arg1--
							if arg1 < 0 {
								return
							}
							GouraudRaster(pix2d.Data, arg2, arg3>>16, arg4>>16, arg6>>7, arg7>>7)
							arg4 += var11
							arg3 += var9
							arg7 += var12
							arg6 += var10
							arg2 += pix2d.Width2D
						}
					}
					GouraudRaster(pix2d.Data, arg2, arg5>>16, arg4>>16, arg8>>7, arg7>>7)
					arg4 += var11
					arg5 += var13
					arg7 += var12
					arg8 += var14
					arg2 += pix2d.Width2D
				}
			}
		} else {
			arg5 <<= 0x10
			arg3 = arg5
			arg8 <<= 0xF
			arg6 = arg8
			if arg2 < 0 {
				arg3 -= var11 * arg2
				arg5 -= var13 * arg2
				arg6 -= var12 * arg2
				arg8 -= var14 * arg2
				arg2 = 0
			}
			arg4 <<= 0x10
			arg7 <<= 0xF
			if arg1 < 0 {
				arg4 -= var9 * arg1
				arg7 -= var10 * arg1
				arg1 = 0
			}
			if var11 < var13 {
				arg0 -= arg1
				arg1 -= arg2
				arg2 = LineOffset[arg2]
				for {
					arg1--
					if arg1 < 0 {
						for {
							arg0--
							if arg0 < 0 {
								return
							}
							GouraudRaster(pix2d.Data, arg2, arg4>>16, arg5>>16, arg7>>7, arg8>>7)
							arg4 += var9
							arg5 += var13
							arg7 += var10
							arg8 += var14
							arg2 += pix2d.Width2D
						}
					}
					GouraudRaster(pix2d.Data, arg2, arg3>>16, arg5>>16, arg6>>7, arg8>>7)
					arg3 += var11
					arg5 += var13
					arg6 += var12
					arg8 += var14
					arg2 += pix2d.Width2D
				}
			} else {
				arg0 -= arg1
				arg1 -= arg2
				arg2 = LineOffset[arg2]
				for {
					arg1--
					if arg1 < 0 {
						for {
							arg0--
							if arg0 < 0 {
								return
							}
							GouraudRaster(pix2d.Data, arg2, arg5>>16, arg4>>16, arg8>>7, arg7>>7)
							arg4 += var9
							arg5 += var13
							arg7 += var10
							arg8 += var14
							arg2 += pix2d.Width2D
						}
					}
					GouraudRaster(pix2d.Data, arg2, arg5>>16, arg3>>16, arg8>>7, arg6>>7)
					arg3 += var11
					arg5 += var13
					arg6 += var12
					arg8 += var14
					arg2 += pix2d.Width2D
				}
			}
		}
	}
}

func GouraudRaster(arg0 []int, arg1, arg4, arg5, arg6, arg7 int) {
	var8 := 0
	var9 := 0
	var10 := 0
	var11 := 0
	var17 := 0
	if Jagged {
		arg3 := 0
		if HClip {
			if arg5-arg4 > 3 {
				var8 = (arg7 - arg6) / (arg5 - arg4)
			} else {
				var8 = 0
			}
			if arg5 > pix2d.SafeWidth {
				arg5 = pix2d.SafeWidth
			}
			if arg4 < 0 {
				arg6 -= arg4 * var8
				arg4 = 0
			}
			if arg4 >= arg5 {
				return
			}
			arg1 += arg4
			arg3 = (arg5 - arg4) >> 2
			var8 <<= 0x2
		} else if arg4 < arg5 {
			arg1 += arg4
			arg3 = (arg5 - arg4) >> 2
			if arg3 > 0 {
				var8 = ((arg7 - arg6) * DivTable[arg3]) >> 15
			} else {
				var8 = 0
			}
		} else {
			return
		}
		var13 := 0
		if Trans == 0 {
			for {
				arg3--
				if arg3 < 0 {
					var17 = (arg5 - arg4) & 0x3
					if var17 > 0 {
						var11 = ColourTable[arg6>>8]
						for ok := true; ok; ok = var17 > 0 {
							arg0[arg1] = var11
							arg1++
							var17--
						}
						return
					}
					break
				}
				var11 = ColourTable[arg6>>8]
				arg6 += var8
				var13 = arg1 + 1
				arg0[arg1] = var11
				var14 := var13 + 1
				arg0[var13] = var11
				var15 := var14 + 1
				arg0[var14] = var11
				arg1 = var15 + 1
				arg0[var15] = var11
			}
		} else {
			var9 = Trans
			var10 = 256 - Trans
			for {
				arg3--
				if arg3 < 0 {
					var17 = (arg5 - arg4) & 0x3
					if var17 > 0 {
						var11 = ColourTable[arg6>>8]
						var11 = ((((var11 & 0xFF00FF) * var10) >> 8) & 0xFF00FF) + ((((var11 & 0xFF00) * var10) >> 8) & 0xFF00)
						for ok := true; ok; ok = var17 > 0 {
							arg1++
							arg0[arg1-1] = var11 + ((((arg0[arg1] & 0xFF00FF) * var9) >> 8) & 0xFF00FF) + ((((arg0[arg1] & 0xFF00) * var9) >> 8) & 0xFF00)
							var17--
						}
					}
					break
				}
				var11 = ColourTable[arg6>>8]
				arg6 += var8
				var11 = ((((var11 & 0xFF00FF) * var10) >> 8) & 0xFF00FF) + ((((var11 & 0xFF00) * var10) >> 8) & 0xFF00)
				var13 = arg1 + 1
				arg0[arg1] = var11 + ((((arg0[var13] & 0xFF00FF) * var9) >> 8) & 0xFF00FF) + ((((arg0[var13] & 0xFF00) * var9) >> 8) & 0xFF00)
				var13++
				arg0[var13-1] = var11 + ((((arg0[var13] & 0xFF00FF) * var9) >> 8) & 0xFF00FF) + ((((arg0[var13] & 0xFF00) * var9) >> 8) & 0xFF00)
				var13++
				arg0[var13-1] = var11 + ((((arg0[var13] & 0xFF00FF) * var9) >> 8) & 0xFF00FF) + ((((arg0[var13] & 0xFF00) * var9) >> 8) & 0xFF00)
				arg1 = var13 + 1
				arg0[var13] = var11 + ((((arg0[arg1] & 0xFF00FF) * var9) >> 8) & 0xFF00FF) + ((((arg0[arg1] & 0xFF00) * var9) >> 8) & 0xFF00)
			}
		}
	} else if arg4 < arg5 {
		var8 = (arg7 - arg6) / (arg5 - arg4)
		if HClip {
			if arg5 > pix2d.SafeWidth {
				arg5 = pix2d.SafeWidth
			}
			if arg4 < 0 {
				arg6 -= arg4 * var8
				arg4 = 0
			}
			if arg4 >= arg5 {
				return
			}
		}
		var16 := arg1 + arg4
		var17 = arg5 - arg4
		if Trans == 0 {
			for ok := true; ok; ok = var17 > 0 {
				arg0[var16] = ColourTable[arg6>>8]
				var16++
				arg6 += var8
				var17--
			}
		} else {
			var9 = Trans
			var10 = 256 - Trans
			for ok := true; ok; ok = var17 > 0 {
				var11 = ColourTable[arg6>>8]
				arg6 += var8
				var12 := ((((var11 & 0xFF00FF) * var10) >> 8) & 0xFF00FF) + ((((var11 & 0xFF00) * var10) >> 8) & 0xFF00)
				var16++
				arg0[var16-1] = var12 + ((((arg0[var16] & 0xFF00FF) * var9) >> 8) & 0xFF00FF) + ((((arg0[var16] & 0xFF00) * var9) >> 8) & 0xFF00)
				var17--
			}
		}
	}
}

func FlatTriangle(arg0, arg1, arg2, arg3, arg4, arg5, arg6 int) {
	var7 := 0
	if arg1 != arg0 {
		var7 = ((arg4 - arg3) << 16) / (arg1 - arg0)
	}
	var8 := 0
	if arg2 != arg1 {
		var8 = ((arg5 - arg4) << 16) / (arg2 - arg1)
	}
	var9 := 0
	if arg2 != arg0 {
		var9 = ((arg3 - arg5) << 16) / (arg0 - arg2)
	}
	if arg0 <= arg1 && arg0 <= arg2 {
		if arg0 < pix2d.BoundBottom {
			if arg1 > pix2d.BoundBottom {
				arg1 = pix2d.BoundBottom
			}
			if arg2 > pix2d.BoundBottom {
				arg2 = pix2d.BoundBottom
			}
			if arg1 < arg2 {
				arg3 <<= 0x10
				arg5 = arg3
				if arg0 < 0 {
					arg5 -= var9 * arg0
					arg3 -= var7 * arg0
					arg0 = 0
				}
				arg4 <<= 0x10
				if arg1 < 0 {
					arg4 -= var8 * arg1
					arg1 = 0
				}
				if arg0 != arg1 && var9 < var7 || arg0 == arg1 && var9 > var8 {
					arg2 -= arg1
					arg1 -= arg0
					arg0 = LineOffset[arg0]
					for {
						arg1--
						if arg1 < 0 {
							for {
								arg2--
								if arg2 < 0 {
									return
								}
								FlatRaster(pix2d.Data, arg0, arg6, arg5>>16, arg4>>16)
								arg5 += var9
								arg4 += var8
								arg0 += pix2d.Width2D
							}
						}
						FlatRaster(pix2d.Data, arg0, arg6, arg5>>16, arg3>>16)
						arg5 += var9
						arg3 += var7
						arg0 += pix2d.Width2D
					}
				} else {
					arg2 -= arg1
					arg1 -= arg0
					arg0 = LineOffset[arg0]
					for {
						arg1--
						if arg1 < 0 {
							for {
								arg2--
								if arg2 < 0 {
									return
								}
								FlatRaster(pix2d.Data, arg0, arg6, arg4>>16, arg5>>16)
								arg5 += var9
								arg4 += var8
								arg0 += pix2d.Width2D
							}
						}
						FlatRaster(pix2d.Data, arg0, arg6, arg3>>16, arg5>>16)
						arg5 += var9
						arg3 += var7
						arg0 += pix2d.Width2D
					}
				}
			} else {
				arg3 <<= 0x10
				arg4 = arg3
				if arg0 < 0 {
					arg4 -= var9 * arg0
					arg3 -= var7 * arg0
					arg0 = 0
				}
				arg5 <<= 0x10
				if arg2 < 0 {
					arg5 -= var8 * arg2
					arg2 = 0
				}
				if arg0 != arg2 && var9 < var7 || arg0 == arg2 && var8 > var7 {
					arg1 -= arg2
					arg2 -= arg0
					arg0 = LineOffset[arg0]
					for {
						arg2--
						if arg2 < 0 {
							for {
								arg1--
								if arg1 < 0 {
									return
								}
								FlatRaster(pix2d.Data, arg0, arg6, arg5>>16, arg3>>16)
								arg5 += var8
								arg3 += var7
								arg0 += pix2d.Width2D
							}
						}
						FlatRaster(pix2d.Data, arg0, arg6, arg4>>16, arg3>>16)
						arg4 += var9
						arg3 += var7
						arg0 += pix2d.Width2D
					}
				} else {
					arg1 -= arg2
					arg2 -= arg0
					arg0 = LineOffset[arg0]
					for {
						arg2--
						if arg2 < 0 {
							for {
								arg1--
								if arg1 < 0 {
									return
								}
								FlatRaster(pix2d.Data, arg0, arg6, arg3>>16, arg5>>16)
								arg5 += var8
								arg3 += var7
								arg0 += pix2d.Width2D
							}
						}
						FlatRaster(pix2d.Data, arg0, arg6, arg3>>16, arg4>>16)
						arg4 += var9
						arg3 += var7
						arg0 += pix2d.Width2D
					}
				}
			}
		}
	} else if arg1 <= arg2 {
		if arg1 < pix2d.BoundBottom {
			if arg2 > pix2d.BoundBottom {
				arg2 = pix2d.BoundBottom
			}
			if arg0 > pix2d.BoundBottom {
				arg0 = pix2d.BoundBottom
			}
			if arg2 < arg0 {
				arg4 <<= 0x10
				arg3 = arg4
				if arg1 < 0 {
					arg3 -= var7 * arg1
					arg4 -= var8 * arg1
					arg1 = 0
				}
				arg5 <<= 0x10
				if arg2 < 0 {
					arg5 -= var9 * arg2
					arg2 = 0
				}
				if arg1 != arg2 && var7 < var8 || arg1 == arg2 && var7 > var9 {
					arg0 -= arg2
					arg2 -= arg1
					arg1 = LineOffset[arg1]
					for {
						arg2--
						if arg2 < 0 {
							for {
								arg0--
								if arg0 < 0 {
									return
								}
								FlatRaster(pix2d.Data, arg1, arg6, arg3>>16, arg5>>16)
								arg3 += var7
								arg5 += var9
								arg1 += pix2d.Width2D
							}
						}
						FlatRaster(pix2d.Data, arg1, arg6, arg3>>16, arg4>>16)
						arg3 += var7
						arg4 += var8
						arg1 += pix2d.Width2D
					}
				} else {
					arg0 -= arg2
					arg2 -= arg1
					arg1 = LineOffset[arg1]
					for {
						arg2--
						if arg2 < 0 {
							for {
								arg0--
								if arg0 < 0 {
									return
								}
								FlatRaster(pix2d.Data, arg1, arg6, arg5>>16, arg3>>16)
								arg3 += var7
								arg5 += var9
								arg1 += pix2d.Width2D
							}
						}
						FlatRaster(pix2d.Data, arg1, arg6, arg4>>16, arg3>>16)
						arg3 += var7
						arg4 += var8
						arg1 += pix2d.Width2D
					}
				}
			} else {
				arg4 <<= 0x10
				arg5 = arg4
				if arg1 < 0 {
					arg5 -= var7 * arg1
					arg4 -= var8 * arg1
					arg1 = 0
				}
				arg3 <<= 0x10
				if arg0 < 0 {
					arg3 -= var9 * arg0
					arg0 = 0
				}
				if var7 < var8 {
					arg2 -= arg0
					arg0 -= arg1
					arg1 = LineOffset[arg1]
					for {
						arg0--
						if arg0 < 0 {
							for {
								arg2--
								if arg2 < 0 {
									return
								}
								FlatRaster(pix2d.Data, arg1, arg6, arg3>>16, arg4>>16)
								arg3 += var9
								arg4 += var8
								arg1 += pix2d.Width2D
							}
						}
						FlatRaster(pix2d.Data, arg1, arg6, arg5>>16, arg4>>16)
						arg5 += var7
						arg4 += var8
						arg1 += pix2d.Width2D
					}
				} else {
					arg2 -= arg0
					arg0 -= arg1
					arg1 = LineOffset[arg1]
					for {
						arg0--
						if arg0 < 0 {
							for {
								arg2--
								if arg2 < 0 {
									return
								}
								FlatRaster(pix2d.Data, arg1, arg6, arg4>>16, arg3>>16)
								arg3 += var9
								arg4 += var8
								arg1 += pix2d.Width2D
							}
						}
						FlatRaster(pix2d.Data, arg1, arg6, arg4>>16, arg5>>16)
						arg5 += var7
						arg4 += var8
						arg1 += pix2d.Width2D
					}
				}
			}
		}
	} else if arg2 < pix2d.BoundBottom {
		if arg0 > pix2d.BoundBottom {
			arg0 = pix2d.BoundBottom
		}
		if arg1 > pix2d.BoundBottom {
			arg1 = pix2d.BoundBottom
		}
		if arg0 < arg1 {
			arg5 <<= 0x10
			arg4 = arg5
			if arg2 < 0 {
				arg4 -= var8 * arg2
				arg5 -= var9 * arg2
				arg2 = 0
			}
			arg3 <<= 0x10
			if arg0 < 0 {
				arg3 -= var7 * arg0
				arg0 = 0
			}
			if var8 < var9 {
				arg1 -= arg0
				arg0 -= arg2
				arg2 = LineOffset[arg2]
				for {
					arg0--
					if arg0 < 0 {
						for {
							arg1--
							if arg1 < 0 {
								return
							}
							FlatRaster(pix2d.Data, arg2, arg6, arg4>>16, arg3>>16)
							arg4 += var8
							arg3 += var7
							arg2 += pix2d.Width2D
						}
					}
					FlatRaster(pix2d.Data, arg2, arg6, arg4>>16, arg5>>16)
					arg4 += var8
					arg5 += var9
					arg2 += pix2d.Width2D
				}
			} else {
				arg1 -= arg0
				arg0 -= arg2
				arg2 = LineOffset[arg2]
				for {
					arg0--
					if arg0 < 0 {
						for {
							arg1--
							if arg1 < 0 {
								return
							}
							FlatRaster(pix2d.Data, arg2, arg6, arg3>>16, arg4>>16)
							arg4 += var8
							arg3 += var7
							arg2 += pix2d.Width2D
						}
					}
					FlatRaster(pix2d.Data, arg2, arg6, arg5>>16, arg4>>16)
					arg4 += var8
					arg5 += var9
					arg2 += pix2d.Width2D
				}
			}
		} else {
			arg5 <<= 0x10
			arg3 = arg5
			if arg2 < 0 {
				arg3 -= var8 * arg2
				arg5 -= var9 * arg2
				arg2 = 0
			}
			arg4 <<= 0x10
			if arg1 < 0 {
				arg4 -= var7 * arg1
				arg1 = 0
			}
			if var8 < var9 {
				arg0 -= arg1
				arg1 -= arg2
				arg2 = LineOffset[arg2]
				for {
					arg1--
					if arg1 < 0 {
						for {
							arg0--
							if arg0 < 0 {
								return
							}
							FlatRaster(pix2d.Data, arg2, arg6, arg4>>16, arg5>>16)
							arg4 += var7
							arg5 += var9
							arg2 += pix2d.Width2D
						}
					}
					FlatRaster(pix2d.Data, arg2, arg6, arg3>>16, arg5>>16)
					arg3 += var8
					arg5 += var9
					arg2 += pix2d.Width2D
				}
			} else {
				arg0 -= arg1
				arg1 -= arg2
				arg2 = LineOffset[arg2]
				for {
					arg1--
					if arg1 < 0 {
						for {
							arg0--
							if arg0 < 0 {
								return
							}
							FlatRaster(pix2d.Data, arg2, arg6, arg5>>16, arg4>>16)
							arg4 += var7
							arg5 += var9
							arg2 += pix2d.Width2D
						}
					}
					FlatRaster(pix2d.Data, arg2, arg6, arg5>>16, arg3>>16)
					arg3 += var8
					arg5 += var9
					arg2 += pix2d.Width2D
				}
			}
		}
	}
}

func FlatRaster(arg0 []int, arg1, arg2, arg4, arg5 int) {
	if HClip {
		if arg5 > pix2d.SafeWidth {
			arg5 = pix2d.SafeWidth
		}
		if arg4 < 0 {
			arg4 = 0
		}
	}
	if arg4 >= arg5 {
		return
	}
	arg1 += arg4
	var15 := (arg5 - arg4) >> 2
	var8 := 0
	if Trans == 0 {
		for {
			var15--
			if var15 < 0 {
				var15 = (arg5 - arg4) & 0x3
				for {
					var15--
					if var15 < 0 {
						return
					}
					arg0[arg1] = arg2
					arg1++
				}
			}
			var8 = arg1 + 1
			arg0[arg1] = arg2
			arg0[var8] = arg2
			var8++
			arg0[var8] = arg2
			var8++
			arg1 = var8 + 1
			arg0[var8] = arg2
		}
	}
	var6 := Trans
	var7 := 256 - Trans
	var10 := ((((arg2 & 0xFF00FF) * var7) >> 8) & 0xFF00FF) + ((((arg2 & 0xFF00) * var7) >> 8) & 0xFF00)
	for {
		var15--
		if var15 < 0 {
			var15 = (arg5 - arg4) & 0x3
			for {
				var15--
				if var15 < 0 {
					return
				}
				arg1++
				arg0[arg1-1] = var10 + ((((arg0[arg1] & 0xFF00FF) * var6) >> 8) & 0xFF00FF) + ((((arg0[arg1] & 0xFF00) * var6) >> 8) & 0xFF00)
			}
		}
		var8 = arg1 + 1
		arg0[arg1] = var10 + ((((arg0[var8] & 0xFF00FF) * var6) >> 8) & 0xFF00FF) + ((((arg0[var8] & 0xFF00) * var6) >> 8) & 0xFF00)
		var9 := var8 + 1
		arg0[var8] = var10 + ((((arg0[var9] & 0xFF00FF) * var6) >> 8) & 0xFF00FF) + ((((arg0[var9] & 0xFF00) * var6) >> 8) & 0xFF00)
		var11 := var9 + 1
		arg0[var9] = var10 + ((((arg0[var11] & 0xFF00FF) * var6) >> 8) & 0xFF00FF) + ((((arg0[var11] & 0xFF00) * var6) >> 8) & 0xFF00)
		arg1 = var11 + 1
		arg0[var11] = var10 + ((((arg0[arg1] & 0xFF00FF) * var6) >> 8) & 0xFF00FF) + ((((arg0[arg1] & 0xFF00) * var6) >> 8) & 0xFF00)
	}
}

func TextureTriangle(arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7, arg8, arg9, arg10, arg11, arg12, arg13, arg14, arg15, arg16, arg17, arg18 int) {
	var19 := GetTexels(arg18)
	Opaque = !TextureTranslucent[arg18]
	var36 := arg9 - arg10
	var38 := arg12 - arg13
	var40 := arg15 - arg16
	var37 := arg11 - arg9
	var39 := arg14 - arg12
	var41 := arg17 - arg15
	var20 := (var37*arg12 - var39*arg9) << 14
	var21 := (var39*arg15 - var41*arg12) << 8
	var22 := (var41*arg9 - var37*arg15) << 5
	var23 := (var36*arg12 - var38*arg9) << 14
	var24 := (var38*arg15 - var40*arg12) << 8
	var25 := (var40*arg9 - var36*arg15) << 5
	var26 := (var38*var37 - var36*var39) << 14
	var27 := (var40*var39 - var38*var41) << 8
	var28 := (var36*var41 - var40*var37) << 5
	var29 := 0
	var30 := 0
	if arg1 != arg0 {
		var29 = ((arg4 - arg3) << 16) / (arg1 - arg0)
		var30 = ((arg7 - arg6) << 16) / (arg1 - arg0)
	}
	var31 := 0
	var32 := 0
	if arg2 != arg1 {
		var31 = ((arg5 - arg4) << 16) / (arg2 - arg1)
		var32 = ((arg8 - arg7) << 16) / (arg2 - arg1)
	}
	var33 := 0
	var34 := 0
	if arg2 != arg0 {
		var33 = ((arg3 - arg5) << 16) / (arg0 - arg2)
		var34 = ((arg6 - arg8) << 16) / (arg0 - arg2)
	}
	var35 := 0
	if arg0 <= arg1 && arg0 <= arg2 {
		if arg0 < pix2d.BoundBottom {
			if arg1 > pix2d.BoundBottom {
				arg1 = pix2d.BoundBottom
			}
			if arg2 > pix2d.BoundBottom {
				arg2 = pix2d.BoundBottom
			}
			if arg1 < arg2 {
				arg3 <<= 0x10
				arg5 = arg3
				arg6 <<= 0x10
				arg8 = arg6
				if arg0 < 0 {
					arg5 -= var33 * arg0
					arg3 -= var29 * arg0
					arg8 -= var34 * arg0
					arg6 -= var30 * arg0
					arg0 = 0
				}
				arg4 <<= 0x10
				arg7 <<= 0x10
				if arg1 < 0 {
					arg4 -= var31 * arg1
					arg7 -= var32 * arg1
					arg1 = 0
				}
				var35 = arg0 - CenterH3D
				var20 += var22 * var35
				var23 += var25 * var35
				var26 += var28 * var35
				if arg0 != arg1 && var33 < var29 || arg0 == arg1 && var33 > var31 {
					arg2 -= arg1
					arg1 -= arg0
					arg0 = LineOffset[arg0]
					for {
						arg1--
						if arg1 < 0 {
							for {
								arg2--
								if arg2 < 0 {
									return
								}
								TextureRaster(pix2d.Data, var19, 0, 0, arg0, arg15>>16, arg14>>16, arg8>>8, arg7>>8, var20, var23, var26, var21, var24, var27)
								arg5 += var33
								arg4 += var31
								arg8 += var34
								arg7 += var32
								arg0 += pix2d.Width2D
								var20 += var22
								var23 += var25
								var26 += var28
							}
						}
						TextureRaster(pix2d.Data, var19, 0, 0, arg0, arg5>>16, arg3>>16, arg8>>8, arg6>>8, var20, var23, var26, var21, var24, var27)
						arg5 += var33
						arg3 += var29
						arg8 += var34
						arg6 += var30
						arg0 += pix2d.Width2D
						var20 += var22
						var23 += var25
						var26 += var28
					}
				} else {
					arg2 -= arg1
					arg1 -= arg0
					arg0 = LineOffset[arg0]
					for {
						arg1--
						if arg1 < 0 {
							for {
								arg2--
								if arg2 < 0 {
									return
								}
								TextureRaster(pix2d.Data, var19, 0, 0, arg0, arg4>>16, arg5>>16, arg7>>8, arg8>>8, var20, var23, var26, var21, var24, var27)
								arg5 += var33
								arg4 += var31
								arg8 += var34
								arg7 += var32
								arg0 += pix2d.Width2D
								var20 += var22
								var23 += var25
								var26 += var28
							}
						}
						TextureRaster(pix2d.Data, var19, 0, 0, arg0, arg3>>16, arg5>>16, arg6>>8, arg8>>8, var20, var23, var26, var21, var24, var27)
						arg5 += var33
						arg3 += var29
						arg8 += var34
						arg6 += var30
						arg0 += pix2d.Width2D
						var20 += var22
						var23 += var25
						var26 += var28
					}
				}
			} else {
				arg3 <<= 0x10
				arg4 = arg3
				arg6 <<= 0x10
				arg7 = arg6
				if arg0 < 0 {
					arg4 -= var33 * arg0
					arg3 -= var29 * arg0
					arg7 -= var34 * arg0
					arg6 -= var30 * arg0
					arg0 = 0
				}
				arg5 <<= 0x10
				arg8 <<= 0x10
				if arg2 < 0 {
					arg5 -= var31 * arg2
					arg8 -= var32 * arg2
					arg2 = 0
				}
				var35 = arg0 - CenterH3D
				var20 += var22 * var35
				var23 += var25 * var35
				var26 += var28 * var35
				if (arg0 == arg2 || var33 >= var29) && (arg0 != arg2 || var31 <= var29) {
					arg1 -= arg2
					arg2 -= arg0
					arg0 = LineOffset[arg0]
					for {
						arg2--
						if arg2 < 0 {
							for {
								arg1--
								if arg1 < 0 {
									return
								}
								TextureRaster(pix2d.Data, var19, 0, 0, arg0, arg3>>16, arg5>>16, arg6>>8, arg8>>8, var20, var23, var26, var21, var24, var27)
								arg5 += var31
								arg3 += var29
								arg8 += var32
								arg6 += var30
								arg0 += pix2d.Width2D
								var20 += var22
								var23 += var25
								var26 += var28
							}
						}
						TextureRaster(pix2d.Data, var19, 0, 0, arg0, arg3>>16, arg4>>16, arg6>>8, arg7>>8, var20, var23, var26, var21, var24, var27)
						arg4 += var33
						arg3 += var29
						arg7 += var34
						arg6 += var30
						arg0 += pix2d.Width2D
						var20 += var22
						var23 += var25
						var26 += var28
					}
				} else {
					arg1 -= arg2
					arg2 -= arg0
					arg0 = LineOffset[arg0]
					for {
						arg2--
						if arg2 < 0 {
							for {
								arg1--
								if arg1 < 0 {
									return
								}
								TextureRaster(pix2d.Data, var19, 0, 0, arg0, arg5>>16, arg3>>16, arg8>>8, arg6>>8, var20, var23, var26, var21, var24, var27)
								arg5 += var31
								arg3 += var29
								arg8 += var32
								arg6 += var30
								arg0 += pix2d.Width2D
								var20 += var22
								var23 += var25
								var26 += var28
							}
						}
						TextureRaster(pix2d.Data, var19, 0, 0, arg0, arg4>>16, arg3>>16, arg7>>8, arg6>>8, var20, var23, var26, var21, var24, var27)
						arg4 += var33
						arg3 += var29
						arg7 += var34
						arg6 += var30
						arg0 += pix2d.Width2D
						var20 += var22
						var23 += var25
						var26 += var28
					}
				}
			}
		}
	} else if arg1 <= arg2 {
		if arg1 < pix2d.BoundBottom {
			if arg2 > pix2d.BoundBottom {
				arg2 = pix2d.BoundBottom
			}
			if arg0 > pix2d.BoundBottom {
				arg0 = pix2d.BoundBottom
			}
			if arg2 < arg0 {
				arg4 <<= 0x10
				arg3 = arg4
				arg7 <<= 0x10
				arg6 = arg7
				if arg1 < 0 {
					arg3 -= var29 * arg1
					arg4 -= var31 * arg1
					arg6 -= var30 * arg1
					arg7 -= var32 * arg1
					arg1 = 0
				}
				arg5 <<= 0x10
				arg8 <<= 0x10
				if arg2 < 0 {
					arg5 -= var33 * arg2
					arg8 -= var34 * arg2
					arg2 = 0
				}
				var35 = arg1 - CenterH3D
				var20 += var22 * var35
				var23 += var25 * var35
				var26 += var28 * var35
				if arg1 != arg2 && var29 < var31 || arg1 == arg2 && var29 > var33 {
					arg0 -= arg2
					arg2 -= arg1
					arg1 = LineOffset[arg1]
					for {
						arg2--
						if arg2 < 0 {
							for {
								arg0--
								if arg0 < 0 {
									return
								}
								TextureRaster(pix2d.Data, var19, 0, 0, arg1, arg3>>16, arg5>>16, arg6>>8, arg8>>8, var20, var23, var26, var21, var24, var27)
								arg3 += var29
								arg5 += var33
								arg6 += var30
								arg8 += var34
								arg1 += pix2d.Width2D
								var20 += var22
								var23 += var25
								var26 += var28
							}
						}
						TextureRaster(pix2d.Data, var19, 0, 0, arg1, arg3>>16, arg4>>16, arg6>>8, arg7>>8, var20, var23, var26, var21, var24, var27)
						arg3 += var29
						arg4 += var31
						arg6 += var30
						arg7 += var32
						arg1 += pix2d.Width2D
						var20 += var22
						var23 += var25
						var26 += var28
					}
				} else {
					arg0 -= arg2
					arg2 -= arg1
					arg1 = LineOffset[arg1]
					for {
						arg2--
						if arg2 < 0 {
							for {
								arg0--
								if arg0 < 0 {
									return
								}
								TextureRaster(pix2d.Data, var19, 0, 0, arg1, arg5>>16, arg3>>16, arg8>>8, arg6>>8, var20, var23, var26, var21, var24, var27)
								arg3 += var29
								arg5 += var33
								arg6 += var30
								arg8 += var34
								arg1 += pix2d.Width2D
								var20 += var22
								var23 += var25
								var26 += var28
							}
						}
						TextureRaster(pix2d.Data, var19, 0, 0, arg1, arg4>>16, arg3>>16, arg7>>8, arg6>>8, var20, var23, var26, var21, var24, var27)
						arg3 += var29
						arg4 += var31
						arg6 += var30
						arg7 += var32
						arg1 += pix2d.Width2D
						var20 += var22
						var23 += var25
						var26 += var28
					}
				}
			} else {
				arg4 <<= 0x10
				arg5 = arg4
				arg7 <<= 0x10
				arg8 = arg7
				if arg1 < 0 {
					arg5 -= var29 * arg1
					arg4 -= var31 * arg1
					arg8 -= var30 * arg1
					arg7 -= var32 * arg1
					arg1 = 0
				}
				arg3 <<= 0x10
				arg6 <<= 0x10
				if arg0 < 0 {
					arg3 -= var33 * arg0
					arg6 -= var34 * arg0
					arg0 = 0
				}
				var35 = arg1 - CenterH3D
				var20 += var22 * var35
				var23 += var25 * var35
				var26 += var28 * var35
				if var29 < var31 {
					arg2 -= arg0
					arg0 -= arg1
					arg1 = LineOffset[arg1]
					for {
						arg0--
						if arg0 < 0 {
							for {
								arg2--
								if arg2 < 0 {
									return
								}
								TextureRaster(pix2d.Data, var19, 0, 0, arg1, arg3>>16, arg4>>16, arg6>>8, arg7>>8, var20, var23, var26, var21, var24, var27)
								arg3 += var33
								arg4 += var31
								arg6 += var34
								arg7 += var32
								arg1 += pix2d.Width2D
								var20 += var22
								var23 += var25
								var26 += var28
							}
						}
						TextureRaster(pix2d.Data, var19, 0, 0, arg1, arg5>>16, arg4>>16, arg8>>8, arg7>>8, var20, var23, var26, var21, var24, var27)
						arg5 += var29
						arg4 += var31
						arg8 += var30
						arg7 += var32
						arg1 += pix2d.Width2D
						var20 += var22
						var23 += var25
						var26 += var28
					}
				} else {
					arg2 -= arg0
					arg0 -= arg1
					arg1 = LineOffset[arg1]
					for {
						arg0--
						if arg0 < 0 {
							for {
								arg2--
								if arg2 < 0 {
									return
								}
								TextureRaster(pix2d.Data, var19, 0, 0, arg1, arg4>>16, arg3>>16, arg7>>8, arg6>>8, var20, var23, var26, var21, var24, var27)
								arg3 += var33
								arg4 += var31
								arg6 += var34
								arg7 += var32
								arg1 += pix2d.Width2D
								var20 += var22
								var23 += var25
								var26 += var28
							}
						}
						TextureRaster(pix2d.Data, var19, 0, 0, arg1, arg4>>16, arg5>>16, arg7>>8, arg8>>8, var20, var23, var26, var21, var24, var27)
						arg5 += var29
						arg4 += var31
						arg8 += var30
						arg7 += var32
						arg1 += pix2d.Width2D
						var20 += var22
						var23 += var25
						var26 += var28
					}
				}
			}
		}
	} else if arg2 < pix2d.BoundBottom {
		if arg0 > pix2d.BoundBottom {
			arg0 = pix2d.BoundBottom
		}
		if arg1 > pix2d.BoundBottom {
			arg1 = pix2d.BoundBottom
		}
		if arg0 < arg1 {
			arg5 <<= 0x10
			arg4 = arg5
			arg8 <<= 0x10
			arg7 = arg8
			if arg2 < 0 {
				arg4 -= var31 * arg2
				arg5 -= var33 * arg2
				arg7 -= var32 * arg2
				arg8 -= var34 * arg2
				arg2 = 0
			}
			arg3 <<= 0x10
			arg6 <<= 0x10
			if arg0 < 0 {
				arg3 -= var29 * arg0
				arg6 -= var30 * arg0
				arg0 = 0
			}
			var35 = arg2 - CenterH3D
			var20 += var22 * var35
			var23 += var25 * var35
			var26 += var28 * var35
			if var31 < var33 {
				arg1 -= arg0
				arg0 -= arg2
				arg2 = LineOffset[arg2]
				for {
					arg0--
					if arg0 < 0 {
						for {
							arg1--
							if arg1 < 0 {
								return
							}
							TextureRaster(pix2d.Data, var19, 0, 0, arg2, arg4>>16, arg3>>16, arg7>>8, arg6>>8, var20, var23, var26, var21, var24, var27)
							arg4 += var31
							arg3 += var29
							arg7 += var32
							arg6 += var30
							arg2 += pix2d.Width2D
							var20 += var22
							var23 += var25
							var26 += var28
						}
					}
					TextureRaster(pix2d.Data, var19, 0, 0, arg2, arg4>>16, arg5>>16, arg7>>8, arg8>>8, var20, var23, var26, var21, var24, var27)
					arg4 += var31
					arg5 += var33
					arg7 += var32
					arg8 += var34
					arg2 += pix2d.Width2D
					var20 += var22
					var23 += var25
					var26 += var28
				}
			} else {
				arg1 -= arg0
				arg0 -= arg2
				arg2 = LineOffset[arg2]
				for {
					arg0--
					if arg0 < 0 {
						for {
							arg1--
							if arg1 < 0 {
								return
							}
							TextureRaster(pix2d.Data, var19, 0, 0, arg2, arg3>>16, arg4>>16, arg6>>8, arg7>>8, var20, var23, var26, var21, var24, var27)
							arg4 += var31
							arg3 += var29
							arg7 += var32
							arg6 += var30
							arg2 += pix2d.Width2D
							var20 += var22
							var23 += var25
							var26 += var28
						}
					}
					TextureRaster(pix2d.Data, var19, 0, 0, arg2, arg5>>16, arg4>>16, arg8>>8, arg7>>8, var20, var23, var26, var21, var24, var27)
					arg4 += var31
					arg5 += var33
					arg7 += var32
					arg8 += var34
					arg2 += pix2d.Width2D
					var20 += var22
					var23 += var25
					var26 += var28
				}
			}
		} else {
			arg5 <<= 0x10
			arg3 = arg5
			arg8 <<= 0x10
			arg6 = arg8
			if arg2 < 0 {
				arg3 -= var31 * arg2
				arg5 -= var33 * arg2
				arg6 -= var32 * arg2
				arg8 -= var34 * arg2
				arg2 = 0
			}
			arg4 <<= 0x10
			arg7 <<= 0x10
			if arg1 < 0 {
				arg4 -= var29 * arg1
				arg7 -= var30 * arg1
				arg1 = 0
			}
			var35 = arg2 - CenterH3D
			var20 += var22 * var35
			var23 += var25 * var35
			var26 += var28 * var35
			if var31 < var33 {
				arg0 -= arg1
				arg1 -= arg2
				arg2 = LineOffset[arg2]
				for {
					arg1--
					if arg1 < 0 {
						for {
							arg0--
							if arg0 < 0 {
								return
							}
							TextureRaster(pix2d.Data, var19, 0, 0, arg2, arg4>>16, arg5>>16, arg7>>8, arg8>>8, var20, var23, var26, var21, var24, var27)
							arg4 += var29
							arg5 += var33
							arg7 += var30
							arg8 += var34
							arg2 += pix2d.Width2D
							var20 += var22
							var23 += var25
							var26 += var28
						}
					}
					TextureRaster(pix2d.Data, var19, 0, 0, arg2, arg3>>16, arg5>>16, arg6>>8, arg8>>8, var20, var23, var26, var21, var24, var27)
					arg3 += var31
					arg5 += var33
					arg6 += var32
					arg8 += var34
					arg2 += pix2d.Width2D
					var20 += var22
					var23 += var25
					var26 += var28
				}
			} else {
				arg0 -= arg1
				arg1 -= arg2
				arg2 = LineOffset[arg2]
				for {
					arg1--
					if arg1 < 0 {
						for {
							arg0--
							if arg0 < 0 {
								return
							}
							TextureRaster(pix2d.Data, var19, 0, 0, arg2, arg5>>16, arg4>>16, arg8>>8, arg7>>8, var20, var23, var26, var21, var24, var27)
							arg4 += var29
							arg5 += var33
							arg7 += var30
							arg8 += var34
							arg2 += pix2d.Width2D
							var20 += var22
							var23 += var25
							var26 += var28
						}
					}
					TextureRaster(pix2d.Data, var19, 0, 0, arg2, arg5>>16, arg3>>16, arg8>>8, arg6>>8, var20, var23, var26, var21, var24, var27)
					arg3 += var31
					arg5 += var33
					arg6 += var32
					arg8 += var34
					arg2 += pix2d.Width2D
					var20 += var22
					var23 += var25
					var26 += var28
				}
			}
		}
	}
}

func TextureRaster(arg0 []int, arg1 []int, arg2, arg3, arg4, arg5, arg6, arg7, arg8, arg9, arg10, arg11, arg12, arg13, arg14 int) {
	if arg5 >= arg6 {
		return
	}
	var15 := 0
	var16 := 0
	if HClip {
		var15 = (arg8 - arg7) / (arg6 - arg5)
		if arg6 > pix2d.SafeWidth {
			arg6 = pix2d.SafeWidth
		}
		if arg5 < 0 {
			arg7 -= arg5 * var15
			arg5 = 0
		}
		if arg5 >= arg6 {
			return
		}
		var16 = (arg6 - arg5) >> 3
		var15 <<= 0xC
		arg7 <<= 0x9
	} else {
		if arg6-arg5 > 7 {
			var16 = (arg6 - arg5) >> 3
			var15 = ((arg8 - arg7) * DivTable[var16]) >> 6
		} else {
			var16 = 0
			var15 = 0
		}
		arg7 <<= 0x9
	}
	arg4 += arg5
	var17 := 0
	var18 := 0
	var19 := 0
	var20 := 0
	var21 := 0
	var22 := 0
	var23 := 0
	var25 := 0
	var32 := 0
	var33 := 0
	var34 := 0
	if LowDetail {
		var17 = 0
		var18 = 0
		var20 = arg5 - CenterW3D
		var32 = arg9 + (arg12>>3)*var20
		var33 = arg10 + (arg13>>3)*var20
		var34 = arg11 + (arg14>>3)*var20
		var19 = var34 >> 12
		if var19 != 0 {
			arg2 = var32 / var19
			arg3 = var33 / var19
			if arg2 < 0 {
				arg2 = 0
			} else if arg2 > 4032 {
				arg2 = 4032
			}
		}
		arg9 = var32 + arg12
		arg10 = var33 + arg13
		arg11 = var34 + arg14
		var19 = arg11 >> 12
		if var19 != 0 {
			var17 = arg9 / var19
			var18 = arg10 / var19
			if var17 < 7 {
				var17 = 7
			} else if var17 > 4032 {
				var17 = 4032
			}
		}
		var21 = (var17 - arg2) >> 3
		var22 = (var18 - arg3) >> 3
		arg2 += (arg7 >> 3) & 0xC0000
		var23 = arg7 >> 23
		if Opaque {
			for ; var16 > 0; var16-- {
				var25 = arg4 + 1
				arg0[arg4] = arg1[(arg3&0xFC0)+(arg2>>6)] >> var23
				arg2 += var21
				arg3 += var22
				arg0[var25] = arg1[(arg3&0xFC0)+(arg2>>6)] >> var23
				var25++
				arg2 += var21
				arg3 += var22
				arg0[var25] = arg1[(arg3&0xFC0)+(arg2>>6)] >> var23
				var25++
				arg2 += var21
				arg3 += var22
				arg0[var25] = arg1[(arg3&0xFC0)+(arg2>>6)] >> var23
				var25++
				arg2 += var21
				arg3 += var22
				arg0[var25] = arg1[(arg3&0xFC0)+(arg2>>6)] >> var23
				var25++
				arg2 += var21
				arg3 += var22
				arg0[var25] = arg1[(arg3&0xFC0)+(arg2>>6)] >> var23
				var25++
				arg2 += var21
				arg3 += var22
				arg0[var25] = arg1[(arg3&0xFC0)+(arg2>>6)] >> var23
				var25++
				arg2 += var21
				arg3 += var22
				arg4 = var25 + 1
				arg0[var25] = arg1[(arg3&0xFC0)+(arg2>>6)] >> var23
				arg2 = var17
				arg3 = var18
				arg9 += arg12
				arg10 += arg13
				arg11 += arg14
				var19 = arg11 >> 12
				if var19 != 0 {
					var17 = arg9 / var19
					var18 = arg10 / var19
					if var17 < 7 {
						var17 = 7
					} else if var17 > 4032 {
						var17 = 4032
					}
				}
				var21 = (var17 - arg2) >> 3
				var22 = (var18 - arg3) >> 3
				arg7 += var15
				arg2 += (arg7 >> 3) & 0xC0000
				var23 = arg7 >> 23
			}
			var16 = (arg6 - arg5) & 0x7
			for ; var16 > 0; var16-- {
				arg0[arg4] = arg1[(arg3&0xFC0)+(arg2>>6)] >> var23
				arg4++
				arg2 += var21
				arg3 += var22
			}
		} else {
			for ; var16 > 0; var16-- {
				if v := arg1[(arg3&0xFC0)+(arg2>>6)] >> var23; v != 0 {
					arg0[arg4] = v
				}
				var25 = arg4 + 1
				arg2 += var21
				arg3 += var22
				if v := arg1[(arg3&0xFC0)+(arg2>>6)] >> var23; v != 0 {
					arg0[var25] = v
				}
				var25++
				arg2 += var21
				arg3 += var22
				if v := arg1[(arg3&0xFC0)+(arg2>>6)] >> var23; v != 0 {
					arg0[var25] = v
				}
				var25++
				arg2 += var21
				arg3 += var22
				if v := arg1[(arg3&0xFC0)+(arg2>>6)] >> var23; v != 0 {
					arg0[var25] = v
				}
				var25++
				arg2 += var21
				arg3 += var22
				if v := arg1[(arg3&0xFC0)+(arg2>>6)] >> var23; v != 0 {
					arg0[var25] = v
				}
				var25++
				arg2 += var21
				arg3 += var22
				if v := arg1[(arg3&0xFC0)+(arg2>>6)] >> var23; v != 0 {
					arg0[var25] = v
				}
				var25++
				arg2 += var21
				arg3 += var22
				if v := arg1[(arg3&0xFC0)+(arg2>>6)] >> var23; v != 0 {
					arg0[var25] = v
				}
				var25++
				arg2 += var21
				arg3 += var22
				if v := arg1[(arg3&0xFC0)+(arg2>>6)] >> var23; v != 0 {
					arg0[var25] = v
				}
				arg4 = var25 + 1
				arg2 = var17
				arg3 = var18
				arg9 += arg12
				arg10 += arg13
				arg11 += arg14
				var19 = arg11 >> 12
				if var19 != 0 {
					var17 = arg9 / var19
					var18 = arg10 / var19
					if var17 < 7 {
						var17 = 7
					} else if var17 > 4032 {
						var17 = 4032
					}
				}
				var21 = (var17 - arg2) >> 3
				var22 = (var18 - arg3) >> 3
				arg7 += var15
				arg2 += (arg7 >> 3) & 0xC0000
				var23 = arg7 >> 23
			}
			var16 = (arg6 - arg5) & 0x7
			for ; var16 > 0; var16-- {
				if v := arg1[(arg3&0xFC0)+(arg2>>6)] >> var23; v != 0 {
					arg0[arg4] = v
				}
				arg4++
				arg2 += var21
				arg3 += var22
			}
		}
		return
	}
	var17 = 0
	var18 = 0
	var20 = arg5 - CenterW3D
	var32 = arg9 + (arg12>>3)*var20
	var33 = arg10 + (arg13>>3)*var20
	var34 = arg11 + (arg14>>3)*var20
	var19 = var34 >> 14
	if var19 != 0 {
		arg2 = var32 / var19
		arg3 = var33 / var19
		if arg2 < 0 {
			arg2 = 0
		} else if arg2 > 16256 {
			arg2 = 16256
		}
	}
	arg9 = var32 + arg12
	arg10 = var33 + arg13
	arg11 = var34 + arg14
	var19 = arg11 >> 14
	if var19 != 0 {
		var17 = arg9 / var19
		var18 = arg10 / var19
		if var17 < 7 {
			var17 = 7
		} else if var17 > 16256 {
			var17 = 16256
		}
	}
	var21 = (var17 - arg2) >> 3
	var22 = (var18 - arg3) >> 3
	arg2 += arg7 & 0x600000
	var23 = arg7 >> 23
	if Opaque {
		for ; var16 > 0; var16-- {
			var25 = arg4 + 1
			arg0[arg4] = arg1[(arg3&0x3F80)+(arg2>>7)] >> var23
			arg2 += var21
			arg3 += var22
			var26 := var25 + 1
			arg0[var25] = arg1[(arg3&0x3F80)+(arg2>>7)] >> var23
			arg2 += var21
			arg3 += var22
			var27 := var26 + 1
			arg0[var26] = arg1[(arg3&0x3F80)+(arg2>>7)] >> var23
			arg2 += var21
			arg3 += var22
			var28 := var27 + 1
			arg0[var27] = arg1[(arg3&0x3F80)+(arg2>>7)] >> var23
			arg2 += var21
			arg3 += var22
			var29 := var28 + 1
			arg0[var28] = arg1[(arg3&0x3F80)+(arg2>>7)] >> var23
			arg2 += var21
			arg3 += var22
			var30 := var29 + 1
			arg0[var29] = arg1[(arg3&0x3F80)+(arg2>>7)] >> var23
			arg2 += var21
			arg3 += var22
			var31 := var30 + 1
			arg0[var30] = arg1[(arg3&0x3F80)+(arg2>>7)] >> var23
			arg2 += var21
			arg3 += var22
			arg4 = var31 + 1
			arg0[var31] = arg1[(arg3&0x3F80)+(arg2>>7)] >> var23
			arg2 = var17
			arg3 = var18
			arg9 += arg12
			arg10 += arg13
			arg11 += arg14
			var19 = arg11 >> 14
			if var19 != 0 {
				var17 = arg9 / var19
				var18 = arg10 / var19
				if var17 < 7 {
					var17 = 7
				} else if var17 > 16256 {
					var17 = 16256
				}
			}
			var21 = (var17 - arg2) >> 3
			var22 = (var18 - arg3) >> 3
			arg7 += var15
			arg2 += arg7 & 0x600000
			var23 = arg7 >> 23
		}
		var16 = (arg6 - arg5) & 0x7
		for ; var16 > 0; var16-- {
			arg0[arg4] = arg1[(arg3&0x3F80)+(arg2>>7)] >> var23
			arg4++
			arg2 += var21
			arg3 += var22
		}
		return
	}
	for ; var16 > 0; var16-- {
		if v := arg1[(arg3&0x3F80)+(arg2>>7)] >> var23; v != 0 {
			arg0[arg4] = v
		}
		var25 = arg4 + 1
		arg2 += var21
		arg3 += var22
		if v := arg1[(arg3&0x3F80)+(arg2>>7)] >> var23; v != 0 {
			arg0[var25] = v
		}
		var25++
		arg2 += var21
		arg3 += var22
		if v := arg1[(arg3&0x3F80)+(arg2>>7)] >> var23; v != 0 {
			arg0[var25] = v
		}
		var25++
		arg2 += var21
		arg3 += var22
		if v := arg1[(arg3&0x3F80)+(arg2>>7)] >> var23; v != 0 {
			arg0[var25] = v
		}
		var25++
		arg2 += var21
		arg3 += var22
		if v := arg1[(arg3&0x3F80)+(arg2>>7)] >> var23; v != 0 {
			arg0[var25] = v
		}
		var25++
		arg2 += var21
		arg3 += var22
		if v := arg1[(arg3&0x3F80)+(arg2>>7)] >> var23; v != 0 {
			arg0[var25] = v
		}
		var25++
		arg2 += var21
		arg3 += var22
		if v := arg1[(arg3&0x3F80)+(arg2>>7)] >> var23; v != 0 {
			arg0[var25] = v
		}
		var25++
		arg2 += var21
		arg3 += var22
		if v := arg1[(arg3&0x3F80)+(arg2>>7)] >> var23; v != 0 {
			arg0[var25] = v
		}
		arg4 = var25 + 1
		arg2 = var17
		arg3 = var18
		arg9 += arg12
		arg10 += arg13
		arg11 += arg14
		var19 = arg11 >> 14
		if var19 != 0 {
			var17 = arg9 / var19
			var18 = arg10 / var19
			if var17 < 7 {
				var17 = 7
			} else if var17 > 16256 {
				var17 = 16256
			}
		}
		var21 = (var17 - arg2) >> 3
		var22 = (var18 - arg3) >> 3
		arg7 += var15
		arg2 += arg7 & 0x600000
		var23 = arg7 >> 23
	}
	var16 = (arg6 - arg5) & 0x7
	for ; var16 > 0; var16-- {
		if v := arg1[(arg3&0x3F80)+(arg2>>7)] >> var23; v != 0 {
			arg0[arg4] = v
		}
		arg4++
		arg2 += var21
		arg3 += var22
	}
}
