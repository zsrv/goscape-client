package world

import (
	"fmt"
	"math"

	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/entity"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/model"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/typ"
	"github.com/zsrv/goscape-client/pkg/jagex2/datastruct"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pix2d"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pix3d"
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
	LocBuffer                            []*typ.Sprite = make([]*typ.Sprite, 100)
	WALL_DECORATION_INSET_X                            = []int{53, -53, -53, 53}
	WALL_DECORATION_INSET_Z                            = []int{-53, -53, 53, 53}
	WALL_DECORATION_OUTSET_X                           = []int{-45, 45, 45, -45}
	WALL_DECORATION_OUTSET_Z                           = []int{45, 45, -45, -45}
	ClickTileX                           int           = -1
	ClickTileZ                           int           = -1
	LEVEL_COUNT                          int           = 4
	LevelOccluderCount                   []int         = make([]int, LEVEL_COUNT)
	LevelOccluders                       [][]*dash3d.Occlude
	ActiveOccluders                      []*dash3d.Occlude = make([]*dash3d.Occlude, 500)
	DrawTileQueue                                          = datastruct.NewLinkList[*typ.Square]()
	FRONT_WALL_TYPES                                       = []int{19, 55, 38, 155, 0xFF, 110, 137, 205, 76}
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

type World struct {
	MaxLevel                 int
	MaxTileX                 int
	MaxTileZ                 int
	LevelHeightMaps          [][][]int
	LevelTiles               [][][]*typ.Square
	Minlevel                 int
	TemporaryLocCount        int
	TemporaryLocs            []*typ.Sprite
	LevelTileOcclusionCycles [][][]int
	MergeIndexA              []int
	MergeIndexB              []int
	TmpMergeIndex            int
	MINIMAP_OVERLAY_SHAPE    [][]int
	MINIMAP_OVERLAY_ROTATION [][]int
}

func NewWorld(LevelHeightMaps [][][]int, maxTileZ, maxLevel, maxTileX int) *World {
	var w World
	w.TemporaryLocs = make([]*typ.Sprite, 5000)
	w.MergeIndexA = make([]int, 10000)
	w.MergeIndexB = make([]int, 10000)
	w.MINIMAP_OVERLAY_SHAPE = [][]int{make([]int, 16), {1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}, {1, 0, 0, 0, 1, 1, 0, 0, 1, 1, 1, 0, 1, 1, 1, 1}, {1, 1, 0, 0, 1, 1, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0}, {0, 0, 1, 1, 0, 0, 1, 1, 0, 0, 0, 1, 0, 0, 0, 1}, {0, 1, 1, 1, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}, {1, 1, 1, 0, 1, 1, 1, 0, 1, 1, 1, 1, 1, 1, 1, 1}, {1, 1, 0, 0, 1, 1, 0, 0, 1, 1, 0, 0, 1, 1, 0, 0}, {0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 1, 1, 0, 0}, {1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 1, 1, 0, 0, 1, 1}, {1, 1, 1, 1, 1, 1, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0}, {0, 0, 0, 0, 0, 0, 1, 1, 0, 1, 1, 1, 0, 1, 1, 1}, {0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 0, 1, 1, 1, 1}}
	w.MINIMAP_OVERLAY_ROTATION = [][]int{{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}, {12, 8, 4, 0, 13, 9, 5, 1, 14, 10, 6, 2, 15, 11, 7, 3}, {15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}, {3, 7, 11, 15, 2, 6, 10, 14, 1, 5, 9, 13, 0, 4, 8, 12}}

	w.MaxLevel = maxLevel
	w.MaxTileX = maxTileX
	w.MaxTileZ = maxTileZ
	w.LevelTiles = make([][][]*typ.Square, maxLevel)
	for i := range w.LevelTiles {
		w.LevelTiles[i] = make([][]*typ.Square, maxTileX)
		for j := range w.LevelTiles[i] {
			w.LevelTiles[i][j] = make([]*typ.Square, maxTileZ)
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

func (w *World) Reset() {
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

func (w *World) SetMinLevel(level int) {
	w.Minlevel = level
	for i := range w.MaxTileX {
		for j := range w.MaxTileZ {
			w.LevelTiles[level][i][j] = typ.NewSquare(level, i, j)
		}
	}
}

func (w *World) SetBridge(arg0, arg1 int) {
	var4 := w.LevelTiles[0][arg1][arg0]
	for i := range 3 {
		w.LevelTiles[i][arg1][arg0] = w.LevelTiles[i+1][arg1][arg0]
		if w.LevelTiles[i][arg1][arg0] != nil {
			w.LevelTiles[i][arg1][arg0].Level--
		}
	}
	if w.LevelTiles[0][arg1][arg0] == nil {
		w.LevelTiles[0][arg1][arg0] = typ.NewSquare(0, arg1, arg0)
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

func (w *World) SetDrawLevel(arg0, arg1, arg2, arg3 int) {
	var5 := w.LevelTiles[arg0][arg1][arg2]
	if var5 != nil {
		w.LevelTiles[arg0][arg1][arg2].DrawLevel = arg3
	}
}

func (w *World) SetTile(arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7, arg8, arg9, arg10, arg11, arg12, arg13, arg14, arg15, arg16, arg17, arg18, arg19 int) {
	switch arg3 {
	case 0:
		var21 := typ.NewQuickGround(arg10, arg11, arg12, arg13, -1, arg18, false)
		for i := arg0; i >= 0; i-- {
			if w.LevelTiles[i][arg1][arg2] == nil {
				w.LevelTiles[i][arg1][arg2] = typ.NewSquare(i, arg1, arg2)
			}
		}
		w.LevelTiles[arg0][arg1][arg2].Underlay = var21
	case 1:
		var21 := typ.NewQuickGround(arg14, arg15, arg16, arg17, arg5, arg19, arg6 == arg7 && arg6 == arg8 && arg6 == arg9)
		for i := arg0; i >= 0; i-- {
			if w.LevelTiles[i][arg1][arg2] == nil {
				w.LevelTiles[i][arg1][arg2] = typ.NewSquare(i, arg1, arg2)
			}
		}
		w.LevelTiles[arg0][arg1][arg2].Underlay = var21
	default:
		var23 := typ.NewGround(arg1, arg3, arg15, arg7, arg12, arg4, arg10, arg9, arg19, arg14, arg5, arg17, arg18, arg8, arg16, arg13, arg6, arg2, arg11)
		for i := arg0; i >= 0; i-- {
			if w.LevelTiles[i][arg1][arg2] == nil {
				w.LevelTiles[i][arg1][arg2] = typ.NewSquare(i, arg1, arg2)
			}
		}
		w.LevelTiles[arg0][arg1][arg2].Overlay = var23
	}
}

func (w *World) AddGroundDecoration(arg0 entity.ModelSource, arg2, arg3, arg4, arg5 int, arg6 byte, arg7 int) {
	if arg0 == nil {
		return
	}
	var9 := typ.NewGroundDecor()
	var9.Model = arg0
	var9.X = arg2*128 + 64
	var9.Z = arg4*128 + 64
	var9.Y = arg7
	var9.BitSet = arg3
	var9.Info = int8(arg6) // Java: byte field; (byte) reinterpret of the typecode
	if w.LevelTiles[arg5][arg2][arg4] == nil {
		w.LevelTiles[arg5][arg2][arg4] = typ.NewSquare(arg5, arg2, arg4)
	}
	w.LevelTiles[arg5][arg2][arg4].GroundDecor = var9
}

func (w *World) AddObjStack(arg0, arg1 entity.ModelSource, arg2, arg3, arg4, arg5, arg6 int, arg7 entity.ModelSource) {
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
			// Java: rev-244 addGroundObject — only static loc models (instanceof
			// Model) contribute objRaise; self-animating ClientLocAnim locs are
			// skipped by the type assertion.
			if m, ok := var12.Locs[i].Model.(*model.Model); ok {
				var14 := m.ObjRaise
				var11 = max(var14, var11)
			}
		}
	}
	var10.Offset = var11
	if w.LevelTiles[arg3][arg6][arg5] == nil {
		w.LevelTiles[arg3][arg6][arg5] = typ.NewSquare(arg3, arg6, arg5)
	}
	w.LevelTiles[arg3][arg6][arg5].GroundObj = var10
}

func (w *World) AddWall(angle2, y, level, angle1 int, model1 entity.ModelSource, model2 entity.ModelSource, tileX, typecode1, tileZ int, typecode2 byte) {
	if model1 == nil && model2 == nil {
		return
	}

	wall := typ.NewWall()
	wall.BitSet = typecode1
	wall.Info = int8(typecode2)
	wall.X = tileX*128 + 64
	wall.Z = tileZ*128 + 64
	wall.Y = y
	wall.ModelA = model1
	wall.ModelB = model2
	wall.TypeA = angle1
	wall.TypeB = angle2

	for i := level; i >= 0; i-- {
		if w.LevelTiles[i][tileX][tileZ] == nil {
			w.LevelTiles[i][tileX][tileZ] = typ.NewSquare(i, tileX, tileZ)
		}
	}

	w.LevelTiles[level][tileX][tileZ].Wall = wall
}

// AddDecor
func (w *World) SetWallDecoration(y, z, zOffset, typecode, angle2, angle1, xOffset, x int, modelSrc entity.ModelSource, typecode2 byte, level int) {
	if modelSrc == nil {
		return
	}

	decor := typ.NewDecor()
	decor.BitSet = typecode
	decor.Info = int8(typecode2)
	decor.X = x*128 + 64 + xOffset
	decor.Z = z*128 + 64 + zOffset
	decor.Y = y
	decor.Model = modelSrc
	// Seed the cached MinY from a static model immediately (rev-244 reads
	// ModelSource.minY for the decor visibility cull). Animated ClientLocAnim
	// nodes keep the ctor default 1000 until first drawn.
	if m, ok := modelSrc.(*model.Model); ok {
		decor.MinY = m.MaxY // Java: model.minY == Go Model.MaxY (lineage inversion; height above origin)
	}
	decor.Type = angle1
	decor.Angle = angle2
	for l := level; l >= 0; l-- {
		if w.LevelTiles[l][x][z] == nil {
			w.LevelTiles[l][x][z] = typ.NewSquare(l, x, z)
		}
	}
	w.LevelTiles[level][x][z].Decor = decor
}

func (w *World) AddLoc1(arg0 int, arg2 int, arg4, arg5, arg6, arg7 int, arg8 byte, arg9 entity.ModelSource, arg10, arg11 int) bool {
	if arg9 == nil {
		return true
	}
	var13 := arg6*128 + arg7*64
	var14 := arg5*128 + arg11*64
	return w.AddLoc2(arg2, arg6, arg5, arg7, arg11, var13, var14, arg0, arg9, arg10, false, arg4, arg8)
}

func (w *World) AddTemporary1(arg1, arg2, yaw, arg4, arg5 int, forwardPadding bool, arg8 entity.ModelSource, arg9, arg10 int) bool {
	if arg8 == nil {
		return true
	}

	x0 := arg4 - arg2
	z0 := arg1 - arg2
	x1 := arg4 + arg2
	z1 := arg1 + arg2

	if forwardPadding {
		if yaw > 640 && yaw < 1408 {
			z1 += 128
		}

		if yaw > 1152 && yaw < 1920 {
			x1 += 128
		}

		if yaw > 1664 || yaw < 384 {
			z0 -= 128
		}

		if yaw > 128 && yaw < 896 {
			x0 -= 128
		}
	}

	x0 /= 128
	z0 /= 128
	x1 /= 128
	z1 /= 128

	return w.AddLoc2(arg10, x0, z0, x1-x0+1, z1-z0+1, arg4, arg1, arg9, arg8, yaw, true, arg5, byte(0))
}

func (w *World) AddTemporary2(arg0 int, arg3, arg4, arg5, arg6, arg7, arg8 int, arg9 entity.ModelSource, arg11, arg12, arg13 int) bool {
	if arg9 == nil {
		return true
	}
	return w.AddLoc2(arg11, arg8, arg7, arg0-arg8+1, arg12-arg7+1, arg13, arg3, arg4, arg9, arg6, true, arg5, byte(0))
}

func (w *World) AddLoc2(arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7 int, arg8 entity.ModelSource, arg10 int, arg11 bool, arg12 int, arg13 byte) bool {
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
	var22 := typ.NewSprite()
	var22.BitSet = arg12
	var22.Info = int8(arg13)
	var22.Level = arg0
	var22.X = arg5
	var22.Z = arg6
	var22.Y = arg7
	var22.Model = arg8
	// Seed the cached MinY from a static model immediately (rev-244 reads
	// ModelSource.minY for the loc visibility cull). Animated ClientLocAnim
	// nodes keep the ctor default 1000 until first drawn.
	if m, ok := arg8.(*model.Model); ok {
		var22.MinY = m.MaxY // Java: model.minY == Go Model.MaxY (lineage inversion; height above origin)
	}
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
					w.LevelTiles[k][i][j] = typ.NewSquare(k, i, j)
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

// ClearLocChanges
func (w *World) ClearTemporaryLocs() {
	for i := range w.TemporaryLocCount {
		loc := w.TemporaryLocs[i]
		w.RemoveLoc1(loc)
		w.TemporaryLocs[i] = nil
	}

	w.TemporaryLocCount = 0
}

func (w *World) RemoveLoc1(arg0 *typ.Sprite) {
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
				for k := range var5.LocCount {
					var5.LocSpans |= var5.LocSpan[k]
				}
			}
		}
	}
}

// Java: setDecorOffset (World.java:569 @176a85f)
func (w *World) SetWallDecorationOffset(arg0, arg1, arg2, arg3, arg4 int) {
	var6 := w.LevelTiles[arg1][arg4][arg0]
	if var6 == nil {
		return
	}
	var7 := var6.Decor
	if var7 != nil {
		var8 := arg4*128 + 64
		var9 := arg0*128 + 64
		var7.X = var8 + (var7.X-var8)*arg2/16
		if arg3 == -23232 {
			var7.Z = var9 + (var7.Z-var9)*arg2/16
		}
	}
}

func (w *World) RemoveWall(arg0, arg1, arg2 int) {
	var5 := w.LevelTiles[arg1][arg0][arg2]
	if var5 != nil {
		var5.Wall = nil
	}
}

func (w *World) RemoveWallDecoration(arg0, arg1, arg3 int) {
	var5 := w.LevelTiles[arg0][arg3][arg1]
	if var5 != nil {
		var5.Decor = nil
	}
}

func (w *World) RemoveLoc2(arg0, arg1, arg3 int) {
	var5 := w.LevelTiles[arg3][arg0][arg1]
	if var5 == nil {
		return
	}
	for i := range var5.LocCount {
		var7 := var5.Locs[i]
		if (var7.BitSet>>29)&0x3 == 2 && var7.MinSceneTileX == arg0 && var7.MinSceneTileZ == arg1 {
			w.RemoveLoc1(var7)
			return
		}
	}
}

func (w *World) RemoveGroundDecoration(arg0, arg2, arg3 int) {
	var5 := w.LevelTiles[arg0][arg2][arg3]
	if var5 != nil {
		var5.GroundDecor = nil
	}
}

func (w *World) RemoveObjStack(arg0, arg1, arg2 int) {
	var4 := w.LevelTiles[arg0][arg1][arg2]
	if var4 != nil {
		var4.GroundObj = nil
	}
}

// GetWall returns the wall node at (arg2=level, arg0, arg1), or nil. Java:
// rev-244 World.getWall — used by the dynamic loc-anim / loc-change apply path
// to retarget a node's ModelSource.
func (w *World) GetWall(arg0, arg1, arg2 int) *typ.Wall {
	var6 := w.LevelTiles[arg2][arg0][arg1]
	if var6 == nil {
		return nil
	}
	return var6.Wall
}

// GetDecor returns the decor node at (arg1=level, arg0, arg3), or nil. Java:
// rev-244 World.getDecor.
func (w *World) GetDecor(arg0, arg1, arg3 int) *typ.Decor {
	var5 := w.LevelTiles[arg1][arg0][arg3]
	if var5 == nil {
		return nil
	}
	return var5.Decor
}

// GetSprite returns the layer-2 loc sprite at (arg0=level, arg3, arg1), or nil.
// Java: rev-244 World.getSprite.
func (w *World) GetSprite(arg0, arg1, arg3 int) *typ.Sprite {
	var5 := w.LevelTiles[arg0][arg3][arg1]
	if var5 == nil {
		return nil
	}
	for i := range var5.LocCount {
		var7 := var5.Locs[i]
		if (var7.BitSet>>29)&0x3 == 2 && var7.MinSceneTileX == arg3 && var7.MinSceneTileZ == arg1 {
			return var7
		}
	}
	return nil
}

// GetGroundDecor returns the ground-decor node at (arg3=level, arg1, arg2), or
// nil. Java: rev-244 World.getGroundDecor.
func (w *World) GetGroundDecor(arg1, arg2, arg3 int) *typ.GroundDecor {
	var5 := w.LevelTiles[arg3][arg1][arg2]
	if var5 == nil || var5.GroundDecor == nil {
		return nil
	}
	return var5.GroundDecor
}

func (w *World) GetWallBitSet(arg0, arg1, arg2 int) int {
	var4 := w.LevelTiles[arg0][arg1][arg2]
	if var4 == nil || var4.Wall == nil {
		return 0
	}
	return var4.Wall.BitSet
}

func (w *World) GetWallDecorationBitSet(arg0, arg1, arg3 int) int {
	var5 := w.LevelTiles[arg0][arg3][arg1]
	if var5 == nil || var5.Decor == nil {
		return 0
	}
	return var5.Decor.BitSet
}

func (w *World) GetLocBitSet(arg0, arg1, arg2 int) int {
	var4 := w.LevelTiles[arg0][arg1][arg2]
	if var4 == nil {
		return 0
	}
	for i := range var4.LocCount {
		var6 := var4.Locs[i]
		if (var6.BitSet>>29)&0x3 == 2 && var6.MinSceneTileX == arg1 && var6.MinSceneTileZ == arg2 {
			return var6.BitSet
		}
	}
	return 0
}

func (w *World) GetGroundDecorationBitSet(arg0, arg1, arg2 int) int {
	var4 := w.LevelTiles[arg0][arg1][arg2]
	if var4 == nil || var4.GroundDecor == nil {
		return 0
	}
	return var4.GroundDecor.BitSet
}

func (w *World) GetInfo(arg0, arg1, arg2, arg3 int) int {
	var5 := w.LevelTiles[arg0][arg1][arg2]
	if var5 == nil {
		return -1
	}
	if var5.Wall != nil && var5.Wall.BitSet == arg3 {
		return int(var5.Wall.Info) & 0xFF
	}
	if var5.Decor != nil && var5.Decor.BitSet == arg3 {
		return int(var5.Decor.Info) & 0xFF
	}
	if var5.GroundDecor != nil && var5.GroundDecor.BitSet == arg3 {
		return int(var5.GroundDecor.Info) & 0xFF
	}
	for i := range var5.LocCount {
		if var5.Locs[i].BitSet == arg3 {
			return int(var5.Locs[i].Info) & 0xFF
		}
	}
	return -1
}

func (w *World) BuildModels(arg0, arg1, arg2, lightAttenuation, arg4 int) {
	lightMagnitude := int(math.Sqrt(float64(arg2*arg2 + arg0*arg0 + arg4*arg4)))
	attenuation := (lightAttenuation * lightMagnitude) >> 8

	for level := range w.MaxLevel {
		for tileX := range w.MaxTileX {
			for tileZ := range w.MaxTileZ {
				tile := w.LevelTiles[level][tileX][tileZ]
				if tile != nil {
					// Java: rev-244 reads vertexNormal off the ModelSource base
					// field then casts (Model); only static models populate
					// vertexNormal, so the (*model.Model) type assertion both
					// gates on "is a static Model" and skips self-animating
					// ClientLocAnim locs (instanceof Model).
					var13 := tile.Wall
					if var13 != nil {
						if mA, ok := var13.ModelA.(*model.Model); ok && mA.VertexNormal != nil {
							w.MergeLocNormals(tileX, 1, 1, level, mA, tileZ)
							if mB, ok := var13.ModelB.(*model.Model); ok && mB.VertexNormal != nil {
								w.MergeLocNormals(tileX, 1, 1, level, mB, tileZ)
								w.MergeNormals(mA, mB, 0, 0, 0, false)
								mB.ApplyLighting(arg1, attenuation, arg2, arg0, arg4)
							}
							mA.ApplyLighting(arg1, attenuation, arg2, arg0, arg4)
						}
					}

					for l := range tile.LocCount {
						loc := tile.Locs[l]
						if loc != nil {
							if m, ok := loc.Model.(*model.Model); ok && m.VertexNormal != nil {
								w.MergeLocNormals(tileX, loc.MaxSceneTileX-loc.MinSceneTileX+1, loc.MaxSceneTileZ-loc.MinSceneTileZ+1, level, m, tileZ)
								m.ApplyLighting(arg1, attenuation, arg2, arg0, arg4)
							}
						}
					}

					decor := tile.GroundDecor
					if decor != nil {
						if m, ok := decor.Model.(*model.Model); ok && m.VertexNormal != nil {
							w.MergeGroundDecorationNormals(level, tileZ, m, tileX)
							m.ApplyLighting(arg1, attenuation, arg2, arg0, arg4)
						}
					}
				}
			}
		}
	}
}

func (w *World) MergeGroundDecorationNormals(arg1 int, arg2 int, arg3 *model.Model, arg4 int) {
	// Java: rev-244 reads groundDecor.model.vertexNormal off the ModelSource
	// then casts (Model); the (*model.Model) assertion below gates on a static
	// model and skips self-animating ClientLocAnim ground decor.
	if arg4 < w.MaxTileX {
		var6 := w.LevelTiles[arg1][arg4+1][arg2]
		if var6 != nil && var6.GroundDecor != nil {
			if m, ok := var6.GroundDecor.Model.(*model.Model); ok && m.VertexNormal != nil {
				w.MergeNormals(arg3, m, 128, 0, 0, true)
			}
		}
	}
	if arg2 < w.MaxTileX {
		var6 := w.LevelTiles[arg1][arg4][arg2+1]
		if var6 != nil && var6.GroundDecor != nil {
			if m, ok := var6.GroundDecor.Model.(*model.Model); ok && m.VertexNormal != nil {
				w.MergeNormals(arg3, m, 0, 0, 128, true)
			}
		}
	}
	if arg4 < w.MaxTileX && arg2 < w.MaxTileZ {
		var6 := w.LevelTiles[arg1][arg4+1][arg2+1]
		if var6 != nil && var6.GroundDecor != nil {
			if m, ok := var6.GroundDecor.Model.(*model.Model); ok && m.VertexNormal != nil {
				w.MergeNormals(arg3, m, 128, 0, 128, true)
			}
		}
	}
	if arg4 >= w.MaxTileX || arg2 <= 0 {
		return
	}
	var6 := w.LevelTiles[arg1][arg4+1][arg2-1]
	if var6 != nil && var6.GroundDecor != nil {
		if m, ok := var6.GroundDecor.Model.(*model.Model); ok && m.VertexNormal != nil {
			w.MergeNormals(arg3, m, 128, 0, -128, true)
		}
	}
}

func (w *World) MergeLocNormals(arg0, arg1, arg2, arg3 int, arg5 *model.Model, arg6 int) {
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
								if var18 != nil {
									if m, ok := var18.ModelA.(*model.Model); ok && m.VertexNormal != nil {
										w.MergeNormals(arg5, m, (j-arg0)*128+(1-arg1)*64, var17, (k-arg6)*128+(1-arg2)*64, var8)
									}
									if m, ok := var18.ModelB.(*model.Model); ok && m.VertexNormal != nil {
										w.MergeNormals(arg5, m, (j-arg0)*128+(1-arg1)*64, var17, (k-arg6)*128+(1-arg2)*64, var8)
									}
								}
								for l := range var16.LocCount {
									var20 := var16.Locs[l]
									if var20 != nil {
										if m, ok := var20.Model.(*model.Model); ok && m.VertexNormal != nil {
											var21 := var20.MaxSceneTileX - var20.MinSceneTileX + 1
											var22 := var20.MaxSceneTileZ - var20.MinSceneTileZ + 1
											w.MergeNormals(arg5, m, (var20.MinSceneTileX-arg0)*128+(var21-arg1)*64, var17, (var20.MinSceneTileZ-arg6)*128+(var22-arg2)*64, var8)
										}
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

func (w *World) MergeNormals(modelA, modelB *model.Model, arg2, offsetY, arg4 int, arg5 bool) {
	w.TmpMergeIndex++

	merged := 0
	vertexX := modelB.VertexX
	vertexCountB := modelB.VertexCount

	for vertexA := range modelA.VertexCount {
		originalNormalA := modelA.VertexNormalOriginal[vertexA]

		if originalNormalA.W != 0 {
			y := modelA.VertexY[vertexA] - offsetY
			if y <= modelB.MinY {
				x := modelA.VertexX[vertexA] - arg2
				if x >= modelB.MinX && x <= modelB.MaxX {
					z := modelA.VertexZ[vertexA] - arg4
					if z >= modelB.MinZ && z <= modelB.MaxZ {
						for j := range vertexCountB {
							var18 := modelB.VertexNormalOriginal[j]
							if x == vertexX[j] && z == modelB.VertexZ[j] && y == modelB.VertexY[j] && var18.W != 0 {
								modelA.VertexNormal[vertexA].X += var18.X
								modelA.VertexNormal[vertexA].Y += var18.Y
								modelA.VertexNormal[vertexA].Z += var18.Z
								modelA.VertexNormal[vertexA].W += var18.W
								modelB.VertexNormal[j].X += originalNormalA.X
								modelB.VertexNormal[j].Y += originalNormalA.Y
								modelB.VertexNormal[j].Z += originalNormalA.Z
								modelB.VertexNormal[j].W += originalNormalA.W
								merged++
								w.MergeIndexA[vertexA] = w.TmpMergeIndex
								w.MergeIndexB[j] = w.TmpMergeIndex
							}
						}
					}
				}
			}
		}
	}
	if merged < 3 || !arg5 {
		return
	}
	for i := range modelA.FaceCount {
		if w.MergeIndexA[modelA.FaceVertexA[i]] == w.TmpMergeIndex && w.MergeIndexA[modelA.FaceVertexB[i]] == w.TmpMergeIndex && w.MergeIndexA[modelA.FaceVertexC[i]] == w.TmpMergeIndex {
			modelA.FaceInfo[i] = -1
		}
	}
	for i := range modelB.FaceCount {
		if w.MergeIndexB[modelB.FaceVertexA[i]] == w.TmpMergeIndex && w.MergeIndexB[modelB.FaceVertexB[i]] == w.TmpMergeIndex && w.MergeIndexB[modelB.FaceVertexC[i]] == w.TmpMergeIndex {
			modelB.FaceInfo[i] = -1
		}
	}
}

func (w *World) DrawMinimapTile(arg0 []int, arg1, arg2, arg3, arg4, arg5 int) {
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

func Init(arg0 []int, arg1, arg2, arg4, arg5 int) {
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
	var4 := (arg1*SinEyeYaw + arg0*CosEyeYaw) >> 16
	var5 := (arg1*CosEyeYaw - arg0*SinEyeYaw) >> 16
	var6 := (arg2*SinEyePitch + var5*CosEyePitch) >> 16
	var7 := (arg2*CosEyePitch - var5*SinEyePitch) >> 16
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

func (w *World) Click(mouseY, mouseX int) {
	TakingInput = true
	MouseX = mouseX
	MouseY = mouseY
	ClickTileX = -1
	ClickTileZ = -1
}

func (w *World) Draw(arg0, arg1, arg2, arg3, arg4, arg5 int) {
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
	MinDrawTileX = max(MinDrawTileX, 0)
	MinDrawTileZ = EyeTileZ - 25
	MinDrawTileZ = max(MinDrawTileZ, 0)
	MaxDrawTileX = EyeTileX + 25
	MaxDrawTileX = min(MaxDrawTileX, w.MaxTileX)
	MaxDrawTileZ = EyeTileZ + 25
	MaxDrawTileZ = min(MaxDrawTileZ, w.MaxTileZ)
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
					var var17 *typ.Square
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
					var var18 *typ.Square
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

func (w *World) DrawTile(next *typ.Square, checkAdjacent bool) {
	DrawTileQueue.AddTail(next.DrawQueueNode)

	for {
		var tile *typ.Square
		tileX := 0
		tileZ := 0
		level := 0
		originalLevel := 0
		var tiles [][]*typ.Square
		var var9 *typ.Square

		for ok1 := true; ok1; ok1 = var9 != nil && var9.Update {
			for ok2 := true; ok2; ok2 = var9 != nil && var9.Update {
				for ok3 := true; ok3; ok3 = var9 != nil && var9.Update {
					for ok4 := true; ok4; ok4 = var9 != nil && var9.Update {
						for ok5 := true; ok5; ok5 = tile.CheckLocSpans != 0 {
							for ok6 := true; ok6; ok6 = !tile.Update {
								for {
									var var12 *typ.Sprite
									for {
										for ok7 := true; ok7; ok7 = !tile.Update {
											// Java: `Square extends Linkable`, so `removeHead()`
											// returns a Square (or null on empty queue). In Go the
											// Linkable wraps the Square, so the nil check belongs on
											// the *Linkable — `.Value` on a nil Linkable would panic
											// before the existing tile-nil check below could fire.
											linkable := DrawTileQueue.RemoveHead()
											if linkable == nil {
												return
											}
											tile = linkable.Value
										}

										tileX = tile.X
										tileZ = tile.Z
										level = tile.Level
										originalLevel = tile.OccludeLevel
										tiles = w.LevelTiles[level]
										if !tile.Visible {

											break
										}

										if checkAdjacent {
											if level > 0 {
												above := w.LevelTiles[level-1][tileX][tileZ]
												if above != nil && above.Update {
													continue
												}
											}

											if tileX <= EyeTileX && tileX > MinDrawTileX {
												t := tiles[tileX-1][tileZ]
												if t != nil && t.Update && (t.Visible || (tile.LocSpans&0x1) == 0) {
													continue
												}
											}
											if tileX >= EyeTileX && tileX < MaxDrawTileX-1 {
												t := tiles[tileX+1][tileZ]
												if t != nil && t.Update && (t.Visible || (tile.LocSpans&0x4) == 0) {
													continue
												}
											}
											if tileZ <= EyeTileZ && tileZ > MinDrawTileZ {
												t := tiles[tileX][tileZ-1]
												if t != nil && t.Update && (t.Visible || (tile.LocSpans&0x8) == 0) {
													continue
												}
											}
											if tileZ >= EyeTileZ && tileZ < MaxDrawTileZ-1 {
												adjacent := tiles[tileX][tileZ+1]
												if adjacent != nil && adjacent.Update && (adjacent.Visible || (tile.LocSpans&0x2) == 0) {
													continue
												}
											}
										} else {
											checkAdjacent = true
										}

										tile.Visible = false

										if tile.Bridge != nil {
											var9 = tile.Bridge
											if var9.Underlay == nil {
												if var9.Overlay != nil && !w.TileVisible(0, tileX, tileZ) {
													w.DrawTileOverlay(SinEyeYaw, tileZ, var9.Overlay, tileX, CosEyePitch, SinEyePitch, CosEyeYaw)
												}
											} else if !w.TileVisible(0, tileX, tileZ) {
												w.DrawTileUnderlay(var9.Underlay, 0, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, tileX, tileZ)
											}
											var10 := var9.Wall
											if var10 != nil {
												if m := var10.ModelA.GetModel(); m != nil {
													m.Draw1(0, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, var10.X-EyeX, var10.Y-EyeY, var10.Z-EyeZ, var10.BitSet)
												}
											}
											for i := range var9.LocCount {
												var12 = var9.Locs[i]
												if var12 != nil {
													if m := var12.Model.GetModel(); m != nil {
														var12.MinY = m.MaxY // Java: model.minY == Go Model.MaxY (lineage inversion; height above origin)
														m.Draw1(var12.Yaw, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, var12.X-EyeX, var12.Y-EyeY, var12.Z-EyeZ, var12.BitSet)
													}
												}
											}
										}
										tileDrawn := false
										if tile.Underlay == nil {
											if tile.Overlay != nil && !w.TileVisible(originalLevel, tileX, tileZ) {
												tileDrawn = true
												w.DrawTileOverlay(SinEyeYaw, tileZ, tile.Overlay, tileX, CosEyePitch, SinEyePitch, CosEyeYaw)
											}
										} else if !w.TileVisible(originalLevel, tileX, tileZ) {
											tileDrawn = true
											w.DrawTileUnderlay(tile.Underlay, originalLevel, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, tileX, tileZ)
										}

										direction := 0
										frontWallTypes := 0

										wall := tile.Wall
										decor := tile.Decor

										if wall != nil || decor != nil {
											if EyeTileX == tileX {
												direction++
											} else if EyeTileX < tileX {
												direction += 2
											}

											if EyeTileZ == tileZ {
												direction += 3
											} else if EyeTileZ > tileZ {
												direction += 6
											}

											frontWallTypes = FRONT_WALL_TYPES[direction]
											tile.BackWallTypes = BACK_WALL_TYPES[direction]
										}

										if wall != nil {
											if wall.TypeA&DIRECTION_ALLOW_WALL_CORNER_TYPE[direction] == 0 {
												tile.CheckLocSpans = 0
											} else if wall.TypeA == 16 {
												tile.CheckLocSpans = 3
												tile.BlockLocSpans = WALL_CORNER_TYPE_16_BLOCK_LOC_SPANS[direction]
												tile.InverseBlockLocSpans = 3 - tile.BlockLocSpans
											} else if wall.TypeA == 32 {
												tile.CheckLocSpans = 6
												tile.BlockLocSpans = WALL_CORNER_TYPE_32_BLOCK_LOC_SPANS[direction]
												tile.InverseBlockLocSpans = 6 - tile.BlockLocSpans
											} else if wall.TypeA == 64 {
												tile.CheckLocSpans = 12
												tile.BlockLocSpans = WALL_CORNER_TYPE_64_BLOCK_LOC_SPANS[direction]
												tile.InverseBlockLocSpans = 12 - tile.BlockLocSpans
											} else {
												tile.CheckLocSpans = 9
												tile.BlockLocSpans = WALL_CORNER_TYPE_128_BLOCK_LOC_SPANS[direction]
												tile.InverseBlockLocSpans = 9 - tile.BlockLocSpans
											}
											if wall.TypeA&frontWallTypes != 0 && !w.WallVisible(originalLevel, tileX, tileZ, wall.TypeA) {
												if m := wall.ModelA.GetModel(); m != nil {
													m.Draw1(0, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, wall.X-EyeX, wall.Y-EyeY, wall.Z-EyeZ, wall.BitSet)
												}
											}
											if wall.TypeB&frontWallTypes != 0 && !w.WallVisible(originalLevel, tileX, tileZ, wall.TypeB) {
												if m := wall.ModelB.GetModel(); m != nil {
													m.Draw1(0, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, wall.X-EyeX, wall.Y-EyeY, wall.Z-EyeZ, wall.BitSet)
												}
											}
										}

										// rev-244 visibility cull reads the source's cached minY
										// (set when this decor was last drawn / seeded from a
										// static model), not the resolved model's maxY (rev-225).
										if decor != nil && !w.Visible(originalLevel, tileX, tileZ, decor.MinY) {
											if decor.Type&frontWallTypes != 0 {
												if m := decor.Model.GetModel(); m != nil {
													decor.MinY = m.MaxY // Java: model.minY == Go Model.MaxY (lineage inversion; height above origin)
													m.Draw1(decor.Angle, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, decor.X-EyeX, decor.Y-EyeY, decor.Z-EyeZ, decor.BitSet)
												}
											} else if decor.Type&0x300 != 0 {
												x := decor.X - EyeX
												y := decor.Y - EyeY
												z := decor.Z - EyeZ
												angle := decor.Angle

												nearestX := 0
												if angle == 1 || angle == 2 {
													nearestX = -x
												} else {
													nearestX = x
												}

												nearestZ := 0
												if angle == 2 || angle == 3 {
													nearestZ = -z
												} else {
													nearestZ = z
												}

												if decor.Type&0x100 != 0 && nearestZ < nearestX {
													drawX := x + WALL_DECORATION_INSET_X[angle]
													drawZ := z + WALL_DECORATION_INSET_Z[angle]
													if m := decor.Model.GetModel(); m != nil {
														decor.MinY = m.MaxY // Java: model.minY == Go Model.MaxY (lineage inversion; height above origin)
														m.Draw1(angle*512+256, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, drawX, y, drawZ, decor.BitSet)
													}
												}
												if decor.Type&0x200 != 0 && nearestZ > nearestX {
													drawX := x + WALL_DECORATION_OUTSET_X[angle]
													drawZ := z + WALL_DECORATION_OUTSET_Z[angle]
													if m := decor.Model.GetModel(); m != nil {
														decor.MinY = m.MaxY // Java: model.minY == Go Model.MaxY (lineage inversion; height above origin)
														m.Draw1((angle*512+1280)&0x7FF, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, drawX, y, drawZ, decor.BitSet)
													}
												}
											}
										}

										if tileDrawn {
											groundDecor := tile.GroundDecor
											if groundDecor != nil {
												if m := groundDecor.Model.GetModel(); m != nil {
													m.Draw1(0, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, groundDecor.X-EyeX, groundDecor.Y-EyeY, groundDecor.Z-EyeZ, groundDecor.BitSet)
												}
											}

											objs := tile.GroundObj
											if objs != nil && objs.Offset == 0 {
												if objs.BottomObj != nil {
													if m := objs.BottomObj.GetModel(); m != nil {
														m.Draw1(0, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, objs.X-EyeX, objs.Y-EyeY, objs.Z-EyeZ, objs.BitSet)
													}
												}
												if objs.MiddleObj != nil {
													if m := objs.MiddleObj.GetModel(); m != nil {
														m.Draw1(0, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, objs.X-EyeX, objs.Y-EyeY, objs.Z-EyeZ, objs.BitSet)
													}
												}
												if objs.TopObj != nil {
													if m := objs.TopObj.GetModel(); m != nil {
														m.Draw1(0, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, objs.X-EyeX, objs.Y-EyeY, objs.Z-EyeZ, objs.BitSet)
													}
												}
											}
										}

										spans := tile.LocSpans
										if spans != 0 {
											if tileX < EyeTileX && spans&0x4 != 0 {
												adjacent := tiles[tileX+1][tileZ]
												if adjacent != nil && adjacent.Update {
													DrawTileQueue.AddTail(adjacent.DrawQueueNode)
												}
											}

											if tileZ < EyeTileZ && spans&0x2 != 0 {
												adjacent := tiles[tileX][tileZ+1]
												if adjacent != nil && adjacent.Update {
													DrawTileQueue.AddTail(adjacent.DrawQueueNode)
												}
											}

											if tileX > EyeTileX && spans&0x1 != 0 {
												adjacent := tiles[tileX-1][tileZ]
												if adjacent != nil && adjacent.Update {
													DrawTileQueue.AddTail(adjacent.DrawQueueNode)
												}
											}

											if tileZ > EyeTileZ && spans&0x8 != 0 {
												adjacent := tiles[tileX][tileZ-1]
												if adjacent != nil && adjacent.Update {
													DrawTileQueue.AddTail(adjacent.DrawQueueNode)
												}
											}
										}
										break
									}

									if tile.CheckLocSpans != 0 {
										draw := true
										for i := range tile.LocCount {
											if tile.Locs[i].Cycle != Cycle && tile.LocSpan[i]&tile.CheckLocSpans == tile.BlockLocSpans {
												draw = false
												break
											}
										}

										if draw {
											wall := tile.Wall

											if !w.WallVisible(originalLevel, tileX, tileZ, wall.TypeA) {
												if m := wall.ModelA.GetModel(); m != nil {
													m.Draw1(0, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, wall.X-EyeX, wall.Y-EyeY, wall.Z-EyeZ, wall.BitSet)
												}
											}

											tile.CheckLocSpans = 0
										}
									}

									if !tile.ContainsLocs {
										break
									}

									locCount := tile.LocCount
									tile.ContainsLocs = false
									locBufferSize := 0
								iterateLocs:
									for var11 := range locCount {
										loc := tile.Locs[var11]
										if loc.Cycle != Cycle {
											for x := loc.MinSceneTileX; x <= loc.MaxSceneTileX; x++ {
												for z := loc.MinSceneTileZ; z <= loc.MaxSceneTileZ; z++ {
													other := tiles[x][z]

													if other.Visible {
														tile.ContainsLocs = true
														continue iterateLocs
													}

													if other.CheckLocSpans != 0 {
														spans := 0
														if x > loc.MinSceneTileX {
															spans++
														}

														if x < loc.MaxSceneTileX {
															spans += 4
														}

														if z > loc.MinSceneTileZ {
															spans += 8
														}

														if z < loc.MaxSceneTileZ {
															spans += 2
														}

														if spans&other.CheckLocSpans == tile.InverseBlockLocSpans {
															tile.ContainsLocs = true
															continue iterateLocs
														}
													}
												}
											}

											LocBuffer[locBufferSize] = loc
											locBufferSize++

											minTileDistanceX := EyeTileX - loc.MinSceneTileX
											maxTileDistanceX := loc.MaxSceneTileX - EyeTileX
											minTileDistanceX = max(maxTileDistanceX, minTileDistanceX)

											minTileDistanceZ := EyeTileZ - loc.MinSceneTileZ
											maxTileDistanceZ := loc.MaxSceneTileZ - EyeTileZ
											if maxTileDistanceZ > minTileDistanceZ {
												loc.Distance = minTileDistanceX + maxTileDistanceZ
											} else {
												loc.Distance = minTileDistanceX + minTileDistanceZ
											}
										}
									}

									for locBufferSize > 0 {
										farthestDistance := -50
										farthestIndex := -1

										for index := range locBufferSize {
											loc := LocBuffer[index]

											if loc.Distance > farthestDistance && loc.Cycle != Cycle {
												farthestDistance = loc.Distance
												farthestIndex = index
											}
										}

										if farthestIndex == -1 {
											break
										}

										farthest := LocBuffer[farthestIndex]
										farthest.Cycle = Cycle

										// rev-244 culls on the source's cached minY (seeded from a
										// static model / updated when last drawn), then resolves
										// and draws — vs rev-225's resolved model.maxY.
										if !w.LocVisible(originalLevel, farthest.MinSceneTileX, farthest.MaxSceneTileX, farthest.MinSceneTileZ, farthest.MaxSceneTileZ, farthest.MinY) {
											if m := farthest.Model.GetModel(); m != nil {
												farthest.MinY = m.MaxY // Java: model.minY == Go Model.MaxY (lineage inversion; height above origin)
												m.Draw1(farthest.Yaw, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, farthest.X-EyeX, farthest.Y-EyeY, farthest.Z-EyeZ, farthest.BitSet)
											}
										}

										for x := farthest.MinSceneTileX; x <= farthest.MaxSceneTileX; x++ {
											for z := farthest.MinSceneTileZ; z <= farthest.MaxSceneTileZ; z++ {
												occupied := tiles[x][z]

												if occupied.CheckLocSpans != 0 {
													DrawTileQueue.AddTail(occupied.DrawQueueNode)
												} else if (x != tileX || z != tileZ) && occupied.Update {
													DrawTileQueue.AddTail(occupied.DrawQueueNode)
												}
											}
										}
									}

									if !tile.ContainsLocs {
										break
									}
								}
							}
						}
						if tileX > EyeTileX || tileX <= MinDrawTileX {
							break
						}
						var9 = tiles[tileX-1][tileZ]
					}
					if tileX < EyeTileX || tileX >= MaxDrawTileX-1 {
						break
					}
					var9 = tiles[tileX+1][tileZ]
				}
				if tileZ > EyeTileZ || tileZ <= MinDrawTileZ {
					break
				}
				var9 = tiles[tileX][tileZ-1]
			}
			if tileZ < EyeTileZ || tileZ >= MaxDrawTileZ-1 {
				break
			}
			var9 = tiles[tileX][tileZ+1]
		}

		tile.Update = false
		TilesRemaining--

		objs := tile.GroundObj
		if objs != nil && objs.Offset != 0 {
			if objs.BottomObj != nil {
				if m := objs.BottomObj.GetModel(); m != nil {
					m.Draw1(0, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, objs.X-EyeX, objs.Y-EyeY-objs.Offset, objs.Z-EyeZ, objs.BitSet)
				}
			}

			if objs.MiddleObj != nil {
				if m := objs.MiddleObj.GetModel(); m != nil {
					m.Draw1(0, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, objs.X-EyeX, objs.Y-EyeY-objs.Offset, objs.Z-EyeZ, objs.BitSet)
				}
			}

			if objs.TopObj != nil {
				if m := objs.TopObj.GetModel(); m != nil {
					m.Draw1(0, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, objs.X-EyeX, objs.Y-EyeY-objs.Offset, objs.Z-EyeZ, objs.BitSet)
				}
			}
		}

		if tile.BackWallTypes != 0 {
			decor := tile.Decor
			if decor != nil && !w.Visible(originalLevel, tileX, tileZ, decor.MinY) {
				if decor.Type&tile.BackWallTypes != 0 {
					if m := decor.Model.GetModel(); m != nil {
						decor.MinY = m.MaxY // Java: model.minY == Go Model.MaxY (lineage inversion; height above origin)
						m.Draw1(decor.Angle, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, decor.X-EyeX, decor.Y-EyeY, decor.Z-EyeZ, decor.BitSet)
					}
				} else if decor.Type&0x300 != 0 {
					x := decor.X - EyeX
					y := decor.Y - EyeY
					z := decor.Z - EyeZ
					angle := decor.Angle

					nearestX := 0
					if angle == 1 || angle == 2 {
						nearestX = -x
					} else {
						nearestX = x
					}

					nearestZ := 0
					if angle == 2 || angle == 3 {
						nearestZ = -z
					} else {
						nearestZ = z
					}

					if decor.Type&0x100 != 0 && nearestZ >= nearestX {
						drawX := x + WALL_DECORATION_INSET_X[angle]
						drawZ := z + WALL_DECORATION_INSET_Z[angle]
						if m := decor.Model.GetModel(); m != nil {
							decor.MinY = m.MaxY // Java: model.minY == Go Model.MaxY (lineage inversion; height above origin)
							m.Draw1(angle*512+256, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, drawX, y, drawZ, decor.BitSet)
						}
					}
					if decor.Type&0x200 != 0 && nearestZ <= nearestX {
						drawX := x + WALL_DECORATION_OUTSET_X[angle]
						drawZ := z + WALL_DECORATION_OUTSET_Z[angle]
						if m := decor.Model.GetModel(); m != nil {
							decor.MinY = m.MaxY // Java: model.minY == Go Model.MaxY (lineage inversion; height above origin)
							m.Draw1((angle*512+1280)&0x7FF, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, drawX, y, drawZ, decor.BitSet)
						}
					}
				}
			}

			wall := tile.Wall
			if wall != nil {
				if wall.TypeB&tile.BackWallTypes != 0 && !w.WallVisible(originalLevel, tileX, tileZ, wall.TypeB) {
					if m := wall.ModelB.GetModel(); m != nil {
						m.Draw1(0, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, wall.X-EyeX, wall.Y-EyeY, wall.Z-EyeZ, wall.BitSet)
					}
				}

				if wall.TypeA&tile.BackWallTypes != 0 && !w.WallVisible(originalLevel, tileX, tileZ, wall.TypeA) {
					if m := wall.ModelA.GetModel(); m != nil {
						m.Draw1(0, SinEyePitch, CosEyePitch, SinEyeYaw, CosEyeYaw, wall.X-EyeX, wall.Y-EyeY, wall.Z-EyeZ, wall.BitSet)
					}
				}
			}
		}

		if level < w.MaxLevel-1 {
			above := w.LevelTiles[level+1][tileX][tileZ]
			if above != nil && above.Update {
				DrawTileQueue.AddTail(above.DrawQueueNode)
			}
		}

		if tileX < EyeTileX {
			adjacent := tiles[tileX+1][tileZ]
			if adjacent != nil && adjacent.Update {
				DrawTileQueue.AddTail(adjacent.DrawQueueNode)
			}
		}

		if tileZ < EyeTileZ {
			adjacent := tiles[tileX][tileZ+1]
			if adjacent != nil && adjacent.Update {
				DrawTileQueue.AddTail(adjacent.DrawQueueNode)
			}
		}

		if tileX > EyeTileX {
			adjacent := tiles[tileX-1][tileZ]
			if adjacent != nil && adjacent.Update {
				DrawTileQueue.AddTail(adjacent.DrawQueueNode)
			}
		}

		if tileZ > EyeTileZ {
			adjacent := tiles[tileX][tileZ-1]
			if adjacent != nil && adjacent.Update {
				DrawTileQueue.AddTail(adjacent.DrawQueueNode)
			}
		}
	}
}

func (w *World) DrawTileUnderlay(arg0 *typ.QuickGround, arg1, arg2, arg3, arg4, arg5, arg6, arg7 int) {
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
	var21 := (var12*arg4 + var10*arg5) >> 16
	var35 := (var12*arg5 - var10*arg4) >> 16
	var32 := var21
	var41 := (var17*arg3 - var35*arg2) >> 16
	var36 := (var17*arg2 + var35*arg3) >> 16
	var40 := var41
	if var36 < 50 {
		return
	}
	var21 = (var11*arg4 + var14*arg5) >> 16
	var33 := (var11*arg5 - var14*arg4) >> 16
	var14 = var21
	var21 = (var18*arg3 - var33*arg2) >> 16
	var34 := (var18*arg2 + var33*arg3) >> 16
	var18 = var21
	if var34 < 50 {
		return
	}
	var21 = (var16*arg4 + var13*arg5) >> 16
	var16 = (var16*arg5 - var13*arg4) >> 16
	var37 := var21
	var21 = (var19*arg3 - var16*arg2) >> 16
	var16 = (var19*arg2 + var16*arg3) >> 16
	var19 = var21
	if var16 < 50 {
		return
	}
	var21 = (var15*arg4 + var9*arg5) >> 16
	var38 := (var15*arg5 - var9*arg4) >> 16
	var31 := var21
	var21 = (var20*arg3 - var38*arg2) >> 16
	var39 := (var20*arg2 + var38*arg3) >> 16
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

func (w *World) DrawTileOverlay(arg0 int, arg1 int, arg2 *typ.Ground, arg3, arg4, arg5, arg6 int) {
	var9 := len(arg2.VertexX)
	for i := range var9 {
		var11 := arg2.VertexX[i] - EyeX
		var12 := arg2.VertexY[i] - EyeY
		var13 := arg2.VertexZ[i] - EyeZ
		var14 := (var13*arg0 + var11*arg6) >> 16
		var23 := (var13*arg6 - var11*arg0) >> 16
		var25 := (var12*arg4 - var23*arg5) >> 16
		var24 := (var12*arg5 + var23*arg4) >> 16
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

func (w *World) MulLightness(arg0, arg1 int) int {
	var4 := 127 - arg0
	arg0 = var4 * (arg1 & 0x7F) / 160
	if arg0 < 2 {
		arg0 = 2
	} else if arg0 > 126 {
		arg0 = 126
	}
	return (arg1 & 0xFF80) + arg0
}

func (w *World) PointInsideTriangle(arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7 int) bool {
	if arg1 < arg2 && arg1 < arg3 && arg1 < arg4 {
		return false
	}
	// Java: World.java:1889 — all-three-greater early reject.
	if arg1 > arg2 && arg1 > arg3 && arg1 > arg4 {
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

func (w *World) UpdateActiveOccluders() {
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
				var7 = max(var7, 0)
				var8 = var5.MaxTileZ - EyeTileZ + 25
				var8 = min(var8, 50)
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
					var5.MinDeltaZ = ((var5.MinZ - EyeZ) << 8) / var10
					var5.MaxDeltaZ = ((var5.MaxZ - EyeZ) << 8) / var10
					var5.MinDeltaY = ((var5.MinY - EyeY) << 8) / var10
					var5.MaxDeltaY = ((var5.MaxY - EyeY) << 8) / var10
					ActiveOccluders[ActiveOccluderCount] = var5
					ActiveOccluderCount++
				}
			}
		} else if var5.Type == 2 {
			var6 = var5.MinTileZ - EyeTileZ + 25
			if var6 >= 0 && var6 <= 50 {
				var7 = var5.MinTileX - EyeTileX + 25
				var7 = max(var7, 0)
				var8 = var5.MaxTileX - EyeTileX + 25
				var8 = min(var8, 50)
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
					var5.MinDeltaX = ((var5.MinX - EyeX) << 8) / var10
					var5.MaxDeltaX = ((var5.MaxX - EyeX) << 8) / var10
					var5.MinDeltaY = ((var5.MinY - EyeY) << 8) / var10
					var5.MaxDeltaY = ((var5.MaxY - EyeY) << 8) / var10
					ActiveOccluders[ActiveOccluderCount] = var5
					ActiveOccluderCount++
				}
			}
		} else if var5.Type == 4 {
			var6 = var5.MinY - EyeY
			if var6 > 128 {
				var7 = var5.MinTileZ - EyeTileZ + 25
				var7 = max(var7, 0)
				var8 = var5.MaxTileZ - EyeTileZ + 25
				var8 = min(var8, 50)
				if var7 <= var8 {
					var9 := var5.MinTileX - EyeTileX + 25
					var9 = max(var9, 0)
					var10 = var5.MaxTileX - EyeTileX + 25
					var10 = min(var10, 50)
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
						var5.MinDeltaX = ((var5.MinX - EyeX) << 8) / var6
						var5.MaxDeltaX = ((var5.MaxX - EyeX) << 8) / var6
						var5.MinDeltaZ = ((var5.MinZ - EyeZ) << 8) / var6
						var5.MaxDeltaZ = ((var5.MaxZ - EyeZ) << 8) / var6
						ActiveOccluders[ActiveOccluderCount] = var5
						ActiveOccluderCount++
					}
				}
			}
		}
	}
}

func (w *World) TileVisible(arg0, arg1, arg2 int) bool {
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

func (w *World) WallVisible(arg0, arg1, arg2, arg3 int) bool {
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

func (w *World) Visible(arg0, arg1, arg2, arg3 int) bool {
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

func (w *World) LocVisible(arg0, arg1, arg2, arg3, arg4, arg5 int) bool {
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

func (w *World) Occluded(arg0, arg1, arg2 int) bool {
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
				var7 = var5.MinZ + ((var5.MinDeltaZ * var6) >> 8)
				var8 = var5.MaxZ + ((var5.MaxDeltaZ * var6) >> 8)
				var9 = var5.MinY + ((var5.MinDeltaY * var6) >> 8)
				var10 = var5.MaxY + ((var5.MaxDeltaY * var6) >> 8)
				if arg2 >= var7 && arg2 <= var8 && arg1 >= var9 && arg1 <= var10 {
					return true
				}
			}
		case 2:
			var6 = arg0 - var5.MinX
			if var6 > 0 {
				var7 = var5.MinZ + ((var5.MinDeltaZ * var6) >> 8)
				var8 = var5.MaxZ + ((var5.MaxDeltaZ * var6) >> 8)
				var9 = var5.MinY + ((var5.MinDeltaY * var6) >> 8)
				var10 = var5.MaxY + ((var5.MaxDeltaY * var6) >> 8)
				if arg2 >= var7 && arg2 <= var8 && arg1 >= var9 && arg1 <= var10 {
					return true
				}
			}
		case 3:
			var6 = var5.MinZ - arg2
			if var6 > 0 {
				var7 = var5.MinX + ((var5.MinDeltaX * var6) >> 8)
				var8 = var5.MaxX + ((var5.MaxDeltaX * var6) >> 8)
				var9 = var5.MinY + ((var5.MinDeltaY * var6) >> 8)
				var10 = var5.MaxY + ((var5.MaxDeltaY * var6) >> 8)
				if arg0 >= var7 && arg0 <= var8 && arg1 >= var9 && arg1 <= var10 {
					return true
				}
			}
		case 4:
			var6 = arg2 - var5.MinZ
			if var6 > 0 {
				var7 = var5.MinX + ((var5.MinDeltaX * var6) >> 8)
				var8 = var5.MaxX + ((var5.MaxDeltaX * var6) >> 8)
				var9 = var5.MinY + ((var5.MinDeltaY * var6) >> 8)
				var10 = var5.MaxY + ((var5.MaxDeltaY * var6) >> 8)
				if arg0 >= var7 && arg0 <= var8 && arg1 >= var9 && arg1 <= var10 {
					return true
				}
			}
		case 5:
			var6 = arg1 - var5.MinY
			if var6 > 0 {
				var7 = var5.MinX + ((var5.MinDeltaX * var6) >> 8)
				var8 = var5.MaxX + ((var5.MaxDeltaX * var6) >> 8)
				var9 = var5.MinZ + ((var5.MinDeltaZ * var6) >> 8)
				var10 = var5.MaxZ + ((var5.MaxDeltaZ * var6) >> 8)
				if arg0 >= var7 && arg0 <= var8 && arg2 >= var9 && arg2 <= var10 {
					return true
				}
			}
		}
	}
	return false
}
