package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/niekvdm/digit-link/internal/client"
	"github.com/niekvdm/digit-link/internal/tunnel"
)

func main() {
	// Check for --tcp flag or no arguments (interactive mode)
	tcpMode := flag.Bool("tcp", false, "Use new TCP tunnel client with interactive setup")

	// Legacy WebSocket client flags
	serverAddr := flag.String("server", "link.digit.zone", "Tunnel server address")
	subdomain := flag.String("subdomain", "", "Subdomain to register (optional, random if not specified)")
	port := flag.Int("port", 0, "Local port to forward to")
	localAddr := flag.String("a", "localhost", "Local address to forward to (e.g., localhost, 127.0.0.1, 192.168.1.100)")
	localHTTPS := flag.Bool("https", false, "Use HTTPS for local forwarding (default: HTTP)")
	token := flag.String("token", "", "Authentication token (required)")
	secret := flag.String("secret", "", "Server secret (deprecated, use --token)")
	timeout := flag.Duration("timeout", 5*time.Minute, "Request timeout for forwarding (e.g., 5m, 10m, 1h)")
	insecure := flag.Bool("insecure", false, "Skip TLS verification (for local testing)")
	flag.Parse()

	// Determine mode: TCP if --tcp flag, no args, or saved config exists
	useTCP := *tcpMode || (*port == 0 && *token == "" && *secret == "")

	if useTCP {
		runTCPClient(*insecure, *timeout)
	} else {
		runWebSocketClient(*serverAddr, *subdomain, *port, *localAddr, *localHTTPS, *token, *secret, *timeout, *insecure)
	}
}

// runTCPClient runs the new TCP tunnel client with interactive setup
func runTCPClient(insecure bool, timeout time.Duration) {
	// Create setup model
	setupModel := client.NewSetupModel()

	// Try to load saved config
	if err := setupModel.LoadSavedConfig(); err != nil {
		fmt.Printf("Warning: Failed to load saved config: %v\n", err)
	}

	// Variables to capture setup results
	var (
		server   string
		token    string
		forwards []tunnel.ForwardConfig
		useInsecure bool
	)

	// Set callback for when setup completes
	setupModel.SetOnConnect(func(s, t string, f []tunnel.ForwardConfig, ins bool) {
		server = s
		token = t
		forwards = f
		useInsecure = ins
	})

	// Run setup TUI
	p := tea.NewProgram(setupModel, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running setup: %v\n", err)
		os.Exit(1)
	}

	// Check if setup was completed (not cancelled)
	if server == "" || len(forwards) == 0 {
		// User cancelled setup
		return
	}

	// Create TCP client
	tcpClient := client.NewTCPClient(client.TCPConfig{
		Server:         server,
		Token:          token,
		Forwards:       forwards,
		Insecure:       useInsecure,
		MaxRetries:     -1, // Infinite retries
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     30 * time.Second,
		Timeout:        timeout,
	})

	// Create model for connected view
	model := client.NewTCPModel()
	tcpClient.SetModel(model)

	// Start client in goroutine
	go func() {
		if err := tcpClient.Run(); err != nil {
			if model != nil {
				model.SendUpdate(client.QuitMsg{})
			}
		}
	}()

	// Run connected TUI
	connectedProgram := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := connectedProgram.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}

	// Cleanup
	tcpClient.Close()
}

// runWebSocketClient runs the legacy WebSocket tunnel client
func runWebSocketClient(serverAddr, subdomain string, port int, localAddr string, localHTTPS bool, token, secret string, timeout time.Duration, insecure bool) {
	// Validate required flags
	if port == 0 {
		fmt.Println("Error: --port is required for legacy WebSocket mode")
		fmt.Println()
		fmt.Println("For the new interactive TCP client, run without arguments:")
		fmt.Println("  digit-link")
		fmt.Println()
		fmt.Println("Or explicitly use TCP mode:")
		fmt.Println("  digit-link --tcp")
		fmt.Println()
		flag.Usage()
		os.Exit(1)
	}

	// Token can also come from environment
	authToken := token
	if authToken == "" {
		authToken = os.Getenv("DIGIT_LINK_TOKEN")
	}

	// Warn if no token provided
	if authToken == "" && secret == "" {
		fmt.Println("Warning: No --token provided. Authentication may fail.")
		fmt.Println("Get a token from your digit-link administrator.")
	}

	// Deprecation warning for WebSocket client
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║  DEPRECATION NOTICE: WebSocket tunnel client is deprecated.        ║")
	fmt.Println("║                                                                    ║")
	fmt.Println("║  Please migrate to the new TCP tunnel client which supports:       ║")
	fmt.Println("║    • Multi-port forwarding via single connection                   ║")
	fmt.Println("║    • Better performance with yamux multiplexing                    ║")
	fmt.Println("║    • Interactive TUI setup                                         ║")
	fmt.Println("║                                                                    ║")
	fmt.Println("║  Run 'digit-link' without arguments for the new interactive setup. ║")
	fmt.Println("║                                                                    ║")
	fmt.Println("║  WebSocket support will be removed in a future version.            ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Create client
	c := client.New(client.Config{
		Server:         serverAddr,
		Subdomain:      subdomain,
		Token:          authToken,
		Secret:         secret, // Legacy support
		LocalPort:      port,
		LocalAddr:      localAddr,
		LocalHTTPS:     localHTTPS,
		Timeout:        timeout,
		MaxRetries:     -1, // Infinite retries
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     30 * time.Second,
		Insecure:       insecure,
	})

	// Get the model from the client
	model := c.Model()

	// Start client in goroutine
	go func() {
		if err := c.Run(); err != nil {
			// Send quit message to model on error
			if model != nil {
				model.SendUpdate(client.QuitMsg{})
			}
		}
	}()

	// Run Bubbletea program (blocks until quit)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}

	// Cleanup on exit
	c.Close()
}
