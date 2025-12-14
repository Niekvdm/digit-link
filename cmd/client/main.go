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
	server := flag.String("server", "tunnel.digit.zone", "Tunnel server address")
	subdomain := flag.String("subdomain", "", "Subdomain to register")
	port := flag.Int("port", 0, "Local port to forward to")
	secret := flag.String("secret", "", "Server secret (optional)")
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

	// Create client
	c := client.New(client.Config{
		Server:         *server,
		Subdomain:      *subdomain,
		Secret:         *secret,
		LocalPort:      *port,
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
