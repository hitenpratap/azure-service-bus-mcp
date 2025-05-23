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
	var receiver *azservicebus.Receiver
	var err error

	if c.dlqQueueName != "" {
		receiver, err = c.rawClient.NewReceiverForQueue(c.dlqQueueName, nil)
	} else if c.dlqSubName != "" {
		receiver, err = c.rawClient.NewReceiverForQueue(c.dlqSubName, nil)
	} else {
		return nil, fmt.Errorf("no DLQ configured")
	}
	if err != nil {
		return nil, fmt.Errorf("error creating DLQ receiver: %w", err)
	}
	defer receiver.Close(ctx)

	var result []DeadLetterInfo
	for {
		msgs, err := receiver.PeekMessages(ctx, 100, nil)
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
	var receiver *azservicebus.Receiver
	var err error

	if deadLetter {
		if c.dlqQueueName != "" {
			receiver, err = c.rawClient.NewReceiverForQueue(c.dlqQueueName, nil)
		} else if c.dlqSubName != "" {
			receiver, err = c.rawClient.NewReceiverForQueue(c.dlqSubName, nil)
		} else {
			return nil, fmt.Errorf("no DLQ configured")
		}
	} else {
		if c.queueName != "" {
			receiver, err = c.rawClient.NewReceiverForQueue(c.queueName, nil)
		} else if c.topicName != "" && c.subscription != "" {
			receiver, err = c.rawClient.NewReceiverForSubscription(c.topicName, c.subscription, nil)
		} else {
			return nil, fmt.Errorf("no queue or topic/subscription configured")
		}
	}
	if err != nil {
		return nil, fmt.Errorf("error creating receiver: %w", err)
	}
	defer receiver.Close(ctx)

	opts := &azservicebus.PeekMessagesOptions{
		FromSequenceNumber: &seq,
	}
	msgs, err := receiver.PeekMessages(ctx, 1, opts)
	if err != nil {
		return nil, fmt.Errorf("peek error: %w", err)
	}
	if len(msgs) == 0 {
		return nil, fmt.Errorf("message %d not found", seq)
	}
	return msgs[0], nil
}
