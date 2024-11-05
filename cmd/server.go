package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"sync/atomic"
	"time"
)

var requests = atomic.Int64{}

func main() {
	http.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)

		randomResp := rand.Intn(100) + 1

		if randomResp <= 10 {
			w.WriteHeader(http.StatusNotFound)
			return
		} else if randomResp >= 11 && randomResp <= 15 {
			w.WriteHeader(http.StatusPermanentRedirect)
			return
		} else if randomResp >= 16 && randomResp <= 20 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		} else if randomResp == 21 {
			w.WriteHeader(http.StatusSwitchingProtocols)
			return
		} else {
			w.WriteHeader(http.StatusOK)
		}

		w.Write([]byte("hello"))
	})

	http.HandleFunc("POST /", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)

		body := struct{ Test string }{}
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			fmt.Println("cant read the request body", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		fmt.Println(body)

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("posted"))
	})

	slog.Info("Server running at 3000")
	http.ListenAndServe("127.0.0.1:3000", nil)
}
