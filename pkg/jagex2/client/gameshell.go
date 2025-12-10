package client

import (
	"goscape-client/pkg/jagex2/graphics/pixmap"
)

type GameShell struct {
	State        int
	DelTime      int
	MinDel       int
	OTim         []int64
	FPS          int
	ScreenWidth  int
	ScreenHeight int
	//Graphics
	DrawArea         *pixmap.PixMap
	Frame            *ViewBox
	Refresh          bool
	IdleCycles       int
	MouseButton      int
	MouseX           int
	MouseY           int
	MouseClickButton int
	MouseClickX      int
	MouseClickY      int
	ActionKey        []int
	KeyQueue         []int
	KeyQueueReadPos  int
	KeyQueueWritePos int
}

func (g *GameShell) InitApplication(screenHeight int, screenWidth int) {
	g.ScreenWidth = screenWidth
	g.ScreenHeight = screenHeight
	//g.Frame
	//g.Graphics
	//g.DrawArea
	// TODO: go GameShell.run()
}

func (g *GameShell) Run() {

}
