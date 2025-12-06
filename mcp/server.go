package mcp

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/spetr/mcp-mail/config"
	"github.com/spetr/mcp-mail/imap"
	"github.com/spetr/mcp-mail/mcp/tools"
	"github.com/mark3labs/mcp-go/server"
)

// Server wraps the MCP server with IMAP functionality
type Server struct {
	mcpServer    *server.MCPServer
	configMgr    *config.Manager
	imapMgr      *imap.Manager
	toolRegistry *tools.Registry
	serverCfg    config.ServerConfig
}

// NewServer creates a new MCP-Mail server
func NewServer(configMgr *config.Manager, imapMgr *imap.Manager) *Server {
	// Create MCP server
	mcpServer := server.NewMCPServer(
		"mcp-mail",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)

	s := &Server{
		mcpServer: mcpServer,
		configMgr: configMgr,
		imapMgr:   imapMgr,
		serverCfg: configMgr.GetServerConfig(),
	}

	// Create and register tools
	s.toolRegistry = tools.NewRegistry(configMgr, imapMgr)
	s.registerTools()

	return s
}

// registerTools registers all MCP tools
func (s *Server) registerTools() {
	allTools := s.toolRegistry.GetAllTools()
	for _, tool := range allTools {
		s.mcpServer.AddTool(tool.Definition, tool.Handler)
	}
}

// ServeStdio starts the server using stdio transport
func (s *Server) ServeStdio() error {
	log.Println("Starting MCP-Mail server (stdio transport)")
	return server.ServeStdio(s.mcpServer)
}

// ServeSSE starts the server using SSE transport
func (s *Server) ServeSSE(host string, port int) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("Starting MCP-Mail server (SSE transport) on %s", addr)

	// Create SSE server
	sseServer := server.NewSSEServer(s.mcpServer,
		server.WithBaseURL(fmt.Sprintf("http://%s", addr)),
	)

	// Create router
	mux := http.NewServeMux()

	// Apply authentication middleware if configured
	authMiddleware := s.createAuthMiddleware()

	mux.Handle("/sse", authMiddleware(sseServer.SSEHandler()))
	mux.Handle("/message", authMiddleware(sseServer.MessageHandler()))

	// Health check endpoint (no auth required)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Start HTTP server
	httpServer := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return httpServer.ListenAndServe()
}

// ServeSSEWithContext starts the SSE server with context for graceful shutdown
func (s *Server) ServeSSEWithContext(ctx context.Context, host string, port int) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("Starting MCP-Mail server (SSE transport) on %s", addr)

	// Create SSE server
	sseServer := server.NewSSEServer(s.mcpServer,
		server.WithBaseURL(fmt.Sprintf("http://%s", addr)),
	)

	// Create router
	mux := http.NewServeMux()

	// Apply authentication middleware if configured
	authMiddleware := s.createAuthMiddleware()

	mux.Handle("/sse", authMiddleware(sseServer.SSEHandler()))
	mux.Handle("/message", authMiddleware(sseServer.MessageHandler()))

	// Health check endpoint (no auth required)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Start HTTP server
	httpServer := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Handle shutdown
	go func() {
		<-ctx.Done()
		log.Println("Shutting down SSE server...")
		httpServer.Shutdown(context.Background())
	}()

	err := httpServer.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// createAuthMiddleware creates an authentication middleware based on config
func (s *Server) createAuthMiddleware() func(http.Handler) http.Handler {
	authCfg := s.serverCfg.Auth

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch authCfg.Type {
			case "none", "":
				// No authentication
				next.ServeHTTP(w, r)

			case "bearer":
				if !validateBearerToken(r, authCfg.Token) {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				next.ServeHTTP(w, r)

			case "basic":
				if !validateBasicAuth(r, authCfg.Username, authCfg.Password) {
					w.Header().Set("WWW-Authenticate", `Basic realm="MCP-IMAP"`)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				next.ServeHTTP(w, r)

			case "api_key":
				headerName := authCfg.HeaderName
				if headerName == "" {
					headerName = "X-API-Key"
				}
				if !validateAPIKey(r, headerName, authCfg.Token) {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				next.ServeHTTP(w, r)

			default:
				log.Printf("Unknown auth type: %s", authCfg.Type)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		})
	}
}

// GetMCPServer returns the underlying MCP server
func (s *Server) GetMCPServer() *server.MCPServer {
	return s.mcpServer
}
