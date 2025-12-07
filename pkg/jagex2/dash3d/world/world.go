package world

import (
	"math"
	"math/rand"
	"strings"

	"goscape-client/pkg/jagex2/config/flotype"
	"goscape-client/pkg/jagex2/config/loctype"
	"goscape-client/pkg/jagex2/config/seqtype"
	"goscape-client/pkg/jagex2/dash3d"
	"goscape-client/pkg/jagex2/dash3d/entity"
	"goscape-client/pkg/jagex2/dash3d/world3d"
	"goscape-client/pkg/jagex2/datastruct"
	"goscape-client/pkg/jagex2/graphics/model"
	"goscape-client/pkg/jagex2/graphics/pix3d"
	"goscape-client/pkg/jagex2/io"
)

var (
	LowMemory                          bool = true
	LevelBuilt                         int
	FullBright                         bool
	ROTATION_WALL_TYPE                 = []int{1, 2, 4, 8}
	ROTATION_WALL_CORNER_TYPE          = []int{16, 32, 64, 128}
	WALL_DECORATION_ROTATION_FORWARD_X = []int{1, 0, -1, 0}
	WALL_DECORATION_ROTATION_FORWARD_Z = []int{0, -1, 0, 1}
	RandomHueOffset                    = int((rand.Float64() * 17.0) - 8)
	RandomLightnessOffset              = int((rand.Float64() * 33.0) - 16)
)

type World struct {
	MaxTileX                 int
	MaxTileZ                 int
	LevelHeightMap           [][][]int
	LevelTileFlags           [][][]byte
	LevelTileUnderlayIDs     [][][]byte
	LevelTileOverlayIDs      [][][]byte
	LevelTileOverlayShape    [][][]byte
	LevelTileOverlayRotation [][][]byte
	LevelShadeMap            [][][]byte
	LevelLightMap            [][]int
	BlendChroma              []int
	BlendSaturation          []int
	BlendLightness           []int
	BlendLuminance           []int
	BlendMagnitude           []int
	LevelOccludeMap          [][][]int
}

func NewWorld(arg0 int, arg1 [][][]byte, arg2 int, arg3 [][][]int) *World {
	var w World
	w.MaxTileX = arg2
	w.MaxTileZ = arg0
	w.LevelHeightMap = arg3
	w.LevelTileFlags = arg1
	w.LevelTileUnderlayIDs = make([][][]byte, 4)
	for i := range w.LevelTileUnderlayIDs {
		w.LevelTileUnderlayIDs[i] = make([][]byte, w.MaxTileX)
		for j := range w.LevelTileUnderlayIDs[i] {
			w.LevelTileUnderlayIDs[i][j] = make([]byte, w.MaxTileZ)
		}
	}
	w.LevelTileOverlayIDs = make([][][]byte, 4)
	for i := range w.LevelTileOverlayIDs {
		w.LevelTileOverlayIDs[i] = make([][]byte, w.MaxTileX)
		for j := range w.LevelTileOverlayIDs[i] {
			w.LevelTileOverlayIDs[i][j] = make([]byte, w.MaxTileZ)
		}
	}
	w.LevelTileOverlayShape = make([][][]byte, 4)
	for i := range w.LevelTileOverlayShape {
		w.LevelTileOverlayShape[i] = make([][]byte, w.MaxTileX)
		for j := range w.LevelTileOverlayShape[i] {
			w.LevelTileOverlayShape[i][j] = make([]byte, w.MaxTileZ)
		}
	}
	w.LevelTileOverlayRotation = make([][][]byte, 4)
	for i := range w.LevelTileOverlayRotation {
		w.LevelTileOverlayRotation[i] = make([][]byte, w.MaxTileX)
		for j := range w.LevelTileOverlayRotation[i] {
			w.LevelTileOverlayRotation[i][j] = make([]byte, w.MaxTileZ)
		}
	}
	w.LevelOccludeMap = make([][][]int, 4)
	for i := range w.LevelOccludeMap {
		w.LevelOccludeMap[i] = make([][]int, w.MaxTileX+1)
		for j := range w.LevelOccludeMap[i] {
			w.LevelOccludeMap[i][j] = make([]int, w.MaxTileZ+1)
		}
	}
	w.LevelShadeMap = make([][][]byte, 4)
	for i := range w.LevelShadeMap {
		w.LevelShadeMap[i] = make([][]byte, w.MaxTileX+1)
		for j := range w.LevelShadeMap[i] {
			w.LevelShadeMap[i][j] = make([]byte, w.MaxTileZ+1)
		}
	}
	w.LevelLightMap = make([][]int, w.MaxTileX+1)
	for i := range w.LevelLightMap {
		w.LevelLightMap[i] = make([]int, w.MaxTileZ+1)
	}
	w.BlendChroma = make([]int, w.MaxTileZ)
	w.BlendSaturation = make([]int, w.MaxTileZ)
	w.BlendLightness = make([]int, w.MaxTileZ)
	w.BlendLuminance = make([]int, w.MaxTileZ)
	w.BlendMagnitude = make([]int, w.MaxTileZ)
	return &w
}

func (w *World) ClearLandscape(arg0, arg1, arg3, arg4 int) {
	var6 := byte(0)
	for i := range flotype.Count {
		if strings.EqualFold(flotype.Instances[i].Name, "water") {
			var6 = byte(i + 1)
			break
		}
	}
	for i := arg1; i < arg1+arg4; i++ {
		for j := arg0; j < arg0+arg3; j++ {
			if j >= 0 && j < w.MaxTileX && i >= 0 && i < w.MaxTileZ {
				w.LevelTileOverlayIDs[0][j][i] = var6
				for k := range 4 {
					w.LevelHeightMap[k][j][i] = 0
					w.LevelTileFlags[k][j][i] = 0
				}
			}
		}
	}
}

