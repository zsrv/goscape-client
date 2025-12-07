package entity

import "goscape-client/pkg/jagex2/config/seqtype"

type PathingEntity struct {
	X                        int
	Z                        int
	Yaw                      int
	SeqStretches             bool
	Size                     int
	SeqStandID               int
	SeqTurnID                int
	SeqWalkID                int
	SeqTurnAroundID          int
	SeqTurnLeftID            int
	SeqTurnRightId           int
	SeqRunID                 int
	Chat                     string
	ChatTimer                int
	ChatColor                int
	ChatStyle                int
	Damage                   int
	DamageType               int
	CombatCycle              int
	Health                   int
	DstYaw                   int
	PathLength               int
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

func NewPathingEntity() *PathingEntity {
	return &PathingEntity{
		Size:            1,
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

func (e *PathingEntity) Teleport(arg1 bool, arg2 int, arg3 int) {
	if e.PrimarySeqID != -1 && seqtype.Instances[e.PrimarySeqID].Priority <= 1 {
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
	e.PathLength = 0
	e.SeqTrigger = 0
	e.PathTileX[0] = arg2
	e.PathTileZ[0] = arg3
	e.X = e.PathTileX[0]*128 + e.Size*64
	e.Z = e.PathTileZ[0]*128 + e.Size*64
}

func (e *PathingEntity) MoveAlongRoute(arg0 bool, arg1 int) {
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
	if e.PrimarySeqID != -1 && seqtype.Instances[e.PrimarySeqID].Priority <= 1 {
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

// Deprecated: TODO: this is overridden by classes extending PathingEntity.. use an interface for this instead, add to teh structs?
func (e *PathingEntity) IsVisible() bool {
	return false
}

type PathableEntity interface {
	Teleport(bool, int, int)
	MoveAlongRoute(bool, int)
	IsVisible() bool
}
