package loctype_test

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/config/loctype"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/model"
	jio "github.com/zsrv/goscape-client/pkg/jagex2/io"
)

// countingProvider records every RequestModel call made by model.Request on a
// cache miss. Satisfies io.OnDemandProvider.
type countingProvider struct{ calls []int }

func (p *countingProvider) RequestModel(id int) { p.calls = append(p.calls, id) }

var _ jio.OnDemandProvider = (*countingProvider)(nil)

// TestCheckModelAll verifies the return-value cases of CheckModelAll.
// (The non-short-circuit property is proved separately in TestCheckModelAll_NonShortCircuit.)
func TestCheckModelAll(t *testing.T) {
	// Ensure model package state is clean (no Metadata table).
	model.Reset()

	// Both models are -1 → all skipped → ready stays true.
	allSkipped := &loctype.LocType{
		Models: []int{-1, -1},
		Shapes: []int{10, 22},
	}
	if !allSkipped.CheckModelAll() {
		t.Error("CheckModelAll with all -1 models: want true, got false")
	}

	// One real model (id=5) + one -1. model.RequestDownload(5) returns false (Metadata==nil).
	oneMiss := &loctype.LocType{
		Models: []int{-1, 5},
		Shapes: []int{10, 22},
	}
	if oneMiss.CheckModelAll() {
		t.Error("CheckModelAll with one real model and Metadata==nil: want false, got true")
	}

	// nil Models → returns true immediately.
	nilModels := &loctype.LocType{}
	if !nilModels.CheckModelAll() {
		t.Error("CheckModelAll with nil Models: want true, got false")
	}
}

// TestCheckModelAll_NonShortCircuit proves that CheckModelAll uses a
// non-short-circuit AND: model.Request must be called for EVERY non-(-1) model
// even after the first miss.
//
// Mechanism: model.Init(count, provider) allocates Metadata[0..count-1] (all
// nil entries). model.RequestDownload(id) with Metadata[id]==nil calls
// provider.RequestModel(id) and returns false. A short-circuit implementation
// (ready = ready && model.RequestDownload(id)) would skip the second call after the
// first miss; the correct implementation calls both.
func TestCheckModelAll_NonShortCircuit(t *testing.T) {
	cp := &countingProvider{}
	model.Init(100, cp)
	t.Cleanup(model.Reset) // restore global state after this test

	lt := &loctype.LocType{Models: []int{5, 7}}
	got := lt.CheckModelAll()

	if got {
		t.Error("CheckModelAll with two missing models: want false, got true")
	}

	// Both ids must appear in provider.calls — proving both Request calls ran.
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

// TestCheckModel verifies shape-based model lookup and nil-guard behaviour.
func TestCheckModel(t *testing.T) {
	model.Reset()

	loc := &loctype.LocType{
		Models: []int{-1, 7},
		Shapes: []int{10, 22},
	}

	// shape 10 → index 0 → model -1 → always ready.
	if !loc.CheckModel(10) {
		t.Error("CheckModel shape=10 (model=-1): want true, got false")
	}

	// shape 22 → index 1 → model 7. model.RequestDownload(7) returns false (Metadata==nil).
	if loc.CheckModel(22) {
		t.Error("CheckModel shape=22 (model=7, Metadata==nil): want false, got true")
	}

	// shape not in Shapes → index==-1 → returns true.
	if !loc.CheckModel(99) {
		t.Error("CheckModel shape=99 (not in Shapes): want true, got false")
	}

	// nil Models → returns true.
	nilLoc := &loctype.LocType{
		Models: nil,
		Shapes: []int{10},
	}
	// shape 10 → index 0 → Models==nil → true.
	if !nilLoc.CheckModel(10) {
		t.Error("CheckModel with nil Models: want true, got false")
	}
}
