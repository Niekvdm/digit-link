package client

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/niekvdm/digit-link/internal/tunnel"
)

// SetupView represents different setup screen states
type SetupView int

const (
	SetupViewMain SetupView = iota
	SetupViewAddForward
)

// SetupModel holds the state for the setup TUI
type SetupModel struct {
	// View state
	view SetupView

	// Main view inputs
	serverInput textinput.Model
	tokenInput  textinput.Model

	// Add forward view inputs
	subdomainInput textinput.Model
	portInput      textinput.Model
	localHTTPS     bool // HTTPS for local forwarding (in add forward view)

	// Forwards list
	forwards       []tunnel.ForwardConfig
	selectedFwd    int
	primaryFwdIdx  int // Index of primary forward

	// Focus management (main view)
	focusIndex int // 0=server, 1=token, 2=forwards list, 3=add button, 4=connect button

	// Error message
	errorMsg string

	// Callback when setup is complete
	onConnect func(server, token string, forwards []tunnel.ForwardConfig, insecure bool)

	// Additional options
	insecure bool
}

// NewSetupModel creates a new setup model
func NewSetupModel() *SetupModel {
	// Server input
	serverInput := textinput.New()
	serverInput.Placeholder = "link.digit.zone"
	serverInput.CharLimit = 100
	serverInput.Width = 40
	serverInput.Prompt = ""
	serverInput.Focus()

	// Token input
	tokenInput := textinput.New()
	tokenInput.Placeholder = "Enter your token"
	tokenInput.CharLimit = 200
	tokenInput.Width = 40
	tokenInput.Prompt = ""
	tokenInput.EchoMode = textinput.EchoPassword
	tokenInput.EchoCharacter = '*'

	// Subdomain input
	subdomainInput := textinput.New()
	subdomainInput.Placeholder = "myapp"
	subdomainInput.CharLimit = 50
	subdomainInput.Width = 30
	subdomainInput.Prompt = ""

	// Port input
	portInput := textinput.New()
	portInput.Placeholder = "3000"
	portInput.CharLimit = 5
	portInput.Width = 10
	portInput.Prompt = ""

	return &SetupModel{
		view:           SetupViewMain,
		serverInput:    serverInput,
		tokenInput:     tokenInput,
		subdomainInput: subdomainInput,
		portInput:      portInput,
		forwards:       make([]tunnel.ForwardConfig, 0),
		selectedFwd:    -1,
		primaryFwdIdx:  0,
		focusIndex:     0,
	}
}

// SetOnConnect sets the callback for when setup is complete
func (m *SetupModel) SetOnConnect(fn func(server, token string, forwards []tunnel.ForwardConfig, insecure bool)) {
	m.onConnect = fn
}

// LoadSavedConfig loads saved configuration and populates the model
func (m *SetupModel) LoadSavedConfig() error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	if cfg == nil {
		return nil // No saved config
	}

	// Populate fields
	if cfg.Server != "" {
		m.serverInput.SetValue(cfg.Server)
	}
	if cfg.Token != "" {
		m.tokenInput.SetValue(cfg.Token)
	}
	m.forwards = cfg.Forwards
	m.insecure = cfg.Insecure

	// Find primary forward index
	for i, fwd := range m.forwards {
		if fwd.Primary {
			m.primaryFwdIdx = i
			break
		}
	}

	// If forwards exist, start with focus on connect button
	if len(m.forwards) > 0 {
		m.focusIndex = 4
		m.serverInput.Blur()
		m.selectedFwd = 0
	}

	return nil
}

// SaveCurrentConfig saves the current configuration to disk
func (m *SetupModel) SaveCurrentConfig() error {
	server := strings.TrimSpace(m.serverInput.Value())
	if server == "" {
		server = "link.digit.zone"
	}

	// Update primary flag on forwards
	forwards := make([]tunnel.ForwardConfig, len(m.forwards))
	copy(forwards, m.forwards)
	for i := range forwards {
		forwards[i].Primary = i == m.primaryFwdIdx
	}

	return SaveConfig(SavedConfig{
		Server:   server,
		Token:    strings.TrimSpace(m.tokenInput.Value()),
		Forwards: forwards,
		Insecure: m.insecure,
	})
}

