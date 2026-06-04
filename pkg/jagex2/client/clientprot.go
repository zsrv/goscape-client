package client

// 254 client→server opcodes. Numbers: Client.java pIsaac/interactWithLoc
// sites @2e62978 (83 distinct: 76 pIsaac literals + the 7 OPLOC* routed
// through interactWithLoc's pIsaac(arg4) re-emission, Client.java:6312);
// names: the deob's adjacent // LABEL comments. vs 245.2: full renumber plus
// seven NEW messages — the EVENT_* mouse/camera/focus telemetry set,
// MAP_BUILD_COMPLETE, ANTICHEAT_CYCLELOGIC7, and OPPLAYER5.
// Written RAW through Packet.P1Isaac (opcode + ISAAC keystream). Java's
// CLIENTPROT_LOOKUP table is a deobfuscation artifact (never read at runtime;
// re-verified at @2e62978: declaration only) — intentionally not ported.
const (
	CLIENTPROT_NO_TIMEOUT            = 239
	CLIENTPROT_IDLE_TIMER            = 144
	CLIENTPROT_EVENT_TRACKING        = 142
	CLIENTPROT_ANTICHEAT_OPLOGIC1    = 28
	CLIENTPROT_ANTICHEAT_OPLOGIC2    = 77
	CLIENTPROT_ANTICHEAT_OPLOGIC3    = 56
	CLIENTPROT_ANTICHEAT_OPLOGIC4    = 121
	CLIENTPROT_ANTICHEAT_OPLOGIC5    = 233
	CLIENTPROT_ANTICHEAT_OPLOGIC6    = 131
	CLIENTPROT_ANTICHEAT_OPLOGIC7    = 187
	CLIENTPROT_ANTICHEAT_OPLOGIC8    = 206
	CLIENTPROT_ANTICHEAT_OPLOGIC9    = 162
	CLIENTPROT_ANTICHEAT_CYCLELOGIC1 = 51
	CLIENTPROT_ANTICHEAT_CYCLELOGIC2 = 225
	CLIENTPROT_ANTICHEAT_CYCLELOGIC3 = 4
	CLIENTPROT_ANTICHEAT_CYCLELOGIC4 = 226
	CLIENTPROT_ANTICHEAT_CYCLELOGIC5 = 100
	CLIENTPROT_ANTICHEAT_CYCLELOGIC6 = 36
	// NEW in 254: sent from gameLoop when field1294 overflows its threshold
	// (Client.java:2927 @2e62978). Handler lands with WS5 anticheat.
	CLIENTPROT_ANTICHEAT_CYCLELOGIC7 = 182
	CLIENTPROT_OPOBJ1                = 141
	CLIENTPROT_OPOBJ2                = 67
	CLIENTPROT_OPOBJ3                = 178
	CLIENTPROT_OPOBJ4                = 47
	CLIENTPROT_OPOBJ5                = 97
	CLIENTPROT_OPOBJT                = 202
	CLIENTPROT_OPOBJU                = 245
	CLIENTPROT_OPNPC1                = 143
	CLIENTPROT_OPNPC2                = 195
	CLIENTPROT_OPNPC3                = 69
	CLIENTPROT_OPNPC4                = 122
	CLIENTPROT_OPNPC5                = 118
	CLIENTPROT_OPNPCT                = 231
	CLIENTPROT_OPNPCU                = 119
	CLIENTPROT_OPLOC1                = 33
	CLIENTPROT_OPLOC2                = 213
	CLIENTPROT_OPLOC3                = 98
	CLIENTPROT_OPLOC4                = 87
	CLIENTPROT_OPLOC5                = 147
	CLIENTPROT_OPLOCT                = 26
	CLIENTPROT_OPLOCU                = 240
	CLIENTPROT_OPPLAYER1             = 192
	CLIENTPROT_OPPLAYER2             = 17
	CLIENTPROT_OPPLAYER3             = 18
	CLIENTPROT_OPPLAYER4             = 72
	// NEW in 254: fifth player op slot, paired with the server-driven
	// SET_PLAYER_OP options (Client.java:9077 @2e62978; WS5).
	CLIENTPROT_OPPLAYER5            = 230
	CLIENTPROT_OPPLAYERT            = 68
	CLIENTPROT_OPPLAYERU            = 113
	CLIENTPROT_OPHELD1              = 243
	CLIENTPROT_OPHELD2              = 228
	CLIENTPROT_OPHELD3              = 80
	CLIENTPROT_OPHELD4              = 163
	CLIENTPROT_OPHELD5              = 74
	CLIENTPROT_OPHELDT              = 102
	CLIENTPROT_OPHELDU              = 200
	CLIENTPROT_INV_BUTTON1          = 181
	CLIENTPROT_INV_BUTTON2          = 70
	CLIENTPROT_INV_BUTTON3          = 59
	CLIENTPROT_INV_BUTTON4          = 160
	CLIENTPROT_INV_BUTTON5          = 62
	CLIENTPROT_IF_BUTTON            = 244
	CLIENTPROT_RESUME_PAUSEBUTTON   = 146
	CLIENTPROT_CLOSE_MODAL          = 58
	CLIENTPROT_RESUME_P_COUNTDIALOG = 161
	CLIENTPROT_TUTORIAL_CLICKSIDE   = 201
	CLIENTPROT_MOVE_OPCLICK         = 127
	CLIENTPROT_REPORT_ABUSE         = 203
	CLIENTPROT_MOVE_MINIMAPCLICK    = 220
	CLIENTPROT_INV_BUTTOND          = 176
	CLIENTPROT_IGNORELIST_DEL       = 193
	CLIENTPROT_IGNORELIST_ADD       = 189
	CLIENTPROT_IF_PLAYERDESIGN      = 13
	CLIENTPROT_CHAT_SETMODE         = 129
	CLIENTPROT_MESSAGE_PRIVATE      = 214
	CLIENTPROT_FRIENDLIST_DEL       = 84
	CLIENTPROT_FRIENDLIST_ADD       = 9
	CLIENTPROT_CLIENT_CHEAT         = 86
	CLIENTPROT_MESSAGE_PUBLIC       = 83
	CLIENTPROT_MOVE_GAMECLICK       = 6
	// NEW in 254: outbound telemetry set, emitted by gameLoop (WS5).
	// EVENT_MOUSE_MOVE: packed mouse deltas (Client.java:2711 @2e62978);
	// EVENT_MOUSE_CLICK: time/button/position word (:2793);
	// EVENT_CAMERA_POSITION: camera state on a 20-cycle gate (:2806);
	// EVENT_APPLET_FOCUS: focus gained/lost p1(1/0) (:2813,2819).
	CLIENTPROT_EVENT_MOUSE_MOVE      = 232
	CLIENTPROT_EVENT_MOUSE_CLICK     = 234
	CLIENTPROT_EVENT_CAMERA_POSITION = 91
	CLIENTPROT_EVENT_APPLET_FOCUS    = 8
	// NEW in 254: zero-length notification sent by checkScene immediately
	// after mapBuild() (Client.java:3121 @2e62978).
	CLIENTPROT_MAP_BUILD_COMPLETE = 134
)
