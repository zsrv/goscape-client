package entity

type LocChange struct {
	Plane        int
	Layer        int
	X            int
	Z            int
	LocIndex     int
	Angle        int
	Shape        int
	LastLocIndex int
	LastAngle    int
	LastShape    int
}

func NewLocChange() *LocChange {
	return new(LocChange)
}
