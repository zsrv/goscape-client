package entity

import "github.com/zsrv/goscape-client/pkg/jagex2/dash3d/model"

type ModelSource interface {
	GetModel() *model.Model
}
