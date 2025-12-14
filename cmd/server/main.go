package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/niekvdm/digit-link/internal/auth"
	"github.com/niekvdm/digit-link/internal/db"
	"github.com/niekvdm/digit-link/internal/server"
)

func main() {
	// Parse command line flags
	setupAdmin := flag.Bool("setup-admin", false, "Create initial admin account and exit")
	adminUsername := flag.String("admin-username", "admin", "Username for initial admin account")
	flag.Parse()

	// Get configuration from environment
	domain := server.GetDomain()
	secret := server.GetSecret()
	port := server.GetPort()
	dbPath := db.GetDBPath()

	// Initialize database
	database, err := db.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	log.Printf("Database initialized at %s", dbPath)

	// Handle admin setup mode
	if *setupAdmin {
		if err := createInitialAdmin(database, *adminUsername); err != nil {
			log.Fatalf("Failed to create admin account: %v", err)
		}
		return
	}

	// Check if admin account exists
	hasAdmin, err := database.HasAdminAccount()
	if err != nil {
		log.Fatalf("Failed to check admin account: %v", err)
	}

	if !hasAdmin {
		log.Println("WARNING: No admin account exists!")
		log.Println("Run with --setup-admin to create the initial admin account")
		log.Println("Or set ADMIN_TOKEN environment variable to auto-create on startup")

		// Auto-create admin if ADMIN_TOKEN is set
		if adminToken := os.Getenv("ADMIN_TOKEN"); adminToken != "" {
			if err := createAdminWithToken(database, "admin", adminToken); err != nil {
				log.Printf("Failed to create admin from ADMIN_TOKEN: %v", err)
			}
		}
	}

	// Start server
	srv := server.New(domain, secret, database)
	log.Fatal(srv.Run(port))
}

// createInitialAdmin creates the initial admin account and prints the token
func createInitialAdmin(database *db.DB, username string) error {
	// Check if admin already exists
	hasAdmin, err := database.HasAdminAccount()
	if err != nil {
		return fmt.Errorf("failed to check admin account: %w", err)
	}

	if hasAdmin {
		fmt.Println("An admin account already exists.")
		fmt.Println("To create a new admin, use the admin dashboard or API.")
		return nil
	}

	// Generate token
	token, tokenHash, err := auth.GenerateToken()
	if err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}

	// Create admin account
	account, err := database.CreateAccount(username, tokenHash, true)
	if err != nil {
		return fmt.Errorf("failed to create account: %w", err)
	}

	fmt.Println("================================================================================")
	fmt.Println("                         ADMIN ACCOUNT CREATED")
	fmt.Println("================================================================================")
	fmt.Printf("Username: %s\n", account.Username)
	fmt.Printf("Token:    %s\n", token)
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("IMPORTANT: Save this token securely. It will not be shown again!")
	fmt.Println("Use this token to access the admin dashboard and API.")
	fmt.Println("================================================================================")

	return nil
}

// createAdminWithToken creates an admin account with a specific token
func createAdminWithToken(database *db.DB, username, token string) error {
	tokenHash := auth.HashToken(token)

	// Check if account already exists with this token
	existing, err := database.GetAccountByTokenHash(tokenHash)
	if err != nil {
		return err
	}
	if existing != nil {
		log.Printf("Admin account already exists with provided token")
		return nil
	}

	// Create admin account
	_, err = database.CreateAccount(username, tokenHash, true)
	if err != nil {
		return err
	}

	log.Printf("Admin account created from ADMIN_TOKEN environment variable")
	return nil
}
