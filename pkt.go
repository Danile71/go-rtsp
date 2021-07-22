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
}

// Height if image or 0 if audio
func (pkt *Packet) Height() int {
	return pkt.height
}

// Width if image or 0 if audio
func (pkt *Packet) Width() int {
	return pkt.width
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
