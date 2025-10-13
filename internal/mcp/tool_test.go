package mcp

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/hitenpratap/mcp-azure-service-bus/internal/servicebus"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/require"
)

type listCall struct {
	from *time.Time
	to   *time.Time
}

type fetchCall struct {
	seq        int64
	deadLetter bool
}

type stubServiceBus struct {
	listMessagesResult []servicebus.MessageInfo
	listMessagesErr    error
	listMessagesCalls  []listCall

	listDeadLettersResult []servicebus.DeadLetterInfo
	listDeadLettersErr    error
	listDeadLettersCalls  []listCall

	fetchMessageResult *azservicebus.ReceivedMessage
	fetchMessageErr    error
	fetchMessageCalls  []fetchCall
}

func (s *stubServiceBus) ListMessages(ctx context.Context, from, to *time.Time) ([]servicebus.MessageInfo, error) {
	s.listMessagesCalls = append(s.listMessagesCalls, listCall{cloneTimePtr(from), cloneTimePtr(to)})
	if s.listMessagesErr != nil {
		return nil, s.listMessagesErr
	}
	return s.listMessagesResult, nil
}

func (s *stubServiceBus) ListDeadLetters(ctx context.Context, from, to *time.Time) ([]servicebus.DeadLetterInfo, error) {
	s.listDeadLettersCalls = append(s.listDeadLettersCalls, listCall{cloneTimePtr(from), cloneTimePtr(to)})
	if s.listDeadLettersErr != nil {
		return nil, s.listDeadLettersErr
	}
	return s.listDeadLettersResult, nil
}

func (s *stubServiceBus) FetchMessage(ctx context.Context, seq int64, deadLetter bool) (*azservicebus.ReceivedMessage, error) {
	s.fetchMessageCalls = append(s.fetchMessageCalls, fetchCall{seq: seq, deadLetter: deadLetter})
	if s.fetchMessageErr != nil {
		return nil, s.fetchMessageErr
	}
	return s.fetchMessageResult, nil
}

func cloneTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	cloned := *t
	return &cloned
}

func getToolHandler(t *testing.T, srv *server.MCPServer, name string) server.ToolHandlerFunc {
	t.Helper()

	toolsField := reflect.ValueOf(srv).Elem().FieldByName("tools")
	require.True(t, toolsField.IsValid(), "tools map should exist")

	toolsValue := reflect.NewAt(toolsField.Type(), unsafe.Pointer(toolsField.UnsafeAddr())).Elem().Interface().(map[string]server.ServerTool)
	tool, ok := toolsValue[name]
	require.Truef(t, ok, "tool %s should be registered", name)
	return tool.Handler
}

func TestRegister_ListMessagesSuccess(t *testing.T) {
	stub := &stubServiceBus{
		listMessagesResult: []servicebus.MessageInfo{{
			SequenceNumber: 42,
			EnqueuedTime:   time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC),
		}},
	}

	srv := server.NewMCPServer("test", "1.0.0")
	Register(stub, srv)

	handler := getToolHandler(t, srv, "ListMessages")

	from := time.Now().Add(-time.Hour).UTC().Truncate(time.Second)
	to := from.Add(2 * time.Hour)

	req := mcp.CallToolRequest{}
	req.Params.Name = "ListMessages"
	req.Params.Arguments = map[string]any{
		"from": from.Format(time.RFC3339),
		"to":   to.Format(time.RFC3339),
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.False(t, result.IsError)
	require.Len(t, result.Content, 1)

	require.Len(t, stub.listMessagesCalls, 1)
	call := stub.listMessagesCalls[0]
	require.NotNil(t, call.from)
	require.NotNil(t, call.to)
	require.WithinDuration(t, from, *call.from, time.Second)
	require.WithinDuration(t, to, *call.to, time.Second)

	textContent, ok := result.Content[0].(mcp.TextContent)
	require.True(t, ok, "expected text content")

	var decoded []servicebus.MessageInfo
	require.NoError(t, json.Unmarshal([]byte(textContent.Text), &decoded))
	require.Equal(t, stub.listMessagesResult, decoded)
}

