package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"MQTTmicroService/internal/auth"
	"MQTTmicroService/internal/config"
	"MQTTmicroService/internal/database"
	"MQTTmicroService/internal/logger"
	"MQTTmicroService/internal/metrics"
	"MQTTmicroService/internal/mqtt"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gorilla/mux"
)

// Server represents the HTTP API server
type Server struct {
	router      *mux.Router
	mqttManager *mqtt.Manager
	logger      *logger.Logger
	metrics     *metrics.Metrics
	auth        *auth.Auth
	db          database.Database
	server      *http.Server
	config      *config.Config
}

// PublishRequest represents a request to publish a message
type PublishRequest struct {
	Topic    string      `json:"topic"`
	Payload  interface{} `json:"payload"`
	QoS      byte        `json:"qos"`
	Retained bool        `json:"retained"`
	Broker   string      `json:"broker,omitempty"`
}

// SubscribeRequest represents a request to subscribe to a topic
type SubscribeRequest struct {
	Topic  string `json:"topic"`
	QoS    byte   `json:"qos"`
	Broker string `json:"broker,omitempty"`
}

// StatusResponse represents the status of MQTT connections
type StatusResponse struct {
	Status    string                  `json:"status"`
	Brokers   map[string]BrokerStatus `json:"brokers"`
	Timestamp string                  `json:"timestamp"`
}

// BrokerStatus represents the status of a single MQTT broker
type BrokerStatus struct {
	Connected     bool     `json:"connected"`
	Subscriptions []string `json:"subscriptions"`
}

// WebhookPayload represents the payload sent to the webhook
type WebhookPayload struct {
	Topic     string      `json:"topic"`
	Payload   interface{} `json:"payload"`
	QoS       byte        `json:"qos"`
	Timestamp string      `json:"timestamp"`
	Broker    string      `json:"broker"`
}

// NewServer creates a new HTTP API server
func NewServer(mqttManager *mqtt.Manager, log *logger.Logger, metricsCollector *metrics.Metrics, authService *auth.Auth, db database.Database, cfg *config.Config, addr string) *Server {
	router := mux.NewRouter()

	server := &Server{
		router:      router,
		mqttManager: mqttManager,
		logger:      log,
		metrics:     metricsCollector,
		auth:        authService,
		db:          db,
		config:      cfg,
		server: &http.Server{
			Addr:         addr,
			Handler:      router,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}

	server.setupRoutes()
	return server
}

// setupRoutes sets up the HTTP routes
func (s *Server) setupRoutes() {
	// Add metrics middleware if metrics collector is initialized
	if s.metrics != nil {
		s.router.Use(s.metricsMiddleware)
	}

	// Add authentication middleware if auth service is initialized
	if s.auth != nil {
		s.logger.WithFields(map[string]interface{}{
			"auth":         s.auth != nil,
			"enableAPIKey": s.auth.GetEnableAPIKey(),
		}).Info("Adding authentication middleware to router")
		s.router.Use(s.auth.AuthMiddleware)
	}

	s.router.HandleFunc("/publish", s.handlePublish).Methods("POST")
	s.router.HandleFunc("/subscribe", s.handleSubscribe).Methods("POST")
	s.router.HandleFunc("/unsubscribe", s.handleUnsubscribe).Methods("POST")
	s.router.HandleFunc("/status", s.handleStatus).Methods("GET")
	s.router.HandleFunc("/healthz", s.handleHealthCheck).Methods("GET")
	s.router.HandleFunc("/metrics", s.handleMetrics).Methods("GET")
	s.router.HandleFunc("/logs", s.handleLogs).Methods("GET")

	// Database-related endpoints
	if s.db != nil {
		// Message endpoints
		s.router.HandleFunc("/messages", s.handleGetMessages).Methods("GET")
		s.router.HandleFunc("/messages/{id}", s.handleGetMessage).Methods("GET")
		s.router.HandleFunc("/messages/{id}/confirm", s.handleConfirmMessage).Methods("POST")
		s.router.HandleFunc("/messages/{id}", s.handleDeleteMessage).Methods("DELETE")
		s.router.HandleFunc("/messages/confirmed", s.handleDeleteConfirmedMessages).Methods("DELETE")

		// Webhook endpoints
		s.router.HandleFunc("/webhooks", s.handleGetWebhooks).Methods("GET")
		s.router.HandleFunc("/webhooks", s.handleCreateWebhook).Methods("POST")
		s.router.HandleFunc("/webhooks/{id}", s.handleGetWebhook).Methods("GET")
		s.router.HandleFunc("/webhooks/{id}", s.handleUpdateWebhook).Methods("PUT")
		s.router.HandleFunc("/webhooks/{id}", s.handleDeleteWebhook).Methods("DELETE")
	}
}

// metricsMiddleware is middleware that tracks API requests and errors
func (s *Server) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Increment API requests counter
		s.metrics.IncrementAPIRequests()

		// Create a response writer wrapper to capture the status code
		rww := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Call the next handler
		next.ServeHTTP(rww, r)

		// If the status code is an error (>= 400), increment the API errors counter
		if rww.statusCode >= 400 {
			s.metrics.IncrementAPIErrors()
		}
	})
}

