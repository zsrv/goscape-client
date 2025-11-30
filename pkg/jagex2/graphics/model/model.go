package model

import (
	"fmt"
	"math"
	"slices"

	"goscape-client/pkg/jagex2/graphics/animframe"
	"goscape-client/pkg/jagex2/graphics/metadata"
	"goscape-client/pkg/jagex2/graphics/pix2d"
	"goscape-client/pkg/jagex2/graphics/pix3d"
	"goscape-client/pkg/jagex2/graphics/vertexnormal"
	"goscape-client/pkg/jagex2/io"
)

var (
	Face1                  *io.Packet
	Face2                  *io.Packet
	Face3                  *io.Packet
	Face4                  *io.Packet
	Face5                  *io.Packet
	Point1                 *io.Packet
	Point2                 *io.Packet
	Point3                 *io.Packet
	Point4                 *io.Packet
	Point5                 *io.Packet
	Vertex1                *io.Packet
	Vertex2                *io.Packet
	Axis                   *io.Packet
	FaceClippedX           []bool  = make([]bool, 4096)
	FaceNearClipped        []bool  = make([]bool, 4096)
	VertexScreenX          []int   = make([]int, 4096)
	VertexScreenY          []int   = make([]int, 4096)
	VertexScreenZ          []int   = make([]int, 4096)
	VertexViewSpaceX       []int   = make([]int, 4096)
	VertexViewSpaceY       []int   = make([]int, 4096)
	VertexViewSpaceZ       []int   = make([]int, 4096)
	TmpDepthFaceCount      []int   = make([]int, 1500)
	TmpDepthFaces          [][]int = make([][]int, 1500)
	TmpPriorityFaceCount   []int   = make([]int, 12)
	TmpPriorityFaces       [][]int = make([][]int, 12)
	TmpPriority10FaceDepth []int   = make([]int, 2000)
	TmpPriority11FaceDepth []int   = make([]int, 2000)
	TmpPriorityDepthSum    []int   = make([]int, 12)
	ClippedX               []int   = make([]int, 10)
	ClippedY               []int   = make([]int, 10)
	ClippedColour          []int   = make([]int, 10)
	PickedBitsets          []int   = make([]int, 1000)
	Sin                    []int   = pix3d.SinTable
	Cos                    []int   = pix3d.CosTable
	Palette                []int   = pix3d.ColourTable
	Reciprocal16           []int   = pix3d.DivTable2
	BaseX                  int
	BaseY                  int
	BaseZ                  int
	MouseX                 int
	MouseZ                 int
	PickedCount            int
	Head                   *io.Packet
	CheckHover             bool
	Metadata               []*metadata.Metadata
)

func init() {
	for i := range TmpDepthFaces {
		TmpDepthFaces[i] = make([]int, 512)
	}
	for i := range TmpPriorityFaces {
		TmpPriorityFaces[i] = make([]int, 2000)
	}
}

type Model struct {
	VertexCount          int
	VertexX              []int
	VertexY              []int
	VertexZ              []int
	FaceCount            int
	FaceVertexA          []int
	FaceVertexB          []int
	FaceVertexC          []int
	FaceColourA          []int
	FaceColourB          []int
	FaceColourC          []int
	FaceInfo             []int
	FacePriority         []int
	Pickable             bool
	TexturedFaceCount    int
	TexturedVertexA      []int
	TexturedVertexB      []int
	TexturedVertexC      []int
	VertexLabel          []int
	Priority             int
	FaceAlpha            []int
	FaceLabel            []int
	FaceColour           []int
	VertexNormal         []*vertexnormal.VertexNormal
	VertexNormalOriginal []*vertexnormal.VertexNormal
	MaxY                 int
	MinY                 int
	Radius               int
	MinDepth             int
	MaxDepth             int
	MinX                 int
	MaxZ                 int
	MinZ                 int
	MaxX                 int
	LabelFaces           [][]int
	LabelVertices        [][]int
	ObjRaise             int
}

func Unload() {
	Metadata = nil
	Head = nil
	Face1 = nil
	Face2 = nil
	Face3 = nil
	Face4 = nil
	Face5 = nil
	Point1 = nil
	Point2 = nil
	Point3 = nil
	Point4 = nil
	Point5 = nil
	Vertex1 = nil
	Vertex2 = nil
	Axis = nil
	FaceClippedX = nil
	FaceNearClipped = nil
	VertexScreenX = nil
	VertexScreenY = nil
	VertexScreenZ = nil
	VertexViewSpaceX = nil
	VertexViewSpaceY = nil
	VertexViewSpaceZ = nil
	TmpDepthFaceCount = nil
	TmpDepthFaces = nil
	TmpPriorityFaceCount = nil
	TmpPriorityFaces = nil
	TmpPriority10FaceDepth = nil
	TmpPriority11FaceDepth = nil
	TmpPriorityDepthSum = nil
	Sin = nil
	Cos = nil
	Palette = nil
	Reciprocal16 = nil
}

func Unpack(arg1 io.Jagfile) {
	Head = io.NewPacket(arg1.Read("ob_head.dat", nil))
	Face1 = io.NewPacket(arg1.Read("ob_face1.dat", nil))
	Face2 = io.NewPacket(arg1.Read("ob_face2.dat", nil))
	Face3 = io.NewPacket(arg1.Read("ob_face3.dat", nil))
	Face4 = io.NewPacket(arg1.Read("ob_face4.dat", nil))
	Face5 = io.NewPacket(arg1.Read("ob_face5.dat", nil))
	Point1 = io.NewPacket(arg1.Read("ob_point1.dat", nil))
	Point2 = io.NewPacket(arg1.Read("ob_point2.dat", nil))
	Point3 = io.NewPacket(arg1.Read("ob_point3.dat", nil))
	Point4 = io.NewPacket(arg1.Read("ob_point4.dat", nil))
	Point5 = io.NewPacket(arg1.Read("ob_point5.dat", nil))
	Vertex1 = io.NewPacket(arg1.Read("ob_vertex1.dat", nil))
	Vertex2 = io.NewPacket(arg1.Read("ob_vertex2.dat", nil))
	Axis = io.NewPacket(arg1.Read("ob_axis.dat", nil))
	Head.Pos = 0
	Point1.Pos = 0
	Point2.Pos = 0
	Point3.Pos = 0
	Point4.Pos = 0
	Vertex1.Pos = 0
	Vertex2.Pos = 0
	var2 := Head.G2()
	Metadata = make([]*metadata.Metadata, var2+100)
	var3 := 0
	var4 := 0
	var5 := 0
	var6 := 0
	var7 := 0
	var8 := 0
	var9 := 0
	for range var2 {
		var11 := Head.G2()
		Metadata[var11] = new(metadata.Metadata)
		var12 := Metadata[var11]
		var12.VertexCount = Head.G2()
		var12.FaceCount = Head.G2()
		var12.TexturedFaceCount = Head.G1()
		var12.VertexFlagsOffset = Point1.Pos
		var12.VertexXOffset = Point2.Pos
		var12.VertexYOffset = Point3.Pos
		var12.VertexZOffset = Point4.Pos
		var12.FaceVerticesOffset = Vertex1.Pos
		var12.FaceOrientationsOffset = Vertex2.Pos
		var13 := Head.G1()
		var14 := Head.G1()
		var15 := Head.G1()
		var16 := Head.G1()
		var17 := Head.G1()
		var19 := 0
		for range var12.VertexCount {
			var19 = Point1.G1()
			if var19&0x1 != 0 {
				Point2.GSmart()
			}
			if var19&0x2 != 0 {
				Point3.GSmart()
			}
			if var19&0x4 != 0 {
				Point4.GSmart()
			}
		}
		for range var12.FaceCount {
			var20 := Vertex2.G1()
			if var20 == 1 {
				Vertex1.GSmart()
				Vertex1.GSmart()
			}
			Vertex1.GSmart()
		}
		var12.FaceColoursOffset = var5
		var5 += var12.FaceCount * 2
		if var13 == 1 {
			var12.FaceInfosOffset = var6
			var6 += var12.FaceCount
		} else {
			var12.FaceInfosOffset = -1
		}
		if var14 == 255 {
			var12.FacePrioritiesOffset = var7
			var7 += var12.FaceCount
		} else {
			var12.FacePrioritiesOffset = int(-var14 - 1)
		}
		if var15 == 1 {
			var12.FaceAlphasOffset = var8
			var8 += var12.FaceCount
		} else {
			var12.FaceAlphasOffset = -1
		}
		if var16 == 1 {
			var12.FaceLabelsOffset = var9
			var9 += var12.FaceCount
		} else {
			var12.FaceLabelsOffset = -1
		}
		if var17 == 1 {
			var12.VertexLabelsOffset = var4
			var4 += var12.VertexCount
		} else {
			var12.VertexLabelsOffset = -1
		}
		var12.FaceTextureAxisOffset = var3
		var3 += var12.TexturedFaceCount
	}
}

