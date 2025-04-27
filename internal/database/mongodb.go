package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// MongoDBDatabase implements the Database interface for MongoDB
type MongoDBDatabase struct {
	client     *mongo.Client
	db         *mongo.Database
	collection *mongo.Collection
	config     *Config
}

// NewMongoDBDatabase creates a new MongoDB database instance
func NewMongoDBDatabase(config *Config) (Database, error) {
	return &MongoDBDatabase{
		config: config,
	}, nil
}

// init registers the MongoDB database provider
func init() {
	Register("mongodb", NewMongoDBDatabase)
}

// Connect establishes a connection to the MongoDB database
func (m *MongoDBDatabase) Connect(ctx context.Context) error {
	// Set default values if not provided
	uri := m.config.MongoDB.URI
	if uri == "" {
		// Build URI from components if URI is not provided
		host := "localhost"
		port := 27017
		if m.config.MongoDB.Port > 0 {
			port = m.config.MongoDB.Port
		}
		
		uri = fmt.Sprintf("mongodb://%s:%d", host, port)
		
		// Add credentials if provided
		if m.config.MongoDB.Username != "" && m.config.MongoDB.Password != "" {
			uri = fmt.Sprintf("mongodb://%s:%s@%s:%d", 
				m.config.MongoDB.Username, 
				m.config.MongoDB.Password, 
				host, 
				port)
		}
	}

	// Set default database name if not provided
	dbName := m.config.MongoDB.Database
	if dbName == "" {
		dbName = "mqtt_messages"
	}

	// Create client options
	clientOptions := options.Client().ApplyURI(uri)
	clientOptions.SetConnectTimeout(10 * time.Second)
	clientOptions.SetServerSelectionTimeout(5 * time.Second)

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping the database to verify connection
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		client.Disconnect(ctx)
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	// Get database and collection
	db := client.Database(dbName)
	collection := db.Collection("messages")

	// Create indexes
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "confirmed", Value: 1}},
		Options: options.Index().SetBackground(true),
	}
	_, err = collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		client.Disconnect(ctx)
		return fmt.Errorf("failed to create index: %w", err)
	}

	// Store client, database, and collection
	m.client = client
	m.db = db
	m.collection = collection

	return nil
}

// Close closes the database connection
func (m *MongoDBDatabase) Close(ctx context.Context) error {
	if m.client != nil {
		return m.client.Disconnect(ctx)
	}
	return nil
}

// StoreMessage stores a message in the database
func (m *MongoDBDatabase) StoreMessage(ctx context.Context, msg *Message) error {
	if m.collection == nil {
		return ErrConnectionFailed
	}

	// Generate an ID if one is not provided
	if msg.ID == "" {
		msg.ID = primitive.NewObjectID().Hex()
	}

	// Set the timestamp if not already set
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	// Insert the message
	_, err := m.collection.InsertOne(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to insert message: %w", err)
	}

	return nil
}

// GetMessages retrieves messages from the database
func (m *MongoDBDatabase) GetMessages(ctx context.Context, confirmed bool, limit int) ([]*Message, error) {
	if m.collection == nil {
		return nil, ErrConnectionFailed
	}

	// Default limit if not specified
	if limit <= 0 {
		limit = 100
	}

	// Create filter
	filter := bson.M{"confirmed": confirmed}

	// Create options
	findOptions := options.Find().
		SetSort(bson.D{{Key: "timestamp", Value: -1}}).
		SetLimit(int64(limit))

	// Query the database
	cursor, err := m.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer cursor.Close(ctx)

	// Parse the results
	var messages []*Message
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, fmt.Errorf("failed to decode messages: %w", err)
	}

	return messages, nil
}

// GetMessageByID retrieves a message by its ID
func (m *MongoDBDatabase) GetMessageByID(ctx context.Context, id string) (*Message, error) {
	if m.collection == nil {
		return nil, ErrConnectionFailed
	}

	// Create filter
	filter := bson.M{"_id": id}

	// Query the database
	var msg Message
	err := m.collection.FindOne(ctx, filter).Decode(&msg)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrMessageNotFound
		}
		return nil, fmt.Errorf("failed to query message: %w", err)
	}

	return &msg, nil
}

// ConfirmMessage marks a message as confirmed
func (m *MongoDBDatabase) ConfirmMessage(ctx context.Context, id string) error {
	if m.collection == nil {
		return ErrConnectionFailed
	}

	// Create filter
	filter := bson.M{"_id": id}

	// Create update
	update := bson.M{"$set": bson.M{"confirmed": true}}

	// Update the message
	result, err := m.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}

	// Check if the message was found
	if result.MatchedCount == 0 {
		return ErrMessageNotFound
	}

	return nil
}

// DeleteMessage deletes a message from the database
func (m *MongoDBDatabase) DeleteMessage(ctx context.Context, id string) error {
	if m.collection == nil {
		return ErrConnectionFailed
	}

	// Create filter
	filter := bson.M{"_id": id}

	// Delete the message
	result, err := m.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	// Check if the message was found
	if result.DeletedCount == 0 {
		return ErrMessageNotFound
	}

	return nil
}

// DeleteConfirmedMessages deletes all confirmed messages
func (m *MongoDBDatabase) DeleteConfirmedMessages(ctx context.Context) (int, error) {
	if m.collection == nil {
		return 0, ErrConnectionFailed
	}

	// Create filter
	filter := bson.M{"confirmed": true}

	// Delete the messages
	result, err := m.collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to delete messages: %w", err)
	}

	return int(result.DeletedCount), nil
}

// Ping checks if the database is reachable
func (m *MongoDBDatabase) Ping(ctx context.Context) error {
	if m.client == nil {
		return ErrConnectionFailed
	}

	return m.client.Ping(ctx, readpref.Primary())
}