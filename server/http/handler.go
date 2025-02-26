package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/shafigh75/Memorandum/config"
	"github.com/shafigh75/Memorandum/server/db"
	"github.com/shafigh75/Memorandum/utils/logger"
)

// APIResponse represents a standard API response.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Handler struct to hold the store
type Handler struct {
	Store  *db.ShardedInMemoryStore
	Logger *logger.Logger
}

// NewHandler creates a new HTTP handler.
func NewHandler(store *db.ShardedInMemoryStore, logger *logger.Logger) *Handler {
	return &Handler{Store: store, Logger: logger}
}

// ServeHTTP implements the http.Handler interface.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	configFilePath := "config/config.json"
	cfg, err := config.LoadConfig(configFilePath)
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}
	if cfg.AuthEnabled {
		// Check for authentication if enabled
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer "+cfg.AuthToken {
			// Create a structured log message
			logMessage := map[string]interface{}{
				"timestamp": time.Now().Format(time.RFC3339),
				"method":    r.Method + ": UNAUTHORIZED",
				"url":       r.URL.String(),
				"ip":        r.RemoteAddr,
			}

			// Convert the log message to JSON
			logJSON, err := json.Marshal(logMessage)
			if err != nil {
				// Handle JSON marshaling error (optional)
				h.Logger.Log("Error marshaling log message to JSON")
				return
			}

			// Log the structured message as a JSON string
			h.Logger.Log(string(logJSON))
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// Create a structured log message
	logMessage := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"method":    r.Method,
		"url":       r.URL.String(),
		"ip":        r.RemoteAddr,
	}

	// Convert the log message to JSON
	logJSON, err := json.Marshal(logMessage)
	if err != nil {
		// Handle JSON marshaling error (optional)
		h.Logger.Log("Error marshaling log message to JSON")
		return
	}

	// Log the structured message as a JSON string
	h.Logger.Log(string(logJSON))
	switch r.Method {
	case http.MethodPost:
		h.SetHandler(w, r)
	case http.MethodGet:
		h.GetHandler(w, r)
	case http.MethodDelete:
		h.DeleteHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// SetHandler handles the set request.
func (h *Handler) SetHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key   string `json:"key"`
		Value string `json:"value"`
		TTL   int64  `json:"ttl"` // TTL in seconds
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	h.Store.Set(req.Key, req.Value, req.TTL)
	json.NewEncoder(w).Encode(APIResponse{Success: true})
}

// GetHandler handles the get request.
func (h *Handler) GetHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if value, exists := h.Store.Get(key); exists {
		json.NewEncoder(w).Encode(APIResponse{Success: true, Data: value})
	} else {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Error: "Key not found or expired"})
	}
}

// DeleteHandler handles the delete request.
func (h *Handler) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	h.Store.Delete(key)
	json.NewEncoder(w).Encode(APIResponse{Success: true})
}
