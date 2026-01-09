package client

import (
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/niekvdm/digit-link/internal/tunnel"
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
	colorOrange        = lipgloss.Color("208") // Orange for client errors (4xx)
	colorBlue          = lipgloss.Color("75")  // Soft blue
	colorCyan          = lipgloss.Color("87")  // Cyan blue for redirects (3xx)

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
	statusCodeSuccess     = lipgloss.NewStyle().Foreground(colorGreen).Bold(true)      // 2xx
	statusCodeRedirect    = lipgloss.NewStyle().Foreground(colorCyan).Bold(true)       // 3xx
	statusCodeClientError = lipgloss.NewStyle().Foreground(colorOrange).Bold(true)     // 4xx
	statusCodeServerError = lipgloss.NewStyle().Foreground(colorRed).Bold(true)        // 5xx
	statusCodePending     = lipgloss.NewStyle().Foreground(colorGray).Italic(true)     // Pending

	// Time style
	timeStyle = lipgloss.NewStyle().Foreground(colorGray)

	// URL styles
	urlPublicStyle = lipgloss.NewStyle().Foreground(colorMustardYellow).Bold(true)
	urlLocalStyle  = lipgloss.NewStyle().Foreground(colorWhite)
)

// Message types for Bubbletea
type StatusUpdateMsg struct {
	Status       string
	Server       string
	PublicURL    string              // Primary URL (for backward compatibility)
	Tunnels      []tunnel.TunnelInfo // All tunnel URLs (for multi-forward)
	Error        string              // Error message for rejected/error status
	RetryCount   int                 // Retry count when reconnecting
	RetryBackoff time.Duration       // Backoff duration when reconnecting
}

type RequestAddedMsg struct {
	ID        string
	Method    string
	Path      string
	Subdomain string // Subdomain this request came from (for multi-forward)
	BytesRecv int64
}

type RequestCompletedMsg struct {
	ID         string
	StatusCode int
	Duration   time.Duration
	BytesSent  int64
	BytesRecv  int64
}

type TickMsg time.Time

type FastTickMsg time.Time // Fast tick for pending request timer updates

type QuitMsg struct{}

// WebSocket connection messages
type WebSocketConnectedMsg struct {
	ID        string
	Path      string
	Subdomain string
}

type WebSocketDataMsg struct {
	ID        string
	BytesSent int64
	BytesRecv int64
}

type WebSocketClosedMsg struct {
	ID string
}

// WebSocketLog represents an active or closed WebSocket connection
type WebSocketLog struct {
	ID        string
	Path      string
	Subdomain string
	StartTime time.Time
	EndTime   time.Time
	BytesSent int64
	BytesRecv int64
	Active    bool
}

// Model holds the state for the Bubbletea TUI
type Model struct {
	status       string
	server       string
	publicURL    string              // Primary URL (for backward compatibility)
	tunnels      []tunnel.TunnelInfo // All tunnels (for multi-forward)
	localPort    int
	localAddr    string
	localHTTPS   bool
	errorMessage string // Error message when status is "rejected"

	requests    []RequestLog
	maxRequests int
	bytesSent   int64
	bytesRecv   int64
	startTime   time.Time

	// WebSocket connections (separate from HTTP requests)
	wsConnections    []WebSocketLog
	maxWSConnections int
	wsBytesSent      int64
	wsBytesRecv      int64
	totalWSConns     int64
	activeWSConns    int

	// Request view (0=HTTP, 1=WebSocket)
	requestView   int
	wsPaginator   paginator.Model
	wsSelectedIdx int

	spinner   spinner.Model
	paginator paginator.Model
	pageSize  int

	updateCh chan tea.Msg
	client   *Client

	// Stats tabs (0=Traffic, 1=Requests, 2=Connection, 3=Performance)
	statsTab int

	// Request metrics
	totalRequests int64
	successCount  int64
	errorCount    int64

	// Connection metrics
	reconnectCount  int
	connectionStart time.Time

	// Performance metrics (ring buffer for avg/P95)
	responseTimes     []time.Duration
	responseTimeIdx   int
	responseTimeCount int
	slowestRequest    time.Duration
	slowestPath       string

	// Selection and detail view
	selectedIndex  int
	detailExpanded bool

	// Clipboard notification
	clipboardMsg     string
	clipboardMsgTime time.Time

	// Request filtering
	filterText    string
	filterEnabled bool

	// Reconnection state (exposed for UI)
	retryCount   int
	retryBackoff time.Duration

	// Deprecation notice (for WebSocket client)
	deprecated bool
}

// NewModel creates a new Bubbletea model
func NewModel(client *Client, server string, localAddr string, localPort int, localHTTPS bool) *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorMustardYellow)

	p := paginator.New()
	p.Type = paginator.Dots
	p.PerPage = 5
	p.ActiveDot = lipgloss.NewStyle().Foreground(colorMustardYellow).Render("•")
	p.InactiveDot = lipgloss.NewStyle().Foreground(colorGray).Render("•")

	now := time.Now()
	return &Model{
		status:          "connecting",
		server:          server,
		localPort:       localPort,
		localAddr:       localAddr,
		localHTTPS:      localHTTPS,
		requests:        make([]RequestLog, 0, 50),
		maxRequests:     50,
		startTime:       now,
		spinner:         s,
		paginator:       p,
		pageSize:        5,
		updateCh:        make(chan tea.Msg, 500),
		client:          client,
		connectionStart: now,
		responseTimes:   make([]time.Duration, 100), // Ring buffer for P95 calculation
		deprecated:      true,                       // WebSocket client is deprecated
	}
}

