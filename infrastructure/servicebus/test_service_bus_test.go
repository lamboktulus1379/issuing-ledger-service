package servicebus_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/servicebus"
)

// TestNewTestServiceBus tests the creation of a new TestServiceBus
func TestNewTestServiceBus(t *testing.T) {
	// This is a simple test to ensure the function exists and returns an object
	// We can't do much more without mocking the Azure Service Bus client
	testServiceBus := servicebus.NewTestServiceBus(nil)
	assert.NotNil(t, testServiceBus)
}
