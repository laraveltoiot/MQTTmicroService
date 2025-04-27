package database

import (
	"context"
	"time"

	"MQTTmicroService/internal/models"
)

// Message represents a message stored in the database
type Message struct {
	ID        string      `json:"id" bson:"_id,omitempty"`
	Topic     string      `json:"topic" bson:"topic"`
	Payload   interface{} `json:"payload" bson:"payload"`
	QoS       byte        `json:"qos" bson:"qos"`
	Retained  bool        `json:"retained" bson:"retained"`
	Timestamp time.Time   `json:"timestamp" bson:"timestamp"`
	Confirmed bool        `json:"confirmed" bson:"confirmed"`
}

// Database is the interface that must be implemented by database providers
type Database interface {
	// Connect establishes a connection to the database
	Connect(ctx context.Context) error

	// Close closes the database connection
	Close(ctx context.Context) error

	// StoreMessage stores a message in the database
	StoreMessage(ctx context.Context, msg *Message) error

	// GetMessages retrieves messages from the database
	GetMessages(ctx context.Context, confirmed bool, limit int) ([]*Message, error)

	// GetMessageByID retrieves a message by its ID
	GetMessageByID(ctx context.Context, id string) (*Message, error)

	// ConfirmMessage marks a message as confirmed
	ConfirmMessage(ctx context.Context, id string) error

	// DeleteMessage deletes a message from the database
	DeleteMessage(ctx context.Context, id string) error

	// DeleteConfirmedMessages deletes all confirmed messages
	DeleteConfirmedMessages(ctx context.Context) (int, error)

	// Webhook operations
	StoreWebhook(ctx context.Context, webhook *models.Webhook) error
	GetWebhooks(ctx context.Context, limit int) ([]*models.Webhook, error)
	GetWebhookByID(ctx context.Context, id string) (*models.Webhook, error)
	UpdateWebhook(ctx context.Context, webhook *models.Webhook) error
	DeleteWebhook(ctx context.Context, id string) error
	GetWebhooksByTopicFilter(ctx context.Context, topic string) ([]*models.Webhook, error)

	// Ping checks if the database is reachable
	Ping(ctx context.Context) error
}

// Config holds the configuration for the database
type Config struct {
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

// Provider is a factory function that returns a database implementation
type Provider func(config *Config) (Database, error)

// providers is a map of database providers
var providers = make(map[string]Provider)

// Register registers a database provider
func Register(name string, provider Provider) {
	providers[name] = provider
}

// New creates a new database instance
func New(config *Config) (Database, error) {
	provider, exists := providers[config.Type]
	if !exists {
		return nil, ErrUnsupportedDatabaseType
	}

	return provider(config)
}

// Errors
var (
	ErrUnsupportedDatabaseType = NewError("unsupported database type")
	ErrConnectionFailed        = NewError("failed to connect to database")
	ErrMessageNotFound         = NewError("message not found")
)

// Error represents a database error
type Error struct {
	Message string
}

// Error returns the error message
func (e *Error) Error() string {
	return e.Message
}

// NewError creates a new database error
func NewError(message string) *Error {
	return &Error{
		Message: message,
	}
}
