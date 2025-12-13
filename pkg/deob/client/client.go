package client

import (
	"math"
	"math/big"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"goscape-client/pkg/jagex2/client"
	"goscape-client/pkg/jagex2/config/component"
	"goscape-client/pkg/jagex2/config/flotype"
	"goscape-client/pkg/jagex2/config/idktype"
	"goscape-client/pkg/jagex2/config/loctype"
	"goscape-client/pkg/jagex2/config/npctype"
	"goscape-client/pkg/jagex2/config/objtype"
	"goscape-client/pkg/jagex2/config/seqtype"
	"goscape-client/pkg/jagex2/config/spotanimtype"
	"goscape-client/pkg/jagex2/config/varptype"
	"goscape-client/pkg/jagex2/dash3d"
	"goscape-client/pkg/jagex2/dash3d/entity"
	"goscape-client/pkg/jagex2/dash3d/entity/playerentity"
	"goscape-client/pkg/jagex2/dash3d/world"
	"goscape-client/pkg/jagex2/dash3d/world3d"
	"goscape-client/pkg/jagex2/datastruct"
	"goscape-client/pkg/jagex2/graphics/animbase"
	"goscape-client/pkg/jagex2/graphics/animframe"
	"goscape-client/pkg/jagex2/graphics/model"
	"goscape-client/pkg/jagex2/graphics/pix2d"
	"goscape-client/pkg/jagex2/graphics/pix32"
	"goscape-client/pkg/jagex2/graphics/pix3d"
	"goscape-client/pkg/jagex2/graphics/pix8"
	"goscape-client/pkg/jagex2/graphics/pixfont"
	"goscape-client/pkg/jagex2/graphics/pixmap"
	"goscape-client/pkg/jagex2/io"
	"goscape-client/pkg/sign/signlink"
)

var (
	CycleLogic2     int
	OpLogic3        int
	CHARSET         string = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!\"£$%^&*()-_=+[{]};:'@#~,<.>/?\\| "
	LevelExperience []int  = make([]int, 99)
	NodeID          int
	Members         bool
	RSA_EXPONENT    *big.Int
	Field1307       [][]int
	RSA_MODULUS     *big.Int
	Field1438       []int
	OpLogic5        int
	OpLogic1        int
	OpLogic4        int
	OpLogic6        int
	OpLogic2        int
	OpLogic9        int
	CycleLogic1     int
	PortOffset      int
	OpLogic8        int
	CycleLogic6     int
	OpLogic7        int
	LoopCycle       int
	CycleLogic3     int
	CycleLogic4     int
	CycleLogic5     int
	LowMemory       bool
	Started         bool
)

type Client struct {
	client.GameShell

	HintTileZ                 int
	HintHeight                int
	HintOffsetX               int
	HintOffsetZ               int
	MinimapOffsetCycle        int
	RedrawBackground          bool
	LocList                   *datastruct.LinkList[*entity.LocEntity]
	RandomIn                  *io.Isaac
	CameraModifierEnabled     []bool
	PrivateChatSetting        int
	SelectedTab               int
	BFSCost                   [][]int
	SocialAction              int
	SceneBaseTileX            int
	SceneBaseTileZ            int
	MapLastBaseX              int
	MapLastBaseZ              int
	SocialInput               string
	MergedLocations           *datastruct.LinkList[*entity.LocMergeEntity]
	IgnoreName37              []int64
	WeightCarried             int
	SceneMapLandData          [][]byte
	Out                       *io.Packet
	StartMidiThread           bool
	ChatEffects               int
	HintNPC                   int
	OverrideChat              int
	SkillLevel                []int
	ChatInterface             *component.Component
	WaveLoops                 []int
	MouseButtonsOption        int
	LocalPID                  int
	DesignColors              []int
	Login                     *io.Packet
	FriendWorld               []int
	MinimapLevel              int
	SocialMessage             string
	ImageHitmarks             []*pix32.Pix32
	ChatbackInput             string
	LastWaveID                int
	UpdateDesignModel         bool
	DesignIdentikits          []int
	ActiveMapFunctions        []*pix32.Pix32
	ChatScrollHeight          int
	In                        *io.Packet
	ArchiveChecksum           []int
	MidiThreadActive          bool
	ImageSideIcons            []*pix8.Pix8
	OrbitCameraPitch          int
	MAX_PLAYER_COUNT          int
	LOCAL_PLAYER_INDEX        int
	Players                   []*playerentity.PlayerEntity
	PlayerIDs                 []int
	EntityUpdateIDs           []int
	PlayerAppearanceBuffer    []*io.Packet
	Projectiles               *datastruct.LinkList[*entity.ProjectileEntity]
	MenuOption                []string
	MidiActive                bool
	DesignGenderMale          bool
	FlameLineOffset           []int
	CompassMaskLineOffsets    []int
	WaveDelay                 []int
	TabInterfaceID            []int
	ErrorLoading              bool
	ShowSocialInput           bool
	PressedContinueOption     bool
	MessageIDs                []int
	MenuVisible               bool
	ReportAbuseMuteOption     bool
	SpawnedLocations          *datastruct.LinkList[*entity.LocAddEntity]
	MessageType               []int
	MessageSender             []string
	MessageText               []string
	FlameActive               bool
	ReportAbuseInterfaceID    int
	ActiveMapFunctionX        []int
	ActiveMapFunctionZ        []int
	TileLastOccupiedCycle     [][]int
	RedrawPrivacySettings     bool
	ErrorHost                 bool
	SkillBaseLevel            []int
	NPCs                      []*entity.NpcEntity
	NPCIDs                    []int
	MinimapZoomModifier       int
	Varps                     []int
	EntityRemovalIDs          []int
	FriendName37              []int64
	MinimapMaskLineLengths    []int
	LevelCollisionMap         []*dash3d.CollisionMap
	ImageHeadIcons            []*pix32.Pix32
	CameraModifierJitter      []int
	ObjGrabThreshold          bool
	RedrawSidebar             bool
	RedrawChatback            bool
	CameraModifierWobbleScale []int
	Cutscene                  bool
	ReportAbuseInput          string
	ViewportInterfaceID       int
	InGame                    bool
	FlamesThread              bool
	SCROLLBAR_GRIP_LOWLIGHT   int
	SCROLLBAR_GRIP_HIGHLIGHT  int
	BFSStepX                  []int
	BFSStepZ                  []int
	//CRC32 CRC32 // TODO
	ChatInterfaceID               int
	ProjectX                      int
	ProjectY                      int
	StickyChatInterfaceID         int
	Rights                        bool
	CameraModifierCycle           []int
	ImageMapscene                 []*pix8.Pix8
	CHAT_COLORS                   []int
	SCROLLBAR_TRACK               int
	ChatbackInputOpen             bool
	Spotanims                     *datastruct.LinkList[*entity.SpotAnimEntity]
	LastWaveLoops                 int
	Username                      string
	Password                      string
	TextureBuffer                 []byte
	ErrorStarted                  bool
	VarCache                      []int
	SkillExperience               []int
	RedrawSideIcons               bool
	LoginMessage0                 string
	LoginMessage1                 string
	MinimapAngleModifier          int
	MAX_CHATS                     int
	ChatX                         []int
	ChatY                         []int
	ChatHeight                    []int
	ChatWidth                     []int
	ChatColors                    []int
	ChatStyles                    []int
	ChatTimers                    []int
	Chats                         []string
	LOC_SHAPE_TO_LAYER            []int
	CompassMaskLineLengths        []int
	BFSDirection                  [][]int
	ImageCrosses                  []*pix32.Pix32
	FlameThread                   bool
	MidiSync                      any
	WaveIDs                       []int
	CameraOffsetXModifier         int
	FriendName                    []string
	FlashingTab                   int
	SidebarInterfaceID            int
	CameraOffsetZModifier         int
	MinimapMaskLineOffsets        []int
	CameraOffsetYawModifier       int
	ChatTyped                     string
	ImageMapFunction              []*pix32.Pix32
	MenuParamB                    []int
	MenuParamC                    []int
	MenuAction                    []int
	MenuParamA                    []int
	ScrollGrabbed                 bool
	WaveEnabled                   bool
	LevelObjStacks                [][][]*datastruct.LinkList[*entity.ObjStackEntity]
	SCROLLBAR_GRIP_FOREGROUND     int
	CameraModifierWobbleSpeed     []int
	MidiSyncLen                   int
	CutsceneSrcLocalTileX         int
	CutsceneSrcLocalTileZ         int
	CutsceneSrcHeight             int
	CutsceneMoveSpeed             int
	CutsceneMoveAcceleration      int
	CrossX                        int
	CrossY                        int
	CrossCycle                    int
	CrossMode                     int
	NextMusicDelay                int
	HintTileX                     int
	PacketSize                    int
	PacketType                    int
	IdleNetCycles                 int
	HeartbeatTimer                int
	IdleTimeout                   int
	CameraOffsetCycle             int
	IgnoreCount                   int
	LastWaveLength                int
	OrbitCameraYaw                int
	OrbitCameraYawVelocity        int
	OrbitCameraPitchVelocity      int
	PlayerCount                   int
	EntityUpdateCount             int
	LastPacketType0               int
	LastPacketType1               int
	LastPacketType2               int
	SplitPrivateChat              int
	SceneCycle                    int
	SceneCenterZoneX              int
	SceneCenterZoneZ              int
	ObjDragInterfaceID            int
	ObjDragSlot                   int
	ObjDragArea                   int
	ObjGrabX                      int
	ObjGrabY                      int
	PrivateMessageCount           int
	ChatHoveredInterfaceIndex     int
	BaseX                         int
	BaseZ                         int
	LastHoveredInterfaceID        int
	DaysSinceLastLogin            int
	FlameGradientCycle0           int
	FlameGradientCycle1           int
	CurrentLevel                  int
	TradeChatSetting              int
	DaysSinceRecoveriesChanged    int
	HintType                      int
	OrbitCameraX                  int
	OrbitCameraZ                  int
	CameraMovedWrite              int
	ActiveMapFunctionCount        int
	ObjDragCycles                 int
	NPCCount                      int
	MinimapZoom                   int
	CameraPitchClamp              int
	WorldLocationState            int
	DragCycles                    int
	EntityRemovalCount            int
	SidebarHoveredInterfaceIndex  int
	SelectedCycle                 int
	SelectedInterface             int
	SelectedItem                  int
	SelectedArea                  int
	CutsceneDstLocalTileX         int
	CutsceneDstLocalTileZ         int
	CutsceneDstHeight             int
	CutsceneRotateSpeed           int
	CutsceneRotateAcceleration    int
	SystemUpdateTimer             int
	MidiSyncCRC                   int
	SceneDelta                    int
	TitleLoginField               int
	PublicChatSetting             int
	ChatScrollOffset              int
	InMultizone                   int
	TryMoveNearest                int
	ObjSelected                   int
	ObjSelectedSlot               int
	ObjSelectedInterface          int
	ObjInterface                  int
	WaveCount                     int
	SpellSelected                 int
	ActiveSpellID                 int
	ActiveSpellFlags              int
	FlagSceneTileX                int
	FlagSceneTileZ                int
	UnreadMessages                int
	LastAddress                   int
	ViewportHoveredInterfaceIndex int
	Energy                        int
	MenuSize                      int
	HintPlayer                    int
	SceneState                    int
	MinimapAnticheatAngle         int
	HoveredSlot                   int
	HoveredSlotParentID           int
	FriendCount                   int
	ChatCount                     int
	WildernessLevel               int
	TitleScreenState              int
	MidiCRC                       int
	CameraX                       int
	CameraY                       int
	CameraZ                       int
	CameraPitch                   int
	CameraYaw                     int
	CameraAnticheatOffsetX        int
	CameraAnticheatOffsetZ        int
	CameraAnticheatAngle          int
	MenuArea                      int
	MenuX                         int
	MenuY                         int
	MenuWidth                     int
	MenuHeight                    int
	ScrollInputPadding            int
	MidiSize                      int
	FlameCycle0                   int
	LastWaveStartTime             int64
	SocialName37                  int64
	ServerSeed                    int64
	Scene                         *world3d.World3D
	LocalPlayer                   *playerentity.PlayerEntity
	GenderButtonImage0            *pix32.Pix32
	GenderButtonImage1            *pix32.Pix32
	ImageFlamesLeft               *pix32.Pix32
	ImageFlamesRight              *pix32.Pix32
	ImageMapflag                  *pix32.Pix32
	ImageMinimap                  *pix32.Pix32
	ImageMapdot0                  *pix32.Pix32
	ImageMapdot1                  *pix32.Pix32
	ImageMapdot2                  *pix32.Pix32
	ImageMapdot3                  *pix32.Pix32
	ImageCompass                  *pix32.Pix32
	ImageRedstone1                *pix8.Pix8
	ImageRedstone2                *pix8.Pix8
	ImageRedstone3                *pix8.Pix8
	ImageRedstone1h               *pix8.Pix8
	ImageRedstone2h               *pix8.Pix8
	ImageBackbase1                *pix8.Pix8
	ImageBackbase2                *pix8.Pix8
	ImageBackhmid1                *pix8.Pix8
	ImageInvback                  *pix8.Pix8
	ImageMapback                  *pix8.Pix8
	ImageChatback                 *pix8.Pix8
	ImageRedstone1v               *pix8.Pix8
	ImageRedstone2v               *pix8.Pix8
	ImageRedstone3v               *pix8.Pix8
	ImageRedstone1hv              *pix8.Pix8
	ImageRedstone2hv              *pix8.Pix8
	ImageScrollbar0               *pix8.Pix8
	ImageScrollbar1               *pix8.Pix8
	ImageTitlebox                 *pix8.Pix8
	ImageTitleButton              *pix8.Pix8
	FontPlain11                   *pixfont.PixFont
	FontPlain12                   *pixfont.PixFont
	FontBold12                    *pixfont.PixFont
	FontQuill8                    *pixfont.PixFont
	AreaBackbase1                 *pixmap.PixMap
	AreaBackbase2                 *pixmap.PixMap
	AreaBackhmid1                 *pixmap.PixMap
	AreaBackleft1                 *pixmap.PixMap
	AreaBackleft2                 *pixmap.PixMap
	AreaBackright1                *pixmap.PixMap
	AreaBackright2                *pixmap.PixMap
	AreaBacktop1                  *pixmap.PixMap
	AreaBacktop2                  *pixmap.PixMap
	AreaBackvmid1                 *pixmap.PixMap
	AreaBackvmid2                 *pixmap.PixMap
	AreaBackvmid3                 *pixmap.PixMap
	AreaBackhmid2                 *pixmap.PixMap
	ImageTitle2                   *pixmap.PixMap
	ImageTitle3                   *pixmap.PixMap
	ImageTitle4                   *pixmap.PixMap
	ImageTitle0                   *pixmap.PixMap
	ImageTitle1                   *pixmap.PixMap
	ImageTitle5                   *pixmap.PixMap
	ImageTitle6                   *pixmap.PixMap
	ImageTitle7                   *pixmap.PixMap
	ImageTitle8                   *pixmap.PixMap
	AreaSidebar                   *pixmap.PixMap
	AreaMapback                   *pixmap.PixMap
	AreaViewport                  *pixmap.PixMap
	AreaChatback                  *pixmap.PixMap
	ArchiveTitle                  *io.Jagfile
	//Stream ClientStream // TODO
	ModalMessage        string
	ObjSelectedName     string
	SpellCaption        string
	MidiSyncName        string
	CurrentMidi         string
	AreaChatbackOffsets []int
	AreaSidebarOffsets  []int
	AreaViewportOffsets []int
	FlameBuffer0        []int
	FlameBuffer1        []int
	FlameGradient       []int
	FlameGradient0      []int
	FlameGradient1      []int
	FlameGradient2      []int
	SceneMapIndex       []int
	FlameBuffer3        []int
	FlameBuffer2        []int
	ImageRunes          []*pix8.Pix8
	SceneMapLocData     [][]byte
	LevelTileFlags      [][][]byte
	LevelHeightmap      [][][]int
}

