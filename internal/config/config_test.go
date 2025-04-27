package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Save current environment variables
	oldEnv := os.Environ()
	
	// Restore environment variables after test
	defer func() {
		os.Clearenv()
		for _, env := range oldEnv {
			key, value, _ := splitEnv(env)
			os.Setenv(key, value)
		}
	}()
	
	// Clear environment and set test values
	os.Clearenv()
	os.Setenv("MQTT_DEFAULT_CONNECTION", "test")
	os.Setenv("MQTT_TEST_HOST", "localhost")
	os.Setenv("MQTT_TEST_PORT", "1883")
	os.Setenv("MQTT_TEST_CLIENT_ID", "test-client")
	os.Setenv("MQTT_TEST_CLEAN_SESSION", "true")
	os.Setenv("MQTT_TLS_ENABLED", "false")
	
	// Load configuration
	cfg, err := LoadConfig()
	
	// Verify results
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if cfg.DefaultConnection != "test" {
		t.Errorf("Expected DefaultConnection to be 'test', got '%s'", cfg.DefaultConnection)
	}
	
	broker, exists := cfg.Brokers["test"]
	if !exists {
		t.Fatal("Expected broker 'test' to exist")
	}
	
	if broker.Host != "localhost" {
		t.Errorf("Expected Host to be 'localhost', got '%s'", broker.Host)
	}
	
	if broker.Port != 1883 {
		t.Errorf("Expected Port to be 1883, got %d", broker.Port)
	}
	
	if broker.ClientID != "test-client" {
		t.Errorf("Expected ClientID to be 'test-client', got '%s'", broker.ClientID)
	}
	
	if !broker.CleanSession {
		t.Error("Expected CleanSession to be true")
	}
	
	if broker.TLSEnabled {
		t.Error("Expected TLSEnabled to be false")
	}
}

func TestGetBrokerConfig(t *testing.T) {
	// Create test configuration
	cfg := &Config{
		DefaultConnection: "default",
		Brokers: map[string]*BrokerConfig{
			"default": {
				Name:     "default",
				Host:     "localhost",
				Port:     1883,
				ClientID: "default-client",
			},
			"secondary": {
				Name:     "secondary",
				Host:     "example.com",
				Port:     8883,
				ClientID: "secondary-client",
			},
		},
	}
	
	// Test getting default broker
	broker, err := cfg.GetBrokerConfig("")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if broker.Name != "default" {
		t.Errorf("Expected Name to be 'default', got '%s'", broker.Name)
	}
	
	// Test getting specific broker
	broker, err = cfg.GetBrokerConfig("secondary")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if broker.Name != "secondary" {
		t.Errorf("Expected Name to be 'secondary', got '%s'", broker.Name)
	}
	
	// Test getting non-existent broker
	_, err = cfg.GetBrokerConfig("nonexistent")
	if err == nil {
		t.Fatal("Expected error for non-existent broker, got nil")
	}
}

func TestBrokerConfigValidate(t *testing.T) {
	// Test valid configuration
	validConfig := &BrokerConfig{
		Name:     "test",
		Host:     "localhost",
		Port:     1883,
		ClientID: "test-client",
	}
	
	if err := validConfig.Validate(); err != nil {
		t.Errorf("Expected no error for valid config, got %v", err)
	}
	
	// Test missing host
	invalidConfig := &BrokerConfig{
		Name:     "test",
		Port:     1883,
		ClientID: "test-client",
	}
	
	if err := invalidConfig.Validate(); err == nil {
		t.Error("Expected error for missing host, got nil")
	}
	
	// Test missing port
	invalidConfig = &BrokerConfig{
		Name:     "test",
		Host:     "localhost",
		ClientID: "test-client",
	}
	
	if err := invalidConfig.Validate(); err == nil {
		t.Error("Expected error for missing port, got nil")
	}
	
	// Test missing client ID
	invalidConfig = &BrokerConfig{
		Name: "test",
		Host: "localhost",
		Port: 1883,
	}
	
	if err := invalidConfig.Validate(); err == nil {
		t.Error("Expected error for missing client ID, got nil")
	}
}

// Helper function to split environment variable string
func splitEnv(env string) (key, value string, found bool) {
	for i := 0; i < len(env); i++ {
		if env[i] == '=' {
			return env[:i], env[i+1:], true
		}
	}
	return env, "", false
}