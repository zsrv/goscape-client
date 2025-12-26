package typ

var (
	TmpScreenX    []int   = make([]int, 6)
	TmpScreenY    []int   = make([]int, 6)
	TmpViewSpaceX []int   = make([]int, 6)
	TmpViewSpaceY []int   = make([]int, 6)
	TmpViewSpaceZ []int   = make([]int, 6)
	SHAPE_POINTS  [][]int = [][]int{{1, 3, 5, 7}, {1, 3, 5, 7}, {1, 3, 5, 7}, {1, 3, 5, 7, 6}, {1, 3, 5, 7, 6}, {1, 3, 5, 7, 6}, {1, 3, 5, 7, 6}, {1, 3, 5, 7, 2, 6}, {1, 3, 5, 7, 2, 8}, {1, 3, 5, 7, 2, 8}, {1, 3, 5, 7, 11, 12}, {1, 3, 5, 7, 11, 12}, {1, 3, 5, 7, 13, 14}}
	SHAPE_PATHS   [][]int = [][]int{{0, 1, 2, 3, 0, 0, 1, 3}, {1, 1, 2, 3, 1, 0, 1, 3}, {0, 1, 2, 3, 1, 0, 1, 3}, {0, 0, 1, 2, 0, 0, 2, 4, 1, 0, 4, 3}, {0, 0, 1, 4, 0, 0, 4, 3, 1, 1, 2, 4}, {0, 0, 4, 3, 1, 0, 1, 2, 1, 0, 2, 4}, {0, 1, 2, 4, 1, 0, 1, 4, 1, 0, 4, 3}, {0, 4, 1, 2, 0, 4, 2, 5, 1, 0, 4, 5, 1, 0, 5, 3}, {0, 4, 1, 2, 0, 4, 2, 3, 0, 4, 3, 5, 1, 0, 4, 5}, {0, 0, 4, 5, 1, 4, 1, 2, 1, 4, 2, 3, 1, 4, 3, 5}, {0, 0, 1, 5, 0, 1, 4, 5, 0, 1, 2, 4, 1, 0, 5, 3, 1, 5, 4, 3, 1, 4, 2, 3}, {1, 0, 1, 5, 1, 1, 4, 5, 1, 1, 2, 4, 0, 0, 5, 3, 0, 5, 4, 3, 0, 4, 2, 3}, {1, 0, 5, 4, 1, 0, 1, 5, 0, 0, 4, 3, 0, 4, 5, 3, 0, 5, 2, 3, 0, 1, 2, 5}}
)

type TileOverlay struct {
	VertexX            []int
	VertexY            []int
	VertexZ            []int
	TriangleColorA     []int
	TriangleColorB     []int
	TriangleColorC     []int
	TriangleVertexA    []int
	TriangleVertexB    []int
	TriangleVertexC    []int
	TriangleTextureIDs []int
	Flat               bool
	Shape              int
	Rotation           int
	BackgroundRGB      int
	ForegroundRGB      int
}

