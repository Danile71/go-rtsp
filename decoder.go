package rtsp

/*
#cgo LDFLAGS: -lavformat -lavutil -lavcodec
#include "ffmpeg.h"
*/
import "C"
import (
	"fmt"
	"runtime"
	"unsafe"
)

type decoder struct {
	index      int
	codecCtx   *C.AVCodecContext
	codec      *C.AVCodec
	swrContext *C.SwrContext
	codecType  int
}

func decodeAVPacket(packet *C.AVPacket) (data []byte) {
	data = make([]byte, int(packet.size))
	copy(data, *(*[]byte)(unsafe.Pointer(&packet.data)))
	return
}

func newDecoder(cstream *C.AVStream) (*decoder, error) {
	decoder := &decoder{index: int(cstream.index)}
	runtime.SetFinalizer(decoder, freeDecoder)

	decoder.swrContext = nil
	decoder.codecCtx = C.avcodec_alloc_context3(nil)
	C.avcodec_parameters_to_context(decoder.codecCtx, cstream.codecpar)
	decoder.codec = C.avcodec_find_decoder(decoder.codecCtx.codec_id)
	decoder.codecType = int(cstream.codecpar.codec_type)
	if decoder.codec == nil {
		return nil, fmt.Errorf("ffmpeg: avcodec_find_decoder failed: codec %d not found", decoder.codecCtx.codec_id)
	}

	if cerr := C.avcodec_open2(decoder.codecCtx, decoder.codec, nil); int(cerr) != 0 {
		return nil, fmt.Errorf("ffmpeg: avcodec_open2 failed: %s", CErr2Str(cerr))
	}
	return decoder, nil
}

func freeDecoder(decoder *decoder) {
	if decoder.codecCtx != nil {
		C.avcodec_close(decoder.codecCtx)
		C.av_free(unsafe.Pointer(decoder.codecCtx))
		decoder.codecCtx = nil
	}
	if decoder.codec != nil {
		decoder.codec = nil
	}
	if decoder.swrContext != nil {
		C.swr_close(decoder.swrContext)
		C.swr_free(&decoder.swrContext)
	}
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
		err = fmt.Errorf("ffmpeg: avcodec_send_packet failed: %s", CErr2Str(cerr))
		return
	}

	frame := C.av_frame_alloc()
	defer C.av_frame_free(&frame)

	cerr = C.avcodec_receive_frame(decoder.codecCtx, frame)
	if int(cerr) < 0 {
		err = fmt.Errorf("ffmpeg: avcodec_receive_frame failed: %s", CErr2Str(cerr))
		return
	}

	pkt.duration = int64(frame.pkt_duration)
	pkt.position = int64(frame.pkt_pos)

	switch decoder.codecType {
	case C.AVMEDIA_TYPE_VIDEO:
		pkt.width = int(frame.width)
		pkt.height = int(frame.height)
		pkt.keyFrame = int(frame.key_frame) == 1

		var encPacket C.AVPacket
		defer C.av_packet_unref(&encPacket)

		switch frame.format {
		case C.AV_PIX_FMT_NONE, C.AV_PIX_FMT_YUVJ420P:
			if cerr = C.rtsp_avcodec_encode_jpeg(decoder.codecCtx, frame, &encPacket); cerr != C.int(0) {
				err = fmt.Errorf("ffmpeg: rtsp_avcodec_encode_jpeg failed: %s", CErr2Str(cerr))
				return
			}

		default:
			if cerr = C.rtsp_avcodec_encode_jpeg_nv12(decoder.codecCtx, frame, &encPacket); cerr != C.int(0) {
				err = fmt.Errorf("ffmpeg: rtsp_avcodec_encode_jpeg_nv12 failed: %s", CErr2Str(cerr))
				return
			}
		}

		pkt.data = decodeAVPacket(&encPacket)

	case C.AVMEDIA_TYPE_AUDIO:
		pkt.bitRate = int(decoder.codecCtx.bit_rate)
		pkt.sampleRate = int(frame.sample_rate)
		pkt.channels = int(frame.channels)

		switch frame.format {
		case C.AV_SAMPLE_FMT_FLTP:
			if decoder.swrContext == nil {
				layout := uint64(frame.channel_layout)

				decoder.swrContext = C.swr_alloc_set_opts(nil, // we're allocating a new context
					C.longlong(layout),  // out_ch_layout
					C.AV_SAMPLE_FMT_S16, // out_sample_fmt
					frame.sample_rate,   // out_sample_rate

					C.longlong(layout),          // in_ch_layout
					decoder.codecCtx.sample_fmt, // in_sample_fmt
					frame.sample_rate,           // in_sample_rate

					0,   // log_offset
					nil) // log_ctx

				if cerr = C.swr_init(decoder.swrContext); cerr < C.int(0) {
					decoder.swrContext = nil
					err = fmt.Errorf("ffmpeg: swr_init failed: %s", CErr2Str(cerr))
					return
				}
			}

			var encPacket C.AVPacket
			defer C.av_packet_unref(&encPacket)

			if cerr = C.rtsp_avcodec_encode_resample_wav(decoder.codecCtx, decoder.swrContext, frame, &encPacket); cerr < C.int(0) {
				err = fmt.Errorf("ffmpeg: rtsp_avcodec_encode_resample_wav failed: %s", CErr2Str(cerr))
				return
			}
			pkt.data = decodeAVPacket(&encPacket)

		case C.AV_SAMPLE_FMT_S16:
			var encPacket C.AVPacket
			defer C.av_packet_unref(&encPacket)

			if cerr = C.rtsp_avcodec_encode_wav(decoder.codecCtx, frame, &encPacket); cerr != C.int(0) {
				err = fmt.Errorf("ffmpeg: rtsp_avcodec_encode_wav failed: %s", CErr2Str(cerr))
				return
			}
			pkt.data = decodeAVPacket(&encPacket)

		case C.AV_SAMPLE_FMT_S32:
			if decoder.swrContext == nil {
				layout := uint64(frame.channel_layout)

				decoder.swrContext = C.swr_alloc_set_opts(nil, // we're allocating a new context
					C.longlong(layout),  // out_ch_layout
					C.AV_SAMPLE_FMT_S16, // out_sample_fmt
					frame.sample_rate,   // out_sample_rate

					C.longlong(layout),          // in_ch_layout
					decoder.codecCtx.sample_fmt, // in_sample_fmt
					frame.sample_rate,           // in_sample_rate

					0,   // log_offset
					nil) // log_ctx

				if cerr = C.swr_init(decoder.swrContext); cerr < C.int(0) {
					decoder.swrContext = nil
					err = fmt.Errorf("ffmpeg: swr_init failed: %s", CErr2Str(cerr))
					return
				}
			}

			var encPacket C.AVPacket
			defer C.av_packet_unref(&encPacket)

			if cerr = C.rtsp_avcodec_encode_resample_wav(decoder.codecCtx, decoder.swrContext, frame, &encPacket); cerr < C.int(0) {
				err = fmt.Errorf("ffmpeg: rtsp_avcodec_encode_resample_wav failed: %s", CErr2Str(cerr))
				return
			}
			pkt.data = decodeAVPacket(&encPacket)

		default:
			err = fmt.Errorf("ffmpeg: audio format %d not supported", frame.format)
			return
		}

	default:
	}

	return
}
