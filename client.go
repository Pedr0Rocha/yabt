package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

var client = http.Client{
	Timeout: 10 * time.Second,
}

var startTime time.Time

var requestInterval = 500 * time.Millisecond

func run(ctx context.Context, responsesCh chan int) {
	startTime = time.Now()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("stopped sending requests")
			return
		default:
			resp, err := http.Get(URL)
			if err != nil {
				fmt.Println("err sending req", err)
				continue
			}
			code := resp.StatusCode

			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()

			responsesCh <- code
			time.Sleep(requestInterval)
		}
	}
}
