# Yet Another Benchmark Tool

However, this one is **interactive** (and beautiful, thanks to [bubbletea](https://github.com/charmbracelet/bubbletea)).
The use case is to tweak the req/sec on the fly by increasing or decreasing the interval between each request.
Each time you press up, it will divide the request interval by 2. Same for arrow down, which will multiply it by 2.

Response time is calculated from the moment the connection is successfully established until the first byte of the
response header is available. Meaning it should only measure the server processing time.

It also features reset stats ("r"). This will reset all the stats and keep the req/s as is. So you can find the right req/s
and start calculating stats from there.

## DEMO

It starts with `Request Interval: 500ms` by default. You can change it by pressing up/down. The server in the demo
returns random http statuses back and sleeps for 0~10ms.


https://github.com/Pedr0Rocha/yabt/assets/7094503/262af3b0-3253-4bb2-a0bc-74fdc3f3246e

