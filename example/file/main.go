package main

import (
	"net/http"

	"github.com/Danile71/go-logger"
	"github.com/Danile71/go-rtsp"
	"github.com/gorilla/mux"
	"github.com/mattn/go-mjpeg"
)

const uri = "./sample.mp4"

func main() {
	s := mjpeg.NewStream()

	stream, err := rtsp.Open(uri)
	if logger.OnError(err) {
		return
	}

	go func() {
		for {
			pkt, err := stream.ReadPacket()
			if logger.OnError(err) {
				continue
			}

			if pkt.IsVideo() {
				s.Update(pkt.Data())
			}
		}
	}()

	streamHandler := func(w http.ResponseWriter, r *http.Request) {
		s.ServeHTTP(w, r)
	}

	router := mux.NewRouter()
	router.HandleFunc("/stream", streamHandler)
	http.Handle("/", router)
	http.ListenAndServe(":8181", nil)
}
