package world

import (
	"math"
	"math/rand"

	"github.com/zsrv/goscape-client/pkg/jagex2/config/flotype"
	"github.com/zsrv/goscape-client/pkg/jagex2/config/loctype"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/entity"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/model"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/world3d"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pix3d"
	"github.com/zsrv/goscape-client/pkg/jagex2/io"
	"github.com/zsrv/goscape-client/pkg/jagex2/io/ondemand"
)

var (
	LowMemory                          bool = true
	LevelBuilt                         int
	FullBright                         bool
	ROTATION_WALL_TYPE                 = []int{1, 2, 4, 8}
	ROTATION_WALL_CORNER_TYPE          = []int{16, 32, 64, 128}
	WALL_DECORATION_ROTATION_FORWARD_X = []int{1, 0, -1, 0}
	WALL_DECORATION_ROTATION_FORWARD_Z = []int{0, -1, 0, 1}
	// Java: randomHueOffset = (int)(Math.random()*17.0) - 8 (World.java:86),
	// randomLightnessOffset = (int)(Math.random()*33.0) - 16 (World.java:89).
	// The cast binds ONLY to the non-negative product, then the subtraction is
	// integer arithmetic — yielding a uniform -8..8 / -16..16. Casting the whole
	// expression instead would truncate negatives toward zero (int(-2.5) == -2
	// vs Java's floor -3), skewing the distribution.
	RandomHueOffset       = int(rand.Float64()*17.0) - 8
	RandomLightnessOffset = int(rand.Float64()*33.0) - 16
)

// Reset clears every package-level binding to its first-load state. Intended
// for tests that need to start from a clean slate so a previous test's
// configuration can't leak into the next (world building keeps its state as
// package vars by design — see CLAUDE.md "Global State Pattern").
//
// Excluded: ROTATION_WALL_TYPE, ROTATION_WALL_CORNER_TYPE,
// WALL_DECORATION_ROTATION_FORWARD_X, WALL_DECORATION_ROTATION_FORWARD_Z —
// const-shaped lookup tables populated once at package load.
// Excluded: RandomHueOffset, RandomLightnessOffset — randomized seeds set
// once at package load that are meant to be stable for the process lifetime.
func Reset() {
	LowMemory = true
	LevelBuilt = 0
	FullBright = false
}

type World struct {
	MaxTileX       int
	MaxTileZ       int
	LevelHeightMap [][][]int
	// Java declares these as signed byte[][][]; ported as int8 so widening to int
	// sign-extends like Java (per the project byte->int8 mapping rule).
	LevelTileFlags           [][][]int8
	LevelTileUnderlayIDs     [][][]int8
	LevelTileOverlayIDs      [][][]int8
	LevelTileOverlayShape    [][][]int8
	LevelTileOverlayRotation [][][]int8
	LevelShadeMap            [][][]int8
	LevelLightMap            [][]int
	BlendChroma              []int
	BlendSaturation          []int
	BlendLightness           []int
	BlendLuminance           []int
	BlendMagnitude           []int
	LevelOccludeMap          [][][]int
}

