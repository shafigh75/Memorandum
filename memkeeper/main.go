package main

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/shafigh75/Memorandum/config"
)

type Node struct {
	IP         string
	ShardCount int
	LastSeen   time.Time
	StartShard int
	EndShard   int
}

type ZooKeeperMaster struct {
	Nodes       map[string]Node
	TotalShards int
	mu          sync.Mutex
	MasterIP    string
}

func NewZooKeeperMaster(IP string) *ZooKeeperMaster {
	configFilePath := "config/config.json"
	cfg, err := config.LoadConfig(configFilePath)
	if err != nil {
		fmt.Println("Error loading config:", err)
		return nil
	}
	zm := &ZooKeeperMaster{
		Nodes:       make(map[string]Node),
		TotalShards: cfg.NumShards,
		MasterIP:    IP,
	}

	return zm
}

// GenerateToken generates a random token of the specified length.
func GenerateHashToken(length int) (string, error) {
	// Create a byte slice to hold the random bytes
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	// Encode the random bytes to a base64 string
	// We use base64.RawURLEncoding to avoid padding and make it URL-safe
	token := base64.RawURLEncoding.EncodeToString(bytes)

	// Trim the token to the desired length
	if len(token) > length {
		token = token[:length]
	}

	return token, nil
}

func (zm *ZooKeeperMaster) generateToken() (string, error) {
	HashToken, err := GenerateHashToken(64)
	return HashToken, err
}

func (zm *ZooKeeperMaster) RegisterNode(args *RegisterArgs, reply *RegisterReply) error {
	zm.mu.Lock()
	defer zm.mu.Unlock()
	zm.Nodes[args.IP] = Node{
		IP:         args.IP,
		ShardCount: args.ShardCount,
		LastSeen:   time.Now(),
		StartShard: zm.TotalShards,
		EndShard:   zm.TotalShards + args.ShardCount - 1,
	}
	zm.TotalShards += args.ShardCount
	reply.InitialTotalShards = zm.TotalShards
	return nil
}

type RegisterArgs struct {
	IP         string
	ShardCount int
}

type RegisterReply struct {
	InitialTotalShards int
}

type JointRequest struct {
	Token       string
	Nodes       map[string]Node
	TotalShards int
}

func (zm *ZooKeeperMaster) AgentJoinRequest(args *JointRequest, reply *ClusterStatusReply) error {
	zm.mu.Lock()
	defer zm.mu.Unlock()

	// check if token is correct:
	if args.Token != token {
		return errors.New("invalid token")
	}

	NodeIP := zm.MasterIP
	NodeShardCounts := zm.TotalShards
	// register itself as new node
	RegisterArgs := &RegisterArgs{IP: NodeIP, ShardCount: NodeShardCounts}
	var RegisterReply *RegisterReply
	err := zm.RegisterNode(RegisterArgs, RegisterReply)
	if err != nil {
		return err
	}

	return nil
}

func (zm *ZooKeeperMaster) GetClusterStatus(args *ZooKeeperMaster, reply *ClusterStatusReply) error {
	zm.mu.Lock()
	defer zm.mu.Unlock()
	reply.Nodes = make(map[string]Node)
	for ip, node := range zm.Nodes {
		reply.Nodes[ip] = node
	}
	reply.TotalShards = zm.TotalShards
	return nil
}

type ClusterStatusReply struct {
	Nodes       map[string]Node
	TotalShards int
}

func (zm *ZooKeeperMaster) monitorNodes() {
	for {
		time.Sleep(10 * time.Second)
		zm.mu.Lock()
		for ip, node := range zm.Nodes {
			_, err := net.DialTimeout("tcp", ip+":2181", 3*time.Second)
			if err == nil {
				zm.Nodes[ip] = Node{
					IP:         node.IP,
					ShardCount: node.ShardCount,
					LastSeen:   time.Now(),
					StartShard: node.StartShard,
					EndShard:   node.EndShard,
				}
			}
			if time.Since(node.LastSeen) > 30*time.Second {
				fmt.Printf("Node %s unresponsive, removing from cluster\n", ip)
				zm.TotalShards -= node.ShardCount
				delete(zm.Nodes, ip)

			}
		}
		zm.mu.Unlock()
	}
}

var token string

func (zm *ZooKeeperMaster) StartServer(port string) error {
	rpc.Register(zm)
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("Error starting server:", err)
		return err
	}
	defer ln.Close()

	fmt.Println("ZooKeeper master started with token:", token)

	go zm.monitorNodes()

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go rpc.ServeConn(conn)
	}
}

func (zm *ZooKeeperMaster) disconnectNode(ip string) {
	zm.mu.Lock()
	defer zm.mu.Unlock()
	fmt.Println(zm.Nodes)
	if node, exists := zm.Nodes[ip]; exists {
		zm.TotalShards -= node.ShardCount
		delete(zm.Nodes, ip)
		fmt.Printf("Node %s disconnected and removed from cluster\n", ip)
	} else {
		fmt.Printf("Node %s not found in cluster\n", ip)
	}
}