func NewModel1(arg1 int) *Model {
	if Metadata == nil {
		// TODO: or return a Model?
		return nil
	}
	var3 := Metadata[arg1]
	if var3 == nil {
		// TODO: or return a Model?
		fmt.Println("Error model", arg1, "not found!")
		return nil
	}
	var m Model
	m.VertexCount = var3.VertexCount
	m.FaceCount = var3.FaceCount
	m.TexturedFaceCount = var3.TexturedFaceCount
	m.VertexX = make([]int, m.VertexCount)
	m.VertexY = make([]int, m.VertexCount)
	m.VertexZ = make([]int, m.VertexCount)
	m.FaceVertexA = make([]int, m.FaceCount)
	m.FaceVertexB = make([]int, m.FaceCount)
	m.FaceVertexC = make([]int, m.FaceCount)
	m.TexturedVertexA = make([]int, m.TexturedFaceCount)
	m.TexturedVertexB = make([]int, m.TexturedFaceCount)
	m.TexturedVertexC = make([]int, m.TexturedFaceCount)
	if var3.VertexLabelsOffset >= 0 {
		m.VertexLabel = make([]int, m.VertexCount)
	}
	if var3.FaceInfosOffset >= 0 {
		m.FaceInfo = make([]int, m.FaceCount)
	}
	if var3.FacePrioritiesOffset >= 0 {
		m.FacePriority = make([]int, m.FaceCount)
	} else {
		m.Priority = -var3.FacePrioritiesOffset - 1
	}
	if var3.FaceAlphasOffset >= 0 {
		m.FaceAlpha = make([]int, m.FaceCount)
	}
	if var3.FaceLabelsOffset >= 0 {
		m.FaceLabel = make([]int, m.FaceCount)
	}
	m.FaceColour = make([]int, m.FaceCount)
	Point1.Pos = var3.VertexFlagsOffset
	Point2.Pos = var3.VertexXOffset
	Point3.Pos = var3.VertexYOffset
	Point4.Pos = var3.VertexZOffset
	Point5.Pos = var3.VertexLabelsOffset
	var4 := 0
	var5 := 0
	var6 := 0
	var9 := 0
	var10 := 0
	var11 := 0
	for i := range m.VertexCount {
		var8 := Point1.G1()
		var9 = 0
		if var8&0x1 != 0 {
			var9 = Point2.GSmart()
		}
		var10 = 0
		if var8&0x2 != 0 {
			var10 = Point3.GSmart()
		}
		var11 = 0
		if var8&0x4 != 0 {
			var11 = Point4.GSmart()
		}
		m.VertexX[i] = var4 + var9
		m.VertexY[i] = var5 + var10
		m.VertexZ[i] = var6 + var11
		var4 = m.VertexX[i]
		var5 = m.VertexY[i]
		var6 = m.VertexZ[i]
		if m.VertexLabel != nil {
			m.VertexLabel[i] = Point5.G1()
		}
	}
	Face1.Pos = var3.FaceColoursOffset
	Face2.Pos = var3.FaceInfosOffset
	Face3.Pos = var3.FacePrioritiesOffset
	Face4.Pos = var3.FaceAlphasOffset
	Face5.Pos = var3.FaceLabelsOffset
	for i := range m.FaceCount {
		m.FaceColour[i] = Face1.G2()
		if m.FaceInfo != nil {
			m.FaceInfo[i] = Face2.G1()
		}
		if m.FacePriority != nil {
			m.FacePriority[i] = Face3.G1()
		}
		if m.FaceAlpha != nil {
			m.FaceAlpha[i] = Face4.G1()
		}
		if m.FaceLabel != nil {
			m.FaceLabel[i] = Face5.G1()
		}
	}
	Vertex1.Pos = var3.FaceVerticesOffset
	Vertex2.Pos = var3.FaceOrientationsOffset
	var9 = 0
	var10 = 0
	var11 = 0
	var12 := 0
	var14 := 0
	for i := range m.FaceCount {
		var14 = Vertex2.G1()
		if var14 == 1 {
			var9 = Vertex1.GSmart() + var12
			var10 = Vertex1.GSmart() + var9
			var11 = Vertex1.GSmart() + var10
			var12 = var11
			m.FaceVertexA[i] = var9
			m.FaceVertexB[i] = var10
			m.FaceVertexC[i] = var11
		}
		if var14 == 2 {
			var10 = var11
			var11 = Vertex1.GSmart() + var12
			var12 = var11
			m.FaceVertexA[i] = var9
			m.FaceVertexB[i] = var10
			m.FaceVertexC[i] = var11
		}
		if var14 == 3 {
			var9 = var11
			var11 = Vertex1.GSmart() + var12
			var12 = var11
			m.FaceVertexA[i] = var9
			m.FaceVertexB[i] = var10
			m.FaceVertexC[i] = var11
		}
		if var14 == 4 {
			var15 := var9
			var9 = var10
			var10 = var15
			var11 = Vertex1.GSmart() + var12
			var12 = var11
			m.FaceVertexA[i] = var9
			m.FaceVertexB[i] = var15
			m.FaceVertexC[i] = var11
		}
	}
	Axis.Pos = var3.FaceTextureAxisOffset * 6
	for i := range m.TexturedFaceCount {
		m.TexturedVertexA[i] = Axis.G2()
		m.TexturedVertexB[i] = Axis.G2()
		m.TexturedVertexC[i] = Axis.G2()
	}
	return &m
}

func NewModel2(arg1 []*Model, arg2 int) *Model {
	var m Model

	var4 := false
	var5 := false
	var6 := false
	var7 := false
	m.VertexCount = 0
	m.FaceCount = 0
	m.TexturedFaceCount = 0
	m.Priority = -1
	for i := range arg2 {
		var9 := arg1[i]
		if var9 != nil {
			m.VertexCount += var9.VertexCount
			m.FaceCount += var9.FaceCount
			m.TexturedFaceCount += var9.TexturedFaceCount
			var4 = var4 || var9.FaceInfo != nil
			if var9.FacePriority == nil {
				if m.Priority == -1 {
					m.Priority = var9.Priority
				}
				if m.Priority != var9.Priority {
					var5 = true
				}
			} else {
				var5 = true
			}
			var6 = var6 || var9.FaceAlpha != nil
			var7 = var7 || var9.FaceLabel != nil
		}
	}
	m.VertexX = make([]int, m.VertexCount)
	m.VertexY = make([]int, m.VertexCount)
	m.VertexZ = make([]int, m.VertexCount)
	m.VertexLabel = make([]int, m.VertexCount)
	m.FaceVertexA = make([]int, m.FaceCount)
	m.FaceVertexB = make([]int, m.FaceCount)
	m.FaceVertexC = make([]int, m.FaceCount)
	m.TexturedVertexA = make([]int, m.TexturedFaceCount)
	m.TexturedVertexB = make([]int, m.TexturedFaceCount)
	m.TexturedVertexC = make([]int, m.TexturedFaceCount)
	if var4 {
		m.FaceInfo = make([]int, m.FaceCount)
	}
	if var5 {
		m.FacePriority = make([]int, m.FaceCount)
	}
	if var6 {
		m.FaceAlpha = make([]int, m.FaceCount)
	}
	if var7 {
		m.FaceLabel = make([]int, m.FaceCount)
	}
	m.FaceColour = make([]int, m.FaceCount)
	m.VertexCount = 0
	m.FaceCount = 0
	m.TexturedFaceCount = 0
	for i := range arg2 {
		var10 := arg1[i]
		if var10 != nil {
			for j := range var10.FaceCount {
				if var4 {
					if var10.FaceInfo == nil {
						m.FaceInfo[m.FaceCount] = 0
					} else {
						m.FaceInfo[m.FaceCount] = var10.FaceInfo[j]
					}
				}
				if var5 {
					if var10.FacePriority == nil {
						m.FacePriority[m.FaceCount] = var10.Priority
					} else {
						m.FacePriority[m.FaceCount] = var10.FacePriority[j]
					}
				}
				if var6 {
					if var10.FaceAlpha == nil {
						m.FaceAlpha[m.FaceCount] = 0
					} else {
						m.FaceAlpha[m.FaceCount] = var10.FaceAlpha[j]
					}
				}
				if var7 && var10.FaceLabel != nil {
					m.FaceLabel[m.FaceCount] = var10.FaceLabel[j]
				}
				m.FaceColourC[m.FaceCount] = var10.FaceColour[j]
				m.FaceVertexA[m.FaceCount] = m.AddVertex(var10, var10.FaceVertexA[j])
				m.FaceVertexB[m.FaceCount] = m.AddVertex(var10, var10.FaceVertexB[j])
				m.FaceVertexC[m.FaceCount] = m.AddVertex(var10, var10.FaceVertexC[j])
				m.FaceCount++
			}
			for j := range var10.TexturedFaceCount {
				m.TexturedVertexA[m.TexturedFaceCount] = m.AddVertex(var10, var10.TexturedVertexA[j])
				m.TexturedVertexB[m.TexturedFaceCount] = m.AddVertex(var10, var10.TexturedVertexB[j])
				m.TexturedVertexC[m.TexturedFaceCount] = m.AddVertex(var10, var10.TexturedVertexC[j])
				m.TexturedFaceCount++
			}
		}
	}
	return &m
}

