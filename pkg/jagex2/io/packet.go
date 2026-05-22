package io

import (
	"math/big"
	"strings"
	"sync"
)

// Packet buffer sizes selected by Alloc's typ argument; mirror Java's
// Packet.alloc(int): typ 0 → 100, typ 1 → 5000, typ 2 → 30000.
const (
	minPacketSize = 100
	midPacketSize = 5_000
	maxPacketSize = 30_000
)

var (
	CRCTable []int = make([]int, 256)
	Bitmask  []int = []int{0, 1, 3, 7, 15, 31, 63, 127, 0xFF, 511, 1023, 2047, 4095, 8191, 16383, 32767, 0xFFFF, 131071, 262143, 524287, 1048575, 2097151, 4194303, 8388607, 0xFFFFFF, 33554431, 67108863, 134217727, 268435455, 536870911, 1073741823, 2147483647, -1}
	CacheMin       = sync.Pool{
		New: func() any { return NewPacket(make([]byte, minPacketSize)) },
	}
	CacheMid = sync.Pool{
		New: func() any { return NewPacket(make([]byte, midPacketSize)) },
	}
	CacheMax = sync.Pool{
		New: func() any { return NewPacket(make([]byte, maxPacketSize)) },
	}
)

func init() {
	for b := range 256 {
		remainder := b

		for range 8 {
			if remainder&0x1 == 1 {
				remainder = (remainder >> 1) ^ 0xEDB88320
			} else {
				remainder >>= 0x1
			}
		}

		CRCTable[b] = int(remainder)
	}
}

type Packet struct {
	Data   []byte
	Pos    int
	BitPos int
	Random *Isaac
}

func NewPacket(src []byte) *Packet {
	return &Packet{
		Data: src,
		Pos:  0,
	}
}

func packetPool(typ int) *sync.Pool {
	switch typ {
	case 0:
		return &CacheMin
	case 1:
		return &CacheMid
	case 2:
		return &CacheMax
	default:
		return nil
	}
}

func Alloc(typ int) *Packet {
	pool := packetPool(typ)
	if pool != nil {
		p := pool.Get().(*Packet)
		p.Pos = 0
		return p
	}
	return nil
}

func (p *Packet) Release() {
	p.Pos = 0
	switch len(p.Data) {
	case minPacketSize:
		CacheMin.Put(p)
	case midPacketSize:
		CacheMid.Put(p)
	case maxPacketSize:
		CacheMax.Put(p)
	}
}

func (p *Packet) P1Isaac(ptype int) {
	p.Data[p.Pos] = byte(ptype + int(p.Random.TakeNextValue()))
	p.Pos++
}

func (p *Packet) P1(n int) {
	p.Data[p.Pos] = byte(n)
	p.Pos++
}

func (p *Packet) P2(n int) {
	p.Data[p.Pos] = byte(n >> 8)
	p.Pos++
	p.Data[p.Pos] = byte(n)
	p.Pos++
}

func (p *Packet) IP2(n int) {
	p.Data[p.Pos] = byte(n)
	p.Pos++
	p.Data[p.Pos] = byte(n >> 8)
	p.Pos++
}

func (p *Packet) P3(n int) {
	p.Data[p.Pos] = byte(n >> 16)
	p.Pos++
	p.Data[p.Pos] = byte(n >> 8)
	p.Pos++
	p.Data[p.Pos] = byte(n)
	p.Pos++
}

func (p *Packet) P4(n int) {
	p.Data[p.Pos] = byte(n >> 24)
	p.Pos++
	p.Data[p.Pos] = byte(n >> 16)
	p.Pos++
	p.Data[p.Pos] = byte(n >> 8)
	p.Pos++
	p.Data[p.Pos] = byte(n)
	p.Pos++
}

func (p *Packet) IP4(n int) {
	p.Data[p.Pos] = byte(n)
	p.Pos++
	p.Data[p.Pos] = byte(n >> 8)
	p.Pos++
	p.Data[p.Pos] = byte(n >> 16)
	p.Pos++
	p.Data[p.Pos] = byte(n >> 24)
	p.Pos++
}

func (p *Packet) P8(n int64) {
	p.Data[p.Pos] = byte(n >> 56)
	p.Pos++
	p.Data[p.Pos] = byte(n >> 48)
	p.Pos++
	p.Data[p.Pos] = byte(n >> 40)
	p.Pos++
	p.Data[p.Pos] = byte(n >> 32)
	p.Pos++
	p.Data[p.Pos] = byte(n >> 24)
	p.Pos++
	p.Data[p.Pos] = byte(n >> 16)
	p.Pos++
	p.Data[p.Pos] = byte(n >> 8)
	p.Pos++
	p.Data[p.Pos] = byte(n)
	p.Pos++
}