// NewTCPModel creates a new Bubbletea model for TCP client (not deprecated)
func NewTCPModel() *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorMustardYellow)

	p := paginator.New()
	p.Type = paginator.Dots
	p.PerPage = 5
	p.ActiveDot = lipgloss.NewStyle().Foreground(colorMustardYellow).Render("•")
	p.InactiveDot = lipgloss.NewStyle().Foreground(colorGray).Render("•")

	// WebSocket paginator
	wsP := paginator.New()
	wsP.Type = paginator.Dots
	wsP.PerPage = 5
	wsP.ActiveDot = lipgloss.NewStyle().Foreground(colorMustardYellow).Render("•")
	wsP.InactiveDot = lipgloss.NewStyle().Foreground(colorGray).Render("•")

	now := time.Now()
	return &Model{
		status:           "connecting",
		requests:         make([]RequestLog, 0, 50),
		wsConnections:    make([]WebSocketLog, 0, 50),
		maxWSConnections: 50,
		wsPaginator:      wsP,
		maxRequests:     50,
		startTime:       now,
		spinner:         s,
		paginator:       p,
		pageSize:        5,
		updateCh:        make(chan tea.Msg, 500),
		connectionStart: now,
		responseTimes:   make([]time.Duration, 100),
		deprecated:      false, // TCP client is not deprecated
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
		case "1":
			// Switch to HTTP view
			m.requestView = 0
			return m, nil
		case "2":
			// Switch to WebSocket view
			m.requestView = 1
			return m, nil
		case "tab":
			// Cycle stats tabs (0=Traffic, 1=Requests, 2=Connection, 3=Performance)
			m.statsTab = (m.statsTab + 1) % 4
			return m, nil
		case "shift+tab":
			// Cycle stats tabs backwards
			m.statsTab = (m.statsTab + 3) % 4
			return m, nil
		case "up", "k":
			// Select previous item in current view
			if m.requestView == 0 {
				if m.selectedIndex > 0 {
					m.selectedIndex--
					m.ensureSelectionVisible()
				}
			} else {
				if m.wsSelectedIdx > 0 {
					m.wsSelectedIdx--
					m.ensureWSSelectionVisible()
				}
			}
			return m, nil
		case "down", "j":
			// Select next item in current view
			if m.requestView == 0 {
				if m.selectedIndex < len(m.requests)-1 {
					m.selectedIndex++
					m.ensureSelectionVisible()
				}
			} else {
				if m.wsSelectedIdx < len(m.wsConnections)-1 {
					m.wsSelectedIdx++
					m.ensureWSSelectionVisible()
				}
			}
			return m, nil
		case "enter":
			// Toggle detail view
			m.detailExpanded = !m.detailExpanded
			return m, nil
		case "c":
			// Copy primary URL to clipboard
			if m.publicURL != "" {
				if err := clipboard.WriteAll(m.publicURL); err == nil {
					m.clipboardMsg = "Copied!"
					m.clipboardMsgTime = time.Now()
				} else {
					m.clipboardMsg = "Copy failed"
					m.clipboardMsgTime = time.Now()
				}
			}
			return m, nil
		case "/":
			// Toggle filter mode
			m.filterEnabled = !m.filterEnabled
			if !m.filterEnabled {
				m.filterText = ""
			}
			return m, nil
		case "esc":
			// Clear filter if active
			if m.filterEnabled {
				m.filterEnabled = false
				m.filterText = ""
				return m, nil
			}
		case "backspace":
			// Handle backspace in filter mode
			if m.filterEnabled && len(m.filterText) > 0 {
				m.filterText = m.filterText[:len(m.filterText)-1]
				return m, nil
			}
		default:
			// Handle typing in filter mode
			if m.filterEnabled && len(msg.String()) == 1 {
				m.filterText += msg.String()
				return m, nil
			}
		}
		// Let paginator handle navigation keys (left/right/h/l)
		var cmd tea.Cmd
		m.paginator, cmd = m.paginator.Update(msg)
		return m, cmd

	case StatusUpdateMsg:
		prevStatus := m.status
		m.status = msg.Status
		if msg.Server != "" {
			m.server = msg.Server
		}
		if msg.PublicURL != "" {
			m.publicURL = msg.PublicURL
		}
		if len(msg.Tunnels) > 0 {
			m.tunnels = msg.Tunnels
			// Set primary URL from first tunnel if not already set
			if m.publicURL == "" && len(m.tunnels) > 0 {
				m.publicURL = m.tunnels[0].URL
			}
		}
		if msg.Error != "" {
			m.errorMessage = msg.Error
		}
		// Copy retry info for reconnecting status
		m.retryCount = msg.RetryCount
		m.retryBackoff = msg.RetryBackoff
		// Track reconnections
		if prevStatus == "reconnecting" && msg.Status == "online" {
			m.reconnectCount++
		}
		if msg.Status == "online" && (prevStatus == "connecting" || prevStatus == "reconnecting") {
			m.connectionStart = time.Now()
		}
		// Keep draining channel
		return m, m.waitForUpdates()

	case RequestAddedMsg:
		req := RequestLog{
			ID:        msg.ID,
			Time:      time.Now(),
			Method:    msg.Method,
			Path:      msg.Path,
			Subdomain: msg.Subdomain,
			Pending:   true,
			BytesRecv: msg.BytesRecv,
		}
		m.requests = append(m.requests, req)
		m.totalRequests++
		if len(m.requests) > m.maxRequests {
			m.requests = m.requests[1:]
			// Adjust selectedIndex if needed
			if m.selectedIndex > 0 {
				m.selectedIndex--
			}
		}
		// Update paginator total pages (pass item count, not page count)
		m.paginator.SetTotalPages(len(m.requests))
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(nil)
		// Keep draining channel + fast tick for pending display
		return m, tea.Batch(m.waitForUpdates(), m.fastTick(), cmd)

	case RequestCompletedMsg:
		for i := range m.requests {
			if m.requests[i].ID == msg.ID {
				m.requests[i].StatusCode = msg.StatusCode
				m.requests[i].Duration = msg.Duration
				m.requests[i].Pending = false
				m.requests[i].BytesSent = msg.BytesSent
				// Track slowest request
				if msg.Duration > m.slowestRequest {
					m.slowestRequest = msg.Duration
					m.slowestPath = m.requests[i].Path
				}
				break
			}
		}
		m.bytesSent += msg.BytesSent
		m.bytesRecv += msg.BytesRecv
		// Update success/error counts
		if msg.StatusCode >= 200 && msg.StatusCode < 400 {
			m.successCount++
		} else {
			m.errorCount++
		}
		// Track response times for avg/P95
		m.addResponseTime(msg.Duration)
		// Update paginator total pages (pass item count, not page count)
		m.paginator.SetTotalPages(len(m.requests))
		// Keep draining channel for more completions
		return m, m.waitForUpdates()

	case WebSocketConnectedMsg:
		ws := WebSocketLog{
			ID:        msg.ID,
			Path:      msg.Path,
			Subdomain: msg.Subdomain,
			StartTime: time.Now(),
			Active:    true,
		}
		m.wsConnections = append(m.wsConnections, ws)
		m.totalWSConns++
		m.activeWSConns++
		if len(m.wsConnections) > m.maxWSConnections {
			m.wsConnections = m.wsConnections[1:]
			if m.wsSelectedIdx > 0 {
				m.wsSelectedIdx--
			}
		}
		m.wsPaginator.SetTotalPages(len(m.wsConnections))
		return m, tea.Batch(m.waitForUpdates(), m.fastTick())

	case WebSocketDataMsg:
		for i := range m.wsConnections {
			if m.wsConnections[i].ID == msg.ID {
				m.wsConnections[i].BytesSent += msg.BytesSent
				m.wsConnections[i].BytesRecv += msg.BytesRecv
				break
			}
		}
		m.wsBytesSent += msg.BytesSent
		m.wsBytesRecv += msg.BytesRecv
		return m, m.waitForUpdates()

	case WebSocketClosedMsg:
		for i := range m.wsConnections {
			if m.wsConnections[i].ID == msg.ID {
				m.wsConnections[i].Active = false
				m.wsConnections[i].EndTime = time.Now()
				m.activeWSConns--
				break
			}
		}
		return m, m.waitForUpdates()

	case TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(nil)
		return m, tea.Batch(
			m.tick(),
			m.waitForUpdates(),
			cmd,
		)

	case FastTickMsg:
		// Only continue fast ticking if we have pending requests
		if m.hasPendingRequests() {
			return m, m.fastTick()
		}
		return m, nil

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
		// Show uptime when online
		uptime := time.Since(m.connectionStart)
		uptimeStr := formatDuration(uptime)
		return statusBadgeOnline.Render("● ONLINE") + timeStyle.MarginLeft(2).Render("("+uptimeStr+")")
	case "connecting":
		return statusBadgeConnecting.Render("◉ CONNECTING")
	case "reconnecting":
		// Show retry count and backoff when reconnecting
		retryInfo := fmt.Sprintf("◉ RECONNECTING #%d", m.retryCount)
		if m.retryBackoff > 0 {
			retryInfo += fmt.Sprintf(" (%.0fs)", m.retryBackoff.Seconds())
		}
		return statusBadgeReconnecting.Render(retryInfo)
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

