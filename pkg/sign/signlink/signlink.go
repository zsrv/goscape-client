// Package signlink is used for signed Java applets to be able to use the filesystem etc.
// Not used in unsigned mode.
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
	"sync"
	"time"

	"goscape-client/pkg/jagex2/client/clientextras"
)

// mu serializes the signlink polling protocol's request setup, mirroring
// the `synchronized` modifier on the Java methods. The polling loop
// (Run) dispatches one request slot per cycle (DNSReq, LoadReq, SaveReq,
// URLReq); without this mutex, two goroutines calling CacheLoad
// concurrently overwrite each other's LoadReq and both readers receive
// the bytes of whichever file the polling loop happened to process.
// Symptom: RunMidi receives the `config` jagfile when it asks for
// `scape_main.mid`.
// Java: signlink (sign/signlink.java) — all protocol methods are
// `static synchronized`, sharing the class monitor.
//
// mu also acts as the memory barrier for the protocol fields below
// (LoadReq/LoadBuf, SaveReq/SaveBuf/SaveLen, URLReq/URLStream,
// DNSReq/DNS, and the Wave/Midi flag fields). The polling goroutine
// (Run) and request-submitting goroutines both take this lock when
// reading or writing those fields. Long I/O operations inside Run
// release the lock and reacquire it before publishing the result.
var mu sync.Mutex

// cond signals state transitions on the protocol fields protected by mu.
// Run.Broadcast()s after clearing a request (LoadReq, SaveReq, URLReq,
// DNSReq); the corresponding caller (CacheLoad, CacheSave, OpenURL)
// Wait()s on cond instead of spin-sleeping. This replaces the Java
// `while (loadreq != null) Thread.sleep(1L)` busy-wait
// (signlink.java:249-254, 262-267, 271-276, 295-300) with a proper
// condition-variable handoff that carries a memory barrier.
var cond = sync.NewCond(&mu)

// slotMu serializes callers contending for the same single-slot request
// fields (LoadReq, SaveReq, URLReq). cond.Wait() drops mu while the
// polling goroutine works, so without an outer serializing mutex a
// second caller could steal the slot after Run clears it but before the
// original caller wakes, causing the wrong LoadBuf/URLStream to be
// returned. Java doesn't need this because its `synchronized` methods
// retain the monitor across `Thread.sleep()`; Go's sync.Cond does not.
var slotMu sync.Mutex

var (
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
	mu.Lock()
	DNSReq = ""
	LoadReq = ""
	SaveReq = ""
	URLReq = ""
	mu.Unlock()
	Run()
}

