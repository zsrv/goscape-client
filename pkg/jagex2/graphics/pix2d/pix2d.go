package pix2d

var (
	Data        []int
	Width2D     int
	Height2D    int
	BoundTop    int
	BoundBottom int
	BoundLeft   int
	BoundRight  int
	SafeWidth   int
	CenterW2D   int
	CenterH2D   int
)

//type Pix2D struct {
//	datastruct.DoublyLinkable[Pix2D]
//}

func Bind(width2d int, data []int, height2d int) {
	Data = data
	Width2D = width2d
	Height2D = height2d
	SetClipping(height2d, 0, width2d, 0)
}

func ResetClipping() {
	BoundLeft = 0
	BoundTop = 0
	BoundRight = Width2D
	BoundBottom = Height2D
	SafeWidth = BoundRight - 1
	CenterW2D = BoundRight / 2
}

func SetClipping(height2d int, boundTop int, width2d int, boundLeft int) {
	if boundLeft < 0 {
		boundLeft = 0
	}
	if boundTop < 0 {
		boundTop = 0
	}
	if width2d > Width2D {
		width2d = Width2D
	}
	if height2d > Height2D {
		height2d = Height2D
	}
	BoundLeft = boundLeft
	BoundTop = boundTop
	BoundRight = width2d
	BoundBottom = height2d
	SafeWidth = BoundRight - 1
	CenterW2D = BoundRight / 2
	CenterH2D = BoundBottom / 2
}

func Clear() {
	var1 := Width2D * Height2D
	for i := range var1 {
		Data[i] = 0
	}
}

func FillRect(arg0 int, arg1 int, arg2 int, arg4 int, arg5 int) {
	if arg1 < BoundLeft {
		arg4 -= BoundLeft - arg1
		arg1 = BoundLeft
	}
	if arg0 < BoundTop {
		arg5 -= BoundTop - arg0
		arg0 = BoundTop
	}
	if arg1+arg4 > BoundRight {
		arg4 = BoundRight - arg1
	}
	if arg0+arg5 > BoundBottom {
		arg5 = BoundBottom - arg0
	}
	var6 := Width2D - arg4
	var7 := arg1 + arg0*Width2D
	for i := -arg5; i < 0; i++ {
		for j := -arg4; j < 0; j++ {
			Data[var7] = arg2
			var7++
		}
		var7 += var6
	}
}

func DrawRect(arg1 int, arg2 int, arg3 int, arg4 int, arg5 int) {
	HLine(arg2, arg4, arg5, arg1)
	HLine(arg2, arg4+arg3-1, arg5, arg1)
	VLine(arg2, arg4, arg3, arg1)
	VLine(arg2, arg4, arg3, arg1+arg5-1)
}

func HLine(arg0 int, arg2 int, arg3 int, arg4 int) {
	if arg2 < BoundTop || arg2 >= BoundBottom {
		return
	}
	if arg4 < BoundLeft {
		arg3 -= BoundLeft - arg4
		arg4 = BoundLeft
	}
	if arg4+arg3 > BoundRight {
		arg3 = BoundRight - arg4
	}
	var5 := arg4 + arg2*Width2D
	for i := range arg3 {
		Data[var5+i] = arg0
	}
}

func VLine(arg0 int, arg2 int, arg3 int, arg4 int) {
	if arg4 < BoundLeft || arg4 >= BoundRight {
		return
	}
	if arg2 < BoundTop {
		arg3 -= BoundTop - arg2
		arg2 = BoundTop
	}
	if arg2+arg3 > BoundBottom {
		arg3 = BoundBottom - arg2
	}
	var5 := arg4 + arg2*Width2D
	for i := range arg3 {
		Data[var5+i*Width2D] = arg0
	}
}
