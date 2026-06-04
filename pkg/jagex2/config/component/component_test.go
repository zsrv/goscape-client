package component

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/model"
	"github.com/zsrv/goscape-client/pkg/jagex2/datastruct"
	"github.com/zsrv/goscape-client/pkg/jagex2/io"
)

// buildDataArchive returns a Jagfile whose single "data" entry yields payload
// verbatim. Real archives are bzip2-compressed (and the repo has no bzip2
// encoder), so the struct is populated directly with Unpacked=true — that path
// in Jagfile.Read does a plain copy out of Buffer (no decompression).
func buildDataArchive(payload []byte) *io.Jagfile {
	// Java: Jagfile.read hashes the upper-cased entry name; "DATA" must match.
	hash := int32(0)
	for _, ch := range "DATA" {
		hash = hash*61 + ch - 32
	}
	return &io.Jagfile{
		Buffer:           payload,
		FileCount:        1,
		FileHash:         []int{int(hash)},
		FileUnpackedSize: []int{len(payload)},
		FilePackedSize:   []int{len(payload)},
		FileOffset:       []int{0},
		Unpacked:         true,
	}
}

// buildType6Component encodes a single Component header for a type==6,
// buttonType==1 component (buttonType 1 terminates Unpack's inner read loop),
// followed by the trailing option string. modelHi/modelLo are the two g1
// values that produce the deferred model id ((modelHi-1)<<8)+modelLo.
func buildType6Payload(id, modelHi, modelLo int) []byte {
	p := io.NewPacket(make([]byte, 64))
	// Unpack reads a leading g2 as the Instances array size before the loop.
	p.P2(id + 1)
	p.P2(id) // component id
	p.P1(6)  // type
	p.P1(1)  // buttonType (1 → terminates inner loop)
	p.P2(0)  // clientCode
	p.P2(1)  // width
	p.P2(1)  // height
	p.P1(0)  // trans (named alpha at 244)
	p.P1(0)  // overLayer (0 → -1, no extra read)
	p.P1(0)  // scriptComparator count
	p.P1(0)  // scripts count
	// type == 6 block
	p.P1(modelHi) // model selector
	if modelHi != 0 {
		p.P1(modelLo)
	}
	p.P1(0) // activeModel selector (0 → leaves ActiveModel default)
	p.P1(0) // anim selector (0 → Anim = -1)
	p.P1(0) // activeAnim selector (0 → ActiveAnim = -1)
	p.P2(0) // zoom
	p.P2(0) // xan
	p.P2(0) // yan
	// buttonType == 1 → option string read after inner loop terminates.
	p.PJStr("")
	return p.Data[:p.Pos]
}

func TestUnpackType6DeferredModel(t *testing.T) {
	const id = 3
	const modelHi = 2
	const modelLo = 5
	payload := buildType6Payload(id, modelHi, modelLo)
	archive := buildDataArchive(payload)

	Unpack(nil, nil, archive)

	if id >= len(Instances) || Instances[id] == nil {
		t.Fatalf("component %d not decoded; Instances len=%d", id, len(Instances))
	}
	com := Instances[id]
	if com.Type != 6 {
		t.Fatalf("Type = %d, want 6", com.Type)
	}
	if com.ModelType != 1 {
		t.Errorf("ModelType = %d, want 1", com.ModelType)
	}
	wantModel := ((modelHi - 1) << 8) + modelLo
	if com.Model != wantModel {
		t.Errorf("Model = %d, want %d", com.Model, wantModel)
	}
	// activeModel selector was 0 → defaults untouched.
	if com.ActiveModelType != 0 || com.ActiveModel != 0 {
		t.Errorf("ActiveModel(Type) = (%d,%d), want (0,0)", com.ActiveModelType, com.ActiveModel)
	}
}

func TestLoadModelType5Uncached(t *testing.T) {
	ModelCache = datastruct.NewLruCache[*model.Model](30)
	defer func() { ModelCache = nil }()

	com := &Component{}
	// type 5 returns nil and does not deref localPlayer.
	if got := com.LoadModel(5, 0, nil); got != nil {
		t.Errorf("LoadModel(5, 0, nil) = %v, want nil", got)
	}
}

func TestLoadModelCacheKeyByTypeAndId(t *testing.T) {
	ModelCache = datastruct.NewLruCache[*model.Model](30)
	defer func() { ModelCache = nil }()

	m := &model.Model{}
	const typ = 5
	const modelID = 0

	CacheModel(m, modelID, typ)

	com := &Component{}
	// cache-check-first short-circuits before the type-5 nil case.
	if got := com.LoadModel(typ, modelID, nil); got != m {
		t.Errorf("LoadModel(%d, %d, nil) = %v, want cached %v", typ, modelID, got, m)
	}

	// A different key must miss (type 5 → nil) rather than return the cached model.
	if got := com.LoadModel(typ, modelID+1, nil); got != nil {
		t.Errorf("LoadModel(%d, %d, nil) = %v, want nil (cache miss)", typ, modelID+1, got)
	}
}
