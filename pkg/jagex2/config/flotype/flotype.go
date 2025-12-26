package flotype

import (
	"fmt"
	"math/rand"

	"goscape-client/pkg/jagex2/io"
)

var (
	Count     int
	Instances []*FloType
)

type FloType struct {
	RGB        int
	Texture    int
	Overlay    bool
	Occlude    bool
	Name       string
	Hue        int
	Saturation int
	Lightness  int
	Chroma     int
	Luminance  int
	HSL        int
}

func NewFloType() *FloType {
	return &FloType{
		Texture: -1,
		Overlay: false,
		Occlude: true,
	}
}

func Unpack(arg0 *io.Jagfile) {
	var2 := io.NewPacket(arg0.Read("flo.dat", nil))
	Count = var2.G2()
	if Instances == nil {
		Instances = make([]*FloType, Count)
	}
	for i := range Count {
		if Instances[i] == nil {
			Instances[i] = NewFloType()
		}
		Instances[i].Decode(var2)
	}
}

func (f *FloType) Decode(arg1 *io.Packet) {
	for {
		var3 := arg1.G1()
		switch var3 {
		case 0:
			return
		case 1:
			f.RGB = arg1.G3()
			f.SetColour(f.RGB)
		case 2:
			f.Texture = arg1.G1()
		case 3:
			f.Overlay = true
		case 5:
			f.Occlude = true
		case 6:
			f.Name = arg1.GJStr()
		default:
			fmt.Println("Error unrecognised config code:", var3)
		}
	}
}

func (f *FloType) SetColour(arg1 int) {
	var3 := float64(((arg1 >> 16) & 0xFF) / 256.0)
	var22 := float64(((arg1 >> 8) & 0xFF) / 256.0)
	var7 := float64((arg1 & 0xFF) / 256.0)
	var9 := var3
	if var22 < var3 {
		var9 = var22
	}
	if var7 < var9 {
		var9 = var7
	}
	var11 := var3
	if var22 > var3 {
		var11 = var22
	}
	if var7 > var11 {
		var11 = var7
	}
	var13 := float64(0.0)
	var15 := float64(0.0)
	var17 := float64((var9 + var11) / 2.0)
	if var9 != var11 {
		if var17 < 0.5 {
			var15 = (var11 - var9) / (var11 + var9)
		}
		if var17 >= 0.5 {
			var15 = (var11 - var9) / (2.0 - var11 - var9)
		}
		if var3 == var11 {
			var13 = (var22 - var7) / (var11 - var9)
		} else if var22 == var11 {
			var13 = (var7-var3)/(var11-var9) + 2.0
		} else if var7 == var11 {
			var13 = (var3-var22)/(var11-var9) + 4.0
		}
	}
	var13 /= 6.0
	f.Hue = int(var13 * 256.0)
	f.Saturation = int(var15 * 256.0)
	f.Lightness = int(var17 * 256.0)
	if f.Saturation < 0 {
		f.Saturation = 0
	} else if f.Saturation > 255 {
		f.Saturation = 255
	}
	if f.Lightness < 0 {
		f.Lightness = 0
	} else if f.Lightness > 255 {
		f.Lightness = 255
	}
	if var17 > 0.5 {
		f.Luminance = int((1.0 - var17) * var15 * 512.0)
	} else {
		f.Luminance = int(var17 * var15 * 512.0)
	}
	if f.Luminance < 1 {
		f.Luminance = 1
	}
	f.Chroma = int(var13 * float64(f.Luminance))
	var19 := f.Hue + int((rand.Float64()*16.0)-8)
	if var19 < 0 {
		var19 = 0
	} else if var19 > 255 {
		var19 = 255
	}
	var20 := f.Saturation + int((rand.Float64()*48.0)-24)
	if var20 < 0 {
		var20 = 0
	} else if var20 > 255 {
		var20 = 255
	}
	var21 := f.Lightness + int((rand.Float64()*48.0)-24)
	if var21 < 0 {
		var21 = 0
	} else if var21 > 255 {
		var21 = 255
	}
	f.HSL = f.HSL24To16(var19, var20, var21)
}

func (f *FloType) HSL24To16(arg0, arg1, arg2 int) int {
	if arg2 > 179 {
		arg1 /= 2
	}
	if arg2 > 192 {
		arg1 /= 2
	}
	if arg2 > 217 {
		arg1 /= 2
	}
	if arg2 >= 243 {
		arg1 /= 2
	}
	return ((arg0 / 4) << 10) + ((arg1 / 32) << 7) + arg2/2
}
