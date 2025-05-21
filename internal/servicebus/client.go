package servicebus

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/spf13/viper"
)

// Client wraps the Azure Service Bus client and entity names
type Client struct {
	rawClient    *azservicebus.Client
	queueName    string
	dlqQueueName string
	topicName    string
	subscription string
	dlqSubName   string
}

// NewClient reads connection info from config and returns a Client
func NewClient(configPath string) (*Client, error) {
	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}

	connStr := viper.GetString("azure.servicebus.connectionString")
	queue := viper.GetString("azure.servicebus.queueName")
	topic := viper.GetString("azure.servicebus.topicName")
	sub := viper.GetString("azure.servicebus.subscriptionName")

	dlq := ""
	dlqSub := ""
	if queue != "" {
		dlq = fmt.Sprintf("%s/$DeadLetterQueue", queue)
	}
	if topic != "" && sub != "" {
		dlqSub = fmt.Sprintf("%s/Subscriptions/%s/$DeadLetterQueue", topic, sub)
	}

	raw, err := azservicebus.NewClientFromConnectionString(connStr, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating service bus client: %w", err)
	}

	return &Client{
		rawClient:    raw,
		queueName:    queue,
		dlqQueueName: dlq,
		topicName:    topic,
		subscription: sub,
		dlqSubName:   dlqSub,
	}, nil
}
