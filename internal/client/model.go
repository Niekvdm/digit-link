package client

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Style definitions
var (
	// Colors - Mustard yellow accent theme
	colorMustardYellow = lipgloss.Color("220") // Bright yellow/gold (mustard yellow)
	colorYellow        = lipgloss.Color("178") // Softer yellow
	colorGreen         = lipgloss.Color("82")  // Bright green for success
	colorWhite         = lipgloss.Color("255") // Pure white
	colorGray          = lipgloss.Color("244") // Light gray
	colorDarkGray      = lipgloss.Color("238") // Darker gray
	colorRed           = lipgloss.Color("196") // Bright red for errors
	colorBlue          = lipgloss.Color("75")  // Soft blue
	colorCyan          = lipgloss.Color("87")  // Cyan blue

	// Header styles
	headerTitleStyle = lipgloss.NewStyle().
				Foreground(colorMustardYellow).
				Bold(true)
	headerSubtextStyle = lipgloss.NewStyle().
				Foreground(colorGray)

	// Section box styles
	sectionBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorGray).
			Padding(0, 1)
	mainBoxStyle = lipgloss.NewStyle().
			Padding(1, 0)

	// Label styles
	labelStyle = lipgloss.NewStyle().
			Foreground(colorMustardYellow).
			Bold(true)
	valueStyle = lipgloss.NewStyle().
			Foreground(colorWhite)

	// Status badge styles
	statusBadgeOnline = lipgloss.NewStyle().
				Foreground(lipgloss.Color("232")).
				Background(colorGreen).
				Bold(true).
				Padding(0, 1).
				MarginLeft(1)
	statusBadgeConnecting = lipgloss.NewStyle().
				Foreground(lipgloss.Color("232")).
				Background(colorMustardYellow).
				Bold(true).
				Padding(0, 1).
				MarginLeft(1)
	statusBadgeReconnecting = lipgloss.NewStyle().
				Foreground(lipgloss.Color("232")).
				Background(colorMustardYellow).
				Bold(true).
				Padding(0, 1).
				MarginLeft(1)
	statusBadgeRejected = lipgloss.NewStyle().
				Foreground(lipgloss.Color("232")).
				Background(colorRed).
				Bold(true).
				Padding(0, 1).
				MarginLeft(1)

	// Method badge styles
	methodGETStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("232")).
			Background(colorBlue).
			Bold(true).
			Padding(0, 1).
			Width(7).
			Align(lipgloss.Center)
	methodPOSTStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("232")).
			Background(colorGreen).
			Bold(true).
			Padding(0, 1).
			Width(7).
			Align(lipgloss.Center)
	methodPUTStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("232")).
			Background(colorMustardYellow).
			Bold(true).
			Padding(0, 1).
			Width(7).
			Align(lipgloss.Center)
	methodDELETEStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("232")).
				Background(colorRed).
				Bold(true).
				Padding(0, 1).
				Width(7).
				Align(lipgloss.Center)
	methodDefaultStyle = lipgloss.NewStyle().
				Foreground(colorWhite).
				Background(colorDarkGray).
				Bold(true).
				Padding(0, 1).
				Width(7).
				Align(lipgloss.Center)

	// Request row styles
	requestRowStyle = lipgloss.NewStyle().
			Padding(0, 1)
	requestRowPendingStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Background(colorDarkGray).
				Foreground(colorWhite)

	// Status code styles
	statusCodeSuccess     = lipgloss.NewStyle().Foreground(colorGreen).Bold(true)
	statusCodeClientError = lipgloss.NewStyle().Foreground(colorMustardYellow).Bold(true)
	statusCodeServerError = lipgloss.NewStyle().Foreground(colorRed).Bold(true)

	// Time style
	timeStyle = lipgloss.NewStyle().Foreground(colorGray)

	// URL styles
	urlPublicStyle = lipgloss.NewStyle().Foreground(colorMustardYellow).Bold(true)
	urlLocalStyle  = lipgloss.NewStyle().Foreground(colorWhite)
)