// Init initializes the model
func (m *SetupModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (m *SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.view {
	case SetupViewMain:
		return m.updateMain(msg)
	case SetupViewAddForward:
		return m.updateAddForward(msg)
	}
	return m, nil
}

// updateMain handles main setup view
func (m *SetupModel) updateMain(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.errorMsg = "" // Clear error on any key

		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "tab", "down":
			m.focusNext()
			return m, nil

		case "shift+tab", "up":
			m.focusPrev()
			return m, nil

		case "enter":
			return m.handleEnter()

		case "delete", "backspace":
			// Delete selected forward
			if m.focusIndex == 2 && len(m.forwards) > 0 && m.selectedFwd >= 0 {
				m.forwards = append(m.forwards[:m.selectedFwd], m.forwards[m.selectedFwd+1:]...)
				if m.selectedFwd >= len(m.forwards) {
					m.selectedFwd = len(m.forwards) - 1
				}
				// Adjust primary if needed
				if m.primaryFwdIdx >= len(m.forwards) {
					m.primaryFwdIdx = max(0, len(m.forwards)-1)
				}
				return m, nil
			}

		case "p":
			// Toggle primary on selected forward
			if m.focusIndex == 2 && len(m.forwards) > 0 && m.selectedFwd >= 0 {
				m.primaryFwdIdx = m.selectedFwd
				return m, nil
			}

		case "i":
			// Toggle insecure mode
			m.insecure = !m.insecure
			return m, nil
		}

	}

	// Update focused input
	var cmd tea.Cmd
	switch m.focusIndex {
	case 0:
		m.serverInput, cmd = m.serverInput.Update(msg)
	case 1:
		m.tokenInput, cmd = m.tokenInput.Update(msg)
	}

	return m, cmd
}

// updateAddForward handles add forward view
func (m *SetupModel) updateAddForward(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.errorMsg = "" // Clear error on any key

		switch msg.String() {
		case "esc":
			// Go back to main view
			m.view = SetupViewMain
			m.focusIndex = 3 // Focus add button
			return m, nil

		case "tab", "down":
			// Toggle between subdomain and port
			if m.subdomainInput.Focused() {
				m.subdomainInput.Blur()
				m.portInput.Focus()
			} else {
				m.portInput.Blur()
				m.subdomainInput.Focus()
			}
			return m, nil

		case "shift+tab", "up":
			// Toggle between port and subdomain
			if m.portInput.Focused() {
				m.portInput.Blur()
				m.subdomainInput.Focus()
			} else {
				m.subdomainInput.Blur()
				m.portInput.Focus()
			}
			return m, nil

		case "enter":
			return m.handleAddForward()

		case "h":
			// Toggle local HTTPS
			m.localHTTPS = !m.localHTTPS
			return m, nil
		}
	}

	// Update focused input
	var cmd tea.Cmd
	if m.subdomainInput.Focused() {
		m.subdomainInput, cmd = m.subdomainInput.Update(msg)
	} else {
		m.portInput, cmd = m.portInput.Update(msg)
	}

	return m, cmd
}

// focusNext moves focus to the next element
func (m *SetupModel) focusNext() {
	m.serverInput.Blur()
	m.tokenInput.Blur()

	maxFocus := 4
	if len(m.forwards) == 0 {
		// Skip forwards list if empty
		if m.focusIndex == 1 {
			m.focusIndex = 3
		} else {
			m.focusIndex++
		}
	} else {
		m.focusIndex++
	}

	if m.focusIndex > maxFocus {
		m.focusIndex = 0
	}

	// Handle forwards list selection
	if m.focusIndex == 2 {
		if m.selectedFwd < 0 {
			m.selectedFwd = 0
		}
	}

	m.updateInputFocus()
}

// focusPrev moves focus to the previous element
func (m *SetupModel) focusPrev() {
	m.serverInput.Blur()
	m.tokenInput.Blur()

	if len(m.forwards) == 0 {
		// Skip forwards list if empty
		if m.focusIndex == 3 {
			m.focusIndex = 1
		} else {
			m.focusIndex--
		}
	} else {
		m.focusIndex--
	}

	if m.focusIndex < 0 {
		m.focusIndex = 4
	}

	// Handle forwards list selection
	if m.focusIndex == 2 {
		if m.selectedFwd < 0 {
			m.selectedFwd = len(m.forwards) - 1
		}
	}

	m.updateInputFocus()
}

// updateInputFocus updates input focus state
func (m *SetupModel) updateInputFocus() {
	switch m.focusIndex {
	case 0:
		m.serverInput.Focus()
	case 1:
		m.tokenInput.Focus()
	}
}

// handleEnter handles enter key press
func (m *SetupModel) handleEnter() (tea.Model, tea.Cmd) {
	switch m.focusIndex {
	case 2:
		// In forwards list - could show details or toggle primary
		if len(m.forwards) > 0 && m.selectedFwd >= 0 {
			m.primaryFwdIdx = m.selectedFwd
		}
		return m, nil

	case 3:
		// Add forward button
		m.view = SetupViewAddForward
		m.subdomainInput.SetValue("")
		m.portInput.SetValue("")
		m.localHTTPS = false // Reset for new forward
		m.subdomainInput.Focus()
		m.portInput.Blur()
		return m, nil

	case 4:
		// Connect button
		return m.handleConnect()
	}

	// Move to next field
	m.focusNext()
	return m, nil
}

