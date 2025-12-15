package client

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Message types for Bubbletea
type StatusUpdateMsg struct {
	Status    string
	Server    string
	PublicURL string
}

type RequestAddedMsg struct {
	ID     string
	Method string
	Path   string
}

type RequestCompletedMsg struct {
	ID         string
	StatusCode int
	Duration   time.Duration
	BytesSent  int64
	BytesRecv  int64
}

type TickMsg time.Time

type QuitMsg struct{}

// Model holds the state for the Bubbletea TUI
type Model struct {
	status    string
	server    string
	publicURL string
	localPort int

	requests    []RequestLog
	maxRequests int
	bytesSent   int64
	bytesRecv   int64
	startTime   time.Time

	updateCh chan tea.Msg
	client   *Client
}

// NewModel creates a new Bubbletea model
func NewModel(client *Client, server string, localPort int) *Model {
	return &Model{
		status:      "connecting",
		server:      server,
		localPort:   localPort,
		requests:    make([]RequestLog, 0, 5),
		maxRequests: 5,
		startTime:   time.Now(),
		updateCh:    make(chan tea.Msg, 100),
		client:      client,
	}
}

// Init initializes the model and returns initial commands
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.tick(),
		m.waitForUpdates(),
	)
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	case StatusUpdateMsg:
		m.status = msg.Status
		if msg.Server != "" {
			m.server = msg.Server
		}
		if msg.PublicURL != "" {
			m.publicURL = msg.PublicURL
		}
		return m, m.tick()

	case RequestAddedMsg:
		req := RequestLog{
			ID:      msg.ID,
			Time:    time.Now(),
			Method:  msg.Method,
			Path:    msg.Path,
			Pending: true,
		}
		m.requests = append(m.requests, req)
		if len(m.requests) > m.maxRequests {
			m.requests = m.requests[1:]
		}
		return m, m.tick()

	case RequestCompletedMsg:
		for i := range m.requests {
			if m.requests[i].ID == msg.ID {
				m.requests[i].StatusCode = msg.StatusCode
				m.requests[i].Duration = msg.Duration
				m.requests[i].Pending = false
				break
			}
		}
		m.bytesSent += msg.BytesSent
		m.bytesRecv += msg.BytesRecv
		return m, m.tick()

	case TickMsg:
		return m, tea.Batch(
			m.tick(),
			m.waitForUpdates(),
		)

	case QuitMsg:
		return m, tea.Quit
	}

	return m, m.waitForUpdates()
}

// View renders the UI
func (m *Model) View() string {
	var sections []string

	// Header
	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("36")) // Cyan
	grayStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("90"))
	header := headerStyle.Render("digit-link") +
		"                                                    " +
		grayStyle.Render("(Ctrl+C to quit)")
	sections = append(sections, header)
	sections = append(sections, "")

	// Status section
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("33")) // Yellow
	statusColor := lipgloss.Color("32")                                // Green
	if m.status != "online" {
		statusColor = lipgloss.Color("33") // Yellow
	}
	statusStyle := lipgloss.NewStyle().Foreground(statusColor)

	sections = append(sections, fmt.Sprintf("%-20s %s",
		labelStyle.Render("Session Status"),
		statusStyle.Render(m.status)))
	sections = append(sections, fmt.Sprintf("%-20s %s",
		labelStyle.Render("Version"),
		"1.0.0"))
	sections = append(sections, fmt.Sprintf("%-20s %s",
		labelStyle.Render("Server"),
		m.server))

	// Forwarding line
	magentaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("35"))
	whiteStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("37"))
	forwardingText := m.publicURL
	if forwardingText == "" {
		forwardingText = "..."
	}
	forwarding := fmt.Sprintf("%s -> %s",
		magentaStyle.Render(forwardingText),
		whiteStyle.Render(fmt.Sprintf("http://localhost:%d", m.localPort)))
	sections = append(sections, fmt.Sprintf("%-20s %s",
		labelStyle.Render("Forwarding"),
		forwarding))
	sections = append(sections, "")

	// Stats section
	uptime := time.Since(m.startTime)
	sections = append(sections, fmt.Sprintf("%s          %-15s %-15s %-15s",
		labelStyle.Render("Stats"),
		"Uptime",
		"Sent",
		"Received"))
	sections = append(sections, fmt.Sprintf("               %-15s %-15s %-15s",
		formatUptime(uptime),
		formatBytes(m.bytesSent),
		formatBytes(m.bytesRecv)))
	sections = append(sections, "")

	// Recent requests header
	sections = append(sections, labelStyle.Render("Recent Requests"))
	dividerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("90"))
	sections = append(sections, dividerStyle.Render("─────────────────────────────────────────────────────────────────────────────────"))

	// Display last 5 requests
	for i := 0; i < m.maxRequests; i++ {
		if i < len(m.requests) {
			req := m.requests[len(m.requests)-1-i] // Most recent first

			// Truncate path if too long
			path := req.Path
			if len(path) > 40 {
				path = path[:37] + "..."
			}

			timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("90"))

			if req.Pending {
				// Show pending request with spinner
				spinChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
				spinIdx := int(time.Since(req.Time).Milliseconds()/100) % len(spinChars)
				spinnerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
				sections = append(sections, fmt.Sprintf("%s %-7s %-40s %s %8s",
					timeStyle.Render(req.Time.Format("15:04:05")),
					req.Method,
					path,
					spinnerStyle.Render(spinChars[spinIdx]),
					"...",
				))
			} else {
				statusCodeColor := lipgloss.Color("32") // Green
				if req.StatusCode >= 400 && req.StatusCode < 500 {
					statusCodeColor = lipgloss.Color("33") // Yellow
				} else if req.StatusCode >= 500 {
					statusCodeColor = lipgloss.Color("31") // Red
				}
				statusCodeStyle := lipgloss.NewStyle().Foreground(statusCodeColor)

				sections = append(sections, fmt.Sprintf("%s %-7s %-40s %s %8s",
					timeStyle.Render(req.Time.Format("15:04:05")),
					req.Method,
					path,
					statusCodeStyle.Render(fmt.Sprintf("%3d", req.StatusCode)),
					formatDuration(req.Duration),
				))
			}
		} else {
			// Empty line placeholder
			sections = append(sections, "")
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// tick returns a command that sends a TickMsg after 1 second
func (m *Model) tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// waitForUpdates waits for messages from the update channel
func (m *Model) waitForUpdates() tea.Cmd {
	return func() tea.Msg {
		select {
		case msg := <-m.updateCh:
			return msg
		case <-time.After(100 * time.Millisecond):
			return nil
		}
	}
}

// SendUpdate sends a message to the model via the update channel
func (m *Model) SendUpdate(msg tea.Msg) {
	select {
	case m.updateCh <- msg:
	default:
		// Channel full, drop message (shouldn't happen with buffer size 100)
	}
}

