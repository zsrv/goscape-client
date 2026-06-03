package entity

// LocChange is Java: jagex2.dash3d.LocChange (LocChange.java). It is the rev-244
// merge of the rev-225 LocChange (the old loc captured via Last*) and
// LocMergeEntity (the timed merge via LastCycle): one type carries both a
// pending new loc (New*) and the loc it replaces (Old*), gated by StartTime /
// EndTime. endTime == -1 means a permanent change; endTime == 0 means "revert
// now"; a positive endTime counts down each cycle.
type LocChange struct {
	Level     int // Java: level
	Layer     int // Java: layer
	X         int // Java: x
	Z         int // Java: z
	OldType   int // Java: oldType
	OldAngle  int // Java: oldAngle
	OldShape  int // Java: oldShape
	NewType   int // Java: newType
	NewAngle  int // Java: newAngle
	NewShape  int // Java: newShape
	StartTime int // Java: startTime
	EndTime   int // Java: endTime (defaults to -1; Go zero-values to 0)
}

// NewLocChange mirrors Java's field initializer `endTime = -1` (Go zero-values
// all fields to 0, so EndTime must be set explicitly).
func NewLocChange() *LocChange {
	return &LocChange{EndTime: -1}
}
