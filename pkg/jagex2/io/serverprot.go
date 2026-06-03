package io

// 244 server→client opcodes. Java: jagex2.io.Protocol (numbers) +
// Client-TS src/io/ServerProt.ts (names). Renumbered from rev-225.
const (
	SERVERPROT_IF_OPENCHAT              = 189
	SERVERPROT_IF_OPENMAIN_SIDE         = 207
	SERVERPROT_IF_CLOSE                 = 214
	SERVERPROT_IF_SETTAB                = 200
	SERVERPROT_IF_OPENMAIN              = 10
	SERVERPROT_IF_OPENSIDE              = 176
	SERVERPROT_IF_OPENOVERLAY           = 158 // NEW in 244 (no 225 Go branch)
	SERVERPROT_IF_SETTAB_ACTIVE         = 56
	SERVERPROT_IF_SETCOLOUR             = 78
	SERVERPROT_IF_SETHIDE               = 123
	SERVERPROT_IF_SETOBJECT             = 164
	SERVERPROT_IF_SETMODEL              = 245
	SERVERPROT_IF_SETANIM               = 219
	SERVERPROT_IF_SETPLAYERHEAD         = 108
	SERVERPROT_IF_SETTEXT               = 154
	SERVERPROT_IF_SETNPCHEAD            = 129
	SERVERPROT_IF_SETPOSITION           = 241
	SERVERPROT_TUT_FLASH                = 168
	SERVERPROT_TUT_OPEN                 = 174
	SERVERPROT_UPDATE_INV_STOP_TRANSMIT = 162
	SERVERPROT_UPDATE_INV_FULL          = 72
	SERVERPROT_UPDATE_INV_PARTIAL       = 132
	SERVERPROT_CAM_LOOKAT               = 222
	SERVERPROT_CAM_SHAKE                = 50
	SERVERPROT_CAM_MOVETO               = 12
	SERVERPROT_CAM_RESET                = 53
	SERVERPROT_NPC_INFO                 = 244
	SERVERPROT_PLAYER_INFO              = 86
	SERVERPROT_FINISH_TRACKING          = 60
	SERVERPROT_ENABLE_TRACKING          = 22
	SERVERPROT_MESSAGE_GAME             = 95
	SERVERPROT_UPDATE_IGNORELIST        = 7
	SERVERPROT_CHAT_FILTER_SETTINGS     = 9
	SERVERPROT_MESSAGE_PRIVATE          = 30
	// NEW in 244, zero-length; unnamed in both Java (ptype == 192 -> field1264
	// = 255, Client.java:7377) and Client-TS. Triggers the yellow viewport
	// flash overlay.
	SERVERPROT_VIEWPORT_FLASH               = 192
	SERVERPROT_UPDATE_FRIENDLIST            = 70
	SERVERPROT_UNSET_MAP_FLAG               = 62
	SERVERPROT_UPDATE_RUNWEIGHT             = 160
	SERVERPROT_HINT_ARROW                   = 49
	SERVERPROT_UPDATE_REBOOT_TIMER          = 85
	SERVERPROT_UPDATE_STAT                  = 24
	SERVERPROT_UPDATE_RUNENERGY             = 177
	SERVERPROT_RESET_ANIMS                  = 242
	SERVERPROT_UPDATE_PID                   = 210
	SERVERPROT_LAST_LOGIN_INFO              = 44
	SERVERPROT_LOGOUT                       = 17
	SERVERPROT_P_COUNTDIALOG                = 152
	SERVERPROT_SET_MULTIWAY                 = 97
	SERVERPROT_REBUILD_NORMAL               = 165
	SERVERPROT_VARP_SMALL                   = 236
	SERVERPROT_VARP_LARGE                   = 226
	SERVERPROT_RESET_CLIENT_VARCACHE        = 87
	SERVERPROT_SYNTH_SOUND                  = 151
	SERVERPROT_MIDI_SONG                    = 240
	SERVERPROT_MIDI_JINGLE                  = 173 // NEW in 244 (no 225 Go branch)
	SERVERPROT_UPDATE_ZONE_PARTIAL_FOLLOWS  = 94
	SERVERPROT_UPDATE_ZONE_FULL_FOLLOWS     = 131
	SERVERPROT_UPDATE_ZONE_PARTIAL_ENCLOSED = 233
	SERVERPROT_LOC_MERGE                    = 29
	SERVERPROT_LOC_ANIM                     = 155
	SERVERPROT_OBJ_DEL                      = 39
	SERVERPROT_OBJ_REVEAL                   = 69
	SERVERPROT_LOC_ADD_CHANGE               = 232
	SERVERPROT_MAP_PROJANIM                 = 137
	SERVERPROT_LOC_DEL                      = 125
	SERVERPROT_OBJ_COUNT                    = 209
	SERVERPROT_MAP_ANIM                     = 198
	SERVERPROT_OBJ_ADD                      = 234
)
