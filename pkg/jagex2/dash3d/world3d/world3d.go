package world3d

import (
	"fmt"
	"math"

	"goscape-client/pkg/jagex2/dash3d"
	"goscape-client/pkg/jagex2/dash3d/entity"
	"goscape-client/pkg/jagex2/dash3d/typ"
	"goscape-client/pkg/jagex2/datastruct"
	"goscape-client/pkg/jagex2/graphics/model"
	"goscape-client/pkg/jagex2/graphics/pix2d"
	"goscape-client/pkg/jagex2/graphics/pix3d"
)

var (
	LowMemory                            bool = true
	TilesRemaining                       int
	TopLevel                             int
	Cycle                                int
	MinDrawTileX                         int
	MaxDrawTileX                         int
	MinDrawTileZ                         int
	MaxDrawTileZ                         int
	EyeTileX                             int
	EyeTileZ                             int
	WALL_CORNER_TYPE_16_BLOCK_LOC_SPANS  = []int{0, 0, 2, 0, 0, 2, 1, 1, 0}
	WALL_CORNER_TYPE_32_BLOCK_LOC_SPANS  = []int{2, 0, 0, 2, 0, 0, 0, 4, 4}
	WALL_CORNER_TYPE_64_BLOCK_LOC_SPANS  = []int{0, 4, 4, 8, 0, 0, 8, 0, 0}
	WALL_CORNER_TYPE_128_BLOCK_LOC_SPANS = []int{1, 1, 0, 0, 0, 8, 0, 0, 8}
	TEXTURE_HSL                          = []int{41, 39248, 41, 4643, 41, 41, 41, 41, 41, 41, 41, 41, 41, 41, 41, 43086, 41, 41, 41, 41, 41, 41, 41, 8602, 41, 28992, 41, 41, 41, 41, 41, 5056, 41, 41, 41, 41, 41, 41, 41, 41, 41, 41, 41, 41, 41, 41, 3131, 41, 41, 41}
	VisibilityMatrix                     [][][][]bool
	VisibilityMap                        [][]bool
	ViewportCenterX                      int
	ViewportCenterY                      int
	ViewportLeft                         int
	ViewportTop                          int
	ViewportRight                        int
	ViewportBottom                       int
	LocBuffer                            []*typ.Location = make([]*typ.Location, 100)
	WALL_DECORATION_INSET_X                              = []int{53, -53, -53, 53}
	WALL_DECORATION_INSET_Z                              = []int{-53, -53, 53, 53}
	WALL_DECORATION_OUTSET_X                             = []int{-45, 45, 45, -45}
	WALL_DECORATION_OUTSET_Z                             = []int{45, 45, -45, -45}
	ClickTileX                           int             = -1
	ClickTileZ                           int             = -1
	LEVEL_COUNT                          int             = 4
	LevelOccluderCount                   []int           = make([]int, LEVEL_COUNT)
	LevelOccluders                       [][]*dash3d.Occlude
	ActiveOccluders                      []*dash3d.Occlude = make([]*dash3d.Occlude, 500)
	DrawTileQueue                                          = datastruct.NewLinkList[*typ.Ground]()
	FRONT_WALL_TYPES                                       = []int{19, 55, 38, 155, 255, 110, 137, 205, 76}
	DIRECTION_ALLOW_WALL_CORNER_TYPE                       = []int{160, 192, 80, 96, 0, 144, 80, 48, 160}
	BACK_WALL_TYPES                                        = []int{76, 8, 137, 4, 0, 1, 38, 2, 19}
	EyeX                                 int
	EyeY                                 int
	EyeZ                                 int
	SinEyePitch                          int
	CosEyePitch                          int
	SinEyeYaw                            int
	CosEyeYaw                            int
	MouseX                               int
	MouseY                               int
	ActiveOccluderCount                  int
	TakingInput                          bool
)

func init() {
	VisibilityMatrix = make([][][][]bool, 8)
	for i := range VisibilityMatrix {
		VisibilityMatrix[i] = make([][][]bool, 32)
		for j := range VisibilityMatrix[i] {
			VisibilityMatrix[i][j] = make([][]bool, 51)
			for k := range VisibilityMatrix[i][j] {
				VisibilityMatrix[i][j][k] = make([]bool, 51)
			}
		}
	}

	LevelOccluders = make([][]*dash3d.Occlude, LEVEL_COUNT)
	for i := range LevelOccluders {
		LevelOccluders[i] = make([]*dash3d.Occlude, 500)
	}
}

type World3D struct {
	MaxLevel                 int
	MaxTileX                 int
	MaxTileZ                 int
	LevelHeightMaps          [][][]int
	LevelTiles               [][][]*typ.Ground
	Minlevel                 int
	TemporaryLocCount        int
	TemporaryLocs            []*typ.Location
	LevelTileOcclusionCycles [][][]int
	MergeIndexA              []int
	MergeIndexB              []int
	TmpMergeIndex            int
	MINIMAP_OVERLAY_SHAPE    [][]int
	MINIMAP_OVERLAY_ROTATION [][]int
}

func NewWorld3D(LevelHeightMaps [][][]int, maxTileZ, maxLevel, maxTileX int) *World3D {
	var w World3D
	w.TemporaryLocs = make([]*typ.Location, 5000)
	w.MergeIndexA = make([]int, 10000)
	w.MergeIndexB = make([]int, 10000)
	w.MINIMAP_OVERLAY_SHAPE = [][]int{make([]int, 16), {1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}, {1, 0, 0, 0, 1, 1, 0, 0, 1, 1, 1, 0, 1, 1, 1, 1}, {1, 1, 0, 0, 1, 1, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0}, {0, 0, 1, 1, 0, 0, 1, 1, 0, 0, 0, 1, 0, 0, 0, 1}, {0, 1, 1, 1, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}, {1, 1, 1, 0, 1, 1, 1, 0, 1, 1, 1, 1, 1, 1, 1, 1}, {1, 1, 0, 0, 1, 1, 0, 0, 1, 1, 0, 0, 1, 1, 0, 0}, {0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 1, 1, 0, 0}, {1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 1, 1, 0, 0, 1, 1}, {1, 1, 1, 1, 1, 1, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0}, {0, 0, 0, 0, 0, 0, 1, 1, 0, 1, 1, 1, 0, 1, 1, 1}, {0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 0, 1, 1, 1, 1}}
	w.MINIMAP_OVERLAY_ROTATION = [][]int{{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}, {12, 8, 4, 0, 13, 9, 5, 1, 14, 10, 6, 2, 15, 11, 7, 3}, {15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}, {3, 7, 11, 15, 2, 6, 10, 14, 1, 5, 9, 13, 0, 4, 8, 12}}

	w.MaxLevel = maxLevel
	w.MaxTileX = maxTileX
	w.MaxTileZ = maxTileZ
	w.LevelTiles = make([][][]*typ.Ground, maxLevel)
	for i := range w.LevelTiles {
		w.LevelTiles[i] = make([][]*typ.Ground, maxTileX)
		for j := range w.LevelTiles[i] {
			w.LevelTiles[i][j] = make([]*typ.Ground, maxTileZ)
		}
	}
	w.LevelTileOcclusionCycles = make([][][]int, maxLevel)
	for i := range w.LevelTileOcclusionCycles {
		w.LevelTileOcclusionCycles[i] = make([][]int, maxTileX+1)
		for j := range w.LevelTileOcclusionCycles[i] {
			w.LevelTileOcclusionCycles[i][j] = make([]int, maxTileZ+1)
		}
	}
	w.LevelHeightMaps = LevelHeightMaps
	w.Reset()
	return &w
}

func Unload() {
	LocBuffer = nil
	LevelOccluderCount = nil
	LevelOccluders = nil
	DrawTileQueue = nil
	VisibilityMatrix = nil
	VisibilityMap = nil
}

func (w *World3D) Reset() {
	for i := range w.MaxLevel {
		for j := range w.MaxTileX {
			for k := range w.MaxTileZ {
				w.LevelTiles[i][j][k] = nil
			}
		}
	}
	for i := range LEVEL_COUNT {
		for j := range LevelOccluderCount[i] {
			LevelOccluders[i][j] = nil
		}
		LevelOccluderCount[i] = 0
	}
	for i := range w.TemporaryLocCount {
		w.TemporaryLocs[i] = nil
	}
	w.TemporaryLocCount = 0
	for i := range len(LocBuffer) {
		LocBuffer[i] = nil
	}
}

func (w *World3D) SetMinLevel(level int) {
	w.Minlevel = level
	for i := range w.MaxTileX {
		for j := range w.MaxTileZ {
			w.LevelTiles[level][i][j] = typ.NewGround(level, i, j)
		}
	}
}

func (w *World3D) SetBridge(arg0, arg1 int) {
	var4 := w.LevelTiles[0][arg1][arg0]
	for i := range 3 {
		w.LevelTiles[i][arg1][arg0] = w.LevelTiles[i+1][arg1][arg0]
		if w.LevelTiles[i][arg1][arg0] != nil {
			w.LevelTiles[i][arg1][arg0].Level--
		}
	}
	if w.LevelTiles[0][arg1][arg0] == nil {
		w.LevelTiles[0][arg1][arg0] = typ.NewGround(0, arg1, arg0)
	}
	w.LevelTiles[0][arg1][arg0].Bridge = var4
	w.LevelTiles[3][arg1][arg0] = nil
}

func AddOccluder(arg0, arg1, arg3, arg4, arg5, arg6, arg7, arg8 int) {
	var9 := dash3d.NewOcclude()
	var9.MinTileX = arg1 / 128
	var9.MaxTileX = arg5 / 128
	var9.MinTileZ = arg8 / 128
	var9.MaxTileZ = arg0 / 128
	var9.Type = arg4
	var9.MinX = arg1
	var9.MaxX = arg5
	var9.MinZ = arg8
	var9.MaxZ = arg0
	var9.MinY = arg7
	var9.MaxY = arg3
	LevelOccluders[arg6][LevelOccluderCount[arg6]] = var9
	LevelOccluderCount[arg6]++
}

func (w *World3D) SetDrawLevel(arg0, arg1, arg2, arg3 int) {
	var5 := w.LevelTiles[arg0][arg1][arg2]
	if var5 != nil {
		w.LevelTiles[arg0][arg1][arg2].DrawLevel = arg3
	}
}