func NewModel3(arg0 []*Model, arg2 int, arg3 bool) *Model {
	var m Model

	var5 := false
	var6 := false
	var7 := false
	var8 := false
	m.VertexCount = 0
	m.FaceCount = 0
	m.TexturedFaceCount = 0
	m.Priority = -1
	for i := range arg2 {
		var10 := arg0[i]
		if var10 != nil {
			m.VertexCount += var10.VertexCount
			m.FaceCount += var10.FaceCount
			m.TexturedFaceCount += var10.TexturedFaceCount
			var5 = var5 || var10.FaceInfo != nil
			if var10.FacePriority == nil {
				if m.Priority == -1 {
					m.Priority = var10.Priority
				}
				if m.Priority != var10.Priority {
					var6 = true
				}
			} else {
				var6 = true
			}
			var7 = var7 || var10.FaceAlpha != nil
			var8 = var8 || var10.FaceColour != nil
		}
	}
	m.VertexX = make([]int, m.VertexCount)
	m.VertexY = make([]int, m.VertexCount)
	m.VertexZ = make([]int, m.VertexCount)
	m.FaceVertexA = make([]int, m.FaceCount)
	m.FaceVertexB = make([]int, m.FaceCount)
	m.FaceVertexC = make([]int, m.FaceCount)
	m.FaceColourA = make([]int, m.FaceCount)
	m.FaceColourB = make([]int, m.FaceCount)
	m.FaceColourC = make([]int, m.FaceCount)
	m.TexturedVertexA = make([]int, m.TexturedFaceCount)
	m.TexturedVertexB = make([]int, m.TexturedFaceCount)
	m.TexturedVertexC = make([]int, m.TexturedFaceCount)
	if var5 {
		m.FaceInfo = make([]int, m.FaceCount)
	}
	if var6 {
		m.FacePriority = make([]int, m.FaceCount)
	}
	if var7 {
		m.FaceAlpha = make([]int, m.FaceCount)
	}
	if var8 {
		m.FaceColour = make([]int, m.FaceCount)
	}
	m.VertexCount = 0
	m.FaceCount = 0
	m.TexturedFaceCount = 0
	for i := range arg2 {
		var11 := arg0[i]
		if var11 != nil {
			var12 := m.VertexCount
			for j := range var11.VertexCount {
				m.VertexX[m.VertexCount] = var11.VertexX[j]
				m.VertexY[m.VertexCount] = var11.VertexY[j]
				m.VertexZ[m.VertexCount] = var11.VertexZ[j]
				m.VertexCount++
			}
			for j := range var11.FaceCount {
				m.FaceVertexA[m.FaceCount] = var11.FaceVertexA[j] + var12
				m.FaceVertexB[m.FaceCount] = var11.FaceVertexB[j] + var12
				m.FaceVertexC[m.FaceCount] = var11.FaceVertexC[j] + var12
				m.FaceColourA[m.FaceCount] = var11.FaceColourA[j]
				m.FaceColourB[m.FaceCount] = var11.FaceColourB[j]
				m.FaceColourC[m.FaceCount] = var11.FaceColourC[j]
				if var5 {
					if var11.FaceInfo == nil {
						m.FaceInfo[m.FaceCount] = 0
					} else {
						m.FaceInfo[m.FaceCount] = var11.FaceInfo[j]
					}
				}
				if var6 {
					if var11.FacePriority == nil {
						m.FacePriority[m.FaceCount] = var11.Priority
					} else {
						m.FacePriority[m.FaceCount] = var11.FacePriority[j]
					}
				}
				if var7 {
					if var11.FaceAlpha == nil {
						m.FaceAlpha[m.FaceCount] = 0
					} else {
						m.FaceAlpha[m.FaceCount] = var11.FaceAlpha[j]
					}
				}
				if var8 && var11.FaceColour != nil {
					m.FaceColour[m.FaceCount] = var11.FaceColour[j]
				}
				m.FaceCount++
			}
			for j := range var11.TexturedFaceCount {
				m.TexturedVertexA[m.TexturedFaceCount] = var11.TexturedVertexA[j] + var12
				m.TexturedVertexB[m.TexturedFaceCount] = var11.TexturedVertexB[j] + var12
				m.TexturedVertexC[m.TexturedFaceCount] = var11.TexturedVertexC[j] + var12
				m.TexturedFaceCount++
			}
		}
	}
	m.CalculateBoundsCylinder()
	return &m
}

func NewModel4(arg0 *Model, arg1 bool, arg2 bool, arg4 bool) *Model {
	var m Model

	m.VertexCount = arg0.VertexCount
	m.FaceCount = arg0.FaceCount
	m.TexturedFaceCount = arg0.TexturedFaceCount
	if arg4 {
		m.VertexX = arg0.VertexX
		m.VertexY = arg0.VertexY
		m.VertexZ = arg0.VertexZ
	} else {
		m.VertexX = make([]int, m.VertexCount)
		m.VertexY = make([]int, m.VertexCount)
		m.VertexZ = make([]int, m.VertexCount)
		for i := range m.VertexCount {
			m.VertexX[i] = arg0.VertexX[i]
			m.VertexY[i] = arg0.VertexY[i]
			m.VertexZ[i] = arg0.VertexZ[i]
		}
	}
	if arg1 {
		m.FaceColour = arg0.FaceColour
	} else {
		m.FaceColour = make([]int, m.FaceCount)
		for i := range m.FaceCount {
			m.FaceColour[i] = arg0.FaceColour[i]
		}
	}
	if arg2 {
		m.FaceAlpha = arg0.FaceAlpha
	} else {
		m.FaceAlpha = make([]int, m.FaceCount)
		if arg0.FaceAlpha == nil {
			for i := range m.FaceCount {
				m.FaceAlpha[i] = 0
			}
		} else {
			for i := range m.FaceCount {
				m.FaceAlpha[i] = arg0.FaceAlpha[i]
			}
		}
	}
	m.VertexLabel = arg0.VertexLabel
	m.FaceLabel = arg0.FaceLabel
	m.FaceInfo = arg0.FaceInfo
	m.FaceVertexA = arg0.FaceVertexA
	m.FaceVertexB = arg0.FaceVertexB
	m.FaceVertexC = arg0.FaceVertexC
	m.FacePriority = arg0.FacePriority
	m.Priority = arg0.Priority
	m.TexturedVertexA = arg0.TexturedVertexA
	m.TexturedVertexB = arg0.TexturedVertexB
	m.TexturedVertexC = arg0.TexturedVertexC
	return &m
}

func NewModel5(arg0 *Model, arg2 bool, arg3 bool) *Model {
	var m Model

	m.VertexCount = arg0.VertexCount
	m.FaceCount = arg0.FaceCount
	m.TexturedFaceCount = arg0.TexturedFaceCount
	if arg2 {
		m.VertexY = make([]int, m.VertexCount)
		for i := range m.VertexCount {
			m.VertexY[i] = arg0.VertexY[i]
		}
	} else {
		m.VertexY = arg0.VertexY
	}
	if arg3 {
		m.FaceColourA = make([]int, m.FaceCount)
		m.FaceColourB = make([]int, m.FaceCount)
		m.FaceColourC = make([]int, m.FaceCount)
		for i := range m.FaceCount {
			m.FaceColourA[i] = arg0.FaceColourA[i]
			m.FaceColourB[i] = arg0.FaceColourB[i]
			m.FaceColourC[i] = arg0.FaceColourC[i]
		}
		m.FaceInfo = make([]int, m.FaceCount)
		if arg0.FaceInfo == nil {
			for i := range m.FaceCount {
				m.FaceInfo[i] = 0
			}
		} else {
			for i := range m.FaceCount {
				m.FaceInfo[i] = arg0.FaceInfo[i]
			}
		}
		m.VertexNormal = make([]*vertexnormal.VertexNormal, m.VertexCount)
		for i := range m.VertexCount {
			m.VertexNormal[i] = vertexnormal.NewVertexNormal()
			var7 := m.VertexNormal[i]
			var8 := arg0.VertexNormal[i]
			var7.X = var8.X
			var7.Y = var8.Y
			var7.Z = var8.Z
			var7.W = var8.W
		}
		m.VertexNormalOriginal = arg0.VertexNormalOriginal
	} else {
		m.FaceColourA = arg0.FaceColourA
		m.FaceColourB = arg0.FaceColourB
		m.FaceColourC = arg0.FaceColourC
		m.FaceInfo = arg0.FaceInfo
	}
	m.VertexX = arg0.VertexX
	m.VertexZ = arg0.VertexZ
	m.FaceColour = arg0.FaceColour
	m.FaceAlpha = arg0.FaceAlpha
	m.FacePriority = arg0.FacePriority
	m.Priority = arg0.Priority
	m.FaceVertexA = arg0.FaceVertexA
	m.FaceVertexB = arg0.FaceVertexB
	m.FaceVertexC = arg0.FaceVertexC
	m.TexturedVertexA = arg0.TexturedVertexA
	m.TexturedVertexB = arg0.TexturedVertexB
	m.TexturedVertexC = arg0.TexturedVertexC
	m.MaxY = arg0.MaxY
	m.MinY = arg0.MinY
	m.Radius = arg0.Radius
	m.MinDepth = arg0.MinDepth
	m.MaxDepth = arg0.MaxDepth
	m.MinX = arg0.MinX
	m.MaxZ = arg0.MaxZ
	m.MinZ = arg0.MinZ
	m.MaxX = arg0.MaxX
	return &m
}

