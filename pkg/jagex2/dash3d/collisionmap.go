package dash3d

type CollisionMap struct {
	OffsetX int
	OffsetZ int
	SizeX   int
	SizeZ   int
	Flags   [][]int
}

func NewCollisionMap(x, z int) *CollisionMap {
	var m CollisionMap
	m.SizeX = x
	m.SizeZ = z
	m.Flags = make([][]int, m.SizeX)
	for i := range m.Flags {
		m.Flags[i] = make([]int, m.SizeZ)
	}
	m.Reset()
	return &m
}

func (m *CollisionMap) Reset() {
	for x := range m.SizeX {
		for z := range m.SizeZ {
			if x == 0 || z == 0 || x == m.SizeX-1 || z == m.SizeZ-1 {
				m.Flags[x][z] = 0xFFFFFF
			} else {
				m.Flags[x][z] = 0
			}
		}
	}
}

func (m *CollisionMap) AddWall(arg1 int, arg2 int, arg3 int, arg4 bool, arg5 int) {
	var8 := arg3 - m.OffsetX
	var7 := arg2 - m.OffsetZ
	switch arg5 {
	case 0:
		switch arg1 {
		case 0:
			m.AddCMap(var8, var7, 128)
			m.AddCMap(var8-1, var7, 8)
		case 1:
			m.AddCMap(var8, var7, 2)
			m.AddCMap(var8, var7+1, 32)
		case 2:
			m.AddCMap(var8, var7, 8)
			m.AddCMap(var8+1, var7, 128)
		case 3:
			m.AddCMap(var8, var7, 32)
			m.AddCMap(var8, var7-1, 2)
		}
	case 1, 3:
		switch arg1 {
		case 0:
			m.AddCMap(var8, var7, 1)
			m.AddCMap(var8-1, var7+1, 16)
		case 1:
			m.AddCMap(var8, var7, 4)
			m.AddCMap(var8+1, var7+1, 64)
		case 2:
			m.AddCMap(var8, var7, 16)
			m.AddCMap(var8+1, var7-1, 1)
		case 3:
			m.AddCMap(var8, var7, 64)
			m.AddCMap(var8-1, var7-1, 4)
		}
	case 2:
		switch arg1 {
		case 0:
			m.AddCMap(var8, var7, 130)
			m.AddCMap(var8-1, var7, 8)
			m.AddCMap(var8, var7+1, 32)
		case 1:
			m.AddCMap(var8, var7, 10)
			m.AddCMap(var8, var7+1, 32)
			m.AddCMap(var8+1, var7, 128)
		case 2:
			m.AddCMap(var8, var7, 40)
			m.AddCMap(var8+1, var7, 128)
			m.AddCMap(var8, var7-1, 2)
		case 3:
			m.AddCMap(var8, var7, 160)
			m.AddCMap(var8, var7-1, 2)
			m.AddCMap(var8-1, var7, 8)
		}
	}
	if !arg4 {
		return
	}
	switch arg5 {
	case 0:
		switch arg1 {
		case 0:
			m.AddCMap(var8, var7, 65536)
			m.AddCMap(var8-1, var7, 4096)
		case 1:
			m.AddCMap(var8, var7, 0x400)
			m.AddCMap(var8, var7+1, 16384)
		case 2:
			m.AddCMap(var8, var7, 4096)
			m.AddCMap(var8+1, var7, 65536)
		case 3:
			m.AddCMap(var8, var7, 16384)
			m.AddCMap(var7, var7-1, 0x400)
		}
	case 1, 3:
		switch arg1 {
		case 0:
			m.AddCMap(var8, var7, 512)
			m.AddCMap(var8-1, var7+1, 8192)
		case 1:
			m.AddCMap(var8, var7, 2048)
			m.AddCMap(var8+1, var7+1, 32768)
		case 2:
			m.AddCMap(var8, var7, 8192)
			m.AddCMap(var8+1, var7-1, 512)
		case 3:
			m.AddCMap(var8, var7, 32768)
			m.AddCMap(var8-1, var7-1, 2048)
		}
	case 2:
		switch arg1 {
		case 0:
			m.AddCMap(var8, var7, 65560)
			m.AddCMap(var8-1, var7, 4096)
			m.AddCMap(var8, var7+1, 16384)
		case 1:
			m.AddCMap(var8, var7, 5120)
			m.AddCMap(var8, var7+1, 16384)
			m.AddCMap(var8+1, var7, 65536)
		case 2:
			m.AddCMap(var8, var7, 20480)
			m.AddCMap(var8+1, var7, 65536)
			m.AddCMap(var8, var7-1, 0x400)
		case 3:
			m.AddCMap(var8, var7, 81920)
			m.AddCMap(var8, var7-1, 0x400)
			m.AddCMap(var8-1, var7, 4096)
		}
	}
}