func NewTileOverlay(arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7, arg8, arg9, arg10, arg11, arg12, arg13, arg14, arg15, arg17, arg18, arg19 int) *TileOverlay {
	var t TileOverlay
	t.Flat = true

	if arg17 != arg3 || arg17 != arg13 || arg17 != arg7 {
		t.Flat = false
	}
	t.Shape = arg1
	t.Rotation = arg5
	t.BackgroundRGB = arg12
	t.ForegroundRGB = arg8
	var21 := 128
	var22 := var21 / 2
	var23 := var21 / 4
	var24 := var21 * 3 / 4
	var25 := SHAPE_POINTS[arg1]
	var26 := len(var25)
	t.VertexX = make([]int, var26)
	t.VertexY = make([]int, var26)
	t.VertexZ = make([]int, var26)
	var27 := make([]int, var26)
	var28 := make([]int, var26)
	var29 := arg0 * var21
	var30 := arg18 * var21
	var33 := 0
	var34 := 0
	var35 := 0
	var36 := 0
	var37 := 0
	for i := range var26 {
		var32 := var25[i]
		if var32&0x1 == 0 && var32 <= 8 {
			var32 = ((var32 - arg5 - arg5 - 1) & 0x7) + 1
		}
		if var32 > 8 && var32 <= 12 {
			var32 = ((var32 - 9 - arg5) & 0x3) + 9
		}
		if var32 > 12 && var32 <= 16 {
			var32 = ((var32 - 13 - arg5) & 0x3) + 13
		}
		switch var32 {
		case 1:
			var33 = var29
			var34 = var30
			var35 = arg17
			var36 = arg6
			var37 = arg9
		case 2:
			var33 = var29 + var22
			var34 = var30
			var35 = (arg17 + arg3) >> 1
			var36 = (arg6 + arg19) >> 1
			var37 = (arg9 + arg2) >> 1
		case 3:
			var33 = var29 + var21
			var34 = var30
			var35 = arg3
			var36 = arg19
			var37 = arg2
		case 4:
			var33 = var29 + var21
			var34 = var30 + var22
			var35 = (arg3 + arg13) >> 1
			var36 = (arg19 + arg4) >> 1
			var37 = (arg2 + arg14) >> 1
		case 5:
			var33 = var29 + var21
			var34 = var30 + var21
			var35 = arg13
			var36 = arg4
			var37 = arg14
		case 6:
			var33 = var29 + var22
			var34 = var30 + var21
			var35 = (arg13 + arg7) >> 1
			var36 = (arg4 + arg15) >> 1
			var37 = (arg14 + arg11) >> 1
		case 7:
			var33 = var29
			var34 = var30 + var21
			var35 = arg7
			var36 = arg15
			var37 = arg11
		case 8:
			var33 = var29
			var34 = var30 + var22
			var35 = (arg7 + arg17) >> 1
			var36 = (arg15 + arg6) >> 1
			var37 = (arg11 + arg9) >> 1
		case 9:
			var33 = var29 + var22
			var34 = var30 + var23
			var35 = (arg17 + arg3) >> 1
			var36 = (arg6 + arg19) >> 1
			var37 = (arg9 + arg2) >> 1
		case 10:
			var33 = var29 + var24
			var34 = var30 + var22
			var35 = (arg3 + arg13) >> 1
			var36 = (arg19 + arg4) >> 1
			var37 = (arg2 + arg14) >> 1
		case 11:
			var33 = var29 + var22
			var34 = var30 + var24
			var35 = (arg13 + arg7) >> 1
			var36 = (arg4 + arg15) >> 1
			var37 = (arg14 + arg11) >> 1
		case 12:
			var33 = var29 + var23
			var34 = var30 + var22
			var35 = (arg7 + arg17) >> 1
			var36 = (arg15 + arg6) >> 1
			var37 = (arg11 + arg9) >> 1
		case 13:
			var33 = var29 + var23
			var34 = var30 + var23
			var35 = arg17
			var36 = arg6
			var37 = arg9
		case 14:
			var33 = var29 + var24
			var34 = var30 + var23
			var35 = arg3
			var36 = arg19
			var37 = arg2
		case 15:
			var33 = var29 + var24
			var34 = var30 + var24
			var35 = arg13
			var36 = arg4
			var37 = arg14
		default:
			var33 = var29 + var23
			var34 = var30 + var24
			var35 = arg7
			var36 = arg15
			var37 = arg11
		}
		t.VertexX[i] = var33
		t.VertexY[i] = var35
		t.VertexZ[i] = var34
		var27[i] = var36
		var28[i] = var37
	}
	var40 := SHAPE_PATHS[arg1]
	var33 = len(var40) / 4
	t.TriangleVertexA = make([]int, var33)
	t.TriangleVertexB = make([]int, var33)
	t.TriangleVertexC = make([]int, var33)
	t.TriangleColorA = make([]int, var33)
	t.TriangleColorB = make([]int, var33)
	t.TriangleColorC = make([]int, var33)
	if arg10 != -1 {
		t.TriangleTextureIDs = make([]int, var33)
	}
	var34 = 0
	for i := range var33 {
		var36 = var40[var34]
		var37 = var40[var34+1]
		var38 := var40[var34+2]
		var39 := var40[var34+3]
		var34 += 4
		if var37 < 4 {
			var37 = (var37 - arg5) & 0x3
		}
		if var38 < 4 {
			var38 = (var38 - arg5) & 0x3
		}
		if var39 < 4 {
			var39 = (var39 - arg5) & 0x3
		}
		t.TriangleVertexA[i] = var37
		t.TriangleVertexB[i] = var38
		t.TriangleVertexC[i] = var39
		if var36 == 0 {
			t.TriangleColorA[i] = var27[var37]
			t.TriangleColorB[i] = var27[var38]
			t.TriangleColorC[i] = var27[var39]
			if t.TriangleTextureIDs != nil {
				t.TriangleTextureIDs[i] = -1
			}
		} else {
			t.TriangleColorA[i] = var28[var37]
			t.TriangleColorB[i] = var28[var38]
			t.TriangleColorC[i] = var28[var39]
			if t.TriangleTextureIDs != nil {
				t.TriangleTextureIDs[i] = arg10
			}
		}
	}
	var36 = arg17
	var37 = arg3
	if arg3 < arg17 {
		var36 = arg3
	}
	if arg3 > arg3 {
		var37 = arg3
	}
	if arg13 < var36 {
		var36 = arg13
	}
	if arg13 > var37 {
		var37 = arg13
	}
	if arg7 < var36 {
		var36 = arg7
	}
	if arg7 > var37 {
		var37 = arg7
	}
	var36 /= 14
	var37 /= 14

	return &t
}