func NewModel6(arg1 *Model, arg2 bool) *Model {
	var m Model
	m.VertexCount = arg1.VertexCount
	m.FaceCount = arg1.FaceCount
	m.TexturedFaceCount = arg1.TexturedFaceCount
	m.VertexX = make([]int, m.VertexCount)
	m.VertexY = make([]int, m.VertexCount)
	m.VertexZ = make([]int, m.VertexCount)
	for i := range m.VertexCount {
		m.VertexX[i] = arg1.VertexX[i]
		m.VertexY[i] = arg1.VertexY[i]
		m.VertexZ[i] = arg1.VertexZ[i]
	}
	if arg2 {
		m.FaceAlpha = arg1.FaceAlpha
	} else {
		m.FaceAlpha = make([]int, m.FaceCount)
		if arg1.FaceAlpha == nil {
			for i := range m.FaceCount {
				m.FaceAlpha[i] = 0
			}
		} else {
			for i := range m.FaceCount {
				m.FaceAlpha[i] = arg1.FaceAlpha[i]
			}
		}
	}
	m.FaceInfo = arg1.FaceInfo
	m.FaceColour = arg1.FaceColour
	m.FacePriority = arg1.FacePriority
	m.Priority = arg1.Priority
	m.LabelFaces = arg1.LabelFaces
	m.LabelVertices = arg1.LabelVertices
	m.FaceVertexA = arg1.FaceVertexA
	m.FaceVertexB = arg1.FaceVertexB
	m.FaceVertexC = arg1.FaceVertexC
	m.FaceColourA = arg1.FaceColourA
	m.FaceColourB = arg1.FaceColourB
	m.FaceColourC = arg1.FaceColourC
	m.TexturedVertexA = arg1.TexturedVertexA
	m.TexturedVertexB = arg1.TexturedVertexB
	m.TexturedVertexC = arg1.TexturedVertexC
	return &m
}

func (m *Model) AddVertex(arg0 *Model, arg1 int) int {
	var3 := -1
	var4 := arg0.VertexX[arg1]
	var5 := arg0.VertexY[arg1]
	var6 := arg0.VertexZ[arg1]
	for i := range m.VertexCount {
		if var4 == m.VertexX[i] && var5 == m.VertexY[i] && var6 == m.VertexZ[i] {
			var3 = i
			break
		}
	}
	if var3 == -1 {
		m.VertexX[m.VertexCount] = var4
		m.VertexY[m.VertexCount] = var5
		m.VertexZ[m.VertexCount] = var6
		if arg0.VertexLabel != nil {
			m.VertexLabel[m.VertexCount] = arg0.VertexLabel[arg1]
		}
		var3 = m.VertexCount
		m.VertexCount++
	}
	return var3
}

func (m *Model) CalculateBoundsCylinder() {
	m.MaxY = 0
	m.Radius = 0
	m.MinY = 0
	for i := range m.VertexCount {
		var3 := m.VertexX[i]
		var4 := m.VertexY[i]
		var5 := m.VertexZ[i]
		if -var4 > m.MaxY {
			m.MaxY = -var4
		}
		if var4 > m.MinY {
			m.MinY = var4
		}
		var6 := var3*var3 + var5*var5
		if var6 > m.Radius {
			m.Radius = var6
		}
	}
	m.Radius = int(math.Sqrt(float64(m.Radius)) + 0.99)
	m.MinDepth = int(math.Sqrt(float64(m.Radius*m.Radius+m.MaxY*m.MaxY)) + 0.99)
	m.MaxDepth = m.MinDepth + int(math.Sqrt(float64(m.Radius*m.Radius+m.MinY*m.MinY))+0.99)
}

func (m *Model) CalculateBoundsY() {
	m.MaxY = 0
	m.MinY = 0
	for i := range m.VertexCount {
		var3 := m.VertexY[i]
		if -var3 > m.MaxY {
			m.MaxY = -var3
		}
		if var3 > m.MinY {
			m.MinY = var3
		}
	}
	m.MinDepth = int(math.Sqrt(float64(m.Radius*m.Radius+m.MaxY*m.MaxY)) + 0.99)
	m.MaxDepth = m.MinDepth + int(math.Sqrt(float64(m.Radius*m.Radius+m.MinY*m.MinY))+0.99)
}

func (m *Model) CalculateBoundsAABB() {
	m.MaxY = 0
	m.Radius = 0
	m.MinY = 0
	m.MinX = 999999
	m.MaxX = -999999
	m.MaxZ = -99999
	m.MinZ = 99999
	for i := range m.VertexCount {
		var3 := m.VertexX[i]
		var4 := m.VertexY[i]
		var5 := m.VertexZ[i]
		if var3 < m.MinX {
			m.MinX = var3
		}
		if var3 > m.MaxX {
			m.MaxX = var3
		}
		if var5 < m.MinZ {
			m.MinZ = var5
		}
		if var5 > m.MaxZ {
			m.MaxZ = var5
		}
		if -var4 > m.MaxY {
			m.MaxY = -var4
		}
		if var4 > m.MinY {
			m.MinY = var4
		}
		var6 := var3*var3 + var5*var5
		if var6 > m.Radius {
			m.Radius = var6
		}
	}
	m.Radius = int(math.Sqrt(float64(m.Radius)))
	m.MinDepth = int(math.Sqrt(float64(m.Radius*m.Radius + m.MaxY*m.MaxY)))
	m.MaxDepth = m.MinDepth + int(math.Sqrt(float64(m.Radius*m.Radius+m.MinY*m.MinY)))
}

func (m *Model) CreateLabelReferences() {
	if m.VertexLabel != nil {
		var2 := make([]int, 256)
		var3 := 0
		for i := range m.VertexCount {
			var5 := m.VertexLabel[i]
			var2[var5]++
			if var5 > var3 {
				var3 = var5
			}
		}
		m.LabelVertices = make([][]int, var3+1)
		for i := 0; i <= var3; i++ {
			m.LabelVertices[i] = make([]int, var2[i])
			var2[i] = 0
		}
		for i := 0; i < m.VertexCount; i++ {
			var7 := m.VertexLabel[i]
			m.LabelVertices[var7][var2[var7]] = i
			var2[var7]++
		}
		m.VertexLabel = nil
	}
	if m.FaceLabel == nil {
		return
	}
	var2 := make([]int, 256)
	var3 := 0
	for i := range m.FaceCount {
		var5 := m.FaceLabel[i]
		var2[var5]++
		if var5 > var3 {
			var3 = var5
		}
	}
	m.LabelFaces = make([][]int, var3+1)
	for i := 0; i <= var3; i++ {
		m.LabelFaces[i] = make([]int, var2[i])
		var2[i] = 0
	}
	for var6 := 0; var6 < m.FaceCount; var6++ {
		var7 := m.FaceLabel[var6]
		m.LabelFaces[var7][var2[var7]] = var6
		var2[var7]++
	}
	m.FaceLabel = nil
}

func (m *Model) ApplyTransform(arg1 int) {
	if m.LabelVertices == nil || arg1 == -1 {
		return
	}
	var3 := animframe.Instances[arg1]
	var4 := var3.Base
	BaseX = 0
	BaseY = 0
	BaseZ = 0
	for i := range var3.Length {
		var6 := var3.Groups[i]
		m.ApplyTransform2(var4.Types[var6], var4.Labels[var6], var3.X[i], var3.Y[i], var3.Z[i])
	}
}

func (m *Model) ApplyTransforms(arg0 int, arg2 int, arg3 []int) {
	if arg2 == -1 {
		return
	}
	if arg3 == nil || arg0 == -1 {
		m.ApplyTransform(arg2)
		return
	}
	var5 := animframe.Instances[arg2]
	var6 := animframe.Instances[arg0]
	var7 := var5.Base
	BaseX = 0
	BaseY = 0
	BaseZ = 0
	var8 := 0
	var13 := var8 + 1
	var9 := arg3[var8]
	for i := range var5.Length {
		var11 := var5.Groups[i]
		for var11 > var9 {
			var9 = arg3[var13]
			var13++
		}
		if var11 != var9 || var7.Types[var11] == 0 {
			m.ApplyTransform2(var7.Types[var11], var7.Labels[var11], var5.X[i], var5.Y[i], var5.Z[i])
		}
	}
	BaseX = 0
	BaseY = 0
	BaseZ = 0
	var8 = 0
	var13 = var8 + 1
	var9 = arg3[var8]
	for i := range var6.Length {
		var12 := var6.Groups[i]
		for var12 > var9 {
			var9 = arg3[var13]
			var13++
		}
		if var12 == var9 || var7.Types[var12] == 0 {
			m.ApplyTransform2(var7.Types[var12], var7.Labels[var12], var6.X[i], var6.Y[i], var6.Z[i])
		}
	}
}

