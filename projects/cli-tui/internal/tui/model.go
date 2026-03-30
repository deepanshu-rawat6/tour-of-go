package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"tour_of_go/projects/cli-tui/internal/client"
)

// Tab represents a view tab.
type Tab int

const (
	TabConcurrency Tab = iota
	TabSubmit
	tabCount
)

var tabNames = []string{"Concurrency Pool", "Submit Job"}

// Model is the Bubble Tea model — all application state lives here.
// This is the "M" in the Elm architecture (Model-Update-View).
type Model struct {
	// Navigation
	activeTab Tab
	width     int
	height    int

	// Data
	concurrencyKeys map[string]int64
	lastRefresh     time.Time
	err             error

	// Submit form fields
	jobNameInput  textinput.Model
	tenantInput   textinput.Model
	priorityInput textinput.Model
	focusedField  int
	submitResult  string

	// Loading state
	spinner  spinner.Model
	loading  bool

	// Scheduler client
	sched *client.Client
}

// NewModel creates the initial model.
func NewModel(sched *client.Client) Model {
	jobName := textinput.New()
	jobName.Placeholder = "DataIngestion"
	jobName.Focus()
	jobName.CharLimit = 64

	tenant := textinput.New()
	tenant.Placeholder = "1"
	tenant.CharLimit = 10

	priority := textinput.New()
	priority.Placeholder = "5"
	priority.CharLimit = 2

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return Model{
		sched:         sched,
		jobNameInput:  jobName,
		tenantInput:   tenant,
		priorityInput: priority,
		spinner:       sp,
		loading:       true,
	}
}
