package mqtt

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"time"
	"context"

	"MQTTmicroService/internal/config"
	"MQTTmicroService/internal/database"
	"MQTTmicroService/internal/logger"
	"MQTTmicroService/internal/metrics"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Client represents an MQTT client
type Client struct {
	config     *config.BrokerConfig
	client     mqtt.Client
	logger     *logger.Logger
	subscriptions map[string]mqtt.MessageHandler
	manager    *Manager
	mu         sync.RWMutex
}

// Manager manages multiple MQTT clients
type Manager struct {
	config     *config.Config
	clients    map[string]*Client
	logger     *logger.Logger
	metrics    *metrics.Metrics
	db         database.Database
	mu         sync.RWMutex
}

// GetAllClients returns all MQTT clients
func (m *Manager) GetAllClients() map[string]*Client {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create a copy to avoid race conditions
	clients := make(map[string]*Client, len(m.clients))
	for name, client := range m.clients {
		clients[name] = client
	}

	return clients
}

// NewManager creates a new MQTT client manager
func NewManager(cfg *config.Config, log *logger.Logger, metricsCollector *metrics.Metrics, db database.Database) *Manager {
	return &Manager{
		config:  cfg,
		clients: make(map[string]*Client),
		logger:  log,
		metrics: metricsCollector,
		db:      db,
	}
}

// GetClient returns an MQTT client for the specified broker
func (m *Manager) GetClient(brokerName string) (*Client, error) {
	if brokerName == "" {
		brokerName = m.config.DefaultConnection
	}

	m.mu.RLock()
	client, exists := m.clients[brokerName]
	m.mu.RUnlock()

	if exists {
		return client, nil
	}

	// Get broker config
	brokerConfig, err := m.config.GetBrokerConfig(brokerName)
	if err != nil {
		return nil, err
	}

	// Create new client
	client, err = m.createClient(brokerConfig)
	if err != nil {
		return nil, err
	}

	// Store client
	m.mu.Lock()
	m.clients[brokerName] = client
	m.mu.Unlock()

	return client, nil
}

// GetDefaultClient returns the default MQTT client
func (m *Manager) GetDefaultClient() (*Client, error) {
	return m.GetClient(m.config.DefaultConnection)
}

// createClient creates a new MQTT client
func (m *Manager) createClient(cfg *config.BrokerConfig) (*Client, error) {
	// Validate config
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Create options
	opts := mqtt.NewClientOptions()

	// Use ssl:// protocol for TLS connections, tcp:// for non-TLS
	protocol := "tcp"
	if cfg.TLSEnabled {
		protocol = "ssl"
	}
	opts.AddBroker(fmt.Sprintf("%s://%s:%d", protocol, cfg.Host, cfg.Port))
	opts.SetClientID(cfg.ClientID)
	opts.SetCleanSession(cfg.CleanSession)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(1 * time.Minute)
	opts.SetKeepAlive(30 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	opts.SetWriteTimeout(10 * time.Second)
	opts.SetOrderMatters(false)
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		m.logger.WithError(err).Error("MQTT connection lost")
		// Update metrics if available
		if m.metrics != nil {
			m.metrics.IncrementDisconnections()
		}
	})
	opts.SetReconnectingHandler(func(client mqtt.Client, opts *mqtt.ClientOptions) {
		m.logger.Info("MQTT reconnecting")
		// Update metrics if available
		if m.metrics != nil {
			m.metrics.IncrementConnectionAttempts()
		}
	})
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		m.logger.WithField("broker", cfg.Name).Info("MQTT connected")
		// Update metrics if available
		if m.metrics != nil {
			m.metrics.IncrementConnectionSuccesses()
		}
	})

	// Set credentials if provided
	if cfg.Username != "" && cfg.Password != "" {
		opts.SetUsername(cfg.Username)
		opts.SetPassword(cfg.Password)
	}

	// Configure TLS if enabled
	if cfg.TLSEnabled {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: !cfg.TLSVerifyPeer,
		}

		// Load CA certificate if provided
		if cfg.TLSCAFile != "" {
			// Convert path separators for Windows
			filePath := strings.ReplaceAll(cfg.TLSCAFile, "/", "\\")

			caCert, err := ioutil.ReadFile(filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to read CA certificate: %w", err)
			}

			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("failed to parse CA certificate")
			}

			tlsConfig.RootCAs = caCertPool
		}

		opts.SetTLSConfig(tlsConfig)
	}

	// Create client
	client := mqtt.NewClient(opts)

	// Create client wrapper
	return &Client{
		config:     cfg,
		client:     client,
		logger:     m.logger,
		subscriptions: make(map[string]mqtt.MessageHandler),
		manager:    m,
	}, nil
}

