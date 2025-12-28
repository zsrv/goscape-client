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

func Bind(width int, data []int, height int) {
	Data = data
	Width2D = width
	Height2D = height
	SetClipping(height, 0, width, 0)
}

func ResetClipping() {
	BoundLeft = 0
	BoundTop = 0
	BoundRight = Width2D
	BoundBottom = Height2D
	SafeWidth = BoundRight - 1
	CenterW2D = BoundRight / 2
}

func SetClipping(bottom int, top int, right int, left int) {
	if left < 0 {
		left = 0
	}
	if top < 0 {
		top = 0
	}
	if right > Width2D {
		right = Width2D
	}
	if bottom > Height2D {
		bottom = Height2D
	}
	BoundLeft = left
	BoundTop = top
	BoundRight = right
	BoundBottom = bottom
	SafeWidth = BoundRight - 1
	CenterW2D = BoundRight / 2
	CenterH2D = BoundBottom / 2
}

func Clear() {
	length := Width2D * Height2D
	for i := range length {
		Data[i] = 0
	}
}

func FillRect(y int, x int, colour int, width int, height int) {
	if x < BoundLeft {
		width -= BoundLeft - x
		x = BoundLeft
	}
	if y < BoundTop {
		height -= BoundTop - y
		y = BoundTop
	}
	if x+width > BoundRight {
		width = BoundRight - x
	}
	if y+height > BoundBottom {
		height = BoundBottom - y
	}
	step := Width2D - width
	offset := x + y*Width2D
	for i := -height; i < 0; i++ {
		for j := -width; j < 0; j++ {
			Data[offset] = colour
			offset++
		}
		offset += step
	}
}

func DrawRect(x int, colour int, height int, y int, width int) {
	HLine(colour, y, width, x)
	HLine(colour, y+height-1, width, x)
	VLine(colour, y, height, x)
	VLine(colour, y, height, x+width-1)
}

func HLine(colour int, y int, width int, x int) {
	if y < BoundTop || y >= BoundBottom {
		return
	}
	if x < BoundLeft {
		width -= BoundLeft - x
		x = BoundLeft
	}
	if x+width > BoundRight {
		width = BoundRight - x
	}
	offset := x + y*Width2D
	for i := range width {
		Data[offset+i] = colour
	}
}

func VLine(colour int, y int, height int, x int) {
	if x < BoundLeft || x >= BoundRight {
		return
	}
	if y < BoundTop {
		height -= BoundTop - y
		y = BoundTop
	}
	if y+height > BoundBottom {
		height = BoundBottom - y
	}
	offset := x + y*Width2D
	for i := range height {
		Data[offset+i*Width2D] = colour
	}
}
