package usecase_test

import (
	"cloud.google.com/go/pubsub"
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/clients/tulustech/models"
	"github.com/lamboktulus1379/issuing-ledger-service/usecase"
)

// Mock implementations
type MockTulusTechHost struct {
	mock.Mock
}

func (m *MockTulusTechHost) GetRandomTyping(reqHeader models.ReqHeader) (models.ResTypingRandom, error) {
	args := m.Called(reqHeader)
	return args.Get(0).(models.ResTypingRandom), args.Error(1)
}

type MockTestPubSub struct {
	mock.Mock
}

func (m *MockTestPubSub) Publish(ctx context.Context, topic string, payload []byte) (string, error) {
	args := m.Called(ctx, topic, payload)
	return args.String(0), args.Error(1)
}

func (m *MockTestPubSub) GetSubscription(subID string) (*pubsub.Subscription, error) {
	args := m.Called(subID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pubsub.Subscription), args.Error(1)
}

type MockTestServiceBus struct {
	mock.Mock
}

func (m *MockTestServiceBus) SendMessage(message []byte) error {
	args := m.Called(message)
	return args.Error(0)
}

func (m *MockTestServiceBus) GetMessage(count int) {
	m.Called(count)
}

type MockTestCache struct {
	mock.Mock
}

func (m *MockTestCache) Set(ctx context.Context, key string, value interface{}) {
	m.Called(ctx, key, value)
}

func (m *MockTestCache) Get(ctx context.Context, key string) (interface{}, error) {
	args := m.Called(ctx, key)
	return args.Get(0), args.Error(1)
}

func TestTestUsecase_Test(t *testing.T) {
	// Create mocks
	mockTulusTechHost := new(MockTulusTechHost)
	mockTestPubSub := new(MockTestPubSub)
	mockTestServiceBus := new(MockTestServiceBus)
	mockTestCache := new(MockTestCache)

	// Set up expectations
	msg := "Hello"
	byteMsg, _ := json.Marshal(msg)

	// PubSub expectations
	mockTestPubSub.On("Publish", mock.Anything, "topic", byteMsg).
		Return("message-id", nil).
		Once()

	// ServiceBus expectations
	mockTestServiceBus.On("SendMessage", byteMsg).
		Return(nil).
		Once()

	// Cache expectations
	mockTestCache.On("Set", mock.Anything, "test", "test").
		Once()
	mockTestCache.On("Get", mock.Anything, "test").
		Return("test", nil).
		Once()

	// TulusTechHost expectations
	mockTulusTechHost.On("GetRandomTyping", mock.AnythingOfType("models.ReqHeader")).
		Return(models.ResTypingRandom{
			ID:      "1",
			Author:  "Test Author",
			Content: "Test Content",
		}, nil).
		Once()

	// Create the usecase with mocks
	testUsecase := usecase.NewTestUsecase(
		mockTulusTechHost,
		mockTestPubSub,
		mockTestServiceBus,
		mockTestCache,
	)

	// Call the method
	result := testUsecase.Test(context.Background())

	// Assert the result
	assert.Equal(t, "OK", result.PubSub)
	assert.Equal(t, "OK", result.ServiceBus)
	assert.Equal(t, "test", result.Cache)
	assert.Equal(t, "OK", result.TulusTech)

	// Verify all expectations were met
	mockTulusTechHost.AssertExpectations(t)
	mockTestPubSub.AssertExpectations(t)
	mockTestServiceBus.AssertExpectations(t)
	mockTestCache.AssertExpectations(t)
}

func TestTestUsecase_Test_PubSubError(t *testing.T) {
	// Create mocks
	mockTulusTechHost := new(MockTulusTechHost)
	mockTestPubSub := new(MockTestPubSub)
	mockTestServiceBus := new(MockTestServiceBus)
	mockTestCache := new(MockTestCache)

	// Set up expectations
	msg := "Hello"
	byteMsg, _ := json.Marshal(msg)

	// PubSub expectations with error
	mockTestPubSub.On("Publish", mock.Anything, "topic", byteMsg).
		Return("", assert.AnError).
		Once()

	// No ServiceBus, Cache, or TulusTech expectations because the function returns early on PubSub error

	// Create the usecase with mocks
	testUsecase := usecase.NewTestUsecase(
		mockTulusTechHost,
		mockTestPubSub,
		mockTestServiceBus,
		mockTestCache,
	)

	// Call the method
	result := testUsecase.Test(context.Background())

	// Assert the result
	// The implementation now returns the error message when there's an error
	assert.Equal(t, assert.AnError.Error(), result.PubSub)

	// These assertions are not reached because the function returns early on PubSub error
	// But we'll keep them for completeness
	// assert.Equal(t, "OK", result.ServiceBus)
	// assert.Equal(t, "test", result.Cache)
	// assert.Equal(t, "OK", result.TulusTech)

	// Verify all expectations were met
	mockTulusTechHost.AssertExpectations(t)
	mockTestPubSub.AssertExpectations(t)
	mockTestServiceBus.AssertExpectations(t)
	mockTestCache.AssertExpectations(t)
}

func TestTestUsecase_Test_ServiceBusError(t *testing.T) {
	// Create mocks
	mockTulusTechHost := new(MockTulusTechHost)
	mockTestPubSub := new(MockTestPubSub)
	mockTestServiceBus := new(MockTestServiceBus)
	mockTestCache := new(MockTestCache)

	// Set up expectations
	msg := "Hello"
	byteMsg, _ := json.Marshal(msg)

	// PubSub expectations
	mockTestPubSub.On("Publish", mock.Anything, "topic", byteMsg).
		Return("message-id", nil).
		Once()

	// ServiceBus expectations with error
	mockTestServiceBus.On("SendMessage", byteMsg).
		Return(assert.AnError).
		Once()

	// No Cache or TulusTech expectations because the function returns early on ServiceBus error

	// Create the usecase with mocks
	testUsecase := usecase.NewTestUsecase(
		mockTulusTechHost,
		mockTestPubSub,
		mockTestServiceBus,
		mockTestCache,
	)

	// Call the method
	result := testUsecase.Test(context.Background())

	// Assert the result
	assert.Equal(t, "OK", result.PubSub)
	// The implementation now returns the error message when there's an error
	assert.Equal(t, assert.AnError.Error(), result.ServiceBus)

	// These assertions are not reached because the function returns early on ServiceBus error
	// But we'll keep them for completeness
	// assert.Equal(t, "test", result.Cache)
	// assert.Equal(t, "OK", result.TulusTech)

	// Verify all expectations were met
	mockTulusTechHost.AssertExpectations(t)
	mockTestPubSub.AssertExpectations(t)
	mockTestServiceBus.AssertExpectations(t)
	mockTestCache.AssertExpectations(t)
}
