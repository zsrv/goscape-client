package client

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/datastruct/jstring"
	"github.com/zsrv/goscape-client/pkg/jagex2/io"
)

// AddIgnore is a faithful port of client.java's addIgnore: five branches —
// zero is a no-op, a full list adds a "list is full" chat message, a
// duplicate adds an "already on your ignore list" message, a friend-name
// collision adds a "remove from your friend list first" message, and the
// success path appends to IgnoreName37, increments IgnoreCount, requests a
// sidebar redraw, and writes the (ISAAC-masked opcode 79, int64 username)
// pair to the outbound packet.

func TestAddIgnore_ZeroIsNoop(t *testing.T) {
	c := NewClient()
	c.IgnoreCount = 5

	c.AddIgnore(0)

	if got, want := c.IgnoreCount, 5; got != want {
		t.Errorf("IgnoreCount = %d; want %d (zero arg is early-return)", got, want)
	}
	if c.MessageText[0] != "" {
		t.Errorf("MessageText[0] = %q; want empty (no message on zero arg)", c.MessageText[0])
	}
}

func TestAddIgnore_FullListAddsMessage(t *testing.T) {
	c := NewClient()
	c.IgnoreCount = 100

	c.AddIgnore(jstring.ToBase37("alice"))

	if got, want := c.IgnoreCount, 100; got != want {
		t.Errorf("IgnoreCount = %d; want %d (full list rejects)", got, want)
	}
	if got, want := c.MessageText[0], "Your ignore list is full. Max of 100 hit"; got != want {
		t.Errorf("MessageText[0] = %q; want %q", got, want)
	}
	if got, want := c.MessageType[0], 0; got != want {
		t.Errorf("MessageType[0] = %d; want %d (system message)", got, want)
	}
}

func TestAddIgnore_DuplicateAddsMessage(t *testing.T) {
	c := NewClient()
	alice := jstring.ToBase37("alice")
	c.IgnoreName37[0] = alice
	c.IgnoreCount = 1

	c.AddIgnore(alice)

	if got, want := c.IgnoreCount, 1; got != want {
		t.Errorf("IgnoreCount = %d; want %d (duplicate rejected)", got, want)
	}
	if got, want := c.MessageText[0], "Alice is already on your ignore list"; got != want {
		t.Errorf("MessageText[0] = %q; want %q", got, want)
	}
}

func TestAddIgnore_FriendConflictAddsMessage(t *testing.T) {
	c := NewClient()
	alice := jstring.ToBase37("alice")
	c.FriendName37[0] = alice
	c.FriendCount = 1

	c.AddIgnore(alice)

	if got, want := c.IgnoreCount, 0; got != want {
		t.Errorf("IgnoreCount = %d; want %d (friend conflict rejects)", got, want)
	}
	if got, want := c.MessageText[0], "Please remove Alice from your friend list first"; got != want {
		t.Errorf("MessageText[0] = %q; want %q", got, want)
	}
}

func TestAddIgnore_SuccessAppendsAndWritesPacket(t *testing.T) {
	c := NewClient()
	// AddIgnore's success path writes the opcode through ISAAC's keystream;
	// supply a deterministic Isaac so the test isn't dependent on memory
	// state that an unrelated test could mutate.
	c.Out.Random = io.NewIsaac([4]int{0, 0, 0, 0})
	alice := jstring.ToBase37("alice")
	posBefore := c.Out.Pos

	c.AddIgnore(alice)

	if got, want := c.IgnoreCount, 1; got != want {
		t.Errorf("IgnoreCount = %d; want %d (successful add)", got, want)
	}
	if got, want := c.IgnoreName37[0], alice; got != want {
		t.Errorf("IgnoreName37[0] = %d; want %d", got, want)
	}
	if !c.RedrawSidebar {
		t.Errorf("RedrawSidebar = false; want true (sidebar redraw requested)")
	}
	if got, want := c.Out.Pos-posBefore, 9; got != want {
		t.Errorf("Out.Pos advanced by %d; want %d (1 ISAAC-masked opcode + 8 int64 bytes)", got, want)
	}
	// Bytes 1..9 of the new write must be alice in big-endian (P8 layout).
	// We can't easily assert byte 0 without re-running ISAAC, so verify the
	// payload portion only.
	for i := range 8 {
		shift := uint((7 - i) * 8)
		want := byte((alice >> shift) & 0xFF)
		got := c.Out.Data[posBefore+1+i]
		if got != want {
			t.Errorf("Out.Data[%d] = 0x%02x; want 0x%02x (P8 byte %d of int64)", posBefore+1+i, got, want, i)
		}
	}
}
