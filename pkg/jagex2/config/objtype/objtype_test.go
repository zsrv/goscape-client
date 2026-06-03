package objtype_test

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/config/objtype"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/model"
)

// TestGetInterfaceModelNilWhenMetadataAbsent verifies that GetInterfaceModel
// returns nil (not an empty husk via NewModel1) when model.Metadata is nil,
// i.e. the lazy TryGet path is taken correctly.
func TestGetInterfaceModelNilWhenMetadataAbsent(t *testing.T) {
	// Ensure Metadata is nil so TryGet returns nil.
	model.Reset()

	obj := &objtype.ObjType{
		Index:    42,
		Model:    7,
		ResizeX:  128,
		ResizeY:  128,
		ResizeZ:  128,
		Ambient:  0,
		Contrast: 0,
	}

	got := obj.GetInterfaceModel(1)
	if got != nil {
		t.Errorf("GetInterfaceModel with nil Metadata: want nil, got non-nil model")
	}
}