// Connect connects to the MQTT broker
func (c *Client) Connect() error {
	if token := c.client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
	}
	return nil
}

// Disconnect disconnects from the MQTT broker
func (c *Client) Disconnect() {
	c.client.Disconnect(250)
}

// IsConnected returns true if the client is connected
func (c *Client) IsConnected() bool {
	return c.client.IsConnected()
}

// Publish publishes a message to the specified topic
func (c *Client) Publish(topic string, qos byte, retained bool, payload interface{}) error {
	if !c.IsConnected() {
		return fmt.Errorf("client is not connected")
	}

	// Convert payload to appropriate format based on type
	var finalPayload interface{}
	switch p := payload.(type) {
	case string:
		finalPayload = p
	case []byte:
		finalPayload = p
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
		// For primitive types, convert to string
		finalPayload = fmt.Sprintf("%v", p)
	default:
		// For complex types (maps, structs, etc.), convert to JSON string
		jsonBytes, err := json.Marshal(p)
		if err != nil {
			return fmt.Errorf("failed to marshal payload to JSON: %w", err)
		}
		finalPayload = jsonBytes
	}

	token := c.client.Publish(topic, qos, retained, finalPayload)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to publish message: %w", token.Error())
	}

	// Store message in database if available
	if c.manager != nil && c.manager.db != nil {
		// Create a context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Create a database message
		dbMsg := &database.Message{
			Topic:     topic,
			Payload:   payload,
			QoS:       qos,
			Retained:  retained,
			Timestamp: time.Now(),
			Confirmed: false,
		}

		// Store the message in the database
		if err := c.manager.db.StoreMessage(ctx, dbMsg); err != nil {
			c.logger.WithError(err).Error("Failed to store message in database")
			// Don't return error here, as the message was successfully published to MQTT
		} else {
			c.logger.WithField("id", dbMsg.ID).Debug("Message stored in database")
		}
	}

	c.logger.WithFields(map[string]interface{}{
		"topic":    topic,
		"qos":      qos,
		"retained": retained,
	}).Debug("Message published")

	return nil
}

// Subscribe subscribes to the specified topic
func (c *Client) Subscribe(topic string, qos byte, callback mqtt.MessageHandler) error {
	if !c.IsConnected() {
		return fmt.Errorf("client is not connected")
	}

	token := c.client.Subscribe(topic, qos, callback)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to subscribe to topic: %w", token.Error())
	}

	c.mu.Lock()
	c.subscriptions[topic] = callback
	c.mu.Unlock()

	c.logger.WithFields(map[string]interface{}{
		"topic": topic,
		"qos":   qos,
	}).Info("Subscribed to topic")

	return nil
}

// Unsubscribe unsubscribes from the specified topic
func (c *Client) Unsubscribe(topic string) error {
	if !c.IsConnected() {
		return fmt.Errorf("client is not connected")
	}

	token := c.client.Unsubscribe(topic)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to unsubscribe from topic: %w", token.Error())
	}

	c.mu.Lock()
	delete(c.subscriptions, topic)
	c.mu.Unlock()

	c.logger.WithField("topic", topic).Info("Unsubscribed from topic")

	return nil
}

// GetSubscriptions returns all active subscriptions
func (c *Client) GetSubscriptions() map[string]mqtt.MessageHandler {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Create a copy to avoid race conditions
	subscriptions := make(map[string]mqtt.MessageHandler, len(c.subscriptions))
	for topic, handler := range c.subscriptions {
		subscriptions[topic] = handler
	}

	return subscriptions
}

// ResubscribeAll resubscribes to all topics
func (c *Client) ResubscribeAll() error {
	c.mu.RLock()
	subscriptions := make(map[string]mqtt.MessageHandler, len(c.subscriptions))
	for topic, handler := range c.subscriptions {
		subscriptions[topic] = handler
	}
	c.mu.RUnlock()

	for topic, handler := range subscriptions {
		if err := c.Subscribe(topic, 1, handler); err != nil {
			return err
		}
	}

	return nil
}
