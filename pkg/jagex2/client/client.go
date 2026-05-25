package client

import (
	"bytes"
	"fmt"
	"hash/crc32"
	io2 "io"
	"log"
	"math"
	"math/big"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zsrv/goscape-client/pkg/jagex2/client/clientextras"
	"github.com/zsrv/goscape-client/pkg/jagex2/client/inputtracking"
	"github.com/zsrv/goscape-client/pkg/jagex2/config/component"
	"github.com/zsrv/goscape-client/pkg/jagex2/config/flotype"
	"github.com/zsrv/goscape-client/pkg/jagex2/config/idktype"
	"github.com/zsrv/goscape-client/pkg/jagex2/config/loctype"
	"github.com/zsrv/goscape-client/pkg/jagex2/config/npctype"
	"github.com/zsrv/goscape-client/pkg/jagex2/config/objtype"
	"github.com/zsrv/goscape-client/pkg/jagex2/config/seqtype"
	"github.com/zsrv/goscape-client/pkg/jagex2/config/spotanimtype"
	"github.com/zsrv/goscape-client/pkg/jagex2/config/varptype"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/entity"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/entity/playerentity"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/world"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/world3d"
	"github.com/zsrv/goscape-client/pkg/jagex2/datastruct"
	"github.com/zsrv/goscape-client/pkg/jagex2/datastruct/jstring"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/animbase"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/animframe"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/errorfont"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/model"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pix2d"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pix32"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pix3d"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pix8"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pixfont"
	"github.com/zsrv/goscape-client/pkg/jagex2/graphics/pixmap"
	"github.com/zsrv/goscape-client/pkg/jagex2/io"
	"github.com/zsrv/goscape-client/pkg/jagex2/io/bzip2"
	"github.com/zsrv/goscape-client/pkg/jagex2/io/clientstream"
	"github.com/zsrv/goscape-client/pkg/jagex2/sound/audio"
	"github.com/zsrv/goscape-client/pkg/jagex2/sound/wave"
	"github.com/zsrv/goscape-client/pkg/jagex2/wordenc/wordfilter"
	"github.com/zsrv/goscape-client/pkg/jagex2/wordenc/wordpack"
	"github.com/zsrv/goscape-client/pkg/sign/signlink"
)

func RecoverPanic() {
	if err := recover(); err != nil {
		log.Printf("client: recovered from panic: %v", err)
	}
}

var (
	CycleLogic2     int
	OpLogic3        int
	CHARSET         string = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!\"£$%^&*()-_=+[{]};:'@#~,<.>/?\\| "
	LevelExperience []int  = make([]int, 99)
	NodeID          int    = 10
	MembersWorld    bool   = true
	RSA_EXPONENT    *big.Int
	RSA_MODULUS     *big.Int
	OpLogic5        int
	OpLogic1        int
	OpLogic4        int
	OpLogic6        int
	OpLogic2        int
	OpLogic9        int
	CycleLogic1     int
	OpLogic8        int
	CycleLogic6     int
	OpLogic7        int
	CycleLogic3     int
	CycleLogic4     int
	CycleLogic5     int
	LowMemory       bool
	AlreadyStarted  bool
)

func init() {
	modulus, ok := new(big.Int).SetString("7162900525229798032761816791230527296329313291232324290237849263501208207972894053929065636522363163621000728841182238772712427862772219676577293600221789", 10)
	if !ok {
		panic("bad rsa modulus")
	}
	RSA_MODULUS = modulus

	exponent, ok := new(big.Int).SetString("58778699976184461502525193738213253649000149147835990136706041084440742975821", 10)
	if !ok {
		panic("bad rsa exponent")
	}
	RSA_EXPONENT = exponent

	var0 := 0
	for i := range 99 {
		var2 := i + 1
		var3 := int(float64(var2) + math.Pow(2.0, float64(var2)/7.0)*300.0)
		var0 += var3
		LevelExperience[i] = var0 / 4
	}
}

type Client struct {
	//*GameShell
	// BEGIN GameShell
	State   int
	DelTime int
	MinDel  int
	OTim    []int64
	// Java: GameShell.java:38 declares `int fps`, computed every frame at
	// gameshell.java:187 but never read anywhere. Pure deob residue;
	// field omitted and the assignment dropped per the deob-artifact
	// exclusion policy.
	ScreenWidth  int
	ScreenHeight int
	//Graphics
	// Java: GameShell.java:50 declares `PixMap drawArea` (the AWT
	// backbuffer) which Java code allocates and nils but never reads.
	// The Go port blits via platform.Active (uploaded via OverlayPixMap /
	// PixMap.Draw), so drawArea is pure deob residue here; field omitted
	// per the deob-artifact exclusion policy. Three nil-assignment sites
	// in client.go and one allocation in the old gameshell.go boot path
	// were dropped alongside.
	// Java: GameShell.java:53 declares `Pix32[] temp = new Pix32[6]`, a
	// dead deob array never read. Intentionally not ported per the
	// deob-artifact exclusion policy.
	OverlayPixMap    *pixmap.PixMap
	Frame            *ViewBox
	Refresh          bool
	IdleCycles       int
	MouseButton      int
	MouseX           int
	MouseY           int
	MouseClickButton int
	MouseClickX      int
	MouseClickY      int
	ActionKey        []int
	KeyQueue         []int
	KeyQueueReadPos  int
	KeyQueueWritePos int

	// flameMu guards concurrent access to ImageTitle0/1 pixel buffers between
	// the RunFlames goroutine (writer) and the render loop (reader, via
	// DrawTitleScreen, DrawGame, DrawProgress). Replaces the former global
	// pixmap.OpsMu for this narrow writer↔reader hand-off.
	flameMu sync.Mutex
	// END GameShell

	HintTileZ                     int
	HintHeight                    int
	HintOffsetX                   int
	HintOffsetZ                   int
	MinimapOffsetCycle            int
	RedrawFrame                   bool // Java: redrawBackground (deob/client.java:74)
	LocList                       *datastruct.LinkList[*entity.LocEntity]
	RandomIn                      *io.Isaac
	CameraModifierEnabled         []bool
	PrivateChatSetting            int
	SelectedTab                   int
	BFSCost                       [][]int
	SocialAction                  int
	SceneBaseTileX                int
	SceneBaseTileZ                int
	MapLastBaseX                  int
	MapLastBaseZ                  int
	SocialInput                   string
	MergedLocations               *datastruct.LinkList[*entity.LocMergeEntity]
	IgnoreName37                  []int64
	WeightCarried                 int
	SceneMapLandData              [][]byte
	Out                           *io.Packet
	StartMidiThread               bool
	ChatEffects                   int
	HintNPC                       int
	OverrideChat                  int
	SkillLevel                    []int
	ChatInterface                 *component.Component
	WaveLoops                     []int
	MouseButtonsOption            int
	LocalPID                      int
	DesignColors                  []int
	Login                         *io.Packet
	FriendWorld                   []int
	MinimapLevel                  int
	SocialMessage                 string
	ImageHitmarks                 []*pix32.Pix32
	ChatbackInput                 string
	LastWaveID                    int
	UpdateDesignModel             bool
	DesignIdentikits              []int
	ActiveMapFunctions            []*pix32.Pix32
	ChatScrollHeight              int
	In                            *io.Packet
	JagChecksum                   []int
	MidiThreadActive              bool
	ImageSideIcons                []*pix8.Pix8
	OrbitCameraPitch              int
	MAX_PLAYER_COUNT              int
	LOCAL_PLAYER_INDEX            int
	Players                       []*playerentity.PlayerEntity
	PlayerIDs                     []int
	EntityUpdateIDs               []int
	PlayerAppearanceBuffer        []*io.Packet
	Projectiles                   *datastruct.LinkList[*entity.ProjectileEntity]
	MenuOption                    []string
	MidiActive                    bool
	DesignGenderMale              bool
	FlameLineOffset               []int
	CompassMaskLineOffsets        []int
	WaveDelay                     []int
	TabInterfaceID                []int
	ErrorLoading                  bool
	ShowSocialInput               bool
	PressedContinueOption         bool
	MessageIDs                    []int
	MenuVisible                   bool
	ReportAbuseMuteOption         bool
	SpawnedLocations              *datastruct.LinkList[*entity.LocAddEntity]
	MessageType                   []int
	MessageSender                 []string
	MessageText                   []string
	FlameActive                   bool
	ReportAbuseInterfaceID        int
	ActiveMapFunctionX            []int
	ActiveMapFunctionZ            []int
	TileLastOccupiedCycle         [][]int
	RedrawPrivacySettings         bool
	ErrorHost                     bool
	SkillBaseLevel                []int
	NPCs                          []*entity.NpcEntity
	NPCIDs                        []int
	MinimapZoomModifier           int
	Varps                         []int
	EntityRemovalIDs              []int
	FriendName37                  []int64
	MinimapMaskLineLengths        []int
	LevelCollisionMap             []*dash3d.CollisionMap
	ImageHeadIcons                []*pix32.Pix32
	CameraModifierJitter          []int
	ObjGrabThreshold              bool
	RedrawSidebar                 bool
	RedrawChatback                bool
	CameraModifierWobbleScale     []int
	Cutscene                      bool
	ReportAbuseInput              string
	ViewportInterfaceID           int
	InGame                        bool
	FlamesThread                  bool
	SCROLLBAR_GRIP_LOWLIGHT       int
	SCROLLBAR_GRIP_HIGHLIGHT      int
	BFSStepX                      []int
	BFSStepZ                      []int
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
	MidiSyncMu                    sync.Mutex // Java: midiSync (deob/client.java:491) — dedicated lock for the MidiSyncName/CRC/Len handoff between SetMidi and RunMidi
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
	MessageIds                    []int
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
	ImageTitleBox                 *pix8.Pix8
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
	JagTitle                      *io.Jagfile
	Stream                        *clientstream.ClientStream
	ModalMessage                  string
	ObjSelectedName               string
	SpellCaption                  string
	MidiSyncName                  string
	CurrentMidi                   string
	AreaChatbackOffsets           []int
	AreaSidebarOffsets            []int
	AreaViewportOffsets           []int
	FlameBuffer0                  []int
	FlameBuffer1                  []int
	FlameGradient                 []int
	FlameGradient0                []int
	FlameGradient1                []int
	FlameGradient2                []int
	SceneMapIndex                 []int
	FlameBuffer3                  []int
	FlameBuffer2                  []int
	ImageRunes                    []*pix8.Pix8
	SceneMapLocData               [][]byte
	LevelTileFlags                [][][]byte
	LevelHeightMap                [][][]int
}

