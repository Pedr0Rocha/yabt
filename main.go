package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	appStyle     = lipgloss.NewStyle().Margin(1, 2, 0, 2)
	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("57"))
	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Margin(1, 0)
	descStyle    = helpStyle.Copy().UnsetMargins()
	infoStyle    = helpStyle.Copy()
	spinners     = []spinner.Spinner{
		spinner.Line,
		spinner.Dot,
		spinner.MiniDot,
		spinner.Jump,
		spinner.Pulse,
		spinner.Points,
		// spinner.Globe,
		// spinner.Moon,
	}
	baseTableStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))
)

var (
	requestUrl = "http://localhost:8083"
	method     = "GET"
)

var stop = make(chan os.Signal, 1)

type responseMsg Response

type ResponseStats struct {
	Requests        int64
	ResponseTimes   time.Duration
	AvgResponseTime time.Duration
	MinResponseTime time.Duration
	MaxResponseTime time.Duration
}

type Response struct {
	StatusCode   int
	ResponseTime time.Duration
}

type ResponseStatsMap struct {
	Mutex sync.RWMutex
	Stats map[int]*ResponseStats
}

var responseMap *ResponseStatsMap = NewResponseStatsMap()

func NewResponseStatsMap() *ResponseStatsMap {
	return &ResponseStatsMap{
		Stats: map[int]*ResponseStats{
			1: {0, 0, 0, 0, 0},
			2: {0, 0, 0, 0, 0},
			3: {0, 0, 0, 0, 0},
			4: {0, 0, 0, 0, 0},
			5: {0, 0, 0, 0, 0},
		},
	}
}

func (rsm *ResponseStatsMap) AddResponse(response Response) {
	responseMap.Mutex.Lock()
	defer responseMap.Mutex.Unlock()

	statusXX := response.StatusCode / 100

	entry := responseMap.Stats[statusXX]

	entry.Requests++
	entry.ResponseTimes += response.ResponseTime

	if entry.MinResponseTime == 0 {
		entry.MinResponseTime = response.ResponseTime
	} else {
		entry.MinResponseTime = min(entry.MinResponseTime, response.ResponseTime)
	}
	entry.MaxResponseTime = max(entry.MaxResponseTime, response.ResponseTime)

	avg := entry.ResponseTimes.Nanoseconds() / entry.Requests
	entry.AvgResponseTime = time.Duration(avg)
}

func (rsm *ResponseStatsMap) MapToRows() []table.Row {
	responseMap.Mutex.RLock()
	defer responseMap.Mutex.RUnlock()

	table := []table.Row{}

	for i := 1; i < 6; i++ {
		table = append(table, []string{
			fmt.Sprintf("%dxx", i),
			fmt.Sprint(responseMap.Stats[i].Requests),
			fmt.Sprintf("%s", responseMap.Stats[i].AvgResponseTime),
			fmt.Sprintf("%s", responseMap.Stats[i].MinResponseTime),
			fmt.Sprintf("%s", responseMap.Stats[i].MaxResponseTime),
		})
	}

	return table
}

type model struct {
	sub       chan Response
	responses int
	spinner   spinner.Model
	table     table.Model
	quitting  bool
}

var totalRequests = atomic.Int64{}

func waitForResponses(sub chan Response) tea.Cmd {
	return func() tea.Msg {
		resp := <-sub

		responseMap.AddResponse(resp)
		totalRequests.Add(1)

		return responseMsg(resp)
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		waitForResponses(m.sub),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.table.SetRows(responseMap.MapToRows())

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyUp {
			requestInterval = requestInterval / 2
			return m, nil
		}

		if msg.Type == tea.KeyDown {
			requestInterval = requestInterval * 2
			return m, nil
		}

		if msg.String() == "r" {
			startTime = time.Now()
			totalRequests.Store(0)
			bestReqPerSecond.Store(0)
			responseMap = NewResponseStatsMap()
			return m, nil
		}

		stop <- os.Interrupt
		m.quitting = true
		return m, tea.Quit
	case responseMsg:
		m.responses++
		return m, waitForResponses(m.sub) // wait for next event
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	default:
		return m, nil
	}
}

func (m model) View() string {
	s := fmt.Sprintf(
		"%s Time Elapsed: %s\n",
		m.spinner.View(),
		time.Since(startTime).Round(time.Second).String(),
	)
	s += fmt.Sprintf("%s Requests Sent: %d\n", m.spinner.View(), totalRequests.Load())
	s += fmt.Sprintf(
		"%s Req/Second: %d %s\n",
		m.spinner.View(),
		reqPerSecond.Load(),
		descStyle.Render(fmt.Sprintf("(%d best)", bestReqPerSecond.Load())),
	)
	s += fmt.Sprintf("\nRequests Interval: %s\n", requestInterval)

	s += fmt.Sprintf("\n\n%s", m.table.View())

	s += infoStyle.Render(fmt.Sprintf("\n%s %s\n", method, requestUrl))

	if !m.quitting {
		s += helpStyle.Render(fmt.Sprintf("\n↑/↓: req/s • r: reset stats • q: exit\n"))
	}

	if m.quitting {
		s += "\n\nshutting down...\n"
	}

	return appStyle.Render(s)
}

var reqPerSecond = atomic.Int64{}
var bestReqPerSecond = atomic.Int64{}

func calc(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	lastTickReq := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			reqPerSecond.Store(totalRequests.Load() - int64(lastTickReq))
			lastTickReq = int(totalRequests.Load())

			if reqPerSecond.Load() > bestReqPerSecond.Load() {
				bestReqPerSecond.Store(reqPerSecond.Load())
			}
		}
	}
}

var urlFlag = flag.String("url", "", "URL to send requests to | e.g. https://google.com")

func main() {
	flag.Parse()

	if urlFlag == nil || *urlFlag == "" {
		fmt.Println("URL is required, please provite it with -url flag")
		os.Exit(1)
	}

	reqUrl, err := url.Parse(*urlFlag)
	if err != nil || reqUrl.Scheme == "" || reqUrl.Host == "" {
		fmt.Println("could not parse url", reqUrl, err)
		os.Exit(1)
	}
	requestUrl = reqUrl.String()

	ctx, cancel := context.WithCancel(context.Background())

	respCh := make(chan Response)

	go run(ctx, respCh)
	go calc(ctx)

	spin := spinner.New()
	spin.Spinner = spinners[rand.Intn(len(spinners))]
	spin.Style = spinnerStyle

	columns := []table.Column{
		{Title: "Status", Width: 6},
		{Title: "Requests", Width: 8},
		{Title: "Resp. Time (avg)", Width: 16},
		{Title: "Min.", Width: 14},
		{Title: "Max.", Width: 14},
	}

	rows := []table.Row{}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(false),
		table.WithHeight(5),
	)

	s := table.DefaultStyles()
	s.Selected = lipgloss.Style{}

	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		BorderBottom(true).
		Bold(false)
	t.SetStyles(s)

	program := tea.NewProgram(model{
		sub:     respCh,
		spinner: spin,
		table:   t,
	})

	if _, err := program.Run(); err != nil {
		fmt.Println("could not start program:", err)
		os.Exit(1)
	}

	signal.Notify(stop, os.Interrupt, os.Kill)
	<-stop
	cancel()

	time.Sleep(1 * time.Second)
}
