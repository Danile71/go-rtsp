package main

import (
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/Danile71/go-rtsp"
	"github.com/mattn/go-mjpeg"
)

const uri = "rtsp://192.168.139.24:8554/mystream"

func main() {
	// Set ffmpeg log level
	rtsp.SetLogLevel(rtsp.AV_LOG_TRACE)

	// Create mjpeg instance
	s := mjpeg.NewStream()

	// Prepare stream
	stream := rtsp.New(uri,
		// Set transport
		rtsp.WithType(rtsp.Tcp),

		// Set timeout
		// rtsp.WithTimeout("1000"),
	)

	// Setup and open stream
	err := stream.Setup()
	if err != nil {
		slog.Error(
			"setup stream",

			"error", err,
		)
		return
	}

	go func() {
		for {
			pkt, err := stream.ReadPacket()
			if err != nil {
				if err == io.EOF {
					os.Exit(0)
				}
				slog.Error(
					"read packet",

					"error", err,
				)
				continue
			}

			if pkt.IsVideo() {
				if err := s.Update(pkt.Data()); err != nil {
					slog.Error(
						"write packet",

						"error", err,
					)
				}
			}
		}
	}()

	http.Handle("/stream", s)

	if err := http.ListenAndServe(":8181", nil); err != nil {
		slog.Error(
			"listen",

			"error", err,
		)
	}
}
