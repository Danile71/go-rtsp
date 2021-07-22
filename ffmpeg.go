package rtsp

/*
#cgo LDFLAGS: -lavformat -lavutil -lavcodec -lswresample -lswscale -lm
#include "ffmpeg.h"
*/
import "C"

// Init old version's ffmpeg
func init() {
	C.ffmpeginit()
}
