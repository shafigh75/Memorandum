package main

import (
    "encoding/json"
    "fmt"
    "net"
    "net/http"
    "net/rpc"
    "sync"
    "time"
    "db/config"
)

// ValueWithTTL represents a value with its expiration time.
type ValueWithTTL struct {
    Value      string
    Expiration int64 // Unix timestamp in seconds
}

// InMemoryStore represents a simple in-memory key-value store with TTL.
type InMemoryStore struct {
    mu    sync.RWMutex
    store map[string]ValueWithTTL
}

// NewInMemoryStore creates a new instance of InMemoryStore.
func NewInMemoryStore() *InMemoryStore {
    return &InMemoryStore{
        store: make(map[string]ValueWithTTL),
    }
}

// Set adds a key-value pair to the store with an optional TTL.
func (s *InMemoryStore) Set(key, value string, ttl int64) {
    s.mu.Lock()
    defer s.mu.Unlock()
    expiration := time.Now().Add(time.Duration(ttl) * time.Second).Unix()
    s.store[key] = ValueWithTTL{Value: value, Expiration: expiration}
}

// Get retrieves a value by key from the store, checking for expiration.
func (s *InMemoryStore) Get(key string) (string, bool) {
    s.mu.RLock()
    valueWithTTL, exists := s.store[key]
    s.mu.RUnlock() // Unlock before potentially deleting

    if !exists || (valueWithTTL.Expiration > 0 && time.Now().Unix() > valueWithTTL.Expiration) {
        // If the key does not exist or has expired, attempt to delete it
        if exists {
            s.Delete(key) // Delete the expired key
        }
        return "", false
    }

    return valueWithTTL.Value, true
}

// Delete removes a key-value pair from the store.
func (s *InMemoryStore) Delete(key string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    delete(s.store, key)
}

// Cleanup removes expired keys from the store concurrently.
func (s *InMemoryStore) Cleanup() {
    s.mu.Lock()
    defer s.mu.Unlock()
    now := time.Now().Unix()
    for key, valueWithTTL := range s.store {
        if valueWithTTL.Expiration > 0 && now > valueWithTTL.Expiration {
            delete(s.store, key)
        }
    }
}

// StartCleanupRoutine starts a background goroutine to periodically clean up expired keys.
func (s *InMemoryStore) StartCleanupRoutine(interval time.Duration) {
    go func() {
        ticker := time.NewTicker(interval)
        defer ticker.Stop()
        for {
            <-ticker.C
            s.Cleanup() // Perform cleanup at regular intervals
        }
    }()
}

// APIResponse represents a standard API response.
type APIResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
}

// RPC request and response structures
type RPCRequest struct {
    Key   string `json:"key"`
    Value string `json:"value,omitempty"`
    TTL   int64  `json:"ttl"` // TTL in seconds
}

type RPCResponse struct {
    Success bool   `json:"success"`
    Data    string `json:"data,omitempty"`
    Error   string `json:"error,omitempty"`
}

// RPC methods
func (s *InMemoryStore) RPCSet(req *RPCRequest, resp *RPCResponse) error {
    s.Set(req.Key, req.Value, req.TTL)
    resp.Success = true
    return nil
}

func (s *InMemoryStore) RPCGet(req *RPCRequest, resp *RPCResponse) error {
    if value, exists := s.Get(req.Key); exists {
        resp.Success = true
        resp.Data = value
    } else {
        resp.Success = false
        resp.Error = "Key not found or expired"
    }
    return nil
}

func (s *InMemoryStore) RPCDelete(req *RPCRequest, resp *RPCResponse) error {
    s.Delete(req.Key)
    resp.Success = true
    return nil
}

// HTTP handlers
func (store *InMemoryStore) setHandler(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Key   string `json:"key"`
        Value string `json:"value"`
        TTL   int64  `json:"ttl"` // TTL in seconds
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    store.Set(req.Key, req.Value, req.TTL)
    json.NewEncoder(w).Encode(APIResponse{Success: true})
}

func (store *InMemoryStore) getHandler(w http.ResponseWriter, r *http.Request) {
    key := r.URL.Query().Get("key")
    if value, exists := store.Get(key); exists {
        json.NewEncoder(w).Encode(APIResponse{Success: true, Data: value})
    } else {
        json.NewEncoder(w).Encode(APIResponse{Success: false, Error: "Key not found or expired"})
    }
}

func (store *InMemoryStore) deleteHandler(w http.ResponseWriter, r *http.Request) {
    key := r.URL.Query().Get("key")
    store.Delete(key)
    json.NewEncoder(w).Encode(APIResponse{Success: true})
}

func main() {
    // Load configuration
    config, err := config.LoadConfig("config/config.json")
    if err != nil {
        fmt.Println("Error loading config:", err)
        return
    }

    store := NewInMemoryStore()

    // Start the cleanup routine based on the config
    store.StartCleanupRoutine(time.Duration(config.CleanupInterval) * time.Second)

    // Register the RPC service
    rpc.Register(store)

    // Start the HTTP server
    http.HandleFunc("/set", store.setHandler)
    http.HandleFunc("/get", store.getHandler)
    http.HandleFunc("/delete", store.deleteHandler)

    // Start the RPC server
    go func() {
        rpcListener, err := net.Listen("tcp", config.RPCPort)
        if err != nil {
            fmt.Println("Error starting RPC server:", err)
            return
        }
        defer rpcListener.Close()
        fmt.Println("RPC server is listening on port", config.RPCPort)
        for {
            conn, err := rpcListener.Accept()
            if err != nil {
                fmt.Println("Error accepting connection:", err)
                continue
            }
            go rpc.ServeConn(conn) // Handle each RPC connection in a new goroutine
        }
    }()

    fmt.Println("Starting HTTP server on", config.HTTPPort)
    if err := http.ListenAndServe(config.HTTPPort, nil); err != nil {
        fmt.Println("Error starting server:", err)
    }
}

