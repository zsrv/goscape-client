// Package class61 mirrors the Java deobfuscator's `deob/class61.java`
// stub. The Java class has a single static field `instances` and no
// other members or callers besides one nilling-out assignment in
// `client.unload()`. It is dead state — likely a deobfuscation
// artifact for an unidentified obfuscated class — but kept here for
// literal-port symmetry so Go's Unload matches Java line-for-line.
//
// Do not add behavior to this package. If a future audit identifies
// the real class behind `class61`, the package can be retired in
// favor of the proper type.
package class61

// Class61 mirrors Java's empty `class class61`. No fields, no
// methods — purely a type name for the static array's element type.
type Class61 struct{}

// Instances mirrors Java's `public static class61[] instances`. Only
// referenced by client.Unload (`class61.Instances = nil`); never
// read.
var Instances []*Class61
