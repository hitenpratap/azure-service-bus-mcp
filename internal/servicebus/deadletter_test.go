package servicebus

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/stretchr/testify/assert"
	// "github.com/stretchr/testify/mock" // Removed as it might be indirectly used via mocks from messages_test.go
)

// createDLQMsg is a helper for creating azservicebus.ReceivedMessage for tests
func createDLQMsg(seq int64, ts time.Time) *azservicebus.ReceivedMessage {
	return &azservicebus.ReceivedMessage{SequenceNumber: to.Ptr(seq), EnqueuedTime: to.Ptr(ts)}
}

func TestListDeadLetters_Unit(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	oneHourAgo := now.Add(-time.Hour)
	twoHoursAgo := now.Add(-2 * time.Hour)
	oneHourLater := now.Add(time.Hour)
	twoHoursLater := now.Add(2 * time.Hour)

	t.Run("Client not configured for DLQ", func(t *testing.T) {
		client := &Client{} // No DlqQueueName or DlqSubName
		_, err := client.ListDeadLetters(ctx, nil, nil)
		assert.Error(t, err)
		assert.EqualError(t, err, "no DLQ configured")
	})

	testCases := []struct {
		name                string
		dlqQueueName        string // For queue DLQ
		dlqSubName          string // For subscription DLQ
		fromTime            *time.Time
		toTime              *time.Time
		setupMock           func(t *testing.T, mockSDKClient *MockSDKClient, mockSDKReceiver *MockSDKReceiver)
		expectedMessages    []DeadLetterInfo
		expectedError       string
		expectReceiverClose bool
	}{
		{
			name:         "Queue DLQ: Successful, all match (no filter)",
			dlqQueueName: "test-queue/$DeadLetterQueue",
			setupMock: func(t *testing.T, mockSDKClient *MockSDKClient, mockSDKReceiver *MockSDKReceiver) {
				mockSDKClient.On("NewReceiverForQueue", "test-queue/$DeadLetterQueue", (*azservicebus.ReceiverOptions)(nil)).Return(mockSDKReceiver, nil).Once()
				mockSDKReceiver.On("PeekMessages", ctx, int32(100), (*azservicebus.PeekMessagesOptions)(nil)).Return([]*azservicebus.ReceivedMessage{createDLQMsg(1, oneHourAgo), createDLQMsg(2, now)}, nil).Once()
				mockSDKReceiver.On("PeekMessages", ctx, int32(100), (*azservicebus.PeekMessagesOptions)(nil)).Return([]*azservicebus.ReceivedMessage{}, nil).Once() // End loop
			},
			expectedMessages:    []DeadLetterInfo{{SequenceNumber: 1, EnqueuedTime: oneHourAgo}, {SequenceNumber: 2, EnqueuedTime: now}},
			expectReceiverClose: true,
		},
		{
			name:       "Subscription DLQ: Successful, some match filter",
			dlqSubName: "test-topic/Subscriptions/test-sub/$DeadLetterQueue",
			fromTime:   &oneHourAgo,
			toTime:     &now,
			setupMock: func(t *testing.T, mockSDKClient *MockSDKClient, mockSDKReceiver *MockSDKReceiver) {
				mockSDKClient.On("NewReceiverForQueue", "test-topic/Subscriptions/test-sub/$DeadLetterQueue", (*azservicebus.ReceiverOptions)(nil)).Return(mockSDKReceiver, nil).Once()
				mockSDKReceiver.On("PeekMessages", ctx, int32(100), (*azservicebus.PeekMessagesOptions)(nil)).Return([]*azservicebus.ReceivedMessage{createDLQMsg(1, twoHoursAgo), createDLQMsg(2, oneHourAgo), createDLQMsg(3, now)}, nil).Once()
				mockSDKReceiver.On("PeekMessages", ctx, int32(100), (*azservicebus.PeekMessagesOptions)(nil)).Return([]*azservicebus.ReceivedMessage{}, nil).Once()
			},
			expectedMessages:    []DeadLetterInfo{{SequenceNumber: 2, EnqueuedTime: oneHourAgo}, {SequenceNumber: 3, EnqueuedTime: now}},
			expectReceiverClose: true,
		},
		{
			name:         "Queue DLQ: NewReceiverForQueue returns error",
			dlqQueueName: "test-queue-fail/$DeadLetterQueue",
			setupMock: func(t *testing.T, mockSDKClient *MockSDKClient, mockSDKReceiver *MockSDKReceiver) {
				mockSDKClient.On("NewReceiverForQueue", "test-queue-fail/$DeadLetterQueue", (*azservicebus.ReceiverOptions)(nil)).Return(nil, fmt.Errorf("receiver create error")).Once()
			},
			expectedError:       "error creating DLQ receiver: receiver create error",
			expectReceiverClose: false,
		},
		{
			name:       "Subscription DLQ: NewReceiverForQueue returns error",
			dlqSubName: "test-sub-fail/$DeadLetterQueue",
			setupMock: func(t *testing.T, mockSDKClient *MockSDKClient, mockSDKReceiver *MockSDKReceiver) {
				mockSDKClient.On("NewReceiverForQueue", "test-sub-fail/$DeadLetterQueue", (*azservicebus.ReceiverOptions)(nil)).Return(nil, fmt.Errorf("sub receiver create error")).Once()
			},
			expectedError:       "error creating DLQ receiver: sub receiver create error",
			expectReceiverClose: false,
		},
		{
			name:         "Queue DLQ: PeekMessages returns error",
			dlqQueueName: "test-queue-peek-fail/$DeadLetterQueue",
			setupMock: func(t *testing.T, mockSDKClient *MockSDKClient, mockSDKReceiver *MockSDKReceiver) {
				mockSDKClient.On("NewReceiverForQueue", "test-queue-peek-fail/$DeadLetterQueue", (*azservicebus.ReceiverOptions)(nil)).Return(mockSDKReceiver, nil).Once()
				mockSDKReceiver.On("PeekMessages", ctx, int32(100), (*azservicebus.PeekMessagesOptions)(nil)).Return(nil, fmt.Errorf("peek error")).Once()
			},
			expectedError:       "DLQ receive error: peek error",
			expectReceiverClose: true,
		},
		{
			name:         "Queue DLQ: Empty",
			dlqQueueName: "empty-dlq/$DeadLetterQueue",
			setupMock: func(t *testing.T, mockSDKClient *MockSDKClient, mockSDKReceiver *MockSDKReceiver) {
				mockSDKClient.On("NewReceiverForQueue", "empty-dlq/$DeadLetterQueue", (*azservicebus.ReceiverOptions)(nil)).Return(mockSDKReceiver, nil).Once()
				mockSDKReceiver.On("PeekMessages", ctx, int32(100), (*azservicebus.PeekMessagesOptions)(nil)).Return([]*azservicebus.ReceivedMessage{}, nil).Once()
			},
			expectedMessages:    []DeadLetterInfo{},
			expectReceiverClose: true,
		},
		{
			name:         "Queue DLQ: Messages present, none match time filter",
			dlqQueueName: "test-queue-none-match/$DeadLetterQueue",
			fromTime:     &oneHourLater,
			toTime:       &twoHoursLater,
			setupMock: func(t *testing.T, mockSDKClient *MockSDKClient, mockSDKReceiver *MockSDKReceiver) {
				mockSDKClient.On("NewReceiverForQueue", "test-queue-none-match/$DeadLetterQueue", (*azservicebus.ReceiverOptions)(nil)).Return(mockSDKReceiver, nil).Once()
				mockSDKReceiver.On("PeekMessages", ctx, int32(100), (*azservicebus.PeekMessagesOptions)(nil)).Return([]*azservicebus.ReceivedMessage{createDLQMsg(1, twoHoursAgo), createDLQMsg(2, oneHourAgo)}, nil).Once()
				mockSDKReceiver.On("PeekMessages", ctx, int32(100), (*azservicebus.PeekMessagesOptions)(nil)).Return([]*azservicebus.ReceivedMessage{}, nil).Once()
			},
			expectedMessages:    []DeadLetterInfo{},
			expectReceiverClose: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockSDKClient := new(MockSDKClient) // From messages_test.go (shared in package)
			mockSDKReceiver := new(MockSDKReceiver) // From messages_test.go (shared in package)

			client := &Client{AzClient: mockSDKClient, DlqQueueName: tc.dlqQueueName, DlqSubName: tc.dlqSubName}
			
			tc.setupMock(t, mockSDKClient, mockSDKReceiver)
			if tc.expectReceiverClose {
				mockSDKReceiver.On("Close", ctx).Return(nil).Once()
			}

			messages, err := client.ListDeadLetters(ctx, tc.fromTime, tc.toTime)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				assert.Nil(t, messages)
			} else {
				assert.NoError(t, err)
				if len(tc.expectedMessages) == 0 {
					assert.Empty(t, messages)
				} else {
					assert.Equal(t, len(tc.expectedMessages), len(messages))
					for i := range tc.expectedMessages {
						assert.Equal(t, tc.expectedMessages[i].SequenceNumber, messages[i].SequenceNumber)
						assert.True(t, tc.expectedMessages[i].EnqueuedTime.Equal(messages[i].EnqueuedTime))
					}
				}
			}
			mockSDKClient.AssertExpectations(t)
			mockSDKReceiver.AssertExpectations(t)
		})
	}
}


