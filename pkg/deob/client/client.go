package client

import (
	"math/big"
	"strconv"

	"goscape-client/pkg/jagex2/config/npctype"
	"goscape-client/pkg/jagex2/config/seqtype"
	"goscape-client/pkg/jagex2/dash3d/entity"
	"goscape-client/pkg/jagex2/dash3d/world3d"
	"goscape-client/pkg/jagex2/datastruct"
	"goscape-client/pkg/jagex2/graphics/pix2d"
	"goscape-client/pkg/jagex2/graphics/pix32"
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
	HintTileZ             int
	HintHeight            int
	HintOffsetX           int
	HintOffsetZ           int
	MinimapOffsetCycle    int
	RedrawBackground      bool
	LocList               *datastruct.LinkList[*entity.LocEntity]
	RandomIn              *io.Isaac
	CameraModifierEnabled []bool
	PrivateChatSetting    int
	SelectedTab           int
	BFSCost               [][]int
	SocialAction          int
	SceneBaseTileX        int
	SceneBaseTileZ        int
	MapLastBaseX          int
	MapLastBaseZ          int
	SocialInput           string
	MergedLocations       *datastruct.LinkList[*entity.LocMergeEntity]
	IgnoreName37          []int64
	WeightCarried         int
	SceneMapLandData      [][]byte
	Out                   *io.Packet
	StartMidiThread       bool
	ChatEffects           int
	HintNPC               int
	OverrideChat          int
	SkillLevel            []int
	//ChatInterface // TODO
	WaveLoops              []int
	MouseButtonsOption     int
	LocalPID               int
	DesignColors           []int
	Login                  *io.Packet
	FriendWorld            []int
	MinimapLevel           int
	SocialMessage          int
	ImageHitmarks          []*pix32.Pix32
	ChatbackInput          string
	LastWaveID             int
	UpdateDesignModel      bool
	DesignIdentikits       []int
	ActiveMapFunctions     []*pix32.Pix32
	ChatScrollHeight       int
	In                     *io.Packet
	ArchiveChecksum        []int
	MidiThreadActive       bool
	ImageSideIcons         []*pix8.Pix8
	OrbitCameraPitch       int
	MAX_PLAYER_COUNT       int
	LOCAL_PLAYER_INDEX     int
	Players                []*entity.PlayerEntity
	PlayerIDs              []int
	EntityUpdateIDs        []int
	PlayerAppearanceBuffer []*io.Packet
	Projectiles            *datastruct.LinkList[*entity.ProjectileEntity]
	MenuOption             []string
	MidiActive             bool
	DesignGenderMale       bool
	FlameLineOffset        []int
	CompassMaskLineOffsets []int
	WaveDelay              []int
	TabInterfaceID         []int
	ErrorLoading           bool
	ShowSocialInput        bool
	PressedContinueOption  bool
	MessageIDs             []int
	MenuVisible            bool
	ReportAbuseMuteOption  bool
	SpawnedLocations       *datastruct.LinkList[*entity.LocAddEntity]
	MessageType            []int
	MessageSender          []string
	MessageText            []string
	FlameActive            bool
	ReportAbuseInterfaceID int
	ActiveMapFunctionX     []int
	ActiveMapFunctionZ     []int
	TileLastOccupiedCycle  [][]int
	RedrawPrivacySettings  bool
	ErrorHost              bool
	SkillBaseLevel         []int
	NPCs                   []*entity.NpcEntity
	NPCIDs                 []int
	MinimapZoomModifier    int
	Varps                  []int
	EntityRemovalIDs       []int
	FriendName37           []int64
	MinimapMaskLineLengths []int
	//LevelCollisionMap []CollisionMap // TODO
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
	ObjDragArena                  int
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
	LocalPlayer                   *entity.PlayerEntity
	GenderButtonImage0            pix32.Pix32
	GenderButtonImage1            pix32.Pix32
	ImageFlamesLeft               pix32.Pix32
	ImageFlamesRight              pix32.Pix32
	ImageMapflag                  pix32.Pix32
	ImageMinimap                  pix32.Pix32
	ImageMapdot0                  pix32.Pix32
	ImageMapdot1                  pix32.Pix32
	ImageMapdot2                  pix32.Pix32
	ImageMapdot3                  pix32.Pix32
	ImageCompass                  pix32.Pix32
	ImageRedstone1                pix8.Pix8
	ImageRedstone2                pix8.Pix8
	ImageRedstone3                pix8.Pix8
	ImageRedstone1h               pix8.Pix8
	ImageRedstone2h               pix8.Pix8
	ImageBackbase1                pix8.Pix8
	ImageBackbase2                pix8.Pix8
	ImageBackhmid1                pix8.Pix8
	ImageInvback                  pix8.Pix8
	ImageMapback                  pix8.Pix8
	ImageChatback                 pix8.Pix8
	ImageRedstone1v               pix8.Pix8
	ImageRedstone2v               pix8.Pix8
	ImageRedstone3v               pix8.Pix8
	ImageRedstone1hv              pix8.Pix8
	ImageRedstone2hv              pix8.Pix8
	ImageScrollbar0               pix8.Pix8
	ImageScrollbar1               pix8.Pix8
	ImageTitlebox                 pix8.Pix8
	ImageTitleButton              pix8.Pix8
	FontPlain11                   pixfont.PixFont
	FontPlain12                   pixfont.PixFont
	FontBold12                    pixfont.PixFont
	FontQuill8                    pixfont.PixFont
	AreaBackbase1                 pixmap.PixMap
	AreaBackbase2                 pixmap.PixMap
	AreaBackhmid1                 pixmap.PixMap
	AreaBackleft1                 pixmap.PixMap
	AreaBackleft2                 pixmap.PixMap
	AreaBackright1                pixmap.PixMap
	AreaBackright2                pixmap.PixMap
	AreaBacktop1                  pixmap.PixMap
	AreaBacktop2                  pixmap.PixMap
	AreaBackvmid1                 pixmap.PixMap
	AreaBackvmid2                 pixmap.PixMap
	AreaBackvmid3                 pixmap.PixMap
	AreaBackhmid2                 pixmap.PixMap
	ImageTitle2                   pixmap.PixMap
	ImageTitle3                   pixmap.PixMap
	ImageTitle4                   pixmap.PixMap
	ImageTitle0                   pixmap.PixMap
	ImageTitle1                   pixmap.PixMap
	ImageTitle5                   pixmap.PixMap
	ImageTitle6                   pixmap.PixMap
	ImageTitle7                   pixmap.PixMap
	ImageTitle8                   pixmap.PixMap
	AreaSidebar                   pixmap.PixMap
	AreaMapback                   pixmap.PixMap
	AreaViewport                  pixmap.PixMap
	AreaChatback                  pixmap.PixMap
	ArchiveTitle                  io.Jagfile
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
	ImageRunes          []pix8.Pix8
	SceneMapLocData     [][]byte
	LevelTileFlags      [][][]byte
	LevelHeightmap      [][][]int
}

