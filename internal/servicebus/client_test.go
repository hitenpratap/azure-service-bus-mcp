package servicebus

import (
	"fmt"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAzServiceClient is a mock type for the AzServiceClient type
// as defined in client.go
type MockAzServiceClient struct {
	mock.Mock
}

// NewClientFromConnectionString is a mock implementation of AzServiceClient.NewClientFromConnectionString
func (m *MockAzServiceClient) NewClientFromConnectionString(connectionString string, options *azservicebus.ClientOptions) (*azservicebus.Client, error) {
	args := m.Called(connectionString, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	client, ok := args.Get(0).(*azservicebus.Client)
	if !ok && args.Get(0) != nil {
		// This can happen if the mock is set up to return something other than *azservicebus.Client
		panic("MockAzServiceClient.NewClientFromConnectionString was called with a mock that did not return an *azservicebus.Client or nil")
	}
	return client, args.Error(1)
}

func TestNewClient(t *testing.T) {
	// Backup original function and defer restoration
	originalFunc := newAzServiceClientFunc
	defer func() { newAzServiceClientFunc = originalFunc }()

	// Create a dummy azservicebus.Client to be returned by the mock
	// Note: In a real scenario, you might need a more sophisticated mock if methods on this client are called.
	// For NewClient, we just need to ensure it's successfully passed through.
	dummyAzClient := &azservicebus.Client{}

	t.Run("Successful client creation", func(t *testing.T) {
		mockProvider := new(MockAzServiceClient)
		newAzServiceClientFunc = func() AzServiceClient {
			return mockProvider
		}

		// Expect call to NewClientFromConnectionString and return mock client
		mockProvider.On("NewClientFromConnectionString", "test-connection-string", (*azservicebus.ClientOptions)(nil)).Return(dummyAzClient, nil)

		// Create a temporary valid config file
		configContent := `
serviceBus:
  connectionString: "test-connection-string"
  queueName: "test-queue"
  topicName: "test-topic"
  subscriptionName: "test-subscription"
`
		tmpFile, err := os.CreateTemp("", "config-*.yaml")
		assert.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.Write([]byte(configContent))
		assert.NoError(t, err)
		err = tmpFile.Close()
		assert.NoError(t, err)

		client, err := NewClient(tmpFile.Name())
		assert.NoError(t, err)
		assert.NotNil(t, client)
		assert.Equal(t, "test-queue", client.QueueName)
		assert.Equal(t, "test-topic", client.TopicName)
		assert.Equal(t, "test-subscription", client.Subscription)
		
		// Assert that AzClient is not nil and is of the expected wrapper type
		assert.NotNil(t, client.AzClient)
		assert.IsType(t, (*sbClientWrapper)(nil), client.AzClient, "client.AzClient should be of type *sbClientWrapper")

		// Ensure the mock provider's NewClientFromConnectionString was called with the correct parameters
		// and returned the dummyAzClient (which was then wrapped).
		mockProvider.AssertExpectations(t)
	})

	t.Run("Invalid config file path", func(t *testing.T) {
		mockProvider := new(MockAzServiceClient)
		newAzServiceClientFunc = func() AzServiceClient {
			return mockProvider
		}
		// No call to NewClientFromConnectionString is expected as it should fail before that

		client, err := NewClient("non-existent-config.yaml")
		assert.Error(t, err)
		assert.Nil(t, client)
		// The error comes from viper.ReadInConfig() which wraps os.Open()
		assert.Contains(t, err.Error(), "error reading config:")
		assert.Contains(t, err.Error(), "no such file or directory")


		mockProvider.AssertExpectations(t) // Should have no expectations as the provider is not used.
	})

	t.Run("Malformed config file - YAML error", func(t *testing.T) {
		mockProvider := new(MockAzServiceClient)
		newAzServiceClientFunc = func() AzServiceClient {
			return mockProvider
		}

		// Create a temporary malformed config file
		configContent := `
serviceBus:
  connectionString: "test-connection-string"
  queueName: "test-queue
` // Malformed YAML - missing closing quote
		tmpFile, err := os.CreateTemp("", "config-*.yaml")
		assert.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.Write([]byte(configContent))
		assert.NoError(t, err)
		err = tmpFile.Close()
		assert.NoError(t, err)

		client, err := NewClient(tmpFile.Name())
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "error reading config:") // Viper error
		assert.Contains(t, err.Error(), "yaml:") // YAML parsing error indication

		mockProvider.AssertExpectations(t)
	})

	t.Run("Missing connection string in config", func(t *testing.T) {
		mockProvider := new(MockAzServiceClient)
		newAzServiceClientFunc = func() AzServiceClient {
			return mockProvider
		}

		// Create a temporary config file missing the connection string
		configContent := `
serviceBus:
  queueName: "test-queue"
`
		tmpFile, err := os.CreateTemp("", "config-*.yaml")
		assert.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.Write([]byte(configContent))
		assert.NoError(t, err)
		err = tmpFile.Close()
		assert.NoError(t, err)

		client, err := NewClient(tmpFile.Name())
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.EqualError(t, err, "serviceBus.connectionString is required")

		mockProvider.AssertExpectations(t)
	})

	t.Run("Error from NewClientFromConnectionString", func(t *testing.T) {
		mockProvider := new(MockAzServiceClient)
		newAzServiceClientFunc = func() AzServiceClient {
			return mockProvider
		}

		// Expect call to NewClientFromConnectionString and return an error
		expectedErr := fmt.Errorf("azure connection error")
		mockProvider.On("NewClientFromConnectionString", "test-connection-string-err", (*azservicebus.ClientOptions)(nil)).Return(nil, expectedErr)

		// Create a temporary valid config file
		configContent := `
serviceBus:
  connectionString: "test-connection-string-err"
  queueName: "test-queue"
`
		tmpFile, err := os.CreateTemp("", "config-*.yaml")
		assert.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.Write([]byte(configContent))
		assert.NoError(t, err)
		err = tmpFile.Close()
		assert.NoError(t, err)

		client, err := NewClient(tmpFile.Name())
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.EqualError(t, err, "failed to create Azure Service Bus client: azure connection error")

		mockProvider.AssertExpectations(t)
	})
}
