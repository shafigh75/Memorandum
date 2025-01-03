<p align="center">
  <h1 align="center">Memorandum</h1>
</p>
<p align="center">
  <img src="original.png" alt="Memorandum logo" width="200"/>
</p>
<hr>

## Overview
**Memorandum** is an open-source, self-hosted, sharded in-memory key-value store written in Go, designed for efficient storage and retrieval of data with support for TTL (time-to-live) and Write-Ahead Logging (WAL). It was developed in response to recent changes in popular database licensing models (read [this](https://www.theregister.com/2024/03/22/redis_changes_license/) for details). This project serves as a learning resource for building in-memory databases and understanding database internals.

## Background

The recent shift towards more restrictive licensing models in some popular databases (more specifically redis) has led many developers to reconsider their approach to data storage. As a response to these changes, i've created Memorandum - a lightweight, easy-to-use key-value store that puts control firmly in the hands of its users. 

## Features
- **In-Memory Storage**: Fast access to key-value pairs stored in memory.
- **Sharded Architecture**: Data is distributed across multiple shards to reduce contention and improve concurrency.
- **TTL Support**: Optional time-to-live for each key to automatically expire data.
- **Write-Ahead Logging (WAL)**: Logs write operations to ensure data durability and facilitate recovery.
- **interfaces**: Implemented as a command-line interface and a network server with two http and RPC interfaces so far.

## Installation
To get started with Memorandum, clone the repository and build the project using the simple script:

```sh
curl -sSL https://raw.githubusercontent.com/shafigh75/Memorandum/main/build.sh | bash
```
### NOTE:
make sure golang is installed on your server or else the build script will not work for you.

## Usage
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
Memorandum uses a configuration file to set various parameters such as the number of shards, WAL file path, buffer size, and flush interval. Update the `config.yaml` file with your desired settings.

```json
# Example config/config.json
{
  "http_port": ":6060",
  "rpc_port": ":1234",
  "WAL_path": "/home/test/Memorandum/data/WAL.gz",
  "http_log_path": "/home/test/Memorandum/logs/http.log",
  "rpc_log_path": "/home/test/Memorandum/logs/rpc.log",
  "WAL_bufferSize": 4096,
  "WAL_flushInterval": 30,
  "cleanup_interval": 10,
  "auth_enabled": true,
  "wal_enabled": false,
  "shard_count": 32,
  "auth_token": "f5e0c51b7f3c6e6b57deb13b3017c32e"
}
```
**NOTE**: make sure to check the config file and set the values based on your requirements. 

### Example Code
Here is an example of how to use the Memorandum library in your Go project:

get the package:
```sh
  go get github.com/shafigh75/Memorandum
```

use in your project:
```go
package main

import (
    "fmt"
    "github.com/shafigh75/Memorandum/server/db"
    "time"
)

func main() {
    wal, err := db.NewWAL("data/wal.log", 100, 10*time.Second)
    if err != nil {
        panic(err)
    }

    store := db.NewShardedInMemoryStore(16, wal)
    defer store.Close()

    // Set a key-value pair with TTL
    store.Set("foo", "bar", 30) // TTL of 30 seconds

    // Get the value
    value, exists := store.Get("foo")
    if exists {
        fmt.Println("Value:", value)
    } else {
        fmt.Println("Key not found or expired")
    }

    // Delete the key
    store.Delete("foo")
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

