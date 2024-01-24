package rtsp

// Open rtsp stream or file
func Open(uri string, opts ...StreamOption) (*Stream, error) {
	stream := New(uri, opts...)
	return stream, stream.Setup()
}
