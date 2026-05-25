package client

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/platform"
)

func TestHandleCharInputQueuesPrintable(t *testing.T) {
	c := NewClient()
	c.handleCharInput(platform.CharInput{Rune: 'x'})
	if c.KeyQueueWritePos != 1 || c.KeyQueue[0] != int('x') {
		t.Fatalf("printable not queued: pos=%d q0=%d", c.KeyQueueWritePos, c.KeyQueue[0])
	}
}

func TestHandleCharInputDropsControl(t *testing.T) {
	c := NewClient()
	c.handleCharInput(platform.CharInput{Rune: rune(7)}) // bell, < 30
	if c.KeyQueueWritePos != 0 {
		t.Fatalf("control char should be dropped, pos=%d", c.KeyQueueWritePos)
	}
}
