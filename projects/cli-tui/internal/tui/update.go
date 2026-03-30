package tui

import (
	"context"
	"github.com/charmbracelet/bubbles/spinner"
	"fmt"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Messages — Bubble Tea uses typed messages to communicate between goroutines and the model.

type tickMsg time.Time
type concurrencyMsg struct{ keys map[string]int64 }
type errMsg struct{ err error }
type submitResultMsg struct{ msg string }

// Init returns the initial command: start spinner + fetch data immediately.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		fetchConcurrency(m.sched),
		tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return tickMsg(t) }),
	)
}

// Update handles all incoming messages and returns the new model + next command.
// This is the "U" in Elm architecture — pure function, no side effects.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.activeTab = (m.activeTab + 1) % tabCount
			m.submitResult = ""
			if m.activeTab == TabSubmit {
				m.jobNameInput.Focus()
				m.focusedField = 0
			}
		case "r":
			m.loading = true
			return m, fetchConcurrency(m.sched)
		case "enter":
			if m.activeTab == TabSubmit {
				return m, m.submitJob()
			}
		case "down", "j":
			if m.activeTab == TabSubmit {
				m.focusedField = (m.focusedField + 1) % 3
				m.updateFocus()
			}
		case "up", "k":
			if m.activeTab == TabSubmit {
				m.focusedField = (m.focusedField + 2) % 3
				m.updateFocus()
			}
		}

	case tickMsg:
		return m, tea.Batch(
			fetchConcurrency(m.sched),
			tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return tickMsg(t) }),
		)

	case concurrencyMsg:
		m.loading = false
		m.concurrencyKeys = msg.keys
		m.lastRefresh = time.Now()
		m.err = nil

	case errMsg:
		m.loading = false
		m.err = msg.err

	case submitResultMsg:
		m.submitResult = msg.msg
		m.loading = false
		return m, fetchConcurrency(m.sched)

	case spinner.TickMsg:
		// handled below
	}

	// Update sub-components
	var cmds []tea.Cmd
	var cmd tea.Cmd

	m.spinner, cmd = m.spinner.Update(msg)
	cmds = append(cmds, cmd)

	if m.activeTab == TabSubmit {
		m.jobNameInput, cmd = m.jobNameInput.Update(msg)
		cmds = append(cmds, cmd)
		m.tenantInput, cmd = m.tenantInput.Update(msg)
		cmds = append(cmds, cmd)
		m.priorityInput, cmd = m.priorityInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) updateFocus() {
	m.jobNameInput.Blur()
	m.tenantInput.Blur()
	m.priorityInput.Blur()
	switch m.focusedField {
	case 0:
		m.jobNameInput.Focus()
	case 1:
		m.tenantInput.Focus()
	case 2:
		m.priorityInput.Focus()
	}
}

// fetchConcurrency is a Bubble Tea command — runs in a goroutine, returns a message.
func fetchConcurrency(sched interface {
	GetConcurrencyKeys(context.Context) (map[string]int64, error)
}) tea.Cmd {
	return func() tea.Msg {
		keys, err := sched.GetConcurrencyKeys(context.Background())
		if err != nil {
			return errMsg{err}
		}
		return concurrencyMsg{keys}
	}
}

func (m Model) submitJob() tea.Cmd {
	jobName := m.jobNameInput.Value()
	if jobName == "" {
		jobName = "DataIngestion"
	}
	tenant, _ := strconv.Atoi(m.tenantInput.Value())
	if tenant == 0 {
		tenant = 1
	}
	priority, _ := strconv.Atoi(m.priorityInput.Value())
	if priority == 0 {
		priority = 5
	}

	return func() tea.Msg {
		err := m.sched.SubmitJob(context.Background(), jobName, tenant, priority)
		if err != nil {
			return submitResultMsg{fmt.Sprintf("✗ Error: %v", err)}
		}
		return submitResultMsg{fmt.Sprintf("✓ Submitted %s (tenant=%d, priority=%d)", jobName, tenant, priority)}
	}
}