func (m *CollisionMap) AddLoc(arg0, arg1, arg2, arg3, arg5 int, arg6 bool) {
	var8 := 256
	if arg6 {
		var8 += 131072
	}
	var11 := arg3 - m.OffsetX
	var12 := arg5 - m.OffsetZ
	if arg0 == 1 || arg0 == 3 {
		var9 := arg2
		arg2 = arg1
		arg1 = var9
	}
	for i := var11; i < var11+arg2; i++ {
		if i >= 0 && i < m.SizeX {
			for j := var12; j < var12+arg1; j++ {
				if j >= 0 && j < m.SizeZ {
					m.AddCMap(i, j, var8)
				}
			}
		}
	}
}

func (m *CollisionMap) SetBlocked(arg1, arg2 int) {
	var5 := arg2 - m.OffsetX
	var4 := arg1 - m.OffsetZ
	m.Flags[var5][var4] |= 0x200000
}

func (m *CollisionMap) AddCMap(x, z, flag int) {
	m.Flags[x][z] |= flag
}

func (m *CollisionMap) DelWall(arg0 bool, arg1, arg2, arg3, arg5 int) {
	var7 := arg2 - m.OffsetX
	var8 := arg3 - m.OffsetZ
	switch arg5 {
	case 0:
		switch arg1 {
		case 0:
			m.RemCMap(var8, var7, 128)
			m.RemCMap(var8, var7-1, 8)
		case 1:
			m.RemCMap(var8, var7, 2)
			m.RemCMap(var8+1, var7, 32)
		case 2:
			m.RemCMap(var8, var7, 8)
			m.RemCMap(var8, var7+1, 128)
		case 3:
			m.RemCMap(var8, var7, 32)
			m.RemCMap(var8-1, var7, 2)
		}
	case 1, 3:
		switch arg1 {
		case 0:
			m.RemCMap(var8, var7, 1)
			m.RemCMap(var8+1, var7-1, 16)
		case 1:
			m.RemCMap(var8, var7, 4)
			m.RemCMap(var8+1, var7+1, 64)
		case 2:
			m.RemCMap(var8, var7, 16)
			m.RemCMap(var8-1, var7+1, 1)
		case 3:
			m.RemCMap(var8, var7, 64)
			m.RemCMap(var8-1, var7-1, 4)
		}
	case 2:
		switch arg1 {
		case 0:
			m.RemCMap(var8, var7, 130)
			m.RemCMap(var8, var7-1, 8)
			m.RemCMap(var8+1, var7, 32)
		case 1:
			m.RemCMap(var8, var7, 10)
			m.RemCMap(var8+1, var7, 32)
			m.RemCMap(var8, var7+1, 128)
		case 2:
			m.RemCMap(var8, var7, 40)
			m.RemCMap(var8, var7+1, 128)
			m.RemCMap(var8-1, var7, 2)
		case 3:
			m.RemCMap(var8, var7, 160)
			m.RemCMap(var8-1, var7, 2)
			m.RemCMap(var8, var7-1, 8)
		}
	}
	if !arg0 {
		return
	}
	switch arg5 {
	case 0:
		switch arg1 {
		case 0:
			m.RemCMap(var8, var7, 65536)
			m.RemCMap(var8, var7-1, 4096)
		case 1:
			m.RemCMap(var8, var7, 0x400)
			m.RemCMap(var8+1, var7, 16384)
		case 2:
			m.RemCMap(var8, var7, 4096)
			m.RemCMap(var8, var7+1, 65536)
		case 3:
			m.RemCMap(var8, var7, 16384)
			m.RemCMap(var8-1, var7, 0x400)
		}
	case 1, 3:
		switch arg1 {
		case 0:
			m.RemCMap(var8, var7, 512)
			m.RemCMap(var8+1, var7-1, 8192)
		case 1:
			m.RemCMap(var8, var7, 2048)
			m.RemCMap(var8+1, var7+1, 32768)
		case 2:
			m.RemCMap(var8, var7, 8192)
			m.RemCMap(var8-1, var7+1, 512)
		case 3:
			m.RemCMap(var8, var7, 32768)
			m.RemCMap(var8-1, var7-1, 2048)
		}
	case 2:
		switch arg1 {
		case 0:
			m.RemCMap(var8, var7, 66560)
			m.RemCMap(var8, var7-1, 4096)
			m.RemCMap(var8+1, var7, 16384)
		case 1:
			m.RemCMap(var8, var7, 5120)
			m.RemCMap(var8+1, var7, 16384)
			m.RemCMap(var8, var7+1, 65536)
		case 2:
			m.RemCMap(var8, var7, 20480)
			m.RemCMap(var8, var7+1, 65536)
			m.RemCMap(var8-1, var7, 0x400)
		case 3:
			m.RemCMap(var8, var7, 81920)
			m.RemCMap(var8-1, var7, 0x400)
			m.RemCMap(var8, var7-1, 4096)
		}
	}
}

