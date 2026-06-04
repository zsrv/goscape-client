package client

// 245.2 server→client opcodes. Numbers: Client.java readPacket/readZonePacket
// dispatch @176a85f; names: the deob's adjacent // LABEL comments (verified
// against the 244 set — every label pairs 1:1 and SERVERPROT_LENGTH agrees at
// old/new indices for all of them; IF_SETSCROLLPOS is the single addition).
// Zone opcodes (LOC_*/OBJ_*/MAP_*) share this number space: the server sends
// them standalone or batched inside UPDATE_ZONE_PARTIAL_ENCLOSED.
const (
	SERVERPROT_IF_OPENCHAT              = 7
	SERVERPROT_IF_OPENMAIN_SIDE         = 229
	SERVERPROT_IF_CLOSE                 = 174
	SERVERPROT_IF_SETTAB                = 29
	SERVERPROT_IF_OPENMAIN              = 177
	SERVERPROT_IF_OPENSIDE              = 236
	SERVERPROT_IF_OPENOVERLAY           = 115 // NEW in 244 (no 225 Go branch)
	SERVERPROT_IF_SETTAB_ACTIVE         = 8
	SERVERPROT_IF_SETCOLOUR             = 135
	SERVERPROT_IF_SETHIDE               = 225
	SERVERPROT_IF_SETOBJECT             = 153
	SERVERPROT_IF_SETMODEL              = 60
	SERVERPROT_IF_SETANIM               = 69
	SERVERPROT_IF_SETPLAYERHEAD         = 83
	SERVERPROT_IF_SETTEXT               = 32
	SERVERPROT_IF_SETNPCHEAD            = 76
	SERVERPROT_IF_SETPOSITION           = 230
	SERVERPROT_IF_SETSCROLLPOS          = 226 // NEW in 245.2, length 4 (g2 comId + g2 pos)
	SERVERPROT_TUT_FLASH                = 132
	SERVERPROT_TUT_OPEN                 = 152
	SERVERPROT_UPDATE_INV_STOP_TRANSMIT = 143
	SERVERPROT_UPDATE_INV_FULL          = 156
	SERVERPROT_UPDATE_INV_PARTIAL       = 95
	SERVERPROT_CAM_LOOKAT               = 123
	SERVERPROT_CAM_SHAKE                = 103
	SERVERPROT_CAM_MOVETO               = 86
	SERVERPROT_CAM_RESET                = 134
	SERVERPROT_NPC_INFO                 = 105
	SERVERPROT_PLAYER_INFO              = 161
	SERVERPROT_FINISH_TRACKING          = 165
	SERVERPROT_ENABLE_TRACKING          = 28
	SERVERPROT_MESSAGE_GAME             = 175
	SERVERPROT_UPDATE_IGNORELIST        = 181
	SERVERPROT_CHAT_FILTER_SETTINGS     = 2
	SERVERPROT_MESSAGE_PRIVATE          = 207
	// NEW in 244, zero-length; unnamed in both Java (ptype == 108 ->
	// field1504 = 255, Client.java:8005 @176a85f) and Client-TS. Triggers the
	// yellow viewport flash overlay.
	SERVERPROT_VIEWPORT_FLASH               = 108
	SERVERPROT_UPDATE_FRIENDLIST            = 109
	SERVERPROT_UNSET_MAP_FLAG               = 233
	SERVERPROT_UPDATE_RUNWEIGHT             = 70
	SERVERPROT_HINT_ARROW                   = 243
	SERVERPROT_UPDATE_REBOOT_TIMER          = 26
	SERVERPROT_UPDATE_STAT                  = 110
	SERVERPROT_UPDATE_RUNENERGY             = 208
	SERVERPROT_RESET_ANIMS                  = 144
	SERVERPROT_UPDATE_PID                   = 49
	SERVERPROT_LAST_LOGIN_INFO              = 238
	SERVERPROT_LOGOUT                       = 36
	SERVERPROT_P_COUNTDIALOG                = 56
	SERVERPROT_SET_MULTIWAY                 = 35
	SERVERPROT_REBUILD_NORMAL               = 66
	SERVERPROT_VARP_SMALL                   = 192
	SERVERPROT_VARP_LARGE                   = 75
	SERVERPROT_RESET_CLIENT_VARCACHE        = 25
	SERVERPROT_SYNTH_SOUND                  = 209
	SERVERPROT_MIDI_SONG                    = 96
	SERVERPROT_MIDI_JINGLE                  = 39 // NEW in 244 (no 225 Go branch)
	SERVERPROT_UPDATE_ZONE_PARTIAL_FOLLOWS  = 203
	SERVERPROT_UPDATE_ZONE_FULL_FOLLOWS     = 140
	SERVERPROT_UPDATE_ZONE_PARTIAL_ENCLOSED = 15
	SERVERPROT_LOC_MERGE                    = 188
	SERVERPROT_LOC_ANIM                     = 71
	SERVERPROT_OBJ_DEL                      = 13
	SERVERPROT_OBJ_REVEAL                   = 190
	SERVERPROT_LOC_ADD_CHANGE               = 119
	SERVERPROT_MAP_PROJANIM                 = 187
	SERVERPROT_LOC_DEL                      = 198
	SERVERPROT_OBJ_COUNT                    = 151
	SERVERPROT_MAP_ANIM                     = 141
	SERVERPROT_OBJ_ADD                      = 94
)
