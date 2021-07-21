#include <libavformat/avformat.h>
#include <libavcodec/avcodec.h>
#include <libavutil/avutil.h>
#include <libavutil/imgutils.h>
#include <libswresample/swresample.h>
#include <libswscale/swscale.h>
#include <libavutil/opt.h>
#include <string.h>
#include "ffmpeg.h"


struct AVStream * stream_at(struct AVFormatContext *c, int idx) {
    if (idx >= 0 && idx < c->nb_streams)
        return c->streams[idx];
    return NULL;
}

int open(AVFormatContext* format_ctx,AVCodecContext* codec_ctx,const char *uri)
{
    int ret;

    ret = avformat_open_input(&format_ctx, uri, NULL, NULL);
    if (ret < 0)
        return ret;

    return avformat_find_stream_info(format_ctx, NULL);
}

int decode2(AVCodecContext *avctx, AVFrame *frame,AVPacket *pkt, int *got_frame)
{
    int ret;

    *got_frame = 0;
    
    ret = avcodec_send_packet(avctx, pkt);
    
    av_packet_unref(pkt);
    
    if (ret < 0)
      return ret == AVERROR_EOF ? 0 : ret;
    

    ret = avcodec_receive_frame(avctx, frame);
    if (ret < 0 && ret != AVERROR(EAGAIN) && ret != AVERROR_EOF)
        return ret;
    if (ret >= 0)
        *got_frame = 1;

    return 0;
}


int decode(AVCodecContext *avctx, AVFrame *frame, uint8_t *data, int size, int *got_frame)
{
    int ret;
	struct AVPacket pkt = {.data = data, .size = size};

    *got_frame = 0;
    
    ret = avcodec_send_packet(avctx, &pkt);
    
    av_packet_unref(&pkt);
    
    if (ret < 0)
      return ret == AVERROR_EOF ? 0 : ret;
    

    ret = avcodec_receive_frame(avctx, frame);
    if (ret < 0 && ret != AVERROR(EAGAIN) && ret != AVERROR_EOF)
        return ret;
    if (ret >= 0)
        *got_frame = 1;

    return 0;
}

int encode(AVCodecContext *avctx, AVPacket *pkt, int *got_packet, AVFrame *frame)
{
    int ret;

    *got_packet = 0;

    ret = avcodec_send_frame(avctx, frame);
    if (ret < 0)
        return ret;

    ret = avcodec_receive_packet(avctx, pkt);
    if (!ret)
        *got_packet = 1;
    if (ret == AVERROR(EAGAIN))
        return 0;

    return ret;
}

int avcodec_encode_jpeg(AVCodecContext *pCodecCtx, AVFrame *pFrame,AVPacket *packet) {
    AVCodec *jpegCodec = avcodec_find_encoder(AV_CODEC_ID_MJPEG);
    int ret = -1;

    if (!jpegCodec) {
        return ret;
    }

    AVCodecContext *jpegContext = avcodec_alloc_context3(jpegCodec);
    if (!jpegContext) {
        jpegCodec = NULL;
        return ret;
    }

    jpegContext->pix_fmt = AV_PIX_FMT_YUVJ420P;
    jpegContext->height = pFrame->height;
    jpegContext->width = pFrame->width;
    jpegContext->time_base= (AVRational){1,25};

    ret = avcodec_open2(jpegContext, jpegCodec, NULL);
    if (ret < 0) {
        goto error;
    }
    
    int gotFrame;

    ret = encode(jpegContext, packet, &gotFrame, pFrame);
    if (ret < 0) {
        goto error;
    }
    
    error:
        avcodec_close(jpegContext);
        avcodec_free_context(&jpegContext);
        jpegCodec = NULL;
    return ret;
}

uint8_t *convert(AVCodecContext *pCodecCtx,AVFrame *pFrame,AVFrame *nFrame,int *size, int format) {
    struct SwsContext *img_convert_ctx = sws_getCachedContext( NULL, pCodecCtx->width, pCodecCtx->height, pCodecCtx->pix_fmt, pFrame->width, pFrame->height, format, SWS_BICUBIC, NULL, NULL, NULL );
    nFrame->format = format;
    nFrame->width = pFrame->width;
    nFrame->height = pFrame->height;
    
    *size = av_image_get_buffer_size( format, pFrame->width, pFrame->height, 1);
    
    uint8_t *tmp_picture_buf = (uint8_t *)malloc(*size);    
    
    av_image_fill_arrays(nFrame->data, nFrame->linesize, tmp_picture_buf, format, pFrame->width, pFrame->height, 1);
    
    sws_scale(img_convert_ctx, (const uint8_t* const*)pFrame->data, pFrame->linesize, 0, nFrame->height, nFrame->data, nFrame->linesize);   
    sws_freeContext(img_convert_ctx);
    return tmp_picture_buf;
}

int avcodec_encode_jpeg_nv12(AVCodecContext *pCodecCtx, AVFrame *pFrame,AVFrame *nFrame,AVPacket *packet) {
    int size = 0;
    uint8_t * data = convert(pCodecCtx, pFrame, nFrame, &size, AV_PIX_FMT_YUV420P);  
    int ret = avcodec_encode_jpeg(pCodecCtx,nFrame,packet);
    free(data);
    return ret;
}

