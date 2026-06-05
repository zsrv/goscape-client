package client

// stats.go ports Java's jagex2.client.Stats @2e62978 (new in 254): the
// skill-slot table that sizes the per-player stat arrays and drives
// GetIfVar's opcode-9 total-level loop.

// StatsCount is the number of skill slots. Java: Stats.COUNT
// (Stats.java:9 @2e62978). 254 sizes statEffectiveLevel/statBaseLevel/statXP
// with this (245.2 used a flat 50).
const StatsCount = 25

// StatsNames maps slot index to skill name. Java: Stats.NAMES
// (Stats.java:12 @2e62978), ported verbatim incl. the five "-unused-"
// placeholder slots. Zero readers at 2e62978 (tree-wide) — kept as the
// canonical slot→skill mapping documenting StatsEnabled's holes.
var StatsNames = []string{
	"attack", "defence", "strength", "hitpoints", "ranged", "prayer",
	"magic", "cooking", "woodcutting", "fletching", "fishing", "firemaking",
	"crafting", "smithing", "mining", "herblore", "agility", "thieving",
	"slayer", "-unused-", "runecraft", "-unused-", "-unused-", "-unused-",
	"-unused-",
}

// StatsEnabled gates which slots count toward GetIfVar's opcode-9 skill
// total. Java: Stats.ENABLED (Stats.java:15 @2e62978) — false at 18
// (slayer), 19 and 21-24 (the "-unused-" slots); 20 (runecraft) counts.
var StatsEnabled = []bool{
	true, true, true, true, true, true,
	true, true, true, true, true, true,
	true, true, true, true, true, true,
	false, false, true, false, false, false,
	false,
}
