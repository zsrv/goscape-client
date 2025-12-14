package io

import (
	"math/big"
	"sync"
)

var (
	CRCTable []int = make([]int, 256)
	BITMASK  []int = []int{0, 1, 3, 7, 15, 31, 63, 127, 255, 511, 1023, 2047, 4095, 8191, 16383, 32767, 65535, 131071, 262143, 524287, 1048575, 2097151, 4194303, 8388607, 16777215, 33554431, 67108863, 134217727, 268435455, 536870911, 1073741823, 2147483647, -1}
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
	for i := 0; i < 256; i++ {
		var0 := i
		for j := 0; j < 8; j++ {
			if var0&0x1 == 1 {
				var0 = var0>>1 ^ 0xEDB88320
			} else {
				var0 >>= 0x1
			}
		}
		CRCTable[i] = int(var0)
	}
}

type Packet struct {
	Data   []byte
	Pos    int
	BitPos int
	Random *Isaac
}

func NewPacket(arg1 []byte) *Packet {
	return &Packet{
		Data: arg1,
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

func (p *Packet) P1Isaac(arg1 int) {
	p.Data[p.Pos] = byte(arg1 + int(p.Random.TakeNextValue()))
	p.Pos++
}

func (p *Packet) P1(arg0 int) {
	p.Data[p.Pos] = byte(arg0)
	p.Pos++
}

func (p *Packet) P2(arg0 int) {
	p.Data[p.Pos] = byte(arg0 >> 8)
	p.Pos++
	p.Data[p.Pos] = byte(arg0)
	p.Pos++
}

func (p *Packet) IP2(arg1 int) {
	p.Data[p.Pos] = byte(arg1)
	p.Pos++
	p.Data[p.Pos] = byte(arg1 >> 8)
	p.Pos++
}

func (p *Packet) P3(arg0 int) {
	p.Data[p.Pos] = byte(arg0 >> 16)
	p.Pos++
	p.Data[p.Pos] = byte(arg0 >> 8)
	p.Pos++
	p.Data[p.Pos] = byte(arg0)
	p.Pos++
}

func (p *Packet) P4(arg0 int) {
	p.Data[p.Pos] = byte(arg0 >> 24)
	p.Pos++
	p.Data[p.Pos] = byte(arg0 >> 16)
	p.Pos++
	p.Data[p.Pos] = byte(arg0 >> 8)
	p.Pos++
	p.Data[p.Pos] = byte(arg0)
	p.Pos++
}

func (p *Packet) IP4(arg0 bool, arg1 int) {
	p.Data[p.Pos] = byte(arg1)
	p.Pos++
	p.Data[p.Pos] = byte(arg1 >> 8)
	p.Pos++
	p.Data[p.Pos] = byte(arg1 >> 16)
	p.Pos++
	if !arg0 {
		p.Data[p.Pos] = byte(arg1 >> 24)
		p.Pos++
	}
}

func (p *Packet) P8(arg1 int64) {
	p.Data[p.Pos] = byte(arg1 >> 56)
	p.Pos++
	p.Data[p.Pos] = byte(arg1 >> 48)
	p.Pos++
	p.Data[p.Pos] = byte(arg1 >> 40)
	p.Pos++
	p.Data[p.Pos] = byte(arg1 >> 32)
	p.Pos++
	p.Data[p.Pos] = byte(arg1 >> 24)
	p.Pos++
	p.Data[p.Pos] = byte(arg1 >> 16)
	p.Pos++
	p.Data[p.Pos] = byte(arg1 >> 8)
	p.Pos++
	p.Data[p.Pos] = byte(arg1)
	p.Pos++
}

func (p *Packet) PJStr(arg0 string) {
	for _, r := range arg0 {
		p.Data[p.Pos] = byte(r)
		p.Pos++
	}
	p.Data[p.Pos] = byte(10)
	p.Pos++
}

func (p *Packet) PData(arg0 []byte, arg1 int, arg2 int) {
	for var5 := arg2; var5 < arg2+arg1; var5++ {
		p.Data[p.Pos] = arg0[var5]
		p.Pos++
	}
}

func (p *Packet) PSize1(arg1 int) {
	p.Data[p.Pos-arg1-1] = byte(arg1)
}

func (p *Packet) G1() int {
	i := int(p.Data[p.Pos]) & 0xFF
	p.Pos++
	return i
}

func (p *Packet) G1B() byte {
	i := p.Data[p.Pos]
	p.Pos++
	return i
}

func (p *Packet) G2() int {
	p.Pos += 2
	return ((int(p.Data[p.Pos-2]) & 0xFF) << 8) + (int(p.Data[p.Pos-1]) & 0xFF)
}

func (p *Packet) G2B() int {
	p.Pos += 2
	var1 := ((int(p.Data[p.Pos-2]) & 0xFF) << 8) + (int(p.Data[p.Pos-1]) & 0xFF)
	if var1 > 32767 {
		var1 -= 65536
	}
	return var1
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
	var2 := int64(p.G4()) & 0xFFFFFFFF
	var4 := int64(p.G4()) & 0xFFFFFFFF
	return (var2 << 32) + var4
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

func (p *Packet) GStrByte() []byte {
	var2 := p.Pos
	for p.Data[p.Pos] != 10 {
		p.Pos++
	}
	p.Pos++
	var3 := make([]byte, p.Pos-var2-1)
	for var4 := var2; var4 < p.Pos-1; var4++ {
		var3[var4-var2] = p.Data[var4]
	}
	return var3
}

func (p *Packet) GData(arg0 int, arg2 int, arg3 []byte) {
	for var5 := arg2; var5 < arg2+arg0; var5++ {
		arg3[var5] = p.Data[p.Pos]
		p.Pos++
	}
}

func (p *Packet) AccessBits() {
	p.BitPos = p.Pos * 8
}

func (p *Packet) GBit(arg1 int) int {
	var3 := p.BitPos >> 3
	var4 := 8 - (p.BitPos & 0x7)
	var5 := int(0)
	p.BitPos += arg1
	for arg1 > var4 {
		var5 += (int(p.Data[var3])&BITMASK[var4])<<arg1 - var4
		var3++
		arg1 -= var4
		var4 = 8
	}
	if arg1 == var4 {
		var5 += int(p.Data[var3]) & BITMASK[var4]
	} else {
		var5 += int(p.Data[var3])>>var4 - arg1&BITMASK[arg1]
	}
	return int(var5)
}

func (p *Packet) AccessBytes() {
	p.Pos = (p.BitPos + 7) / 8
}

func (p *Packet) GSmart() int {
	var1 := int(p.Data[p.Pos] & 0xFF)
	if var1 < 128 {
		return p.G1() - 64
	}
	return p.G2() - 49152
}

func (p *Packet) GSmartS() int {
	var1 := int(p.Data[p.Pos] & 0xFF)
	if var1 < 128 {
		return p.G1()
	}
	return p.G2() - 32768
}

func (p *Packet) RSAEnc(modulus *big.Int, exponent *big.Int) {
	length := p.Pos
	p.Pos = 0

	plaintextBytes := make([]byte, length)
	p.GData(length, 0, plaintextBytes)

	plaintext := new(big.Int).SetBytes(plaintextBytes)
	ciphertext := plaintext.Exp(plaintext, exponent, modulus)
	ciphertextBytes := ciphertext.Bytes()

	p.Pos = 0

	p.P1(int(len(ciphertextBytes)))
	p.PData(ciphertextBytes, int(len(ciphertextBytes)), 0)
}
