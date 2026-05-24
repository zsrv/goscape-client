package signlink

import (
	"bytes"
	"testing"
)

func TestMemStoreRoundTrip(t *testing.T) {
	s := newMemStore()

	if got := s.load("missing"); got != nil {
		t.Fatalf("miss should be nil, got %v", got)
	}

	s.save("config", []byte{1, 2, 3})
	if got := s.load("config"); !bytes.Equal(got, []byte{1, 2, 3}) {
		t.Fatalf("load: got %v, want [1 2 3]", got)
	}
}

func TestMemStoreCopySemantics(t *testing.T) {
	s := newMemStore()

	in := []byte{4, 5}
	s.save("x", in)
	in[0] = 9 // mutate caller's slice after save
	if got := s.load("x"); got[0] != 4 {
		t.Fatalf("store aliased the input slice: got %v", got)
	}

	out := s.load("x")
	out[0] = 7 // mutate returned slice
	if again := s.load("x"); again[0] != 4 {
		t.Fatalf("store aliased the returned slice: got %v", again)
	}
}

func TestMemStoreOverwrite(t *testing.T) {
	s := newMemStore()
	s.save("k", []byte{1})
	s.save("k", []byte{2, 3})
	if got := s.load("k"); !bytes.Equal(got, []byte{2, 3}) {
		t.Fatalf("overwrite: got %v, want [2 3]", got)
	}
}

func TestMemStoreUID(t *testing.T) {
	s := newMemStore()
	if got := s.uid(); got != browserUID {
		t.Fatalf("uid: got %d, want %d", got, browserUID)
	}
}