func NewWorld(arg0 int, arg1 [][][]int8, arg2 int, arg3 [][][]int) *World {
	var w World
	w.MaxTileX = arg2
	w.MaxTileZ = arg0
	w.LevelHeightMap = arg3
	w.LevelTileFlags = arg1
	w.LevelTileUnderlayIDs = make([][][]int8, 4)
	for i := range w.LevelTileUnderlayIDs {
		w.LevelTileUnderlayIDs[i] = make([][]int8, w.MaxTileX)
		for j := range w.LevelTileUnderlayIDs[i] {
			w.LevelTileUnderlayIDs[i][j] = make([]int8, w.MaxTileZ)
		}
	}
	w.LevelTileOverlayIDs = make([][][]int8, 4)
	for i := range w.LevelTileOverlayIDs {
		w.LevelTileOverlayIDs[i] = make([][]int8, w.MaxTileX)
		for j := range w.LevelTileOverlayIDs[i] {
			w.LevelTileOverlayIDs[i][j] = make([]int8, w.MaxTileZ)
		}
	}
	w.LevelTileOverlayShape = make([][][]int8, 4)
	for i := range w.LevelTileOverlayShape {
		w.LevelTileOverlayShape[i] = make([][]int8, w.MaxTileX)
		for j := range w.LevelTileOverlayShape[i] {
			w.LevelTileOverlayShape[i][j] = make([]int8, w.MaxTileZ)
		}
	}
	w.LevelTileOverlayRotation = make([][][]int8, 4)
	for i := range w.LevelTileOverlayRotation {
		w.LevelTileOverlayRotation[i] = make([][]int8, w.MaxTileX)
		for j := range w.LevelTileOverlayRotation[i] {
			w.LevelTileOverlayRotation[i][j] = make([]int8, w.MaxTileZ)
		}
	}
	w.LevelOccludeMap = make([][][]int, 4)
	for i := range w.LevelOccludeMap {
		w.LevelOccludeMap[i] = make([][]int, w.MaxTileX+1)
		for j := range w.LevelOccludeMap[i] {
			w.LevelOccludeMap[i][j] = make([]int, w.MaxTileZ+1)
		}
	}
	w.LevelShadeMap = make([][][]int8, 4)
	for i := range w.LevelShadeMap {
		w.LevelShadeMap[i] = make([][]int8, w.MaxTileX+1)
		for j := range w.LevelShadeMap[i] {
			w.LevelShadeMap[i][j] = make([]int8, w.MaxTileZ+1)
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

// Java: spreadHeight (World.java:108-133) — new in 244, replacing 225's
// clearLandscape (water-fill, deleted in 244) for absent neighbouring
// mapsquares: clears the shadow and copies the ground heightmap inward from
// the loaded edges so terrain slopes off smoothly instead of dropping to sea
// level. Note the inclusive (<=) loop bounds: the spread covers 65 rows/cols
// for a 64-tile square, faithful to Java.
func (w *World) SpreadHeight(startX, startZ, endX, endZ int) {
	for z := startZ; z <= startZ+endZ; z++ {
		for x := startX; x <= startX+endX; x++ {
			if x >= 0 && x < w.MaxTileX && z >= 0 && z < w.MaxTileZ {
				w.LevelShadeMap[0][x][z] = 127
				if startX == x && x > 0 {
					w.LevelHeightMap[0][x][z] = w.LevelHeightMap[0][x-1][z]
				}
				if startX+endX == x && x < w.MaxTileX-1 {
					w.LevelHeightMap[0][x][z] = w.LevelHeightMap[0][x+1][z]
				}
				if startZ == z && z > 0 {
					w.LevelHeightMap[0][x][z] = w.LevelHeightMap[0][x][z-1]
				}
				if startZ+endZ == z && z < w.MaxTileZ-1 {
					w.LevelHeightMap[0][x][z] = w.LevelHeightMap[0][x][z+1]
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
							w.LevelTileOverlayIDs[i][var11][var12] = var7.G1B() // Java: (byte) g1b
							w.LevelTileOverlayShape[i][var11][var12] = int8((var13 - 2) / 4)
							w.LevelTileOverlayRotation[i][var11][var12] = int8((var13 - 2) & 0x3)
						} else if var13 <= 81 {
							w.LevelTileFlags[i][var11][var12] = int8(var13 - 49)
						} else {
							w.LevelTileUnderlayIDs[i][var11][var12] = int8(var13 - 81)
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

func (w *World) LoadLocations(src []byte, scene *world3d.World3D, collision []*dash3d.CollisionMap, zOffset int, xOffset int) {
	buf := io.NewPacket(src)
	locId := -1

	for {
		deltaId := buf.GSmartS()
		if deltaId == 0 {
			return
		}

		locId += deltaId

		locPos := 0
		for {
			deltaPos := buf.GSmartS()
			if deltaPos == 0 {
				break
			}

			locPos += deltaPos - 1

			z := locPos & 0x3F
			x := (locPos >> 6) & 0x3F
			level := locPos >> 12

			info := buf.G1()
			shape := info >> 2
			angle := info & 0x3
			stx := x + xOffset
			stz := z + zOffset

			if stx > 0 && stz > 0 && stx < 103 && stz < 103 {
				currentLevel := level
				if w.LevelTileFlags[1][stx][stz]&0x2 == 2 {
					currentLevel = level - 1
				}

				var collisionMap *dash3d.CollisionMap
				if currentLevel >= 0 {
					collisionMap = collision[currentLevel]
				}

				w.AddLoc(collisionMap, level, stz, angle, shape, scene, locId, stx)
			}
		}
	}
}

func (w *World) AddLoc(collision *dash3d.CollisionMap, level, z, angle, shape int, scene *world3d.World3D, locId, x int) {
	if LowMemory {
		if w.LevelTileFlags[level][x][z]&0x10 != 0 {
			return
		}

		if w.GetDrawLevel(level, x, z) != LevelBuilt {
			return
		}
	}

	// Corner labels reflect their actual world positions (matches the
	// standalone AddLoc at world.go:1015-1019 below): (x, z) = SW,
	// (x+1, z) = SE, (x+1, z+1) = NE, (x, z+1) = NW. The prior labels
	// here had NW/NE swapped — values still flowed correctly through
	// Java-positional GetModel calls but the names misled readers.
	heightSW := w.LevelHeightMap[level][x][z]
	heightSE := w.LevelHeightMap[level][x+1][z]
	heightNE := w.LevelHeightMap[level][x+1][z+1]
	heightNW := w.LevelHeightMap[level][x][z+1]
	y := (heightSW + heightSE + heightNE + heightNW) >> 2

	loc := loctype.Get(locId)

	typeCode := x + (z << 7) + (locId << 14) + 0x40000000
	if !loc.Active {
		typeCode += math.MinInt32
	}

	info := byte((angle << 6) + shape)

	// buildModel returns the scene ModelSource for a (shape, angle): a static loc
	// model when this loc has no anim, else a self-animating ClientLocAnim. Java:
	// rev-244 World.addLoc inlines this `if (loc.anim == -1) model =
	// loc.getModel(...) else model = new ClientLocAnim(...)` idiom in every shape
	// branch; it is hoisted to a closure here. ClientLocAnim's ctor takes the
	// corner heights in (NW, NE, SW, ..., SE) order (its source arg labels are
	// permuted but the values match the static getModel call positionally).
	buildModel := func(shape, angle int) entity.ModelSource {
		if loc.Anim == -1 {
			return entity.ModelSourceOf(loc.GetModel(shape, angle, heightSW, heightSE, heightNE, heightNW, -1))
		}
		return entity.NewClientLocAnim(heightNW, heightNE, heightSW, shape, angle, true, heightSE, locId, loc.Anim)
	}

	if shape != 22 {
		if shape == 10 || shape == 11 {
			modelSrc := buildModel(10, angle)

			if modelSrc != nil {
				yaw := 0
				if shape == 11 {
					yaw += 256
				}

				width := 0
				length := 0
				if angle == 1 || angle == 3 {
					length = loc.Length
					width = loc.Width
				} else {
					length = loc.Width
					width = loc.Length
				}

				// Java: World.java:274-285 — addLoc args are (var15=y, arg2=level,
				// var17=typeCode, arg3=z, arg9=x, var20=length, var18=info, var19=model,
				// var22=yaw, var21=width). The shademap is indexed [level][x+i][z+j] and the
				// outer loop bound is var20 (length); both match here. For an animated loc
				// (model is a ClientLocAnim) the shadow re-resolves the static frame model.
				if scene.AddLoc1(y, level, typeCode, z, x, length, info, modelSrc, yaw, width) && loc.Shadow {
					var model2 *model.Model
					if m, ok := modelSrc.(*model.Model); ok {
						model2 = m
					} else {
						model2 = loc.GetModel(10, angle, heightSW, heightSE, heightNE, heightNW, -1)
					}
					if model2 != nil {
						for dx := 0; dx <= length; dx++ {
							for dz := 0; dz <= width; dz++ {
								shade := model2.Radius / 4
								shade = min(shade, 30)

								if shade > int(w.LevelShadeMap[level][x+dx][z+dz]) {
									w.LevelShadeMap[level][x+dx][z+dz] = int8(shade)
								}
							}
						}
					}
				}
			}

			if loc.BlockWalk && collision != nil {
				collision.AddLoc(angle, loc.Length, loc.Width, x, z, loc.BlockRange)
			}
		} else if shape >= 12 {
			modelSrc := buildModel(shape, angle)

			scene.AddLoc1(y, level, typeCode, z, x, 1, info, modelSrc, 0, 1)

			if shape >= 12 && shape <= 17 && shape != 13 && level > 0 {
				w.LevelOccludeMap[level][x][z] |= 0x924
			}

			if loc.BlockWalk && collision != nil {
				collision.AddLoc(angle, loc.Length, loc.Width, x, z, loc.BlockRange)
			}
		} else if shape == 0 {
			modelSrc := buildModel(0, angle)

			scene.AddWall(0, y, level, ROTATION_WALL_TYPE[angle], modelSrc, nil, x, typeCode, z, info)

			switch angle {
			case 0:
				if loc.Shadow {
					w.LevelShadeMap[level][x][z] = 50
					w.LevelShadeMap[level][x][z+1] = 50
				}
				if loc.Occlude {
					w.LevelOccludeMap[level][x][z] |= 0x249
				}
			case 1:
				if loc.Shadow {
					w.LevelShadeMap[level][x][z+1] = 50
					w.LevelShadeMap[level][x+1][z+1] = 50
				}
				if loc.Occlude {
					w.LevelOccludeMap[level][x][z+1] |= 0x492
				}
			case 2:
				if loc.Shadow {
					w.LevelShadeMap[level][x+1][z] = 50
					w.LevelShadeMap[level][x+1][z+1] = 50
				}
				if loc.Occlude {
					w.LevelOccludeMap[level][x+1][z] |= 0x249
				}
			case 3:
				if loc.Shadow {
					w.LevelShadeMap[level][x][z] = 50
					w.LevelShadeMap[level][x+1][z] = 50
				}
				if loc.Occlude {
					w.LevelOccludeMap[level][x][z] |= 0x492
				}
			}

			if loc.BlockWalk && collision != nil {
				collision.AddWall(angle, z, x, loc.BlockRange, shape)
			}

			if loc.WallWidth != 16 {
				scene.SetWallDecorationOffset(level, z, x, loc.WallWidth)
			}
		} else if shape == 1 {
			modelSrc := buildModel(1, angle)

			scene.AddWall(0, y, level, ROTATION_WALL_CORNER_TYPE[angle], modelSrc, nil, x, typeCode, z, info)

			if loc.Shadow {
				switch angle {
				case 0:
					w.LevelShadeMap[level][x][z+1] = 50
				case 1:
					w.LevelShadeMap[level][x+1][z+1] = 50
				case 2:
					w.LevelShadeMap[level][x+1][z] = 50
				case 3:
					w.LevelShadeMap[level][x][z] = 50
				}
			}

			if loc.BlockWalk && collision != nil {
				collision.AddWall(angle, z, x, loc.BlockRange, shape)
			}
		} else {
			switch shape {
			case 2:
				offset := (angle + 1) & 0x3

				modelSrc1 := buildModel(2, angle+4)
				modelSrc2 := buildModel(2, offset)

				scene.AddWall(ROTATION_WALL_TYPE[offset], y, level, ROTATION_WALL_TYPE[angle], modelSrc1, modelSrc2, x, typeCode, z, info)

				if loc.Occlude {
					switch angle {
					case 0:
						w.LevelOccludeMap[level][x][z] |= 0x249
						w.LevelOccludeMap[level][x][z+1] |= 0x492
					case 1:
						w.LevelOccludeMap[level][x][z+1] |= 0x492
						w.LevelOccludeMap[level][x+1][z] |= 0x249
					case 2:
						w.LevelOccludeMap[level][x+1][z] |= 0x249
						w.LevelOccludeMap[level][x][z] |= 0x492
					case 3:
						w.LevelOccludeMap[level][x][z] |= 0x492
						w.LevelOccludeMap[level][x][z] |= 0x249
					}
				}

				if loc.BlockWalk && collision != nil {
					collision.AddWall(angle, z, x, loc.BlockRange, shape)
				}

				if loc.WallWidth != 16 {
					scene.SetWallDecorationOffset(level, z, x, loc.WallWidth)
				}
			case 3:
				modelSrc := buildModel(3, angle)

				scene.AddWall(0, y, level, ROTATION_WALL_CORNER_TYPE[angle], modelSrc, nil, x, typeCode, z, info)

				if loc.Shadow {
					switch angle {
					case 0:
						w.LevelShadeMap[level][x][z+1] = 50
					case 1:
						w.LevelShadeMap[level][x+1][z+1] = 50
					case 2:
						w.LevelShadeMap[level][x+1][z] = 50
					case 3:
						w.LevelShadeMap[level][x][z] = 50
					}
				}

				if loc.BlockWalk && collision != nil {
					collision.AddWall(angle, z, x, loc.BlockRange, shape)
				}
			case 9:
				modelSrc := buildModel(shape, angle)

				scene.AddLoc1(y, level, typeCode, z, x, 1, info, modelSrc, 0, 1)

				if loc.BlockWalk && collision != nil {
					collision.AddLoc(angle, loc.Length, loc.Width, x, z, loc.BlockRange)
				}
			case 4:
				modelSrc := buildModel(4, 0)

				scene.SetWallDecoration(y, z, 0, typeCode, angle*512, ROTATION_WALL_TYPE[angle], 0, x, modelSrc, info, level)
			case 5:
				wallWidth := 16

				wallType := scene.GetWallBitSet(level, x, z)
				if wallType > 0 {
					wallWidth = loctype.Get((wallType >> 14) & 0x7FFF).WallWidth
				}

				modelSrc := buildModel(4, 0)

				scene.SetWallDecoration(y, z, WALL_DECORATION_ROTATION_FORWARD_Z[angle]*wallWidth, typeCode, angle*512, ROTATION_WALL_TYPE[angle], WALL_DECORATION_ROTATION_FORWARD_X[angle]*wallWidth, x, modelSrc, info, level)
			case 6:
				modelSrc := buildModel(4, 0)

				scene.SetWallDecoration(y, z, 0, typeCode, angle, 256, 0, x, modelSrc, info, level)
			case 7:
				modelSrc := buildModel(4, 0)

				scene.SetWallDecoration(y, z, 0, typeCode, angle, 512, 0, x, modelSrc, info, level)
			case 8:
				modelSrc := buildModel(4, 0)

				scene.SetWallDecoration(y, z, 0, typeCode, angle, 768, 0, x, modelSrc, info, level)
			}
		}
	} else if !LowMemory || loc.Active || loc.ForceDecor {
		modelSrc := buildModel(22, angle)
		scene.AddGroundDecoration(modelSrc, x, typeCode, z, level, info, y)
		if loc.BlockWalk && loc.Active && collision != nil {
			collision.SetBlocked(z, x)
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
	RandomHueOffset = max(RandomHueOffset, -8)
	RandomHueOffset = min(RandomHueOffset, 8)
	RandomLightnessOffset += int(rand.Float64()*5.0) - 2
	RandomLightnessOffset = max(RandomLightnessOffset, -16)
	RandomLightnessOffset = min(RandomLightnessOffset, 16)
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
		var13 = (var8 * var12) >> 8
		for j := 1; j < w.MaxTileZ-1; j++ {
			for k := 1; k < w.MaxTileX-1; k++ {
				var16 = w.LevelHeightMap[i][k+1][j] - w.LevelHeightMap[i][k-1][j]
				var17 = w.LevelHeightMap[i][k][j+1] - w.LevelHeightMap[i][k][j-1]
				var18 = int(math.Sqrt(float64(var16*var16 + 65536 + var17*var17)))
				var19 = (var16 << 8) / var18
				var20 = 65536 / var18
				var21 = (var17 << 8) / var18
				var22 = var46 + (var9*var19+var10*var20+var11*var21)/var13
				// Java: World.java:543 — `(var45[k-1][j] >> 2) + ...`. Java `var45` is
				// `byte[][]`; each element is promoted byte->int (sign-extending) before the
				// arithmetic `>>`. LevelShadeMap is now `[][]int8`, so int(...) per term
				// reproduces that byte->int promotion exactly (values are 0..50, so the
				// result is identical, but this now matches Java's width and signedness).
				var23 = (int(var45[k-1][j]) >> 2) + (int(var45[k+1][j]) >> 3) + (int(var45[k][j-1]) >> 2) + (int(var45[k][j+1]) >> 3) + (int(var45[k][j]) >> 1)
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
					var19 = int(w.LevelTileUnderlayIDs[i][var18][k]) & 0xFF
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
					var20 = int(w.LevelTileUnderlayIDs[i][var19][k]) & 0xFF
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
						var26 := int(w.LevelTileUnderlayIDs[i][j][l]) & 0xFF
						var27 := int(w.LevelTileOverlayIDs[i][j][l]) & 0xFF
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
								var54 := (var38 + RandomHueOffset) & 0xFF
								var40 += RandomLightnessOffset
								if var40 < 0 {
									var40 = 0
								} else if var40 > 0xFF {
									var40 = 0xFF
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
								} else if var41.RGB == 0xFF00FF {
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
	var2 := InterpolatedNoise(arg0+45365, arg1+91923, 4) - 128 + ((InterpolatedNoise(arg0+10294, arg1+37821, 2) - 128) >> 1) + ((InterpolatedNoise(arg0, arg1, 1) - 128) >> 2)
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
	var4 := arg0 & (arg2 - 1)
	var5 := arg1 / arg2
	var6 := arg1 & (arg2 - 1)
	var7 := SmoothNoise(var3, var5)
	var8 := SmoothNoise(var3+1, var5)
	var9 := SmoothNoise(var3, var5+1)
	var10 := SmoothNoise(var3+1, var5+1)
	var11 := Interpolate(var7, var8, var4, arg2)
	var12 := Interpolate(var9, var10, var4, arg2)
	return Interpolate(var11, var12, var6, arg2)
}

func Interpolate(arg0, arg1, arg2, arg3 int) int {
	var4 := (65536 - pix3d.CosTable[arg2*0x400/arg3]) >> 1
	return ((arg0 * (65536 - var4)) >> 16) + ((arg1 * var4) >> 16)
}

func SmoothNoise(arg0, arg1 int) int {
	var2 := Noise(arg0-1, arg1-1) + Noise(arg0+1, arg1-1) + Noise(arg0-1, arg1+1) + Noise(arg0+1, arg1+1)
	var3 := Noise(arg0-1, arg1) + Noise(arg0+1, arg1) + Noise(arg0, arg1-1) + Noise(arg0, arg1+1)
	var4 := Noise(arg0, arg1)
	return var2/16 + var3/8 + var4/4
}

func Noise(arg0, arg1 int) int {
	var2 := arg0 + arg1*57
	var4 := (var2 << 13) ^ var2
	var3 := (var4*(var4*var4*15731+789221) + 1376312589) & math.MaxInt32
	return (var3 >> 19) & 0xFF
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
	return ((arg0 / 4) << 10) + ((arg1 / 32) << 7) + arg2/2
}

// ChangeLocAvailable reports whether the LocType id has the model required to
// render at the given shape, normalizing shape variants first.
//
// Java: World.changeLocAvailable (World.java:1096-1105).
func ChangeLocAvailable(id, shape int) bool {
	loc := loctype.Get(id)
	if shape == 11 {
		shape = 10
	}
	if shape >= 5 && shape <= 8 {
		shape = 4
	}
	return loc.CheckModel(shape)
}

func AddLoc(x int, collision *dash3d.CollisionMap, z int, angle int, heightMap [][][]int, arg7 int, arg8 int, shape int, scene *world3d.World3D, level int) {
	heightSW := heightMap[level][x][z]
	heightSE := heightMap[level][x+1][z]
	heightNE := heightMap[level][x+1][z+1]
	heightNW := heightMap[level][x][z+1]

	y := (heightSW + heightSE + heightNE + heightNW) >> 2

	locType := loctype.Get(arg8)

	var18 := x + (z << 7) + (arg8 << 14) + 1073741824
	if !locType.Active {
		var18 += math.MinInt32
	}

	var19 := byte((angle << 6) + shape)

	// See World.AddLoc.buildModel — a static loc model, or a self-animating
	// ClientLocAnim when this loc has an anim. Java: rev-244 World.addLoc (static).
	buildModel := func(shape, angle int) entity.ModelSource {
		if locType.Anim == -1 {
			return entity.ModelSourceOf(locType.GetModel(shape, angle, heightSW, heightSE, heightNE, heightNW, -1))
		}
		return entity.NewClientLocAnim(heightNW, heightNE, heightSW, shape, angle, true, heightSE, arg8, locType.Anim)
	}

	if shape == 22 {
		modelSrc := buildModel(22, angle)
		scene.AddGroundDecoration(modelSrc, x, var18, z, arg7, var19, y)
		if locType.BlockWalk && locType.Active {
			collision.SetBlocked(z, x)
		}
		return
	}
	var21 := 0
	if shape == 10 || shape == 11 {
		modelSrc := buildModel(10, angle)
		if modelSrc != nil {
			var23 := 0
			if shape == 11 {
				var23 += 256
			}
			var22 := 0
			if angle == 1 || angle == 3 {
				var21 = locType.Length
				var22 = locType.Width
			} else {
				var21 = locType.Width
				var22 = locType.Length
			}
			scene.AddLoc1(y, arg7, var18, z, x, var21, var19, modelSrc, var23, var22)
		}
		if locType.BlockWalk {
			collision.AddLoc(angle, locType.Length, locType.Width, x, z, locType.BlockRange)
		}
	} else if shape >= 12 {
		modelSrc := buildModel(shape, angle)
		scene.AddLoc1(y, arg7, var18, z, x, 1, var19, modelSrc, 0, 1)
		if locType.BlockWalk {
			collision.AddLoc(angle, locType.Length, locType.Width, x, z, locType.BlockRange)
		}
	} else if shape == 0 {
		modelSrc := buildModel(0, angle)
		scene.AddWall(0, y, arg7, ROTATION_WALL_TYPE[angle], modelSrc, nil, x, var18, z, var19)
		if locType.BlockWalk {
			collision.AddWall(angle, z, x, locType.BlockRange, shape)
		}
	} else if shape == 1 {
		modelSrc := buildModel(1, angle)
		scene.AddWall(0, y, arg7, ROTATION_WALL_CORNER_TYPE[angle], modelSrc, nil, x, var18, z, var19)
		if locType.BlockWalk {
			collision.AddWall(angle, z, x, locType.BlockRange, shape)
		}
	} else {
		var24 := 0
		switch shape {
		case 2:
			var24 = (angle + 1) & 0x3
			modelSrc1 := buildModel(2, angle+4)
			modelSrc2 := buildModel(2, var24)
			scene.AddWall(ROTATION_WALL_TYPE[var24], y, arg7, ROTATION_WALL_TYPE[angle], modelSrc1, modelSrc2, x, var18, z, var19)
			if locType.BlockWalk {
				collision.AddWall(angle, z, x, locType.BlockRange, shape)
			}
		case 3:
			modelSrc := buildModel(3, angle)
			scene.AddWall(0, y, arg7, ROTATION_WALL_CORNER_TYPE[angle], modelSrc, nil, x, var18, z, var19)
			if locType.BlockWalk {
				collision.AddWall(angle, z, x, locType.BlockRange, shape)
			}
		case 9:
			modelSrc := buildModel(shape, angle)
			scene.AddLoc1(y, arg7, var18, z, x, 1, var19, modelSrc, 0, 1)
			if locType.BlockWalk {
				collision.AddLoc(angle, locType.Length, locType.Width, x, z, locType.BlockRange)
			}
		case 4:
			modelSrc := buildModel(4, 0)
			scene.SetWallDecoration(y, z, 0, var18, angle*512, ROTATION_WALL_TYPE[angle], 0, x, modelSrc, var19, arg7)
		case 5:
			var24 = 16
			var21 = scene.GetWallBitSet(arg7, x, z)
			if var21 > 0 {
				var24 = loctype.Get((var21 >> 14) & 0x7FFF).WallWidth
			}
			modelSrc := buildModel(4, 0)
			scene.SetWallDecoration(y, z, WALL_DECORATION_ROTATION_FORWARD_Z[angle]*var24, var18, angle*512, ROTATION_WALL_TYPE[angle], WALL_DECORATION_ROTATION_FORWARD_X[angle]*var24, x, modelSrc, var19, arg7)
		case 6:
			modelSrc := buildModel(4, 0)
			scene.SetWallDecoration(y, z, 0, var18, angle, 256, 0, x, modelSrc, var19, arg7)
		case 7:
			modelSrc := buildModel(4, 0)
			scene.SetWallDecoration(y, z, 0, var18, angle, 512, 0, x, modelSrc, var19, arg7)
		case 8:
			modelSrc := buildModel(4, 0)
			scene.SetWallDecoration(y, z, 0, var18, angle, 768, 0, x, modelSrc, var19, arg7)
		}
	}
}

// CheckLocations checks whether all models needed to render the locs encoded
// in src are loaded, requesting any missing models as a side effect.
// Returns true when every visible loc model is ready.
// Java: World.checkLocations(int xOffset, int zOffset, byte[] src) (c.a(II[B)Z), lines 197-246.
// Called by the WS2 scene-readiness check (Java: Client.java:3277).
func CheckLocations(xOffset, zOffset int, src []byte) bool {
	ready := true
	buf := io.NewPacket(src)
	locId := -1

outer:
	for {
		deltaId := buf.GSmartS()
		if deltaId == 0 {
			return ready
		}
		locId += deltaId

		locPos := 0
		// skip: set once a visible loc for this position has been model-checked; the inner loop then drains the remaining position deltas.
		skip := false
		for {
			for !skip {
				deltaPos := buf.GSmartS()
				if deltaPos == 0 {
					continue outer // Java: continue label54
				}
				locPos += deltaPos - 1

				z := locPos & 0x3F
				x := locPos >> 6 & 0x3F

				shape := buf.G1() >> 2
				stx := xOffset + x
				stz := zOffset + z

				if stx > 0 && stz > 0 && stx < 103 && stz < 103 {
					loc := loctype.Get(locId)
					if shape != 22 || !LowMemory || loc.Active || loc.ForceDecor {
						// Java: ready &= loc.checkModelAll() — non-short-circuit bitwise AND.
						// CheckModelAll always invokes model.Request for every model (side effect).
						if !loc.CheckModelAll() {
							ready = false
						}
						skip = true
					}
				}
			}

			deltaPos := buf.GSmartS()
			if deltaPos == 0 {
				break
			}
			buf.G1()
		}
	}
}

// PrefetchLocations queues model prefetches for all locs encoded in buf.
// Java: World.prefetchLocations(Packet buf, OnDemand od) (c.a(ILmb;Lvb;)V), lines 248-270.
func PrefetchLocations(buf *io.Packet, od *ondemand.OnDemand) {
	locId := -1
	for {
		deltaId := buf.GSmartS()
		if deltaId == 0 {
			return
		}
		locId += deltaId

		loc := loctype.Get(locId)
		loc.Prefetch(od)

		for {
			deltaPos := buf.GSmartS()
			if deltaPos == 0 {
				break
			}
			buf.G1()
		}
	}
}
