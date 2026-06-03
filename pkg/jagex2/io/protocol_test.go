package io

import "testing"

func TestServerProtSizes244(t *testing.T) {
	// Java: Protocol.SERVERPROT_LENGTH has 257 entries (indices 0-256).
	if got := len(SERVERPROT_SIZES); got != 257 {
		t.Fatalf("len(SERVERPROT_SIZES) = %d, want 257", got)
	}

	// Pin the 244 renumber against named constants.

	// SERVERPROT_LOC_MERGE (29) == 14
	if got := SERVERPROT_SIZES[SERVERPROT_LOC_MERGE]; got != 14 {
		t.Errorf("SERVERPROT_SIZES[SERVERPROT_LOC_MERGE=%d] = %d, want 14", SERVERPROT_LOC_MERGE, got)
	}

	// SERVERPROT_REBUILD_NORMAL (165) == 4
	if got := SERVERPROT_SIZES[SERVERPROT_REBUILD_NORMAL]; got != 4 {
		t.Errorf("SERVERPROT_SIZES[SERVERPROT_REBUILD_NORMAL=%d] = %d, want 4", SERVERPROT_REBUILD_NORMAL, got)
	}

	// SERVERPROT_RESET_ANIMS (242) == 0
	if got := SERVERPROT_SIZES[242]; got != 0 {
		t.Errorf("SERVERPROT_SIZES[242 (SERVERPROT_RESET_ANIMS)] = %d, want 0", got)
	}

	// SERVERPROT_NPC_INFO (244) == -2 (variable-length, delta-encoded)
	if got := SERVERPROT_SIZES[SERVERPROT_NPC_INFO]; got != -2 {
		t.Errorf("SERVERPROT_SIZES[SERVERPROT_NPC_INFO=%d] = %d, want -2", SERVERPROT_NPC_INFO, got)
	}

	// SERVERPROT_MESSAGE_GAME (95) == -1 (variable-length with byte prefix)
	if got := SERVERPROT_SIZES[SERVERPROT_MESSAGE_GAME]; got != -1 {
		t.Errorf("SERVERPROT_SIZES[SERVERPROT_MESSAGE_GAME=%d] = %d, want -1", SERVERPROT_MESSAGE_GAME, got)
	}
}
