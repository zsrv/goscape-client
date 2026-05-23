package audio

import (
	"bytes"
	"errors"
	"fmt"
	"log"

	"github.com/sinshu/go-meltysynth/meltysynth"

	"goscape-client/pkg/sign/signlink"
)

// soundFontCacheName is the filename used inside signlink's cache
// directory for the bundled SoundFont. The hashed cache filename is
// derived inside signlink.CacheSave/CacheLoad; this string just gives
// it a stable identity so subsequent runs reuse the same cached blob.
const soundFontCacheName = "soundfont.sf2"

// soundFontURL is the relative URL the SoundFont is fetched from on
// first run. The 2004Scape server hosts SCC1_Florestan.sf2 at this
// path (server2 layout: public/SCC1_Florestan.sf2 → /SCC1_Florestan.sf2).
// signlink.OpenURL prepends "http://127.0.0.1:<port>/".
const soundFontURL = "SCC1_Florestan.sf2"

// loadSoundFont returns a parsed SoundFont, reusing the signlink cache
// directory if present and otherwise downloading from the server and
// caching for next time. Mirrors the cache-then-fetch pattern in
// client.RunMidi (client.go:1566-1591) so the user only pays the
// download cost once.
func loadSoundFont() (*meltysynth.SoundFont, error) {
	if buf := signlink.CacheLoad(soundFontCacheName); len(buf) > 0 {
		if sf, err := meltysynth.NewSoundFont(bytes.NewReader(buf)); err == nil {
			return sf, nil
		} else {
			log.Printf("audio: cached soundfont was corrupt, refetching: %v", err)
		}
	}

	buf, err := signlink.OpenURL(soundFontURL)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", soundFontURL, err)
	}
	if len(buf) == 0 {
		return nil, errors.New("soundfont download returned empty body")
	}

	sf, err := meltysynth.NewSoundFont(bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("parse soundfont: %w", err)
	}

	signlink.CacheSave(soundFontCacheName, buf)
	return sf, nil
}
