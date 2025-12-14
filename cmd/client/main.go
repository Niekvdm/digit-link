package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/niekvdm/digit-link/internal/client"
)

func main() {
	// Parse flags
	serverAddr := flag.String("server", "tunnel.digit.zone", "Tunnel server address")
	subdomain := flag.String("subdomain", "", "Subdomain to register")
	port := flag.Int("port", 0, "Local port to forward to")
	token := flag.String("token", "", "Authentication token (required)")
	secret := flag.String("secret", "", "Server secret (deprecated, use --token)")
	timeout := flag.Duration("timeout", 5*time.Minute, "Request timeout for forwarding (e.g., 5m, 10m, 1h)")
	flag.Parse()

	// Validate required flags
	if *subdomain == "" {
		fmt.Println("Error: --subdomain is required")
		flag.Usage()
		os.Exit(1)
	}

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
		Timeout:        *timeout,
		MaxRetries:     -1, // Infinite retries
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     30 * time.Second,
	})

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("Shutting down...")
		c.Close()
	}()

	// Run client
	if err := c.Run(); err != nil {
		log.Fatalf("Client error: %v", err)
	}
}
