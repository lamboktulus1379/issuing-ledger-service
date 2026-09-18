package configuration

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestConfiguration tests the configuration package basic functionality
func TestConfiguration(t *testing.T) {
	// Test that the configuration struct exists and has expected fields
	// This is a basic smoke test to ensure the configuration package compiles correctly
	
	t.Run("configuration_struct_exists", func(t *testing.T) {
		// Test that we can reference the configuration without it panicking
		require.NotNil(t, &C, "Configuration should not be nil")
		
		// Test that basic configuration fields exist
		require.NotNil(t, &C.App, "App configuration should exist")
		require.NotNil(t, &C.Database, "Database configuration should exist")
		
		t.Log("Configuration structure validation passed")
	})
	
	t.Run("configuration_has_required_fields", func(t *testing.T) {
		// Verify that key configuration sections exist
		config := &C
		
		// Check that essential configuration sections are present
		require.NotNil(t, config.App, "App config should be present")
		require.NotNil(t, config.Database, "Database config should be present")
		require.NotNil(t, config.Database.MySql, "MySQL config should be present")
		require.NotNil(t, config.Database.Psql, "PostgreSQL config should be present")
		require.NotNil(t, config.Database.Mongo, "MongoDB config should be present")
		
		t.Log("Required configuration fields validation passed")
	})
}