// handleAddForward adds a new forward
func (m *SetupModel) handleAddForward() (tea.Model, tea.Cmd) {
	subdomain := strings.TrimSpace(m.subdomainInput.Value())
	portStr := strings.TrimSpace(m.portInput.Value())

	// Validate
	if subdomain == "" {
		m.errorMsg = "Subdomain is required"
		return m, nil
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		m.errorMsg = "Invalid port number (1-65535)"
		return m, nil
	}

	// Check for duplicate subdomain
	for _, fwd := range m.forwards {
		if fwd.Subdomain == subdomain {
			m.errorMsg = "Subdomain already exists"
			return m, nil
		}
	}

	// Add forward
	m.forwards = append(m.forwards, tunnel.ForwardConfig{
		Subdomain:  subdomain,
		LocalPort:  port,
		LocalHTTPS: m.localHTTPS,
		Primary:    len(m.forwards) == 0, // First one is primary
	})

	// Go back to main view
	m.view = SetupViewMain
	m.focusIndex = 2 // Focus forwards list
	m.selectedFwd = len(m.forwards) - 1

	return m, nil
}

// handleConnect validates and triggers connection
func (m *SetupModel) handleConnect() (tea.Model, tea.Cmd) {
	server := strings.TrimSpace(m.serverInput.Value())
	if server == "" {
		server = "link.digit.zone"
	}

	token := strings.TrimSpace(m.tokenInput.Value())
	if token == "" {
		m.errorMsg = "Token is required"
		return m, nil
	}

	if len(m.forwards) == 0 {
		m.errorMsg = "At least one forward is required"
		return m, nil
	}

	// Update primary flag on forwards
	for i := range m.forwards {
		m.forwards[i].Primary = i == m.primaryFwdIdx
	}

	// Save config before connecting
	_ = m.SaveCurrentConfig() // Ignore errors, saving is best-effort

	// Trigger callback
	if m.onConnect != nil {
		m.onConnect(server, token, m.forwards, m.insecure)
	}

	return m, tea.Quit
}

// View renders the UI
func (m *SetupModel) View() string {
	switch m.view {
	case SetupViewMain:
		return m.viewMain()
	case SetupViewAddForward:
		return m.viewAddForward()
	}
	return ""
}

// viewMain renders the main setup view
func (m *SetupModel) viewMain() string {
	var b strings.Builder

	// Header
	b.WriteString(headerTitleStyle.Render("digit-link setup"))
	b.WriteString("\n")
	b.WriteString(headerSubtextStyle.Render("Configure your tunnel connection"))
	b.WriteString("\n\n")

	// Server input
	serverLabel := "Server"
	if m.focusIndex == 0 {
		serverLabel = labelStyle.Render("▶ Server")
	} else {
		serverLabel = timeStyle.Render("  Server")
	}
	b.WriteString(serverLabel + "\n")
	serverStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.getBorderColor(0)).
		Padding(0, 1).
		Width(44)
	b.WriteString(serverStyle.Render(m.serverInput.View()))
	b.WriteString("\n\n")

	// Token input
	tokenLabel := "Token"
	if m.focusIndex == 1 {
		tokenLabel = labelStyle.Render("▶ Token")
	} else {
		tokenLabel = timeStyle.Render("  Token")
	}
	b.WriteString(tokenLabel + "\n")
	tokenStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.getBorderColor(1)).
		Padding(0, 1).
		Width(44)
	b.WriteString(tokenStyle.Render(m.tokenInput.View()))
	b.WriteString("\n\n")

	// Forwards list
	fwdLabel := "Forwards"
	if m.focusIndex == 2 {
		fwdLabel = labelStyle.Render("▶ Forwards")
	} else {
		fwdLabel = timeStyle.Render("  Forwards")
	}
	b.WriteString(fwdLabel + "\n")

	fwdBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.getBorderColor(2)).
		Padding(0, 1).
		Width(44)

	var fwdContent strings.Builder
	if len(m.forwards) == 0 {
		fwdContent.WriteString(timeStyle.Render("No forwards configured"))
	} else {
		for i, fwd := range m.forwards {
			proto := "http"
			if fwd.LocalHTTPS {
				proto = "https"
			}
			line := fmt.Sprintf("%s://:%d → %s", proto, fwd.LocalPort, fwd.Subdomain)
			if i == m.primaryFwdIdx {
				line += " ★"
			}

			if m.focusIndex == 2 && i == m.selectedFwd {
				line = urlPublicStyle.Render("▶ " + line)
			} else {
				line = valueStyle.Render("  " + line)
			}
			fwdContent.WriteString(line)
			if i < len(m.forwards)-1 {
				fwdContent.WriteString("\n")
			}
		}
	}
	b.WriteString(fwdBoxStyle.Render(fwdContent.String()))
	b.WriteString("\n\n")

	// Buttons row
	addBtnStyle := m.getButtonStyle(3)
	connectBtnStyle := m.getButtonStyle(4)

	addBtn := addBtnStyle.Render("[ + Add Forward ]")
	connectBtn := connectBtnStyle.Render("[    Connect    ]")

	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Left, addBtn, "  ", connectBtn))
	b.WriteString("\n\n")

	// Insecure toggle
	insecureStatus := timeStyle.Render("○ off")
	if m.insecure {
		insecureStatus = statusCodeClientError.Render("● on")
	}
	b.WriteString(timeStyle.Render("Insecure mode: ") + insecureStatus + timeStyle.Render(" (press 'i' to toggle)"))
	b.WriteString("\n\n")

	// Error message
	if m.errorMsg != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(colorRed).
			Bold(true)
		b.WriteString(errorStyle.Render("Error: " + m.errorMsg))
		b.WriteString("\n\n")
	}

	// Help
	b.WriteString(timeStyle.Render("Tab/↑↓: navigate | Enter: select | Del: remove forward | p: set primary | Esc: quit"))

	return mainBoxStyle.Render(b.String())
}

