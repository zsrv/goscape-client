package client

import (
	"sync"
	"sync/atomic"
	"time"
)

// MouseTracking ports Java's jagex2.client.MouseTracking @2e62978: a 50ms
// mouse-position sampler thread. 254 is the first revision in this port's
// lineage where its ring buffer is actually consumed — gameLoop's
// EVENT_MOUSE_MOVE telemetry (WS5) drains X/Y/Length under Lock, and login
// reply 2 resets Length (WS5).
//
// Concurrency audit (actual goroutine layout, not Java style):
//   - Lock ports Java's `final Object lock` + the synchronized blocks in
//     run()/gameLoop. Writer: the sampler goroutine (Run). Reader/resetter:
//     the loop goroutine (gameLoop serializer + login reply 2, both WS5).
//   - Active ports Java's plain `boolean active`: written by Unload on the
//     loop goroutine, read by the sampler — a cross-goroutine stop flag, so
//     atomic.Bool instead of Java's bare (JMM-tolerated) field.
//   - The App.MouseX/MouseY sample reads are unsynchronized in Java (AWT
//     event thread writes vs tracker thread reads) and stay unsynchronized
//     here (loop goroutine writes vs sampler reads): word-sized values used
//     only as telemetry samples, mirroring Java's tolerated race.
type MouseTracking struct {
	App    *Client     // Java: fc.b
	Active atomic.Bool // Java: fc.c — plain boolean; see concurrency audit
	Lock   sync.Mutex  // Java: fc.d — final Object lock
	Length int         // Java: fc.e
	X      []int       // Java: fc.f — new int[500]
	Y      []int       // Java: fc.g — new int[500]
}

func NewMouseTracking(arg1 *Client) *MouseTracking {
	t := &MouseTracking{
		App: arg1,
		X:   make([]int, 500),
		Y:   make([]int, 500),
	}
	t.Active.Store(true)
	return t
}

// Run is the sampler loop. Java: MouseTracking.run @2e62978; started by
// Client.load via startThread(mouseTracking, 10) — Go starts the goroutine
// directly (thread priority not ported).
func (t *MouseTracking) Run() {
	for t.Active.Load() {
		t.Lock.Lock()
		if t.Length < 500 {
			t.X[t.Length] = t.App.MouseX
			t.Y[t.Length] = t.App.MouseY
			t.Length++
		}
		t.Lock.Unlock()
		time.Sleep(50 * time.Millisecond) // Java: Thread.sleep(50L)
	}
}
