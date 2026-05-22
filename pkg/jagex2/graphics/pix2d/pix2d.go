package pix2d

var (
	Data      []int
	Width2D   int
	Height2D  int
	Top       int // was BoundTop
	Bottom    int // was BoundBottom
	Left      int // was BoundLeft
	Right     int // was BoundRight
	SafeWidth int
	CenterW2D int
	CenterH2D int
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

// Reset clears every package-level binding to its zero value. Intended for
// tests that need to start from a clean slate so a previous test's Bind
// can't leak into the next (the rendering pipeline keeps its state as
// package vars by design — see CLAUDE.md "Global State Pattern").
func Reset() {
	Data = nil
	Width2D = 0
	Height2D = 0
	Top = 0
	Bottom = 0
	Left = 0
	Right = 0
	SafeWidth = 0
	CenterW2D = 0
	CenterH2D = 0
}

func ResetClipping() {
	Left = 0
	Top = 0
	Right = Width2D
	Bottom = Height2D
	SafeWidth = Right - 1
	CenterW2D = Right / 2
}

func SetClipping(bottom int, top int, right int, left int) {
	left = max(left, 0)
	top = max(top, 0)
	right = min(right, Width2D)
	bottom = min(bottom, Height2D)
	Left = left
	Top = top
	Right = right
	Bottom = bottom
	SafeWidth = Right - 1
	CenterW2D = Right / 2
	CenterH2D = Bottom / 2
}

func Clear() {
	length := Width2D * Height2D
	for i := range length {
		Data[i] = 0
	}
}

func FillRect(y int, x int, colour int, width int, height int) {
	if x < Left {
		width -= Left - x
		x = Left
	}
	if y < Top {
		height -= Top - y
		y = Top
	}
	if x+width > Right {
		width = Right - x
	}
	if y+height > Bottom {
		height = Bottom - y
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

func DrawRect(x int, hexColour int, height int, y int, width int) {
	HLine(hexColour, y, width, x)
	HLine(hexColour, y+height-1, width, x)
	VLine(hexColour, y, height, x)
	VLine(hexColour, y, height, x+width-1)
}

func HLine(colour int, y int, width int, x int) {
	if y < Top || y >= Bottom {
		return
	}
	if x < Left {
		width -= Left - x
		x = Left
	}
	if x+width > Right {
		width = Right - x
	}
	offset := x + y*Width2D
	for i := range width {
		Data[offset+i] = colour
	}
}

func VLine(colour int, y int, height int, x int) {
	if x < Left || x >= Right {
		return
	}
	if y < Top {
		height -= Top - y
		y = Top
	}
	if y+height > Bottom {
		height = Bottom - y
	}
	offset := x + y*Width2D
	for i := range height {
		Data[offset+i*Width2D] = colour
	}
}