func (m *Model) ApplyTransform2(arg0 int, arg1 []int, arg2 int, arg3 int, arg4 int) {
	var6 := len(arg1)
	if arg0 == 0 {
		var7 := 0
		BaseX = 0
		BaseY = 0
		BaseZ = 0
		for i := range var6 {
			var18 := arg1[i]
			if var18 < len(m.LabelVertices) {
				var19 := m.LabelVertices[var18]
				for j := range len(var19) {
					var12 := var19[j]
					BaseX += m.VertexX[var12]
					BaseY += m.VertexY[var12]
					BaseZ += m.VertexZ[var12]
					var7++
				}
			}
		}
		if var7 > 0 {
			BaseX = BaseX/var7 + arg2
			BaseY = BaseY/var7 + arg3
			BaseZ = BaseZ/var7 + arg4
		} else {
			BaseX = arg2
			BaseY = arg3
			BaseZ = arg4
		}
		return
	}
	if arg0 == 1 {
		for i := range var6 {
			var8 := arg1[i]
			if var8 < len(m.LabelVertices) {
				var9 := m.LabelVertices[var8]
				for j := range len(var9) {
					var11 := var9[j]
					m.VertexX[var11] += arg2
					m.VertexY[var11] += arg3
					m.VertexZ[var11] += arg4
				}
			}
		}
	} else if arg0 == 2 {
		for i := range var6 {
			var8 := arg1[i]
			if var8 < len(m.LabelVertices) {
				var9 := m.LabelVertices[var8]
				for j := range len(var9) {
					var11 := var9[j]
					m.VertexX[var11] -= BaseX
					m.VertexY[var11] -= BaseY
					m.VertexZ[var11] -= BaseZ
					var12 := (arg2 & 0xFF) * 8
					var13 := (arg3 & 0xFF) * 8
					var14 := (arg4 & 0xFF) * 8
					var15 := 0
					var16 := 0
					var17 := 0
					if var14 != 0 {
						var15 = Sin[var14]
						var16 = Cos[var14]
						var17 = m.VertexY[var11]*var15 + m.VertexX[var11]*var16>>16
						m.VertexY[var11] = m.VertexY[var11]*var16 - m.VertexX[var11]*var15>>16
						m.VertexX[var11] = var17
					}
					if var12 != 0 {
						var15 = Sin[var12]
						var16 = Cos[var12]
						var17 = m.VertexY[var11]*var16 - m.VertexZ[var11]*var15>>16
						m.VertexZ[var11] = m.VertexY[var11]*var15 + m.VertexZ[var11]*var16>>16
						m.VertexY[var11] = var17
					}
					if var13 != 0 {
						var15 = Sin[var13]
						var16 = Cos[var13]
						var17 = m.VertexZ[var11]*var15 + m.VertexX[var11]*var16>>16
						m.VertexZ[var11] = m.VertexZ[var11]*var16 - m.VertexX[var11]*var15>>16
						m.VertexX[var11] = var17
					}
					m.VertexX[var11] += BaseX
					m.VertexY[var11] += BaseY
					m.VertexZ[var11] += BaseZ
				}
			}
		}
	} else if arg0 == 3 {
		for i := range var6 {
			var8 := arg1[i]
			if var8 < len(m.LabelVertices) {
				var9 := m.LabelVertices[var8]
				for j := range len(var9) {
					var11 := var9[j]
					m.VertexX[var11] -= BaseX
					m.VertexY[var11] -= BaseY
					m.VertexZ[var11] -= BaseZ
					m.VertexX[var11] = m.VertexX[var11] * arg2 / 128
					m.VertexY[var11] = m.VertexY[var11] * arg3 / 128
					m.VertexZ[var11] = m.VertexZ[var11] * arg4 / 128
					m.VertexX[var11] += BaseX
					m.VertexY[var11] += BaseY
					m.VertexZ[var11] += BaseZ
				}
			}
		}
	} else if arg0 == 5 && m.LabelFaces != nil && m.FaceAlpha != nil {
		for i := range var6 {
			var8 := arg1[i]
			if var8 < len(m.LabelFaces) {
				var9 := m.LabelFaces[var8]
				for j := range len(var9) {
					var11 := var9[j]
					m.FaceAlpha[var11] += arg2 * 8
					if m.FaceAlpha[var11] < 0 {
						m.FaceAlpha[var11] = 0
					}
					if m.FaceAlpha[var11] > 255 {
						m.FaceAlpha[var11] = 255
					}
				}
			}
		}
	}
}

func (m *Model) RotateY90() {
	for i := range m.VertexCount {
		var3 := m.VertexX[i]
		m.VertexX[i] = m.VertexZ[i]
		m.VertexZ[i] = -var3
	}
}

func (m *Model) RotateX(arg1 int) {
	var3 := Sin[arg1]
	var4 := Cos[arg1]
	for i := range m.VertexCount {
		var6 := m.VertexY[i]*var4 - m.VertexZ[i]*var3>>16
		m.VertexZ[i] = m.VertexY[i]*var3 + m.VertexZ[i]*var4>>16
		m.VertexY[i] = var6
	}
}

func (m *Model) Translate(arg0 int, arg1 int, arg3 int) {
	for i := range m.VertexCount {
		m.VertexX[i] += arg1
		m.VertexY[i] += arg0
		m.VertexZ[i] += arg3
	}
}

func (m *Model) Recolor(arg0 int, arg1 int) {
	for i := range m.FaceCount {
		if m.FaceColour[i] == arg0 {
			m.FaceColour[i] = arg1
		}
	}
}

func (m *Model) RotateY180() {
	for i := range m.VertexCount {
		m.VertexZ[i] = -m.VertexZ[i]
	}
	for i := range m.FaceCount {
		var4 := m.FaceVertexA[i]
		m.FaceVertexA[i] = m.FaceVertexC[i]
		m.FaceVertexC[i] = var4
	}
}

func (m *Model) Scale(arg0, arg2, arg3 int) {
	for i := range m.VertexCount {
		m.VertexX[i] = m.VertexX[i] * arg3 / 128
		m.VertexY[i] = m.VertexY[i] * arg2 / 128
		m.VertexZ[i] = m.VertexZ[i] * arg0 / 128
	}
}

func (m *Model) CalculateNormals(arg0, arg1, arg2, arg3, arg4 int, arg5 bool) {
	var7 := int(math.Sqrt(float64(arg2*arg2 + arg3*arg3 + arg4*arg4)))
	var8 := arg1 * var7 >> 8
	if m.FaceColourA == nil {
		m.FaceColourA = make([]int, m.FaceCount)
		m.FaceColourB = make([]int, m.FaceCount)
		m.FaceColourC = make([]int, m.FaceCount)
	}
	if m.VertexNormal == nil {
		m.VertexNormal = make([]*vertexnormal.VertexNormal, m.VertexCount)
		for i := range m.VertexCount {
			m.VertexNormal[i] = vertexnormal.NewVertexNormal()
		}
	}
	for i := range m.FaceCount {
		var10 := m.FaceVertexA[i]
		var11 := m.FaceVertexB[i]
		var12 := m.FaceVertexC[i]
		var13 := m.VertexX[var11] - m.VertexX[var10]
		var14 := m.VertexY[var11] - m.VertexY[var10]
		var15 := m.VertexZ[var11] - m.VertexZ[var10]
		var16 := m.VertexX[var12] - m.VertexX[var10]
		var17 := m.VertexY[var12] - m.VertexY[var10]
		var18 := m.VertexZ[var12] - m.VertexZ[var10]
		var19 := var14*var18 - var17*var15
		var20 := var15*var16 - var18*var13
		var21 := var13*var17 - var16*var14
		for var19 > 8192 || var20 > 8192 || var21 > 8192 || var19 < -8192 || var20 < -8192 || var21 < -8192 {
			var19 >>= 0x1
			var20 >>= 0x1
			var21 >>= 0x1
		}
		var22 := int(math.Sqrt(float64(var19*var19 + var20*var20 + var21*var21)))
		if var22 <= 0 {
			var22 = 1
		}
		var19 = var19 * 256 / var22
		var20 = var20 * 256 / var22
		var21 = var21 * 256 / var22
		if m.FaceInfo == nil || m.FaceInfo[i]&0x1 == 0 {
			var23 := m.VertexNormal[var10]
			var23.X += var19
			var23.Y += var20
			var23.Z += var21
			var23.W++
			var26 := m.VertexNormal[var11]
			var26.X += var19
			var26.Y += var20
			var26.Z += var21
			var26.W++
			var27 := m.VertexNormal[var12]
			var27.X += var19
			var27.Y += var20
			var27.Z += var21
			var27.W++
		} else {
			var28 := arg0 + (arg2*var19+arg3*var20+arg4*var21)/(var8+var8/2)
			m.FaceColourA[i] = MulColourLightness(m.FaceColour[i], var28, m.FaceInfo[i])
		}
	}
	if arg5 {
		m.ApplyLighting(arg0, var8, arg2, arg3, arg4)
	} else {
		m.VertexNormalOriginal = make([]*vertexnormal.VertexNormal, m.VertexCount)
		for i := range m.VertexCount {
			var24 := m.VertexNormal[i]
			m.VertexNormalOriginal[i] = vertexnormal.NewVertexNormal()
			var25 := m.VertexNormalOriginal[i]
			var25.X = var24.X
			var25.Y = var24.Y
			var25.Z = var24.Z
			var25.W = var24.W
		}
	}
	if arg5 {
		m.CalculateBoundsCylinder()
	} else {
		m.CalculateBoundsAABB()
	}
}