func (w *World3D) SetTile(arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7, arg8, arg9, arg10, arg11, arg12, arg13, arg14, arg15, arg16, arg17, arg18, arg19 int) {
	switch arg3 {
	case 0:
		var21 := typ.NewTileUnderlay(arg10, arg11, arg12, arg13, -1, arg18, false)
		for i := arg0; i >= 0; i-- {
			if w.LevelTiles[i][arg1][arg2] == nil {
				w.LevelTiles[i][arg1][arg2] = typ.NewGround(i, arg1, arg2)
			}
		}
		w.LevelTiles[arg0][arg1][arg2].Underlay = var21
	case 1:
		var21 := typ.NewTileUnderlay(arg14, arg15, arg16, arg17, arg5, arg19, arg6 == arg7 && arg6 == arg8 && arg6 == arg9)
		for i := arg0; i >= 0; i-- {
			if w.LevelTiles[i][arg1][arg2] == nil {
				w.LevelTiles[i][arg1][arg2] = typ.NewGround(i, arg1, arg2)
			}
		}
		w.LevelTiles[arg0][arg1][arg2].Underlay = var21
	default:
		var23 := typ.NewTileOverlay(arg1, arg3, arg15, arg7, arg12, arg4, arg10, arg9, arg19, arg14, arg5, arg17, arg18, arg8, arg16, arg13, arg6, arg2, arg11)
		for i := arg0; i >= 0; i-- {
			if w.LevelTiles[i][arg1][arg2] == nil {
				w.LevelTiles[i][arg1][arg2] = typ.NewGround(i, arg1, arg2)
			}
		}
		w.LevelTiles[arg0][arg1][arg2].Overlay = var23
	}
}

func (w *World3D) AddGroundDecoration(arg0 *model.Model, arg2, arg3, arg4, arg5 int, arg6 byte, arg7 int) {
	var9 := typ.NewGroundDecor()
	var9.Model = arg0
	var9.X = arg2*128 + 64
	var9.Z = arg4*128 + 64
	var9.Y = arg7
	var9.BitSet = arg3
	var9.Info = arg6
	if w.LevelTiles[arg5][arg2][arg4] == nil {
		w.LevelTiles[arg5][arg2][arg4] = typ.NewGround(arg5, arg2, arg4)
	}
	w.LevelTiles[arg5][arg2][arg4].GroundDecor = var9
}

func (w *World3D) AddObjStack(arg0, arg1 *model.Model, arg2, arg3, arg4, arg5, arg6 int, arg7 *model.Model) {
	var10 := typ.NewGroundObject()
	var10.TopObj = arg0
	var10.X = arg6*128 + 64
	var10.Z = arg5*128 + 64
	var10.Y = arg2
	var10.BitSet = arg4
	var10.BottomObj = arg1
	var10.MiddleObj = arg7

	var11 := 0
	var12 := w.LevelTiles[arg3][arg6][arg5]
	if var12 != nil {
		for i := range var12.LocCount {
			var14 := var12.Locs[i].Model.ObjRaise
			if var14 > var11 {
				var11 = var14
			}
		}
	}
	var10.Offset = var11
	if w.LevelTiles[arg3][arg6][arg5] == nil {
		w.LevelTiles[arg3][arg6][arg5] = typ.NewGround(arg3, arg6, arg5)
	}
	w.LevelTiles[arg3][arg6][arg5].GroundObj = var10
}

func (w *World3D) AddWall(arg0, arg1, arg2, arg3 int, arg5 *model.Model, arg6 *model.Model, arg7, arg8, arg9 int, arg10 byte) {
	if arg5 == nil && arg6 == nil {
		return
	}
	var12 := typ.NewWall()
	var12.BitSet = arg8
	var12.Info = arg10
	var12.X = arg7*128 + 64
	var12.Z = arg9*128 + 64
	var12.Y = arg1
	var12.ModelA = arg5
	var12.ModelB = arg6
	var12.TypeA = arg3
	var12.TypeB = arg0
	for i := arg2; i >= 0; i-- {
		if w.LevelTiles[i][arg7][arg9] == nil {
			w.LevelTiles[i][arg7][arg9] = typ.NewGround(i, arg7, arg9)
		}
	}
	w.LevelTiles[arg2][arg7][arg9].Wall = var12
}

func (w *World3D) SetWallDecoration(arg0, arg1, arg2, arg3, arg4, arg5, arg7, arg8 int, arg9 *model.Model, arg10 byte, arg11 int) {
	if arg9 == nil {
		return
	}
	var13 := typ.NewDecor()
	var13.BitSet = arg3
	var13.Info = arg10
	var13.X = arg8*128 + 64 + arg7
	var13.Z = arg1*128 + 64 + arg2
	var13.Y = arg0
	var13.Model = arg9
	var13.Type = arg5
	var13.Angle = arg4
	for i := arg11; i >= 0; i-- {
		if w.LevelTiles[i][arg8][arg1] == nil {
			w.LevelTiles[i][arg8][arg1] = typ.NewGround(i, arg8, arg1)
		}
	}
	w.LevelTiles[arg11][arg8][arg1].Decor = var13
}

func (w *World3D) AddLoc1(arg0 int, arg2 int, arg3 entity.Entity, arg4, arg5, arg6, arg7 int, arg8 byte, arg9 *model.Model, arg10, arg11 int) bool {
	if arg9 == nil && arg3 == nil {
		return true
	}
	var13 := arg6*128 + arg7*64
	var14 := arg5*128 + arg11*64
	return w.AddLoc2(arg2, arg6, arg5, arg7, arg11, var13, var14, arg0, arg9, arg3, arg10, false, arg4, arg8)
}

func (w *World3D) AddTemporary1(arg1, arg2, arg3, arg4, arg5 int, arg6 bool, arg7 *model.Model, arg8 entity.Entity, arg9, arg10 int) bool {
	if arg7 == nil && arg8 == nil {
		return true
	}
	var12 := arg4 - arg2
	var13 := arg1 - arg2
	var14 := arg4 + arg2
	var15 := arg1 + arg2
	if arg6 {
		if arg3 > 640 && arg3 < 1408 {
			var15 += 128
		}
		if arg3 > 1152 && arg3 < 1920 {
			var14 += 128
		}
		if arg3 > 1664 || arg3 < 384 {
			var13 -= 128
		}
		if arg3 > 128 && arg3 < 896 {
			var12 -= 128
		}
	}
	var12 /= 128
	var13 /= 128
	var14 /= 128
	var15 /= 128
	return w.AddLoc2(arg10, var12, var13, var14-var12+1, var15-var13+1, arg4, arg1, arg9, arg7, arg8, arg3, true, arg5, byte(0))
}

func (w *World3D) AddTemporary2(arg0 int, arg2 *model.Model, arg3, arg4, arg5, arg6, arg7, arg8 int, arg9 entity.Entity, arg11, arg12, arg13 int) bool {
	if arg2 == nil && arg9 == nil {
		return true
	}
	return w.AddLoc2(arg11, arg8, arg7, arg0-arg8+1, arg12-arg7+1, arg13, arg3, arg4, arg2, arg9, arg6, true, arg5, byte(0))
}

func (w *World3D) AddLoc2(arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7 int, arg8 *model.Model, arg9 entity.Entity, arg10 int, arg11 bool, arg12 int, arg13 byte) bool {
	if arg8 == nil && arg9 == nil {
		return false
	}
	for i := arg1; i < arg1+arg3; i++ {
		for j := arg2; j < arg2+arg4; j++ {
			if i < 0 || j < 0 || i >= w.MaxTileX || j >= w.MaxTileZ {
				return false
			}
			var17 := w.LevelTiles[arg0][i][j]
			if var17 != nil && var17.LocCount >= 5 {
				return false
			}
		}
	}
	var22 := typ.NewLocation()
	var22.BitSet = arg12
	var22.Info = arg13
	var22.Level = arg0
	var22.X = arg5
	var22.Z = arg6
	var22.Y = arg7
	var22.Model = arg8
	var22.Entity = arg9
	var22.Yaw = arg10
	var22.MinSceneTileX = arg1
	var22.MinSceneTileZ = arg2
	var22.MaxSceneTileX = arg1 + arg3 - 1
	var22.MaxSceneTileZ = arg2 + arg4 - 1
	for i := arg1; i < arg1+arg3; i++ {
		for j := arg2; j < arg2+arg4; j++ {
			var19 := 0
			if i > arg1 {
				var19++
			}
			if i < arg1+arg3-1 {
				var19 += 4
			}
			if j > arg2 {
				var19 += 8
			}
			if j < arg2+arg4-1 {
				var19 += 2
			}
			for k := arg0; k >= 0; k-- {
				if w.LevelTiles[k][i][j] == nil {
					w.LevelTiles[k][i][j] = typ.NewGround(k, i, j)
				}
			}
			var21 := w.LevelTiles[arg0][i][j]
			var21.Locs[var21.LocCount] = var22
			var21.LocSpan[var21.LocCount] = var19
			var21.LocSpans |= var19
			var21.LocCount++
		}
	}
	if arg11 {
		w.TemporaryLocs[w.TemporaryLocCount] = var22
		w.TemporaryLocCount++
	}
	return true
}

func (w *World3D) ClearTemporaryLocs() {
	for i := range w.TemporaryLocCount {
		var3 := w.TemporaryLocs[i]
		w.RemoveLoc1(var3)
		w.TemporaryLocs[i] = nil
	}
	w.TemporaryLocCount = 0
}

func (w *World3D) RemoveLoc1(arg0 *typ.Location) {
	for i := arg0.MinSceneTileX; i <= arg0.MaxSceneTileX; i++ {
		for j := arg0.MinSceneTileZ; j <= arg0.MaxSceneTileZ; j++ {
			var5 := w.LevelTiles[arg0.Level][i][j]
			if var5 != nil {
				for k := range var5.LocCount {
					if var5.Locs[k] == arg0 {
						var5.LocCount--
						for l := k; l < var5.LocCount; l++ {
							var5.Locs[l] = var5.Locs[l+1]
							var5.LocSpan[l] = var5.LocSpan[l+1]
						}
						var5.Locs[var5.LocCount] = nil
						break
					}
				}
				var5.LocSpans = 0
				for k := 0; k < var5.LocCount; k++ {
					var5.LocSpans |= var5.LocSpan[k]
				}
			}
		}
	}
}

func (w *World3D) SetLocModel(arg0 int, arg1 *model.Model, arg3 int, arg4 int) {
	if arg1 == nil {
		return
	}
	var6 := w.LevelTiles[arg3][arg0][arg4]
	if var6 == nil {
		return
	}
	for i := range var6.LocCount {
		var8 := var6.Locs[i]
		if var8.BitSet>>29&0x3 == 2 {
			var8.Model = arg1
			return
		}
	}
}

func (w *World3D) SetWallDecorationOffset(arg0, arg1, arg2, arg3 int) {
	var6 := w.LevelTiles[arg0][arg2][arg1]
	if var6 == nil {
		return
	}
	var10 := var6.Decor
	if var10 != nil {
		var8 := arg2*128 + 64
		var9 := arg1*128 + 64
		var10.X = var8 + (var10.X-var8)*arg3/16
		var10.Z = var9 + (var10.Z-var9)*arg3/16
	}
}

func (w *World3D) SetWallDecorationModel(arg1 int, arg2 int, arg3 *model.Model, arg4 int) {
	if arg3 == nil {
		return
	}
	var6 := w.LevelTiles[arg4][arg2][arg1]
	if var6 != nil {
		var7 := var6.Decor
		if var7 != nil {
			var7.Model = arg3
		}
	}
}

