package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

// Config holds the configuration settings.
type Config struct {
	HTTPPort            string `json:"http_port"`            // port for http
	RPCPort             string `json:"rpc_port"`             // port for rpc
	ClusterPort         string `json:"cluster_port"`         // port for clustreing
	CleanupInterval     int64  `json:"cleanup_interval"`     // memory cleanup interval in seconds
	HeartbeatInterval   int64  `json:"heartbeat_interval"`   // check nodes health interval in seconds
	ConfigCheckInterval int64  `json:"configCheck_interval"` // interval to re-add nodes in seconds
	AuthEnabled         bool   `json:"auth_enabled"`         // set to true to enable auth
	AuthToken           string `json:"auth_token"`           // Token for authentication
	WalPath             string `json:"WAL_path"`             // path for wal.bin file
	HttpLogPath         string `json:"http_log_path"`        // http log file path
	RPCLogPath          string `json:"rpc_log_path"`         // rpc log file path
	WalBufferSize       int    `json:"WAL_bufferSize"`       // buffer size for each wal flush
	WalEnabled          bool   `json:"wal_enabled"`          // turn wal logging on or off
	WalFlushInterval    int    `json:"WAL_flushInterval"`    // wal flush interval in seconds
	NumShards           int    `json:"shard_count"`          // number of node shards
	ReplicaCount        int    `json:"replica_count"`        // number of nodes to replicate our data
}

// LoadConfig reads the configuration from a JSON file.
func LoadConfig(filePath string) (*Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	byteValue, _ := ioutil.ReadAll(file)
	var config Config
	if err := json.Unmarshal(byteValue, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