func (m *Model) ApplyLighting(arg0, arg1, arg2, arg3, arg4 int) {
	for i := range m.FaceCount {
		var7 := m.FaceVertexA[i]
		var8 := m.FaceVertexB[i]
		var9 := m.FaceVertexC[i]
		if m.FaceInfo == nil {
			var12 := m.FaceColour[i]
			var10 := m.VertexNormal[var7]
			var11 := arg0 + (arg2*var10.X+arg3*var10.Y+arg4*var10.Z)/(arg1*var10.W)
			m.FaceColourA[i] = MulColourLightness(var12, var11, 0)
			var14 := m.VertexNormal[var8]
			var16 := arg0 + (arg2*var14.X+arg3*var14.Y+arg4*var14.Z)/(arg1*var14.W)
			m.FaceColourB[i] = MulColourLightness(var12, var16, 0)
			var15 := m.VertexNormal[var9]
			var17 := arg0 + (arg2*var15.X+arg3*var15.Y+arg4*var15.Z)/(arg1*var15.W)
			m.FaceColourC[i] = MulColourLightness(var12, var17, 0)
		} else if m.FaceInfo[i]&0x1 == 0 {
			var12 := m.FaceColour[i]
			var13 := m.FaceInfo[i]
			var10 := m.VertexNormal[var7]
			var11 := arg0 + (arg2*var10.X+arg3*var10.Y+arg4*var10.Z)/(arg1*var10.W)
			m.FaceColourA[i] = MulColourLightness(var12, var11, var13)
			var10 = m.VertexNormal[var8]
			var11 = arg0 + (arg2*var10.X+arg3*var10.Y+arg4*var10.Z)/(arg1*var10.W)
			m.FaceColourB[i] = MulColourLightness(var12, var11, var13)
			var10 = m.VertexNormal[var9]
			var11 = arg0 + (arg2*var10.X+arg3*var10.Y+arg4*var10.Z)/(arg1*var10.W)
			m.FaceColourC[i] = MulColourLightness(var12, var11, var13)
		}
	}
	m.VertexNormal = nil
	m.VertexNormalOriginal = nil
	m.VertexLabel = nil
	m.FaceLabel = nil
	if m.FaceInfo != nil {
		for i := range m.FaceCount {
			if m.FaceInfo[i]&0x2 == 2 {
				return
			}
		}
	}
	m.FaceColour = nil
}

func MulColourLightness(arg0, arg1, arg2 int) int {
	if arg2&0x2 == 2 {
		if arg1 < 0 {
			arg1 = 0
		} else if arg1 > 127 {
			arg1 = 127
		}
		return 127 - arg1
	}
	arg1 = arg1 * (arg0 & 0x7F) >> 7
	if arg1 < 2 {
		arg1 = 2
	} else if arg1 > 126 {
		arg1 = 126
	}
	return (arg0 & 0xFF80) + arg1
}

func (m *Model) DrawSimple(arg0, arg1, arg2, arg3, arg4, arg5, arg6 int) {
	var8 := pix3d.CenterW3D
	var9 := pix3d.CenterH3D
	var10 := Sin[arg0]
	var11 := Cos[arg0]
	var12 := Sin[arg1]
	var13 := Cos[arg1]
	var14 := Sin[arg2]
	var15 := Cos[arg2]
	var16 := Sin[arg3]
	var17 := Cos[arg3]
	var18 := arg5*var16 + arg6*var17>>16
	for i := range m.VertexCount {
		var20 := m.VertexX[i]
		var21 := m.VertexY[i]
		var22 := m.VertexZ[i]
		var23 := 0
		if arg2 != 0 {
			var23 = var21*var14 + var20*var15>>16
			var21 = var21*var15 - var20*var14>>16
			var20 = var23
		}
		if arg0 != 0 {
			var23 = var21*var11 - var22*var10>>16
			var22 = var21*var10 + var22*var11>>16
			var21 = var23
		}
		if arg1 != 0 {
			var23 = var22*var12 + var20*var13>>16
			var22 = var22*var13 - var20*var12>>16
			var20 = var23
		}
		var20 += arg4
		var21 += arg5
		var22 += arg6
		var23 = var21*var17 - var22*var16>>16
		var22 = var21*var16 + var22*var17>>16
		VertexScreenZ[i] = var22 - var18
		VertexScreenX[i] = var8 + (var20<<9)/var22
		VertexScreenY[i] = var9 + (var23<<9)/var22
		if m.TexturedFaceCount > 0 {
			VertexViewSpaceX[i] = var20
			VertexViewSpaceY[i] = var23
			VertexViewSpaceZ[i] = var22
		}
	}
	m.Draw2(false, false, 0)
}

func (m *Model) Draw1(arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7, arg8 int) {
	var10 := arg7*arg4 - arg5*arg3>>16
	var11 := arg6*arg1 + var10*arg2>>16
	var12 := m.Radius * arg2 >> 16
	var13 := var11 + var12
	if var13 <= 50 || var11 >= 3500 {
		return
	}
	var14 := arg7*arg3 + arg5*arg4>>16
	var15 := var14 - m.Radius<<9
	if var15/var13 >= pix2d.CenterW2D {
		return
	}
	var16 := var14 + m.Radius<<9
	if var16/var13 <= -pix2d.CenterW2D {
		return
	}
	var17 := arg6*arg2 - var10*arg1>>16
	var18 := m.Radius * arg1 >> 16
	var19 := var17 + var18<<9
	if var19/var13 <= -pix2d.CenterH2D {
		return
	}
	var20 := var18 + (m.MaxY * arg2 >> 16)
	var21 := var17 - var20<<9
	if var21/var13 >= pix2d.CenterH2D {
		return
	}
	var22 := var12 + (m.MaxY * arg1 >> 16)
	var23 := false
	if var11-var22 <= 50 {
		var23 = true
	}
	var24 := false
	var25 := 0
	var26 := 0
	var27 := 0
	if arg8 > 0 && CheckHover {
		var25 = var11 - var12
		if var25 <= 50 {
			var25 = 50
		}
		if var14 > 0 {
			var15 /= var13
			var16 /= var25
		} else {
			var16 /= var13
			var15 /= var25
		}
		if var17 > 0 {
			var21 /= var13
			var19 /= var25
		} else {
			var19 /= var13
			var21 /= var25
		}
		var26 = MouseX - pix3d.CenterW3D
		var27 = MouseZ - pix3d.CenterH3D
		if var26 > var15 && var26 < var16 && var27 > var21 && var27 < var19 {
			if m.Pickable {
				PickedBitsets[PickedCount] = arg8
				PickedCount++
			} else {
				var24 = true
			}
		}
	}
	var25 = pix3d.CenterW3D
	var26 = pix3d.CenterH3D
	var27 = 0
	var28 := 0
	if arg0 != 0 {
		var27 = Sin[arg0]
		var28 = Cos[arg0]
	}
	for i := range m.VertexCount {
		var30 := m.VertexX[i]
		var31 := m.VertexY[i]
		var32 := m.VertexZ[i]
		var33 := 0
		if arg0 != 0 {
			var33 = var32*var27 + var30*var28>>16
			var32 = var32*var28 - var30*var27>>16
			var30 = var33
		}
		var30 += arg5
		var31 += arg6
		var32 += arg7
		var33 = var32*arg3 + var30*arg4>>16
		var32 = var32*arg4 - var30*arg3>>16
		var30 = var33
		var33 = var31*arg2 - var32*arg1>>16
		var32 = var31*arg1 + var32*arg2>>16
		VertexScreenZ[i] = var32 - var11
		if var32 >= 50 {
			VertexScreenX[i] = var25 + (var30<<9)/var32
			VertexScreenY[i] = var26 + (var33<<9)/var32
		} else {
			VertexScreenX[i] = -5000
			var23 = true
		}
		if var23 || m.TexturedFaceCount > 0 {
			VertexViewSpaceX[i] = var30
			VertexViewSpaceY[i] = var33
			VertexViewSpaceZ[i] = var32
		}
	}
	m.Draw2(var23, var24, arg8)
}

