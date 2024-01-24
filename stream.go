package rtsp

/*
#include "ffmpeg.h"
*/
import "C"
import (
	"fmt"
	"io"
	"runtime"
	"sync"
	"unsafe"
)

type LogLevel C.int

const (
	AV_LOG_QUIET   = C.AV_LOG_QUIET
	AV_LOG_PANIC   = C.AV_LOG_PANIC
	AV_LOG_FATAL   = C.AV_LOG_FATAL
	AV_LOG_ERROR   = C.AV_LOG_ERROR
	AV_LOG_WARNING = C.AV_LOG_WARNING
	AV_LOG_INFO    = C.AV_LOG_INFO
	AV_LOG_VERBOSE = C.AV_LOG_VERBOSE
	AV_LOG_DEBUG   = C.AV_LOG_DEBUG
	AV_LOG_TRACE   = C.AV_LOG_TRACE
)

// SetLogLevel ffmpeg log level
func SetLogLevel(logLevel LogLevel) {
	C.av_log_set_level(C.int(logLevel))
}

// Type rtsp transport protocol
type Type int

const (
	// Tcp use tcp transport protocol
	Tcp = iota
	// Udp use udp transport  protocol
	Udp
)

// Stream media stream
type Stream struct {
	formatCtx  *C.AVFormatContext
	dictionary *C.AVDictionary

	decoders map[int]*decoder
	mu       sync.RWMutex
	uri      string
}

// New media stream
func New(uri string, opts ...StreamOption) (stream *Stream) {
	stream = &Stream{uri: uri}
	stream.decoders = make(map[int]*decoder)
	stream.formatCtx = C.avformat_alloc_context()

	for _, opt := range opts {
		opt(stream)
	}

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

// ErrTimeout ETIMEDOUT
type ErrTimeout struct {
	err error
}

// Error error interface
func (e ErrTimeout) Error() string {
	return e.err.Error()
}

// Setup stream
func (stream *Stream) Setup() (err error) {
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

// ReadPacket read frame from stream and decode it to Packet
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
		return decoder.decode(packet)
	}

	err = fmt.Errorf("ffmpeg: decoder not found %d", int(packet.stream_index))
	return
}