func (w *World) LoadGround(arg0 []byte, arg1, arg3, arg4, arg5 int) {
	var7 := io.NewPacket(arg0)
	for i := range 4 {
		for j := range 64 {
			for k := range 64 {
				var11 := j + arg4
				var12 := k + arg3
				var13 := 0
				if var11 >= 0 && var11 < 104 && var12 >= 0 && var12 < 104 {
					w.LevelTileFlags[i][var11][var12] = 0
					for {
						var13 = var7.G1()
						if var13 == 0 {
							if i == 0 {
								w.LevelHeightMap[0][var11][var12] = -PerlinNoise(var11+932731+arg1, var12+556238+arg5) * 8
							} else {
								w.LevelHeightMap[i][var11][var12] = w.LevelHeightMap[i-1][var11][var12] - 240
							}
							break
						}
						if var13 == 1 {
							var14 := var7.G1()
							if var14 == 1 {
								var14 = 0
							}
							if i == 0 {
								w.LevelHeightMap[0][var11][var12] = -var14 * 8
							} else {
								w.LevelHeightMap[i][var11][var12] = w.LevelHeightMap[i-1][var11][var12] - var14*8
							}
							break
						}
						if var13 <= 49 {
							w.LevelTileOverlayIDs[i][var11][var12] = var7.G1B()
							w.LevelTileOverlayShape[i][var11][var12] = byte((var13 - 2) / 4)
							w.LevelTileOverlayRotation[i][var11][var12] = byte(var13 - 2&0x3)
						} else if var13 <= 81 {
							w.LevelTileFlags[i][var11][var12] = byte(var13 - 49)
						} else {
							w.LevelTileUnderlayIDs[i][var11][var12] = byte(var13 - 81)
						}
					}
				} else {
					for {
						var13 = var7.G1()
						if var13 == 0 {
							break
						}
						if var13 == 1 {
							var7.G1()
							break
						}
						if var13 <= 49 {
							var7.G1()
						}
					}
				}
			}
		}
	}
}

func (w *World) LoadLocations(arg0 []byte, arg1 *world3d.World3D, arg2 []*dash3d.CollisionMap, arg3 datastruct.LinkList[*entity.LocEntity], arg5 int, arg6 int) {
	var8 := io.NewPacket(arg0)
	var9 := -1
	for {
		var10 := var8.GSmartS()
		if var10 == 0 {
			return
		}
		var9 += var10
		var11 := 0
		for {
			var12 := var8.GSmartS()
			if var12 == 0 {
				break
			}
			var11 += var12 - 1
			var13 := var11 & 0x3F
			var14 := var11 >> 6 & 0x3F
			var15 := var11 >> 12
			var16 := var8.G1()
			var17 := var16 >> 2
			var18 := var16 & 0x3
			var19 := var14 + arg6
			var20 := var13 + arg5
			if var19 > 0 && var20 > 0 && var19 < 103 && var20 < 103 {
				var21 := var15
				if w.LevelTileFlags[1][var19][var20]&0x2 == 2 {
					var21 = var15 - 1
				}
				var var22 *dash3d.CollisionMap
				if var21 >= 0 {
					var22 = arg2[var21]
				}
				w.AddLoc(var22, var15, var20, var18, var17, arg1, arg3, var9, var19)
			}
		}
	}
}

