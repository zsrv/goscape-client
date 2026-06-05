package client

// skill.go ports Java's jagex2.client.Stats @2e62978 (254 name);
// Skill.java @32f3062 — values at 254 parity, P4 pending: the skill-slot
// table that sizes the per-player stat arrays and drives GetIfVar's
// opcode-9 total-level loop.

// SkillCount is the number of skill slots. Java: Stats.COUNT
// (Stats.java:9 @2e62978). 254 sizes statEffectiveLevel/statBaseLevel/statXP
// with this (245.2 used a flat 50).
const SkillCount = 25

// SkillNames maps slot index to skill name. Java: Stats.NAMES
// (Stats.java:12 @2e62978), ported verbatim incl. the five "-unused-"
// placeholder slots. Zero readers at 2e62978 (tree-wide) — kept as the
// canonical slot→skill mapping documenting SkillUsed's holes.
var SkillNames = []string{
	"attack", "defence", "strength", "hitpoints", "ranged", "prayer",
	"magic", "cooking", "woodcutting", "fletching", "fishing", "firemaking",
	"crafting", "smithing", "mining", "herblore", "agility", "thieving",
	"slayer", "-unused-", "runecraft", "-unused-", "-unused-", "-unused-",
	"-unused-",
}

// SkillUsed gates which slots count toward GetIfVar's opcode-9 skill
// total. Java: Stats.ENABLED (Stats.java:15 @2e62978) — false at 18
// (slayer), 19 and 21-24 (the "-unused-" slots); 20 (runecraft) counts.
var SkillUsed = []bool{
	true, true, true, true, true, true,
	true, true, true, true, true, true,
	true, true, true, true, true, true,
	false, false, true, false, false, false,
	false,
}
