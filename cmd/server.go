package main

import (
	"log/slog"
	"math/rand"
	"net/http"
	"sync/atomic"
)

var requests = atomic.Int64{}

func main() {
	http.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		slog.Info("got request")

		randomResp := rand.Intn(10) + 1

		if randomResp <= 2 {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if randomResp == 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello"))
	})

	slog.Info("Server running at 3000")
	http.ListenAndServe("127.0.0.1:3000", nil)
}
