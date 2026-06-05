package entity

import (
	"github.com/zsrv/goscape-client/pkg/jagex2/config/seqtype"
)

type ClientEntity struct {
	X               int
	Z               int
	Yaw             int
	SeqStretches    bool
	Size            int
	SeqStandID      int
	SeqTurnID       int
	SeqWalkID       int
	SeqTurnAroundID int
	SeqTurnLeftID   int
	SeqTurnRightId  int
	SeqRunID        int
	Chat            string
	ChatTimer       int
	ChatColor       int
	ChatStyle       int
	// Java: ClientEntity damage/damageType/damageCycle = new int[4]
	// (ClientEntity.java:98-104, new in 244) — up to four simultaneous
	// hitsplats, each expiring 70 cycles after Hit() records it.
	Damage      [4]int
	DamageType  [4]int
	DamageCycle [4]int
	CombatCycle int
	Health      int
	DstYaw      int
	// Java: turnspeed = 32 (ClientEntity.java:77 @2e62978) — NEW in 254;
	// per-entity yaw step per cycle. NPCs overwrite it from
	// NpcType.turnspeed in getNpcPosNewVis/Extended; 0 disables turning.
	TurnSpeed  int
	PathLength int
	// Java: preanimRouteLength (z.ob, new in 244) — the route length captured
	// when an ANIM block accepts a new primary seq; gates whether the seq
	// blocks movement (preanim_move) vs plays after it (postanim_mode).
	PreanimRouteLength       int
	PathTileX                []int
	PathTileZ                []int
	PathRunning              []bool
	SeqTrigger               int
	TargetID                 int
	SecondarySeqID           int
	PrimarySeqID             int
	SpotanimID               int
	TotalHealth              int
	TargetTileX              int
	TargetTileZ              int
	SecondarySeqFrame        int
	SecondarySeqCycle        int
	PrimarySeqFrame          int
	PrimarySeqCycle          int
	PrimarySeqDelay          int
	PrimarySeqLoop           int
	SpotanimFrame            int
	SpotanimCycle            int
	SpotanimLastCycle        int
	SpotanimOffset           int
	ForceMoveStartSceneTileX int
	ForceMoveEndSceneTileX   int
	ForceMoveStartSceneTileZ int
	ForceMoveEndSceneTileZ   int
	ForceMoveEndCycle        int
	ForceMoveStartCycle      int
	ForceMoveFaceDirection   int
	Cycle                    int
	Height                   int
}

func NewClientEntity() *ClientEntity {
	return &ClientEntity{
		Size:            1,
		Height:          200, // Java: ClientEntity.java:71 @2e62978 — default 0→200 in 254
		TurnSpeed:       32,  // Java: ClientEntity.java:77 @2e62978 (NEW in 254)
		SeqStandID:      -1,
		SeqTurnID:       -1,
		SeqWalkID:       -1,
		SeqTurnAroundID: -1,
		SeqTurnLeftID:   -1,
		SeqTurnRightId:  -1,
		SeqRunID:        -1,
		ChatTimer:       100,
		CombatCycle:     -1000,
		PathTileX:       make([]int, 10),
		PathTileZ:       make([]int, 10),
		PathRunning:     make([]bool, 10),
		TargetID:        -1,
		SecondarySeqID:  -1,
		PrimarySeqID:    -1,
		SpotanimID:      -1,
	}
}

// AddHitmark records a hitsplat in the first free of the four damage slots;
// each slot lives for 70 cycles. Java: ClientEntity.addHitmark
// (ClientEntity.java:266-275 @32f3062; was `hit` in 254). 274's real change:
// the current cycle arrives as a parameter (callers pass loopCycle) instead
// of the method reading the Client global — which also frees this package
// from the clientextras import. Java args: arg0=cycle, arg2=value, arg3=type.
func (e *ClientEntity) AddHitmark(cycle int, value int, damageType int) {
	for var5 := range 4 {
		if e.DamageCycle[var5] <= cycle {
			e.Damage[var5] = value
			e.DamageType[var5] = damageType
			e.DamageCycle[var5] = cycle + 70
			return
		}
	}
}

