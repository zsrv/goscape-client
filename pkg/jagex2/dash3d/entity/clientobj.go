package entity

import (
	"github.com/zsrv/goscape-client/pkg/jagex2/config/objtype"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/model"
)

type ClientObj struct {
	Index int
	Count int
}

func NewClientObj() *ClientObj {
	return new(ClientObj)
}

// GetTempModel returns the ground obj-stack model, count-aware (stacked-amount
// variants). Java: ClientObj.getTempModel @2e62978 (was getModel in ≤245.2) —
// ObjType.get(index).getModel(count); the Go ObjType count-aware world model
// is GetInterfaceModel (its rev-225 name). Makes ClientObj satisfy the
// ModelSource interface (wired into the scene in WS3 3c/3d).
func (e *ClientObj) GetTempModel() *model.Model {
	return objtype.Get(e.Index).GetInterfaceModel(e.Count)
}
