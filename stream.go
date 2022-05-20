package rtsp

/*
#include "ffmpeg.h"
*/
import "C"
import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"unsafe"
)

type Type int

const (
	Auto Type = iota
	Tcp
	Udp
)

type Stream struct {
	formatCtx  *C.AVFormatContext
	dictionary *C.AVDictionary

	decoders map[int]*decoder
	mu       sync.RWMutex
	uri      string
}

// New stream
func New(uri string) (stream *Stream) {
	stream = &Stream{uri: uri}
	stream.decoders = make(map[int]*decoder)
	stream.formatCtx = C.avformat_alloc_context()

	C.av_log_set_level(C.AV_LOG_QUIET)

	runtime.SetFinalizer(stream, free)
	return
}

func free(stream *Stream) {
	if stream.formatCtx != nil {
		C.avformat_close_input(&stream.formatCtx)
		C.avformat_free_context(stream.formatCtx)
		stream.formatCtx = nil
	}

	if stream.dictionary != nil {
		C.av_dict_free(&stream.dictionary)
		stream.dictionary = nil
	}
}

type ErrTimeout struct {
	err error
}

func (e ErrTimeout) Error() string {
	return e.err.Error()
}

// Setup transport (tcp or udp)
func (stream *Stream) Setup(t Type) (err error) {
	transport := C.CString("rtsp_transport")
	defer C.free(unsafe.Pointer(transport))

	tcp := C.CString("tcp")
	defer C.free(unsafe.Pointer(tcp))

	udp := C.CString("udp")
	defer C.free(unsafe.Pointer(udp))

	timeoutKey := C.CString("timeout")
	defer C.free(unsafe.Pointer(timeoutKey))

	goTimeout := os.Getenv("FFMPEG_TIMEOUT")
	if goTimeout == "" {
		goTimeout = "10000000"
	}

	timeout := C.CString(goTimeout)
	defer C.free(unsafe.Pointer(timeout))

	C.av_dict_set(&stream.dictionary, timeoutKey, timeout, 0)

	switch t {
	case Tcp:
		C.av_dict_set(&stream.dictionary, transport, tcp, 0)
	case Udp:
		C.av_dict_set(&stream.dictionary, transport, udp, 0)
	default:
	}

	uri := C.CString(stream.uri)
	defer C.free(unsafe.Pointer(uri))

	cerr := C.avformat_open_input(&stream.formatCtx, uri, nil, &stream.dictionary)
	if int(cerr) != 0 {
		err = fmt.Errorf("ffmpeg: avformat_open_input failed: %v", CErr2Str(cerr))
		return
	}

	cerr = C.avformat_find_stream_info(stream.formatCtx, nil)
	if int(cerr) != 0 {
		err = fmt.Errorf("ffmpeg: avformat_find_stream_info failed: %s", CErr2Str(cerr))
		return
	}

	stream.mu.Lock()
	defer stream.mu.Unlock()

	for i := 0; i < int(stream.formatCtx.nb_streams); i++ {
		cstream := C.stream_at((*C.struct_AVFormatContext)(stream.formatCtx), C.int(i))

		switch cstream.codecpar.codec_type {
		case C.AVMEDIA_TYPE_VIDEO, C.AVMEDIA_TYPE_AUDIO:
			decoder, err := newDecoder(cstream)
			if err != nil {
				return err
			}

			stream.decoders[decoder.index] = decoder
		case C.AVMEDIA_TYPE_DATA, C.AVMEDIA_TYPE_SUBTITLE, C.AVMEDIA_TYPE_NB, C.AVMEDIA_TYPE_ATTACHMENT:
		// do nothing
		default:
			err = fmt.Errorf("ffmpeg: failed: codec_type %d not found", cstream.codecpar.codec_type)
			return
		}
	}
	return
}

func (stream *Stream) ReadPacket() (pkt *Packet, err error) {
	packet := C.av_packet_alloc()
	defer C.av_packet_free(&packet)

	if cerr := C.av_read_frame(stream.formatCtx, packet); int(cerr) != 0 {
		if cerr == C.AVERROR_EOF {
			err = io.EOF
		} else if cerr == -C.ETIMEDOUT {
			err = ErrTimeout{fmt.Errorf("ffmpeg: av_read_frame failed: %s", CErr2Str(cerr))}
		} else {
			err = fmt.Errorf("ffmpeg: av_read_frame failed: %s", CErr2Str(cerr))
		}
		return
	}

	stream.mu.RLock()
	defer stream.mu.RUnlock()

	if decoder, ok := stream.decoders[int(packet.stream_index)]; ok {
		return decoder.Decode(packet)
	}

	err = fmt.Errorf("ffmpeg: decoder not found %d", int(packet.stream_index))
	return
}
