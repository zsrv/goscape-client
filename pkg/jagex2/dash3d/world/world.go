package world

import (
	"math/rand"
	"strings"

	"goscape-client/pkg/jagex2/config/flotype"
	"goscape-client/pkg/jagex2/dash3d"
	"goscape-client/pkg/jagex2/datastruct"
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

func (w *World) LoadLocations(arg0 []byte, arg1 *World3D, arg2 []*dash3d.CollisionMap, arg3 datastruct.LinkList[any], arg5 int, arg6 int) {
	// TODO
}