func (w *World3D) SetGroundDecorationModel(arg0 *model.Model, arg1, arg3, arg4 int) {
	if arg0 == nil {
		return
	}
	var6 := w.LevelTiles[arg4][arg3][arg1]
	if var6 != nil {
		var7 := var6.GroundDecor
		if var7 != nil {
			var7.Model = arg0
		}
	}
}

func (w *World3D) SetWallModel(arg1 *model.Model, arg2, arg3, arg4 int) {
	if arg1 == nil {
		return
	}
	var8 := w.LevelTiles[arg4][arg3][arg2]
	if var8 != nil {
		var7 := var8.Wall
		if var7 != nil {
			var7.ModelA = arg1
		}
	}
}

func (w *World3D) SetWallModels(arg0, arg1 *model.Model, arg2, arg4, arg5 int) {
	if arg0 == nil {
		return
	}
	var7 := w.LevelTiles[arg5][arg4][arg2]
	if var7 == nil {
		return
	}
	var8 := var7.Wall
	if var8 == nil {
		return
	}
	var8.ModelA = arg0
	var8.ModelB = arg1
}

func (w *World3D) RemoveWall(arg0, arg1, arg2 int) {
	var5 := w.LevelTiles[arg1][arg0][arg2]
	if var5 != nil {
		var5.Wall = nil
	}
}

func (w *World3D) RemoveWallDecoration(arg0, arg1, arg3 int) {
	var5 := w.LevelTiles[arg0][arg3][arg1]
	if var5 != nil {
		var5.Decor = nil
	}
}

func (w *World3D) RemoveLoc2(arg0, arg1, arg3 int) {
	var5 := w.LevelTiles[arg3][arg0][arg1]
	if var5 == nil {
		return
	}
	for i := range var5.LocCount {
		var7 := var5.Locs[i]
		if var7.BitSet>>29&0x3 == 2 && var7.MinSceneTileX == arg0 && var7.MinSceneTileZ == arg1 {
			w.RemoveLoc1(var7)
			return
		}
	}
}

func (w *World3D) RemoveGroundDecoration(arg0, arg2, arg3 int) {
	var5 := w.LevelTiles[arg0][arg2][arg3]
	if var5 != nil {
		var5.GroundDecor = nil
	}
}

func (w *World3D) RemoveObjStack(arg0, arg1, arg2 int) {
	var4 := w.LevelTiles[arg0][arg1][arg2]
	if var4 != nil {
		var4.GroundObj = nil
	}
}

func (w *World3D) GetWallBitSet(arg0, arg1, arg2 int) int {
	var4 := w.LevelTiles[arg0][arg1][arg2]
	if var4 == nil || var4.Wall == nil {
		return 0
	}
	return var4.Wall.BitSet
}

func (w *World3D) GetWallDecorationBitSet(arg0, arg1, arg3 int) int {
	var5 := w.LevelTiles[arg0][arg3][arg1]
	if var5 == nil || var5.Decor == nil {
		return 0
	}
	return var5.Decor.BitSet
}

func (w *World3D) GetLocBitSet(arg0, arg1, arg2 int) int {
	var4 := w.LevelTiles[arg0][arg1][arg2]
	if var4 == nil {
		return 0
	}
	for i := range var4.LocCount {
		var6 := var4.Locs[i]
		if var6.BitSet>>29&0x3 == 2 && var6.MinSceneTileX == arg1 && var6.MinSceneTileZ == arg2 {
			return var6.BitSet
		}
	}
	return 0
}

func (w *World3D) GetGroundDecorationBitSet(arg0, arg1, arg2 int) int {
	var4 := w.LevelTiles[arg0][arg1][arg1]
	if var4 == nil || var4.GroundDecor == nil {
		return 0
	}
	return var4.GroundDecor.BitSet
}

func (w *World3D) GetInfo(arg0, arg1, arg2, arg3 int) int {
	var5 := w.LevelTiles[arg0][arg1][arg2]
	if var5 == nil {
		return -1
	}
	if var5.Wall != nil && var5.Wall.BitSet == arg3 {
		return int(var5.Wall.Info & 0xFF)
	}
	if var5.Decor != nil && var5.Decor.BitSet == arg3 {
		return int(var5.Decor.Info & 0xFF)
	}
	if var5.GroundDecor != nil && var5.GroundDecor.BitSet == arg3 {
		return int(var5.GroundDecor.Info & 0xFF)
	}
	for i := range var5.LocCount {
		if var5.Locs[i].BitSet == arg3 {
			return int(var5.Locs[i].Info & 0xFF)
		}
	}
	return -1
}

func (w *World3D) BuildModels(arg0, arg1, arg2, arg3, arg4 int) {
	var7 := int(math.Sqrt(float64(arg2*arg2 + arg0*arg0 + arg4*arg4)))
	var8 := arg3 * var7 >> 8
	for i := range w.MaxLevel {
		for j := range w.MaxTileX {
			for k := range w.MaxTileZ {
				var12 := w.LevelTiles[i][j][k]
				if var12 != nil {
					var13 := var12.Wall
					if var13 != nil && var13.ModelA != nil && var13.ModelA.VertexNormal != nil {
						w.MergeLocNormals(j, 1, 1, i, var13.ModelA, k)
						if var13.ModelB != nil && var13.ModelB.VertexNormal != nil {
							w.MergeLocNormals(j, 1, 1, i, var13.ModelB, k)
							w.MergeNormals(var13.ModelA, var13.ModelB, 0, 0, 0, false)
							var13.ModelB.ApplyLighting(arg1, var8, arg2, arg0, arg4)
						}
						var13.ModelA.ApplyLighting(arg1, var8, arg2, arg0, arg4)
					}
					for l := range var12.LocCount {
						var15 := var12.Locs[l]
						if var15 != nil && var15.Model != nil && var15.Model.VertexNormal != nil {
							w.MergeLocNormals(j, var15.MaxSceneTileX-var15.MinSceneTileX+1, var15.MaxSceneTileZ-var15.MinSceneTileZ+1, i, var15.Model, k)
							var15.Model.ApplyLighting(arg1, var8, arg2, arg0, arg4)
						}
					}
					var16 := var12.GroundDecor
					if var16 != nil && var16.Model.VertexNormal != nil {
						w.MergeGroundDecorationNormals(i, k, var16.Model, j)
						var16.Model.ApplyLighting(arg1, var8, arg2, arg0, arg4)
					}
				}
			}
		}
	}
}

func (w *World3D) MergeGroundDecorationNormals(arg1 int, arg2 int, arg3 *model.Model, arg4 int) {
	if arg4 < w.MaxTileX {
		var6 := w.LevelTiles[arg1][arg4+1][arg2]
		if var6 != nil && var6.GroundDecor != nil && var6.GroundDecor.Model.VertexNormal != nil {
			w.MergeNormals(arg3, var6.GroundDecor.Model, 128, 0, 0, true)
		}
	}
	if arg2 < w.MaxTileX {
		var6 := w.LevelTiles[arg1][arg4][arg2+1]
		if var6 != nil && var6.GroundDecor != nil && var6.GroundDecor.Model.VertexNormal != nil {
			w.MergeNormals(arg3, var6.GroundDecor.Model, 0, 0, 128, true)
		}
	}
	if arg4 < w.MaxTileX && arg2 < w.MaxTileZ {
		var6 := w.LevelTiles[arg1][arg4+1][arg2+1]
		if var6 != nil && var6.GroundDecor != nil && var6.GroundDecor.Model.VertexNormal != nil {
			w.MergeNormals(arg3, var6.GroundDecor.Model, 128, 0, 128, true)
		}
	}
	if arg4 >= w.MaxTileX || arg2 <= 0 {
		return
	}
	var6 := w.LevelTiles[arg1][arg4+1][arg2-1]
	if var6 != nil && var6.GroundDecor != nil && var6.GroundDecor.Model.VertexNormal != nil {
		w.MergeNormals(arg3, var6.GroundDecor.Model, 128, 0, -128, true)
	}
}

func (w *World3D) MergeLocNormals(arg0, arg1, arg2, arg3 int, arg5 *model.Model, arg6 int) {
	var8 := true
	var9 := arg0
	var10 := arg0 + arg1
	var11 := arg6 - 1
	var12 := arg6 + arg2
	for i := arg3; i <= arg3+1; i++ {
		if i != w.MaxLevel {
			for j := var9; j <= var10; j++ {
				if j >= 0 && j < w.MaxTileX {
					for k := var11; k <= var12; k++ {
						if k >= 0 && k < w.MaxTileZ && (!var8 || j >= var10 || k >= var12 || k < arg6 && j != arg0) {
							var16 := w.LevelTiles[i][j][k]
							if var16 != nil {
								var17 := (w.LevelHeightMaps[i][j][k]+w.LevelHeightMaps[i][j+1][k]+w.LevelHeightMaps[i][j][k+1]+w.LevelHeightMaps[i][j+1][k+1])/4 - (w.LevelHeightMaps[arg3][arg0][arg6]+w.LevelHeightMaps[arg3][arg0+1][arg6]+w.LevelHeightMaps[arg3][arg0][arg6+1]+w.LevelHeightMaps[arg3][arg0+1][arg6+1])/4
								var18 := var16.Wall
								if var18 != nil && var18.ModelA != nil && var18.ModelA.VertexNormal != nil {
									w.MergeNormals(arg5, var18.ModelA, (j-arg0)*128+(1-arg1)*64, var17, (k-arg6)*128+(1-arg2)*64, var8)
								}
								if var18 != nil && var18.ModelB != nil && var18.ModelB.VertexNormal != nil {
									w.MergeNormals(arg5, var18.ModelB, (j-arg0)*128+(1-arg1)*64, var17, (k-arg6)*128+(1-arg2)*64, var8)
								}
								for l := range var16.LocCount {
									var20 := var16.Locs[l]
									if var20 != nil && var20.Model != nil && var20.Model.VertexNormal != nil {
										var21 := var20.MaxSceneTileX - var20.MinSceneTileX + 1
										var22 := var20.MaxSceneTileZ - var20.MinSceneTileZ + 1
										w.MergeNormals(arg5, var20.Model, (var20.MinSceneTileX-arg0)*128+(var21-arg1)*64, var17, (var20.MinSceneTileZ-arg6)*128+(var22-arg2)*64, var8)
									}
								}
							}
						}
					}
				}
			}
			var9--
			var8 = false
		}
	}
}

