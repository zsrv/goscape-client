module github.com/zsrv/goscape-client

go 1.26

require (
	gioui.org v0.10.0
	github.com/coder/websocket v1.8.14
	github.com/ebitengine/oto/v3 v3.4.0
	github.com/sinshu/go-meltysynth v0.0.0-20230205031334-05d311382fc4
	golang.org/x/image v0.41.0
)

require (
	gioui.org/shader v1.0.8 // indirect
	github.com/ebitengine/purego v0.10.0 // indirect
	github.com/go-gl/gl v0.0.0-20260331235117-4566fea9a276 // indirect
	github.com/go-gl/glfw/v3.3/glfw v0.0.0-20260406072232-3ac4aa2bb164 // indirect
	github.com/go-text/typesetting v0.3.4 // indirect
	golang.org/x/exp/shiny v0.0.0-20260508232706-74f9aab9d74a // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
)

replace gioui.org => ./third_party/gioui.org
