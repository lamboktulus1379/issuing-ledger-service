package tulustech_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/clients/tulustech"
)

// TestNewTulusHost tests the creation of a new TulusHost
func TestNewTulusHost(t *testing.T) {
	// This is a simple test to ensure the function exists and returns an object
	host := tulustech.NewTulusHost("https://example.com")
	assert.NotNil(t, host)
}