func (w *World3D) MergeNormals(arg0, arg1 *model.Model, arg2, arg3, arg4 int, arg5 bool) {
	w.TmpMergeIndex++
	var7 := 0
	var8 := arg1.VertexX
	var9 := arg1.VertexCount
	for i := range arg0.VertexCount {
		var11 := arg0.VertexNormal[i]
		var12 := arg0.VertexNormalOriginal[i]
		if var12.W != 0 {
			var13 := arg0.VertexY[i] - arg3
			if var13 <= arg1.MinY {
				var14 := arg0.VertexX[i] - arg2
				if var14 >= arg1.MinX && var14 <= arg1.MaxX {
					var15 := arg0.VertexZ[i] - arg4
					if var15 >= arg1.MinZ && var15 <= arg1.MaxZ {
						for j := range var9 {
							var17 := arg1.VertexNormal[j]
							var18 := arg1.VertexNormalOriginal[j]
							if var14 == var8[j] && var15 == arg1.VertexZ[j] && var13 == arg1.VertexY[j] && var18.W != 0 {
								var11.X += var18.X
								var11.Y += var18.Y
								var11.Z += var18.Z
								var11.W += var18.W
								var17.X += var12.X
								var17.Y += var12.Y
								var17.Z += var12.Z
								var17.W += var12.W
								var7++
								w.MergeIndexA[i] = w.TmpMergeIndex
								w.MergeIndexB[j] = w.TmpMergeIndex
							}
						}
					}
				}
			}
		}
	}
	if var7 < 3 || !arg5 {
		return
	}
	for i := range arg0.FaceCount {
		if w.MergeIndexA[arg0.FaceVertexA[i]] == w.TmpMergeIndex && w.MergeIndexA[arg0.FaceVertexB[i]] == w.TmpMergeIndex && w.MergeIndexA[arg0.FaceVertexC[i]] == w.TmpMergeIndex {
			arg0.FaceInfo[i] = -1
		}
	}
	for i := range arg1.FaceCount {
		if w.MergeIndexB[arg1.FaceVertexA[i]] == w.TmpMergeIndex && w.MergeIndexB[arg1.FaceVertexB[i]] == w.TmpMergeIndex && w.MergeIndexB[arg1.FaceVertexC[i]] == w.TmpMergeIndex {
			arg1.FaceInfo[i] = -1
		}
	}
}

func (w *World3D) DrawMinimapTile(arg0 []int, arg1, arg2, arg3, arg4, arg5 int) {
	var7 := w.LevelTiles[arg3][arg4][arg5]
	if var7 == nil {
		return
	}
	var8 := var7.Underlay
	if var8 != nil {
		var9 := var8.RGB
		if var9 != 0 {
			for range 4 {
				arg0[arg1] = var9
				arg0[arg1+1] = var9
				arg0[arg1+2] = var9
				arg0[arg1+3] = var9
				arg1 += arg2
			}
		}
		return
	}
	var18 := var7.Overlay
	if var18 == nil {
		return
	}
	var10 := var18.Shape
	var11 := var18.Rotation
	var12 := var18.BackgroundRGB
	var13 := var18.ForegroundRGB
	var14 := w.MINIMAP_OVERLAY_SHAPE[var10]
	var15 := w.MINIMAP_OVERLAY_ROTATION[var11]
	var16 := 0
	if var12 != 0 {
		for range 4 {
			if var14[var15[var16]] == 0 {
				arg0[arg1] = var12
			} else {
				arg0[arg1] = var13
			}
			var16++

			if var14[var15[var16]] == 0 {
				arg0[arg1+1] = var12
			} else {
				arg0[arg1+1] = var13
			}
			var16++

			if var14[var15[var16]] == 0 {
				arg0[arg1+2] = var12
			} else {
				arg0[arg1+2] = var13
			}
			var16++

			if var14[var15[var16]] == 0 {
				arg0[arg1+3] = var12
			} else {
				arg0[arg1+3] = var13
			}
			var16++

			arg1 += arg2
		}
		return
	}
	for range 4 {
		if var14[var15[var16]] != 0 {
			arg0[arg1] = var13
		}
		var16++

		if var14[var15[var16]] != 0 {
			arg0[arg1+1] = var13
		}
		var16++

		if var14[var15[var16]] != 0 {
			arg0[arg1+2] = var13
		}
		var16++

		if var14[var15[var16]] != 0 {
			arg0[arg1+3] = var13
		}
		var16++

		arg1 += arg2
	}
}

func (w *World3D) Init(arg0 []int, arg1, arg2, arg4, arg5 int) {
	ViewportLeft = 0
	ViewportTop = 0
	ViewportRight = arg2
	ViewportBottom = arg4
	ViewportCenterX = arg2 / 2
	ViewportCenterY = arg4 / 2
	var6 := make([][][][]bool, 9)
	for i := range var6 {
		var6[i] = make([][][]bool, 32)
		for j := range var6[i] {
			var6[i][j] = make([][]bool, 53)
			for k := range var6[i][j] {
				var6[i][j][k] = make([]bool, 53)
			}
		}
	}
	for i := 128; i <= 384; i += 32 {
		for j := 0; j < 2048; j += 64 {
			SinEyePitch = model.Sin[i]
			CosEyePitch = model.Cos[i]
			SinEyeYaw = model.Sin[j]
			CosEyeYaw = model.Cos[j]
			var9 := (i - 128) / 32
			var10 := j / 64
			for k := -26; k <= 26; k++ {
				for l := -26; l <= 26; l++ {
					var13 := k * 128
					var14 := l * 128
					var15 := false
					for m := -arg5; m <= arg1; m += 128 {
						if TestPoint(var13, var14, arg0[var9]+m) {
							var15 = true
							break
						}
					}
					var6[var9][var10][k+25+1][l+25+1] = var15
				}
			}
		}
	}
	for i := range 8 {
		for j := range 32 {
			for k := -25; k < 25; k++ {
				for l := -25; l < 25; l++ {
					var17 := false
				label80:
					for m := -1; m <= 1; m++ {
						for n := -1; n <= 1; n++ {
							if var6[i][j][k+m+25+1][l+n+25+1] {
								var17 = true
								break label80
							}
							if var6[i][(j+1)%31][k+m+25+1][l+n+25+1] {
								var17 = true
								break label80
							}
							if var6[i+1][j][k+m+25+1][l+n+25+1] {
								var17 = true
								break label80
							}
							if var6[i+1][(j+1)%31][k+m+25+1][l+n+25+1] {
								var17 = true
								break label80
							}
						}
					}
					VisibilityMatrix[i][j][k+25][l+25] = var17
				}
			}
		}
	}
}

func TestPoint(arg0, arg1, arg2 int) bool {
	var4 := arg1*SinEyeYaw + arg0*CosEyeYaw>>16
	var5 := arg1*CosEyeYaw - arg0*SinEyeYaw>>16
	var6 := arg2*SinEyePitch + var5*CosEyePitch>>16
	var7 := arg2*CosEyePitch - var5*SinEyePitch>>16
	if var6 < 50 || var6 > 3500 {
		return false
	}
	var8 := ViewportCenterX + (var4<<9)/var6
	var9 := ViewportCenterY + (var7<<9)/var6
	if var8 >= ViewportLeft && var8 <= ViewportRight && var9 >= ViewportTop && var9 <= ViewportBottom {
		return true
	} else {
		return false
	}
}

func (w *World3D) Click(mouseY, mouseX int) {
	TakingInput = true
	MouseX = mouseX
	MouseY = mouseY
	ClickTileX = -1
	ClickTileZ = -1
}

func (w *World3D) Draw(arg0, arg1, arg2, arg3, arg4, arg5 int) {
	if arg1 < 0 {
		arg1 = 0
	} else if arg1 >= w.MaxTileX*128 {
		arg1 = w.MaxTileX*128 - 1
	}
	if arg5 < 0 {
		arg5 = 0
	} else if arg5 >= w.MaxTileZ*128 {
		arg5 = w.MaxTileZ*128 - 1
	}
	Cycle++
	SinEyePitch = model.Sin[arg3]
	CosEyePitch = model.Cos[arg3]
	SinEyeYaw = model.Sin[arg0]
	CosEyeYaw = model.Cos[arg0]
	VisibilityMap = VisibilityMatrix[(arg3-128)/32][arg0/64]
	EyeX = arg1
	EyeY = arg4
	EyeZ = arg5
	EyeTileX = arg1 / 128
	EyeTileZ = arg5 / 128
	TopLevel = arg2
	MinDrawTileX = EyeTileX - 25
	if MinDrawTileX < 0 {
		MinDrawTileX = 0
	}
	MinDrawTileZ = EyeTileZ - 25
	if MinDrawTileZ < 0 {
		MinDrawTileZ = 0
	}
	MaxDrawTileX = EyeTileX + 25
	if MaxDrawTileX > w.MaxTileX {
		MaxDrawTileX = w.MaxTileX
	}
	MaxDrawTileZ = EyeTileZ + 25
	if MaxDrawTileZ > w.MaxTileZ {
		MaxDrawTileZ = w.MaxTileZ
	}
	w.UpdateActiveOccluders()
	TilesRemaining = 0
	for i := w.Minlevel; i < w.MaxLevel; i++ {
		var9 := w.LevelTiles[i]
		for j := MinDrawTileX; j < MaxDrawTileX; j++ {
			for k := MinDrawTileZ; k < MaxDrawTileZ; k++ {
				var12 := var9[j][k]
				if var12 != nil {
					if var12.DrawLevel <= arg2 && (VisibilityMap[j-EyeTileX+25][k-EyeTileZ+25] || w.LevelHeightMaps[i][j][k]-arg4 >= 2000) {
						var12.Visible = true
						var12.Update = true
						if var12.LocCount > 0 {
							var12.ContainsLocs = true
						} else {
							var12.ContainsLocs = false
						}
						TilesRemaining++
					} else {
						var12.Visible = false
						var12.Update = false
						var12.CheckLocSpans = 0
					}
				}
			}
		}
	}
	for i := w.Minlevel; i < w.MaxLevel; i++ {
		var20 := w.LevelTiles[i]
		for j := -25; j <= 0; j++ {
			var22 := EyeTileX + j
			var13 := EyeTileX - j
			if var22 >= MinDrawTileX || var13 < MaxDrawTileX {
				for k := -25; k <= 0; k++ {
					var15 := EyeTileZ + k
					var16 := EyeTileZ - k
					var var17 *typ.Ground
					if var22 >= MinDrawTileX {
						if var15 >= MinDrawTileZ {
							var17 = var20[var22][var15]
							if var17 != nil && var17.Visible {
								w.DrawTile(var17, true)
							}
						}
						if var16 < MaxDrawTileZ {
							var17 = var20[var22][var16]
							if var17 != nil && var17.Visible {
								w.DrawTile(var17, true)
							}
						}
					}
					if var13 < MaxDrawTileX {
						if var15 >= MinDrawTileZ {
							var17 = var20[var13][var15]
							if var17 != nil && var17.Visible {
								w.DrawTile(var17, true)
							}
						}
						if var16 < MaxDrawTileZ {
							var17 = var20[var13][var16]
							if var17 != nil && var17.Visible {
								w.DrawTile(var17, true)
							}
						}
					}
					if TilesRemaining == 0 {
						TakingInput = false
						return
					}
				}
			}
		}
	}
	for i := w.Minlevel; i < w.MaxLevel; i++ {
		var21 := w.LevelTiles[i]
		for j := -25; j <= 0; j++ {
			var13 := EyeTileX + j
			var14 := EyeTileX - j
			if var13 >= MinDrawTileX || var14 < MaxDrawTileX {
				for k := -25; k <= 0; k++ {
					var16 := EyeTileZ + k
					var23 := EyeTileZ - k
					var var18 *typ.Ground
					if var13 >= MinDrawTileX {
						if var16 >= MinDrawTileZ {
							var18 = var21[var13][var16]
							if var18 != nil && var18.Visible {
								w.DrawTile(var18, false)
							}
						}
						if var23 < MaxDrawTileZ {
							var18 = var21[var13][var23]
							if var18 != nil && var18.Visible {
								w.DrawTile(var18, false)
							}
						}
					}
					if var14 < MaxDrawTileX {
						if var16 >= MinDrawTileZ {
							var18 = var21[var14][var16]
							if var18 != nil && var18.Visible {
								w.DrawTile(var18, false)
							}
						}
						if var23 < MaxDrawTileZ {
							var18 = var21[var14][var23]
							if var18 != nil && var18.Visible {
								w.DrawTile(var18, false)
							}
						}
					}
					if TilesRemaining == 0 {
						TakingInput = false
						return
					}
				}
			}
		}
	}
}

