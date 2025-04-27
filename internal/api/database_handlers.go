package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"MQTTmicroService/internal/database"

	"github.com/gorilla/mux"
)

// handleGetMessages handles requests to get messages from the database
func (s *Server) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		s.writeError(w, http.StatusInternalServerError, "Database not initialized")
		return
	}

	// Get query parameters
	confirmed := r.URL.Query().Get("confirmed") == "true"
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

	// Get messages from the database
	messages, err := s.db.GetMessages(ctx, confirmed, limit)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get messages: %v", err))
		return
	}

	// Write the response
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "success",
		"messages": messages,
		"count":    len(messages),
	})
}

// handleGetMessage handles requests to get a specific message from the database
func (s *Server) handleGetMessage(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		s.writeError(w, http.StatusInternalServerError, "Database not initialized")
		return
	}

	// Get the message ID from the URL
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		s.writeError(w, http.StatusBadRequest, "Message ID is required")
		return
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Get the message from the database
	message, err := s.db.GetMessageByID(ctx, id)
	if err != nil {
		if err == database.ErrMessageNotFound {
			s.writeError(w, http.StatusNotFound, "Message not found")
		} else {
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get message: %v", err))
		}
		return
	}

	// Write the response
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": message,
	})
}

// handleConfirmMessage handles requests to confirm a message
func (s *Server) handleConfirmMessage(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		s.writeError(w, http.StatusInternalServerError, "Database not initialized")
		return
	}

	// Get the message ID from the URL
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		s.writeError(w, http.StatusBadRequest, "Message ID is required")
		return
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Confirm the message
	err := s.db.ConfirmMessage(ctx, id)
	if err != nil {
		if err == database.ErrMessageNotFound {
			s.writeError(w, http.StatusNotFound, "Message not found")
		} else {
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to confirm message: %v", err))
		}
		return
	}

	// Write the response
	s.writeJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Message %s confirmed", id),
	})
}

// handleDeleteMessage handles requests to delete a message
func (s *Server) handleDeleteMessage(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		s.writeError(w, http.StatusInternalServerError, "Database not initialized")
		return
	}

	// Get the message ID from the URL
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		s.writeError(w, http.StatusBadRequest, "Message ID is required")
		return
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Delete the message
	err := s.db.DeleteMessage(ctx, id)
	if err != nil {
		if err == database.ErrMessageNotFound {
			s.writeError(w, http.StatusNotFound, "Message not found")
		} else {
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete message: %v", err))
		}
		return
	}

	// Write the response
	s.writeJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Message %s deleted", id),
	})
}

// handleDeleteConfirmedMessages handles requests to delete all confirmed messages
func (s *Server) handleDeleteConfirmedMessages(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		s.writeError(w, http.StatusInternalServerError, "Database not initialized")
		return
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Delete confirmed messages
	count, err := s.db.DeleteConfirmedMessages(ctx)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete confirmed messages: %v", err))
		return
	}

	// Write the response
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("%d confirmed messages deleted", count),
		"count":   count,
	})
}