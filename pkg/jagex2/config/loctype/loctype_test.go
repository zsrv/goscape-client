package loctype_test

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/config/loctype"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/model"
	jio "github.com/zsrv/goscape-client/pkg/jagex2/io"
)

// countingProvider records every RequestModel call made by
// model.RequestDownload on a cache miss. Satisfies io.OnDemandProvider.
type countingProvider struct{ calls []int }

func (p *countingProvider) RequestModel(id int) { p.calls = append(p.calls, id) }

var _ jio.OnDemandProvider = (*countingProvider)(nil)

// TestCheckModelAll verifies the return-value cases of CheckModelAll at 254
// semantics: the 245.2 per-model -1 guard is gone, so a -1 model id masks to
// 65535 and is requested like any other id (a miss).
// (The non-short-circuit property is proved separately in TestCheckModelAll_NonShortCircuit.)
func TestCheckModelAll(t *testing.T) {
	// Ensure model package state is clean (no Metadata table).
	model.Reset()

	// 254: -1 models are no longer skipped — RequestDownload(65535) misses
	// (Metadata==nil), so ready goes false.
	withNegOnes := &loctype.LocType{
		Models: []int{-1, -1},
		Shapes: []int{10, 22},
	}
	if withNegOnes.CheckModelAll() {
		t.Error("CheckModelAll with -1 models at 254 (no guard, Metadata==nil): want false, got true")
	}

	// nil Models → returns true immediately.
	nilModels := &loctype.LocType{}
	if !nilModels.CheckModelAll() {
		t.Error("CheckModelAll with nil Models: want true, got false")
	}
}

// TestCheckModelAll_NonShortCircuit proves that CheckModelAll uses a
// non-short-circuit AND: model.RequestDownload must be called for EVERY model
// even after the first miss.
//
// Mechanism: model.Init(count, provider) allocates Metadata[0..count-1] (all
// nil entries). model.RequestDownload(id) with Metadata[id]==nil calls
// provider.RequestModel(id) and returns false. A short-circuit implementation
// (ready = ready && model.RequestDownload(id)) would skip the second call
// after the first miss; the correct implementation calls both.
func TestCheckModelAll_NonShortCircuit(t *testing.T) {
	cp := &countingProvider{}
	model.Init(100, cp)
	t.Cleanup(model.Reset) // restore global state after this test

	lt := &loctype.LocType{Models: []int{5, 7}}
	got := lt.CheckModelAll()

	if got {
		t.Error("CheckModelAll with two missing models: want false, got true")
	}

	// Both ids must appear in provider.calls — proving both RequestDownload calls ran.
	has5 := false
	has7 := false
	for _, id := range cp.calls {
		if id == 5 {
			has5 = true
		}
		if id == 7 {
			has7 = true
		}
	}
	if !has5 || !has7 {
		t.Errorf("non-short-circuit: want provider called for both 5 and 7, got calls=%v", cp.calls)
	}
}

// TestCheckModel verifies the 254 checkModel branches: the shapes path
// returns the RequestDownload result for the first shape match (no -1
// guard), and the new Shapes==nil (opcode-5) path gates on shape==10.
func TestCheckModel(t *testing.T) {
	model.Reset()

	loc := &loctype.LocType{
		Models: []int{6, 7},
		Shapes: []int{10, 22},
	}

	// shape 22 → index 1 → model 7. model.RequestDownload(7) returns false
	// (Metadata==nil).
	if loc.CheckModel(22) {
		t.Error("CheckModel shape=22 (model=7, Metadata==nil): want false, got true")
	}

	// shape not in Shapes → no match → returns true.
	if !loc.CheckModel(99) {
		t.Error("CheckModel shape=99 (not in Shapes): want true, got false")
	}

	// 254 models-only loc (Shapes==nil): shape != 10 → true without requests.
	modelsOnly := &loctype.LocType{
		Models: []int{5, 7},
		Shapes: nil,
	}
	if !modelsOnly.CheckModel(22) {
		t.Error("CheckModel models-only shape=22: want true (only shape 10 builds), got false")
	}

	// 254 models-only loc, shape == 10 → AND of RequestDownload over all
	// models → false with Metadata==nil.
	if modelsOnly.CheckModel(10) {
		t.Error("CheckModel models-only shape=10 (Metadata==nil): want false, got true")
	}

	// Shapes==nil and Models==nil → true.
	nilLoc := &loctype.LocType{}
	if !nilLoc.CheckModel(10) {
		t.Error("CheckModel with nil Shapes and nil Models: want true, got false")
	}
}

// TestCheckModel_ModelsOnlyNonShortCircuit proves the shape==10 models-only
// branch ANDs non-short-circuit: every model is requested even after a miss.
// Java: var3 &= Model.requestDownload(...) (LocType.java:362-365 @2e62978).
func TestCheckModel_ModelsOnlyNonShortCircuit(t *testing.T) {
	cp := &countingProvider{}
	model.Init(100, cp)
	t.Cleanup(model.Reset)

	lt := &loctype.LocType{Models: []int{5, 7}}
	if lt.CheckModel(10) {
		t.Error("CheckModel models-only shape=10 with two missing models: want false, got true")
	}

	has5 := false
	has7 := false
	for _, id := range cp.calls {
		if id == 5 {
			has5 = true
		}
		if id == 7 {
			has7 = true
		}
	}
	if !has5 || !has7 {
		t.Errorf("non-short-circuit: want provider called for both 5 and 7, got calls=%v", cp.calls)
	}
}
