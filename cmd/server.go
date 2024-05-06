package main

import (
	"log/slog"
	"net/http"
	"sync/atomic"
)

var requests = atomic.Int64{}

func main() {
	http.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		slog.Info("got request")
		w.Write([]byte("hello"))
	})

	slog.Info("Server running at 3000")
	http.ListenAndServe("127.0.0.1:3000", nil)
}