func (m *Model) Draw2(arg0 bool, arg1 bool, arg2 int) {
	for i := range m.MaxDepth {
		TmpDepthFaceCount[i] = 0
	}
	var11 := 0
	var12 := 0
	for i := range m.FaceCount {
		if m.FaceInfo == nil || m.FaceInfo[i] != -1 {
			var6 := m.FaceVertexA[i]
			var7 := m.FaceVertexB[i]
			var8 := m.FaceVertexC[i]
			var9 := VertexScreenX[var6]
			var10 := VertexScreenX[var7]
			var11x := VertexScreenX[var8]
			if arg0 && (var9 == -5000 || var10 == -5000 || var11x == -5000) {
				FaceNearClipped[i] = true
				var12 = (VertexScreenZ[var6]+VertexScreenZ[var7]+VertexScreenZ[var8])/3 + m.MinDepth
				TmpDepthFaces[var12][TmpDepthFaceCount[var12]] = i
				TmpDepthFaceCount[var12]++
			} else {
				if arg1 && m.PointWithinTriangle(MouseX, MouseZ, VertexScreenY[var6], VertexScreenY[var7], VertexScreenY[var8], var9, var10, var11x) {
					PickedBitsets[PickedCount] = arg2
					PickedCount++
					arg1 = false
				}
				if (var9-var10)*(VertexScreenY[var8]-VertexScreenY[var7])-(VertexScreenY[var6]-VertexScreenY[var7])*(var11x-var10) > 0 {
					FaceNearClipped[i] = false
					if var9 >= 0 && var10 >= 0 && var11x >= 0 && var9 <= pix2d.SafeWidth && var10 <= pix2d.SafeWidth && var11x <= pix2d.SafeWidth {
						FaceClippedX[i] = false
					} else {
						FaceClippedX[i] = true
					}
					var12 = (VertexScreenZ[var6]+VertexScreenZ[var7]+VertexScreenZ[var8])/3 + m.MinDepth
					TmpDepthFaces[var12][TmpDepthFaceCount[var12]] = i
					TmpDepthFaceCount[var12]++
				}
			}
		}
	}
	if m.FacePriority == nil {
		for i := m.MaxDepth - 1; i >= 0; i-- {
			var7 := TmpDepthFaceCount[i]
			if var7 > 0 {
				var21 := TmpDepthFaces[i]
				for j := range var7 {
					m.DrawFace(var21[j])
				}
			}
		}
		return
	}
	for i := range 12 {
		TmpPriorityFaceCount[i] = 0
		TmpPriorityDepthSum[i] = 0
	}
	for i := m.MaxDepth - 1; i >= 0; i-- {
		var8 := TmpDepthFaceCount[i]
		if var8 > 0 {
			var20 := TmpDepthFaces[i]
			for j := range var8 {
				var11x := var20[j]
				var12 = m.FacePriority[var11x]
				var13 := TmpPriorityFaceCount[var12]
				TmpPriorityFaceCount[var12]++
				TmpPriorityFaces[var12][var13] = var11x
				if var12 < 10 {
					TmpPriorityDepthSum[var12] += i
				} else if var12 == 10 {
					TmpPriority10FaceDepth[var13] = i
				} else {
					TmpPriority11FaceDepth[var13] = i
				}
			}
		}
	}
	var8 := 0
	if TmpPriorityFaceCount[1] > 0 || TmpPriorityFaceCount[2] > 0 {
		var8 = (TmpPriorityDepthSum[1] + TmpPriorityDepthSum[2]) / (TmpPriorityFaceCount[1] + TmpPriorityFaceCount[2])
	}
	var9 := 0
	if TmpPriorityFaceCount[3] > 0 || TmpPriorityFaceCount[4] > 0 {
		var9 = (TmpPriorityDepthSum[3] + TmpPriorityDepthSum[4]) / (TmpPriorityFaceCount[3] + TmpPriorityFaceCount[4])
	}
	var10 := 0
	if TmpPriorityFaceCount[6] > 0 || TmpPriorityFaceCount[8] > 0 {
		var10 = (TmpPriorityDepthSum[6] + TmpPriorityDepthSum[8]) / (TmpPriorityFaceCount[6] + TmpPriorityFaceCount[8])
	}
	var12 = 0
	var13 := TmpPriorityFaceCount[10]
	var14 := TmpPriorityFaces[10]
	var15 := TmpPriority10FaceDepth
	if var12 == var13 {
		var12 = 0
		var13 = TmpPriorityFaceCount[11]
		var14 = TmpPriorityFaces[11]
		var15 = TmpPriority11FaceDepth
	}
	if var12 < var13 {
		var11 = var15[var12]
	} else {
		var11 = -1000
	}
	for i := range 10 {
		for i == 0 && var11 > var8 {
			m.DrawFace(var14[var12])
			var12++
			if var12 == var13 && !slices.Equal(var14, TmpPriorityFaces[11]) {
				var12 = 0
				var13 = TmpPriorityFaceCount[11]
				var14 = TmpPriorityFaces[11]
				var15 = TmpPriority11FaceDepth
			}
			if var12 < var13 {
				var11 = var15[var12]
			} else {
				var11 = -1000
			}
		}
		for i == 3 && var11 > var9 {
			m.DrawFace(var14[var12])
			var12++
			if var12 == var13 && !slices.Equal(var14, TmpPriorityFaces[11]) {
				var12 = 0
				var13 = TmpPriorityFaceCount[11]
				var14 = TmpPriorityFaces[11]
				var15 = TmpPriority11FaceDepth
			}
			if var12 < var13 {
				var11 = var15[var12]
			} else {
				var11 = -1000
			}
		}
		for i == 5 && var11 > var10 {
			m.DrawFace(var14[var12])
			var12++
			if var12 == var13 && !slices.Equal(var14, TmpPriorityFaces[11]) {
				var12 = 0
				var13 = TmpPriorityFaceCount[11]
				var14 = TmpPriorityFaces[11]
				var15 = TmpPriority11FaceDepth
			}
			if var12 < var13 {
				var11 = var15[var12]
			} else {
				var11 = -1000
			}
		}
		var17 := TmpPriorityFaceCount[i]
		var18 := TmpPriorityFaces[i]
		for j := range var17 {
			m.DrawFace(var18[j])
		}
	}
	for var11 != -1000 {
		m.DrawFace(var14[var12])
		var12++
		if var12 == var13 && !slices.Equal(var14, TmpPriorityFaces[11]) {
			var12 = 0
			var14 = TmpPriorityFaces[11]
			var13 = TmpPriorityFaceCount[11]
			var15 = TmpPriority11FaceDepth
		}
		if var12 < var13 {
			var11 = var15[var12]
		} else {
			var11 = -1000
		}
	}
}

func (m *Model) DrawFace(arg0 int) {
	if FaceNearClipped[arg0] {
		m.DrawNearClippedFace(arg0)
		return
	}
	var2 := m.FaceVertexA[arg0]
	var3 := m.FaceVertexB[arg0]
	var4 := m.FaceVertexC[arg0]
	pix3d.HClip = FaceClippedX[arg0]
	if m.FaceAlpha == nil {
		pix3d.Trans = 0
	} else {
		pix3d.Trans = m.FaceAlpha[arg0]
	}
	var5 := 0
	if m.FaceInfo == nil {
		var5 = 0
	} else {
		var5 = m.FaceInfo[arg0] & 0x3
	}
	if var5 == 0 {
		pix3d.GouraudTriangle(VertexScreenY[var2], VertexScreenY[var3], VertexScreenY[var4], VertexScreenX[var2], VertexScreenX[var3], VertexScreenX[var4], m.FaceColourA[arg0], m.FaceColourB[arg0], m.FaceColourC[arg0])
	} else if var5 == 1 {
		pix3d.FlatTriangle(VertexScreenY[var2], VertexScreenY[var3], VertexScreenY[var4], VertexScreenX[var2], VertexScreenX[var3], VertexScreenX[var4], Palette[m.FaceColourA[arg0]])
	} else {
		var6 := 0
		var7 := 0
		var8 := 0
		var9 := 0
		if var5 == 2 {
			var6 = m.FaceInfo[arg0] >> 2
			var7 = m.TexturedVertexA[var6]
			var8 = m.TexturedVertexB[var6]
			var9 = m.TexturedVertexC[var6]
			pix3d.TextureTriangle(VertexScreenY[var2], VertexScreenY[var3], VertexScreenY[var4], VertexScreenX[var2], VertexScreenX[var3], VertexScreenX[var4], m.FaceColourA[arg0], m.FaceColourB[arg0], m.FaceColourC[arg0], VertexViewSpaceX[var7], VertexViewSpaceX[var8], VertexViewSpaceX[var9], VertexViewSpaceY[var7], VertexViewSpaceY[var8], VertexViewSpaceY[var9], VertexViewSpaceZ[var7], VertexViewSpaceZ[var8], VertexViewSpaceZ[var9], m.FaceColour[arg0])
		} else if var5 == 3 {
			var6 = m.FaceInfo[arg0] >> 2
			var7 = m.TexturedVertexA[var6]
			var8 = m.TexturedVertexB[var6]
			var9 = m.TexturedVertexC[var6]
			pix3d.TextureTriangle(VertexScreenY[var2], VertexScreenY[var3], VertexScreenY[var4], VertexScreenX[var2], VertexScreenX[var3], VertexScreenX[var4], m.FaceColourA[arg0], m.FaceColourA[arg0], m.FaceColourA[arg0], VertexViewSpaceX[var7], VertexViewSpaceX[var8], VertexViewSpaceX[var9], VertexViewSpaceY[var7], VertexViewSpaceY[var8], VertexViewSpaceY[var9], VertexViewSpaceZ[var7], VertexViewSpaceZ[var8], VertexViewSpaceZ[var9], m.FaceColour[arg0])
		}
	}
}