// Add these new structures for the disconnect operation
type DisconnectArgs struct {
	IP string
}

type DisconnectReply struct {
	Success bool
}

// Implement the RPC method for disconnecting a node
func (zm *ZooKeeperMaster) DisconnectNodeRPC(args *DisconnectArgs, reply *DisconnectReply) error {
	zm.mu.Lock()
	defer zm.mu.Unlock()
	if node, exists := zm.Nodes[args.IP]; exists {
		zm.TotalShards -= node.ShardCount
		delete(zm.Nodes, args.IP)
		fmt.Printf("Node %s disconnected and removed from cluster\n", args.IP)
		reply.Success = true
	} else {
		fmt.Printf("Node %s not found in cluster\n", args.IP)
		reply.Success = false
	}
	return nil
}

func cli(master *ZooKeeperMaster) {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		parts := strings.Split(input, " ")

		switch parts[0] {
		case "add":
			if len(parts) < 3 {
				fmt.Println("Usage: add <IP> <ShardCount>")
				continue
			}
			ip := parts[1]
			shardCount := parts[2]
			shardCounts, _ := strconv.Atoi(shardCount)

			// send agent token to server:
			// send the total shards and node list to agent:

			args := &RegisterArgs{IP: ip, ShardCount: shardCounts}
			var reply RegisterReply

			client, err := rpc.Dial("tcp", "localhost:2181")
			if err != nil {
				fmt.Println("Error connecting to RPC server:", err)
				return
			}
			defer client.Close()

			err = client.Call("ZooKeeperMaster.RegisterNode", args, &reply)
			if err != nil {
				fmt.Println("Error calling RegisterNode:", err)
				continue
			}
			fmt.Printf("Registered node %s with %d shards, initial total shards: %d\n", ip, shardCounts, reply.InitialTotalShards)

		case "status":
			var reply ClusterStatusReply
			client, err := rpc.Dial("tcp", "localhost:2181")
			if err != nil {
				fmt.Println("Error connecting to RPC server:", err)
				return
			}
			defer client.Close()
			err = client.Call("ZooKeeperMaster.GetClusterStatus", &master, &reply)
			if err != nil {
				fmt.Println("Error calling GetClusterStatus:", err)
				continue
			}
			fmt.Println("Current cluster status:")
			for ip, node := range reply.Nodes {
				fmt.Printf("Node: %s, Shards: %d (%d-%d), Last Seen: %v\n", ip, node.ShardCount, node.StartShard, node.EndShard, node.LastSeen)
			}
			fmt.Printf("Total shards: %d\n", reply.TotalShards)

		case "disconnect":
			if len(parts) < 2 {
				fmt.Println("Usage: disconnect <IP>")
				continue
			}
			ip := parts[1]

			// Create a new RPC client to call the DisconnectNodeRPC method
			client, err := rpc.Dial("tcp", "localhost:2181")
			if err != nil {
				fmt.Println("Error connecting to RPC server:", err)
				continue
			}
			defer client.Close()

			args := &DisconnectArgs{IP: ip}
			var reply DisconnectReply

			err = client.Call("ZooKeeperMaster.DisconnectNodeRPC", args, &reply)
			if err != nil {
				fmt.Println("Error calling DisconnectNodeRPC:", err)
				continue
			}
			if reply.Success {
				fmt.Printf("Successfully disconnected node %s\n", ip)
			} else {
				fmt.Printf("Failed to disconnect node %s: not found\n", ip)
			}

		case "exit":
			fmt.Println("Exiting CLI...")
			return

		default:
			fmt.Println("Unknown command. Available commands: add, status, disconnect, exit")
		}
	}
}

func main() {
	// Run the CLI
	StartIp := flag.String("start-server", "", "start the Memkeeper on this node")
	IsCli := flag.Bool("attach", false, "attach to Memkeeper cli tool")
	master := NewZooKeeperMaster(*StartIp)
	token, _ = master.generateToken()

	// Parse the command-line flags
	flag.Parse()

	if *StartIp != "" {
		master.Nodes[*StartIp] = Node{
			IP:         *StartIp,
			ShardCount: master.TotalShards,
			StartShard: 0,
			EndShard:   master.TotalShards - 1,
			LastSeen:   time.Now(),
		}
		err := master.StartServer("2181")
		if err != nil {
			os.Exit(0)
		}

		// Handle SIGINT and SIGTERM to gracefully shut down
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigs
			fmt.Println("Shutting down...")
			os.Exit(0)
		}()

		// Keep the main goroutine alive
		select {}
	}

	// Use the flags
	if *IsCli {
		cli(master)
		return // Run CLI in a separate goroutine
	}

}

/*
1- server zookeeper is up and running (command ./Zookeeper -start-server)
2- server add agent by ip and agent's token and sends total shards and node list
3- agent verify server add request and updates the total shards, appends itself to the list and send ok to server
4- server receives ok from agent and updates the node list
5- db package is changed so that if zookeeper is on, the zookeeper respond to get shard key otherwise db handles it
(we can add RPC method on the Memorandum to update useZooKeeper flag)
*/
