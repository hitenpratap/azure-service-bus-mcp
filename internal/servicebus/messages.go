package servicebus

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/hitenpratap/mcp-azure-service-bus/pkg/filter"
)

// MessageInfo holds the minimal message metadata
type MessageInfo struct {
	SequenceNumber int64     `json:"sequenceNumber"`
	EnqueuedTime   time.Time `json:"enqueuedTime"`
}

// ListMessages returns messages in the queue or subscription, optionally filtered by datetime range
func (c *Client) ListMessages(ctx context.Context, from, to *time.Time) ([]MessageInfo, error) {
	var receiver *azservicebus.Receiver
	var err error

	if c.queueName != "" {
		receiver, err = c.rawClient.NewReceiverForQueue(c.queueName, nil)
	} else if c.topicName != "" && c.subscription != "" {
		receiver, err = c.rawClient.NewReceiverForSubscription(c.topicName, c.subscription, nil)
	} else {
		return nil, fmt.Errorf("no queue or topic/subscription configured")
	}
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
