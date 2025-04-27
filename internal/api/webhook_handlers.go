package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"MQTTmicroService/internal/models"

	"github.com/gorilla/mux"
)

// WebhookRequest represents a request to create or update a webhook
type WebhookRequest struct {
	Name        string            `json:"name"`
	URL         string            `json:"url"`
	Method      string            `json:"method"`
	TopicFilter string            `json:"topic_filter"`
	Enabled     bool              `json:"enabled"`
	Headers     map[string]string `json:"headers,omitempty"`
	Timeout     int               `json:"timeout"`
	RetryCount  int               `json:"retry_count"`
	RetryDelay  int               `json:"retry_delay"`
}

// handleGetWebhooks handles requests to get all webhooks
func (s *Server) handleGetWebhooks(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		s.writeError(w, http.StatusInternalServerError, "Database not initialized")
		return
	}

	// Get query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 100 // Default limit
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, "Invalid limit parameter")
			return
		}
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Get webhooks from the database
	webhooks, err := s.db.GetWebhooks(ctx, limit)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get webhooks: %v", err))
		return
	}

	// Write the response
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "success",
		"webhooks": webhooks,
		"count":    len(webhooks),
	})
}

// handleGetWebhook handles requests to get a specific webhook
func (s *Server) handleGetWebhook(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		s.writeError(w, http.StatusInternalServerError, "Database not initialized")
		return
	}

	// Get the webhook ID from the URL
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		s.writeError(w, http.StatusBadRequest, "Webhook ID is required")
		return
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Get the webhook from the database
	webhook, err := s.db.GetWebhookByID(ctx, id)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get webhook: %v", err))
		return
	}

	// Write the response
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"webhook": webhook,
	})
}

// handleCreateWebhook handles requests to create a new webhook
func (s *Server) handleCreateWebhook(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		s.writeError(w, http.StatusInternalServerError, "Database not initialized")
		return
	}

	// Parse the request body
	var req WebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate the request
	if req.URL == "" {
		s.writeError(w, http.StatusBadRequest, "URL is required")
		return
	}
	if req.TopicFilter == "" {
		s.writeError(w, http.StatusBadRequest, "Topic filter is required")
		return
	}

	// Create a new webhook
	webhook := models.NewWebhook()
	webhook.Name = req.Name
	webhook.URL = req.URL
	webhook.Method = req.Method
	webhook.TopicFilter = req.TopicFilter
	webhook.Enabled = req.Enabled
	webhook.Headers = req.Headers
	webhook.Timeout = req.Timeout
	webhook.RetryCount = req.RetryCount
	webhook.RetryDelay = req.RetryDelay

	// Validate the webhook
	if err := webhook.Validate(); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid webhook: %v", err))
		return
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Store the webhook in the database
	if err := s.db.StoreWebhook(ctx, webhook); err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to store webhook: %v", err))
		return
	}

	// Write the response
	s.writeJSON(w, http.StatusCreated, map[string]interface{}{
		"status":  "success",
		"message": "Webhook created successfully",
		"webhook": webhook,
	})
}

// handleUpdateWebhook handles requests to update a webhook
func (s *Server) handleUpdateWebhook(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		s.writeError(w, http.StatusInternalServerError, "Database not initialized")
		return
	}

	// Get the webhook ID from the URL
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		s.writeError(w, http.StatusBadRequest, "Webhook ID is required")
		return
	}

	// Parse the request body
	var req WebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Get the existing webhook
	webhook, err := s.db.GetWebhookByID(ctx, id)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get webhook: %v", err))
		return
	}

	// Update the webhook
	if req.Name != "" {
		webhook.Name = req.Name
	}
	if req.URL != "" {
		webhook.URL = req.URL
	}
	if req.Method != "" {
		webhook.Method = req.Method
	}
	if req.TopicFilter != "" {
		webhook.TopicFilter = req.TopicFilter
	}
	webhook.Enabled = req.Enabled
	if req.Headers != nil {
		webhook.Headers = req.Headers
	}
	if req.Timeout > 0 {
		webhook.Timeout = req.Timeout
	}
	if req.RetryCount >= 0 {
		webhook.RetryCount = req.RetryCount
	}
	if req.RetryDelay > 0 {
		webhook.RetryDelay = req.RetryDelay
	}

	// Validate the webhook
	if err := webhook.Validate(); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid webhook: %v", err))
		return
	}

	// Update the webhook in the database
	if err := s.db.UpdateWebhook(ctx, webhook); err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update webhook: %v", err))
		return
	}

	// Write the response
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Webhook updated successfully",
		"webhook": webhook,
	})
}

// handleDeleteWebhook handles requests to delete a webhook
func (s *Server) handleDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		s.writeError(w, http.StatusInternalServerError, "Database not initialized")
		return
	}

	// Get the webhook ID from the URL
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		s.writeError(w, http.StatusBadRequest, "Webhook ID is required")
		return
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Delete the webhook from the database
	if err := s.db.DeleteWebhook(ctx, id); err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete webhook: %v", err))
		return
	}

	// Write the response
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Webhook %s deleted successfully", id),
	})
}
