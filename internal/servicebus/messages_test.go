package servicebus

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSDKClient is a mock implementation of the SDKClient interface.
type MockSDKClient struct {
	mock.Mock
}

func (m *MockSDKClient) NewReceiverForQueue(queueName string, options *azservicebus.ReceiverOptions) (SDKReceiver, error) {
	args := m.Called(queueName, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(SDKReceiver), args.Error(1)
}

func (m *MockSDKClient) NewReceiverForSubscription(topicName string, subscriptionName string, options *azservicebus.ReceiverOptions) (SDKReceiver, error) {
	args := m.Called(topicName, subscriptionName, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(SDKReceiver), args.Error(1)
}

// MockSDKReceiver is a mock implementation of the SDKReceiver interface.
type MockSDKReceiver struct {
	mock.Mock
}

func (m *MockSDKReceiver) PeekMessages(ctx context.Context, maxMessageCount int32, options *azservicebus.PeekMessagesOptions) ([]*azservicebus.ReceivedMessage, error) {
	args := m.Called(ctx, maxMessageCount, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*azservicebus.ReceivedMessage), args.Error(1)
}

func (m *MockSDKReceiver) Close(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestListMessages_Unit(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	oneHourAgo := now.Add(-time.Hour)
	oneHourLater := now.Add(time.Hour)
	twoHoursAgo := now.Add(-2 * time.Hour)
	twoHoursLater := now.Add(2 * time.Hour)

	createMsg := func(seq int64, ts time.Time) *azservicebus.ReceivedMessage {
		return &azservicebus.ReceivedMessage{SequenceNumber: to.Ptr(seq), EnqueuedTime: to.Ptr(ts)}
	}

	t.Run("Client not configured", func(t *testing.T) {
		// For this specific case, AzClient can be nil as it's checked before AzClient is used.
		client := &Client{} 
		_, err := client.ListMessages(ctx, nil, nil)
		assert.Error(t, err)
		assert.EqualError(t, err, "no queue or topic/subscription configured")
	})

	testCases := []struct {
		name                string
		isQueue             bool
		queueName           string
		topicName           string
		subscriptionName    string
		fromTime            *time.Time
		toTime              *time.Time
		setupMock           func(t *testing.T, mockSDKClient *MockSDKClient, mockSDKReceiver *MockSDKReceiver)
		expectedMessages    []MessageInfo
		expectedError       string
		expectReceiverClose bool
	}{
		{
			name:      "Queue: Successful, all match (no filter)",
			isQueue:   true,
			queueName: "test-queue",
			setupMock: func(t *testing.T, mockSDKClient *MockSDKClient, mockSDKReceiver *MockSDKReceiver) {
				mockSDKClient.On("NewReceiverForQueue", "test-queue", (*azservicebus.ReceiverOptions)(nil)).Return(mockSDKReceiver, nil).Once()
				mockSDKReceiver.On("PeekMessages", ctx, int32(100), (*azservicebus.PeekMessagesOptions)(nil)).Return([]*azservicebus.ReceivedMessage{createMsg(1, oneHourAgo), createMsg(2, now)}, nil).Once()
				mockSDKReceiver.On("PeekMessages", ctx, int32(100), (*azservicebus.PeekMessagesOptions)(nil)).Return([]*azservicebus.ReceivedMessage{}, nil).Once() // End loop
			},
			expectedMessages:    []MessageInfo{{SequenceNumber: 1, EnqueuedTime: oneHourAgo}, {SequenceNumber: 2, EnqueuedTime: now}},
			expectReceiverClose: true,
		},
		{
			name:             "Topic: Successful, some match filter",
			isQueue:          false,
			topicName:        "test-topic",
			subscriptionName: "test-sub",
			fromTime:         &oneHourAgo,
			toTime:           &now,
			setupMock: func(t *testing.T, mockSDKClient *MockSDKClient, mockSDKReceiver *MockSDKReceiver) {
				mockSDKClient.On("NewReceiverForSubscription", "test-topic", "test-sub", (*azservicebus.ReceiverOptions)(nil)).Return(mockSDKReceiver, nil).Once()
				mockSDKReceiver.On("PeekMessages", ctx, int32(100), (*azservicebus.PeekMessagesOptions)(nil)).Return([]*azservicebus.ReceivedMessage{createMsg(1, twoHoursAgo), createMsg(2, oneHourAgo), createMsg(3, now)}, nil).Once()
				mockSDKReceiver.On("PeekMessages", ctx, int32(100), (*azservicebus.PeekMessagesOptions)(nil)).Return([]*azservicebus.ReceivedMessage{}, nil).Once()
			},
			expectedMessages:    []MessageInfo{{SequenceNumber: 2, EnqueuedTime: oneHourAgo}, {SequenceNumber: 3, EnqueuedTime: now}},
			expectReceiverClose: true,
		},
		{
			name:      "Queue: NewReceiverForQueue returns error",
			isQueue:   true,
			queueName: "test-queue-fail",
			setupMock: func(t *testing.T, mockSDKClient *MockSDKClient, mockSDKReceiver *MockSDKReceiver) {
				mockSDKClient.On("NewReceiverForQueue", "test-queue-fail", (*azservicebus.ReceiverOptions)(nil)).Return(nil, fmt.Errorf("receiver create error")).Once()
			},
			expectedError:       "error creating receiver: receiver create error",
			expectReceiverClose: false,
		},
		{
			name:             "Topic: NewReceiverForSubscription returns error",
			isQueue:          false,
			topicName:        "test-topic-fail",
			subscriptionName: "test-sub-fail",
			setupMock: func(t *testing.T, mockSDKClient *MockSDKClient, mockSDKReceiver *MockSDKReceiver) {
				mockSDKClient.On("NewReceiverForSubscription", "test-topic-fail", "test-sub-fail", (*azservicebus.ReceiverOptions)(nil)).Return(nil, fmt.Errorf("sub receiver create error")).Once()
			},
			expectedError:       "error creating receiver: sub receiver create error",
			expectReceiverClose: false,
		},
		{
			name:      "Queue: PeekMessages returns error",
			isQueue:   true,
			queueName: "test-queue-peek-fail",
			setupMock: func(t *testing.T, mockSDKClient *MockSDKClient, mockSDKReceiver *MockSDKReceiver) {
				mockSDKClient.On("NewReceiverForQueue", "test-queue-peek-fail", (*azservicebus.ReceiverOptions)(nil)).Return(mockSDKReceiver, nil).Once()
				mockSDKReceiver.On("PeekMessages", ctx, int32(100), (*azservicebus.PeekMessagesOptions)(nil)).Return(nil, fmt.Errorf("peek error")).Once()
			},
			expectedError:       "receive error: peek error",
			expectReceiverClose: true,
		},
		{
			name:      "Queue: Empty queue",
			isQueue:   true,
			queueName: "empty-queue",
			setupMock: func(t *testing.T, mockSDKClient *MockSDKClient, mockSDKReceiver *MockSDKReceiver) {
				mockSDKClient.On("NewReceiverForQueue", "empty-queue", (*azservicebus.ReceiverOptions)(nil)).Return(mockSDKReceiver, nil).Once()
				mockSDKReceiver.On("PeekMessages", ctx, int32(100), (*azservicebus.PeekMessagesOptions)(nil)).Return([]*azservicebus.ReceivedMessage{}, nil).Once()
			},
			expectedMessages:    []MessageInfo{},
			expectReceiverClose: true,
		},
		{
			name:      "Queue: Messages present, none match time filter",
			isQueue:   true,
			queueName: "test-queue-none-match",
			fromTime:  &oneHourLater,
			toTime:    &twoHoursLater,
			setupMock: func(t *testing.T, mockSDKClient *MockSDKClient, mockSDKReceiver *MockSDKReceiver) {
				mockSDKClient.On("NewReceiverForQueue", "test-queue-none-match", (*azservicebus.ReceiverOptions)(nil)).Return(mockSDKReceiver, nil).Once()
				mockSDKReceiver.On("PeekMessages", ctx, int32(100), (*azservicebus.PeekMessagesOptions)(nil)).Return([]*azservicebus.ReceivedMessage{createMsg(1, twoHoursAgo), createMsg(2, oneHourAgo)}, nil).Once()
				mockSDKReceiver.On("PeekMessages", ctx, int32(100), (*azservicebus.PeekMessagesOptions)(nil)).Return([]*azservicebus.ReceivedMessage{}, nil).Once()
			},
			expectedMessages:    []MessageInfo{},
			expectReceiverClose: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockSDKClient := new(MockSDKClient)
			mockSDKReceiver := new(MockSDKReceiver)

			client := &Client{AzClient: mockSDKClient}
			if tc.isQueue {
				client.QueueName = tc.queueName
			} else {
				client.TopicName = tc.topicName
				client.Subscription = tc.subscriptionName
			}
			
			tc.setupMock(t, mockSDKClient, mockSDKReceiver)
			if tc.expectReceiverClose {
				mockSDKReceiver.On("Close", ctx).Return(nil).Once()
			}

			messages, err := client.ListMessages(ctx, tc.fromTime, tc.toTime)

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
						assert.True(t, tc.expectedMessages[i].EnqueuedTime.Equal(messages[i].EnqueuedTime), "Times should be equal for msg %d", tc.expectedMessages[i].SequenceNumber)
					}
				}
			}

			mockSDKClient.AssertExpectations(t)
			mockSDKReceiver.AssertExpectations(t)
		})
	}
}
