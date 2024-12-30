package rpc

import (
	"encoding/json"
	"fmt"
	"net"
	"net/rpc"
	"time"

	"github.com/shafigh75/Memorandum/server/db"
	"github.com/shafigh75/Memorandum/utils/logger"
)

// RPCRequest represents the structure of an RPC request.
type RPCRequest struct {
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
	TTL   int64  `json:"ttl"` // TTL in seconds
}

// RPCResponse represents the structure of an RPC response.
type RPCResponse struct {
	Success bool   `json:"success"`
	Data    string `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// RPCService provides the RPC methods for the InMemoryStore.
type RPCService struct {
	Store  *db.ShardedInMemoryStore
	Logger *logger.Logger
}

// RPCSet sets a key-value pair in the store.
func (s *RPCService) RPCSet(req *RPCRequest, resp *RPCResponse) error {
	s.Store.Set(req.Key, req.Value, req.TTL)
	resp.Success = true
	// Create a structured log message
	logMessage := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"method":    "rpc-set",
		"request":   req,
	}

	// Convert the log message to JSON
	logJSON, err := json.Marshal(logMessage)
	if err != nil {
		// Handle JSON marshaling error (optional)
		s.Logger.Log("Error marshaling log message to JSON")
		return err
	}

	// Log the structured message as a JSON string
	s.Logger.Log(string(logJSON))
	return nil
}

// RPCGet retrieves a value by key from the store.
func (s *RPCService) RPCGet(req *RPCRequest, resp *RPCResponse) error {
	if value, exists := s.Store.Get(req.Key); exists {
		resp.Success = true
		resp.Data = value
	} else {
		resp.Success = false
		resp.Error = "Key not found or expired"
	}
	// Create a structured log message
	logMessage := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"method":    "rpc-get",
		"request":   req,
	}

	// Convert the log message to JSON
	logJSON, err := json.Marshal(logMessage)
	if err != nil {
		// Handle JSON marshaling error (optional)
		s.Logger.Log("Error marshaling log message to JSON")
		return err
	}

	// Log the structured message as a JSON string
	s.Logger.Log(string(logJSON))
	return nil
}

// RPCDelete removes a key-value pair from the store.
func (s *RPCService) RPCDelete(req *RPCRequest, resp *RPCResponse) error {
	s.Store.Delete(req.Key)
	resp.Success = true
	// Create a structured log message
	logMessage := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"method":    "rpc-delete",
		"request":   req,
	}

	// Convert the log message to JSON
	logJSON, err := json.Marshal(logMessage)
	if err != nil {
		// Handle JSON marshaling error (optional)
		s.Logger.Log("Error marshaling log message to JSON")
		return err
	}

	// Log the structured message as a JSON string
	s.Logger.Log(string(logJSON))
	return nil
}

// StartRPCServer starts the RPC server.
func StartRPCServer(store *db.ShardedInMemoryStore, port string, logger *logger.Logger) {
	rpcService := &RPCService{Store: store, Logger: logger}
	rpc.Register(rpcService)

	listener, err := net.Listen("tcp", port)
	if err != nil {
		panic("Error starting RPC server: " + err.Error())
	}
	defer listener.Close()

	fmt.Println("Starting RPC server on", port)
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go rpc.ServeConn(conn) // Handle each RPC connection in a new goroutine
	}
}