func (w *World3D) DrawTile(arg0 *typ.Ground, arg1 bool) {
	DrawTileQueue.AddTail(arg0)
	for {
		var var3 *typ.Ground
		var4 := 0
		var5 := 0
		var6 := 0
		var7 := 0
		var var8 [][]*typ.Ground
		var var9 *typ.Ground
		var11 := 0
		var14 := 0
		var15 := 0
		var16 := 0
		var17 := 0
		var18 := 0
		var var25 *typ.Wall
		var26 := 0
		var29 := 0
		for ok1 := true; ok1; ok1 = var9 != nil && var9.Update {
			for ok2 := true; ok2; ok2 = var9 != nil && var9.Update {
				for ok3 := true; ok3; ok3 = var9 != nil && var9.Update {
					for ok4 := true; ok4; ok4 = var9 != nil && var9.Update {
						for ok5 := true; ok5; ok5 = var3.CheckLocSpans != 0 {
							for ok6 := true; ok6; ok6 = !var3.Update {
								for {
									var var12 *typ.Location
									var22 := 0
									var23 := false
									var var35 *typ.Ground
									for {
										for ok7 := true; ok7; ok7 = !var3.Update {
											var3 = DrawTileQueue.RemoveHead()
											if var3 == nil {
												return
											}
										}
										var4 = var3.X
										var5 = var3.Z
										var6 = var3.Level
										var7 = var3.OccludeLevel
										var8 = w.LevelTiles[var6]
										if !var3.Visible {
											break
										}
										if arg1 {
											if var6 > 0 {
												var9 = w.LevelTiles[var6-1][var4][var5]
												if var9 != nil && var9.Update {
													continue
												}
											}
											if var4 <= EyeTileX && var4 > MinDrawTileX {
												var9 = var8[var4-1][var5]
												if var9 != nil && var9.Update && (var9.Visible || (var3.LocSpans&0x1) == 0) {
													continue
												}
											}
											if var4 >= EyeTileX && var4 < MaxDrawTileX-1 {
												var9 = var8[var4+1][var5]
												if var9 != nil && var9.Update && (var9.Visible || (var3.LocSpans&0x4) == 0) {
													continue
												}
											}
											if var5 <= EyeTileZ && var5 > MinDrawTileZ {
												var9 = var8[var4][var5-1]
												if var9 != nil && var9.Update && (var9.Visible || (var3.LocSpans&0x8) == 0) {
													continue
												}
											}
											if var5 >= EyeTileZ && var5 < MaxDrawTileZ-1 {
												var9 = var8[var4][var5+1]
												if var9 != nil && var9.Update && (var9.Visible || (var3.LocSpans&0x2) == 0) {
													continue
												}
											}
										} else {
											arg1 = true
										}
										var3.Visible = false
										if var3.Bridge != nil {
											var9 = var3.Bridge
											if var9.Underlay == nil {
												if var9.Overlay != nil && !w.TileVisible(0, var4, var5) {
													w.DrawTileOverlay(SinEyeYaw, var5, var9.Overlay, var4, CosEyePitch, SinEyePitch, CosEyeYaw)
												}
											} else if !w.TileVisible(0, var4, var5) {
												w.DrawTileUnderlay(var9.Underlay, 0, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, var4, var5)
											}
											var10 := var9.Wall
											if var10 != nil {
												var10.ModelA.Draw1(0, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, var10.X-EyeX, var10.Y-EyeY, var10.Z-EyeZ, var10.BitSet)
											}
											for var11 = 0; var11 < var9.LocCount; var11++ {
												var12 = var9.Locs[var11]
												if var12 != nil {
													var13 := var12.Model
													if var13 == nil {
														var13 = var12.Entity.Draw()
													}
													var13.Draw1(var12.Yaw, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, var12.X-EyeX, var12.Y-EyeY, var12.Z-EyeZ, var12.BitSet)
												}
											}
										}
										var23 = false
										if var3.Underlay == nil {
											if var3.Overlay != nil && !w.TileVisible(var7, var4, var5) {
												var23 = true
												w.DrawTileOverlay(SinEyeYaw, var5, var3.Overlay, var4, CosEyePitch, SinEyePitch, CosEyeYaw)
											}
										} else if !w.TileVisible(var7, var4, var5) {
											var23 = true
											w.DrawTileUnderlay(var3.Underlay, var7, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, var4, var5)
										}
										var22 = 0
										var11 = 0
										var24 := var3.Wall
										var28 := var3.Decor
										if var24 != nil || var28 != nil {
											if EyeTileX == var4 {
												var22++
											} else if EyeTileX < var4 {
												var22 += 2
											}
											if EyeTileZ == var5 {
												var22 += 3
											} else if EyeTileZ > var5 {
												var22 += 6
											}
											var11 = FRONT_WALL_TYPES[var22]
											var3.BackWallTypes = BACK_WALL_TYPES[var22]
										}
										if var24 != nil {
											if var24.TypeA&DIRECTION_ALLOW_WALL_CORNER_TYPE[var22] == 0 {
												var3.CheckLocSpans = 0
											} else if var24.TypeA == 16 {
												var3.CheckLocSpans = 3
												var3.BlockLocSpans = WALL_CORNER_TYPE_16_BLOCK_LOC_SPANS[var22]
												var3.InverseBlockLocSpans = 3 - var3.BlockLocSpans
											} else if var24.TypeA == 32 {
												var3.CheckLocSpans = 6
												var3.BlockLocSpans = WALL_CORNER_TYPE_32_BLOCK_LOC_SPANS[var22]
												var3.InverseBlockLocSpans = 6 - var3.BlockLocSpans
											} else if var24.TypeA == 64 {
												var3.CheckLocSpans = 12
												var3.BlockLocSpans = WALL_CORNER_TYPE_64_BLOCK_LOC_SPANS[var22]
												var3.InverseBlockLocSpans = 12 - var3.BlockLocSpans
											} else {
												var3.CheckLocSpans = 9
												var3.BlockLocSpans = WALL_CORNER_TYPE_128_BLOCK_LOC_SPANS[var22]
												var3.InverseBlockLocSpans = 9 - var3.BlockLocSpans
											}
											if var24.TypeA&var11 != 0 && !w.WallVisible(var7, var4, var5, var24.TypeA) {
												var24.ModelA.Draw1(0, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, var24.X-EyeX, var24.Y-EyeY, var24.Z-EyeZ, var24.BitSet)
											}
											if var24.TypeB&var11 != 0 && w.WallVisible(var7, var4, var5, var24.TypeB) {
												var24.ModelB.Draw1(0, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, var24.X-EyeX, var24.Y-EyeY, var24.Z-EyeZ, var24.BitSet)
											}
										}
										if var28 != nil && !w.Visible(var7, var4, var5, var28.Model.MaxY) {
											if var28.Type&var11 != 0 {
												var28.Model.Draw1(var28.Angle, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, var28.X-EyeX, var28.Y-EyeY, var28.Z-EyeZ, var28.BitSet)
											} else if var28.Type&0x300 != 0 {
												var14 = var28.X - EyeX
												var15 = var28.Y - EyeY
												var16 = var28.Z - EyeZ
												var17 = var28.Angle
												if var17 == 1 || var17 == 2 {
													var18 = -var14
												} else {
													var18 = var14
												}
												var19 := 0
												if var17 == 2 || var17 == 3 {
													var19 = -var16
												} else {
													var19 = var16
												}
												var20 := 0
												var21 := 0
												if var28.Type&0x100 != 0 && var19 < var18 {
													var20 = var14 + WALL_DECORATION_INSET_X[var17]
													var21 = var16 + WALL_DECORATION_INSET_Z[var17]
													var28.Model.Draw1(var17*512+256, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, var20, var15, var21, var28.BitSet)
												}
												if var28.Type&0x200 != 0 && var19 > var18 {
													var20 = var14 + WALL_DECORATION_OUTSET_X[var17]
													var21 = var16 + WALL_DECORATION_OUTSET_Z[var17]
													var28.Model.Draw1(var17*512+1280&0x7FF, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, var20, var15, var21, var28.BitSet)
												}
											}
										}
										if var23 {
											var30 := var3.GroundDecor
											if var30 != nil {
												var30.Model.Draw1(0, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, var30.X-EyeX, var30.Y-EyeY, var30.Z-EyeZ, var30.BitSet)
											}
											var34 := var3.GroundObj
											if var34 != nil && var34.Offset == 0 {
												if var34.BottomObj != nil {
													var34.BottomObj.Draw1(0, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, var34.X-EyeX, var34.Y-EyeY, var34.Z-EyeZ, var34.BitSet)
												}
												if var34.MiddleObj != nil {
													var34.MiddleObj.Draw1(0, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, var34.X-EyeX, var34.Y-EyeY, var34.Z-EyeZ, var34.BitSet)
												}
												if var34.TopObj != nil {
													var34.TopObj.Draw1(0, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, var34.X-EyeX, var34.Y-EyeY, var34.Z-EyeZ, var34.BitSet)
												}
											}
										}
										var14 = var3.LocSpans
										if var14 != 0 {
											if var4 < EyeTileX && var14&0x4 != 0 {
												var35 = var8[var4+1][var5]
												if var35 != nil && var35.Update {
													DrawTileQueue.AddTail(var35)
												}
											}
											if var5 < EyeTileZ && var14&0x2 != 0 {
												var35 = var8[var4][var5+1]
												if var35 != nil && var35.Update {
													DrawTileQueue.AddTail(var35)
												}
											}
											if var4 > EyeTileX && var14&0x1 != 0 {
												var35 = var8[var4-1][var5]
												if var35 != nil && var35.Update {
													DrawTileQueue.AddTail(var35)
												}
											}
											if var5 > EyeTileZ && var14&0x8 != 0 {
												var35 = var8[var4][var5-1]
												if var35 != nil && var35.Update {
													DrawTileQueue.AddTail(var35)
												}
											}
										}
										break
									}
									if var3.CheckLocSpans != 0 {
										var23 = true
										for var22 = 0; var22 < var3.LocCount; var22++ {
											if var3.Locs[var22].Cycle != Cycle && var3.LocSpan[var22]&var3.CheckLocSpans == var3.BlockLocSpans {
												var23 = false
												break
											}
										}
										if var23 {
											var25 = var3.Wall
											if !w.WallVisible(var7, var4, var5, var25.TypeA) {
												var25.ModelA.Draw1(0, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, var25.X-EyeX, var25.Y-EyeY, var25.Z-EyeZ, var25.BitSet)
											}
											var3.CheckLocSpans = 0
										}
									}
									if !var3.ContainsLocs {
										break
									}
									var27 := var3.LocCount
									var3.ContainsLocs = false
									var22 = 0
								label559:
									for var11 = 0; var11 < var27; var11++ {
										var12 = var3.Locs[var11]
										if var12.Cycle != Cycle {
											for var29 = var12.MinSceneTileX; var29 <= var12.MaxSceneTileX; var29++ {
												for var14 = var12.MinSceneTileZ; var14 <= var12.MaxSceneTileZ; var14++ {
													var35 = var8[var29][var14]
													if var35.Visible {
														var3.ContainsLocs = true
														continue label559
													}
													if var35.CheckLocSpans != 0 {
														var16 = 0
														if var29 > var12.MinSceneTileX {
															var16++
														}
														if var29 < var12.MaxSceneTileX {
															var16 += 4
														}
														if var14 > var12.MinSceneTileZ {
															var16 += 8
														}
														if var14 < var12.MaxSceneTileZ {
															var16 += 2
														}
														if var16&var35.CheckLocSpans == var3.InverseBlockLocSpans {
															var3.ContainsLocs = true
															continue label559
														}
													}
												}
											}
											LocBuffer[var22] = var12
											var22++
											var14 = EyeTileX - var12.MinSceneTileX
											var15 = var12.MaxSceneTileX - EyeTileX
											if var15 > var14 {
												var14 = var15
											}
											var16 = EyeTileZ - var12.MinSceneTileZ
											var17 = var12.MaxSceneTileZ - EyeTileZ
											if var17 > var16 {
												var12.Distance = var14 + var17
											} else {
												var12.Distance = var14 + var16
											}
										}
									}
									for var22 > 0 {
										var26 = -50
										var29 = -1
										var var36 *typ.Location
										for var14 = 0; var14 < var22; var14++ {
											var36 = LocBuffer[var14]
											if var36.Distance > var26 && var36.Cycle != Cycle {
												var26 = var36.Distance
												var29 = var14
											}
										}
										if var29 == -1 {
											break
										}
										var36 = LocBuffer[var29]
										var36.Cycle = Cycle
										var37 := var36.Model
										if var37 == nil {
											var37 = var36.Entity.Draw()
										}
										if !w.LocVisible(var7, var36.MinSceneTileX, var36.MaxSceneTileX, var36.MinSceneTileZ, var36.MaxSceneTileZ, var37.MaxY) {
											var37.Draw1(var36.Yaw, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, var36.X-EyeX, var36.Y-EyeY, var36.Z-EyeZ, var36.BitSet)
										}
										for var17 = var36.MinSceneTileX; var17 <= var36.MaxSceneTileX; var17++ {
											for var18 = var36.MinSceneTileZ; var18 <= var36.MaxSceneTileZ; var18++ {
												var38 := var8[var17][var18]
												if var38.CheckLocSpans != 0 {
													DrawTileQueue.AddTail(var38)
												} else if (var17 != var4 || var18 != var5) && var38.Update {
													DrawTileQueue.AddTail(var38)
												}
											}
										}
									}
									if !var3.ContainsLocs {
										break
									}
								}
							}
						}
						if var4 > EyeTileX || var4 <= MinDrawTileX {
							break
						}
						var9 = var8[var4-1][var5]
					}
					if var4 < EyeTileX || var4 >= MaxDrawTileX-1 {
						break
					}
					var9 = var8[var4+1][var5]
				}
				if var5 > EyeTileZ || var5 <= MinDrawTileZ {
					break
				}
				var9 = var8[var4][var5-1]
			}
			if var5 < EyeTileZ || var5 >= MaxDrawTileZ-1 {
				break
			}
			var9 = var8[var4][var5+1]
		}
		var3.Update = false
		TilesRemaining--
		var32 := var3.GroundObj
		if var32 != nil && var32.Offset != 0 {
			if var32.BottomObj != nil {
				var32.BottomObj.Draw1(0, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, var32.X-EyeX, var32.Y-EyeY-var32.Offset, var32.Z-EyeZ, var32.BitSet)
			}
			if var32.MiddleObj != nil {
				var32.MiddleObj.Draw1(0, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, var32.X-EyeX, var32.Y-EyeY-var32.Offset, var32.Z-EyeZ, var32.BitSet)
			}
			if var32.TopObj != nil {
				var32.TopObj.Draw1(0, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, var32.X-EyeX, var32.Y-EyeY-var32.Offset, var32.Z-EyeZ, var32.BitSet)
			}
		}
		if var3.BackWallTypes != 0 {
			var31 := var3.Decor
			if var31 != nil && !w.Visible(var7, var4, var5, var31.Model.MaxY) {
				if var31.Type&var3.BackWallTypes != 0 {
					var31.Model.Draw1(var31.Angle, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, var31.X-EyeX, var31.Y-EyeY, var31.Z-EyeZ, var31.BitSet)
				} else if var31.Type&0x300 != 0 {
					var11 = var31.X - EyeX
					var26 = var31.Y - EyeY
					var29 = var31.Z - EyeZ
					var14 = var31.Angle
					if var14 == 1 || var14 == 2 {
						var15 = -var11
					} else {
						var15 = var11
					}
					if var14 == 2 || var14 == 3 {
						var16 = -var29
					} else {
						var16 = var29
					}
					if var31.Type&0x100 != 0 && var16 >= var15 {
						var17 = var11 + WALL_DECORATION_INSET_X[var14]
						var18 = var29 + WALL_DECORATION_INSET_Z[var14]
						var31.Model.Draw1(var14*512+256, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, var17, var26, var18, var31.BitSet)
					}
					if var31.Type&0x200 != 0 && var16 <= var15 {
						var17 = var11 + WALL_DECORATION_OUTSET_X[var14]
						var18 = var29 + WALL_DECORATION_OUTSET_Z[var14]
						var31.Model.Draw1(var14*512+1280&0x7FF, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, var17, var26, var18, var31.BitSet)
					}
				}
			}
			var25 = var3.Wall
			if var25 != nil {
				if var25.TypeB&var3.BackWallTypes != 0 && !w.WallVisible(var7, var4, var5, var25.TypeB) {
					var25.ModelB.Draw1(0, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, var25.X-EyeX, var25.Y-EyeY, var25.Z-EyeZ, var25.BitSet)
				}
				if var25.TypeA&var3.BackWallTypes != 0 && !w.WallVisible(var7, var4, var5, var25.TypeA) {
					var25.ModelA.Draw1(0, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, var25.X-EyeX, var25.Y-EyeY, var25.Z-EyeZ, var25.BitSet)
				}
			}
		}
		var var33 *typ.Ground
		if var6 < w.MaxLevel-1 {
			var33 = w.LevelTiles[var6+1][var4][var5]
			if var33 != nil && var33.Update {
				DrawTileQueue.AddTail(var33)
			}
		}
		if var4 < EyeTileX {
			var33 = var8[var4+1][var5]
			if var33 != nil && var33.Update {
				DrawTileQueue.AddTail(var33)
			}
		}
		if var5 < EyeTileZ {
			var33 = var8[var4][var5+1]
			if var33 != nil && var33.Update {
				DrawTileQueue.AddTail(var33)
			}
		}
		if var4 > EyeTileX {
			var33 = var8[var4-1][var5]
			if var33 != nil && var33.Update {
				DrawTileQueue.AddTail(var33)
			}
		}
		if var5 > EyeTileZ {
			var33 = var8[var4][var5-1]
			if var33 != nil && var33.Update {
				DrawTileQueue.AddTail(var33)
			}
		}
	}
}

