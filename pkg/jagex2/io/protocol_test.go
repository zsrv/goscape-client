package io_test

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/client"
	io "github.com/zsrv/goscape-client/pkg/jagex2/io"
)

func TestServerProtSizes274(t *testing.T) {
	// Java: Protocol.SERVERPROT_SIZE has 257 entries (indices 0-256).
	if got := len(io.SERVERPROT_SIZES); got != 257 {
		t.Fatalf("len(SERVERPROT_SIZES) = %d, want 257", got)
	}

	// Pin the 274 renumber against named constants (values from the
	// Client.java tcpIn/zonePacket dispatch @32f3062). The per-message sizes
	// are revision-invariant — the constants move, the size follows the
	// message — so a mismatch here means a constant landed on the wrong
	// index.
	checks := []struct {
		name   string
		opcode int
		want   int
	}{
		{"LOC_MERGE", client.SERVERPROT_LOC_MERGE, 14},
		{"REBUILD_NORMAL", client.SERVERPROT_REBUILD_NORMAL, 4},
		{"RESET_ANIMS", client.SERVERPROT_RESET_ANIMS, 0},
		{"NPC_INFO", client.SERVERPROT_NPC_INFO, -2},                  // variable-length, g2 prefix
		{"MESSAGE_GAME", client.SERVERPROT_MESSAGE_GAME, -1},          // variable-length, g1 prefix
		{"SET_PLAYER_OP", client.SERVERPROT_SET_PLAYER_OP, -1},        // NEW in 254
		{"FRIENDLIST_LOADED", client.SERVERPROT_FRIENDLIST_LOADED, 1}, // NEW in 254
		{"SET_MINIMAP_STATE", client.SERVERPROT_SET_MINIMAP_STATE, 1}, // NEW in 274
		{"UPDATE_INV_FULL", client.SERVERPROT_UPDATE_INV_FULL, -2},
		{"CAM_LOOKAT", client.SERVERPROT_CAM_LOOKAT, 6},
		// Same-shape pair adjudicated twice in P3 (model1Type=2 vs =1):
		{"IF_SETNPCHEAD", client.SERVERPROT_IF_SETNPCHEAD, 4},
		{"IF_SETMODEL", client.SERVERPROT_IF_SETMODEL, 4},
	}
	for _, c := range checks {
		if got := io.SERVERPROT_SIZES[c.opcode]; got != c.want {
			t.Errorf("SERVERPROT_SIZES[SERVERPROT_%s=%d] = %d, want %d", c.name, c.opcode, got, c.want)
		}
	}
}
