package signlink

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"goscape-client/pkg/jagex2/client/clientextras"
)

var (
	//Socket // TODO
	DNSReq        string
	DNS           string
	LoadReq       string
	LoadBuf       []byte
	SaveReq       string
	SaveBuf       []byte
	URLReq        string
	URLStream     []byte // this was DataInputStream in java
	LoopRate      int    = 50
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
	//ThreadLiveID  int // not needed in go
	UID     int
	WavePos int
	WaveVol int
	//MainApp Applet
	//SocketIP net.IPAddr // not needed in go
	MidiPlay bool
	SunJava  bool
	WavePlay bool
)

type SignLink struct {
}

func StartPriv() {
	SocketReq = 0
	DNSReq = ""
	LoadReq = ""
	SaveReq = ""
	URLReq = ""
	Run()
}

func Run() {
	var1 := FindCacheDir()
	UID = GetUID(var1)
	for {
		if SocketReq != 0 {
			// TODO: try/catch
			// TODO: make a net.conn or something here and set it to Socket?
			SocketReq = 0
		} else if DNSReq != "" {
			names, err := net.LookupAddr(DNSReq)
			if err != nil || len(names) == 0 {
				DNS = "unknown"
			} else {
				DNS = names[0]
			}
			DNSReq = ""
		} else if LoadReq != "" {
			LoadBuf = nil
			if _, err := os.Stat(var1 + LoadReq); err == nil {
				LoadBuf, err = os.ReadFile(var1 + LoadReq)
				if err != nil {
					fmt.Printf("failed to read file %s: %v\n", var1+LoadReq, err)
				}
			}
			LoadReq = ""
		} else if SaveReq != "" {
			if SaveBuf != nil {
				if err := os.WriteFile(var1+SaveReq, SaveBuf[0:SaveLen], 0644); err != nil {
					fmt.Printf("failed to write file %s: %v\n", var1+SaveReq, err)
				}
			}
			if WavePlay {
				Wave = var1 + SaveReq
				WavePlay = false
			}
			if MidiPlay {
				Midi = var1 + SaveReq
				MidiPlay = false
			}
			SaveReq = ""
		} else if URLReq != "" {
			// TODO: extracted from client.getCodeBase() - no applet here
			resp, err := http.Get("http://127.0.0.1:" + strconv.Itoa(clientextras.PortOffset+8888) + "/" + URLReq)
			if err != nil {
				URLStream = nil
				goto End
			}
			defer resp.Body.Close()
			b, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("failed to read response body: %v\n", err)
				goto End
			}
			URLStream = b
		}
	End:
		time.Sleep(time.Duration(LoopRate) * time.Millisecond)
	}
}

func FindCacheDir() string {
	var0 := []string{"c:/windows/", "c:/winnt/", "d:/windows/", "d:/winnt/", "e:/windows/", "e:/winnt/", "f:/windows/", "f:/winnt/", "c:/", "~/", "/tmp/", ""}
	var1 := ".file_store_32"
	for i := range len(var0) {
		var3 := var0[i]
		if len(var3) > 0 {
			if _, err := os.Stat(var3); err != nil {
				fmt.Printf("couldn't find cache at %s: %v\n", var3, err)
				continue
			}
		}
		var4 := path.Join(var3, var1)
		_, err := os.Stat(var4)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				err2 := os.Mkdir(var4, 0755)
				if err2 != nil {
					fmt.Printf("couldn't create cache at %s: %v\n", var4, err2)
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
	LoadReq = strconv.FormatInt(GetHash(arg0), 10)
	for LoadReq != "" {
		time.Sleep(1 * time.Millisecond)
	}
	return LoadBuf
}

func CacheSave(arg0 string, arg1 []byte) {
	// TODO: synchronized
	if len(arg1) > 2000000 {
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

// TODO: OpenSocket

func OpenURL(arg0 string) ([]byte, error) {
	URLReq = arg0
	for URLReq != "" {
		time.Sleep(50 * time.Millisecond)
	}
	if URLStream == nil {
		return nil, errors.New("could not open: " + arg0)
	}
	return URLStream, nil
}

func DNSLookup(arg0 string) {
	DNS = arg0
	DNSReq = arg0
}

func WaveSave(arg0 []byte, arg1 int) bool {
	if arg1 > 2_000_000 {
		return false
	}
	if SaveReq == "" {
		WavePos = (WavePos + 1) % 5
		SaveLen = arg1
		SaveBuf = arg0
		WavePlay = true
		SaveReq = "sound" + strconv.Itoa(WavePos) + ".wav"
		return true
	}
	return false
}

func WaveReplay() bool {
	if SaveReq == "" {
		SaveBuf = nil
		WavePlay = true
		SaveReq = "sound" + strconv.Itoa(WavePos) + ".wav"
		return true
	}
	return false
}

func MidiSave(saveBuf []byte, saveLen int) {
	if saveLen > 2_000_000 || SaveReq != "" {
		return
	}
	MidiPos = (MidiPos + 1) % 5
	SaveLen = saveLen
	SaveBuf = saveBuf
	MidiPlay = true
	SaveReq = "jingle" + strconv.Itoa(MidiPos) + ".mid"
}

func ReportErrorFunc(e string) {
	if !ReportError {
		return
	}
	fmt.Println("error: " + e)
	var3 := strings.ReplaceAll(e, "@", "_")
	var4 := strings.ReplaceAll(var3, "&", "_")
	var5 := strings.ReplaceAll(var4, "#", "_")
	_, err := OpenURL("reporterror" + strconv.Itoa(225) + ".cgi?error=" + ErrorName + " " + var5)
	if err != nil {
		fmt.Printf("failed to open url: %v\n", err)
		return
	}
}