// PJStr writes a Latin-1 (CP-1252-ish) encoded null-line-terminated string on
// the wire. Java's `arg0.getBytes(0, arg0.length(), this.data, this.pos)`
// (Packet.java:171) copies the low byte of each Java `char` (UTF-16 code unit)
// into the byte array — i.e. Latin-1 encoding for any character < 256.
//
// In this port, Go strings hold UTF-8. We iterate runes and write a single
// byte per rune (the low byte of the code point). Game strings are bounded to
// Latin-1 in practice (only known non-ASCII glyph is '£' = U+00A3, which fits).
func (p *Packet) PJStr(s string) {
	for _, r := range s {
		p.Data[p.Pos] = byte(r)
		p.Pos++
	}
	p.Data[p.Pos] = byte(10)
	p.Pos++
}

func (p *Packet) PData(src []byte, len int, off int) {
	for var5 := off; var5 < off+len; var5++ {
		p.Data[p.Pos] = src[var5]
		p.Pos++
	}
}

func (p *Packet) PSize1(start int) {
	p.Data[p.Pos-start-1] = byte(start)
}

func (p *Packet) G1() int {
	n := int(p.Data[p.Pos]) & 0xFF
	p.Pos++
	return n
}

// G1B reads a signed byte (Java byte; -128..127). Use int(p.G1B()) when
// widening — Go's int(int8) sign-extends, matching Java's byte->int
// promotion. For unsigned storage of the raw byte pattern, wrap as
// byte(p.G1B()) (this is a bit-preserving reinterpretation).
func (p *Packet) G1B() int8 {
	n := int8(p.Data[p.Pos])
	p.Pos++
	return n
}

func (p *Packet) G2() int {
	p.Pos += 2
	return ((int(p.Data[p.Pos-2]) & 0xFF) << 8) + (int(p.Data[p.Pos-1]) & 0xFF)
}

func (p *Packet) G2B() int {
	p.Pos += 2
	n := ((int(p.Data[p.Pos-2]) & 0xFF) << 8) + (int(p.Data[p.Pos-1]) & 0xFF)
	if n > 32767 {
		n -= 65536
	}
	return n
}

func (p *Packet) G3() int {
	p.Pos += 3
	return ((int(p.Data[p.Pos-3]) & 0xFF) << 16) + ((int(p.Data[p.Pos-2]) & 0xFF) << 8) + (int(p.Data[p.Pos-1]) & 0xFF)
}

func (p *Packet) G4() int {
	p.Pos += 4
	return ((int(p.Data[p.Pos-4]) & 0xFF) << 24) + ((int(p.Data[p.Pos-3]) & 0xFF) << 16) + ((int(p.Data[p.Pos-2]) & 0xFF) << 8) + (int(p.Data[p.Pos-1]) & 0xFF)
}

func (p *Packet) G8() int64 {
	high := int64(p.G4()) & 0xFFFFFFFF
	low := int64(p.G4()) & 0xFFFFFFFF
	return (high << 32) + low
}

// GJStr reads a null-line-terminated (\n) Latin-1 encoded string from the wire
// and returns it as a Go (UTF-8) string. Java's `new String(this.data, var1,
// this.pos - var1 - 1)` (Packet.java:239) uses the default platform charset to
// decode bytes, but on the client this is effectively Latin-1: every byte
// 0x00-0xFF maps 1:1 to the matching Unicode code point. We must transcode the
// raw byte slice from Latin-1 → UTF-8 so the resulting Go string is valid UTF-8
// and `for _, r := range s` recovers the original chars (e.g. byte 0xA3 → '£').
func (p *Packet) GJStr() string {
	start := p.Pos
	for p.Data[p.Pos] != 10 {
		p.Pos++
	}
	p.Pos++
	return latin1ToUTF8(p.Data[start : p.Pos-1])
}

// GStrByte reads a null-line-terminated (\n) byte sequence verbatim — Java
// returns the raw bytes here (Packet.java:243-252, `gstrbyte`). Description
// strings on objtype/loctype/npctype are stored byte-for-byte and later
// decoded by the caller via `new String(bytes)`. We preserve the raw byte
// semantics; consumers that need a Go string must call `latin1ToUTF8` or
// otherwise transcode.
func (p *Packet) GStrByte() []byte {
	start := p.Pos
	for p.Data[p.Pos] != 10 {
		p.Pos++
	}
	p.Pos++
	data := make([]byte, p.Pos-start-1)
	for i := start; i < p.Pos-1; i++ {
		data[i-start] = p.Data[i]
	}
	return data
}

// latin1ToUTF8 transcodes a Latin-1 byte slice to a valid UTF-8 Go string.
// Each input byte is treated as a Unicode code point in 0x00..0xFF and
// re-encoded as 1 or 2 UTF-8 bytes. Pure ASCII slices round-trip unchanged.
func latin1ToUTF8(b []byte) string {
	// Fast path: all ASCII.
	ascii := true
	for _, x := range b {
		if x >= 0x80 {
			ascii = false
			break
		}
	}
	if ascii {
		return string(b)
	}
	var sb strings.Builder
	sb.Grow(len(b) + len(b)/4)
	for _, x := range b {
		sb.WriteRune(rune(x))
	}
	return sb.String()
}

