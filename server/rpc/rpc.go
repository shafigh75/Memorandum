package rpc

import (
	"Memorandum/server/db"
	"net"
	"net/rpc"
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
	Store *db.InMemoryStore
}

// RPCSet sets a key-value pair in the store.
func (s *RPCService) RPCSet(req *RPCRequest, resp *RPCResponse) error {
	s.Store.Set(req.Key, req.Value, req.TTL)
	resp.Success = true
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
	return nil
}

// RPCDelete removes a key-value pair from the store.
func (s *RPCService) RPCDelete(req *RPCRequest, resp *RPCResponse) error {
	s.Store.Delete(req.Key)
	resp.Success = true
	return nil
}

// StartRPCServer starts the RPC server.
func StartRPCServer(store *db.InMemoryStore, port string) {
	rpcService := &RPCService{Store: store}
	rpc.Register(rpcService)

	listener, err := net.Listen("tcp", port)
	if err != nil {
		panic("Error starting RPC server: " + err.Error())
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go rpc.ServeConn(conn) // Handle each RPC connection in a new goroutine
	}
}