// HTTP status code range constants
const (
	httpStatusSuccessMin  = 200
	httpStatusSuccessMax  = 299
	httpStatusRedirectMax = 399
	httpStatusClientMax   = 499
)

// getStatusCodeStyle returns the appropriate status code style
func getStatusCodeStyle(code int) lipgloss.Style {
	switch {
	case code == 0:
		return statusCodePending // Pending request
	case code >= httpStatusSuccessMin && code <= httpStatusSuccessMax:
		return statusCodeSuccess // 2xx success
	case code > httpStatusSuccessMax && code <= httpStatusRedirectMax:
		return statusCodeRedirect // 3xx redirect
	case code > httpStatusRedirectMax && code <= httpStatusClientMax:
		return statusCodeClientError // 4xx client error
	case code > httpStatusClientMax:
		return statusCodeServerError // 5xx server error
	default:
		return statusCodeSuccess
	}
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

	// Show deprecation notice for WebSocket client
	if m.deprecated {
		content = append(content, "")
		deprecationStyle := lipgloss.NewStyle().
			Foreground(colorMustardYellow).
			Bold(true)
		content = append(content, deprecationStyle.Render("⚠ DEPRECATED: Use 'digit-link --tcp' for the new multi-forward client"))
	}

	content = append(content, "")
	content = append(content, labelStyle.Render("Version")+valueStyle.MarginLeft(2).Render("1.0.0"))
	content = append(content, labelStyle.Render("Server")+valueStyle.MarginLeft(2).Render(m.server))

	// Forwarding section - show all tunnels if multi-forward, otherwise single line
	if len(m.tunnels) > 1 {
		// Multi-tunnel display
		content = append(content, labelStyle.Render("Forwarding"))
		for _, t := range m.tunnels {
			localScheme := "http"
			if t.LocalHTTPS {
				localScheme = "https"
			}
			line := "  " + urlPublicStyle.Render(t.URL) +
				" → " +
				urlLocalStyle.Render(fmt.Sprintf("%s://localhost:%d", localScheme, t.LocalPort))
			content = append(content, line)
		}
	} else {
		// Single tunnel display (backward compatible)
		forwardingText := m.publicURL
		if forwardingText == "" {
			forwardingText = "..."
		}
		localScheme := "http"
		if m.localHTTPS {
			localScheme = "https"
		}
		forwarding := urlPublicStyle.Render(forwardingText) +
			" → " +
			urlLocalStyle.Render(fmt.Sprintf("%s://%s:%d", localScheme, m.localAddr, m.localPort))
		content = append(content, labelStyle.Render("Forwarding")+valueStyle.MarginLeft(2).Render(forwarding))
	}

	// Show clipboard notification (fades after 2 seconds)
	if m.clipboardMsg != "" && time.Since(m.clipboardMsgTime) < 2*time.Second {
		clipStyle := lipgloss.NewStyle().Foreground(colorGreen).Italic(true)
		content = append(content, clipStyle.Render("  "+m.clipboardMsg))
	}
	content = append(content, "")

	// Stats Section with tabs
	content = append(content, m.renderStatsSection())
	content = append(content, "")
	content = append(content, timeStyle.Render(strings.Repeat("─", 80)))

	// Request view tabs (HTTP / WebSocket)
	content = append(content, m.renderViewTabs())
	content = append(content, "")

	// Render either HTTP or WebSocket view based on requestView
	if m.requestView == 1 {
		content = append(content, m.renderWebSocketView()...)
	} else {
		content = append(content, m.renderHTTPView()...)
	}

	// Add detail view if expanded
	if m.detailExpanded {
		content = append(content, m.renderDetailView())
	}

	// Combine everything into one box (no border)
	boxContent := mainBoxStyle.Render(strings.Join(content, "\n"))

	return lipgloss.JoinVertical(lipgloss.Left, header, "", boxContent)
}