// viewAddForward renders the add forward view
func (m *SetupModel) viewAddForward() string {
	var b strings.Builder

	// Header
	b.WriteString(headerTitleStyle.Render("Add Forward"))
	b.WriteString("\n")
	b.WriteString(headerSubtextStyle.Render("Configure a new port forward"))
	b.WriteString("\n\n")

	// Subdomain input
	subLabel := "Subdomain"
	if m.subdomainInput.Focused() {
		subLabel = labelStyle.Render("▶ Subdomain")
	} else {
		subLabel = timeStyle.Render("  Subdomain")
	}
	b.WriteString(subLabel + "\n")
	subStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.getInputBorderColor(m.subdomainInput.Focused())).
		Padding(0, 1).
		Width(34)
	b.WriteString(subStyle.Render(m.subdomainInput.View()))
	b.WriteString("\n\n")

	// Port input
	portLabel := "Local Port"
	if m.portInput.Focused() {
		portLabel = labelStyle.Render("▶ Local Port")
	} else {
		portLabel = timeStyle.Render("  Local Port")
	}
	b.WriteString(portLabel + "\n")
	portStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.getInputBorderColor(m.portInput.Focused())).
		Padding(0, 1).
		Width(14)
	b.WriteString(portStyle.Render(m.portInput.View()))
	b.WriteString("\n\n")

	// Local HTTPS toggle
	httpsStatus := timeStyle.Render("○ http")
	if m.localHTTPS {
		httpsStatus = urlPublicStyle.Render("● https")
	}
	b.WriteString(timeStyle.Render("Local protocol: ") + httpsStatus + timeStyle.Render(" (press 'h' to toggle)"))
	b.WriteString("\n\n")

	// Preview
	subdomain := m.subdomainInput.Value()
	if subdomain == "" {
		subdomain = "myapp"
	}
	port := m.portInput.Value()
	if port == "" {
		port = "3000"
	}
	localProto := "http"
	if m.localHTTPS {
		localProto = "https"
	}
	preview := fmt.Sprintf("%s.link.digit.zone → %s://localhost:%s", subdomain, localProto, port)
	b.WriteString(timeStyle.Render("Preview: ") + urlPublicStyle.Render(preview))
	b.WriteString("\n\n")

	// Error message
	if m.errorMsg != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(colorRed).
			Bold(true)
		b.WriteString(errorStyle.Render("Error: " + m.errorMsg))
		b.WriteString("\n\n")
	}

	// Help
	b.WriteString(timeStyle.Render("Tab: switch field | Enter: add forward | Esc: cancel"))

	return mainBoxStyle.Render(b.String())
}

// getBorderColor returns border color based on focus
func (m *SetupModel) getBorderColor(index int) lipgloss.Color {
	if m.focusIndex == index {
		return colorMustardYellow
	}
	return colorGray
}

// getInputBorderColor returns border color for inputs
func (m *SetupModel) getInputBorderColor(focused bool) lipgloss.Color {
	if focused {
		return colorMustardYellow
	}
	return colorGray
}

// getButtonStyle returns button style based on focus
func (m *SetupModel) getButtonStyle(index int) lipgloss.Style {
	base := lipgloss.NewStyle().Padding(0, 1)
	if m.focusIndex == index {
		return base.
			Foreground(lipgloss.Color("232")).
			Background(colorMustardYellow).
			Bold(true)
	}
	return base.
		Foreground(colorWhite).
		Background(colorDarkGray)
}
