package cache_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/cache"
)

// TestNewTestCache tests the creation of a new TestCache
func TestNewTestCache(t *testing.T) {
	// This is a simple test to ensure the function exists and returns an object
	// We can't do much more without mocking the Redis client
	testCache := cache.NewTestCache(nil)
	assert.NotNil(t, testCache)
}