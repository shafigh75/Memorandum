package manager

import (
	"encoding/json"
	"hash/crc32"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
	"sync"
	"time"

	"github.com/shafigh75/Memorandum/config"
)

type Node struct {
	Address string
	Active  bool
	Index   int
}

type ClusterManager struct {
	Nodes               []*Node
	Mutex               sync.Mutex
	HeartbeatInterval   time.Duration
	configFile          string
	LastModTime         time.Time
	configCheckInterval time.Duration
}

func NewClusterManager(configFile string) *ClusterManager {
	cfg, err := config.LoadConfig("config/config.json")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	return &ClusterManager{
		Nodes:               make([]*Node, 0),
		HeartbeatInterval:   time.Duration(cfg.HeartbeatInterval) * time.Second,
		configFile:          configFile,
		configCheckInterval: time.Duration(cfg.ConfigCheckInterval) * time.Second,
	}
}

func (cm *ClusterManager) AddNode(address string) {
	cm.Mutex.Lock()
	defer cm.Mutex.Unlock()

	for _, node := range cm.Nodes {
		if node.Address == address {
			log.Printf("Node updated: %s", address)
			cm.Nodes[node.Index] = &Node{
				Address: address,
				Active:  true,
				Index:   node.Index,
			}
			return
		}
	}

	newNode := &Node{
		Address: address,
		Active:  true,
		Index:   len(cm.Nodes),
	}
	cm.Nodes = append(cm.Nodes, newNode)
	log.Printf("Node added: %s", address)
}

func (cm *ClusterManager) RemoveNode(address string) {
	cm.Mutex.Lock()
	defer cm.Mutex.Unlock()

	for i, node := range cm.Nodes {
		if node.Address == address {
			cm.Nodes = append(cm.Nodes[:i], cm.Nodes[i+1:]...)
			// Re-index remaining nodes
			for j := i; j < len(cm.Nodes); j++ {
				cm.Nodes[j].Index = j
			}
			log.Printf("Node removed: %s", address)
			return
		}
	}
}

func (cm *ClusterManager) StartHealthCheck() {
	ticker := time.NewTicker(cm.HeartbeatInterval)
	defer ticker.Stop()

	for range ticker.C {
		cm.Mutex.Lock()
		for _, node := range cm.Nodes {
			if !cm.PingNode(node.Address) {
				node.Active = false
				log.Printf("Node inactive: %s", node.Address)
			}
		}
		cm.Mutex.Unlock()
	}
}

func (cm *ClusterManager) PingNode(address string) bool {
	client, err := rpc.Dial("tcp", address)
	if err != nil {
		return false
	}
	defer client.Close()

	var reply bool
	err = client.Call("RPCService.Ping", struct{}{}, &reply)
	return err == nil && reply
}

func (cm *ClusterManager) GetActiveNodes() []string {
	cm.Mutex.Lock()
	defer cm.Mutex.Unlock()

	active := make([]string, 0)
	for _, node := range cm.Nodes {
		if node.Active {
			active = append(active, node.Address)
		}
	}
	return active
}

func (cm *ClusterManager) GetNodes(key string, replicas int) []*Node {
	cm.Mutex.Lock()
	defer cm.Mutex.Unlock()

	if len(cm.Nodes) == 0 {
		return nil
	}

	hash := crc32.ChecksumIEEE([]byte(key))
	primaryIdx := int(hash) % len(cm.Nodes)
	nodes := make([]*Node, 0)

	active := make([]*Node, 0)
	for _, node := range cm.Nodes {
		if node.Active {
			active = append(active, node)
		}
	}
	for i := 0; i <= replicas; i++ {
		idx := (primaryIdx + i) % len(active)
		if idx < len(active) {
			nodes = append(nodes, active[idx])
		}
	}
	return nodes
}

func (cm *ClusterManager) StartConfigMonitor() {
	ticker := time.NewTicker(cm.configCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		cm.syncWithConfig()
	}
}

func (cm *ClusterManager) syncWithConfig() {
	cm.Mutex.Lock()

	log.Println("syncing cluster ...")
	// 1. Check file modification time
	fi, err := os.Stat(cm.configFile)
	if err != nil || fi.ModTime().Before(cm.LastModTime) {
		cm.Mutex.Unlock()
		return
	}
	cm.LastModTime = fi.ModTime()

	// 2. Read config file
	file, err := os.Open(cm.configFile)
	if err != nil {
		cm.Mutex.Unlock()
		log.Printf("Error opening config: %v", err)
		return
	}
	bytes, err := ioutil.ReadAll(file)
	file.Close()
	if err != nil {
		cm.Mutex.Unlock()
		log.Printf("Error reading config: %v", err)
		return
	}

	var nodeConfig struct{ Nodes []string }
	if err := json.Unmarshal(bytes, &nodeConfig); err != nil {
		cm.Mutex.Unlock()
		log.Printf("Error parsing config: %v", err)
		return
	}

	// 3. Identify changes while locked
	var toAdd []string
	var toRemove []string

	// Find new nodes to add
	for _, addr := range nodeConfig.Nodes {
		exists := false
		for _, node := range cm.Nodes {
			if node.Address == addr && node.Active {
				exists = true
				break
			}
		}
		if !exists {
			toAdd = append(toAdd, addr)
		}
	}

	// Find stale nodes to remove
	for _, node := range cm.Nodes {
		found := false
		for _, configAddr := range nodeConfig.Nodes {
			if node.Address == configAddr {
				found = true
				break
			}
		}
		if !found {
			toRemove = append(toRemove, node.Address)
		}
	}

	cm.Mutex.Unlock() // Release lock before making changes

	// 4. Apply changes without holding the lock
	for _, addr := range toAdd {
		if cm.PingNode(addr) {
			cm.AddNode(addr) // Safe: AddNode handles its own locking
		}
	}

	for _, addr := range toRemove {
		cm.RemoveNode(addr) // Safe: RemoveNode handles its own locking
	}
}
