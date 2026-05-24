// Package signlink is used for signed Java applets to be able to use the filesystem etc.
// Not used in unsigned mode.
package signlink

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zsrv/goscape-client/pkg/jagex2/client/clientextras"
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

// StartPriv clears the request slots and enters the polling loop.
//
// Java: startpriv (sign/signlink.java:80-105) seeds threadliveid and then spins
// `while (!active) Thread.sleep(50)` so no caller can submit a request before
// run() finishes initializing. The `active` lifecycle flag, the threadliveid
// stale-instance re-entrancy guard, and the !active early-return-null paths in
// cacheload/cachesave/reporterror are INTENTIONALLY NOT PORTED: they exist to
// coordinate restarts of the signed applet's privileged thread, which has no
// analog in this single-shot standalone process (the loop never terminates and
// is never restarted — see cmd/client/main.go). The mu/cond handoff already
// orders submissions safely; a request issued before Run's loop spins up simply
// waits on cond and is serviced late (benign) rather than returning null —
// which here would be worse, reading as a spurious cache miss.
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
	var1 := store.cacheDir()
	uid := store.uid()
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
			// Java: sign/signlink.java:127-131 —
			//   try { dns = InetAddress.getByName(dnsreq).getHostName(); }
			//   catch (Exception) { dns = "unknown"; }
			// getByName(x).getHostName() has two distinct paths:
			//   - x is an IP literal: getByName parses it (never throws) and
			//     getHostName does a REVERSE lookup, returning the PTR name or,
			//     on failure, the IP text itself.
			//   - x is a hostname: getByName FORWARD-resolves it (throwing
			//     UnknownHostException -> "unknown" if unresolvable) and
			//     getHostName returns the original (cached) host string.
			// The sole caller passes jstring.FormatIPv4(...) (an IP literal), so
			// the IP branch is taken in practice; the hostname branch restores
			// the "unknown" sentinel for parity completeness.
			var resolved string
			if net.ParseIP(dnsReq) != nil {
				names, err := net.LookupAddr(dnsReq)
				if err == nil && len(names) > 0 {
					resolved = strings.TrimSuffix(names[0], ".")
				} else {
					resolved = dnsReq
				}
			} else if _, err := net.LookupHost(dnsReq); err != nil {
				resolved = "unknown" // Java: catch -> UnknownHostException path
			} else {
				resolved = dnsReq // getHostName returns the cached input host
			}
			mu.Lock()
			DNS = resolved
			DNSReq = ""
			cond.Broadcast()
			mu.Unlock()
		case loadReq != "":
			buf := store.load(loadReq)
			mu.Lock()
			LoadBuf = buf
			LoadReq = ""
			cond.Broadcast()
			mu.Unlock()
		case saveReq != "":
			if saveBuf != nil {
				store.save(saveReq, saveBuf[0:saveLen])
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
			// Java: signlink.openurl dispatches to applet.getCodeBase() (signlink.java).
			// Go is always standalone, so we construct the URL inline against the
			// configured port offset.
			resp, err := http.Get("http://127.0.0.1:" + strconv.Itoa(clientextras.PortOffset+8888) + "/" + urlReq)
			var body []byte
			if err == nil {
				// Java: URL.openStream() throws on non-2xx; the catch block
				// sets urlstream = null. http.Get does NOT error on 4xx/5xx
				// (it returns a response with the error body), so we must
				// reject non-2xx explicitly to match Java.
				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					b, readErr := io.ReadAll(resp.Body)
					if readErr != nil {
						log.Printf("signlink: failed to read response body: %v", readErr)
					} else {
						body = b
					}
				} else {
					log.Printf("signlink: openurl %s: HTTP %d", urlReq, resp.StatusCode)
				}
				resp.Body.Close()
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

// GetHash is Java's signlink.gethash: a base-37 hash of the first 12 chars
// (case-insensitive). It was the file-store key derivation, but CacheLoad/
// CacheSave now key by the plain name (see the DEVIATION note there), so this
// is no longer used for caching. Retained as a faithful port of the Java
// algorithm; remove if a future cleanup wants it gone.
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
	// DEVIATION from Java: the original signlink hashes the resource name via
	// GetHash and keys the file store by the decimal hash (Client-Java does the
	// same). We use the plain name as the key instead — readable cache keys
	// matching the Client-TS browser client, easy to inspect in IndexedDB and
	// on disk. Java: cacheload set loadreq = String.valueOf(gethash(name)).
	LoadReq = arg0
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
	// DEVIATION from Java: plain name as the key, not GetHash. See CacheLoad.
	SaveReq = arg0
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
// Transport branch (Go-original extension): ws:// or wss:// hosts dial a
// WebSocket instead of a raw TCP socket, enabling a future js/wasm browser
// build. TCP remains the Java-parity default. See
// docs/superpowers/specs/2026-05-24-websocket-transport-design.md.
func OpenSocket(port int) (net.Conn, error) {
	const dialTimeout = 10 * time.Second
	switch clientextras.Transport {
	case clientextras.TransportWS, clientextras.TransportWSS:
		return openWebSocket(clientextras.Transport, clientextras.Host, port, dialTimeout)
	default:
		return dialTCP(clientextras.Host, port, dialTimeout)
	}
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

// ConsumeMidi atomically reads and clears the Midi path/command. Returns
// the empty string when nothing is queued. The audio playback driver
// polls this; Java's signlink consumer (in the signed-applet wrapper)
// read `midi` and reset it to `null` after acting, which this mirrors.
//
// Recognised values: a filesystem path to a .mid file (play it), the
// sentinel "stop" (stop current track honoring MidiFade), or "voladjust"
// (re-read MidiVol without restarting the track).
func ConsumeMidi() string {
	mu.Lock()
	defer mu.Unlock()
	s := Midi
	Midi = ""
	return s
}

// ConsumeWave is the SFX counterpart of ConsumeMidi. Returns the path to
// the most recently saved .wav, or "" if nothing is queued.
func ConsumeWave() string {
	mu.Lock()
	defer mu.Unlock()
	s := Wave
	Wave = ""
	return s
}

// SetMidiCommand publishes a command/path to the audio driver. Used for
// the "stop" and "voladjust" sentinels written from the game thread; the
// I/O polling loop publishes resolved paths directly under mu.
func SetMidiCommand(s string) {
	mu.Lock()
	defer mu.Unlock()
	Midi = s
}

// SetMidiFade publishes the fade flag (0 = immediate, 1 = crossfade) that
// the audio driver reads when honoring the next "stop" or track change.
func SetMidiFade(v int) {
	mu.Lock()
	defer mu.Unlock()
	MidiFade = v
}

// SetMidiVol publishes the music volume in centibels (negative = quieter,
// 0 = full). The audio driver applies it on the next "voladjust" or
// track change.
func SetMidiVol(v int) {
	mu.Lock()
	defer mu.Unlock()
	MidiVol = v
}

// ReadMidiFade snapshots MidiFade for the audio driver.
func ReadMidiFade() int {
	mu.Lock()
	defer mu.Unlock()
	return MidiFade
}

// ReadMidiVol snapshots MidiVol for the audio driver.
func ReadMidiVol() int {
	mu.Lock()
	defer mu.Unlock()
	return MidiVol
}

// SetWaveVol publishes the SFX volume in centibels (negative =
// quieter, 0 = full). The audio driver reads it when spawning a
// per-SFX Player so the slider in the in-game audio options affects
// freshly-triggered sound effects.
func SetWaveVol(v int) {
	mu.Lock()
	defer mu.Unlock()
	WaveVol = v
}

// ReadWaveVol snapshots WaveVol for the audio driver. Race-free
// counterpart to the bare `signlink.WaveVol` field reads that used
// to be in client code.
func ReadWaveVol() int {
	mu.Lock()
	defer mu.Unlock()
	return WaveVol
}

func ReportErrorFunc(e string) {
	if !ReportError {
		return
	}
	fmt.Println("error: " + e)
	var3 := strings.ReplaceAll(e, "@", "_")
	var4 := strings.ReplaceAll(var3, "&", "_")
	var5 := strings.ReplaceAll(var4, "#", "_")
	// Java: sign/signlink.java:366-368 explicitly does
	//   DataInputStream var1 = openurl(...);
	//   var1.readLine();
	//   var1.close();
	// Go's OpenURL reads the full response body (and closes the
	// connection) before returning, so the readLine + close pair
	// is subsumed by the call itself. Discarding the body is the
	// equivalent of Java's readLine-then-close pattern; the HTTP
	// transaction is observably identical.
	_, err := OpenURL("reporterror" + strconv.Itoa(225) + ".cgi?error=" + ErrorName + " " + var5)
	if err != nil {
		log.Printf("signlink: failed to open url: %v", err)
		return
	}
}