// renderViewTabs renders the HTTP/WebSocket tab bar
func (m *Model) renderViewTabs() string {
	httpTab := " HTTP "
	wsTab := fmt.Sprintf(" WebSocket (%d) ", m.activeWSConns)

	if m.requestView == 0 {
		httpTab = labelStyle.Render("[HTTP]")
		wsTab = timeStyle.Render(wsTab)
	} else {
		httpTab = timeStyle.Render(httpTab)
		wsTab = labelStyle.Render(fmt.Sprintf("[WebSocket (%d)]", m.activeWSConns))
	}

	return lipgloss.JoinHorizontal(lipgloss.Left,
		httpTab,
		timeStyle.Render(" | "),
		wsTab,
		timeStyle.Render("  (1: HTTP, 2: WebSocket)"),
	)
}

// renderHTTPView renders the HTTP requests view
func (m *Model) renderHTTPView() []string {
	var content []string

	// Filter bar (if enabled)
	if m.filterEnabled {
		filterStyle := lipgloss.NewStyle().
			Foreground(colorMustardYellow).
			Bold(true)
		filterBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorMustardYellow).
			Padding(0, 1)
		filterText := m.filterText
		if filterText == "" {
			filterText = timeStyle.Render("type to filter...")
		}
		content = append(content, filterStyle.Render("Filter: ")+filterBox.Render(filterText))
		content = append(content, "")
	}

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

	// Get filtered requests (if filter is active)
	filteredRequests := m.getFilteredRequests()
	requestCount := len(filteredRequests)
	m.paginator.SetTotalPages(requestCount)
	totalPages := m.paginator.TotalPages
	if totalPages == 0 {
		totalPages = 1
	}

	// Calculate start/end indices for the current page (in reversed order)
	start, end := m.paginator.GetSliceBounds(requestCount)
	pageItemCount := end - start
	if pageItemCount < 0 {
		pageItemCount = 0
	}

	// Display paginated requests (always show 5 rows, most recent first)
	for i := 0; i < m.pageSize; i++ {
		if i < pageItemCount {
			// Reverse index for most recent first
			actualIndex := start + i
			reverseIdx := requestCount - 1 - actualIndex
			if reverseIdx < 0 || reverseIdx >= requestCount {
				continue
			}
			req := filteredRequests[reverseIdx]

			// Build display path with subdomain prefix for multi-tunnel
			path := req.Path
			if len(m.tunnels) > 1 && req.Subdomain != "" {
				// Show subdomain prefix for multi-tunnel mode
				path = "[" + req.Subdomain + "] " + path
			}
			if len(path) > 40 {
				path = path[:37] + "..."
			}

			// Determine row style based on selection and pending state
			var rowStyle lipgloss.Style
			isSelected := actualIndex == m.selectedIndex
			if isSelected {
				rowStyle = lipgloss.NewStyle().
					Background(colorMustardYellow).
					Foreground(lipgloss.Color("232")).
					Padding(0, 1)
			} else if req.Pending {
				rowStyle = requestRowPendingStyle
			} else {
				rowStyle = requestRowStyle
			}

			if req.Pending {
				// Show pending request with spinner + elapsed time
				elapsed := time.Since(req.Time)
				elapsedStr := fmt.Sprintf("%.1fs", elapsed.Seconds())
				pendingIndicator := m.spinner.View() + " " + elapsedStr

				row := lipgloss.JoinHorizontal(lipgloss.Left,
					lipgloss.NewStyle().Width(10).Render(timeStyle.Render(req.Time.Format("15:04:05"))),
					lipgloss.NewStyle().Width(9).Render(getMethodBadge(req.Method)),
					lipgloss.NewStyle().Width(42).Render(valueStyle.Render(path)),
					lipgloss.NewStyle().Width(18).Render(timeStyle.Render(pendingIndicator)),
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

	// Add pagination and help
	content = append(content, "")
	if totalPages > 1 {
		paginationText := lipgloss.NewStyle().
			Foreground(colorGray).
			Render(fmt.Sprintf("Page %d of %d", m.paginator.Page+1, totalPages))
		content = append(content, paginationText)
		content = append(content, m.paginator.View())
	}
	// Help text
	var helpText string
	if m.filterEnabled {
		helpText = timeStyle.Render("Type to filter | Esc: clear filter | Enter: details | q: quit")
	} else {
		helpText = timeStyle.Render("Tab: stats | ↑↓: select | Enter: details | c: copy URL | /: filter | q: quit")
	}
	content = append(content, helpText)

	return content
}

// renderWebSocketView renders the WebSocket connections view
func (m *Model) renderWebSocketView() []string {
	var content []string

	// Table header
	headerRow := lipgloss.JoinHorizontal(lipgloss.Left,
		lipgloss.NewStyle().Width(10).Render(timeStyle.Render("Time")),
		lipgloss.NewStyle().Width(8).Render(labelStyle.Render("Status")),
		lipgloss.NewStyle().Width(36).Render(labelStyle.Render("Path")),
		lipgloss.NewStyle().Width(12).Render(labelStyle.Render("Duration")),
		lipgloss.NewStyle().Width(14).Render(labelStyle.Render("Data")),
	)
	content = append(content, headerRow)
	content = append(content, timeStyle.Render(strings.Repeat("─", 80)))

	wsCount := len(m.wsConnections)
	m.wsPaginator.SetTotalPages(wsCount)
	totalPages := m.wsPaginator.TotalPages
	if totalPages == 0 {
		totalPages = 1
	}

	// Calculate start/end indices for the current page (in reversed order)
	start, end := m.wsPaginator.GetSliceBounds(wsCount)
	pageItemCount := end - start
	if pageItemCount < 0 {
		pageItemCount = 0
	}

	// WebSocket badge style (purple)
	wsBadgeActive := lipgloss.NewStyle().
		Foreground(lipgloss.Color("232")).
		Background(lipgloss.Color("141")). // Purple
		Bold(true).
		Padding(0, 1).
		Width(6).
		Align(lipgloss.Center)
	wsBadgeClosed := lipgloss.NewStyle().
		Foreground(colorWhite).
		Background(colorDarkGray).
		Bold(true).
		Padding(0, 1).
		Width(6).
		Align(lipgloss.Center)

	// Display paginated WebSocket connections (most recent first)
	for i := 0; i < m.pageSize; i++ {
		if i < pageItemCount {
			actualIndex := start + i
			reverseIdx := wsCount - 1 - actualIndex
			if reverseIdx < 0 || reverseIdx >= wsCount {
				continue
			}
			ws := m.wsConnections[reverseIdx]

			// Build display path with subdomain prefix for multi-tunnel
			path := ws.Path
			if len(m.tunnels) > 1 && ws.Subdomain != "" {
				path = "[" + ws.Subdomain + "] " + path
			}
			if len(path) > 34 {
				path = path[:31] + "..."
			}

			// Determine row style based on selection
			var rowStyle lipgloss.Style
			isSelected := actualIndex == m.wsSelectedIdx
			if isSelected {
				rowStyle = lipgloss.NewStyle().
					Background(colorMustardYellow).
					Foreground(lipgloss.Color("232")).
					Padding(0, 1)
			} else {
				rowStyle = requestRowStyle
			}

			// Status badge
			var statusBadge string
			if ws.Active {
				statusBadge = wsBadgeActive.Render("LIVE")
			} else {
				statusBadge = wsBadgeClosed.Render("DONE")
			}

			// Duration
			var duration time.Duration
			if ws.Active {
				duration = time.Since(ws.StartTime)
			} else {
				duration = ws.EndTime.Sub(ws.StartTime)
			}

			// Data transferred
			dataStr := fmt.Sprintf("↑%s ↓%s", formatBytes(ws.BytesSent), formatBytes(ws.BytesRecv))

			row := lipgloss.JoinHorizontal(lipgloss.Left,
				lipgloss.NewStyle().Width(10).Render(timeStyle.Render(ws.StartTime.Format("15:04:05"))),
				lipgloss.NewStyle().Width(8).Render(statusBadge),
				lipgloss.NewStyle().Width(36).Render(valueStyle.Render(path)),
				lipgloss.NewStyle().Width(12).Render(timeStyle.Render(formatDuration(duration))),
				lipgloss.NewStyle().Width(14).Render(timeStyle.Render(dataStr)),
			)
			content = append(content, rowStyle.Render(row))
		} else {
			// Empty row placeholder
			emptyRow := lipgloss.JoinHorizontal(lipgloss.Left,
				lipgloss.NewStyle().Width(10).Render(""),
				lipgloss.NewStyle().Width(8).Render(""),
				lipgloss.NewStyle().Width(36).Render(""),
				lipgloss.NewStyle().Width(12).Render(""),
				lipgloss.NewStyle().Width(14).Render(""),
			)
			content = append(content, requestRowStyle.Render(emptyRow))
		}
	}

	// Add pagination and help
	content = append(content, "")
	if totalPages > 1 {
		paginationText := lipgloss.NewStyle().
			Foreground(colorGray).
			Render(fmt.Sprintf("Page %d of %d", m.wsPaginator.Page+1, totalPages))
		content = append(content, paginationText)
		content = append(content, m.wsPaginator.View())
	}

	// WebSocket stats summary
	statsLine := timeStyle.Render(fmt.Sprintf("Total: %d | Active: %d | ↑%s ↓%s",
		m.totalWSConns, m.activeWSConns,
		formatBytes(m.wsBytesSent), formatBytes(m.wsBytesRecv)))
	content = append(content, statsLine)

	// Help text
	helpText := timeStyle.Render("Tab: stats | ↑↓: select | c: copy URL | q: quit")
	content = append(content, helpText)

	return content
}

// ensureWSSelectionVisible ensures the selected WebSocket is visible in the paginator
func (m *Model) ensureWSSelectionVisible() {
	pageStart := m.wsPaginator.Page * m.pageSize
	pageEnd := pageStart + m.pageSize
	if m.wsSelectedIdx < pageStart {
		m.wsPaginator.Page = m.wsSelectedIdx / m.pageSize
	} else if m.wsSelectedIdx >= pageEnd {
		m.wsPaginator.Page = m.wsSelectedIdx / m.pageSize
	}
}

// tick returns a command that sends a TickMsg after 1 second
func (m *Model) tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// waitForUpdates waits for a message from the update channel
func (m *Model) waitForUpdates() tea.Cmd {
	return func() tea.Msg {
		// Use a reusable timer to avoid memory leaks from time.After
		timer := time.NewTimer(100 * time.Millisecond)
		defer timer.Stop() // Critical: stop timer to prevent leak

		select {
		case msg := <-m.updateCh:
			return msg
		case <-timer.C:
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

// fastTick returns a command that sends a FastTickMsg after 100ms (for pending timer updates)
func (m *Model) fastTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return FastTickMsg(t)
	})
}

// getFilteredRequests returns requests filtered by the current filter text
func (m *Model) getFilteredRequests() []RequestLog {
	if !m.filterEnabled || m.filterText == "" {
		return m.requests
	}

	filter := strings.ToLower(m.filterText)
	filtered := make([]RequestLog, 0)
	for _, req := range m.requests {
		// Match against path, method, subdomain, or status code
		if strings.Contains(strings.ToLower(req.Path), filter) ||
			strings.Contains(strings.ToLower(req.Method), filter) ||
			strings.Contains(strings.ToLower(req.Subdomain), filter) ||
			strings.Contains(fmt.Sprintf("%d", req.StatusCode), filter) {
			filtered = append(filtered, req)
		}
	}
	return filtered
}

// hasPendingRequests checks if any requests are still pending
// Also auto-expires stale pending requests (older than 5 minutes)
func (m *Model) hasPendingRequests() bool {
	now := time.Now()
	const maxPendingAge = 5 * time.Minute

	hasPending := false
	for i := range m.requests {
		if m.requests[i].Pending {
			// Auto-expire stale pending requests
			if now.Sub(m.requests[i].Time) > maxPendingAge {
				m.requests[i].Pending = false
				m.requests[i].StatusCode = 504 // Gateway Timeout
				m.requests[i].Duration = maxPendingAge
			} else {
				hasPending = true
			}
		}
	}
	return hasPending
}

// addResponseTime adds a duration to the ring buffer for avg/P95 calculation
func (m *Model) addResponseTime(d time.Duration) {
	m.responseTimes[m.responseTimeIdx] = d
	m.responseTimeIdx = (m.responseTimeIdx + 1) % len(m.responseTimes)
	if m.responseTimeCount < len(m.responseTimes) {
		m.responseTimeCount++
	}
}

// calculateAvgResponseTime calculates the average response time
func (m *Model) calculateAvgResponseTime() time.Duration {
	if m.responseTimeCount == 0 {
		return 0
	}
	var sum time.Duration
	for i := 0; i < m.responseTimeCount; i++ {
		sum += m.responseTimes[i]
	}
	return sum / time.Duration(m.responseTimeCount)
}

// calculateP95ResponseTime calculates the 95th percentile response time
func (m *Model) calculateP95ResponseTime() time.Duration {
	if m.responseTimeCount == 0 {
		return 0
	}
	// Copy and sort
	sorted := make([]time.Duration, m.responseTimeCount)
	copy(sorted, m.responseTimes[:m.responseTimeCount])
	// Simple insertion sort for small arrays
	for i := 1; i < len(sorted); i++ {
		key := sorted[i]
		j := i - 1
		for j >= 0 && sorted[j] > key {
			sorted[j+1] = sorted[j]
			j--
		}
		sorted[j+1] = key
	}
	// 95th percentile
	idx := int(float64(m.responseTimeCount) * 0.95)
	if idx >= m.responseTimeCount {
		idx = m.responseTimeCount - 1
	}
	return sorted[idx]
}

// ensureSelectionVisible adjusts the paginator page so selectedIndex is visible
func (m *Model) ensureSelectionVisible() {
	if len(m.requests) == 0 {
		return
	}
	pageStart := m.paginator.Page * m.pageSize
	pageEnd := pageStart + m.pageSize
	if m.selectedIndex < pageStart {
		m.paginator.Page = m.selectedIndex / m.pageSize
	} else if m.selectedIndex >= pageEnd {
		m.paginator.Page = m.selectedIndex / m.pageSize
	}
}

// Stats tab names
var statsTabNames = []string{"Traffic", "Requests", "Connection", "Performance"}

// renderStatsSection renders the stats section with tab bar
func (m *Model) renderStatsSection() string {
	// Tab bar
	var tabs []string
	for i, name := range statsTabNames {
		if i == m.statsTab {
			tabs = append(tabs, labelStyle.Render("["+name+"]"))
		} else {
			tabs = append(tabs, timeStyle.Render(" "+name+" "))
		}
	}
	tabBar := lipgloss.JoinHorizontal(lipgloss.Left, tabs...)
	tabHint := timeStyle.Render("  (Tab to switch)")

	// Content based on selected tab
	var statsContent string
	switch m.statsTab {
	case 0:
		statsContent = m.renderTrafficStats()
	case 1:
		statsContent = m.renderRequestStats()
	case 2:
		statsContent = m.renderConnectionStats()
	case 3:
		statsContent = m.renderPerformanceStats()
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Left, tabBar, tabHint),
		"",
		statsContent,
	)
}

// renderTrafficStats renders the Traffic tab content
func (m *Model) renderTrafficStats() string {
	uptime := time.Since(m.startTime)

	row1 := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(26).Render(
			labelStyle.Render("Uptime")+"\n"+
				valueStyle.Render(formatUptime(uptime)),
		),
		lipgloss.NewStyle().Width(26).Render(
			labelStyle.Render("Sent")+"\n"+
				valueStyle.Render(formatBytes(m.bytesSent)),
		),
		lipgloss.NewStyle().Width(26).Render(
			labelStyle.Render("Received")+"\n"+
				valueStyle.Render(formatBytes(m.bytesRecv)),
		),
	)

	// Rate row
	var row2 string
	if uptime.Seconds() > 0 {
		sentRate := float64(m.bytesSent) / uptime.Seconds()
		recvRate := float64(m.bytesRecv) / uptime.Seconds()
		row2 = lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(26).Render(""),
			lipgloss.NewStyle().Width(26).Render(
				timeStyle.Render(fmt.Sprintf("↑ %s/s", formatBytes(int64(sentRate)))),
			),
			lipgloss.NewStyle().Width(26).Render(
				timeStyle.Render(fmt.Sprintf("↓ %s/s", formatBytes(int64(recvRate)))),
			),
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left, row1, row2)
}

// renderRequestStats renders the Requests tab content
func (m *Model) renderRequestStats() string {
	uptime := time.Since(m.startTime)
	reqPerMin := float64(0)
	if uptime.Minutes() >= 1 {
		reqPerMin = float64(m.totalRequests) / uptime.Minutes()
	} else if uptime.Seconds() > 0 {
		reqPerMin = float64(m.totalRequests) / uptime.Minutes()
	}

	return lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(20).Render(
			labelStyle.Render("Total")+"\n"+
				valueStyle.Render(fmt.Sprintf("%d", m.totalRequests)),
		),
		lipgloss.NewStyle().Width(20).Render(
			labelStyle.Render("Success")+"\n"+
				statusCodeSuccess.Render(fmt.Sprintf("%d", m.successCount)),
		),
		lipgloss.NewStyle().Width(20).Render(
			labelStyle.Render("Errors")+"\n"+
				statusCodeServerError.Render(fmt.Sprintf("%d", m.errorCount)),
		),
		lipgloss.NewStyle().Width(20).Render(
			labelStyle.Render("Req/min")+"\n"+
				valueStyle.Render(fmt.Sprintf("%.1f", reqPerMin)),
		),
	)
}