// responseWriterWrapper is a wrapper for http.ResponseWriter that captures the status code
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code and calls the underlying ResponseWriter's WriteHeader
func (rww *responseWriterWrapper) WriteHeader(statusCode int) {
	rww.statusCode = statusCode
	rww.ResponseWriter.WriteHeader(statusCode)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.logger.WithField("addr", s.server.Addr).Info("Starting HTTP server")
	return s.server.ListenAndServe()
}

// Stop stops the HTTP server
func (s *Server) Stop() error {
	s.logger.Info("Stopping HTTP server")
	return s.server.Close()
}

// handlePublish handles requests to publish messages
func (s *Server) handlePublish(w http.ResponseWriter, r *http.Request) {
	var req PublishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Topic == "" {
		s.writeError(w, http.StatusBadRequest, "Topic is required")
		return
	}

	client, err := s.mqttManager.GetClient(req.Broker)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get MQTT client: %v", err))
		return
	}

	if !client.IsConnected() {
		if err := client.Connect(); err != nil {
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to connect to MQTT broker: %v", err))
			return
		}
	}

	// Start timing for latency measurement
	startTime := time.Now()

	if err := client.Publish(req.Topic, req.QoS, req.Retained, req.Payload); err != nil {
		// Increment failed publishes counter
		if s.metrics != nil {
			s.metrics.IncrementFailedPublishes()
		}
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to publish message: %v", err))
		return
	}

	// Calculate and record latency
	if s.metrics != nil {
		s.metrics.IncrementPublishedMessages()
		s.metrics.AddPublishLatency(time.Since(startTime))
	}

	s.writeJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Message published successfully",
	})
}

// handleSubscribe handles requests to subscribe to topics
func (s *Server) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	var req SubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Topic == "" {
		s.writeError(w, http.StatusBadRequest, "Topic is required")
		return
	}

	client, err := s.mqttManager.GetClient(req.Broker)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get MQTT client: %v", err))
		return
	}

	if !client.IsConnected() {
		if err := client.Connect(); err != nil {
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to connect to MQTT broker: %v", err))
			return
		}
	}

	// Start timing for latency measurement
	startTime := time.Now()

	// Create a message handler that logs received messages, updates metrics, and sends webhook notifications
	messageHandler := func(client pahomqtt.Client, msg pahomqtt.Message) {
		s.logger.WithFields(map[string]interface{}{
			"topic":   msg.Topic(),
			"payload": string(msg.Payload()),
			"qos":     msg.Qos(),
		}).Info("Received message")

		// Increment received messages counter
		if s.metrics != nil {
			s.metrics.IncrementReceivedMessages()
		}

		// Try to parse the payload as JSON
		var payloadData interface{} = string(msg.Payload())
		var jsonPayload interface{}
		if err := json.Unmarshal(msg.Payload(), &jsonPayload); err == nil {
			payloadData = jsonPayload
		}

		// Send webhook notification
		go s.sendWebhookNotification(msg.Topic(), req.Broker, payloadData, msg.Qos())
	}

	if err := client.Subscribe(req.Topic, req.QoS, pahomqtt.MessageHandler(messageHandler)); err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to subscribe to topic: %v", err))
		return
	}

	// Calculate and record latency
	if s.metrics != nil {
		s.metrics.AddSubscribeLatency(time.Since(startTime))

		// Count total subscriptions across all clients
		var subscriptionCount int64
		for _, client := range s.mqttManager.GetAllClients() {
			subscriptionCount += int64(len(client.GetSubscriptions()))
		}
		s.metrics.SetSubscriptionCount(subscriptionCount)
	}

	s.writeJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Subscribed to topic %s", req.Topic),
	})
}

