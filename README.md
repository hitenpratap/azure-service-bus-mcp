# azure-service-bus-mcp

A Go-based tool to browse Azure Service Bus queues and dead-letter queues, with datetime filtering and full message inspection, exposed via the [MCP protocol](https://github.com/mark3labs/mcp-go) for integration with Claude Desktop, VS Code, and Docker Desktop.

---

## Features

- **Browse Azure Service Bus Queues**: List and inspect messages in queues or topic subscriptions.
- **Dead-letter Queue Support**: Access and inspect messages in dead-letter queues.
- **Datetime Filtering**: Filter messages by enqueue time.
- **Full Message Inspection**: View message bodies and all properties.
- **MCP Integration**: Exposes all features via the MCP protocol for use in Claude Desktop, VS Code, and Docker Desktop.

---

## Project Structure

```
.
├── bin/                        # Compiled binaries
│   └── azure-servicebus-mcp
├── cmd/
│   └── azure-servicebus/       # Main application entrypoint
│       ├── main.go
│       └── rest.go             # Optional REST server (not used by MCP)
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

---

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

---

## Configuration

Edit the configuration file at `config/config.yaml` to set your Azure Service Bus connection string and other options.

Example:

```yaml
azure:
  servicebus:
    connectionString: "Endpoint=sb://<YOUR_NAMESPACE>.servicebus.windows.net/;SharedAccessKeyName=<KEY_NAME>;SharedAccessKey=<KEY>"
    queueName: "your-queue-name"
    topicName: "your-topic-name"
    subscriptionName: "your-subscription"
```

- For queue browsing, set `queueName`.
- For topic subscription browsing, set `topicName` and `subscriptionName`.
- Only one of `queueName` or (`topicName` + `subscriptionName`) should be set for a given run.

---

## Usage

Run the MCP tool:

```sh
./bin/azure-servicebus-mcp --config config/config.yaml
```

This will start the MCP server and allow Claude Desktop, VS Code, or Docker Desktop to connect for queue and dead-letter browsing.

---

## MCP Tools Overview

When you connect this project as an MCP tool (in Claude Desktop, VS Code, or Docker Desktop), the following tools are available:

### 1. `ListMessages`
- **Description:** Lists messages in the configured queue or topic subscription.
- **Parameters:**
  - `from` (string, optional): RFC3339 start time for filtering messages by enqueue time.
  - `to` (string, optional): RFC3339 end time for filtering messages by enqueue time.
- **Returns:** A JSON array of messages, each with `sequenceNumber` and `enqueuedTime`.

### 2. `ListDeadLetters`
- **Description:** Lists messages in the dead-letter queue (DLQ) for the configured queue or subscription.
- **Parameters:**
  - `from` (string, optional): RFC3339 start time for filtering DLQ messages.
  - `to` (string, optional): RFC3339 end time for filtering DLQ messages.
- **Returns:** A JSON array of DLQ messages, each with `sequenceNumber` and `enqueuedTime`.

### 3. `PrintMessage`
- **Description:** Fetches and displays the full details of a message by its sequence number.
- **Parameters:**
  - `sequenceNumber` (number, required): The sequence number of the message to fetch.
  - `deadLetter` (boolean, required): Set to `true` to fetch from the dead-letter queue, or `false` for the main queue/subscription.
- **Returns:** A JSON object with:
  - `sequenceNumber`
  - `enqueuedTime`
  - `body` (message body as string)
  - `properties` (application properties)
  - `systemProperties` (system properties like MessageID, ContentType, etc.)

---

## Integration with Claude Desktop, VS Code, and Docker Desktop

### Claude Desktop

1. Open Claude Desktop.
2. Go to **Settings > Developer**.
3. Click **Edit Config**.
4. Enter:
   ```json
   "azure-servicebus-mcp": {
    "command": "/absolute/path/to/bin/azure-servicebus-mcp",
    "args": [
        "--config",
        "/absolute/path/to/config/config.yaml"
    ],
    "env": {}
    }
   ```
   - Adjust the paths as needed
5. Save and enable the tool.
6. Claude will now be able to browse and inspect your Azure Service Bus messages via chat.

### VS Code
To Be Added...

### Docker Desktop
To Be Added...

---

## Code Overview

- `internal/servicebus/client.go`: Azure Service Bus client setup and connection.
- `internal/servicebus/messages.go`: Message retrieval, listing, and filtering.
- `internal/servicebus/deadletter.go`: Dead-letter queue access and message fetching.
- `pkg/filter/datetime.go`: Datetime filtering utilities.
- `internal/mcp/tool.go`: MCP protocol implementation for Claude Desktop, VS Code, and Docker Desktop.
- `cmd/azure-servicebus/main.go`: Application entrypoint and CLI handling.

---

## Development

Install dependencies:

```sh
go mod tidy
```

Run tests (if available):

```sh
go test ./...
```

---

## Contributing

Contributions are welcome! Please open issues or submit pull requests.

---