func NewClient() *Client {
	c := &Client{
		LocList:                datastruct.NewLinkList[*entity.LocEntity](),
		CameraModifierEnabled:  make([]bool, 5),
		MergedLocations:        datastruct.NewLinkList[*entity.LocMergeEntity](),
		IgnoreName37:           make([]int64, 100),
		Out:                    io.Alloc(1),
		SkillLevel:             make([]int, 50),
		WaveLoops:              make([]int, 50),
		LocalPID:               -1,
		DesignColors:           make([]int, 5),
		Login:                  io.Alloc(1),
		FriendWorld:            make([]int, 100),
		MinimapLevel:           -1,
		ImageHitmarks:          make([]*pix32.Pix32, 20),
		LastWaveID:             -1,
		DesignIdentikits:       make([]int, 7),
		ActiveMapFunctions:     make([]*pix32.Pix32, 1000),
		ChatScrollHeight:       78,
		In:                     io.Alloc(1),
		ArchiveChecksum:        make([]int, 9),
		MidiThreadActive:       true,
		ImageSideIcons:         make([]*pix8.Pix8, 13),
		OrbitCameraPitch:       128,
		MAX_PLAYER_COUNT:       2048,
		LOCAL_PLAYER_INDEX:     2047,
		Projectiles:            datastruct.NewLinkList[*entity.ProjectileEntity](),
		MenuOption:             make([]string, 500),
		MidiActive:             true,
		DesignGenderMale:       true,
		FlameLineOffset:        make([]int, 256),
		CompassMaskLineOffsets: make([]int, 33),
		WaveDelay:              make([]int, 50),
		TabInterfaceID:         []int{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		MessageIDs:             make([]int, 100),
		SpawnedLocations:       datastruct.NewLinkList[*entity.LocAddEntity](),
		MessageType:            make([]int, 100),
		MessageSender:          make([]string, 100),
		MessageText:            make([]string, 100),
		ReportAbuseInterfaceID: -1,
		ActiveMapFunctionX:     make([]int, 1000),
		ActiveMapFunctionZ:     make([]int, 1000),
		SkillBaseLevel:         make([]int, 50),
		NPCs:                   make([]*entity.NpcEntity, 8192),
		NPCIDs:                 make([]int, 8192),
		MinimapZoomModifier:    1,
		Varps:                  make([]int, 2000),
		EntityRemovalIDs:       make([]int, 1000),
		FriendName37:           make([]int64, 100),
		MinimapMaskLineLengths: make([]int, 151),
		// TODO: LevelCollisionMap
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
	c.Players = make([]*entity.PlayerEntity, c.MAX_PLAYER_COUNT)
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
			var5 := var3.(*entity.PlayerEntity) // mine - moved here from below
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
							//var32 // TODO: UNLINK!
						}
					}
				}
			}
		}
	}
}
