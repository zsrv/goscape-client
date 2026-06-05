package client

// 274 client→server opcodes. Numbers: Client.java p1Enc/interactWithLoc
// sites @32f3062 (82 messages: 75 as p1Enc literals across 90 sites, plus
// the 7 OPLOC* routed through interactWithLoc's p1Enc(arg3) re-emission,
// Client.java:3876 @32f3062); names: carried from the 254 deob's // LABEL
// comments via the LOGIC-DELTA-SCOPE-274.md pairing (the 274 deob has no
// inline labels; every pairing adversarially verified against both pins).
// vs 254: full renumber — ANTICHEAT_CYCLELOGIC5 (100) is the ONLY value
// unchanged; EVENT_TRACKING is removed with InputTracking (WS2). Beware
// intra/cross-rev value reuse (e.g. 254 op9=FRIENDLIST_ADD but 274
// op9=IF_BUTTON) — never port by value, always by message.
// Written RAW through Packet.P1Isaac (Java 274 p1Enc: opcode + ISAAC
// keystream). Java's CLIENTPROT_SCRAMBLED table is a deobfuscation artifact
// (never read at runtime; re-verified at @32f3062: declaration only) —
// intentionally not ported.
const (
	CLIENTPROT_NO_TIMEOUT = 120
	CLIENTPROT_IDLE_TIMER = 209
	// 254 value — the message does not exist in 274; constant and all its
	// sites are deleted with InputTracking (WS2).
	CLIENTPROT_EVENT_TRACKING        = 142
	CLIENTPROT_ANTICHEAT_OPLOGIC1    = 219
	CLIENTPROT_ANTICHEAT_OPLOGIC2    = 201
	CLIENTPROT_ANTICHEAT_OPLOGIC3    = 41
	CLIENTPROT_ANTICHEAT_OPLOGIC4    = 80
	CLIENTPROT_ANTICHEAT_OPLOGIC5    = 235
	CLIENTPROT_ANTICHEAT_OPLOGIC6    = 250
	CLIENTPROT_ANTICHEAT_OPLOGIC7    = 25
	CLIENTPROT_ANTICHEAT_OPLOGIC8    = 0
	CLIENTPROT_ANTICHEAT_OPLOGIC9    = 24
	CLIENTPROT_ANTICHEAT_CYCLELOGIC1 = 12
	CLIENTPROT_ANTICHEAT_CYCLELOGIC2 = 149
	CLIENTPROT_ANTICHEAT_CYCLELOGIC3 = 52
	CLIENTPROT_ANTICHEAT_CYCLELOGIC4 = 230
	CLIENTPROT_ANTICHEAT_CYCLELOGIC5 = 100
	CLIENTPROT_ANTICHEAT_CYCLELOGIC6 = 188
	// NEW in 254: sent from gameLoop when cyclelogic7 overflows its threshold
	// (Client.java:9553 @32f3062).
	CLIENTPROT_ANTICHEAT_CYCLELOGIC7 = 89
	CLIENTPROT_OPOBJ1                = 247
	CLIENTPROT_OPOBJ2                = 169
	CLIENTPROT_OPOBJ3                = 108
	CLIENTPROT_OPOBJ4                = 62
	CLIENTPROT_OPOBJ5                = 117
	CLIENTPROT_OPOBJT                = 91
	CLIENTPROT_OPOBJU                = 39
	CLIENTPROT_OPNPC1                = 236
	CLIENTPROT_OPNPC2                = 233
	CLIENTPROT_OPNPC3                = 223
	CLIENTPROT_OPNPC4                = 147
	CLIENTPROT_OPNPC5                = 189
	CLIENTPROT_OPNPCT                = 181
	CLIENTPROT_OPNPCU                = 150
	CLIENTPROT_OPLOC1                = 215
	CLIENTPROT_OPLOC2                = 103
	CLIENTPROT_OPLOC3                = 187
	CLIENTPROT_OPLOC4                = 157
	CLIENTPROT_OPLOC5                = 127
	CLIENTPROT_OPLOCT                = 213
	CLIENTPROT_OPLOCU                = 60
	CLIENTPROT_OPPLAYER1             = 109
	CLIENTPROT_OPPLAYER2             = 166
	CLIENTPROT_OPPLAYER3             = 196
	CLIENTPROT_OPPLAYER4             = 98
	// NEW in 254: fifth player op slot, paired with the server-driven
	// SET_PLAYER_OP options (Client.java:4412 @32f3062).
	CLIENTPROT_OPPLAYER5            = 174
	CLIENTPROT_OPPLAYERT            = 240
	CLIENTPROT_OPPLAYERU            = 36
	CLIENTPROT_OPHELD1              = 185
	CLIENTPROT_OPHELD2              = 2
	CLIENTPROT_OPHELD3              = 123
	CLIENTPROT_OPHELD4              = 216
	CLIENTPROT_OPHELD5              = 42
	CLIENTPROT_OPHELDT              = 135
	CLIENTPROT_OPHELDU              = 136
	CLIENTPROT_INV_BUTTON1          = 74
	CLIENTPROT_INV_BUTTON2          = 82
	CLIENTPROT_INV_BUTTON3          = 239
	CLIENTPROT_INV_BUTTON4          = 179
	CLIENTPROT_INV_BUTTON5          = 46
	CLIENTPROT_IF_BUTTON            = 9
	CLIENTPROT_RESUME_PAUSEBUTTON   = 72
	CLIENTPROT_CLOSE_MODAL          = 51
	CLIENTPROT_RESUME_P_COUNTDIALOG = 102
	CLIENTPROT_TUTORIAL_CLICKSIDE   = 94
	CLIENTPROT_MOVE_OPCLICK         = 138
	CLIENTPROT_REPORT_ABUSE         = 137
	CLIENTPROT_MOVE_MINIMAPCLICK    = 86
	CLIENTPROT_INV_BUTTOND          = 93
	CLIENTPROT_IGNORELIST_DEL       = 101
	CLIENTPROT_IGNORELIST_ADD       = 255
	CLIENTPROT_IF_PLAYERDESIGN      = 125
	CLIENTPROT_CHAT_SETMODE         = 154
	CLIENTPROT_MESSAGE_PRIVATE      = 139
	CLIENTPROT_FRIENDLIST_DEL       = 106
	CLIENTPROT_FRIENDLIST_ADD       = 13
	CLIENTPROT_CLIENT_CHEAT         = 224
	CLIENTPROT_MESSAGE_PUBLIC       = 253
	CLIENTPROT_MOVE_GAMECLICK       = 207
	// NEW in 254: outbound telemetry set, emitted by gameLoop (274 sites).
	// EVENT_MOUSE_MOVE: packed mouse deltas (Client.java:9352 @32f3062);
	// EVENT_MOUSE_CLICK: time/button/position word (:9433);
	// EVENT_CAMERA_POSITION: camera state on a 20-cycle gate (:9445);
	// EVENT_APPLET_FOCUS: focus gained/lost p1(1/0) (:9451,9456).
	CLIENTPROT_EVENT_MOUSE_MOVE      = 222
	CLIENTPROT_EVENT_MOUSE_CLICK     = 20
	CLIENTPROT_EVENT_CAMERA_POSITION = 53
	CLIENTPROT_EVENT_APPLET_FOCUS    = 73
	// NEW in 254: zero-length notification sent by checkScene immediately
	// after mapBuild() (Client.java:3283 @32f3062).
	CLIENTPROT_MAP_BUILD_COMPLETE = 214
)
