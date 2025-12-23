package client

import (
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/op"
	"gioui.org/unit"

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

	go func() {
		// Create new window
		w := new(app.Window)
		w.Option(app.Title("Jagex"))
		w.Option(app.Size(unit.Dp(screenWidth), unit.Dp(screenHeight)))
		w.Option(app.MinSize(unit.Dp(screenWidth), unit.Dp(screenHeight)))
		w.Option(app.MaxSize(unit.Dp(screenWidth), unit.Dp(screenHeight)))

		if err := g.draw(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func (g *GameShell) draw(w *app.Window) error {
	// ops are the operations from the UI
	var ops op.Ops

	// Listen for events in the window
	for {
		// detect what type of event
		switch e := w.Event().(type) {

		case app.FrameEvent:
			// A request to draw the window state
			// This is sent when the application should re-render
			gtx := app.NewContext(&ops, e)

			// Draw the state into ops

			// Update the display
			e.Frame(gtx.Ops)

		case app.DestroyEvent:
			// The window was closed
			return e.Err
		}
	}
}

func (g *GameShell) Run() {

}

func (g *GameShell) PollKey() int {
	return 0 // TODO: stub
}

func (g *GameShell) DrawProgress(arg1 string, arg2 int) {
	// TODO: stub
}
