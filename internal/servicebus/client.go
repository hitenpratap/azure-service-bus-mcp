package servicebus

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/spf13/viper"
)

// AzServiceClient an interface to allow mocking of Azure Service Bus client creation
type AzServiceClient interface {
	NewClientFromConnectionString(connectionString string, options *azservicebus.ClientOptions) (*azservicebus.Client, error)
}

// defaultAzServiceClient is the production implementation of AzServiceClient
type defaultAzServiceClient struct{}

// NewClientFromConnectionString calls the actual azservicebus.NewClientFromConnectionString
func (c *defaultAzServiceClient) NewClientFromConnectionString(connectionString string, options *azservicebus.ClientOptions) (*azservicebus.Client, error) {
	return azservicebus.NewClientFromConnectionString(connectionString, options)
}

var newAzServiceClientFunc func() AzServiceClient = func() AzServiceClient {
	return &defaultAzServiceClient{}
}

// Client wraps the Azure Service Bus client and entity names
type Client struct {
	AzClient     SDKClient // Changed from *azservicebus.Client to our SDKClient interface
	QueueName    string    // Made public for tests
	DlqQueueName string    // Made public for tests
	TopicName    string               // Made public for tests
	Subscription string               // Made public for tests
	DlqSubName   string               // Made public for tests
}

// NewClient reads connection info from config and returns a Client
func NewClient(configPath string) (*Client, error) {
	viper.Reset() // Reset viper for tests
	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}

	connStr := viper.GetString("serviceBus.connectionString") // Updated path to match test config
	if connStr == "" {
		return nil, fmt.Errorf("serviceBus.connectionString is required")
	}
	queue := viper.GetString("serviceBus.queueName") // Updated path to match test config
	topic := viper.GetString("serviceBus.topicName") // Updated path to match test config
	sub := viper.GetString("serviceBus.subscriptionName") // Updated path to match test config

	dlq := ""
	dlqSub := ""
	if queue != "" {
		dlq = fmt.Sprintf("%s/$DeadLetterQueue", queue)
	}
	if topic != "" && sub != "" {
		dlqSub = fmt.Sprintf("%s/Subscriptions/%s/$DeadLetterQueue", topic, sub)
	}

	azClientProvider := newAzServiceClientFunc()
	rawAzClient, err := azClientProvider.NewClientFromConnectionString(connStr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure Service Bus client: %w", err)
	}

	return &Client{
		AzClient:     NewSBClientWrapper(rawAzClient), // Wrap the raw client
		QueueName:    queue,
		DlqQueueName: dlq,
		TopicName:    topic,
		Subscription: sub,
		DlqSubName:   dlqSub,
	}, nil
}
