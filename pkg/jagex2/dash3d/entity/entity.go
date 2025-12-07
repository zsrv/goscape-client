package entity

import "goscape-client/pkg/jagex2/graphics/model"

type Entity interface {
	Draw() *model.Model
}
