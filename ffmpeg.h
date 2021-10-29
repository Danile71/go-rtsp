#include <libavformat/avformat.h>
#include <libavcodec/avcodec.h>
#include <libavutil/avutil.h>
#include <libavutil/imgutils.h>
#include <libswresample/swresample.h>
#include <libswscale/swscale.h>
#include <libavutil/opt.h>
#include <string.h>

uint8_t *rtsp_convert(AVCodecContext *pCodecCtx,AVFrame *pFrame,AVFrame *nFrame,int *size, int format);
int rtsp_avcodec_encode_jpeg(AVCodecContext *pCodecCtx, AVFrame *pFrame,AVPacket *packet);
int rtsp_avcodec_encode_jpeg_nv12(AVCodecContext *pCodecCtx, AVFrame *pFrame,AVPacket *packet);
struct AVStream * stream_at(struct AVFormatContext *c, int idx);
void ffmpeginit();
int rtsp_avcodec_encode_wav(AVCodecContext *pCodecCtx, AVFrame *pFrame,AVPacket *packet);
int rtsp_avcodec_encode_resample_wav(AVCodecContext *pCodecCtx,SwrContext *swr_ctx, AVFrame *pFrame,AVPacket *packet);