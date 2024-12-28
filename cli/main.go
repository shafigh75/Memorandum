package main

import (
	"Memorandum/config"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/rpc"
	"strings"

	"github.com/chzyer/readline"
	"github.com/spf13/cobra"
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

var (
	client          *rpc.Client
	authToken       string
	isAuthenticated bool
)

var rootCmd = &cobra.Command{
	Use:   "mycli",
	Short: "My CLI application",
	Long:  `This is a sample CLI application using Cobra with autocompletion and REPL.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Welcome to My CLI! Type 'help' for available commands.")
		startREPL()
	},
}

func startREPL() {
	// Define completer
	completer := readline.NewPrefixCompleter(
		readline.PcItem("help"),
		readline.PcItem("exit"),
		readline.PcItem("auth"),
		readline.PcItem("passwd"),
		readline.PcItem("set", readline.PcItem("key"), readline.PcItem("value"), readline.PcItem("ttl")),
		readline.PcItem("get", readline.PcItem("key")),
		readline.PcItem("delete", readline.PcItem("key")),
	)

	// Create readline instance
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "> ",
		HistoryFile:     "/tmp/readline.tmp",
		AutoComplete:    completer,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		fmt.Println("Error creating readline:", err)
		return
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil {
			break
		}
		line = strings.TrimSpace(line)

		if line == "exit" {
			fmt.Println("Exiting REPL.")
			break
		}

		handleCommand(line)
	}
}

func handleCommand(input string) {
	args := strings.Fields(input)
	if len(args) == 0 {
		return
	}

	if !isAuthenticated && args[0] != "auth" {
		fmt.Println("Authentication required. Please use 'auth <token>' to authenticate.")
		return
	}

	switch args[0] {
	case "help":
		fmt.Println("Available commands: help, exit, auth [token], passwd, set [key] [value] [ttl], get [key], delete [key]")
	case "auth":
		if len(args) != 2 {
			fmt.Println("Usage: auth [token]")
			return
		}
		authenticate(args[1])
	case "passwd":
		generatePassword()
	case "set":
		if len(args) < 4 {
			fmt.Println("Usage: set [key] [value] [ttl]")
			return
		}
		key := args[1]
		ttl := int64(0)
		if _, err := fmt.Sscanf(args[len(args)-1], "%d", &ttl); err != nil {
			fmt.Println("Invalid TTL value.")
			return
		}
		value := strings.Join(args[2:len(args)-1], " ")
		setKey(key, value, ttl)
	case "get":
		if len(args) != 2 {
			fmt.Println("Usage: get [key]")
			return
		}
		getKey(args[1])
	case "delete":
		if len(args) != 2 {
			fmt.Println("Usage: delete [key]")
			return
		}
		deleteKey(args[1])
	default:
		fmt.Printf("Unknown command: %s\n", input)
	}
}

func authenticate(token string) {
	if token == authToken {
		isAuthenticated = true
		fmt.Println("Authentication successful.")
	} else {
		fmt.Println("Authentication failed. Please check your token.")
	}
}

func generatePassword() {
	password := make([]byte, 16) // 16 bytes = 128 bits
	if _, err := rand.Read(password); err != nil {
		fmt.Println("Error generating password:", err)
		return
	}
	newToken := fmt.Sprintf("%x", password)

	// Update the config file with the new token
	configFilePath := "config/config.json"
	cfg, err := config.LoadConfig(configFilePath)
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	cfg.AuthToken = newToken
	configData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling config:", err)
		return
	}

	if err := ioutil.WriteFile(configFilePath, configData, 0644); err != nil {
		fmt.Println("Error writing config file:", err)
		return
	}

	fmt.Println("New password generated and saved to config:", newToken)
}

func setKey(key, value string, ttl int64) {
	req := RPCRequest{Key: key, Value: value, TTL: ttl}
	var resp RPCResponse
	err := client.Call("RPCService.RPCSet", &req, &resp)
	if err != nil {
		fmt.Println("Error calling RPCSet:", err)
		return
	}
	if resp.Success {
		fmt.Println("Key set successfully.")
	} else {
		fmt.Println("Error:", resp.Error)
	}
}

func getKey(key string) {
	req := RPCRequest{Key: key}
	var resp RPCResponse
	err := client.Call("RPCService.RPCGet", &req, &resp)
	if err != nil {
		fmt.Println("Error calling RPCGet:", err)
		return
	}
	if resp.Success {
		fmt.Printf("Value: %s\n", resp.Data)
	} else {
		fmt.Println("Error:", resp.Error)
	}
}

func deleteKey(key string) {
	req := RPCRequest{Key: key}
	var resp RPCResponse
	err := client.Call("RPCService.RPCDelete", &req, &resp)
	if err != nil {
		fmt.Println("Error calling RPCDelete:", err)
		return
	}
	if resp.Success {
		fmt.Println("Key deleted successfully.")
	} else {
		fmt.Println("Error:", resp.Error)
	}
}

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("config/config.json")
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	// Connect to the RPC server
	client, err = rpc.Dial("tcp", cfg.RPCPort)
	if err != nil {
		fmt.Println("Error connecting to RPC server:", err)
		return
	}
	defer client.Close()

	// Load the configuration to get the auth token
	authToken = cfg.AuthToken

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		return
	}
}
