# azure-service-bus-mcp
Exposes Azure Service Bus queue and dead-letter browsing to Claude Desktop via MCP, with datetime filtering and full message inspection, built in Go.

## Features

- **Browse Azure Service Bus Queues**: List and inspect messages in Azure Service Bus queues.
- **Dead-letter Queue Support**: Access and inspect messages in dead-letter queues.
- **Datetime Filtering**: Filter messages by enqueue time or other datetime fields.
- **Full Message Inspection**: View message bodies and properties.
- **MCP Integration**: Communicate with Claude Desktop via the MCP protocol.

## Project Structure

```
.
├── bin/                        # Compiled binaries
│   └── azure-servicebus-mcp
├── cmd/
│   └── azure-servicebus/       # Main application entrypoint
│       └── main.go
├── config/
│   └── config.yaml             # Application configuration
├── internal/
│   ├── mcp/
│   │   └── tool.go             # MCP protocol implementation
│   └── servicebus/
│       ├── client.go           # Azure Service Bus client logic
│       ├── deadletter.go       # Dead-letter queue handling
│       └── messages.go         # Message retrieval and filtering
├── pkg/
│   └── filter/
│       └── datetime.go         # Datetime filtering utilities
├── go.mod
├── go.sum
└── README.md
```

## Getting Started

### Prerequisites

- Go 1.20+ (see `go.mod`)
- Azure Service Bus namespace and credentials

### Installation

Clone the repository:

```sh
git clone https://github.com/hitenpratap/azure-service-bus-mcp.git
cd azure-service-bus-mcp
```

Build the binary:

```sh
go build -o bin/azure-servicebus-mcp ./cmd/azure-servicebus
```

### Configuration

Edit the configuration file at `config/config.yaml` to set your Azure Service Bus connection string and other options.

Example:

```yaml
servicebus:
  connection_string: "Endpoint=sb://<namespace>.servicebus.windows.net/;SharedAccessKeyName=<keyname>;SharedAccessKey=<key>"
  queue_name: "your-queue-name"
```

### Usage

Run the MCP tool:

```sh
./bin/azure-servicebus-mcp --config config/config.yaml
```

This will start the MCP server and allow Claude Desktop to connect for queue and dead-letter browsing.

## Code Overview

- `internal/servicebus/client.go`: Azure Service Bus client setup and connection.
- `internal/servicebus/messages.go`: Message retrieval, listing, and filtering.
- `internal/servicebus/deadletter.go`: Dead-letter queue access.
- `pkg/filter/datetime.go`: Datetime filtering utilities.
- `internal/mcp/tool.go`: MCP protocol implementation for Claude Desktop integration.
- `cmd/azure-servicebus/main.go`: Application entrypoint and CLI handling.

## Development

Install dependencies:

```sh
go mod tidy
```

Run tests (if available):

```sh
go test ./...
```

## Contributing

Contributions are welcome! Please open issues or submit pull requests.

---