// AbortRoute resets the walk route and its preanim capture.
// Java: ClientEntity.abortRoute (ClientEntity.java:256 @2e62978; was
// clearRoute in 244/245.2, new in 244).
func (e *ClientEntity) AbortRoute() {
	e.PathLength = 0
	e.PreanimRouteLength = 0
}

func (e *ClientEntity) Teleport(arg1 bool, arg2 int, arg3 int) {
	// Java: 244 cancels on postanim_mode == 1 (ClientEntity.java:174), not the
	// 225 `priority <= 1` test.
	if e.PrimarySeqID != -1 && seqtype.List[e.PrimarySeqID].PostanimMode == 1 {
		e.PrimarySeqID = -1
	}
	if !arg1 {
		var5 := arg2 - e.PathTileX[0]
		var6 := arg3 - e.PathTileZ[0]
		if var5 >= -8 && var5 <= 8 && var6 >= -8 && var6 <= 8 {
			if e.PathLength < 9 {
				e.PathLength++
			}
			for i := e.PathLength; i > 0; i-- {
				e.PathTileX[i] = e.PathTileX[i-1]
				e.PathTileZ[i] = e.PathTileZ[i-1]
				e.PathRunning[i] = e.PathRunning[i-1]
			}
			e.PathTileX[0] = arg2
			e.PathTileZ[0] = arg3
			e.PathRunning[0] = false
			return
		}
	}
	// Java: move() resets THREE fields on the teleport path
	// (ClientEntity.java:195-197): routeLength, preanimRouteLength, seqDelayMove.
	e.PathLength = 0
	e.PreanimRouteLength = 0
	e.SeqTrigger = 0
	e.PathTileX[0] = arg2
	e.PathTileZ[0] = arg3
	e.X = e.PathTileX[0]*128 + e.Size*64
	e.Z = e.PathTileZ[0]*128 + e.Size*64
}

// MoveCode advances the entity one tile in compass direction arg1.
// Java: ClientEntity.moveCode (ClientEntity.java:208 @2e62978; was step
// in ≤245.2).
func (e *ClientEntity) MoveCode(arg0 bool, arg1 int) {
	var4 := e.PathTileX[0]
	var5 := e.PathTileZ[0]
	switch arg1 {
	case 0:
		var4--
		var5++
	case 1:
		var5++
	case 2:
		var4++
		var5++
	case 3:
		var4--
	case 4:
		var4++
	case 5:
		var4--
		var5--
	case 6:
		var5--
	case 7:
		var4++
		var5--
	}
	// Java: 244 cancels on postanim_mode == 1 (ClientEntity.java:236), not the
	// 225 `priority <= 1` test.
	if e.PrimarySeqID != -1 && seqtype.List[e.PrimarySeqID].PostanimMode == 1 {
		e.PrimarySeqID = -1
	}
	if e.PathLength < 9 {
		e.PathLength++
	}
	for i := e.PathLength; i > 0; i-- {
		e.PathTileX[i] = e.PathTileX[i-1]
		e.PathTileZ[i] = e.PathTileZ[i-1]
		e.PathRunning[i] = e.PathRunning[i-1]
	}
	e.PathTileX[0] = var4
	e.PathTileZ[0] = var5
	e.PathRunning[0] = arg0
}

// IsReady is the default implementation; ClientNpc and ClientPlayer
// override it. Callers reach it via the PathableEntity interface, which
// dispatches to the concrete type's method. Java: isReady
// (ClientEntity.java:262 @2e62978; was isVisible in ≤245.2).
func (e *ClientEntity) IsReady() bool {
	return false
}

// Pathing exposes the embedded *ClientEntity through the PathableEntity
// interface. Both ClientNpc and ClientPlayer embed ClientEntity by value,
// so Go's method promotion makes (*ClientNpc).Pathing() and
// (*ClientPlayer).Pathing() return a pointer to their embedded base — the
// Go equivalent of Java treating the reference as its ClientEntity parent.
func (e *ClientEntity) Pathing() *ClientEntity {
	return e
}

type PathableEntity interface {
	Teleport(bool, int, int)
	MoveCode(bool, int)
	IsReady() bool
	Pathing() *ClientEntity
}
