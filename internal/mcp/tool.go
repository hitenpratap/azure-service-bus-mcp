package mcp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hitenpratap/mcp-azure-service-bus/internal/servicebus"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func Register(sb *servicebus.Client, srv *server.MCPServer) {
	// --- ListMessages ---
	listMessages := mcp.NewTool(
		"ListMessages",
		mcp.WithDescription("List messages in the queue with optional datetime filters"),
		mcp.WithString("from", mcp.Description("RFC3339 start time")),
		mcp.WithString("to", mcp.Description("RFC3339 end time")),
	)
	srv.AddTool(listMessages, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var fromTime, toTime *time.Time
		if raw, _ := req.Params.Arguments["from"].(string); raw != "" {
			t, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid `from` datetime", err), nil
			}
			fromTime = &t
		}
		if raw, _ := req.Params.Arguments["to"].(string); raw != "" {
			t, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid `to` datetime", err), nil
			}
			toTime = &t
		}

		msgs, err := sb.ListMessages(ctx, fromTime, toTime)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to list messages", err), nil
		}

		data, err := json.Marshal(msgs)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to marshal messages", err), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	})

	// --- ListDeadLetters ---
	listDLQ := mcp.NewTool(
		"ListDeadLetters",
		mcp.WithDescription("List dead-letter queue messages with filters"),
		mcp.WithString("from", mcp.Description("RFC3339 start time")),
		mcp.WithString("to", mcp.Description("RFC3339 end time")),
	)
	srv.AddTool(listDLQ, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var fromTime, toTime *time.Time
		if raw, _ := req.Params.Arguments["from"].(string); raw != "" {
			t, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid `from` datetime", err), nil
			}
			fromTime = &t
		}
		if raw, _ := req.Params.Arguments["to"].(string); raw != "" {
			t, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid `to` datetime", err), nil
			}
			toTime = &t
		}

		dlqs, err := sb.ListDeadLetters(ctx, fromTime, toTime)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to list dead-letter messages", err), nil
		}

		data, err := json.Marshal(dlqs)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to marshal dead-letter messages", err), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	})

	// --- PrintMessage ---
	printMsg := mcp.NewTool(
		"PrintMessage",
		mcp.WithDescription("Fetch and display a full message by sequence number"),
		mcp.WithNumber("sequenceNumber",
			mcp.Required(),
			mcp.Description("The sequence number of the message"),
		),
		mcp.WithBoolean("deadLetter",
			mcp.Required(),
			mcp.Description("Set to true to fetch from the dead-letter queue"),
		),
	)
	srv.AddTool(printMsg, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		seqFloat, ok := req.Params.Arguments["sequenceNumber"].(float64)
		if !ok {
			return mcp.NewToolResultError("`sequenceNumber` must be a number"), nil
		}
		seq := int64(seqFloat)

		dead, ok := req.Params.Arguments["deadLetter"].(bool)
		if !ok {
			return mcp.NewToolResultError("`deadLetter` must be a boolean"), nil
		}

		msg, err := sb.FetchMessage(ctx, seq, dead)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to fetch message", err), nil
		}

		data, err := json.Marshal(msg)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to marshal message", err), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	})
}