// renderConnectionStats renders the Connection tab content
func (m *Model) renderConnectionStats() string {
	connDuration := time.Since(m.connectionStart)

	return lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(26).Render(
			labelStyle.Render("Reconnects")+"\n"+
				valueStyle.Render(fmt.Sprintf("%d", m.reconnectCount)),
		),
		lipgloss.NewStyle().Width(26).Render(
			labelStyle.Render("Connected")+"\n"+
				valueStyle.Render(formatUptime(connDuration)),
		),
		lipgloss.NewStyle().Width(26).Render(
			labelStyle.Render("Status")+"\n"+
				valueStyle.Render(m.status),
		),
	)
}

// renderPerformanceStats renders the Performance tab content
func (m *Model) renderPerformanceStats() string {
	avgTime := m.calculateAvgResponseTime()
	p95Time := m.calculateP95ResponseTime()

	row1 := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(26).Render(
			labelStyle.Render("Avg Response")+"\n"+
				valueStyle.Render(formatDuration(avgTime)),
		),
		lipgloss.NewStyle().Width(26).Render(
			labelStyle.Render("P95 Response")+"\n"+
				valueStyle.Render(formatDuration(p95Time)),
		),
		lipgloss.NewStyle().Width(26).Render(
			labelStyle.Render("Slowest")+"\n"+
				valueStyle.Render(formatDuration(m.slowestRequest)),
		),
	)

	// Show slowest request path if available
	var row2 string
	if m.slowestPath != "" {
		path := m.slowestPath
		if len(path) > 60 {
			path = path[:57] + "..."
		}
		row2 = timeStyle.Render("Slowest: " + path)
	}

	return lipgloss.JoinVertical(lipgloss.Left, row1, row2)
}