func (m *CollisionMap) DelLoc(arg0, arg1, arg2, arg3 int, arg5 bool, arg6 int) {
	var8 := 256
	if arg5 {
		var8 += 131072
	}
	var12 := arg1 - m.OffsetX
	var11 := arg0 - m.OffsetZ
	if arg2 == 1 || arg2 == 3 {
		var9 := arg3
		arg3 = arg6
		arg6 = var9
	}
	for i := var12; i < var12+arg3; i++ {
		if i >= 0 && i < m.SizeX {
			for j := var11; j < var11+arg6; j++ {
				if j >= 0 && j < m.SizeZ {
					m.RemCMap(j, i, var8)
				}
			}
		}
	}
}

func (m *CollisionMap) RemCMap(z, x, flag int) {
	m.Flags[x][z] &= 0xFFFFFF - flag
}

func (m *CollisionMap) RemoveBlocked(arg0, arg1 int) {
	var5 := arg1 - m.OffsetX
	var4 := arg0 - m.OffsetZ
	m.Flags[var5][var4] &= 0xDFFFFF
}

func (m *CollisionMap) TestWall(arg1, arg2, arg3, arg4, arg5, arg6 int) bool {
	if arg6 == arg5 && arg4 == arg2 {
		return true
	}
	var11 := arg6 - m.OffsetX
	var9 := arg4 - m.OffsetZ
	var10 := arg5 - m.OffsetX
	var8 := arg2 - m.OffsetZ
	switch arg3 {
	case 0:
		switch arg1 {
		case 0:
			if var11 == var10-1 && var9 == var8 {
				return true
			}
			if var11 == var10 && var9 == var8+1 && m.Flags[var11][var9]&0x280120 == 0 {
				return true
			}
			if var11 == var10 && var9 == var8-1 && m.Flags[var11][var9]&0x280102 == 0 {
				return true
			}
		case 1:
			if var11 == var10 && var9 == var8+1 {
				return true
			}
			if var11 == var10-1 && var9 == var8 && m.Flags[var11][var9]&0x280108 == 0 {
				return true
			}
			if var11 == var10+1 && var9 == var8 && m.Flags[var11][var9]&0x280180 == 0 {
				return true
			}
		case 2:
			if var11 == var10+1 && var9 == var8 {
				return true
			}
			if var11 == var10 && var9 == var8+1 && m.Flags[var11][var9]&0x280120 == 0 {
				return true
			}
			if var11 == var10 && var9 == var8-1 && m.Flags[var11][var9]&0x280102 == 0 {
				return true
			}
		case 3:
			if var11 == var10 && var9 == var8-1 {
				return true
			}
			if var11 == var10-1 && var9 == var8 && m.Flags[var11][var9]&0x280108 == 0 {
				return true
			}
			if var11 == var10+1 && var9 == var8 && m.Flags[var11][var9]&0x280180 == 0 {
				return true
			}
		}
	case 2:
		switch arg1 {
		case 0:
			if var11 == var10-1 && var9 == var8 {
				return true
			}
			if var11 == var10 && var9 == var8+1 {
				return true
			}
			if var11 == var10+1 && var9 == var8 && m.Flags[var11][var9]&0x280180 == 0 {
				return true
			}
			if var11 == var10 && var9 == var8-1 && m.Flags[var11][var9]&0x280102 == 0 {
				return true
			}
		case 1:
			if var11 == var10-1 && var9 == var8 && m.Flags[var11][var9]&0x280108 == 0 {
				return true
			}
			if var11 == var10 && var9 == var8+1 {
				return true
			}
			if var11 == var10+1 && var9 == var8 {
				return true
			}
			if var11 == var10 && var9 == var8-1 && m.Flags[var11][var9]&0x280102 == 0 {
				return true
			}
		case 2:
			if var11 == var10-1 && var9 == var8 && m.Flags[var11][var9]&0x280108 == 0 {
				return true
			}
			if var11 == var10 && var9 == var8+1 && m.Flags[var11][var9]&0x280120 == 0 {
				return true
			}
			if var11 == var10+1 && var9 == var8 {
				return true
			}
			if var11 == var10 && var9 == var8-1 {
				return true
			}
		case 3:
			if var11 == var10-1 && var9 == var8 {
				return true
			}
			if var11 == var10 && var9 == var8+1 && m.Flags[var11][var9]&0x280120 == 0 {
				return true
			}
			if var11 == var10+1 && var9 == var8 && m.Flags[var11][var9]&0x280180 == 0 {
				return true
			}
			if var11 == var10 && var9 == var8-1 {
				return true
			}
		}
	case 9:
		if var11 == var10 && var9 == var8+1 && m.Flags[var11][var9]&0x20 == 0 {
			return true
		}
		if var11 == var10 && var9 == var8-1 && m.Flags[var11][var9]&0x2 == 0 {
			return true
		}
		if var11 == var10-1 && var9 == var8 && m.Flags[var11][var9]&0x8 == 0 {
			return true
		}
		if var11 == var10+1 && var9 == var8 && m.Flags[var11][var9]&0x80 == 0 {
			return true
		}
	}
	return false
}

