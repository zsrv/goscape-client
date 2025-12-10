package signlink

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

var (
	//Socket // TODO
	DNSReq  string
	DNS     string
	LoadReq string
	LoadBuf []byte
	SaveReq string
	SaveBuf []byte
	URLReq  string
	//URLStream // TODO: byte buffer etc?
	LoopRate      int = 50
	Midi          string
	Save          string
	Wave          string
	ReportError   bool = true
	ErrorName     string
	ClientVersion int = 225
	MidiFade      int
	MidiPos       int
	MidiVol       int
	SaveLen       int
	SocketReq     int
	ThreadLiveID  int
	UID           int
	WavePos       int
	WaveVol       int
	//MainApp Applet
	SocketIP net.IPAddr // TODO: string?
	Active   bool
	MidiPlay bool
	SunJava  bool
	WavePlay bool
)

type SignLink struct {
}

func StartPriv(arg0 string) {
	if Active {
		time.Sleep(500 * time.Millisecond)
		Active = false
	}
	SocketReq = 0
	// ThreadReq = nil
	DNSReq = ""
	LoadReq = ""
	SaveReq = ""
	URLReq = ""
	//SocketIP = nil
	// TODO: go signlink.run()
	for !Active {
		time.Sleep(50 * time.Millisecond)
	}
}

func Run() {
	Active = true
	//var1 := FindCacheDir()

}

func FindCacheDir() string {
	var0 := []string{"c:/windows/", "c:/winnt/", "d:/windows/", "d:/winnt/", "e:/windows/", "e:/winnt/", "f:/windows/", "f:/winnt/", "c:/", "~/", "/tmp/", ""}
	var1 := ".file_store_32"
	for i := range len(var0) {
		var3 := var0[i]
		if len(var3) > 0 {
			if _, err := os.Stat(var3); err != nil {
				fmt.Printf("couldn't find cache at %s\n: %v", var3, err)
				continue
			}
		}
		var4 := path.Join(var3, var1)
		_, err := os.Stat(var4)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				err2 := os.Mkdir(var4, 0755)
				if err2 != nil {
					fmt.Printf("couldn't create cache at %s\n: %v", var4, err2)
					continue
				}
				return path.Join(var3, var1, "/")
			}
		}
		return path.Join(var3, var1, "/")
	}
	return ""
}

func GetUID(arg0 string) int {
	var1 := path.Join(arg0, "uid.dat")
	stat, err := os.Stat(var1)
	if err != nil || stat.Size() < 4 {
		bs := make([]byte, 4)
		binary.LittleEndian.PutUint32(bs, uint32(rand.Float64()*9.9999999e7))
		os.WriteFile(var1, bs, 0644)
	}

	var5, err := os.ReadFile(var1)
	if err != nil {
		fmt.Println("couldn't read uid.dat")
		return 0
	}
	var6 := binary.LittleEndian.Uint32(var5)
	return int(var6 + 1)
}

func GetHash(arg0 string) int64 {
	var5 := strings.TrimSpace(arg0)
	var1 := int64(0)
	for i := 0; i < len(var5) && i < 12; i++ {
		var4 := var5[i]
		var1 *= 37
		if var4 >= 'A' && var4 <= 'Z' {
			var1 += int64(var4) + 1 - 65
		} else if var4 >= 'a' && var4 <= 'z' {
			var1 += int64(var4) + 1 - 97
		} else if var4 >= '0' && var4 <= '9' {
			var1 += int64(var4) + 27 - 48
		}
	}
	return var1
}

func CacheLoad(arg0 string) []byte {
	// TODO: synchronized
	if !Active {
		return nil
	}
	LoadReq = strconv.FormatInt(GetHash(arg0), 10)
	for LoadReq != "" {
		time.Sleep(1 * time.Millisecond)
	}
	return LoadBuf
}

func CacheSave(arg0 string, arg1 []byte) {
	// TODO: synchronized
	if !Active || len(arg1) > 2000000 {
		return
	}
	for SaveReq != "" {
		time.Sleep(1 * time.Millisecond)
	}
	SaveLen = len(arg1)
	SaveBuf = arg1
	SaveReq = strconv.FormatInt(GetHash(arg0), 10)
	for SaveReq != "" {
		time.Sleep(1 * time.Millisecond)
	}
}

// TODO
