package audio

import "encoding/binary"

// parseWave8Mono validates a RIFF/WAV file emitted by sound/jagfx.GetWave
// (22050 Hz, 1 ch, 8-bit unsigned PCM) and returns the raw 8-bit unsigned
// mono sample bytes. ok is false if the header doesn't match exactly.
func parseWave8Mono(data []byte) (samples []byte, ok bool) {
	if len(data) < 44 {
		return nil, false
	}
	if string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return nil, false
	}
	if string(data[12:16]) != "fmt " {
		return nil, false
	}
	audioFormat := binary.LittleEndian.Uint16(data[20:22])
	channels := binary.LittleEndian.Uint16(data[22:24])
	sampleRate := binary.LittleEndian.Uint32(data[24:28])
	bitsPerSample := binary.LittleEndian.Uint16(data[34:36])
	if audioFormat != 1 || channels != 1 || sampleRate != SampleRate || bitsPerSample != 8 {
		return nil, false
	}
	if string(data[36:40]) != "data" {
		return nil, false
	}
	dataLen := int(binary.LittleEndian.Uint32(data[40:44]))
	if 44+dataLen > len(data) {
		dataLen = len(data) - 44
	}
	return data[44 : 44+dataLen], true
}
