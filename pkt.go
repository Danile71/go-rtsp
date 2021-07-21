package rtsp

// #include "ffmpeg.h"
import "C"

type Packet struct {
	streamIndex int
	codecType   int
	data        []byte
	width       int
	height      int
}

func (pkt *Packet) Height() int {
	return pkt.height
}

func (pkt *Packet) Width() int {
	return pkt.width
}

func (pkt *Packet) Data() []byte {
	return pkt.data
}

func (pkt *Packet) IsAudio() bool {
	return pkt.codecType == C.AVMEDIA_TYPE_AUDIO
}

func (pkt *Packet) IsVideo() bool {
	return pkt.codecType == C.AVMEDIA_TYPE_VIDEO
}
