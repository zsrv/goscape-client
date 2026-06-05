package io

// Java: Protocol.CLIENTPROT_SCRAMBLED (jagex2/io/Protocol.java:9 @32f3062;
// named CLIENTPROT_LOOKUP in the 254 deob @2e62978) — a 257-entry opcode
// table — is intentionally NOT ported. It has zero references anywhere in
// the Java client (only its declaration exists; the opcode-encryption path
// that would consume it is unused), re-verified at 274 via tree-wide grep
// @32f3062, so it is a dead deobfuscation artifact. The intentional
// non-port stands.

// Java: Protocol.SERVERPROT_SIZE (jagex2/io/Protocol.java:12 @32f3062, 274;
// named SERVERPROT_LENGTH at jagex2/client/Protocol.java:12 @2e62978, 254).
// vs 254: 111 of 257 entries changed — a pure permutation of the old
// multiset plus exactly one extra size-1 entry, the NEW SET_MINIMAP_STATE
// (op 194). Verbatim copy, machine-extracted from the pinned Java source
// (never patch incrementally — values are fully renumbered each revision).
var SERVERPROT_SIZES = []int{0, 0, 0, -2, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 4, 2, -1, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 6, 0, 0, 0, 2, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, -2, 0, 0, 0, 4, 0, 0, 0, 3, 0, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 6, 0, 0, 0, 5, 0, 1, 0, 6, 0, 0, 0, 2, 1, 10, 0, 0, 0, 7, 0, 0, 0, 0, 0, 0, 0, 0, 0, 6, -2, 15, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 6, 0, 0, 0, 4, 2, 0, 0, 3, 4, 0, 0, 0, 4, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 6, 0, 4, 0, 0, -1, 0, 0, 0, 0, 2, -2, 0, 0, 0, 0, -2, 2, 0, 0, 14, 0, 0, 0, 0, 0, 0, 4, 0, 1, 0, 0, 0, 0, 0, 0, 2, 0, 1, -2, 0, -2, 0, 0, 6, 0, 0, 3, 0, 0, 0, 1, 0, 0, 0, 2, 0, 0, 0, 3, 0, 0, 0, 7, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 4, 0, 6, 0, -1, 0, 0, 0, 0, 2, 1, 0, 0, 0, 6, 0, 9, 0, 0, 0, 0, 0, 0, 0, 0, 0}
