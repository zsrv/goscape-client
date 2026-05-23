package inputtracking

import (
	"sync"
	"time"

	"github.com/zsrv/goscape-client/pkg/jagex2/io"
)

var (
	Enabled   bool
	OutBuffer *io.Packet
	OldBuffer *io.Packet
	LastTime  int64
	// Java: InputTracking.java:22 declares trackedCount, incremented in
	// 9 sites (every record* function) and never read. Pure deob residue;
	// removed per the deob-artifact exclusion policy. The 9 increment
	// sites are likewise dropped.
	LastMoveTime int64
	LastX        int
	LastY        int
)

// mu guards every package-level var above. Java made each tracker method
// `synchronized` (one monitor per InputTracking class) because the AWT event
// dispatch thread wrote tracker state while the game thread drained it via
// Flush/Stop. The Go port runs Gio's app.Main() goroutine as the producer and
// Client.Run() as the consumer, so the same race exists. Stop() calls
// SetDisabled() while already holding the lock, so SetDisabled has a locked
// public wrapper plus an unlocked internal body to avoid Go's non-reentrant
// sync.Mutex deadlocking.
var mu sync.Mutex

func SetEnabled() {
	mu.Lock()
	defer mu.Unlock()
	OutBuffer = io.Alloc(1)
	OldBuffer = nil
	LastTime = time.Now().UnixMilli()
	Enabled = true
}

func SetDisabled() {
	mu.Lock()
	defer mu.Unlock()
	setDisabledLocked()
}

// setDisabledLocked is the body of SetDisabled without taking the lock; for
// internal callers (Stop) that already hold mu.
func setDisabledLocked() {
	Enabled = false
	OutBuffer = nil
	OldBuffer = nil
}

func Flush() *io.Packet {
	mu.Lock()
	defer mu.Unlock()
	var var1 *io.Packet
	if OldBuffer != nil && Enabled {
		var1 = OldBuffer
	}
	OldBuffer = nil
	return var1
}

func Stop() *io.Packet {
	mu.Lock()
	defer mu.Unlock()
	var var1 *io.Packet
	if OutBuffer != nil && OutBuffer.Pos > 0 && Enabled {
		var1 = OutBuffer
	}
	setDisabledLocked()
	return var1
}

// EnsureCapacity is intended for internal use by the other functions in this
// package, which call it while already holding mu. External callers must take
// the lock themselves; the function does not.
func EnsureCapacity(arg1 int) {
	if OutBuffer.Pos+arg1 >= 500 {
		var2 := OutBuffer
		OutBuffer = io.Alloc(1)
		OldBuffer = var2
	}
}

func MousePressed(arg0, arg1, arg2 int) {
	mu.Lock()
	defer mu.Unlock()
	if !Enabled || (arg0 < 0 || arg0 >= 789 || arg2 < 0 || arg2 >= 532) {
		return
	}
	var4 := time.Now().UnixMilli()
	var6 := (var4 - LastTime) / 10
	var6 = min(var6, 250)
	LastTime = var4
	EnsureCapacity(5)
	if arg1 == 1 {
		OutBuffer.P1(1)
	} else {
		OutBuffer.P1(2)
	}
	OutBuffer.P1(int(var6))
	OutBuffer.P3(arg0 + (arg2 << 10))
}

func MouseReleased(arg0 int) {
	mu.Lock()
	defer mu.Unlock()
	if !Enabled {
		return
	}
	var2 := time.Now().UnixMilli()
	var4 := (var2 - LastTime) / 10
	var4 = min(var4, 250)
	LastTime = var2
	EnsureCapacity(2)
	if arg0 == 1 {
		OutBuffer.P1(3)
	} else {
		OutBuffer.P1(4)
	}
	OutBuffer.P1(int(var4))
}