func (w *World) AddLoc(arg0 *dash3d.CollisionMap, arg2, arg3, arg4, arg5 int, arg6 *world3d.World3D, arg7 datastruct.LinkList[*entity.LocEntity], arg8, arg9 int) {
	if LowMemory {
		if w.LevelTileFlags[arg2][arg9][arg3]&0x10 != 0 {
			return
		}
		if w.GetDrawLevel(arg2, arg9, arg3) != LevelBuilt {
			return
		}
	}
	var11 := w.LevelHeightMap[arg2][arg9][arg3]
	var12 := w.LevelHeightMap[arg2][arg9+1][arg3]
	var13 := w.LevelHeightMap[arg2][arg9+1][arg3+1]
	var14 := w.LevelHeightMap[arg2][arg9][arg3+1]
	var15 := var11 + var12 + var13 + var14>>2 // TODO: order of operations different in java with shifts?
	var16 := loctype.Get(arg8)
	var17 := arg9 + (arg3 << 7) + (arg8 << 14) + 1073741824
	if !var16.Active {
		var17 += math.MinInt32
	}
	var18 := byte((arg4 << 6) + arg5)
	var var19 *model.Model
	if arg5 != 22 {
		var20 := 0
		if arg5 == 10 || arg5 == 11 {
			var19 = var16.GetModel(10, arg4, var11, var12, var13, var14, -1)
			if var19 != nil {
				var22 := 0
				if arg5 == 11 {
					var22 += 256
				}
				var21 := 0
				if arg4 == 1 || arg4 == 3 {
					var20 = var16.Length
					var21 = var16.Width
				} else {
					var20 = var16.Width
					var21 = var16.Length
				}
				if arg6.AddLoc1(var15, arg2, nil, var17, arg3, arg9, var20, var18, var19, var22, var21) && var16.Shadow {
					for i := 0; i <= var20; i++ {
						for j := 0; j <= var21; j++ {
							var25 := var19.Radius / 4
							if var25 > 30 {
								var25 = 30
							}
							if var25 > int(w.LevelShadeMap[arg2][arg9+i][arg3+j]) {
								w.LevelShadeMap[arg2][arg9+i][arg3+j] = byte(var25)
							}
						}
					}
				}
			}
			if var16.BlockWalk && arg0 != nil {
				arg0.AddLoc(arg4, var16.Length, var16.Width, arg9, arg3, var16.BlockRange)
			}
			if var16.Anim != -1 {
				arg7.AddTail(entity.NewLocEntity(true, arg8, arg2, 2, seqtype.Instances[var16.Anim], arg3, arg9))
			}
		} else if arg5 >= 12 {
			var19 = var16.GetModel(arg5, arg4, var11, var12, var13, var14, -1)
			arg6.AddLoc1(var15, arg2, nil, var17, arg3, arg9, 1, var18, var19, 0, 1)
			if arg5 >= 12 && arg5 <= 17 && arg5 != 13 && arg2 > 0 {
				w.LevelOccludeMap[arg2][arg9][arg3] |= 0x924
			}
			if var16.BlockWalk && arg0 != nil {
				arg0.AddLoc(arg4, var16.Length, var16.Width, arg9, arg3, var16.BlockRange)
			}
			if var16.Anim != -1 {
				arg7.AddTail(entity.NewLocEntity(true, arg8, arg2, 2, seqtype.Instances[var16.Anim], arg3, arg9))
			}
		} else if arg5 == 0 {
			var19 = var16.GetModel(0, arg4, var11, var12, var13, var14, -1)
			arg6.AddWall(0, var15, arg2, ROTATION_WALL_TYPE[arg4], var19, nil, arg9, var17, arg3, var18)
			switch arg4 {
			case 0:
				if var16.Shadow {
					w.LevelShadeMap[arg2][arg9][arg3] = 50
					w.LevelShadeMap[arg2][arg9][arg3+1] = 50
				}
				if var16.Occlude {
					w.LevelOccludeMap[arg2][arg9][arg3] |= 0x249
				}
			case 1:
				if var16.Shadow {
					w.LevelShadeMap[arg2][arg9][arg3+1] = 50
					w.LevelShadeMap[arg2][arg9+1][arg3+1] = 50
				}
				if var16.Occlude {
					w.LevelOccludeMap[arg2][arg9][arg3+1] |= 0x492
				}
			case 2:
				if var16.Shadow {
					w.LevelShadeMap[arg2][arg9+1][arg3] = 50
					w.LevelShadeMap[arg2][arg9+1][arg3+1] = 50
				}
				if var16.Occlude {
					w.LevelOccludeMap[arg2][arg9+1][arg3] |= 0x249
				}
			case 3:
				if var16.Shadow {
					w.LevelShadeMap[arg2][arg9][arg3] = 50
					w.LevelShadeMap[arg2][arg9+1][arg3] = 50
				}
				if var16.Occlude {
					w.LevelOccludeMap[arg2][arg9][arg3] |= 0x492
				}
			}
			if var16.BlockWalk && arg0 != nil {
				arg0.AddWall(arg4, arg3, arg9, var16.BlockRange, arg5)
			}
			if var16.Anim != -1 {
				arg7.AddTail(entity.NewLocEntity(true, arg8, arg2, 0, seqtype.Instances[var16.Anim], arg3, arg9))
			}
			if var16.WallWidth != 16 {
				arg6.SetWallDecorationOffset(arg2, arg3, arg9, var16.WallWidth)
			}
		} else if arg5 == 1 {
			var19 = var16.GetModel(1, arg4, var11, var12, var13, var14, -1)
			arg6.AddWall(0, var15, arg2, ROTATION_WALL_CORNER_TYPE[arg4], var19, nil, arg9, var17, arg3, var18)
			if var16.Shadow {
				switch arg4 {
				case 0:
					w.LevelShadeMap[arg2][arg9][arg3+1] = 50
				case 1:
					w.LevelShadeMap[arg2][arg9+1][arg3+1] = 50
				case 2:
					w.LevelShadeMap[arg2][arg9+1][arg3] = 50
				case 3:
					w.LevelShadeMap[arg2][arg9][arg3] = 50
				}
			}
			if var16.BlockWalk && arg0 != nil {
				arg0.AddWall(arg4, arg3, arg9, var16.BlockRange, arg5)
			}
			if var16.Anim != -1 {
				arg7.AddTail(entity.NewLocEntity(true, arg8, arg2, 0, seqtype.Instances[var16.Anim], arg3, arg9))
			}
		} else {
			var26 := 0
			var var28 *model.Model
			if arg5 == 2 {
				var26 = arg4 + 1&0x3
				var27 := var16.GetModel(2, arg4+4, var11, var12, var13, var14, -1)
				var28 = var16.GetModel(2, var26, var11, var12, var13, var14, -1)
				arg6.AddWall(ROTATION_WALL_TYPE[var26], var15, arg2, ROTATION_WALL_TYPE[arg4], var27, var28, arg9, var17, arg3, var18)
				if var16.Occlude {
					switch arg4 {
					case 0:
						w.LevelOccludeMap[arg2][arg9][arg3] |= 0x249
						w.LevelOccludeMap[arg2][arg9][arg3+1] |= 0x492
					case 1:
						w.LevelOccludeMap[arg2][arg9][arg3+1] |= 0x492
						w.LevelOccludeMap[arg2][arg9+1][arg3] |= 0x249
					case 2:
						w.LevelOccludeMap[arg2][arg9+1][arg3] |= 0x249
						w.LevelOccludeMap[arg2][arg9][arg3] |= 0x492
					case 3:
						w.LevelOccludeMap[arg2][arg9][arg3] |= 0x492
						w.LevelOccludeMap[arg2][arg9][arg3] |= 0x249
					}
				}
				if var16.BlockWalk && arg0 != nil {
					arg0.AddWall(arg4, arg3, arg9, var16.BlockRange, arg5)
				}
				if var16.Anim != -1 {
					arg7.AddTail(entity.NewLocEntity(true, arg8, arg2, 0, seqtype.Instances[var16.Anim], arg3, arg9))
				}
				if var16.WallWidth != 16 {
					arg6.SetWallDecorationOffset(arg2, arg3, arg9, var16.WallWidth)
				}
			} else if arg5 == 3 {
				var19 = var16.GetModel(3, arg4, var11, var12, var13, var14, -1)
				arg6.AddWall(0, var15, arg2, ROTATION_WALL_CORNER_TYPE[arg4], var19, nil, arg9, var17, arg3, var18)
				if var16.Shadow {
					switch arg4 {
					case 0:
						w.LevelShadeMap[arg2][arg9][arg3+1] = 50
					case 1:
						w.LevelShadeMap[arg2][arg9+1][arg3+1] = 50
					case 2:
						w.LevelShadeMap[arg2][arg9+1][arg3] = 50
					case 3:
						w.LevelShadeMap[arg2][arg9][arg3] = 50
					}
				}
				if var16.BlockWalk && arg0 != nil {
					arg0.AddWall(arg4, arg3, arg9, var16.BlockRange, arg5)
				}
				if var16.Anim != -1 {
					arg7.AddTail(entity.NewLocEntity(true, arg8, arg2, 0, seqtype.Instances[var16.Anim], arg3, arg9))
				}
			} else if arg5 == 9 {
				var19 = var16.GetModel(arg5, arg4, var11, var12, var13, var14, -1)
				arg6.AddLoc1(var15, arg2, nil, var17, arg3, arg9, 1, var18, var19, 0, 1)
				if var16.BlockWalk && arg0 != nil {
					arg0.AddLoc(arg4, var16.Length, var16.Width, arg9, arg3, var16.BlockRange)
				}
				if var16.Anim != -1 {
					arg7.AddTail(entity.NewLocEntity(true, arg8, arg2, 2, seqtype.Instances[var16.Anim], arg3, arg9))
				}
			} else if arg5 == 4 {
				var19 = var16.GetModel(4, 0, var11, var12, var13, var14, -1)
				arg6.SetWallDecoration(var15, arg3, 0, var17, arg4*512, ROTATION_WALL_TYPE[arg4], 0, arg9, var19, var18, arg2)
				if var16.Anim != -1 {
					arg7.AddTail(entity.NewLocEntity(true, arg8, arg2, 1, seqtype.Instances[var16.Anim], arg3, arg9))
				}
			} else if arg5 == 5 {
				var26 = 16
				var20 = arg6.GetWallBitSet(arg2, arg9, arg3)
				if var20 > 0 {
					var26 = loctype.Get(var20 >> 14 & 0x7FFF).WallWidth
				}
				var28 = var16.GetModel(4, 0, var11, var12, var13, var14, -1)
				arg6.SetWallDecoration(var15, arg3, WALL_DECORATION_ROTATION_FORWARD_Z[arg4]*var26, var17, arg4*512, ROTATION_WALL_TYPE[arg4], WALL_DECORATION_ROTATION_FORWARD_X[arg4]*var26, arg9, var28, var18, arg2)
				if var16.Anim != -1 {
					arg7.AddTail(entity.NewLocEntity(true, arg8, arg2, 1, seqtype.Instances[var16.Anim], arg3, arg9))
				}
			} else if arg5 == 6 {
				var19 = var16.GetModel(4, 0, var11, var12, var13, var14, -1)
				arg6.SetWallDecoration(var15, arg3, 0, var17, arg4, 256, 0, arg9, var19, var18, arg2)
				if var16.Anim != -1 {
					arg7.AddTail(entity.NewLocEntity(true, arg8, arg2, 1, seqtype.Instances[var16.Anim], arg3, arg9))
				}
			} else if arg5 == 7 {
				var19 = var16.GetModel(4, 0, var11, var12, var13, var14, -1)
				arg6.SetWallDecoration(var15, arg3, 0, var17, arg4, 512, 0, arg9, var19, var18, arg2)
				if var16.Anim != -1 {
					arg7.AddTail(entity.NewLocEntity(true, arg8, arg2, 1, seqtype.Instances[var16.Anim], arg3, arg9))
				}
			} else if arg5 == 8 {
				var19 = var16.GetModel(4, 0, var11, var12, var13, var14, -1)
				arg6.SetWallDecoration(var15, arg3, 0, var17, arg4, 768, 0, arg9, var19, var18, arg2)
				if var16.Anim != -1 {
					arg7.AddTail(entity.NewLocEntity(true, arg8, arg2, 1, seqtype.Instances[var16.Anim], arg3, arg9))
				}
			}
		}
	} else if !LowMemory || var16.Active || var16.ForceDecor {
		var19 = var16.GetModel(22, arg4, var11, var12, var13, var14, -1)
		arg6.AddGroundDecoration(var19, arg9, var17, arg3, arg2, var18, var15)
		if var16.BlockWalk && var16.Active && arg0 != nil {
			arg0.SetBlocked(arg3, arg9)
		}
		if var16.Anim != -1 {
			arg7.AddTail(entity.NewLocEntity(true, arg8, arg2, 3, seqtype.Instances[var16.Anim], arg3, arg9))
		}
	}
}