func (w *World3D) DrawTileUnderlay(arg0 *typ.TileUnderlay, arg1, arg2, arg3, arg4, arg5, arg6, arg7 int) {
	var9 := (arg6 << 7) - EyeX
	var10 := var9
	var11 := (arg7 << 7) - EyeZ
	var12 := var11
	var13 := var10 + 128
	var14 := var13
	var15 := var12 + 128
	var16 := var15
	var17 := w.LevelHeightMaps[arg1][arg6][arg7] - EyeY
	var18 := w.LevelHeightMaps[arg1][arg6+1][arg7] - EyeY
	var19 := w.LevelHeightMaps[arg1][arg6+1][arg7+1] - EyeY
	var20 := w.LevelHeightMaps[arg1][arg6][arg7+1] - EyeY
	var21 := var12*arg4 + var10*arg5>>16
	var35 := var12*arg5 - var10*arg4>>16
	var32 := var21
	var41 := var17*arg3 - var35*arg2>>16
	var36 := var17*arg2 + var35*arg3>>16
	var40 := var41
	if var36 < 50 {
		return
	}
	var21 = var11*arg4 + var14*arg5>>16
	var33 := var11*arg5 - var14*arg4>>16
	var14 = var21
	var21 = var18*arg3 - var33*arg2>>16
	var34 := var18*arg2 + var33*arg3>>16
	var18 = var21
	if var34 < 50 {
		return
	}
	var21 = var16*arg4 + var13*arg5>>16
	var16 = var16*arg5 - var13*arg4>>16
	var37 := var21
	var21 = var19*arg3 - var16*arg2>>16
	var16 = var19*arg2 + var16*arg3>>16
	var19 = var21
	if var16 < 50 {
		return
	}
	var21 = var15*arg4 + var9*arg5>>16
	var38 := var15*arg5 - var9*arg4>>16
	var31 := var21
	var21 = var20*arg3 - var38*arg2>>16
	var39 := var20*arg2 + var38*arg3>>16
	if var39 < 50 {
		return
	}
	var22 := pix3d.CenterW3D + (var32<<9)/var36
	var23 := pix3d.CenterH3D + (var40<<9)/var36
	var24 := pix3d.CenterW3D + (var14<<9)/var34
	var25 := pix3d.CenterH3D + (var18<<9)/var34
	var26 := pix3d.CenterW3D + (var37<<9)/var16
	var27 := pix3d.CenterH3D + (var19<<9)/var16
	var28 := pix3d.CenterW3D + (var31<<9)/var39
	var29 := pix3d.CenterH3D + (var21<<9)/var39
	pix3d.Trans = 0
	var30 := 0
	if (var26-var28)*(var25-var29)-(var27-var29)*(var24-var28) > 0 {
		pix3d.HClip = false
		if var26 < 0 || var28 < 0 || var24 < 0 || var26 > pix2d.SafeWidth || var28 > pix2d.SafeWidth || var24 > pix2d.SafeWidth {
			pix3d.HClip = true
		}
		if TakingInput && w.PointInsideTriangle(MouseX, MouseY, var27, var29, var25, var26, var28, var24) {
			ClickTileX = arg6
			ClickTileZ = arg7
		}
		if arg0.TextureID == -1 {
			if arg0.NortheastColor != 12345678 {
				pix3d.GouraudTriangle(var27, var29, var25, var26, var28, var24, arg0.NortheastColor, arg0.NorthwestColor, arg0.SoutheastColor)
			}
		} else if LowMemory {
			var30 = TEXTURE_HSL[arg0.TextureID]
			pix3d.GouraudTriangle(var27, var29, var25, var26, var28, var24, w.MulLightness(arg0.NortheastColor, var30), w.MulLightness(arg0.NorthwestColor, var30), w.MulLightness(arg0.SoutheastColor, var30))
		} else if arg0.Flat {
			pix3d.TextureTriangle(var27, var29, var25, var26, var28, var24, arg0.NortheastColor, arg0.NorthwestColor, arg0.SoutheastColor, var32, var14, var31, var40, var18, var21, var36, var34, var39, arg0.TextureID)
		} else {
			pix3d.TextureTriangle(var27, var29, var25, var26, var28, var24, arg0.NortheastColor, arg0.NorthwestColor, arg0.SoutheastColor, var37, var31, var14, var19, var21, var18, var16, var39, var34, arg0.TextureID)
		}
	}
	if (var22-var24)*(var29-var25)-(var23-var25)*(var28-var24) <= 0 {
		return
	}
	pix3d.HClip = false
	if var22 < 0 || var24 < 0 || var28 < 0 || var22 > pix2d.SafeWidth || var24 > pix2d.SafeWidth || var28 > pix2d.SafeWidth {
		pix3d.HClip = true
	}
	if TakingInput && w.PointInsideTriangle(MouseX, MouseY, var23, var25, var29, var22, var24, var28) {
		ClickTileX = arg6
		ClickTileZ = arg7
	}
	if arg0.TextureID != -1 {
		if !LowMemory {
			pix3d.TextureTriangle(var23, var25, var29, var22, var24, var28, arg0.SouthwestColor, arg0.SoutheastColor, arg0.NorthwestColor, var32, var14, var31, var40, var18, var21, var36, var34, var39, arg0.TextureID)
			return
		}
		var30 = TEXTURE_HSL[arg0.TextureID]
		pix3d.GouraudTriangle(var23, var25, var29, var22, var24, var28, w.MulLightness(arg0.SouthwestColor, var30), w.MulLightness(arg0.SoutheastColor, var30), w.MulLightness(arg0.NorthwestColor, var30))
	} else if arg0.SouthwestColor != 12345678 {
		pix3d.GouraudTriangle(var23, var25, var29, var22, var24, var28, arg0.SouthwestColor, arg0.SoutheastColor, arg0.NorthwestColor)
	}
}