// handleUnsubscribe handles requests to unsubscribe from topics
func (s *Server) handleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	var req SubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Topic == "" {
		s.writeError(w, http.StatusBadRequest, "Topic is required")
		return
	}

	client, err := s.mqttManager.GetClient(req.Broker)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get MQTT client: %v", err))
		return
	}

	if !client.IsConnected() {
		s.writeError(w, http.StatusInternalServerError, "MQTT client is not connected")
		return
	}

	if err := client.Unsubscribe(req.Topic); err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to unsubscribe from topic: %v", err))
		return
	}

	// Update subscription count in metrics
	if s.metrics != nil {
		// Count total subscriptions across all clients
		var subscriptionCount int64
		for _, client := range s.mqttManager.GetAllClients() {
			subscriptionCount += int64(len(client.GetSubscriptions()))
		}
		s.metrics.SetSubscriptionCount(subscriptionCount)
	}

	s.writeJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Unsubscribed from topic %s", req.Topic),
	})
}

// handleStatus handles requests to get the status of MQTT connections
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	// Get all clients
	clients := s.mqttManager.GetAllClients()

	// Create response
	response := StatusResponse{
		Status:    "ok",
		Brokers:   make(map[string]BrokerStatus),
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Check if any client is connected
	allConnected := true

	// Get status for each client
	for name, client := range clients {
		connected := client.IsConnected()
		if !connected {
			allConnected = false
		}

		// Get subscriptions
		subscriptions := make([]string, 0)
		for topic := range client.GetSubscriptions() {
			subscriptions = append(subscriptions, topic)
		}

		response.Brokers[name] = BrokerStatus{
			Connected:     connected,
			Subscriptions: subscriptions,
		}
	}

	if !allConnected {
		response.Status = "partial"
	}

	if len(clients) == 0 {
		response.Status = "no_clients"
	}

	s.writeJSON(w, http.StatusOK, response)
}

// handleHealthCheck handles health check requests
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// handleMetrics handles requests to get metrics
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if s.metrics == nil {
		s.writeError(w, http.StatusInternalServerError, "Metrics collector not initialized")
		return
	}

	metrics := s.metrics.GetMetrics()
	s.writeJSON(w, http.StatusOK, metrics)
}

// handleLogs handles requests to view logs
func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	// Get the log file path from query parameter or use default
	logFilePath := r.URL.Query().Get("file")
	if logFilePath == "" {
		logFilePath = "mqtt-service.log" // Default log file name
	}

	// Ensure the path is safe (no directory traversal)
	if filepath.IsAbs(logFilePath) || filepath.Clean(logFilePath) != logFilePath {
		s.writeError(w, http.StatusBadRequest, "Invalid log file path")
		return
	}

	// Check if the file exists
	if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
		s.writeError(w, http.StatusNotFound, "Log file not found")
		return
	}

	// Read the log file
	logData, err := ioutil.ReadFile(logFilePath)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to read log file: %v", err))
		return
	}

	// Get the number of lines to return from query parameter
	lines := r.URL.Query().Get("lines")
	if lines != "" {
		// TODO: Implement line limiting logic if needed
	}

	// Set content type to text/plain for log data
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write(logData)
}

