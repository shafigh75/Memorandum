package handler

import (
	"Memorandum/server/db"
	"encoding/json"
	"net/http"
)

// APIResponse represents a standard API response.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Handler struct to hold the store
type Handler struct {
	Store *db.ShardedInMemoryStore
}

// NewHandler creates a new HTTP handler.
func NewHandler(store *db.ShardedInMemoryStore) *Handler {
	return &Handler{Store: store}
}

// ServeHTTP implements the http.Handler interface.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
