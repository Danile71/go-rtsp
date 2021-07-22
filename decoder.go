package rtsp

/*
#cgo LDFLAGS: -lavformat -lavutil -lavcodec
#include "ffmpeg.h"
*/
import "C"
import (
	"fmt"
	"unsafe"
)

type decoder struct {
	index     int
	codecCtx  *C.AVCodecContext
	codec     *C.AVCodec
	codecType int
}

func (decoder *decoder) Decode(packet *C.AVPacket) (pkt *Packet, err error) {
	pkt = &Packet{}

	pkt.streamIndex = int(packet.stream_index)
	pkt.codecType = decoder.codecType

	switch decoder.codecType {
	case int(C.AVMEDIA_TYPE_AUDIO):
	case int(C.AVMEDIA_TYPE_VIDEO):
	default:
		// do nothing
		return
	}

	cerr := C.avcodec_send_packet(decoder.codecCtx, packet)
	if int(cerr) != 0 {
		err = fmt.Errorf("ffmpeg: avcodec_send_packet failed: %d", cerr)
		return
	}

	frame := C.av_frame_alloc()
	defer C.av_frame_free(&frame)

	cerr = C.avcodec_receive_frame(decoder.codecCtx, frame)
	if int(cerr) < 0 {
		err = fmt.Errorf("ffmpeg: avcodec_receive_frame failed: %d", cerr)
		return
	}

	switch decoder.codecType {
	case C.AVMEDIA_TYPE_VIDEO:
		pkt.width = int(frame.width)
		pkt.height = int(frame.height)

		var encPacket C.AVPacket
		defer C.av_packet_unref(&encPacket)

		switch frame.format {
		case C.AV_PIX_FMT_NONE, C.AV_PIX_FMT_YUVJ420P:
			if cerr = C.rtsp_avcodec_encode_jpeg(decoder.codecCtx, frame, &encPacket); cerr != C.int(0) {
				err = fmt.Errorf("ffmpeg: rtsp_avcodec_encode_jpeg failed: %d", cerr)
				return
			}

		default:
			if cerr = C.rtsp_avcodec_encode_jpeg_nv12(decoder.codecCtx, frame, &encPacket); cerr != C.int(0) {
				err = fmt.Errorf("ffmpeg: rtsp_avcodec_encode_jpeg_nv12 failed: %d", cerr)
				return
			}
		}

		pkt.data = make([]byte, int(encPacket.size))
		copy(pkt.data, *(*[]byte)(unsafe.Pointer(&encPacket.data)))
	case C.AVMEDIA_TYPE_AUDIO:
		var encPacket C.AVPacket
		defer C.av_packet_unref(&encPacket)

		if cerr = C.rtsp_avcodec_encode_wav(decoder.codecCtx, frame, &encPacket); cerr != C.int(0) {
			err = fmt.Errorf("ffmpeg: rtsp_avcodec_encode_wav failed: %d", cerr)
			return
		}

		pkt.data = make([]byte, int(encPacket.size))
		copy(pkt.data, *(*[]byte)(unsafe.Pointer(&encPacket.data)))
	default:
	}

	return
}
