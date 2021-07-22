package rtsp

// Open rtsp stream of file
func Open(url string) (*Stream, error) {
	stream := New(url)
	return stream, stream.Setup(Auto)
}
