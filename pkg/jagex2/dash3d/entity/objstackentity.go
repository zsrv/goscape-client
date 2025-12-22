package entity

type ObjStackEntity struct {
	Index int
	Count int
}

func NewObjStackEntity() *ObjStackEntity {
	return new(ObjStackEntity)
}
