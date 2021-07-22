package rtsp

// #include "ffmpeg.h"
import "C"

type Packet struct {
	streamIndex int
	codecType   int
	data        []byte

	// only image
	width  int
	height int

	// only audio
	sampleRate int
	bitRate    int
	channels   int
}

// Height if image or 0 if audio
func (pkt *Packet) Height() int {
	return pkt.height
}

// Width if image or 0 if audio
func (pkt *Packet) Width() int {
	return pkt.width
}

// SampleRate if audio or 0 if image
func (pkt *Packet) SampleRate() int {
	return pkt.sampleRate
}

// BitRate if audio or 0 if image
func (pkt *Packet) BitRate() int {
	return pkt.bitRate
}

// BitRate if audio or 0 if image
func (pkt *Packet) Channels() int {
	return pkt.channels
}

// Data encoded jpeg if image or wav if audio
func (pkt *Packet) Data() []byte {
	return pkt.data
}

// IsAudio packet
func (pkt *Packet) IsAudio() bool {
	return pkt.codecType == C.AVMEDIA_TYPE_AUDIO
}

// IsVideo packet
func (pkt *Packet) IsVideo() bool {
	return pkt.codecType == C.AVMEDIA_TYPE_VIDEO
}