func NewClient() *Client {
	c := &Client{
		LocList:                   datastruct.NewLinkList[*entity.LocEntity](),
		CameraModifierEnabled:     make([]bool, 5),
		MergedLocations:           datastruct.NewLinkList[*entity.LocMergeEntity](),
		IgnoreName37:              make([]int64, 100),
		Out:                       io.Alloc(1),
		SkillLevel:                make([]int, 50),
		ChatInterface:             component.NewComponent(),
		WaveLoops:                 make([]int, 50),
		LocalPID:                  -1,
		DesignColors:              make([]int, 5),
		Login:                     io.Alloc(1),
		FriendWorld:               make([]int, 100),
		MinimapLevel:              -1,
		ImageHitmarks:             make([]*pix32.Pix32, 20),
		LastWaveID:                -1,
		DesignIdentikits:          make([]int, 7),
		ActiveMapFunctions:        make([]*pix32.Pix32, 1000),
		ChatScrollHeight:          78,
		In:                        io.Alloc(1),
		ArchiveChecksum:           make([]int, 9),
		MidiThreadActive:          true,
		ImageSideIcons:            make([]*pix8.Pix8, 13),
		OrbitCameraPitch:          128,
		MAX_PLAYER_COUNT:          2048,
		LOCAL_PLAYER_INDEX:        2047,
		Projectiles:               datastruct.NewLinkList[*entity.ProjectileEntity](),
		MenuOption:                make([]string, 500),
		MidiActive:                true,
		DesignGenderMale:          true,
		FlameLineOffset:           make([]int, 256),
		CompassMaskLineOffsets:    make([]int, 33),
		WaveDelay:                 make([]int, 50),
		TabInterfaceID:            []int{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		MessageIDs:                make([]int, 100),
		SpawnedLocations:          datastruct.NewLinkList[*entity.LocAddEntity](),
		MessageType:               make([]int, 100),
		MessageSender:             make([]string, 100),
		MessageText:               make([]string, 100),
		ReportAbuseInterfaceID:    -1,
		ActiveMapFunctionX:        make([]int, 1000),
		ActiveMapFunctionZ:        make([]int, 1000),
		SkillBaseLevel:            make([]int, 50),
		NPCs:                      make([]*entity.NpcEntity, 8192),
		NPCIDs:                    make([]int, 8192),
		MinimapZoomModifier:       1,
		Varps:                     make([]int, 2000),
		EntityRemovalIDs:          make([]int, 1000),
		FriendName37:              make([]int64, 100),
		MinimapMaskLineLengths:    make([]int, 151),
		LevelCollisionMap:         make([]*dash3d.CollisionMap, 4),
		ImageHeadIcons:            make([]*pix32.Pix32, 20),
		CameraModifierJitter:      make([]int, 5),
		CameraModifierWobbleScale: make([]int, 5),
		ViewportInterfaceID:       -1,
		SCROLLBAR_GRIP_LOWLIGHT:   3353893,
		SCROLLBAR_GRIP_HIGHLIGHT:  7759444,
		BFSStepX:                  make([]int, 4000),
		BFSStepZ:                  make([]int, 4000),
		// TODO: crc32
		ChatInterfaceID:        -1,
		ProjectX:               -1,
		ProjectY:               -1,
		StickyChatInterfaceID:  -1,
		CameraModifierCycle:    make([]int, 5),
		ImageMapscene:          make([]*pix8.Pix8, 50),
		CHAT_COLORS:            []int{16776960, 16711680, 65280, 65535, 16711935, 16777215},
		SCROLLBAR_TRACK:        2301979,
		Spotanims:              datastruct.NewLinkList[*entity.SpotAnimEntity](),
		LastWaveLoops:          -1,
		TextureBuffer:          make([]byte, 16384),
		VarCache:               make([]int, 2000),
		SkillExperience:        make([]int, 50),
		MinimapAngleModifier:   2,
		MAX_CHATS:              50,
		LOC_SHAPE_TO_LAYER:     []int{0, 0, 0, 0, 1, 1, 1, 1, 1, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 3},
		CompassMaskLineLengths: make([]int, 33),
		ImageCrosses:           make([]*pix32.Pix32, 8),
		//MidiSync: // TODO
		WaveIDs:                   make([]int, 50),
		CameraOffsetXModifier:     2,
		FriendName:                make([]string, 100),
		FlashingTab:               -1,
		SidebarInterfaceID:        -1,
		CameraOffsetZModifier:     2,
		MinimapMaskLineOffsets:    make([]int, 151),
		CameraOffsetYawModifier:   1,
		ImageMapFunction:          make([]*pix32.Pix32, 50),
		MenuParamB:                make([]int, 500),
		MenuParamC:                make([]int, 500),
		MenuAction:                make([]int, 500),
		MenuParamA:                make([]int, 500),
		WaveEnabled:               true,
		SCROLLBAR_GRIP_FOREGROUND: 5063219,
		CameraModifierWobbleSpeed: make([]int, 5),
	}
	c.BFSCost = make([][]int, 104)
	for i := range c.BFSCost {
		c.BFSCost[i] = make([]int, 104)
	}
	c.Players = make([]*playerentity.PlayerEntity, c.MAX_PLAYER_COUNT)
	c.PlayerIDs = make([]int, c.MAX_PLAYER_COUNT)
	c.EntityUpdateIDs = make([]int, c.MAX_PLAYER_COUNT)
	c.PlayerAppearanceBuffer = make([]*io.Packet, c.MAX_PLAYER_COUNT)

	c.TileLastOccupiedCycle = make([][]int, 104)
	for i := range c.TileLastOccupiedCycle {
		c.TileLastOccupiedCycle[i] = make([]int, 104)
	}

	c.ChatX = make([]int, c.MAX_CHATS)
	c.ChatY = make([]int, c.MAX_CHATS)
	c.ChatHeight = make([]int, c.MAX_CHATS)
	c.ChatWidth = make([]int, c.MAX_CHATS)
	c.ChatColors = make([]int, c.MAX_CHATS)
	c.ChatStyles = make([]int, c.MAX_CHATS)
	c.ChatTimers = make([]int, c.MAX_CHATS)
	c.Chats = make([]string, c.MAX_CHATS)

	c.BFSDirection = make([][]int, 104)
	for i := range c.BFSDirection {
		c.BFSDirection[i] = make([]int, 104)
	}

	c.LevelObjStacks = make([][][]*datastruct.LinkList[*entity.ObjStackEntity], 4)
	for i := range c.LevelObjStacks {
		c.LevelObjStacks[i] = make([][]*datastruct.LinkList[*entity.ObjStackEntity], 104)
		for j := range c.LevelObjStacks[i] {
			c.LevelObjStacks[i][j] = make([]*datastruct.LinkList[*entity.ObjStackEntity], 104)
		}
	}

	return c
}

func (c *Client) SetMidi(crc int, name string, length int) {
	if name == "" {
		return
	}
	// TODO: synchronized midiSync
	c.MidiSyncName = name
	c.MidiSyncCRC = crc
	c.MidiSyncLen = length
	// END synchronized
}

func (c *Client) Draw2DEntityElements() {
	c.ChatCount = 0
	var4 := 0
	for i := -1; i < c.PlayerCount+c.NPCCount; i++ {
		var var3 entity.PathableEntity
		if i == -1 {
			var3 = c.LocalPlayer
		} else if i < c.PlayerCount {
			var3 = c.Players[c.PlayerIDs[i]]
		} else {
			var3 = c.NPCs[c.NPCIDs[i-c.PlayerCount]]
		}
		if var3 != nil && var3.IsVisible() {
			var5 := var3.(*playerentity.PlayerEntity) // mine - moved here from below
			if i < c.PlayerCount {
				var4 = 30
				if var5.HeadIcons != 0 {
					c.ProjectFromGround(var5.Height+15, var5)
					if c.ProjectX > -1 {
						for j := range 8 {
							if var5.HeadIcons&0x1<<j != 0 {
								c.ImageHeadIcons[j].Draw(c.ProjectY-var4, c.ProjectX-12)
								var4 -= 25
							}
						}
					}
				}
				if i >= 0 && c.HintType == 10 && c.HintPlayer == c.PlayerIDs[i] {
					c.ProjectFromGround(var5.Height+15, var5)
					if c.ProjectX > -1 {
						c.ImageHeadIcons[7].Draw(c.ProjectY-var4, c.ProjectX-12)
					}
				}
			} else if c.HintType == 1 && c.HintNPC == c.NPCIDs[i-c.PlayerCount] && LoopCycle%20 < 10 {
				c.ProjectFromGround(var5.Height+15, var5)
				if c.ProjectX > -1 {
					c.ImageHeadIcons[2].Draw(c.ProjectY-28, c.ProjectX-12)
				}
			}
			if var5.Chat != "" && (i >= c.PlayerCount || c.PublicChatSetting == 0 || c.PublicChatSetting == 3 || c.PublicChatSetting == 1 && c.IsFriend(var5.Name)) {
				c.ProjectFromGround(var5.Height, var5)
				if c.ProjectX > -1 && c.ChatCount < c.MAX_CHATS {
					c.ChatWidth[c.ChatCount] = c.FontBold12.StringWidth(var5.Chat) / 2
					c.ChatHeight[c.ChatCount] = c.FontBold12.Height
					c.ChatX[c.ChatCount] = c.ProjectX
					c.ChatY[c.ChatCount] = c.ProjectY
					c.ChatColors[c.ChatCount] = var5.ChatColor
					c.ChatStyles[c.ChatCount] = var5.ChatStyle
					c.ChatTimers[c.ChatCount] = var5.ChatTimer
					c.Chats[c.ChatCount] = var5.Chat
					c.ChatCount++
					if c.ChatEffects == 0 && var5.ChatStyle == 1 {
						c.ChatHeight[c.ChatCount] += 10
						c.ChatY[c.ChatCount] += 5
					}
					if c.ChatEffects == 0 && var5.ChatStyle == 2 {
						c.ChatWidth[c.ChatCount] = 60
					}
				}
			}
			if var5.CombatCycle > LoopCycle+100 {
				c.ProjectFromGround(var5.Height+15, var5)
				if c.ProjectX > -1 {
					var4 = var5.Health * 30 / var5.TotalHealth
					if var4 > 30 {
						var4 = 30
					}
					pix2d.FillRect(c.ProjectY-3, c.ProjectX-15, 65280, var4, 5)
					pix2d.FillRect(c.ProjectY-3, c.ProjectX-15+var4, 16711680, 30-var4, 5)
				}
			}
			if var5.CombatCycle > LoopCycle+330 {
				c.ProjectFromGround(var5.Height/2, var5)
				if c.ProjectX > -1 {
					c.ImageHitmarks[var5.DamageType].Draw(c.ProjectY-12, c.ProjectX-12)
					c.FontPlain11.DrawStringCenter(c.ProjectY+4, 0, strconv.Itoa(var5.Damage), c.ProjectX)
					c.FontPlain11.DrawStringCenter(c.ProjectY+3, 16777215, strconv.Itoa(var5.Damage), c.ProjectX-1)
				}
			}
		}
	}
	for i := range c.ChatCount {
		var4 = c.ChatX[i]
		var14 := c.ChatY[i]
		var6 := c.ChatWidth[i]
		var7 := c.ChatHeight[i]
		var8 := true
		for var8 {
			var8 = false
			for j := range i {
				if var14+2 > c.ChatY[j]-c.ChatHeight[j] && var14-var7 < c.ChatY[j]+2 && var4-var6 < c.ChatX[j]+c.ChatWidth[j] && var4+var6 > c.ChatX[j]-c.ChatWidth[j] && c.ChatY[j]-c.ChatHeight[j] < var14 {
					var14 = c.ChatY[j] - c.ChatHeight[j]
					var8 = true
				}
			}
		}
		c.ProjectX = c.ChatX[i]
		c.ChatY[i] = var14
		c.ProjectY = c.ChatY[i]
		var15 := c.Chats[i]
		if c.ChatEffects == 0 {
			var10 := 16776960
			if c.ChatColors[i] < 6 {
				var10 = c.CHAT_COLORS[c.ChatColors[i]]
			}
			if c.ChatColors[i] == 6 {
				if c.SceneCycle%20 < 10 {
					var10 = 16711680
				} else {
					var10 = 16776960
				}
			}
			if c.ChatColors[i] == 7 {
				if c.SceneCycle%20 < 10 {
					var10 = 255
				} else {
					var10 = 65535
				}
			}
			if c.ChatColors[i] == 8 {
				if c.SceneCycle%20 < 10 {
					var10 = 45056
				} else {
					var10 = 8454016
				}
			}
			var11 := 0
			if c.ChatColors[i] == 9 {
				var11 = 150 - c.ChatTimers[i]
				if var11 < 50 {
					var10 = var11*1280 + 16711680
				} else if var11 < 100 {
					var10 = 16776960 - (var11-50)*327680
				} else if var11 < 150 {
					var10 = (var11-100)*5 + 65280
				}
			}
			if c.ChatColors[i] == 10 {
				var11 = 150 - c.ChatTimers[i]
				if var11 < 50 {
					var10 = var11*5 + 16711680
				} else if var11 < 100 {
					var10 = 16711935 - (var11-50)*327680
				} else if var11 < 150 {
					var10 = (var11-100)*327680 + 255 - (var11-100)*5
				}
			}
			if c.ChatColors[i] == 11 {
				var11 = 150 - c.ChatTimers[i]
				if var11 < 50 {
					var10 = 16777215 - var11*327685
				} else if var11 < 100 {
					var10 = (var11-50)*327685 + 65280
				} else if var11 < 150 {
					var10 = 16777215 - (var11-100)*327680
				}
			}
			if c.ChatStyles[i] == 0 {
				c.FontBold12.DrawStringCenter(c.ProjectY+1, 0, var15, c.ProjectX)
				c.FontBold12.DrawStringCenter(c.ProjectY, var10, var15, c.ProjectX)
			}
			if c.ChatStyles[i] == 1 {
				c.FontBold12.DrawCenteredWave(c.SceneCycle, c.ProjectX, c.ProjectY+1, 0, var15)
				c.FontBold12.DrawCenteredWave(c.SceneCycle, c.ProjectX, c.ProjectY, var10, var15)
			}
			if c.ChatStyles[i] == 2 {
				var11 = c.FontBold12.StringWidth(var15)
				var12 := (150 - c.ChatTimers[i]) * (var11 + 100) / 150
				pix2d.SetClipping(334, 0, c.ProjectX+50, c.ProjectX-50)
				c.FontBold12.DrawString(c.ProjectX+50-var12, c.ProjectY+1, 0, var15)
				c.FontBold12.DrawString(c.ProjectX+50-var12, c.ProjectY, var10, var15)
				pix2d.ResetClipping()
			}
		} else {
			c.FontBold12.DrawStringCenter(c.ProjectY+1, 0, var15, c.ProjectX)
			c.FontBold12.DrawStringCenter(c.ProjectY, 16776960, var15, c.ProjectX)
		}
	}
}

func (c *Client) CloseInterfaces() {
	c.Out.P1Isaac(231)
	if c.SidebarInterfaceID != -1 {
		c.SidebarInterfaceID = -1
		c.RedrawSidebar = true
		c.PressedContinueOption = false
		c.RedrawSideIcons = true
	}
	if c.ChatInterfaceID != -1 {
		c.ChatInterfaceID = -1
		c.RedrawChatback = true
		c.PressedContinueOption = false
	}
	c.ViewportInterfaceID = -1
}

func (c *Client) StopMidi() {
	signlink.MidiFade = 0
	signlink.Midi = "stop"
}

func (c *Client) DrawWildyLevel() {
	var2 := (c.LocalPlayer.X >> 7) + c.SceneBaseTileX
	var3 := (c.LocalPlayer.Z >> 7) + c.SceneBaseTileZ
	if var2 >= 2944 && var2 < 3392 && var3 >= 3520 && var3 < 6400 {
		c.WildernessLevel = (var3-3520)/8 + 1
	} else if var2 >= 2944 && var2 < 3392 && var3 >= 9920 && var3 < 12800 {
		c.WildernessLevel = (var3-9920)/8 + 1
	} else {
		c.WildernessLevel = 0
	}
	c.WorldLocationState = 0
	if var2 >= 3328 && var2 < 3392 && var3 >= 3200 && var3 < 3264 {
		var4 := var2 & 0x3F
		var5 := var3 & 0x3F
		if var4 >= 4 && var4 <= 29 && var5 >= 44 && var5 <= 58 {
			c.WorldLocationState = 1
		}
		if var4 >= 36 && var4 <= 61 && var5 >= 44 && var5 <= 58 {
			c.WorldLocationState = 1
		}
		if var4 >= 4 && var4 <= 29 && var5 >= 25 && var5 <= 39 {
			c.WorldLocationState = 1
		}
		if var4 >= 36 && var4 <= 61 && var5 >= 25 && var5 <= 39 {
			c.WorldLocationState = 1
		}
		if var4 >= 4 && var4 <= 29 && var5 >= 6 && var5 <= 20 {
			c.WorldLocationState = 1
		}
		if var4 >= 36 && var4 <= 61 && var5 >= 6 && var5 <= 20 {
			c.WorldLocationState = 1
		}
	}
	if c.WorldLocationState == 0 && var2 >= 3328 && var2 <= 3393 && var3 >= 3203 && var3 <= 3325 {
		c.WorldLocationState = 2
	}
	c.OverrideChat = 0
	if var2 >= 3053 && var2 <= 3156 && var3 >= 3056 && var3 <= 3136 {
		c.OverrideChat = 1
	}
	if var2 >= 3072 && var2 <= 3118 && var3 >= 9492 && var3 <= 9535 {
		c.OverrideChat = 1
	}
	if c.OverrideChat == 1 && var2 >= 3139 && var2 <= 3199 && var3 >= 3008 && var3 <= 3062 {
		c.OverrideChat = 0
	}
}

func (c *Client) DrawPrivateMessages() {
	if c.SplitPrivateChat == 0 {
		return
	}
	var2 := c.FontPlain12
	var3 := 0
	if c.SystemUpdateTimer != 0 {
		var3 = 1
	}
	for i := range 100 {
		if c.MessageText[i] != "" {
			var5 := c.MessageType[i]
			var6 := 0
			if (var5 == 3 || var5 == 7) && (var5 == 7 || c.PrivateChatSetting == 0 || c.PrivateChatSetting == 1 && c.IsFriend(c.MessageSender[i])) {
				var6 = 329 - var3*13
				var2.DrawString(4, var6, 0, "From "+c.MessageSender[i]+": "+c.MessageText[i])
				var2.DrawString(4, var6-1, 65535, "From "+c.MessageSender[i]+": "+c.MessageText[i])
				var3++
				if var3 >= 5 {
					return
				}
			}
			if var5 == 5 && c.PrivateChatSetting < 2 {
				var6 = 329 - var3*13
				var2.DrawString(4, var6, 0, c.MessageText[i])
				var2.DrawString(4, var6-1, 65535, c.MessageText[i])
				var3++
				if var3 >= 5 {
					return
				}
			}
			if var5 == 6 && c.PrivateChatSetting < 2 {
				var6 = 329 - var3*13
				var2.DrawString(4, var6, 0, "To "+c.MessageSender[i]+": "+c.MessageText[i])
				var2.DrawString(4, var6-1, 65535, "To "+c.MessageSender[i]+": "+c.MessageText[i])
				var3++
				if var3 >= 5 {
					return
				}
			}
		}
	}
}

func (c *Client) GetNpcPosExtended(arg0 *io.Packet) {
	for i := range c.EntityUpdateCount {
		var5 := c.EntityUpdateIDs[i]
		var6 := c.NPCs[var5]
		var7 := arg0.G1()
		var8 := 0
		if var7&0x2 == 2 {
			var8 = arg0.G2()
			if var8 == 65535 {
				var8 = -1
			}
			if var8 == var6.PrimarySeqID {
				var6.PrimarySeqLoop = 0
			}
			var9 := arg0.G1()
			if var8 == -1 || var6.PrimarySeqID == -1 || seqtype.Instances[var8].Priority > seqtype.Instances[var6.PrimarySeqID].Priority || seqtype.Instances[var6.PrimarySeqID].Priority == 0 {
				var6.PrimarySeqID = var8
				var6.PrimarySeqFrame = 0
				var6.PrimarySeqCycle = 0
				var6.PrimarySeqDelay = var9
				var6.PrimarySeqLoop = 0
			}
		}
		if var7&0x4 == 4 {
			var6.TargetID = arg0.G2()
			if var6.TargetID == 65535 {
				var6.TargetID = -1
			}
		}
		if var7&0x8 == 8 {
			var6.Chat = arg0.GJStr()
			var6.ChatTimer = 100
		}
		if var7&0x10 == 16 {
			var6.Damage = arg0.G1()
			var6.DamageType = arg0.G1()
			var6.CombatCycle = LoopCycle + 400
			var6.Health = arg0.G1()
			var6.TotalHealth = arg0.G1()
		}
		if var7&0x20 == 32 {
			var6.Type = npctype.Get(arg0.G2())
			var6.SeqWalkID = var6.Type.WalkAnim
			var6.SeqTurnAroundID = var6.Type.WalkAnimB
			var6.SeqTurnLeftID = var6.Type.WalkAnimR
			var6.SeqTurnRightId = var6.Type.WalkAnimL
			var6.SeqStandID = var6.Type.ReadyAnim
		}
		if var7&0x40 == 64 {
			var6.SpotanimID = arg0.G2()
			var8 = arg0.G4()
			var6.SpotanimOffset = var8 >> 16
			var6.SpotanimLastCycle = LoopCycle + (var8 & 0xFFFF)
			var6.SpotanimFrame = 0
			var6.SpotanimCycle = 0
			if var6.SpotanimLastCycle > LoopCycle {
				var6.SpotanimFrame = -1
			}
			if var6.SpotanimID == 65535 {
				var6.SpotanimID = -1
			}
		}
		if var7&0x80 == 128 {
			var6.TargetTileX = arg0.G2()
			var6.TargetTileZ = arg0.G2()
		}
	}
}

func (c *Client) AddIgnore(arg0 int64) {
	if arg0 == 0 {
		return
	}
	if c.IgnoreCount >= 100 {
		c.AddMessage(0, "Your ignore list is full. Max of 100 hit", "")
		return
	}
	var4 := datastruct.FormatName(datastruct.FromBase37(arg0))
	for i := range c.IgnoreCount {
		if c.IgnoreName37[i] == arg0 {
			c.AddMessage(0, var4+" is already on your ignore list", "")
			return
		}
	}
	for i := range c.FriendCount {
		if c.FriendName37[i] == arg0 {
			c.AddMessage(0, "Please remove "+var4+" from your friend list first", "")
			return
		}
	}
	c.IgnoreName37[c.IgnoreCount] = arg0
	c.IgnoreCount++
	c.RedrawSidebar = true
	c.Out.P1Isaac(79)
	c.Out.P8(arg0)
}

func (c *Client) ReadZonePacket(arg1 *io.Packet, arg2 int) {
	var4 := 0
	var5 := 0
	var6 := 0
	var7 := 0
	var8 := 0
	var9 := 0
	var10 := 0
	var11 := 0
	var14 := 0
	var15 := 0
	var16 := 0
	if arg2 == 59 || arg2 == 76 {
		var4 = arg1.G1()
		var5 = c.BaseX + (var4 >> 4 & 0x7)
		var6 = c.BaseZ + (var4 & 0x7)
		var7 = arg1.G1()
		var8 = var7 >> 2
		var9 = var7 & 0x3
		var10 = c.LOC_SHAPE_TO_LAYER[var8]
		if arg2 == 76 {
			var11 = -1
		} else {
			var11 = arg1.G2()
		}
		if var5 >= 0 && var6 >= 0 && var5 < 104 && var6 < 104 {
			var var12 *entity.LocAddEntity
			for var13 := c.SpawnedLocations.Head().Value; var13 != nil; var13 = c.SpawnedLocations.Next().Value {
				if var13.Plane == c.CurrentLevel && var13.X == var5 && var13.Z == var6 && var13.Layer == var10 {
					var12 = var13
					break
				}
			}
			if var12 == nil {
				var14 = 0
				var15 = -1
				var16 = 0
				var17 := 0
				if var10 == 0 {
					var14 = c.Scene.GetWallBitSet(c.CurrentLevel, var5, var6)
				}
				if var10 == 1 {
					var14 = c.Scene.GetWallDecorationBitSet(c.CurrentLevel, var6, var5)
				}
				if var10 == 2 {
					var14 = c.Scene.GetLocBitSet(c.CurrentLevel, var5, var6)
				}
				if var10 == 3 {
					var14 = c.Scene.GetGroundDecorationBitSet(c.CurrentLevel, var5, var6)
				}
				if var14 != 0 {
					var18 := c.Scene.GetInfo(c.CurrentLevel, var5, var6, var14)
					var15 = var14 >> 14 & 0x7FFF
					var16 = var18 & 0x1F
					var17 = var18 >> 6
				}
				var12 = entity.NewLocAddEntity()
				var12.Plane = c.CurrentLevel
				var12.Layer = var10
				var12.X = var5
				var12.Z = var6
				var12.LastLocIndex = var15
				var12.LastShape = var16
				var12.LastAngle = var17
				c.SpawnedLocations.AddTail(var12)
			}
			var12.LocIndex = var11
			var12.Shape = var8
			var12.Angle = var9
			c.AddLoc(var9, var5, var6, var10, var11, var8, c.CurrentLevel)
		}
	} else if arg2 == 42 {
		var4 = arg1.G1()
		var5 = c.BaseX + (var4 >> 4 & 0x7)
		var6 = c.BaseZ + (var4 & 0x7)
		var7 = arg1.G1()
		var8 = var7 >> 2
		var9 = c.LOC_SHAPE_TO_LAYER[var8]
		var10 = arg1.G2()
		if var5 >= 0 && var6 >= 0 && var5 < 104 && var6 < 104 {
			var11 = 0
			if var9 == 0 {
				var11 = c.Scene.GetWallBitSet(c.CurrentLevel, var5, var6)
			}
			if var9 == 1 {
				var11 = c.Scene.GetWallDecorationBitSet(c.CurrentLevel, var6, var5)
			}
			if var9 == 2 {
				var11 = c.Scene.GetLocBitSet(c.CurrentLevel, var5, var6)
			}
			if var9 == 3 {
				var11 = c.Scene.GetGroundDecorationBitSet(c.CurrentLevel, var5, var6)
			}
			if var11 != 0 {
				var38 := entity.NewLocEntity(false, var11>>14&0x7FFF, c.CurrentLevel, var9, seqtype.Instances[var10], var6, var5)
				c.LocList.AddTail(var38)
			}
		}
	} else {
		var var32 *entity.ObjStackEntity
		if arg2 == 223 {
			var4 = arg1.G1()
			var5 = c.BaseX + (var4 >> 4 & 0x7)
			var6 = c.BaseZ + (var4 & 0x7)
			var7 = arg1.G2()
			var8 = arg1.G2()
			if var5 >= 0 && var6 >= 0 && var5 < 104 && var6 < 104 {
				var32 = entity.NewObjStackEntity()
				var32.Index = var7
				var32.Count = var8
				if c.LevelObjStacks[c.CurrentLevel][var5][var6] == nil {
					c.LevelObjStacks[c.CurrentLevel][var5][var6] = datastruct.NewLinkList[*entity.ObjStackEntity]()
				}
				c.LevelObjStacks[c.CurrentLevel][var5][var6].AddTail(var32)
				c.SortObjStacks(var5, var6)
			}
		} else if arg2 == 49 {
			var4 = arg1.G1()
			var5 = c.BaseX + (var4 >> 4 & 0x7)
			var6 = c.BaseZ + (var4 & 0x7)
			var7 = arg1.G2()
			if var5 >= 0 && var6 >= 0 && var5 < 104 && var6 < 104 {
				var30 := c.LevelObjStacks[c.CurrentLevel][var5][var6]
				if var30 != nil {
					for var32 = var30.Head(); var32 != nil; var32 = var30.Next() {
						if var32.Index == var7&0x7FFF {
							var32.Unlink()
							break
						}
					}
					if var30.Head() == nil {
						c.LevelObjStacks[c.CurrentLevel][var5][var6] = nil
					}
					c.SortObjStacks(var5, var6)
				}
			}
		} else {
			var36 := 0
			var37 := 0
			if arg2 == 69 {
				var4 = arg1.G1()
				var5 = c.BaseX + (var4 >> 4 & 0x7)
				var6 = c.BaseZ + (var4 & 0x7)
				var7 = var5 + int(arg1.G1B())
				var8 = var6 + int(arg1.G1B())
				var9 = arg1.G2B()
				var10 = arg1.G2()
				var11 = arg1.G1()
				var36 = arg1.G1()
				var37 = arg1.G2()
				var14 = arg1.G2()
				var15 = arg1.G1()
				var16 = arg1.G1()
				if var5 >= 0 && var6 >= 0 && var5 < 104 && var6 < 104 && var7 >= 0 && var8 >= 0 && var7 < 104 && var8 < 104 {
					var5 = var5*128 + 64
					var6 = var6*128 + 64
					var7 = var7*128 + 64
					var8 = var8*128 + 64
					var43 := entity.NewProjectileEntity(var36, var15, var6, var14+LoopCycle, c.CurrentLevel, var9, var37+LoopCycle, var16, c.GetHeightMapY(c.CurrentLevel, var5, var6)-var11, var10, var5)
					var43.UpdateVelocity(c.GetHeightMapY(c.CurrentLevel, var7, var8)-var36, var8, var7, var37+LoopCycle)
					c.Projectiles.AddTail(var43)
				}
			} else if arg2 == 191 {
				var4 = arg1.G1()
				var5 = c.BaseX + (var4 >> 4 & 0x7)
				var6 = c.BaseZ + (var4 & 0x7)
				var7 = arg1.G2()
				var8 = arg1.G1()
				var9 = arg1.G2()
				if var5 >= 0 && var6 >= 0 && var5 < 104 && var6 < 104 {
					var5 = var5*128 + 64
					var6 = var6*128 + 64
					var34 := entity.NewSpotAnimEntity(var5, var7, var6, var9, c.GetHeightMapY(c.CurrentLevel, var5, var6)-var8, c.CurrentLevel, LoopCycle)
					c.Spotanims.AddTail(var34)
				}
			} else if arg2 == 50 {
				var4 = arg1.G1()
				var5 = c.BaseX + (var4 >> 4 & 0x7)
				var6 = c.BaseZ + (var4 & 0x7)
				var7 = arg1.G2()
				var8 = arg1.G2()
				var9 = arg1.G2()
				if var5 >= 0 && var6 >= 0 && var5 < 104 && var6 < 104 && var9 != c.LocalPID {
					var33 := entity.NewObjStackEntity()
					var33.Index = var7
					var33.Count = var8
					if c.LevelObjStacks[c.CurrentLevel][var5][var6] == nil {
						c.LevelObjStacks[c.CurrentLevel][var5][var6] = datastruct.NewLinkList[*entity.ObjStackEntity]()
					}
					c.LevelObjStacks[c.CurrentLevel][var5][var6].AddTail(var33)
					c.SortObjStacks(var5, var6)
				}
			} else {
				if arg2 == 23 {
					var4 = arg1.G1()
					var5 = c.BaseX + (var4 >> 4 & 0x7)
					var6 = c.BaseZ + (var4 & 0x7)
					var7 = arg1.G1()
					var8 = var7 >> 2
					var9 = var7 & 0x3
					var10 = c.LOC_SHAPE_TO_LAYER[var8]
					var11 = arg1.G2()
					var36 = arg1.G2()
					var37 = arg1.G2()
					var14 = arg1.G2()
					var39 := arg1.G1B()
					var40 := arg1.G1B()
					var41 := arg1.G1B()
					var42 := arg1.G1B()
					var var19 *playerentity.PlayerEntity
					if var14 == c.LocalPID {
						var19 = c.LocalPlayer
					} else {
						var19 = c.Players[var14]
					}
					if var19 != nil {
						var20 := entity.NewLocMergeEntity(c.CurrentLevel, var9, var6, var36+LoopCycle, var8, -1, var5, var10)
						c.MergedLocations.AddTail(var20)
						var21 := entity.NewLocMergeEntity(c.CurrentLevel, var9, var6, var37+LoopCycle, var8, var11, var5, var10)
						c.MergedLocations.AddTail(var21)
						var22 := c.LevelHeightmap[c.CurrentLevel][var5][var6]
						var23 := c.LevelHeightmap[c.CurrentLevel][var5+1][var6]
						var24 := c.LevelHeightmap[c.CurrentLevel][var5+1][var6+1]
						var25 := c.LevelHeightmap[c.CurrentLevel][var5][var6+1]
						var26 := loctype.Get(var11)
						var19.LocStartCycle = var36 + LoopCycle
						var19.LocStopCycle = var37 + LoopCycle
						var19.LocModel = var26.GetModel(var8, var9, var22, var23, var24, var25, -1)
						var27 := var26.Width
						var28 := var26.Length
						if var9 == 1 || var9 == 3 {
							var27 = var26.Length
							var28 = var26.Width
						}
						var19.LocOffsetX = var5*128 + var27*64
						var19.LocOffsetZ = var6*128 + var28*64
						var19.LocOffsetY = c.GetHeightMapY(c.CurrentLevel, var19.LocOffsetX, var19.LocOffsetZ)
						var29 := byte(0)
						if var39 > var41 {
							var29 = var39
							var39 = var41
							var41 = var29
						}
						if var40 > var42 {
							var29 = var40
							var40 = var42
							var42 = var29
						}
						var19.MinTileX = var5 + int(var39)
						var19.MaxTileX = var5 + int(var41)
						var19.MinTileZ = var6 + int(var40)
						var19.MaxTileZ = var6 + int(var42)
					}
				}
				if arg2 == 151 {
					var4 = arg1.G1()
					var5 = c.BaseX + (var4 >> 4 & 0x7)
					var6 = c.BaseZ + (var4 & 0x7)
					var7 = arg1.G2()
					var8 = arg1.G2()
					var9 = arg1.G2()
					if var5 >= 0 && var6 >= 0 && var5 < 104 && var6 < 104 {
						var31 := c.LevelObjStacks[c.CurrentLevel][var5][var6]
						if var31 != nil {
							for var35 := var31.Head(); var35 != nil; var35 = var31.Next() {
								if var35.Index == var7&0x7FFF && var35.Count == var8 {
									var35.Count = var9
									break
								}
							}
							c.SortObjStacks(var5, var6)
						}
					}
				}
			}
		}
	}
}

func (c *Client) GetTopLevel() int {
	var2 := 3
	if c.CameraPitch < 310 {
		var3 := c.CameraX >> 7
		var4 := c.CameraZ >> 7
		var5 := c.LocalPlayer.X >> 7
		var6 := c.LocalPlayer.Z >> 7
		if c.LevelTileFlags[c.CurrentLevel][var3][var4]&0x4 != 0 {
			var2 = c.CurrentLevel
		}
		var7 := 0
		if var5 > var3 {
			var7 = var5 - var3
		} else {
			var7 = var3 - var5
		}
		var8 := 0
		if var6 > var4 {
			var8 = var6 - var4
		} else {
			var8 = var4 - var6
		}
		var9 := 0
		var10 := 0
		if var7 > var8 {
			var9 = var8 * 65536 / var7
			var10 = 32768
			for var3 != var5 {
				if var3 < var5 {
					var3++
				} else if var3 > var5 {
					var3--
				}
				if c.LevelTileFlags[c.CurrentLevel][var3][var4]&0x4 != 0 {
					var2 = c.CurrentLevel
				}
				var10 += var9
				if var10 >= 65536 {
					var10 -= 65536
					if var4 < var6 {
						var4++
					} else if var4 > var6 {
						var4--
					}
					if c.LevelTileFlags[c.CurrentLevel][var3][var4]&0x4 != 0 {
						var2 = c.CurrentLevel
					}
				}
			}
		} else {
			var9 = var7 * 65536 / var8
			var10 = 32768
			for var4 != var6 {
				if var4 < var6 {
					var4++
				} else if var4 > var6 {
					var4--
				}
				if c.LevelTileFlags[c.CurrentLevel][var3][var4]&0x4 != 0 {
					var2 = c.CurrentLevel
				}
				var10 += var9
				if var10 >= 65536 {
					var10 -= 65536
					if var3 < var5 {
						var3++
					} else if var3 > var5 {
						var3--
					}
					if c.LevelTileFlags[c.CurrentLevel][var3][var4]&0x4 != 0 {
						var2 = c.CurrentLevel
					}
				}
			}
		}
	}
	if c.LevelTileFlags[c.CurrentLevel][c.LocalPlayer.X>>7][c.LocalPlayer.Z>>7]&0x4 != 0 {
		var2 = c.CurrentLevel
	}
	return var2
}

func (c *Client) GetTopLevelCutscene(arg0 int) int {
	var2 := c.GetHeightMapY(c.CurrentLevel, c.CameraX, c.CameraZ)
	c.PacketSize += arg0
	if var2-c.CameraYaw >= 800 || c.LevelTileFlags[c.CurrentLevel][c.CameraX>>7][c.CameraZ>>7]&0x4 == 0 {
		return 3
	}
	return c.CurrentLevel
}

func (c *Client) DrawScene(arg0 int) {
	c.SceneCycle++
	c.PushPlayers()
	c.PushNPCs()
	c.PacketSize += arg0
	c.PushProjectiles()
	c.PushSpotanims()
	c.PushLocs()
	var2 := 0
	var3 := 0
	var4 := 0
	if !c.Cutscene {
		var2 = c.OrbitCameraPitch
		if c.CameraPitchClamp/256 > var2 {
			var2 = c.CameraPitchClamp / 256
		}
		if c.CameraModifierEnabled[4] && c.CameraModifierWobbleScale[4]+128 > var2 {
			var2 = c.CameraModifierWobbleScale[4] + 128
		}
		var3 = c.OrbitCameraYaw + c.CameraAnticheatAngle&0x7FF
		c.OrbitCamera(c.GetHeightMapY(c.CurrentLevel, c.LocalPlayer.X, c.LocalPlayer.Z)-50, c.OrbitCameraX, var3, var2, c.OrbitCameraZ, var2*3+600)
		CycleLogic2++
		if CycleLogic2 > 1802 {
			CycleLogic2 = 0
			c.Out.P1Isaac(146)
			c.Out.P1(0)
			var4 = c.Out.Pos
			c.Out.P2(29711)
			c.Out.P1(70)
			c.Out.P1(int(rand.Float64() * 256.0))
			c.Out.P1(242)
			c.Out.P1(186)
			c.Out.P1(39)
			c.Out.P1(61)
			if int(rand.Float64()*2.0) == 0 {
				c.Out.P1(13)
			}
			if int(rand.Float64()*2.0) == 0 {
				c.Out.P2(57856)
			}
			c.Out.P2(int(rand.Float64() * 65536.0))
			c.Out.PSize1(c.Out.Pos - var4)
		}
	}
	if c.Cutscene {
		var2 = c.GetTopLevelCutscene(0)
	} else {
		var2 = c.GetTopLevel()
	}
	var3 = c.CameraX
	var4 = c.CameraY
	var5 := c.CameraZ
	var6 := c.CameraPitch
	var7 := c.CameraYaw
	var9 := 0
	for i := range 5 {
		if c.CameraModifierEnabled[i] {
			var9 = int(rand.Float64()*float64(c.CameraModifierJitter[i]*2+1) - float64(c.CameraModifierJitter[i]) + math.Sin(float64(c.CameraModifierCycle[i])*(float64(c.CameraModifierWobbleSpeed[i])/100.0))*float64(c.CameraModifierWobbleScale[i]))
			if i == 0 {
				c.CameraX += var9
			}
			if i == 1 {
				c.CameraY += var9
			}
			if i == 2 {
				c.CameraZ += var9
			}
			if i == 3 {
				c.CameraYaw = c.CameraYaw + var9&0x7FF
			}
			if i == 4 {
				c.CameraPitch += var9
				if c.CameraPitch < 128 {
					c.CameraPitch = 128
				}
				if c.CameraPitch > 383 {
					c.CameraPitch = 383
				}
			}
		}
	}
	var9 = pix3d.Cycle
	model.CheckHover = true
	model.PickedCount = 0
	model.MouseX = c.MouseX - 8
	model.MouseZ = c.MouseY - 11
	pix2d.Clear()
	c.Scene.Draw(c.CameraYaw, c.CameraX, var2, c.CameraPitch, c.CameraY, c.CameraZ)
	c.Scene.ClearTemporaryLocs()
	c.Draw2DEntityElements()
	c.DrawTileHint()
	c.UpdateTextures(var9)
	c.Draw3DEntityElements()
	//c.AreaViewport // TODO: pixmap
	c.CameraX = var3
	c.CameraY = var4
	c.CameraZ = var5
	c.CameraPitch = var6
	c.CameraYaw = var7
}

func (c *Client) RunMidi() {
	// TODO
}

func SetLowMemory() {
	world3d.LowMemory = true
	pix3d.LowDetail = true
	LowMemory = true
	world.LowMemory = true
}

func (c *Client) DrawFlames() {
	var2 := 256
	//var3 := 0
	if c.FlameGradientCycle0 > 0 {
		for i := range 256 {
			if c.FlameGradientCycle0 > 768 {
				c.FlameGradient[i] = c.Mix(c.FlameGradient0[i], 1024-c.FlameGradientCycle0, c.FlameGradient1[i])
			} else if c.FlameGradientCycle0 > 256 {
				c.FlameGradient[i] = c.FlameGradient1[i]
			} else {
				c.FlameGradient[i] = c.Mix(c.FlameGradient1[i], 256-c.FlameGradientCycle0, c.FlameGradient0[i])
			}
		}
	} else if c.FlameGradientCycle1 > 0 {
		for i := range 256 {
			if c.FlameGradientCycle1 > 768 {
				c.FlameGradient[i] = c.Mix(c.FlameGradient0[i], 1024-c.FlameGradientCycle1, c.FlameGradient2[i])
			} else if c.FlameGradientCycle1 > 256 {
				c.FlameGradient[i] = c.FlameGradient2[i]
			} else {
				c.FlameGradient[i] = c.Mix(c.FlameGradient2[i], 256-c.FlameGradientCycle1, c.FlameGradient0[i])
			}
		}
	} else {
		for i := range 256 {
			c.FlameGradient[i] = c.FlameGradient0[i]
		}
	}
	for range 33920 {
		//c.ImageTitle0 // TODO: pixmap
	}
	var4 := 0
	var5 := 1152
	var7 := 0
	var8 := 0
	var10 := 0
	//var11 := 0
	var12 := 0
	//var13 := 0
	for i := 1; i < var2-1; i++ {
		var7 = c.FlameLineOffset[i] * (var2 - i) / var2
		var8 = var7 + 22
		if var8 < 0 {
			var8 = 0
		}
		var4 += var8
		for range 128 {
			var10 = c.FlameBuffer3[var4]
			var4++
			if var10 == 0 {
				var5++
			} else {
				//var11 = var10
				var12 = 256 - var10
				var10 = c.FlameGradient[var10]
				//var13 = c.ImageTitle0 // TODO: pixmap
				// TODO: pixmap
			}
		}
		var5 += var8
	}
	//c.ImageTitle0 // TODO: pixmap
	for range 33920 {
		// TODO: pixmap
	}
	var4 = 0
	var5 = 1176
	for i := 1; i < var2-1; i++ {
		var9 := c.FlameLineOffset[i] * (var2 - i) / var2
		var10 = 103 - var9
		var5 += var9
		for range var10 {
			var12 = c.FlameBuffer3[var4]
			var4++
			if var12 == 0 {
				var5++
			} else {
				//var13 = var12
				//var14 := 256-var12
				var12 = c.FlameGradient[var12]
				//var15 = c.ImageTitle1 // TODO: pixmap
				//c.ImageTitle1 // TODO: pixmap
			}
		}
	}
}

func (c *Client) HandleInterfaceInput(arg0, arg1, arg2 int, arg3 *component.Component, arg5 int, arg6 int) {
	if arg3.Type != 0 || arg3.ChildID == nil || arg3.Hide || (arg1 < arg5 || arg0 < arg2 || arg1 > arg5+arg3.Width || arg0 > arg2+arg3.Height) {
		return
	}
	var8 := len(arg3.ChildID)
	for i := range var8 {
		var10 := arg3.ChildX[i] + arg5
		var11 := arg3.ChildY[i] + arg2 - arg6
		var12 := component.Instances[arg3.ChildID[i]]
		var20 := var10 + var12.X
		var21 := var11 + var12.Y
		if (var12.OverLayer >= 0 || var12.OverColour != 0) && arg1 >= var20 && arg0 >= var21 && arg1 < var20+var12.Width && arg0 < var21+var12.Height {
			if var12.OverLayer >= 0 {
				c.LastHoveredInterfaceID = var12.OverLayer
			} else {
				c.LastHoveredInterfaceID = var12.Id
			}
		}
		if var12.Type == 0 {
			c.HandleInterfaceInput(arg0, arg1, var21, var12, var20, var12.ScrollPosition)
			if var12.Scroll > var12.Height {
				c.HandleScrollInput(arg1, 0, arg0, var12.Scroll, var12.Height, true, var20+var12.Width, var21, var12)
			}
		} else {
			if var12.ButtonType == 1 && arg1 >= var20 && arg0 >= var21 && arg1 < var20+var12.Width && arg0 < var21+var12.Height {
				var13 := false
				if var12.ClientCode != 0 {
					var13 = c.HandleSocialMenuOption(var12)
				}
				if !var13 {
					c.MenuOption[c.MenuSize] = var12.Option
					c.MenuAction[c.MenuSize] = 951
					c.MenuParamC[c.MenuSize] = var12.Id
					c.MenuSize++
				}
			}
			if var12.ButtonType == 2 && c.SpellSelected == 0 && arg1 >= var20 && arg0 >= var21 && arg1 < var20+var12.Width && arg0 < var21+var12.Height {
				var22 := var12.ActionVerb
				if strings.Index(var22, " ") != -1 {
					var22 = var22[0:strings.Index(var22, " ")]
				}
				c.MenuOption[c.MenuSize] = var22 + " @gre@" + var12.Action
				c.MenuAction[c.MenuSize] = 930
				c.MenuParamC[c.MenuSize] = var12.Id
				c.MenuSize++
			}
			if var12.ButtonType == 3 && arg1 >= var20 && arg0 >= var21 && arg1 < var20+var12.Width && arg0 < var21+var12.Height {
				c.MenuOption[c.MenuSize] = "Close"
				c.MenuAction[c.MenuSize] = 947
				c.MenuParamC[c.MenuSize] = var12.Id
				c.MenuSize++
			}
			if var12.ButtonType == 4 && arg1 >= var20 && arg0 >= var21 && arg1 < var20+var12.Width && arg0 < var21+var12.Height {
				c.MenuOption[c.MenuSize] = var12.Option
				c.MenuAction[c.MenuSize] = 465
				c.MenuParamC[c.MenuSize] = var12.Id
				c.MenuSize++
			}
			if var12.ButtonType == 5 && arg1 >= var20 && arg0 >= var21 && arg1 < var20+var12.Width && arg0 < var21+var12.Height {
				c.MenuOption[c.MenuSize] = var12.Option
				c.MenuAction[c.MenuSize] = 960
				c.MenuParamC[c.MenuSize] = var12.Id
				c.MenuSize++
			}
			if var12.ButtonType == 6 && !c.PressedContinueOption && arg1 >= var20 && arg0 >= var21 && arg1 < var20+var12.Width && arg0 < var21+var12.Height {
				c.MenuOption[c.MenuSize] = var12.Option
				c.MenuAction[c.MenuSize] = 44
				c.MenuParamC[c.MenuSize] = var12.Id
				c.MenuSize++
			}
			if var12.Type == 2 {
				var23 := 0
				for j := range var12.Height {
					for k := range var12.Width {
						var16 := var20 + k*(var12.MarginX+32)
						var17 := var21 + j*(var12.MarginY+32)
						if var23 < 20 {
							var16 += var12.InvSlotOffsetX[var23]
							var17 += var12.InvSlotOffsetY[var23]
						}
						if arg1 >= var16 && arg0 >= var17 && arg1 < var16+32 && arg0 < var17+32 {
							c.HoveredSlot = var23
							c.HoveredSlotParentID = var12.Id
							if var12.InvSlotObjId[var23] > 0 {
								var18 := objtype.Get(var12.InvSlotObjId[var23] - 1)
								if c.ObjSelected == 1 && var12.Interactable {
									if var12.Id != c.ObjSelectedInterface || var23 != c.ObjSelectedSlot {
										c.MenuOption[c.MenuSize] = "Use " + c.ObjSelectedName + " with @lre@" + var18.Name
										c.MenuAction[c.MenuSize] = 881
										c.MenuParamA[c.MenuSize] = var18.Index
										c.MenuParamB[c.MenuSize] = var23
										c.MenuParamC[c.MenuSize] = var12.Id
										c.MenuSize++
									}
								} else if c.SpellSelected != 1 && !var12.Interactable {
									if var12.Interactable {
										for l := 4; l >= 3; l-- {
											if var18.IOp != nil && var18.IOp[l] != nil {
												c.MenuOption[c.MenuSize] = var18.IOp[l] + " @lre@" + var18.Name
												if l == 3 {
													c.MenuAction[c.MenuSize] = 478
												}
												if l == 4 {
													c.MenuAction[c.MenuSize] = 347
												}
												c.MenuParamA[c.MenuSize] = var18.Index
												c.MenuParamB[c.MenuSize] = var23
												c.MenuParamC[c.MenuSize] = var12.Id
												c.MenuSize++
											} else if l == 4 {
												c.MenuOption[c.MenuSize] = "Drop @lre@" + var18.Name
												c.MenuAction[c.MenuSize] = 347
												c.MenuParamA[c.MenuSize] = var18.Index
												c.MenuParamB[c.MenuSize] = var23
												c.MenuParamC[c.MenuSize] = var12.Id
												c.MenuSize++
											}
										}
									}
									if var12.Usable {
										c.MenuOption[c.MenuSize] = "Use @lre@" + var18.Name
										c.MenuAction[c.MenuSize] = 188
										c.MenuParamA[c.MenuSize] = var18.Index
										c.MenuParamB[c.MenuSize] = var23
										c.MenuParamC[c.MenuSize] = var12.Id
										c.MenuSize++
									}
									if var12.Interactable && var18.IOp != nil {
										for l := 2; l >= 0; l-- {
											if var18.IOp[l] != nil {
												c.MenuOption[c.MenuSize] = var18.IOp[l] + " @lre@" + var18.Name
												if l == 0 {
													c.MenuAction[c.MenuSize] = 405
												}
												if l == 1 {
													c.MenuAction[c.MenuSize] = 38
												}
												if l == 2 {
													c.MenuAction[c.MenuSize] = 422
												}
												c.MenuParamA[c.MenuSize] = var18.Index
												c.MenuParamB[c.MenuSize] = var23
												c.MenuParamC[c.MenuSize] = var12.Id
												c.MenuSize++
											}
										}
									}
									if var12.IOps != nil {
										for l := 4; l >= 0; l-- {
											if var12.IOps[l] != "" {
												c.MenuOption[c.MenuSize] = var12.IOps[l] + " @lre@" + var18.Name
												if l == 0 {
													c.MenuAction[c.MenuSize] = 602
												}
												if l == 1 {
													c.MenuAction[c.MenuSize] = 596
												}
												if l == 2 {
													c.MenuAction[c.MenuSize] = 22
												}
												if l == 3 {
													c.MenuAction[c.MenuSize] = 892
												}
												if l == 4 {
													c.MenuAction[c.MenuSize] = 415
												}
												c.MenuParamA[c.MenuSize] = var18.Index
												c.MenuParamB[c.MenuSize] = var23
												c.MenuParamC[c.MenuSize] = var12.Id
												c.MenuSize++
											}
										}
									}
									c.MenuOption[c.MenuSize] = "Examine @lre@" + var18.Name
									c.MenuAction[c.MenuSize] = 1773
									c.MenuParamA[c.MenuSize] = var18.Index
									c.MenuParamC[c.MenuSize] = var12.InvSlotObjCount[var23]
									c.MenuSize++
								} else if c.ActiveSpellFlags&0x10 == 16 {
									c.MenuOption[c.MenuSize] = c.SpellCaption + " @lre@" + var18.Name
									c.MenuAction[c.MenuSize] = 391
									c.MenuParamA[c.MenuSize] = var18.Index
									c.MenuParamB[c.MenuSize] = var23
									c.MenuParamC[c.MenuSize] = var12.Id
									c.MenuSize++
								}
							}
						}
						var23++
					}
				}
			}
		}
	}
}

func (c *Client) HandleChatSettingsInput(arg0 int) {
	c.PacketSize += arg0
	if c.MouseClickButton != 1 {
		return
	}
	if c.MouseClickX >= 8 && c.MouseClickX <= 108 && c.MouseClickY >= 490 && c.MouseClickY <= 522 {
		c.PublicChatSetting = (c.PublicChatSetting + 1) % 4
		c.RedrawPrivacySettings = true
		c.RedrawChatback = true
		c.Out.P1Isaac(244)
		c.Out.P1(c.PublicChatSetting)
		c.Out.P1(c.PrivateChatSetting)
		c.Out.P1(c.TradeChatSetting)
	}
	if c.MouseClickX >= 137 && c.MouseClickX <= 237 && c.MouseClickY >= 490 && c.MouseClickY <= 522 {
		c.PrivateChatSetting = (c.PrivateChatSetting + 1) % 3
		c.RedrawPrivacySettings = true
		c.RedrawChatback = true
		c.Out.P1Isaac(244)
		c.Out.P1(c.PublicChatSetting)
		c.Out.P1(c.PrivateChatSetting)
		c.Out.P1(c.TradeChatSetting)
	}
	if c.MouseClickX >= 275 && c.MouseClickX <= 375 && c.MouseClickY >= 490 && c.MouseClickY <= 522 {
		c.TradeChatSetting = (c.TradeChatSetting + 1) % 3
		c.RedrawPrivacySettings = true
		c.RedrawChatback = true
		c.Out.P1Isaac(244)
		c.Out.P1(c.PublicChatSetting)
		c.Out.P1(c.PrivateChatSetting)
		c.Out.P1(c.TradeChatSetting)
	}
	if c.MouseClickX < 416 || c.MouseClickX > 516 || c.MouseClickY < 490 || c.MouseClickY > 522 {
		return
	}
	c.CloseInterfaces()
	c.ReportAbuseInput = ""
	c.ReportAbuseMuteOption = false
	for i := range len(component.Instances) {
		if component.Instances[i] != nil && component.Instances[i].ClientCode == 600 {
			c.ViewportInterfaceID = component.Instances[i].Layer
			c.ReportAbuseInterfaceID = c.ViewportInterfaceID
			return
		}
	}
}

func (c *Client) HandleChatMouseInput(arg0, arg1 int) {
	var4 := 0
	for i := range 100 {
		if c.MessageText[i] != "" {
			var6 := c.MessageType[i]
			var7 := 70 - var4*14 + c.ChatScrollOffset + 4
			if var7 < -20 {
				break
			}
			if var6 == 0 {
				var4++
			}
			if (var6 == 1 || var6 == 2) && (var6 == 1 || c.PublicChatSetting == 0 || c.PublicChatSetting == 1 && c.IsFriend(c.MessageSender[i])) {
				if arg0 > var7-14 && arg0 <= var7 && c.MessageSender[i] != c.LocalPlayer.Name {
					if c.Rights {
						c.MenuOption[c.MenuSize] = "Report abuse @whi@" + c.MessageSender[i]
						c.MenuAction[c.MenuSize] = 34
						c.MenuSize++
					}
					c.MenuOption[c.MenuSize] = "Add ignore @whi@" + c.MessageSender[i]
					c.MenuAction[c.MenuSize] = 436
					c.MenuSize++
					c.MenuOption[c.MenuSize] = "Add friend @whi@" + c.MessageSender[i]
					c.MenuAction[c.MenuSize] = 406
					c.MenuSize++
				}
				var4++
			}
			if (var6 == 3 || var6 == 7) && c.SplitPrivateChat == 0 && (var6 == 7 || c.PrivateChatSetting == 0 || c.PrivateChatSetting == 1 && c.IsFriend(c.MessageSender[i])) {
				if arg0 > var7-14 && arg0 <= var7 {
					if c.Rights {
						c.MenuOption[c.MenuSize] = "Report abuse @whi@" + c.MessageSender[i]
						c.MenuAction[c.MenuSize] = 34
						c.MenuSize++
					}
					c.MenuOption[c.MenuSize] = "Add ignore @whi@" + c.MessageSender[i]
					c.MenuAction[c.MenuSize] = 436
					c.MenuSize++
					c.MenuOption[c.MenuSize] = "Add friend @whi@" + c.MessageSender[i]
					c.MenuAction[c.MenuSize] = 406
					c.MenuSize++
				}
				var4++
			}
			if var6 == 4 && (c.TradeChatSetting == 0 || c.TradeChatSetting == 1 && c.IsFriend(c.MessageSender[i])) {
				if arg0 > var7-14 && arg0 <= var7 {
					c.MenuOption[c.MenuSize] = "Accept trade @whi@" + c.MessageSender[i]
					c.MenuAction[c.MenuSize] = 903
					c.MenuSize++
				}
				var4++
			}
			if (var6 == 5 || var6 == 6) && c.SplitPrivateChat == 0 && c.PrivateChatSetting < 2 {
				var4++
			}
			if var6 == 8 && (c.TradeChatSetting == 0 || c.TradeChatSetting == 1 && c.IsFriend(c.MessageSender[i])) {
				if arg0 > var7-14 && arg0 <= var7 {
					c.MenuOption[c.MenuSize] = "Accept duel @whi@" + c.MessageSender[i]
					c.MenuAction[c.MenuSize] = 363
					c.MenuSize++
				}
				var4++
			}
		}
	}
	c.PacketSize += arg1
}

func (c *Client) PushPlayers() {
	if c.LocalPlayer.X>>7 == c.FlagSceneTileX && c.LocalPlayer.Z>>7 == c.FlagSceneTileZ {
		c.FlagSceneTileX = 0
	}
	for i := -1; i < c.PlayerCount; i++ {
		var var3 *playerentity.PlayerEntity
		var4 := 0
		if i == -1 {
			var3 = c.LocalPlayer
			var4 = c.LOCAL_PLAYER_INDEX << 14
		} else {
			var3 = c.Players[c.PlayerIDs[i]]
			var4 = c.PlayerIDs[i] << 14
		}
		if var3 != nil && var3.IsVisible() {
			var3.LowMemory = false
			if (LowMemory && c.PlayerCount > 50 || c.PlayerCount > 200) && i != -1 && var3.SecondarySeqID == var3.SeqStandID {
				var3.LowMemory = true
			}
			var5 := var3.X >> 7
			var6 := var3.Z >> 7
			if var5 >= 0 && var5 < 104 && var6 >= 0 && var6 < 104 {
				if var3.LocModel == nil || LoopCycle < var3.LocStartCycle || LoopCycle >= var3.LocStopCycle {
					if (var3.X&0x7F) == 64 && (var3.Z&0x7F) == 64 {
						if c.TileLastOccupiedCycle[var5][var6] == c.SceneCycle {
							continue
						}
						c.TileLastOccupiedCycle[var5][var6] = c.SceneCycle
					}
					var3.Y = c.GetHeightMapY(c.CurrentLevel, var3.X, var3.Z)
					c.Scene.AddTemporary1(var3.Z, 60, var3.Yaw, var3.X, var4, var3.SeqStretches, nil, var3, var3.Y, c.CurrentLevel)
				} else {
					var3.LowMemory = false
					var3.Y = c.GetHeightMapY(c.CurrentLevel, var3.X, var3.Z)
					c.Scene.AddTemporary2(var3.MaxTileX, nil, var3.Z, var3.Y, var4, var3.Yaw, var3.MinTileZ, var3.MinTileX, var3, c.CurrentLevel, var3.MaxTileZ, var3.X)
				}
			}
		}
	}
}

func (c *Client) GetHeightMapY(arg0, arg1, arg3 int) int {
	var5 := arg1 >> 7
	var6 := arg3 >> 7
	var7 := arg0
	if arg0 < 3 && c.LevelTileFlags[1][var5][var6]&0x2 == 2 {
		var7 = arg0 + 1
	}
	var8 := arg1 & 0x7F
	var9 := arg3 & 0x7F
	var10 := c.LevelHeightmap[var7][var5][var6]*(128-var8) + c.LevelHeightmap[var7][var5+1][var6]*var8>>7
	var11 := c.LevelHeightmap[var7][var5][var6+1]*(128-var8) + c.LevelHeightmap[var7][var5+1][var6+1]*var8>>7
	return var10*(128-var9) + var11*var9>>7
}

func (c *Client) AddNPCOptions(arg0 *npctype.NpcType, arg2, arg3, arg4 int) {
	if c.MenuSize >= 400 {
		return
	}
	var6 := arg0.Name
	if arg0.VisLevel != 0 {
		var6 = var6 + GetCombatLevelColorTag(c.LocalPlayer.CombatLevel, arg0.VisLevel) + " (level-" + arg0.VisLevel + ")"
	}
	if c.ObjSelected == 1 {
		c.MenuOption[c.MenuSize] = "Use " + c.ObjSelectedName + " with @yel@" + var6
		c.MenuAction[c.MenuSize] = 900
		c.MenuParamA[c.MenuSize] = arg4
		c.MenuParamB[c.MenuSize] = arg3
		c.MenuParamC[c.MenuSize] = arg2
		c.MenuSize++
	} else if c.SpellSelected != 1 {
		if arg0.Op != nil {
			for i := 4; i >= 0; i-- {
				if arg0.Op[i] != "" && !strings.EqualFold(arg0.Op[i], "attack") {
					c.MenuOption[c.MenuSize] = arg0.Op[i] + " @yel@" + var6
					if i == 0 {
						c.MenuAction[c.MenuSize] = 728
					}
					if i == 1 {
						c.MenuAction[c.MenuSize] = 542
					}
					if i == 2 {
						c.MenuAction[c.MenuSize] = 6
					}
					if i == 3 {
						c.MenuAction[c.MenuSize] = 963
					}
					if i == 4 {
						c.MenuAction[c.MenuSize] = 245
					}
					c.MenuParamA[c.MenuSize] = arg4
					c.MenuParamB[c.MenuSize] = arg3
					c.MenuParamC[c.MenuSize] = arg2
					c.MenuSize++
				}
			}
		}
		if arg0.Op != nil {
			for i := 4; i >= 0; i-- {
				if arg0.Op[i] != "" && strings.EqualFold(arg0.Op[i], "attack") {
					var8 := 0
					if arg0.VisLevel > c.LocalPlayer.CombatLevel {
						var8 = 2000
					}
					c.MenuOption[c.MenuSize] = arg0.Op[i] + " @yel@" + var6
					if i == 0 {
						c.MenuAction[c.MenuSize] = var8 + 728
					}
					if i == 1 {
						c.MenuAction[c.MenuSize] = var8 + 542
					}
					if i == 2 {
						c.MenuAction[c.MenuSize] = var8 + 6
					}
					if i == 3 {
						c.MenuAction[c.MenuSize] = var8 + 963
					}
					if i == 4 {
						c.MenuAction[c.MenuSize] = var8 + 245
					}
					c.MenuParamA[c.MenuSize] = arg4
					c.MenuParamB[c.MenuSize] = arg3
					c.MenuParamC[c.MenuSize] = arg2
					c.MenuSize++
				}
			}
		}
		c.MenuOption[c.MenuSize] = "Examien @yel@" + var6
		c.MenuAction[c.MenuSize] = 1607
		c.MenuParamA[c.MenuSize] = arg4
		c.MenuParamB[c.MenuSize] = arg3
		c.MenuParamC[c.MenuSize] = arg2
		c.MenuSize++
	} else if c.ActiveSpellFlags&0x2 == 2 {
		c.MenuOption[c.MenuSize] = c.SpellCaption + " @yel@" + var6
		c.MenuAction[c.MenuSize] = 265
		c.MenuParamA[c.MenuSize] = arg4
		c.MenuParamB[c.MenuSize] = arg3
		c.MenuParamC[c.MenuSize] = arg2
		c.MenuSize++
	}
}

func (c *Client) HandleInputKey() {
	for {
		var2 := 0
		for ok := true; ok; ok = (var2 < 97 || var2 > 122) && (var2 < 65 || var2 > 90) && (var2 < 48 || var2 > 57) && var2 != 32 {
			for {
				var2 = c.PollKey()
				if var2 == -1 {
					return
				}
				if c.ViewportInterfaceID != -1 && c.ViewportInterfaceID == c.ReportAbuseInterfaceID {
					if var2 == 8 && len(c.ReportAbuseInput) > 0 {
						c.ReportAbuseInput = c.ReportAbuseInput[0 : len(c.ReportAbuseInput)-1]
					}
					break
				}
				var7 := 0
				if c.ShowSocialInput {
					if var2 >= 32 && var2 <= 132 && len(c.SocialInput) < 80 {
						c.SocialInput = c.SocialInput + strconv.Itoa(var2)
						c.RedrawChatback = true
					}
					if var2 == 8 && len(c.SocialInput) > 0 {
						c.SocialInput = c.SocialInput[0 : len(c.SocialInput)-1]
						c.RedrawChatback = true
					}
					if var2 == 13 || var2 == 10 {
						c.ShowSocialInput = false
						c.RedrawChatback = true
						var8 := int64(0)
						if c.SocialAction == 1 {
							var8 = datastruct.ToBase37(c.SocialInput)
							c.AddFriend(var8)
						}
						if c.SocialAction == 2 && c.FriendCount > 0 {
							var8 = datastruct.ToBase37(c.SocialInput)
							c.RemoveFriend(var8)
						}
						if c.SocialAction == 3 && len(c.SocialInput) > 0 {
							c.Out.P1Isaac(148)
							c.Out.P1(0)
							var7 = c.Out.Pos
							c.Out.P8(c.SocialName37)
							// TODO: WordPack.pack
							c.Out.PSize1(c.Out.Pos - var7)
							c.SocialInput = datastruct.ToSentenceCase(c.SocialInput)
							//c.SocialInput = // TODO: WordFilter.filter
							c.AddMessage(6, c.SocialInput, datastruct.FormatName(datastruct.FromBase37(c.SocialName37)))
							if c.PrivateChatSetting == 2 {
								c.PrivateChatSetting = 1
								c.RedrawPrivacySettings = true
								c.Out.P1Isaac(244)
								c.Out.P1(c.PublicChatSetting)
								c.Out.P1(c.PrivateChatSetting)
								c.Out.P1(c.TradeChatSetting)
							}
						}
						if c.SocialAction == 4 && c.IgnoreCount < 100 {
							var8 = datastruct.ToBase37(c.SocialInput)
							c.AddIgnore(var8)
						}
						if c.SocialAction == 5 && c.IgnoreCount > 0 {
							var8 = datastruct.ToBase37(c.SocialInput)
							c.RemoveIgnore(var8)
						}
					}
				} else if c.ChatbackInputOpen {
					if var2 >= 48 && var2 <= 57 && len(c.ChatbackInput) < 10 {
						c.ChatbackInput = c.ChatbackInput + strconv.Itoa(var2)
						c.RedrawChatback = true
					}
					if var2 == 8 && len(c.ChatbackInput) > 0 {
						c.ChatbackInput = c.ChatbackInput[0 : len(c.ChatbackInput)-1]
						c.RedrawChatback = true
					}
					if var2 == 13 || var2 == 10 {
						if len(c.ChatbackInput) > 0 {
							var7, _ = strconv.Atoi(c.ChatbackInput)
							c.Out.P1Isaac(237)
							c.Out.P4(var7)
						}
						c.ChatbackInputOpen = false
						c.RedrawChatback = true
					}
				} else if c.ChatInterfaceID == -1 {
					if var2 >= 32 && var2 <= 122 && len(c.ChatTyped) < 80 {
						c.ChatTyped = c.ChatTyped + strconv.Itoa(var2)
						c.RedrawChatback = true
					}
					if var2 == 8 && len(c.ChatTyped) > 0 {
						c.ChatTyped = c.ChatTyped[0 : len(c.ChatTyped)-1]
						c.RedrawChatback = true
					}
					if (var2 == 13 || var2 == 10) && len(c.ChatTyped) > 0 {
						if c.ChatTyped == "::clientdrop" && (c.Frame != nil || strings.Index(c.GetHost(), "192.168.1.") != -1) {
							c.TryReconnect()
						} else if strings.HasPrefix(c.ChatTyped, "::") {
							c.Out.P1Isaac(4)
							c.Out.P1(len(c.ChatTyped) - 1)
							c.Out.PJStr(c.ChatTyped[2:])
						} else {
							var3 := 0
							if strings.HasPrefix(c.ChatTyped, "yellow:") {
								var3 = 0
								c.ChatTyped = c.ChatTyped[7:]
							}
							if strings.HasPrefix(c.ChatTyped, "red:") {
								var3 = 1
								c.ChatTyped = c.ChatTyped[4:]
							}
							if strings.HasPrefix(c.ChatTyped, "green:") {
								var3 = 2
								c.ChatTyped = c.ChatTyped[6:]
							}
							if strings.HasPrefix(c.ChatTyped, "cyan:") {
								var3 = 3
								c.ChatTyped = c.ChatTyped[5:]
							}
							if strings.HasPrefix(c.ChatTyped, "purple:") {
								var3 = 4
								c.ChatTyped = c.ChatTyped[7:]
							}
							if strings.HasPrefix(c.ChatTyped, "white:") {
								var3 = 5
								c.ChatTyped = c.ChatTyped[6:]
							}
							if strings.HasPrefix(c.ChatTyped, "flash1:") {
								var3 = 6
								c.ChatTyped = c.ChatTyped[7:]
							}
							if strings.HasPrefix(c.ChatTyped, "flash2:") {
								var3 = 7
								c.ChatTyped = c.ChatTyped[7:]
							}
							if strings.HasPrefix(c.ChatTyped, "flash3:") {
								var3 = 8
								c.ChatTyped = c.ChatTyped[7:]
							}
							if strings.HasPrefix(c.ChatTyped, "glow1:") {
								var3 = 9
								c.ChatTyped = c.ChatTyped[6:]
							}
							if strings.HasPrefix(c.ChatTyped, "glow2:") {
								var3 = 10
								c.ChatTyped = c.ChatTyped[6:]
							}
							if strings.HasPrefix(c.ChatTyped, "glow3:") {
								var3 = 11
								c.ChatTyped = c.ChatTyped[6:]
							}
							var4 := 0
							if strings.HasPrefix(c.ChatTyped, "wave:") {
								var4 = 1
								c.ChatTyped = c.ChatTyped[5:]
							}
							if strings.HasPrefix(c.ChatTyped, "scroll:") {
								var4 = 2
								c.ChatTyped = c.ChatTyped[7:]
							}
							c.Out.P1Isaac(158)
							c.Out.P1(0)
							var5 := c.Out.Pos
							c.Out.P1(var3)
							c.Out.P1(var4)
							// TODO: WordPack.pack
							c.Out.PSize1(c.Out.Pos - var5)
							c.ChatTyped = datastruct.ToSentenceCase(c.ChatTyped)
							//c.ChatTyped = WordFilter.Filter // TODO: wordfilter
							c.LocalPlayer.Chat = c.ChatTyped
							c.LocalPlayer.ChatColor = var3
							c.LocalPlayer.ChatStyle = var4
							c.LocalPlayer.ChatTimer = 150
							c.AddMessage(2, c.LocalPlayer.Chat, c.LocalPlayer.Name)
							if c.PublicChatSetting == 2 {
								c.PublicChatSetting = 3
								c.RedrawPrivacySettings = true
								c.Out.P1Isaac(244)
								c.Out.P1(c.PublicChatSetting)
								c.Out.P1(c.PrivateChatSetting)
								c.Out.P1(c.TradeChatSetting)
							}
						}
						c.ChatTyped = ""
						c.RedrawChatback = true
					}
				}
			}
		}
		if len(c.ReportAbuseInput) < 12 {
			c.ReportAbuseInput = c.ReportAbuseInput + strconv.Itoa(var2)
		}
	}
}

func (c *Client) Draw() {
	if c.ErrorStarted || c.ErrorLoading || c.ErrorHost {
		c.DrawError()
		return
	}
	if c.InGame {
		c.DrawGame()
	} else {
		c.DrawTitleScreen()
	}
	c.DragCycles = 0
}

func (c *Client) UpdateTitle() {
	var2 := 0
	var3 := 0
	if c.TitleScreenState == 0 {
		var2 = c.ScreenWidth/2 - 80
		var3 = c.ScreenHeight/2 + 20
		var3 += 20
		if c.MouseClickButton == 1 && c.MouseClickX >= var2-75 && c.MouseClickX <= var2+75 && c.MouseClickY >= var3-20 && c.MouseClickY <= var3+20 {
			c.TitleScreenState = 3
			c.TitleLoginField = 0
		}
		var2 = c.ScreenWidth/2 + 80
		if c.MouseClickButton == 1 && c.MouseClickX >= var2-75 && c.MouseClickX <= var2+75 && c.MouseClickY >= var3-20 && c.MouseClickY <= var3+20 {
			c.LoginMessage0 = ""
			c.LoginMessage1 = "Enter your username & password."
			c.TitleScreenState = 2
			c.TitleLoginField = 0
		}
	} else if c.TitleScreenState == 2 {
		var2 = c.ScreenHeight/2 - 40
		var2 += 30
		var2 += 25
		if c.MouseClickButton == 1 && c.MouseClickY >= var2-15 && c.MouseClickY < var2 {
			c.TitleLoginField = 0
		}
		var2 += 15
		if c.MouseClickButton == 1 && c.MouseClickY >= var2-15 && c.MouseClickY < var2 {
			c.TitleLoginField = 1
		}
		var2 += 15
		var3 = c.ScreenWidth/2 - 80
		var4 := c.ScreenHeight/2 + 50
		var9 := var4 + 20
		if c.MouseClickButton == 1 && c.MouseClickX >= var3-75 && c.MouseClickX <= var3+75 && c.MouseClickY >= var9-20 && c.MouseClickY <= var9+20 {
			c.LoginFunc(c.Username, c.Password, false)
		}
		var3 = c.ScreenWidth/2 + 80
		if c.MouseClickButton == 1 && c.MouseClickX >= var3-75 && c.MouseClickX <= var3+75 && c.MouseClickY >= var9-20 && c.MouseClickY <= var9+20 {
			c.TitleScreenState = 0
			c.Username = ""
			c.Password = ""
		}
		for {
			var5 := c.PollKey()
			if var5 == -1 {
				return
			}
			var6 := false
			for i := range len(CHARSET) {
				if var5 == CHARSET[i] {
					var6 = true
					break
				}
			}
			if c.TitleLoginField == 0 {
				if var5 == 8 && len(c.Username) > 0 {
					c.Username = c.Username[0 : len(c.Username)-1]
				}
				if var5 == 9 || var5 == 10 || var5 == 13 {
					c.TitleLoginField = 1
				}
				if var6 {
					c.Username = c.Username + strconv.Itoa(var5)
				}
				if len(c.Username) > 12 {
					c.Username = c.Username[:12]
				}
			} else if c.TitleLoginField == 1 {
				if var5 == 8 && len(c.Password) > 0 {
					c.Password = c.Password[0 : len(c.Password)-1]
				}
				if var5 == 9 || var5 == 10 || var5 == 13 {
					c.TitleLoginField = 0
				}
				if var6 {
					c.Password = c.Password + strconv.Itoa(var5)
				}
				if len(c.Password) > 20 {
					c.Password = c.Password[:20]
				}
			}
		}
	} else if c.TitleScreenState == 3 {
		var2 = c.ScreenWidth / 2
		var3 = c.ScreenHeight/2 + 50
		var8 := var3 + 20
		if c.MouseClickButton == 1 && c.MouseClickX >= var2-75 && c.MouseClickX <= var2+75 && c.MouseClickY >= var8-20 && c.MouseClickY <= var8+20 {
			c.TitleScreenState = 0
		}
	}
}

func (c *Client) LoadArchive(arg0 string, arg1 int, arg2 string, arg3 int) *io.Jagfile {
	var7 := 5
	var6 := signlink.CacheLoad(arg2)
	var8 := 0
	if var6 != nil {
		// TODO: crc32
	}
	if var6 != nil {
		return io.NewJagfile(var6)
	}
	for var6 == nil {
		c.DrawProgress(true, "Requesting "+arg0, arg3)
		// TODO: try/except
		var8 = 0
		//var9 := c.OpenURL
	}
	// TODO
}

func (c *Client) UnloadTitle() {
	c.FlameActive = false
	for c.FlameThread {
		c.FlameActive = false
		time.Sleep(50 * time.Millisecond)
	}
	c.ImageTitlebox = nil
	c.ImageTitleButton = nil
	c.ImageRunes = nil
	c.FlameGradient = nil
	c.FlameGradient0 = nil
	c.FlameGradient1 = nil
	c.FlameGradient2 = nil
	c.FlameBuffer0 = nil
	c.FlameBuffer1 = nil
	c.FlameBuffer3 = nil
	c.FlameBuffer2 = nil
	c.ImageFlamesLeft = nil
	c.ImageFlamesRight = nil
}

func (c *Client) OrbitCamera(arg0, arg1, arg2, arg3, arg5, arg6 int) {
	var8 := 2048 - arg3&0x7FF
	var9 := 2048 - arg2&0x7FF
	var10 := 0
	var11 := 0
	var12 := arg6
	var13 := 0
	var14 := 0
	var15 := 0
	if var8 != 0 {
		var13 = model.Sin[var8]
		var14 = model.Cos[var8]
		var15 = var11*var14 - arg6*var13>>16
		var12 = var11*var13 + arg6*var14>>16
		var11 = var15
	}
	if var9 != 0 {
		var13 = model.Sin[var9]
		var14 = model.Cos[var9]
		var15 = var12*var13 + var10*var14>>16
		var12 = var12*var14 - var10*var13>>16
		var10 = var15
	}
	c.CameraX = arg1 - var10
	c.CameraY = arg0 - var11
	c.CameraZ = arg5 - var12
	c.CameraPitch = arg3
	c.CameraYaw = arg2
}

func FormatObjCountTagged(arg0 int) string {
	var2 := strconv.Itoa(arg0)
	for i := len(var2) - 3; i > 0; i -= 3 {
		var2 = var2[0:i] + "," + var2[i:]
	}
	if len(var2) > 8 {
		var2 = "@gre@" + var2[0:len(var2)-8] + " million @whi@(" + var2 + ")"
	} else if len(var2) > 4 {
		var2 = "@cya@" + var2[0:len(var2)-4] + "K @whi@(" + var2 + ")"
	}
	return " " + var2
}

func (c *Client) UpdateTextures(arg0 int) {
	if LowMemory {
		return
	}
	var var3 *pix8.Pix8
	var4 := 0
	var5 := 0
	var var6 []byte
	var var7 []byte
	if pix3d.TextureCycle[17] >= arg0 {
		var3 = pix3d.Textures[17]
		var4 = var3.Width*var3.Height - 1
		var5 = var3.Width * c.SceneDelta * 2
		var6 = var3.Pixels
		var7 = c.TextureBuffer
		for i := 0; i <= var4; i++ {
			var7[i] = var6[i-var5&var4]
		}
		var3.Pixels = var7
		c.TextureBuffer = var6
		pix3d.PushTexture(17)
	}
	if pix3d.TextureCycle[24] < arg0 {
		return
	}
	var3 = pix3d.Textures[24]
	var4 = var3.Width*var3.Height - 1
	var5 = var3.Width * c.SceneDelta * 2
	var6 = var3.Pixels
	var7 = c.TextureBuffer
	for i := 0; i <= var4; i++ {
		var7[i] = var6[i-var5&var4]
	}
	var3.Pixels = var7
	c.TextureBuffer = var6
	pix3d.PushTexture(24)
}

func (c *Client) UpdateFlames() {
	var2 := 256
	for i := 10; i < 117; i++ {
		var4 := int(rand.Float64() * 100.0)
		if var4 < 50 {
			c.FlameBuffer3[i+(var2-2<<7)] = 255
		}
	}
	var5 := 0
	var6 := 0
	var7 := 0
	for i := range 100 {
		var5 = int(rand.Float64()*124.0) + 2
		var6 = int(rand.Float64()*128.0) + 128
		var7 = var5 + (var6 << 7)
		c.FlameBuffer3[var7] = 192
	}
	for i := 1; i < var2-1; i++ {
		for j := 1; j < 127; j++ {
			var7 = j + (i << 7)
			c.FlameBuffer2[var7] = (c.FlameBuffer3[var7-1] + c.FlameBuffer3[var7+1] + c.FlameBuffer3[var7-128] + c.FlameBuffer3[var7+128]) / 4
		}
	}
	c.FlameCycle0 += 128
	if c.FlameCycle0 > len(c.FlameBuffer0) {
		c.FlameCycle0 -= len(c.FlameBuffer0)
		var6 = int(rand.Float64() * 12.0)
		c.UpdateFlameBuffer(c.ImageRunes[var6])
	}
	var8 := 0
	for i := 1; i < var2-1; i++ {
		for j := 1; j < 127; j++ {
			var8 = j + (i << 7)
			var9 := c.FlameBuffer2[var8+128] - c.FlameBuffer0[var8+c.FlameCycle0&len(c.FlameBuffer0)-1]/5
			if var9 < 0 {
				var9 = 0
			}
			c.FlameBuffer3[var8] = var9
		}
	}
	for i := range var2 - 1 {
		c.FlameLineOffset[i] = c.FlameLineOffset[i+1]
	}
	c.FlameLineOffset[var2-1] = int(math.Sin(float64(LoopCycle/14.0))*16.0 + math.Sin(float64(LoopCycle/15.0))*14.0 + math.Sin(float64(LoopCycle/16.0))*12.0)
	if c.FlameGradientCycle0 > 0 {
		c.FlameGradientCycle0 -= 4
	}
	if c.FlameGradientCycle1 > 0 {
		c.FlameGradientCycle1 -= 4
	}
	if c.FlameGradientCycle0 != 0 || c.FlameGradientCycle1 != 0 {
		return
	}
	var8 = int(rand.Float64() * 2000.0)
	if var8 == 0 {
		c.FlameGradientCycle0 = 1024
	}
	if var8 == 1 {
		c.FlameGradientCycle1 = 1024
	}
}

func (c *Client) DrawMinimap() {
	c.AreaMapback.Bind()
	var2 := c.OrbitCameraYaw + c.MinimapAnticheatAngle&0x7FF
	var3 := c.LocalPlayer.X/32 + 48
	var4 := 464 - c.LocalPlayer.Z/32
	c.ImageMinimap.DrawRotatedMasked(var2, 146, c.MinimapMaskLineOffsets, 151, var4, c.MinimapZoom+256, var3, 21, 9, c.MinimapMaskLineLengths)
	c.ImageCompass.DrawRotatedMasked(c.OrbitCameraYaw, 33, c.CompassMaskLineOffsets, 33, 25, 256, 25, 0, 0, c.CompassMaskLineLengths)
	for i := range c.ActiveMapFunctionCount {
		var3 = c.ActiveMapFunctionX[i]*4 + 2 - c.LocalPlayer.X/32
		var4 = c.ActiveMapFunctionZ[i]*4 + 2 - c.LocalPlayer.Z/32
		c.DrawOnMinimap(var4, c.ActiveMapFunctions[i], var3)
	}
	for i := range 104 {
		for j := range 104 {
			var8 := c.LevelObjStacks[c.CurrentLevel][i][j]
			if var8 != nil {
				var3 = i*4 + 2 - c.LocalPlayer.X/32
				var4 = j*4 + 2 - c.LocalPlayer.Z/32
				c.DrawOnMinimap(var4, c.ImageMapdot0, var3)
			}
		}
	}
	for i := range c.NPCCount {
		var14 := c.NPCs[c.NPCIDs[i]]
		if var14 != nil && var14.IsVisible() && var14.Type.Minimap {
			var3 = var14.X/32 - c.LocalPlayer.X/32
			var4 = var14.Z/32 - c.LocalPlayer.Z/32
			c.DrawOnMinimap(var4, c.ImageMapdot1, var3)
		}
	}
	for i := range c.PlayerCount {
		var9 := c.Players[c.PlayerIDs[i]]
		if var9 != nil && var9.IsVisible() {
			var3 = var9.X/32 - c.LocalPlayer.X/32
			var4 = var9.Z/32 - c.LocalPlayer.Z/32
			var10 := false
			var11 := datastruct.ToBase37(var9.Name)
			for j := range c.FriendCount {
				if var11 == c.FriendName37[j] && c.FriendWorld[j] != 0 {
					var10 = true
					break
				}
			}
			if var10 {
				c.DrawOnMinimap(var4, c.ImageMapdot3, var3)
			} else {
				c.DrawOnMinimap(var4, c.ImageMapdot2, var3)
			}
		}
	}
	if c.FlagSceneTileX != 0 {
		var3 = c.FlagSceneTileX*4 + 2 - c.LocalPlayer.X/32
		var4 = c.FlagSceneTileZ*4 + 2 - c.LocalPlayer.Z/32
		c.DrawOnMinimap(var4, c.ImageMapflag, var3)
	}
	pix2d.FillRect(82, 93, 16777215, 3, 3)
	c.AreaViewport.Bind()
}

// TODO: GetBaseComponent()

func (c *Client) UpdateMergeLocs() {
	if c.SceneState != 2 {
		return
	}
	for var2 := c.MergedLocations.Head(); var2 != nil; var2 = c.MergedLocations.Next() {
		if LoopCycle >= var2.LastCycle {
			c.AddLoc(var2.Angle, var2.X, var2.Z, var2.Layer, var2.LocIndex, var2.Shape, var2.Plane)
			var2.Unlink()
		}
	}
	CycleLogic5++
	if CycleLogic5 > 85 {
		CycleLogic5 = 0
		c.Out.P1Isaac(85)
	}
}

func (c *Client) CreateMinimap(arg0 int) {
	var3 := c.ImageMinimap.Pixels
	var4 := len(var3)
	for i := range var4 {
		var3[i] = 0
	}
	for i := 1; i < 103; i++ {
		var7 := (103-i)*512*4 + 24628
		for j := 1; j < 103; j++ {
			if c.LevelTileFlags[arg0][j][i]&0x18 == 0 {
				c.Scene.DrawMinimapTile(var3, var7, 512, arg0, j, i)
			}
			if arg0 < 3 && c.LevelTileFlags[arg0+1][j][i]&0x8 != 0 {
				c.Scene.DrawMinimapTile(var3, var7, 512, arg0+1, j, i)
			}
			var7 += 4
		}
	}
	var7 := (int(rand.Float64()*20.0) + 238 - 10<<16) + (int(rand.Float64()*20.0) + 238 - 10<<8) + (int(rand.Float64()*20.0) + 238 - 10)
	var8 := int(rand.Float64()*20.0) + 238 - 10<<16
	c.ImageMinimap.Bind()
	for i := 1; i < 103; i++ {
		for j := 1; j < 103; j++ {
			if c.LevelTileFlags[arg0][j][i]&0x18 == 0 {
				c.DrawMinimapLoc(arg0, var7, j, var8, i)
			}
			if arg0 < 3 && c.LevelTileFlags[arg0+1][j][i]&0x8 != 0 {
				c.DrawMinimapLoc(arg0+1, var7, j, var8, i)
			}
		}
	}
	c.AreaViewport.Bind()
	c.ActiveMapFunctionCount = 0
	for i := range 104 {
		for j := range 104 {
			var12 := c.Scene.GetGroundDecorationBitSet(c.CurrentLevel, i, j)
			if var12 != 0 {
				var12 = var12 >> 14 & 0x7FFF
				var13 := loctype.Get(var12).MapFunction
				if var13 >= 0 {
					var14 := i
					var15 := j
					if var13 != 22 && var13 != 29 && var13 != 34 && var13 != 36 && var13 != 46 && var13 != 47 && var13 != 48 {
						var16 := 104
						var17 := 104
						var18 := c.LevelCollisionMap[c.CurrentLevel].Flags
						for k := range 10 {
							var20 := int(rand.Float64() * 4.0)
							if var20 == 0 && var14 > 0 && var14 > i-3 && var18[var14-1][var15]&0x280108 == 0 {
								var14--
							}
							if var20 == 1 && var14 < var16-1 && var14 < i+3 && var18[var14+1][var15]&0x280180 == 0 {
								var14++
							}
							if var20 == 2 && var15 > 0 && var15 > j-3 && var18[var14][var15-1]&0x280102 == 0 {
								var15--
							}
							if var20 == 3 && var15 < var17-1 && var15 < j+3 && var18[var14][var15+1]&0x280120 == 0 {
								var15++
							}
						}
					}
					c.ActiveMapFunctions[c.ActiveMapFunctionCount] = c.ImageMapFunction[var13]
					c.ActiveMapFunctionX[c.ActiveMapFunctionCount] = var14
					c.ActiveMapFunctionZ[c.ActiveMapFunctionCount] = var15
					c.ActiveMapFunctionCount++
				}
			}
		}
	}
}

func (c *Client) DrawMinimapLoc(arg1, arg2, arg3, arg4, arg5 int) {
	var7 := c.Scene.GetWallBitSet(arg1, arg3, arg5)
	var8 := 0
	var9 := 0
	var10 := 0
	var11 := 0
	var13 := 0
	var14 := 0
	if var7 != 0 {
		var8 = c.Scene.GetInfo(arg1, arg3, arg5, var7)
		var9 = var8 >> 6 & 0x3
		var10 = var8 & 0x1F
		var11 = arg2
		if var7 > 0 {
			var11 = arg4
		}
		var12 := c.ImageMinimap.Pixels
		var13 = arg3*4 + 24624 + (103-arg5)*512*4
		var14 = var7 >> 14 & 0x7FFF
		var15 := loctype.Get(var14)
		if var15.MapScene == -1 {
			if var10 == 0 || var10 == 2 {
				if var9 == 0 {
					var12[var13] = var11
					var12[var13+512] = var11
					var12[var13+1024] = var11
					var12[var13+1536] = var11
				} else if var9 == 1 {
					var12[var13] = var11
					var12[var13+1] = var11
					var12[var13+2] = var11
					var12[var13+3] = var11
				} else if var9 == 2 {
					var12[var13+3] = var11
					var12[var13+3+512] = var11
					var12[var13+3+1024] = var11
					var12[var13+3+1536] = var11
				} else if var9 == 3 {
					var12[var13+1536] = var11
					var12[var13+1536+1] = var11
					var12[var13+1536+2] = var11
					var12[var13+1536+3] = var11
				}
			}
			if var10 == 3 {
				if var9 == 0 {
					var12[var13] = var11
				} else if var9 == 1 {
					var12[var13+3] = var11
				} else if var9 == 2 {
					var12[var13+3+1536] = var11
				} else if var9 == 3 {
					var12[var13+1536] = var11
				}
			}
			if var10 == 2 {
				switch var9 {
				case 3:
					var12[var13] = var11
					var12[var13+512] = var11
					var12[var13+1024] = var11
					var12[var13+1536] = var11
				case 0:
					var12[var13] = var11
					var12[var13+1] = var11
					var12[var13+2] = var11
					var12[var13+3] = var11
				case 1:
					var12[var13+3] = var11
					var12[var13+3+512] = var11
					var12[var13+3+1024] = var11
					var12[var13+3+1536] = var11
				case 2:
					var12[var13+1536] = var11
					var12[var13+1536+1] = var11
					var12[var13+1536+2] = var11
					var12[var13+1536+3] = var11
				}
			}
		} else {
			var16 := c.ImageMapscene[var15.MapScene]
			if var16 != nil {
				var17 := (var15.Width*4 - var16.Width) / 2
				var18 := (var15.Length*4 - var16.Height) / 2
				var16.Draw((104-arg5-var15.Length)*4+48+var18, arg3*4+48+var17)
			}
		}
	}
	var7 = c.Scene.GetLocBitSet(arg1, arg3, arg5)
	if var7 != 0 {
		var8 = c.Scene.GetInfo(arg1, arg3, arg5, var7)
		var9 = var8 >> 6 & 0x3
		var10 = var8 & 0x1F
		var11 = var7 >> 14 & 0x7FFF
		var22 := loctype.Get(var11)
		var26 := 0
		if var22.MapScene != -1 {
			var24 := c.ImageMapscene[var22.MapScene]
			if var24 != nil {
				var14 = (var22.Width*4 - var24.Width) / 2
				var26 = (var22.Length*4 - var24.Height) / 2
				var24.Draw((104-arg5-var22.Length)*4+48+var26, arg3*4+48+var14)
			}
		} else if var10 == 9 {
			var13 = 15658734
			if var7 > 0 {
				var13 = 15597568
			}
			var25 := c.ImageMinimap.Pixels
			var26 = arg3*4 + 24624 + (103-arg5)*512*4
			if var9 == 0 || var9 == 2 {
				var25[var26+1536] = var13
				var25[var26+1024+1] = var13
				var25[var26+512+2] = var13
				var25[var26+3] = var13
			} else {
				var25[var26] = var13
				var25[var26+512+1] = var13
				var25[var26+1024+2] = var13
				var25[var26+1536+3] = var13
			}
		}
	}
	var7 = c.Scene.GetGroundDecorationBitSet(arg1, arg3, arg5)
	if var7 == 0 {
		return
	}
	var8 = var7 >> 14 & 0x7FFF
	var20 := loctype.Get(var8)
	if var20.MapScene == -1 {
		return
	}
	var21 := c.ImageMapscene[var20.MapScene]
	if var21 != nil {
		var11 = (var20.Width*4 - var21.Width) / 2
		var23 := (var20.Length*4 - var21.Height) / 2
		var21.Draw((104-arg5-var20.Length)*4+48+var23, arg3*4+48+var11)
	}
}

func (c *Client) GetNpcPos(arg0 *io.Packet, psize int) {
	c.EntityRemovalCount = 0
	c.EntityUpdateCount = 0
	c.GetNpcPosOldVis(arg0)
	c.GetNpcPosNewVis(arg0, psize)
	c.GetNpcPosExtended(arg0)
	for i := range c.EntityRemovalCount {
		var5 := c.EntityRemovalIDs[i]
		if c.NPCs[var5].Cycle != LoopCycle {
			c.NPCs[var5].Type = nil
			c.NPCs[var5] = nil
		}
	}
	if arg0.Pos != psize {
		//signlink.reporterror // TODO: signlink.reporterror
		panic(c.Username + " size mismatch in getnpcpos - pos:" + strconv.Itoa(arg0.Pos) + " psize:" + strconv.Itoa(psize))
	}
	for i := range c.NPCCount {
		if c.NPCs[c.NPCIDs[i]] == nil {
			// TODO: signlink.reporterror
			panic(c.Username + " null entry in npc list - pos:" + strconv.Itoa(i) + " size:" + strconv.Itoa(c.NPCCount))
		}
	}
}

// TODO: startThread

func (c *Client) LoadTitleImages() {
	c.ImageTitlebox = pix8.NewPix8(c.ArchiveTitle, "titlebox", 0)
	c.ImageTitleButton = pix8.NewPix8(c.ArchiveTitle, "titlebutton", 0)
	c.ImageRunes = make([]*pix8.Pix8, 12)
	for i := range 12 {
		c.ImageRunes[i] = pix8.NewPix8(c.ArchiveTitle, "runes", i)
	}
	c.ImageFlamesLeft = pix32.NewPix321(128, 265)
	c.ImageFlamesRight = pix32.NewPix321(128, 265)
	//for i := range 33920 {
	//	c.ImageFlamesLeft.Pixels[i] = c.ImageTitle0.Pixels[i] // TODO: pixmap
	//}
	//for i := range 33920 {
	//	c.ImageFlamesRight.Pixels[i] = c.ImageTitle1.Pixels[i] // TODO: pixmap
	//}
	c.FlameGradient0 = make([]int, 256)
	for i := range 64 {
		c.FlameGradient0[i] = i * 262144
	}
	for i := range 64 {
		c.FlameGradient0[i+64] = i*1024 + 16711680
	}
	for i := range 64 {
		c.FlameGradient0[i+128] = i*4 + 16776960
	}
	for i := range 64 {
		c.FlameGradient0[i+192] = 16777215
	}
	c.FlameGradient1 = make([]int, 256)
	for i := range 64 {
		c.FlameGradient1[i] = i * 1024
	}
	for i := range 64 {
		c.FlameGradient1[i+64] = i*4 + 65280
	}
	for i := range 64 {
		c.FlameGradient1[i+128] = i*262144 + 65535
	}
	for i := range 64 {
		c.FlameGradient1[i+192] = 16777215
	}
	c.FlameGradient2 = make([]int, 256)
	for i := range 64 {
		c.FlameGradient2[i] = i * 4
	}
	for i := range 64 {
		c.FlameGradient2[i+64] = i*262144 + 255
	}
	for i := range 64 {
		c.FlameGradient2[i+128] = i*1024 + 16711935
	}
	for i := range 64 {
		c.FlameGradient2[i+192] = 16777215
	}
	c.FlameGradient = make([]int, 256)
	c.FlameBuffer0 = make([]int, 32768)
	c.FlameBuffer1 = make([]int, 32768)
	c.UpdateFlameBuffer(nil)
	c.FlameBuffer3 = make([]int, 32768)
	c.FlameBuffer2 = make([]int, 32768)
	c.DrawProgress(true, "Connecting to fileserver", 10)
	if !c.FlameActive {
		c.FlamesThread = true
		c.FlameActive = true
		//c.StartThread() // TODO: StartThread
	}
}

func (c *Client) GetPlayerOldVis(arg1 *io.Packet) {
	var4 := arg1.GBit(8)
	if var4 < c.PlayerCount {
		for i := var4; i < c.PlayerCount; i++ {
			c.EntityRemovalIDs[c.EntityRemovalCount] = c.PlayerIDs[i]
			c.EntityRemovalCount++
		}
	}
	if var4 > c.PlayerCount {
		// TODO: signlink.reporterror
		panic(c.Username + " Too many players")
	}
	c.PlayerCount = 0
	for i := range var4 {
		var6 := c.PlayerIDs[i]
		var7 := c.Players[var6]
		var8 := arg1.GBit(1)
		if var8 == 0 {
			c.PlayerIDs[c.PlayerCount] = var6
			c.PlayerCount++
			var7.Cycle = LoopCycle
		} else {
			var9 := arg1.GBit(2)
			if var9 == 0 {
				c.PlayerIDs[c.PlayerCount] = var6
				c.PlayerCount++
				var7.Cycle = LoopCycle
				c.EntityUpdateIDs[c.EntityUpdateCount] = var6
				c.EntityUpdateCount++
			} else {
				var10 := 0
				var11 := 0
				if var9 == 1 {
					c.PlayerIDs[c.PlayerCount] = var6
					c.PlayerCount++
					var7.Cycle = LoopCycle
					var10 = arg1.GBit(3)
					var7.MoveAlongRoute(false, var10)
					var11 = arg1.GBit(1)
					if var11 == 1 {
						c.EntityUpdateIDs[c.EntityUpdateCount] = var6
						c.EntityUpdateCount++
					}
				} else if var9 == 2 {
					c.PlayerIDs[c.PlayerCount] = var6
					c.PlayerCount++
					var7.Cycle = LoopCycle
					var10 = arg1.GBit(3)
					var7.MoveAlongRoute(true, var10)
					var11 = arg1.GBit(3)
					var7.MoveAlongRoute(true, var11)
					var12 := arg1.GBit(1)
					if var12 == 1 {
						c.EntityUpdateIDs[c.EntityUpdateCount] = var6
						c.EntityUpdateCount++
					}
				} else if var9 == 3 {
					c.EntityRemovalIDs[c.EntityRemovalCount] = var6
					c.EntityRemovalCount++
				}
			}
		}
	}
}

func (c *Client) DrawScrollbar(arg1, arg2, arg3, arg4, arg5 int) {
	c.ImageScrollbar0.Draw(arg2, arg1)
	c.ImageScrollbar1.Draw(arg2+arg5-16, arg1)
	pix2d.FillRect(arg2+16, arg1, c.SCROLLBAR_TRACK, 16, arg5-32)
	var7 := (arg5 - 32) * arg5 / arg4
	if var7 < 8 {
		var7 = 8
	}
	var8 := (arg5 - 32 - var7) * arg3 / (arg4 - arg5)
	pix2d.FillRect(arg2+16+var8, arg1, c.SCROLLBAR_GRIP_FOREGROUND, 16, var7)
	pix2d.VLine(c.SCROLLBAR_GRIP_HIGHLIGHT, arg2+16+var8, var7, arg1)
	pix2d.VLine(c.SCROLLBAR_GRIP_HIGHLIGHT, arg2+16+var8, var7, arg1+1)
	pix2d.HLine(c.SCROLLBAR_GRIP_HIGHLIGHT, arg2+16+var8, 16, arg1)
	pix2d.HLine(c.SCROLLBAR_GRIP_HIGHLIGHT, arg2+17+var8, 16, arg1)
	pix2d.VLine(c.SCROLLBAR_GRIP_LOWLIGHT, arg2+16+var8, var7, arg1+15)
	pix2d.VLine(c.SCROLLBAR_GRIP_LOWLIGHT, arg2+17+var8, var7-1, arg1+14)
	pix2d.HLine(c.SCROLLBAR_GRIP_LOWLIGHT, arg2+15+var8+var7, 16, arg1)
	pix2d.HLine(c.SCROLLBAR_GRIP_LOWLIGHT, arg2+14+var8+var7, 15, arg1+1)
}

func (c *Client) ValidateCharacterDesign() {
	c.UpdateDesignModel = true
	for i := range 7 {
		c.DesignIdentikits[i] = -1
		for j := range idktype.Count {
			x := 7
			if c.DesignGenderMale {
				x = 0
			}
			if !idktype.Instances[j].Disable && idktype.Instances[j].Type == i+x {
				c.DesignIdentikits[i] = j
				break
			}
		}
	}
}

func (c *Client) SaveMidi(arg0 []byte, arg2 int, arg3 bool) {
	if arg3 {
		signlink.MidiFade = 1
	} else {
		signlink.MidiFade = 0
	}
	//signlink.midisave // TODO: signlink.midisave
}

func (c *Client) PushNPCs() {
	for i := range c.NPCCount {
		var3 := c.NPCs[c.NPCIDs[i]]
		var4 := (c.NPCIDs[i] << 14) + 536870912
		if var3 != nil && var3.IsVisible() {
			var5 := var3.X >> 7
			var6 := var3.Z >> 7
			if var5 >= 0 && var5 < 104 && var6 >= 0 && var6 < 104 {
				if var3.Size == 1 && (var3.X&0x7F) == 64 && (var3.Z&0x7F) == 64 {
					if c.TileLastOccupiedCycle[var5][var6] == c.SceneCycle {
						continue
					}
					c.TileLastOccupiedCycle[var5][var6] = c.SceneCycle
				}
				c.Scene.AddTemporary1(var3.Z, (var3.Size-1)*64+60, var3.Yaw, var3.X, var4, var3.SeqStretches, nil, var3, c.GetHeightMapY(c.CurrentLevel, var3.X, var3.Z), c.CurrentLevel)
			}
		}
	}
}

func (c *Client) SetMidiVolume(arg0 int, arg1 int, arg2 bool) {
	signlink.MidiVol = arg1
	c.PacketSize += arg0
	if arg2 {
		signlink.Midi = "voladjust"
	}
}

func (c *Client) DrawTitleScreen() {
	c.LoadTitle()
	c.ImageTitle4.Bind()
	c.ImageTitlebox.Draw(0, 0)
	var2 := 360
	var3 := 200
	var4 := 0
	var5 := 0
	var6 := 0
	if c.TitleScreenState == 0 {
		var4 = var3/2 - 20
		c.FontBold12.DrawStringTaggableCenter(var2/2, 16776960, true, var4, "Welcome to RuneScape")
		_ = var4 + 30
		var5 = var2/2 - 80
		var6 = var3/2 + 20
		c.ImageTitleButton.Draw(var6-20, var5-73)
		c.FontBold12.DrawStringTaggableCenter(var5, 16777215, true, var6+5, "New user")
		var8 := var2/2 + 80
		c.ImageTitleButton.Draw(var6-20, var8-73)
		c.FontBold12.DrawStringTaggableCenter(var8, 16777215, true, var6+5, "Existing User")
	}
	if c.TitleScreenState == 2 {
		var4 = var3/2 - 40
		if len(c.LoginMessage0) > 0 {
			c.FontBold12.DrawStringTaggableCenter(var2/2, 16776960, true, var4-15, c.LoginMessage0)
			c.FontBold12.DrawStringTaggableCenter(var2/2, 16776960, true, var4, c.LoginMessage1)
			var4 += 30
		} else {
			c.FontBold12.DrawStringTaggableCenter(var2/2, 16776960, true, var4-7, c.LoginMessage1)
			var4 += 30
		}
		tmp := ""
		if c.TitleLoginField == 0 && LoopCycle%40 < 20 {
			tmp = "@yel@|"
		}
		c.FontBold12.DrawStringTaggable(var2/2-90, var4, "Username: "+c.Username+tmp, true, 16777215)
		var4 += 15
		tmp2 := ""
		if c.TitleLoginField == 1 && LoopCycle%40 < 20 {
			tmp2 = "@yel@|"
		}
		c.FontBold12.DrawStringTaggable(var2/2-88, var4, "Password: "+datastruct.ToAsterisks(c.Password)+tmp2, true, 16777215)
		var4 += 15
		var5 = var2/2 - 80
		var6 = var3/2 + 50
		c.ImageTitleButton.Draw(var6-20, var5-73)
		c.FontBold12.DrawStringTaggableCenter(var5, 16777215, true, var6+5, "Login")
		var5 = var2/2 + 80
		c.ImageTitleButton.Draw(var6-20, var5-73)
		c.FontBold12.DrawStringTaggableCenter(var5, 16777215, true, var6+5, "Cancel")
	}
	if c.TitleScreenState == 3 {
		c.FontBold12.DrawStringTaggableCenter(var2/2, 16776960, true, var3/2-60, "Create a free account")
		var4 = var3/2 - 35
		c.FontBold12.DrawStringTaggableCenter(var2/2, 16777215, true, var4, "To create a new account you need to")
		var4 += 15
		c.FontBold12.DrawStringTaggableCenter(var2/2, 16777215, true, var4, "go back to the main RuneScape webpage")
		var4 += 15
		c.FontBold12.DrawStringTaggableCenter(var2/2, 16777215, true, var4, "and choose the red 'create account'")
		var4 += 15
		c.FontBold12.DrawStringTaggableCenter(var2/2, 16777215, true, var4, "button at the top right of that page.")
		var4 += 15
		var5 = var2 / 2
		var6 = var3/2 + 50
		c.ImageTitleButton.Draw(var6-20, var5-73)
		c.FontBold12.DrawStringTaggableCenter(var5, 16777215, true, var6+5, "Cancel")
	}
	//c.ImageTitle4.Draw // TODO: pixmap
	if !c.RedrawBackground {
		return
	}
	c.RedrawBackground = false
	//c.ImageTitle2.Draw // TODO pixmap
}

func (c *Client) PrepareGameScreen() {
	if c.AreaChatback != nil {
		return
	}
	c.UnloadTitle()
	c.DrawArea = nil
	c.ImageTitle2 = nil
	c.ImageTitle3 = nil
	c.ImageTitle4 = nil
	c.ImageTitle0 = nil
	c.ImageTitle1 = nil
	c.ImageTitle5 = nil
	c.ImageTitle6 = nil
	c.ImageTitle7 = nil
	c.ImageTitle8 = nil
	// TODO: pixmap
	pix2d.Clear()
	c.ImageMapback.Draw(0, 0)
	// TODO: pixmap
	pix2d.Clear()
	// TODO: pixmap
	c.RedrawBackground = true
}

func (c *Client) GetPlayerNewVis(arg1 int, arg2 *io.Packet) {
	var4 := 0
	for arg2.BitPos+10 < arg1*8 {
		var4 = arg2.GBit(11)
		if var4 == 2047 {
			break
		}
		if c.Players[var4] == nil {
			c.Players[var4] = playerentity.NewPlayerEntity()
			if c.PlayerAppearanceBuffer[var4] != nil {
				c.Players[var4].Read(c.PlayerAppearanceBuffer[var4])
			}
		}
		c.PlayerIDs[c.PlayerCount] = var4
		c.PlayerCount++
		var5 := c.Players[var4]
		var5.Cycle = LoopCycle
		var6 := arg2.GBit(5)
		if var6 > 15 {
			var6 -= 32
		}
		var7 := arg2.GBit(5)
		if var7 > 15 {
			var7 -= 32
		}
		var8 := arg2.GBit(1)
		var5.Teleport(var8 == 1, c.LocalPlayer.PathTileX[0]+var6, c.LocalPlayer.PathTileZ[0]+var7)
		var9 := arg2.GBit(1)
		if var9 == 1 {
			c.EntityUpdateIDs[c.EntityUpdateCount] = var4
			c.EntityUpdateCount++
		}
	}
	arg2.AccessBytes()
}

func (c *Client) Logout() {
	// TODO: c.Stream.Close()
	// TODO: c.Stream = nil
	c.TitleScreenState = 0
	c.Username = ""
	c.Password = ""
	// TODO: InputTracking.SetDisabled()
	c.ClearCaches()
	c.Scene.Reset()
	for i := range 4 {
		c.LevelCollisionMap[i].Reset()
	}
	c.StopMidi()
	c.CurrentMidi = ""
	c.NextMusicDelay = 0
}

func (c *Client) DrawInterface(arg0 int, arg1 int, arg3 *component.Component, arg4 int) {
	if arg3.Type != 0 || arg3.ChildID != nil || arg3.Hide && c.ViewportHoveredInterfaceIndex != arg3.Id && c.SidebarHoveredInterfaceIndex != arg3.Id && c.ChatHoveredInterfaceIndex != arg3.Id {
		return
	}
	var6 := pix2d.BoundLeft
	var7 := pix2d.BoundTop
	var8 := pix2d.BoundRight
	var9 := pix2d.BoundBottom
	pix2d.SetClipping(arg0+arg3.Height, arg0, arg1+arg3.Width, arg1)
	var10 := len(arg3.ChildID)
	for i := range var10 {
		var12 := arg3.ChildX[i] + arg1
		var13 := arg3.ChildY[i] + arg0 - arg4
		var14 := component.Instances[arg3.ChildID[i]]
		var25 := var12 + var14.X
		var26 := var13 + var14.Y
		if var14.ClientCode > 0 {
			c.UpdateInterfaceContent(var14)
		}
		if var14.Type == 0 {
			if var14.ScrollPosition > var14.Scroll-var14.Height {
				var14.ScrollPosition = var14.Scroll - var14.Height
			}
			if var14.ScrollPosition < 0 {
				var14.ScrollPosition = 0
			}
			c.DrawInterface(var26, var25, var14, var14.ScrollPosition)
			if var14.Scroll > var14.Height {
				c.DrawScrollbar(var25+var14.Width, var26, var14.ScrollPosition, var14.Scroll, var14.Height)
			}
		} else if var14.Type != 1 {
			var16 := 0
			var17 := 0
			var18 := 0
			var21 := 0
			var22 := 0
			var27 := 0
			var32 := 0
			var33 := 0
			if var14.Type == 2 {
				var27 = 0
				for j := range var14.Height {
					for k := range var14.Width {
						var18 = var25 + k*(var14.MarginX+32)
						var32 = var26 + j*(var14.MarginY+32)
						if var27 < 20 {
							var18 += var14.InvSlotOffsetX[var27]
							var32 += var14.InvSlotOffsetY[var27]
						}
						if var14.InvSlotObjId[var27] > 0 {
							var33 = 0
							var21 = 0
							var22 = var14.InvSlotObjId[var27] - 1
							if var18 >= -32 && var18 <= 512 && var32 >= -32 && var32 <= 334 || c.ObjDragArea != 0 && c.ObjDragSlot == var27 {
								var23 := objtype.GetIcon(var22, var14.InvSlotObjCount[var27])
								if c.ObjDragArea != 0 && c.ObjDragSlot == var27 && c.ObjDragInterfaceID == var14.Id {
									var33 = c.MouseX - c.ObjGrabX
									var21 = c.MouseY - c.ObjGrabY
									if var33 < 5 && var33 > -5 {
										var33 = 0
									}
									if var21 < 5 && var21 > -5 {
										var21 = 0
									}
									if c.ObjDragCycles < 5 {
										var33 = 0
										var21 = 0
									}
									var23.DrawAlpha(128, var18+var33, var32+var21)
								} else if c.SelectedArea != 0 && c.SelectedItem == var27 && c.SelectedInterface == var14.Id {
									var23.DrawAlpha(128, var18, var32)
								} else {
									var23.Draw(var32, var18)
								}
								if var23.CropW == 33 || var14.InvSlotObjCount[var27] != 1 {
									var24 := var14.InvSlotObjCount[var27]
									c.FontPlain11.DrawString(var18+1+var33, var32+10+var21, 0, FormatObjCount(var24))
									c.FontPlain11.DrawString(var18+var33, var32+9+var21, 16776960, FormatObjCount(var24))
								}
							}
						} else if var14.InvSlotSprite != nil && var27 < 20 {
							var36 := var14.InvSlotSprite[var27]
							if var36 != nil {
								var36.Draw(var32, var18)
							}
						}
						var27++
					}
				}
			} else if var14.Type != 3 {
				var var15 *pixfont.PixFont
				if var14.Type == 4 {
					var15 = var14.Font
					var16 = var14.Colour
					var29 := var14.Text
					if (c.ChatHoveredInterfaceIndex == var14.Id || c.SidebarHoveredInterfaceIndex == var14.Id || c.ViewportHoveredInterfaceIndex == var14.Id) && var14.OverColour != 0 {
						var16 = var14.OverColour
					}
					if c.ExecuteInterfaceScript(var14) {
						var16 = var14.ActiveColour
						if len(var14.ActiveText) > 0 {
							var29 = var14.ActiveText
						}
					}
					if var14.ButtonType == 6 && c.PressedContinueOption {
						var29 = "Please wait..."
						var16 = var14.Colour
					}
					var32 = var26 + var15.Height
					for len(var29) > 0 {
						if strings.Index(var29, "%") != -1 {
						label260:
							for {
								var33 = strings.Index(var29, "%1")
								if var33 == 1 {
									for {
										var33 = strings.Index(var29, "%2")
										if var33 == -1 {
											for {
												var33 = strings.Index(var29, "%3")
												if var33 == -1 {
													for {
														var33 = strings.Index(var29, "%4")
														if var33 == -1 {
															for {
																var33 = strings.Index(var29, "%5")
																if var33 == -1 {
																	break label260
																}
																var29 = var29[0:var33] + c.GetIntString(c.ExecuteClientscript1(var14, 4)) + var29[var33+2:]
															}
														}
														var29 = var29[0:var33] + c.GetIntString(c.ExecuteClientscript1(var14, 3)) + var29[var33+2:]
													}
												}
												var29 = var29[0:var33] + c.GetIntString(c.ExecuteClientscript1(var14, 2)) + var29[var33+2:]
											}
										}
										var29 = var29[0:var33] + c.GetIntString(c.ExecuteClientscript1(var14, 1)) + var29[var33+2:]
									}
								}
								var29 = var29[0:var33] + c.GetIntString(c.ExecuteClientscript1(var14, 0)) + var29[var33+2:]
							}
						}
						var33 = strings.Index(var29, "\\n")
						var var30 string
						if var33 == -1 {
							var30 = var29
							var29 = ""
						} else {
							var30 = var29[0:var33]
							var29 = var29[var33+2:]
						}
						if var14.Center {
							var15.DrawStringTaggableCenter(var25+var14.Width/2, var16, var14.Shadowed, var32, var30)
						} else {
							var15.DrawStringTaggable(var25, var32, var30, var14.Shadowed, var16)
						}
						var32 += var15.Height
					}
				} else if var14.Type == 5 {
					var var28 *pix32.Pix32
					if c.ExecuteInterfaceScript(var14) {
						var28 = var14.ActiveGraphic
					} else {
						var28 = var14.Graphic
					}
					if var28 != nil {
						var28.Draw(var26, var25)
					}
				} else if var14.Type == 6 {
					var27 = pix3d.CenterW3D
					var16 = pix3d.CenterH3D
					pix3d.CenterW3D = var25 + var14.Width/2
					pix3d.CenterH3D = var26 + var14.Height/2
					var17 = pix3d.SinTable[var14.Xan] * var14.Zoom >> 16
					var18 = pix3d.CosTable[var14.Xan] * var14.Zoom >> 16
					var31 := c.ExecuteInterfaceScript(var14)
					if var31 {
						var33 = var14.ActiveAnim
					} else {
						var33 = var14.Anim
					}
					var var34 *model.Model
					if var33 == -1 {
						var34 = var14.GetModel(-1, -1, var31)
					} else {
						var35 := seqtype.Instances[var33]
						var34 = var14.GetModel(var35.Frames[var14.SeqFrame], var35.IFrames[var14.SeqFrame], var31)
					}
					if var34 != nil {
						var34.DrawSimple(0, var14.Yan, 0, var14.Xan, 0, var17, var18)
					}
					pix3d.CenterW3D = var27
					pix3d.CenterH3D = var16
				} else if var14.Type == 7 {
					var15 = var14.Font
					var16 = 0
					for j := range var14.Height {
						for k := range var14.Width {
							if var14.InvSlotObjId[var16] > 0 {
								var19 := objtype.Get(var14.InvSlotObjId[var16] - 1)
								var20 := var19.Name
								if var19.Stackable || var14.InvSlotObjCount[var16] != 1 {
									var20 = var20 + " x" + FormatObjCountTagged(var14.InvSlotObjCount[var16])
								}
								var21 = var25 + k*(var14.MarginX+115)
								var22 = var26 + j*(var14.MarginY+12)
								if var14.Center {
									var15.DrawStringTaggableCenter(var21+var14.Width/2, var14.Colour, var14.Shadowed, var22, var20)
								} else {
									var15.DrawStringTaggable(var21, var22, var20, var14.Shadowed, var14.Colour)
								}
							}
							var16++
						}
					}
				}
			} else if var14.Fill {
				pix2d.FillRect(var26, var25, var14.Colour, var14.Width, var14.Height)
			} else {
				pix2d.DrawRect(var25, var14.Colour, var14.Height, var26, var14.Width)
			}
		}
	}
	pix2d.SetClipping(var9, var7, var8, var6)
}

func (c *Client) GetPlayerExtended1(arg2 *io.Packet) {
	for i := range c.EntityUpdateCount {
		var5 := c.EntityUpdateIDs[i]
		var6 := c.Players[var5]
		var7 := arg2.G1()
		if var7&0x80 == 128 {
			var7 += arg2.G1() << 8
		}
		c.GetPlayerExtended2(var5, var7, arg2, var6)
	}
}

func (c *Client) UpdateVarp(arg0 int) {
	var3 := varptype.Instances[arg0].ClientCode
	if var3 == 0 {
		return
	}
	var4 := c.Varps[arg0]
	if var3 == 1 {
		switch var4 {
		case 1:
			pix3d.SetBrightness(0.9)
		case 2:
			pix3d.SetBrightness(0.8)
		case 3:
			pix3d.SetBrightness(0.7)
		case 4:
			pix3d.SetBrightness(0.6)
		}
		objtype.IconCache.Clear()
		c.RedrawBackground = true
	}
	if var3 == 3 {
		var5 := c.MidiActive
		switch var4 {
		case 0:
			c.SetMidiVolume(0, 0, c.MidiActive)
			c.MidiActive = true
		case 1:
			c.SetMidiVolume(0, -400, c.MidiActive)
			c.MidiActive = true
		case 2:
			c.SetMidiVolume(0, -800, c.MidiActive)
			c.MidiActive = true
		case 3:
			c.SetMidiVolume(0, -1200, c.MidiActive)
			c.MidiActive = true
		case 4:
			c.MidiActive = false
		}
		if c.MidiActive != var5 {
			if c.MidiActive {
				c.SetMidi(c.MidiCRC, c.CurrentMidi, c.MidiSize)
			} else {
				c.StopMidi()
			}
			c.NextMusicDelay = 0
		}
	}
	if var3 == 4 {
		switch var4 {
		case 0:
			c.WaveEnabled = true
			c.SetWaveVolume(0)
		case 1:
			c.WaveEnabled = true
			c.SetWaveVolume(-400)
		case 2:
			c.WaveEnabled = true
			c.SetWaveVolume(-800)
		case 3:
			c.WaveEnabled = true
			c.SetWaveVolume(-1200)
		case 4:
			c.WaveEnabled = false
		}
	}
	if var3 == 5 {
		c.MouseButtonsOption = var4
	}
	if var3 == 6 {
		c.ChatEffects = var4
	}
	if var3 == 8 {
		c.SplitPrivateChat = var4
		c.RedrawChatback = true
	}
}

func (c *Client) UpdateNpcs() {
	for i := range c.NPCCount {
		var3 := c.NPCIDs[i]
		var4 := c.NPCs[var3]
		if var4 != nil {
			c.UpdateEntity(&var4.PathingEntity)
		}
	}
}

func (c *Client) UpdatePlayerEntity(arg0 *playerentity.PlayerEntity) {
	if arg0.X < 128 || arg0.Z < 128 || arg0.X >= 13184 || arg0.Z >= 13184 {
		arg0.PrimarySeqID = -1
		arg0.SpotanimID = -1
		arg0.ForceMoveEndCycle = 0
		arg0.ForceMoveStartCycle = 0
		arg0.X = arg0.PathTileX[0]*128 + arg0.Size*64
		arg0.Z = arg0.PathTileZ[0]*128 + arg0.Size*64
		arg0.PathLength = 0
	}
	if arg0 == c.LocalPlayer && (arg0.X < 1536 || arg0.Z < 1536 || arg0.X >= 11776 || arg0.Z >= 11776) {
		arg0.PrimarySeqID = -1
		arg0.SpotanimID = -1
		arg0.ForceMoveEndCycle = 0
		arg0.ForceMoveStartCycle = 0
		arg0.X = arg0.PathTileX[0]*128 + arg0.Size*64
		arg0.Z = arg0.PathTileZ[0]*128 + arg0.Size*64
		arg0.PathLength = 0
	}
	if arg0.ForceMoveEndCycle > LoopCycle {
		c.UpdateForceMovement(&arg0.PathingEntity)
	} else if arg0.ForceMoveStartCycle >= LoopCycle {
		c.StartForceMovement(&arg0.PathingEntity, 0)
	} else {
		c.UpdateMovement(&arg0.PathingEntity)
	}
	c.UpdateFacingDirection(&arg0.PathingEntity)
	c.UpdateSequences(&arg0.PathingEntity)
}

func (c *Client) UpdateNpcEntity(arg0 *entity.NpcEntity) {
	if arg0.X < 128 || arg0.Z < 128 || arg0.X >= 13184 || arg0.Z >= 13184 {
		arg0.PrimarySeqID = -1
		arg0.SpotanimID = -1
		arg0.ForceMoveEndCycle = 0
		arg0.ForceMoveStartCycle = 0
		arg0.X = arg0.PathTileX[0]*128 + arg0.Size*64
		arg0.Z = arg0.PathTileZ[0]*128 + arg0.Size*64
		arg0.PathLength = 0
	}
	if arg0.ForceMoveEndCycle > LoopCycle {
		c.UpdateForceMovement(&arg0.PathingEntity)
	} else if arg0.ForceMoveStartCycle >= LoopCycle {
		c.StartForceMovement(&arg0.PathingEntity, 0)
	} else {
		c.UpdateMovement(&arg0.PathingEntity)
	}
	c.UpdateFacingDirection(&arg0.PathingEntity)
	c.UpdateSequences(&arg0.PathingEntity)
}

func (c *Client) UpdateForceMovement(arg0 *entity.PathingEntity) {
	var3 := arg0.ForceMoveEndCycle - LoopCycle
	var4 := arg0.ForceMoveStartSceneTileX*128 + arg0.Size*64
	var5 := arg0.ForceMoveStartSceneTileZ*128 + arg0.Size*64
	arg0.X += (var4 - arg0.X) / var3
	arg0.Z += (var5 - arg0.Z) / var3
	arg0.SeqTrigger = 0
	switch arg0.ForceMoveFaceDirection {
	case 0:
		arg0.DstYaw = 1024
	case 1:
		arg0.DstYaw = 1536
	case 2:
		arg0.DstYaw = 0
	case 3:
		arg0.DstYaw = 512
	}
}

func (c *Client) StartForceMovement(arg0 *entity.PathingEntity, arg1 int) {
	c.PacketSize += arg1
	if arg0.ForceMoveStartCycle == LoopCycle || arg0.PrimarySeqID == -1 || arg0.PrimarySeqDelay != 0 || arg0.PrimarySeqCycle+1 > seqtype.Instances[arg0.PrimarySeqID].Delay[arg0.PrimarySeqFrame] {
		var3 := arg0.ForceMoveStartCycle - arg0.ForceMoveEndCycle
		var4 := LoopCycle - arg0.ForceMoveEndCycle
		var5 := arg0.ForceMoveStartSceneTileX*128 + arg0.Size*64
		var6 := arg0.ForceMoveStartSceneTileZ*128 + arg0.Size*64
		var7 := arg0.ForceMoveEndSceneTileX*128 + arg0.Size*64
		var8 := arg0.ForceMoveEndSceneTileZ*128 + arg0.Size*64
		arg0.X = (var5*(var3-var4) + var7*var4) / var3
		arg0.Z = (var6*(var3-var4) + var8*var4) / var3
	}
	arg0.SeqTrigger = 0
	switch arg0.ForceMoveFaceDirection {
	case 0:
		arg0.DstYaw = 1024
	case 1:
		arg0.DstYaw = 1536
	case 2:
		arg0.DstYaw = 0
	case 3:
		arg0.DstYaw = 512
	}
	arg0.Yaw = arg0.DstYaw
}

func (c *Client) UpdateMovement(arg1 *entity.PathingEntity) {
	arg1.SecondarySeqID = arg1.SeqStandID
	if arg1.PathLength == 0 {
		arg1.SeqTrigger = 0
		return
	}
	if arg1.PrimarySeqID != -1 && arg1.PrimarySeqDelay == 0 {
		var3 := seqtype.Instances[arg1.PrimarySeqID]
		if var3.WalkMerge == nil {
			arg1.SeqTrigger++
			return
		}
	}
	var11 := arg1.X
	var4 := arg1.Z
	var5 := arg1.PathTileX[arg1.PathLength-1]*128 + arg1.Size*64
	var6 := arg1.PathTileZ[arg1.PathLength-1]*128 + arg1.Size*64
	if var5-var11 > 256 || var5-var11 < -256 || var6-var4 > 256 || var6-var4 < -256 {
		arg1.X = var5
		arg1.Z = var6
		return
	}
	if var11 < var5 {
		if var4 < var6 {
			arg1.DstYaw = 1280
		} else if var4 > var6 {
			arg1.DstYaw = 1792
		} else {
			arg1.DstYaw = 1536
		}
	} else if var11 > var5 {
		if var4 < var6 {
			arg1.DstYaw = 768
		} else if var4 > var6 {
			arg1.DstYaw = 256
		} else {
			arg1.DstYaw = 512
		}
	} else if var4 < var6 {
		arg1.DstYaw = 1024
	} else {
		arg1.DstYaw = 0
	}
	var7 := arg1.DstYaw - arg1.Yaw&0x7FF
	if var7 > 1024 {
		var7 -= 2048
	}
	var8 := arg1.SeqTurnAroundID
	if var7 > +-256 && var7 <= 256 {
		var8 = arg1.SeqWalkID
	} else if var7 >= 256 && var7 < 768 {
		var8 = arg1.SeqTurnRightId
	} else if var7 >= -768 && var7 <= -256 {
		var8 = arg1.SeqTurnLeftID
	}
	if var8 == -1 {
		var8 = arg1.SeqWalkID
	}
	arg1.SecondarySeqID = var8
	var9 := 4
	if arg1.Yaw != arg1.DstYaw && arg1.TargetID == -1 {
		var9 = 2
	}
	if arg1.PathLength > 2 {
		var9 = 6
	}
	if arg1.PathLength > 3 {
		var9 = 8
	}
	if arg1.SeqTrigger > 0 && arg1.PathLength > 1 {
		var9 = 8
		arg1.SeqTrigger--
	}
	if arg1.PathRunning[arg1.PathLength-1] {
		var9 <<= 0x1
	}
	if var9 >= 8 && arg1.SecondarySeqID == arg1.SeqWalkID && arg1.SeqRunID != -1 {
		arg1.SecondarySeqID = arg1.SeqRunID
	}
	if var11 < var5 {
		arg1.X += var9
		if arg1.X > var5 {
			arg1.X = var5
		}
	} else if var11 > var5 {
		arg1.X -= var9
		if arg1.X < var5 {
			arg1.X = var5
		}
	}
	if var4 < var6 {
		arg1.Z += var9
		if arg1.Z > var6 {
			arg1.Z = var6
		}
	} else if var4 > var6 {
		arg1.Z -= var9
		if arg1.Z < var6 {
			arg1.Z = var6
		}
	}
	if arg1.X == var5 && arg1.Z == var6 {
		arg1.PathLength--
	}
}

func (c *Client) UpdateFacingDirection(arg0 *entity.PathingEntity) {
	var4 := 0
	var5 := 0
	if arg0.TargetID != -1 && arg0.TargetID < 32768 {
		var3 := c.NPCs[arg0.TargetID]
		if var3 != nil {
			var4 = arg0.X - var3.X
			var5 = arg0.Z - var3.Z
			if var4 != 0 || var5 != 0 {
				arg0.DstYaw = int(math.Atan2(float64(var4), float64(var5))*325.949) & 0x7FF
			}
		}
	}
	var7 := 0
	if arg0.TargetID >= 32768 {
		var7 = arg0.TargetID - 32768
		if var7 == c.LocalPID {
			var7 = c.LOCAL_PLAYER_INDEX
		}
		var8 := c.Players[var7]
		if var8 != nil {
			var5 = arg0.X - var8.X
			var6 := arg0.Z - var8.Z
			if var5 != 0 || var6 != 0 {
				arg0.DstYaw = int(math.Atan2(float64(var5), float64(var6))*325.949) & 0x7FF
			}
		}
	}
	if (arg0.TargetTileX != 0 || arg0.TargetTileZ != 0) && (arg0.PathLength == 0 || arg0.SeqTrigger > 0) {
		var7 = arg0.X - (arg0.TargetTileX-c.SceneBaseTileX-c.SceneBaseTileX)*64
		var4 = arg0.Z - (arg0.TargetTileZ-c.SceneBaseTileZ-c.SceneBaseTileZ)*64
		if var7 != 0 || var4 != 0 {
			arg0.DstYaw = int(math.Atan2(float64(var7), float64(var4))*325.949) & 0x7FF
		}
		arg0.TargetTileX = 0
		arg0.TargetTileZ = 0
	}
	var7 = arg0.DstYaw - arg0.Yaw&0x7FF
	if var7 == 0 {
		return
	}
	if var7 < 32 || var7 > 2016 {
		arg0.Yaw = arg0.DstYaw
	} else if var7 > 1024 {
		arg0.Yaw -= 32
	} else {
		arg0.Yaw += 32
	}
	arg0.Yaw &= 0x7FF
	if arg0.SecondarySeqID != arg0.SeqStandID || arg0.Yaw == arg0.DstYaw {
		return
	}
	if arg0.SeqTurnID != -1 {
		arg0.SecondarySeqID = arg0.SeqTurnID
		return
	}
	arg0.SecondarySeqID = arg0.SeqWalkID
}

func (c *Client) UpdateSequences(arg1 *entity.PathingEntity) {
	arg1.SeqStretches = false
	var var3 *seqtype.SeqType
	if arg1.SecondarySeqID != -1 {
		var3 = seqtype.Instances[arg1.SecondarySeqID]
		arg1.SecondarySeqCycle++
		if arg1.SecondarySeqFrame < var3.FrameCount && arg1.SecondarySeqCycle > var3.Delay[arg1.SecondarySeqFrame] {
			arg1.SecondarySeqCycle = 0
			arg1.SecondarySeqFrame++
		}
		if arg1.SecondarySeqFrame >= var3.FrameCount {
			arg1.SecondarySeqCycle = 0
			arg1.SecondarySeqFrame = 0
		}
	}
	if arg1.PrimarySeqID != -1 && arg1.PrimarySeqDelay == 0 {
		var3 = seqtype.Instances[arg1.PrimarySeqID]
		arg1.PrimarySeqCycle++
		for arg1.PrimarySeqFrame < var3.FrameCount && arg1.PrimarySeqCycle > var3.Delay[arg1.PrimarySeqFrame] {
			arg1.PrimarySeqCycle -= var3.Delay[arg1.PrimarySeqFrame]
			arg1.PrimarySeqFrame++
		}
		if arg1.PrimarySeqFrame >= var3.FrameCount {
			arg1.PrimarySeqFrame -= var3.ReplayOff
			arg1.PrimarySeqLoop++
			if arg1.PrimarySeqLoop >= var3.ReplayCount {
				arg1.PrimarySeqID = -1
			}
			if arg1.PrimarySeqFrame < 0 || arg1.PrimarySeqFrame >= var3.FrameCount {
				arg1.PrimarySeqID = -1
			}
		}
		arg1.SeqStretches = var3.Stretches
	}
	if arg1.PrimarySeqDelay > 0 {
		arg1.PrimarySeqDelay--
	}
	if arg1.SpotanimID == -1 || LoopCycle < arg1.SpotanimLastCycle {
		return
	}
	if arg1.SpotanimFrame < 0 {
		arg1.SpotanimFrame = 0
	}
	var3 = spotanimtype.Instances[arg1.SpotanimID].Seq
	arg1.SpotanimCycle++
	for arg1.SpotanimFrame < var3.FrameCount && arg1.SpotanimCycle > var3.Delay[arg1.SpotanimFrame] {
		arg1.SpotanimCycle -= var3.Delay[arg1.SpotanimFrame]
		arg1.SpotanimFrame++
	}
	if arg1.SpotanimFrame >= var3.FrameCount {
		if arg1.SpotanimFrame < 0 || arg1.SpotanimFrame >= var3.FrameCount {
			arg1.SpotanimID = -1
		}
	}
}

func (c *Client) DrawGame() {
	if c.RedrawBackground {
		c.RedrawBackground = false
		// TODO: pixmap
		c.RedrawSidebar = true
		c.RedrawChatback = true
		c.RedrawSideIcons = true
		c.RedrawPrivacySettings = true
		if c.SceneState != 2 {
			// TODO: pixmap
		}
	}
	if c.SceneState == 2 {
		c.DrawScene(0)
	}
	if c.MenuVisible && c.MenuArea == 1 {
		c.RedrawSidebar = true
	}
	var2 := false
	if c.SidebarInterfaceID != -1 {
		var2 = c.UpdateInterfaceAnimation(c.SidebarInterfaceID, c.SceneDelta)
		if var2 {
			c.RedrawSidebar = true
		}
	}
	if c.SelectedArea == 2 {
		c.RedrawSidebar = true
	}
	if c.ObjDragArea == 2 {
		c.RedrawSidebar = true
	}
	if c.RedrawSidebar {
		c.DrawSidebar()
		c.RedrawSidebar = false
	}
	if c.ChatInterfaceID == -1 {
		c.ChatInterface.ScrollPosition = c.ChatScrollHeight - c.ChatScrollOffset - 77
		if c.MouseX > 453 && c.MouseX < 565 && c.MouseY > 350 {
			c.HandleScrollInput(c.MouseX-22, 0, c.MouseY-375, c.ChatScrollHeight, 77, false, 463, 0, c.ChatInterface)
		}
		var3 := c.ChatScrollHeight - 77 - c.ChatInterface.ScrollPosition
		if var3 < 0 {
			var3 = 0
		}
		if var3 > c.ChatScrollHeight-77 {
			var3 = c.ChatScrollHeight - 77
		}
		if c.ChatScrollOffset != var3 {
			c.ChatScrollOffset = var3
			c.RedrawChatback = true
		}
	}
	if c.ChatInterfaceID != -1 {
		var2 = c.UpdateInterfaceAnimation(c.ChatInterfaceID, c.SceneDelta)
		if var2 {
			c.RedrawChatback = true
		}
	}
	if c.SelectedArea == 3 {
		c.RedrawChatback = true
	}
	if c.ObjDragArea == 3 {
		c.RedrawChatback = true
	}
	if c.ModalMessage != "" {
		c.RedrawChatback = true
	}
	if c.MenuVisible && c.MenuArea == 2 {
		c.RedrawChatback = true
	}
	if c.RedrawChatback {
		c.DrawChatback()
		c.RedrawChatback = false
	}
	if c.SceneState == 2 {
		c.DrawMinimap()
		//c.AreaMapback // TODO: pixmap
	}
	if c.FlashingTab != -1 {
		c.RedrawSideIcons = true
	}
	if c.RedrawSideIcons {
		if c.FlashingTab != -1 && c.FlashingTab == c.SelectedTab {
			c.FlashingTab = -1
			c.Out.P1Isaac(175)
			c.Out.P1(c.SelectedTab)
		}
		c.RedrawSideIcons = false
		c.AreaBackhmid1.Bind()
		c.ImageBackhmid1.Draw(0, 0)
		if c.SidebarInterfaceID == -1 {
			if c.TabInterfaceID[c.SelectedTab] != -1 {
				switch c.SelectedTab {
				case 0:
					c.ImageRedstone1.Draw(30, 29)
				case 1:
					c.ImageRedstone2.Draw(29, 59)
				case 2:
					c.ImageRedstone2.Draw(29, 87)
				case 3:
					c.ImageRedstone3.Draw(29, 115)
				case 4:
					c.ImageRedstone2h.Draw(29, 156)
				case 5:
					c.ImageRedstone2h.Draw(29, 184)
				case 6:
					c.ImageRedstone1h.Draw(30, 212)
				}
			}
			if c.TabInterfaceID[0] != -1 && (c.FlashingTab != 0 || LoopCycle%20 < 10) {
				c.ImageSideIcons[0].Draw(34, 35)
			}
			if c.TabInterfaceID[1] != -1 && (c.FlashingTab != 1 || LoopCycle%20 < 10) {
				c.ImageSideIcons[1].Draw(32, 59)
			}
			if c.TabInterfaceID[2] != -1 && (c.FlashingTab != 2 || LoopCycle%20 < 10) {
				c.ImageSideIcons[2].Draw(32, 86)
			}
			if c.TabInterfaceID[3] != -1 && (c.FlashingTab != 3 || LoopCycle%20 < 10) {
				c.ImageSideIcons[3].Draw(33, 121)
			}
			if c.TabInterfaceID[4] != -1 && (c.FlashingTab != 4 || LoopCycle%20 < 10) {
				c.ImageSideIcons[4].Draw(34, 157)
			}
			if c.TabInterfaceID[5] != -1 && (c.FlashingTab != 5 || LoopCycle%20 < 10) {
				c.ImageSideIcons[5].Draw(32, 185)
			}
			if c.TabInterfaceID[6] != -1 && (c.FlashingTab != 6 || LoopCycle%20 < 10) {
				c.ImageSideIcons[6].Draw(34, 212)
			}
		}
		c.AreaBackhmid1.Draw // TODO: pixmap
		c.AreaBackbase2.Bind()
		c.ImageBackbase2.Draw(0, 0)
		if c.SidebarInterfaceID == -1 {
			if c.TabInterfaceID[c.SelectedTab] != -1 {
				switch c.SelectedTab {
				case 7:
					c.ImageRedstone1v.Draw(0, 49)
				case 8:
					c.ImageRedstone2v.Draw(0, 81)
				case 9:
					c.ImageRedstone2v.Draw(0, 108)
				case 10:
					c.ImageRedstone3v.Draw(1, 136)
				case 11:
					c.ImageRedstone2hv.Draw(0, 178)
				case 12:
					c.ImageRedstone2hv.Draw(0, 205)
				case 13:
					c.ImageRedstone1hv.Draw(0, 233)
				}
			}
			if c.TabInterfaceID[8] != 1 && (c.FlashingTab != 8 || LoopCycle%20 < 10) {
				c.ImageSideIcons[7].Draw(2, 80)
			}
			if c.TabInterfaceID[9] != 1 && (c.FlashingTab != 9 || LoopCycle%20 < 10) {
				c.ImageSideIcons[8].Draw(3, 107)
			}
			if c.TabInterfaceID[10] != 1 && (c.FlashingTab != 10 || LoopCycle%20 < 10) {
				c.ImageSideIcons[9].Draw(4, 142)
			}
			if c.TabInterfaceID[11] != 1 && (c.FlashingTab != 11 || LoopCycle%20 < 10) {
				c.ImageSideIcons[10].Draw(2, 179)
			}
			if c.TabInterfaceID[12] != 1 && (c.FlashingTab != 12 || LoopCycle%20 < 10) {
				c.ImageSideIcons[11].Draw(2, 206)
			}
			if c.TabInterfaceID[13] != 1 && (c.FlashingTab != 13 || LoopCycle%20 < 10) {
				c.ImageSideIcons[12].Draw(2, 230)
			}
		}
		//c.AreaBackbase2.Draw // TODO: pixmap
		c.AreaViewport.Bind()
	}
	if c.RedrawPrivacySettings {
		c.RedrawPrivacySettings = false
		c.AreaBackbase1.Bind()
		c.ImageBackbase1.Draw(0, 0)
		c.FontPlain12.DrawStringTaggableCenter(57, 16777215, true, 33, "Public chat")
		switch c.PublicChatSetting {
		case 0:
			c.FontPlain12.DrawStringTaggableCenter(57, 65280, true, 46, "On")
		case 1:
			c.FontPlain12.DrawStringTaggableCenter(57, 16776960, true, 46, "Friends")
		case 2:
			c.FontPlain12.DrawStringTaggableCenter(57, 16711680, true, 46, "Off")
		case 3:
			c.FontPlain12.DrawStringTaggableCenter(57, 65535, true, 46, "Hide")
		}
		c.FontPlain12.DrawStringTaggableCenter(186, 16777215, true, 33, "Private chat")
		switch c.PrivateChatSetting {
		case 0:
			c.FontPlain12.DrawStringTaggableCenter(186, 65280, true, 46, "On")
		case 1:
			c.FontPlain12.DrawStringTaggableCenter(186, 16776960, true, 46, "Friends")
		case 2:
			c.FontPlain12.DrawStringTaggableCenter(186, 16711680, true, 46, "Off")
		}
		c.FontPlain12.DrawStringTaggableCenter(326, 16777215, true, 33, "Trade/duel")
		switch c.TradeChatSetting {
		case 0:
			c.FontPlain12.DrawStringTaggableCenter(326, 65280, true, 46, "On")
		case 1:
			c.FontPlain12.DrawStringTaggableCenter(326, 16776960, true, 46, "Friends")
		case 2:
			c.FontPlain12.DrawStringTaggableCenter(326, 16711680, true, 46, "Off")
		}
		c.FontPlain12.DrawStringTaggableCenter(462, 16777215, true, 38, "Report abuse")
		//c.AreaBackbase1.Draw() // TODO: pixmap
		c.AreaViewport.Bind()
	}
	c.SceneDelta = 0
}

func (c *Client) IsAddFriendOption(arg1 int) bool {
	if arg1 < 0 {
		return false
	}
	var3 := c.MenuAction[arg1]
	if var3 >= 2000 {
		var3 -= 2000
	}
	return var3 == 406
}

func (c *Client) UseMenuOption(arg1 int) {
	if arg1 < 0 {
		return
	}
	if c.ChatbackInputOpen {
		c.ChatbackInputOpen = false
		c.RedrawChatback = true
	}
	var3 := c.MenuParamB[arg1]
	var4 := c.MenuParamC[arg1]
	var5 := c.MenuAction[arg1]
	var6 := c.MenuParamA[arg1]
	if var5 >= 2000 {
		var5 -= 2000
	}
	var7 := ""
	var8 := 0
	var9 := ""
	var11 := 0
	if var5 == 903 || var5 == 363 {
		var7 = c.MenuOption[arg1]
		var8 = strings.Index(var7, "@whi@")
		if var8 != -1 {
			var7 = strings.TrimSpace(var7[var8+5:])
			var9 = datastruct.FormatName(datastruct.FromBase37(datastruct.ToBase37(var7)))
			var10 := false
			for i := range c.PlayerCount {
				var12 := c.Players[c.PlayerIDs[i]]
				if var12 != nil && var12.Name != "" && strings.EqualFold(var12.Name, var9) {
					c.TryMove(c.LocalPlayer.PathTileX[0], 1, false, var12.PathTileX[0], c.LocalPlayer.PathTileZ[0], 0, 2, 1, var12.PathTileZ[0], 0, 0, 0)
					if var5 == 903 {
						c.Out.P1Isaac(206)
					}
					if var5 == 363 {
						c.Out.P1Isaac(164)
					}
					c.Out.P2(c.PlayerIDs[i])
					var10 = true
					break
				}
			}
			if !var10 {
				c.AddMessage(0, "Unable to find "+var9, "")
			}
		}
	}
	if var5 == 450 && c.InteractWithLoc(75, var3, var4, var6) {
		c.Out.P2(c.ObjInterface)
		c.Out.P2(c.ObjSelectedSlot)
		c.Out.P2(c.ObjSelectedInterface)
	}
	if var5 == 405 || var5 == 38 || var5 == 422 || var5 == 478 || var5 == 347 {
		if var5 == 478 {
			if var3&0x3 == 0 {
				OpLogic5++
			}
			if OpLogic5 >= 90 {
				c.Out.P1Isaac(220)
			}
			c.Out.P1Isaac(157)
		}
		if var5 == 347 {
			c.Out.P1Isaac(211)
		}
		if var5 == 422 {
			c.Out.P1Isaac(133)
		}
		if var5 == 405 {
			OpLogic3 += var6
			if OpLogic3 >= 97 {
				c.Out.P1Isaac(30)
				c.Out.P3(14953816)
			}
			c.Out.P1Isaac(195)
		}
		if var5 == 38 {
			c.Out.P1Isaac(71)
		}
		c.Out.P2(var6)
		c.Out.P2(var3)
		c.Out.P2(var4)
		c.SelectedCycle = 0
		c.SelectedInterface = var4
		c.SelectedItem = var3
		c.SelectedArea = 2
		if component.Instances[var4].Layer == c.ViewportInterfaceID {
			c.SelectedArea = 1
		}
		if component.Instances[var4].Layer == c.ChatInterfaceID {
			c.SelectedArea = 3
		}
	}
	var var13 *entity.NpcEntity
	if var5 == 728 || var5 == 542 || var5 == 6 || var5 == 963 || var5 == 245 {
		var13 = c.NPCs[var6]
		if var13 != nil {
			c.TryMove(c.LocalPlayer.PathTileX[0], 1, false, var13.PathTileX[0], c.LocalPlayer.PathTileZ[0], 0, 2, 1, var13.PathTileZ[0], 0, 0, 0)
			c.CrossX = c.MouseClickX
			c.CrossY = c.MouseClickY
			c.CrossMode = 2
			c.CrossCycle = 0
			if var5 == 542 {
				c.Out.P1Isaac(8)
			}
			if var5 == 6 {
				if var6&0x3 == 0 {
					OpLogic2++
				}
				if OpLogic2 >= 124 {
					c.Out.P1Isaac(88)
					c.Out.P4(0)
				}
				c.Out.P1Isaac(27)
			}
			if var5 == 963 {
				c.Out.P1Isaac(113)
			}
			if var5 == 728 {
				c.Out.P1Isaac(194)
			}
			if var5 == 245 {
				if var6&0x3 == 0 {
					OpLogic4++
				}
				if OpLogic4 >= 85 {
					c.Out.P1Isaac(176)
					c.Out.P2(39596)
				}
				c.Out.P1Isaac(100)
			}
			c.Out.P2(var6)
		}
	}
	var14 := false
	if var5 == 217 {
		var14 = c.TryMove(c.LocalPlayer.PathTileX[0], 0, false, var3, c.LocalPlayer.PathTileZ[0], 0, 2, 0, var4, 0, 0, 0)
		if !var14 {
			c.TryMove(c.LocalPlayer.PathTileX[0], 1, false, var3, c.LocalPlayer.PathTileZ[0], 0, 2, 1, var4, 0, 0, 0)
		}
		c.CrossX = c.MouseClickX
		c.CrossY = c.MouseClickY
		c.CrossMode = 2
		c.CrossCycle = 0
		c.Out.P1Isaac(239)
		c.Out.P2(var3 + c.SceneBaseTileX)
		c.Out.P2(var4 + c.SceneBaseTileZ)
		c.Out.P2(var6)
		c.Out.P2(c.ObjInterface)
		c.Out.P2(c.ObjSelectedSlot)
		c.Out.P2(c.ObjSelectedInterface)
	}
	if var5 == 1175 {
		var15 := var6 >> 14 & 0x7FFF
		var16 := loctype.Get(var15)
		if var16.Desc == nil {
			var9 = "It's a " + var16.Name + "."
		} else {
			var9 = string(var16.Desc)
		}
		c.AddMessage(0, var9, "")
	}
	if var5 == 285 {
		c.InteractWithLoc(245, var3, var4, var6)
	}
	if var5 == 881 {
		c.Out.P1Isaac(130)
		c.Out.P2(var6)
		c.Out.P2(var3)
		c.Out.P2(var4)
		c.Out.P2(c.ObjInterface)
		c.Out.P2(c.ObjSelectedSlot)
		c.Out.P2(c.ObjSelectedInterface)
		c.SelectedCycle = 0
		c.SelectedInterface = var4
		c.SelectedItem = var3
		c.SelectedArea = 2
		if component.Instances[var4].Layer == c.ViewportInterfaceID {
			c.SelectedArea = 1
		}
		if component.Instances[var4].Layer == c.ChatInterfaceID {
			c.SelectedArea = 3
		}
	}
	if var5 == 391 {
		c.Out.P1Isaac(48)
		c.Out.P2(var6)
		c.Out.P2(var3)
		c.Out.P2(var4)
		c.Out.P2(c.ActiveSpellID)
		c.SelectedCycle = 0
		c.SelectedInterface = var4
		c.SelectedItem = var3
		c.SelectedArea = 2
		if component.Instances[var4].Layer == c.ViewportInterfaceID {
			c.SelectedArea = 1
		}
		if component.Instances[var4].Layer == c.ChatInterfaceID {
			c.SelectedArea = 3
		}
	}
	if var5 == 660 {
		if c.MenuVisible {
			c.Scene.Click(var4-11, var3-8)
		} else {
			c.Scene.Click(c.MouseClickY-11, c.MouseClickX-8)
		}
	}
	if var5 == 188 {
		c.ObjSelected = 1
		c.ObjSelectedSlot = var3
		c.ObjSelectedInterface = var4
		c.ObjInterface = var6
		c.ObjSelectedName = objtype.Get(var6).Name
		c.SpellSelected = 0
		return
	}
	if var5 == 44 && !c.PressedContinueOption {
		c.Out.P1Isaac(235)
		c.Out.P2(var4)
		c.PressedContinueOption = true
	}
	var var17 *objtype.ObjType
	var18 := ""
	if var5 == 1773 {
		var17 = objtype.Get(var6)
		if var4 >= 100000 {
			var18 = strconv.Itoa(var4) + " x " + var17.Name
		} else if var17.Desc == nil {
			var18 = "It's a " + var17.Name + "."
		} else {
			var18 = string(var17.Desc)
		}
		c.AddMessage(0, var18, "")
	}
	if var5 == 900 {
		var13 = c.NPCs[var6]
		if var13 != nil {
			c.TryMove(c.LocalPlayer.PathTileX[0], 1, false, var13.PathTileX[0], c.LocalPlayer.PathTileZ[0], 0, 2, 1, var13.PathTileZ[0], 0, 0, 0)
			c.CrossX = c.MouseClickX
			c.CrossY = c.MouseClickY
			c.CrossMode = 2
			c.CrossCycle = 0
			c.Out.P1Isaac(202)
			c.Out.P2(var6)
			c.Out.P2(c.ObjInterface)
			c.Out.P2(c.ObjSelectedSlot)
			c.Out.P2(c.ObjSelectedInterface)
		}
	}
	var var19 *playerentity.PlayerEntity
	if var5 == 1373 || var5 == 1544 || var5 == 151 || var5 == 1101 {
		var19 = c.Players[var6]
		if var19 != nil {
			c.TryMove(c.LocalPlayer.PathTileX[0], 1, false, var19.PathTileX[0], c.LocalPlayer.PathTileZ[0], 0, 2, 1, var19.PathTileZ[0], 0, 0, 0)
			c.CrossX = c.MouseClickX
			c.CrossY = c.MouseClickY
			c.CrossMode = 2
			c.CrossCycle = 0
			if var5 == 1101 {
				c.Out.P1Isaac(164)
			}
			if var5 == 151 {
				OpLogic8++
				if OpLogic8 >= 90 {
					c.Out.P1Isaac(2)
					c.Out.P2(31114)
				}
				c.Out.P1Isaac(53)
			}
			if var5 == 1373 {
				c.Out.P1Isaac(206)
			}
			if var5 == 1544 {
				c.Out.P1Isaac(185)
			}
			c.Out.P2(var6)
		}
	}
	if var5 == 265 {
		var13 = c.NPCs[var6]
		if var13 != nil {
			c.TryMove(c.LocalPlayer.PathTileX[0], 1, false, var13.PathTileX[0], c.LocalPlayer.PathTileZ[0], 0, 2, 1, var13.PathTileZ[0], 0, 0, 0)
			c.CrossX = c.MouseClickX
			c.CrossY = c.MouseClickY
			c.CrossMode = 2
			c.CrossCycle = 0
			c.Out.P1Isaac(134)
			c.Out.P2(var6)
			c.Out.P2(c.ActiveSpellID)
		}
	}
	var20 := int64(0)
	if var5 == 679 {
		var7 = c.MenuOption[arg1]
		var8 = strings.Index(var7, "@whi@")
		if var8 != -1 {
			var20 = datastruct.ToBase37(strings.TrimSpace(var7[var8+5:]))
			var11 = -1
			for i := range c.FriendCount {
				if c.FriendName37[i] == var20 {
					var11 = i
					break
				}
			}
			if var11 != -1 && c.FriendWorld[var11] > 0 {
				c.RedrawChatback = true
				c.ChatbackInputOpen = false
				c.ShowSocialInput = true
				c.SocialInput = ""
				c.SocialAction = 3
				c.SocialName37 = c.FriendName37[var11]
				c.SocialMessage = "Enter message to send to " + c.FriendName[var11]
			}
		}
	}
	if var5 == 55 && c.InteractWithLoc(9, var3, var4, var6) {
		c.Out.P2(c.ActiveSpellID)
	}
	if var5 == 224 || var5 == 993 || var5 == 99 || var5 == 746 || var5 == 877 {
		var14 = c.TryMove(c.LocalPlayer.PathTileX[0], 0, false, var3, c.LocalPlayer.PathTileZ[0], 0, 2, 0, var4, 0, 0, 0)
		if !var14 {
			c.TryMove(c.LocalPlayer.PathTileX[0], 1, false, var3, c.LocalPlayer.PathTileZ[0], 0, 2, 1, var4, 0, 0, 0)
		}
		c.CrossX = c.MouseClickX
		c.CrossY = c.MouseClickY
		c.CrossMode = 2
		c.CrossCycle = 0
		if var5 == 224 {
			c.Out.P1Isaac(140)
		}
		if var5 == 746 {
			c.Out.P1Isaac(178)
		}
		if var5 == 877 {
			c.Out.P1Isaac(247)
		}
		if var5 == 99 {
			c.Out.P1Isaac(200)
		}
		if var5 == 993 {
			c.Out.P1Isaac(40)
		}
		c.Out.P2(var3 + c.SceneBaseTileX)
		c.Out.P2(var4 + c.SceneBaseTileZ)
		c.Out.P2(var6)
	}
	if var5 == 1607 {
		var13 = c.NPCs[var6]
		if var13 != nil {
			if var13.Type.Desc == nil {
				var18 = "It's a " + var13.Type.Name + "."
			} else {
				var18 = string(var13.Type.Desc)
			}
			c.AddMessage(0, var18, "")
		}
	}
	if var5 == 504 {
		c.InteractWithLoc(172, var3, var4, var6)
	}
	var var22 *component.Component
	if var5 == 930 {
		var22 = component.Instances[var4]
		c.SpellSelected = 1
		c.ActiveSpellID = var4
		c.ActiveSpellFlags = var22.ActionTarget
		c.ObjSelected = 0
		var18 = var22.ActionVerb
		if strings.Index(var18, " ") != -1 {
			var18 = var18[0:strings.Index(var18, " ")]
		}
		var9 = var22.ActionVerb
		if strings.Index(var9, " ") != -1 {
			var9 = var9[strings.Index(var9, " ")+1:]
		}
		c.SpellCaption = var18 + " " + var22.Action + " " + var9
		if c.ActiveSpellFlags == 16 {
			c.RedrawSidebar = true
			c.SelectedTab = 3
			c.RedrawSideIcons = true
		}
		return
	}
	if var5 == 951 {
		var22 = component.Instances[var4]
		var23 := true
		if var22.ClientCode > 0 {
			var23 = c.HandleInterfaceAction(var22)
		}
		if var23 {
			c.Out.P1Isaac(155)
			c.Out.P2(var4)
		}
	}
	if var5 == 602 || var5 == 596 || var5 == 22 || var5 == 892 || var5 == 415 {
		if var5 == 22 {
			c.Out.P1Isaac(212)
		}
		if var5 == 415 {
			if var4&0x3 == 0 {
				OpLogic7++
			}
			if OpLogic7 >= 55 {
				c.Out.P1Isaac(17)
				c.Out.P4(0)
			}
			c.Out.P1Isaac(6)
		}
		if var5 == 602 {
			c.Out.P1Isaac(31)
		}
		if var5 == 892 {
			if var3&0x3 == 0 {
				OpLogic9++
			}
			if OpLogic9 >= 130 {
				c.Out.P1Isaac(238)
				c.Out.P1(177)
			}
			c.Out.P1Isaac(38)
		}
		if var5 == 596 {
			c.Out.P1Isaac(59)
		}
		c.Out.P2(var6)
		c.Out.P2(var3)
		c.Out.P2(var4)
		c.SelectedCycle = 0
		c.SelectedInterface = var4
		c.SelectedItem = var3
		c.SelectedArea = 2
		if component.Instances[var4].Layer == c.ViewportInterfaceID {
			c.SelectedArea = 1
		}
		if component.Instances[var4].Layer == c.ChatInterfaceID {
			c.SelectedArea = 3
		}
	}
	if var5 == 581 {
		if var6&0x3 == 0 {
			OpLogic1++
		}
		if OpLogic1 >= 99 {
			c.Out.P1Isaac(7)
			c.Out.P4(0)
		}
		c.InteractWithLoc(97, var3, var4, var6)
	}
	if var5 == 965 {
		var14 = c.TryMove(c.LocalPlayer.PathTileX[0], 0, false, var3, c.LocalPlayer.PathTileZ[0], 0, 2, 0, var4, 0, 0, 0)
		if !var14 {
			c.TryMove(c.LocalPlayer.PathTileX[0], 1, false, var3, c.LocalPlayer.PathTileZ[0], 0, 2, 1, var4, 0, 0, 0)
		}
		c.CrossX = c.MouseClickX
		c.CrossY = c.MouseClickY
		c.CrossMode = 2
		c.CrossCycle = 0
		c.Out.P1Isaac(138)
		c.Out.P2(var3 + c.SceneBaseTileX)
		c.Out.P2(var4 + c.SceneBaseTileZ)
		c.Out.P2(var6)
		c.Out.P2(c.ActiveSpellID)
	}
	if var5 == 1501 {
		OpLogic6 += c.SceneBaseTileZ
		if OpLogic6 >= 92 {
			c.Out.P1Isaac(66)
			c.Out.P4(0)
		}
		c.InteractWithLoc(116, var3, var4, var6)
	}
	if var5 == 364 {
		c.InteractWithLoc(96, var3, var4, var6)
	}
	if var5 == 1102 {
		var17 = objtype.Get(var6)
		if var17.Desc == nil {
			var18 = "It's a " + var17.Name + "."
		} else {
			var18 = string(var17.Desc)
		}
		c.AddMessage(0, var18, "")
	}
	if var5 == 960 {
		c.Out.P1Isaac(155)
		c.Out.P2(var4)
		var22 = component.Instances[var4]
		if var22.Scripts != nil && var22.Scripts[0][0] == 5 {
			var8 = var22.Scripts[0][1]
			if c.Varps[var8] != var22.ScriptOperand[0] {
				c.Varps[var8] = var22.ScriptOperand[0]
				c.UpdateVarp(var8)
				c.RedrawSidebar = true
			}
		}
	}
	if var5 == 34 {
		var7 = c.MenuOption[arg1]
		var8 = strings.Index(var7, "@whi@")
		if var8 != -1 {
			c.CloseInterfaces()
			c.ReportAbuseInput = strings.TrimSpace(var7[var8+5:])
			c.ReportAbuseMuteOption = false
			for i := range len(component.Instances) {
				if component.Instances[i] != nil && component.Instances[i].ClientCode == 600 {
					c.ViewportInterfaceID = component.Instances[i].Layer
					c.ReportAbuseInterfaceID = c.ViewportInterfaceID
					break
				}
			}
		}
	}
	if var5 == 947 {
		c.CloseInterfaces()
	}
	if var5 == 367 {
		var19 = c.Players[var6]
		if var19 != nil {
			c.TryMove(c.LocalPlayer.PathTileX[0], 1, false, var19.PathTileX[0], c.LocalPlayer.PathTileZ[0], 0, 2, 1, var19.PathTileZ[0], 0, 0, 0)
			c.CrossX = c.MouseClickX
			c.CrossY = c.MouseClickY
			c.CrossMode = 2
			c.CrossCycle = 0
			c.Out.P1Isaac(248)
			c.Out.P2(var6)
			c.Out.P2(c.ObjInterface)
			c.Out.P2(c.ObjSelectedSlot)
			c.Out.P2(c.ObjSelectedInterface)
		}
	}
	if var5 == 465 {
		c.Out.P1Isaac(155)
		c.Out.P2(var4)
		var22 = component.Instances[var4]
		if var22.Scripts != nil && var22.Scripts[0][0] == 5 {
			var8 = var22.Scripts[0][1]
			c.Varps[var8] = 1 - c.Varps[var8]
			c.UpdateVarp(var8)
			c.RedrawSidebar = true
		}
	}
	if var5 == 406 || var5 == 436 || var5 == 557 || var5 == 556 {
		var7 = c.MenuOption[arg1]
		var8 = strings.Index(var7, "@whi@")
		if var8 != -1 {
			var20 = datastruct.ToBase37(strings.TrimSpace(var7[var8+5:]))
			if var5 == 406 {
				c.AddFriend(var20)
			}
			if var5 == 436 {
				c.AddIgnore(var20)
			}
			if var5 == 557 {
				c.RemoveFriend(var20)
			}
			if var5 == 556 {
				c.RemoveIgnore(var20)
			}
		}
	}
	if var5 == 651 {
		var19 = c.Players[var6]
		if var19 != nil {
			c.TryMove(c.LocalPlayer.PathTileX[0], 1, false, var19.PathTileX[0], c.LocalPlayer.PathTileZ[0], 0, 2, 1, var19.PathTileZ[0], 0, 0, 0)
			c.CrossX = c.MouseClickX
			c.CrossY = c.MouseClickY
			c.CrossMode = 2
			c.CrossCycle = 0
			c.Out.P1Isaac(177)
			c.Out.P2(var6)
			c.Out.P2(c.ActiveSpellID)
		}
	}
	c.ObjSelected = 0
	c.SpellSelected = 0
}

func GetCombatLevelColorTag(arg0 int, arg2 int) string {
	var3 := arg0 - arg2
	if var3 < -9 {
		return "@red@"
	}
	if var3 < -6 {
		return "@or3@"
	}
	if var3 < -3 {
		return "@or2@"
	}
	if var3 < 0 {
		return "@or1@"
	}
	if var3 > 9 {
		return "@gre@"
	}
	if var3 > 6 {
		return "@gr3@"
	}
	if var3 > 3 {
		return "@gr2@"
	}
	if var3 > 0 {
		return "@gr1@"
	}
	return "@yel@"
}

func (c *Client) GetHost() string {
	// TODO
}

func (c *Client) DrawMenu() {
	var2 := c.MenuX
	var3 := c.MenuY
	var4 := c.MenuWidth
	var5 := c.MenuHeight
	var6 := 6116423
	pix2d.FillRect(var3, var2, var6, var4, var5)
	pix2d.FillRect(var3+1, var2+1, 0, var4-2, 16)
	pix2d.DrawRect(var2+1, 0, var5-19, var3+18, var4-2)
	c.FontBold12.DrawString(var2+3, var3+14, var6, "Choose Option")
	var7 := c.MouseX
	var8 := c.MouseY
	switch c.MenuArea {
	case 0:
		var7 -= 8
		var8 -= 11
	case 1:
		var7 -= 562
		var8 -= 231
	case 2:
		var7 -= 22
		var8 -= 375
	}
	for i := range c.MenuSize {
		var10 := var3 + 31 + (c.MenuSize-1-i)*15
		var11 := 16777215
		if var7 > var2 && var7 < var2+var4 && var8 > var10-13 && var8 < var10+3 {
			var11 = 16776960
		}
		c.FontBold12.DrawStringTaggable(var2+3, var10, c.MenuOption[i], true, var11)
	}
}

func (c *Client) HandlePrivateChatInput(arg2 int) {
	if c.SplitPrivateChat == 0 {
		return
	}
	var4 := 0
	if c.SystemUpdateTimer != 0 {
		var4 = 1
	}
	for i := range 100 {
		if c.MessageText[i] != "" {
			var6 := c.MessageType[i]
			if (var6 == 3 || var6 == 7) && (var6 == 7 || c.PrivateChatSetting == 0 || c.PrivateChatSetting == 1 && c.IsFriend(c.MessageSender[i])) {
				var7 := 329 - var4*13
				if c.MouseX > 8 && c.MouseX < 520 && arg2-11 > var7-10 && arg2-11 <= var7+3 {
					if c.Rights {
						c.MenuOption[c.MenuSize] = "Report abuse @whi@" + c.MessageSender[i]
						c.MenuAction[c.MenuSize] = 2034
						c.MenuSize++
					}
					c.MenuOption[c.MenuSize] = "Add ignore @whi@" + c.MessageSender[i]
					c.MenuAction[c.MenuSize] = 2436
					c.MenuSize++
					c.MenuOption[c.MenuSize] = "Add friend @whi@" + c.MessageSender[i]
					c.MenuAction[c.MenuSize] = 2406
					c.MenuSize++
				}
				var4++
				if var4 >= 5 {
					return
				}
			}
			if (var6 == 5 || var6 == 6) && c.PrivateChatSetting < 2 {
				var4++
				if var4 >= 5 {
					return
				}
			}
		}
	}
}

func (c *Client) UpdateInterfaceContent(arg1 *component.Component) {
	var3 := arg1.ClientCode
	if var3 >= 1 && var3 <= 100 {
		var3--
		if var3 >= c.FriendCount {
			arg1.Text = ""
			arg1.ButtonType = 0
		} else {
			arg1.Text = c.FriendName[var3]
			arg1.ButtonType = 1
		}
	} else if var3 >= 101 && var3 <= 200 {
		var3 -= 101
		if var3 >= c.FriendCount {
			arg1.Text = ""
			arg1.ButtonType = 0
		} else {
			if c.FriendWorld[var3] == 0 {
				arg1.Text = "@red@Offline"
			} else if c.FriendWorld[var3] == NodeID {
				arg1.Text = "@gre@World-" + strconv.Itoa(c.FriendWorld[var3]-9)
			} else {
				arg1.Text = "@yel@World-" + strconv.Itoa(c.FriendWorld[var3]-9)
			}
			arg1.ButtonType = 1
		}
	} else if var3 == 203 {
		arg1.Scroll = c.FriendCount*15 + 20
		if arg1.Scroll <= arg1.Height {
			arg1.Scroll = arg1.Height + 1
		}
	} else if var3 >= 401 && var3 <= 500 {
		var3 -= 401
		if var3 >= c.IgnoreCount {
			arg1.Text = ""
			arg1.ButtonType = 0
		} else {
			arg1.Text = datastruct.FormatName(datastruct.FromBase37(c.IgnoreName37[var3]))
			arg1.ButtonType = 1
		}
	} else if var3 == 503 {
		arg1.Scroll = c.IgnoreCount*15 + 20
		if arg1.Scroll <= arg1.Height {
			arg1.Scroll = arg1.Height + 1
		}
	} else if var3 == 327 {
		arg1.Xan = 150
		arg1.Yan = int(math.Sin(float64(LoopCycle)/40.0)*256.0) & 0x7FF
		if c.UpdateDesignModel {
			c.UpdateDesignModel = false
			var9 := make([]*model.Model, 7)
			var5 := 0
			for i := range 7 {
				var7 := c.DesignIdentikits[i]
				if var7 >= 0 {
					var9[var5] = idktype.Instances[var7].GetModel()
					var5++
				}
			}
			var10 := model.NewModel2(var9, var5)
			for i := range 5 {
				if c.DesignColors[i] != 0 {
					var10.Recolor(Field1307[i][0], Field1307[i][c.DesignColors[i]])
					if i == 1 {
						var10.Recolor(Field1438[0], Field1438[c.DesignColors[i]])
					}
				}
			}
			var10.CreateLabelReferences()
			var10.ApplyTransform(seqtype.Instances[c.LocalPlayer.SeqStandID].Frames[0])
			var10.CalculateNormals(64, 850, -30, -50, -30, true)
			arg1.Model = var10
		}
	} else if var3 == 324 {
		if c.GenderButtonImage0 == nil {
			c.GenderButtonImage0 = arg1.Graphic
			c.GenderButtonImage1 = arg1.ActiveGraphic
		}
		if c.DesignGenderMale {
			arg1.Graphic = c.GenderButtonImage1
		} else {
			arg1.Graphic = c.GenderButtonImage0
		}
	} else if var3 == 325 {
		if c.GenderButtonImage0 == nil {
			c.GenderButtonImage0 = arg1.Graphic
			c.GenderButtonImage1 = arg1.ActiveGraphic
		}
		if c.DesignGenderMale {
			arg1.Graphic = c.GenderButtonImage0
		} else {
			arg1.Graphic = c.GenderButtonImage1
		}
	} else if var3 == 600 {
		arg1.Text = c.ReportAbuseInput
		if LoopCycle%20 < 10 {
			arg1.Text = arg1.Text + "|"
		} else {
			arg1.Text = arg1.Text + " "
		}
	} else {
		if var3 == 613 {
			if !c.Rights {
				arg1.Text = ""
			} else if c.ReportAbuseMuteOption {
				arg1.Colour = 16711680
				arg1.Text = "Moderator option: Mute player for 48 hours: <ON>"
			} else {
				arg1.Colour = 16777215
				arg1.Text = "Moderator option: Mute player for 48 hours: <OFF>"
			}
		}
		var4 := ""
		if var3 == 650 || var3 == 655 {
			if c.LastAddress == 0 {
				arg1.Text = ""
			} else {
				if c.DaysSinceLastLogin == 0 {
					var4 = "earlier today"
				} else if c.DaysSinceLastLogin == 1 {
					var4 = "yesterday"
				} else {
					var4 = strconv.Itoa(c.DaysSinceLastLogin) + " days ago"
				}
				arg1.Text = "You last logged in " + var4 + " from: " + signlink.DNS
			}
		}
		if var3 == 651 {
			if c.UnreadMessages == 0 {
				arg1.Text = "0 unread messages"
				arg1.Colour = 16776960
			}
			if c.UnreadMessages == 1 {
				arg1.Text = "1 unread message"
				arg1.Colour = 65280
			}
			if c.UnreadMessages > 1 {
				arg1.Text = strconv.Itoa(c.UnreadMessages) + " unread messages"
				arg1.Colour = 65280
			}
		}
		if var3 == 652 {
			if c.DaysSinceRecoveriesChanged == 201 {
				arg1.Text = ""
			} else if c.DaysSinceRecoveriesChanged == 200 {
				arg1.Text = "You have not yet set any password recovery questions."
			} else {
				if c.DaysSinceRecoveriesChanged == 0 {
					var4 = "Earlier today"
				} else if c.DaysSinceRecoveriesChanged == 1 {
					var4 = "Yesterday"
				} else {
					var4 = strconv.Itoa(c.DaysSinceRecoveriesChanged) + " days ago"
				}
				arg1.Text = var4 + " you changed your recovery questions"
			}
		}
		if var3 == 653 {
			if c.DaysSinceRecoveriesChanged == 201 {
				arg1.Text = ""
			} else if c.DaysSinceRecoveriesChanged == 200 {
				arg1.Text = "We strongly recommend you do so now to secure your account."
			} else {
				arg1.Text = "If you do not remember making this change then cancel it immediately"
			}
		}
		if var3 == 654 {
			if c.DaysSinceRecoveriesChanged == 201 {
				arg1.Text = ""
			} else if c.DaysSinceRecoveriesChanged == 200 {
				arg1.Text = "Do this from the 'account management' area on our front webpage"
			} else {
				arg1.Text = "Do this from the 'account management' area on our front webpage"
			}
		}
	}
}

func (c *Client) SaveWave(arg0 []byte, arg1 int) bool {
	if arg0 == nil {
		return true
	}
	// TODO: signlink.wavesave
}

func (c *Client) ReplayWave() bool {
	// TODO: signlink.wavereplay
}

func (c *Client) SetWaveVolume(vol int) {
	signlink.WaveVol = vol
}

func (c *Client) GetNpcPosNewVis(arg1 *io.Packet, arg2 int) {
	for arg1.BitPos+21 < arg2*8 {
		var4 := arg1.GBit(13)
		if var4 == 8191 {
			break
		}
		if c.NPCs[var4] == nil {
			c.NPCs[var4] = entity.NewNpcEntity()
		}
		var5 := c.NPCs[var4]
		c.NPCIDs[c.NPCCount] = var4
		c.NPCCount++
		var5.Cycle = LoopCycle
		var5.Type = npctype.Get(arg1.GBit(11))
		var5.Size = int(var5.Type.Size)
		var5.SeqWalkID = var5.Type.WalkAnim
		var5.SeqTurnAroundID = var5.Type.WalkAnimB
		var5.SeqTurnLeftID = var5.Type.WalkAnimR
		var5.SeqTurnRightId = var5.Type.WalkAnimL
		var5.SeqStandID = var5.Type.ReadyAnim
		var6 := arg1.GBit(5)
		if var6 > 15 {
			var6 -= 32
		}
		var7 := arg1.GBit(5)
		if var7 > 15 {
			var7 -= 32
		}
		var5.Teleport(false, c.LocalPlayer.PathTileX[0]+var6, c.LocalPlayer.PathTileZ[0]+var7)
		var8 := arg1.GBit(1)
		if var8 == 1 {
			c.EntityUpdateIDs[c.EntityUpdateCount] = var4
			c.EntityUpdateCount++
		}
	}
	arg1.AccessBytes()
}

func (c *Client) HandleInterfaceAction(arg1 *component.Component) bool {
	var3 := arg1.ClientCode
	switch var3 {
	case 201:
		c.RedrawChatback = true
		c.ChatbackInputOpen = false
		c.ShowSocialInput = true
		c.SocialInput = ""
		c.SocialAction = 1
		c.SocialMessage = "Enter name of friend to add to list"
	case 202:
		c.RedrawChatback = true
		c.ChatbackInputOpen = false
		c.ShowSocialInput = true
		c.SocialInput = ""
		c.SocialAction = 2
		c.SocialMessage = "Enter name of friend to delete from list"
	case 205:
		c.IdleTimeout = 250
		return true
	case 501:
		c.RedrawChatback = true
		c.ChatbackInputOpen = false
		c.ShowSocialInput = true
		c.SocialInput = ""
		c.SocialAction = 4
		c.SocialMessage = "Enter name of player to add to list"
	case 502:
		c.RedrawChatback = true
		c.ChatbackInputOpen = false
		c.ShowSocialInput = true
		c.SocialInput = ""
		c.SocialAction = 5
		c.SocialMessage = "Enter name of player to delete from list"
	}
	var4 := 0
	var5 := 0
	var6 := 0
	if var3 >= 300 && var3 <= 313 {
		var4 = (var3 - 300) / 2
		var5 = var3 & 0x1
		var6 = c.DesignIdentikits[var4]
		if var6 != -1 {
			for {
				if var5 == 0 {
					var6--
					if var6 < 0 {
						var6 = idktype.Count - 1
					}
				}
				if var5 == 1 {
					var6++
					if var6 >= idktype.Count {
						var6 = 0
					}
				}
				tmp := 0
				if !c.DesignGenderMale {
					tmp = 7
				}
				if !idktype.Instances[var6].Disable && idktype.Instances[var6].Type == var4+tmp {
					c.DesignIdentikits[var4] = var6
					c.UpdateDesignModel = true
					break
				}
			}
		}
	}
	if var3 >= 314 && var3 <= 323 {
		var4 = (var3 - 314) / 2
		var5 = var3 & 0x1
		var6 = c.DesignColors[var4]
		if var5 == 0 {
			var6--
			if var6 < 0 {
				var6 = len(Field1307[var4]) - 1
			}
		}
		if var5 == 1 {
			var6++
			if var6 >= len(Field1307[var4]) {
				var6 = 0
			}
		}
		c.DesignColors[var4] = var6
		c.UpdateDesignModel = true
	}
	if var3 == 324 && !c.DesignGenderMale {
		c.DesignGenderMale = true
		c.ValidateCharacterDesign()
	}
	if var3 == 325 && c.DesignGenderMale {
		c.DesignGenderMale = false
		c.ValidateCharacterDesign()
	}
	if var3 == 326 {
		c.Out.P1Isaac(52)
		if c.DesignGenderMale {
			c.Out.P1(0)
		} else {
			c.Out.P1(1)
		}
		for i := range 7 {
			c.Out.P1(c.DesignIdentikits[i])
		}
		for i := range 5 {
			c.Out.P1(c.DesignColors[i])
		}
		return true
	}
	if var3 == 613 {
		c.ReportAbuseMuteOption = !c.ReportAbuseMuteOption
	}
	if var3 >= 601 && var3 <= 612 {
		c.CloseInterfaces()
		if len(c.ReportAbuseInput) > 0 {
			c.Out.P1Isaac(190)
			c.Out.P8(datastruct.ToBase37(c.ReportAbuseInput))
			c.Out.P1(var3 - 601)
			if c.ReportAbuseMuteOption {
				c.Out.P1(1)
			} else {
				c.Out.P1(0)
			}
		}
	}
	return false
}

func (c *Client) Load() {
	if signlink.SunJava {
		c.MinDel = 5
	}
	if !LowMemory {
		c.StartMidiThread = true
		c.MidiThreadActive = true
		//c.startthread(this, 2) // TODO: this.startthread
		c.SetMidi(12345678, "scape_main", 40000)
	}
	if Started {
		c.ErrorStarted = true
		return
	}
	Started = true
	var1 := false
	var2 := c.GetHost()
	if strings.HasSuffix(var2, "jagex.com") {
		var1 = true
	}
	if strings.HasSuffix(var2, "runescape.com") {
		var1 = true
	}
	if strings.HasSuffix(var2, "192.168.1.2") {
		var1 = true
	}
	if strings.HasSuffix(var2, "192.168.1.249") {
		var1 = true
	}
	if strings.HasSuffix(var2, "192.168.1.252") {
		var1 = true
	}
	if strings.HasSuffix(var2, "192.168.1.253") {
		var1 = true
	}
	if strings.HasSuffix(var2, "192.168.1.254") {
		var1 = true
	}
	if strings.HasSuffix(var2, "127.0.0.1") {
		var1 = true
	}
	if !var1 {
		c.ErrorHost = true
		return
	}
	// TODO: try/except - recover panic?
	var3 := 5
	c.ArchiveChecksum[8] = 0
	for c.ArchiveChecksum[8] == 0 {
		c.DrawProgress("Connecting to fileserver", 10)
		// TODO: try/except - error loading retry
		var35 := c.OpenURL("crc" + strconv.Itoa(int(rand.Float64()*9.9999999e7)))
		var5 := io.NewPacket(make([]byte, 36))
		var35.ReadFully(var5.Data, 0, 36)
		for i := range 9 {
			c.ArchiveChecksum[i] = var5.G4()
		}
		var35.Close()
	}
	c.ArchiveTitle = c.LoadArchive("title screen", c.ArchiveChecksum[1], "title", 10)
	c.FontPlain11 = pixfont.NewPixFont(c.ArchiveTitle, "p11")
	c.FontPlain12 = pixfont.NewPixFont(c.ArchiveTitle, "p12")
	c.FontBold12 = pixfont.NewPixFont(c.ArchiveTitle, "b12")
	c.FontQuill8 = pixfont.NewPixFont(c.ArchiveTitle, "q8")
	c.LoadTitleBackground()
	c.LoadTitleImages()
	var36 := c.LoadArchive("config", c.ArchiveChecksum[2], "config", 15)
	var37 := c.LoadArchive("interface", c.ArchiveChecksum[3], "interface", 20)
	var38 := c.LoadArchive("2d graphics", c.ArchiveChecksum[4], "media", 30)
	var7 := c.LoadArchive("3d graphics", c.ArchiveChecksum[5], "models", 40)
	var8 := c.LoadArchive("textures", c.ArchiveChecksum[6], "textures", 60)
	var9 := c.LoadArchive("chat system", c.ArchiveChecksum[7], "wordenc", 65)
	var10 := c.LoadArchive("sound effects", c.ArchiveChecksum[8], "sounds", 70)
	c.LevelTileFlags = make([][][]byte, 4)
	for i := range c.LevelTileFlags {
		c.LevelTileFlags[i] = make([][]byte, 104)
		for j := range c.LevelTileFlags[i] {
			c.LevelTileFlags[i][j] = make([]byte, 104)
		}
	}
	c.LevelHeightmap = make([][][]int, 4)
	for i := range c.LevelHeightmap {
		c.LevelHeightmap[i] = make([][]int, 105)
		for j := range c.LevelHeightmap[i] {
			c.LevelHeightmap[i][j] = make([]int, 105)
		}
	}
	c.Scene = world3d.NewWorld3D(c.LevelHeightmap, 104, 4, 104)
	for i := range 4 {
		c.LevelCollisionMap[i] = dash3d.NewCollisionMap(104, 104)
	}
	c.ImageMinimap = pix32.NewPix321(512, 512)
	c.DrawProgress("Unpacking media", 75)
	c.ImageInvback = pix8.NewPix8(var38, "invback", 0)
	c.ImageChatback = pix8.NewPix8(var38, "chatback", 0)
	c.ImageMapback = pix8.NewPix8(var38, "mapback", 0)
	c.ImageBackbase1 = pix8.NewPix8(var38, "backbase1", 0)
	c.ImageBackbase2 = pix8.NewPix8(var38, "backbase2", 0)
	c.ImageBackhmid1 = pix8.NewPix8(var38, "backhmid1", 0)
	for i := range 13 {
		c.ImageSideIcons[i] = pix8.NewPix8(var8, "sideicons", i)
	}
	c.ImageCompass = pix32.NewPix323(var38, "compass", 0)
	for i := range 50 {
		c.ImageMapscene[i] = pix8.NewPix8(var38, "mapscene", i)
	}
	for i := range 50 {
		c.ImageMapFunction[i] = pix32.NewPix323(var8, "mapfunction", i)
	}
	for i := range 20 {
		c.ImageHitmarks[i] = pix32.NewPix323(var38, "hitmarks", i)
	}
	for i := range 20 {
		c.ImageHeadIcons[i] = pix32.NewPix323(var38, "headicons", i)
	}
	c.ImageMapflag = pix32.NewPix323(var38, "mapflag", 0)
	for i := range 8 {
		c.ImageCrosses[i] = pix32.NewPix323(var38, "cross", i)
	}
	c.ImageMapdot0 = pix32.NewPix323(var38, "mapdots", 0)
	c.ImageMapdot1 = pix32.NewPix323(var38, "mapdots", 1)
	c.ImageMapdot2 = pix32.NewPix323(var38, "mapdots", 2)
	c.ImageMapdot3 = pix32.NewPix323(var38, "mapdots", 3)
	c.ImageScrollbar0 = pix8.NewPix8(var38, "scrollbar", 0)
	c.ImageScrollbar1 = pix8.NewPix8(var38, "scrollbar", 1)
	c.ImageRedstone1 = pix8.NewPix8(var38, "redstone1", 0)
	c.ImageRedstone2 = pix8.NewPix8(var38, "redstone2", 0)
	c.ImageRedstone3 = pix8.NewPix8(var38, "redstone3", 0)
	c.ImageRedstone1h = pix8.NewPix8(var38, "redstone1", 0)
	c.ImageRedstone1h.FlipHorizontally()
	c.ImageRedstone2h = pix8.NewPix8(var38, "redstone2", 0)
	c.ImageRedstone2h.FlipHorizontally()
	c.ImageRedstone1v = pix8.NewPix8(var38, "redstone1", 0)
	c.ImageRedstone1v.FlipVertically()
	c.ImageRedstone2v = pix8.NewPix8(var38, "redstone2", 0)
	c.ImageRedstone2v.FlipVertically()
	c.ImageRedstone3v = pix8.NewPix8(var38, "redstone3", 0)
	c.ImageRedstone3v.FlipVertically()
	c.ImageRedstone1hv = pix8.NewPix8(var38, "redstone1", 0)
	c.ImageRedstone1hv.FlipHorizontally()
	c.ImageRedstone1hv.FlipVertically()
	c.ImageRedstone2hv = pix8.NewPix8(var38, "redstone2", 0)
	c.ImageRedstone2hv.FlipHorizontally()
	c.ImageRedstone2hv.FlipVertically()
	var14 := pix32.NewPix323(var38, "backleft1", 0)
	//c.AreaBackleft1 = // TODO: pixmap
	var14.BlitOpaque(0, 0)
	var39 := pix32.NewPix323(var38, "backleft2", 0)
	// TODO: pixmap
	var39.BlitOpaque(0, 0)
	var40 := pix32.NewPix323(var38, "backright1", 0)
	// TODO: pixmap
	var40.BlitOpaque(0, 0)
	var41 := pix32.NewPix323(var38, "backright2", 0)
	// TODO: pixmap
	var41.BlitOpaque(0, 0)
	var42 := pix32.NewPix323(var38, "backtop1", 0)
	// TODO: pixmap
	var42.BlitOpaque(0, 0)
	var43 := pix32.NewPix323(var38, "backtop2", 0)
	// TODO: pixmap
	var43.BlitOpaque(0, 0)
	var44 := pix32.NewPix323(var38, "backvmid1", 0)
	// TODO: pixmap
	var44.BlitOpaque(0, 0)
	var45 := pix32.NewPix323(var38, "backvmid2", 0)
	// TODO: pixmap
	var45.BlitOpaque(0, 0)
	var46 := pix32.NewPix323(var38, "backvmid3", 0)
	// TODO: pixmap
	var46.BlitOpaque(0, 0)
	var47 := pix32.NewPix323(var38, "backhmid2", 0)
	// TODO: pixmap
	var47.BlitOpaque(0, 0)
	var15 := int(rand.Float64()*21.0) - 10
	var16 := int(rand.Float64()*21.0) - 10
	var17 := int(rand.Float64()*21.0) - 10
	var18 := int(rand.Float64()*41.0) - 20
	for i := range 50 {
		if c.ImageMapFunction[i] != nil {
			c.ImageMapFunction[i].Translate(var15+var18, var16+var18, var17+var18)
		}
		if c.ImageMapscene[i] != nil {
			c.ImageMapscene[i].Translate(var15+var18, var16+var18, var17+var18)
		}
	}
	c.DrawProgress("Unpacking textures", 80)
	pix3d.UnpackTextures(var8)
	pix3d.SetBrightness(0.8)
	pix3d.InitPool(20)
	c.DrawProgress("Unpacking models", 83)
	model.Unpack(var7)
	animbase.Unpack(var7)
	animframe.Unpack(var7)
	c.DrawProgress("Unpacking config", 86)
	seqtype.Unpack(var36)
	loctype.Unpack(var36)
	flotype.Unpack(var36)
	objtype.Unpack(var36)
	npctype.Unpack(var36)
	idktype.Unpack(var36)
	spotanimtype.Unpack(var36)
	varptype.Unpack(var36)
	objtype.MembersWorld = Members
	if !LowMemory {
		c.DrawProgress("Unpacking sounds", 90)
		var20 := var10.Read("sounds.dat", nil)
		var21 := io.NewPacket(var20)
		// TODO: wave.unpack
	}
	c.DrawProgress("Unpacking interfaces", 92)
	var48 := []*pixfont.PixFont{c.FontPlain11, c.FontPlain12, c.FontBold12, c.FontQuill8}
	component.Unpack(var38, var48, var37)
	c.DrawProgress("Preparing game engine", 97)
	for i := range 33 {
		var22 := 999
		var23 := 0
		for j := range 35 {
			if c.ImageMapback.Pixels[j+i*c.ImageMapback.Width] == 0 {
				if var22 == 999 {
					var22 = j
				}
			} else if var22 != 999 {
				var23 = j
				break
			}
		}
		c.CompassMaskLineOffsets[i] = var22
		c.CompassMaskLineLengths[i] = var23 - var22
	}
	for i := 9; i < 160; i++ {
		var23 := 999
		var24 := 0
		for j := 10; j < 168; j++ {
			if c.ImageMapback.Pixels[j+i*c.ImageMapback.Width] == 0 && (j > 34 || i > 34) {
				if var23 == 999 {
					var23 = j
				}
			} else if var23 != 999 {
				var24 = j
				break
			}
		}
		c.MinimapMaskLineOffsets[i-9] = var23 - 21
		c.MinimapMaskLineLengths[i-9] = var24 - var23
	}
	pix3d.Init3D(96, 479)
	c.AreaChatbackOffsets = pix3d.LineOffset
	pix3d.Init3D(261, 190)
	c.AreaSidebarOffsets = pix3d.LineOffset
	pix3d.Init3D(334, 512)
	c.AreaViewportOffsets = pix3d.LineOffset
	var50 := make([]int, 9)
	for i := range 9 {
		var25 := i*32 + 128 + 15
		var26 := var25*3 + 600
		var27 := pix3d.SinTable[var25]
		var50[i] = var26 * var27 >> 16
	}
	world3d.Init(var50, 800, 512, 334, 500)
	// TODO: wordfilter.unpack
}

func (c *Client) HandleInput() {
	if c.ObjDragArea != 0 {
		return
	}
	c.MenuOption[0] = "Cancel"
	c.MenuAction[0] = 1252
	c.MenuSize = 1
	c.HandlePrivateChatInput(c.MouseY)
	c.LastHoveredInterfaceID = 0
	if c.MouseX > 8 && c.MouseY > 11 && c.MouseX < 520 && c.MouseY < 345 {
		if c.ViewportInterfaceID == -1 {
			c.HandleViewportOptions()
		} else {
			c.HandleInterfaceInput(c.MouseY, c.MouseX, 11, component.Instances[c.ViewportInterfaceID], 8, 0)
		}
	}
	if c.LastHoveredInterfaceID != c.ViewportHoveredInterfaceIndex {
		c.ViewportHoveredInterfaceIndex = c.LastHoveredInterfaceID
	}
	c.LastHoveredInterfaceID = 0
	if c.MouseX > 562 && c.MouseY > 231 && c.MouseX < 752 && c.MouseY < 492 {
		if c.SidebarInterfaceID != -1 {
			c.HandleInterfaceInput(c.MouseY, c.MouseX, 231, component.Instances[c.SidebarInterfaceID], 562, 0)
		} else if c.TabInterfaceID[c.SelectedTab] != -1 {
			c.HandleInterfaceInput(c.MouseY, c.MouseX, 231, component.Instances[c.TabInterfaceID[c.SelectedTab]], 562, 0)
		}
	}
	if c.LastHoveredInterfaceID != c.SidebarHoveredInterfaceIndex {
		c.RedrawSidebar = true
		c.SidebarHoveredInterfaceIndex = c.LastHoveredInterfaceID
	}
	c.LastHoveredInterfaceID = 0
	if c.MouseX > 22 && c.MouseY > 375 && c.MouseX < 431 && c.MouseY < 471 {
		if c.ChatInterfaceID == -1 {
			c.HandleChatMouseInput(c.MouseY-375, 0)
		} else {
			c.HandleInterfaceInput(c.MouseY, c.MouseX, 375, component.Instances[c.ChatInterfaceID], 22, 0)
		}
	}
	if c.ChatInterfaceID != -1 && c.LastHoveredInterfaceID != c.ChatHoveredInterfaceIndex {
		c.RedrawChatback = true
		c.ChatHoveredInterfaceIndex = c.LastHoveredInterfaceID
	}
	var2 := false
	for !var2 {
		var2 = true
		for i := range c.MenuSize - 1 {
			if c.MenuAction[i] < 1000 && c.MenuAction[i+1] > 1000 {
				var4 := c.MenuOption[i]
				c.MenuOption[i] = c.MenuOption[i+1]
				c.MenuOption[i+1] = var4
				var5 := c.MenuAction[i]
				c.MenuAction[i] = c.MenuAction[i+1]
				c.MenuAction[i+1] = var5
				var7 := c.MenuParamB[i]
				c.MenuParamB[i] = c.MenuParamB[i+1]
				c.MenuParamB[i+1] = var7
				var8 := c.MenuParamC[i]
				c.MenuParamC[i] = c.MenuParamC[i+1]
				c.MenuParamC[i+1] = var8
				var9 := c.MenuParamA[i]
				c.MenuParamA[i] = c.MenuParamA[i+1]
				c.MenuParamA[i+1] = var9
				var2 = false
			}
		}
	}
}

func (c *Client) ClearCaches() {
	loctype.ModelCacheStatic.Clear()
	loctype.ModelCacheDynamic.Clear()
	npctype.ModelCache.Clear()
	objtype.ModelCache.Clear()
	objtype.IconCache.Clear()
	playerentity.ModelCache.Clear()
	spotanimtype.ModelCache.Clear()
}