func (w *World3D) DrawTileOverlay(arg0 int, arg1 int, arg2 *typ.TileOverlay, arg3, arg4, arg5, arg6 int) {
	var9 := len(arg2.VertexX)
	for i := range var9 {
		var11 := arg2.VertexX[i] - EyeX
		var12 := arg2.VertexY[i] - EyeY
		var13 := arg2.VertexZ[i] - EyeZ
		var14 := var13*arg0 + var11*arg6>>16
		var23 := var13*arg6 - var11*arg0>>16
		var25 := var12*arg4 - var23*arg5>>16
		var24 := var12*arg5 + var23*arg4>>16
		if var24 < 50 {
			return
		}
		if arg2.TriangleTextureIDs != nil {
			typ.TmpViewSpaceX[i] = var14
			typ.TmpViewSpaceY[i] = var25
			typ.TmpViewSpaceZ[i] = var24
		}
		typ.TmpScreenX[i] = pix3d.CenterW3D + (var14<<9)/var24
		typ.TmpScreenY[i] = pix3d.CenterH3D + (var25<<9)/var24
	}
	pix3d.Trans = 0
	var9 = len(arg2.TriangleVertexA)
	for i := range var9 {
		var12 := arg2.TriangleVertexA[i]
		var13 := arg2.TriangleVertexB[i]
		var14 := arg2.TriangleVertexC[i]
		var15 := typ.TmpScreenX[var12]
		var16 := typ.TmpScreenX[var13]
		var17 := typ.TmpScreenX[var14]
		var18 := typ.TmpScreenY[var12]
		var19 := typ.TmpScreenY[var13]
		var20 := typ.TmpScreenY[var14]
		if (var15-var16)*(var20-var19)-(var18-var19)*(var17-var16) > 0 {
			pix3d.HClip = false
			if var15 < 0 || var16 < 0 || var17 < 0 || var15 > pix2d.SafeWidth || var16 > pix2d.SafeWidth || var17 > pix2d.SafeWidth {
				pix3d.HClip = true
			}
			if TakingInput && w.PointInsideTriangle(MouseX, MouseY, var18, var19, var20, var15, var16, var17) {
				ClickTileX = arg3
				ClickTileZ = arg1
			}
			if arg2.TriangleTextureIDs == nil || arg2.TriangleTextureIDs[i] == -1 {
				if arg2.TriangleColorA[i] != 12345678 {
					pix3d.GouraudTriangle(var18, var19, var20, var15, var16, var17, arg2.TriangleColorA[i], arg2.TriangleColorB[i], arg2.TriangleColorC[i])
				}
			} else if LowMemory {
				var21 := TEXTURE_HSL[arg2.TriangleTextureIDs[i]]
				pix3d.GouraudTriangle(var18, var19, var20, var15, var16, var17, w.MulLightness(arg2.TriangleColorA[i], var21), w.MulLightness(arg2.TriangleColorB[i], var21), w.MulLightness(arg2.TriangleColorC[i], var21))
			} else if arg2.Flat {
				pix3d.TextureTriangle(var18, var19, var20, var15, var16, var17, arg2.TriangleColorA[i], arg2.TriangleColorB[i], arg2.TriangleColorC[i], typ.TmpViewSpaceX[0], typ.TmpViewSpaceX[1], typ.TmpViewSpaceX[3], typ.TmpViewSpaceY[0], typ.TmpViewSpaceY[1], typ.TmpViewSpaceY[3], typ.TmpViewSpaceZ[0], typ.TmpViewSpaceZ[1], typ.TmpViewSpaceZ[3], arg2.TriangleTextureIDs[i])
			} else {
				pix3d.TextureTriangle(var18, var19, var20, var15, var16, var17, arg2.TriangleColorA[i], arg2.TriangleColorB[i], arg2.TriangleColorC[i], typ.TmpViewSpaceX[var12], typ.TmpViewSpaceX[var13], typ.TmpViewSpaceX[var14], typ.TmpViewSpaceY[var12], typ.TmpViewSpaceY[var13], typ.TmpViewSpaceY[var14], typ.TmpViewSpaceZ[var12], typ.TmpViewSpaceZ[var13], typ.TmpViewSpaceZ[var14], arg2.TriangleTextureIDs[i])
			}
		}
	}
}

func (w *World3D) MulLightness(arg0, arg1 int) int {
	var4 := 127 - arg0
	arg0 = var4 * (arg1 & 0x7F) / 160
	if arg0 < 2 {
		arg0 = 2
	} else if arg0 > 126 {
		arg0 = 126
	}
	return (arg1 & 0xFF80) + arg0
}

func (w *World3D) PointInsideTriangle(arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7 int) bool {
	if arg1 < arg2 && arg1 < arg3 && arg1 < arg4 {
		return false
	}
	if arg1 > arg2 && arg1 > arg3 && arg1 < arg4 {
		return false
	}
	if arg0 < arg5 && arg0 < arg6 && arg0 < arg7 {
		return false
	}
	if arg0 > arg5 && arg0 > arg6 && arg0 > arg7 {
		return false
	}
	var9 := (arg1-arg2)*(arg6-arg5) - (arg0-arg5)*(arg3-arg2)
	var10 := (arg1-arg4)*(arg5-arg7) - (arg0-arg7)*(arg2-arg4)
	var11 := (arg1-arg3)*(arg7-arg6) - (arg0-arg6)*(arg4-arg3)
	return var9*var11 > 0 && var11*var10 > 0
}

func (w *World3D) UpdateActiveOccluders() {
	var2 := LevelOccluderCount[TopLevel]
	var3 := LevelOccluders[TopLevel]
	ActiveOccluderCount = 0
	for i := range var2 {
		var5 := var3[i]
		var6 := 0
		var7 := 0
		var8 := 0
		var10 := 0
		var14 := false
		if var5.Type == 1 {
			var6 = var5.MinTileX - EyeTileX + 25
			if var6 >= 0 && var6 <= 50 {
				var7 = var5.MinTileZ - EyeTileZ + 25
				if var7 < 0 {
					var7 = 0
				}
				var8 = var5.MaxTileZ - EyeTileZ + 25
				if var8 > 50 {
					var8 = 50
				}
				var14 = false
				for var7 <= var8 {
					var7++
					if VisibilityMap[var6][var7-1] {
						var14 = true
						break
					}
				}
				if var14 {
					var10 = EyeX - var5.MinX
					if var10 > 32 {
						var5.Mode = 1
					} else {
						if var10 >= -32 {
							continue
						}
						var5.Mode = 2
						var10 = -var10
					}
					var5.MinDeltaZ = (var5.MinZ - EyeZ<<8) / var10
					var5.MaxDeltaZ = (var5.MaxZ - EyeZ<<8) / var10
					var5.MinDeltaY = (var5.MinY - EyeY<<8) / var10
					var5.MaxDeltaY = (var5.MaxY - EyeY<<8) / var10
					ActiveOccluders[ActiveOccluderCount] = var5
					ActiveOccluderCount++
				}
			}
		} else if var5.Type == 2 {
			var6 = var5.MinTileZ - EyeTileZ + 25
			if var6 >= 0 && var6 <= 50 {
				var7 = var5.MinTileX - EyeTileX + 25
				if var7 < 0 {
					var7 = 0
				}
				var8 = var5.MaxTileX - EyeTileX + 25
				if var8 > 50 {
					var8 = 50
				}
				var14 = false
				for var7 <= var8 {
					var7++
					if VisibilityMap[var7-1][var6] {
						var14 = true
						break
					}
				}
				if var14 {
					var10 = EyeZ - var5.MinZ
					if var10 > 32 {
						var5.Mode = 3
					} else {
						if var10 >= -32 {
							continue
						}
						var5.Mode = 4
						var10 = -var10
					}
					var5.MinDeltaX = (var5.MinX - EyeX<<8) / var10
					var5.MaxDeltaX = (var5.MaxX - EyeX<<8) / var10
					var5.MinDeltaY = (var5.MinY - EyeY<<8) / var10
					var5.MaxDeltaY = (var5.MaxY - EyeY<<8) / var10
					ActiveOccluders[ActiveOccluderCount] = var5
					ActiveOccluderCount++
				}
			}
		} else if var5.Type == 4 {
			var6 = var5.MinY - EyeY
			if var6 > 128 {
				var7 = var5.MinTileZ - EyeTileZ + 25
				if var7 < 0 {
					var7 = 0
				}
				var8 = var5.MaxTileZ - EyeTileZ + 25
				if var8 > 50 {
					var8 = 50
				}
				if var7 <= var8 {
					var9 := var5.MinTileX - EyeTileX + 25
					if var9 < 0 {
						var9 = 0
					}
					var10 = var5.MaxTileX - EyeTileX + 25
					if var10 > 50 {
						var10 = 50
					}
					var11 := false
				label146:
					for j := var9; j <= var10; j++ {
						for k := var7; k <= var8; k++ {
							if VisibilityMap[j][k] {
								var11 = true
								break label146
							}
						}
					}
					if var11 {
						var5.Mode = 5
						var5.MinDeltaX = (var5.MinX - EyeX<<8) / var6
						var5.MaxDeltaX = (var5.MaxX - EyeX<<8) / var6
						var5.MinDeltaZ = (var5.MinZ - EyeZ<<8) / var6
						var5.MaxDeltaZ = (var5.MaxZ - EyeZ<<8) / var6
						ActiveOccluders[ActiveOccluderCount] = var5
						ActiveOccluderCount++
					}
				}
			}
		}
	}
}

