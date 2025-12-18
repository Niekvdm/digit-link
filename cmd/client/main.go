package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/niekvdm/digit-link/internal/client"
)

func main() {
	// Parse flags
	serverAddr := flag.String("server", "link.digit.zone", "Tunnel server address")
	subdomain := flag.String("subdomain", "", "Subdomain to register (optional, random if not specified)")
	port := flag.Int("port", 0, "Local port to forward to")
	localAddr := flag.String("a", "localhost", "Local address to forward to (e.g., localhost, 127.0.0.1, 192.168.1.100)")
	localHTTPS := flag.Bool("https", false, "Use HTTPS for local forwarding (default: HTTP)")
	token := flag.String("token", "", "Authentication token (required)")
	secret := flag.String("secret", "", "Server secret (deprecated, use --token)")
	timeout := flag.Duration("timeout", 5*time.Minute, "Request timeout for forwarding (e.g., 5m, 10m, 1h)")
	insecure := flag.Bool("insecure", false, "Use ws:// instead of wss:// (for local testing)")
	flag.Parse()

	// Validate required flags
	if *port == 0 {
		fmt.Println("Error: --port is required")
		flag.Usage()
		os.Exit(1)
	}

	// Token can also come from environment
	authToken := *token
	if authToken == "" {
		authToken = os.Getenv("DIGIT_LINK_TOKEN")
	}

	// Warn if no token provided
	if authToken == "" && *secret == "" {
		fmt.Println("Warning: No --token provided. Authentication may fail.")
		fmt.Println("Get a token from your digit-link administrator.")
	}

	// Create client
	c := client.New(client.Config{
		Server:         *serverAddr,
		Subdomain:      *subdomain,
		Token:          authToken,
		Secret:         *secret, // Legacy support
		LocalPort:      *port,
		LocalAddr:      *localAddr,
		LocalHTTPS:     *localHTTPS,
		Timeout:        *timeout,
		MaxRetries:     -1, // Infinite retries
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     30 * time.Second,
		Insecure:       *insecure,
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
