package entity

type LocAddEntity struct {
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

func NewLocAddEntity() *LocAddEntity {
	return &LocAddEntity{}
}