func (w *World3D) TileVisible(arg0, arg1, arg2 int) bool {
	var4 := w.LevelTileOcclusionCycles[arg0][arg1][arg2]
	if var4 == -Cycle {
		return false
	}
	if var4 == Cycle {
		return true
	}
	var5 := arg1 << 7
	var6 := arg2 << 7
	if w.Occluded(var5+1, w.LevelHeightMaps[arg0][arg1][arg2], var6+1) &&
		w.Occluded(var5+128-1, w.LevelHeightMaps[arg0][arg1+1][arg2], var6+1) &&
		w.Occluded(var5+128-1, w.LevelHeightMaps[arg0][arg1+1][arg2+1], var6+128-1) &&
		w.Occluded(var5+1, w.LevelHeightMaps[arg0][arg1][arg2+1], var6+128-1) {
		w.LevelTileOcclusionCycles[arg0][arg1][arg2] = Cycle
		return true
	}
	w.LevelTileOcclusionCycles[arg0][arg1][arg2] = -Cycle
	return false
}

func (w *World3D) WallVisible(arg0, arg1, arg2, arg3 int) bool {
	if !w.TileVisible(arg0, arg1, arg2) {
		return false
	}
	var5 := arg1 << 7
	var6 := arg2 << 7
	var7 := w.LevelHeightMaps[arg0][arg1][arg2] - 1
	var8 := var7 - 120
	var9 := var7 - 230
	var10 := var7 - 238
	if arg3 < 16 {
		if arg3 == 1 {
			if var5 > EyeX {
				if !w.Occluded(var5, var7, var6) {
					return false
				}
				if !w.Occluded(var5, var7, var6+128) {
					return false
				}
			}
			if arg0 > 0 {
				if !w.Occluded(var5, var8, var6) {
					return false
				}
				if !w.Occluded(var5, var8, var6+128) {
					return false
				}
			}
			if !w.Occluded(var5, var9, var6) {
				return false
			}
			if !w.Occluded(var5, var9, var6+128) {
				return false
			}
			return true
		}
		if arg3 == 2 {
			if var6 < EyeZ {
				if !w.Occluded(var5, var7, var6+128) {
					return false
				}
				if !w.Occluded(var5+128, var7, var6+128) {
					return false
				}
			}
			if arg0 > 0 {
				if !w.Occluded(var5, var8, var6+128) {
					return false
				}
				if !w.Occluded(var5+128, var8, var6+128) {
					return false
				}
			}
			if !w.Occluded(var5, var9, var6+128) {
				return false
			}
			if !w.Occluded(var5+128, var9, var6+128) {
				return false
			}
			return true
		}
		if arg3 == 4 {
			if var5 < EyeX {
				if !w.Occluded(var5+128, var7, var6) {
					return false
				}
				if !w.Occluded(var5+128, var7, var6+128) {
					return false
				}
			}
			if arg0 > 0 {
				if !w.Occluded(var5+128, var8, var6) {
					return false
				}
				if !w.Occluded(var5+128, var8, var6+128) {
					return false
				}
			}
			if !w.Occluded(var5+128, var9, var6) {
				return false
			}
			if !w.Occluded(var5+128, var9, var6+128) {
				return false
			}
			return true
		}
		if arg3 == 8 {
			if var6 > EyeZ {
				if !w.Occluded(var5, var7, var6) {
					return false
				}
				if !w.Occluded(var5+128, var7, var6) {
					return false
				}
			}
			if arg0 > 0 {
				if !w.Occluded(var5, var8, var6) {
					return false
				}
				if !w.Occluded(var5+128, var8, var6) {
					return false
				}
			}
			if !w.Occluded(var5, var9, var6) {
				return false
			}
			if !w.Occluded(var5+128, var9, var6) {
				return false
			}
			return true
		}
	}
	if !w.Occluded(var5+64, var10, var6+64) {
		return false
	}
	if arg3 == 16 {
		return w.Occluded(var5, var9, var6+128)
	}
	if arg3 == 32 {
		return w.Occluded(var5+128, var9, var6+128)
	}
	if arg3 == 64 {
		return w.Occluded(var5+128, var9, var6)
	}
	if arg3 == 128 {
		return w.Occluded(var5, var9, var6)
	}
	fmt.Println("Warning unsupported wall type")
	return true
}

func (w *World3D) Visible(arg0, arg1, arg2, arg3 int) bool {
	if w.TileVisible(arg0, arg1, arg2) {
		var5 := arg1 << 7
		var6 := arg2 << 7
		return w.Occluded(var5+1, w.LevelHeightMaps[arg0][arg1][arg2]-arg3, var6+1) &&
			w.Occluded(var5+128-1, w.LevelHeightMaps[arg0][arg1+1][arg2]-arg3, var6+1) &&
			w.Occluded(var5+128-1, w.LevelHeightMaps[arg0][arg1+1][arg2+1]-arg3, var6+128-1) &&
			w.Occluded(var5+1, w.LevelHeightMaps[arg0][arg1][arg2+1]-arg3, var6+128-1)
	}
	return false
}

func (w *World3D) LocVisible(arg0, arg1, arg2, arg3, arg4, arg5 int) bool {
	if arg1 != arg2 || arg3 != arg4 {
		for i := arg1; i <= arg2; i++ {
			for j := arg3; j <= arg4; j++ {
				if w.LevelTileOcclusionCycles[arg0][i][j] == -Cycle {
					return false
				}
			}
		}
		var8 := (arg1 << 7) + 1
		var9 := (arg3 << 7) + 2
		var10 := w.LevelHeightMaps[arg0][arg1][arg3] - arg5
		if !w.Occluded(var8, var10, var9) {
			return false
		}
		var11 := (arg2 << 7) - 1
		if !w.Occluded(var11, var10, var9) {
			return false
		}
		var12 := (arg4 << 7) - 1
		if !w.Occluded(var8, var10, var12) {
			return false
		}
		if w.Occluded(var11, var10, var12) {
			return true
		}
		return false
	}
	if w.TileVisible(arg0, arg1, arg3) {
		var7 := arg1 << 7
		var8 := arg3 << 7
		return w.Occluded(var7+1, w.LevelHeightMaps[arg0][arg1][arg3]-arg5, var8+1) &&
			w.Occluded(var7+128-1, w.LevelHeightMaps[arg0][arg1+1][arg3]-arg5, var8+1) &&
			w.Occluded(var7+128-1, w.LevelHeightMaps[arg0][arg1+1][arg3+1]-arg5, var8+128-1) &&
			w.Occluded(var7+1, w.LevelHeightMaps[arg0][arg1][arg3+1]-arg5, var8+128-1)
	}
	return false
}

func (w *World3D) Occluded(arg0, arg1, arg2 int) bool {
	for i := range ActiveOccluderCount {
		var5 := ActiveOccluders[i]
		var6 := 0
		var7 := 0
		var8 := 0
		var9 := 0
		var10 := 0
		switch var5.Mode {
		case 1:
			var6 = var5.MinX - arg0
			if var6 > 0 {
				var7 = var5.MinZ + (var5.MinDeltaZ * var6 >> 8)
				var8 = var5.MaxZ + (var5.MaxDeltaZ * var6 >> 8)
				var9 = var5.MinY + (var5.MinDeltaY * var6 >> 8)
				var10 = var5.MaxY + (var5.MaxDeltaY * var6 >> 8)
				if arg2 >= var7 && arg2 <= var8 && arg1 >= var9 && arg1 <= var10 {
					return true
				}
			}
		case 2:
			var6 = arg0 - var5.MinX
			if var6 > 0 {
				var7 = var5.MinZ + (var5.MinDeltaZ * var6 >> 8)
				var8 = var5.MaxZ + (var5.MaxDeltaZ * var6 >> 8)
				var9 = var5.MinY + (var5.MinDeltaY * var6 >> 8)
				var10 = var5.MaxY + (var5.MaxDeltaY * var6 >> 8)
				if arg2 >= var7 && arg2 <= var8 && arg1 >= var9 && arg1 <= var10 {
					return true
				}
			}
		case 3:
			var6 = var5.MinZ - arg2
			if var6 > 0 {
				var7 = var5.MinX + (var5.MinDeltaX * var6 >> 8)
				var8 = var5.MaxX + (var5.MaxDeltaX * var6 >> 8)
				var9 = var5.MinY + (var5.MinDeltaY * var6 >> 8)
				var10 = var5.MaxY + (var5.MaxDeltaY * var6 >> 8)
				if arg0 >= var7 && arg0 <= var8 && arg1 >= var9 && arg1 <= var10 {
					return true
				}
			}
		case 4:
			var6 = arg2 - var5.MinZ
			if var6 > 0 {
				var7 = var5.MinX + (var5.MinDeltaX * var6 >> 8)
				var8 = var5.MaxX + (var5.MaxDeltaX * var6 >> 8)
				var9 = var5.MinY + (var5.MinDeltaY * var6 >> 8)
				var10 = var5.MaxY + (var5.MaxDeltaY * var6 >> 8)
				if arg0 >= var7 && arg0 <= var8 && arg1 >= var9 && arg1 <= var10 {
					return true
				}
			}
		case 5:
			var6 = arg1 - var5.MinY
			if var6 > 0 {
				var7 = var5.MinX + (var5.MinDeltaX * var6 >> 8)
				var8 = var5.MaxX + (var5.MaxDeltaX * var6 >> 8)
				var9 = var5.MinZ + (var5.MinDeltaX * var6 >> 8)
				var10 = var5.MaxZ + (var5.MaxDeltaZ * var6 >> 8)
				if arg0 >= var7 && arg0 <= var8 && arg2 >= var9 && arg2 <= var10 {
					return true
				}
			}
		}
	}
	return false
}
