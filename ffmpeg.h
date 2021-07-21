#include <libavformat/avformat.h>
#include <libavcodec/avcodec.h>
#include <libavutil/avutil.h>

uint8_t *convert(AVCodecContext *pCodecCtx,AVFrame *pFrame,AVFrame *nFrame,int *size, int format);
int avcodec_encode_jpeg(AVCodecContext *pCodecCtx, AVFrame *pFrame,AVPacket *packet);
int avcodec_encode_jpeg_nv12(AVCodecContext *pCodecCtx, AVFrame *pFrame,AVFrame *nFrame,AVPacket *packet);

int open(AVFormatContext* format_ctx,AVCodecContext* codec_ctx,const char *uri);
struct AVStream * stream_at(struct AVFormatContext *c, int idx) ;