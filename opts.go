package rtsp

/*
#include "ffmpeg.h"
*/
import "C"
import (
	"unsafe"
)

type StreamOption func(*Stream)

func WithTimeout(timeout string) StreamOption {
	return func(stream *Stream) {
		timeoutKey := C.CString("listen_timeout")
		defer C.free(unsafe.Pointer(timeoutKey))

		timeoutStr := C.CString(timeout)
		defer C.free(unsafe.Pointer(timeoutStr))

		C.av_dict_set(&stream.dictionary, timeoutKey, timeoutStr, 0)
	}
}

func WithType(streamType Type) StreamOption {
	return func(stream *Stream) {
		transport := C.CString("rtsp_transport")
		defer C.free(unsafe.Pointer(transport))

		switch streamType {
		case Tcp:
			tcp := C.CString("tcp")
			defer C.free(unsafe.Pointer(tcp))

			C.av_dict_set(&stream.dictionary, transport, tcp, 0)
		case Udp:
			udp := C.CString("udp")
			defer C.free(unsafe.Pointer(udp))

			C.av_dict_set(&stream.dictionary, transport, udp, 0)
		default:
		}
	}
}