func MouseMoved(arg0, arg2 int) {
	mu.Lock()
	defer mu.Unlock()
	if !Enabled || (arg2 < 0 || arg2 >= 789 || arg0 < 0 || arg0 >= 532) {
		return
	}
	var3 := time.Now().UnixMilli()
	if var3-LastMoveTime < 50 {
		return
	}
	LastMoveTime = var3
	var5 := (var3 - LastTime) / 10
	var5 = min(var5, 250)
	LastTime = var3
	if arg2-LastX < 8 && arg2-LastX >= -8 && arg0-LastY < 8 && arg0-LastY >= -8 {
		EnsureCapacity(3)
		OutBuffer.P1(5)
		OutBuffer.P1(int(var5))
		// Java: outBuffer.p1(arg2 - lastX + 8 + (arg0 - lastY + 8 << 4))
		// Java additive is higher precedence than <<, so the inner expr is
		// ((arg0 - lastY + 8) << 4). Go shift is HIGHER than additive, so
		// parens are required on both sides of the shift to preserve Java
		// grouping.
		OutBuffer.P1(arg2 - LastX + 8 + ((arg0 - LastY + 8) << 4))
	} else if arg2-LastX < 128 && arg2-LastX >= -128 && arg0-LastY < 128 && arg0-LastY >= -128 {
		EnsureCapacity(4)
		OutBuffer.P1(6)
		OutBuffer.P1(int(var5))
		OutBuffer.P1(arg2 - LastX + 128)
		OutBuffer.P1(arg0 - LastY + 128)
	} else {
		EnsureCapacity(5)
		OutBuffer.P1(7)
		OutBuffer.P1(int(var5))
		OutBuffer.P3(arg2 + (arg0 << 10))
	}
	LastX = arg2
	LastY = arg0
}

func KeyPressed(arg0 int) {
	mu.Lock()
	defer mu.Unlock()
	if !Enabled {
		return
	}
	var2 := time.Now().UnixMilli()
	var4 := (var2 - LastTime) / 10
	var4 = min(var4, 250)
	LastTime = var2
	if arg0 == 1000 {
		arg0 = 11
	}
	if arg0 == 1001 {
		arg0 = 12
	}
	if arg0 == 1002 {
		arg0 = 14
	}
	if arg0 == 1003 {
		arg0 = 15
	}
	if arg0 >= 1008 {
		arg0 -= 992
	}
	EnsureCapacity(3)
	OutBuffer.P1(8)
	OutBuffer.P1(int(var4))
	OutBuffer.P1(arg0)
}

func KeyReleased(arg0 int) {
	mu.Lock()
	defer mu.Unlock()
	if !Enabled {
		return
	}
	var2 := time.Now().UnixMilli()
	var4 := (var2 - LastTime) / 10
	var4 = min(var4, 250)
	LastTime = var2
	if arg0 == 1000 {
		arg0 = 11
	}
	if arg0 == 1001 {
		arg0 = 12
	}
	if arg0 == 1002 {
		arg0 = 14
	}
	if arg0 == 1003 {
		arg0 = 15
	}
	if arg0 >= 1008 {
		arg0 -= 992
	}
	EnsureCapacity(3)
	OutBuffer.P1(9)
	OutBuffer.P1(int(var4))
	OutBuffer.P1(arg0)
}

func FocusGained() {
	mu.Lock()
	defer mu.Unlock()
	if !Enabled {
		return
	}
	var1 := time.Now().UnixMilli()
	var3 := (var1 - LastTime) / 10
	var3 = min(var3, 250)
	LastTime = var1
	EnsureCapacity(2)
	OutBuffer.P1(10)
	OutBuffer.P1(int(var3))
}

func FocusLost() {
	mu.Lock()
	defer mu.Unlock()
	if !Enabled {
		return
	}
	var1 := time.Now().UnixMilli()
	var3 := (var1 - LastTime) / 10
	var3 = min(var3, 250)
	LastTime = var1
	EnsureCapacity(2)
	OutBuffer.P1(11)
	OutBuffer.P1(int(var3))
}

func MouseEntered() {
	mu.Lock()
	defer mu.Unlock()
	if !Enabled {
		return
	}
	var1 := time.Now().UnixMilli()
	var3 := (var1 - LastTime) / 10
	var3 = min(var3, 250)
	LastTime = var1
	EnsureCapacity(2)
	OutBuffer.P1(12)
	OutBuffer.P1(int(var3))
}

func MouseExited() {
	mu.Lock()
	defer mu.Unlock()
	if !Enabled {
		return
	}
	var1 := time.Now().UnixMilli()
	var3 := (var1 - LastTime) / 10
	var3 = min(var3, 250)
	LastTime = var1
	EnsureCapacity(2)
	OutBuffer.P1(13)
	OutBuffer.P1(int(var3))
}