// renderDetailView renders the expanded detail view for the selected request
func (m *Model) renderDetailView() string {
	if len(m.requests) == 0 || m.selectedIndex >= len(m.requests) {
		return ""
	}

	// Get selected request (reversed order - most recent first)
	reversedIndex := len(m.requests) - 1 - m.selectedIndex
	if reversedIndex < 0 || reversedIndex >= len(m.requests) {
		return ""
	}
	req := m.requests[reversedIndex]

	var lines []string
	lines = append(lines, "")
	lines = append(lines, timeStyle.Render(strings.Repeat("─", 80)))
	lines = append(lines, labelStyle.Render("Request Details"))
	lines = append(lines, "")

	// Full path
	lines = append(lines, labelStyle.Render("Path: ")+valueStyle.Render(req.Path))

	// Method and status
	statusStr := "Pending..."
	if !req.Pending {
		statusStyle := getStatusCodeStyle(req.StatusCode)
		statusStr = statusStyle.Render(fmt.Sprintf("%d", req.StatusCode))
	}
	lines = append(lines, labelStyle.Render("Method: ")+getMethodBadge(req.Method)+"  "+labelStyle.Render("Status: ")+statusStr)

	// Timing
	if req.Pending {
		elapsed := time.Since(req.Time)
		lines = append(lines, labelStyle.Render("Elapsed: ")+timeStyle.Render(fmt.Sprintf("%.2fs (pending)", elapsed.Seconds())))
	} else {
		lines = append(lines, labelStyle.Render("Duration: ")+valueStyle.Render(formatDuration(req.Duration)))
	}

	// Data sizes
	lines = append(lines, labelStyle.Render("Data: ")+
		valueStyle.Render(fmt.Sprintf("↓ %s", formatBytes(req.BytesRecv)))+
		"  "+
		valueStyle.Render(fmt.Sprintf("↑ %s", formatBytes(req.BytesSent))))

	lines = append(lines, "")
	lines = append(lines, timeStyle.Render("(Press Enter to close)"))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
