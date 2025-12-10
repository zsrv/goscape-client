package entity

import "goscape-client/pkg/jagex2/datastruct"

type ObjStackEntity struct {
	datastruct.Linkable[ObjStackEntity]

	Index int
	Count int
}

func NewObjStackEntity() *ObjStackEntity {
	return &ObjStackEntity{}
}
