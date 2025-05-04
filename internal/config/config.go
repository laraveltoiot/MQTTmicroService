package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// BrokerConfig holds the configuration for a single MQTT broker
type BrokerConfig struct {
	Name          string
	Host          string
	Port          int
	ClientID      string
	CleanSession  bool
	EnableLogging bool
	LogChannel    string
	TLSEnabled    bool
	TLSVerifyPeer bool
	TLSCAFile     string
	Username      string
	Password      string
}

// DatabaseConfig holds the configuration for the database
type DatabaseConfig struct {
	// Type is the type of database to use (sqlite or mongodb)
	Type string
	// Connection is the connection string for the database
	Connection string
	// MongoDB specific settings
	MongoDB struct {
		URI      string
		Database string
		Username string
		Password string
		Port     int
	}
	// SQLite specific settings
	SQLite struct {
		Path string
	}
}

// WebhookConfig holds the configuration for webhook notifications
type WebhookConfig struct {
	// Enabled indicates whether webhook notifications are enabled
	Enabled bool
	// URL is the URL to send webhook notifications to
	URL string
	// Method is the HTTP method to use (GET, POST, etc.)
	Method string
	// Timeout is the timeout for webhook requests in seconds
	Timeout int
	// RetryCount is the number of times to retry failed webhook requests
	RetryCount int
	// RetryDelay is the delay between retries in seconds
	RetryDelay int
}

// MQTTBrokerConfig holds the configuration for the MQTT broker
type MQTTBrokerConfig struct {
	// Enable indicates whether the broker is enabled
	Enable bool
	// Host is the hostname or IP address to bind to
	Host string
	// Port is the port to listen on
	Port int
	// TLS configuration
	TLSEnable   bool
	TLSCertFile string
	TLSKeyFile  string
	// Authentication
	AuthEnable     bool
	AllowAnonymous bool
	Credentials    map[string]string
}

// Config holds the configuration for the MQTT microservice
type Config struct {
	DefaultConnection string
	Brokers           map[string]*BrokerConfig
	// API key authentication
	EnableAPIKey bool
	APIKeys      []string
	// Database configuration
	Database *DatabaseConfig
	// Webhook configuration
	Webhook *WebhookConfig
	// MQTT Broker configuration
	MQTTBroker *MQTTBrokerConfig
}

