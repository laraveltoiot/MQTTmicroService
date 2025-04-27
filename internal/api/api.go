package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"MQTTmicroService/internal/auth"
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

// NewServer creates a new HTTP API server
func NewServer(mqttManager *mqtt.Manager, log *logger.Logger, metricsCollector *metrics.Metrics, authService *auth.Auth, db database.Database, addr string) *Server {
	router := mux.NewRouter()

	server := &Server{
		router:      router,
		mqttManager: mqttManager,
		logger:      log,
		metrics:     metricsCollector,
		auth:        authService,
		db:          db,
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
		s.router.HandleFunc("/messages", s.handleGetMessages).Methods("GET")
		s.router.HandleFunc("/messages/{id}", s.handleGetMessage).Methods("GET")
		s.router.HandleFunc("/messages/{id}/confirm", s.handleConfirmMessage).Methods("POST")
		s.router.HandleFunc("/messages/{id}", s.handleDeleteMessage).Methods("DELETE")
		s.router.HandleFunc("/messages/confirmed", s.handleDeleteConfirmedMessages).Methods("DELETE")
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

	// Create a message handler that logs received messages and updates metrics
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