func TestFetchMessage_Unit(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	targetSequenceNumber := int64(123)
	messageToReturn := &azservicebus.ReceivedMessage{
		SequenceNumber: to.Ptr(targetSequenceNumber),
		EnqueuedTime:   to.Ptr(now),
		Body:           []byte("hello world"),
	}

	type clientConfig struct {
		QueueName        string
		TopicName        string
		SubscriptionName string
		DlqQueueName     string
		DlqSubName       string
	}

	testCases := []struct {
		name                string
		fetchDeadLetter     bool
		sequenceNumber      int64
		clientSetup         clientConfig
		setupMock           func(t *testing.T, mockSDKClient *MockSDKClient, mockSDKReceiver *MockSDKReceiver, clientCfg clientConfig, fetchDL bool, seqNum int64)
		expectedMessage     *azservicebus.ReceivedMessage
		expectedError       string
		expectReceiverClose bool
	}{
		{
			name:            "Queue: Found",
			fetchDeadLetter: false,
			sequenceNumber:  targetSequenceNumber,
			clientSetup:     clientConfig{QueueName: "my-queue"},
			setupMock: func(t *testing.T, mockSDKClient *MockSDKClient, mockSDKReceiver *MockSDKReceiver, clientCfg clientConfig, fetchDL bool, seqNum int64) {
				mockSDKClient.On("NewReceiverForQueue", clientCfg.QueueName, (*azservicebus.ReceiverOptions)(nil)).Return(mockSDKReceiver, nil).Once()
				opts := &azservicebus.PeekMessagesOptions{FromSequenceNumber: &seqNum}
				mockSDKReceiver.On("PeekMessages", ctx, int32(1), opts).Return([]*azservicebus.ReceivedMessage{messageToReturn}, nil).Once()
			},
			expectedMessage:     messageToReturn,
			expectReceiverClose: true,
		},
		{
			name:            "Topic/Sub: Found",
			fetchDeadLetter: false,
			sequenceNumber:  targetSequenceNumber,
			clientSetup:     clientConfig{TopicName: "my-topic", SubscriptionName: "my-sub"},
			setupMock: func(t *testing.T, mockSDKClient *MockSDKClient, mockSDKReceiver *MockSDKReceiver, clientCfg clientConfig, fetchDL bool, seqNum int64) {
				mockSDKClient.On("NewReceiverForSubscription", clientCfg.TopicName, clientCfg.SubscriptionName, (*azservicebus.ReceiverOptions)(nil)).Return(mockSDKReceiver, nil).Once()
				opts := &azservicebus.PeekMessagesOptions{FromSequenceNumber: &seqNum}
				mockSDKReceiver.On("PeekMessages", ctx, int32(1), opts).Return([]*azservicebus.ReceivedMessage{messageToReturn}, nil).Once()
			},
			expectedMessage:     messageToReturn,
			expectReceiverClose: true,
		},
		{
			name:            "Queue DLQ: Found",
			fetchDeadLetter: true,
			sequenceNumber:  targetSequenceNumber,
			clientSetup:     clientConfig{DlqQueueName: "my-queue/$DeadLetterQueue"},
			setupMock: func(t *testing.T, mockSDKClient *MockSDKClient, mockSDKReceiver *MockSDKReceiver, clientCfg clientConfig, fetchDL bool, seqNum int64) {
				mockSDKClient.On("NewReceiverForQueue", clientCfg.DlqQueueName, (*azservicebus.ReceiverOptions)(nil)).Return(mockSDKReceiver, nil).Once()
				opts := &azservicebus.PeekMessagesOptions{FromSequenceNumber: &seqNum}
				mockSDKReceiver.On("PeekMessages", ctx, int32(1), opts).Return([]*azservicebus.ReceivedMessage{messageToReturn}, nil).Once()
			},
			expectedMessage:     messageToReturn,
			expectReceiverClose: true,
		},
		{
			name:            "Subscription DLQ: Found",
			fetchDeadLetter: true,
			sequenceNumber:  targetSequenceNumber,
			clientSetup:     clientConfig{DlqSubName: "my-topic/Subscriptions/my-sub/$DeadLetterQueue"},
			setupMock: func(t *testing.T, mockSDKClient *MockSDKClient, mockSDKReceiver *MockSDKReceiver, clientCfg clientConfig, fetchDL bool, seqNum int64) {
				mockSDKClient.On("NewReceiverForQueue", clientCfg.DlqSubName, (*azservicebus.ReceiverOptions)(nil)).Return(mockSDKReceiver, nil).Once()
				opts := &azservicebus.PeekMessagesOptions{FromSequenceNumber: &seqNum}
				mockSDKReceiver.On("PeekMessages", ctx, int32(1), opts).Return([]*azservicebus.ReceivedMessage{messageToReturn}, nil).Once()
			},
			expectedMessage:     messageToReturn,
			expectReceiverClose: true,
		},
		{
			name:            "Queue: Not Found",
			fetchDeadLetter: false,
			sequenceNumber:  targetSequenceNumber,
			clientSetup:     clientConfig{QueueName: "my-queue"},
			setupMock: func(t *testing.T, mockSDKClient *MockSDKClient, mockSDKReceiver *MockSDKReceiver, clientCfg clientConfig, fetchDL bool, seqNum int64) {
				mockSDKClient.On("NewReceiverForQueue", clientCfg.QueueName, (*azservicebus.ReceiverOptions)(nil)).Return(mockSDKReceiver, nil).Once()
				opts := &azservicebus.PeekMessagesOptions{FromSequenceNumber: &seqNum}
				mockSDKReceiver.On("PeekMessages", ctx, int32(1), opts).Return([]*azservicebus.ReceivedMessage{}, nil).Once() // Empty slice
			},
			expectedError:       fmt.Sprintf("message %d not found", targetSequenceNumber),
			expectReceiverClose: true,
		},
		{
			name:            "Queue: NewReceiverForQueue error",
			fetchDeadLetter: false,
			sequenceNumber:  targetSequenceNumber,
			clientSetup:     clientConfig{QueueName: "my-queue-fail"},
			setupMock: func(t *testing.T, mockSDKClient *MockSDKClient, mockSDKReceiver *MockSDKReceiver, clientCfg clientConfig, fetchDL bool, seqNum int64) {
				mockSDKClient.On("NewReceiverForQueue", clientCfg.QueueName, (*azservicebus.ReceiverOptions)(nil)).Return(nil, fmt.Errorf("receiver create error")).Once()
			},
			expectedError:       "error creating receiver: receiver create error",
			expectReceiverClose: false,
		},
		{
			name:            "Topic/Sub: NewReceiverForSubscription error",
			fetchDeadLetter: false,
			sequenceNumber:  targetSequenceNumber,
			clientSetup:     clientConfig{TopicName: "my-topic-fail", SubscriptionName: "my-sub-fail"},
			setupMock: func(t *testing.T, mockSDKClient *MockSDKClient, mockSDKReceiver *MockSDKReceiver, clientCfg clientConfig, fetchDL bool, seqNum int64) {
				mockSDKClient.On("NewReceiverForSubscription", clientCfg.TopicName, clientCfg.SubscriptionName, (*azservicebus.ReceiverOptions)(nil)).Return(nil, fmt.Errorf("sub receiver create error")).Once()
			},
			expectedError:       "error creating receiver: sub receiver create error",
			expectReceiverClose: false,
		},
		{
			name:            "Queue: PeekMessages error",
			fetchDeadLetter: false,
			sequenceNumber:  targetSequenceNumber,
			clientSetup:     clientConfig{QueueName: "my-queue-peek-fail"},
			setupMock: func(t *testing.T, mockSDKClient *MockSDKClient, mockSDKReceiver *MockSDKReceiver, clientCfg clientConfig, fetchDL bool, seqNum int64) {
				mockSDKClient.On("NewReceiverForQueue", clientCfg.QueueName, (*azservicebus.ReceiverOptions)(nil)).Return(mockSDKReceiver, nil).Once()
				opts := &azservicebus.PeekMessagesOptions{FromSequenceNumber: &seqNum}
				mockSDKReceiver.On("PeekMessages", ctx, int32(1), opts).Return(nil, fmt.Errorf("peek error")).Once()
			},
			expectedError:       "peek error: peek error",
			expectReceiverClose: true,
		},
		{
			name:            "Client not configured (normal queue)",
			fetchDeadLetter: false,
			sequenceNumber:  targetSequenceNumber,
			clientSetup:     clientConfig{}, // No names configured
			setupMock:       func(t *testing.T, mockSDKClient *MockSDKClient, mockSDKReceiver *MockSDKReceiver, clientCfg clientConfig, fetchDL bool, seqNum int64) {},
			expectedError:       "no queue or topic/subscription configured",
			expectReceiverClose: false,
		},
		{
			name:            "Client not configured (DLQ)",
			fetchDeadLetter: true,
			sequenceNumber:  targetSequenceNumber,
			clientSetup:     clientConfig{}, // No DLQ names configured
			setupMock:       func(t *testing.T, mockSDKClient *MockSDKClient, mockSDKReceiver *MockSDKReceiver, clientCfg clientConfig, fetchDL bool, seqNum int64) {},
			expectedError:       "no DLQ configured",
			expectReceiverClose: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockSDKClient := new(MockSDKClient)
			mockSDKReceiver := new(MockSDKReceiver)

			client := &Client{
				AzClient:     mockSDKClient,
				QueueName:    tc.clientSetup.QueueName,
				TopicName:    tc.clientSetup.TopicName,
				Subscription: tc.clientSetup.SubscriptionName,
				DlqQueueName: tc.clientSetup.DlqQueueName,
				DlqSubName:   tc.clientSetup.DlqSubName,
			}
			
			tc.setupMock(t, mockSDKClient, mockSDKReceiver, tc.clientSetup, tc.fetchDeadLetter, tc.sequenceNumber)
			if tc.expectReceiverClose {
				mockSDKReceiver.On("Close", ctx).Return(nil).Once()
			}

			msg, err := client.FetchMessage(ctx, tc.sequenceNumber, tc.fetchDeadLetter)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				assert.Nil(t, msg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedMessage, msg)
			}
			mockSDKClient.AssertExpectations(t)
			mockSDKReceiver.AssertExpectations(t)
		})
	}
}