// LoadConfig loads the configuration from environment variables
func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	config := &Config{
		Brokers:    make(map[string]*BrokerConfig),
		Database:   &DatabaseConfig{},
		Webhook:    &WebhookConfig{},
		MQTTBroker: &MQTTBrokerConfig{},
	}

	// Get default connection
	config.DefaultConnection = os.Getenv("MQTT_DEFAULT_CONNECTION")
	if config.DefaultConnection == "" {
		return nil, errors.New("MQTT_DEFAULT_CONNECTION environment variable is required")
	}

	// Find all broker configurations
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "MQTT_") && !strings.HasPrefix(env, "MQTT_DEFAULT_") && !strings.HasPrefix(env, "MQTT_TLS_") && !strings.HasPrefix(env, "MQTT_AUTH_") {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) != 2 {
				continue
			}

			key := parts[0]
			keyParts := strings.Split(key, "_")
			if len(keyParts) < 3 {
				continue
			}

			brokerName := strings.ToLower(keyParts[1])
			configKey := strings.ToUpper(strings.Join(keyParts[2:], "_"))

			// Initialize broker config if it doesn't exist
			if _, exists := config.Brokers[brokerName]; !exists {
				config.Brokers[brokerName] = &BrokerConfig{
					Name: brokerName,
				}
			}

			// Set broker config values
			broker := config.Brokers[brokerName]
			switch configKey {
			case "HOST":
				broker.Host = os.Getenv(key)
			case "PORT":
				port, err := strconv.Atoi(os.Getenv(key))
				if err == nil {
					broker.Port = port
				}
			case "CLIENT_ID":
				broker.ClientID = os.Getenv(key)
			case "CLEAN_SESSION":
				broker.CleanSession = os.Getenv(key) == "true"
			case "ENABLE_LOGGING":
				broker.EnableLogging = os.Getenv(key) == "true"
			case "LOG_CHANNEL":
				broker.LogChannel = os.Getenv(key)
			}
		}
	}

	// Process TLS settings
	tlsEnabled := os.Getenv("MQTT_TLS_ENABLED") == "true"
	tlsVerifyPeer := os.Getenv("MQTT_TLS_VERIFY_PEER") == "true"
	tlsCAFile := os.Getenv("MQTT_TLS_CA_FILE")

	// Process auth settings
	username := os.Getenv("MQTT_AUTH_USERNAME")
	password := os.Getenv("MQTT_AUTH_PASSWORD")

	// Process API key authentication settings
	apiKeyEnabled := os.Getenv("API_KEY_ENABLED")
	config.EnableAPIKey = apiKeyEnabled == "true"
	fmt.Printf("API_KEY_ENABLED environment variable: %s, resulting EnableAPIKey flag: %v\n", apiKeyEnabled, config.EnableAPIKey)

	apiKeys := os.Getenv("API_KEYS")
	if apiKeys != "" {
		config.APIKeys = strings.Split(apiKeys, ",")
	}

	// Process database settings
	dbType := os.Getenv("DB_CONNECTION")
	if dbType == "" {
		dbType = "sqlite" // Default to SQLite if not specified
	}
	config.Database.Type = dbType

	// Process MongoDB settings
	if dbType == "mongodb" {
		config.Database.MongoDB.URI = os.Getenv("DB_URI")
		config.Database.MongoDB.Database = os.Getenv("DB_DATABASE")
		config.Database.MongoDB.Username = os.Getenv("DB_USERNAME")
		config.Database.MongoDB.Password = os.Getenv("DB_PASSWORD")

		// Parse port if provided
		portStr := os.Getenv("DB_PORT")
		if portStr != "" {
			port, err := strconv.Atoi(portStr)
			if err == nil {
				config.Database.MongoDB.Port = port
			}
		}
	}

	// Process SQLite settings
	if dbType == "sqlite" {
		config.Database.SQLite.Path = os.Getenv("DB_PATH")
		if config.Database.SQLite.Path == "" {
			config.Database.SQLite.Path = "mqtt-messages.db" // Default SQLite database path
		}
	}

	// Process webhook settings
	webhookEnabled := os.Getenv("WEBHOOK_ENABLED") == "true"
	config.Webhook.Enabled = webhookEnabled
	config.Webhook.URL = os.Getenv("WEBHOOK_URL")
	config.Webhook.Method = os.Getenv("WEBHOOK_METHOD")
	if config.Webhook.Method == "" {
		config.Webhook.Method = "POST" // Default to POST if not specified
	}

	// Parse webhook timeout
	webhookTimeoutStr := os.Getenv("WEBHOOK_TIMEOUT")
	if webhookTimeoutStr != "" {
		webhookTimeout, err := strconv.Atoi(webhookTimeoutStr)
		if err == nil && webhookTimeout > 0 {
			config.Webhook.Timeout = webhookTimeout
		}
	}
	if config.Webhook.Timeout == 0 {
		config.Webhook.Timeout = 10 // Default to 10 seconds if not specified or invalid
	}

	// Parse webhook retry count
	webhookRetryCountStr := os.Getenv("WEBHOOK_RETRY_COUNT")
	if webhookRetryCountStr != "" {
		webhookRetryCount, err := strconv.Atoi(webhookRetryCountStr)
		if err == nil && webhookRetryCount >= 0 {
			config.Webhook.RetryCount = webhookRetryCount
		}
	}
	if config.Webhook.RetryCount == 0 {
		config.Webhook.RetryCount = 3 // Default to 3 retries if not specified or invalid
	}

	// Parse webhook retry delay
	webhookRetryDelayStr := os.Getenv("WEBHOOK_RETRY_DELAY")
	if webhookRetryDelayStr != "" {
		webhookRetryDelay, err := strconv.Atoi(webhookRetryDelayStr)
		if err == nil && webhookRetryDelay > 0 {
			config.Webhook.RetryDelay = webhookRetryDelay
		}
	}
	if config.Webhook.RetryDelay == 0 {
		config.Webhook.RetryDelay = 5 // Default to 5 seconds if not specified or invalid
	}

	// Process MQTT broker settings
	brokerEnabled := os.Getenv("MQTT_BROKER_ENABLED") == "true"
	config.MQTTBroker.Enable = brokerEnabled
	config.MQTTBroker.Host = os.Getenv("MQTT_BROKER_HOST")
	if config.MQTTBroker.Host == "" {
		config.MQTTBroker.Host = "0.0.0.0" // Default to all interfaces if not specified
	}

	// Parse broker port
	brokerPortStr := os.Getenv("MQTT_BROKER_PORT")
	if brokerPortStr != "" {
		brokerPort, err := strconv.Atoi(brokerPortStr)
		if err == nil && brokerPort > 0 {
			config.MQTTBroker.Port = brokerPort
		}
	}
	if config.MQTTBroker.Port == 0 {
		config.MQTTBroker.Port = 1883 // Default to standard MQTT port if not specified or invalid
	}

	// Process broker TLS settings
	config.MQTTBroker.TLSEnable = os.Getenv("MQTT_BROKER_TLS_ENABLED") == "true"
	config.MQTTBroker.TLSCertFile = os.Getenv("MQTT_BROKER_TLS_CERT_FILE")
	config.MQTTBroker.TLSKeyFile = os.Getenv("MQTT_BROKER_TLS_KEY_FILE")

	// Process broker authentication settings
	config.MQTTBroker.AuthEnable = os.Getenv("MQTT_BROKER_AUTH_ENABLED") == "true"
	config.MQTTBroker.AllowAnonymous = os.Getenv("MQTT_BROKER_ALLOW_ANONYMOUS") == "true"

	// Parse broker credentials
	brokerCredentials := os.Getenv("MQTT_BROKER_CREDENTIALS")
	if brokerCredentials != "" {
		credentials := make(map[string]string)
		credentialPairs := strings.Split(brokerCredentials, ",")
		for _, pair := range credentialPairs {
			parts := strings.SplitN(pair, ":", 2)
			if len(parts) == 2 {
				credentials[parts[0]] = parts[1]
			}
		}
		config.MQTTBroker.Credentials = credentials
	} else {
		config.MQTTBroker.Credentials = make(map[string]string)
	}

	// Apply TLS and auth settings to all brokers
	for _, broker := range config.Brokers {
		broker.TLSEnabled = tlsEnabled
		broker.TLSVerifyPeer = tlsVerifyPeer
		broker.TLSCAFile = tlsCAFile
		broker.Username = username
		broker.Password = password
	}

	// Validate configuration
	if len(config.Brokers) == 0 {
		return nil, errors.New("no MQTT broker configurations found")
	}

	// Validate default connection
	if _, exists := config.Brokers[config.DefaultConnection]; !exists {
		return nil, fmt.Errorf("default connection '%s' not found in broker configurations", config.DefaultConnection)
	}

	return config, nil
}

