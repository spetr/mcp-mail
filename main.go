package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spetr/mcp-mail/cache"
	"github.com/spetr/mcp-mail/config"
	"github.com/spetr/mcp-mail/imap"
	"github.com/spetr/mcp-mail/mcp"
)

var (
	version = "1.0.0"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.json", "Path to configuration file")
	transport := flag.String("transport", "", "Transport type: stdio or sse (overrides config)")
	host := flag.String("host", "", "Host for SSE server (overrides config)")
	port := flag.Int("port", 0, "Port for SSE server (overrides config)")
	authToken := flag.String("auth-token", "", "Bearer token for SSE authentication (overrides config)")
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("mcp-mail version %s\n", version)
		os.Exit(0)
	}

	// Load configuration
	configMgr := config.NewManager()
	if err := configMgr.Load(*configPath); err != nil {
		log.Printf("Warning: Failed to load config from %s: %v", *configPath, err)
		log.Println("Using default configuration")
	}

	cfg := configMgr.Get()

	// Override config with command line flags
	if *transport != "" {
		cfg.Server.Transport = *transport
	}
	if *host != "" {
		cfg.Server.Host = *host
	}
	if *port != 0 {
		cfg.Server.Port = *port
	}
	if *authToken != "" {
		cfg.Server.Auth.Type = "bearer"
		cfg.Server.Auth.Token = *authToken
	}

	// Create cache manager
	cacheMgr := cache.NewManager(cfg.Cache)
	if cacheMgr.IsEnabled() {
		log.Println("Cache enabled")
	}

	// Create IMAP manager
	imapMgr := imap.NewManager(cacheMgr)
	defer imapMgr.DisconnectAll()

	// Auto-connect to accounts if configured
	if cfg.Settings.AutoConnect {
		for _, acc := range cfg.Accounts {
			if err := imapMgr.Connect(acc); err != nil {
				log.Printf("Warning: Failed to auto-connect to %s: %v", acc.ID, err)
			} else {
				log.Printf("Connected to account: %s", acc.ID)
			}
		}
	}

	// Create MCP server
	mcpServer := mcp.NewServer(configMgr, imapMgr)

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		cancel()
	}()

	// Start server based on transport
	switch cfg.Server.Transport {
	case "stdio":
		log.Println("Starting MCP-Mail server with stdio transport")
		if err := mcpServer.ServeStdio(); err != nil {
			log.Fatalf("Server error: %v", err)
		}

	case "sse":
		log.Printf("Starting MCP-Mail server with SSE transport on %s:%d", cfg.Server.Host, cfg.Server.Port)
		if cfg.Server.Auth.Type != "none" && cfg.Server.Auth.Type != "" {
			log.Printf("Authentication enabled: %s", cfg.Server.Auth.Type)
		}
		if err := mcpServer.ServeSSEWithContext(ctx, cfg.Server.Host, cfg.Server.Port); err != nil {
			log.Fatalf("Server error: %v", err)
		}

	default:
		log.Fatalf("Unknown transport: %s (use 'stdio' or 'sse')", cfg.Server.Transport)
	}

	log.Println("Server stopped")
}
