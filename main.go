package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	appStyle      = lipgloss.NewStyle().Margin(1, 2, 0, 2)
	spinnerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("57"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Margin(1, 0)
	durationStyle = helpStyle.Copy().UnsetMargins()
	spinners      = []spinner.Spinner{
		spinner.Line,
		spinner.Dot,
		spinner.MiniDot,
		spinner.Jump,
		spinner.Pulse,
		spinner.Points,
		spinner.Globe,
		spinner.Moon,
		spinner.Monkey,
	}
	baseTableStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))
)

const URL = "http://localhost:3000"

var stop = make(chan os.Signal, 1)

type responseMsg int

var mapMutex sync.RWMutex
var responseMap map[int]int = map[int]int{
	1: 0, // each index represents 1XX status
	2: 0,
	3: 0,
	4: 0,
	5: 0,
}

type model struct {
	sub       chan int
	responses int
	spinner   spinner.Model
	table     table.Model
	quitting  bool
}

// A command that waits for responses from the client
func waitForResponses(sub chan int) tea.Cmd {
	return func() tea.Msg {
		resp := <-sub

		statusXX := resp / 100

		mapMutex.Lock()
		responseMap[statusXX]++
		mapMutex.Unlock()

		return responseMsg(resp)
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		waitForResponses(m.sub), // wait for responses
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.table.SetRows(mapToRows())

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
		"\n %s Time Elapsed: %s\n",
		m.spinner.View(),
		time.Since(startTime).Round(time.Second).String(),
	)
	s += fmt.Sprintf(" %s Requests Sent: %d\n", m.spinner.View(), m.responses)
	s += fmt.Sprintf(" %s Requests Interval: %s\n", m.spinner.View(), requestInterval)

	s += fmt.Sprintf("\n\n%s", m.table.View())

	if !m.quitting {
		s += helpStyle.Render(fmt.Sprintf("\n↑/↓ req/s • q: exit\n"))
	}

	if m.quitting {
		s += "closing UI...\n"
	}

	return appStyle.Render(s)
}

func mapToRows() []table.Row {
	mapMutex.RLock()
	defer mapMutex.RUnlock()

	return []table.Row{
		{"1xx", fmt.Sprint(responseMap[1]), "0ms"},
		{"2xx", fmt.Sprint(responseMap[2]), "37ms"},
		{"3xx", fmt.Sprint(responseMap[3]), "10ms"},
		{"4xx", fmt.Sprint(responseMap[4]), "212ms"},
		{"5xx", fmt.Sprint(responseMap[5]), "303ms"},
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	respCh := make(chan int)

	go run(ctx, respCh)

	spin := spinner.New()
	spin.Spinner = spinners[rand.Intn(len(spinners))]
	spin.Style = spinnerStyle

	columns := []table.Column{
		{Title: "Status", Width: 6},
		{Title: "Requests", Width: 8},
		{Title: "Avg.Resp", Width: 8},
	}

	rows := []table.Row{
		{"1xx", "0", "0ms"},
		{"2xx", "6013", "37ms"},
		{"3xx", "3", "10ms"},
		{"4xx", "67", "212ms"},
		{"5xx", "1", "303ms"},
	}

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
