package client

import (
	"log"
	"os"
	"time"

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

	// MINE
	Ops op.Ops // Ops are the operations from the UI
}

func NewGameShell() *GameShell {
	return &GameShell{
		DelTime:   20,
		MinDel:    1,
		OTim:      make([]int64, 10),
		Refresh:   true,
		ActionKey: make([]int, 128),
		KeyQueue:  make([]int, 128),
	}
}

func (g *GameShell) InitApplication(screenHeight int, screenWidth int) {
	g.ScreenWidth = screenWidth
	g.ScreenHeight = screenHeight
	//g.Frame
	//g.Graphics
	g.DrawArea = pixmap.NewPixMap(screenWidth, screenHeight)
	g.Run()
}

func (g *GameShell) Run() {
	// TODO: listeners
	g.DrawProgress("Loading...", 0)
	// TODO: client.Load()...
	var3 := 0
	var4 := 256
	var5 := 1
	var6 := 0
	for i := range 10 {
		g.OTim[i] = time.Now().UnixMilli()
	}
	var1 := int64(0)
	for g.State >= 0 {
		if g.State > 0 {
			g.State--
			if g.State == 0 {
				g.Shutdown()
				return
			}
		}
		var8 := var4
		var9 := var5
		var4 = 300
		var5 = 1
		var1 = time.Now().UnixMilli()
		if g.OTim[var3] == 0 {
			var4 = var8
			var5 = var9
		} else if var1 > g.OTim[var3] {
			var4 = int(int64(g.DelTime*2560) / (var1 - g.OTim[var3]))
		}
		if var4 < 25 {
			var4 = 25
		}
		if var4 > 256 {
			var4 = 256
			var5 = int(int64(g.DelTime) - (var1-g.OTim[var3])/10)
		}
		g.OTim[var3] = var1
		var3 = (var3 + 1) % 10
		if var5 > 1 {
			for i := range 10 {
				if g.OTim[i] != 0 {
					g.OTim[i] += int64(var5)
				}
			}
		}
		if var5 < g.MinDel {
			var5 = g.MinDel
		}
		time.Sleep(time.Duration(var5) * time.Millisecond)
		for var6 < 256 {
			g.Update()
			g.MouseClickButton = 0
			g.KeyQueueReadPos = g.KeyQueueWritePos
			var6 += var4
		}
		var6 &= 0xFF
		if g.DelTime > 0 {
			g.FPS = var4 * 1000 / (g.DelTime * 256)
		}
		g.Draw()

		// TODO: start mine
		go func() {
			// Create new window
			w := new(app.Window)
			w.Option(app.Title("Jagex"))
			w.Option(app.Size(unit.Dp(g.ScreenWidth), unit.Dp(g.ScreenHeight)))
			w.Option(app.MinSize(unit.Dp(g.ScreenWidth), unit.Dp(g.ScreenHeight)))
			w.Option(app.MaxSize(unit.Dp(g.ScreenWidth), unit.Dp(g.ScreenHeight)))

			if err := g.draw(w); err != nil {
				log.Fatal(err)
			}
			os.Exit(0)
		}()
		app.Main()
		// TODO: end mine
	}
	if g.State == -1 {
		g.Shutdown()
	}

}

func (g *GameShell) draw(w *app.Window) error {
	//// ops are the operations from the UI
	//var ops op.Ops

	// Listen for events in the window
	for {
		// detect what type of event
		switch e := w.Event().(type) {

		case app.FrameEvent:
			// A request to draw the window state
			// This is sent when the application should re-render
			gtx := app.NewContext(&g.Ops, e)

			// Draw the state into ops

			// Update the display
			e.Frame(gtx.Ops)

		case app.DestroyEvent:
			// The window was closed
			return e.Err
		}
	}
}

func (g *GameShell) Shutdown() {
	g.State = -2
	g.Unload()
	time.Sleep(1 * time.Second)
	os.Exit(0)
}

func (g *GameShell) SetFrameRate(arg1 int) {
	g.DelTime = 1000 / arg1
}

func (g *GameShell) PollKey() int {
	return 0 // TODO: stub
}

func (g *GameShell) Update() {

}

func (g *GameShell) Unload() {

}

func (g *GameShell) Draw() {

}

func (g *GameShell) DrawProgress(arg1 string, arg2 int) {
	// TODO: stub
}
