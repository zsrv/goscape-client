package client

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/config/iftype"
	"github.com/zsrv/goscape-client/pkg/jagex2/config/varbittype"
	"github.com/zsrv/goscape-client/pkg/jagex2/dash3d/entity/playerentity"
)

// TestBitmaskTable pins the 254 client BITMASK static (Client.java:577,
// 1279-1284 @2e62978), including the int32-wrap fidelity at index 31:
// Java's 32-bit `var0 += var0` overflows to 0 there, so BITMASK[31] = -1.
// A naive 64-bit Go port would yield 4294967295, which diverges when
// op 14 masks a negative varp with a full-width varbit.
func TestBitmaskTable(t *testing.T) {
	for i, want := range map[int]int{0: 1, 1: 3, 7: 255, 15: 65535, 30: 2147483647, 31: -1} {
		if Bitmask[i] != want {
			t.Errorf("Bitmask[%d] = %d, want %d", i, Bitmask[i], want)
		}
	}
}

// TestGetIfVar exercises 254's clientscript value VM (Java: getIfVar,
// Client.java:9783 @2e62978): the op 15/16/17 operator state machine, the
// new value sources (14 varbit, 18/19 world tile, 20 literal), the
// Stats-table-driven op 9, and the -2/-1 sentinels. Ops 4/10 are not
// covered — they pull ObjType instances out of the cache-backed registry,
// which needs real config data.
func TestGetIfVar(t *testing.T) {
	prevVarbits := varbittype.Instances
	t.Cleanup(func() { varbittype.Instances = prevVarbits })
	// varbit 0: bits 2..5 of varp 3.
	varbittype.Instances = []*varbittype.VarBitType{{BaseVar: 3, StartBit: 2, EndBit: 5}}

	c := &Client{}
	c.Varps = make([]int, 8)
	// Client BITMASK[i] = 2^(i+1)-1 (one wider than io.Bitmask!), so
	// endbit-startbit = 3 masks FOUR bits: (44 >> 2) & 15 = 11.
	c.Varps[3] = 0b101100
	c.SkillBaseLevel = make([]int, StatsCount)
	for i := range StatsCount {
		c.SkillBaseLevel[i] = i
	}
	c.LocalPlayer = playerentity.NewClientPlayer()
	c.LocalPlayer.X = 384 // >> 7 = tile 3
	c.LocalPlayer.Z = 640 // >> 7 = tile 5
	c.SceneBaseTileX = 3200
	c.SceneBaseTileZ = 3400

	// Sum of enabled slot indices: 0..17 plus 20 (StatsEnabled holes at
	// 18, 19, 21-24) = 153 + 20.
	const statTotal = 153 + 20

	tests := []struct {
		name   string
		script []int
		want   int
	}{
		{"literal", []int{20, 7, 0}, 7},
		{"default add chains", []int{20, 7, 20, 5, 0}, 12},
		{"op15 subtract", []int{20, 10, 15, 20, 3, 0}, 7},
		{"op16 divide", []int{20, 10, 16, 20, 2, 0}, 5},
		{"op16 divide by zero skipped", []int{20, 10, 16, 20, 0, 0}, 10},
		{"op17 multiply", []int{20, 10, 17, 20, 3, 0}, 30},
		{"operator chain mul then sub", []int{20, 2, 17, 20, 3, 15, 20, 1, 0}, 5},
		{"operator consumed once", []int{20, 10, 15, 20, 3, 20, 4, 0}, 11},
		{"op14 varbit", []int{14, 0, 0}, 11},
		{"op18 world tile x", []int{18, 0}, 3203},
		{"op19 world tile z", []int{19, 0}, 3405},
		{"op9 stats total", []int{9, 0}, statTotal},
		{"malformed script returns -1", []int{20}, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			com := &iftype.IfType{Scripts: [][]int{tt.script}}
			if got := c.GetIfVar(0, com); got != tt.want {
				t.Errorf("GetIfVar(%v) = %d, want %d", tt.script, got, tt.want)
			}
		})
	}

	t.Run("nil scripts returns -2", func(t *testing.T) {
		if got := c.GetIfVar(0, &iftype.IfType{}); got != -2 {
			t.Errorf("GetIfVar(nil scripts) = %d, want -2", got)
		}
	})
	t.Run("script index out of range returns -2", func(t *testing.T) {
		com := &iftype.IfType{Scripts: [][]int{{0}}}
		if got := c.GetIfVar(1, com); got != -2 {
			t.Errorf("GetIfVar(idx 1 of 1) = %d, want -2", got)
		}
	})
}
