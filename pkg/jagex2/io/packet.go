package io

import (
	"math/big"
	"sync"
)

var (
	CRCTable []int = make([]int, 256)
	Bitmask  []int = []int{0, 1, 3, 7, 15, 31, 63, 127, 255, 511, 1023, 2047, 4095, 8191, 16383, 32767, 65535, 131071, 262143, 524287, 1048575, 2097151, 4194303, 8388607, 16777215, 33554431, 67108863, 134217727, 268435455, 536870911, 1073741823, 2147483647, -1}
	CacheMin       = sync.Pool{
		New: func() any { return NewPacket(make([]byte, 0, 100)) },
	}
	CacheMid = sync.Pool{
		New: func() any { return NewPacket(make([]byte, 0, 5_000)) },
	}
	CacheMax = sync.Pool{
		New: func() any { return NewPacket(make([]byte, 0, 30_000)) },
	}
)

func init() {
	for b := 0; b < 256; b++ {
		remainder := b

		for bit := 0; bit < 8; bit++ {
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
	pool := packetPool(int(typ))
	if pool != nil {
		if v := pool.Get(); v != nil {
			p := v.(*Packet)
			p.Pos = 0
			return p
		}
	}
	return NewPacket(make([]byte, 0, typ))
}

func (p *Packet) Release() {
	p.Pos = 0
	if pool := packetPool(len(p.Data)); pool != nil {
		pool.Put(p)
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

func (p *Packet) G1B() byte {
	n := p.Data[p.Pos]
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

func (p *Packet) GJStr() string {
	start := p.Pos
	for p.Data[p.Pos] != 10 {
		p.Pos++
	}
	p.Pos++
	length := p.Pos - start - 1
	return string(p.Data[start : start+length])
}

// GJStrRaw
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
		value += int(p.Data[bytePos]) >> ((remainingBits - n) & Bitmask[n])
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

	plaintext := new(big.Int).SetBytes(plaintextBytes)
	ciphertext := plaintext.Exp(plaintext, exp, mod)
	ciphertextBytes := ciphertext.Bytes()

	p.Pos = 0
	p.P1(int(len(ciphertextBytes)))
	p.PData(ciphertextBytes, int(len(ciphertextBytes)), 0)
}
