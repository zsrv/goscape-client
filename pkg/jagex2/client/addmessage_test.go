package client

import (
	"testing"
)

// AddMessage is a faithful port of client.java:7909-7925: prepend a chat
// message to a 100-slot ring, shifting older messages down. A zero-typed
// message also captures the modal text when a sticky chat interface is
// active, and any new message triggers a chatback redraw when no chat
// interface is open.

func TestAddMessage_PopulatesIndexZero(t *testing.T) {
	c := NewClient()

	c.AddMessage(2, "hello", "alice")

	if got, want := c.MessageType[0], 2; got != want {
		t.Errorf("MessageType[0] = %d; want %d", got, want)
	}
	if got, want := c.MessageText[0], "hello"; got != want {
		t.Errorf("MessageText[0] = %q; want %q", got, want)
	}
	if got, want := c.MessageSender[0], "alice"; got != want {
		t.Errorf("MessageSender[0] = %q; want %q", got, want)
	}
}

func TestAddMessage_ShiftsExistingDown(t *testing.T) {
	c := NewClient()
	c.AddMessage(1, "older", "alice")
	c.AddMessage(2, "newer", "bob")

	if got, want := c.MessageText[0], "newer"; got != want {
		t.Errorf("MessageText[0] = %q; want %q (newest at head)", got, want)
	}
	if got, want := c.MessageText[1], "older"; got != want {
		t.Errorf("MessageText[1] = %q; want %q (previous shifted down)", got, want)
	}
	if got, want := c.MessageType[0], 2; got != want {
		t.Errorf("MessageType[0] = %d; want %d", got, want)
	}
	if got, want := c.MessageType[1], 1; got != want {
		t.Errorf("MessageType[1] = %d; want %d", got, want)
	}
	if got, want := c.MessageSender[1], "alice"; got != want {
		t.Errorf("MessageSender[1] = %q; want %q", got, want)
	}
}

func TestAddMessage_DropsOldestAtSlot99(t *testing.T) {
	c := NewClient()
	for i := range 100 {
		c.MessageType[i] = 1000 + i
		c.MessageText[i] = "fill"
		c.MessageSender[i] = "x"
	}
	// Sentinel at the bottom of the ring — the one that should be dropped.
	c.MessageType[99] = 9999

	c.AddMessage(7, "incoming", "carol")

	// New message lands at the head.
	if got, want := c.MessageType[0], 7; got != want {
		t.Errorf("MessageType[0] = %d; want %d", got, want)
	}
	// Java's loop runs for (i = 99; i > 0; i--), so slot 99 receives the old
	// slot 98 — the original slot 99 (9999) is overwritten and dropped.
	if got, want := c.MessageType[99], 1000+98; got != want {
		t.Errorf("MessageType[99] = %d; want %d (slot-98 shifted in, old 99 dropped)", got, want)
	}
}

func TestAddMessage_ZeroTypeWithStickyInterfaceCapturesModal(t *testing.T) {
	c := NewClient()
	c.StickyChatInterfaceID = 42
	c.MouseClickButton = 1

	c.AddMessage(0, "you are dead", "")

	if got, want := c.ModalMessage, "you are dead"; got != want {
		t.Errorf("ModalMessage = %q; want %q", got, want)
	}
	if got, want := c.MouseClickButton, 0; got != want {
		t.Errorf("MouseClickButton = %d; want %d (cleared on modal capture)", got, want)
	}
}

func TestAddMessage_ZeroTypeWithoutStickyInterfaceSkipsModal(t *testing.T) {
	c := NewClient()
	// StickyChatInterfaceID stays at NewClient's default of -1.
	c.MouseClickButton = 1

	c.AddMessage(0, "you are dead", "")

	if c.ModalMessage != "" {
		t.Errorf("ModalMessage = %q; want empty (no sticky interface)", c.ModalMessage)
	}
	if got, want := c.MouseClickButton, 1; got != want {
		t.Errorf("MouseClickButton = %d; want %d (preserved when modal not captured)", got, want)
	}
}

func TestAddMessage_NonZeroTypeNeverCapturesModal(t *testing.T) {
	c := NewClient()
	c.StickyChatInterfaceID = 42
	c.MouseClickButton = 1

	c.AddMessage(2, "regular chat", "alice")

	if c.ModalMessage != "" {
		t.Errorf("ModalMessage = %q; want empty (type != 0 never captures)", c.ModalMessage)
	}
	if got, want := c.MouseClickButton, 1; got != want {
		t.Errorf("MouseClickButton = %d; want %d (only zero-type clears it)", got, want)
	}
}

func TestAddMessage_NoChatInterfaceRequestsRedrawChatback(t *testing.T) {
	c := NewClient()
	// ChatInterfaceID stays at NewClient's default of -1.
	c.RedrawChatback = false

	c.AddMessage(2, "hi", "alice")

	if !c.RedrawChatback {
		t.Errorf("RedrawChatback = false; want true (no chat interface open)")
	}
}

func TestAddMessage_OpenChatInterfaceLeavesRedrawAlone(t *testing.T) {
	c := NewClient()
	c.ChatInterfaceID = 100
	c.RedrawChatback = false

	c.AddMessage(2, "hi", "alice")

	if c.RedrawChatback {
		t.Errorf("RedrawChatback = true; want false (chat interface open suppresses redraw)")
	}
}
