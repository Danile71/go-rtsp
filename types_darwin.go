//go:build darwin

package rtsp

/*
#cgo LDFLAGS: -lavformat -lavutil -lavcodec -lswresample -lswscale -lm
#include "ffmpeg.h"
*/
import "C"
import (
	"unsafe"
)

// CErr2Str convert C error code to Go string
func CErr2Str(code C.int) string {
	buf := make([]byte, 64)

	C.av_strerror(code, (*C.char)(unsafe.Pointer(&buf[0])), C.ulonglong(len(buf)))

	return string(buf)
}

func swrAllocSetOpts(layout uint64, sampleRate C.int, sampleFmt int32) *C.SwrContext {
	swrContext := C.swr_alloc_set_opts(nil, // we're allocating a new context
		C.longlong(layout),  // out_ch_layout
		C.AV_SAMPLE_FMT_S16, // out_sample_fmt
		sampleRate,          // out_sample_rate

		C.longlong(layout), // in_ch_layout
		sampleFmt,          // in_sample_fmt
		sampleRate,         // in_sample_rate

		0,   // log_offset
		nil) // log_ctx
	return swrContext
}