func (w *World) Build(arg0 *world3d.World3D, arg2 []*dash3d.CollisionMap) {
	var7 := 0
	for i := range 4 {
		for j := range 104 {
			for k := range 104 {
				if w.LevelTileFlags[i][j][k]&0x1 == 1 {
					var7 = i
					if w.LevelTileFlags[1][j][k]&0x2 == 2 {
						var7 = i - 1
					}
					if var7 >= 0 {
						arg2[var7].SetBlocked(k, j)
					}
				}
			}
		}
	}
	RandomHueOffset += int(rand.Float64()*5.0) - 2
	if RandomHueOffset < -8 {
		RandomHueOffset = -8
	}
	if RandomHueOffset > 8 {
		RandomHueOffset = 8
	}
	RandomLightnessOffset += int(rand.Float64()*5.0) - 2
	if RandomLightnessOffset < -16 {
		RandomLightnessOffset = -16
	}
	if RandomLightnessOffset > 16 {
		RandomLightnessOffset = 16
	}
	var12 := 0
	var13 := 0
	var14 := 0
	var15 := 0
	var16 := 0
	var17 := 0
	var18 := 0
	var19 := 0
	var20 := 0
	var21 := 0
	var22 := 0
	var23 := 0
	for i := range 4 {
		var45 := w.LevelShadeMap[i]
		var46 := 96
		var8 := 768
		var9 := -50
		var10 := -10
		var11 := -50
		var12 = int(math.Sqrt(float64(var9*var9 + var10*var10 + var11*var11)))
		var13 = var8 * var12 >> 8
		for j := 1; j < w.MaxTileZ-1; j++ {
			for k := 1; k < w.MaxTileX-1; k++ {
				var16 = w.LevelHeightMap[i][k+1][j] - w.LevelHeightMap[i][k-1][j]
				var17 = w.LevelHeightMap[i][k][j+1] - w.LevelHeightMap[i][k][j-1]
				var18 = int(math.Sqrt(float64(var16*var16 + 65536 + var17*var17)))
				var19 = (var16 << 8) / var18
				var20 = 65536 / var18
				var21 = (var17 << 8) / var18
				var22 = var46 + (var9*var19+var10*var20+var11*var21)/var13
				var23 = int((var45[k-1][j] >> 2) + (var45[k+1][j] >> 3) + (var45[k][j-1] >> 2) + (var45[k][j+1] >> 3) + (var45[k][j] >> 1)) // TODO: questionable conversion
				w.LevelLightMap[k][j] = var22 - var23
			}
		}
		for j := range w.MaxTileZ {
			w.BlendChroma[j] = 0
			w.BlendSaturation[j] = 0
			w.BlendLightness[j] = 0
			w.BlendLuminance[j] = 0
			w.BlendMagnitude[j] = 0
		}
		for j := -5; j < w.MaxTileX+5; j++ {
			for k := range w.MaxTileZ {
				var18 = j + 5
				if var18 >= 0 && var18 < w.MaxTileX {
					var19 = int(w.LevelTileUnderlayIDs[i][var18][k] & 0xFF)
					if var19 > 0 {
						var51 := flotype.Instances[var19-1]
						w.BlendChroma[k] += var51.Chroma
						w.BlendSaturation[k] += var51.Saturation
						w.BlendLightness[k] += var51.Lightness
						w.BlendLuminance[k] += var51.Luminance
						w.BlendMagnitude[k]++
					}
				}
				var19 = j - 5
				if var19 >= 0 && var19 < w.MaxTileX {
					var20 = int(w.LevelTileUnderlayIDs[i][var19][k] & 0xFF)
					if var20 > 0 {
						var52 := flotype.Instances[var20-1]
						w.BlendChroma[k] -= var52.Chroma
						w.BlendSaturation[k] -= var52.Saturation
						w.BlendLightness[k] -= var52.Lightness
						w.BlendLuminance[k] -= var52.Luminance
						w.BlendMagnitude[k]--
					}
				}
			}
			if j >= 1 && j < w.MaxTileX-1 {
				var18 = 0
				var19 = 0
				var20 = 0
				var21 = 0
				var22 = 0
				for l := -5; l < w.MaxTileZ+5; l++ {
					var24 := l + 5
					if var24 >= 0 && var24 < w.MaxTileZ {
						var18 += w.BlendChroma[var24]
						var19 += w.BlendSaturation[var24]
						var20 += w.BlendLightness[var24]
						var21 += w.BlendLuminance[var24]
						var22 += w.BlendMagnitude[var24]
					}
					var25 := l - 5
					if var25 >= 0 && var25 < w.MaxTileZ {
						var18 -= w.BlendChroma[var25]
						var19 -= w.BlendSaturation[var25]
						var20 -= w.BlendLightness[var25]
						var21 -= w.BlendLuminance[var25]
						var22 -= w.BlendMagnitude[var25]
					}
					if l >= 1 && l < w.MaxTileZ-1 && (!LowMemory || (w.LevelTileFlags[i][j][l]&0x10) == 0 && w.GetDrawLevel(i, j, l) == LevelBuilt) {
						var26 := w.LevelTileUnderlayIDs[i][j][l] & 0xFF
						var27 := w.LevelTileOverlayIDs[i][j][l] & 0xFF
						if var26 > 0 || var27 > 0 {
							var28 := w.LevelHeightMap[i][j][l]
							var29 := w.LevelHeightMap[i][j+1][l]
							var30 := w.LevelHeightMap[i][j+1][l+1]
							var31 := w.LevelHeightMap[i][j][l+1]
							var32 := w.LevelLightMap[j][l]
							var33 := w.LevelLightMap[j+1][l]
							var34 := w.LevelLightMap[j+1][l+1]
							var35 := w.LevelLightMap[j][l+1]
							var36 := -1
							var37 := -1
							var38 := 0
							var39 := 0
							if var26 > 0 {
								var38 = var18 * 256 / var21
								var39 = var19 / var22
								var40 := var20 / var22
								var36 = w.HSL24To16(var38, var39, var40)
								var54 := var38 + RandomHueOffset&0xFF
								var40 += RandomLightnessOffset
								if var40 < 0 {
									var40 = 0
								} else if var40 > 255 {
									var40 = 255
								}
								var37 = w.HSL24To16(var54, var39, var40)
							}
							if i > 0 {
								var55 := true
								if var26 == 0 && w.LevelTileOverlayShape[i][j][l] != 0 {
									var55 = false
								}
								if var27 > 0 && !flotype.Instances[var27-1].Occlude {
									var55 = false
								}
								if var55 && var28 == var29 && var28 == var30 && var28 == var31 {
									w.LevelOccludeMap[i][j][l] |= 0x924
								}
							}
							var38 = 0
							if var36 != -1 {
								var38 = pix3d.ColourTable[MulHSL(var37, 96)]
							}
							if var27 == 0 {
								arg0.SetTile(i, j, l, 0, 0, -1, var28, var29, var30, var31, MulHSL(var36, var32), MulHSL(var36, var33), MulHSL(var36, var34), MulHSL(var36, var35), 0, 0, 0, 0, var38, 0)
							} else {
								var39 = int(w.LevelTileOverlayShape[i][j][l] + 1)
								var56 := int(w.LevelTileOverlayRotation[i][j][l])
								var41 := flotype.Instances[var27-1]
								var42 := var41.Texture
								var43 := 0
								var44 := 0
								if var42 >= 0 {
									var44 = pix3d.GetAverageTextureRGB(var42)
									var43 = -1
								} else if var41.RGB == 16711935 {
									var44 = 0
									var43 = -2
									var42 = -1
								} else {
									var43 = w.HSL24To16(var41.Hue, var41.Saturation, var41.Lightness)
									var44 = pix3d.ColourTable[w.AdjustLightness(var41.HSL, 96)]
								}
								arg0.SetTile(i, j, l, var39, var56, var42, var28, var29, var30, var31, MulHSL(var36, var32), MulHSL(var36, var33), MulHSL(var36, var34), MulHSL(var36, var35), w.AdjustLightness(var43, var32), w.AdjustLightness(var43, var33), w.AdjustLightness(var43, var34), w.AdjustLightness(var43, var35), var38, var44)
							}
						}
					}
				}
			}
		}
		for j := 1; j < w.MaxTileZ-1; j++ {
			for k := 1; k < w.MaxTileX-1; k++ {
				arg0.SetDrawLevel(i, k, j, w.GetDrawLevel(i, k, j))
			}
		}
	}
	if !FullBright {
		arg0.BuildModels(-10, 64, -50, 768, -50)
	}
	for i := range w.MaxTileX {
		for j := range w.MaxTileZ {
			if w.LevelTileFlags[1][i][j]&0x2 == 2 {
				arg0.SetBridge(j, i)
			}
		}
	}
	if FullBright {
		return
	}
	var7 = 1
	var47 := 2
	var48 := 4
	for i := range 4 {
		if i > 0 {
			var7 <<= 0x3
			var47 <<= 0x3
			var48 <<= 0x3
		}
		for j := 0; j <= i; j++ {
			for k := 0; k <= w.MaxTileZ; k++ {
				for l := 0; l <= w.MaxTileX; l++ {
					var53 := 0
					if w.LevelOccludeMap[j][l][k]&var7 != 0 {
						var14 = k
						var15 = k
						var16 = j
						var17 = j
						for var14 > 0 && w.LevelOccludeMap[j][l][var14-1]&var7 != 0 {
							var14--
						}
						for var15 < w.MaxTileZ && w.LevelOccludeMap[j][l][var15+1]&var7 != 0 {
							var15++
						}
					label334:
						for var16 > 0 {
							for m := var14; m <= var15; m++ {
								if w.LevelOccludeMap[var16-1][l][m]&var7 == 0 {
									break label334
								}
							}
							var16--
						}
					label323:
						for var17 < i {
							for m := var14; m <= var15; m++ {
								if w.LevelOccludeMap[var17+1][l][m]&var7 == 0 {
									break label323
								}
							}
							var17++
						}
						var18 = (var17 + 1 - var16) * (var15 - var14 + 1)
						if var18 >= 8 {
							var53 = 240
							var20 = w.LevelHeightMap[var17][l][var14] - var53
							var21 = w.LevelHeightMap[var16][l][var14]
							world3d.AddOccluder(var15*128+128, l*128, var21, 1, l*128, i, var20, var14*128)
							for m := var16; m <= var17; m++ {
								for n := var14; n <= var15; n++ {
									w.LevelOccludeMap[m][l][n] &= ^var7
								}
							}
						}
					}
					if w.LevelOccludeMap[j][l][k]&var47 != 0 {
						var14 = l
						var15 = l
						var16 = j
						var17 = j
						for var14 > 0 && w.LevelOccludeMap[j][var14-1][k]&var47 != 0 {
							var14--
						}
						for var15 < w.MaxTileX && w.LevelOccludeMap[j][var15+1][k]&var47 != 0 {
							var15++
						}
					label387:
						for var16 > 0 {
							for m := var14; m <= var15; m++ {
								if w.LevelOccludeMap[var16-1][m][k]&var47 == 0 {
									break label387
								}
							}
							var16--
						}
					label376:
						for var17 < i {
							for m := var14; m <= var15; m++ {
								if w.LevelOccludeMap[var17+1][m][k]&var47 == 0 {
									break label376
								}
							}
							var17++
						}
						var18 = (var17 + 1 - var16) * (var15 - var14 + 1)
						if var18 >= 8 {
							var53 = 240
							var20 = w.LevelHeightMap[var17][var14][k] - var53
							var21 = w.LevelHeightMap[var16][var14][k]
							world3d.AddOccluder(k*128, var14*128, var21, 2, var15*128+128, i, var20, k*128)
							for m := var16; m <= var17; m++ {
								for n := var14; n <= var15; n++ {
									w.LevelOccludeMap[m][n][k] &= ^var47
								}
							}
						}
					}
					if w.LevelOccludeMap[j][l][k]&var48 != 0 {
						var14 = l
						var15 = l
						var16 = k
						var17 = k
						for var16 > 0 && w.LevelOccludeMap[j][l][var16-1]&var48 != 0 {
							var16--
						}
						for var17 < w.MaxTileZ && w.LevelOccludeMap[j][l][var17+1]&var48 != 0 {
							var17++
						}
					label440:
						for var14 > 0 {
							for m := var16; m <= var17; m++ {
								if w.LevelOccludeMap[j][var14-1][m]&var48 == 0 {
									break label440
								}
							}
							var14--
						}
					label429:
						for var15 < w.MaxTileX {
							for m := var16; m <= var17; m++ {
								if w.LevelOccludeMap[j][var15+1][m]&var48 == 0 {
									break label429
								}
							}
							var15++
						}
						if (var15-var14+1)*(var17-var16+1) >= 4 {
							var18 = w.LevelHeightMap[j][var14][var16]
							world3d.AddOccluder(var17*128+128, var14*128, var18, 4, var15*128+128, i, var18, var16*128)
							for m := var14; m <= var15; m++ {
								for n := var16; n <= var17; n++ {
									w.LevelOccludeMap[j][m][n] &= ^var48
								}
							}
						}
					}
				}
			}
		}
	}
}

