package client

import (
	"log"
	"os"
	"time"

	"gioui.org/app"
	"gioui.org/unit"

	"goscape-client/pkg/jagex2/graphics/pixmap"
)

type GameShell struct {
}

func (c *Client) InitApplication(height int, width int) {
	c.ScreenWidth = width
	c.ScreenHeight = height
	//c.Frame // app.Window with gio
	//c.Graphics
	c.DrawArea = pixmap.NewPixMap(width, height) // TODO: component is linked to this in java.. automatically draws stuff?
	// TODO: open the window here, before Run()
	// TODO: start mine
	go func() {
		// Create new window
		w := new(app.Window)
		w.Option(app.Title("Jagex"))
		w.Option(app.Size(unit.Dp(c.ScreenWidth), unit.Dp(c.ScreenHeight)))
		w.Option(app.MinSize(unit.Dp(c.ScreenWidth), unit.Dp(c.ScreenHeight)))
		w.Option(app.MaxSize(unit.Dp(c.ScreenWidth), unit.Dp(c.ScreenHeight)))

		if err := c.draw(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	go app.Main() // TODO: go?
	// TODO: end mine
	c.Run()
}

func (c *Client) draw(w *app.Window) error {
	// for every frame, clear the screen and redraw the whole layout

	//// ops are the operations from the UI
	//var ops op.Ops

	// Listen for events in the window
	for {
		// detect what type of event
		switch e := w.Event().(type) {

		case app.DestroyEvent:
			// The window was closed
			return e.Err

		case app.FrameEvent:
			// A request to draw the window state.
			// Gio only issues FrameEvents when the window is resized or
			// the user interacts with the window.
			//
			// When the program receives a FrameEvent, it is responsible
			// for updating the display by calling the e.Frame function with
			// an operation list representing the new state.

			// This is sent when the application should re-render.
			// Resets the layout.Context for a new frame.
			//gtx := app.NewContext(&c.Ops, e)
			//gtx := app.NewContext(&ops, e)

			// PlotSprite the state into ops based on events in e.Queue.
			//draw(gtx)
			//draw(&c.Ops)
			//gtx.Execute(op.InvalidateCmd{})

			// Update the display - pass the operations list to the window driver
			//e.Frame(gtx.Ops)
			e.Frame(&c.Ops)
			w.Invalidate()
		}
	}
}

func (c *Client) RunGameShell() {
	// TODO: listeners

	c.DrawProgress("Loading...", 0)
	c.Load()

	var3 := 0
	var4 := 256
	var5 := 1
	var6 := 0
	for i := range 10 {
		c.OTim[i] = time.Now().UnixMilli()
	}
	var1 := int64(0)
	for c.State >= 0 {
		if c.State > 0 {
			c.State--
			if c.State == 0 {
				c.Shutdown()
				return
			}
		}
		var8 := var4
		var9 := var5
		var4 = 300
		var5 = 1
		var1 = time.Now().UnixMilli()
		if c.OTim[var3] == 0 {
			var4 = var8
			var5 = var9
		} else if var1 > c.OTim[var3] {
			var4 = int(int64(c.DelTime*2560) / (var1 - c.OTim[var3]))
		}
		if var4 < 25 {
			var4 = 25
		}
		if var4 > 256 {
			var4 = 256
			var5 = int(int64(c.DelTime) - (var1-c.OTim[var3])/10)
		}
		c.OTim[var3] = var1
		var3 = (var3 + 1) % 10
		if var5 > 1 {
			for i := range 10 {
				if c.OTim[i] != 0 {
					c.OTim[i] += int64(var5)
				}
			}
		}
		if var5 < c.MinDel {
			var5 = c.MinDel
		}
		time.Sleep(time.Duration(var5) * time.Millisecond)
		for var6 < 256 {
			c.Update()
			c.MouseClickButton = 0
			c.KeyQueueReadPos = c.KeyQueueWritePos
			var6 += var4
		}
		var6 &= 0xFF
		if c.DelTime > 0 {
			c.FPS = var4 * 1000 / (c.DelTime * 256)
		}
		c.Draw()
	}
	if c.State == -1 {
		c.Shutdown()
	}

}

func (c *Client) Shutdown() {
	c.State = -2
	c.Unload()
	time.Sleep(1 * time.Second)
	os.Exit(0)
}

func (c *Client) SetFrameRate(arg1 int) {
	c.DelTime = 1000 / arg1
}

func (c *Client) PollKey() int {
	return 0 // TODO: stub
}

func (c *Client) DrawProgressGameShell(message string, percent int) {
	// TODO: stub
}
