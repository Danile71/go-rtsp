package main

import (
	"io"
	"net/http"
	"os"

	"github.com/Danile71/go-logger"
	"github.com/Danile71/go-rtsp"
	"github.com/gorilla/mux"
	"github.com/mattn/go-mjpeg"
)

const uri = "rtsp://admin:admin@127.0.0.1:554"

func main() {
	s := mjpeg.NewStream()

	stream := rtsp.New(uri)

	err := stream.Setup(rtsp.Tcp) // or rtsp.Udp or rtsp.Auto
	if logger.OnError(err) {
		return
	}

	go func() {
		for {
			pkt, err := stream.ReadPacket()
			if logger.OnError(err) {
				if err == io.EOF {
					os.Exit(1)
				}
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