func (w *World) GetDrawLevel(arg0, arg2, arg3 int) int {
	if w.LevelTileFlags[arg0][arg2][arg3]&0x8 == 0 {
		if arg0 <= 0 || w.LevelTileFlags[1][arg2][arg3]&0x2 == 0 {
			return arg0
		}
		return arg0 - 1
	}
	return 0
}

func PerlinNoise(arg0, arg1 int) int {
	var2 := InterpolatedNoise(arg0+45365, arg1+91923, 4) - 128 + (InterpolatedNoise(arg0+10294, arg1+37821, 2) - 128>>1) + (InterpolatedNoise(arg0, arg1, 1) - 128>>2)
	var2 = int(float64(var2)*0.3) + 35
	if var2 < 10 {
		var2 = 10
	} else if var2 > 60 {
		var2 = 60
	}
	return var2
}

func InterpolatedNoise(arg0, arg1, arg2 int) int {
	var3 := arg0 / arg2
	var4 := arg0&arg2 - 1
	var5 := arg1 / arg2
	var6 := arg1&arg2 - 1
	var7 := SmoothNoise(var3, var5)
	var8 := SmoothNoise(var3+1, var5)
	var9 := SmoothNoise(var3, var5+1)
	var10 := SmoothNoise(var3+1, var5+1)
	var11 := Interpolate(var7, var8, var4, arg2)
	var12 := Interpolate(var9, var10, var4, arg2)
	return Interpolate(var11, var12, var6, arg2)
}

