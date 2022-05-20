// +build linux
package rtsp

/*
#cgo LDFLAGS: -lavformat -lavutil -lavcodec -lswresample -lswscale -lm
#include "ffmpeg.h"
*/
import "C"
import (
	"unsafe"
)

func CErr2Str(code C.int) string {
	buf := make([]byte, 64)

	C.av_strerror(code, (*C.char)(unsafe.Pointer(&buf[0])), C.ulong(len(buf)))

	return string(buf)
}

func swrAllocSetOpts(layout uint64, sampleRate C.int, sampleFmt int32) *C.SwrContext {
	swrContext := C.swr_alloc_set_opts(nil, // we're allocating a new context
		C.long(layout),      // out_ch_layout
		C.AV_SAMPLE_FMT_S16, // out_sample_fmt
		sampleRate,          // out_sample_rate

		C.long(layout), // in_ch_layout
		sampleFmt,      // in_sample_fmt
		sampleRate,     // in_sample_rate

		0,   // log_offset
		nil) // log_ctx
	return swrContext
}