func (m *Model) DrawNearClippedFace(arg0 int) {
	var2 := pix3d.CenterW3D
	var3 := pix3d.CenterH3D
	var4 := 0
	var5 := m.FaceVertexA[arg0]
	var6 := m.FaceVertexB[arg0]
	var7 := m.FaceVertexC[arg0]
	var8 := VertexViewSpaceZ[var5]
	var9 := VertexViewSpaceZ[var6]
	var10 := VertexViewSpaceZ[var7]
	var11 := 0
	var12 := 0
	var13 := 0
	var14 := 0
	if var8 >= 50 {
		ClippedX[var4] = VertexScreenX[var5]
		ClippedY[var4] = VertexScreenY[var5]
		ClippedColour[var4] = m.FaceColourA[arg0]
		var4++
	} else {
		var11 = VertexViewSpaceX[var5]
		var12 = VertexViewSpaceY[var5]
		var13 = m.FaceColourA[arg0]
		if var10 >= 50 {
			var14 = (50 - var8) * Reciprocal16[var10-var8]
			ClippedX[var4] = var2 + (var11+((VertexViewSpaceX[var7]-var11)*var14>>16)<<9)/50
			ClippedY[var4] = var3 + (var12+((VertexViewSpaceY[var7]-var12)*var14>>16)<<9)/50
			ClippedColour[var4] = var13 + ((m.FaceColourC[arg0] - var13) * var14 >> 16)
			var4++
		}
		if var9 >= 50 {
			var14 = (50 - var8) * Reciprocal16[var9-var8]
			ClippedX[var4] = var2 + (var11+((VertexViewSpaceX[var6]-var11)*var14>>16)<<9)/50
			ClippedY[var4] = var3 + (var12+((VertexViewSpaceY[var6]-var12)*var14>>16)<<9)/50
			ClippedColour[var4] = var13 + ((m.FaceColourB[arg0] - var13) * var14 >> 16)
			var4++
		}
	}
	if var9 >= 50 {
		ClippedX[var4] = VertexScreenX[var6]
		ClippedY[var4] = VertexScreenY[var6]
		ClippedColour[var4] = m.FaceColourB[arg0]
		var4++
	} else {
		var11 = VertexViewSpaceX[var6]
		var12 = VertexViewSpaceY[var6]
		var13 = m.FaceColourB[arg0]
		if var8 >= 50 {
			var14 = (50 - var9) * Reciprocal16[var8-var9]
			ClippedX[var4] = var2 + (var11+((VertexViewSpaceX[var5]-var11)*var14>>16)<<9)/50
			ClippedY[var4] = var3 + (var12+((VertexViewSpaceY[var5]-var12)*var14>>16)<<9)/50
			ClippedColour[var4] = var13 + ((m.FaceColourA[arg0] - var13) * var14 >> 16)
			var4++
		}
		if var10 >= 50 {
			var14 = (50 - var9) * Reciprocal16[var10-var9]
			ClippedX[var4] = var2 + (var11+((VertexViewSpaceX[var7]-var11)*var14>>16)<<9)/50
			ClippedY[var4] = var3 + (var12+((VertexViewSpaceY[var7]-var12)*var14>>16)<<9)/50
			ClippedColour[var4] = var13 + ((m.FaceColourC[arg0] - var13) * var14 >> 16)
			var4++
		}
	}
	if var10 >= 50 {
		ClippedX[var4] = VertexScreenX[var7]
		ClippedY[var4] = VertexScreenY[var7]
		ClippedColour[var4] = m.FaceColourC[arg0]
		var4++
	} else {
		var11 = VertexViewSpaceX[var7]
		var12 = VertexViewSpaceY[var7]
		var13 = m.FaceColourC[arg0]
		if var9 >= 50 {
			var14 = (50 - var10) * Reciprocal16[var9-var10]
			ClippedX[var4] = var2 + (var11+((VertexViewSpaceX[var6]-var11)*var14>>16)<<9)/50
			ClippedY[var4] = var3 + (var12+((VertexViewSpaceY[var6]-var12)*var14>>16)<<9)/50
			ClippedColour[var4] = var13 + ((m.FaceColourB[arg0] - var13) * var14 >> 16)
			var4++
		}
		if var8 >= 50 {
			var14 = (50 - var10) * Reciprocal16[var8-var10]
			ClippedX[var4] = var2 + (var11+((VertexViewSpaceX[var5]-var11)*var14>>16)<<9)/50
			ClippedY[var4] = var3 + (var12+((VertexViewSpaceY[var5]-var12)*var14>>16)<<9)/50
			ClippedColour[var4] = var13 + ((m.FaceColourA[arg0] - var13) * var14 >> 16)
			var4++
		}
	}
	var11 = ClippedX[0]
	var12 = ClippedX[1]
	var13 = ClippedX[2]
	var14 = ClippedY[0]
	var15 := ClippedY[1]
	var16 := ClippedY[2]
	if (var11-var12)*(var16-var15)-(var14-var15)*(var13-var12) <= 0 {
		return
	}
	pix3d.HClip = false
	var17 := 0
	var18 := 0
	var19 := 0
	var20 := 0
	var21 := 0
	if var4 == 3 {
		if var11 < 0 || var12 < 0 || var13 < 0 || var11 > pix2d.SafeWidth || var12 > pix2d.SafeWidth || var13 > pix2d.SafeWidth {
			pix3d.HClip = true
		}
		if m.FaceInfo == nil {
			var17 = 0
		} else {
			var17 = m.FaceInfo[arg0] & 0x3
		}
		if var17 == 0 {
			pix3d.GouraudTriangle(var14, var15, var16, var11, var12, var13, ClippedColour[0], ClippedColour[1], ClippedColour[2])
		} else if var17 == 1 {
			pix3d.FlatTriangle(var14, var15, var16, var11, var12, var13, Palette[m.FaceColourA[arg0]])
		} else if var17 == 2 {
			var18 = m.FaceInfo[arg0] >> 2
			var19 = m.TexturedVertexA[var18]
			var20 = m.TexturedVertexB[var18]
			var21 = m.TexturedVertexC[var18]
			pix3d.TextureTriangle(var14, var15, var16, var11, var12, var13, ClippedColour[0], ClippedColour[1], ClippedColour[2], VertexViewSpaceX[var19], VertexViewSpaceX[var20], VertexViewSpaceX[var21], VertexViewSpaceY[var19], VertexViewSpaceY[var20], VertexViewSpaceY[var21], VertexViewSpaceZ[var19], VertexViewSpaceZ[var20], VertexViewSpaceZ[var21], m.FaceColour[arg0])
		} else if var17 == 3 {
			var18 = m.FaceInfo[arg0] >> 2
			var19 = m.TexturedVertexA[var18]
			var20 = m.TexturedVertexB[var18]
			var21 = m.TexturedVertexC[var18]
			pix3d.TextureTriangle(var14, var15, var16, var11, var12, var13, m.FaceColourA[arg0], m.FaceColourA[arg0], m.FaceColourA[arg0], VertexViewSpaceX[var19], VertexViewSpaceX[var20], VertexViewSpaceX[var21], VertexViewSpaceY[var19], VertexViewSpaceY[var20], VertexViewSpaceY[var21], VertexViewSpaceZ[var19], VertexViewSpaceZ[var20], VertexViewSpaceZ[var21], m.FaceColour[arg0])
		}
	}
	if var4 != 4 {
		return
	}
	if var11 < 0 || var12 < 0 || var13 < 0 || var11 > pix2d.SafeWidth || var12 > pix2d.SafeWidth || var13 > pix2d.SafeWidth || ClippedX[3] < 0 || ClippedX[3] > pix2d.SafeWidth {
		pix3d.HClip = true
	}
	if m.FaceInfo == nil {
		var17 = 0
	} else {
		var17 = m.FaceInfo[arg0] & 0x3
	}
	if var17 == 0 {
		pix3d.GouraudTriangle(var14, var15, var16, var11, var12, var13, ClippedColour[0], ClippedColour[1], ClippedColour[2])
		pix3d.GouraudTriangle(var14, var16, ClippedY[3], var11, var13, ClippedX[3], ClippedColour[0], ClippedColour[2], ClippedColour[3])
		return
	}
	if var17 == 1 {
		var18 = Palette[m.FaceColourA[arg0]]
		pix3d.FlatTriangle(var14, var15, var16, var11, var12, var13, var18)
		pix3d.FlatTriangle(var14, var16, ClippedY[3], var11, var13, ClippedX[3], var18)
		return
	}
	if var17 == 2 {
		var18 = m.FaceInfo[arg0] >> 2
		var19 = m.TexturedVertexA[var18]
		var20 = m.TexturedVertexB[var18]
		var21 = m.TexturedVertexC[var18]
		pix3d.TextureTriangle(var14, var15, var16, var11, var12, var13, ClippedColour[0], ClippedColour[1], ClippedColour[2], VertexViewSpaceX[var19], VertexViewSpaceX[var20], VertexViewSpaceX[var21], VertexViewSpaceY[var19], VertexViewSpaceY[var20], VertexViewSpaceY[var21], VertexViewSpaceZ[var19], VertexViewSpaceZ[var20], VertexViewSpaceZ[var21], m.FaceColour[arg0])
		pix3d.TextureTriangle(var14, var16, ClippedY[3], var11, var13, ClippedX[3], ClippedColour[0], ClippedColour[2], ClippedColour[3], VertexViewSpaceX[var19], VertexViewSpaceX[var20], VertexViewSpaceX[var21], VertexViewSpaceY[var19], VertexViewSpaceY[var20], VertexViewSpaceY[var21], VertexViewSpaceZ[var19], VertexViewSpaceZ[var20], VertexViewSpaceZ[var21], m.FaceColour[arg0])
		return
	}
	if var17 != 3 {
		return
	}
	var18 = m.FaceInfo[arg0] >> 2
	var19 = m.TexturedVertexA[var18]
	var20 = m.TexturedVertexB[var18]
	var21 = m.TexturedVertexC[var18]
	pix3d.TextureTriangle(var14, var15, var16, var11, var12, var13, m.FaceColourA[arg0], m.FaceColourA[arg0], m.FaceColourA[arg0], VertexViewSpaceX[var19], VertexViewSpaceX[var20], VertexViewSpaceX[var21], VertexViewSpaceY[var19], VertexViewSpaceY[var20], VertexViewSpaceY[var21], VertexViewSpaceZ[var19], VertexViewSpaceZ[var20], VertexViewSpaceZ[var21], m.FaceColour[arg0])
	pix3d.TextureTriangle(var14, var16, ClippedY[3], var11, var13, ClippedX[3], m.FaceColourA[arg0], m.FaceColourA[arg0], m.FaceColourA[arg0], VertexViewSpaceX[var19], VertexViewSpaceX[var20], VertexViewSpaceX[var21], VertexViewSpaceY[var19], VertexViewSpaceY[var20], VertexViewSpaceY[var21], VertexViewSpaceZ[var19], VertexViewSpaceZ[var20], VertexViewSpaceZ[var21], m.FaceColour[arg0])
}

func (m *Model) PointWithinTriangle(arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7 int) bool {
	if arg1 < arg2 && arg1 < arg3 && arg1 < arg4 {
		return false
	} else if arg1 > arg2 && arg1 > arg3 && arg1 > arg4 {
		return false
	} else if arg0 < arg5 && arg0 < arg6 && arg0 < arg7 {
		return false
	} else {
		return arg0 <= arg5 || arg0 <= arg6 || arg0 <= arg7
	}
}
