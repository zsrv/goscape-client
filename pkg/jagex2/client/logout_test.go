package client

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/world"
)

// TestLogout_ResetsInGameAndTitleState pins the post-fix behavior at
// client.go:3420 — that Logout transitions the client out of the
// in-game render dispatch and back to title-screen rendering. Java
// `deob/client.java:3963` sets `this.ingame = false`; the prior Go
// port omitted this assignment, so `UpdateGame`'s `if !c.InGame { return }`
// guard at client.go:6818 (and the in-game render branches) never
// fired and the title screen never reappeared after logout.
//
// We populate just enough state for Logout to complete without
// panicking (Scene and LevelCollisionMap entries — the rest of
// Logout's path uses package-level globals that are init()-allocated).
func TestLogout_ResetsInGameAndTitleState(t *testing.T) {
	c := NewClient()
	c.Scene = &world.World{}
	for i := range 4 {
		c.LevelCollisionMap[i] = dash3d.NewCollisionMap(0, 0)
	}
	c.InGame = true
	c.TitleScreenState = 2

	c.Logout()

	if c.InGame {
		t.Errorf("InGame = true after Logout; want false — deob/client.java:3963 sets ingame=false and the Go port had been missing it")
	}
	if c.TitleScreenState != 0 {
		t.Errorf("TitleScreenState = %d after Logout; want 0", c.TitleScreenState)
	}
	if c.Stream != nil {
		t.Errorf("Stream = %v after Logout; want nil", c.Stream)
	}
}
