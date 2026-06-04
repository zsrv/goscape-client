package client

// 254 server→client opcodes. Numbers: Client.java tcpIn/zonePacket dispatch
// @2e62978; names: the deob's inline // LABEL comments (all 60 single-opcode
// labels + the 10-opcode zone set verified 1:1 against the dispatch bodies;
// every value cross-checked against SERVERPROT_LENGTH at the new indices).
// vs 245.2: full renumber; VIEWPORT_FLASH removed (the field1504 yellow-flash
// overlay is gone in 254); SET_PLAYER_OP and FRIENDLIST_LOADED are new.
// Zone opcodes (LOC_*/OBJ_*/MAP_*) share this number space: the server sends
// them standalone or batched inside UPDATE_ZONE_PARTIAL_ENCLOSED.
const (
	SERVERPROT_IF_OPENCHAT              = 141
	SERVERPROT_IF_OPENMAIN_SIDE         = 249
	SERVERPROT_IF_CLOSE                 = 174
	SERVERPROT_IF_SETTAB                = 91
	SERVERPROT_IF_OPENMAIN              = 197
	SERVERPROT_IF_OPENSIDE              = 187
	SERVERPROT_IF_OPENOVERLAY           = 85
	SERVERPROT_IF_SETTAB_ACTIVE         = 138
	SERVERPROT_IF_SETCOLOUR             = 38
	SERVERPROT_IF_SETHIDE               = 227
	SERVERPROT_IF_SETOBJECT             = 222
	SERVERPROT_IF_SETMODEL              = 211
	SERVERPROT_IF_SETANIM               = 95
	SERVERPROT_IF_SETPLAYERHEAD         = 161
	SERVERPROT_IF_SETTEXT               = 41
	SERVERPROT_IF_SETNPCHEAD            = 3
	SERVERPROT_IF_SETPOSITION           = 27
	SERVERPROT_IF_SETSCROLLPOS          = 14
	SERVERPROT_TUT_FLASH                = 58
	SERVERPROT_TUT_OPEN                 = 239
	SERVERPROT_UPDATE_INV_STOP_TRANSMIT = 168
	SERVERPROT_UPDATE_INV_FULL          = 28
	SERVERPROT_UPDATE_INV_PARTIAL       = 170
	SERVERPROT_CAM_LOOKAT               = 0
	SERVERPROT_CAM_SHAKE                = 225
	SERVERPROT_CAM_MOVETO               = 55
	SERVERPROT_CAM_RESET                = 167
	SERVERPROT_NPC_INFO                 = 123
	SERVERPROT_PLAYER_INFO              = 87
	SERVERPROT_FINISH_TRACKING          = 29
	SERVERPROT_ENABLE_TRACKING          = 251
	SERVERPROT_MESSAGE_GAME             = 73
	SERVERPROT_UPDATE_IGNORELIST        = 63
	SERVERPROT_CHAT_FILTER_SETTINGS     = 24
	SERVERPROT_MESSAGE_PRIVATE          = 60
	// NEW in 254: server-driven player right-click options (Client.java:6607
	// @2e62978). Writes playerOptions/playerOptionsPushDown.
	SERVERPROT_SET_PLAYER_OP = 204
	// NEW in 254: friend-server connection status (Client.java:6919 @2e62978).
	// Writes friendListStatus (0 none / 1 connecting / 2 loaded).
	SERVERPROT_FRIENDLIST_LOADED            = 255
	SERVERPROT_UPDATE_FRIENDLIST            = 111
	SERVERPROT_UNSET_MAP_FLAG               = 108
	SERVERPROT_UPDATE_RUNWEIGHT             = 164
	SERVERPROT_HINT_ARROW                   = 64
	SERVERPROT_UPDATE_REBOOT_TIMER          = 143
	SERVERPROT_UPDATE_STAT                  = 136
	SERVERPROT_UPDATE_RUNENERGY             = 94
	SERVERPROT_RESET_ANIMS                  = 203
	SERVERPROT_UPDATE_PID                   = 213
	SERVERPROT_LAST_LOGIN_INFO              = 146
	SERVERPROT_LOGOUT                       = 21
	SERVERPROT_P_COUNTDIALOG                = 5
	SERVERPROT_SET_MULTIWAY                 = 75
	SERVERPROT_REBUILD_NORMAL               = 209
	SERVERPROT_VARP_SMALL                   = 186
	SERVERPROT_VARP_LARGE                   = 196
	SERVERPROT_RESET_CLIENT_VARCACHE        = 140
	SERVERPROT_SYNTH_SOUND                  = 25
	SERVERPROT_MIDI_SONG                    = 163
	SERVERPROT_MIDI_JINGLE                  = 242
	SERVERPROT_UPDATE_ZONE_PARTIAL_FOLLOWS  = 173
	SERVERPROT_UPDATE_ZONE_FULL_FOLLOWS     = 159
	SERVERPROT_UPDATE_ZONE_PARTIAL_ENCLOSED = 61
	SERVERPROT_LOC_MERGE                    = 218
	SERVERPROT_LOC_ANIM                     = 30
	SERVERPROT_OBJ_DEL                      = 115
	SERVERPROT_OBJ_REVEAL                   = 8
	SERVERPROT_LOC_ADD_CHANGE               = 70
	SERVERPROT_MAP_PROJANIM                 = 37
	SERVERPROT_LOC_DEL                      = 88
	SERVERPROT_OBJ_COUNT                    = 98
	SERVERPROT_MAP_ANIM                     = 114
	SERVERPROT_OBJ_ADD                      = 120
)
