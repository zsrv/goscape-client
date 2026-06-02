package entity

import "github.com/zsrv/goscape-client/pkg/jagex2/dash3d/model"

type Entity interface {
	Draw() *model.Model
}
