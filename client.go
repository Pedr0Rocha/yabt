package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"strings"
	"time"
)

var client = http.Client{
	Timeout: 10 * time.Second,
}

var startTime time.Time

var requestInterval = 1 * time.Second

func run(ctx context.Context, responsesCh chan Response) {
	startTime = time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			request, err := http.NewRequest(method, requestUrl, strings.NewReader(""))
			if err != nil {
				fmt.Println("could not create request", err)
				return
			}

			var startT time.Time // represents when a successful connection is obtained
			var endT time.Time   // represents when the first byte of the response headers is available.

			trace := &httptrace.ClientTrace{
				// API also provides `ConnectDone`
				GotConn:              func(_ httptrace.GotConnInfo) { startT = time.Now() },
				GotFirstResponseByte: func() { endT = time.Now() },
			}
			request = request.WithContext(httptrace.WithClientTrace(ctx, trace))

			resp, err := client.Do(request)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				fmt.Println("err sending request", err)
				continue
			}
			code := resp.StatusCode
			serverProcessingTime := endT.Sub(startT)

			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()

			responsesCh <- Response{
				StatusCode:   code,
				ResponseTime: serverProcessingTime,
			}
			time.Sleep(requestInterval)
		}
	}
}