func TestRegister_ListMessagesInvalidFrom(t *testing.T) {
	stub := &stubServiceBus{}
	srv := server.NewMCPServer("test", "1.0.0")
	Register(stub, srv)

	handler := getToolHandler(t, srv, "ListMessages")

	req := mcp.CallToolRequest{}
	req.Params.Name = "ListMessages"
	req.Params.Arguments = map[string]any{
		"from": "not-a-time",
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.True(t, result.IsError)
	require.Empty(t, stub.listMessagesCalls)

	require.Len(t, result.Content, 1)
	textContent, ok := result.Content[0].(mcp.TextContent)
	require.True(t, ok)
	require.Contains(t, textContent.Text, "invalid `from` datetime")
}

func TestRegister_PrintMessageSuccess(t *testing.T) {
	seq := int64(99)
	enqueued := time.Date(2025, 3, 4, 5, 6, 7, 0, time.UTC)
	stub := &stubServiceBus{
		fetchMessageResult: &azservicebus.ReceivedMessage{
			SequenceNumber: &seq,
			EnqueuedTime:   &enqueued,
			Body:           []byte("hello world"),
			ApplicationProperties: map[string]any{
				"foo": "bar",
			},
			MessageID:        "msg-id",
			ContentType:      to.Ptr("application/json"),
			CorrelationID:    to.Ptr("corr-id"),
			Subject:          to.Ptr("subject"),
			To:               to.Ptr("to-queue"),
			ReplyTo:          to.Ptr("reply-queue"),
			ReplyToSessionID: to.Ptr("reply-session"),
			SessionID:        to.Ptr("session"),
		},
	}

	srv := server.NewMCPServer("test", "1.0.0")
	Register(stub, srv)

	handler := getToolHandler(t, srv, "PrintMessage")

	req := mcp.CallToolRequest{}
	req.Params.Name = "PrintMessage"
	req.Params.Arguments = map[string]any{
		"sequenceNumber": float64(seq),
		"deadLetter":     false,
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.False(t, result.IsError)
	require.Len(t, result.Content, 1)

	require.Len(t, stub.fetchMessageCalls, 1)
	call := stub.fetchMessageCalls[0]
	require.Equal(t, seq, call.seq)
	require.False(t, call.deadLetter)

	textContent, ok := result.Content[0].(mcp.TextContent)
	require.True(t, ok)

	var payload struct {
		SequenceNumber int64          `json:"sequenceNumber"`
		EnqueuedTime   string         `json:"enqueuedTime"`
		Body           string         `json:"body"`
		Properties     map[string]any `json:"properties"`
		SystemProps    map[string]any `json:"systemProperties"`
	}
	require.NoError(t, json.Unmarshal([]byte(textContent.Text), &payload))

	require.Equal(t, seq, payload.SequenceNumber)
	require.Equal(t, enqueued.Format(time.RFC3339), payload.EnqueuedTime)
	require.Equal(t, "hello world", payload.Body)
	require.Equal(t, map[string]any{"foo": "bar"}, payload.Properties)
	require.Equal(t, "msg-id", payload.SystemProps["MessageID"])
	require.Equal(t, "application/json", payload.SystemProps["ContentType"])
	require.Equal(t, "corr-id", payload.SystemProps["CorrelationID"])
	require.Equal(t, "subject", payload.SystemProps["Subject"])
	require.Equal(t, "to-queue", payload.SystemProps["To"])
	require.Equal(t, "reply-queue", payload.SystemProps["ReplyTo"])
	require.Equal(t, "reply-session", payload.SystemProps["ReplyToSession"])
	require.Equal(t, "session", payload.SystemProps["SessionID"])
}
