package client

import "testing"

func TestServerProtSizes254(t *testing.T) {
	// Java: Protocol.SERVERPROT_LENGTH has 257 entries (indices 0-256).
	if got := len(SERVERPROT_SIZES); got != 257 {
		t.Fatalf("len(SERVERPROT_SIZES) = %d, want 257", got)
	}

	// Pin the 254 renumber against named constants (values from the
	// Client.java tcpIn/zonePacket dispatch @2e62978).
	checks := []struct {
		name   string
		opcode int
		want   int
	}{
		{"LOC_MERGE", SERVERPROT_LOC_MERGE, 14},
		{"REBUILD_NORMAL", SERVERPROT_REBUILD_NORMAL, 4},
		{"RESET_ANIMS", SERVERPROT_RESET_ANIMS, 0},
		{"NPC_INFO", SERVERPROT_NPC_INFO, -2},                  // variable-length, g2 prefix
		{"MESSAGE_GAME", SERVERPROT_MESSAGE_GAME, -1},          // variable-length, g1 prefix
		{"SET_PLAYER_OP", SERVERPROT_SET_PLAYER_OP, -1},        // NEW in 254
		{"FRIENDLIST_LOADED", SERVERPROT_FRIENDLIST_LOADED, 1}, // NEW in 254
		{"UPDATE_INV_FULL", SERVERPROT_UPDATE_INV_FULL, -2},
		{"CAM_LOOKAT", SERVERPROT_CAM_LOOKAT, 6},
	}
	for _, c := range checks {
		if got := SERVERPROT_SIZES[c.opcode]; got != c.want {
			t.Errorf("SERVERPROT_SIZES[SERVERPROT_%s=%d] = %d, want %d", c.name, c.opcode, got, c.want)
		}
	}
}
