package rtsp

/*
#cgo LDFLAGS: -lavformat -lavutil -lavcodec -lswresample -lswscale
#include "ffmpeg.h"
void ffinit() {
	#if LIBAVCODEC_VERSION_INT < AV_VERSION_INT(58, 9, 100)
	av_register_all();
	avformat_network_init();
	#endif
}
*/
import "C"
import (
	"fmt"
	"runtime"
	"unsafe"
)

func init() {
	C.ffinit()
}

type Decoder interface {
	decode(packet *C.AVPacket, pkt *Packet) (err error)
}

type decoder struct {
	index     int
	codecCtx  *C.AVCodecContext
	codec     *C.AVCodec
	codecType int
}

type Stream struct {
	formatCtx *C.AVFormatContext
	decoders  []*decoder
}

func newStream() (streamDecoder *Stream) {
	streamDecoder = &Stream{}
	streamDecoder.formatCtx = C.avformat_alloc_context()

	runtime.SetFinalizer(streamDecoder, freeStream)
	return
}

func freeStream(streamDecoder *Stream) {
	if streamDecoder.formatCtx != nil {
		C.avformat_free_context(streamDecoder.formatCtx)
		streamDecoder.formatCtx = nil
	}

	for _, decoder := range streamDecoder.decoders {
		if decoder != nil {
			if decoder.codecCtx != nil {
				C.avcodec_close(decoder.codecCtx)
				C.av_free(unsafe.Pointer(decoder.codecCtx))
				decoder.codecCtx = nil
			}
			if decoder.codec != nil {
				C.av_free(unsafe.Pointer(decoder.codec))
				decoder.codec = nil
			}

			decoder = nil
		}
	}

}

func Open(url string) (streamDecoder *Stream, err error) {
	streamDecoder = newStream()

	uri := C.CString(url)
	defer C.free(unsafe.Pointer(uri))

	cerr := C.avformat_open_input(&streamDecoder.formatCtx, uri, nil, nil)
	if int(cerr) != 0 {
		err = fmt.Errorf("ffmpeg: avformat_open_input failed: %d", cerr)
		return
	}

	cerr = C.avformat_find_stream_info(streamDecoder.formatCtx, nil)
	if int(cerr) != 0 {
		err = fmt.Errorf("ffmpeg: avformat_find_stream_info failed: %d", cerr)
		return
	}

	for i := 0; i < int(streamDecoder.formatCtx.nb_streams); i++ {
		stream := C.stream_at((*C.struct_AVFormatContext)(streamDecoder.formatCtx), C.int(i))

		switch stream.codecpar.codec_type {
		case C.AVMEDIA_TYPE_VIDEO, C.AVMEDIA_TYPE_AUDIO:
			decoder := &decoder{index: i}
			streamDecoder.decoders = append(streamDecoder.decoders, decoder)
			decoder.codecCtx = C.avcodec_alloc_context3(nil)
			C.avcodec_parameters_to_context(decoder.codecCtx, stream.codecpar)
			decoder.codec = C.avcodec_find_decoder(decoder.codecCtx.codec_id)
			decoder.codecType = int(stream.codecpar.codec_type)
			if decoder.codec == nil {
				err = fmt.Errorf("ffmpeg: avcodec_find_decoder failed: codec %d not found", decoder.codecCtx.codec_id)
				return
			}

			cerr = C.avcodec_open2(decoder.codecCtx, decoder.codec, nil)
			if int(cerr) != 0 {
				err = fmt.Errorf("ffmpeg: avcodec_open2 failed: %d", cerr)
				return
			}
		default:
			err = fmt.Errorf("ffmpeg: failed: codec_type %d not found", stream.codecpar.codec_type)
			return
		}

	}

	return
}

func (decoder *decoder) decode(packet *C.AVPacket, pkt *Packet) (err error) {
	pkt.codecType = decoder.codecType

	// now skip audio
	if decoder.codecType == C.AVMEDIA_TYPE_AUDIO {
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

		encPacket := C.AVPacket{}
		defer C.av_packet_unref(&encPacket)

		switch frame.format {
		case C.AV_PIX_FMT_NONE, C.AV_PIX_FMT_YUVJ420P:
			cerr = C.avcodec_encode_jpeg(decoder.codecCtx, frame, &encPacket)
			if cerr != C.int(0) {
				err = fmt.Errorf("ffmpeg: avcodec_encode_jpeg failed: %d", cerr)
				return
			}

			pkt.data = make([]byte, int(packet.size))
			copy(pkt.data, *(*[]byte)(unsafe.Pointer(&encPacket.data)))
		default:
			nframe := C.av_frame_alloc()
			defer C.av_frame_free(&nframe)

			cerr := C.avcodec_encode_jpeg_nv12(decoder.codecCtx, frame, nframe, &encPacket)
			if cerr != C.int(0) {
				err = fmt.Errorf("ffmpeg: avcodec_encode_jpeg_nv12 failed: %d", cerr)
				return
			}

			pkt.data = make([]byte, int(encPacket.size))
			copy(pkt.data, *(*[]byte)(unsafe.Pointer(&encPacket.data)))
		}
	default:
	}

	return
}

func (streamDecoder *Stream) ReadPacket() (pkt *Packet, err error) {
	pkt = &Packet{}
	var packet C.AVPacket
	C.av_init_packet(&packet)

	defer C.av_packet_unref(&packet)

	cerr := C.av_read_frame(streamDecoder.formatCtx, &packet)
	if int(cerr) != 0 {
		err = fmt.Errorf("ffmpeg: av_read_frame failed: %d", cerr)
		return
	}

	pkt.streamIndex = int(packet.stream_index)

	for _, d := range streamDecoder.decoders {
		if d.index == int(packet.stream_index) {
			err = d.decode(&packet, pkt)
			return
		}
	}
	err = fmt.Errorf("ffmpeg: decoder not found")
	return
}
