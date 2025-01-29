package manager

import (
	"fmt"
	"log"
	"net/rpc"

	"github.com/shafigh75/Memorandum/config"
)

type NodeService struct {
	ClusterManager *ClusterManager
}

func NewNodeService(cm *ClusterManager) *NodeService {
	return &NodeService{ClusterManager: cm}
}

type RPCRequest struct {
	Key   string
	Value string
	TTL   int64
}

type RPCResponse struct {
	Success bool
	Data    string
	Error   string
}

func (ns *NodeService) GetConfig() *config.Config {
	cfg, err := config.LoadConfig("config/config.json")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	return cfg
}

func (ns *NodeService) SetData(data map[string]string, ttl int64, reply *bool) error {
	cfg := ns.GetConfig()
	replica := cfg.ReplicaCount
	for key, value := range data {
		nodes := ns.ClusterManager.GetNodes(key, replica) // 1 replica
		if len(nodes) == 0 {
			return fmt.Errorf("no active nodes available")
		}

		for _, node := range nodes {
			if !node.Active {
				continue
			}

			client, err := rpc.Dial("tcp", node.Address)
			if err != nil {
				log.Printf("RPC connection failed: %s - %v", node.Address, err)
				continue
			}

			req := RPCRequest{Key: key, Value: value, TTL: ttl}
			var resp RPCResponse
			if err := client.Call("RPCService.RPCSet", &req, &resp); err != nil {
				log.Printf("RPCSet failed: %s - %v", node.Address, err)
				client.Close()
				continue
			}

			client.Close()
			if !resp.Success {
				return fmt.Errorf("node %s failed to set key", node.Address)
			}
		}
	}

	*reply = true
	return nil
}

func (ns *NodeService) GetData(key string, reply *RPCResponse) error {
	cfg := ns.GetConfig()
	replica := cfg.ReplicaCount
	nodes := ns.ClusterManager.GetNodes(key, replica)
	if len(nodes) == 0 {
		return fmt.Errorf("no active nodes available")
	}

	for _, node := range nodes {
		if !node.Active {
			continue
		}

		client, err := rpc.Dial("tcp", node.Address)
		if err != nil {
			log.Printf("RPC connection failed: %s - %v", node.Address, err)
			continue
		}

		req := RPCRequest{Key: key}
		var resp RPCResponse
		if err := client.Call("RPCService.RPCGet", &req, &resp); err != nil {
			log.Printf("RPCGet failed: %s - %v", node.Address, err)
			client.Close()
			continue
		}

		client.Close()
		if resp.Success {
			*reply = resp
			return nil
		}
	}

	return fmt.Errorf("no key was found")
}

func (ns *NodeService) DeleteData(key string, reply *bool) error {
	cfg := ns.GetConfig()
	replica := cfg.ReplicaCount
	nodes := ns.ClusterManager.GetNodes(key, replica)
	if len(nodes) == 0 {
		return fmt.Errorf("no active nodes available")
	}

	for _, node := range nodes {
		if !node.Active {
			continue
		}

		client, err := rpc.Dial("tcp", node.Address)
		if err != nil {
			log.Printf("RPC connection failed: %s - %v", node.Address, err)
			continue
		}

		req := RPCRequest{Key: key}
		var resp RPCResponse
		if err := client.Call("RPCService.RPCDelete", &req, &resp); err != nil {
			log.Printf("RPCDelete failed: %s - %v", node.Address, err)
			client.Close()
			continue
		}

		client.Close()
		if resp.Success {
			*reply = true
			return nil
		}
	}

	return fmt.Errorf("all nodes failed to delete key")
}
