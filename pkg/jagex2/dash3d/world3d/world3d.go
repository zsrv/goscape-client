package world3d

import (
	"math"

	"goscape-client/pkg/jagex2/dash3d"
	"goscape-client/pkg/jagex2/dash3d/entity"
	"goscape-client/pkg/jagex2/dash3d/typ"
	"goscape-client/pkg/jagex2/datastruct"
	"goscape-client/pkg/jagex2/graphics/model"
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

func (w *World3D) AddOccluder(arg0, arg1, arg3, arg4, arg5, arg6, arg7, arg8 int) {
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
}