func (m *CollisionMap) TestWDecor(arg0, arg1, arg3, arg4, arg5, arg6 int) bool {
	if arg3 == arg4 && arg5 == arg6 {
		return true
	}
	var8 := arg3 - m.OffsetX
	var10 := arg5 - m.OffsetZ
	var9 := arg4 - m.OffsetX
	var11 := arg6 - m.OffsetZ
	if arg1 == 6 || arg1 == 7 {
		if arg1 == 7 {
			arg0 = (arg0 + 2) & 0x3
		}
		switch arg0 {
		case 0:
			if var8 == var9+1 && var10 == var11 && m.Flags[var8][var10]&0x80 == 0 {
				return true
			}
			if var8 == var9 && var10 == var11-1 && m.Flags[var8][var10]&0x2 == 0 {
				return true
			}
		case 1:
			if var8 == var9-1 && var10 == var11 && m.Flags[var8][var10]&0x8 == 0 {
				return true
			}
			if var8 == var9 && var10 == var11-1 && m.Flags[var8][var10]&0x2 == 0 {
				return true
			}
		case 2:
			if var8 == var9-1 && var10 == var11 && m.Flags[var8][var10]&0x8 == 0 {
				return true
			}
			if var8 == var9 && var10 == var11+1 && m.Flags[var8][var10]&0x20 == 0 {
				return true
			}
		case 3:
			if var8 == var9+1 && var10 == var11 && m.Flags[var8][var10]&0x80 == 0 {
				return true
			}
			if var8 == var9 && var10 == var11+1 && m.Flags[var8][var10]&0x20 == 0 {
				return true
			}
		}
	}
	if arg1 == 8 {
		if var8 == var9 && var10 == var11+1 && m.Flags[var8][var10]&0x20 == 0 {
			return true
		}
		if var8 == var9 && var10 == var11-1 && m.Flags[var8][var10]&0x2 == 0 {
			return true
		}
		if var8 == var9-1 && var10 == var11 && m.Flags[var8][var10]&0x8 == 0 {
			return true
		}
		if var8 == var9+1 && var10 == var11 && m.Flags[var8][var10]&0x80 == 0 {
			return true
		}
	}
	return false
}

func (m *CollisionMap) TestLoc(arg0, arg1, arg2, arg3, arg4, arg5, arg6 int) bool {
	var9 := arg3 + arg6 - 1
	var10 := arg5 + arg1 - 1
	if arg2 >= arg3 && arg2 <= var9 && arg0 >= arg5 && arg0 <= var10 {
		return true
	} else if arg2 == arg3-1 && arg0 >= arg5 && arg0 <= var10 && m.Flags[arg2-m.OffsetX][arg0-m.OffsetZ]&0x8 == 0 && arg4&0x8 == 0 {
		return true
	} else if arg2 == var9+1 && arg0 >= arg5 && arg0 <= var10 && m.Flags[arg2-m.OffsetX][arg0-m.OffsetZ]&0x80 == 0 && arg4&0x2 == 0 {
		return true
	} else if arg0 == arg5-1 && arg2 >= arg3 && arg2 <= var9 && m.Flags[arg2-m.OffsetX][arg0-m.OffsetZ]&0x2 == 0 && arg4&0x4 == 0 {
		return true
	} else {
		return arg0 == var10+1 && arg2 >= arg3 && arg2 <= var9 && m.Flags[arg2-m.OffsetX][arg0-m.OffsetZ]&0x20 == 0 && arg4&0x1 == 0
	}
}
