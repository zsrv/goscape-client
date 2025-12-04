package entity

type ObjStackEntity struct {
	Index int
	Count int
}

func NewObjStackEntity() *ObjStackEntity {
	return &ObjStackEntity{}
}
