package client

// 274 server→client opcodes. Numbers: Client.java tcpIn/zonePacket dispatch
// @32f3062 (69 distinct dispatch values: 59 single-opcode handlers + the
// 10-opcode zone set, verified 1:1 against the dispatch bodies via the
// LOGIC-DELTA-SCOPE-274.md pairing; every value cross-checked against
// SERVERPROT_SIZE at the new indices). vs 254: full renumber;
// FINISH_TRACKING (29) / ENABLE_TRACKING (251) were removed with
// InputTracking; SET_MINIMAP_STATE is
// new. Same-shape pair IF_SETNPCHEAD (3→142) vs IF_SETMODEL (211→129) was
// adjudicated twice by model1Type=2 vs =1.
// Zone opcodes (LOC_*/OBJ_*/MAP_*) share this number space: the server sends
// them standalone or batched inside UPDATE_ZONE_PARTIAL_ENCLOSED.
const (
	SERVERPROT_IF_OPENCHAT              = 166
	SERVERPROT_IF_OPENMAIN_SIDE         = 158
	SERVERPROT_IF_CLOSE                 = 171
	SERVERPROT_IF_SETTAB                = 215
	SERVERPROT_IF_OPENMAIN              = 211
	SERVERPROT_IF_OPENSIDE              = 16
	SERVERPROT_IF_OPENOVERLAY           = 240
	SERVERPROT_IF_SETTAB_ACTIVE         = 241
	SERVERPROT_IF_SETCOLOUR             = 183
	SERVERPROT_IF_SETHIDE               = 10
	SERVERPROT_IF_SETOBJECT             = 28
	SERVERPROT_IF_SETMODEL              = 129
	SERVERPROT_IF_SETANIM               = 134
	SERVERPROT_IF_SETPLAYERHEAD         = 192
	SERVERPROT_IF_SETTEXT               = 44
	SERVERPROT_IF_SETNPCHEAD            = 142
	SERVERPROT_IF_SETPOSITION           = 77
	SERVERPROT_IF_SETSCROLLPOS          = 54
	SERVERPROT_TUT_FLASH                = 90
	SERVERPROT_TUT_OPEN                 = 130
	SERVERPROT_UPDATE_INV_STOP_TRANSMIT = 227
	SERVERPROT_UPDATE_INV_FULL          = 106
	SERVERPROT_UPDATE_INV_PARTIAL       = 172
	SERVERPROT_CAM_LOOKAT               = 233
	SERVERPROT_CAM_SHAKE                = 64
	SERVERPROT_CAM_MOVETO               = 200
	SERVERPROT_CAM_RESET                = 101
	SERVERPROT_NPC_INFO                 = 197
	SERVERPROT_PLAYER_INFO              = 167
	SERVERPROT_MESSAGE_GAME             = 161
	SERVERPROT_UPDATE_IGNORELIST        = 3
	SERVERPROT_CHAT_FILTER_SETTINGS     = 114
	SERVERPROT_MESSAGE_PRIVATE          = 235
	// NEW in 254: server-driven player right-click options (Client.java:8547
	// @32f3062). Writes playerOptions/playerOptionsPushDown.
	SERVERPROT_SET_PLAYER_OP = 17
	// NEW in 254: friend-server connection status (Client.java:8648 @32f3062).
	// Writes friendListStatus (0 none / 1 connecting / 2 loaded).
	SERVERPROT_FRIENDLIST_LOADED = 185
	// NEW in 274: minimap state machine (Client.java:8227 @32f3062) —
	// minimapState = g1 (0 normal / 2 blackout draw / click-to-walk gated
	// unless 0). Handler lands with WS5.
	SERVERPROT_SET_MINIMAP_STATE            = 194
	SERVERPROT_UPDATE_FRIENDLIST            = 247
	SERVERPROT_UNSET_MAP_FLAG               = 115
	SERVERPROT_UPDATE_RUNWEIGHT             = 67
	SERVERPROT_HINT_ARROW                   = 156
	SERVERPROT_UPDATE_REBOOT_TIMER          = 89
	SERVERPROT_UPDATE_STAT                  = 105
	SERVERPROT_UPDATE_RUNENERGY             = 83
	SERVERPROT_RESET_ANIMS                  = 47
	SERVERPROT_UPDATE_PID                   = 133
	SERVERPROT_LAST_LOGIN_INFO              = 91
	SERVERPROT_LOGOUT                       = 88
	SERVERPROT_P_COUNTDIALOG                = 210
	SERVERPROT_SET_MULTIWAY                 = 207
	SERVERPROT_REBUILD_NORMAL               = 231
	SERVERPROT_VARP_SMALL                   = 203
	SERVERPROT_VARP_LARGE                   = 245
	SERVERPROT_RESET_CLIENT_VARCACHE        = 190
	SERVERPROT_SYNTH_SOUND                  = 34
	SERVERPROT_MIDI_SONG                    = 23
	SERVERPROT_MIDI_JINGLE                  = 15
	SERVERPROT_UPDATE_ZONE_PARTIAL_FOLLOWS  = 32
	SERVERPROT_UPDATE_ZONE_FULL_FOLLOWS     = 153
	SERVERPROT_UPDATE_ZONE_PARTIAL_ENCLOSED = 195
	SERVERPROT_LOC_MERGE                    = 176
	SERVERPROT_LOC_ANIM                     = 48
	SERVERPROT_OBJ_DEL                      = 52
	SERVERPROT_OBJ_REVEAL                   = 219
	SERVERPROT_LOC_ADD_CHANGE               = 138
	SERVERPROT_MAP_PROJANIM                 = 107
	SERVERPROT_LOC_DEL                      = 173
	SERVERPROT_OBJ_COUNT                    = 95
	SERVERPROT_MAP_ANIM                     = 85
	SERVERPROT_OBJ_ADD                      = 81
)