func Interpolate(arg0, arg1, arg2, arg3 int) int {
	var4 := 65536 - pix3d.CosTable[arg2*1024/arg3]>>1
	return (arg0 * (65536 - var4) >> 16) + (arg1 * var4 >> 16)
}

func SmoothNoise(arg0, arg1 int) int {
	var2 := Noise(arg0-1, arg1-1) + Noise(arg0+1, arg1-1) + Noise(arg0-1, arg1+1) + Noise(arg0+1, arg1+1)
	var3 := Noise(arg0-1, arg1) + Noise(arg0+1, arg1) + Noise(arg0, arg1-1) + Noise(arg0, arg1+1)
	var4 := Noise(arg0, arg1)
	return var2/16 + var3/8 + var4/4
}

func Noise(arg0, arg1 int) int {
	var2 := arg0 + arg1*57
	var4 := var2<<13 ^ var2
	var3 := var4*(var4*var4*15731+789221) + 1376312589&math.MaxInt32
	return var3 >> 19 & 0xFF
}

func MulHSL(arg0, arg1 int) int {
	if arg0 == -1 {
		return 12345678
	}
	arg1 = arg1 * (arg0 & 0x7F) / 128
	if arg1 < 2 {
		arg1 = 2
	} else if arg1 > 126 {
		arg1 = 126
	}
	return (arg0 & 0xFF80) + arg1
}