// GetBrokerConfig returns the configuration for a specific broker
func (c *Config) GetBrokerConfig(name string) (*BrokerConfig, error) {
	if name == "" {
		name = c.DefaultConnection
	}

	broker, exists := c.Brokers[name]
	if !exists {
		return nil, fmt.Errorf("broker configuration '%s' not found", name)
	}

	return broker, nil
}

// GetDefaultBrokerConfig returns the configuration for the default broker
func (c *Config) GetDefaultBrokerConfig() (*BrokerConfig, error) {
	return c.GetBrokerConfig(c.DefaultConnection)
}

// Validate checks if the broker configuration is valid
func (b *BrokerConfig) Validate() error {
	if b.Host == "" {
		return fmt.Errorf("host is required for broker '%s'", b.Name)
	}
	if b.Port == 0 {
		return fmt.Errorf("port is required for broker '%s'", b.Name)
	}
	if b.ClientID == "" {
		return fmt.Errorf("client ID is required for broker '%s'", b.Name)
	}
	if b.TLSEnabled && b.TLSCAFile != "" {
		// Check if the CA file exists
		if _, err := os.Stat(b.TLSCAFile); os.IsNotExist(err) {
			return fmt.Errorf("TLS CA file '%s' does not exist for broker '%s'", b.TLSCAFile, b.Name)
		}
	}
	return nil
}
