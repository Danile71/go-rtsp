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

type Type int

const (
	Auto Type = iota
	Tcp
	Udp
)

type Stream struct {
	formatCtx  *C.AVFormatContext
	dictionary *C.AVDictionary
	decoders   map[int]*decoder
	mu         sync.RWMutex
	url        string
}

func New(url string) (stream *Stream) {
	stream = &Stream{url: url}
	stream.decoders = make(map[int]*decoder)
	stream.formatCtx = C.avformat_alloc_context()

	runtime.SetFinalizer(stream, free)
	return
}

func free(stream *Stream) {
	if stream.formatCtx != nil {
		C.avformat_free_context(stream.formatCtx)
		stream.formatCtx = nil

	}

	if stream.dictionary != nil {
		C.av_dict_free(&stream.dictionary)
		stream.dictionary = nil
	}

	for _, decoder := range stream.decoders {
		if decoder != nil {
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

			decoder = nil
		}
	}
	stream.decoders = nil
}

func (stream *Stream) Setup(t Type) (err error) {
	transport := C.CString("rtsp_transport")
	defer C.free(unsafe.Pointer(transport))

	tcp := C.CString("tcp")
	defer C.free(unsafe.Pointer(tcp))

	udp := C.CString("udp")
	defer C.free(unsafe.Pointer(udp))

	switch t {
	case Tcp:
		C.av_dict_set(&stream.dictionary, transport, tcp, 0)
	case Udp:
		C.av_dict_set(&stream.dictionary, transport, udp, 0)
	default:
	}

	uri := C.CString(stream.url)
	defer C.free(unsafe.Pointer(uri))

	cerr := C.avformat_open_input(&stream.formatCtx, uri, nil, &stream.dictionary)
	if int(cerr) != 0 {
		err = fmt.Errorf("ffmpeg: avformat_open_input failed: %d", cerr)
		return
	}

	cerr = C.avformat_find_stream_info(stream.formatCtx, nil)
	if int(cerr) != 0 {
		err = fmt.Errorf("ffmpeg: avformat_find_stream_info failed: %d", cerr)
		return
	}

	stream.mu.Lock()
	defer stream.mu.Unlock()

	for i := 0; i < int(stream.formatCtx.nb_streams); i++ {
		cstream := C.stream_at((*C.struct_AVFormatContext)(stream.formatCtx), C.int(i))

		switch cstream.codecpar.codec_type {
		case C.AVMEDIA_TYPE_VIDEO, C.AVMEDIA_TYPE_AUDIO:
			decoder := &decoder{index: int(cstream.index)}
			decoder.swrContext = nil
			stream.decoders[decoder.index] = decoder
			decoder.codecCtx = C.avcodec_alloc_context3(nil)
			C.avcodec_parameters_to_context(decoder.codecCtx, cstream.codecpar)
			decoder.codec = C.avcodec_find_decoder(decoder.codecCtx.codec_id)
			decoder.codecType = int(cstream.codecpar.codec_type)
			if decoder.codec == nil {
				err = fmt.Errorf("ffmpeg: avcodec_find_decoder failed: codec %d not found", decoder.codecCtx.codec_id)
				return
			}

			if cerr = C.avcodec_open2(decoder.codecCtx, decoder.codec, nil); int(cerr) != 0 {
				err = fmt.Errorf("ffmpeg: avcodec_open2 failed: %d", cerr)
				return
			}
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
	var packet C.AVPacket
	C.av_init_packet(&packet)

	defer C.av_packet_unref(&packet)

	if cerr := C.av_read_frame(stream.formatCtx, &packet); int(cerr) != 0 {
		if cerr == C.AVERROR_EOF {
			err = io.EOF
		} else {
			err = fmt.Errorf("ffmpeg: av_read_frame failed: %d", cerr)
		}
		return
	}

	stream.mu.RLock()
	defer stream.mu.RUnlock()

	if decoder, ok := stream.decoders[int(packet.stream_index)]; ok {
		return decoder.Decode(&packet)
	}

	err = fmt.Errorf("ffmpeg: decoder not found %d", int(packet.stream_index))
	return
}
