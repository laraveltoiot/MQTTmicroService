package broker

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"

	"MQTTmicroService/internal/logger"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/mochi-mqtt/server/v2/packets"
)

// Config holds the configuration for the MQTT broker
type Config struct {
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

	// Logging
	EnableLogging bool
}

// Broker represents an MQTT broker
type Broker struct {
	config  *Config
	logger  *logger.Logger
	server  *mqtt.Server
	mu      sync.RWMutex
	running bool
}

// LoggingHook is a custom hook for logging MQTT messages
type LoggingHook struct {
	mqtt.HookBase
	logger *logger.Logger
}

// ID returns the ID of the hook
func (h *LoggingHook) ID() string {
	return "logging-hook"
}

// OnPublish logs the publish event
func (h *LoggingHook) OnPublish(cl *mqtt.Client, pk packets.Packet) (packets.Packet, error) {
	if cl != nil {
		h.logger.WithFields(map[string]interface{}{
			"client_id": cl.ID,
			"topic":     pk.TopicName,
			"qos":       pk.FixedHeader.Qos,
			"payload":   string(pk.Payload),
		}).Info("Broker received message")
	}
	return pk, nil
}

// OnPublished logs the published event
func (h *LoggingHook) OnPublished(cl *mqtt.Client, pk packets.Packet) {
	if cl != nil {
		h.logger.WithFields(map[string]interface{}{
			"client_id": cl.ID,
			"topic":     pk.TopicName,
			"qos":       pk.FixedHeader.Qos,
		}).Info("Broker published message")
	}
}

// New creates a new MQTT broker
func New(cfg *Config, log *logger.Logger) (*Broker, error) {
	if cfg == nil {
		return nil, fmt.Errorf("broker configuration is required")
	}

	if log == nil {
		return nil, fmt.Errorf("logger is required")
	}

	// Create a new MQTT server with default options
	server := mqtt.New(&mqtt.Options{})

	return &Broker{
		config:  cfg,
		logger:  log,
		server:  server,
		running: false,
	}, nil
}

// Start starts the MQTT broker
func (b *Broker) Start() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.config.Enable {
		b.logger.Info("MQTT broker is disabled, not starting")
		return nil
	}

	if b.running {
		return fmt.Errorf("broker is already running")
	}

	// Configure authentication if enabled
	if b.config.AuthEnable {
		// Create a simple authentication hook
		authHook := &auth.Hook{}

		// Create a ledger for authentication
		ledger := &auth.Ledger{
			Users: make(auth.Users),
		}

		// Add credentials to the ledger
		for username, password := range b.config.Credentials {
			ledger.Users[username] = auth.UserRule{
				Username: auth.RString(username),
				Password: auth.RString(password),
				Disallow: false,
			}
			b.logger.WithFields(map[string]interface{}{
				"username": username,
			}).Info("Registering user for MQTT broker authentication")
		}

		// Create options for the auth hook
		authOpts := &auth.Options{
			Ledger: ledger,
		}

		// Register the auth hook with options
		err := b.server.AddHook(authHook, authOpts)
		if err != nil {
			return fmt.Errorf("failed to add auth hook: %w", err)
		}
	}

	// Add logging hook
	if b.config.EnableLogging {
		loggingHook := &LoggingHook{
			logger: b.logger,
		}
		err := b.server.AddHook(loggingHook, nil)
		if err != nil {
			return fmt.Errorf("failed to add logging hook: %w", err)
		}
		b.logger.Info("Message logging enabled for MQTT broker")
	}

	// Create TCP listener
	addr := fmt.Sprintf("%s:%d", b.config.Host, b.config.Port)
	tcpListener := listeners.NewTCP(listeners.Config{
		ID:      "tcp",
		Address: addr,
	})

	// Add the listener
	err := b.server.AddListener(tcpListener)
	if err != nil {
		return fmt.Errorf("failed to add TCP listener: %w", err)
	}

	// Add TLS listener if enabled
	if b.config.TLSEnable && b.config.TLSCertFile != "" && b.config.TLSKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(b.config.TLSCertFile, b.config.TLSKeyFile)
		if err != nil {
			return fmt.Errorf("failed to load TLS certificates: %w", err)
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}

		tlsAddr := fmt.Sprintf("%s:%d", b.config.Host, b.config.Port+1) // TLS on next port
		tlsListener := listeners.NewTCP(listeners.Config{
			ID:        "tls",
			Address:   tlsAddr,
			TLSConfig: tlsConfig,
		})

		err = b.server.AddListener(tlsListener)
		if err != nil {
			return fmt.Errorf("failed to add TLS listener: %w", err)
		}
	}

	// Start the server
	err = b.server.Serve()
	if err != nil {
		return fmt.Errorf("failed to start MQTT broker: %w", err)
	}

	b.running = true
	b.logger.WithFields(map[string]interface{}{
		"host": b.config.Host,
		"port": b.config.Port,
	}).Info("MQTT broker started")

	return nil
}

// Stop stops the MQTT broker
func (b *Broker) Stop(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.running {
		return nil
	}

	// Create a channel to signal completion
	done := make(chan struct{})

	// Stop the server in a goroutine
	go func() {
		b.server.Close()
		close(done)
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		b.running = false
		b.logger.Info("MQTT broker stopped")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("timeout stopping MQTT broker: %w", ctx.Err())
	}
}

// IsRunning returns true if the broker is running
func (b *Broker) IsRunning() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.running
}

// GetStatus returns the status of the broker
func (b *Broker) GetStatus() map[string]interface{} {
	b.mu.RLock()
	defer b.mu.RUnlock()

	status := map[string]interface{}{
		"running": b.running,
		"enabled": b.config.Enable,
	}

	if b.running {
		// Get basic statistics
		status["clients"] = 0 // We don't have access to client count directly
		// More detailed statistics could be added here
	}

	return status
}
