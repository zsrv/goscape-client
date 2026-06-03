package io

// Java: Protocol.CLIENTPROT_SCRAMBLED (Protocol.java:8-9) — a 256-entry opcode
// scramble table — is intentionally NOT ported. It has zero references anywhere
// in the rev#225 Java client (only its declaration exists; the opcode-encryption
// path that would consume it is unused), so it is a dead deobfuscation artifact.
// If a future revision wires up client-opcode scrambling, restore it here.

// Java: Protocol.SERVERPROT_LENGTH (jagex2/io/Protocol.java)
var SERVERPROT_SIZES = []int{0, 0, 0, 0, 0, 0, 0, -2, 0, 3, 2, 0, 6, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 6, 0, 0, 0, 0, 14, -1, 0, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 10, 0, 0, 0, 0, 6, 4, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 7, 9, 0, -2, 0, 0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 2, -2, 0, 0, 0, 0, 0, 0, 0, 2, -1, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3, 0, 2, 0, 0, 0, 4, 0, 2, -2, 0, 0, 0, 0, 15, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5, 0, 0, -2, 4, 0, 0, 2, 0, 2, 0, 2, 0, 6, 4, 0, 0, 1, 0, 0, 0, 0, 4, 2, 0, 2, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 6, 0, 3, 0, 0, 0, 0, 0, 0, 4, 0, 7, 3, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0, 6, 0, 0, 0, 6, 0, 0, 0, 0, 0, 4, -2, 5, 0, 3, 0, 0, 0, 2, 6, 0, 0, -2, 4, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0, 0}
