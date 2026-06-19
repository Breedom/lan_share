package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Breedom/lan_share/internal/core"
	"github.com/Breedom/lan_share/internal/server"
)

func main() {
	config := core.LoadConfig()

	discovery := core.NewDiscovery(config)
	if err := discovery.Start(); err != nil {
		log.Fatalf("Failed to start discovery: %v", err)
	}
	defer discovery.Stop()

	httpServer := server.NewHTTPServer(config)
	go httpServer.Start()

	transfer := core.NewTransferManager(config)
	transfer.Start()

	fmt.Println("LanShare started. Press Ctrl+C to exit.")
	fmt.Printf("Web UI: http://localhost:%d\n", config.Server.HTTPPort)

	select {}
}

func init() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println("LanShare v1.0.0")
		os.Exit(0)
	}
}
