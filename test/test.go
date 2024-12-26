package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "sync"
    "time"
)

const (
    baseURL = "http://localhost:6060"
    numKeys = 10000 // Number of keys to set
    numGoroutines = 1000 // Number of concurrent goroutines
)

func setKey(key int, wg *sync.WaitGroup) {
    defer wg.Done()
    value := fmt.Sprintf("value-%d", key)
    ttl := 10 // TTL in seconds

    reqBody, _ := json.Marshal(map[string]interface{}{
        "key":   fmt.Sprintf("key-%d", key),
        "value": value,
        "ttl":   ttl,
    })

    _, err := http.Post(baseURL+"/set", "application/json", bytes.NewBuffer(reqBody))
    if err != nil {
        fmt.Printf("Error setting key: %v\n", err)
    }
}

func getKey(key int, wg *sync.WaitGroup) {
    defer wg.Done()
    resp, err := http.Get(fmt.Sprintf("%s/get?key=key-%d", baseURL, key))
    if err != nil {
        fmt.Printf("Error getting key: %v\n", err)
        return
    }
    defer resp.Body.Close()
}

func deleteKey(key int, wg *sync.WaitGroup) {
    defer wg.Done()
    req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/delete?key=key-%d", baseURL, key), nil)
    if err != nil {
        fmt.Printf("Error creating delete request: %v\n", err)
        return
    }
    _, err = http.DefaultClient.Do(req)
    if err != nil {
        fmt.Printf("Error deleting key: %v\n", err)
    }
}

func main() {
    var wg sync.WaitGroup

    // Set keys
    start := time.Now()
    for i := 0; i < numKeys; i++ {
        wg.Add(1)
        go setKey(i, &wg)
    }
    wg.Wait()
    fmt.Printf("Set %d keys in %v\n", numKeys, time.Since(start))

    // Get keys
    start = time.Now()
    for i := 0; i < numKeys; i++ {
        wg.Add(1)
        go getKey(i, &wg)
    }
    wg.Wait()
    fmt.Printf("Got %d keys in %v\n", numKeys, time.Since(start))

    // Delete keys
    start = time.Now()
    for i := 0; i < numKeys; i++ {
        wg.Add(1)
        go deleteKey(i, &wg)
    }
    wg.Wait()
    fmt.Printf("Deleted %d keys in %v\n", numKeys, time.Since(start))
}