// Message types for Bubbletea
type StatusUpdateMsg struct {
	Status    string
	Server    string
	PublicURL string
	Error     string // Error message for rejected/error status
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
	status       string
	server       string
	publicURL    string
	localPort    int
	errorMessage string // Error message when status is "rejected"

	requests    []RequestLog
	maxRequests int
	bytesSent   int64
	bytesRecv   int64
	startTime   time.Time

	spinner   spinner.Model
	paginator paginator.Model
	pageSize  int

	updateCh chan tea.Msg
	client   *Client
}

// NewModel creates a new Bubbletea model
func NewModel(client *Client, server string, localPort int) *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorMustardYellow)

	p := paginator.New()
	p.Type = paginator.Dots
	p.PerPage = 5
	p.ActiveDot = lipgloss.NewStyle().Foreground(colorMustardYellow).Render("•")
	p.InactiveDot = lipgloss.NewStyle().Foreground(colorGray).Render("•")

	return &Model{
		status:      "connecting",
		server:      server,
		localPort:   localPort,
		requests:    make([]RequestLog, 0, 50),
		maxRequests: 50,
		startTime:   time.Now(),
		spinner:     s,
		paginator:   p,
		pageSize:    5,
		updateCh:    make(chan tea.Msg, 100),
		client:      client,
	}
}

// Init initializes the model and returns initial commands
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.tick(),
		m.waitForUpdates(),
		m.spinner.Tick,
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
		// Let paginator handle navigation keys (left/right/h/l)
		var cmd tea.Cmd
		m.paginator, cmd = m.paginator.Update(msg)
		return m, cmd

	case StatusUpdateMsg:
		m.status = msg.Status
		if msg.Server != "" {
			m.server = msg.Server
		}
		if msg.PublicURL != "" {
			m.publicURL = msg.PublicURL
		}
		if msg.Error != "" {
			m.errorMessage = msg.Error
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
		// Update paginator total pages (pass item count, not page count)
		m.paginator.SetTotalPages(len(m.requests))
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(nil)
		return m, tea.Batch(m.tick(), cmd)

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
		// Update paginator total pages (pass item count, not page count)
		m.paginator.SetTotalPages(len(m.requests))
		return m, m.tick()

	case TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(nil)
		return m, tea.Batch(
			m.tick(),
			m.waitForUpdates(),
			cmd,
		)

	case QuitMsg:
		return m, tea.Quit

	default:
		// Handle spinner tick messages and other unknown messages
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, m.waitForUpdates()
}

// getStatusBadge returns the appropriate status badge style
func (m *Model) getStatusBadge() string {
	switch m.status {
	case "online":
		return statusBadgeOnline.Render("● ONLINE")
	case "connecting":
		return statusBadgeConnecting.Render("◉ CONNECTING")
	case "reconnecting":
		return statusBadgeReconnecting.Render("◉ RECONNECTING")
	case "rejected":
		return statusBadgeRejected.Render("✕ REJECTED")
	default:
		return statusBadgeConnecting.Render("◉ " + strings.ToUpper(m.status))
	}
}

// getMethodBadge returns the styled method badge
func getMethodBadge(method string) string {
	switch strings.ToUpper(method) {
	case "GET":
		return methodGETStyle.Render("GET")
	case "POST":
		return methodPOSTStyle.Render("POST")
	case "PUT":
		return methodPUTStyle.Render("PUT")
	case "DELETE":
		return methodDELETEStyle.Render("DELETE")
	case "PATCH":
		return methodPUTStyle.Render("PATCH")
	default:
		return methodDefaultStyle.Render(method)
	}
}

// getStatusCodeStyle returns the appropriate status code style
func getStatusCodeStyle(code int) lipgloss.Style {
	if code >= 200 && code < 300 {
		return statusCodeSuccess
	} else if code >= 400 && code < 500 {
		return statusCodeClientError
	} else if code >= 500 {
		return statusCodeServerError
	}
	return statusCodeSuccess
}

