package rtsp

// Open rtsp stream or file
func Open(uri string) (*Stream, error) {
	stream := New(uri)
	return stream, stream.Setup(Auto)
}
