package servicebus

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/hitenpratap/mcp-azure-service-bus/pkg/filter"
)

// DeadLetterInfo holds dead-letter queue message metadata
type DeadLetterInfo struct {
	SequenceNumber int64     `json:"sequenceNumber"`
	EnqueuedTime   time.Time `json:"enqueuedTime"`
}

// ListDeadLetters returns messages in the dead-letter queue or subscription DLQ, optionally filtered by datetime range
func (c *Client) ListDeadLetters(ctx context.Context, from, to *time.Time) ([]DeadLetterInfo, error) {
	var receiver SDKReceiver // Changed from *azservicebus.Receiver
	var err error

	if c.DlqQueueName != "" {
		receiver, err = c.AzClient.NewReceiverForQueue(c.DlqQueueName, nil)
	} else if c.DlqSubName != "" {
		// Assuming NewReceiverForQueue is the correct method for DLQ of a subscription based on previous structure.
		// If there's a specific NewReceiverForSubscriptionDeadLetterQueue, that should be on the SDKClient interface.
		// For now, sticking to the existing pattern.
		receiver, err = c.AzClient.NewReceiverForQueue(c.DlqSubName, nil)
	} else {
		return nil, fmt.Errorf("no DLQ configured")
	}
	if err != nil {
		return nil, fmt.Errorf("error creating DLQ receiver: %w", err)
	}
	if receiver != nil {
		defer receiver.Close(ctx)
	}

	var result []DeadLetterInfo
	for {
		msgs, err := receiver.PeekMessages(ctx, 100, nil) // Call on SDKReceiver
		if err != nil {
			return nil, fmt.Errorf("DLQ receive error: %w", err)
		}
		if len(msgs) == 0 {
			break
		}
		for _, msg := range msgs {
			ti := msg.EnqueuedTime
			if filter.InRange(*ti, from, to) {
				result = append(result, DeadLetterInfo{
					SequenceNumber: *msg.SequenceNumber,
					EnqueuedTime:   *ti,
				})
			}
		}
	}
	return result, nil
}

// FetchMessage returns the full message body and properties for a given sequence number
func (c *Client) FetchMessage(ctx context.Context, seq int64, deadLetter bool) (*azservicebus.ReceivedMessage, error) {
	var receiver SDKReceiver // Changed from *azservicebus.Receiver
	var err error

	if deadLetter {
		if c.DlqQueueName != "" {
			receiver, err = c.AzClient.NewReceiverForQueue(c.DlqQueueName, nil)
		} else if c.DlqSubName != "" {
			receiver, err = c.AzClient.NewReceiverForQueue(c.DlqSubName, nil) // Assuming same as above for DLQs
		} else {
			return nil, fmt.Errorf("no DLQ configured")
		}
	} else {
		if c.QueueName != "" {
			receiver, err = c.AzClient.NewReceiverForQueue(c.QueueName, nil)
		} else if c.TopicName != "" && c.Subscription != "" {
			receiver, err = c.AzClient.NewReceiverForSubscription(c.TopicName, c.Subscription, nil)
		} else {
			return nil, fmt.Errorf("no queue or topic/subscription configured")
		}
	}
	if err != nil {
		return nil, fmt.Errorf("error creating receiver: %w", err)
	}
	if receiver != nil {
		defer receiver.Close(ctx)
	}

	opts := &azservicebus.PeekMessagesOptions{
		FromSequenceNumber: &seq,
	}
	msgs, err := receiver.PeekMessages(ctx, 1, opts) // Call on SDKReceiver
	if err != nil {
		return nil, fmt.Errorf("peek error: %w", err)
	}
	if len(msgs) == 0 {
		return nil, fmt.Errorf("message %d not found", seq)
	}
	return msgs[0], nil
}
