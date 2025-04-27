package models

import (
	"time"
)

// Webhook represents a webhook configuration
type Webhook struct {
	ID          string            `json:"id" bson:"_id,omitempty"`
	Name        string            `json:"name" bson:"name"`
	URL         string            `json:"url" bson:"url"`
	Method      string            `json:"method" bson:"method"`
	TopicFilter string            `json:"topic_filter" bson:"topic_filter"`
	Enabled     bool              `json:"enabled" bson:"enabled"`
	Headers     map[string]string `json:"headers,omitempty" bson:"headers,omitempty"`
	Timeout     int               `json:"timeout" bson:"timeout"`
	RetryCount  int               `json:"retry_count" bson:"retry_count"`
	RetryDelay  int               `json:"retry_delay" bson:"retry_delay"`
	CreatedAt   time.Time         `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" bson:"updated_at"`
}

// NewWebhook creates a new webhook with default values
func NewWebhook() *Webhook {
	return &Webhook{
		Method:     "POST",
		Enabled:    true,
		Timeout:    10,
		RetryCount: 3,
		RetryDelay: 5,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Headers:    make(map[string]string),
	}
}

// Validate validates the webhook configuration
func (w *Webhook) Validate() error {
	if w.URL == "" {
		return NewValidationError("URL is required")
	}
	if w.Method == "" {
		return NewValidationError("Method is required")
	}
	if w.TopicFilter == "" {
		return NewValidationError("Topic filter is required")
	}
	if w.Timeout <= 0 {
		return NewValidationError("Timeout must be greater than 0")
	}
	if w.RetryCount < 0 {
		return NewValidationError("Retry count must be greater than or equal to 0")
	}
	if w.RetryDelay <= 0 {
		return NewValidationError("Retry delay must be greater than 0")
	}
	return nil
}

// ValidationError represents a validation error
type ValidationError struct {
	Message string
}

// Error returns the error message
func (e *ValidationError) Error() string {
	return e.Message
}

// NewValidationError creates a new validation error
func NewValidationError(message string) *ValidationError {
	return &ValidationError{
		Message: message,
	}
}
