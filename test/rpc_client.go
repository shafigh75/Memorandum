package main

import (
	"fmt"
	"net/rpc"
)

// RPCRequest and RPCResponse structures
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

func main() {
	// Connect to the RPC server
	client, err := rpc.Dial("tcp", "localhost:1234")
	if err != nil {
		fmt.Println("Error connecting to RPC server:", err)
		return
	}
	defer client.Close()

	// Example: Set a key
	setReq := RPCRequest{Key: "exampleKey", Value: "exampleValue", TTL: 60}
	var setResp RPCResponse
	err = client.Call("InMemoryStore.RPCSet", &setReq, &setResp)
	if err != nil {
		fmt.Println("Error calling RPCSet:", err)
		return
	}
	fmt.Println("Set Response:", setResp)

	// Example: Get the key
	getReq := RPCRequest{Key: "exampleKey"}
	var getResp RPCResponse
	err = client.Call("InMemoryStore.RPCGet", &getReq, &getResp)
	if err != nil {
		fmt.Println("Error calling RPCGet:", err)
		return
	}
	fmt.Println("Get Response:", getResp)

	// Example: Delete the key
	deleteReq := RPCRequest{Key: "exampleKey"}
	var deleteResp RPCResponse
	err = client.Call("InMemoryStore.RPCDelete", &deleteReq, &deleteResp)
	if err != nil {
		fmt.Println("Error calling RPCDelete:", err)
		return
	}
	fmt.Println("Delete Response:", deleteResp)

	// Attempt to get the key again after deletion
	err = client.Call("InMemoryStore.RPCGet", &getReq, &getResp)
	if err != nil {
		fmt.Println("Error calling RPCGet after deletion:", err)
		return
	}
	fmt.Println("Get Response after deletion:", getResp)
}
