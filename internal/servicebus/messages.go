package servicebus

import (
	"context"
	"fmt"
	"time"

	azsb "github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus" // Added alias
	"github.com/hitenpratap/mcp-azure-service-bus/pkg/filter"
)

// Explicitly use a type from the aliased package to mark it as used.
var _ *azsb.ReceivedMessage

// MessageInfo holds the minimal message metadata
type MessageInfo struct {
	SequenceNumber int64     `json:"sequenceNumber"`
	EnqueuedTime   time.Time `json:"enqueuedTime"`
}

// ListMessages returns messages in the queue or subscription, optionally filtered by datetime range
func (c *Client) ListMessages(ctx context.Context, from, to *time.Time) ([]MessageInfo, error) {
	var receiver SDKReceiver // Changed from *azservicebus.Receiver
	var err error

	if c.QueueName != "" {
		receiver, err = c.AzClient.NewReceiverForQueue(c.QueueName, nil)
	} else if c.TopicName != "" && c.Subscription != "" {
		receiver, err = c.AzClient.NewReceiverForSubscription(c.TopicName, c.Subscription, nil)
	} else {
		return nil, fmt.Errorf("no queue or topic/subscription configured")
	}
	if err != nil {
		return nil, fmt.Errorf("error creating receiver: %w", err)
	}
	// Ensure receiver is not nil before attempting to close it, especially if NewReceiverFor... could return a nil receiver AND nil error (though unlikely for this SDK).
	if receiver != nil {
		defer receiver.Close(ctx)
	}


	var result []MessageInfo
	for {
		// Assuming PeekMessages is available on SDKReceiver
		msgs, err := receiver.PeekMessages(ctx, 100, nil) // This call remains the same due to the interface
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