// Run is the polling goroutine. It owns the I/O side of every request
// slot: it reads requests under mu, performs the I/O without holding mu
// (so submissions are not blocked on slow disk/network), then reacquires
// mu to publish results and clear the request, broadcasting cond so any
// goroutine waiting in CacheLoad/CacheSave/OpenURL wakes up.
//
// Java: signlink.run() (sign/signlink.java:107-178).
func Run() {
	var1 := FindCacheDir()
	uid := GetUID(var1)
	mu.Lock()
	UID = uid
	mu.Unlock()

	for {
		mu.Lock()
		dnsReq := DNSReq
		loadReq := LoadReq
		saveReq := SaveReq
		saveBuf := SaveBuf
		saveLen := SaveLen
		wavePlay := WavePlay
		midiPlay := MidiPlay
		urlReq := URLReq
		loopRate := LoopRate
		mu.Unlock()

		switch {
		case dnsReq != "":
			names, err := net.LookupAddr(dnsReq)
			var resolved string
			if err != nil || len(names) == 0 {
				resolved = "unknown"
			} else {
				resolved = names[0]
			}
			mu.Lock()
			DNS = resolved
			DNSReq = ""
			cond.Broadcast()
			mu.Unlock()
		case loadReq != "":
			var buf []byte
			p := path.Join(var1, loadReq)
			if _, err := os.Stat(p); err == nil {
				b, err := os.ReadFile(p)
				if err != nil {
					fmt.Printf("failed to read file %s: %v\n", p, err)
				} else {
					buf = b
				}
			}
			mu.Lock()
			LoadBuf = buf
			LoadReq = ""
			cond.Broadcast()
			mu.Unlock()
		case saveReq != "":
			if saveBuf != nil {
				if err := os.WriteFile(path.Join(var1, saveReq), saveBuf[0:saveLen], 0644); err != nil {
					fmt.Printf("failed to write file %s: %v\n", path.Join(var1, saveReq), err)
				}
			}
			waveOut := ""
			midiOut := ""
			if wavePlay {
				waveOut = path.Join(var1, saveReq)
			}
			if midiPlay {
				midiOut = path.Join(var1, saveReq)
			}
			mu.Lock()
			if wavePlay {
				Wave = waveOut
				WavePlay = false
			}
			if midiPlay {
				Midi = midiOut
				MidiPlay = false
			}
			SaveReq = ""
			cond.Broadcast()
			mu.Unlock()
		case urlReq != "":
			// TODO: extracted from client.getCodeBase() - no applet here
			resp, err := http.Get("http://127.0.0.1:" + strconv.Itoa(clientextras.PortOffset+8888) + "/" + urlReq)
			var body []byte
			if err == nil {
				b, readErr := io.ReadAll(resp.Body)
				resp.Body.Close()
				if readErr != nil {
					fmt.Printf("failed to read response body: %v\n", readErr)
				} else {
					body = b
				}
			}
			mu.Lock()
			URLStream = body
			URLReq = ""
			cond.Broadcast()
			mu.Unlock()
		}

		time.Sleep(time.Duration(loopRate) * time.Millisecond)
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

// CacheLoad submits a LoadReq to the polling goroutine and blocks until
// Run clears it. The mu/cond pair replaces the Java
// `while (loadreq != null) Thread.sleep(1L)` busy-wait
// (signlink.java:249-254) with a condition-variable handoff that also
// supplies the memory barrier the spin loop lacked.
//
// slotMu serializes callers so that exactly one CacheLoad is in flight
// at a time. Without it, cond.Wait() would drop mu and let a second
// caller steal the LoadReq slot before the first observed LoadBuf,
// returning the wrong file's bytes.
func CacheLoad(arg0 string) []byte {
	slotMu.Lock()
	defer slotMu.Unlock()
	mu.Lock()
	defer mu.Unlock()
	LoadReq = strconv.FormatInt(GetHash(arg0), 10)
	for LoadReq != "" {
		cond.Wait()
	}
	return LoadBuf
}

// CacheSave is the dual of CacheLoad on the SaveReq slot.
// Java: signlink.java:258-277.
func CacheSave(arg0 string, arg1 []byte) {
	slotMu.Lock()
	defer slotMu.Unlock()
	mu.Lock()
	defer mu.Unlock()
	if len(arg1) > 2000000 {
		return
	}
	for SaveReq != "" {
		cond.Wait()
	}
	SaveLen = len(arg1)
	SaveBuf = arg1
	SaveReq = strconv.FormatInt(GetHash(arg0), 10)
	for SaveReq != "" {
		cond.Wait()
	}
}

// OpenSocket dials clientextras.Host on the given port and returns the
// connected net.Conn.
//
// Java: opensocket (sign/signlink.java:279-291). The Java version sets a
// socketreq field and busy-waits while a privileged polling thread performs
// the dial — required because the signed applet's network stack is gated by
// AccessController. Go has no such sandbox; we dial directly. The Java
// caller's IOException maps onto Go's returned error.
//
// Deviation: a 10s connect timeout is applied (Java has none) so a stuck DNS
// or unreachable host doesn't hang the caller indefinitely.
func OpenSocket(port int) (net.Conn, error) {
	const dialTimeout = 10 * time.Second
	return net.DialTimeout("tcp", net.JoinHostPort(clientextras.Host, strconv.Itoa(port)), dialTimeout)
}

// OpenURL submits a URLReq to the polling goroutine and waits until Run
// clears it. See CacheLoad for the mu/cond + slotMu pattern.
// Java: signlink.java:293-305.
func OpenURL(arg0 string) ([]byte, error) {
	slotMu.Lock()
	defer slotMu.Unlock()
	mu.Lock()
	URLReq = arg0
	for URLReq != "" {
		cond.Wait()
	}
	stream := URLStream
	mu.Unlock()
	if stream == nil {
		return nil, errors.New("could not open: " + arg0)
	}
	return stream, nil
}

func DNSLookup(arg0 string) {
	mu.Lock()
	defer mu.Unlock()
	DNS = arg0
	DNSReq = arg0
}

func WaveSave(arg0 []byte, arg1 int) bool {
	mu.Lock()
	defer mu.Unlock()
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
	mu.Lock()
	defer mu.Unlock()
	if SaveReq == "" {
		SaveBuf = nil
		WavePlay = true
		SaveReq = "sound" + strconv.Itoa(WavePos) + ".wav"
		return true
	}
	return false
}

func MidiSave(saveBuf []byte, saveLen int) {
	mu.Lock()
	defer mu.Unlock()
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