// writeJSON writes a JSON response
func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.WithError(err).Error("Failed to encode JSON response")
	}
}

// writeError writes an error response
func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.logger.WithFields(map[string]interface{}{
		"status":  status,
		"message": message,
	}).Error("API error")

	s.writeJSON(w, status, map[string]string{
		"status":  "error",
		"message": message,
	})
}

// sendWebhookNotification sends a notification to the configured webhook URL and any matching webhooks from the database
func (s *Server) sendWebhookNotification(topic, broker string, payload interface{}, qos byte) {
	// Create webhook payload
	webhookPayload := WebhookPayload{
		Topic:     topic,
		Payload:   payload,
		QoS:       qos,
		Timestamp: time.Now().Format(time.RFC3339),
		Broker:    broker,
	}

	// Send to global webhook if enabled
	if s.config != nil && s.config.Webhook != nil && s.config.Webhook.Enabled && s.config.Webhook.URL != "" {
		s.sendWebhookNotificationToURL(
			webhookPayload,
			s.config.Webhook.URL,
			s.config.Webhook.Method,
			nil, // No custom headers for global webhook
			s.config.Webhook.Timeout,
			s.config.Webhook.RetryCount,
			s.config.Webhook.RetryDelay,
		)
	}

	// Send to database webhooks if database is available
	if s.db != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Get webhooks that match the topic
		webhooks, err := s.db.GetWebhooksByTopicFilter(ctx, topic)
		if err != nil {
			s.logger.WithError(err).Error("Failed to get webhooks for topic")
			return
		}

		// Send notification to each matching webhook
		for _, webhook := range webhooks {
			if webhook.Enabled {
				s.sendWebhookNotificationToURL(
					webhookPayload,
					webhook.URL,
					webhook.Method,
					webhook.Headers,
					webhook.Timeout,
					webhook.RetryCount,
					webhook.RetryDelay,
				)
			}
		}
	}
}

// sendWebhookNotificationToURL sends a notification to a specific webhook URL
func (s *Server) sendWebhookNotificationToURL(
	webhookPayload WebhookPayload,
	url string,
	method string,
	headers map[string]string,
	timeout int,
	retryCount int,
	retryDelay int,
) {
	// Convert payload to JSON
	jsonPayload, err := json.Marshal(webhookPayload)
	if err != nil {
		s.logger.WithError(err).Error("Failed to marshal webhook payload")
		return
	}

	// Create HTTP request
	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		s.logger.WithError(err).Error("Failed to create webhook request")
		return
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "MQTT-Microservice")

	// Add custom headers if provided
	if headers != nil {
		for key, value := range headers {
			req.Header.Set(key, value)
		}
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	// Send request with retry logic
	var resp *http.Response
	var lastErr error
	for i := 0; i <= retryCount; i++ {
		if i > 0 {
			s.logger.WithFields(map[string]interface{}{
				"attempt": i,
				"error":   lastErr,
			}).Warn("Retrying webhook notification")
			time.Sleep(time.Duration(retryDelay) * time.Second)
		}

		resp, err = client.Do(req)
		if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			// Success
			resp.Body.Close()
			s.logger.WithFields(map[string]interface{}{
				"topic":  webhookPayload.Topic,
				"broker": webhookPayload.Broker,
				"url":    url,
			}).Info("Webhook notification sent successfully")
			return
		}

		if err != nil {
			lastErr = err
			s.logger.WithError(err).Error("Failed to send webhook notification")
		} else {
			lastErr = fmt.Errorf("webhook returned status code %d", resp.StatusCode)
			resp.Body.Close()
			s.logger.WithFields(map[string]interface{}{
				"status": resp.StatusCode,
				"url":    url,
			}).Error("Webhook notification failed")
		}
	}

	s.logger.WithFields(map[string]interface{}{
		"topic":       webhookPayload.Topic,
		"broker":      webhookPayload.Broker,
		"url":         url,
		"retry_count": retryCount,
	}).Error("Webhook notification failed after retries")
}
