package client

// Java: Protocol.CLIENTPROT_LOOKUP (Protocol.java @176a85f; named
// CLIENTPROT_SCRAMBLED in the rev#225 deob) — a 257-entry opcode table — is
// intentionally NOT ported. It has zero references anywhere in the Java
// client (only its declaration exists; the opcode-encryption path that would
// consume it is unused), re-verified at 245.2, so it is a dead deobfuscation
// artifact. If a future revision wires up client-opcode scrambling, restore
// it here.

// Java: Protocol.SERVERPROT_LENGTH (jagex2/client/Protocol.java @176a85f, 245.2).
// vs 244: same multiset reindexed + one new length-4 entry (IF_SETSCROLLPOS,
// index 226).
var SERVERPROT_SIZES = []int{0, 0, 3, 0, 0, 0, 0, 2, 1, 0, 0, 0, 0, 3, 0, -2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 3, 0, 0, -2, 0, 0, 1, 0, 0, 0, 4, 0, 0, 0, 0, 6, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 4, 0, 0, 4, 2, 4, 0, 0, 0, 6, 4, 0, 0, 0, 0, 0, 0, 2, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0, 5, -2, 2, 0, 0, 0, 0, 0, 0, 4, 0, -2, 0, 0, 0, 9, 6, 0, 0, 0, 0, 2, 0, 0, 0, 4, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 4, 0, 0, 0, 0, 2, 6, 0, 2, 0, 0, 0, 0, 0, 0, 0, 7, 2, 6, 0, 0, -2, 0, 0, 0, 0, -2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, -1, 0, 2, 0, 0, 0, -2, 0, 0, 0, 0, 0, 15, 14, 0, 7, 0, 3, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 2, 0, 0, 0, -1, 1, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3, 4, 0, 0, 4, 6, 0, 0, 0, 0, 0, 2, 0, 10, 0, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
