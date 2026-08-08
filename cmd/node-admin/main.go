package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"valence/internal/cliclient"
)

func main() {
	nodeFlag := flag.String("node", "localhost:3000", "Node HTTP API address")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: node-admin [flags] <command> [args]")
		fmt.Println("\nCommands:")
		fmt.Println("  status                        Show node status")
		fmt.Println("  peers                         Show connected peers")
		fmt.Println("  mine                          Trigger mining on the node")
		fmt.Println("\nFlags:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	command := args[0]
	nodeURL := "http://" + *nodeFlag
	if strings.HasPrefix(*nodeFlag, "http") {
		nodeURL = *nodeFlag
	}

	switch command {
	case "status":
		cliclient.HandleGet(nodeURL, "/status")
	case "peers":
		if len(args) > 1 && args[1] == "add" {
			if len(args) < 3 {
				fmt.Println("Usage: node-admin peers add <peer_address>")
				os.Exit(1)
			}
			cliclient.HandlePeersAdd(nodeURL, args[2])
		} else {
			cliclient.HandleGet(nodeURL, "/peers")
		}
	case "mine":
		cliclient.HandlePost(nodeURL, "/mine")
	default:
		fmt.Printf("Unknown admin command: %s\n", command)
		os.Exit(1)
	}
}
