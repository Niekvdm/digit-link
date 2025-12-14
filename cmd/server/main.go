package main

import (
	"log"

	"github.com/niekvdm/digit-link/internal/server"
)

func main() {
	domain := server.GetDomain()
	secret := server.GetSecret()
	port := server.GetPort()

	srv := server.New(domain, secret)
	log.Fatal(srv.Run(port))
}
