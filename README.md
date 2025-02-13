<p align="center">
  <h1 align="center">Memorandum</h1>
</p>
<p align="center">
  <img src="original.png" alt="Memorandum logo" width="200"/>
</p>
<hr>

## Overview
**Memorandum** is an open-source, self-hosted, sharded in-memory key-value store written in Go, designed for efficient storage and retrieval of data with support for TTL (time-to-live) and Write-Ahead Logging (WAL) with clustering capabilities. It was developed in response to recent changes in popular database licensing models (read [this](https://www.theregister.com/2024/03/22/redis_changes_license/) for details). This project serves as a learning resource for building in-memory databases and understanding database internals. It also can be used in small to medium scale environments which need a light-weight IMDG. 

## Background

The recent shift towards more restrictive licensing models in some popular databases (more specifically redis) has led many developers to reconsider their approach to data storage. As a response to these changes, i've created Memorandum - a lightweight, easy-to-use key-value store that puts control firmly in the hands of its users. 

## Features
- **In-Memory Storage**: Fast access to key-value pairs stored in memory.
- **Sharded Architecture**: Data is distributed across multiple shards to reduce contention and improve concurrency.
- **TTL Support**: Optional time-to-live for each key to automatically expire data.
- **Write-Ahead Logging (WAL)**: Logs write operations to ensure data durability and facilitate recovery.
- **interfaces**: Implemented as a command-line interface and a network server with two http and RPC interfaces so far.
- **clustering**: This project contains distributed clustering capabilities. Features include: Data Replication, Health Checks, Dynamic Node Management and Authentication.


## Usage as Standalone service

## Installation
To get started with Memorandum, clone the repository and build the project using the simple script:

```sh
curl -sSL https://raw.githubusercontent.com/shafigh75/Memorandum/main/build.sh | bash
```
### NOTE:
make sure golang is installed on your server or else the build script will not work for you.

### Running the Server
To start the Memorandum server, run the following command:

```sh
./Memorandum
```

### Running the cli
use this command to start the CLI:
```sh
./Memorandum-cli
```

### Configuration
Memorandum uses a configuration file to set various parameters such as the number of shards, WAL file path, buffer size, and flush interval. Update the `config.yaml` file with your desired settings(detailed explanation later on).

```json
# Example config/config.json
{
  "http_port": ":6060",
  "rpc_port": ":1234",
  "cluster_port": ":5036",
  "WAL_path": "/home/test/Memorandum/data/wal.bin",
  "http_log_path": "/home/test/Memorandum/logs/http.log",
  "rpc_log_path": "/home/test/Memorandum/logs/rpc.log",
  "WAL_bufferSize": 4096,
  "WAL_flushInterval": 30,
  "cleanup_interval": 10,
  "heartbeat_interval": 10,
  "configCheck_interval": 10,
  "auth_enabled": true,
  "wal_enabled": true,
  "cluster_enabled": true,
  "shard_count": 32,
  "replica_count": 0,
  "auth_token": "f5e0c51b7f3c6e6b57deb13b3017c32e"
}
```
**NOTE**: make sure to check the config file and set the values based on your requirements.
**NOTE**: In cli there is a `passwd` command that will generate a new auth token for you :)

## Usage as module
Here is an example of how to use the Memorandum library in your Go project:

### Installation
To get started with Memorandum, you need to have Go installed on your machine. You can install Memorandum by running:
```bash
go get -u github.com/shafigh75/Memorandum
```

```go
package main

import (
"fmt"
"github.com/shafigh75/Memorandum/server/db"
)

func main() {
// Load configuration and create store
store, err := db.LoadConfigAndCreateStore("config.json")
if err != nil {
fmt.Println("Error:", err)
return
}

// Set a key-value pair with TTL
store.Set("key1", "value1", 60) // TTL in seconds

// Retrieve the value
value, exists := store.Get("key1")
if exists {
fmt.Println("Retrieved value:", value)
}

// Delete the key
store.Delete("key1")

// Close the store
store.Close()
}
```

## API Reference

### Set
Sets a key-value pair in the store with an optional TTL.
```go
func (s *ShardedInMemoryStore) Set(key, value string, ttl int64)
```
- `key`: The key to store.
- `value`: The value associated with the key.
- `ttl`: Time-To-Live in seconds. If `0`, the key never expires.

### Get
Retrieves the value for a given key, considering expiration.
```go
func (s *ShardedInMemoryStore) Get(key string) (string, bool)
```
- `key`: The key to retrieve.
- Returns the value and a boolean indicating if the key exists and is not expired.

### Delete
Removes a key-value pair from the store.
```go
func (s *ShardedInMemoryStore) Delete(key string)
```
- `key`: The key to delete.

### Cleanup
Removes expired keys from the store. Can be run periodically.
```go
func (s *ShardedInMemoryStore) Cleanup()
```

# Memorandum Configuration

The `config.json` file is used to configure various aspects of the Memorandum in-memory data store. Below is a detailed description of each configuration parameter:

## Configuration Parameters

### Server Configuration
- **http_port**: Specifies the port on which the HTTP server listens.
- Example: `":6060"`

- **rpc_port**: Specifies the port on which the RPC server listens.
- Example: `":1234"`

- **cluster_port**: Specifies the port on which the http cluster server listens.
- Example: `":5036"`

### Paths
- **WAL_path**: Specifies the path to the Write-Ahead Log (WAL) file.
- Example: `"/home/test/Memorandum/data/wal.bin"`

- **http_log_path**: Specifies the path to the HTTP log file.
- Example: `"/home/test/Memorandum/logs/http.log"`

- **rpc_log_path**: Specifies the path to the RPC log file.
- Example: `"/home/test/Memorandum/logs/rpc.log"`

### WAL Configuration
- **WAL_bufferSize**: Specifies the buffer size for the Write-Ahead Log (in bytes).
- Example: `4096`

- **WAL_flushInterval**: Specifies the interval (in seconds) at which the WAL is flushed to disk.
- Example: `30`

### Cleanup Configuration
- **cleanup_interval**: Specifies the interval (in seconds) at which expired keys are cleaned up.
- Example: `10`

### heartbeat Configuration
- **heartbeat_interval**: Specifies the interval (in seconds) at which the nodes in cluster will be checked.
- Example: `100`

### re-add to cluster Configuration
- **configCheck_interval**: Specifies the interval (in seconds) at which the disabled nodes in cluster will be checked and re-add to cluster if node is up again.
- Example: `100`

### replication factor
- **replica_count**: Specifies the number of nodes the data will be replicated to. starting from 0 (meaning no replication, data is stored on one node only) to n-1 (n is the total node count).
- Example: `0`

### clustering enabled
- **cluster_enabled**: Specifies whether clustering is enabled or is it running as standalone server with a single node.
- Example: `true`

### Authentication
- **auth_enabled**: Enables or disables authentication.
- Example: `true`

- **auth_token**: Specifies the authentication token required for accessing the service.
- Example: `"f5e0c51b7f3c6e6b57deb13b3017c32e"`

### Sharding
- **shard_count**: Specifies the number of shards to be used for the in-memory store.
- Example: `32`

### WAL Enable
- **wal_enabled**: Enables or disables the Write-Ahead Log.
- Example: `true`

## Sample Configuration File

```json
{
  "http_port": ":6060",
  "rpc_port": ":1234",
  "cluster_port": ":5036",
  "WAL_path": "/home/test/Memorandum/data/wal.bin",
  "http_log_path": "/home/test/Memorandum/logs/http.log",
  "rpc_log_path": "/home/test/Memorandum/logs/rpc.log",
  "WAL_bufferSize": 4096,
  "WAL_flushInterval": 30,
  "cleanup_interval": 10,
  "heartbeat_interval": 10,
  "configCheck_interval": 10,
  "auth_enabled": true,
  "wal_enabled": true,
  "cluster_enabled": true,
  "shard_count": 32,
  "replica_count": 0,
  "auth_token": "f5e0c51b7f3c6e6b57deb13b3017c32e"
}
```

## Loading Configuration

To load the configuration and create the store, use the following code:

```go
store, err := db.LoadConfigAndCreateStore("config.json")
if err != nil {
fmt.Println("Error loading configuration:", err)
return
}

// Use the store for your operations
// Example: Set a key-value pair
store.Set("key1", "value1", 60)

// Close the store when done
defer store.Close()
```

## Clustering Overview

This project contains distributed clustering capabilities. Features include:
- **Data Replication**: Uses consistent hashing to distribute keys across nodes.
- **Health Checks**: Periodic node pinging to detect failures.
- **Dynamic Node Management**: Nodes can be added via API or `nodes.json` updates.
- **Authentication**: Optional token-based auth for API endpoints.


### Key Components
1. **ClusterManager**  
   - Manages node lifecycle (add/remove).
   - Monitors `nodes.json` for changes.
   - Performs health checks.
2. **NodeService**  
   - Handles RPC calls to nodes for `SET`, `GET`, and `DELETE` operations.
   - Uses replication (default: 1 replica) for fault tolerance.
3. **HTTP Server**  
   - Exposes REST API for data operations and cluster management.

---

**NOTE**: for adding nodes, we can modify the `cluster/nodes.json` file. example of this file:
```json
// format of node address: IP + <RPC_PORT>
{
  "nodes": [
    "127.0.0.1:1234",
    "1.1.1.1:1234",
    "2.2.2.2:1234"
  ]
}
```

**NOTE**: always add the `127.0.0.1:<RPC_PORT>` as this is crucial for your current node. 
**NOTE**: make sure nodes.json file has the same content on all nodes in the cluster.

---


## API Documentation

### Authentication
If enabled in `config.json`, include the header:  
`Authorization: Bearer <AUTH_TOKEN>`

---

### Endpoints

#### 1. Set Key-Value Pair(s)
- **Method**: `POST`
- **URL**: `/set`
- **Body**: 

```json
  [
    {
      "key": "name",
      "value": "mohammad",
      "ttl": 3600
    },
    {
      "key": "age",
      "value": "28",
      "ttl": 1800
    }
  ]
```

Response:
```json   
    {
      "success": true,
      "data": {
        "name": "mohammad",
        "age": "28"
      }
    }
```

#### 2. Get Key-Value Pair(s)
- **Method**: `GET`
- **URL**: `/get/<KEY>`
- **Body**: 

```json
THIS REQUEST HAS NO BODY :)
```

Response:
```json   
    {
      "success": true,
      "data": "mohammad"
    }
```

#### 3. Delete Key
- **Method**: `DELETE`
- **URL**: `/delete/<KEY>`
- **Body**: 

```json
THIS REQUEST HAS NO BODY :)

```

Response:
```json   
    {
      "success": true
    }
```

#### 4. List Active Nodes
- **Method**: `GET`
- **URL**: `/nodes`
- **Body**: 

```json
THIS REQUEST HAS NO BODY :)
```

Response:
```json   
    {
      "success": true,
      "data": ["127.0.0.1:1234", "1.1.1.1:1234"]
    }
```

#### 5. Add Node
- **Method**: `POST`
- **URL**: `/nodes/add`
- **Body**: 

```json
    {
      "address": "192.168.1.100:1234"
    }
```

Response:
```json   
    {
      "success": true,
      "data": "192.168.1.100:1234"
    }
```


## Benchmarks

To measure the performance of the key operations, you can run the benchmark tests. These tests provide insights into the time taken for `Set`, `Get`, and `Delete` operations. 

### Running Benchmarks 
Navigate to the directory containing the `db` package and run: 

```sh
go test -bench=.
```

### example results of Running benchmarks WAL-enabled:
```sh
goos: linux
goarch: amd64
pkg: github.com/shafigh75/Memorandum/server/db
cpu: Intel(R) Xeon(R) Platinum 8280 CPU @ 2.70GHz
BenchmarkSet-24           357301              2919 ns/op
BenchmarkGet-24          4718535               273.0 ns/op
BenchmarkDelete-24        557890              1911 ns/op
PASS
ok      github.com/shafigh75/Memorandum/server/db       5.010s

```

### example results of Running benchmarks WAL-disabled:
```sh
goos: linux
goarch: amd64
pkg: github.com/shafigh75/Memorandum/server/db
cpu: Intel(R) Xeon(R) Platinum 8280 CPU @ 2.70GHz
BenchmarkSet-24           868538              1235 ns/op
BenchmarkGet-24          4332234               282.9 ns/op
BenchmarkDelete-24       5088898               253.6 ns/op
PASS
ok      github.com/shafigh75/Memorandum/server/db       4.256s

```
**NOTE** : notice the huge write differnce when disabling WAL


## Contributing
Contributions are welcome! If you have suggestions for improvements or new features, feel free to open an issue or submit a pull request.

## License
This project is licensed under the GPL-V3 License. See the [LICENSE](LICENSE) file for details.

## Acknowledgments
- Inspired by various in-memory database projects and resources.
- Thanks to the Go community for their excellent documentation and support.

---


## Disclaimer
This is a simple in-memory database implementation which was built as a hobby project and may not perform well under enterprise-level workload.


<hr>

