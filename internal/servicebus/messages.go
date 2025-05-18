package servicebus

import (
	"context"
	"fmt"
	"time"

	"github.com/hitenpratap/mcp-azure-service-bus/pkg/filter"
)

// MessageInfo holds the minimal message metadata
type MessageInfo struct {
	SequenceNumber int64     `json:"sequenceNumber"`
	EnqueuedTime   time.Time `json:"enqueuedTime"`
}

// ListMessages returns messages in the queue, optionally filtered by datetime range
func (c *Client) ListMessages(ctx context.Context, from, to *time.Time) ([]MessageInfo, error) {
	receiver, err := c.rawClient.NewReceiverForQueue(c.queueName, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating receiver: %w", err)
	}
	defer receiver.Close(ctx)

	var result []MessageInfo
	for {
		msgs, err := receiver.PeekMessages(ctx, 100, nil)
		if err != nil {
			return nil, fmt.Errorf("receive error: %w", err)
		}
		if len(msgs) == 0 {
			break
		}
		for _, msg := range msgs {
			ti := msg.EnqueuedTime
			if filter.InRange(*ti, from, to) {
				result = append(result, MessageInfo{
					SequenceNumber: *msg.SequenceNumber,
					EnqueuedTime:   *ti,
				})
			}
		}
	}
	return result, nil
}
