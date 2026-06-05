package io

// Java: Protocol.CLIENTPROT_LOOKUP (jagex2/client/Protocol.java:9 @2e62978;
// named CLIENTPROT_SCRAMBLED in the rev#225 deob) — a 257-entry opcode table
// — is intentionally NOT ported. It has zero references anywhere in the Java
// client (only its declaration exists; the opcode-encryption path that would
// consume it is unused), re-verified at 254 via tree-wide grep @2e62978, so
// it is a dead deobfuscation artifact. In 274 the table reverts to the name
// CLIENTPROT_SCRAMBLED at jagex2/io/Protocol.java:9 @32f3062 and is still
// declaration-only (verified tree-wide). The intentional non-port stands.

// Java: Protocol.SERVERPROT_LENGTH (jagex2/client/Protocol.java:12 @2e62978,
// 254). vs 245.2: 107 of 257 entries changed; NOT a permutation of the old
// multiset (one extra -1, one extra 1). Verbatim copy, diffed against the
// pinned Java source. Moved back to jagex2/io/Protocol.java in 274 @32f3062
// (SERVERPROT_SIZE); values still at 254 parity — full table re-derivation
// is the P4 opcode workstream.
var SERVERPROT_SIZES = []int{6, 0, 0, 4, 0, 0, 0, 0, 7, 0, 0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3, 5, 0, 6, -2, 0, 4, 0, 0, 0, 0, 0, 0, 15, 4, 0, 0, -2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 6, 0, 0, 1, 0, -1, -2, 0, -2, 6, 0, 0, 0, 0, 0, 4, 0, 0, -1, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, -2, 2, 0, 0, 3, 0, 0, 1, 4, 0, 0, 7, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 9, 0, 0, 6, 3, 0, 0, 0, 0, 5, 0, 0, -2, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0, 0, 6, 0, 1, 0, 0, 2, 0, 2, 0, 0, 10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 2, 0, 2, 2, 0, 0, 0, 2, 0, -2, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3, 2, 0, 0, 0, 0, 0, 0, 0, 0, 6, 2, 0, 0, 0, 0, 0, 0, -1, 0, 0, 0, 0, 4, 0, 4, 0, 3, 0, 0, 0, 0, 14, 0, 0, 0, 6, 0, 0, 4, 0, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 4, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 1, 0}