func (w *World) AdjustLightness(arg0, arg1 int) int {
	if arg0 == -2 {
		return 12345678
	}
	if arg0 == -1 {
		if arg1 < 0 {
			arg1 = 0
		} else if arg1 > 127 {
			arg1 = 127
		}
		return 127 - arg1
	}
	arg1 = arg1 * (arg0 & 0x7F) / 128
	if arg1 < 2 {
		arg1 = 2
	} else if arg1 > 126 {
		arg1 = 126
	}
	return (arg0 & 0xFF80) + arg1
}

func (w *World) HSL24To16(arg0, arg1, arg2 int) int {
	if arg2 > 179 {
		arg1 /= 2
	}
	if arg2 > 192 {
		arg1 /= 2
	}
	if arg2 > 217 {
		arg1 /= 2
	}
	if arg2 > 243 {
		arg1 /= 2
	}
	return (arg0 / 4 << 10) + (arg1 / 32 << 7) + arg2/2
}

func AddLoc(arg0 int, arg1 *datastruct.LinkList[*entity.LocEntity], arg2 *dash3d.CollisionMap, arg3 int, arg4 int, arg5 [][][]int, arg7 int, arg8 int, arg9 int, arg10 *world3d.World3D, arg11 int) {
	var12 := arg5[arg11][arg0][arg3]
	var13 := arg5[arg11][arg0+1][arg3]
	var14 := arg5[arg11][arg0+1][arg3+1]
	var15 := arg5[arg11][arg0][arg3+1]
	var16 := var12 + var13 + var14 + var15>>2
	var17 := loctype.Get(arg8)
	var18 := arg0 + (arg3 << 7) + (arg8 << 14) + 1073741824
	if !var17.Active {
		var18 += math.MinInt32
	}
	var19 := byte((arg4 << 6) + arg9)
	var var20 *model.Model
	if arg9 == 22 {
		var20 = var17.GetModel(22, arg4, var12, var13, var14, var15, -1)
		arg10.AddGroundDecoration(var20, arg0, var18, arg3, arg7, var19, var16)
		if var17.BlockWalk && var17.Active {
			arg2.SetBlocked(arg3, arg0)
		}
		if var17.Anim != -1 {
			arg1.AddTail(entity.NewLocEntity(true, arg8, arg7, 3, seqtype.Instances[var17.Anim], arg3, arg0))
		}
		return
	}
	var21 := 0
	if arg9 == 10 || arg9 == 11 {
		var20 = var17.GetModel(10, arg4, var12, var13, var14, var15, -1)
		if var20 != nil {
			var23 := 0
			if arg9 == 11 {
				var23 += 256
			}
			var22 := 0
			if arg4 == 1 || arg4 == 3 {
				var21 = var17.Length
				var22 = var17.Width
			} else {
				var21 = var17.Width
				var22 = var17.Length
			}
			arg10.AddLoc1(var16, arg7, nil, var18, arg3, arg0, var21, var19, var20, var23, var22)
		}
		if var17.BlockWalk {
			arg2.AddLoc(arg4, var17.Length, var17.Width, arg0, arg3, var17.BlockRange)
		}
		if var17.Anim != -1 {
			arg1.AddTail(entity.NewLocEntity(true, arg8, arg7, 2, seqtype.Instances[var17.Anim], arg3, arg0))
		}
	} else if arg9 >= 12 {
		var20 = var17.GetModel(arg9, arg4, var12, var13, var14, var15, -1)
		arg10.AddLoc1(var16, arg7, nil, var18, arg3, arg0, 1, var19, var20, 0, 1)
		if var17.BlockWalk {
			arg2.AddLoc(arg4, var17.Length, var17.Width, arg0, arg3, var17.BlockRange)
		}
		if var17.Anim != -1 {
			arg1.AddTail(entity.NewLocEntity(true, arg8, arg7, 2, seqtype.Instances[var17.Anim], arg3, arg0))
		}
	} else if arg9 == 0 {
		var20 = var17.GetModel(0, arg4, var12, var13, var14, var15, -1)
		arg10.AddWall(0, var16, arg7, ROTATION_WALL_TYPE[arg4], var20, nil, arg0, var18, arg3, var19)
		if var17.BlockWalk {
			arg2.AddWall(arg4, arg3, arg0, var17.BlockRange, arg9)
		}
		if var17.Anim != -1 {
			arg1.AddTail(entity.NewLocEntity(true, arg8, arg7, 0, seqtype.Instances[var17.Anim], arg3, arg0))
		}
	} else if arg9 == 1 {
		var20 = var17.GetModel(1, arg4, var12, var13, var14, var15, -1)
		arg10.AddWall(0, var16, arg7, ROTATION_WALL_CORNER_TYPE[arg4], var20, nil, arg0, var18, arg3, var19)
		if var17.BlockWalk {
			arg2.AddWall(arg4, arg3, arg0, var17.BlockRange, arg9)
		}
		if var17.Anim != -1 {
			arg1.AddTail(entity.NewLocEntity(true, arg8, arg7, 0, seqtype.Instances[var17.Anim], arg3, arg0))
		}
	} else {
		var24 := 0
		var var26 *model.Model
		if arg9 == 2 {
			var24 = arg4 + 1&0x3
			var25 := var17.GetModel(2, arg4+4, var12, var13, var14, var15, -1)
			var26 = var17.GetModel(2, var24, var12, var13, var14, var15, -1)
			arg10.AddWall(ROTATION_WALL_TYPE[var24], var16, arg7, ROTATION_WALL_TYPE[arg4], var25, var26, arg0, var18, arg3, var19)
			if var17.BlockWalk {
				arg2.AddWall(arg4, arg3, arg0, var17.BlockRange, arg9)
			}
			if var17.Anim != -1 {
				arg1.AddTail(entity.NewLocEntity(true, arg8, arg7, 0, seqtype.Instances[var17.Anim], arg3, arg0))
			}
		} else if arg9 == 3 {
			var20 = var17.GetModel(3, arg4, var12, var13, var14, var15, -1)
			arg10.AddWall(0, var16, arg7, ROTATION_WALL_CORNER_TYPE[arg4], var20, nil, arg0, var18, arg3, var19)
			if var17.BlockWalk {
				arg2.AddWall(arg4, arg3, arg0, var17.BlockRange, arg9)
			}
			if var17.Anim != -1 {
				arg1.AddTail(entity.NewLocEntity(true, arg8, arg7, 0, seqtype.Instances[var17.Anim], arg3, arg0))
			}
		} else if arg9 == 9 {
			var20 = var17.GetModel(arg9, arg4, var12, var13, var14, var15, -1)
			arg10.AddLoc1(var16, arg7, nil, var18, arg3, arg0, 1, var19, var20, 0, 1)
			if var17.BlockWalk {
				arg2.AddLoc(arg4, var17.Length, var17.Width, arg0, arg3, var17.BlockRange)
			}
			if var17.Anim != -1 {
				arg1.AddTail(entity.NewLocEntity(true, arg8, arg7, 2, seqtype.Instances[var17.Anim], arg3, arg0))
			}
		} else if arg9 == 4 {
			var20 = var17.GetModel(4, 0, var12, var13, var14, var15, -1)
			arg10.SetWallDecoration(var16, arg3, 0, var18, arg4*512, ROTATION_WALL_TYPE[arg4], 0, arg0, var20, var19, arg7)
			if var17.Anim != -1 {
				arg1.AddTail(entity.NewLocEntity(true, arg8, arg7, 1, seqtype.Instances[var17.Anim], arg3, arg0))
			}
		} else if arg9 == 5 {
			var24 = 16
			var21 = arg10.GetWallBitSet(arg7, arg0, arg3)
			if var21 > 0 {
				var24 = loctype.Get(var21 >> 14 & 0x7FFF).WallWidth
			}
			var26 = var17.GetModel(4, 0, var12, var13, var14, var15, -1)
			arg10.SetWallDecoration(var16, arg3, WALL_DECORATION_ROTATION_FORWARD_Z[arg4]*var24, var18, arg4*512, ROTATION_WALL_TYPE[arg4], WALL_DECORATION_ROTATION_FORWARD_X[arg4]*var24, arg0, var26, var19, arg7)
			if var17.Anim != -1 {
				arg1.AddTail(entity.NewLocEntity(true, arg8, arg7, 1, seqtype.Instances[var17.Anim], arg3, arg0))
			}
		} else if arg9 == 6 {
			var20 = var17.GetModel(4, 0, var12, var13, var14, var15, -1)
			arg10.SetWallDecoration(var16, arg3, 0, var18, arg4, 256, 0, arg0, var20, var19, arg7)
			if var17.Anim != -1 {
				arg1.AddTail(entity.NewLocEntity(true, arg8, arg7, 1, seqtype.Instances[var17.Anim], arg3, arg0))
			}
		} else if arg9 == 7 {
			var20 = var17.GetModel(4, 0, var12, var13, var14, var15, -1)
			arg10.SetWallDecoration(var16, arg3, 0, var18, arg4, 512, 0, arg0, var20, var19, arg7)
			if var17.Anim != -1 {
				arg1.AddTail(entity.NewLocEntity(true, arg8, arg7, 1, seqtype.Instances[var17.Anim], arg3, arg0))
			}
		} else if arg9 == 8 {
			var20 = var17.GetModel(4, 0, var12, var13, var14, var15, -1)
			arg10.SetWallDecoration(var16, arg3, 0, var18, arg4, 768, 0, arg0, var20, var19, arg7)
			if var17.Anim != -1 {
				arg1.AddTail(entity.NewLocEntity(true, arg8, arg7, 1, seqtype.Instances[var17.Anim], arg3, arg0))
			}
		}
	}
}
