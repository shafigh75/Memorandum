package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

// Config holds the configuration settings.
type Config struct {
	HTTPPort         string `json:"http_port"`
	RPCPort          string `json:"rpc_port"`
	CleanupInterval  int64  `json:"cleanup_interval"` // in seconds
	AuthEnabled      bool   `json:"auth_enabled"`
	AuthToken        string `json:"auth_token"`     // Token for authentication
	WalPath          string `json:"WAL_path"`       // Token for authentication
	HttpLogPath      string `json:"http_log_path"`  // Token for authentication
	RPCLogPath       string `json:"rpc_log_path"`   // Token for authentication
	WalBufferSize    int    `json:"WAL_bufferSize"` // Token for authentication
	WalEnabled       bool   `json:"wal_enabled"`
	WalFlushInterval int    `json:"WAL_flushInterval"` // Token for authentication
	NumShards        int    `json:"shard_count"`       // Token for authentication
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
