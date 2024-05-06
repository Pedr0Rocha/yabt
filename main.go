package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	spinnerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Margin(1, 0)
	durationStyle = helpStyle.Copy().UnsetMargins()
	appStyle      = lipgloss.NewStyle().Margin(1, 2, 0, 2)
)

const URL = "http://localhost:3000"

var stop = make(chan os.Signal, 1)

type responseMsg int

var mapMutex sync.RWMutex
var responseMap map[int]int = map[int]int{
	1: 0,
	2: 0,
	3: 0,
	4: 0,
	5: 0,
}

type model struct {
	sub       chan int
	responses int
	spinner   spinner.Model
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

	if !m.quitting {
		s += helpStyle.Render("Press any key to exit")
	}

	if m.quitting {
		s += "closing UI...\n"
	}

	return s
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	respCh := make(chan int)

	go run(ctx, respCh)

	spinner := spinner.New()
	spinner.Style = spinnerStyle
	program := tea.NewProgram(model{
		sub:     respCh,
		spinner: spinner,
	})

	if _, err := program.Run(); err != nil {
		fmt.Println("could not start program:", err)
		os.Exit(1)
	}

	signal.Notify(stop, os.Interrupt, os.Kill)
	<-stop
	cancel()

	fmt.Println("closing client...")
	time.Sleep(1 * time.Second)
}