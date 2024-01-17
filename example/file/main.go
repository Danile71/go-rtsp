package main

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/Danile71/go-rtsp"
	"github.com/gorilla/mux"
	"github.com/mattn/go-mjpeg"
)

const uri = "./sample.mp4"

func main() {
	s := mjpeg.NewStream()

	stream, err := rtsp.Open(uri)
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
				time.Sleep(time.Millisecond * 10)
			}
		}
	}()

	streamHandler := func(w http.ResponseWriter, r *http.Request) {
		s.ServeHTTP(w, r)
	}

	router := mux.NewRouter()
	router.HandleFunc("/stream", streamHandler)
	http.Handle("/", router)
	if err := http.ListenAndServe(":8181", nil); err != nil {
		slog.Error(
			"listen",

			"error", err,
		)
	}
}