func NewClient() *Client {
	c := &Client{
		//GameShell:                 NewGameShell(),
		// BEGIN GameShell
		DelTime:   20,
		MinDel:    1,
		OTim:      make([]int64, 10),
		Refresh:   true,
		ActionKey: make([]int, 128),
		KeyQueue:  make([]int, 128),
		// END GameShell

		LocList:               datastruct.NewLinkList[*entity.LocEntity](),
		CameraModifierEnabled: make([]bool, 5),
		MergedLocations:       datastruct.NewLinkList[*entity.LocMergeEntity](),
		IgnoreName37:          make([]int64, 100),
		MessageIds:            make([]int, 100),
		Out:                   io.Alloc(1),
		SkillLevel:            make([]int, 50),
		ChatInterface:         component.NewComponent(),
		WaveLoops:             make([]int, 50),
		LocalPID:              -1,
		DesignColors:          make([]int, 5),
		Login:                 io.Alloc(1),
		FriendWorld:           make([]int, 100),
		MinimapLevel:          -1,
		ImageHitmarks:         make([]*pix32.Pix32, 20),
		LastWaveID:            -1,
		DesignIdentikits:      make([]int, 7),
		ActiveMapFunctions:    make([]*pix32.Pix32, 1000),
		ChatScrollHeight:      78,
		In:                    io.Alloc(1),
		JagChecksum:           make([]int, 9),
		MidiThreadActive:      true,
		ImageSideIcons:        make([]*pix8.Pix8, 13),
		OrbitCameraPitch:      128,
		// Java: deob/client.java:92 — `public int selectedTab = 3;`
		// Latent in current flows (Login resets to 3 before InGame goes
		// true) but the field-init keeps the Go state aligned with Java
		// for any future caller that reads SelectedTab pre-login.
		SelectedTab:               3,
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
		ChatInterfaceID:           -1,
		ProjectX:                  -1,
		ProjectY:                  -1,
		StickyChatInterfaceID:     -1,
		CameraModifierCycle:       make([]int, 5),
		ImageMapscene:             make([]*pix8.Pix8, 50),
		CHAT_COLORS:               []int{0xFFFF00, 0xFF0000, 0xFF00, 0xFFFF, 0xFF00FF, 0xFFFFFF},
		SCROLLBAR_TRACK:           2301979,
		Spotanims:                 datastruct.NewLinkList[*entity.SpotAnimEntity](),
		LastWaveLoops:             -1,
		TextureBuffer:             make([]byte, 16384),
		VarCache:                  make([]int, 2000),
		SkillExperience:           make([]int, 50),
		MinimapAngleModifier:      2,
		MAX_CHATS:                 50,
		LOC_SHAPE_TO_LAYER:        []int{0, 0, 0, 0, 1, 1, 1, 1, 1, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 3},
		CompassMaskLineLengths:    make([]int, 33),
		ImageCrosses:              make([]*pix32.Pix32, 8),
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
	c.MidiSyncMu.Lock()
	defer c.MidiSyncMu.Unlock()
	c.MidiSyncName = name
	c.MidiSyncCRC = crc
	c.MidiSyncLen = length
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
			pe := var3.Pathing()
			if i < c.PlayerCount {
				var5 := var3.(*playerentity.PlayerEntity)
				var4 = 30
				if var5.HeadIcons != 0 {
					c.ProjectFromGround1(pe.Height+15, pe)
					if c.ProjectX > -1 {
						for j := range 8 {
							if var5.HeadIcons&(0x1<<j) != 0 {
								c.ImageHeadIcons[j].PlotSprite(c.ProjectY-var4, c.ProjectX-12)
								var4 -= 25
							}
						}
					}
				}
				if i >= 0 && c.HintType == 10 && c.HintPlayer == c.PlayerIDs[i] {
					c.ProjectFromGround1(pe.Height+15, pe)
					if c.ProjectX > -1 {
						c.ImageHeadIcons[7].PlotSprite(c.ProjectY-var4, c.ProjectX-12)
					}
				}
			} else if c.HintType == 1 && c.HintNPC == c.NPCIDs[i-c.PlayerCount] && clientextras.LoopCycle%20 < 10 {
				c.ProjectFromGround1(pe.Height+15, pe)
				if c.ProjectX > -1 {
					c.ImageHeadIcons[2].PlotSprite(c.ProjectY-28, c.ProjectX-12)
				}
			}
			if pe.Chat != "" && (i >= c.PlayerCount || c.PublicChatSetting == 0 || c.PublicChatSetting == 3 || c.PublicChatSetting == 1 && c.IsFriend(var3.(*playerentity.PlayerEntity).Name)) {
				c.ProjectFromGround1(pe.Height, pe)
				if c.ProjectX > -1 && c.ChatCount < c.MAX_CHATS {
					c.ChatWidth[c.ChatCount] = c.FontBold12.StringWidth(pe.Chat) / 2
					c.ChatHeight[c.ChatCount] = c.FontBold12.Height
					c.ChatX[c.ChatCount] = c.ProjectX
					c.ChatY[c.ChatCount] = c.ProjectY
					c.ChatColors[c.ChatCount] = pe.ChatColor
					c.ChatStyles[c.ChatCount] = pe.ChatStyle
					c.ChatTimers[c.ChatCount] = pe.ChatTimer
					c.Chats[c.ChatCount] = pe.Chat
					c.ChatCount++
					if c.ChatEffects == 0 && pe.ChatStyle == 1 {
						c.ChatHeight[c.ChatCount] += 10
						c.ChatY[c.ChatCount] += 5
					}
					if c.ChatEffects == 0 && pe.ChatStyle == 2 {
						c.ChatWidth[c.ChatCount] = 60
					}
				}
			}
			if pe.CombatCycle > clientextras.LoopCycle+100 {
				c.ProjectFromGround1(pe.Height+15, pe)
				if c.ProjectX > -1 {
					var4 = pe.Health * 30 / pe.TotalHealth
					var4 = min(var4, 30)
					pix2d.FillRect(c.ProjectY-3, c.ProjectX-15, 0xFF00, var4, 5)
					pix2d.FillRect(c.ProjectY-3, c.ProjectX-15+var4, 0xFF0000, 30-var4, 5)
				}
			}
			if pe.CombatCycle > clientextras.LoopCycle+330 {
				c.ProjectFromGround1(pe.Height/2, pe)
				if c.ProjectX > -1 {
					c.ImageHitmarks[pe.DamageType].PlotSprite(c.ProjectY-12, c.ProjectX-12)
					c.FontPlain11.CentreString(c.ProjectY+4, 0, strconv.Itoa(pe.Damage), c.ProjectX)
					c.FontPlain11.CentreString(c.ProjectY+3, 0xFFFFFF, strconv.Itoa(pe.Damage), c.ProjectX-1)
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
			var10 := 0xFFFF00
			if c.ChatColors[i] < 6 {
				var10 = c.CHAT_COLORS[c.ChatColors[i]]
			}
			if c.ChatColors[i] == 6 {
				if c.SceneCycle%20 < 10 {
					var10 = 0xFF0000
				} else {
					var10 = 0xFFFF00
				}
			}
			if c.ChatColors[i] == 7 {
				if c.SceneCycle%20 < 10 {
					var10 = 0xFF
				} else {
					var10 = 0xFFFF
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
					var10 = var11*1280 + 0xFF0000
				} else if var11 < 100 {
					var10 = 0xFFFF00 - (var11-50)*327680
				} else if var11 < 150 {
					var10 = (var11-100)*5 + 0xFF00
				}
			}
			if c.ChatColors[i] == 10 {
				var11 = 150 - c.ChatTimers[i]
				if var11 < 50 {
					var10 = var11*5 + 0xFF0000
				} else if var11 < 100 {
					var10 = 0xFF00FF - (var11-50)*327680
				} else if var11 < 150 {
					var10 = (var11-100)*327680 + 0xFF - (var11-100)*5
				}
			}
			if c.ChatColors[i] == 11 {
				var11 = 150 - c.ChatTimers[i]
				if var11 < 50 {
					var10 = 0xFFFFFF - var11*327685
				} else if var11 < 100 {
					var10 = (var11-50)*327685 + 0xFF00
				} else if var11 < 150 {
					var10 = 0xFFFFFF - (var11-100)*327680
				}
			}
			if c.ChatStyles[i] == 0 {
				c.FontBold12.CentreString(c.ProjectY+1, 0, var15, c.ProjectX)
				c.FontBold12.CentreString(c.ProjectY, var10, var15, c.ProjectX)
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
			c.FontBold12.CentreString(c.ProjectY+1, 0, var15, c.ProjectX)
			c.FontBold12.CentreString(c.ProjectY, 0xFFFF00, var15, c.ProjectX)
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
	signlink.SetMidiFade(0)
	signlink.SetMidiCommand("stop")
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
				var2.DrawString(4, var6-1, 0xFFFF, "From "+c.MessageSender[i]+": "+c.MessageText[i])
				var3++
				if var3 >= 5 {
					return
				}
			}
			if var5 == 5 && c.PrivateChatSetting < 2 {
				var6 = 329 - var3*13
				var2.DrawString(4, var6, 0, c.MessageText[i])
				var2.DrawString(4, var6-1, 0xFFFF, c.MessageText[i])
				var3++
				if var3 >= 5 {
					return
				}
			}
			if var5 == 6 && c.PrivateChatSetting < 2 {
				var6 = 329 - var3*13
				var2.DrawString(4, var6, 0, "To "+c.MessageSender[i]+": "+c.MessageText[i])
				var2.DrawString(4, var6-1, 0xFFFF, "To "+c.MessageSender[i]+": "+c.MessageText[i])
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
			if var8 == 0xFFFF {
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
			if var6.TargetID == 0xFFFF {
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
			var6.CombatCycle = clientextras.LoopCycle + 400
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
			var6.SpotanimLastCycle = clientextras.LoopCycle + (var8 & 0xFFFF)
			var6.SpotanimFrame = 0
			var6.SpotanimCycle = 0
			if var6.SpotanimLastCycle > clientextras.LoopCycle {
				var6.SpotanimFrame = -1
			}
			if var6.SpotanimID == 0xFFFF {
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
	var4 := jstring.FormatName(jstring.FromBase37(arg0))
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
		var5 = c.BaseX + ((var4 >> 4) & 0x7)
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
			for var13 := c.SpawnedLocations.Head(); var13 != nil; var13 = c.SpawnedLocations.Next() {
				v := var13.Value
				if v.Plane == c.CurrentLevel && v.X == var5 && v.Z == var6 && v.Layer == var10 {
					var12 = v
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
					var15 = (var14 >> 14) & 0x7FFF
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
				c.SpawnedLocations.AddTail(datastruct.NewLinkable(var12))
			}
			var12.LocIndex = var11
			var12.Shape = var8
			var12.Angle = var9
			c.AddLoc(var9, var5, var6, var10, var11, var8, c.CurrentLevel)
		}
	} else if arg2 == 42 {
		var4 = arg1.G1()
		var5 = c.BaseX + ((var4 >> 4) & 0x7)
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
				var38 := entity.NewLocEntity(false, (var11>>14)&0x7FFF, c.CurrentLevel, var9, seqtype.Instances[var10], var6, var5)
				c.LocList.AddTail(datastruct.NewLinkable(var38))
			}
		}
	} else {
		if arg2 == 223 {
			var4 = arg1.G1()
			var5 = c.BaseX + ((var4 >> 4) & 0x7)
			var6 = c.BaseZ + (var4 & 0x7)
			var7 = arg1.G2()
			var8 = arg1.G2()
			if var5 >= 0 && var6 >= 0 && var5 < 104 && var6 < 104 {
				var32 := entity.NewObjStackEntity()
				var32.Index = var7
				var32.Count = var8
				if c.LevelObjStacks[c.CurrentLevel][var5][var6] == nil {
					c.LevelObjStacks[c.CurrentLevel][var5][var6] = datastruct.NewLinkList[*entity.ObjStackEntity]()
				}
				c.LevelObjStacks[c.CurrentLevel][var5][var6].AddTail(datastruct.NewLinkable(var32))
				c.SortObjStacks(var5, var6)
			}
		} else if arg2 == 49 {
			var4 = arg1.G1()
			var5 = c.BaseX + ((var4 >> 4) & 0x7)
			var6 = c.BaseZ + (var4 & 0x7)
			var7 = arg1.G2()
			if var5 >= 0 && var6 >= 0 && var5 < 104 && var6 < 104 {
				var30 := c.LevelObjStacks[c.CurrentLevel][var5][var6]
				if var30 != nil {
					for var32 := var30.Head(); var32 != nil; var32 = var30.Next() {
						v := var32.Value
						if v.Index == var7&0x7FFF {
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
				var5 = c.BaseX + ((var4 >> 4) & 0x7)
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
					var43 := entity.NewProjectileEntity(var36, var15, var6, var14+clientextras.LoopCycle, c.CurrentLevel, var9, var37+clientextras.LoopCycle, var16, c.GetHeightMapY(c.CurrentLevel, var5, var6)-var11, var10, var5)
					var43.UpdateVelocity(c.GetHeightMapY(c.CurrentLevel, var7, var8)-var36, var8, var7, var37+clientextras.LoopCycle)
					c.Projectiles.AddTail(datastruct.NewLinkable(var43))
				}
			} else if arg2 == 191 {
				var4 = arg1.G1()
				var5 = c.BaseX + ((var4 >> 4) & 0x7)
				var6 = c.BaseZ + (var4 & 0x7)
				var7 = arg1.G2()
				var8 = arg1.G1()
				var9 = arg1.G2()
				if var5 >= 0 && var6 >= 0 && var5 < 104 && var6 < 104 {
					var5 = var5*128 + 64
					var6 = var6*128 + 64
					var34 := entity.NewSpotAnimEntity(var5, var7, var6, var9, c.GetHeightMapY(c.CurrentLevel, var5, var6)-var8, c.CurrentLevel, clientextras.LoopCycle)
					c.Spotanims.AddTail(datastruct.NewLinkable(var34))
				}
			} else if arg2 == 50 {
				var4 = arg1.G1()
				var5 = c.BaseX + ((var4 >> 4) & 0x7)
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
					c.LevelObjStacks[c.CurrentLevel][var5][var6].AddTail(datastruct.NewLinkable(var33))
					c.SortObjStacks(var5, var6)
				}
			} else {
				if arg2 == 23 {
					var4 = arg1.G1()
					var5 = c.BaseX + ((var4 >> 4) & 0x7)
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
						var20 := entity.NewLocMergeEntity(c.CurrentLevel, var9, var6, var36+clientextras.LoopCycle, var8, -1, var5, var10)
						c.MergedLocations.AddTail(datastruct.NewLinkable(var20))
						var21 := entity.NewLocMergeEntity(c.CurrentLevel, var9, var6, var37+clientextras.LoopCycle, var8, var11, var5, var10)
						c.MergedLocations.AddTail(datastruct.NewLinkable(var21))
						var22 := c.LevelHeightMap[c.CurrentLevel][var5][var6]
						var23 := c.LevelHeightMap[c.CurrentLevel][var5+1][var6]
						var24 := c.LevelHeightMap[c.CurrentLevel][var5+1][var6+1]
						var25 := c.LevelHeightMap[c.CurrentLevel][var5][var6+1]
						var26 := loctype.Get(var11)
						var19.LocStartCycle = var36 + clientextras.LoopCycle
						var19.LocStopCycle = var37 + clientextras.LoopCycle
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
						var29 := int8(0)
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
					var5 = c.BaseX + ((var4 >> 4) & 0x7)
					var6 = c.BaseZ + (var4 & 0x7)
					var7 = arg1.G2()
					var8 = arg1.G2()
					var9 = arg1.G2()
					if var5 >= 0 && var6 >= 0 && var5 < 104 && var6 < 104 {
						var31 := c.LevelObjStacks[c.CurrentLevel][var5][var6]
						if var31 != nil {
							for var35 := var31.Head(); var35 != nil; var35 = var31.Next() {
								v := var35.Value
								if v.Index == var7&0x7FFF && v.Count == var8 {
									v.Count = var9
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
	// Java: deob/client.java:2014 — `var2 - this.cameraY >= 800` (height
	// of ground above the camera position). The prior Go port substituted
	// CameraYaw (rotation 0..2047), making the threshold meaningless.
	if var2-c.CameraY >= 800 || c.LevelTileFlags[c.CurrentLevel][c.CameraX>>7][c.CameraZ>>7]&0x4 == 0 {
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
		var2 = max(var2, c.CameraPitchClamp/256)
		if c.CameraModifierEnabled[4] && c.CameraModifierWobbleScale[4]+128 > var2 {
			var2 = c.CameraModifierWobbleScale[4] + 128
		}
		var3 = (c.OrbitCameraYaw + c.CameraAnticheatAngle) & 0x7FF
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
				c.CameraYaw = (c.CameraYaw + var9) & 0x7FF
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
	c.AreaViewport.Draw(8, 11)
	c.CameraX = var3
	c.CameraY = var4
	c.CameraZ = var5
	c.CameraPitch = var6
	c.CameraYaw = var7
}

func (c *Client) RunMidi() {
	c.StartMidiThread = false
	for c.MidiThreadActive {
		time.Sleep(50 * time.Millisecond)
		var2 := ""
		var3 := 0
		var4 := 0
		c.MidiSyncMu.Lock()
		var2 = c.MidiSyncName
		var3 = c.MidiSyncCRC
		var4 = c.MidiSyncLen
		c.MidiSyncName = ""
		c.MidiSyncCRC = 0
		c.MidiSyncLen = 0
		c.MidiSyncMu.Unlock()
		if var2 != "" {
			var14 := signlink.CacheLoad(var2 + ".mid")
			var6 := 0
			if var14 != nil && var3 != 12345678 {
				var6 = int(crc32.ChecksumIEEE(var14))
				if var6 != var3 {
					var14 = nil
				}
			}
			if var14 == nil {
				var15, err := c.OpenURL(var2 + "_" + strconv.Itoa(var3) + ".mid")
				if err != nil {
					log.Printf("client: RunMidi error: %v", err)
					return
				}
				var14 = make([]byte, var4)
				var8 := 0
				for i := 0; i < var4; i += var8 {
					var8, err = var15.Read(var14[i:var4])
					if err != nil {
						var14 = var14[:i]
						var4 = i
						break
					}
				}
				signlink.CacheSave(var2+".mid", var14)
			}
			if var14 == nil {
				return
			}
			var6 = io.NewPacket(var14).G4()
			var16 := make([]byte, var6)
			bzip2.Read(var16, var6, var14, var4, 4)
			c.SaveMidi(var16, var6, true)
		}
	}
}

func SetLowMem() {
	world3d.LowMemory = true
	pix3d.LowDetail = true
	LowMemory = true
	world.LowMemory = true
}

func (c *Client) DrawFlames() {
	// DrawFlames runs from the RunFlames goroutine, independent of
	// c.Draw. It updates the ImageTitle0 / ImageTitle1 pixel buffers
	// with the next animation step. The GPU upload of those buffers
	// happens in DrawTitleScreen / DrawGame / DrawProgress each frame.
	//
	// Hold flameMu while writing the pixel buffers so the render loop
	// readers (DrawTitleScreen, DrawGame, DrawProgress) don't race with
	// these writes. The lock is tight-scoped to just this function;
	// each reader wraps only the consecutive ImageTitle0/1 .Draw calls.
	c.flameMu.Lock()
	defer c.flameMu.Unlock()

	var2 := 256
	if c.FlameGradientCycle0 > 0 {
		for i := range 256 {
			if c.FlameGradientCycle0 > 768 {
				c.FlameGradient[i] = c.Mix(c.FlameGradient0[i], 0x400-c.FlameGradientCycle0, c.FlameGradient1[i])
			} else if c.FlameGradientCycle0 > 256 {
				c.FlameGradient[i] = c.FlameGradient1[i]
			} else {
				c.FlameGradient[i] = c.Mix(c.FlameGradient1[i], 256-c.FlameGradientCycle0, c.FlameGradient0[i])
			}
		}
	} else if c.FlameGradientCycle1 > 0 {
		for i := range 256 {
			if c.FlameGradientCycle1 > 768 {
				c.FlameGradient[i] = c.Mix(c.FlameGradient0[i], 0x400-c.FlameGradientCycle1, c.FlameGradient2[i])
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
	for i := range 33920 {
		c.ImageTitle0.Data[i] = c.ImageFlamesLeft.Pixels[i]
	}
	var4 := 0
	var5 := 1152
	var7 := 0
	var8 := 0
	var10 := 0
	var11 := 0
	var12 := 0
	var13 := 0
	for i := 1; i < var2-1; i++ {
		var7 = c.FlameLineOffset[i] * (var2 - i) / var2
		var8 = var7 + 22
		var8 = max(var8, 0)
		var4 += var8
		for j := var8; j < 128; j++ {
			var10 = c.FlameBuffer3[var4]
			var4++
			if var10 == 0 {
				var5++
			} else {
				var11 = var10
				var12 = 256 - var10
				var10 = c.FlameGradient[var10]
				var13 = c.ImageTitle0.Data[var5]
				c.ImageTitle0.Data[var5] = (((((var10 & 0xFF00FF) * var11) + ((var13 & 0xFF00FF) * var12)) & 0xFF00FF00) + ((((var10 & 0xFF00) * var11) + ((var13 & 0xFF00) * var12)) & 0xFF0000)) >> 8
				var5++
			}
		}
		var5 += var8
	}
	// Right-side flame buffer update (left-side ImageTitle0.Data was
	// updated above). GPU upload of both happens in DrawTitleScreen /
	// DrawGame each frame, not here.
	for i := range 33920 {
		c.ImageTitle1.Data[i] = c.ImageFlamesRight.Pixels[i]
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
				var13 = var12
				var14 := 256 - var12
				var12 = c.FlameGradient[var12]
				var15 := c.ImageTitle1.Data[var5]
				c.ImageTitle1.Data[var5] = (((((var12 & 0xFF00FF) * var13) + ((var15 & 0xFF00FF) * var14)) & 0xFF00FF00) + ((((var12 & 0xFF00) * var13) + ((var15 & 0xFF00) * var14)) & 0xFF0000)) >> 8
				var5++
			}
		}
		var4 += 128 - var10
		var5 += 128 - var10 - var9
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
				if strings.Contains(var22, " ") {
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
								} else if c.SpellSelected != 1 || !var12.Interactable {
									if var12.Interactable {
										for l := 4; l >= 3; l-- {
											if var18.IOp != nil && var18.IOp[l] != "" {
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
											if var18.IOp[l] != "" {
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
				if var3.LocModel == nil || clientextras.LoopCycle < var3.LocStartCycle || clientextras.LoopCycle >= var3.LocStopCycle {
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
	var10 := (c.LevelHeightMap[var7][var5][var6]*(128-var8) + c.LevelHeightMap[var7][var5+1][var6]*var8) >> 7
	var11 := (c.LevelHeightMap[var7][var5][var6+1]*(128-var8) + c.LevelHeightMap[var7][var5+1][var6+1]*var8) >> 7
	return (var10*(128-var9) + var11*var9) >> 7
}

func (c *Client) AddNPCOptions(arg0 *npctype.NpcType, arg2, arg3, arg4 int) {
	if c.MenuSize >= 400 {
		return
	}
	var6 := arg0.Name
	if arg0.VisLevel != 0 {
		var6 = var6 + GetCombatLevelColorTag(c.LocalPlayer.CombatLevel, arg0.VisLevel) + " (level-" + strconv.Itoa(arg0.VisLevel) + ")"
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
		c.MenuOption[c.MenuSize] = "Examine @yel@" + var6
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
					// Java: deob/client.java:2758 — `var2 <= 122` (lowercase 'z'
					// upper bound). The prior `<= 132` was a digit-transposition
					// typo and wrongly accepted chars 123-132 ({ | } ~ etc.) into
					// social-input strings that downstream base37 packing then
					// silently drops or corrupts.
					if var2 >= 32 && var2 <= 122 && len(c.SocialInput) < 80 {
						c.SocialInput = c.SocialInput + string(rune(var2))
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
							var8 = jstring.ToBase37(c.SocialInput)
							c.AddFriend(var8)
						}
						if c.SocialAction == 2 && c.FriendCount > 0 {
							var8 = jstring.ToBase37(c.SocialInput)
							c.RemoveFriend(var8)
						}
						if c.SocialAction == 3 && len(c.SocialInput) > 0 {
							c.Out.P1Isaac(148)
							c.Out.P1(0)
							var7 = c.Out.Pos
							c.Out.P8(c.SocialName37)
							wordpack.Pack(c.Out, true, c.SocialInput)
							c.Out.PSize1(c.Out.Pos - var7)
							c.SocialInput = jstring.ToSentenceCase(c.SocialInput)
							c.SocialInput = wordfilter.Filter(c.SocialInput)
							c.AddMessage(6, c.SocialInput, jstring.FormatName(jstring.FromBase37(c.SocialName37)))
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
							var8 = jstring.ToBase37(c.SocialInput)
							c.AddIgnore(var8)
						}
						if c.SocialAction == 5 && c.IgnoreCount > 0 {
							var8 = jstring.ToBase37(c.SocialInput)
							c.RemoveIgnore(var8)
						}
					}
				} else if c.ChatbackInputOpen {
					if var2 >= 48 && var2 <= 57 && len(c.ChatbackInput) < 10 {
						c.ChatbackInput = c.ChatbackInput + string(rune(var2))
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
						c.ChatTyped = c.ChatTyped + string(rune(var2))
						c.RedrawChatback = true
					}
					if var2 == 8 && len(c.ChatTyped) > 0 {
						c.ChatTyped = c.ChatTyped[0 : len(c.ChatTyped)-1]
						c.RedrawChatback = true
					}
					if (var2 == 13 || var2 == 10) && len(c.ChatTyped) > 0 {
						if c.ChatTyped == "::clientdrop" && (c.Frame != nil || strings.Contains(c.GetHost(), "192.168.1.")) {
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
							wordpack.Pack(c.Out, true, c.ChatTyped)
							c.Out.PSize1(c.Out.Pos - var5)
							c.ChatTyped = jstring.ToSentenceCase(c.ChatTyped)
							c.ChatTyped = wordfilter.Filter(c.ChatTyped)
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
			c.ReportAbuseInput = c.ReportAbuseInput + string(rune(var2))
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
		var2 += 15 //nolint:ineffassign // Java: faithful dead final layout increment (var2 not read after)
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
			// Java: client.java:3001-3006 — CHARSET.charAt(i) returns a UTF-16
			// code unit (e.g. '£' = 0x00A3 as a single value). Byte-indexing the
			// Go string would split multi-byte UTF-8 sequences, so iterate runes.
			for _, r := range CHARSET {
				if var5 == int(r) {
					var6 = true
					break
				}
			}
			switch c.TitleLoginField {
			case 0:
				if var5 == 8 && len(c.Username) > 0 {
					c.Username = c.Username[0 : len(c.Username)-1]
				}
				if var5 == 9 || var5 == 10 || var5 == 13 {
					c.TitleLoginField = 1
				}
				if var6 {
					c.Username = c.Username + string(rune(var5))
				}
				if len(c.Username) > 12 {
					c.Username = c.Username[:12]
				}
			case 1:
				if var5 == 8 && len(c.Password) > 0 {
					c.Password = c.Password[0 : len(c.Password)-1]
				}
				if var5 == 9 || var5 == 10 || var5 == 13 {
					c.TitleLoginField = 0
				}
				if var6 {
					c.Password = c.Password + string(rune(var5))
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

// Java: loadArchive (deob/client.java:3046-3047) — renamed to GetJagFile.
func (c *Client) GetJagFile(displayName string, crc int, name string, progress int) *io.Jagfile {
	retry := 5
	data := signlink.CacheLoad(name)
	checksum := 0

	loadingError := func() {
		data = nil
		for checksum = retry; checksum > 0; checksum-- {
			c.DrawProgress("Error loading - Will retry in "+strconv.Itoa(checksum)+" secs.", progress)
			time.Sleep(1 * time.Second)
		}
		retry *= 2
		if retry > 60 {
			retry = 60
		}
	}

	if data != nil {
		checksum = int(crc32.ChecksumIEEE(data))
		if checksum != crc {
			data = nil
		}
	}

	if data != nil {
		return io.NewJagfile(data)
	}

	for data == nil {
		c.DrawProgress("Requesting "+displayName, progress)
		// Java: catch (IOException) — handled inline by loadingError() above on
		// each I/O step (open, read, chunked read). Java retries with exponential
		// backoff (5, 10, 20, 40, 60s).
		lastDownloaded := 0

		reader, err := c.OpenURL(name + strconv.Itoa(crc))
		if err != nil {
			log.Printf("client: GetJagFile error: %v", err)
			loadingError()
			continue
		}

		header := make([]byte, 6)
		n, err := reader.Read(header)
		if err != nil {
			log.Printf("client: GetJagFile read error: %v", err)
			loadingError()
			continue
		}
		if n < 6 {
			log.Printf("client: GetJagFile read %v bytes, expected 6", n)
			loadingError()
			continue
		}

		buf := io.NewPacket(header)
		buf.Pos = 3
		packedSize := buf.G3() + 6
		pos := 6

		data = make([]byte, packedSize)
		for i := range 6 {
			data[i] = header[i]
		}

		readFailed := false
		for pos < packedSize {
			chunkSize := packedSize - pos
			chunkSize = min(chunkSize, 1000)

			n, err := reader.Read(data[pos : pos+chunkSize])
			if err != nil {
				log.Printf("client: GetJagFile read error: %v", err)
				loadingError()
				readFailed = true
				break
			}

			pos += n

			downloaded := pos * 100 / packedSize
			if downloaded != lastDownloaded {
				c.DrawProgress("Loading "+displayName+" - "+strconv.Itoa(downloaded)+"%", progress)
			}
			lastDownloaded = downloaded
		}
		if readFailed {
			continue
		}
	}
	signlink.CacheSave(name, data)
	return io.NewJagfile(data)
}

func (c *Client) UnloadTitle() {
	// Stop the flame animation goroutine (Java: deob/client.java:3111).
	// The flame thread loops on c.FlameActive; set it false and spin
	// until c.FlameThread observes that and exits.
	c.FlameActive = false
	for c.FlameThread {
		c.FlameActive = false
		time.Sleep(50 * time.Millisecond)
	}
	// Java: deob/client.java:3119-3132 also nils imageTitlebox /
	// imageTitlebutton / imageRunes / flameGradient* / flameBuffer* /
	// imageFlamesLeft / imageFlamesRight here as a memory save. Go
	// keeps all of them alive: keeping ImageTitle2 alive ensures
	// LoadTitle's early-return (the `if c.ImageTitle2 != nil` guard)
	// fires on the Logout → title transition, preventing LoadTitle from
	// re-running LoadTitleImages → DrawProgress from mid-render. The
	// keepalive preserves the original invariant and avoids that
	// LoadTitle-mid-render path. Combined memory cost with the kept
	// ImageTitleN PixMaps is well under 2 MB — negligible.
}

func (c *Client) OrbitCamera(arg0, arg1, arg2, arg3, arg5, arg6 int) {
	var8 := (2048 - arg3) & 0x7FF
	var9 := (2048 - arg2) & 0x7FF
	var10 := 0
	var11 := 0
	var12 := arg6
	var13 := 0
	var14 := 0
	var15 := 0
	if var8 != 0 {
		var13 = model.Sin[var8]
		var14 = model.Cos[var8]
		var15 = (var11*var14 - arg6*var13) >> 16
		var12 = (var11*var13 + arg6*var14) >> 16
		var11 = var15
	}
	if var9 != 0 {
		var13 = model.Sin[var9]
		var14 = model.Cos[var9]
		var15 = (var12*var13 + var10*var14) >> 16
		var12 = (var12*var14 - var10*var13) >> 16
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
		var4 = var3.Wi*var3.Hi - 1
		var5 = var3.Wi * c.SceneDelta * 2
		var6 = var3.Pixels
		var7 = c.TextureBuffer
		for i := 0; i <= var4; i++ {
			var7[i] = var6[(i-var5)&var4]
		}
		var3.Pixels = var7
		c.TextureBuffer = var6
		pix3d.PushTexture(17)
	}
	if pix3d.TextureCycle[24] < arg0 {
		return
	}
	var3 = pix3d.Textures[24]
	var4 = var3.Wi*var3.Hi - 1
	var5 = var3.Wi * c.SceneDelta * 2
	var6 = var3.Pixels
	var7 = c.TextureBuffer
	for i := 0; i <= var4; i++ {
		var7[i] = var6[(i-var5)&var4]
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
			c.FlameBuffer3[i+((var2-2)<<7)] = 0xFF
		}
	}
	var5 := 0
	var6 := 0
	var7 := 0
	for range 100 {
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
			var9 := c.FlameBuffer2[var8+128] - c.FlameBuffer0[(var8+c.FlameCycle0)&(len(c.FlameBuffer0)-1)]/5
			var9 = max(var9, 0)
			c.FlameBuffer3[var8] = var9
		}
	}
	for i := range var2 - 1 {
		c.FlameLineOffset[i] = c.FlameLineOffset[i+1]
	}
	c.FlameLineOffset[var2-1] = int(math.Sin(float64(clientextras.LoopCycle)/14.0)*16.0 + math.Sin(float64(clientextras.LoopCycle)/15.0)*14.0 + math.Sin(float64(clientextras.LoopCycle)/16.0)*12.0)
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
		c.FlameGradientCycle0 = 0x400
	}
	if var8 == 1 {
		c.FlameGradientCycle1 = 0x400
	}
}

func (c *Client) DrawMinimap() {
	c.AreaMapback.Bind()
	var2 := (c.OrbitCameraYaw + c.MinimapAnticheatAngle) & 0x7FF
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
			var11 := jstring.ToBase37(var9.Name)
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
	pix2d.FillRect(82, 93, 0xFFFFFF, 3, 3)
	c.AreaViewport.Bind()
}

// Decision: getBaseComponent is NOT being ported. See viewbox.go for the
// architectural precedent (another AWT-shaped helper kept as an intentional
// non-port).
//
// Java getBaseComponent() (deob/client.java:3343-3350) returns the AWT
// Component the game should draw on top of — either super.frame (the AWT
// Frame opened by ViewBox), `this` (the Applet itself), or signlink.mainapp
// when running under the signed-applet bridge. Every caller in the Java
// source uses the result for one of two things:
//
//   1. As the first argument to `new PixMap(Component, w, h)` (~25 call
//      sites in Java client.java) — the AWT PixMap constructor needs a
//      Component to create a peer-backed Image. The Go pixmap package
//      (pkg/jagex2/graphics/pixmap) just allocates a width*height slice
//      directly via NewPixMap(width, height) — there is no Component
//      analogue and none is required.
//   2. drawError() and drawProgress() route AWT Graphics through it for
//      direct screen blits. Go renders everything through the central
//      pixmap.PixMap which is uploaded to the GPU once per frame by
//      gameshell.go, so the "draw to a component" path doesn't exist.
//
// In every case the Go translation already does the right thing without a
// Component reference, so exposing a `GetBaseComponent` method would just
// be a misleading stub that returns nil or *PixMap to no benefit.
//
// Java source: deob/client.java:3343-3350.

func (c *Client) UpdateMergeLocs() {
	if c.SceneState != 2 {
		return
	}
	for var2 := c.MergedLocations.Head(); var2 != nil; var2 = c.MergedLocations.Next() {
		v := var2.Value
		if clientextras.LoopCycle >= v.LastCycle {
			c.AddLoc(v.Angle, v.X, v.Z, v.Layer, v.LocIndex, v.Shape, v.Plane)
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
	var7 := ((int(rand.Float64()*20.0) + 238 - 10) << 16) + ((int(rand.Float64()*20.0) + 238 - 10) << 8) + (int(rand.Float64()*20.0) + 238 - 10)
	var8 := (int(rand.Float64()*20.0) + 238 - 10) << 16
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
				var12 = (var12 >> 14) & 0x7FFF
				var13 := loctype.Get(var12).MapFunction
				if var13 >= 0 {
					var14 := i
					var15 := j
					if var13 != 22 && var13 != 29 && var13 != 34 && var13 != 36 && var13 != 46 && var13 != 47 && var13 != 48 {
						var16 := 104
						var17 := 104
						var18 := c.LevelCollisionMap[c.CurrentLevel].Flags
						for range 10 {
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
		var9 = (var8 >> 6) & 0x3
		var10 = var8 & 0x1F
		var11 = arg2
		if var7 > 0 {
			var11 = arg4
		}
		var12 := c.ImageMinimap.Pixels
		var13 = arg3*4 + 24624 + (103-arg5)*512*4
		var14 = (var7 >> 14) & 0x7FFF
		var15 := loctype.Get(var14)
		if var15.MapScene == -1 {
			if var10 == 0 || var10 == 2 {
				switch var9 {
				case 0:
					var12[var13] = var11
					var12[var13+512] = var11
					var12[var13+0x400] = var11
					var12[var13+1536] = var11
				case 1:
					var12[var13] = var11
					var12[var13+1] = var11
					var12[var13+2] = var11
					var12[var13+3] = var11
				case 2:
					var12[var13+3] = var11
					var12[var13+3+512] = var11
					var12[var13+3+0x400] = var11
					var12[var13+3+1536] = var11
				case 3:
					var12[var13+1536] = var11
					var12[var13+1536+1] = var11
					var12[var13+1536+2] = var11
					var12[var13+1536+3] = var11
				}
			}
			if var10 == 3 {
				switch var9 {
				case 0:
					var12[var13] = var11
				case 1:
					var12[var13+3] = var11
				case 2:
					var12[var13+3+1536] = var11
				case 3:
					var12[var13+1536] = var11
				}
			}
			if var10 == 2 {
				switch var9 {
				case 3:
					var12[var13] = var11
					var12[var13+512] = var11
					var12[var13+0x400] = var11
					var12[var13+1536] = var11
				case 0:
					var12[var13] = var11
					var12[var13+1] = var11
					var12[var13+2] = var11
					var12[var13+3] = var11
				case 1:
					var12[var13+3] = var11
					var12[var13+3+512] = var11
					var12[var13+3+0x400] = var11
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
				var17 := (var15.Width*4 - var16.Wi) / 2
				var18 := (var15.Length*4 - var16.Hi) / 2
				var16.PlotSprite((104-arg5-var15.Length)*4+48+var18, arg3*4+48+var17)
			}
		}
	}
	var7 = c.Scene.GetLocBitSet(arg1, arg3, arg5)
	if var7 != 0 {
		var8 = c.Scene.GetInfo(arg1, arg3, arg5, var7)
		var9 = (var8 >> 6) & 0x3
		var10 = var8 & 0x1F
		var11 = (var7 >> 14) & 0x7FFF
		var22 := loctype.Get(var11)
		var26 := 0
		if var22.MapScene != -1 {
			var24 := c.ImageMapscene[var22.MapScene]
			if var24 != nil {
				var14 = (var22.Width*4 - var24.Wi) / 2
				var26 = (var22.Length*4 - var24.Hi) / 2
				var24.PlotSprite((104-arg5-var22.Length)*4+48+var26, arg3*4+48+var14)
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
				var25[var26+0x400+1] = var13
				var25[var26+512+2] = var13
				var25[var26+3] = var13
			} else {
				var25[var26] = var13
				var25[var26+512+1] = var13
				var25[var26+0x400+2] = var13
				var25[var26+1536+3] = var13
			}
		}
	}
	var7 = c.Scene.GetGroundDecorationBitSet(arg1, arg3, arg5)
	if var7 == 0 {
		return
	}
	var8 = (var7 >> 14) & 0x7FFF
	var20 := loctype.Get(var8)
	if var20.MapScene == -1 {
		return
	}
	var21 := c.ImageMapscene[var20.MapScene]
	if var21 != nil {
		var11 = (var20.Width*4 - var21.Wi) / 2
		var23 := (var20.Length*4 - var21.Hi) / 2
		var21.PlotSprite((104-arg5-var20.Length)*4+48+var23, arg3*4+48+var11)
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
		if c.NPCs[var5].Cycle != clientextras.LoopCycle {
			c.NPCs[var5].Type = nil
			c.NPCs[var5] = nil
		}
	}
	if arg0.Pos != psize {
		msg := c.Username + " size mismatch in getnpcpos - pos:" + strconv.Itoa(arg0.Pos) + " psize:" + strconv.Itoa(psize)
		signlink.ReportErrorFunc(msg)
		panic(msg)
	}
	for i := range c.NPCCount {
		if c.NPCs[c.NPCIDs[i]] == nil {
			msg := c.Username + " null entry in npc list - pos:" + strconv.Itoa(i) + " size:" + strconv.Itoa(c.NPCCount)
			signlink.ReportErrorFunc(msg)
			panic(msg)
		}
	}
}

// Decision: startThread is NOT being ported as a method. Java's
// startThread(Runnable, int) (deob/client.java:3611-3618) is a thin
// dispatcher: when running as an Applet (signlink.mainapp == null) it
// delegates to Applet.startThread(Runnable, int) — which under the hood
// calls `new Thread(runnable).start()` plus `setPriority(arg1)` — and
// when running under the signed-applet bridge it forwards to
// signlink.startthread which does the same thing inside the signed jar.
//
// The Go translation uses goroutines directly at every call site:
//   - client.go: `go c.RunFlames()` for the flames thread
//     (Java: `this.startThread(this, 2)` at deob/client.java:3685)
//   - client.go: `go c.RunMidi()` for the MIDI loader thread
//     (Java: `this.startThread(this, 2)` at deob/client.java:5952)
//   - pkg/jagex2/io/clientstream NewClientStream: `go cs.readRun()`
//     (Java: `shell.startThread(this, 2)` from ClientStream's ctor)
//
// Go has no thread-priority concept, so the priority argument (always 2
// in client.java; sometimes other values elsewhere) is silently dropped.
// The Go scheduler is preemptive and the loops involved are not CPU-bound
// enough for the dropped priority hint to matter in practice.
//
// Java source: deob/client.java:3611-3618.

func (c *Client) LoadTitleImages() {
	c.ImageTitleBox = pix8.NewPix8(c.JagTitle, "titlebox", 0)
	c.ImageTitleButton = pix8.NewPix8(c.JagTitle, "titlebutton", 0)
	c.ImageRunes = make([]*pix8.Pix8, 12)
	for i := range 12 {
		c.ImageRunes[i] = pix8.NewPix8(c.JagTitle, "runes", i)
	}
	c.ImageFlamesLeft = pix32.NewPix321(128, 265)
	c.ImageFlamesRight = pix32.NewPix321(128, 265)

	for i := range 33920 {
		c.ImageFlamesLeft.Pixels[i] = c.ImageTitle0.Data[i]
	}
	for i := range 33920 {
		c.ImageFlamesRight.Pixels[i] = c.ImageTitle1.Data[i]
	}

	c.FlameGradient0 = make([]int, 256)
	for i := range 64 {
		c.FlameGradient0[i] = i * 0x40000
	}
	for i := range 64 {
		c.FlameGradient0[i+64] = i*0x400 + 0xFF0000
	}
	for i := range 64 {
		c.FlameGradient0[i+128] = i*0x4 + 0xFFFF00
	}
	for i := range 64 {
		c.FlameGradient0[i+192] = 0xFFFFFF
	}

	c.FlameGradient1 = make([]int, 256)
	for i := range 64 {
		c.FlameGradient1[i] = i * 0x400
	}
	for i := range 64 {
		c.FlameGradient1[i+64] = i*0x4 + 0xFF00
	}
	for i := range 64 {
		c.FlameGradient1[i+128] = i*0x40000 + 0xFFFF
	}
	for i := range 64 {
		c.FlameGradient1[i+192] = 0xFFFFFF
	}

	c.FlameGradient2 = make([]int, 256)
	for i := range 64 {
		c.FlameGradient2[i] = i * 0x4
	}
	for i := range 64 {
		c.FlameGradient2[i+64] = i*0x40000 + 0xFF
	}
	for i := range 64 {
		c.FlameGradient2[i+128] = i*0x400 + 0xFF00FF
	}
	for i := range 64 {
		c.FlameGradient2[i+192] = 0xFFFFFF
	}

	c.FlameGradient = make([]int, 256)
	c.FlameBuffer0 = make([]int, 32768)
	c.FlameBuffer1 = make([]int, 32768)
	c.UpdateFlameBuffer(nil)
	c.FlameBuffer3 = make([]int, 32768)
	c.FlameBuffer2 = make([]int, 32768)

	c.DrawProgress("Connecting to fileserver", 10)
	if !c.FlameActive {
		c.FlamesThread = true
		c.FlameActive = true
		// Direct call — see Load:5491 for the dispatch-race rationale.
		go c.RunFlames()
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
		msg := c.Username + " Too many players"
		signlink.ReportErrorFunc(msg)
		panic(msg)
	}
	c.PlayerCount = 0
	for i := range var4 {
		var6 := c.PlayerIDs[i]
		var7 := c.Players[var6]
		var8 := arg1.GBit(1)
		if var8 == 0 {
			c.PlayerIDs[c.PlayerCount] = var6
			c.PlayerCount++
			var7.Cycle = clientextras.LoopCycle
		} else {
			var9 := arg1.GBit(2)
			if var9 == 0 {
				c.PlayerIDs[c.PlayerCount] = var6
				c.PlayerCount++
				var7.Cycle = clientextras.LoopCycle
				c.EntityUpdateIDs[c.EntityUpdateCount] = var6
				c.EntityUpdateCount++
			} else {
				var10 := 0
				var11 := 0
				switch var9 {
				case 1:
					c.PlayerIDs[c.PlayerCount] = var6
					c.PlayerCount++
					var7.Cycle = clientextras.LoopCycle
					var10 = arg1.GBit(3)
					var7.MoveAlongRoute(false, var10)
					var11 = arg1.GBit(1)
					if var11 == 1 {
						c.EntityUpdateIDs[c.EntityUpdateCount] = var6
						c.EntityUpdateCount++
					}
				case 2:
					c.PlayerIDs[c.PlayerCount] = var6
					c.PlayerCount++
					var7.Cycle = clientextras.LoopCycle
					var10 = arg1.GBit(3)
					var7.MoveAlongRoute(true, var10)
					var11 = arg1.GBit(3)
					var7.MoveAlongRoute(true, var11)
					var12 := arg1.GBit(1)
					if var12 == 1 {
						c.EntityUpdateIDs[c.EntityUpdateCount] = var6
						c.EntityUpdateCount++
					}
				case 3:
					c.EntityRemovalIDs[c.EntityRemovalCount] = var6
					c.EntityRemovalCount++
				}
			}
		}
	}
}

func (c *Client) DrawScrollbar(arg1, arg2, arg3, arg4, arg5 int) {
	c.ImageScrollbar0.PlotSprite(arg2, arg1)
	c.ImageScrollbar1.PlotSprite(arg2+arg5-16, arg1)
	pix2d.FillRect(arg2+16, arg1, c.SCROLLBAR_TRACK, 16, arg5-32)
	var7 := (arg5 - 32) * arg5 / arg4
	var7 = max(var7, 8)
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
	// Java: deob/client.java:3782 — savemidi(byte[], int, boolean).
	// The Java version wrote bytes through signlink.midisave so the
	// signed-applet wrapper could read jingleN.mid from disk and feed
	// it to javax.sound.midi. In Go there's no process boundary;
	// audio.PlayMIDI accepts the bytes directly, cutting ~70ms of
	// polling + disk write/read latency off the track-change path
	// (most visibly on the title-screen → game-music transition,
	// which the TS reference handles in essentially-zero time via
	// playMidi(buffer)).
	//
	// MidiFade is still published through signlink so the audio
	// watcher's "stop" / "voladjust" handlers can read it — same as
	// before. SaveMidi's per-call fade arg flows directly into
	// PlayMIDI rather than through the signlink field, which removes
	// the same race-window the old signlink-field path had.
	if arg3 {
		signlink.SetMidiFade(1)
	} else {
		signlink.SetMidiFade(0)
	}
	audio.PlayMIDI(arg0[:arg2], arg3)
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
	signlink.SetMidiVol(arg1)
	c.PacketSize += arg0
	if arg2 {
		signlink.SetMidiCommand("voladjust")
	}
}

func (c *Client) DrawTitleScreen() {
	c.LoadTitle()
	c.ImageTitle4.Bind()
	c.ImageTitleBox.PlotSprite(0, 0)
	var2 := 360
	var3 := 200
	var4 := 0
	var5 := 0
	var6 := 0
	if c.TitleScreenState == 0 {
		var4 = var3/2 - 20
		c.FontBold12.DrawStringTaggableCenter(var2/2, 0xFFFF00, true, var4, "Welcome to RuneScape")
		_ = var4 + 30
		var5 = var2/2 - 80
		var6 = var3/2 + 20
		c.ImageTitleButton.PlotSprite(var6-20, var5-73)
		c.FontBold12.DrawStringTaggableCenter(var5, 0xFFFFFF, true, var6+5, "New user")
		var8 := var2/2 + 80
		c.ImageTitleButton.PlotSprite(var6-20, var8-73)
		c.FontBold12.DrawStringTaggableCenter(var8, 0xFFFFFF, true, var6+5, "Existing User")
	}
	if c.TitleScreenState == 2 {
		var4 = var3/2 - 40
		if len(c.LoginMessage0) > 0 {
			c.FontBold12.DrawStringTaggableCenter(var2/2, 0xFFFF00, true, var4-15, c.LoginMessage0)
			c.FontBold12.DrawStringTaggableCenter(var2/2, 0xFFFF00, true, var4, c.LoginMessage1)
			var4 += 30
		} else {
			c.FontBold12.DrawStringTaggableCenter(var2/2, 0xFFFF00, true, var4-7, c.LoginMessage1)
			var4 += 30
		}
		tmp := ""
		if c.TitleLoginField == 0 && clientextras.LoopCycle%40 < 20 {
			tmp = "@yel@|"
		}
		c.FontBold12.DrawStringTaggable(var2/2-90, var4, "Username: "+c.Username+tmp, true, 0xFFFFFF)
		var4 += 15
		tmp2 := ""
		if c.TitleLoginField == 1 && clientextras.LoopCycle%40 < 20 {
			tmp2 = "@yel@|"
		}
		c.FontBold12.DrawStringTaggable(var2/2-88, var4, "Password: "+jstring.ToAsterisks(c.Password)+tmp2, true, 0xFFFFFF)
		var4 += 15 //nolint:ineffassign // Java: faithful dead final layout increment (var4 not read after)
		var5 = var2/2 - 80
		var6 = var3/2 + 50
		c.ImageTitleButton.PlotSprite(var6-20, var5-73)
		c.FontBold12.DrawStringTaggableCenter(var5, 0xFFFFFF, true, var6+5, "Login")
		var5 = var2/2 + 80
		c.ImageTitleButton.PlotSprite(var6-20, var5-73)
		c.FontBold12.DrawStringTaggableCenter(var5, 0xFFFFFF, true, var6+5, "Cancel")
	}
	if c.TitleScreenState == 3 {
		c.FontBold12.DrawStringTaggableCenter(var2/2, 0xFFFF00, true, var3/2-60, "Create a free account")
		var4 = var3/2 - 35
		c.FontBold12.DrawStringTaggableCenter(var2/2, 0xFFFFFF, true, var4, "To create a new account you need to")
		var4 += 15
		c.FontBold12.DrawStringTaggableCenter(var2/2, 0xFFFFFF, true, var4, "go back to the main RuneScape webpage")
		var4 += 15
		c.FontBold12.DrawStringTaggableCenter(var2/2, 0xFFFFFF, true, var4, "and choose the red 'create account'")
		var4 += 15
		c.FontBold12.DrawStringTaggableCenter(var2/2, 0xFFFFFF, true, var4, "button at the top right of that page.")
		var4 += 15 //nolint:ineffassign // Java: faithful dead final layout increment (var4 not read after)
		var5 = var2 / 2
		var6 = var3/2 + 50
		c.ImageTitleButton.PlotSprite(var6-20, var5-73)
		c.FontBold12.DrawStringTaggableCenter(var5, 0xFFFFFF, true, var6+5, "Cancel")
	}
	c.ImageTitle4.Draw(214, 186)
	// The back buffer used to retain pixels across frames (Java/AWT), so the
	// static background tiles only needed re-uploading on a full "dirty"
	// redraw (c.RedrawFrame). The upload-op must re-issue each frame. Hoist
	// the Draw calls out of the dirty-flag guard so they always run. The
	// flame tiles 0 and 1 are uploaded here too (DrawFlames now only updates
	// their pixel buffers; this entry point owns the GPU upload).
	c.RedrawFrame = false
	// flameMu: ImageTitle0/1 buffers are written by the RunFlames goroutine.
	c.flameMu.Lock()
	c.ImageTitle0.Draw(0, 0)
	c.ImageTitle1.Draw(661, 0)
	c.flameMu.Unlock()
	c.ImageTitle2.Draw(128, 0)
	c.ImageTitle3.Draw(214, 386)
	c.ImageTitle5.Draw(0, 265)
	c.ImageTitle6.Draw(574, 265)
	c.ImageTitle7.Draw(128, 186)
	c.ImageTitle8.Draw(574, 186)
}

func (c *Client) PrepareGameScreen() {
	if c.AreaChatback != nil {
		return
	}
	c.UnloadTitle()
	// Java: deob/client.java:3897-3902 nils imageTitle0..8 here for memory.
	// Go keeps all nine alive because:
	//   1. ImageTitle0/1 are uploaded every frame in DrawGame (the top-
	//      corner flame regions) via PixMap.Draw → platform.Active.Blit;
	//      pre-Gio (Java/AWT) the retained back buffer preserved them
	//      between frames, but the current platform model re-blits each frame.
	//   2. ImageTitle2..8 stay alive so c.DrawTitleScreen → c.LoadTitle's
	//      early-return (the `if c.ImageTitle2 != nil` guard) fires on the
	//      Logout transition. Otherwise LoadTitle would re-run LoadTitleImages
	//      → DrawProgress from mid-render, re-initialising title assets
	//      while the render is in progress. The keepalive preserves the
	//      original invariant and avoids that LoadTitle-mid-render path.
	// Combined memory cost ~1.7 MB — negligible.
	c.AreaChatback = pixmap.NewPixMap(479, 96)
	c.AreaMapback = pixmap.NewPixMap(168, 160)
	pix2d.Clear()
	c.ImageMapback.PlotSprite(0, 0)
	c.AreaSidebar = pixmap.NewPixMap(190, 261)
	c.AreaViewport = pixmap.NewPixMap(512, 334)
	// The viewport is re-rendered (DrawScene) almost every frame, so the
	// hashPixels change-detection is pure overhead — upload unconditionally.
	c.AreaViewport.AlwaysUpload = true
	pix2d.Clear()
	c.AreaBackbase1 = pixmap.NewPixMap(501, 61)
	c.AreaBackbase2 = pixmap.NewPixMap(288, 40)
	c.AreaBackhmid1 = pixmap.NewPixMap(269, 66)
	c.RedrawFrame = true
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
		var5.Cycle = clientextras.LoopCycle
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
	if c.Stream != nil {
		c.Stream.Close()
	}
	c.Stream = nil
	// Java: deob/client.java:3963 — `this.ingame = false`. Without
	// this, the game-vs-title render dispatch (e.g. UpdateGame's early
	// return at client.go:6818, and the InGame branches in the main draw
	// path) keeps treating the session as in-game and the title screen
	// never reappears.
	c.InGame = false
	c.TitleScreenState = 0
	c.Username = ""
	c.Password = ""
	inputtracking.SetDisabled()
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
	// Java: deob/client.java:3981 — `arg3.childId == null` (return when there
	// are no children). Java `== null` ports as Go `== nil`; the prior
	// translation flipped the operator to `!= nil`, which made every Type-0
	// layer with children early-return — silently blanking every interface.
	if arg3.Type != 0 || arg3.ChildID == nil || arg3.Hide && c.ViewportHoveredInterfaceIndex != arg3.Id && c.SidebarHoveredInterfaceIndex != arg3.Id && c.ChatHoveredInterfaceIndex != arg3.Id {
		return
	}
	var6 := pix2d.Left
	var7 := pix2d.Top
	var8 := pix2d.Right
	var9 := pix2d.Bottom
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
									var23.PlotSprite(var32, var18)
								}
								if var23.OWi == 33 || var14.InvSlotObjCount[var27] != 1 {
									var24 := var14.InvSlotObjCount[var27]
									c.FontPlain11.DrawString(var18+1+var33, var32+10+var21, 0, FormatObjCount(var24))
									c.FontPlain11.DrawString(var18+var33, var32+9+var21, 0xFFFF00, FormatObjCount(var24))
								}
							}
						} else if var14.InvSlotSprite != nil && var27 < 20 {
							var36 := var14.InvSlotSprite[var27]
							if var36 != nil {
								var36.PlotSprite(var32, var18)
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
						if strings.Contains(var29, "%") {
						label260:
							for {
								var33 = strings.Index(var29, "%1")
								// Java: deob/client.java:4093 — `== -1` (not found
								// → fall through to %2). The prior `== 1` typo dropped
								// the minus and would (a) skip the %1 substitution
								// whenever %1 sat at non-1 positions in text, and
								// (b) panic on var29[0:-1] when no "%1" was present.
								if var33 == -1 {
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
						var28.PlotSprite(var26, var25)
					}
				} else if var14.Type == 6 {
					var27 = pix3d.CenterW3D
					var16 = pix3d.CenterH3D
					pix3d.CenterW3D = var25 + var14.Width/2
					pix3d.CenterH3D = var26 + var14.Height/2
					var17 = (pix3d.SinTable[var14.Xan] * var14.Zoom) >> 16
					var18 = (pix3d.CosTable[var14.Xan] * var14.Zoom) >> 16
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
		c.RedrawFrame = true
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
			c.UpdateNpcEntity(var4)
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
	if arg0.ForceMoveEndCycle > clientextras.LoopCycle {
		c.UpdateForceMovement(&arg0.PathingEntity)
	} else if arg0.ForceMoveStartCycle >= clientextras.LoopCycle {
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
	if arg0.ForceMoveEndCycle > clientextras.LoopCycle {
		c.UpdateForceMovement(&arg0.PathingEntity)
	} else if arg0.ForceMoveStartCycle >= clientextras.LoopCycle {
		c.StartForceMovement(&arg0.PathingEntity, 0)
	} else {
		c.UpdateMovement(&arg0.PathingEntity)
	}
	c.UpdateFacingDirection(&arg0.PathingEntity)
	c.UpdateSequences(&arg0.PathingEntity)
}

func (c *Client) UpdateForceMovement(arg0 *entity.PathingEntity) {
	var3 := arg0.ForceMoveEndCycle - clientextras.LoopCycle
	var4 := arg0.ForceMoveStartSceneTileX*128 + arg0.Size*64
	var5 := arg0.ForceMoveStartSceneTileZ*128 + arg0.Size*64
	arg0.X += (var4 - arg0.X) / var3
	arg0.Z += (var5 - arg0.Z) / var3
	arg0.SeqTrigger = 0
	switch arg0.ForceMoveFaceDirection {
	case 0:
		arg0.DstYaw = 0x400
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
	if arg0.ForceMoveStartCycle == clientextras.LoopCycle || arg0.PrimarySeqID == -1 || arg0.PrimarySeqDelay != 0 || arg0.PrimarySeqCycle+1 > seqtype.Instances[arg0.PrimarySeqID].Delay[arg0.PrimarySeqFrame] {
		var3 := arg0.ForceMoveStartCycle - arg0.ForceMoveEndCycle
		var4 := clientextras.LoopCycle - arg0.ForceMoveEndCycle
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
		arg0.DstYaw = 0x400
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
		arg1.DstYaw = 0x400
	} else {
		arg1.DstYaw = 0
	}
	var7 := (arg1.DstYaw - arg1.Yaw) & 0x7FF
	if var7 > 0x400 {
		var7 -= 2048
	}
	var8 := arg1.SeqTurnAroundID
	if var7 >= -256 && var7 <= 256 {
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
	var7 = (arg0.DstYaw - arg0.Yaw) & 0x7FF
	if var7 == 0 {
		return
	}
	if var7 < 32 || var7 > 2016 {
		arg0.Yaw = arg0.DstYaw
	} else if var7 > 0x400 {
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
	if arg1.SpotanimID == -1 || clientextras.LoopCycle < arg1.SpotanimLastCycle {
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
	// Always upload the static frame-chrome tiles. Pre-Gio (Java/AWT)
	// these were gated by c.RedrawFrame because AWT retained the back
	// buffer; PixMap.Draw → platform.Active.Blit re-blits each frame.
	// Flame tiles (ImageTitle0/1) too — DrawFlames now only updates
	// their pixel buffers, this entry point uploads. The pixmaps stay
	// alive past PrepareGameScreen so the top-left/top-right corners
	// render the last flame-animation frame (matching Java's retained-
	// back-buffer visual). Nil-guarded for the Logout → LoadTitle
	// transition window where the buffers are briefly nil before
	// being re-allocated.
	//
	// flameMu: ImageTitle0/1 buffers are written by the RunFlames goroutine.
	c.flameMu.Lock()
	if c.ImageTitle0 != nil {
		c.ImageTitle0.Draw(0, 0)
	}
	if c.ImageTitle1 != nil {
		c.ImageTitle1.Draw(661, 0)
	}
	c.flameMu.Unlock()
	c.AreaBackleft1.Draw(0, 11)
	c.AreaBackleft2.Draw(0, 375)
	c.AreaBackright1.Draw(729, 5)
	c.AreaBackright2.Draw(752, 231)
	c.AreaBacktop1.Draw(0, 0)
	c.AreaBacktop2.Draw(561, 0)
	c.AreaBackvmid1.Draw(520, 11)
	c.AreaBackvmid2.Draw(520, 231)
	c.AreaBackvmid3.Draw(501, 375)
	c.AreaBackhmid2.Draw(0, 345)
	if c.SceneState != 2 {
		c.AreaViewport.Draw(8, 11)
		c.AreaMapback.Draw(561, 5)
	}
	if c.RedrawFrame {
		c.RedrawFrame = false
		// Java set redrawSidebar/Chatback/SideIcons/PrivacySettings
		// here to force a content rebuild on full-frame dirty. Retained
		// because the inner repaints in those sub-draws are still
		// expensive enough to gate.
		c.RedrawSidebar = true
		c.RedrawChatback = true
		c.RedrawSideIcons = true
		c.RedrawPrivacySettings = true
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
	// DrawSidebar always runs — it internally gates the expensive pixel
	// repaint on c.RedrawSidebar but unconditionally blits AreaSidebar
	// via PixMap.Draw so the GPU always sees the current state.
	c.DrawSidebar()
	if c.ChatInterfaceID == -1 {
		c.ChatInterface.ScrollPosition = c.ChatScrollHeight - c.ChatScrollOffset - 77
		if c.MouseX > 453 && c.MouseX < 565 && c.MouseY > 350 {
			c.HandleScrollInput(c.MouseX-22, 0, c.MouseY-375, c.ChatScrollHeight, 77, false, 463, 0, c.ChatInterface)
		}
		var3 := c.ChatScrollHeight - 77 - c.ChatInterface.ScrollPosition
		var3 = max(var3, 0)
		var3 = min(var3, c.ChatScrollHeight-77)
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
	// DrawChatback always runs — same rationale as DrawSidebar above.
	c.DrawChatback()
	if c.SceneState == 2 {
		c.DrawMinimap()
		c.AreaMapback.Draw(561, 5)
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
		c.ImageBackhmid1.PlotSprite(0, 0)
		if c.SidebarInterfaceID == -1 {
			if c.TabInterfaceID[c.SelectedTab] != -1 {
				switch c.SelectedTab {
				case 0:
					c.ImageRedstone1.PlotSprite(30, 29)
				case 1:
					c.ImageRedstone2.PlotSprite(29, 59)
				case 2:
					c.ImageRedstone2.PlotSprite(29, 87)
				case 3:
					c.ImageRedstone3.PlotSprite(29, 115)
				case 4:
					c.ImageRedstone2h.PlotSprite(29, 156)
				case 5:
					c.ImageRedstone2h.PlotSprite(29, 184)
				case 6:
					c.ImageRedstone1h.PlotSprite(30, 212)
				}
			}
			if c.TabInterfaceID[0] != -1 && (c.FlashingTab != 0 || clientextras.LoopCycle%20 < 10) {
				c.ImageSideIcons[0].PlotSprite(34, 35)
			}
			if c.TabInterfaceID[1] != -1 && (c.FlashingTab != 1 || clientextras.LoopCycle%20 < 10) {
				c.ImageSideIcons[1].PlotSprite(32, 59)
			}
			if c.TabInterfaceID[2] != -1 && (c.FlashingTab != 2 || clientextras.LoopCycle%20 < 10) {
				c.ImageSideIcons[2].PlotSprite(32, 86)
			}
			if c.TabInterfaceID[3] != -1 && (c.FlashingTab != 3 || clientextras.LoopCycle%20 < 10) {
				c.ImageSideIcons[3].PlotSprite(33, 121)
			}
			if c.TabInterfaceID[4] != -1 && (c.FlashingTab != 4 || clientextras.LoopCycle%20 < 10) {
				c.ImageSideIcons[4].PlotSprite(34, 157)
			}
			if c.TabInterfaceID[5] != -1 && (c.FlashingTab != 5 || clientextras.LoopCycle%20 < 10) {
				c.ImageSideIcons[5].PlotSprite(32, 185)
			}
			if c.TabInterfaceID[6] != -1 && (c.FlashingTab != 6 || clientextras.LoopCycle%20 < 10) {
				c.ImageSideIcons[6].PlotSprite(34, 212)
			}
		}
		c.AreaBackbase2.Bind()
		c.ImageBackbase2.PlotSprite(0, 0)
		if c.SidebarInterfaceID == -1 {
			if c.TabInterfaceID[c.SelectedTab] != -1 {
				switch c.SelectedTab {
				case 7:
					c.ImageRedstone1v.PlotSprite(0, 49)
				case 8:
					c.ImageRedstone2v.PlotSprite(0, 81)
				case 9:
					c.ImageRedstone2v.PlotSprite(0, 108)
				case 10:
					c.ImageRedstone3v.PlotSprite(1, 136)
				case 11:
					c.ImageRedstone2hv.PlotSprite(0, 178)
				case 12:
					c.ImageRedstone2hv.PlotSprite(0, 205)
				case 13:
					c.ImageRedstone1hv.PlotSprite(0, 233)
				}
			}
			// Java: deob/client.java:4828-4845 — `!= -1` (TabInterfaceID
			// defaults to -1 meaning "no interface assigned"). The prior
			// port dropped the minus on six consecutive sibling branches,
			// so the bottom-row side icons (Friends / Ignore / Logout
			// etc., tabs 8-13) rendered even when no interface was set.
			// Same defect-class as the `%1` typo just fixed at line 3553.
			if c.TabInterfaceID[8] != -1 && (c.FlashingTab != 8 || clientextras.LoopCycle%20 < 10) {
				c.ImageSideIcons[7].PlotSprite(2, 80)
			}
			if c.TabInterfaceID[9] != -1 && (c.FlashingTab != 9 || clientextras.LoopCycle%20 < 10) {
				c.ImageSideIcons[8].PlotSprite(3, 107)
			}
			if c.TabInterfaceID[10] != -1 && (c.FlashingTab != 10 || clientextras.LoopCycle%20 < 10) {
				c.ImageSideIcons[9].PlotSprite(4, 142)
			}
			if c.TabInterfaceID[11] != -1 && (c.FlashingTab != 11 || clientextras.LoopCycle%20 < 10) {
				c.ImageSideIcons[10].PlotSprite(2, 179)
			}
			if c.TabInterfaceID[12] != -1 && (c.FlashingTab != 12 || clientextras.LoopCycle%20 < 10) {
				c.ImageSideIcons[11].PlotSprite(2, 206)
			}
			if c.TabInterfaceID[13] != -1 && (c.FlashingTab != 13 || clientextras.LoopCycle%20 < 10) {
				c.ImageSideIcons[12].PlotSprite(2, 230)
			}
		}
		c.AreaViewport.Bind()
	}
	// Always upload the two SideIcons pixmaps. Pixel content edits
	// above were gated by RedrawSideIcons; the GPU upload runs every
	// frame so they don't go white between dirty cycles.
	c.AreaBackhmid1.Draw(520, 165)
	c.AreaBackbase2.Draw(501, 492)
	if c.RedrawPrivacySettings {
		c.RedrawPrivacySettings = false
		c.AreaBackbase1.Bind()
		c.ImageBackbase1.PlotSprite(0, 0)
		c.FontPlain12.DrawStringTaggableCenter(57, 0xFFFFFF, true, 33, "Public chat")
		switch c.PublicChatSetting {
		case 0:
			c.FontPlain12.DrawStringTaggableCenter(57, 0xFF00, true, 46, "On")
		case 1:
			c.FontPlain12.DrawStringTaggableCenter(57, 0xFFFF00, true, 46, "Friends")
		case 2:
			c.FontPlain12.DrawStringTaggableCenter(57, 0xFF0000, true, 46, "Off")
		case 3:
			c.FontPlain12.DrawStringTaggableCenter(57, 0xFFFF, true, 46, "Hide")
		}
		c.FontPlain12.DrawStringTaggableCenter(186, 0xFFFFFF, true, 33, "Private chat")
		switch c.PrivateChatSetting {
		case 0:
			c.FontPlain12.DrawStringTaggableCenter(186, 0xFF00, true, 46, "On")
		case 1:
			c.FontPlain12.DrawStringTaggableCenter(186, 0xFFFF00, true, 46, "Friends")
		case 2:
			c.FontPlain12.DrawStringTaggableCenter(186, 0xFF0000, true, 46, "Off")
		}
		c.FontPlain12.DrawStringTaggableCenter(326, 0xFFFFFF, true, 33, "Trade/duel")
		switch c.TradeChatSetting {
		case 0:
			c.FontPlain12.DrawStringTaggableCenter(326, 0xFF00, true, 46, "On")
		case 1:
			c.FontPlain12.DrawStringTaggableCenter(326, 0xFFFF00, true, 46, "Friends")
		case 2:
			c.FontPlain12.DrawStringTaggableCenter(326, 0xFF0000, true, 46, "Off")
		}
		c.FontPlain12.DrawStringTaggableCenter(462, 0xFFFFFF, true, 38, "Report abuse")
		c.AreaViewport.Bind()
	}
	// Always upload the PrivacySettings pixmap. Pixel content edits
	// above were gated by RedrawPrivacySettings.
	c.AreaBackbase1.Draw(0, 471)
	c.SceneDelta = 0
}

// blitIf blits p with its top-left at (x, y) when p is allocated. Out-of-band
// repaints can run in transition windows where a few area pixmaps are briefly
// nil (e.g. Logout → LoadTitle), so guard rather than panic.
func (c *Client) blitIf(p *pixmap.PixMap, x, y int) {
	if p != nil {
		p.Draw(x, y)
	}
}

// blitRetainedScreen re-blits the full in-game screen from the retained area
// pixmaps WITHOUT re-rendering the 3D scene or touching redraw state. Positions
// mirror DrawGame's composite (keep them in sync). The GL backend clears the
// framebuffer on every BeginFrame, so an out-of-band present that blits only
// AreaViewport blacks out the surrounding UI for a frame — Java/AWT retained it
// across the partial drawImage. Re-blitting every area reproduces that retained
// screen; PixMap.Draw re-uploads only the pixmaps whose pixels changed.
func (c *Client) blitRetainedScreen() {
	c.flameMu.Lock()
	c.blitIf(c.ImageTitle0, 0, 0)
	c.blitIf(c.ImageTitle1, 661, 0)
	c.flameMu.Unlock()
	c.blitIf(c.AreaBackleft1, 0, 11)
	c.blitIf(c.AreaBackleft2, 0, 375)
	c.blitIf(c.AreaBackright1, 729, 5)
	c.blitIf(c.AreaBackright2, 752, 231)
	c.blitIf(c.AreaBacktop1, 0, 0)
	c.blitIf(c.AreaBacktop2, 561, 0)
	c.blitIf(c.AreaBackvmid1, 520, 11)
	c.blitIf(c.AreaBackvmid2, 520, 231)
	c.blitIf(c.AreaBackvmid3, 501, 375)
	c.blitIf(c.AreaBackhmid1, 520, 165)
	c.blitIf(c.AreaBackhmid2, 0, 345)
	c.blitIf(c.AreaBackbase1, 0, 471)
	c.blitIf(c.AreaBackbase2, 501, 492)
	c.blitIf(c.AreaViewport, 8, 11)
	c.blitIf(c.AreaMapback, 561, 5)
	c.blitIf(c.AreaSidebar, 562, 231)
	c.blitIf(c.AreaChatback, 22, 375)
}

// presentLoadingMessage shows the current full game screen with the caller's
// "Loading - please wait." text (already drawn into AreaViewport) overlaid in
// the viewport, without re-rendering the scene. Replaces the old
// present(AreaViewport.Draw) repaints, which blacked out everything but the
// viewport for a frame. See blitRetainedScreen.
func (c *Client) presentLoadingMessage() {
	c.present(c.blitRetainedScreen)
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
			var9 = jstring.FormatName(jstring.FromBase37(jstring.ToBase37(var7)))
			var10 := false
			for i := range c.PlayerCount {
				var12 := c.Players[c.PlayerIDs[i]]
				if var12 != nil && var12.Name != "" && strings.EqualFold(var12.Name, var9) {
					c.TryMove(c.LocalPlayer.PathTileX[0], 1, false, var12.PathTileX[0], c.LocalPlayer.PathTileZ[0], 2, 1, var12.PathTileZ[0], 0, 0, 0)
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
			c.TryMove(c.LocalPlayer.PathTileX[0], 1, false, var13.PathTileX[0], c.LocalPlayer.PathTileZ[0], 2, 1, var13.PathTileZ[0], 0, 0, 0)
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
		var14 = c.TryMove(c.LocalPlayer.PathTileX[0], 0, false, var3, c.LocalPlayer.PathTileZ[0], 2, 0, var4, 0, 0, 0)
		if !var14 {
			c.TryMove(c.LocalPlayer.PathTileX[0], 1, false, var3, c.LocalPlayer.PathTileZ[0], 2, 1, var4, 0, 0, 0)
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
		var15 := (var6 >> 14) & 0x7FFF
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
			c.TryMove(c.LocalPlayer.PathTileX[0], 1, false, var13.PathTileX[0], c.LocalPlayer.PathTileZ[0], 2, 1, var13.PathTileZ[0], 0, 0, 0)
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
			c.TryMove(c.LocalPlayer.PathTileX[0], 1, false, var19.PathTileX[0], c.LocalPlayer.PathTileZ[0], 2, 1, var19.PathTileZ[0], 0, 0, 0)
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
			c.TryMove(c.LocalPlayer.PathTileX[0], 1, false, var13.PathTileX[0], c.LocalPlayer.PathTileZ[0], 2, 1, var13.PathTileZ[0], 0, 0, 0)
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
			var20 = jstring.ToBase37(strings.TrimSpace(var7[var8+5:]))
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
		var14 = c.TryMove(c.LocalPlayer.PathTileX[0], 0, false, var3, c.LocalPlayer.PathTileZ[0], 2, 0, var4, 0, 0, 0)
		if !var14 {
			c.TryMove(c.LocalPlayer.PathTileX[0], 1, false, var3, c.LocalPlayer.PathTileZ[0], 2, 1, var4, 0, 0, 0)
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
		if strings.Contains(var18, " ") {
			var18 = var18[0:strings.Index(var18, " ")]
		}
		var9 = var22.ActionVerb
		if strings.Contains(var9, " ") {
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
		var14 = c.TryMove(c.LocalPlayer.PathTileX[0], 0, false, var3, c.LocalPlayer.PathTileZ[0], 2, 0, var4, 0, 0, 0)
		if !var14 {
			c.TryMove(c.LocalPlayer.PathTileX[0], 1, false, var3, c.LocalPlayer.PathTileZ[0], 2, 1, var4, 0, 0, 0)
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
			c.TryMove(c.LocalPlayer.PathTileX[0], 1, false, var19.PathTileX[0], c.LocalPlayer.PathTileZ[0], 2, 1, var19.PathTileZ[0], 0, 0, 0)
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
			var20 = jstring.ToBase37(strings.TrimSpace(var7[var8+5:]))
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
			c.TryMove(c.LocalPlayer.PathTileX[0], 1, false, var19.PathTileX[0], c.LocalPlayer.PathTileZ[0], 2, 1, var19.PathTileZ[0], 0, 0, 0)
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
	// Java: getHost() (deob/client.java:5508-5513). The applet branch
	// (signlink.mainapp != null) and the standalone branch both apply
	// .toLowerCase(); callers (e.g. ::clientdrop, host validation in
	// startApplication) compare against lowercase literals.
	return strings.ToLower(clientextras.Host)
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
		var11 := 0xFFFFFF
		if var7 > var2 && var7 < var2+var4 && var8 > var10-13 && var8 < var10+3 {
			var11 = 0xFFFF00
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
			switch c.FriendWorld[var3] {
			case 0:
				arg1.Text = "@red@Offline"
			case NodeID:
				arg1.Text = "@gre@World-" + strconv.Itoa(c.FriendWorld[var3]-9)
			default:
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
			arg1.Text = jstring.FormatName(jstring.FromBase37(c.IgnoreName37[var3]))
			arg1.ButtonType = 1
		}
	} else if var3 == 503 {
		arg1.Scroll = c.IgnoreCount*15 + 20
		if arg1.Scroll <= arg1.Height {
			arg1.Scroll = arg1.Height + 1
		}
	} else if var3 == 327 {
		arg1.Xan = 150
		arg1.Yan = int(math.Sin(float64(clientextras.LoopCycle)/40.0)*256.0) & 0x7FF
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
					var10.Recolor(clientextras.Field1307[i][0], clientextras.Field1307[i][c.DesignColors[i]])
					if i == 1 {
						var10.Recolor(clientextras.Field1438[0], clientextras.Field1438[c.DesignColors[i]])
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
		if clientextras.LoopCycle%20 < 10 {
			arg1.Text = arg1.Text + "|"
		} else {
			arg1.Text = arg1.Text + " "
		}
	} else {
		if var3 == 613 {
			if !c.Rights {
				arg1.Text = ""
			} else if c.ReportAbuseMuteOption {
				arg1.Colour = 0xFF0000
				arg1.Text = "Moderator option: Mute player for 48 hours: <ON>"
			} else {
				arg1.Colour = 0xFFFFFF
				arg1.Text = "Moderator option: Mute player for 48 hours: <OFF>"
			}
		}
		var4 := ""
		if var3 == 650 || var3 == 655 {
			if c.LastAddress == 0 {
				arg1.Text = ""
			} else {
				switch c.DaysSinceLastLogin {
				case 0:
					var4 = "earlier today"
				case 1:
					var4 = "yesterday"
				default:
					var4 = strconv.Itoa(c.DaysSinceLastLogin) + " days ago"
				}
				arg1.Text = "You last logged in " + var4 + " from: " + signlink.DNS
			}
		}
		if var3 == 651 {
			if c.UnreadMessages == 0 {
				arg1.Text = "0 unread messages"
				arg1.Colour = 0xFFFF00
			}
			if c.UnreadMessages == 1 {
				arg1.Text = "1 unread message"
				arg1.Colour = 0xFF00
			}
			if c.UnreadMessages > 1 {
				arg1.Text = strconv.Itoa(c.UnreadMessages) + " unread messages"
				arg1.Colour = 0xFF00
			}
		}
		if var3 == 652 {
			switch c.DaysSinceRecoveriesChanged {
			case 201:
				arg1.Text = ""
			case 200:
				arg1.Text = "You have not yet set any password recovery questions."
			default:
				switch c.DaysSinceRecoveriesChanged {
				case 0:
					var4 = "Earlier today"
				case 1:
					var4 = "Yesterday"
				default:
					var4 = strconv.Itoa(c.DaysSinceRecoveriesChanged) + " days ago"
				}
				arg1.Text = var4 + " you changed your recovery questions"
			}
		}
		if var3 == 653 {
			switch c.DaysSinceRecoveriesChanged {
			case 201:
				arg1.Text = ""
			case 200:
				arg1.Text = "We strongly recommend you do so now to secure your account."
			default:
				arg1.Text = "If you do not remember making this change then cancel it immediately"
			}
		}
		if var3 == 654 {
			switch c.DaysSinceRecoveriesChanged {
			case 201:
				arg1.Text = ""
			case 200:
				arg1.Text = "Do this from the 'account management' area on our front webpage"
			default:
				arg1.Text = "Do this from the 'account management' area on our front webpage"
			}
		}
	}
}

func (c *Client) SaveWave(arg0 []byte, arg1 int) bool {
	if arg0 == nil {
		return true
	}
	audio.PlayWave(arg0[:arg1])
	return true
}

func (c *Client) ReplayWave() bool {
	audio.ReplayWave()
	return true
}

func (c *Client) SetWaveVolume(vol int) {
	signlink.SetWaveVol(vol)
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
		var5.Cycle = clientextras.LoopCycle
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
				var6 = len(clientextras.Field1307[var4]) - 1
			}
		}
		if var5 == 1 {
			var6++
			if var6 >= len(clientextras.Field1307[var4]) {
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
			c.Out.P8(jstring.ToBase37(c.ReportAbuseInput))
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
		// Java: this.startThread(this, 2) (deob/client.java:5952) —
		// Thread.start() + setPriority(2) on the client Runnable. The
		// Java Runnable's run() dispatched based on flags; Go's goroutine
		// scheduler is asynchronous so calling the target function
		// directly makes intent unambiguous.
		go c.RunMidi()
		c.SetMidi(12345678, "scape_main", 40000)
	}

	if AlreadyStarted {
		c.ErrorStarted = true
		return
	}

	AlreadyStarted = true

	// Java host allowlist (deob/client.java:5962-5987): the applet set
	// errorHost (refusing to load) unless getCodeBase().getHost() ended in
	// jagex.com / runescape.com / a few 192.168.1.x dev IPs / 127.0.0.1.
	//
	// Intentionally NOT enforced in this standalone Go port: there is no
	// browser codebase host to validate, and the optional [host] CLI arg — plus
	// the ws://|wss:// WebSocket transport — exist precisely to point the client
	// at an operator-chosen server, which the allowlist would otherwise reject
	// (the source of the ErrorHost screen for any non-loopback host). Go-original
	// deviation; c.ErrorHost and DrawError's handling of it are left in place but
	// are no longer triggered here.

	// Java: try { ... } catch (Exception) { this.errorLoading = true; } —
	// wraps the entire archive-load / scene-init body below (client.java
	// 5990-6222). Recovering from a panic mirrors Java's swallow-and-flag
	// behavior so GameShell.Update can paint the error screen instead of
	// crashing.
	defer func() {
		if r := recover(); r != nil {
			log.Printf("client: Client.Load panic: %v", r)
			c.ErrorLoading = true
		}
	}()
	// Java: int var3 = 5 (deob/client.java:5991) — initial backoff seconds.
	retry := 5

	errorLoading := func() {
		for i := retry; i > 0; i-- {
			c.DrawProgress("Error loading - Will retry in "+strconv.Itoa(i)+" secs.", 10)
			time.Sleep(1 * time.Second)
		}
		retry *= 2
		if retry > 60 {
			retry = 60
		}
	}

	c.JagChecksum[8] = 0
	for c.JagChecksum[8] == 0 {
		c.DrawProgress("Connecting to fileserver", 10)
		reader, err := c.OpenURL("crc" + strconv.Itoa(int(rand.Float64()*9.9999999e7)))
		if err != nil {
			log.Printf("client: Client.Load OpenURL error: %v", err)
			errorLoading()
			continue
		}

		crc := io.NewPacket(make([]byte, 36))
		// Java: in.readFully(var5.data, 0, 36) (client.java:5997-6002) blocks
		// until all 36 bytes are read. io2.ReadFull matches that "fill exactly"
		// semantics; a bare Read could return fewer bytes if the source were
		// ever a streaming reader rather than the in-memory bytes.Reader.
		_, err = io2.ReadFull(reader, crc.Data[:36])
		if err != nil {
			log.Printf("client: Client.Load Read error: %v", err)
			errorLoading()
			continue
		}

		for i := range 9 {
			c.JagChecksum[i] = crc.G4()
		}
	}

	c.JagTitle = c.GetJagFile("title screen", c.JagChecksum[1], "title", 10)
	c.FontPlain11 = pixfont.NewPixFont(c.JagTitle, "p11")
	c.FontPlain12 = pixfont.NewPixFont(c.JagTitle, "p12")
	c.FontBold12 = pixfont.NewPixFont(c.JagTitle, "b12")
	c.FontQuill8 = pixfont.NewPixFont(c.JagTitle, "q8")

	c.LoadTitleBackground()
	c.LoadTitleImages()

	jagConfig := c.GetJagFile("config", c.JagChecksum[2], "config", 15)
	jagInterface := c.GetJagFile("interface", c.JagChecksum[3], "interface", 20)
	jagMedia := c.GetJagFile("2d graphics", c.JagChecksum[4], "media", 30)
	jagModels := c.GetJagFile("3d graphics", c.JagChecksum[5], "models", 40)
	jagTextures := c.GetJagFile("textures", c.JagChecksum[6], "textures", 60)
	jagWordEnc := c.GetJagFile("chat system", c.JagChecksum[7], "wordenc", 65)
	jagSounds := c.GetJagFile("sound effects", c.JagChecksum[8], "sounds", 70)

	c.LevelTileFlags = make([][][]byte, 4)
	for level := range c.LevelTileFlags {
		c.LevelTileFlags[level] = make([][]byte, 104)
		for x := range c.LevelTileFlags[level] {
			c.LevelTileFlags[level][x] = make([]byte, 104)
		}
	}

	c.LevelHeightMap = make([][][]int, 4)
	for level := range c.LevelHeightMap {
		c.LevelHeightMap[level] = make([][]int, 105)
		for x := range c.LevelHeightMap[level] {
			c.LevelHeightMap[level][x] = make([]int, 105)
		}
	}

	c.Scene = world3d.NewWorld3D(c.LevelHeightMap, 104, 4, 104)
	for i := range 4 {
		c.LevelCollisionMap[i] = dash3d.NewCollisionMap(104, 104)
	}

	c.ImageMinimap = pix32.NewPix321(512, 512)

	c.DrawProgress("Unpacking media", 75)

	c.ImageInvback = pix8.NewPix8(jagMedia, "invback", 0)
	c.ImageChatback = pix8.NewPix8(jagMedia, "chatback", 0)
	c.ImageMapback = pix8.NewPix8(jagMedia, "mapback", 0)

	c.ImageBackbase1 = pix8.NewPix8(jagMedia, "backbase1", 0)
	c.ImageBackbase2 = pix8.NewPix8(jagMedia, "backbase2", 0)
	c.ImageBackhmid1 = pix8.NewPix8(jagMedia, "backhmid1", 0)

	for i := range 13 {
		c.ImageSideIcons[i] = pix8.NewPix8(jagMedia, "sideicons", i)
	}

	c.ImageCompass = pix32.NewPix323(jagMedia, "compass", 0)

	for i := range 50 {
		c.ImageMapscene[i] = pix8.NewPix8(jagMedia, "mapscene", i)
	}

	for i := range 50 {
		c.ImageMapFunction[i] = pix32.NewPix323(jagMedia, "mapfunction", i)
	}

	func() {
		defer RecoverPanic()
		for i := range 20 {
			c.ImageHitmarks[i] = pix32.NewPix323(jagMedia, "hitmarks", i)
		}
	}()

	func() {
		defer RecoverPanic()
		for i := range 20 {
			c.ImageHeadIcons[i] = pix32.NewPix323(jagMedia, "headicons", i)
		}
	}()

	c.ImageMapflag = pix32.NewPix323(jagMedia, "mapflag", 0)

	for i := range 8 {
		c.ImageCrosses[i] = pix32.NewPix323(jagMedia, "cross", i)
	}

	c.ImageMapdot0 = pix32.NewPix323(jagMedia, "mapdots", 0)
	c.ImageMapdot1 = pix32.NewPix323(jagMedia, "mapdots", 1)
	c.ImageMapdot2 = pix32.NewPix323(jagMedia, "mapdots", 2)
	c.ImageMapdot3 = pix32.NewPix323(jagMedia, "mapdots", 3)

	c.ImageScrollbar0 = pix8.NewPix8(jagMedia, "scrollbar", 0)
	c.ImageScrollbar1 = pix8.NewPix8(jagMedia, "scrollbar", 1)

	c.ImageRedstone1 = pix8.NewPix8(jagMedia, "redstone1", 0)
	c.ImageRedstone2 = pix8.NewPix8(jagMedia, "redstone2", 0)
	c.ImageRedstone3 = pix8.NewPix8(jagMedia, "redstone3", 0)

	c.ImageRedstone1h = pix8.NewPix8(jagMedia, "redstone1", 0)
	c.ImageRedstone1h.HFlip()

	c.ImageRedstone2h = pix8.NewPix8(jagMedia, "redstone2", 0)
	c.ImageRedstone2h.HFlip()

	c.ImageRedstone1v = pix8.NewPix8(jagMedia, "redstone1", 0)
	c.ImageRedstone1v.VFlip()

	c.ImageRedstone2v = pix8.NewPix8(jagMedia, "redstone2", 0)
	c.ImageRedstone2v.VFlip()

	c.ImageRedstone3v = pix8.NewPix8(jagMedia, "redstone3", 0)
	c.ImageRedstone3v.VFlip()

	c.ImageRedstone1hv = pix8.NewPix8(jagMedia, "redstone1", 0)
	c.ImageRedstone1hv.HFlip()
	c.ImageRedstone1hv.VFlip()

	c.ImageRedstone2hv = pix8.NewPix8(jagMedia, "redstone2", 0)
	c.ImageRedstone2hv.HFlip()
	c.ImageRedstone2hv.VFlip()

	backleft1 := pix32.NewPix323(jagMedia, "backleft1", 0)
	c.AreaBackleft1 = pixmap.NewPixMap(backleft1.Wi, backleft1.Hi)
	backleft1.QuickPlotSprite(0, 0)

	backleft2 := pix32.NewPix323(jagMedia, "backleft2", 0)
	c.AreaBackleft2 = pixmap.NewPixMap(backleft2.Wi, backleft2.Hi)
	backleft2.QuickPlotSprite(0, 0)

	backright1 := pix32.NewPix323(jagMedia, "backright1", 0)
	c.AreaBackright1 = pixmap.NewPixMap(backright1.Wi, backright1.Hi)
	backright1.QuickPlotSprite(0, 0)

	backright2 := pix32.NewPix323(jagMedia, "backright2", 0)
	c.AreaBackright2 = pixmap.NewPixMap(backright2.Wi, backright2.Hi)
	backright2.QuickPlotSprite(0, 0)

	backtop1 := pix32.NewPix323(jagMedia, "backtop1", 0)
	c.AreaBacktop1 = pixmap.NewPixMap(backtop1.Wi, backtop1.Hi)
	backtop1.QuickPlotSprite(0, 0)

	backtop2 := pix32.NewPix323(jagMedia, "backtop2", 0)
	c.AreaBacktop2 = pixmap.NewPixMap(backtop2.Wi, backtop2.Hi)
	backtop2.QuickPlotSprite(0, 0)

	backvmid1 := pix32.NewPix323(jagMedia, "backvmid1", 0)
	c.AreaBackvmid1 = pixmap.NewPixMap(backvmid1.Wi, backvmid1.Hi)
	backvmid1.QuickPlotSprite(0, 0)

	backvmid2 := pix32.NewPix323(jagMedia, "backvmid2", 0)
	c.AreaBackvmid2 = pixmap.NewPixMap(backvmid2.Wi, backvmid2.Hi)
	backvmid2.QuickPlotSprite(0, 0)

	backvmid3 := pix32.NewPix323(jagMedia, "backvmid3", 0)
	c.AreaBackvmid3 = pixmap.NewPixMap(backvmid3.Wi, backvmid3.Hi)
	backvmid3.QuickPlotSprite(0, 0)

	backhmid2 := pix32.NewPix323(jagMedia, "backhmid2", 0)
	c.AreaBackhmid2 = pixmap.NewPixMap(backhmid2.Wi, backhmid2.Hi)
	backhmid2.QuickPlotSprite(0, 0)

	randomR := int(rand.Float64()*21.0) - 10
	randomG := int(rand.Float64()*21.0) - 10
	randomB := int(rand.Float64()*21.0) - 10
	random := int(rand.Float64()*41.0) - 20

	for i := range 50 {
		if c.ImageMapFunction[i] != nil {
			c.ImageMapFunction[i].RGBAdjust(randomR+random, randomG+random, randomB+random)
		}
		if c.ImageMapscene[i] != nil {
			c.ImageMapscene[i].RGBAdjust(randomR+random, randomG+random, randomB+random)
		}
	}

	c.DrawProgress("Unpacking textures", 80)
	pix3d.UnpackTextures(jagTextures)
	pix3d.SetBrightness(0.8)
	pix3d.InitPool(20)
	c.DrawProgress("Unpacking models", 83)
	model.Unpack(jagModels)
	animbase.Unpack(jagModels)
	animframe.Unpack(jagModels)
	c.DrawProgress("Unpacking config", 86)
	seqtype.Unpack(jagConfig)
	loctype.Unpack(jagConfig)
	flotype.Unpack(jagConfig)
	objtype.Unpack(jagConfig)
	npctype.Unpack(jagConfig)
	idktype.Unpack(jagConfig)
	spotanimtype.Unpack(jagConfig)
	varptype.Unpack(jagConfig)
	objtype.MembersWorld = MembersWorld
	if !LowMemory {
		c.DrawProgress("Unpacking sounds", 90)
		var20 := jagSounds.Read("sounds.dat", nil)
		var21 := io.NewPacket(var20)
		wave.Unpack(var21)
	}
	c.DrawProgress("Unpacking interfaces", 92)
	var48 := []*pixfont.PixFont{c.FontPlain11, c.FontPlain12, c.FontBold12, c.FontQuill8}
	component.Unpack(jagMedia, var48, jagInterface)
	c.DrawProgress("Preparing game engine", 97)
	for i := range 33 {
		var22 := 999
		var23 := 0
		for j := range 35 {
			if c.ImageMapback.Pixels[j+i*c.ImageMapback.Wi] == 0 {
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
			if c.ImageMapback.Pixels[j+i*c.ImageMapback.Wi] == 0 && (j > 34 || i > 34) {
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
		var50[i] = (var26 * var27) >> 16
	}
	world3d.Init(var50, 800, 512, 334, 500)
	wordfilter.Unpack(jagWordEnc)
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

func (c *Client) Draw3DEntityElements() {
	var2 := 0
	c.DrawPrivateMessages()
	if c.CrossMode == 1 {
		c.ImageCrosses[c.CrossCycle/100].PlotSprite(c.CrossY-8-11, c.CrossX-8-8)
	}
	if c.CrossMode == 2 {
		c.ImageCrosses[c.CrossCycle/100+4].PlotSprite(c.CrossY-8-11, c.CrossX-8-8)
	}
	if c.ViewportInterfaceID != -1 {
		c.UpdateInterfaceAnimation(c.ViewportInterfaceID, c.SceneDelta)
		c.DrawInterface(0, 0, component.Instances[c.ViewportInterfaceID], 0)
	}
	c.DrawWildyLevel()
	if !c.MenuVisible {
		c.HandleInput()
		c.DrawTooltip()
	} else if c.MenuArea == 0 {
		c.DrawMenu()
	}
	if c.InMultizone == 1 {
		if c.WildernessLevel > 0 || c.WorldLocationState == 1 {
			c.ImageHeadIcons[1].PlotSprite(258, 472)
		} else {
			c.ImageHeadIcons[1].PlotSprite(296, 472)
		}
	}
	if c.WildernessLevel > 0 {
		c.ImageHeadIcons[0].PlotSprite(296, 472)
		c.FontPlain12.CentreString(329, 0xFFFF00, "Level: "+strconv.Itoa(c.WildernessLevel), 484)
	}
	if c.WorldLocationState == 1 {
		c.ImageHeadIcons[6].PlotSprite(296, 472)
		c.FontPlain12.CentreString(329, 0xFFFF00, "Arena", 484)
	}
	if c.SystemUpdateTimer == 0 {
		return
	}
	var2 = c.SystemUpdateTimer / 50
	var3 := var2 / 60
	var2 %= 60
	if var2 < 10 {
		c.FontPlain12.DrawString(4, 329, 0xFFFF00, "System update in: "+strconv.Itoa(var3)+":0"+strconv.Itoa(var2))
	} else {
		c.FontPlain12.DrawString(4, 329, 0xFFFF00, "System update in: "+strconv.Itoa(var3)+":"+strconv.Itoa(var2))
	}
}

func (c *Client) UpdateOrbitCamera(arg0 int) {
	var2 := c.LocalPlayer.X + c.CameraAnticheatOffsetX
	var3 := c.LocalPlayer.Z + c.CameraAnticheatOffsetZ
	if c.OrbitCameraX-var2 < -500 || c.OrbitCameraX-var2 > 500 || c.OrbitCameraZ-var3 < -500 || c.OrbitCameraZ-var3 > 500 {
		c.OrbitCameraX = var2
		c.OrbitCameraZ = var3
	}
	if c.OrbitCameraX != var2 {
		c.OrbitCameraX += (var2 - c.OrbitCameraX) / 16
	}
	if c.OrbitCameraZ != var3 {
		c.OrbitCameraZ += (var3 - c.OrbitCameraZ) / 16
	}
	if c.ActionKey[1] == 1 {
		c.OrbitCameraYawVelocity += (-24 - c.OrbitCameraYawVelocity) / 2
	} else if c.ActionKey[2] == 1 {
		c.OrbitCameraYawVelocity += (24 - c.OrbitCameraYawVelocity) / 2
	} else {
		c.OrbitCameraYawVelocity /= 2
	}
	if c.ActionKey[3] == 1 {
		c.OrbitCameraPitchVelocity += (12 - c.OrbitCameraPitchVelocity) / 2
	} else if c.ActionKey[4] == 1 {
		c.OrbitCameraPitchVelocity += (-12 - c.OrbitCameraPitchVelocity) / 2
	} else {
		c.OrbitCameraPitchVelocity /= 2
	}
	c.OrbitCameraYaw = (c.OrbitCameraYaw + c.OrbitCameraYawVelocity/2) & 0x7FF
	c.PacketSize += arg0
	c.OrbitCameraPitch += c.OrbitCameraPitchVelocity / 2
	if c.OrbitCameraPitch < 128 {
		c.OrbitCameraPitch = 128
	}
	if c.OrbitCameraPitch > 383 {
		c.OrbitCameraPitch = 383
	}
	var4 := c.OrbitCameraX >> 7
	var5 := c.OrbitCameraZ >> 7
	var6 := c.GetHeightMapY(c.CurrentLevel, c.OrbitCameraX, c.OrbitCameraZ)
	var7 := 0
	if var4 > 3 && var5 > 3 && var4 < 100 && var5 < 100 {
		for i := var4 - 4; i <= var4+4; i++ {
			for j := var5 - 4; j <= var5+4; j++ {
				var10 := c.CurrentLevel
				if var10 < 3 && c.LevelTileFlags[1][i][j]&0x2 == 2 {
					var10++
				}
				var11 := var6 - c.LevelHeightMap[var10][i][j]
				if var11 > var7 {
					var7 = var11
				}
			}
		}
	}
	var8 := var7 * 192
	var8 = min(var8, 98048)
	var8 = max(var8, 32768)
	if var8 > c.CameraPitchClamp {
		c.CameraPitchClamp += (var8 - c.CameraPitchClamp) / 24
	} else if var8 < c.CameraPitchClamp {
		c.CameraPitchClamp += (var8 - c.CameraPitchClamp) / 80
	}
}

func (c *Client) PushProjectiles() {
	for var2 := c.Projectiles.Head(); var2 != nil; var2 = c.Projectiles.Next() {
		v := var2.Value
		if v.Level != c.CurrentLevel || clientextras.LoopCycle > v.LastCycle {
			var2.Unlink()
		} else if clientextras.LoopCycle >= v.StartCycle {
			if v.Target > 0 {
				var3 := c.NPCs[v.Target-1]
				if var3 != nil {
					v.UpdateVelocity(c.GetHeightMapY(v.Level, var3.X, var3.Z)-v.OffsetY, var3.Z, var3.X, clientextras.LoopCycle)
				}
			}
			if v.Target < 0 {
				var4 := -v.Target - 1
				var var5 *playerentity.PlayerEntity
				if var4 == c.LocalPID {
					var5 = c.LocalPlayer
				} else {
					var5 = c.Players[var4]
				}
				if var5 != nil {
					v.UpdateVelocity(c.GetHeightMapY(v.Level, var5.X, var5.Z)-v.OffsetY, var5.Z, var5.X, clientextras.LoopCycle)
				}
			}
			v.Update(c.SceneDelta)
			c.Scene.AddTemporary1(int(v.Z), 60, v.Yaw, int(v.X), -1, false, nil, v, int(v.Y), c.CurrentLevel)
		}
	}
}

func (c *Client) RefreshFunc() {
	c.RedrawFrame = true
}

func (c *Client) DrawOnMinimap(arg0 int, arg2 *pix32.Pix32, arg3 int) {
	var5 := (c.OrbitCameraYaw + c.MinimapAnticheatAngle) & 0x7FF
	var6 := arg3*arg3 + arg0*arg0
	if var6 > 6400 {
		return
	}
	var7 := model.Sin[var5]
	var8 := model.Cos[var5]
	var11 := var7 * 256 / (c.MinimapZoom + 256)
	var12 := var8 * 256 / (c.MinimapZoom + 256)
	var9 := (arg0*var11 + arg3*var12) >> 16
	var10 := (arg0*var12 - arg3*var11) >> 16
	if var6 > 2500 {
		arg2.DrawMasked(c.ImageMapback, 83-var10-arg2.OHi/2, var9+94-arg2.OWi/2)
	} else {
		arg2.PlotSprite(83-var10-arg2.OHi/2, var9+94-arg2.OWi/2)
	}
}

func (c *Client) Mix(arg0, arg1, arg2 int) int {
	var5 := 256 - arg1
	return ((((arg0&0xFF00FF)*var5 + (arg2&0xFF00FF)*arg1) & 0xFF00FF00) + (((arg0&0xFF00)*var5 + (arg2&0xFF00)*arg1) & 0xFF0000)) >> 8
}

func (c *Client) GetIntString(arg0 int) string {
	if arg0 < 999999999 {
		return strconv.Itoa(arg0)
	}
	return "*"
}

func (c *Client) ProjectFromGround1(arg0 int, arg2 *entity.PathingEntity) {
	c.ProjectFromGround2(arg2.Z, arg2.X, arg0)
}

func (c *Client) ProjectFromGround2(arg0, arg1, arg3 int) {
	if arg1 < 128 || arg0 < 128 || arg1 > 13056 || arg0 > 13056 {
		c.ProjectX = -1
		c.ProjectY = -1
		return
	}
	var5 := c.GetHeightMapY(c.CurrentLevel, arg1, arg0) - arg3
	var13 := arg1 - c.CameraX
	var14 := var5 - c.CameraY
	var11 := arg0 - c.CameraZ
	var6 := model.Sin[c.CameraPitch]
	var7 := model.Cos[c.CameraPitch]
	var8 := model.Sin[c.CameraYaw]
	var9 := model.Cos[c.CameraYaw]
	var10 := (var11*var8 + var13*var9) >> 16
	var12 := (var11*var9 - var13*var8) >> 16
	var13 = var10
	var10 = (var14*var7 - var12*var6) >> 16
	var11 = (var14*var6 + var12*var7) >> 16
	if var11 >= 50 {
		c.ProjectX = pix3d.CenterW3D + (var13<<9)/var11
		c.ProjectY = pix3d.CenterH3D + (var10<<9)/var11
	} else {
		c.ProjectX = -1
		c.ProjectY = -1
	}
}

func (c *Client) InteractWithLoc(arg0, arg1, arg2, arg3 int) bool {
	var6 := (arg3 >> 14) & 0x7FFF
	var7 := c.Scene.GetInfo(c.CurrentLevel, arg1, arg2, arg3)
	if var7 == -1 {
		return false
	}
	var8 := var7 & 0x1F
	var9 := (var7 >> 6) & 0x3
	if var8 == 10 || var8 == 11 || var8 == 22 {
		var10 := loctype.Get(var6)
		var11 := 0
		var12 := 0
		if var9 == 0 || var9 == 2 {
			var11 = var10.Width
			var12 = var10.Length
		} else {
			var11 = var10.Length
			var12 = var10.Width
		}
		var13 := var10.ForceApproach
		if var9 != 0 {
			var13 = ((var13 << var9) & 0xF) + (var13 >> (4 - var9))
		}
		c.TryMove(c.LocalPlayer.PathTileX[0], var11, false, arg1, c.LocalPlayer.PathTileZ[0], 2, var12, arg2, 0, 0, var13)
	} else {
		c.TryMove(c.LocalPlayer.PathTileX[0], 0, false, arg1, c.LocalPlayer.PathTileZ[0], 2, 0, arg2, var9, var8+1, 0)
	}
	c.CrossX = c.MouseClickX
	c.CrossY = c.MouseClickY
	c.CrossMode = 2
	c.CrossCycle = 0
	c.Out.P1Isaac(arg0)
	c.Out.P2(arg1 + c.SceneBaseTileX)
	c.Out.P2(arg2 + c.SceneBaseTileZ)
	c.Out.P2(var6)
	return true
}

func (c *Client) ShowContextMenu() {
	var2 := c.FontBold12.StringWidth("Choose Option")
	for i := range c.MenuSize {
		var4 := c.FontBold12.StringWidth(c.MenuOption[i])
		if var4 > var2 {
			var2 = var4
		}
	}
	var2 += 8
	var4 := c.MenuSize*15 + 21
	var5 := 0
	var6 := 0
	if c.MouseClickX > 8 && c.MouseClickY > 11 && c.MouseClickX < 520 && c.MouseClickY < 345 {
		var5 = c.MouseClickX - 8 - var2/2
		if var5+var2 > 512 {
			var5 = 512 - var2
		}
		if var5 < 0 {
			var5 = 0
		}
		var6 = c.MouseClickY - 11
		if var6+var4 > 334 {
			var6 = 334 - var4
		}
		if var6 < 0 {
			var6 = 0
		}
		c.MenuVisible = true
		c.MenuArea = 0
		c.MenuX = var5
		c.MenuY = var6
		c.MenuWidth = var2
		c.MenuHeight = c.MenuSize*15 + 22
	}
	if c.MouseClickX > 562 && c.MouseClickY > 231 && c.MouseClickX < 752 && c.MouseClickY < 492 {
		var5 = c.MouseClickX - 562 - var2/2
		if var5 < 0 {
			var5 = 0
		} else if var5+var2 > 190 {
			var5 = 190 - var2
		}
		var6 = c.MouseClickY - 231
		if var6 < 0 {
			var6 = 0
		} else if var6+var4 > 261 {
			var6 = 261 - var4
		}
		c.MenuVisible = true
		c.MenuArea = 1
		c.MenuX = var5
		c.MenuY = var6
		c.MenuWidth = var2
		c.MenuHeight = c.MenuSize*15 + 22
	}
	if c.MouseClickX <= 22 || c.MouseClickY <= 375 || c.MouseClickX >= 501 || c.MouseClickY >= 471 {
		return
	}
	var5 = c.MouseClickX - 22 - var2/2
	if var5 < 0 {
		var5 = 0
	} else if var5+var2 > 479 {
		var5 = 479 - var2
	}
	var6 = c.MouseClickY - 375
	if var6 < 0 {
		var6 = 0
	} else if var6+var4 > 96 {
		var6 = 96 - var4
	}
	c.MenuVisible = true
	c.MenuArea = 2
	c.MenuX = var5
	c.MenuY = var6
	c.MenuWidth = var2
	c.MenuHeight = c.MenuSize*15 + 22
}

func (c *Client) OpenURL(arg0 string) (*bytes.Reader, error) {
	// Go client is standalone; the Java applet branch (signlink.openurl) is intentionally absent.
	resp, err := http.Get(c.GetCodeBase() + "/" + arg0)
	if err != nil {
		return nil, fmt.Errorf("failed to open url: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openurl %s: HTTP %d", arg0, resp.StatusCode)
	}
	b, err := io2.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	return bytes.NewReader(b), nil
}

func (c *Client) LoadTitle() {
	// Nil the in-game pixmaps unconditionally — Java does this inside
	// the loadTitle body (deob/client.java:6661-6668) and relies on
	// imageTitle2 being nil on Logout transitions to enter. The Go
	// port keeps ImageTitle2 alive past PrepareGameScreen (0f2d815)
	// so the early-return below fires on Logout; hoisting the nils
	// out keeps them firing too. Without this, AreaViewport /
	// AreaMapback retain the last in-game frame's pixel data, and
	// re-login briefly shows the previous session's world + minimap
	// before DrawScene/DrawMinimap repaint (PrepareGameScreen also
	// early-returns when AreaChatback is non-nil, skipping the
	// re-allocation). These assignments are no-ops on boot (already
	// nil) and during continuous title-screen rendering (already nil
	// from a prior LoadTitle).
	c.AreaChatback = nil
	c.AreaMapback = nil
	c.AreaSidebar = nil
	c.AreaViewport = nil
	c.AreaBackbase1 = nil
	c.AreaBackbase2 = nil
	c.AreaBackhmid1 = nil

	if c.ImageTitle2 != nil {
		// Already loaded path. Two cases:
		//   1. Boot, before LoadTitleImages has run — first LoadTitle
		//      allocated ImageTitle0..8 but the flame buffers
		//      (FlameBuffer3 etc.) aren't ready yet. Don't start the
		//      flame goroutine; LoadTitleImages will do it once the
		//      buffers exist.
		//   2. Logout → re-enter title — both ImageTitle2 and the flame
		//      buffers stayed alive across UnloadTitle (b45b14e).
		//      Restart the flame goroutine if it isn't running.
		// `c.FlameBuffer3 != nil` distinguishes the two cases —
		// LoadTitleImages allocates FlameBuffer3 at line 3180, which
		// is the latest of the flame buffers, so its presence
		// guarantees the rest are ready too.
		if !c.FlameActive && c.FlameBuffer3 != nil {
			c.FlamesThread = true
			c.FlameActive = true
			// Direct call — see Load:5491 for the dispatch-race rationale.
			go c.RunFlames()
		}
		return
	}
	c.ImageTitle0 = pixmap.NewPixMap(128, 265)
	pix2d.Clear()
	c.ImageTitle1 = pixmap.NewPixMap(128, 265)
	pix2d.Clear()
	c.ImageTitle2 = pixmap.NewPixMap(533, 186)
	pix2d.Clear()
	c.ImageTitle3 = pixmap.NewPixMap(360, 146)
	pix2d.Clear()
	c.ImageTitle4 = pixmap.NewPixMap(360, 200)
	pix2d.Clear()
	c.ImageTitle5 = pixmap.NewPixMap(214, 267)
	pix2d.Clear()
	c.ImageTitle6 = pixmap.NewPixMap(215, 267)
	pix2d.Clear()
	c.ImageTitle7 = pixmap.NewPixMap(86, 79)
	pix2d.Clear()
	c.ImageTitle8 = pixmap.NewPixMap(87, 79)
	pix2d.Clear()
	if c.JagTitle != nil {
		c.LoadTitleBackground()
		c.LoadTitleImages()
	}
	c.RedrawFrame = true
}

func (c *Client) RunFlames() {
	c.FlameThread = true
	// Java: try { ... } catch (Exception) {} — empty swallow around the flame
	// animation loop. In Go we let any panic propagate naturally; no equivalent
	// handler is needed.
	var2 := time.Now().UnixMilli()
	var4 := 0
	var5 := 20
	for c.FlameActive {
		c.UpdateFlames()
		c.UpdateFlames()
		c.DrawFlames()
		var4++
		if var4 > 10 {
			var6 := time.Now().UnixMilli()
			var8 := int(var6-var2)/10 - var5
			var5 = 40 - var8
			var5 = max(var5, 5)
			var4 = 0
			var2 = var6
		}
		time.Sleep(time.Duration(var5) * time.Millisecond)
	}
	c.FlameThread = false
}

func (c *Client) HandleScrollInput(arg0, arg1, arg2, arg3, arg4 int, arg5 bool, arg6 int, arg7 int, arg8 *component.Component) {
	if c.ScrollGrabbed {
		c.ScrollInputPadding = 32
	} else {
		c.ScrollInputPadding = 0
	}
	c.ScrollGrabbed = false
	c.PacketSize += arg1
	if arg0 >= arg6 && arg0 < arg6+16 && arg2 >= arg7 && arg2 < arg7+16 {
		arg8.ScrollPosition -= c.DragCycles * 4
		if arg5 {
			c.RedrawSidebar = true
			return
		}
	} else if arg0 >= arg6 && arg0 < arg6+16 && arg2 >= arg7+arg4-16 && arg2 < arg7+arg4 {
		arg8.ScrollPosition += c.DragCycles * 4
		if arg5 {
			c.RedrawSidebar = true
			return
		}
	} else if arg0 >= arg6-c.ScrollInputPadding && arg0 < arg6+16+c.ScrollInputPadding && arg2 >= arg7+16 && arg2 < arg7+arg4-16 && c.DragCycles > 0 {
		var10 := (arg4 - 32) * arg4 / arg3
		var10 = max(var10, 8)
		var11 := arg2 - arg7 - 16 - var10/2
		var12 := arg4 - 32 - var10
		arg8.ScrollPosition = (arg3 - arg4) * var11 / var12
		if arg5 {
			c.RedrawSidebar = true
		}
		c.ScrollGrabbed = true
	}
}

func (c *Client) LoginFunc(arg0 string, arg1 string, arg2 bool) {
	signlink.ErrorName = arg0
	if !arg2 {
		c.LoginMessage0 = ""
		c.LoginMessage1 = "Connecting to server..."
		// Out-of-band repaint: show "Connecting to server..." before blocking
		// on the socket dial. Runs on the game goroutine, not the main loop
		// iteration, so we present explicitly.
		c.present(func() { c.DrawTitleScreen() })
	}
	conn, err := c.OpenSocket(clientextras.PortOffset + 43594)
	if err != nil {
		c.LoginMessage0 = ""
		c.LoginMessage1 = "Error connecting to server."
		return
	}
	c.Stream = clientstream.NewClientStream(conn)
	if err := c.Stream.ReadFully(c.In.Data, 0, 8); err != nil {
		c.LoginMessage0 = ""
		c.LoginMessage1 = "Error connecting to server."
		return
	}
	c.In.Pos = 0
	c.ServerSeed = c.In.G8()
	var4 := [4]int{int(rand.Float64() * 9.9999999e7), int(rand.Float64() * 9.9999999e7), int(c.ServerSeed >> 32), int(c.ServerSeed)}
	c.Out.Pos = 0
	c.Out.P1(10)
	c.Out.P4(var4[0])
	c.Out.P4(var4[1])
	c.Out.P4(var4[2])
	c.Out.P4(var4[3])
	c.Out.P4(signlink.UID)
	c.Out.PJStr(arg0)
	c.Out.PJStr(arg1)
	c.Out.RSAEnc(RSA_MODULUS, RSA_EXPONENT)
	c.Login.Pos = 0
	if arg2 {
		c.Login.P1(18)
	} else {
		c.Login.P1(16)
	}
	c.Login.P1(c.Out.Pos + 36 + 1 + 1)
	c.Login.P1(225)
	if LowMemory {
		c.Login.P1(1)
	} else {
		c.Login.P1(0)
	}
	for i := range 9 {
		c.Login.P4(c.JagChecksum[i])
	}
	c.Login.PData(c.Out.Data, c.Out.Pos, 0)
	c.Out.Random = io.NewIsaac(var4)
	for i := range 4 {
		var4[i] += 50
	}
	c.RandomIn = io.NewIsaac(var4)
	if err := c.Stream.Write(c.Login.Data, c.Login.Pos, 0); err != nil {
		c.LoginMessage0 = ""
		c.LoginMessage1 = "Error connecting to server."
		return
	}
	var7, err := c.Stream.Read()
	if err != nil {
		c.LoginMessage0 = ""
		c.LoginMessage1 = "Error connecting to server."
		return
	}
	if var7 == 1 {
		time.Sleep(2000 * time.Millisecond)
		c.LoginFunc(arg0, arg1, arg2)
		return
	}
	if var7 == 2 || var7 == 18 {
		if var7 == 18 {
			c.Rights = true
		} else {
			c.Rights = false
		}
		inputtracking.SetDisabled()
		c.InGame = true
		c.Out.Pos = 0
		c.In.Pos = 0
		c.PacketType = -1
		c.LastPacketType0 = -1
		c.LastPacketType1 = -1
		c.LastPacketType2 = -1
		c.PacketSize = 0
		c.IdleNetCycles = 0
		c.SystemUpdateTimer = 0
		c.IdleTimeout = 0
		c.HintType = 0
		c.MenuSize = 0
		c.MenuVisible = false
		c.IdleCycles = 0
		for i := range 100 {
			c.MessageText[i] = ""
		}
		c.ObjSelected = 0
		c.SpellSelected = 0
		c.SceneState = 0
		c.WaveCount = 0
		c.CameraAnticheatOffsetX = int(rand.Float64()*100.0) - 50
		c.CameraAnticheatOffsetZ = int(rand.Float64()*110.0) - 55
		c.CameraAnticheatAngle = int(rand.Float64()*80.0) - 40
		c.MinimapAnticheatAngle = int(rand.Float64()*120.0) - 60
		c.MinimapZoom = int(rand.Float64()*30.0) - 20
		c.OrbitCameraYaw = (int(rand.Float64()*20.0) - 10) & 0x7FF
		c.MinimapLevel = -1
		c.FlagSceneTileX = 0
		c.FlagSceneTileZ = 0
		c.PlayerCount = 0
		c.NPCCount = 0
		for i := range c.MAX_PLAYER_COUNT {
			c.Players[i] = nil
			c.PlayerAppearanceBuffer[i] = nil
		}
		for i := range 8192 {
			c.NPCs[i] = nil
		}
		c.Players[c.LOCAL_PLAYER_INDEX] = playerentity.NewPlayerEntity()
		c.LocalPlayer = c.Players[c.LOCAL_PLAYER_INDEX]
		c.Projectiles.Clear()
		c.Spotanims.Clear()
		c.MergedLocations.Clear()
		for i := range 4 {
			for j := range 104 {
				for k := range 104 {
					c.LevelObjStacks[i][j][k] = nil
				}
			}
		}
		c.SpawnedLocations = datastruct.NewLinkList[*entity.LocAddEntity]()
		c.FriendCount = 0
		c.StickyChatInterfaceID = -1
		c.ChatInterfaceID = -1
		c.ViewportInterfaceID = -1
		c.SidebarInterfaceID = -1
		c.PressedContinueOption = false
		c.SelectedTab = 3
		c.ChatbackInputOpen = false
		c.MenuVisible = false
		c.ShowSocialInput = false
		c.ModalMessage = ""
		c.InMultizone = 0
		c.FlashingTab = -1
		c.DesignGenderMale = true
		c.ValidateCharacterDesign()
		for i := range 5 {
			c.DesignColors[i] = 0
		}
		OpLogic1 = 0
		OpLogic2 = 0
		OpLogic3 = 0
		OpLogic4 = 0
		OpLogic5 = 0
		OpLogic6 = 0
		OpLogic7 = 0
		OpLogic8 = 0
		OpLogic9 = 0
		// Java: deob/client.java:6915 — `field1382 = 0`. Intentionally
		// not ported: field1382 is a deobfuscator artifact (assigned
		// once, never read). Project policy excludes pure deob state.
		c.PrepareGameScreen()
		return
	}
	if var7 == 3 {
		c.LoginMessage0 = ""
		c.LoginMessage1 = "Invalid username or password."
		return
	}
	if var7 == 4 {
		c.LoginMessage0 = "Your account has been disabled."
		c.LoginMessage1 = "Please check your message-centre for details."
		return
	}
	if var7 == 5 {
		c.LoginMessage0 = "Your account is already logged in."
		c.LoginMessage1 = "Try again in 60 secs..."
		return
	}
	if var7 == 6 {
		c.LoginMessage0 = "RuneScape has been updated!"
		c.LoginMessage1 = "Please reload this page."
		return
	}
	if var7 == 7 {
		c.LoginMessage0 = "This world is full."
		c.LoginMessage1 = "Please use a different world."
		return
	}
	if var7 == 8 {
		c.LoginMessage0 = "Unable to connect."
		c.LoginMessage1 = "Login server offline."
		return
	}
	if var7 == 9 {
		c.LoginMessage0 = "Login limit exceeded."
		c.LoginMessage1 = "Too many connections from your address."
		return
	}
	if var7 == 10 {
		c.LoginMessage0 = "Unable to connect."
		c.LoginMessage1 = "Bad session id."
		return
	}
	if var7 == 11 {
		c.LoginMessage1 = "Login server rejected session."
		c.LoginMessage1 = "Please try again."
		return
	}
	if var7 == 12 {
		c.LoginMessage0 = "You need a members account to login to this world."
		c.LoginMessage1 = "Please subscribe, or use a different world."
		return
	}
	if var7 == 13 {
		c.LoginMessage0 = "Could not complete login."
		c.LoginMessage1 = "Please try using a different world."
		return
	}
	if var7 == 14 {
		c.LoginMessage0 = "The server is being updated."
		c.LoginMessage1 = "Please wait 1 minute and try again."
		return
	}
	if var7 == 15 {
		c.InGame = true
		c.Out.Pos = 0
		c.In.Pos = 0
		c.PacketType = -1
		c.LastPacketType0 = -1
		c.LastPacketType1 = -1
		c.LastPacketType2 = -1
		c.PacketSize = 0
		c.IdleNetCycles = 0
		c.SystemUpdateTimer = 0
		c.MenuSize = 0
		c.MenuVisible = false
		return
	}
	if var7 == 16 {
		c.LoginMessage0 = "Login attempts exceeded."
		c.LoginMessage1 = "Please wait 1 minute and try again."
		return
	}
	if var7 == 17 {
		c.LoginMessage0 = "You are standing in a members-only area."
		c.LoginMessage1 = "To play on this world move to a free area first"
		return
	}
}

func (c *Client) AddLoc(arg0, arg1, arg2, arg3, arg4, arg5, arg7 int) {
	if arg1 < 1 || arg2 < 1 || arg1 > 102 || arg2 > 102 {
		return
	}
	if LowMemory && arg7 != c.CurrentLevel {
		return
	}
	var9 := 0
	if arg3 == 0 {
		var9 = c.Scene.GetWallBitSet(arg7, arg1, arg2)
	}
	if arg3 == 1 {
		var9 = c.Scene.GetWallDecorationBitSet(arg7, arg2, arg1)
	}
	if arg3 == 2 {
		var9 = c.Scene.GetLocBitSet(arg7, arg1, arg2)
	}
	if arg3 == 3 {
		var9 = c.Scene.GetGroundDecorationBitSet(arg7, arg1, arg2)
	}
	if var9 != 0 {
		var13 := c.Scene.GetInfo(arg7, arg1, arg2, var9)
		var15 := (var9 >> 14) & 0x7FFF
		var16 := var13 & 0x1F
		var17 := var13 >> 6
		var var14 *loctype.LocType
		if arg3 == 0 {
			c.Scene.RemoveWall(arg1, arg7, arg2)
			var14 = loctype.Get(var15)
			if var14.BlockWalk {
				c.LevelCollisionMap[arg7].DelWall(var14.BlockRange, var17, arg1, arg2, var16)
			}
		}
		if arg3 == 1 {
			c.Scene.RemoveWallDecoration(arg7, arg2, arg1)
		}
		if arg3 == 2 {
			c.Scene.RemoveLoc2(arg1, arg2, arg7)
			var14 = loctype.Get(var15)
			if arg1+var14.Width > 103 || arg2+var14.Width > 103 || arg1+var14.Length > 103 || arg2+var14.Length > 103 {
				return
			}
			if var14.BlockWalk {
				c.LevelCollisionMap[arg7].DelLoc(arg2, arg1, var17, var14.Width, var14.BlockRange, var14.Length)
			}
		}
		if arg3 == 3 {
			c.Scene.RemoveGroundDecoration(arg7, arg1, arg2)
			var14 = loctype.Get(var15)
			if var14.BlockWalk && var14.Active {
				c.LevelCollisionMap[arg7].RemoveBlocked(arg2, arg1)
			}
		}
	}
	if arg4 < 0 {
		return
	}
	var13 := arg7
	if arg7 < 3 && c.LevelTileFlags[1][arg1][arg2]&0x2 == 2 {
		var13 = arg7 + 1
	}
	world.AddLoc(arg1, c.LocList, c.LevelCollisionMap[arg7], arg2, arg0, c.LevelHeightMap, arg7, arg4, arg5, c.Scene, var13)
}

func (c *Client) AddFriend(arg0 int64) {
	if arg0 == 0 {
		return
	}
	if c.FriendCount >= 100 {
		c.AddMessage(0, "Your friends list is full. Max of 100 hit", "")
		return
	}
	var4 := jstring.FormatName(jstring.FromBase37(arg0))
	for i := range c.FriendCount {
		if c.FriendName37[i] == arg0 {
			c.AddMessage(0, var4+" is already on your friend list", "")
			return
		}
	}
	for i := range c.IgnoreCount {
		if c.IgnoreName37[i] == arg0 {
			c.AddMessage(0, "Please remove "+var4+" from your ignore list first", "")
			return
		}
	}
	if var4 == c.LocalPlayer.Name {
		return
	}
	c.FriendName[c.FriendCount] = var4
	c.FriendName37[c.FriendCount] = arg0
	c.FriendWorld[c.FriendCount] = 0
	c.FriendCount++
	c.RedrawSidebar = true
	c.Out.P1Isaac(118)
	c.Out.P8(arg0)
}

func (c *Client) Unload() {
	signlink.ReportError = false
	if c.Stream != nil {
		c.Stream.Close()
	}
	c.Stream = nil
	c.StopMidi()
	c.MidiThreadActive = false
	c.Out = nil
	c.Login = nil
	c.In = nil
	c.SceneMapIndex = nil
	c.SceneMapLandData = nil
	c.SceneMapLocData = nil
	c.LevelHeightMap = nil
	c.LevelTileFlags = nil
	c.Scene = nil
	c.LevelCollisionMap = nil
	c.BFSDirection = nil
	c.BFSCost = nil
	c.BFSStepX = nil
	c.BFSStepZ = nil
	c.TextureBuffer = nil
	c.AreaSidebar = nil
	c.AreaMapback = nil
	c.AreaViewport = nil
	c.AreaChatback = nil
	c.AreaBackbase1 = nil
	c.AreaBackbase2 = nil
	c.AreaBackhmid1 = nil
	c.AreaBackleft1 = nil
	c.AreaBackleft2 = nil
	c.AreaBackright1 = nil
	c.AreaBackright2 = nil
	c.AreaBacktop1 = nil
	c.AreaBacktop2 = nil
	c.AreaBackvmid1 = nil
	c.AreaBackvmid2 = nil
	c.AreaBackvmid3 = nil
	c.AreaBackhmid2 = nil
	c.ImageInvback = nil
	c.ImageMapback = nil
	c.ImageChatback = nil
	c.ImageBackbase1 = nil
	c.ImageBackbase2 = nil
	c.ImageBackhmid1 = nil
	c.ImageSideIcons = nil
	c.ImageRedstone1 = nil
	c.ImageRedstone2 = nil
	c.ImageRedstone3 = nil
	c.ImageRedstone1h = nil
	c.ImageRedstone2h = nil
	c.ImageRedstone1v = nil
	c.ImageRedstone2v = nil
	c.ImageRedstone3v = nil
	c.ImageRedstone1hv = nil
	c.ImageRedstone2hv = nil
	c.ImageCompass = nil
	c.ImageHitmarks = nil
	c.ImageHeadIcons = nil
	c.ImageCrosses = nil
	c.ImageMapdot0 = nil
	c.ImageMapdot1 = nil
	c.ImageMapdot2 = nil
	c.ImageMapdot3 = nil
	c.ImageMapscene = nil
	c.ImageMapFunction = nil
	c.TileLastOccupiedCycle = nil
	c.Players = nil
	c.PlayerIDs = nil
	c.EntityUpdateIDs = nil
	c.PlayerAppearanceBuffer = nil
	c.EntityRemovalIDs = nil
	c.NPCs = nil
	c.NPCIDs = nil
	c.LevelObjStacks = nil
	c.SpawnedLocations = nil
	c.MergedLocations = nil
	c.Projectiles = nil
	c.Spotanims = nil
	c.LocList = nil
	c.MenuParamB = nil
	c.MenuParamC = nil
	c.MenuAction = nil
	c.MenuParamA = nil
	c.MenuOption = nil
	c.Varps = nil
	c.ActiveMapFunctionX = nil
	c.ActiveMapFunctionZ = nil
	c.ActiveMapFunctions = nil
	c.ImageMinimap = nil
	c.FriendName = nil
	c.FriendName37 = nil
	c.FriendWorld = nil
	c.ImageTitle0 = nil
	c.ImageTitle1 = nil
	c.ImageTitle2 = nil
	c.ImageTitle3 = nil
	c.ImageTitle4 = nil
	c.ImageTitle5 = nil
	c.ImageTitle6 = nil
	c.ImageTitle7 = nil
	c.ImageTitle8 = nil
	c.UnloadTitle()
	loctype.Unload()
	npctype.Unload()
	objtype.Unload()
	flotype.Instances = nil
	idktype.Instances = nil
	component.Instances = nil
	// Java: deob/client.java:7227 — `class61.instances = null`.
	// Intentionally not ported: class61 is a deobfuscator-emitted
	// stub class (one static array, no behavior, only this nilling
	// call site). Project policy excludes pure deob artifacts.
	seqtype.Instances = nil
	spotanimtype.Instances = nil
	spotanimtype.ModelCache = nil
	varptype.Instances = nil
	playerentity.ModelCache = nil
	pix3d.Unload()
	world3d.Unload()
	model.Unload()
	animbase.Instances = nil
	animframe.Instances = nil
}

// OpenSocket dials the game server on the given port.
//
// Java: openSocket (deob/client.java:7243-7245). The Java version branches on
// signlink.mainapp: standalone clients dial directly via Socket, applet
// clients delegate to the privileged signlink.opensocket polling path. Go is
// always standalone (signlink.mainapp is always nil), so both branches
// collapse to a single delegation to signlink.OpenSocket.
func (c *Client) OpenSocket(port int) (net.Conn, error) {
	return signlink.OpenSocket(port)
}

func (c *Client) AddPlayerOptions(arg1 int, arg2 int, arg3 *playerentity.PlayerEntity, arg4 int) {
	if arg3 == c.LocalPlayer || c.MenuSize >= 400 {
		return
	}
	var6 := arg3.Name + GetCombatLevelColorTag(c.LocalPlayer.CombatLevel, arg3.CombatLevel) + " (level-" + strconv.Itoa(arg3.CombatLevel) + ")"
	if c.ObjSelected == 1 {
		c.MenuOption[c.MenuSize] = "Use " + c.ObjSelectedName + " with @whi@" + var6
		c.MenuAction[c.MenuSize] = 367
		c.MenuParamA[c.MenuSize] = arg2
		c.MenuParamB[c.MenuSize] = arg4
		c.MenuParamC[c.MenuSize] = arg1
		c.MenuSize++
	} else if c.SpellSelected != 1 {
		c.MenuOption[c.MenuSize] = "Follow @whi@" + var6
		c.MenuAction[c.MenuSize] = 1544
		c.MenuParamA[c.MenuSize] = arg2
		c.MenuParamB[c.MenuSize] = arg4
		c.MenuParamC[c.MenuSize] = arg1
		c.MenuSize++
		if c.OverrideChat == 0 {
			c.MenuOption[c.MenuSize] = "Trade with @whi@" + var6
			c.MenuAction[c.MenuSize] = 1373
			c.MenuParamA[c.MenuSize] = arg2
			c.MenuParamB[c.MenuSize] = arg4
			c.MenuParamC[c.MenuSize] = arg1
			c.MenuSize++
		}
		if c.WildernessLevel > 0 {
			c.MenuOption[c.MenuSize] = "Attack @whi@" + var6
			if c.LocalPlayer.CombatLevel >= arg3.CombatLevel {
				c.MenuAction[c.MenuSize] = 151
			} else {
				c.MenuAction[c.MenuSize] = 2151
			}
			c.MenuParamA[c.MenuSize] = arg2
			c.MenuParamB[c.MenuSize] = arg4
			c.MenuParamC[c.MenuSize] = arg1
			c.MenuSize++
		}
		if c.WorldLocationState == 1 {
			c.MenuOption[c.MenuSize] = "Fight @whi@" + var6
			c.MenuAction[c.MenuSize] = 151
			c.MenuParamA[c.MenuSize] = arg2
			c.MenuParamB[c.MenuSize] = arg4
			c.MenuParamC[c.MenuSize] = arg1
			c.MenuSize++
		}
		if c.WorldLocationState == 2 {
			c.MenuOption[c.MenuSize] = "Duel-with @whi@" + var6
			c.MenuAction[c.MenuSize] = 1101
			c.MenuParamA[c.MenuSize] = arg2
			c.MenuParamB[c.MenuSize] = arg4
			c.MenuParamC[c.MenuSize] = arg1
			c.MenuSize++
		}
	} else if c.ActiveSpellFlags&0x8 == 8 {
		c.MenuOption[c.MenuSize] = c.SpellCaption + " @whi@" + var6
		c.MenuAction[c.MenuSize] = 651
		c.MenuParamA[c.MenuSize] = arg2
		c.MenuParamB[c.MenuSize] = arg4
		c.MenuParamC[c.MenuSize] = arg1
		c.MenuSize++
	}
	for i := range c.MenuSize {
		if c.MenuAction[i] == 660 {
			c.MenuOption[i] = "Walk here @whi@" + var6
			return
		}
	}
}

func (c *Client) UpdateGame() {
	if c.SystemUpdateTimer > 1 {
		c.SystemUpdateTimer--
	}
	if c.IdleTimeout > 0 {
		c.IdleTimeout--
	}
	for i := 0; i < 5 && c.Read(); i++ {
	}
	if !c.InGame {
		return
	}
	for i := 0; i < c.WaveCount; i++ {
		if c.WaveDelay[i] <= 0 {
			var4 := false
			// Java: try { ... } catch (Exception var10) {} (client.java:7336-7353)
			// — a per-wave audio exception (Wave.Generate/SaveWave/ReplayWave) is
			// silently swallowed so one bad sound can't crash the game loop. The
			// var4 retry flag (captured by reference) and the wave-removal logic
			// below stay OUTSIDE the guard, matching Java.
			func() {
				defer func() { _ = recover() }()
				if c.WaveIDs[i] != c.LastWaveID || c.WaveLoops[i] != c.LastWaveLoops {
					var5 := wave.Generate(c.WaveLoops[i], c.WaveIDs[i])
					if time.Now().UnixMilli()+int64(var5.Pos/22) > c.LastWaveStartTime+int64(c.LastWaveLength/22) {
						c.LastWaveLength = var5.Pos
						c.LastWaveStartTime = time.Now().UnixMilli()
						if c.SaveWave(var5.Data, var5.Pos) {
							c.LastWaveID = c.WaveIDs[i]
							c.LastWaveLoops = c.WaveLoops[i]
						} else {
							var4 = true
						}
					}
				} else if !c.ReplayWave() {
					var4 = true
				}
			}()
			if var4 && c.WaveDelay[i] != -5 {
				c.WaveDelay[i] = -5
			} else {
				c.WaveCount--
				for j := i; j < c.WaveCount; j++ {
					c.WaveIDs[j] = c.WaveIDs[j+1]
					c.WaveLoops[j] = c.WaveLoops[j+1]
					c.WaveDelay[j] = c.WaveDelay[j+1]
				}
				i--
			}
		} else {
			c.WaveDelay[i]--
		}
	}
	if c.NextMusicDelay > 0 {
		c.NextMusicDelay -= 20
		if c.NextMusicDelay < 0 {
			c.NextMusicDelay = 0
		}
		if c.NextMusicDelay == 0 && c.MidiActive && !LowMemory {
			c.SetMidi(c.MidiCRC, c.CurrentMidi, c.MidiSize)
		}
	}
	var11 := inputtracking.Flush()
	if var11 != nil {
		c.Out.P1Isaac(81)
		c.Out.P2(var11.Pos)
		c.Out.PData(var11.Data, var11.Pos, 0)
		var11.Release()
	}
	c.IdleNetCycles++
	if c.IdleNetCycles > 750 {
		c.TryReconnect()
	}
	c.UpdatePlayers()
	c.UpdateNpcs()
	c.UpdateEntityChats()
	c.UpdateMergeLocs()
	if c.ActionKey[1] == 1 || c.ActionKey[2] == 1 || c.ActionKey[3] == 1 || c.ActionKey[4] == 1 {
		// Java: cameraMovedWrite++ > 5 — post-increment returns the pre-
		// increment value to the comparator, then bumps the counter; the
		// increment only happens when an arrow key is held.
		origCMW := c.CameraMovedWrite
		c.CameraMovedWrite++
		if origCMW > 5 {
			c.CameraMovedWrite = 0
			c.Out.P1Isaac(189)
			c.Out.P2(c.OrbitCameraPitch)
			c.Out.P2(c.OrbitCameraYaw)
			c.Out.P1(c.MinimapAnticheatAngle)
			c.Out.P1(c.MinimapZoom)
		}
	}
	c.SceneDelta++
	if c.CrossMode != 0 {
		c.CrossCycle += 20
		if c.CrossCycle >= 400 {
			c.CrossMode = 0
		}
	}
	if c.SelectedArea != 0 {
		c.SelectedCycle++
		if c.SelectedCycle >= 15 {
			if c.SelectedArea == 2 {
				c.RedrawSidebar = true
			}
			if c.SelectedArea == 3 {
				c.RedrawChatback = true
			}
			c.SelectedArea = 0
		}
	}
	var6 := 0
	if c.ObjDragArea != 0 {
		c.ObjDragCycles++
		if c.MouseX > c.ObjGrabX+5 || c.MouseX < c.ObjGrabX-5 || c.MouseY > c.ObjGrabY+5 || c.MouseY < c.ObjGrabY-5 {
			c.ObjGrabThreshold = true
		}
		if c.MouseButton == 0 {
			if c.ObjDragArea == 2 {
				c.RedrawSidebar = true
			}
			if c.ObjDragArea == 3 {
				c.RedrawChatback = true
			}
			c.ObjDragArea = 0
			if c.ObjGrabThreshold && c.ObjDragCycles >= 5 {
				c.HoveredSlotParentID = -1
				c.HandleInput()
				if c.HoveredSlotParentID == c.ObjDragInterfaceID && c.HoveredSlot != c.ObjDragSlot {
					var13 := component.Instances[c.ObjDragInterfaceID]
					var6 = var13.InvSlotObjId[c.HoveredSlot]
					var13.InvSlotObjId[c.HoveredSlot] = var13.InvSlotObjId[c.ObjDragSlot]
					var13.InvSlotObjId[c.ObjDragSlot] = var6
					var14 := var13.InvSlotObjCount[c.HoveredSlot]
					var13.InvSlotObjCount[c.HoveredSlot] = var13.InvSlotObjCount[c.ObjDragSlot]
					var13.InvSlotObjCount[c.ObjDragSlot] = var14
					c.Out.P1Isaac(159)
					c.Out.P2(c.ObjDragInterfaceID)
					c.Out.P2(c.ObjDragSlot)
					c.Out.P2(c.HoveredSlot)
				}
			} else if (c.MouseButtonsOption == 1 || c.IsAddFriendOption(c.MenuSize-1)) && c.MenuSize > 2 {
				c.ShowContextMenu()
			} else if c.MenuSize > 0 {
				c.UseMenuOption(c.MenuSize - 1)
			}
			c.SelectedCycle = 10
			c.MouseClickButton = 0
		}
	}
	CycleLogic3++
	if CycleLogic3 > 127 {
		CycleLogic3 = 0
		c.Out.P1Isaac(215)
		c.Out.P3(4991788)
	}
	if world3d.ClickTileX != -1 {
		var12 := world3d.ClickTileX
		var6 = world3d.ClickTileZ
		var7 := c.TryMove(c.LocalPlayer.PathTileX[0], 0, true, var12, c.LocalPlayer.PathTileZ[0], 0, 0, var6, 0, 0, 0)
		world3d.ClickTileX = -1
		if var7 {
			c.CrossX = c.MouseClickX
			c.CrossY = c.MouseClickY
			c.CrossMode = 1
			c.CrossCycle = 0
		}
	}
	if c.MouseClickButton == 1 && c.ModalMessage != "" {
		c.ModalMessage = ""
		c.RedrawChatback = true
		c.MouseClickButton = 0
	}
	c.HandleMouseInput()
	c.HandleMinimapInput()
	c.HandleTabInput()
	c.HandleChatSettingsInput(0)
	if c.MouseButton == 1 || c.MouseClickButton == 1 {
		c.DragCycles++
	}
	if c.SceneState == 2 {
		c.UpdateOrbitCamera(0)
	}
	if c.SceneState == 2 && c.Cutscene {
		c.ApplyCutscene()
	}
	for i := range 5 {
		c.CameraModifierCycle[i]++
	}
	c.HandleInputKey()
	c.IdleCycles++
	if c.IdleCycles > 4500 {
		c.IdleTimeout = 250
		c.IdleCycles -= 500
		c.Out.P1Isaac(70)
	}
	c.CameraOffsetCycle++
	if c.CameraOffsetCycle > 500 {
		c.CameraOffsetCycle = 0
		var6 = int(rand.Float64() * 8.0)
		if var6&0x1 == 1 {
			c.CameraAnticheatOffsetX += c.CameraOffsetXModifier
		}
		if var6&0x2 == 2 {
			c.CameraAnticheatOffsetZ += c.CameraOffsetZModifier
		}
		if var6&0x4 == 4 {
			c.CameraAnticheatAngle += c.CameraOffsetYawModifier
		}
	}
	if c.CameraAnticheatOffsetX < -50 {
		c.CameraOffsetXModifier = 2
	}
	if c.CameraAnticheatOffsetX > 50 {
		c.CameraOffsetXModifier = -2
	}
	if c.CameraAnticheatOffsetZ < -55 {
		c.CameraOffsetZModifier = 2
	}
	if c.CameraAnticheatOffsetZ > 55 {
		c.CameraOffsetZModifier = -2
	}
	if c.CameraAnticheatAngle < -40 {
		c.CameraOffsetYawModifier = 1
	}
	// Java: deob/client.java:7534 — `> 40`, symmetric with the `< -40` lower bound.
	if c.CameraAnticheatAngle > 40 {
		c.CameraOffsetYawModifier = -1
	}
	c.MinimapOffsetCycle++
	if c.MinimapOffsetCycle > 500 {
		c.MinimapOffsetCycle = 0
		var6 = int(rand.Float64() * 8.0)
		if var6&0x1 == 1 {
			c.MinimapAnticheatAngle += c.MinimapAngleModifier
		}
		if var6&0x2 == 2 {
			c.MinimapZoom += c.MinimapZoomModifier
		}
	}
	if c.MinimapAnticheatAngle < -60 {
		c.MinimapAngleModifier = 2
	}
	if c.MinimapAnticheatAngle > 60 {
		c.MinimapAngleModifier = -2
	}
	if c.MinimapZoom < -20 {
		c.MinimapZoomModifier = 1
	}
	if c.MinimapZoom > 10 {
		c.MinimapZoomModifier = -1
	}
	CycleLogic4++
	if CycleLogic4 > 110 {
		CycleLogic4 = 0
		c.Out.P1Isaac(236)
		c.Out.P4(0)
	}
	c.HeartbeatTimer++
	if c.HeartbeatTimer > 50 {
		c.Out.P1Isaac(108)
	}
	if c.Stream != nil && c.Out.Pos > 0 {
		// Java: try { stream.write(...); out.pos = 0; heartbeatTimer = 0; }
		//   catch (IOException) { tryReconnect(); }
		//   catch (Exception)   { logout(); }
		// (client.java:7569-7580). ClientStream.Write returns a single untyped
		// error for the IOException arm (-> TryReconnect); a genuine runtime
		// panic maps to Java's catch (Exception) -> logout via this recover.
		func() {
			defer func() {
				if recover() != nil {
					c.Logout()
				}
			}()
			if err := c.Stream.Write(c.Out.Data, c.Out.Pos, 0); err != nil {
				c.TryReconnect()
			} else {
				c.Out.Pos = 0
				c.HeartbeatTimer = 0
			}
		}()
	}
}

func (c *Client) DrawTooltip() {
	if c.MenuSize < 2 && c.ObjSelected == 0 && c.SpellSelected == 0 {
		return
	}
	var2 := ""
	if c.ObjSelected == 1 && c.MenuSize < 2 {
		var2 = "Use " + c.ObjSelectedName + " with..."
	} else if c.SpellSelected == 1 && c.MenuSize < 2 {
		var2 = c.SpellCaption + "..."
	} else {
		var2 = c.MenuOption[c.MenuSize-1]
	}
	if c.MenuSize > 2 {
		var2 = var2 + "@whi@ / " + strconv.Itoa(c.MenuSize-2) + " more options"
	}
	c.FontBold12.DrawStringTooltip(clientextras.LoopCycle/1000, true, 15, 0xFFFFFF, var2, 4)
}

func (c *Client) PushSpotanims() {
	for var2 := c.Spotanims.Head(); var2 != nil; var2 = c.Spotanims.Next() {
		v := var2.Value
		if v.Level != c.CurrentLevel || v.SeqComplete {
			var2.Unlink()
		} else if clientextras.LoopCycle >= v.StartCycle {
			v.Update(c.SceneDelta)
			if v.SeqComplete {
				var2.Unlink()
			} else {
				c.Scene.AddTemporary1(v.Z, 60, 0, v.X, -1, false, nil, v, v.Y, v.Level)
			}
		}
	}
}

func (c *Client) GetCodeBase() string {
	// Java: getCodeBase() (deob/client.java:7618-7628) — applet API. The URL is
	// platform-specific (see codebase_native.go / codebase_js.go): the native
	// standalone build synthesizes http://<host>:<portOffset+8888> (Java's
	// frame!=null STANDALONE branch), while the js/wasm browser build returns
	// the page's own origin so cache fetches are same-origin — matching the
	// applet's document-base semantics (frame==null branch) and the Client-TS
	// relative-path fetches, and pairing with signlink.ConfigureTransport, which
	// derives the WebSocket target from the same window.location.
	return codeBaseURL()
}

func SetHighMem() {
	world3d.LowMemory = false
	pix3d.LowDetail = false
	LowMemory = false
	world.LowMemory = false
}

func (c *Client) TryMove(arg0, arg1 int, arg2 bool, arg3, arg4, arg6, arg7, arg8, arg9, arg10, arg11 int) bool {
	var13 := 104
	var14 := 104
	for i := range var13 {
		for j := range var14 {
			c.BFSDirection[i][j] = 0
			c.BFSCost[i][j] = 99999999
		}
	}
	var16 := arg0
	var17 := arg4
	c.BFSDirection[arg0][arg4] = 99
	c.BFSCost[arg0][arg4] = 0
	var18 := 0
	var19 := 0
	c.BFSStepX[var18] = arg0
	var28 := var18 + 1
	c.BFSStepZ[var18] = arg4
	var20 := false
	var21 := len(c.BFSStepX)
	var22 := c.LevelCollisionMap[c.CurrentLevel].Flags
	var23 := 0
	for var19 != var28 {
		var16 = c.BFSStepX[var19]
		var17 = c.BFSStepZ[var19]
		var19 = (var19 + 1) % var21
		if var16 == arg3 && var17 == arg8 {
			var20 = true
			break
		}
		if arg10 != 0 {
			if (arg10 < 5 || arg10 == 10) && c.LevelCollisionMap[c.CurrentLevel].TestWall(arg9, arg8, arg10-1, var17, arg3, var16) {
				var20 = true
				break
			}
			if arg10 < 10 && c.LevelCollisionMap[c.CurrentLevel].TestWDecor(arg9, arg10-1, var16, arg3, var17, arg8) {
				var20 = true
				break
			}
		}
		if arg1 != 0 && arg7 != 0 && c.LevelCollisionMap[c.CurrentLevel].TestLoc(var17, arg7, var16, arg3, arg11, arg8, arg1) {
			var20 = true
			break
		}
		var23 = c.BFSCost[var16][var17] + 1
		if var16 > 0 && c.BFSDirection[var16-1][var17] == 0 && var22[var16-1][var17]&0x280108 == 0 {
			c.BFSStepX[var28] = var16 - 1
			c.BFSStepZ[var28] = var17
			var28 = (var28 + 1) % var21
			c.BFSDirection[var16-1][var17] = 2
			c.BFSCost[var16-1][var17] = var23
		}
		if var16 < var13-1 && c.BFSDirection[var16+1][var17] == 0 && var22[var16+1][var17]&0x280180 == 0 {
			c.BFSStepX[var28] = var16 + 1
			c.BFSStepZ[var28] = var17
			var28 = (var28 + 1) % var21
			c.BFSDirection[var16+1][var17] = 8
			c.BFSCost[var16+1][var17] = var23
		}
		if var17 > 0 && c.BFSDirection[var16][var17-1] == 0 && var22[var16][var17-1]&0x280102 == 0 {
			c.BFSStepX[var28] = var16
			c.BFSStepZ[var28] = var17 - 1
			var28 = (var28 + 1) % var21
			c.BFSDirection[var16][var17-1] = 1
			c.BFSCost[var16][var17-1] = var23
		}
		if var17 < var14-1 && c.BFSDirection[var16][var17+1] == 0 && var22[var16][var17+1]&0x280120 == 0 {
			c.BFSStepX[var28] = var16
			c.BFSStepZ[var28] = var17 + 1
			var28 = (var28 + 1) % var21
			c.BFSDirection[var16][var17+1] = 4
			c.BFSCost[var16][var17+1] = var23
		}
		if var16 > 0 && var17 > 0 && c.BFSDirection[var16-1][var17-1] == 0 && var22[var16-1][var17-1]&0x28010E == 0 && var22[var16-1][var17]&0x280108 == 0 && var22[var16][var17-1]&0x280102 == 0 {
			c.BFSStepX[var28] = var16 - 1
			c.BFSStepZ[var28] = var17 - 1
			var28 = (var28 + 1) % var21
			c.BFSDirection[var16-1][var17-1] = 3
			c.BFSCost[var16-1][var17-1] = var23
		}
		if var16 < var13-1 && var17 > 0 && c.BFSDirection[var16+1][var17-1] == 0 && var22[var16+1][var17-1]&0x280183 == 0 && var22[var16+1][var17]&0x280180 == 0 && var22[var16][var17-1]&0x280102 == 0 {
			c.BFSStepX[var28] = var16 + 1
			c.BFSStepZ[var28] = var17 - 1
			var28 = (var28 + 1) % var21
			c.BFSDirection[var16+1][var17-1] = 9
			c.BFSCost[var16+1][var17-1] = var23
		}
		if var16 > 0 && var17 < var14-1 && c.BFSDirection[var16-1][var17+1] == 0 && var22[var16-1][var17+1]&0x280138 == 0 && var22[var16-1][var17]&0x280108 == 0 && var22[var16][var17+1]&0x280120 == 0 {
			c.BFSStepX[var28] = var16 - 1
			c.BFSStepZ[var28] = var17 + 1
			var28 = (var28 + 1) % var21
			c.BFSDirection[var16-1][var17+1] = 6
			c.BFSCost[var16-1][var17+1] = var23
		}
		if var16 < var13-1 && var17 < var14-1 && c.BFSDirection[var16+1][var17+1] == 0 && var22[var16+1][var17+1]&0x2801E0 == 0 && var22[var16+1][var17]&0x280180 == 0 && var22[var16][var17+1]&0x280120 == 0 {
			c.BFSStepX[var28] = var16 + 1
			c.BFSStepZ[var28] = var17 + 1
			var28 = (var28 + 1) % var21
			c.BFSDirection[var16+1][var17+1] = 12
			c.BFSCost[var16+1][var17+1] = var23
		}
	}
	c.TryMoveNearest = 0
	if !var20 {
		if arg2 {
			var23 = 100
			for i := 1; i < 2; i++ {
				for j := arg3 - i; j <= arg3+i; j++ {
					for k := arg8 - i; k <= arg8+i; k++ {
						if j >= 0 && k >= 0 && j < 104 && k < 104 && c.BFSCost[j][k] < var23 {
							var23 = c.BFSCost[j][k]
							var16 = j
							var17 = k
							c.TryMoveNearest = 1
							var20 = true
						}
					}
				}
				if var20 {
					break
				}
			}
		}
		if !var20 {
			return false
		}
	}
	var29 := 0
	c.BFSStepX[var29] = var16
	var19 = var29 + 1
	c.BFSStepZ[var29] = var17
	var24 := c.BFSDirection[var16][var17]
	var23 = var24
	for var16 != arg0 || var17 != arg4 {
		if var23 != var24 {
			var24 = var23
			c.BFSStepX[var19] = var16
			c.BFSStepZ[var19] = var17
			var19++
		}
		if var23&0x2 != 0 {
			var16++
		} else if var23&0x8 != 0 {
			var16--
		}
		if var23&0x1 != 0 {
			var17++
		} else if var23&0x4 != 0 {
			var17--
		}
		var23 = c.BFSDirection[var16][var17]
	}
	if var19 > 0 {
		var21 = min(var19, 25)
		var19--
		var25 := c.BFSStepX[var19]
		var26 := c.BFSStepZ[var19]
		if arg6 == 0 {
			c.Out.P1Isaac(181)
			c.Out.P1(var21 + var21 + 3)
		}
		if arg6 == 1 {
			c.Out.P1Isaac(165)
			c.Out.P1(var21 + var21 + 3 + 14)
		}
		if arg6 == 2 {
			c.Out.P1Isaac(93)
			c.Out.P1(var21 + var21 + 3)
		}
		if c.ActionKey[5] == 1 {
			c.Out.P1(1)
		} else {
			c.Out.P1(0)
		}
		c.Out.P2(var25 + c.SceneBaseTileX)
		c.Out.P2(var26 + c.SceneBaseTileZ)
		c.FlagSceneTileX = c.BFSStepX[0]
		c.FlagSceneTileZ = c.BFSStepZ[0]
		for i := 1; i < var21; i++ {
			var19--
			c.Out.P1(c.BFSStepX[var19] - var25)
			c.Out.P1(c.BFSStepZ[var19] - var26)
		}
		return true
	} else if arg6 == 1 {
		return false
	} else {
		return true
	}
}

func FormatObjCount(arg1 int) string {
	if arg1 < 100_000 {
		return strconv.Itoa(arg1)
	}
	if arg1 < 10_000_000 {
		return strconv.Itoa(arg1/1_000) + "K"
	}
	return strconv.Itoa(arg1/1_000_000) + "M"
}

func (c *Client) GetPlayer(arg0 *io.Packet, arg1 int) {
	c.EntityRemovalCount = 0
	c.EntityUpdateCount = 0
	c.GetPlayerLocal(arg0)
	c.GetPlayerOldVis(arg0)
	c.GetPlayerNewVis(arg1, arg0)
	c.GetPlayerExtended1(arg0)
	for i := range c.EntityRemovalCount {
		var5 := c.EntityRemovalIDs[i]
		if c.Players[var5].Cycle != clientextras.LoopCycle {
			c.Players[var5] = nil
		}
	}
	if arg0.Pos != arg1 {
		msg := "Error packet size mismatch in getplayer pos:" + strconv.Itoa(arg0.Pos) + " psize:" + strconv.Itoa(arg1)
		signlink.ReportErrorFunc(msg)
		panic(msg)
	}
	for i := range c.PlayerCount {
		if c.Players[c.PlayerIDs[i]] == nil {
			msg := c.Username + " null entry in pl list - pos:" + strconv.Itoa(i) + " size:" + strconv.Itoa(c.PlayerCount)
			signlink.ReportErrorFunc(msg)
			panic(msg)
		}
	}
}

func (c *Client) UpdateInterfaceAnimation(arg0, arg1 int) bool {
	var4 := false
	var5 := component.Instances[arg0]
	for i := 0; i < len(var5.ChildID) && var5.ChildID[i] != -1; i++ {
		var7 := component.Instances[var5.ChildID[i]]
		if var7.Type == 1 {
			// Java `|=` evaluates both sides; Go `||` short-circuits, which
			// would skip the recursive tick once var4 is true.
			if c.UpdateInterfaceAnimation(var7.Id, arg1) {
				var4 = true
			}
		}
		if var7.Type == 6 && (var7.Anim != -1 || var7.ActiveAnim != -1) {
			var8 := c.ExecuteInterfaceScript(var7)
			var9 := 0
			if var8 {
				var9 = var7.ActiveAnim
			} else {
				var9 = var7.Anim
			}
			if var9 != -1 {
				var10 := seqtype.Instances[var9]
				var7.SeqCycle += arg1
				for var7.SeqCycle > var10.Delay[var7.SeqFrame] {
					var7.SeqCycle -= var10.Delay[var7.SeqFrame] + 1
					var7.SeqFrame++
					if var7.SeqFrame >= var10.FrameCount {
						var7.SeqFrame -= var10.ReplayOff
						if var7.SeqFrame < 0 || var7.SeqFrame >= var10.FrameCount {
							var7.SeqFrame = 0
						}
					}
					var4 = true
				}
			}
		}
	}
	return var4
}

func (c *Client) AddMessage(arg0 int, arg1 string, arg3 string) {
	if arg0 == 0 && c.StickyChatInterfaceID != -1 {
		c.ModalMessage = arg1
		c.MouseClickButton = 0
	}
	if c.ChatInterfaceID == -1 {
		c.RedrawChatback = true
	}
	for i := 99; i > 0; i-- {
		c.MessageType[i] = c.MessageType[i-1]
		c.MessageSender[i] = c.MessageSender[i-1]
		c.MessageText[i] = c.MessageText[i-1]
	}
	c.MessageType[0] = arg0
	c.MessageSender[0] = arg3
	c.MessageText[0] = arg1
}

func (c *Client) ResetInterfaceAnimation(arg1 int) {
	var3 := component.Instances[arg1]
	for i := 0; i < len(var3.ChildID) && var3.ChildID[i] != -1; i++ {
		var5 := component.Instances[var3.ChildID[i]]
		if var5.Type == 1 {
			c.ResetInterfaceAnimation(var5.Id)
		}
		var5.SeqFrame = 0
		var5.SeqCycle = 0
	}
}

func (c *Client) RemoveFriend(arg1 int64) {
	if arg1 == 0 {
		return
	}
	for i := range c.FriendCount {
		if c.FriendName37[i] == arg1 {
			c.FriendCount--
			c.RedrawSidebar = true
			for j := i; j < c.FriendCount; j++ {
				c.FriendName[j] = c.FriendName[j+1]
				c.FriendWorld[j] = c.FriendWorld[j+1]
				c.FriendName37[j] = c.FriendName37[j+1]
			}
			c.Out.P1Isaac(11)
			c.Out.P8(arg1)
			return
		}
	}
}

func (c *Client) ExecuteInterfaceScript(arg0 *component.Component) bool {
	if arg0.ScriptComparator == nil {
		return false
	}
	for i := range len(arg0.ScriptComparator) {
		var4 := c.ExecuteClientscript1(arg0, i)
		var5 := arg0.ScriptOperand[i]
		if arg0.ScriptComparator[i] == 2 {
			if var4 >= var5 {
				return false
			}
		} else if arg0.ScriptComparator[i] == 3 {
			if var4 <= var5 {
				return false
			}
		} else if arg0.ScriptComparator[i] == 4 {
			if var4 == var5 {
				return false
			}
		} else if var4 != var5 {
			return false
		}
	}
	return true
}

func (c *Client) HandleMinimapInput() {
	if c.MouseClickButton != 1 {
		return
	}
	var2 := c.MouseClickX - 21 - 561
	var3 := c.MouseClickY - 9 - 5
	if var2 < 0 || var3 < 0 || var2 >= 146 || var3 >= 151 {
		return
	}
	var2 -= 73
	var3 -= 75
	var4 := (c.OrbitCameraYaw + c.MinimapAnticheatAngle) & 0x7FF
	var5 := pix3d.SinTable[var4]
	var6 := pix3d.CosTable[var4]
	var12 := (var5 * (c.MinimapZoom + 256)) >> 8
	var13 := (var6 * (c.MinimapZoom + 256)) >> 8
	var7 := (var3*var12 + var2*var13) >> 11
	var8 := (var3*var13 - var2*var12) >> 11
	var9 := (c.LocalPlayer.X + var7) >> 7
	var10 := (c.LocalPlayer.Z - var8) >> 7
	var11 := c.TryMove(c.LocalPlayer.PathTileX[0], 0, true, var9, c.LocalPlayer.PathTileZ[0], 1, 0, var10, 0, 0, 0)
	if !var11 {
		return
	}
	c.Out.P1(var2)
	c.Out.P1(var3)
	c.Out.P2(c.OrbitCameraYaw)
	c.Out.P1(57)
	c.Out.P1(c.MinimapAnticheatAngle)
	c.Out.P1(c.MinimapZoom)
	c.Out.P1(89)
	c.Out.P2(c.LocalPlayer.X)
	c.Out.P2(c.LocalPlayer.Z)
	c.Out.P1(c.TryMoveNearest)
	c.Out.P1(63)
}

func (c *Client) HandleMouseInput() {
	if c.ObjDragArea != 0 {
		return
	}
	var2 := c.MouseClickButton
	if c.SpellSelected == 1 && c.MouseClickX >= 520 && c.MouseClickY >= 165 && c.MouseClickX <= 788 && c.MouseClickY <= 230 {
		var2 = 0
	}
	var3 := 0
	var4 := 0
	var5 := 0
	if !c.MenuVisible {
		if var2 == 1 && c.MenuSize > 0 {
			var3 = c.MenuAction[c.MenuSize-1]
			if var3 == 602 || var3 == 596 || var3 == 22 || var3 == 892 || var3 == 415 || var3 == 405 || var3 == 38 || var3 == 422 || var3 == 478 || var3 == 347 || var3 == 188 {
				var4 = c.MenuParamB[c.MenuSize-1]
				var5 = c.MenuParamC[c.MenuSize-1]
				var6 := component.Instances[var5]
				if var6.Draggable {
					c.ObjGrabThreshold = false
					c.ObjDragCycles = 0
					c.ObjDragInterfaceID = var5
					c.ObjDragSlot = var4
					c.ObjDragArea = 2
					c.ObjGrabX = c.MouseClickX
					c.ObjGrabY = c.MouseClickY
					if component.Instances[var5].Layer == c.ViewportInterfaceID {
						c.ObjDragArea = 1
					}
					if component.Instances[var5].Layer == c.ChatInterfaceID {
						c.ObjDragArea = 3
					}
					return
				}
			}
		}
		if var2 == 1 && (c.MouseButtonsOption == 1 || c.IsAddFriendOption(c.MenuSize-1)) && c.MenuSize > 2 {
			var2 = 2
		}
		if var2 == 1 && c.MenuSize > 0 {
			c.UseMenuOption(c.MenuSize - 1)
		}
		if var2 != 2 || c.MenuSize <= 0 {
			return
		}
		c.ShowContextMenu()
		return
	}
	if var2 != 1 {
		var3 = c.MouseX
		var4 = c.MouseY
		if c.MenuArea == 0 {
			var3 -= 8
			var4 -= 11
		}
		if c.MenuArea == 1 {
			var3 -= 562
			var4 -= 231
		}
		if c.MenuArea == 2 {
			var3 -= 22
			var4 -= 375
		}
		if var3 < c.MenuX-10 || var3 > c.MenuX+c.MenuWidth+10 || var4 < c.MenuY-10 || var4 > c.MenuY+c.MenuHeight+10 {
			c.MenuVisible = false
			if c.MenuArea == 1 {
				c.RedrawSidebar = true
			}
			if c.MenuArea == 2 {
				c.RedrawChatback = true
			}
		}
	}
	if var2 != 1 {
		return
	}
	var3 = c.MenuX
	var4 = c.MenuY
	var5 = c.MenuWidth
	var11 := c.MouseClickX
	var7 := c.MouseClickY
	if c.MenuArea == 0 {
		var11 -= 8
		var7 -= 11
	}
	if c.MenuArea == 1 {
		var11 -= 562
		var7 -= 231
	}
	if c.MenuArea == 2 {
		var11 -= 22
		var7 -= 375
	}
	var8 := -1
	for i := range c.MenuSize {
		var10 := var4 + 31 + (c.MenuSize-1-i)*15
		if var11 > var3 && var11 < var3+var5 && var7 > var10-13 && var7 < var10+3 {
			var8 = i
		}
	}
	if var8 != -1 {
		c.UseMenuOption(var8)
	}
	c.MenuVisible = false
	if c.MenuArea == 1 {
		c.RedrawSidebar = true
	}
	if c.MenuArea == 2 {
		c.RedrawChatback = true
	}
}

func (c *Client) ApplyCutscene() {
	var2 := c.CutsceneSrcLocalTileX*128 + 64
	var3 := c.CutsceneSrcLocalTileZ*128 + 64
	var4 := c.GetHeightMapY(c.CurrentLevel, c.CutsceneSrcLocalTileX, c.CutsceneSrcLocalTileZ) - c.CutsceneSrcHeight
	if c.CameraX < var2 {
		c.CameraX += c.CutsceneMoveSpeed + (var2-c.CameraX)*c.CutsceneMoveAcceleration/1000
		if c.CameraX > var2 {
			c.CameraX = var2
		}
	}
	if c.CameraX > var2 {
		c.CameraX -= c.CutsceneMoveSpeed + (c.CameraX-var2)*c.CutsceneMoveAcceleration/1000
		if c.CameraX < var2 {
			c.CameraX = var2
		}
	}
	if c.CameraY < var4 {
		c.CameraY += c.CutsceneMoveSpeed + (var4-c.CameraY)*c.CutsceneMoveAcceleration/1000
		if c.CameraY > var4 {
			c.CameraY = var4
		}
	}
	if c.CameraY > var4 {
		c.CameraY -= c.CutsceneMoveSpeed + (c.CameraY-var4)*c.CutsceneMoveAcceleration/1000
		if c.CameraY < var4 {
			c.CameraY = var4
		}
	}
	if c.CameraZ < var3 {
		c.CameraZ += c.CutsceneMoveSpeed + (var3-c.CameraZ)*c.CutsceneMoveAcceleration/1000
		if c.CameraZ > var3 {
			c.CameraZ = var3
		}
	}
	if c.CameraZ > var3 {
		c.CameraZ -= c.CutsceneMoveSpeed + (c.CameraZ-var3)*c.CutsceneMoveAcceleration/1000
		if c.CameraZ < var3 {
			c.CameraZ = var3
		}
	}
	var2 = c.CutsceneDstLocalTileX*128 + 64
	var3 = c.CutsceneDstLocalTileZ*128 + 64
	var4 = c.GetHeightMapY(c.CurrentLevel, c.CutsceneDstLocalTileX, c.CutsceneDstLocalTileZ) - c.CutsceneDstHeight
	var5 := var2 - c.CameraX
	var6 := var4 - c.CameraY
	var7 := var3 - c.CameraZ
	var8 := int(math.Sqrt(float64(var5*var5 + var7*var7)))
	var9 := int(math.Atan2(float64(var6), float64(var8))*325.949) & 0x7FF
	var10 := int(math.Atan2(float64(var5), float64(var7))*-325.949) & 0x7FF
	if var9 < 128 {
		var9 = 128
	}
	if var9 > 383 {
		var9 = 383
	}
	if c.CameraPitch < var9 {
		c.CameraPitch += c.CutsceneRotateSpeed + (var9-c.CameraPitch)*c.CutsceneRotateAcceleration/1000
		if c.CameraPitch > var9 {
			c.CameraPitch = var9
		}
	}
	if c.CameraPitch > var9 {
		c.CameraPitch -= c.CutsceneRotateSpeed + (c.CameraPitch-var9)*c.CutsceneRotateAcceleration/1000
		if c.CameraPitch < var9 {
			c.CameraPitch = var9
		}
	}
	var11 := var10 - c.CameraYaw
	if var11 > 0x400 {
		var11 -= 2048
	}
	if var11 < -0x400 {
		var11 += 2048
	}
	if var11 > 0 {
		c.CameraYaw += c.CutsceneRotateSpeed + var11*c.CutsceneRotateAcceleration/1000
		c.CameraYaw &= 0x7FF
	}
	if var11 < 0 {
		c.CameraYaw -= c.CutsceneRotateSpeed + -var11*c.CutsceneRotateAcceleration/1000
		c.CameraYaw &= 0x7FF
	}
	var12 := var10 - c.CameraYaw
	if var12 > 0x400 {
		var12 -= 2048
	}
	if var12 < -0x400 {
		var12 += 2048
	}
	if var12 < 0 && var11 > 0 || var12 > 0 && var11 < 0 {
		c.CameraYaw = var10
	}
}

func (c *Client) HandleTabInput() {
	if c.MouseClickButton != 1 {
		return
	}
	if c.MouseClickX >= 549 && c.MouseClickX <= 583 && c.MouseClickY >= 195 && c.MouseClickY < 231 && c.TabInterfaceID[0] != -1 {
		c.RedrawSidebar = true
		c.SelectedTab = 0
		c.RedrawSideIcons = true
	}
	if c.MouseClickX >= 579 && c.MouseClickX <= 609 && c.MouseClickY >= 194 && c.MouseClickY < 231 && c.TabInterfaceID[1] != -1 {
		c.RedrawSidebar = true
		c.SelectedTab = 1
		c.RedrawSideIcons = true
	}
	if c.MouseClickX >= 607 && c.MouseClickX <= 637 && c.MouseClickY >= 194 && c.MouseClickY < 231 && c.TabInterfaceID[2] != -1 {
		c.RedrawSidebar = true
		c.SelectedTab = 2
		c.RedrawSideIcons = true
	}
	if c.MouseClickX >= 635 && c.MouseClickX <= 679 && c.MouseClickY >= 194 && c.MouseClickY < 229 && c.TabInterfaceID[3] != -1 {
		c.RedrawSidebar = true
		c.SelectedTab = 3
		c.RedrawSideIcons = true
	}
	if c.MouseClickX >= 676 && c.MouseClickX <= 706 && c.MouseClickY >= 194 && c.MouseClickY < 231 && c.TabInterfaceID[4] != -1 {
		c.RedrawSidebar = true
		c.SelectedTab = 4
		c.RedrawSideIcons = true
	}
	if c.MouseClickX >= 704 && c.MouseClickX <= 734 && c.MouseClickY >= 194 && c.MouseClickY < 231 && c.TabInterfaceID[5] != -1 {
		c.RedrawSidebar = true
		c.SelectedTab = 5
		c.RedrawSideIcons = true
	}
	if c.MouseClickX >= 732 && c.MouseClickX <= 766 && c.MouseClickY >= 195 && c.MouseClickY < 231 && c.TabInterfaceID[6] != -1 {
		c.RedrawSidebar = true
		c.SelectedTab = 6
		c.RedrawSideIcons = true
	}
	if c.MouseClickX >= 550 && c.MouseClickX <= 584 && c.MouseClickY >= 492 && c.MouseClickY < 528 && c.TabInterfaceID[7] != -1 {
		c.RedrawSidebar = true
		c.SelectedTab = 7
		c.RedrawSideIcons = true
	}
	if c.MouseClickX >= 582 && c.MouseClickX <= 612 && c.MouseClickY >= 492 && c.MouseClickY < 529 && c.TabInterfaceID[8] != -1 {
		c.RedrawSidebar = true
		c.SelectedTab = 8
		c.RedrawSideIcons = true
	}
	if c.MouseClickX >= 609 && c.MouseClickX <= 639 && c.MouseClickY >= 492 && c.MouseClickY < 529 && c.TabInterfaceID[9] != -1 {
		c.RedrawSidebar = true
		c.SelectedTab = 9
		c.RedrawSideIcons = true
	}
	if c.MouseClickX >= 637 && c.MouseClickX <= 681 && c.MouseClickY >= 493 && c.MouseClickY < 528 && c.TabInterfaceID[10] != -1 {
		c.RedrawSidebar = true
		c.SelectedTab = 10
		c.RedrawSideIcons = true
	}
	if c.MouseClickX >= 679 && c.MouseClickX <= 709 && c.MouseClickY >= 492 && c.MouseClickY < 529 && c.TabInterfaceID[11] != -1 {
		c.RedrawSidebar = true
		c.SelectedTab = 11
		c.RedrawSideIcons = true
	}
	if c.MouseClickX >= 706 && c.MouseClickX <= 736 && c.MouseClickY >= 492 && c.MouseClickY < 529 && c.TabInterfaceID[12] != -1 {
		c.RedrawSidebar = true
		c.SelectedTab = 12
		c.RedrawSideIcons = true
	}
	if c.MouseClickX >= 734 && c.MouseClickX <= 768 && c.MouseClickY >= 492 && c.MouseClickY < 528 && c.TabInterfaceID[13] != -1 {
		c.RedrawSidebar = true
		c.SelectedTab = 13
		c.RedrawSideIcons = true
	}
	CycleLogic1++
	if CycleLogic1 > 150 {
		CycleLogic1 = 0
		c.Out.P1Isaac(233)
		c.Out.P1(43)
	}
}

func (c *Client) HandleSocialMenuOption(arg0 *component.Component) bool {
	var3 := arg0.ClientCode
	if var3 >= 1 && var3 <= 200 {
		if var3 >= 101 {
			var3 -= 101
		} else {
			var3--
		}
		c.MenuOption[c.MenuSize] = "Remove @whi@" + c.FriendName[var3]
		c.MenuAction[c.MenuSize] = 557
		c.MenuSize++
		c.MenuOption[c.MenuSize] = "Message @whi@" + c.FriendName[var3]
		c.MenuAction[c.MenuSize] = 679
		c.MenuSize++
		return true
	} else if var3 >= 401 && var3 <= 500 {
		c.MenuOption[c.MenuSize] = "Remove @whi@" + arg0.Text
		c.MenuAction[c.MenuSize] = 556
		c.MenuSize++
		return true
	} else {
		return false
	}
}

func (c *Client) GetNpcPosOldVis(arg1 *io.Packet) {
	arg1.AccessBits()
	var4 := arg1.GBit(8)
	if var4 < c.NPCCount {
		for i := var4; i < c.NPCCount; i++ {
			c.EntityRemovalIDs[c.EntityRemovalCount] = c.NPCIDs[i]
			c.EntityRemovalCount++
		}
	}
	if var4 > c.NPCCount {
		msg := c.Username + " Too many npcs"
		signlink.ReportErrorFunc(msg)
		panic(msg)
	}
	c.NPCCount = 0
	for i := range var4 {
		var6 := c.NPCIDs[i]
		var7 := c.NPCs[var6]
		var8 := arg1.GBit(1)
		if var8 == 0 {
			c.NPCIDs[c.NPCCount] = var6
			c.NPCCount++
			var7.Cycle = clientextras.LoopCycle
		} else {
			var9 := arg1.GBit(2)
			if var9 == 0 {
				c.NPCIDs[c.NPCCount] = var6
				c.NPCCount++
				var7.Cycle = clientextras.LoopCycle
				c.EntityUpdateIDs[c.EntityUpdateCount] = var6
				c.EntityUpdateCount++
			} else {
				var10 := 0
				var11 := 0
				switch var9 {
				case 1:
					c.NPCIDs[c.NPCCount] = var6
					c.NPCCount++
					var7.Cycle = clientextras.LoopCycle
					var10 = arg1.GBit(3)
					var7.MoveAlongRoute(false, var10)
					var11 = arg1.GBit(1)
					if var11 == 1 {
						c.EntityUpdateIDs[c.EntityUpdateCount] = var6
						c.EntityUpdateCount++
					}
				case 2:
					c.NPCIDs[c.NPCCount] = var6
					c.NPCCount++
					var7.Cycle = clientextras.LoopCycle
					var10 = arg1.GBit(3)
					var7.MoveAlongRoute(true, var10)
					var11 = arg1.GBit(3)
					var7.MoveAlongRoute(true, var11)
					var12 := arg1.GBit(1)
					if var12 == 1 {
						c.EntityUpdateIDs[c.EntityUpdateCount] = var6
						c.EntityUpdateCount++
					}
				case 3:
					c.EntityRemovalIDs[c.EntityRemovalCount] = var6
					c.EntityRemovalCount++
				}
			}
		}
	}
}

// GetParameter (applet HTML <param>) intentionally not ported: Go client takes config from CLI args / clientextras.

func (c *Client) TryReconnect() {
	if c.IdleTimeout > 0 {
		c.Logout()
		return
	}
	c.AreaViewport.Bind()
	c.FontPlain12.CentreString(144, 0, "Connection lost", 257)
	c.FontPlain12.CentreString(143, 0xFFFFFF, "Connection lost", 256)
	c.FontPlain12.CentreString(159, 0, "Please wait - attempting to reestablish", 257)
	c.FontPlain12.CentreString(158, 0xFFFFFF, "Please wait - attempting to reestablish", 256)
	c.presentLoadingMessage()
	c.FlagSceneTileX = 0
	var2 := c.Stream
	c.InGame = false
	c.LoginFunc(c.Username, c.Password, true)
	if !c.InGame {
		c.Logout()
	}
	if var2 != nil {
		// Java's `try { var2.close(); } catch (Exception) {}` swallows a
		// possible NPE on null stream; Go must nil-check to avoid panic.
		var2.Close()
	}
}

func (c *Client) UpdateFlameBuffer(image *pix8.Pix8) {
	height := 256

	for i := range len(c.FlameBuffer0) {
		c.FlameBuffer0[i] = 0
	}

	for range 5000 {
		random := int(rand.Float64() * 128.0 * float64(height))
		c.FlameBuffer0[random] = int(rand.Float64() * 256.0)
	}

	for range 20 {
		for y := 1; y < height-1; y++ {
			for x := 1; x < 127; x++ {
				index := x + (y << 7)
				c.FlameBuffer1[index] = (c.FlameBuffer0[index-1] + c.FlameBuffer0[index+1] + c.FlameBuffer0[index-128] + c.FlameBuffer0[index+128]) / 4
			}
		}

		last := c.FlameBuffer0
		c.FlameBuffer0 = c.FlameBuffer1
		c.FlameBuffer1 = last
	}

	if image != nil {
		off := 0

		for y := range image.Hi {
			for x := range image.Wi {
				off++
				if image.Pixels[off-1] != 0 {
					x0 := x + 16 + image.XOf
					y0 := y + 16 + image.YOf
					index := x0 + (y0 << 7)

					c.FlameBuffer0[index] = 0
				}
			}
		}
	}
}

func (c *Client) SortObjStacks(arg0, arg1 int) {
	var3 := c.LevelObjStacks[c.CurrentLevel][arg0][arg1]
	if var3 == nil {
		c.Scene.RemoveObjStack(c.CurrentLevel, arg0, arg1)
		return
	}
	var4 := -99999999
	var var5 *entity.ObjStackEntity
	// Java: ObjStackEntity extends Linkable, so addHead(var5) moves the
	// existing list node. In Go, *Linkable is a wrapper around the entity
	// pointer; track the wrapper from the iteration so we re-add it rather
	// than allocating a duplicate. See deob/client.java:8490.
	var var5Link *datastruct.Linkable[*entity.ObjStackEntity]
	for var6 := var3.Head(); var6 != nil; var6 = var3.Next() {
		v := var6.Value
		var7 := objtype.Get(v.Index)
		var8 := var7.Cost
		if var7.Stackable {
			var8 *= v.Count + 1
		}
		if var8 > var4 {
			var4 = var8
			var5 = v
			var5Link = var6
		}
	}
	var3.AddHead(var5Link)
	var15 := -1
	var8 := -1
	var9 := 0
	var10 := 0
	for var6 := var3.Head(); var6 != nil; var6 = var3.Next() {
		v := var6.Value
		if v.Index != var5.Index && var15 == -1 {
			var15 = v.Index
			var9 = v.Count
		}
		if v.Index != var5.Index && v.Index != var15 && var8 == -1 {
			var8 = v.Index
			var10 = v.Count
		}
	}
	var var11 *model.Model
	if var15 != -1 {
		var11 = objtype.Get(var15).GetInterfaceModel(var9)
	}
	var var12 *model.Model
	if var8 != -1 {
		var12 = objtype.Get(var8).GetInterfaceModel(var10)
	}
	var13 := arg0 + (arg1 << 7) + 1610612736
	var14 := objtype.Get(var5.Index)
	c.Scene.AddObjStack(var14.GetInterfaceModel(var5.Count), var11, c.GetHeightMapY(c.CurrentLevel, arg0*128+64, arg1*128+64), c.CurrentLevel, var13, arg1, arg0, var12)
}

func (c *Client) BuildScene() {
	// Java: try { ... } catch (Exception) {} — empty swallow around the entire
	// build-scene body (BZip2 decode + scene/landscape assembly). The Java
	// finally-equivalent (LocType.modelCacheStatic.clear() + Pix3D.initPool(20)
	// at the tail) is preserved unconditionally below. Any panic in Go
	// propagates naturally.
	c.MinimapLevel = -1
	c.MergedLocations.Clear()
	c.LocList.Clear()
	c.Spotanims.Clear()
	c.Projectiles.Clear()
	pix3d.ClearTexels()
	c.ClearCaches()
	c.Scene.Reset()
	for i := range 4 {
		c.LevelCollisionMap[i].Reset()
	}
	var3 := world.NewWorld(104, c.LevelTileFlags, 104, c.LevelHeightMap)
	var4 := make([]byte, 100_000)
	var5 := len(c.SceneMapLandData)
	world.LowMemory = world3d.LowMemory
	for i := range var5 {
		var7 := c.SceneMapIndex[i] >> 8
		var8 := c.SceneMapIndex[i] & 0xFF
		if var7 == 33 && var8 >= 71 && var8 <= 73 {
			world.LowMemory = false
		}
	}
	if world.LowMemory {
		c.Scene.SetMinLevel(c.CurrentLevel)
	} else {
		c.Scene.SetMinLevel(0)
	}
	c.Out.P1Isaac(108)
	for i := range var5 {
		var8 := (c.SceneMapIndex[i]>>8)*64 - c.SceneBaseTileX
		var9 := (c.SceneMapIndex[i]&0xFF)*64 - c.SceneBaseTileZ
		var10 := c.SceneMapLandData[i]
		if var10 != nil {
			var11 := io.NewPacket(var10).G4()
			bzip2.Read(var4, var11, var10, len(var10)-4, 4)
			var3.LoadGround(var4, (c.SceneCenterZoneX-6)*8, var9, var8, (c.SceneCenterZoneZ-6)*8)
		} else if c.SceneCenterZoneZ < 800 {
			var3.ClearLandscape(var8, var9, 64, 64)
		}
	}
	c.Out.P1Isaac(108)
	var16 := 0
	for i := range var5 {
		var14 := c.SceneMapLocData[i]
		if var14 != nil {
			var16 = io.NewPacket(var14).G4()
			bzip2.Read(var4, var16, var14, len(var14)-4, 4)
			var11 := (c.SceneMapIndex[i]>>8)*64 - c.SceneBaseTileX
			var12 := (c.SceneMapIndex[i]&0xFF)*64 - c.SceneBaseTileZ
			var3.LoadLocations(var4, c.Scene, c.LevelCollisionMap, c.LocList, var12, var11)
		}
	}
	c.Out.P1Isaac(108)
	var3.Build(c.Scene, c.LevelCollisionMap)
	c.AreaViewport.Bind()
	c.Out.P1Isaac(108)
	for var15 := c.LocList.Head(); var15 != nil; var15 = c.LocList.Next() {
		v := var15.Value
		if c.LevelTileFlags[1][v.X][v.Z]&0x2 == 2 {
			v.Level--
			if v.Level < 0 {
				var15.Unlink()
			}
		}
	}
	for i := range 104 {
		for j := range 104 {
			c.SortObjStacks(i, j)
		}
	}
	for var17 := c.SpawnedLocations.Head(); var17 != nil; var17 = c.SpawnedLocations.Next() {
		v := var17.Value
		c.AddLoc(v.Angle, v.X, v.Z, v.Layer, v.LocIndex, v.Shape, v.Plane)
	}
	loctype.ModelCacheStatic.Clear()
	pix3d.InitPool(20)
}

func (c *Client) Update() {
	if c.ErrorStarted || c.ErrorLoading || c.ErrorHost {
		return
	}
	clientextras.LoopCycle++
	if c.InGame {
		c.UpdateGame()
	} else {
		c.UpdateTitle()
	}
}

func (c *Client) UpdateEntityChats() {
	var3 := 0
	for i := -1; i < c.PlayerCount; i++ {
		if i == -1 {
			var3 = c.LOCAL_PLAYER_INDEX
		} else {
			var3 = c.PlayerIDs[i]
		}
		var4 := c.Players[var3]
		if var4 != nil && var4.ChatTimer > 0 {
			var4.ChatTimer--
			if var4.ChatTimer == 0 {
				var4.Chat = ""
			}
		}
	}
	for i := range c.NPCCount {
		var6 := c.NPCIDs[i]
		var5 := c.NPCs[var6]
		if var5 != nil && var5.ChatTimer > 0 {
			var5.ChatTimer--
			if var5.ChatTimer == 0 {
				var5.Chat = ""
			}
		}
	}
}

func (c *Client) ExecuteClientscript1(arg0 *component.Component, arg2 int) (result int) {
	if arg0.Scripts == nil || arg2 >= len(arg0.Scripts) {
		return -2
	}
	// Java: catch (Exception) { return -1; } — primarily guards against
	// malformed scripts that walk off the end of the int[] (Java's
	// ArrayIndexOutOfBoundsException). Mirror with a deferred recover that
	// converts any panic (e.g. Go slice-bounds) into the same -1 sentinel.
	defer func() {
		if r := recover(); r != nil {
			result = -1
		}
	}()
	var4 := arg0.Scripts[arg2]
	var5 := 0
	var6 := 0
	for {
		var7 := var4[var6]
		var6++
		if var7 == 0 {
			return var5
		}
		if var7 == 1 {
			var5 += c.SkillLevel[var4[var6]]
			var6++
		}
		if var7 == 2 {
			var5 += c.SkillBaseLevel[var4[var6]]
			var6++
		}
		if var7 == 3 {
			var5 += c.SkillExperience[var4[var6]]
			var6++
		}
		var var8 *component.Component
		var9 := 0
		//var10 := 0
		if var7 == 4 {
			var8 = component.Instances[var4[var6]]
			var6++
			var9 = var4[var6] + 1
			var6++
			for i := range len(var8.InvSlotObjId) {
				if var8.InvSlotObjId[i] == var9 {
					var5 += var8.InvSlotObjCount[i]
				}
			}
		}
		if var7 == 5 {
			var5 += c.Varps[var4[var6]]
			var6++
		}
		if var7 == 6 {
			var5 += LevelExperience[c.SkillBaseLevel[var4[var6]]-1]
			var6++
		}
		if var7 == 7 {
			var5 += c.Varps[var4[var6]] * 100 / 46875
			var6++
		}
		if var7 == 8 {
			var5 += c.LocalPlayer.CombatLevel
		}
		var12 := 0
		if var7 == 9 {
			for i := range 19 {
				if i == 18 {
					i = 20
				}
				var5 += c.SkillBaseLevel[i]
			}
		}
		if var7 == 10 {
			var8 = component.Instances[var4[var6]]
			var6++
			var9 = var4[var6] + 1
			var6++
			for i := range len(var8.InvSlotObjId) {
				if var8.InvSlotObjId[i] == var9 {
					var5 += 999999999
					break
				}
			}
		}
		if var7 == 11 {
			var5 += c.Energy
		}
		if var7 == 12 {
			var5 += c.WeightCarried
		}
		if var7 == 13 {
			var12 = c.Varps[var4[var6]]
			var6++
			var9 = var4[var6]
			var6++
			if var12&(0x1<<var9) == 0 {
				var5 += 0
			} else {
				var5 += 1
			}
		}
	}
}

// DrawError renders the user-facing error screen for the three error
// modes (ErrorLoading, ErrorHost, ErrorStarted). Java: drawError() at
// client.java:8727-8781.
//
// Java painted directly to the AWT base component's Graphics; the Go
// port clears a shared overlay PixMap (via ensureOverlay), draws text with
// the errorfont package (the "Go" bold typeface), then composites via
// OverlayPixMap.Draw. errorfont substitutes for Java's Helvetica BOLD 16/20
// and is always available even when the error fires before the cache fonts
// load (the cause of the nil-FontBold12 SIGSEGV when a host was specified).
// The branch ordering, frame-rate
// throttle, and FlameActive=false side effects mirror Java exactly so
// the rest of the client stays in sync. The early return on
// !ErrorStarted composites first (so any ErrorLoading/ErrorHost draws
// already in the overlay are flushed before returning) — this differs
// from Java only because Java drew directly to the window, where each
// drawString was immediately visible.
func (c *Client) DrawError() {
	c.ensureOverlay()
	c.OverlayPixMap.Bind()
	pix2d.FillRect(0, 0, 0x000000, c.ScreenWidth, c.ScreenHeight)

	c.SetFrameRate(1)

	// DrawError can run before the title fonts are loaded: Client.Load flags
	// ErrorHost/ErrorLoading and returns before the cache "b12" PixFont is
	// built, and the recover() defer can likewise flag ErrorLoading on an early
	// panic. Java drew these screens with an always-available AWT system font
	// (GameShell.java:541), so the original code never depended on game assets
	// being loaded. The errorfont package (the embedded "Go" bold typeface) is
	// the Go analogue — always available and a close match for Java's Helvetica
	// BOLD — so route the error text through it (writing straight to the
	// overlay) instead of the cache-loaded FontBold12, which is nil on these
	// early-error paths. Baseline-y semantics match AWT drawString.
	drawText := func(x, y, color int, s string) {
		errorfont.DrawString(c.OverlayPixMap, x, y, color, s)
	}

	if c.ErrorLoading {
		c.FlameActive = false
		// Java: Font Helvetica BOLD 16, yellow header; BOLD 12 white body.
		// Go: FontBold12 throughout — same divergence as elsewhere.
		drawText(30, 35, 0xFFFF00,
			"Sorry, an error has occured whilst loading RuneScape")
		drawText(30, 85, 0xFFFFFF,
			"To fix this try the following (in order):")
		drawText(30, 135, 0xFFFFFF,
			"1: Try closing ALL open web-browser windows, and reloading")
		drawText(30, 165, 0xFFFFFF,
			"2: Try clearing your web-browsers cache from tools->internet options")
		drawText(30, 195, 0xFFFFFF,
			"3: Try using a different game-world")
		drawText(30, 225, 0xFFFFFF,
			"4: Try rebooting your computer")
		drawText(30, 255, 0xFFFFFF,
			"5: Try selecting a different version of Java from the play-game menu")
	}
	if c.ErrorHost {
		c.FlameActive = false
		// Java: Font Helvetica BOLD 20, white. Go: FontBold12.
		drawText(50, 50, 0xFFFFFF, "Error - unable to load game!")
		drawText(50, 100, 0xFFFFFF, "To play RuneScape make sure you play from")
		drawText(50, 150, 0xFFFFFF, "http://www.runescape.com")
	}
	if !c.ErrorStarted {
		c.OverlayPixMap.Draw(0, 0)
		return
	}
	c.FlameActive = false
	drawText(30, 35, 0xFFFF00,
		"Error a copy of RuneScape already appears to be loaded")
	drawText(30, 85, 0xFFFFFF,
		"To fix this try the following (in order):")
	drawText(30, 135, 0xFFFFFF,
		"1: Try closing ALL open web-browser windows, and reloading")
	drawText(30, 165, 0xFFFFFF,
		"2: Try rebooting your computer, and reloading")
	c.OverlayPixMap.Draw(0, 0)
}

func (c *Client) LoadTitleBackground() {
	src := c.JagTitle.Read("title.dat", nil)
	background := pix32.NewPix322(src)

	c.ImageTitle0.Bind()
	background.QuickPlotSprite(0, 0)

	c.ImageTitle1.Bind()
	background.QuickPlotSprite(-661, 0)

	c.ImageTitle2.Bind()
	background.QuickPlotSprite(-128, 0)

	c.ImageTitle3.Bind()
	background.QuickPlotSprite(-214, -386)

	c.ImageTitle4.Bind()
	background.QuickPlotSprite(-214, -186)

	c.ImageTitle5.Bind()
	background.QuickPlotSprite(0, -265)

	c.ImageTitle6.Bind()
	background.QuickPlotSprite(-574, -265)

	c.ImageTitle7.Bind()
	background.QuickPlotSprite(-128, -186)

	c.ImageTitle8.Bind()
	background.QuickPlotSprite(-574, -186)

	// draw right side (mirror image)
	pixels := make([]int, background.Wi)
	for y := range background.Hi {
		for x := range background.Wi {
			pixels[x] = background.Pixels[background.Wi-x-1+background.Wi*y]
		}

		for x := range background.Wi {
			background.Pixels[x+background.Wi*y] = pixels[x]
		}
	}

	c.ImageTitle0.Bind()
	background.QuickPlotSprite(394, 0)

	c.ImageTitle1.Bind()
	background.QuickPlotSprite(-267, 0)

	c.ImageTitle2.Bind()
	background.QuickPlotSprite(266, 0)

	c.ImageTitle3.Bind()
	background.QuickPlotSprite(180, -386)

	c.ImageTitle4.Bind()
	background.QuickPlotSprite(180, -186)

	c.ImageTitle5.Bind()
	background.QuickPlotSprite(394, -265)

	c.ImageTitle6.Bind()
	background.QuickPlotSprite(-180, -265)

	c.ImageTitle7.Bind()
	background.QuickPlotSprite(212, -186)

	c.ImageTitle8.Bind()
	background.QuickPlotSprite(-180, -186)

	logo := pix32.NewPix323(c.JagTitle, "logo", 0)
	c.ImageTitle2.Bind()
	logo.PlotSprite(18, c.ScreenWidth/2-logo.Wi/2-128)
}

func (c *Client) PushLocs() {
	for var2 := c.LocList.Head(); var2 != nil; var2 = c.LocList.Next() {
		v := var2.Value
		var3 := false
		v.SeqCycle += c.SceneDelta
		if v.SeqFrame == -1 {
			v.SeqFrame = 0
			var3 = true
		}
		for ok := true; ok; ok = v.SeqFrame >= 0 && v.SeqFrame < v.Seq.FrameCount {
			for ok2 := true; ok2; ok2 = v.SeqFrame < v.Seq.FrameCount {
				if v.SeqCycle <= v.Seq.Delay[v.SeqFrame] {
					goto afterLabel67
				}
				v.SeqCycle -= v.Seq.Delay[v.SeqFrame] + 1
				v.SeqFrame++
				var3 = true
			}
			v.SeqFrame -= v.Seq.ReplayOff
		}
		var2.Unlink()
		var3 = false
	afterLabel67:
		if var3 {
			var4 := v.Level
			var5 := v.X
			var6 := v.Z
			var7 := 0
			if v.Type == 0 {
				var7 = c.Scene.GetWallBitSet(var4, var5, var6)
			}
			if v.Type == 1 {
				var7 = c.Scene.GetWallDecorationBitSet(var4, var6, var5)
			}
			if v.Type == 2 {
				var7 = c.Scene.GetLocBitSet(var4, var5, var6)
			}
			if v.Type == 3 {
				var7 = c.Scene.GetGroundDecorationBitSet(var4, var5, var6)
			}
			if var7 != 0 && ((var7>>14)&0x7FFF) == v.Index {
				var8 := c.LevelHeightMap[var4][var5][var6]
				var9 := c.LevelHeightMap[var4][var5+1][var6]
				var10 := c.LevelHeightMap[var4][var5+1][var6+1]
				var11 := c.LevelHeightMap[var4][var5][var6+1]
				var12 := loctype.Get(v.Index)
				var13 := -1
				if v.SeqFrame != -1 {
					var13 = v.Seq.Frames[v.SeqFrame]
				}
				var14 := 0
				var15 := 0
				var16 := 0
				var var17 *model.Model
				switch v.Type {
				case 2:
					var14 = c.Scene.GetInfo(var4, var5, var6, var7)
					var15 = var14 & 0x1F
					var16 = var14 >> 6
					if var15 == 11 {
						var15 = 10
					}
					var17 = var12.GetModel(var15, var16, var8, var9, var10, var11, var13)
					c.Scene.SetLocModel(var5, var17, var4, var6)
				case 1:
					var21 := var12.GetModel(4, 0, var8, var9, var10, var11, var13)
					c.Scene.SetWallDecorationModel(var6, var5, var21, var4)
				case 0:
					var14 = c.Scene.GetInfo(var4, var5, var6, var7)
					var15 = var14 & 0x1F
					var16 = var14 >> 6
					if var15 == 2 {
						var23 := (var16 + 1) & 0x3
						var18 := var12.GetModel(2, var16+4, var8, var9, var10, var11, var13)
						var19 := var12.GetModel(2, var23, var8, var9, var10, var11, var13)
						c.Scene.SetWallModels(var18, var19, var6, var5, var4)
					} else {
						var17 = var12.GetModel(var15, var16, var8, var9, var10, var11, var13)
						c.Scene.SetWallModel(var17, var6, var5, var4)
					}
				case 3:
					var14 = c.Scene.GetInfo(var4, var5, var6, var7)
					var15 = var14 >> 6
					var22 := var12.GetModel(22, var15, var8, var9, var10, var11, var13)
					c.Scene.SetGroundDecorationModel(var22, var6, var5, var4)
				}
			} else {
				var2.Unlink()
			}
		}
	}
}

func (c *Client) RemoveIgnore(arg1 int64) {
	if arg1 == 0 {
		return
	}
	for i := range c.IgnoreCount {
		if c.IgnoreName37[i] == arg1 {
			c.IgnoreCount--
			c.RedrawSidebar = true
			for j := i; j < c.IgnoreCount; j++ {
				c.IgnoreName37[j] = c.IgnoreName37[j+1]
			}
			c.Out.P1Isaac(171)
			c.Out.P8(arg1)
			return
		}
	}
}

func (c *Client) HandleViewportOptions() {
	if c.ObjSelected == 0 && c.SpellSelected == 0 {
		c.MenuOption[c.MenuSize] = "Walk here"
		c.MenuAction[c.MenuSize] = 660
		c.MenuParamB[c.MenuSize] = c.MouseX
		c.MenuParamC[c.MenuSize] = c.MouseY
		c.MenuSize++
	}
	var2 := -1
	for i := range model.PickedCount {
		var4 := model.PickedBitsets[i]
		var5 := var4 & 0x7F
		var6 := (var4 >> 7) & 0x7F
		var7 := (var4 >> 29) & 0x3
		var8 := (var4 >> 14) & 0x7FFF
		if var4 != var2 {
			var2 = var4
			//var10 := 0
			if var7 == 2 && c.Scene.GetInfo(c.CurrentLevel, var5, var6, var4) >= 0 {
				var9 := loctype.Get(var8)
				if c.ObjSelected == 1 {
					c.MenuOption[c.MenuSize] = "Use " + c.ObjSelectedName + " with @cya@" + var9.Name
					c.MenuAction[c.MenuSize] = 450
					c.MenuParamA[c.MenuSize] = var4
					c.MenuParamB[c.MenuSize] = var5
					c.MenuParamC[c.MenuSize] = var6
					c.MenuSize++
				} else if c.SpellSelected != 1 {
					if var9.Op != nil {
						for j := 4; j >= 0; j-- {
							if var9.Op[j] != "" {
								c.MenuOption[c.MenuSize] = var9.Op[j] + " @cya@" + var9.Name
								switch j {
								case 0:
									c.MenuAction[c.MenuSize] = 285
								case 1:
									c.MenuAction[c.MenuSize] = 504
								case 2:
									c.MenuAction[c.MenuSize] = 364
								case 3:
									c.MenuAction[c.MenuSize] = 581
								case 4:
									c.MenuAction[c.MenuSize] = 1501
								}
								c.MenuParamA[c.MenuSize] = var4
								c.MenuParamB[c.MenuSize] = var5
								c.MenuParamC[c.MenuSize] = var6
								c.MenuSize++
							}
						}
					}
					c.MenuOption[c.MenuSize] = "Examine @cya@" + var9.Name
					c.MenuAction[c.MenuSize] = 1175
					c.MenuParamA[c.MenuSize] = var4
					c.MenuParamB[c.MenuSize] = var5
					c.MenuParamC[c.MenuSize] = var6
					c.MenuSize++
				} else if c.ActiveSpellFlags&0x4 == 4 {
					c.MenuOption[c.MenuSize] = c.SpellCaption + " @cya@" + var9.Name
					c.MenuAction[c.MenuSize] = 55
					c.MenuParamA[c.MenuSize] = var4
					c.MenuParamB[c.MenuSize] = var5
					c.MenuParamC[c.MenuSize] = var6
					c.MenuSize++
				}
			}
			var var11 *entity.NpcEntity
			if var7 == 1 {
				var13 := c.NPCs[var8]
				if var13.Type.Size == 1 && var13.X&0x7F == 64 && var13.Z&0x7F == 64 {
					for j := range c.NPCCount {
						var11 = c.NPCs[c.NPCIDs[j]]
						if var11 != nil && var11 != var13 && var11.Type.Size == 1 && var11.X == var13.X && var11.Z == var13.Z {
							c.AddNPCOptions(var11.Type, var6, var5, c.NPCIDs[j])
						}
					}
				}
				c.AddNPCOptions(var13.Type, var6, var5, var8)
			}
			if var7 == 0 {
				var14 := c.Players[var8]
				if var14.X&0x7F == 64 && var14.Z&0x7F == 64 {
					for j := range c.NPCCount {
						var11 = c.NPCs[c.NPCIDs[j]]
						if var11 != nil && var11.Type.Size == 1 && var11.X == var14.X && var11.Z == var14.Z {
							c.AddNPCOptions(var11.Type, var6, var5, c.NPCIDs[j])
						}
					}
					for j := range c.PlayerCount {
						var12 := c.Players[c.PlayerIDs[j]]
						if var12 != nil && var12 != var14 && var12.X == var14.X && var12.Z == var14.Z {
							c.AddPlayerOptions(var6, c.PlayerIDs[j], var12, var5)
						}
					}
				}
				c.AddPlayerOptions(var6, var8, var14, var5)
			}
			if var7 == 3 {
				var15 := c.LevelObjStacks[c.CurrentLevel][var5][var6]
				if var15 != nil {
					for var17 := var15.Tail(); var17 != nil; var17 = var15.Prev() {
						v := var17.Value
						var18 := objtype.Get(v.Index)
						if c.ObjSelected == 1 {
							c.MenuOption[c.MenuSize] = "Use " + c.ObjSelectedName + " with @lre@" + var18.Name
							c.MenuAction[c.MenuSize] = 217
							c.MenuParamA[c.MenuSize] = v.Index
							c.MenuParamB[c.MenuSize] = var5
							c.MenuParamC[c.MenuSize] = var6
							c.MenuSize++
						} else if c.SpellSelected != 1 {
							for j := 4; j >= 0; j-- {
								if var18.Op != nil && var18.Op[j] != "" {
									c.MenuOption[c.MenuSize] = var18.Op[j] + " @lre@" + var18.Name
									switch j {
									case 0:
										c.MenuAction[c.MenuSize] = 224
									case 1:
										c.MenuAction[c.MenuSize] = 993
									case 2:
										c.MenuAction[c.MenuSize] = 99
									case 3:
										c.MenuAction[c.MenuSize] = 746
									case 4:
										c.MenuAction[c.MenuSize] = 877
									}
									c.MenuParamA[c.MenuSize] = v.Index
									c.MenuParamB[c.MenuSize] = var5
									c.MenuParamC[c.MenuSize] = var6
									c.MenuSize++
								} else if j == 2 {
									c.MenuOption[c.MenuSize] = "Take @lre@" + var18.Name
									c.MenuAction[c.MenuSize] = 99
									c.MenuParamA[c.MenuSize] = v.Index
									c.MenuParamB[c.MenuSize] = var5
									c.MenuParamC[c.MenuSize] = var6
									c.MenuSize++
								}
							}
							c.MenuOption[c.MenuSize] = "Examine @lre@" + var18.Name
							c.MenuAction[c.MenuSize] = 1102
							c.MenuParamA[c.MenuSize] = v.Index
							c.MenuParamB[c.MenuSize] = var5
							c.MenuParamC[c.MenuSize] = var6
							c.MenuSize++
						} else if c.ActiveSpellFlags&0x1 == 1 {
							c.MenuOption[c.MenuSize] = c.SpellCaption + " @lre@" + var18.Name
							c.MenuAction[c.MenuSize] = 965
							c.MenuParamA[c.MenuSize] = v.Index
							c.MenuParamB[c.MenuSize] = var5
							c.MenuParamC[c.MenuSize] = var6
							c.MenuSize++
						}
					}
				}
			}
		}
	}
}

func (c *Client) UpdatePlayers() {
	var3 := 0
	for i := -1; i < c.PlayerCount; i++ {
		if i == -1 {
			var3 = c.LOCAL_PLAYER_INDEX
		} else {
			var3 = c.PlayerIDs[i]
		}
		var4 := c.Players[var3]
		if var4 != nil {
			c.UpdatePlayerEntity(var4)
		}
	}
	CycleLogic6++
	if CycleLogic6 <= 1406 {
		return
	}
	CycleLogic6 = 0
	c.Out.P1Isaac(219)
	c.Out.P1(0)
	var3 = c.Out.Pos
	c.Out.P1(162)
	c.Out.P1(22)
	if int(rand.Float64()*2.0) == 0 {
		c.Out.P1(84)
	}
	c.Out.P2(31824)
	c.Out.P2(13490)
	if int(rand.Float64()*2.0) == 0 {
		c.Out.P1(123)
	}
	if int(rand.Float64()*2.0) == 0 {
		c.Out.P1(134)
	}
	c.Out.P1(100)
	c.Out.P1(94)
	c.Out.P2(35521)
	c.Out.PSize1(c.Out.Pos - var3)
}

func (c *Client) DrawTileHint() {
	if c.HintType != 2 {
		return
	}
	c.ProjectFromGround2(((c.HintTileZ-c.SceneBaseTileZ)<<7)+c.HintOffsetZ, ((c.HintTileX-c.SceneBaseTileX)<<7)+c.HintOffsetX, c.HintHeight*2)
	if c.ProjectX > -1 && clientextras.LoopCycle%20 < 10 {
		c.ImageHeadIcons[2].PlotSprite(c.ProjectY-28, c.ProjectX-12)
	}
}

func (c *Client) GetPlayerLocal(arg2 *io.Packet) {
	arg2.AccessBits()
	var4 := arg2.GBit(1)
	if var4 == 0 {
		return
	}
	var5 := arg2.GBit(2)
	if var5 == 0 {
		c.EntityUpdateIDs[c.EntityUpdateCount] = c.LOCAL_PLAYER_INDEX
		c.EntityUpdateCount++
		return
	}
	var6 := 0
	var7 := 0
	if var5 == 1 {
		var6 = arg2.GBit(3)
		c.LocalPlayer.MoveAlongRoute(false, var6)
		var7 = arg2.GBit(1)
		if var7 == 1 {
			c.EntityUpdateIDs[c.EntityUpdateCount] = c.LOCAL_PLAYER_INDEX
			c.EntityUpdateCount++
		}
		return
	}
	var8 := 0
	switch var5 {
	case 2:
		var6 = arg2.GBit(3)
		c.LocalPlayer.MoveAlongRoute(true, var6)
		var7 = arg2.GBit(3)
		c.LocalPlayer.MoveAlongRoute(true, var7)
		var8 = arg2.GBit(1)
		if var8 == 1 {
			c.EntityUpdateIDs[c.EntityUpdateCount] = c.LOCAL_PLAYER_INDEX
			c.EntityUpdateCount++
		}
	case 3:
		c.CurrentLevel = arg2.GBit(2)
		var6 = arg2.GBit(7)
		var7 = arg2.GBit(7)
		var8 = arg2.GBit(1)
		c.LocalPlayer.Teleport(var8 == 1, var6, var7)
		var9 := arg2.GBit(1)
		if var9 == 1 {
			c.EntityUpdateIDs[c.EntityUpdateCount] = c.LOCAL_PLAYER_INDEX
			c.EntityUpdateCount++
		}
	}
}

func (c *Client) DrawChatback() {
	// Pixel repaint is gated on RedrawChatback (expensive: 100-message
	// scrollback walk + interface tree + font rendering). GPU upload
	// always runs — pre-fix, the whole function was wrapped in
	// `if RedrawChatback` at the call site, relying on Java/AWT's
	// retained back buffer.
	if !c.RedrawChatback {
		c.AreaChatback.Draw(22, 375)
		return
	}
	c.RedrawChatback = false
	c.AreaChatback.Bind()
	pix3d.LineOffset = c.AreaChatbackOffsets
	c.ImageChatback.PlotSprite(0, 0)
	if c.ShowSocialInput {
		c.FontBold12.CentreString(40, 0, c.SocialMessage, 239)
		c.FontBold12.CentreString(60, 128, c.SocialInput+"*", 239)
	} else if c.ChatbackInputOpen {
		c.FontBold12.CentreString(40, 0, "Enter amount:", 239)
		c.FontBold12.CentreString(60, 128, c.ChatbackInput+"*", 239)
	} else if c.ModalMessage != "" {
		c.FontBold12.CentreString(40, 0, c.ModalMessage, 239)
		c.FontBold12.CentreString(60, 128, "Click to continue", 239)
	} else if c.ChatInterfaceID != -1 {
		c.DrawInterface(0, 0, component.Instances[c.ChatInterfaceID], 0)
	} else if c.StickyChatInterfaceID == -1 {
		var2 := c.FontPlain12
		var3 := 0
		pix2d.SetClipping(77, 0, 463, 0)
		for i := range 100 {
			if c.MessageText[i] != "" {
				var5 := c.MessageType[i]
				var6 := 70 - var3*14 + c.ChatScrollOffset
				if var5 == 0 {
					if var6 > 0 && var6 < 110 {
						var2.DrawString(4, var6, 0, c.MessageText[i])
					}
					var3++
				}
				if var5 == 1 {
					if var6 > 0 && var6 < 110 {
						var2.DrawString(4, var6, 0xFFFFFF, c.MessageSender[i]+":")
						var2.DrawString(var2.StringWidth(c.MessageSender[i])+12, var6, 0xFF, c.MessageText[i])
					}
					var3++
				}
				if var5 == 2 && (c.PublicChatSetting == 0 || c.PublicChatSetting == 1 && c.IsFriend(c.MessageSender[i])) {
					if var6 > 0 && var6 < 110 {
						var2.DrawString(4, var6, 0, c.MessageSender[i]+":")
						var2.DrawString(var2.StringWidth(c.MessageSender[i])+12, var6, 0xFF, c.MessageText[i])
					}
					var3++
				}
				if (var5 == 3 || var5 == 7) && c.SplitPrivateChat == 0 && (var5 == 7 || c.PrivateChatSetting == 0 || c.PrivateChatSetting == 1 && c.IsFriend(c.MessageSender[i])) {
					if var6 > 0 && var6 < 110 {
						var2.DrawString(4, var6, 0, "From "+c.MessageSender[i]+":")
						var2.DrawString(var2.StringWidth("From "+c.MessageSender[i])+12, var6, 8388608, c.MessageText[i])
					}
					var3++
				}
				if var5 == 4 && (c.TradeChatSetting == 0 || c.TradeChatSetting == 1 && c.IsFriend(c.MessageSender[i])) {
					if var6 > 0 && var6 < 110 {
						var2.DrawString(4, var6, 8388736, c.MessageSender[i]+" "+c.MessageText[i])
					}
					var3++
				}
				if var5 == 5 && c.SplitPrivateChat == 0 && c.PrivateChatSetting < 2 {
					if var6 > 0 && var6 < 110 {
						var2.DrawString(4, var6, 8388608, c.MessageText[i])
					}
					var3++
				}
				if var5 == 6 && c.SplitPrivateChat == 0 && c.PrivateChatSetting < 2 {
					if var6 > 0 && var6 < 110 {
						var2.DrawString(4, var6, 0, "To "+c.MessageSender[i]+":")
						var2.DrawString(var2.StringWidth("To "+c.MessageSender[i])+12, var6, 8388608, c.MessageText[i])
					}
					var3++
				}
				if var5 == 8 && (c.TradeChatSetting == 0 || c.TradeChatSetting == 1 && c.IsFriend(c.MessageSender[i])) {
					if var6 > 0 && var6 < 110 {
						var2.DrawString(4, var6, 13350793, c.MessageSender[i]+" "+c.MessageText[i])
					}
					var3++
				}
			}
		}
		pix2d.ResetClipping()
		c.ChatScrollHeight = var3*14 + 7
		c.ChatScrollHeight = max(c.ChatScrollHeight, 78)
		c.DrawScrollbar(463, 0, c.ChatScrollHeight-c.ChatScrollOffset-77, c.ChatScrollHeight, 77)
		var2.DrawString(4, 90, 0, jstring.FormatName(c.Username)+":")
		var2.DrawString(var2.StringWidth(c.Username+": ")+6, 90, 0xFF, c.ChatTyped+"*")
		pix2d.HLine(0, 77, 479, 0)
	} else {
		c.DrawInterface(0, 0, component.Instances[c.StickyChatInterfaceID], 0)
	}
	if c.MenuVisible && c.MenuArea == 2 {
		c.DrawMenu()
	}
	c.AreaViewport.Bind()
	pix3d.LineOffset = c.AreaViewportOffsets
	c.AreaChatback.Draw(22, 375)
}

func (c *Client) Read() (ok bool) {
	if c.Stream == nil {
		return false
	}
	// Java: read() (client.java:9316-10384). Java wraps the whole body in
	//   try { ... } catch (IOException) { tryReconnect() }
	//                catch (Exception)  { reporterror("T2 ...") ; logout() }
	// The catch (IOException) path is handled inline below: every Available/
	// ReadFully error routes to TryReconnect()+return true. The deferred
	// recover here reproduces catch (Exception): a panic while parsing or
	// dispatching a packet (e.g. a malformed/hostile packet) emits the "T2"
	// diagnostic byte-dump and logs out gracefully instead of crashing the
	// client goroutine.
	defer func() {
		if r := recover(); r != nil {
			// Java: client.java:10374-10382 (catch (Exception) var25).
			var3 := fmt.Sprintf("T2 - %d,%d,%d - %d,%d,%d - ",
				c.PacketType, c.LastPacketType1, c.LastPacketType2, c.PacketSize,
				c.SceneBaseTileX+c.LocalPlayer.PathTileX[0],
				c.SceneBaseTileZ+c.LocalPlayer.PathTileZ[0])
			// Java concatenates the signed byte in.data[var4]; In.Data is []byte
			// (unsigned) here, so cast to int8 to reproduce Java's signed output.
			for var4 := 0; var4 < c.PacketSize && var4 < 50; var4++ {
				var3 += fmt.Sprintf("%d,", int8(c.In.Data[var4]))
			}
			signlink.ReportErrorFunc(var3)
			c.Logout()
			ok = true
		}
	}()
	var2, err := c.Stream.Available()
	if err != nil {
		c.TryReconnect()
		return true
	}
	if var2 == 0 {
		return false
	}
	if c.PacketType == -1 {
		if err := c.Stream.ReadFully(c.In.Data, 0, 1); err != nil {
			c.TryReconnect()
			return true
		}
		c.PacketType = int(c.In.Data[0]) & 0xFF
		if c.RandomIn != nil {
			// Parens preserve Java precedence: `a - b & 0xFF` is `(a-b) & 0xFF`
			// in Java, but `a - (b & 0xFF)` in Go.
			c.PacketType = (c.PacketType - int(c.RandomIn.TakeNextValue())) & 0xFF
		}
		c.PacketSize = io.SERVERPROT_SIZES[c.PacketType]
		var2--
	}
	if c.PacketSize == -1 {
		if var2 <= 0 {
			return false
		}
		if err := c.Stream.ReadFully(c.In.Data, 0, 1); err != nil {
			c.TryReconnect()
			return true
		}
		c.PacketSize = int(c.In.Data[0]) & 0xFF
		var2--
	}
	if c.PacketSize == -2 {
		if var2 <= 1 {
			return false
		}
		if err := c.Stream.ReadFully(c.In.Data, 0, 2); err != nil {
			c.TryReconnect()
			return true
		}
		c.In.Pos = 0
		c.PacketSize = c.In.G2()
		var2 -= 2
	}
	if var2 < c.PacketSize {
		return false
	}
	c.In.Pos = 0
	if err := c.Stream.ReadFully(c.In.Data, 0, c.PacketSize); err != nil {
		c.TryReconnect()
		return true
	}
	c.IdleNetCycles = 0
	c.LastPacketType2 = c.LastPacketType1
	c.LastPacketType1 = c.LastPacketType0
	c.LastPacketType0 = c.PacketType

	// Java: opcode 4 — general chat / trade-req / duel-req (client.java:10139-10173)
	// strings.Index returns a byte offset; Java's indexOf returns a UTF-16
	// code-unit offset. Player names are ASCII-bound by the protocol, so for
	// valid inputs the substring split below is identical to Java's
	// substring(0, indexOf(":")). Fidelity-only divergence on non-ASCII names.
	if c.PacketType == 4 {
		var3 := c.In.GJStr()
		if strings.HasSuffix(var3, ":tradereq:") {
			var28 := var3[:strings.Index(var3, ":")]
			var30 := jstring.ToBase37(var28)
			var32 := false
			for i := range c.IgnoreCount {
				if c.IgnoreName37[i] == var30 {
					var32 = true
					break
				}
			}
			if !var32 && c.OverrideChat == 0 {
				c.AddMessage(4, "wishes to trade with you.", var28)
			}
		} else if strings.HasSuffix(var3, ":duelreq:") {
			var28 := var3[:strings.Index(var3, ":")]
			var30 := jstring.ToBase37(var28)
			var32 := false
			for i := range c.IgnoreCount {
				if c.IgnoreName37[i] == var30 {
					var32 = true
					break
				}
			}
			if !var32 && c.OverrideChat == 0 {
				c.AddMessage(8, "wishes to duel with you.", var28)
			}
		} else {
			c.AddMessage(0, var3, "")
		}
		c.PacketType = -1
		return true
	}
	// Java: opcode 41 — private message inbound (client.java:10034-10069)
	if c.PacketType == 41 {
		var39 := c.In.G8()
		var5 := c.In.G4()
		var6 := c.In.G1()
		var32 := false
		for i := range 100 {
			if c.MessageIds[i] == var5 {
				var32 = true
				break
			}
		}
		if var6 <= 1 {
			for i := range c.IgnoreCount {
				if c.IgnoreName37[i] == var39 {
					var32 = true
					break
				}
			}
		}
		if !var32 && c.OverrideChat == 0 {
			// Java: try { ... } catch (Exception) { signlink.reporterror("cde1"); }
			// (client.java:10054-10067) — a WordPack/WordFilter decode failure is
			// swallowed locally, logged as "cde1", and the read continues. The
			// closure-scoped recover keeps this from escalating to the outer T2
			// logout. The messageIds/privateMessageCount writes happen before the
			// decode (inside Java's try), so they persist even on failure.
			func() {
				defer func() {
					if recover() != nil {
						signlink.ReportErrorFunc("cde1")
					}
				}()
				c.MessageIds[c.PrivateMessageCount] = var5
				c.PrivateMessageCount = (c.PrivateMessageCount + 1) % 100
				var37 := wordpack.Unpack(c.In, c.PacketSize-13)
				var38 := wordfilter.Filter(var37)
				if var6 > 1 {
					c.AddMessage(7, var38, jstring.FormatName(jstring.FromBase37(var39)))
				} else {
					c.AddMessage(3, var38, jstring.FormatName(jstring.FromBase37(var39)))
				}
			}()
		}
		c.PacketType = -1
		return true
	}

	// Java: post-zone opcode dispatch (client.java:9697-10370). Unhandled
	// opcodes fall through to the catch-all at client.java:10371-10372.
	// Java: opcode 1 — NPC info (client.java:9454-9458)
	if c.PacketType == 1 {
		c.GetNpcPos(c.In, c.PacketSize)
		c.PacketType = -1
		return true
	}
	// Java: opcode 162 — player info: base coords + appended zone packets (client.java:10299-10307)
	if c.PacketType == 162 {
		c.BaseX = c.In.G1()
		c.BaseZ = c.In.G1()
		for c.In.Pos < c.PacketSize {
			var26 := c.In.G1()
			c.ReadZonePacket(c.In, var26)
		}
		c.PacketType = -1
		return true
	}
	// Java: opcode 184 — player info + scene build (client.java:10349-10369)
	if c.PacketType == 184 {
		c.GetPlayer(c.In, c.PacketSize)
		if c.SceneState == 1 {
			c.SceneState = 2
			world.LevelBuilt = c.CurrentLevel
			c.BuildScene()
		}
		if LowMemory && c.SceneState == 2 && world.LevelBuilt != c.CurrentLevel {
			c.AreaViewport.Bind()
			c.FontPlain12.CentreString(151, 0, "Loading - please wait.", 257)
			c.FontPlain12.CentreString(150, 0xFFFFFF, "Loading - please wait.", 256)
			c.presentLoadingMessage()
			world.LevelBuilt = c.CurrentLevel
			c.BuildScene()
		}
		if c.CurrentLevel != c.MinimapLevel && c.SceneState == 2 {
			c.MinimapLevel = c.CurrentLevel
			c.CreateMinimap(c.CurrentLevel)
		}
		c.PacketType = -1
		return true
	}
	// Java: zone-packet opcode group (client.java:9697-9700) — ten opcodes,
	// each a thin pass-through to readZonePacket which dispatches internally.
	if c.PacketType == 151 || c.PacketType == 23 || c.PacketType == 50 || c.PacketType == 191 || c.PacketType == 69 || c.PacketType == 49 || c.PacketType == 223 || c.PacketType == 42 || c.PacketType == 76 || c.PacketType == 59 {
		c.ReadZonePacket(c.In, c.PacketType)
		c.PacketType = -1
		return true
	}
	// Java: opcode 150 — varp set (byte) (client.java:9363-9377)
	if c.PacketType == 150 {
		var26 := c.In.G2()
		var52 := c.In.G1B()
		c.VarCache[var26] = int(var52)
		if c.Varps[var26] != int(var52) {
			c.Varps[var26] = int(var52)
			c.UpdateVarp(var26)
			c.RedrawSidebar = true
			if c.StickyChatInterfaceID != -1 {
				c.RedrawChatback = true
			}
		}
		c.PacketType = -1
		return true
	}
	// Java: opcode 152 — friend list add/update + bubble-sort (client.java:9383-9431)
	if c.PacketType == 152 {
		var39 := c.In.G8()
		var5 := c.In.G1()
		var44 := jstring.FormatName(jstring.FromBase37(var39))
		matched := false
		for var7 := range c.FriendCount {
			if var39 == c.FriendName37[var7] {
				if c.FriendWorld[var7] != var5 {
					c.FriendWorld[var7] = var5
					c.RedrawSidebar = true
					if var5 > 0 {
						c.AddMessage(5, var44+" has logged in.", "")
					}
					if var5 == 0 {
						c.AddMessage(5, var44+" has logged out.", "")
					}
				}
				matched = true
				break
			}
		}
		if !matched && c.FriendCount < 100 {
			c.FriendName37[c.FriendCount] = var39
			c.FriendName[c.FriendCount] = var44
			c.FriendWorld[c.FriendCount] = var5
			c.FriendCount++
			c.RedrawSidebar = true
		}
		var41 := false
		for !var41 {
			var41 = true
			for var9 := range c.FriendCount - 1 {
				if (c.FriendWorld[var9] != NodeID && c.FriendWorld[var9+1] == NodeID) || (c.FriendWorld[var9] == 0 && c.FriendWorld[var9+1] != 0) {
					var10 := c.FriendWorld[var9]
					c.FriendWorld[var9] = c.FriendWorld[var9+1]
					c.FriendWorld[var9+1] = var10
					var42 := c.FriendName[var9]
					c.FriendName[var9] = c.FriendName[var9+1]
					c.FriendName[var9+1] = var42
					var50 := c.FriendName37[var9]
					c.FriendName37[var9] = c.FriendName37[var9+1]
					c.FriendName37[var9+1] = var50
					c.RedrawSidebar = true
					var41 = false
				}
			}
		}
		c.PacketType = -1
		return true
	}
	// Java: opcode 43 — system update timer (client.java:9432-9436)
	if c.PacketType == 43 {
		c.SystemUpdateTimer = c.In.G2() * 30
		c.PacketType = -1
		return true
	}
	// Java: opcode 80 — scene map land cache save (client.java:9438-9452)
	if c.PacketType == 80 {
		var26 := c.In.G1()
		var4 := c.In.G1()
		var5 := -1
		for var6 := range len(c.SceneMapIndex) {
			if c.SceneMapIndex[var6] == (var26<<8)+var4 {
				var5 = var6
			}
		}
		if var5 != -1 {
			signlink.CacheSave(fmt.Sprintf("m%d_%d", var26, var4), c.SceneMapLandData[var5])
			c.SceneState = 1
		}
		c.PacketType = -1
		return true
	}
	// Java: opcode 237 — scene rebuild request + delta-shift entities (client.java:9460-9607)
	if c.PacketType == 237 {
		var26 := c.In.G2()
		var4 := c.In.G2()
		if c.SceneCenterZoneX == var26 && c.SceneCenterZoneZ == var4 && c.SceneState != 0 {
			c.PacketType = -1
			return true
		}
		c.SceneCenterZoneX = var26
		c.SceneCenterZoneZ = var4
		c.SceneBaseTileX = (c.SceneCenterZoneX - 6) * 8
		c.SceneBaseTileZ = (c.SceneCenterZoneZ - 6) * 8
		c.SceneState = 1
		c.AreaViewport.Bind()
		c.FontPlain12.CentreString(151, 0, "Loading - please wait.", 257)
		c.FontPlain12.CentreString(150, 0xFFFFFF, "Loading - please wait.", 256)
		c.presentLoadingMessage()
		signlink.LoopRate = 5
		var5 := (c.PacketSize - 2) / 10
		c.SceneMapLandData = make([][]byte, var5)
		c.SceneMapLocData = make([][]byte, var5)
		c.SceneMapIndex = make([]int, var5)
		c.Out.P1Isaac(150)
		c.Out.P1(0)
		var6 := 0
		for var7 := range var5 {
			var8 := c.In.G1()
			var9 := c.In.G1()
			var10 := c.In.G4()
			var11 := c.In.G4()
			c.SceneMapIndex[var7] = (var8 << 8) + var9
			var var12 []byte
			if var10 != 0 {
				var12 = signlink.CacheLoad(fmt.Sprintf("m%d_%d", var8, var9))
				if var12 != nil {
					if int(crc32.ChecksumIEEE(var12)) != var10 {
						var12 = nil
					}
				}
				if var12 == nil {
					c.SceneState = 0
					c.Out.P1(0)
					c.Out.P1(var8)
					c.Out.P1(var9)
					var6 += 3
				} else {
					c.SceneMapLandData[var7] = var12
				}
			}
			if var11 != 0 {
				var12 = signlink.CacheLoad(fmt.Sprintf("l%d_%d", var8, var9))
				if var12 != nil {
					if int(crc32.ChecksumIEEE(var12)) != var11 {
						var12 = nil
					}
				}
				if var12 == nil {
					c.SceneState = 0
					c.Out.P1(1)
					c.Out.P1(var8)
					c.Out.P1(var9)
					var6 += 3
				} else {
					c.SceneMapLocData[var7] = var12
				}
			}
		}
		c.Out.PSize1(var6)
		signlink.LoopRate = 50
		c.AreaViewport.Bind()
		if c.SceneState == 0 {
			c.FontPlain12.CentreString(166, 0, "Map area updated since last visit, so load will take longer this time only", 257)
			c.FontPlain12.CentreString(165, 0xFFFFFF, "Map area updated since last visit, so load will take longer this time only", 256)
		}
		c.presentLoadingMessage()
		var8 := c.SceneBaseTileX - c.MapLastBaseX
		var9 := c.SceneBaseTileZ - c.MapLastBaseZ
		c.MapLastBaseX = c.SceneBaseTileX
		c.MapLastBaseZ = c.SceneBaseTileZ
		for var10 := range 8192 {
			var40 := c.NPCs[var10]
			if var40 != nil {
				for var46 := range 10 {
					var40.PathTileX[var46] -= var8
					var40.PathTileZ[var46] -= var9
				}
				var40.X -= var8 * 128
				var40.Z -= var9 * 128
			}
		}
		for var11 := range c.MAX_PLAYER_COUNT {
			var48 := c.Players[var11]
			if var48 != nil {
				for var13 := range 10 {
					var48.PathTileX[var13] -= var8
					var48.PathTileZ[var13] -= var9
				}
				var48.X -= var8 * 128
				var48.Z -= var9 * 128
			}
		}
		// Java: byte var49/var45/var14 and var15/var16/var17 — step direction and bounds
		// for the four-layer object-stack shift. Stored as bytes in Java for compactness;
		// values are used as int loop control so we widen to int up front in Go.
		var49 := 0
		var45 := 104
		var14 := 1
		if var8 < 0 {
			var49 = 103
			var45 = -1
			var14 = -1
		}
		var15 := 0
		var16 := 104
		var17 := 1
		if var9 < 0 {
			var15 = 103
			var16 = -1
			var17 = -1
		}
		for var18 := var49; var18 != var45; var18 += var14 {
			for var19 := var15; var19 != var16; var19 += var17 {
				var20 := var18 + var8
				var21 := var19 + var9
				for var22 := range 4 {
					if var20 >= 0 && var21 >= 0 && var20 < 104 && var21 < 104 {
						c.LevelObjStacks[var22][var18][var19] = c.LevelObjStacks[var22][var20][var21]
					} else {
						c.LevelObjStacks[var22][var18][var19] = nil
					}
				}
			}
		}
		for var53 := c.SpawnedLocations.Head(); var53 != nil; var53 = c.SpawnedLocations.Next() {
			v := var53.Value
			v.X -= var8
			v.Z -= var9
			if v.X < 0 || v.Z < 0 || v.X >= 104 || v.Z >= 104 {
				var53.Unlink()
			}
		}
		if c.FlagSceneTileX != 0 {
			c.FlagSceneTileX -= var8
			c.FlagSceneTileZ -= var9
		}
		c.Cutscene = false
		c.PacketType = -1
		return true
	}
	// Java: opcode 197 — set component model to local player head (client.java:9609-9613)
	if c.PacketType == 197 {
		var26 := c.In.G2()
		component.Instances[var26].Model = c.LocalPlayer.GetHeadModel()
		c.PacketType = -1
		return true
	}
	// Java: opcode 25 — hint arrow / minimap marker (client.java:9615-9650)
	if c.PacketType == 25 {
		c.HintType = c.In.G1()
		if c.HintType == 1 {
			c.HintNPC = c.In.G2()
		}
		if c.HintType >= 2 && c.HintType <= 6 {
			if c.HintType == 2 {
				c.HintOffsetX = 64
				c.HintOffsetZ = 64
			}
			if c.HintType == 3 {
				c.HintOffsetX = 0
				c.HintOffsetZ = 64
			}
			if c.HintType == 4 {
				c.HintOffsetX = 128
				c.HintOffsetZ = 64
			}
			if c.HintType == 5 {
				c.HintOffsetX = 64
				c.HintOffsetZ = 0
			}
			if c.HintType == 6 {
				c.HintOffsetX = 64
				c.HintOffsetZ = 128
			}
			c.HintType = 2
			c.HintTileX = c.In.G2()
			c.HintTileZ = c.In.G2()
			c.HintHeight = c.In.G1()
		}
		if c.HintType == 10 {
			c.HintPlayer = c.In.G2()
		}
		c.PacketType = -1
		return true
	}
	// Java: opcode 54 — MIDI music change (client.java:9652-9664)
	if c.PacketType == 54 {
		var3 := c.In.GJStr()
		var4 := c.In.G4()
		var5 := c.In.G4()
		if var3 != c.CurrentMidi && c.MidiActive && !LowMemory {
			c.SetMidi(var4, var3, var5)
		}
		c.CurrentMidi = var3
		c.MidiCRC = var4
		c.MidiSize = var5
		c.NextMusicDelay = 0
		c.PacketType = -1
		return true
	}
	// Java: opcode 142 — server-initiated logout (client.java:9666-9670)
	if c.PacketType == 142 {
		c.Logout()
		c.PacketType = -1
		return false
	}
	// Java: opcode 20 — scene map loc cache save (client.java:9671-9685)
	if c.PacketType == 20 {
		var26 := c.In.G1()
		var4 := c.In.G1()
		var5 := -1
		for var6 := range len(c.SceneMapIndex) {
			if c.SceneMapIndex[var6] == (var26<<8)+var4 {
				var5 = var6
			}
		}
		if var5 != -1 {
			signlink.CacheSave(fmt.Sprintf("l%d_%d", var26, var4), c.SceneMapLocData[var5])
			c.SceneState = 1
		}
		c.PacketType = -1
		return true
	}
	// Java: opcode 19 — clear move-flag tile (client.java:9687-9690)
	if c.PacketType == 19 {
		c.FlagSceneTileX = 0
		c.PacketType = -1
		return true
	}
	// Java: opcode 139 — local player id (client.java:9692-9695)
	if c.PacketType == 139 {
		// Java: deob/client.java:9693 — `this.localPid = this.in.g2();`.
		// The prior Go port had a duplicate field `LocalPid` (lowercase
		// d) that this opcode wrote to, while every reader checked
		// `LocalPID` — so local-player identity never updated and every
		// "is this packet about me?" comparison silently failed.
		c.LocalPID = c.In.G2()
		c.PacketType = -1
		return true
	}
	// Java: opcode 28 — open viewport+sidebar interface (client.java:9702-9720)
	if c.PacketType == 28 {
		var26 := c.In.G2()
		var4 := c.In.G2()
		if c.ChatInterfaceID != -1 {
			c.ChatInterfaceID = -1
			c.RedrawChatback = true
		}
		if c.ChatbackInputOpen {
			c.ChatbackInputOpen = false
			c.RedrawChatback = true
		}
		c.ViewportInterfaceID = var26
		c.SidebarInterfaceID = var4
		c.RedrawSidebar = true
		c.RedrawSideIcons = true
		c.PressedContinueOption = false
		c.PacketType = -1
		return true
	}
	// Java: opcode 175 — varp set (g4) (client.java:9721-9735)
	if c.PacketType == 175 {
		var26 := c.In.G2()
		var4 := c.In.G4()
		c.VarCache[var26] = var4
		if c.Varps[var26] != var4 {
			c.Varps[var26] = var4
			c.UpdateVarp(var26)
			c.RedrawSidebar = true
			if c.StickyChatInterfaceID != -1 {
				c.RedrawChatback = true
			}
		}
		c.PacketType = -1
		return true
	}
	// Java: opcode 146 — component anim (client.java:9736-9742)
	if c.PacketType == 146 {
		var26 := c.In.G2()
		var4 := c.In.G2()
		component.Instances[var26].Anim = var4
		c.PacketType = -1
		return true
	}
	// Java: opcode 167 — tab interface assign (client.java:9743-9754)
	if c.PacketType == 167 {
		var26 := c.In.G2()
		var4 := c.In.G1()
		if var26 == 65535 {
			var26 = -1
		}
		c.TabInterfaceID[var4] = var26
		c.RedrawSidebar = true
		c.RedrawSideIcons = true
		c.PacketType = -1
		return true
	}
	// Java: opcode 220 — scene map loc-data inbound (client.java:9755-9774)
	if c.PacketType == 220 {
		var26 := c.In.G1()
		var4 := c.In.G1()
		var5 := c.In.G2()
		var6 := c.In.G2()
		var7 := -1
		for var8 := range len(c.SceneMapIndex) {
			if c.SceneMapIndex[var8] == (var26<<8)+var4 {
				var7 = var8
			}
		}
		if var7 != -1 {
			if c.SceneMapLocData[var7] == nil || len(c.SceneMapLocData[var7]) != var6 {
				c.SceneMapLocData[var7] = make([]byte, var6)
			}
			c.In.GData(c.PacketSize-6, var5, c.SceneMapLocData[var7])
		}
		c.PacketType = -1
		return true
	}
	// Java: opcode 133 — InputTracking.stop → outbound packet 81 (client.java:9775-9785)
	if c.PacketType == 133 {
		var51 := inputtracking.Stop()
		if var51 != nil {
			c.Out.P1Isaac(81)
			c.Out.P2(var51.Pos)
			c.Out.PData(var51.Data, var51.Pos, 0)
			var51.Release()
		}
		c.PacketType = -1
		return true
	}
	// Java: opcode 98 — inventory slot full update (client.java:9787-9806)
	if c.PacketType == 98 {
		c.RedrawSidebar = true
		var26 := c.In.G2()
		var27 := component.Instances[var26]
		var5 := c.In.G1()
		for var6 := range var5 {
			var27.InvSlotObjId[var6] = c.In.G2()
			var7 := c.In.G1()
			if var7 == 255 {
				var7 = c.In.G4()
			}
			var27.InvSlotObjCount[var6] = var7
		}
		for var7 := var5; var7 < len(var27.InvSlotObjId); var7++ {
			var27.InvSlotObjId[var7] = 0
			var27.InvSlotObjCount[var7] = 0
		}
		c.PacketType = -1
		return true
	}
	// Java: opcode 226 — InputTracking.setEnabled (client.java:9807-9811)
	if c.PacketType == 226 {
		inputtracking.SetEnabled()
		c.PacketType = -1
		return true
	}
	// Java: opcode 243 — open chatback input prompt (client.java:9812-9819)
	if c.PacketType == 243 {
		c.ShowSocialInput = false
		c.ChatbackInputOpen = true
		c.ChatbackInput = ""
		c.RedrawChatback = true
		c.PacketType = -1
		return true
	}
	// Java: opcode 15 — clear inventory component (client.java:9820-9829)
	if c.PacketType == 15 {
		var26 := c.In.G2()
		var27 := component.Instances[var26]
		for var5 := range len(var27.InvSlotObjId) {
			var27.InvSlotObjId[var5] = -1
			var27.InvSlotObjId[var5] = 0
		}
		c.PacketType = -1
		return true
	}
	// Java: opcode 140 — last-login info (client.java:9830-9853)
	if c.PacketType == 140 {
		c.LastAddress = c.In.G4()
		c.DaysSinceLastLogin = c.In.G2()
		c.DaysSinceRecoveriesChanged = c.In.G1()
		c.UnreadMessages = c.In.G2()
		if c.LastAddress != 0 && c.ViewportInterfaceID == -1 {
			signlink.DNSLookup(jstring.FormatIPv4(int32(c.LastAddress)))
			c.CloseInterfaces()
			var47 := 650 // Java: short var47
			if c.DaysSinceRecoveriesChanged != 201 {
				var47 = 655
			}
			c.ReportAbuseInput = ""
			c.ReportAbuseMuteOption = false
			for var4 := range len(component.Instances) {
				if component.Instances[var4] != nil && component.Instances[var4].ClientCode == var47 {
					c.ViewportInterfaceID = component.Instances[var4].Layer
					break
				}
			}
		}
		c.PacketType = -1
		return true
	}
	// Java: opcode 126 — flashing tab (client.java:9854-9866)
	if c.PacketType == 126 {
		c.FlashingTab = c.In.G1()
		if c.FlashingTab == c.SelectedTab {
			if c.FlashingTab == 3 {
				c.SelectedTab = 1
			} else {
				c.SelectedTab = 3
			}
			c.RedrawSidebar = true
		}
		c.PacketType = -1
		return true
	}
	// Java: opcode 212 — MIDI music push (bzip2) (client.java:9867-9878)
	if c.PacketType == 212 {
		if c.MidiActive && !LowMemory {
			var26 := c.In.G2()
			var4 := c.In.G4()
			var5 := c.PacketSize - 6
			var35 := make([]byte, var4)
			bzip2.Read(var35, var4, c.In.Data, var5, c.In.Pos)
			c.SaveMidi(var35, var4, false)
			c.NextMusicDelay = var26
		}
		c.PacketType = -1
		return true
	}
	// Java: opcode 254 — multizone flag (client.java:9880-9884)
	if c.PacketType == 254 {
		c.InMultizone = c.In.G1()
		c.PacketType = -1
		return true
	}
	// Java: opcode 12 — queue wave sound (client.java:9885-9897)
	if c.PacketType == 12 {
		var26 := c.In.G2()
		var4 := c.In.G1()
		var5 := c.In.G2()
		if c.WaveEnabled && !LowMemory && c.WaveCount < 50 {
			c.WaveIDs[c.WaveCount] = var26
			c.WaveLoops[c.WaveCount] = var4
			c.WaveDelay[c.WaveCount] = var5 + wave.Delays[var26]
			c.WaveCount++
		}
		c.PacketType = -1
		return true
	}
	// Java: opcode 204 — component model = npc head (client.java:9898-9904)
	if c.PacketType == 204 {
		var26 := c.In.G2()
		var4 := c.In.G2()
		var33 := npctype.Get(var4)
		component.Instances[var26].Model = var33.GetHeadModel()
		c.PacketType = -1
		return true
	}
	// Java: opcode 7 — scene base coords (client.java:9906-9910)
	if c.PacketType == 7 {
		c.BaseX = c.In.G1()
		c.BaseZ = c.In.G1()
		c.PacketType = -1
		return true
	}
	// Java: opcode 103 — recolor component model (client.java:9913-9923)
	if c.PacketType == 103 {
		var26 := c.In.G2()
		var4 := c.In.G2()
		var5 := c.In.G2()
		var34 := component.Instances[var26]
		var43 := var34.Model
		if var43 != nil {
			var43.Recolor(var4, var5)
		}
		c.PacketType = -1
		return true
	}
	// Java: opcode 32 — privacy chat settings (client.java:9925-9932)
	if c.PacketType == 32 {
		c.PublicChatSetting = c.In.G1()
		c.PrivateChatSetting = c.In.G1()
		c.TradeChatSetting = c.In.G1()
		c.RedrawPrivacySettings = true
		c.RedrawChatback = true
		c.PacketType = -1
		return true
	}
	// Java: opcode 195 — open sidebar interface only (client.java:9934-9951)
	if c.PacketType == 195 {
		var26 := c.In.G2()
		c.ResetInterfaceAnimation(var26)
		if c.ChatInterfaceID != -1 {
			c.ChatInterfaceID = -1
			c.RedrawChatback = true
		}
		if c.ChatbackInputOpen {
			c.ChatbackInputOpen = false
			c.RedrawChatback = true
		}
		c.SidebarInterfaceID = var26
		c.RedrawSidebar = true
		c.RedrawSideIcons = true
		c.ViewportInterfaceID = -1
		c.PressedContinueOption = false
		c.PacketType = -1
		return true
	}
	// Java: opcode 14 — open chat interface only (client.java:9953-9966)
	if c.PacketType == 14 {
		var26 := c.In.G2()
		c.ResetInterfaceAnimation(var26)
		if c.SidebarInterfaceID != -1 {
			c.SidebarInterfaceID = -1
			c.RedrawSidebar = true
			c.RedrawSideIcons = true
		}
		c.ChatInterfaceID = var26
		c.RedrawChatback = true
		c.ViewportInterfaceID = -1
		c.PressedContinueOption = false
		c.PacketType = -1
		return true
	}
	// Java: opcode 209 — component x/y position (client.java:9968-9976)
	if c.PacketType == 209 {
		var26 := c.In.G2()
		var4 := c.In.G2B()
		var5 := c.In.G2B()
		var34 := component.Instances[var26]
		var34.X = var4
		var34.Y = var5
		c.PacketType = -1
		return true
	}
	// Java: opcode 3 — cutscene camera init (client.java:9978-9991)
	if c.PacketType == 3 {
		c.Cutscene = true
		c.CutsceneSrcLocalTileX = c.In.G1()
		c.CutsceneSrcLocalTileZ = c.In.G1()
		c.CutsceneSrcHeight = c.In.G2()
		c.CutsceneMoveSpeed = c.In.G1()
		c.CutsceneMoveAcceleration = c.In.G1()
		if c.CutsceneMoveAcceleration >= 100 {
			c.CameraX = c.CutsceneSrcLocalTileX*128 + 64
			c.CameraZ = c.CutsceneSrcLocalTileZ*128 + 64
			c.CameraY = c.GetHeightMapY(c.CurrentLevel, c.CutsceneSrcLocalTileX, c.CutsceneSrcLocalTileZ) - c.CutsceneSrcHeight
		}
		c.PacketType = -1
		return true
	}
	// Java: opcode 135 — clear obj-stacks + reapply spawned locs in 8x8 region (client.java:9993-10011)
	if c.PacketType == 135 {
		c.BaseX = c.In.G1()
		c.BaseZ = c.In.G1()
		for var26 := c.BaseX; var26 < c.BaseX+8; var26++ {
			for var4 := c.BaseZ; var4 < c.BaseZ+8; var4++ {
				if c.LevelObjStacks[c.CurrentLevel][var26][var4] != nil {
					c.LevelObjStacks[c.CurrentLevel][var26][var4] = nil
					c.SortObjStacks(var26, var4)
				}
			}
		}
		for var36 := c.SpawnedLocations.Head(); var36 != nil; var36 = c.SpawnedLocations.Next() {
			v := var36.Value
			if v.X >= c.BaseX && v.X < c.BaseX+8 && v.Z >= c.BaseZ && v.Z < c.BaseZ+8 && v.Plane == c.CurrentLevel {
				c.AddLoc(v.LastAngle, v.X, v.Z, v.Layer, v.LastLocIndex, v.LastShape, v.Plane)
				var36.Unlink()
			}
		}
		c.PacketType = -1
		return true
	}
	// Java: opcode 132 — scene map land-data inbound (client.java:10013-10031)
	if c.PacketType == 132 {
		var26 := c.In.G1()
		var4 := c.In.G1()
		var5 := c.In.G2()
		var6 := c.In.G2()
		var7 := -1
		for var8 := range len(c.SceneMapIndex) {
			if c.SceneMapIndex[var8] == (var26<<8)+var4 {
				var7 = var8
			}
		}
		if var7 != -1 {
			if c.SceneMapLandData[var7] == nil || len(c.SceneMapLandData[var7]) != var6 {
				c.SceneMapLandData[var7] = make([]byte, var6)
			}
			c.In.GData(c.PacketSize-6, var5, c.SceneMapLandData[var7])
		}
		c.PacketType = -1
		return true
	}
	// Java: opcode 193 — flush varCache→varps (client.java:10071-10080)
	if c.PacketType == 193 {
		for var26 := range len(c.Varps) {
			if c.Varps[var26] != c.VarCache[var26] {
				c.Varps[var26] = c.VarCache[var26]
				c.UpdateVarp(var26)
				c.RedrawSidebar = true
			}
		}
		c.PacketType = -1
		return true
	}
	// Java: opcode 87 — component model = new Model(id) (client.java:10082-10087)
	if c.PacketType == 87 {
		var26 := c.In.G2()
		var4 := c.In.G2()
		component.Instances[var26].Model = model.NewModel1(var4)
		c.PacketType = -1
		return true
	}
	// Java: opcode 185 — sticky chat interface (client.java:10089-10094)
	if c.PacketType == 185 {
		var26 := c.In.G2B()
		c.StickyChatInterfaceID = var26
		c.RedrawChatback = true
		c.PacketType = -1
		return true
	}
	// Java: opcode 68 — energy update (client.java:10096-10102)
	if c.PacketType == 68 {
		if c.SelectedTab == 12 {
			c.RedrawSidebar = true
		}
		c.Energy = c.In.G1()
		c.PacketType = -1
		return true
	}
	// Java: opcode 74 — cutscene camera-target init (client.java:10104-10129)
	if c.PacketType == 74 {
		c.Cutscene = true
		c.CutsceneDstLocalTileX = c.In.G1()
		c.CutsceneDstLocalTileZ = c.In.G1()
		c.CutsceneDstHeight = c.In.G2()
		c.CutsceneRotateSpeed = c.In.G1()
		c.CutsceneRotateAcceleration = c.In.G1()
		if c.CutsceneRotateAcceleration >= 100 {
			var26 := c.CutsceneDstLocalTileX*128 + 64
			var4 := c.CutsceneDstLocalTileZ*128 + 64
			var5 := c.GetHeightMapY(c.CurrentLevel, c.CutsceneDstLocalTileX, c.CutsceneDstLocalTileZ) - c.CutsceneDstHeight
			var6 := var26 - c.CameraX
			var7 := var5 - c.CameraY
			var8 := var4 - c.CameraZ
			var9 := int(math.Sqrt(float64(var6*var6 + var8*var8)))
			c.CameraPitch = int(math.Atan2(float64(var7), float64(var9))*325.949) & 0x7FF
			c.CameraYaw = int(math.Atan2(float64(var6), float64(var8))*-325.949) & 0x7FF
			if c.CameraPitch < 128 {
				c.CameraPitch = 128
			}
			if c.CameraPitch > 383 {
				c.CameraPitch = 383
			}
		}
		c.PacketType = -1
		return true
	}
	// Java: opcode 84 — selected sidebar tab (client.java:10131-10136)
	if c.PacketType == 84 {
		c.SelectedTab = c.In.G1()
		c.RedrawSidebar = true
		c.RedrawSideIcons = true
		c.PacketType = -1
		return true
	}
	// Java: opcode 46 — component obj-icon model (client.java:10174-10184)
	if c.PacketType == 46 {
		var26 := c.In.G2()
		var4 := c.In.G2()
		var5 := c.In.G2()
		var31 := objtype.Get(var4)
		component.Instances[var26].Model = var31.GetInterfaceModel(50)
		component.Instances[var26].Xan = var31.Xan2D
		component.Instances[var26].Yan = var31.Yan2D
		component.Instances[var26].Zoom = var31.Zoom2D * 100 / var5
		c.PacketType = -1
		return true
	}
	// Java: opcode 168 — open viewport interface only (client.java:10186-10205)
	if c.PacketType == 168 {
		var26 := c.In.G2()
		c.ResetInterfaceAnimation(var26)
		if c.SidebarInterfaceID != -1 {
			c.SidebarInterfaceID = -1
			c.RedrawSidebar = true
			c.RedrawSideIcons = true
		}
		if c.ChatInterfaceID != -1 {
			c.ChatInterfaceID = -1
			c.RedrawChatback = true
		}
		if c.ChatbackInputOpen {
			c.ChatbackInputOpen = false
			c.RedrawChatback = true
		}
		c.ViewportInterfaceID = var26
		c.PressedContinueOption = false
		c.PacketType = -1
		return true
	}
	// Java: opcode 2 — component RGB15→RGB24 colour (client.java:10207-10215)
	if c.PacketType == 2 {
		var26 := c.In.G2()
		var4 := c.In.G2()
		var5 := var4 >> 10 & 0x1F
		var6 := var4 >> 5 & 0x1F
		var7 := var4 & 0x1F
		component.Instances[var26].Colour = (var5 << 19) + (var6 << 11) + (var7 << 3)
		c.PacketType = -1
		return true
	}
	// Java: opcode 136 — clear all primarySeqIds (client.java:10217-10229)
	if c.PacketType == 136 {
		for var26 := range len(c.Players) {
			if c.Players[var26] != nil {
				c.Players[var26].PrimarySeqID = -1
			}
		}
		for var4 := range len(c.NPCs) {
			if c.NPCs[var4] != nil {
				c.NPCs[var4].PrimarySeqID = -1
			}
		}
		c.PacketType = -1
		return true
	}
	// Java: opcode 26 — component hide flag (client.java:10231-10236)
	if c.PacketType == 26 {
		var26 := c.In.G2()
		var29 := c.In.G1() == 1
		component.Instances[var26].Hide = var29
		c.PacketType = -1
		return true
	}
	// Java: opcode 21 — ignore list bulk update (client.java:10238-10244)
	if c.PacketType == 21 {
		c.IgnoreCount = c.PacketSize / 8
		for var26 := range c.IgnoreCount {
			c.IgnoreName37[var26] = c.In.G8()
		}
		c.PacketType = -1
		return true
	}
	// Java: opcode 239 — cutscene end / clear camera modifiers (client.java:10246-10252)
	if c.PacketType == 239 {
		c.Cutscene = false
		for var26 := range 5 {
			c.CameraModifierEnabled[var26] = false
		}
		c.PacketType = -1
		return true
	}
	// Java: opcode 129 — close all interfaces (client.java:10254-10271)
	if c.PacketType == 129 {
		if c.SidebarInterfaceID != -1 {
			c.SidebarInterfaceID = -1
			c.RedrawSidebar = true
			c.RedrawSideIcons = true
		}
		if c.ChatInterfaceID != -1 {
			c.ChatInterfaceID = -1
			c.RedrawChatback = true
		}
		if c.ChatbackInputOpen {
			c.ChatbackInputOpen = false
			c.RedrawChatback = true
		}
		c.ViewportInterfaceID = -1
		c.PressedContinueOption = false
		c.PacketType = -1
		return true
	}
	// Java: opcode 201 — component text (client.java:10273-10281)
	if c.PacketType == 201 {
		var26 := c.In.G2()
		var28 := c.In.GJStr()
		component.Instances[var26].Text = var28
		if component.Instances[var26].Layer == c.TabInterfaceID[c.SelectedTab] {
			c.RedrawSidebar = true
		}
		c.PacketType = -1
		return true
	}
	// Java: opcode 44 — skill XP/level update (client.java:10283-10297)
	if c.PacketType == 44 {
		c.RedrawSidebar = true
		var26 := c.In.G1()
		var4 := c.In.G4()
		var5 := c.In.G1()
		c.SkillExperience[var26] = var4
		c.SkillLevel[var26] = var5
		c.SkillBaseLevel[var26] = 1
		for var6 := range 98 {
			if var4 >= LevelExperience[var6] {
				c.SkillBaseLevel[var26] = var6 + 2
			}
		}
		c.PacketType = -1
		return true
	}
	// Java: opcode 22 — weight carried (client.java:10309-10315)
	if c.PacketType == 22 {
		if c.SelectedTab == 12 {
			c.RedrawSidebar = true
		}
		c.WeightCarried = c.In.G2B()
		c.PacketType = -1
		return true
	}
	// Java: opcode 13 — camera shake/wobble modifier (client.java:10317-10328)
	if c.PacketType == 13 {
		var26 := c.In.G1()
		var4 := c.In.G1()
		var5 := c.In.G1()
		var6 := c.In.G1()
		c.CameraModifierEnabled[var26] = true
		c.CameraModifierJitter[var26] = var4
		c.CameraModifierWobbleScale[var26] = var5
		c.CameraModifierWobbleSpeed[var26] = var6
		c.CameraModifierCycle[var26] = 0
		c.PacketType = -1
		return true
	}
	// Java: opcode 213 — inventory slot partial update (client.java:10330-10347)
	if c.PacketType == 213 {
		c.RedrawSidebar = true
		var26 := c.In.G2()
		var27 := component.Instances[var26]
		for c.In.Pos < c.PacketSize {
			var5 := c.In.G1()
			var6 := c.In.G2()
			var7 := c.In.G1()
			if var7 == 255 {
				var7 = c.In.G4()
			}
			if var5 >= 0 && var5 < len(var27.InvSlotObjId) {
				var27.InvSlotObjId[var5] = var6
				var27.InvSlotObjCount[var5] = var7
			}
		}
		c.PacketType = -1
		return true
	}

	signlink.ReportErrorFunc(fmt.Sprintf("T1 - %d,%d - %d,%d", c.PacketType, c.PacketSize, c.LastPacketType1, c.LastPacketType2))
	c.Logout()
	return true
}

func (c *Client) DrawSidebar() {
	// Pixel repaint is gated on RedrawSidebar (expensive: interface
	// tree walk + pix3d/pix2d operations). The blit via PixMap.Draw
	// always runs. Pre-fix this whole function was wrapped in `if
	// RedrawSidebar` at the call site, relying on Java/AWT's retained
	// back buffer for the no-redraw frames; PixMap.Draw →
	// platform.Active.Blit must re-issue each frame.
	if c.RedrawSidebar {
		c.RedrawSidebar = false
		c.AreaSidebar.Bind()
		pix3d.LineOffset = c.AreaSidebarOffsets
		c.ImageInvback.PlotSprite(0, 0)
		if c.SidebarInterfaceID != -1 {
			c.DrawInterface(0, 0, component.Instances[c.SidebarInterfaceID], 0)
		} else if c.TabInterfaceID[c.SelectedTab] != -1 {
			c.DrawInterface(0, 0, component.Instances[c.TabInterfaceID[c.SelectedTab]], 0)
		}
		if c.MenuVisible && c.MenuArea == 1 {
			c.DrawMenu()
		}
		c.AreaViewport.Bind()
		pix3d.LineOffset = c.AreaViewportOffsets
	}
	c.AreaSidebar.Draw(562, 231)
}

func (c *Client) IsFriend(arg1 string) bool {
	if arg1 == "" {
		return false
	}
	for i := range c.FriendCount {
		if strings.EqualFold(arg1, c.FriendName[i]) {
			return true
		}
	}
	if strings.EqualFold(arg1, c.LocalPlayer.Name) { //nolint:staticcheck // S1008: explicit if/return mirrors the Java method structure
		return true
	}
	return false
}

// MISSING: init() only used by java applets

func (c *Client) GetPlayerExtended2(arg1 int, arg2 int, arg3 *io.Packet, arg4 *playerentity.PlayerEntity) {
	var6 := 0
	if arg2&0x1 == 1 {
		var6 = arg3.G1()
		var7 := make([]byte, var6)
		var8 := io.NewPacket(var7)
		arg3.GData(var6, 0, var7)
		c.PlayerAppearanceBuffer[arg1] = var8
		arg4.Read(var8)
	}
	var15 := 0
	if arg2&0x2 == 2 {
		var6 = arg3.G2()
		if var6 == 0xFFFF {
			var6 = -1
		}
		if var6 == arg4.PrimarySeqID {
			arg4.PrimarySeqLoop = 0
		}
		var15 = arg3.G1()
		if var6 == -1 || arg4.PrimarySeqID == -1 || seqtype.Instances[var6].Priority > seqtype.Instances[arg4.PrimarySeqID].Priority || seqtype.Instances[arg4.PrimarySeqID].Priority == 0 {
			arg4.PrimarySeqID = var6
			arg4.PrimarySeqFrame = 0
			arg4.PrimarySeqCycle = 0
			arg4.PrimarySeqDelay = var15
			arg4.PrimarySeqLoop = 0
		}
	}
	if arg2&0x4 == 4 {
		arg4.TargetID = arg3.G2()
		if arg4.TargetID == 0xFFFF {
			arg4.TargetID = -1
		}
	}
	if arg2&0x8 == 8 {
		arg4.Chat = arg3.GJStr()
		arg4.ChatColor = 0
		arg4.ChatStyle = 0
		arg4.ChatTimer = 150
		c.AddMessage(2, arg4.Chat, arg4.Name)
	}
	if arg2&0x10 == 16 {
		arg4.Damage = arg3.G1()
		arg4.DamageType = arg3.G1()
		arg4.CombatCycle = clientextras.LoopCycle + 400
		arg4.Health = arg3.G1()
		arg4.TotalHealth = arg3.G1()
	}
	if arg2&0x20 == 32 {
		arg4.TargetTileX = arg3.G2()
		arg4.TargetTileZ = arg3.G2()
	}
	if arg2&0x40 == 64 {
		var6 = arg3.G2()
		var15 = arg3.G1()
		var16 := arg3.G1()
		var9 := arg3.Pos
		if arg4.Name != "" {
			var10 := jstring.ToBase37(arg4.Name)
			var12 := false
			if var15 <= 1 {
				for i := range c.IgnoreCount {
					if c.IgnoreName37[i] == var10 {
						var12 = true
						break
					}
				}
			}
			if !var12 && c.OverrideChat == 0 {
				// Java: try { ... } catch (Exception) { signlink.reporterror("cde2"); }
				// (client.java:10513-10528) — a WordPack/WordFilter decode failure
				// is swallowed locally, logged as "cde2", and processing continues.
				// The closure-scoped recover keeps this from escalating to the outer
				// T2 logout. Note arg3.Pos = var9 + var16 below is OUTSIDE Java's try
				// and must run on failure too, so it stays outside this closure.
				func() {
					defer func() {
						if recover() != nil {
							signlink.ReportErrorFunc("cde2")
						}
					}()
					var17 := wordpack.Unpack(arg3, var16)
					var18 := wordfilter.Filter(var17)
					arg4.Chat = var18
					arg4.ChatColor = var6 >> 8
					arg4.ChatStyle = var6 & 0xFF
					arg4.ChatTimer = 150
					if var15 > 1 {
						c.AddMessage(1, var18, arg4.Name)
					} else {
						c.AddMessage(2, var18, arg4.Name)
					}
				}()
			}
		}
		arg3.Pos = var9 + var16
	}
	if arg2&0x100 == 256 {
		arg4.SpotanimID = arg3.G2()
		var6 = arg3.G4()
		arg4.SpotanimOffset = var6 >> 16
		arg4.SpotanimLastCycle = clientextras.LoopCycle + (var6 & 0xFFFF)
		arg4.SpotanimFrame = 0
		arg4.SpotanimCycle = 0
		if arg4.SpotanimLastCycle > clientextras.LoopCycle {
			arg4.SpotanimFrame = -1
		}
		if arg4.SpotanimID == 0xFFFF {
			arg4.SpotanimID = -1
		}
	}
	if arg2&0x200 != 512 {
		return
	}
	arg4.ForceMoveStartSceneTileX = arg3.G1()
	arg4.ForceMoveStartSceneTileZ = arg3.G1()
	arg4.ForceMoveEndSceneTileX = arg3.G1()
	arg4.ForceMoveEndSceneTileZ = arg3.G1()
	arg4.ForceMoveEndCycle = arg3.G2() + clientextras.LoopCycle
	arg4.ForceMoveStartCycle = arg3.G2() + clientextras.LoopCycle
	arg4.ForceMoveFaceDirection = arg3.G1()
	arg4.PathLength = 0
	arg4.PathTileX[0] = arg4.ForceMoveEndSceneTileX
	arg4.PathTileZ[0] = arg4.ForceMoveEndSceneTileZ
}

func (c *Client) DrawProgress(message string, percent int) {
	c.LoadTitle()

	if c.JagTitle == nil {
		c.DrawProgressGameShell(message, percent)
		return
	}

	// Out-of-band repaint during boot (called from RunShell's prologue and
	// GetJagFile's retry loop before the main loop takes over). Present
	// explicitly since we're not in the per-frame Draw call.
	c.present(func() {
		c.ImageTitle4.Bind()

		x := 360
		y := 200

		offsetY := 20
		c.FontBold12.CentreString(y/2-26-offsetY, 0xFFFFFF, "RuneScape is loading - please wait...", x/2)

		midY := y/2 - 18 - offsetY
		pix2d.DrawRect(x/2-152, 0x8C1111, 34, midY, 304)
		pix2d.DrawRect(x/2-151, 0, 32, midY+1, 302)
		pix2d.FillRect(midY+2, x/2-150, 0x8C1111, percent*3, 30)
		pix2d.FillRect(midY+2, x/2-150+percent*3, 0, 300-percent*3, 30)
		c.FontBold12.CentreString(y/2+5-offsetY, 0xFFFFFF, message, x/2)

		c.ImageTitle4.Draw(214, 186)
		// Always upload the static title tiles + flame tiles.
		//
		// ImageTitle0 / ImageTitle1 are dual-purpose: they hold the
		// static title flame imagery when flames are inactive, and are
		// overwritten by DrawFlames with animated pixels when active.
		// Either way the buffer content is correct, so upload
		// unconditionally — the prior `if !c.FlameActive` skip relied on
		// DrawFlames also issuing the upload (which it no longer does
		// post-refactor) and produced white rectangles at (0,0) /
		// (661,0) during boot when FlameActive is true.
		c.RedrawFrame = false
		// flameMu: ImageTitle0/1 buffers are written by the RunFlames goroutine.
		c.flameMu.Lock()
		c.ImageTitle0.Draw(0, 0)
		c.ImageTitle1.Draw(661, 0)
		c.flameMu.Unlock()
		c.ImageTitle2.Draw(128, 0)
		c.ImageTitle3.Draw(214, 386)
		c.ImageTitle5.Draw(0, 265)
		c.ImageTitle6.Draw(574, 265)
		c.ImageTitle7.Draw(128, 186)
		c.ImageTitle8.Draw(574, 186)
	})
}

// ensureOverlay lazily allocates the fullscreen overlay PixMap used by
// DrawError and DrawProgressGameShell. Lazy because ScreenWidth/Height
// are set before RunShell runs (by the platform backend / caller), after
// NewClient returns; the overlay is allocated lazily on first use because
// NewClient runs before a backend or texture exists. If the screen size
// changed since the last allocation (currently unreachable but cheap to
// guard), reallocate.
func (c *Client) ensureOverlay() {
	if c.OverlayPixMap == nil ||
		c.OverlayPixMap.Width != c.ScreenWidth ||
		c.OverlayPixMap.Height != c.ScreenHeight {
		c.OverlayPixMap = pixmap.NewPixMap(c.ScreenWidth, c.ScreenHeight)
	}
}
