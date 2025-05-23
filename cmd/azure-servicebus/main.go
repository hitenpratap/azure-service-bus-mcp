package main

import (
	"flag"
	"log"

	"github.com/hitenpratap/mcp-azure-service-bus/internal/servicebus"
	"github.com/mark3labs/mcp-go/server"

	sm "github.com/hitenpratap/mcp-azure-service-bus/internal/mcp"
)

func main() {
	// 1) add a flag for config file
	configFile := flag.String("config", "config/config.yaml", "path to config.yaml")
	flag.Parse()

	// 1) Initialize Service Bus client
	sbClient, err := servicebus.NewClient(*configFile)
	if err != nil {
		log.Fatalf("failed to create service bus client: %v", err)
	}

	// 2) Bootstrap an MCP server
	srv := server.NewMCPServer(
		"azure-servicebus-mcp", "1.0.0",
		server.WithToolCapabilities(true),
	)

	// 3) Register MCP tools
	sm.Register(sbClient, srv)

	// 4) Serve via SSE (for Claude Desktop integration)
	sseServer := server.NewSSEServer(srv)
	if err := sseServer.Start(":8080"); err != nil {
		log.Fatalf("failed to start SSE server: %v", err)
	}
}