// View renders the UI
func (m *Model) View() string {
	var content []string

	// Header: title on left, quit text on right
	title := headerTitleStyle.Render("digit-link")
	quitText := headerSubtextStyle.Render("(Press Ctrl+C or 'q' to quit)")
	headerWidth := 80
	spacerWidth := headerWidth - lipgloss.Width(title) - lipgloss.Width(quitText)
	if spacerWidth < 0 {
		spacerWidth = 0
	}
	header := lipgloss.JoinHorizontal(lipgloss.Left,
		title,
		lipgloss.NewStyle().Width(spacerWidth).Render(""),
		quitText,
	)

	// Status Section
	statusLine := labelStyle.Render("Session Status") + m.getStatusBadge()
	if m.status == "connecting" {
		statusLine += " " + m.spinner.View()
	}
	content = append(content, statusLine)

	// Show error message if rejected
	if m.status == "rejected" && m.errorMessage != "" {
		content = append(content, "")
		errorStyle := lipgloss.NewStyle().
			Foreground(colorRed).
			Bold(true)
		content = append(content, errorStyle.Render("Error: "+m.errorMessage))
		content = append(content, "")
		hintStyle := lipgloss.NewStyle().
			Foreground(colorGray).
			Italic(true)
		content = append(content, hintStyle.Render("Check your token, IP whitelist settings, or contact your administrator."))
	}

	content = append(content, "")
	content = append(content, labelStyle.Render("Version")+valueStyle.MarginLeft(2).Render("1.0.0"))
	content = append(content, labelStyle.Render("Server")+valueStyle.MarginLeft(2).Render(m.server))

	// Forwarding line
	forwardingText := m.publicURL
	if forwardingText == "" {
		forwardingText = "..."
	}
	forwarding := urlPublicStyle.Render(forwardingText) +
		" → " +
		urlLocalStyle.Render(fmt.Sprintf("http://localhost:%d", m.localPort))
	content = append(content, labelStyle.Render("Forwarding")+valueStyle.MarginLeft(2).Render(forwarding))
	content = append(content, "")

	// Stats Section (two-column layout)
	uptime := time.Since(m.startTime)
	content = append(content, lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(30).Render(
			labelStyle.Render("Uptime")+"\n"+
				valueStyle.Render(formatUptime(uptime)),
		),
		lipgloss.NewStyle().Width(30).Render(
			labelStyle.Render("Sent")+"\n"+
				valueStyle.Render(formatBytes(m.bytesSent)),
		),
		lipgloss.NewStyle().Width(30).Render(
			labelStyle.Render("Received")+"\n"+
				valueStyle.Render(formatBytes(m.bytesRecv)),
		),
	))

	// Add rate indicators if we have data
	if uptime.Seconds() > 0 {
		sentRate := float64(m.bytesSent) / uptime.Seconds()
		recvRate := float64(m.bytesRecv) / uptime.Seconds()
		content = append(content,
			lipgloss.JoinHorizontal(lipgloss.Top,
				lipgloss.NewStyle().Width(30).Render(""),
				lipgloss.NewStyle().Width(30).Render(
					timeStyle.Render(fmt.Sprintf("↑ %s/s", formatBytes(int64(sentRate)))),
				),
				lipgloss.NewStyle().Width(30).Render(
					timeStyle.Render(fmt.Sprintf("↓ %s/s", formatBytes(int64(recvRate)))),
				),
			),
		)
	}
	content = append(content, "")
	content = append(content, timeStyle.Render(strings.Repeat("─", 80)))

	// Table header
	headerRow := lipgloss.JoinHorizontal(lipgloss.Left,
		lipgloss.NewStyle().Width(10).Render(timeStyle.Render("Time")),
		lipgloss.NewStyle().Width(9).Render(labelStyle.Render("Method")),
		lipgloss.NewStyle().Width(42).Render(labelStyle.Render("Path")),
		lipgloss.NewStyle().Width(8).Render(labelStyle.Render("Status")),
		lipgloss.NewStyle().Width(10).Render(labelStyle.Render("Duration")),
	)
	content = append(content, headerRow)
	content = append(content, timeStyle.Render(strings.Repeat("─", 80)))

	// Reverse requests to show most recent first
	reversed := make([]RequestLog, len(m.requests))
	for i := range m.requests {
		reversed[i] = m.requests[len(m.requests)-1-i]
	}

	// Ensure paginator total pages is set (pass item count, not page count)
	m.paginator.SetTotalPages(len(reversed))
	totalPages := m.paginator.TotalPages
	if totalPages == 0 {
		totalPages = 1
	}

	// Use paginator's GetSliceBounds to get the correct slice
	start, end := m.paginator.GetSliceBounds(len(reversed))
	var paginatedRequests []RequestLog
	if len(reversed) > 0 && start < len(reversed) {
		if end > len(reversed) {
			end = len(reversed)
		}
		paginatedRequests = reversed[start:end]
	}

	// Display paginated requests (always show 5 rows)
	for i := 0; i < m.pageSize; i++ {
		if i < len(paginatedRequests) {
			req := paginatedRequests[i]

			// Truncate path if too long
			path := req.Path
			if len(path) > 40 {
				path = path[:37] + "..."
			}

			var rowStyle lipgloss.Style
			if req.Pending {
				rowStyle = requestRowPendingStyle
			} else {
				rowStyle = requestRowStyle
			}

			if req.Pending {
				// Show pending request with bubbles spinner
				row := lipgloss.JoinHorizontal(lipgloss.Left,
					lipgloss.NewStyle().Width(10).Render(timeStyle.Render(req.Time.Format("15:04:05"))),
					lipgloss.NewStyle().Width(9).Render(getMethodBadge(req.Method)),
					lipgloss.NewStyle().Width(42).Render(valueStyle.Render(path)),
					lipgloss.NewStyle().Width(8).Render(m.spinner.View()),
					lipgloss.NewStyle().Width(10).Render(timeStyle.Render("...")),
				)
				content = append(content, rowStyle.Render(row))
			} else {
				statusCodeStyle := getStatusCodeStyle(req.StatusCode)

				row := lipgloss.JoinHorizontal(lipgloss.Left,
					lipgloss.NewStyle().Width(10).Render(timeStyle.Render(req.Time.Format("15:04:05"))),
					lipgloss.NewStyle().Width(9).Render(getMethodBadge(req.Method)),
					lipgloss.NewStyle().Width(42).Render(valueStyle.Render(path)),
					lipgloss.NewStyle().Width(8).Render(statusCodeStyle.Render(fmt.Sprintf("%3d", req.StatusCode))),
					lipgloss.NewStyle().Width(10).Render(timeStyle.Render(formatDuration(req.Duration))),
				)
				content = append(content, rowStyle.Render(row))
			}
		} else {
			// Empty row placeholder
			emptyRow := lipgloss.JoinHorizontal(lipgloss.Left,
				lipgloss.NewStyle().Width(10).Render(""),
				lipgloss.NewStyle().Width(9).Render(""),
				lipgloss.NewStyle().Width(42).Render(""),
				lipgloss.NewStyle().Width(8).Render(""),
				lipgloss.NewStyle().Width(10).Render(""),
			)
			content = append(content, requestRowStyle.Render(emptyRow))
		}
	}

	// Add pagination indicator if there are multiple pages
	if totalPages > 1 {
		content = append(content, "")
		paginationText := lipgloss.NewStyle().
			Foreground(colorGray).
			Render(fmt.Sprintf("Page %d of %d (← → to navigate)", m.paginator.Page+1, totalPages))
		content = append(content, paginationText)
		content = append(content, m.paginator.View())
	}

	// Combine everything into one box (no border)
	boxContent := mainBoxStyle.Render(strings.Join(content, "\n"))

	return lipgloss.JoinVertical(lipgloss.Left, header, "", boxContent)
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