func (p *Packet) GData(len int, off int, dst []byte) {
	for i := off; i < off+len; i++ {
		dst[i] = p.Data[p.Pos]
		p.Pos++
	}
}

// Bits
func (p *Packet) AccessBits() {
	p.BitPos = p.Pos * 8
}

func (p *Packet) GBit(n int) int {
	bytePos := p.BitPos >> 3
	remainingBits := 8 - (p.BitPos & 0x7)

	value := int(0)
	p.BitPos += n

	for n > remainingBits {
		value += (int(p.Data[bytePos]) & Bitmask[remainingBits]) << (n - remainingBits)
		bytePos++
		n -= remainingBits
		remainingBits = 8
	}

	if n == remainingBits {
		value += int(p.Data[bytePos]) & Bitmask[remainingBits]
	} else {
		// Java: `data[var3] >> var4 - arg1 & BITMASK[arg1]` — shift has
		// higher precedence than `&`, so the mask is applied to the
		// shifted value. Without these outer parens, the bare Go
		// expression evaluates the same way (>>` and `&` are both
		// multiplicative-level, left-to-right); a previous port wrapped
		// the wrong subexpression in parens, applying the mask to the
		// SHIFT COUNT instead and silently losing the high-bit cap.
		value += (int(p.Data[bytePos]) >> (remainingBits - n)) & Bitmask[n]
	}

	return int(value)
}

// Bytes
func (p *Packet) AccessBytes() {
	p.Pos = (p.BitPos + 7) / 8
}

func (p *Packet) GSmart() int {
	n := int(p.Data[p.Pos] & 0xFF)
	if n < 128 {
		return p.G1() - 64
	}
	return p.G2() - 49152
}

func (p *Packet) GSmartS() int {
	n := int(p.Data[p.Pos] & 0xFF)
	if n < 128 {
		return p.G1()
	}
	return p.G2() - 32768
}

func (p *Packet) RSAEnc(mod *big.Int, exp *big.Int) {
	length := p.Pos
	p.Pos = 0

	plaintextBytes := make([]byte, length)
	p.GData(length, 0, plaintextBytes)

	// Java: new BigInteger(byte[]) parses two's-complement big-endian.
	// Go's SetBytes treats the buffer as unsigned magnitude. The two
	// differ when the first byte has the high bit set, so we mirror
	// Java's signed interpretation explicitly.
	plaintext := javaBigIntFromBytes(plaintextBytes)
	ciphertext := new(big.Int).Exp(plaintext, exp, mod)
	// Java: BigInteger.toByteArray() emits two's-complement and includes
	// a leading 0x00 byte when the magnitude's MSB is set, so a positive
	// 0xFF... value round-trips as positive. Go's Bytes() omits that
	// sign byte. The server (Java) round-trips through BigInteger and
	// will mis-parse the unsigned form for any ciphertext whose first
	// magnitude byte is >= 0x80.
	ciphertextBytes := javaBytesFromBigInt(ciphertext)

	p.Pos = 0
	p.P1(len(ciphertextBytes))
	p.PData(ciphertextBytes, len(ciphertextBytes), 0)
}

// javaBigIntFromBytes parses b as a big-endian two's-complement integer the
// way java.math.BigInteger(byte[]) does. If the high bit of b[0] is set, the
// resulting BigInteger is negative.
func javaBigIntFromBytes(b []byte) *big.Int {
	if len(b) == 0 {
		return new(big.Int)
	}
	n := new(big.Int).SetBytes(b)
	if b[0] >= 0x80 {
		shift := new(big.Int).Lsh(big.NewInt(1), uint(8*len(b)))
		n.Sub(n, shift)
	}
	return n
}

// javaBytesFromBigInt emits n in the byte format Java's
// BigInteger.toByteArray() produces: two's-complement big-endian with a
// leading 0x00 prepended when n is positive and the magnitude's MSB has
// bit 7 set. RSA modPow output is always non-negative, so the negative
// branch isn't reached in practice but is included for completeness.
func javaBytesFromBigInt(n *big.Int) []byte {
	if n.Sign() == 0 {
		return []byte{0}
	}
	if n.Sign() < 0 {
		bitLen := n.BitLen() + 1
		byteLen := (bitLen + 7) / 8
		shift := new(big.Int).Lsh(big.NewInt(1), uint(8*byteLen))
		b := new(big.Int).Add(shift, n).Bytes()
		if len(b) < byteLen {
			pad := make([]byte, byteLen-len(b))
			for i := range pad {
				pad[i] = 0xFF
			}
			b = append(pad, b...)
		}
		return b
	}
	b := n.Bytes()
	if b[0] >= 0x80 {
		return append([]byte{0}, b...)
	}
	return b
}
