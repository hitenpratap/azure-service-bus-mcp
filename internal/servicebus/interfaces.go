package servicebus

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
)

// SDKReceiver is an interface wrapper for *azservicebus.Receiver methods used by this package.
type SDKReceiver interface {
	PeekMessages(ctx context.Context, maxMessageCount int32, options *azservicebus.PeekMessagesOptions) ([]*azservicebus.ReceivedMessage, error)
	Close(ctx context.Context) error
	// Add other methods from *azservicebus.Receiver if needed by other functions
}

// SDKClient is an interface wrapper for *azservicebus.Client methods used by this package.
type SDKClient interface {
	NewReceiverForQueue(queueName string, options *azservicebus.ReceiverOptions) (SDKReceiver, error)
	NewReceiverForSubscription(topicName string, subscriptionName string, options *azservicebus.ReceiverOptions) (SDKReceiver, error)
	// Add other methods from *azservicebus.Client if needed by other functions
}

// --- Production Implementations ---

// sbReceiverWrapper implements SDKReceiver using a real *azservicebus.Receiver.
type sbReceiverWrapper struct {
	receiver *azservicebus.Receiver
}

func (r *sbReceiverWrapper) PeekMessages(ctx context.Context, maxMessageCount int32, options *azservicebus.PeekMessagesOptions) ([]*azservicebus.ReceivedMessage, error) {
	// Attempting to cast maxMessageCount to int to match the compiler's apparent expectation in the environment.
	// This contradicts SDK documentation but aims to resolve the build error.
	return r.receiver.PeekMessages(ctx, int(maxMessageCount), options)
}

func (r *sbReceiverWrapper) Close(ctx context.Context) error {
	return r.receiver.Close(ctx)
}

// sbClientWrapper implements SDKClient using a real *azservicebus.Client.
type sbClientWrapper struct {
	client *azservicebus.Client
}

func (c *sbClientWrapper) NewReceiverForQueue(queueName string, options *azservicebus.ReceiverOptions) (SDKReceiver, error) {
	receiver, err := c.client.NewReceiverForQueue(queueName, options)
	if err != nil {
		return nil, err
	}
	return &sbReceiverWrapper{receiver: receiver}, nil
}

func (c *sbClientWrapper) NewReceiverForSubscription(topicName string, subscriptionName string, options *azservicebus.ReceiverOptions) (SDKReceiver, error) {
	receiver, err := c.client.NewReceiverForSubscription(topicName, subscriptionName, options)
	if err != nil {
		return nil, err
	}
	return &sbReceiverWrapper{receiver: receiver}, nil
}

// NewSBClientWrapper is a constructor for sbClientWrapper.
func NewSBClientWrapper(azClient *azservicebus.Client) SDKClient {
	return &sbClientWrapper{client: azClient}
}
