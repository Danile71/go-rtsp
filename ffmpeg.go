package rtsp

/*
#cgo LDFLAGS: -lavformat -lavutil -lavcodec -lswresample -lswscale
#include "ffmpeg.h"
*/
import "C"

func init() {
	C.ffmpeginit()
}
