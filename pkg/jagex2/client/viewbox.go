package client

type ViewBox struct {
	Shell *GameShell
}

func NewViewBox(arg0 int, arg2 *GameShell, arg3 int) *ViewBox {
	var v ViewBox
	v.Shell = arg2
	// TODO
	return &v
}

// TODO
