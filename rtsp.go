package rtsp

func Open(url string) (*Stream, error) {
	stream := New(url)
	return stream, stream.Setup(Auto)
}
