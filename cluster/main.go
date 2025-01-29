package cluster

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/shafigh75/Memorandum/cluster/manager"
	"github.com/shafigh75/Memorandum/config"
)

type NodeConfig struct {
	Nodes []string `json:"nodes"`
}

type HTTPResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

var nodesFileMutex sync.Mutex

func authMiddleware(cfg *config.Config, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cfg.AuthEnabled {
			authHeader := r.Header.Get("Authorization")
			if authHeader != "Bearer "+cfg.AuthToken {
				sendError(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next(w, r)
	}
}

func StartHTTPServer(port string) {
	cfg, err := config.LoadConfig("config/config.json")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	nodeService := initializeCluster()

	http.HandleFunc("/set", authMiddleware(cfg, handleSet(nodeService)))
	http.HandleFunc("/get/", authMiddleware(cfg, handleGet(nodeService)))
	http.HandleFunc("/delete/", authMiddleware(cfg, handleDelete(nodeService)))
	http.HandleFunc("/nodes", authMiddleware(cfg, handleNodes(nodeService)))
	http.HandleFunc("/nodes/add", authMiddleware(cfg, handleAddNode(nodeService)))

	log.Printf("memo-cluster running on port %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

func initializeCluster() *manager.NodeService {
	var nodeConfig NodeConfig
	configFile := "cluster/nodes.json"

	file, err := os.Open(configFile)
	if err != nil {
		log.Fatalf("Failed to open nodes.json: %v", err)
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("Failed to read nodes.json: %v", err)
	}

	if err := json.Unmarshal(bytes, &nodeConfig); err != nil {
		log.Fatalf("Failed to parse node config: %v", err)
	}

	clusterManager := manager.NewClusterManager(configFile)
	nodeService := manager.NewNodeService(clusterManager)

	for _, addr := range nodeConfig.Nodes {
		if clusterManager.PingNode(addr) {
			clusterManager.AddNode(addr)
		}
	}

	go clusterManager.StartHealthCheck()
	go clusterManager.StartConfigMonitor()

	return nodeService
}

type SetRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	TTL   int64  `json:"ttl"`
}

func handleSet(svc *manager.NodeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var requests []SetRequest
		if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
			sendError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		data := make(map[string]string)
		var ttl int64
		for _, req := range requests {
			data[req.Key] = req.Value
			ttl = req.TTL
		}

		var resp bool
		if err := svc.SetData(data, ttl, &resp); err != nil {
			log.Printf("Set error: %v", err)
			sendError(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		sendResponse(w, HTTPResponse{
			Success: true,
			Data:    data,
		}, http.StatusOK)
	}
}

func handleGet(svc *manager.NodeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		key := r.URL.Path[len("/get/"):]
		if key == "" {
			sendError(w, "Key required", http.StatusBadRequest)
			return
		}

		var resp manager.RPCResponse
		if err := svc.GetData(key, &resp); err != nil {
			log.Printf("Get error: %v", err)
			sendError(w, err.Error(), http.StatusOK)
			return
		}

		sendResponse(w, HTTPResponse{
			Success: true,
			Data:    resp.Data,
		}, http.StatusOK)
	}
}

func handleDelete(svc *manager.NodeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		key := r.URL.Path[len("/delete/"):]
		if key == "" {
			sendError(w, "Key required", http.StatusBadRequest)
			return
		}

		var resp bool
		if err := svc.DeleteData(key, &resp); err != nil {
			log.Printf("Delete error: %v", err)
			sendError(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		sendResponse(w, HTTPResponse{
			Success: resp,
		}, http.StatusOK)
	}
}

func handleNodes(svc *manager.NodeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		activeNodes := svc.ClusterManager.GetActiveNodes()
		sendResponse(w, HTTPResponse{
			Success: true,
			Data:    activeNodes,
		}, http.StatusOK)
	}
}

func handleAddNode(svc *manager.NodeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var request struct{ Address string }
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			sendError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := updateNodesJSON(request.Address); err != nil {
			log.Printf("Add node error: %v", err)
			sendError(w, "Failed to update cluster", http.StatusInternalServerError)
			return
		}

		svc.ClusterManager.AddNode(request.Address)
		sendResponse(w, HTTPResponse{
			Success: true,
			Data:    request.Address,
		}, http.StatusCreated)
	}
}

func updateNodesJSON(newNode string) error {
	nodesFileMutex.Lock()
	defer nodesFileMutex.Unlock()

	var nodeConfig NodeConfig
	file, err := os.Open("cluster/nodes.json")
	if err != nil {
		return err
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(bytes, &nodeConfig); err != nil {
		return err
	}

	for _, addr := range nodeConfig.Nodes {
		if addr == newNode {
			return nil
		}
	}

	nodeConfig.Nodes = append(nodeConfig.Nodes, newNode)
	newData, err := json.MarshalIndent(nodeConfig, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile("cluster/nodes.json", newData, 0644)
}

func sendResponse(w http.ResponseWriter, resp HTTPResponse, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

func sendError(w http.ResponseWriter, msg string, status int) {
	sendResponse(w, HTTPResponse{
		Success: false,
		Error:   msg,
	}, status)
}
