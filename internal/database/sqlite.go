package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteDatabase implements the Database interface for SQLite
type SQLiteDatabase struct {
	db     *sql.DB
	config *Config
}

// NewSQLiteDatabase creates a new SQLite database instance
func NewSQLiteDatabase(config *Config) (Database, error) {
	return &SQLiteDatabase{
		config: config,
	}, nil
}

// init registers the SQLite database provider
func init() {
	Register("sqlite", NewSQLiteDatabase)
}

// Connect establishes a connection to the SQLite database
func (s *SQLiteDatabase) Connect(ctx context.Context) error {
	// Ensure the directory exists
	dbPath := s.config.SQLite.Path
	if dbPath == "" {
		dbPath = "mqtt-messages.db"
	}

	// Ensure the directory exists
	dir := filepath.Dir(dbPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Open the database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	// Check if the connection is working
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Create the messages table if it doesn't exist
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			topic TEXT NOT NULL,
			payload BLOB NOT NULL,
			qos INTEGER NOT NULL,
			retained INTEGER NOT NULL,
			timestamp DATETIME NOT NULL,
			confirmed INTEGER NOT NULL
		)
	`)
	if err != nil {
		db.Close()
		return fmt.Errorf("failed to create messages table: %w", err)
	}

	// Create an index on the confirmed column
	_, err = db.ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS idx_messages_confirmed ON messages(confirmed)
	`)
	if err != nil {
		db.Close()
		return fmt.Errorf("failed to create index: %w", err)
	}

	s.db = db
	return nil
}

// Close closes the database connection
func (s *SQLiteDatabase) Close(ctx context.Context) error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// StoreMessage stores a message in the database
func (s *SQLiteDatabase) StoreMessage(ctx context.Context, msg *Message) error {
	if s.db == nil {
		return ErrConnectionFailed
	}

	// Generate an ID if one is not provided
	if msg.ID == "" {
		msg.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	// Set the timestamp if not already set
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	// Convert payload to JSON if it's not a string or []byte
	var payload interface{}
	switch p := msg.Payload.(type) {
	case string:
		payload = p
	case []byte:
		payload = p
	default:
		// For other types, convert to JSON
		jsonBytes, err := json.Marshal(p)
		if err != nil {
			return fmt.Errorf("failed to marshal payload to JSON: %w", err)
		}
		payload = jsonBytes
	}

	// Insert the message
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO messages (id, topic, payload, qos, retained, timestamp, confirmed) 
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		msg.ID, msg.Topic, payload, msg.QoS, boolToInt(msg.Retained), msg.Timestamp, boolToInt(msg.Confirmed))
	if err != nil {
		return fmt.Errorf("failed to insert message: %w", err)
	}

	return nil
}

// GetMessages retrieves messages from the database
func (s *SQLiteDatabase) GetMessages(ctx context.Context, confirmed bool, limit int) ([]*Message, error) {
	if s.db == nil {
		return nil, ErrConnectionFailed
	}

	// Default limit if not specified
	if limit <= 0 {
		limit = 100
	}

	// Query the database
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, topic, payload, qos, retained, timestamp, confirmed 
		 FROM messages 
		 WHERE confirmed = ? 
		 ORDER BY timestamp DESC 
		 LIMIT ?`,
		boolToInt(confirmed), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	// Parse the results
	var messages []*Message
	for rows.Next() {
		var msg Message
		var retained, confirmed int
		var payload []byte
		var timestamp string

		if err := rows.Scan(&msg.ID, &msg.Topic, &payload, &msg.QoS, &retained, &timestamp, &confirmed); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		// Parse the timestamp
		t, err := time.Parse("2006-01-02 15:04:05", timestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to parse timestamp: %w", err)
		}
		msg.Timestamp = t

		// Set the boolean fields
		msg.Retained = intToBool(retained)
		msg.Confirmed = intToBool(confirmed)

		// Set the payload
		msg.Payload = payload

		messages = append(messages, &msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	return messages, nil
}

// GetMessageByID retrieves a message by its ID
func (s *SQLiteDatabase) GetMessageByID(ctx context.Context, id string) (*Message, error) {
	if s.db == nil {
		return nil, ErrConnectionFailed
	}

	// Query the database
	row := s.db.QueryRowContext(ctx,
		`SELECT id, topic, payload, qos, retained, timestamp, confirmed 
		 FROM messages 
		 WHERE id = ?`,
		id)

	// Parse the result
	var msg Message
	var retained, confirmed int
	var payload []byte
	var timestamp string

	if err := row.Scan(&msg.ID, &msg.Topic, &payload, &msg.QoS, &retained, &timestamp, &confirmed); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrMessageNotFound
		}
		return nil, fmt.Errorf("failed to scan message: %w", err)
	}

	// Parse the timestamp
	t, err := time.Parse("2006-01-02 15:04:05", timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp: %w", err)
	}
	msg.Timestamp = t

	// Set the boolean fields
	msg.Retained = intToBool(retained)
	msg.Confirmed = intToBool(confirmed)

	// Set the payload
	msg.Payload = payload

	return &msg, nil
}

// ConfirmMessage marks a message as confirmed
func (s *SQLiteDatabase) ConfirmMessage(ctx context.Context, id string) error {
	if s.db == nil {
		return ErrConnectionFailed
	}

	// Update the message
	result, err := s.db.ExecContext(ctx,
		`UPDATE messages SET confirmed = 1 WHERE id = ?`,
		id)
	if err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}

	// Check if the message was found
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrMessageNotFound
	}

	return nil
}

// DeleteMessage deletes a message from the database
func (s *SQLiteDatabase) DeleteMessage(ctx context.Context, id string) error {
	if s.db == nil {
		return ErrConnectionFailed
	}

	// Delete the message
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM messages WHERE id = ?`,
		id)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	// Check if the message was found
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrMessageNotFound
	}

	return nil
}

// DeleteConfirmedMessages deletes all confirmed messages
func (s *SQLiteDatabase) DeleteConfirmedMessages(ctx context.Context) (int, error) {
	if s.db == nil {
		return 0, ErrConnectionFailed
	}

	// Delete the messages
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM messages WHERE confirmed = 1`)
	if err != nil {
		return 0, fmt.Errorf("failed to delete messages: %w", err)
	}

	// Get the number of deleted messages
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rowsAffected), nil
}

// Ping checks if the database is reachable
func (s *SQLiteDatabase) Ping(ctx context.Context) error {
	if s.db == nil {
		return ErrConnectionFailed
	}

	return s.db.PingContext(ctx)
}

// Helper functions

// boolToInt converts a boolean to an integer (0 or 1)
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// intToBool converts an integer to a boolean (0 = false, non-0 = true)
func intToBool(i int) bool {
	return i != 0
}
