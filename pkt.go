package rtsp

// #include "ffmpeg.h"
import "C"

type Image struct {
	// only image
	width    int
	height   int
	keyFrame bool
}

// Height if image or 0 if audio
func (image *Image) Height() int {
	return image.height
}

// Width if image or 0 if audio
func (image *Image) Width() int {
	return image.width
}

type Audio struct {
	// only audio
	sampleRate int
	bitRate    int
	channels   int
}

// SampleRate if audio or 0 if image
func (audio *Audio) SampleRate() int {
	return audio.sampleRate
}

// BitRate if audio or 0 if image
func (audio *Audio) BitRate() int {
	return audio.bitRate
}

// Channels if audio or 0 if image
func (audio *Audio) Channels() int {
	return audio.channels
}

type Packet struct {
	streamIndex int
	codecType   int
	data        []byte

	duration int64
	position int64

	Image
	Audio
}

// Data encoded jpeg if image or wav if audio
func (packet *Packet) Data() []byte {
	return packet.data
}

// IsAudio packet
func (packet *Packet) IsAudio() bool {
	return packet.codecType == C.AVMEDIA_TYPE_AUDIO
}

// IsVideo packet
func (packet *Packet) IsVideo() bool {
	return packet.codecType == C.AVMEDIA_TYPE_VIDEO
}
